// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/validate"
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
	name, required := javascriptDeclaredName(n)
	if name == nil {
		return declaration{}, false
	}
	return declaration{Name: strings.TrimSpace(name.Utf8Text(src)), Kind: n.Kind(), Line: int(n.StartPosition().Row) + 1, Signature: firstLine(strings.TrimSpace(n.Utf8Text(src))), RequiresTRLC: required}, true
}

// javascriptDeclaredName is the name node of the declaration n makes, or nil
// when n declares nothing, and whether the declaration requires TRLC-LINKS.
// TRLC-LINKS: REQ-EMG-010
func javascriptDeclaredName(n *sitter.Node) (*sitter.Node, bool) {
	switch n.Kind() {
	case "function_declaration", "generator_function_declaration", "method_definition":
		return n.ChildByFieldName("name"), true
	case "class_declaration":
		return n.ChildByFieldName("name"), false
	case "variable_declarator":
		return javascriptIdentifier(javascriptBoundName(n, "value", "name")), true
	case "assignment_expression":
		return javascriptBoundName(n, "right", "left"), true
	case "pair":
		return javascriptBoundName(n, "value", "key"), true
	case "function_expression", "generator_function":
		return javascriptExpressionName(n), true
	}
	return nil, true
}

// javascriptBoundName is the name node of a binding whose value is a
// function, or nil.
// TRLC-LINKS: REQ-EMG-010
func javascriptBoundName(n *sitter.Node, valueField, nameField string) *sitter.Node {
	if !javascriptFunctionValue(n.ChildByFieldName(valueField)) {
		return nil
	}
	return n.ChildByFieldName(nameField)
}

// javascriptIdentifier is name when it is an identifier, else nil.
// TRLC-LINKS: REQ-EMG-010
func javascriptIdentifier(name *sitter.Node) *sitter.Node {
	if name == nil || name.Kind() != "identifier" {
		return nil
	}
	return name
}

// javascriptExpressionName is the name of a function expression, or nil when
// a direct binding already carries the declaration's externally visible name.
// TRLC-LINKS: REQ-EMG-010
func javascriptExpressionName(n *sitter.Node) *sitter.Node {
	if p := n.Parent(); p != nil {
		switch p.Kind() {
		case "variable_declarator", "assignment_expression", "pair":
			return nil
		}
	}
	return n.ChildByFieldName("name")
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
