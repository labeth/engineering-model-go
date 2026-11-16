// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"slices"
	"testing"
)

const javascriptSource = `// ENGMODEL-OWNER-UNIT: FU-WEB
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

var javascriptNames = []string{"poll", "samples", "$", "bound", "window.resize", "callback", "method", "View", "render", "bootstrap"}

// TRLC-LINKS: REQ-EMG-010
func TestJavaScriptTraceDeclarationsAndLiteralIsolation(t *testing.T) {
	for _, ext := range []string{".js", ".mjs", ".cjs"} {
		t.Run(ext, func(t *testing.T) { checkJavaScriptSource(t, "browser"+ext) })
	}
}

// checkJavaScriptSource scans the sample source written under name: it
// declares the sample names, links only REQ-WEB-001, and no literal links.
func checkJavaScriptSource(t *testing.T, name string) {
	t.Helper()
	path := writeSource(t, name, javascriptSource)
	decls, _, diags, err := extractDeclarations(path, []byte(javascriptSource))
	if err != nil {
		t.Fatal(err)
	}
	requireNoDiags(t, diags)
	requireEqual(t, declarationNames(decls), javascriptNames)
	symbols, diags := scanRoot(t, path)
	requireNoDiags(t, diags)
	requireEqual(t, len(symbols), len(javascriptNames))
	linked := slices.DeleteFunc(implementedBy(symbols), func(req string) bool { return req == "REQ-WEB-001" })
	requireEqual(t, len(linked), 0)
}

// TRLC-LINKS: REQ-EMG-010
func TestJavaScriptTraceFailsClosedOnMissingSyntaxAndAmbiguousLines(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"missing", "const callback = () => 1;", "code.missing_trlc_link"},
		{"syntax", "function broken( {", "code.parse_error"},
		{"same line", "// TRLC-LINKS: REQ-WEB-001\nfunction a() {} function b() {}", "code.ambiguous_declaration_line"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := scanRoot(t, writeSource(t, "sample.js", tc.source))
			requireErrorDiag(t, diags, tc.code)
		})
	}
}
