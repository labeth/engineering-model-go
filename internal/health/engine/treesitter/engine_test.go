// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/health"
)

// want is the expected metrics of one function: every number is hand-computed
// and the reasoning is in the comments of the fixture the function lives in.
// codes lists the findings as code:severity in report order, without the
// health. prefix.
type want struct {
	name                            string
	kind                            string
	params, cyc, nest, bumps, bools int
	codes                           []string
}

func fn(name, kind string, params, cyc, nest, bumps, bools int, codes ...string) want {
	return want{name: name, kind: kind, params: params, cyc: cyc, nest: nest, bumps: bumps, bools: bools, codes: codes}
}

// load reads a fixture from testdata; the fixtures carry a .txt suffix so the
// repository's own code scan does not treat them as source.
func load(t *testing.T, fixture string) []byte {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", fixture+".txt"))
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func languageOf(t *testing.T, path string) string {
	t.Helper()
	lang, ok := codemap.LanguageForPath(path)
	if !ok {
		t.Fatalf("no language for %s", path)
	}
	return lang
}

// analyzeSource analyses one in-memory file with the default configuration.
func analyzeSource(t *testing.T, path string, src []byte, cfg health.Config) health.FileResult {
	t.Helper()
	res, err := New().Analyze([]health.FileInput{{Path: path, Language: languageOf(t, path), Source: src}}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("got %d results, want 1", len(res))
	}
	return res[0]
}

// analyzeFixture analyses a fixture under the virtual path path.
func analyzeFixture(t *testing.T, fixture, path string) health.FileResult {
	t.Helper()
	return analyzeSource(t, path, load(t, fixture), health.Config{})
}

func codesOf(f health.Function) []string {
	var out []string
	for _, finding := range f.Findings {
		out = append(out, strings.TrimPrefix(finding.Code, "health.")+":"+string(finding.Severity))
	}
	return out
}

func checkFunctions(t *testing.T, res health.FileResult, wants []want) {
	t.Helper()
	if len(res.Functions) != len(wants) {
		names := []string{}
		for _, f := range res.Functions {
			names = append(names, f.Name)
		}
		t.Fatalf("got %d functions %v, want %d", len(res.Functions), names, len(wants))
	}
	for i, w := range wants {
		f := res.Functions[i]
		got := want{f.Name, f.Kind, f.Parameters, f.Cyclomatic, f.MaxNesting, f.Bumps, f.MaxBoolOps, codesOf(f)}
		if !reflect.DeepEqual(got, w) {
			t.Errorf("function %d:\n got %+v\nwant %+v", i, got, w)
		}
	}
}

// naiveLOC is a second, line-based implementation of logical lines for
// fixtures without block comments or multi-line strings: a line counts when it
// is not blank and does not start with the comment prefix.
func naiveLOC(src []byte, first, last int, prefix string) int {
	lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	count := 0
	for _, line := range lines[first-1 : last] {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, prefix) {
			count++
		}
	}
	return count
}

// checkLOC compares the engine's LOC with the line-based count for every
// function except those named in skip.
func checkLOC(t *testing.T, res health.FileResult, src []byte, prefix string, skip ...string) {
	t.Helper()
	for _, f := range res.Functions {
		if contains(skip, f.Name) {
			continue
		}
		if got := naiveLOC(src, f.StartLine, f.EndLine, prefix); got != f.LOC {
			t.Errorf("%s: loc %d, line-based count %d", f.Name, f.LOC, got)
		}
	}
}

// wantLOC checks the logical lines of one function.
func wantLOC(t *testing.T, res health.FileResult, name string, loc int) {
	t.Helper()
	if got := function(t, res, name).LOC; got != loc {
		t.Errorf("%s: loc %d, want %d", name, got, loc)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func function(t *testing.T, res health.FileResult, name string) health.Function {
	t.Helper()
	for _, f := range res.Functions {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no function %s", name)
	return health.Function{}
}

// TRLC-LINKS: REQ-EMG-150
func TestGoSmells(t *testing.T) {
	res := analyzeFixture(t, "go_smells.go", "pkg/smells.go")
	// Cyclomatic is 1 + decision points. CycPos: nine ifs -> 10 (warn at 10);
	// CycNeg: eight -> 9. Constructs: if 1, && 1, || 1, else-if 1, for 1,
	// range 1, case 1,2 -> 1, case 3 -> 1, type cases 2, comm case 1 = 11
	// decisions (default_case adds none) -> 12; its longest boolean chain has
	// two operators. BoolSplit: two ifs + 2 + 2 operators -> 7, chain 2 (the
	// parentheses of the second condition do not split it). Closures: the go
	// func (if) and defer func (for) are anonymous -> 1 + 2 = 3; the named
	// closure run is its own unit with 1 + 1.
	checkFunctions(t, res, []want{
		fn("f", "function", 0, 1, 0, 0, 0),
		fn("CycPos", "function", 1, 10, 1, 0, 0, "complex_method:warn"),
		fn("CycNeg", "function", 1, 9, 1, 0, 0),
		fn("NestPos", "function", 1, 5, 4, 1, 0, "deep_nesting:warn"),
		fn("NestNeg", "function", 1, 4, 3, 1, 0),
		fn("BumpsPos", "function", 1, 5, 2, 2, 0, "bumpy_road:warn"),
		fn("BumpsNeg", "function", 1, 4, 2, 1, 0),
		fn("BoolPos", "function", 1, 5, 1, 0, 3, "complex_conditional:warn"),
		fn("BoolNeg", "function", 1, 4, 1, 0, 2),
		fn("BoolSplit", "function", 1, 7, 1, 0, 2),
		fn("ArgsPos", "function", 5, 1, 0, 0, 0, "excess_arguments:warn"),
		fn("ArgsNeg", "function", 4, 1, 0, 0, 0),
		fn("PrimPos", "function", 4, 1, 0, 0, 0, "primitive_obsession:warn"),
		fn("PrimNeg", "function", 4, 1, 0, 0, 0),
		fn("PrimFew", "function", 3, 1, 0, 0, 0),
		fn("Variadic", "function", 2, 1, 0, 0, 0),
		fn("Constructs", "function", 4, 12, 1, 0, 2, "complex_method:warn"),
		fn("ElseIf", "function", 1, 4, 1, 0, 0),
		fn("Worker.Method", "method", 1, 2, 1, 0, 0),
		fn("Closures", "function", 1, 3, 1, 0, 0),
		fn("Closures.run", "function", 0, 2, 1, 0, 0),
	})
	// Closures spans 18 logical lines; run owns 5 (its "run := func() {" row
	// included), so Closures counts 13.
	checkLOC(t, res, load(t, "go_smells.go"), "//", "Closures")
	wantLOC(t, res, "Closures", 13)
	if res.IsTest || len(res.Unsupported) != 0 {
		t.Errorf("a plain Go file: IsTest %v, Unsupported %v", res.IsTest, res.Unsupported)
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestRustSmells(t *testing.T) {
	res := analyzeFixture(t, "rust_smells.rs", "src/smells.rs")
	// constructs: if 1, && of the let chain 1, else-if 1, while let 1, for 1,
	// match arms beyond the first 2, && in the guard 1 = 8 decisions -> 9;
	// loop and ? add nothing. Methods of impl and trait blocks are qualified by
	// the type; self is no parameter. closures: the for_each closure is
	// anonymous -> 1 + 1; run, apply and inner are units of their own.
	checkFunctions(t, res, []want{
		fn("f", "function", 0, 1, 0, 0, 0),
		fn("cyc_pos", "function", 1, 10, 1, 0, 0, "complex_method:warn"),
		fn("cyc_neg", "function", 1, 9, 1, 0, 0),
		fn("nest_pos", "function", 1, 5, 4, 1, 0, "deep_nesting:warn"),
		fn("nest_neg", "function", 1, 4, 3, 1, 0),
		fn("bumps_pos", "function", 1, 5, 2, 2, 0, "bumpy_road:warn"),
		fn("bumps_neg", "function", 1, 4, 2, 1, 0),
		fn("bool_pos", "function", 1, 5, 1, 0, 3, "complex_conditional:warn"),
		fn("bool_neg", "function", 1, 4, 1, 0, 2),
		fn("args_pos", "function", 5, 1, 0, 0, 0, "excess_arguments:warn"),
		fn("args_neg", "function", 4, 1, 0, 0, 0),
		fn("prim_pos", "function", 4, 1, 0, 0, 0, "primitive_obsession:warn"),
		fn("prim_neg", "function", 4, 1, 0, 0, 0),
		fn("prim_few", "function", 3, 1, 0, 0, 0),
		fn("constructs", "function", 2, 9, 1, 0, 1),
		fn("Worker.method", "method", 1, 2, 1, 0, 0),
		fn("Worker.associated", "method", 1, 1, 0, 0, 0),
		fn("Speaker.speak", "method", 1, 2, 1, 0, 0),
		fn("closures", "function", 1, 2, 1, 0, 0),
		fn("closures.run", "function", 1, 2, 1, 0, 0),
		fn("closures.apply", "function", 1, 2, 1, 0, 0),
		fn("closures.inner", "function", 1, 2, 1, 0, 0),
	})
	// closures spans 22 logical lines; run, apply and inner own 5 rows each.
	checkLOC(t, res, load(t, "rust_smells.rs"), "//", "closures")
	wantLOC(t, res, "closures", 7)
}

// ecmaWants are the expectations shared by the TypeScript and JavaScript
// fixtures; the only difference is the primitive obsession finding.
func ecmaWants(javascript bool) []want {
	prim := []string{"primitive_obsession:warn"}
	params := fn("params", "function", 2, 1, 0, 0, 0)
	if javascript {
		prim = nil
		params = fn("params", "function", 3, 1, 0, 0, 0)
	}
	wants := []want{}
	if javascript {
		wants = append(wants, fn("f", "function", 0, 1, 0, 0, 0))
	}
	wants = append(wants,
		fn("cycPos", "function", 1, 10, 1, 0, 0, "complex_method:warn"),
		fn("cycNeg", "function", 1, 9, 1, 0, 0),
		fn("nestPos", "function", 1, 5, 4, 1, 0, "deep_nesting:warn"),
		fn("nestNeg", "function", 1, 4, 3, 1, 0),
		fn("bumpsPos", "function", 1, 5, 2, 2, 0, "bumpy_road:warn"),
		fn("bumpsNeg", "function", 1, 4, 2, 1, 0),
		fn("boolPos", "function", 1, 5, 1, 0, 3, "complex_conditional:warn"),
		fn("boolNeg", "function", 1, 4, 1, 0, 2),
		fn("argsPos", "function", 5, 1, 0, 0, 0, "excess_arguments:warn"),
		fn("argsNeg", "function", 4, 1, 0, 0, 0),
		fn("primPos", "function", 4, 1, 0, 0, 0, prim...),
		fn("primNeg", "function", 4, 1, 0, 0, 0),
		fn("primFew", "function", 3, 1, 0, 0, 0),
		params,
		// constructs: if 1, && 1, || 1, else-if 1, for 1, for-of 1, for-in 1,
		// while 1, do 1, case 1 and case 2 -> 2, catch 1, ?: 1, ?? 1, ||= 1,
		// &&= 1, ??= 1 = 17 decisions (default and try add none) -> 18. The
		// longest chain is x.a && x.b || x.c.
		fn("constructs", "function", 3, 18, 1, 0, 2, "complex_method:warn"),
		fn("elseIf", "function", 1, 4, 1, 0, 0),
		fn("arrow", "function", 1, 2, 1, 0, 0),
		fn("Service.run", "method", 1, 2, 1, 0, 0),
		fn("Service.field", "method", 1, 2, 1, 0, 0),
		// closures: the forEach and map callbacks are anonymous -> 1 + 2; run
		// (assigned arrow) and inner (nested declaration) are units.
		fn("closures", "function", 2, 3, 1, 0, 0),
		fn("closures.run", "function", 0, 2, 1, 0, 0),
		fn("closures.inner", "function", 0, 2, 1, 0, 0),
	)
	return wants
}

// TRLC-LINKS: REQ-EMG-150
func TestTypeScriptSmells(t *testing.T) {
	res := analyzeFixture(t, "ts_smells.ts", "src/smells.ts")
	checkFunctions(t, res, ecmaWants(false))
	// closures spans 15 logical lines; run and inner own 3 rows each.
	checkLOC(t, res, load(t, "ts_smells.ts"), "//", "closures")
	wantLOC(t, res, "closures", 9)
	if len(res.Unsupported) != 0 {
		t.Errorf("TypeScript declares parameter types, Unsupported = %v", res.Unsupported)
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestJavaScriptSmells(t *testing.T) {
	res := analyzeFixture(t, "js_smells.js", "src/smells.js")
	wants := append(ecmaWants(true),
		// single: one bare arrow parameter, one ternary; default is the export
		// default function; handler/method/exports.shared are named by their
		// property or assignment target.
		fn("single", "function", 1, 2, 0, 0, 0),
		fn("default", "function", 0, 2, 1, 0, 0),
		fn("handler", "function", 1, 2, 1, 0, 0),
		fn("method", "method", 1, 2, 1, 0, 0),
		fn("exports.shared", "function", 1, 2, 1, 0, 0),
	)
	checkFunctions(t, res, wants)
	checkLOC(t, res, load(t, "js_smells.js"), "//", "closures")
	wantLOC(t, res, "closures", 9)
	if !reflect.DeepEqual(res.Unsupported, []string{health.CodePrimitiveObsession}) {
		t.Errorf("JavaScript has no parameter types, Unsupported = %v", res.Unsupported)
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestTSXSmells(t *testing.T) {
	res := analyzeFixture(t, "tsx_smells.tsx", "src/view.tsx")
	// Component: if 1, ternary 1, && 1 -> 4 (JSX parses with the tsx grammar);
	// the map callback is anonymous. Row has one destructured parameter and a
	// ternary in an expression-bodied arrow.
	checkFunctions(t, res, []want{
		fn("Component", "function", 1, 4, 1, 0, 1),
		fn("Row", "function", 1, 2, 0, 0, 0),
	})
	if got := function(t, res, "Component").LOC; got != 11 {
		t.Errorf("Component loc = %d, want 11", got)
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestPythonSmells(t *testing.T) {
	res := analyzeFixture(t, "python_smells.py", "pkg/smells.py")
	// constructs: if 1, and 1, or 1, elif 1, for 1, while 1, two excepts 2,
	// case 1, case 2|3, case [y] if 3 (the guard is not counted again; case _
	// adds none), conditional expression 1, the list comprehension's two ifs 2,
	// generator 1, dict comprehension 1, set comprehension 1 = 17 -> 18.
	// params: a, b, c, kw (the / and * separators are not parameters).
	// group: except* is one decision point. documented: the docstring and the
	// comment are not logical lines, so its two logical lines are def and f().
	// closures: the anonymous lambda in sort(key=...) counts toward closures;
	// run (nested def) and pick (assigned lambda) are units.
	checkFunctions(t, res, []want{
		fn("f", "function", 0, 1, 0, 0, 0),
		fn("cyc_pos", "function", 1, 10, 1, 0, 0, "complex_method:warn"),
		fn("cyc_neg", "function", 1, 9, 1, 0, 0),
		fn("nest_pos", "function", 1, 5, 4, 1, 0, "deep_nesting:warn"),
		fn("nest_neg", "function", 1, 4, 3, 1, 0),
		fn("bumps_pos", "function", 1, 5, 2, 2, 0, "bumpy_road:warn"),
		fn("bumps_neg", "function", 1, 4, 2, 1, 0),
		fn("bool_pos", "function", 1, 5, 1, 0, 3, "complex_conditional:warn"),
		fn("bool_neg", "function", 1, 4, 1, 0, 2),
		fn("args_pos", "function", 5, 1, 0, 0, 0, "excess_arguments:warn"),
		fn("args_neg", "function", 4, 1, 0, 0, 0),
		fn("prim_pos", "function", 4, 1, 0, 0, 0, "primitive_obsession:warn"),
		fn("prim_neg", "function", 4, 1, 0, 0, 0),
		fn("prim_few", "function", 3, 1, 0, 0, 0),
		fn("params", "function", 4, 1, 0, 0, 0),
		fn("constructs", "function", 3, 18, 1, 0, 2, "complex_method:warn"),
		fn("group", "function", 1, 2, 1, 0, 0),
		fn("else_if", "function", 1, 4, 1, 0, 0),
		fn("documented", "function", 1, 1, 0, 0, 0),
		fn("double", "function", 1, 2, 0, 0, 0),
		fn("Service.run", "method", 1, 2, 1, 0, 0),
		fn("Service.make", "method", 1, 2, 1, 0, 0),
		fn("closures", "function", 2, 2, 0, 0, 0),
		fn("closures.run", "function", 0, 2, 1, 0, 0),
		fn("closures.pick", "function", 1, 2, 0, 0, 0),
	})
	// closures spans 7 logical lines; run owns 3 (def, if, f()) and the assigned
	// lambda "pick = lambda ..." owns its row.
	checkLOC(t, res, load(t, "python_smells.py"), "#", "documented", "closures")
	wantLOC(t, res, "closures", 3)
	if got := function(t, res, "documented"); got.LOC != 2 || got.StartLine != 169 || got.EndLine != 174 {
		t.Errorf("documented: loc %d lines %d-%d, want 2 logical lines in 169-174", got.LOC, got.StartLine, got.EndLine)
	}
	if len(res.Unsupported) != 0 {
		t.Errorf("an annotated Python file supports primitive obsession, Unsupported = %v", res.Unsupported)
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestPrimitiveObsessionUnsupported(t *testing.T) {
	plain := analyzeFixture(t, "python_plain.py", "pkg/plain.py")
	js := analyzeFixture(t, "js_smells.js", "src/smells.js")
	annotated := analyzeFixture(t, "python_smells.py", "pkg/smells.py")
	ts := analyzeFixture(t, "ts_smells.ts", "src/smells.ts")
	only := []string{health.CodePrimitiveObsession}
	for name, tc := range map[string]struct {
		res  health.FileResult
		want []string
	}{
		"unannotated python": {plain, only},
		"javascript":         {js, only},
		"annotated python":   {annotated, nil},
		"typescript":         {ts, nil},
	} {
		if !reflect.DeepEqual(tc.res.Unsupported, tc.want) {
			t.Errorf("%s: Unsupported = %v, want %v", name, tc.res.Unsupported, tc.want)
		}
	}
	// Four unannotated parameters are not a finding, only an unsupported smell.
	if got := function(t, plain, "many"); got.Parameters != 4 || len(got.Findings) != 0 {
		t.Errorf("many: %d parameters, findings %v", got.Parameters, codesOf(got))
	}
}

// assertWants are the expected assertion measures of a test fixture.
type assertWant struct {
	name      string
	test      bool
	assertion int
	codes     []string
}

func checkAssertions(t *testing.T, res health.FileResult, wants []assertWant) {
	t.Helper()
	if len(res.Functions) != len(wants) {
		t.Fatalf("got %d functions, want %d", len(res.Functions), len(wants))
	}
	for i, w := range wants {
		f := res.Functions[i]
		got := assertWant{f.Name, f.IsTest, f.Assertions, codesOf(f)}
		if got.name != w.name || got.test != w.test || got.assertion != w.assertion || strings.Join(got.codes, ",") != strings.Join(w.codes, ",") {
			t.Errorf("function %d:\n got %+v\nwant %+v", i, got, w)
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestGoAssertionBlocks(t *testing.T) {
	res := analyzeFixture(t, "go_asserts_test.go", "pkg/asserts_test.go")
	if !res.IsTest {
		t.Error("a _test.go file is a test file")
	}
	dup := []string{"duplicated_assertion_block:warn"}
	// TestAssertPos: t.Errorf, t.Errorf (a comment between does not break the
	// run), an if that only calls t.Fatal, assert.Equal, require.NoError and
	// t.Error make six; t.Log comes after. TestAssertNeg: 5, t.Log, 5.
	// TestDupA/B share four identical statements; TestNearA/B share three.
	checkAssertions(t, res, []assertWant{
		{"TestAssertPos", true, 6, []string{"large_assertion_block:warn"}},
		{"TestAssertNeg", true, 5, nil},
		{"assertHelper", false, 0, nil},
		{"TestDupA", true, 4, dup},
		{"TestDupB", true, 4, dup},
		{"TestNearA", true, 4, nil},
		{"TestNearB", true, 4, nil},
		{"TestSubtests", true, 3, nil},
		{"BenchmarkSomething", true, 0, nil},
		{"TestMain", false, 0, nil},
	})
	finding := function(t, res, "TestDupA").Findings[0]
	if finding.Metric != "assertions" || finding.Value != 4 || finding.StartLine != 45 || finding.EndLine != 48 {
		t.Errorf("duplicate block finding = %+v", finding)
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestRustAssertionBlocks(t *testing.T) {
	dup := []string{"duplicated_assertion_block:warn"}
	wants := []assertWant{
		{"production", false, 0, nil},
		{"assert_pos", true, 6, []string{"large_assertion_block:warn"}},
		{"assert_neg", true, 5, nil},
		{"helper", false, 0, nil},
		{"dup_a", true, 4, dup},
		{"dup_b", true, 4, dup},
		{"near_a", true, 4, nil},
		{"near_b", true, 4, nil},
	}
	// Under src/ the file is production code with a cfg(test) module: the
	// #[test] functions are tests, the file is not.
	lib := analyzeFixture(t, "rust_asserts.rs", "src/lib.rs")
	if lib.IsTest {
		t.Error("src/lib.rs is not a test file")
	}
	checkAssertions(t, lib, wants)
	// Under tests/ the whole file is a test file.
	if it := analyzeFixture(t, "rust_asserts.rs", "tests/it.rs"); !it.IsTest {
		t.Error("tests/it.rs is a test file")
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestECMAAssertionBlocks(t *testing.T) {
	dup := []string{"duplicated_assertion_block:warn"}
	wants := []assertWant{
		{"suite.assert pos", true, 6, []string{"large_assertion_block:warn"}},
		{"suite.assert neg", true, 5, nil},
		{"suite.nested.dup a", true, 4, dup},
		{"suite.nested.dup b", true, 4, dup},
		{"suite.near a", true, 4, nil},
		{"suite.near b", true, 4, nil},
		{"suite.each %i", true, 1, nil},
		{"node style", true, 2, nil},
		{"helper", false, 0, nil},
	}
	// assert pos: expect, await expect, assert.equal, assert(), expect with
	// not, assert.strictEqual make six; console.log comes after. Test
	// callbacks of it/test (also .only and .each) are the test units and the
	// describe titles qualify them; t.diagnostic is not an assertion.
	for _, tc := range []struct{ fixture, path string }{
		{"ts_asserts.test.ts", "src/a.test.ts"},
		{"js_asserts.spec.js", "src/a.spec.js"},
	} {
		res := analyzeFixture(t, tc.fixture, tc.path)
		if !res.IsTest {
			t.Errorf("%s is a test file", tc.path)
		}
		checkAssertions(t, res, wants)
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestPythonAssertionBlocks(t *testing.T) {
	dup := []string{"duplicated_assertion_block:warn"}
	// test_pos: assert, self.assertEqual, self.assertTrue, with pytest.raises,
	// self.assertIn and a pytest.raises call make six.
	res := analyzeFixture(t, "python_asserts.py", "tests/test_asserts.py")
	if !res.IsTest {
		t.Error("test_*.py is a test file")
	}
	checkAssertions(t, res, []assertWant{
		{"TestSuite.test_pos", true, 6, []string{"large_assertion_block:warn"}},
		{"TestSuite.helper_check", false, 0, nil},
		{"test_neg", true, 5, nil},
		{"test_dup_a", true, 4, dup},
		{"test_dup_b", true, 4, dup},
		{"test_near_a", true, 4, nil},
		{"test_near_b", true, 4, nil},
		{"not_a_test", false, 0, nil},
	})
	// The same file outside a test file has no tests at all.
	if plain := analyzeFixture(t, "python_asserts.py", "pkg/asserts.py"); plain.IsTest || function(t, plain, "test_neg").IsTest {
		t.Error("a Python file not named test_*.py or *_test.py has no tests")
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestTestFilesAreFlagged(t *testing.T) {
	for path, want := range map[string]bool{
		"pkg/a_test.go":        true,
		"pkg/a.go":             false,
		"src/lib.rs":           false,
		"tests/it.rs":          true,
		"crates/x/tests/it.rs": true,
		"pkg/test_a.py":        true,
		"pkg/a_test.py":        true,
		"pkg/a.py":             false,
		"pkg/attest.py":        false,
		"src/a.test.ts":        true,
		"src/a.spec.tsx":       true,
		"src/a.test.js":        true,
		"src/__tests__/a.js":   true,
		"test/a.ts":            true,
		"tests/a.js":           true,
		"src/a.ts":             false,
		"src/latest.ts":        false,
		"src/contest/a.ts":     false,
	} {
		lang := languageOf(t, path)
		res := analyzeSource(t, path, []byte(map[string]string{"go": "package p\n", "rust": "fn a() {}\n", "python": "x = 1\n"}[lang]), health.Config{})
		if res.IsTest != want {
			t.Errorf("%s: IsTest = %v, want %v", path, res.IsTest, want)
		}
	}
	// A file-level #![cfg(test)] makes a Rust file a test file anywhere.
	if res := analyzeSource(t, "src/helpers.rs", []byte("#![cfg(test)]\nfn a() {}\n"), health.Config{}); !res.IsTest {
		t.Error("#![cfg(test)] marks the file as a test file")
	}
}

// bigFunction builds a function with k nested ifs, flat one-line ifs and
// filler statements, with blank and comment lines that must not be counted. It
// returns the source and its number of logical lines.
type shape struct {
	header, open, closer, flat, filler, footer string
	file                                       string
}

var shapes = map[string]shape{
	"go":         {"func big(x *Box) {", "if x.a {", "}", "if x.b { f() }", "f()", "}", "big.go"},
	"rust":       {"fn big(x: &Box_) {", "if x.a {", "}", "if x.b { f() }", "f();", "}", "big.rs"},
	"typescript": {"function big(x: Box): void {", "if (x.a) {", "}", "if (x.b) { f(); }", "f();", "}", "big.ts"},
	"tsx":        {"function big(x: Box): void {", "if (x.a) {", "}", "if (x.b) { f(); }", "f();", "}", "big.tsx"},
	"javascript": {"function big(x) {", "if (x.a) {", "}", "if (x.b) { f(); }", "f();", "}", "big.js"},
	"python":     {"def big(x: Box) -> None:", "if x.a:", "", "if x.b: f()", "f()", "", "big.py"},
}

var shapeOrder = []string{"go", "rust", "typescript", "tsx", "javascript", "python"}

func (s shape) build(nested, flats, fillers int) (string, int) {
	var b strings.Builder
	logical := 0
	line := func(depth int, text string) {
		if text == "" {
			return
		}
		b.WriteString(strings.Repeat("  ", depth) + text + "\n")
		logical++
	}
	comment := "// a comment is not a logical line\n"
	if s.file == "big.py" {
		comment = "# a comment is not a logical line\n"
	}
	line(0, s.header)
	b.WriteString(comment + "\n")
	for i := 0; i < nested; i++ {
		line(1+i, s.open)
	}
	line(1+nested, s.filler)
	for i := nested - 1; i >= 0; i-- {
		line(1+i, s.closer)
	}
	for i := 0; i < flats; i++ {
		line(1, s.flat)
	}
	for i := 0; i < fillers; i++ {
		line(1, s.filler)
		if i%10 == 0 {
			b.WriteString("\n")
		}
	}
	b.WriteString(comment)
	line(0, s.footer)
	return b.String(), logical
}

func analyzeShape(t *testing.T, s shape, nested, flats, fillers int) (health.Function, int) {
	t.Helper()
	src, logical := s.build(nested, flats, fillers)
	res := analyzeSource(t, s.file, []byte(src), health.Config{})
	return function(t, res, "big"), logical
}

// fillersFor returns the filler count that makes the function have loc logical
// lines.
func (s shape) fillersFor(nested, flats, loc int) int {
	_, logical := s.build(nested, flats, 0)
	return loc - logical
}

// TRLC-LINKS: REQ-EMG-150
func TestLargeMethodAndBrainMethod(t *testing.T) {
	for _, name := range shapeOrder {
		s := shapes[name]
		t.Run(name, func(t *testing.T) {
			// Brain method: 70 logical lines (large at warn), 1 + 4 nested + 6 flat
			// ifs = cyclomatic 11 (complex at warn), depth 4 (deep at warn).
			f, logical := analyzeShape(t, s, 4, 6, s.fillersFor(4, 6, 70))
			if logical != 70 || f.LOC != 70 || f.Cyclomatic != 11 || f.MaxNesting != 4 {
				t.Fatalf("brain: generated %d, loc %d, cyclomatic %d, nesting %d; want 70, 70, 11, 4", logical, f.LOC, f.Cyclomatic, f.MaxNesting)
			}
			wantCodes := []string{"complex_method:warn", "large_method:warn", "deep_nesting:warn", "brain_method:high"}
			if got := codesOf(f); !reflect.DeepEqual(got, wantCodes) {
				t.Errorf("brain method findings = %v, want %v", got, wantCodes)
			}
			// 69 lines: just below large_method, so no brain method either.
			f, _ = analyzeShape(t, s, 4, 6, s.fillersFor(4, 6, 69))
			if f.LOC != 69 || contains(codesOf(f), "large_method:warn") || contains(codesOf(f), "brain_method:high") {
				t.Errorf("69 lines: loc %d findings %v", f.LOC, codesOf(f))
			}
			// Depth 3 (cyclomatic 11 still): large and complex, not deep, no brain.
			f, _ = analyzeShape(t, s, 3, 7, s.fillersFor(3, 7, 70))
			if f.MaxNesting != 3 || f.Cyclomatic != 11 || contains(codesOf(f), "brain_method:high") || !contains(codesOf(f), "large_method:warn") {
				t.Errorf("depth 3: nesting %d cyclomatic %d findings %v", f.MaxNesting, f.Cyclomatic, codesOf(f))
			}
			// Cyclomatic 9: large and deep, not complex, no brain.
			f, _ = analyzeShape(t, s, 4, 4, s.fillersFor(4, 4, 70))
			if f.Cyclomatic != 9 || contains(codesOf(f), "brain_method:high") || contains(codesOf(f), "complex_method:warn") {
				t.Errorf("cyclomatic 9: cyclomatic %d findings %v", f.Cyclomatic, codesOf(f))
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestLargeMethodSeverity(t *testing.T) {
	s := shapes["go"]
	for loc, want := range map[int]string{119: "large_method:warn", 120: "large_method:high"} {
		f, _ := analyzeShape(t, s, 0, 0, s.fillersFor(0, 0, loc))
		if f.LOC != loc || !reflect.DeepEqual(codesOf(f), []string{want}) {
			t.Errorf("loc %d: findings %v, want %s", f.LOC, codesOf(f), want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestBumpsAndBooleanAndArgumentSeverity(t *testing.T) {
	src := func(body string) []byte { return []byte("package p\n\nfunc f() {}\n\n" + body) }
	cases := []struct {
		name, body string
		want       []string
	}{
		{"three bumps", "func A(x *Box) {\n" + strings.Repeat("if x.a {\nif x.b {\nf()\n}\n}\n", 3) + "}\n", []string{"bumpy_road:high"}},
		{"five operators", "func A(x *Box) {\nif x.a && x.b && x.c && x.d && x.e && x.f {\nf()\n}\n}\n", []string{"complex_conditional:high"}},
		{"four operators", "func A(x *Box) {\nif x.a && x.b && x.c && x.d && x.e {\nf()\n}\n}\n", []string{"complex_conditional:warn"}},
		{"seven args", "func A(a, b, c, d, e, g, h *Box) {}\n", []string{"excess_arguments:high"}},
		{"six args", "func A(a, b, c, d, e, g *Box) {}\n", []string{"excess_arguments:warn"}},
		{"nesting five", "func A(x *Box) {\nif x.a {\nif x.b {\nif x.c {\nif x.d {\nif x.e {\nf()\n}\n}\n}\n}\n}\n}\n", []string{"deep_nesting:high"}},
		{"cyclomatic twenty", "func A(x *Box) {\n" + strings.Repeat("if x.a { f() }\n", 19) + "}\n", []string{"complex_method:high"}},
		{"cyclomatic nineteen", "func A(x *Box) {\n" + strings.Repeat("if x.a { f() }\n", 18) + "}\n", []string{"complex_method:warn"}},
	}
	for _, tc := range cases {
		res := analyzeSource(t, "p.go", src(tc.body), health.Config{})
		if got := codesOf(function(t, res, "A")); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: findings %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestAssertionBlockSeverityHigh(t *testing.T) {
	body := func(n int) []byte {
		return []byte("package p\n\nfunc TestA(t *testing.T) {\n" + strings.Repeat("\tt.Errorf(\"x\")\n", n) + "}\n")
	}
	for n, want := range map[int]string{9: "large_assertion_block:warn", 10: "large_assertion_block:high"} {
		res := analyzeSource(t, "a_test.go", body(n), health.Config{})
		if got := codesOf(function(t, res, "TestA")); !reflect.DeepEqual(got, []string{want}) {
			t.Errorf("%d assertions: findings %v, want %s", n, got, want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestConfigReplacesThresholds(t *testing.T) {
	src := load(t, "go_smells.go")
	cfg := health.Config{Thresholds: map[string]health.Threshold{
		// A code present replaces both limits: warn 12, high 12 -> Constructs
		// (12) is high, CycPos (10) is no finding any more.
		health.CodeComplexMethod: {Warn: 12, High: 12},
		// Absent codes keep their defaults: BoolPos stays a finding.
		health.CodeExcessArguments: {Warn: 4, High: 5},
	}}
	res := analyzeSource(t, "p.go", src, cfg)
	if got := codesOf(function(t, res, "Constructs")); !reflect.DeepEqual(got, []string{"complex_method:high", "excess_arguments:warn"}) {
		t.Errorf("Constructs findings = %v", got)
	}
	if got := codesOf(function(t, res, "CycPos")); len(got) != 0 {
		t.Errorf("CycPos findings = %v, want none", got)
	}
	if got := codesOf(function(t, res, "BoolPos")); !reflect.DeepEqual(got, []string{"complex_conditional:warn"}) {
		t.Errorf("BoolPos findings = %v", got)
	}
	if got := codesOf(function(t, res, "ArgsNeg")); !reflect.DeepEqual(got, []string{"excess_arguments:warn"}) {
		t.Errorf("ArgsNeg (4 parameters, warn at 4) findings = %v", got)
	}
	if got := codesOf(function(t, res, "ArgsPos")); !reflect.DeepEqual(got, []string{"excess_arguments:high"}) {
		t.Errorf("ArgsPos (5 parameters, high at 5) findings = %v", got)
	}
	// The default map is not modified by a configuration.
	if health.DefaultThresholds()[health.CodeComplexMethod].Warn != 10 {
		t.Error("DefaultThresholds changed")
	}
}

// TRLC-LINKS: REQ-EMG-159
func TestDeterministicOrder(t *testing.T) {
	inputs := []health.FileInput{
		{Path: "b.py", Language: "python", Source: load(t, "python_smells.py")},
		{Path: "a_test.go", Language: "go", Source: load(t, "go_asserts_test.go")},
		{Path: "a.go", Language: "go", Source: load(t, "go_smells.go")},
		{Path: "a.rs", Language: "rust", Source: load(t, "rust_smells.rs")},
		{Path: "a.ts", Language: "typescript", Source: load(t, "ts_smells.ts")},
		{Path: "a.tsx", Language: "tsx", Source: load(t, "tsx_smells.tsx")},
		{Path: "a.js", Language: "javascript", Source: load(t, "js_smells.js")},
	}
	first, err := New().Analyze(inputs, health.Config{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := New().Analyze(inputs, health.Config{})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatal("two runs over the same input differ")
	}
	rank := map[string]int{}
	for i, code := range health.Codes() {
		rank[code] = i
	}
	for i, res := range first {
		if res.Path != inputs[i].Path {
			t.Errorf("file %d is %s, want input order %s", i, res.Path, inputs[i].Path)
		}
		line := 0
		for _, f := range res.Functions {
			if f.StartLine < line {
				t.Errorf("%s: %s starts at %d before the previous function at %d", res.Path, f.Name, f.StartLine, line)
			}
			line = f.StartLine
			last := -1
			for _, finding := range f.Findings {
				if rank[finding.Code] < last {
					t.Errorf("%s: %s findings are not in the order of Codes()", res.Path, f.Name)
				}
				last = rank[finding.Code]
			}
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestWindowsLineEndings(t *testing.T) {
	for _, tc := range []struct{ fixture, path string }{
		{"go_smells.go", "a.go"},
		{"go_asserts_test.go", "a_test.go"},
		{"rust_smells.rs", "a.rs"},
		{"rust_asserts.rs", "a.rs"},
		{"ts_smells.ts", "a.ts"},
		{"tsx_smells.tsx", "a.tsx"},
		{"js_smells.js", "a.js"},
		{"ts_asserts.test.ts", "a.test.ts"},
		{"python_smells.py", "a.py"},
		{"python_asserts.py", "test_a.py"},
	} {
		lf := load(t, tc.fixture)
		crlf := []byte(strings.ReplaceAll(string(lf), "\n", "\r\n"))
		a := analyzeSource(t, tc.path, lf, health.Config{})
		b := analyzeSource(t, tc.path, crlf, health.Config{})
		if !reflect.DeepEqual(a, b) {
			ja, _ := json.Marshal(a)
			jb, _ := json.Marshal(b)
			t.Errorf("%s: CRLF changes the result\nLF   %s\nCRLF %s", tc.fixture, ja, jb)
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestLogicalLines(t *testing.T) {
	src := "package p\r\n" +
		"\r\n" +
		"/* block\r\n   comment */\r\n" +
		"func A() string { // trailing comment\r\n" +
		"\t// line comment\r\n" +
		"\r\n" +
		"\treturn `raw\r\n\r\nstring`\r\n" +
		"}\r\n"
	res := analyzeSource(t, "a.go", []byte(src), health.Config{})
	// Logical lines of the file: package, func, return `raw, string` and }.
	// The block comment, the line comment, the blank lines and the blank line
	// inside the raw string are not counted. Function A spans lines 5 to 11
	// and holds func, return `raw, string` and }.
	if res.LOC != 5 {
		t.Errorf("file LOC = %d, want 5", res.LOC)
	}
	if f := function(t, res, "A"); f.LOC != 4 || f.StartLine != 5 || f.EndLine != 11 {
		t.Errorf("A: loc %d lines %d-%d, want 4 logical lines in 5-11", f.LOC, f.StartLine, f.EndLine)
	}
	if res := analyzeSource(t, "a.go", []byte(""), health.Config{}); res.LOC != 0 || len(res.Functions) != 0 || res.Functions == nil {
		t.Errorf("empty file: %+v", res)
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestEngineIdentityAndErrors(t *testing.T) {
	e := New()
	if e.Name() != "treesitter" || e.Version() != "2" {
		t.Errorf("Name %q Version %q", e.Name(), e.Version())
	}
	var _ health.Engine = e
	if _, err := e.Analyze([]health.FileInput{{Path: "a.cob", Language: "cobol"}}, health.Config{}); err == nil {
		t.Error("an unsupported language is an error")
	}
	// The language is taken from the path when the input leaves it empty.
	res, err := e.Analyze([]health.FileInput{{Path: "a.go", Source: []byte("package p\nfunc A() {}\n")}}, health.Config{})
	if err != nil || len(res) != 1 || res[0].Language != "go" || len(res[0].Functions) != 1 {
		t.Errorf("language from path: %v %+v", err, res)
	}
}

// TRLC-LINKS: REQ-EMG-159
func TestPerformanceOverInternalTree(t *testing.T) {
	if os.Getenv("ENGMODEL_HEALTH_PERF") == "" {
		t.Skip("set ENGMODEL_HEALTH_PERF=1 to time the analysis of this repository's internal/ tree")
	}
	inputs := repositoryInputs(t)
	start := time.Now()
	res, err := New().Analyze(inputs, health.Config{})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	functions := 0
	for _, r := range res {
		functions += len(r.Functions)
	}
	t.Logf("analysed %d files, %d functions in %s", len(inputs), functions, elapsed)
	if elapsed > 5*time.Second {
		t.Errorf("analysis took %s, want a few seconds at most", elapsed)
	}
}

// BenchmarkAnalyzeInternalTree measures the analysis of this repository's
// internal/ tree.
// TRLC-LINKS: REQ-EMG-150
func BenchmarkAnalyzeInternalTree(b *testing.B) {
	inputs := repositoryInputs(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := New().Analyze(inputs, health.Config{}); err != nil {
			b.Fatal(err)
		}
	}
}

func repositoryInputs(tb testing.TB) []health.FileInput {
	tb.Helper()
	root := filepath.Join("..", "..", "..")
	var inputs []health.FileInput
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "testdata" {
			return filepath.SkipDir
		}
		lang, ok := codemap.LanguageForPath(path)
		if d.IsDir() || !ok {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		inputs = append(inputs, health.FileInput{Path: filepath.ToSlash(rel), Language: lang, Source: src})
		return nil
	})
	if err != nil {
		tb.Fatal(err)
	}
	return inputs
}

// A source file of deeply nested syntax must not crash the engine, exhaust its
// stack or stall it. Nesting beyond MaxSyntaxDepth is refused before the
// recursive walk starts (an error, in every language), nesting below it is
// walked under a time budget, and a 4 MiB file of opening brackets, which
// would overflow the stack of the process, is refused too.
//
// Mutation: removing the depth check makes the 4 MiB case abort the test
// binary with a stack overflow; TestExhaustedBudgetIsAnError fails without
// the budget check (the parent lookup of the binding costs the depth of the
// node, so the walk over nested blocks is quadratic).
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-159
func TestDeeplyNestedSourcesDoNotCrashOrStall(t *testing.T) {
	oldBudget := AnalysisBudget
	AnalysisBudget = time.Second
	t.Cleanup(func() { AnalysisBudget = oldBudget })
	type source struct{ path, src string }
	sources := func(depth int) map[string]source {
		blocks := func(header, open, closer, footer string) string {
			return header + strings.Repeat(open, depth) + "f()\n" + strings.Repeat(closer, depth) + footer
		}
		parens := func(header, footer string) string {
			return header + strings.Repeat("(", depth) + "1" + strings.Repeat(")", depth) + footer
		}
		return map[string]source{
			"go blocks":     {"deep.go", blocks("package p\nfunc f(x bool) {\n", "if x {\n", "}\n", "}\n")},
			"go parens":     {"deep_parens.go", parens("package p\nfunc f() int { return ", " }\n")},
			"rust blocks":   {"deep.rs", blocks("fn f(x: bool) {\n", "if x {\n", "}\n", "}\n")},
			"rust parens":   {"deep_parens.rs", parens("fn f() -> i32 { ", " }\n")},
			"ts blocks":     {"deep.ts", blocks("function f(x: boolean): void {\n", "if (x) {\n", "}\n", "}\n")},
			"tsx parens":    {"deep_parens.tsx", parens("function f(): number { return ", "; }\n")},
			"js blocks":     {"deep.js", blocks("function f(x) {\n", "if (x) {\n", "}\n", "}\n")},
			"js parens":     {"deep_parens.js", parens("function f() { return ", "; }\n")},
			"python calls":  {"deep_calls.py", "def f():\n    return " + strings.Repeat("g(", depth) + "1" + strings.Repeat(")", depth) + "\n"},
			"python parens": {"deep_parens.py", parens("def f():\n    return ", "\n")},
		}
	}
	analyze := func(t *testing.T, c source) (time.Duration, error) {
		start := time.Now()
		_, err := New().Analyze([]health.FileInput{{Path: c.path, Language: languageOf(t, c.path), Source: []byte(c.src)}}, health.Config{})
		return time.Since(start), err
	}
	t.Run("beyond the depth limit", func(t *testing.T) {
		for name, c := range sources(MaxSyntaxDepth * 2) {
			if _, err := analyze(t, c); err == nil || !strings.Contains(err.Error(), "levels deep") {
				t.Errorf("%s: err = %v, want a refusal naming the depth", name, err)
			}
		}
	})
	t.Run("below the depth limit", func(t *testing.T) {
		for name, c := range sources(MaxSyntaxDepth / 4) {
			// An error (the budget) is as good as a result; a stall is not.
			if elapsed, err := analyze(t, c); elapsed > 15*time.Second {
				t.Errorf("%s: took %s (error %v), want it bounded by the budget", name, elapsed, err)
			}
		}
	})
	t.Run("a 4 MiB file of opening brackets", func(t *testing.T) {
		const depth = 2000000
		c := source{"huge.go", "package p\nfunc f() int { return " + strings.Repeat("(", depth) + "1" + strings.Repeat(")", depth) + " }\n"}
		if _, err := analyze(t, c); err == nil || !strings.Contains(err.Error(), "levels deep") {
			t.Errorf("err = %v, want a refusal naming the depth", err)
		}
	})
}

// A long flat file (many statements, many functions, many parameters) is
// analysed completely and quickly.
//
// TRLC-LINKS: REQ-EMG-150
func TestLongFlatSourcesAreAnalysed(t *testing.T) {
	cases := map[string]struct {
		path, src string
		functions int
	}{
		"go statements":     {"flat.go", "package p\nfunc f() {\n" + strings.Repeat("\tg()\n", 100000) + "}\n", 1},
		"python statements": {"flat.py", "def f():\n" + strings.Repeat("    g()\n", 100000), 1},
		"rust functions":    {"many.rs", strings.Repeat("fn f() {}\n", 50000), 50000},
		"ts parameters":     {"params.ts", "function f(" + strings.Repeat("a: number, ", 20000) + "z: number) {}\n", 1},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			res := analyzeSource(t, c.path, []byte(c.src), health.Config{})
			if len(res.Functions) != c.functions {
				t.Errorf("found %d functions, want %d", len(res.Functions), c.functions)
			}
			if elapsed := time.Since(start); elapsed > 20*time.Second {
				t.Errorf("took %s", elapsed)
			}
		})
	}
}

// The engine reports an exhausted budget as an error naming the budget, not as
// a partial result.
//
// TRLC-LINKS: REQ-EMG-150
func TestExhaustedBudgetIsAnError(t *testing.T) {
	old := AnalysisBudget
	AnalysisBudget = time.Nanosecond
	t.Cleanup(func() { AnalysisBudget = old })
	src := "package p\n" + strings.Repeat("func f() {}\n", 5000)
	_, err := New().Analyze([]health.FileInput{{Path: "a.go", Language: "go", Source: []byte(src)}}, health.Config{})
	if err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("err = %v, want one naming the budget", err)
	}
}

// High severity is reached in every language, not only in Go: five nested
// ifs are deep_nesting at high and twenty decision points are complex_method
// at high, at the default limits.
//
// Mutation: a language module that cut its nesting or decision count short
// (for example by not counting else-if or elif) fails here.
//
// TRLC-LINKS: REQ-EMG-150
func TestHighSeverityInEveryLanguage(t *testing.T) {
	for _, name := range shapeOrder {
		s := shapes[name]
		t.Run(name, func(t *testing.T) {
			// 5 nested ifs: depth 5 (high), cyclomatic 1 + 5 + 14 flat = 20 (high).
			f, _ := analyzeShape(t, s, 5, 14, 0)
			if f.MaxNesting != 5 || f.Cyclomatic != 20 {
				t.Fatalf("nesting %d cyclomatic %d, want 5 and 20", f.MaxNesting, f.Cyclomatic)
			}
			got := codesOf(f)
			for _, want := range []string{"deep_nesting:high", "complex_method:high"} {
				if !contains(got, want) {
					t.Errorf("findings %v lack %s", got, want)
				}
			}
			// One step below each limit: warn, not high.
			f, _ = analyzeShape(t, s, 4, 14, 0)
			if got := codesOf(f); !contains(got, "deep_nesting:warn") || contains(got, "deep_nesting:high") {
				t.Errorf("depth 4 findings %v, want deep_nesting:warn only", got)
			}
			f, _ = analyzeShape(t, s, 5, 13, 0)
			if f.Cyclomatic != 19 || !contains(codesOf(f), "complex_method:warn") || contains(codesOf(f), "complex_method:high") {
				t.Errorf("cyclomatic %d findings %v, want complex_method:warn only", f.Cyclomatic, codesOf(f))
			}
		})
	}
}

// A named nested unit is reported on its own, so its rows are not counted again
// in the function around it (Q2 of the second review pass): a 15-line function
// with a 60-line named closure is not a large method. Each source has the same
// shape: outer holds a header, a call and a closing brace of its own; mid is a
// named closure of 62 rows (header, 50 plain lines, a nested closure of 5 rows
// with its call, 4 plain lines and the closing brace); inner is nested in mid.
// Mutation: counting the whole range of the outer function gives 70 and more.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-159
func TestNestedUnitsLinesAreNotCountedTwice(t *testing.T) {
	body := func(indent, line string, n int) string {
		return strings.Repeat(indent+line+"\n", n)
	}
	cases := []struct {
		name, path, src   string
		outer, mid, inner string
		outerLOC, midLOC  int
		innerLOC          int
	}{
		{
			name: "go", path: "a.go",
			src: "package p\n\nfunc Outer() {\n\tmid := func() {\n" + body("\t\t", "work()", 50) +
				"\t\tinner := func() {\n" + body("\t\t\t", "more()", 3) + "\t\t}\n\t\tinner()\n" + body("\t\t", "work()", 4) +
				"\t}\n\tmid()\n}\n",
			outer: "Outer", mid: "Outer.mid", inner: "Outer.mid.inner",
			// Outer: func line, mid() and the closing brace. mid: 62 rows minus the
			// 5 of inner. inner: header, 3 plain lines, closing brace.
			outerLOC: 3, midLOC: 57, innerLOC: 5,
		},
		{
			name: "typescript", path: "a.ts",
			src: "function outer(): void {\n  const mid = () => {\n" + body("    ", "work();", 50) +
				"    const inner = () => {\n" + body("      ", "more();", 3) + "    };\n    inner();\n" + body("    ", "work();", 4) +
				"  };\n  mid();\n}\n",
			outer: "outer", mid: "outer.mid", inner: "outer.mid.inner",
			outerLOC: 3, midLOC: 57, innerLOC: 5,
		},
		{
			name: "javascript", path: "a.js",
			src: "function outer() {\n  const mid = function () {\n" + body("    ", "work();", 50) +
				"    const inner = () => {\n" + body("      ", "more();", 3) + "    };\n    inner();\n" + body("    ", "work();", 4) +
				"  };\n  mid();\n}\n",
			outer: "outer", mid: "outer.mid", inner: "outer.mid.inner",
			outerLOC: 3, midLOC: 57, innerLOC: 5,
		},
		{
			name: "python", path: "a.py",
			src: "def outer():\n    def mid():\n" + body("        ", "work()", 50) +
				"        def inner():\n" + body("            ", "more()", 3) + "        inner()\n" + body("        ", "work()", 4) +
				"    mid()\n",
			outer: "outer", mid: "outer.mid", inner: "outer.mid.inner",
			// outer: def and mid(). mid: def, 50 lines, inner() and 4 lines. inner:
			// def and 3 lines.
			outerLOC: 2, midLOC: 56, innerLOC: 4,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := analyzeSource(t, tc.path, []byte(tc.src), health.Config{})
			outer, mid, inner := function(t, res, tc.outer), function(t, res, tc.mid), function(t, res, tc.inner)
			if outer.LOC != tc.outerLOC || mid.LOC != tc.midLOC || inner.LOC != tc.innerLOC {
				t.Errorf("loc outer %d mid %d inner %d, want %d %d %d", outer.LOC, mid.LOC, inner.LOC, tc.outerLOC, tc.midLOC, tc.innerLOC)
			}
			if outer.LOC > 15 {
				t.Errorf("outer has %d lines of its own, want a small function", outer.LOC)
			}
			for _, finding := range outer.Findings {
				if finding.Code == health.CodeLargeMethod || finding.Code == health.CodeBrainMethod {
					t.Errorf("%s: %s although only %d lines are its own", outer.Name, finding.Code, outer.LOC)
				}
			}
			// The lines of the units add up to the lines of the file (the Go file
			// also holds its package clause): nothing is counted twice or lost.
			sum := outer.LOC + mid.LOC + inner.LOC
			if tc.name == "go" {
				sum++
			}
			if sum != res.LOC {
				t.Errorf("the units hold %d lines, the file %d", sum, res.LOC)
			}
		})
	}
}

// Units that share a row, and a unit that ends where the next begins, are
// subtracted once.
//
// TRLC-LINKS: REQ-EMG-150
func TestNestedUnitsOnOneRowAreSubtractedOnce(t *testing.T) {
	src := "function outer() {\n  const a = () => 1, b = () => 2;\n  const c = () => {\n    return 3;\n  };\n  return a() + b() + c();\n}\n"
	res := analyzeSource(t, "a.js", []byte(src), health.Config{})
	if got := function(t, res, "outer").LOC; got != 3 {
		t.Errorf("outer: loc %d, want 3 (header, return, closing brace)", got)
	}
	if got := function(t, res, "outer.c").LOC; got != 3 {
		t.Errorf("outer.c: loc %d, want 3", got)
	}
}

// A file the parser only partly read carries the count of its ERROR and MISSING
// nodes, so it never reads as clean (Q5 of the second review pass); a clean file
// has none, and the count stops at the cap.
// Mutation: not counting leaves SyntaxErrors zero for every broken source.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestSyntaxErrorsAreCounted(t *testing.T) {
	cases := []struct{ name, path, clean, broken string }{
		{"go", "a.go", "package p\n\nfunc F() int { return 1 }\n", "package p\n\nfunc F() int { return 1 }\n\nfunc G( {\n\tif x {\n"},
		{"rust", "a.rs", "fn f() -> i32 { 1 }\n", "fn f() -> i32 { 1 }\n\nfn g( {\n    if x {\n"},
		{"typescript", "a.ts", "function f(): number { return 1; }\n", "function f(): number { return 1; }\n\nfunction g( {\n  if (x) {\n"},
		{"tsx", "a.tsx", "export const A = () => <div />;\n", "export const A = () => <div />;\n\nfunction g( {\n  if (x) {\n"},
		{"javascript", "a.js", "function f() { return 1; }\n", "function f() { return 1; }\n\nfunction g( {\n  if (x) {\n"},
		{"python", "a.py", "def f():\n    return 1\n", "def f():\n    return 1\n\ndef g(:\n    if x\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := analyzeSource(t, tc.path, []byte(tc.clean), health.Config{}).SyntaxErrors; got != 0 {
				t.Errorf("clean source: %d syntax errors", got)
			}
			res := analyzeSource(t, tc.path, []byte(tc.broken), health.Config{})
			if res.SyntaxErrors == 0 {
				t.Errorf("broken source reads as clean: %+v", res)
			}
			if len(res.Functions) == 0 {
				t.Errorf("the function before the error should still be measured: %+v", res.Functions)
			}
		})
	}
	many := strings.Repeat("func ( {\n", 3*health.SyntaxErrorCap)
	if got := analyzeSource(t, "a.go", []byte("package p\n\n"+many), health.Config{}).SyntaxErrors; got < 1 || got > health.SyntaxErrorCap {
		t.Errorf("the count %d must stay within the cap %d", got, health.SyntaxErrorCap)
	}
}
