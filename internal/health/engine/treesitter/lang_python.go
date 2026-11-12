// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/labeth/engineering-model-go/internal/codemap"
)

// pythonLanguage: decision points are if, elif, for, while, except (and
// except*), every case_clause except a bare `case _`, conditional_expression,
// each and/or (boolean_operator) and each if_clause of a comprehension or
// generator expression (a case guard is also an if_clause but is not counted
// again). The for of a comprehension is not a decision point. try and match
// nest; with does not.
var pythonLanguage = &language{
	name: "python",
	rules: map[string]rule{
		"if_statement":           {decision: true, nest: true},
		"elif_clause":            {decision: true},
		"for_statement":          {decision: true, nest: true},
		"while_statement":        {decision: true, nest: true},
		"try_statement":          {nest: true},
		"except_clause":          {decision: true},
		"except_group_clause":    {decision: true},
		"match_statement":        {nest: true},
		"conditional_expression": {decision: true},
		"boolean_operator":       {boolOp: true},
		"if_clause":              {extra: true},
		"case_clause":            {extra: true},
	},
	unitKinds:   map[string]bool{"function_definition": true, "lambda": true},
	unit:        pythonUnit,
	scopeKinds:  map[string]bool{"class_definition": true},
	scope:       func(c *fileCtx, n *sitter.Node) string { return text(n.ChildByFieldName("name"), c.src) },
	params:      pythonParams,
	extra:       pythonExtra,
	boolOps:     func(_ *fileCtx, n *sitter.Node) int { return b2i(n.Kind() == "boolean_operator") },
	transparent: map[string]bool{"parenthesized_expression": true, "not_operator": true},
	containers:  map[string]bool{"block": true},
	assertion:   pythonAssertion,
	fileIsTest:  func(c *fileCtx, _ *sitter.Node) bool { return codemap.IsPythonTestFile(c.path) },
	primitives:  primitiveIfAnnotated,
}

// TRLC-LINKS: REQ-EMG-150
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// TRLC-LINKS: REQ-EMG-150
func pythonUnit(c *fileCtx, n *sitter.Node) (unitInfo, bool) {
	body := n.ChildByFieldName("body")
	if body == nil {
		return unitInfo{}, false
	}
	if n.Kind() == "lambda" {
		parent := n.Parent()
		if parent == nil || parent.Kind() != "assignment" || !isFieldOf(n, "right") {
			return unitInfo{}, false
		}
		left := parent.ChildByFieldName("left")
		if left == nil || left.Kind() != "identifier" {
			return unitInfo{}, false
		}
		return unitInfo{name: text(left, c.src), kind: "function", body: body}, true
	}
	kind := "function"
	if pythonInClass(n) {
		kind = "method"
	}
	return unitInfo{
		name: text(n.ChildByFieldName("name"), c.src), kind: kind, body: body,
		test: c.isTest && codemap.IsPythonTestFunction(n, c.src),
	}, true
}

// pythonInClass reports whether a function is defined directly in a class body.
// TRLC-LINKS: REQ-EMG-150
func pythonInClass(n *sitter.Node) bool {
	parent := n.Parent()
	if parent != nil && parent.Kind() == "decorated_definition" {
		parent = parent.Parent()
	}
	return parent != nil && parent.Kind() == "block" && parent.Parent() != nil && parent.Parent().Kind() == "class_definition"
}

var pythonComprehensions = map[string]bool{
	"list_comprehension": true, "set_comprehension": true, "dictionary_comprehension": true, "generator_expression": true,
}

// pythonExtra adds the case clauses and comprehension conditions.
// TRLC-LINKS: REQ-EMG-150
func pythonExtra(c *fileCtx, n *sitter.Node) int {
	switch n.Kind() {
	case "if_clause":
		return b2i(n.Parent() != nil && pythonComprehensions[n.Parent().Kind()])
	case "case_clause":
		return b2i(!pythonWildcardCase(c, n))
	}
	return 0
}

// pythonWildcardCase reports `case _:` without a guard, the default branch.
// TRLC-LINKS: REQ-EMG-150
func pythonWildcardCase(c *fileCtx, n *sitter.Node) bool {
	if n.ChildByFieldName("guard") != nil {
		return false
	}
	patterns := 0
	for _, child := range namedChildren(n) {
		if child.Kind() == "case_pattern" {
			patterns++
			if strings.TrimSpace(text(child, c.src)) != "_" {
				return false
			}
		}
	}
	return patterns == 1
}

var pythonPrimitiveTypes = map[string]bool{"int": true, "str": true, "float": true, "bool": true, "bytes": true, "complex": true}

// pythonParams counts parameters; the separators * and / are not parameters
// and neither is a leading self or cls of a method. A parameter is primitive
// when its annotation is str, int, float, bool, bytes or complex.
// TRLC-LINKS: REQ-EMG-150
func pythonParams(c *fileCtx, n *sitter.Node, method bool) paramStats {
	var ps paramStats
	list := n.ChildByFieldName("parameters")
	if list == nil {
		return ps
	}
	for i, param := range namedChildren(list) {
		kind := param.Kind()
		if kind == "keyword_separator" || kind == "positional_separator" {
			continue
		}
		if i == 0 && method && pythonReceiver(c, param) {
			continue
		}
		ps.count++
		if kind != "typed_parameter" && kind != "typed_default_parameter" {
			continue
		}
		ps.annotated = true
		if param.NamedChild(0) != nil && (param.NamedChild(0).Kind() == "list_splat_pattern" || param.NamedChild(0).Kind() == "dictionary_splat_pattern") {
			continue
		}
		if pythonPrimitiveTypes[strings.TrimSpace(text(param.ChildByFieldName("type"), c.src))] {
			ps.primitive++
		}
	}
	return ps
}

// TRLC-LINKS: REQ-EMG-150
func pythonReceiver(c *fileCtx, param *sitter.Node) bool {
	name := param
	if param.Kind() == "typed_parameter" || param.Kind() == "default_parameter" || param.Kind() == "typed_default_parameter" {
		name = param.NamedChild(0)
		if param.Kind() != "typed_parameter" {
			name = param.ChildByFieldName("name")
		}
	}
	if name == nil || name.Kind() != "identifier" {
		return false
	}
	id := text(name, c.src)
	return id == "self" || id == "cls"
}

// pythonAssertion: an assert statement, a self.assert* call, pytest.raises as a
// call or as the context manager of a with statement.
// TRLC-LINKS: REQ-EMG-150
func pythonAssertion(c *fileCtx, n *sitter.Node) bool {
	switch n.Kind() {
	case "assert_statement":
		return true
	case "expression_statement":
		return n.NamedChildCount() == 1 && pythonAssertCall(c, n.NamedChild(0))
	case "with_statement":
		clause := n.NamedChild(0)
		if clause == nil || clause.Kind() != "with_clause" {
			return false
		}
		for _, item := range namedChildren(clause) {
			if item.Kind() == "with_item" && pythonAssertCall(c, item.ChildByFieldName("value")) {
				return true
			}
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-150
func pythonAssertCall(c *fileCtx, call *sitter.Node) bool {
	if call == nil || call.Kind() != "call" {
		return false
	}
	fn := call.ChildByFieldName("function")
	if fn == nil || fn.Kind() != "attribute" {
		return false
	}
	object := text(fn.ChildByFieldName("object"), c.src)
	attr := text(fn.ChildByFieldName("attribute"), c.src)
	return (object == "self" && strings.HasPrefix(attr, "assert")) || (object == "pytest" && attr == "raises")
}
