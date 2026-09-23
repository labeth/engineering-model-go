// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/validate"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// TRLC-LINKS: REQ-EMG-010
func isJavaScript(ext string) bool { return ext == ".js" || ext == ".mjs" || ext == ".cjs" }

// TRLC-LINKS: REQ-EMG-010
func javascriptFunctionValue(n *sitter.Node) bool {
	if n == nil {
		return false
	}
	switch n.Kind() {
	case "arrow_function", "function_expression", "generator_function":
		return true
	}
	return false
}

// TRLC-LINKS: REQ-EMG-010
func javascriptDeclaration(n *sitter.Node, src []byte) (declaration, bool) {
	if n == nil || !n.IsNamed() {
		return declaration{}, false
	}
	var name *sitter.Node
	required := true
	switch n.Kind() {
	case "function_declaration", "generator_function_declaration", "method_definition":
		name = n.ChildByFieldName("name")
	case "class_declaration":
		name = n.ChildByFieldName("name")
		required = false
	case "variable_declarator":
		if !javascriptFunctionValue(n.ChildByFieldName("value")) {
			return declaration{}, false
		}
		name = n.ChildByFieldName("name")
		if name == nil || name.Kind() != "identifier" {
			return declaration{}, false
		}
	case "assignment_expression":
		if !javascriptFunctionValue(n.ChildByFieldName("right")) {
			return declaration{}, false
		}
		name = n.ChildByFieldName("left")
	case "pair":
		if !javascriptFunctionValue(n.ChildByFieldName("value")) {
			return declaration{}, false
		}
		name = n.ChildByFieldName("key")
	case "function_expression", "generator_function":
		// Direct bindings already carry the declaration's externally visible name.
		if p := n.Parent(); p != nil {
			switch p.Kind() {
			case "variable_declarator", "assignment_expression", "pair":
				return declaration{}, false
			}
		}
		name = n.ChildByFieldName("name")
	default:
		return declaration{}, false
	}
	if name == nil {
		return declaration{}, false
	}
	return declaration{Name: strings.TrimSpace(name.Utf8Text(src)), Kind: n.Kind(), Line: int(n.StartPosition().Row) + 1, Signature: firstLine(strings.TrimSpace(n.Utf8Text(src))), RequiresTRLC: required}, true
}

// TRLC-LINKS: REQ-EMG-010
func javascriptDiagnostics(path string, root *sitter.Node, decls []declaration) []validate.Diagnostic {
	var diags []validate.Diagnostic
	if root.HasError() {
		diags = append(diags, validate.Diagnostic{Code: "code.parse_error", Severity: validate.SeverityError, Path: path, Message: "JavaScript syntax tree contains errors; declaration coverage is incomplete"})
	}
	counts := map[int]int{}
	for _, d := range decls {
		counts[d.Line]++
	}
	for _, d := range decls {
		if counts[d.Line] > 1 {
			diags = append(diags, validate.Diagnostic{Code: "code.ambiguous_declaration_line", Severity: validate.SeverityError, Path: fmt.Sprintf("%s:%d", path, d.Line), Message: "multiple JavaScript declarations share a line; separate them before attaching trace links"})
			counts[d.Line] = 0
		}
	}
	return diags
}
