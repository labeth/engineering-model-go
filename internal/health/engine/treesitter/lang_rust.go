// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/labeth/engineering-model-go/internal/codemap"
)

// rustLanguage: decision points are if (if let included: a let_condition or
// let_chain condition), while (while let included), for and every match arm
// beyond the first; else-if is an if_expression inside an else_clause. The ?
// operator and loop are not decision points; loop still nests. Macro bodies
// such as assert! are opaque token trees: nothing inside them is counted.
var rustLanguage = &language{
	name: "rust",
	rules: map[string]rule{
		"if_expression":     {decision: true, nest: true},
		"while_expression":  {decision: true, nest: true},
		"for_expression":    {decision: true, nest: true},
		"loop_expression":   {nest: true},
		"match_expression":  {nest: true},
		"match_arm":         {extra: true},
		"binary_expression": {boolOp: true},
		"let_chain":         {boolOp: true},
	},
	unitKinds:  map[string]bool{"function_item": true, "closure_expression": true},
	unit:       rustUnit,
	scopeKinds: map[string]bool{"impl_item": true, "trait_item": true},
	scope:      rustScope,
	params:     rustParams,
	chained: func(n *sitter.Node) bool {
		return n.Kind() == "if_expression" && n.Parent() != nil && n.Parent().Kind() == "else_clause"
	},
	extra:       rustMatchArm,
	boolOps:     rustBoolOps,
	transparent: map[string]bool{"parenthesized_expression": true, "unary_expression": true},
	containers:  map[string]bool{"block": true},
	assertion:   rustAssertion,
	fileIsTest: func(c *fileCtx, root *sitter.Node) bool {
		return pathHasDir(c.path, "tests") || codemap.IsRustInTestScope(root, c.src)
	},
	primitives: primitiveSupported,
}

// TRLC-LINKS: REQ-EMG-150
func rustUnit(c *fileCtx, n *sitter.Node) (unitInfo, bool) {
	body := n.ChildByFieldName("body")
	if body == nil {
		return unitInfo{}, false
	}
	if n.Kind() == "closure_expression" {
		return rustClosureUnit(c, n, body)
	}
	return unitInfo{name: text(n.ChildByFieldName("name"), c.src), kind: rustFunctionKind(n), body: body, test: codemap.IsRustTestFunction(n, c.src)}, true
}

// rustClosureUnit names a closure that is bound by a let declaration; any
// other closure counts toward its enclosing function.
// TRLC-LINKS: REQ-EMG-150
func rustClosureUnit(c *fileCtx, n, body *sitter.Node) (unitInfo, bool) {
	var pattern *sitter.Node
	if decl := n.Parent(); decl != nil && decl.Kind() == "let_declaration" && isFieldOf(n, "value") {
		pattern = decl.ChildByFieldName("pattern")
	}
	if pattern == nil || pattern.Kind() != "identifier" {
		return unitInfo{}, false
	}
	return unitInfo{name: text(pattern, c.src), kind: "function", body: body}, true
}

// rustFunctionKind is "method" for a function declared in an impl or a trait
// and "function" for any other.
// TRLC-LINKS: REQ-EMG-150
func rustFunctionKind(n *sitter.Node) string {
	list := n.Parent()
	if list == nil || list.Kind() != "declaration_list" || list.Parent() == nil {
		return "function"
	}
	if owner := list.Parent().Kind(); owner == "impl_item" || owner == "trait_item" {
		return "method"
	}
	return "function"
}

// rustScope names the type an impl block or trait adds to its functions.
// TRLC-LINKS: REQ-EMG-150
func rustScope(c *fileCtx, n *sitter.Node) string {
	if n.Kind() == "trait_item" {
		return text(n.ChildByFieldName("name"), c.src)
	}
	name := strings.TrimSpace(text(n.ChildByFieldName("type"), c.src))
	if i := strings.Index(name, "<"); i >= 0 {
		name = name[:i]
	}
	return strings.TrimLeft(name, "&* ")
}

// rustMatchArm adds a decision point for every match arm that has an earlier
// arm in the same match; the guard of an arm adds none.
// TRLC-LINKS: REQ-EMG-150
func rustMatchArm(_ *fileCtx, n *sitter.Node) int {
	if n.Kind() != "match_arm" {
		return 0
	}
	for prev := n.PrevNamedSibling(); prev != nil; prev = prev.PrevNamedSibling() {
		if prev.Kind() == "match_arm" {
			return 1
		}
	}
	return 0
}

// rustBoolOps counts && and || in binary expressions and the && tokens of a
// let chain (if let Some(x) = a && b).
// TRLC-LINKS: REQ-EMG-150
func rustBoolOps(c *fileCtx, n *sitter.Node) int {
	if n.Kind() == "let_chain" {
		return countChildKind(n, "&&")
	}
	return binaryBoolOps("&&", "||")(c, n)
}

// rustParams counts parameter and variadic_parameter children of a function and
// every closure parameter; self is not a parameter.
// TRLC-LINKS: REQ-EMG-150
func rustParams(c *fileCtx, n *sitter.Node, _ bool) paramStats {
	var ps paramStats
	list := n.ChildByFieldName("parameters")
	if list == nil {
		return ps
	}
	closure := n.Kind() == "closure_expression"
	for _, param := range namedChildren(list) {
		ps.addRustParam(c, param, closure)
	}
	return ps
}

// addRustParam counts one parameter of a function or closure.
// TRLC-LINKS: REQ-EMG-150
func (ps *paramStats) addRustParam(c *fileCtx, param *sitter.Node, closure bool) {
	switch param.Kind() {
	case "self_parameter", "attribute_item":
	case "parameter":
		ps.count++
		ps.addRustType(c, param)
	case "variadic_parameter":
		ps.count++
	default:
		if closure {
			ps.count++
		}
	}
}

// addRustType notes the annotation of a parameter.
// TRLC-LINKS: REQ-EMG-150
func (ps *paramStats) addRustType(c *fileCtx, param *sitter.Node) {
	t := param.ChildByFieldName("type")
	if t == nil {
		return
	}
	ps.annotated = true
	if rustPrimitive(c, t) {
		ps.primitive++
	}
}

// rustPrimitive: a primitive_type (bool, char, numbers, str), String, or a
// reference to one of them.
// TRLC-LINKS: REQ-EMG-150
func rustPrimitive(c *fileCtx, t *sitter.Node) bool {
	for t != nil && t.Kind() == "reference_type" {
		t = t.ChildByFieldName("type")
	}
	if t == nil {
		return false
	}
	return t.Kind() == "primitive_type" || (t.Kind() == "type_identifier" && text(t, c.src) == "String")
}

var rustAssertMacros = map[string]bool{
	"assert": true, "assert_eq": true, "assert_ne": true,
	"debug_assert": true, "debug_assert_eq": true, "debug_assert_ne": true,
}

// rustAssertion: a statement that is an assert!-family macro call, however the
// macro path is spelled (std::assert!, pretty_assertions::assert_eq!).
// TRLC-LINKS: REQ-EMG-150
func rustAssertion(c *fileCtx, n *sitter.Node) bool {
	if n.Kind() == "expression_statement" && n.NamedChildCount() == 1 {
		n = n.NamedChild(0)
	}
	if n.Kind() != "macro_invocation" {
		return false
	}
	macro := n.ChildByFieldName("macro")
	if macro == nil {
		return false
	}
	if macro.Kind() == "scoped_identifier" {
		macro = macro.ChildByFieldName("name")
	}
	return rustAssertMacros[text(macro, c.src)]
}
