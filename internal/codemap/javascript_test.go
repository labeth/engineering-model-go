// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"os"
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-010
func TestJavaScriptTraceDeclarationsAndLiteralIsolation(t *testing.T) {
	source := `// ENGMODEL-OWNER-UNIT: FU-WEB
// TRLC-LINKS: REQ-WEB-001
export async function poll() { return 1; }
// TRLC-LINKS: REQ-WEB-001
function* samples() { yield 1; }
// TRLC-LINKS: REQ-WEB-001
const $ = id => id;
// TRLC-LINKS: REQ-WEB-001
let bound = function internalName() { return 2; };
// TRLC-LINKS: REQ-WEB-001
window.resize = () => 3;
const object = {
 // TRLC-LINKS: REQ-WEB-001
 callback: function () { return 4; },
 // TRLC-LINKS: REQ-WEB-001
 method() { return 5; }
};
// ENGMODEL-LINKS: FU-WEB
class View {
 // TRLC-LINKS: REQ-WEB-001
 render() { return 6; }
}
// TRLC-LINKS: REQ-WEB-001
(function bootstrap() {})();
const fake = "// TRLC-LINKS: REQ-FAKE-001 function fake() {}";
const regex = /function fakeRegex\(\)/;
// function commentedOut() {}
setTimeout(() => {}, 1);
`
	want := []string{"poll", "samples", "$", "bound", "window.resize", "callback", "method", "View", "render", "bootstrap"}
	for _, ext := range []string{".js", ".mjs", ".cjs"} {
		t.Run(ext, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "browser"+ext)
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			decls, _, diags, err := extractDeclarations(path, []byte(source))
			if err != nil || len(diags) != 0 {
				t.Fatalf("extract: %v %+v", err, diags)
			}
			if len(decls) != len(want) {
				t.Fatalf("declarations=%+v", decls)
			}
			for i, d := range decls {
				if d.Name != want[i] {
					t.Fatalf("declaration %d=%+v want %s", i, d, want[i])
				}
			}
			symbols, diags, err := Scan(path)
			if err != nil || len(diags) != 0 || len(symbols) != len(want) {
				t.Fatalf("scan: %v %+v %+v", err, diags, symbols)
			}
			for _, s := range symbols {
				for _, r := range s.Implements {
					if r != "REQ-WEB-001" {
						t.Fatalf("literal marker escaped: %+v", s)
					}
				}
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestJavaScriptTraceFailsClosedOnMissingSyntaxAndAmbiguousLines(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"missing", "const callback = () => 1;", "code.missing_trlc_link"},
		{"syntax", "function broken( {", "code.parse_error"},
		{"same line", "// TRLC-LINKS: REQ-WEB-001\nfunction a() {} function b() {}", "code.ambiguous_declaration_line"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sample.js")
			if err := os.WriteFile(path, []byte(tc.source), 0600); err != nil {
				t.Fatal(err)
			}
			_, diags, err := Scan(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range diags {
				if d.Code == tc.code && string(d.Severity) == "error" {
					return
				}
			}
			t.Fatalf("missing %s: %+v", tc.code, diags)
		})
	}
}
