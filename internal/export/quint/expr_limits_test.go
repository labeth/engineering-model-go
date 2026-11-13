// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package quint

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/labeth/engineering-model-go/internal/model"
)

func leafVar(name string) model.SemanticExpression {
	return model.SemanticExpression{Kind: model.MachineKindVariable, Value: name}
}

func opNode(op string, operands ...model.SemanticExpression) model.SemanticExpression {
	return model.SemanticExpression{Kind: model.MachineKindOperator, Operator: op, Operands: operands}
}

// leftChain builds a + a + ... + a with n operands, left nested.
func leftChain(n int) model.SemanticExpression {
	e := leafVar("a")
	for i := 1; i < n; i++ {
		e = opNode("+", e, leafVar("a"))
	}
	return e
}

func limitsRenderer() exprRenderer {
	return exprRenderer{variables: map[string]model.MachineType{"a": model.MachineTypeInteger}}
}

// allocatedBytes returns the heap bytes allocated while f runs.
func allocatedBytes(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// TRLC-LINKS: REQ-EMG-141
func TestRenderingLongChainFromSourceIsLinear(t *testing.T) {
	// The longest chain the parser accepts (8 KiB of source).
	src := strings.Repeat("a+", (model.MaxMachineExprSourceBytes-2)/2) + "a"
	expr, diags := model.ParseMachineExpr(src, model.MachineExprBoolean)
	if len(diags) != 0 {
		t.Fatalf("parse: %v", diags)
	}
	r := limitsRenderer()
	text, typ, p := r.expr(expr)
	if p != nil || typ != model.MachineTypeInteger {
		t.Fatalf("render: %v, type %s", p, typ)
	}
	n := (model.MaxMachineExprSourceBytes-2)/2 + 1
	if want := strings.Repeat("(", n-2) + "v_a + v_a" + strings.Repeat(") + v_a", n-2); text != want {
		t.Errorf("rendered chain differs from the explicit left-nested parenthesisation (%d bytes, want %d)", len(text), len(want))
	}
}

// TRLC-LINKS: REQ-EMG-141
func TestRenderingIsLinearInTheSizeOfTheTree(t *testing.T) {
	r := limitsRenderer()
	const n = 2000
	var small, large string
	smallBytes := allocatedBytes(func() {
		var p *problem
		if small, _, p = r.expr(leftChain(n)); p != nil {
			t.Fatal(p)
		}
	})
	largeBytes := allocatedBytes(func() {
		var p *problem
		if large, _, p = r.expr(leftChain(2 * n)); p != nil {
			t.Fatal(p)
		}
	})
	// Output size is linear (parentheses + operands).
	if len(large) < 2*len(small)-100 || len(large) > 2*len(small)+100 {
		t.Fatalf("output sizes %d and %d are not linear", len(small), len(large))
	}
	// The tree itself costs leaf and node allocations, linear in n. The old
	// level-by-level concatenation copied the growing text at every level, so
	// doubling the chain quadrupled the bytes; linear rendering roughly doubles
	// them.
	if ratio := float64(largeBytes) / float64(smallBytes); ratio > 2.6 {
		t.Errorf("doubling the chain multiplied allocated bytes by %.1f (%d -> %d), want about 2", ratio, smallBytes, largeBytes)
	}
	// Absolute bound: a small multiple of the output, not of its square.
	if limit := uint64(64 * len(large)); largeBytes > limit {
		t.Errorf("rendering a %d-byte result allocated %d bytes, want at most %d", len(large), largeBytes, limit)
	}
}

// TRLC-LINKS: REQ-EMG-141
func TestRenderingLongChainIsFast(t *testing.T) {
	r := limitsRenderer()
	start := time.Now()
	for i := 0; i < 20; i++ {
		if _, _, p := r.expr(leftChain(4096)); p != nil {
			t.Fatal(p)
		}
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("20 renderings of a 4096-operand chain took %v", elapsed)
	}
}

// TRLC-LINKS: REQ-EMG-141
func TestRenderingDeepChainKeepsErrorsPrecise(t *testing.T) {
	r := limitsRenderer()
	// An undeclared variable at the deepest leaf is still reported by name.
	e := leafVar("zz")
	for i := 0; i < 3000; i++ {
		e = opNode("+", e, leafVar("a"))
	}
	if _, _, p := r.expr(e); p == nil || p.code != codeUnresolved || !strings.Contains(p.msg, `"zz"`) {
		t.Fatalf("problem = %v", p)
	}
	// A type mismatch is found at the operator that sees it, with its operand text.
	bad := opNode("+", leftChain(10), opNode("<", leafVar("a"), leafVar("a")))
	if _, _, p := r.expr(bad); p == nil || p.code != codeType || !strings.Contains(p.msg, "needs Integer for operand 2") {
		t.Fatalf("problem = %v", p)
	}
	// The result of a failed render is discarded, never partially returned.
	if text, typ, p := r.expr(bad); p == nil || text != "" || typ != "" {
		t.Fatalf("partial result %q %q", text, typ)
	}
}
