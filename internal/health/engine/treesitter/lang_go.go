// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/labeth/engineering-model-go/internal/codemap"
)

// goLanguage: decision points are if, for (range included) and the cases of
// expression switches, type switches and selects; default_case is not one.
// An else-if is a nested if_statement in the alternative field of the first.
var goLanguage = &language{
	name: "go",
	rules: map[string]rule{
		"if_statement":                {decision: true, nest: true},
		"for_statement":               {decision: true, nest: true},
		"expression_switch_statement": {nest: true},
		"type_switch_statement":       {nest: true},
		"select_statement":            {nest: true},
		"expression_case":             {decision: true},
		"type_case":                   {decision: true},
		"communication_case":          {decision: true},
		"binary_expression":           {boolOp: true},
	},
	unitKinds: map[string]bool{"function_declaration": true, "method_declaration": true, "func_literal": true},
	unit:      goUnit,
	params:    goParams,
	chained: func(n *sitter.Node) bool {
		return n.Kind() == "if_statement" && n.Parent() != nil && n.Parent().Kind() == "if_statement"
	},
	boolOps:     binaryBoolOps("&&", "||"),
	transparent: map[string]bool{"parenthesized_expression": true, "unary_expression": true},
	containers:  map[string]bool{"statement_list": true},
	assertion:   goAssertion,
	fileIsTest:  func(c *fileCtx, _ *sitter.Node) bool { return codemap.IsGoTestFile(c.path) },
	primitives:  primitiveSupported,
}

// binaryBoolOps counts one boolean operator for a binary_expression whose
// operator is one of ops.
// TRLC-LINKS: REQ-EMG-150
func binaryBoolOps(ops ...string) func(*fileCtx, *sitter.Node) int {
	return func(_ *fileCtx, n *sitter.Node) int {
		if n.Kind() != "binary_expression" {
			return 0
		}
		op := operatorOf(n)
		for _, want := range ops {
			if op == want {
				return 1
			}
		}
		return 0
	}
}

// TRLC-LINKS: REQ-EMG-150
func goUnit(c *fileCtx, n *sitter.Node) (unitInfo, bool) {
	body := n.ChildByFieldName("body")
	if body == nil {
		return unitInfo{}, false
	}
	name := text(n.ChildByFieldName("name"), c.src)
	switch n.Kind() {
	case "function_declaration":
		return unitInfo{name: name, kind: "function", body: body, test: c.isTest && codemap.IsGoTestFunction(n, name)}, true
	case "method_declaration":
		return unitInfo{name: goReceiverType(c, n) + "." + name, kind: "method", body: body}, true
	}
	if assigned := goAssignedName(c, n); assigned != "" {
		return unitInfo{name: assigned, kind: "function", body: body}, true
	}
	return unitInfo{}, false
}

// goReceiverType returns the receiver type name of a method without pointer
// and type arguments.
// TRLC-LINKS: REQ-EMG-150
func goReceiverType(c *fileCtx, n *sitter.Node) string {
	recv := n.ChildByFieldName("receiver")
	if recv == nil || recv.NamedChildCount() == 0 {
		return ""
	}
	name := text(recv.NamedChild(0).ChildByFieldName("type"), c.src)
	name = strings.TrimLeft(name, "*")
	if i := strings.IndexAny(name, "[("); i >= 0 {
		name = name[:i]
	}
	return strings.TrimSpace(name)
}

// goAssignedName returns the identifier a func literal is assigned to
// (f := func..., var f = func..., f = func...), or "" for an anonymous one.
// TRLC-LINKS: REQ-EMG-150
func goAssignedName(c *fileCtx, n *sitter.Node) string {
	list := n.Parent()
	if list == nil || list.Kind() != "expression_list" || list.Parent() == nil {
		return ""
	}
	index := lastIndexOf(namedChildren(list), n)
	names := goDeclaredNames(list.Parent())
	if index < 0 || index >= len(names) || names[index].Kind() != "identifier" {
		return ""
	}
	return text(names[index], c.src)
}

// goDeclaredNames lists the names a declaration or assignment binds; nil for
// any other node.
// TRLC-LINKS: REQ-EMG-150
func goDeclaredNames(decl *sitter.Node) []*sitter.Node {
	switch decl.Kind() {
	case "short_var_declaration", "assignment_statement":
		if left := decl.ChildByFieldName("left"); left != nil {
			return namedChildren(left)
		}
	case "var_spec":
		return childrenByField(decl, "name")
	}
	return nil
}

var goPrimitiveTypes = map[string]bool{
	"string": true, "bool": true, "byte": true, "rune": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
}

// goParams counts the names of every parameter declaration (a, b int is two)
// and one for each unnamed or variadic one; the receiver is a separate field.
// TRLC-LINKS: REQ-EMG-150
func goParams(c *fileCtx, n *sitter.Node, _ bool) paramStats {
	var ps paramStats
	list := n.ChildByFieldName("parameters")
	if list == nil {
		return ps
	}
	for _, decl := range namedChildren(list) {
		ps.addGoDeclaration(c, decl)
	}
	return ps
}

// addGoDeclaration counts one parameter declaration.
// TRLC-LINKS: REQ-EMG-150
func (ps *paramStats) addGoDeclaration(c *fileCtx, decl *sitter.Node) {
	switch decl.Kind() {
	case "parameter_declaration":
		names := max(countFieldName(decl, "name"), 1)
		ps.count += names
		if goPrimitiveParam(c, decl) {
			ps.primitive += names
		}
		ps.annotated = true
	case "variadic_parameter_declaration":
		ps.count++
		ps.annotated = true
	}
}

// goPrimitiveParam reports a parameter declaration whose type is a primitive.
// TRLC-LINKS: REQ-EMG-150
func goPrimitiveParam(c *fileCtx, decl *sitter.Node) bool {
	t := decl.ChildByFieldName("type")
	return t != nil && t.Kind() == "type_identifier" && goPrimitiveTypes[text(t, c.src)]
}

// goAssertion: t.Error*/t.Fatal* calls, assert.*/require.* calls, and an if
// whose branches only call those (if got != want { t.Errorf(...) }).
// TRLC-LINKS: REQ-EMG-150
func goAssertion(c *fileCtx, n *sitter.Node) bool {
	switch n.Kind() {
	case "expression_statement":
		return goAssertionCall(c, n)
	case "if_statement":
		return goAssertionIf(c, n)
	}
	return false
}

// TRLC-LINKS: REQ-EMG-150
func goAssertionCall(c *fileCtx, stmt *sitter.Node) bool {
	operand, field, ok := goCallSelector(c, stmt)
	return ok && isGoAssertionSelector(operand, field)
}

// goCallSelector returns the operand and the field of a statement that is a
// call of operand.field(...).
// TRLC-LINKS: REQ-EMG-150
func goCallSelector(c *fileCtx, stmt *sitter.Node) (operand, field string, ok bool) {
	if stmt.NamedChildCount() != 1 || stmt.NamedChild(0).Kind() != "call_expression" {
		return "", "", false
	}
	fn := stmt.NamedChild(0).ChildByFieldName("function")
	if fn == nil || fn.Kind() != "selector_expression" {
		return "", "", false
	}
	op := fn.ChildByFieldName("operand")
	if op == nil || op.Kind() != "identifier" {
		return "", "", false
	}
	return text(op, c.src), text(fn.ChildByFieldName("field"), c.src), true
}

// isGoAssertionSelector reports assert.* and require.* and the Error and Fatal
// methods of t, tb and b.
// TRLC-LINKS: REQ-EMG-150
func isGoAssertionSelector(operand, field string) bool {
	switch operand {
	case "assert", "require":
		return true
	case "t", "tb", "b":
		return strings.HasPrefix(field, "Error") || strings.HasPrefix(field, "Fatal")
	}
	return false
}

// goAssertionIf accepts an if (with optional else block) whose blocks hold
// only assertion calls and returns.
// TRLC-LINKS: REQ-EMG-150
func goAssertionIf(c *fileCtx, n *sitter.Node) bool {
	if !goAssertionBlock(c, n.ChildByFieldName("consequence")) {
		return false
	}
	alt := n.ChildByFieldName("alternative")
	return alt == nil || (alt.Kind() == "block" && goAssertionBlock(c, alt))
}

// TRLC-LINKS: REQ-EMG-150
func goAssertionBlock(c *fileCtx, block *sitter.Node) bool {
	stmts, ok := goBlockStatements(block)
	if !ok {
		return false
	}
	asserts := 0
	for _, stmt := range stmts {
		switch {
		case stmt.Kind() == "return_statement":
		case stmt.Kind() == "expression_statement" && goAssertionCall(c, stmt):
			asserts++
		default:
			return false
		}
	}
	return asserts > 0
}

// goBlockStatements returns the statements of a block that holds one
// statement list.
// TRLC-LINKS: REQ-EMG-150
func goBlockStatements(block *sitter.Node) ([]*sitter.Node, bool) {
	if block == nil || block.Kind() != "block" || block.NamedChildCount() != 1 {
		return nil, false
	}
	list := block.NamedChild(0)
	if list.Kind() != "statement_list" {
		return nil, false
	}
	return namedChildren(list), true
}
