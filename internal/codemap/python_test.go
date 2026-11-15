// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"slices"
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
	symbols, diags := scanRoot(t, writeSource(t, "source.py", source))
	requireNoDiags(t, diags)
	got := map[string]int{}
	for _, symbol := range symbols {
		got[symbol.TraceID] = symbol.Line
	}
	requireEqual(t, got, map[string]int{"CODE-SOURCE-PY--OUTER": 6, "CODE-SOURCE-PY--OUTERINNER": 8, "CODE-SOURCE-PY--READER": 12, "CODE-SOURCE-PY--READERVALUE": 15, "CODE-SOURCE-PY--OTHERVALUE": 19, "CODE-SOURCE-PY--FINAL": 25})
	requireEqual(t, slices.Contains(implementedBy(symbols), "REQ-FAKE-999"), false)
}

// TRLC-LINKS: REQ-EMG-010
func TestPythonMissingLinksAndSyntaxErrors(t *testing.T) {
	for _, sample := range []struct{ name, source, code string }{
		{"literal", "text = '''\n# TRLC-LINKS: REQ-PY-001''' # genuine trailing comment\ndef unlinked(): pass\n", "code.missing_trlc_link"},
		{"broken", "# TRLC-LINKS: REQ-PY-001\ndef broken(:\n", "code.parse_error"},
		{"nested", "# TRLC-LINKS: REQ-PY-001\ndef outer():\n    def inner(): pass\n", "code.missing_trlc_link"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			_, diags := scanRoot(t, writeSource(t, "source.py", sample.source))
			requireErrorDiag(t, diags, sample.code)
		})
	}
}
