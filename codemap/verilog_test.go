// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-010
func TestVerilogModuleTrace(t *testing.T) {
	root := t.TempDir()
	source := "/* module ghost(); endmodule */\n// TRLC-LINKS: REQ-RTL-001\nmodule top #(parameter N=1) (input clk);\n initial $display(\"module fake(); endmodule // TRLC-LINKS: REQ-FAKE-001\");\nendmodule\n// TRLC-LINKS: REQ-RTL-002\nmodule \\helper.name (input clk);\nendmodule\nmodule unlinked;\nendmodule\n"
	if err := os.WriteFile(filepath.Join(root, "top.v"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	symbols, diags, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 2 || symbols[0].Line != 3 || symbols[1].Line != 7 || symbols[0].Implements[0] != "REQ-RTL-001" || symbols[1].Signature != "module \\helper.name" {
		t.Fatalf("unexpected symbols: %+v", symbols)
	}
	if len(diags) != 1 || diags[0].Code != "code.missing_trlc_link" || !strings.HasSuffix(diags[0].Path, ":9") {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogExpressionMacrosPreserveBoundaries(t *testing.T) {
	source := "module lanes;\n localparam PAD = `LANEMAP_ENTRY(3);\n assign value = lane[`INDEX2(3)];\n assign mask = 80'b1 << `LANEMAP_ENTRY(3); endmodule\nmodule next; endmodule\n"
	decls, _, diags, err := extractVerilogDeclarations("lanes.v", []byte(source))
	if err != nil || len(decls) != 2 || decls[0].Name != "lanes" || decls[1].Name != "next" || len(diags) != 3 {
		t.Fatalf("lost module boundary: %+v %+v %v", decls, diags, err)
	}
	for i, diagnostic := range diags {
		if diagnostic.Code != "code.verilog_expression_macro" || diagnostic.Severity != validate.SeverityWarning {
			t.Fatalf("expression expansion must remain explicit: %+v", diagnostic)
		}
		if i == 1 && !strings.Contains(diagnostic.Message, "INDEX2") {
			t.Fatalf("macro identifier was truncated: %+v", diagnostic)
		}
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogDeclarationMacrosRemainErrors(t *testing.T) {
	for _, source := range []string{"`MAKE_MODULE(foo)", "module `NAME; endmodule", "module m;\n`BODY\nendmodule", "localparam X = `VALUE;", "module m; endmodule\n`NEXT_MODULE"} {
		_, _, diags, _ := extractVerilogDeclarations("macro.v", []byte(source))
		found := false
		for _, diagnostic := range diags {
			found = found || diagnostic.Code == "code.verilog_parse_error" && diagnostic.Severity == validate.SeverityError
		}
		if !found {
			t.Fatalf("declaration ambiguity accepted: %s: %+v", source, diags)
		}
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogLexicalFailures(t *testing.T) {
	for _, source := range []string{"/* unterminated", "module m; initial $display(\"oops); endmodule", "module m;", "endmodule", "module m; module n; endmodule", "module #(); endmodule", "`MAKE_MODULE(foo)", "module endmodule", "module m; endmodule module n; endmodule"} {
		t.Run(source, func(t *testing.T) {
			_, _, diags, err := extractVerilogDeclarations("bad.v", []byte(source))
			if err != nil || len(diags) == 0 {
				t.Fatalf("expected diagnostic: %v %+v", err, diags)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogDirectiveAndCommentPositions(t *testing.T) {
	source := "`define MODULE_TEXT module ghost; \\\nendmodule\n/* block\n comment */\n// TRLC-LINKS: REQ-RTL-001\nmodule\n real_name\n(input clk);\nendmodule\n"
	decls, comments, diags, err := extractVerilogDeclarations("ok.v", []byte(source))
	if err != nil || len(diags) != 0 || len(decls) != 1 || decls[0].Name != "real_name" || decls[0].Line != 6 || !comments[3] || !comments[4] || !comments[5] || comments[1] {
		t.Fatalf("bad lexical result: %+v %+v %+v %v", decls, comments, diags, err)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogSingleFileRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "capture.v")
	if err := os.WriteFile(path, []byte("// TRLC-LINKS: REQ-CAP-001\nmodule capture; endmodule\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unselected.v"), []byte("module unlinked; endmodule\n"), 0600); err != nil {
		t.Fatal(err)
	}
	symbols, diags, err := Scan(path)
	if err != nil || len(diags) != 0 || len(symbols) != 1 || symbols[0].Path != "capture.v" {
		t.Fatalf("bad single-file scan: %+v %+v %v", symbols, diags, err)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogExpressionMacroContexts(t *testing.T) {
	source := "module m;\n reg [`W-1:0] x = {`W{1'b0}};\n wire z = (`A + `B) & `MASK;\n initial begin write_reg(`SEL, `VALUE); case(x)\n `CHOICE: x = x[0:`LAST];\n endcase end\nendmodule\nmodule next; endmodule\n"
	decls, _, diags, err := extractVerilogDeclarations("contexts.v", []byte(source))
	if err != nil || len(decls) != 2 || decls[1].Name != "next" || len(diags) != 9 {
		t.Fatalf("unexpected extraction: %+v %+v %v", decls, diags, err)
	}
	for _, d := range diags {
		if d.Code != "code.verilog_expression_macro" || d.Severity != validate.SeverityWarning {
			t.Fatalf("unexpected diagnostic: %+v", d)
		}
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogMacroPortAndBodyDeclarationsRemainErrors(t *testing.T) {
	for _, source := range []string{"module m (`PORTS); endmodule", "module m; wire a; `DECL\nendmodule", "module m; initial begin `STATEMENTS\nend endmodule"} {
		_, _, diags, _ := extractVerilogDeclarations("ambiguous.v", []byte(source))
		found := false
		for _, d := range diags {
			found = found || d.Code == "code.verilog_parse_error"
		}
		if !found {
			t.Fatalf("accepted ambiguous declaration: %s", source)
		}
	}
}
