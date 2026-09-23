// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"os"
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-010
func TestPythonDecoratorsScopesAndLiteralIsolation(t *testing.T) {
	source := `# TRLC-LINKS: REQ-PY-001
@decorate(
    """# TRLC-LINKS: REQ-FAKE-999""",
    other=True,
)
async def outer():
    # TRLC-LINKS: REQ-PY-002
    def inner():
        return lambda: 1
    return inner
# ENGMODEL-LINKS: FU-PY
class Reader:
    # TRLC-LINKS: REQ-PY-003
    @property
    def value(self):
        return 1
class Other:
    # TRLC-LINKS: REQ-PY-004
    def value(self): return 2
text = """
# TRLC-LINKS: REQ-FAKE-999
def fake(): pass
"""
# TRLC-LINKS: REQ-PY-005
def final(): pass
`
	root := t.TempDir()
	path := filepath.Join(root, "source.py")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	symbols, diags, err := Scan(path)
	if err != nil || len(diags) != 0 {
		t.Fatalf("scan: %v %+v", err, diags)
	}
	want := map[string]int{"CODE-OUTER": 6, "CODE-OUTERINNER": 8, "CODE-READER": 12, "CODE-READERVALUE": 15, "CODE-OTHERVALUE": 19, "CODE-FINAL": 25}
	if len(symbols) != len(want) {
		t.Fatalf("symbols: %+v", symbols)
	}
	for _, symbol := range symbols {
		if want[symbol.TraceID] != symbol.Line {
			t.Fatalf("wrong declaration: %+v", symbol)
		}
		for _, req := range symbol.Implements {
			if req == "REQ-FAKE-999" {
				t.Fatalf("string created trace: %+v", symbol)
			}
		}
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestPythonMissingLinksAndSyntaxErrors(t *testing.T) {
	for _, sample := range []struct{ name, source, code string }{
		{"literal", "text = '''\n# TRLC-LINKS: REQ-PY-001''' # genuine trailing comment\ndef unlinked(): pass\n", "code.missing_trlc_link"},
		{"broken", "# TRLC-LINKS: REQ-PY-001\ndef broken(:\n", "code.parse_error"},
		{"nested", "# TRLC-LINKS: REQ-PY-001\ndef outer():\n    def inner(): pass\n", "code.missing_trlc_link"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "source.py")
			if err := os.WriteFile(path, []byte(sample.source), 0600); err != nil {
				t.Fatal(err)
			}
			_, diags, err := Scan(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range diags {
				if d.Code == sample.code && string(d.Severity) == "error" {
					return
				}
			}
			t.Fatalf("missing %s: %+v", sample.code, diags)
		})
	}
}
