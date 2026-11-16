// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-010
func TestVerilogModuleTrace(t *testing.T) {
	source := "/* module ghost(); endmodule */\n// TRLC-LINKS: REQ-RTL-001\nmodule top #(parameter N=1) (input clk);\n initial $display(\"module fake(); endmodule // TRLC-LINKS: REQ-FAKE-001\");\nendmodule\n// TRLC-LINKS: REQ-RTL-002\nmodule \\helper.name (input clk);\nendmodule\nmodule unlinked;\nendmodule\n"
	symbols, diags := scanSource(t, "top.v", source)
	requireEqual(t, len(symbols), 2)
	requireEqual(t, []int{symbols[0].Line, symbols[1].Line}, []int{3, 7})
	requireEqual(t, []string{symbols[0].Implements[0], symbols[1].Signature}, []string{"REQ-RTL-001", "module \\helper.name"})
	requireEqual(t, len(diags), 1)
	requireEqual(t, []string{diags[0].Code, diags[0].Path}, []string{"code.missing_trlc_link", "top.v:9"})
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogExpressionMacrosPreserveBoundaries(t *testing.T) {
	source := "module lanes;\n localparam PAD = `LANEMAP_ENTRY(3);\n assign value = lane[`INDEX2(3)];\n assign mask = 80'b1 << `LANEMAP_ENTRY(3); endmodule\nmodule next; endmodule\n"
	decls, _, diags := extractVerilog(t, "lanes.v", source)
	requireEqual(t, declarationNames(decls), []string{"lanes", "next"})
	requireEqual(t, len(diags), 3)
	requireExpressionMacros(t, diags)
	requireEqual(t, strings.Contains(diags[1].Message, "INDEX2"), true)
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogDeclarationMacrosRemainErrors(t *testing.T) {
	for _, source := range []string{"`MAKE_MODULE(foo)", "module `NAME; endmodule", "module m;\n`BODY\nendmodule", "localparam X = `VALUE;", "module m; endmodule\n`NEXT_MODULE"} {
		_, _, diags := extractVerilog(t, "macro.v", source)
		requireErrorDiag(t, diags, "code.verilog_parse_error")
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogLexicalFailures(t *testing.T) {
	for _, source := range []string{"/* unterminated", "module m; initial $display(\"oops); endmodule", "module m;", "endmodule", "module m; module n; endmodule", "module #(); endmodule", "`MAKE_MODULE(foo)", "module endmodule", "module m; endmodule module n; endmodule"} {
		t.Run(source, func(t *testing.T) {
			_, _, diags := extractVerilog(t, "bad.v", source)
			requireEqual(t, len(diags) > 0, true)
		})
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogDirectiveAndCommentPositions(t *testing.T) {
	source := "`define MODULE_TEXT module ghost; \\\nendmodule\n/* block\n comment */\n// TRLC-LINKS: REQ-RTL-001\nmodule\n real_name\n(input clk);\nendmodule\n"
	decls, comments, diags := extractVerilog(t, "ok.v", source)
	requireEqual(t, len(diags), 0)
	requireEqual(t, declarationNames(decls), []string{"real_name"})
	requireEqual(t, decls[0].Line, 6)
	requireEqual(t, comments, map[int]bool{3: true, 4: true, 5: true})
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogSingleFileRoot(t *testing.T) {
	root := writeTree(t, map[string]string{
		"capture.v":    "// TRLC-LINKS: REQ-CAP-001\nmodule capture; endmodule\n",
		"unselected.v": "module unlinked; endmodule\n",
	})
	symbols, diags := scanRoot(t, filepath.Join(root, "capture.v"))
	requireNoDiags(t, diags)
	requireEqual(t, len(symbols), 1)
	requireEqual(t, symbols[0].Path, "capture.v")
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogExpressionMacroContexts(t *testing.T) {
	source := "module m;\n reg [`W-1:0] x = {`W{1'b0}};\n wire z = (`A + `B) & `MASK;\n initial begin write_reg(`SEL, `VALUE); case(x)\n `CHOICE: x = x[0:`LAST];\n endcase end\nendmodule\nmodule next; endmodule\n"
	decls, _, diags := extractVerilog(t, "contexts.v", source)
	requireEqual(t, declarationNames(decls)[1:], []string{"next"})
	requireEqual(t, len(decls), 2)
	requireEqual(t, len(diags), 9)
	requireExpressionMacros(t, diags)
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogMacroPortAndBodyDeclarationsRemainErrors(t *testing.T) {
	for _, source := range []string{"module m (`PORTS); endmodule", "module m; wire a; `DECL\nendmodule", "module m; initial begin `STATEMENTS\nend endmodule"} {
		_, _, diags := extractVerilog(t, "ambiguous.v", source)
		requireDiag(t, diags, "code.verilog_parse_error")
	}
}

// extractVerilog extracts the declarations of a Verilog source and fails the
// test on an error.
func extractVerilog(t *testing.T, name, source string) ([]declaration, map[int]bool, []validate.Diagnostic) {
	t.Helper()
	decls, comments, diags, err := extractVerilogDeclarations(name, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return decls, comments, diags
}

// requireExpressionMacros fails the test unless every diagnostic is the
// warning of an unexpanded expression macro.
func requireExpressionMacros(t *testing.T, diags []validate.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		requireEqual(t, []string{d.Code, string(d.Severity)}, []string{"code.verilog_expression_macro", string(validate.SeverityWarning)})
	}
}
