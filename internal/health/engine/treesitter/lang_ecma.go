// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	"path/filepath"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// The ECMAScript family shares one table: TypeScript, TSX and JavaScript use
// the same node kinds for control flow. Decision points are if, for, for-in
// and for-of (for_in_statement), while, do, every switch_case (switch_default
// is not one), catch_clause, ternary_expression, the operators && || ?? and
// the assignments &&= ||= ??=. An else-if is an if_statement inside an
// else_clause. JavaScript declares no parameter types, so primitive obsession
// is unsupported there.
var (
	typescriptLanguage = newECMALanguage("typescript", true)
	tsxLanguage        = newECMALanguage("tsx", true)
	jsLanguage         = newECMALanguage("javascript", false)
)

var ecmaRules = map[string]rule{
	"if_statement":                    {decision: true, nest: true},
	"for_statement":                   {decision: true, nest: true},
	"for_in_statement":                {decision: true, nest: true},
	"while_statement":                 {decision: true, nest: true},
	"do_statement":                    {decision: true, nest: true},
	"switch_statement":                {nest: true},
	"switch_case":                     {decision: true},
	"try_statement":                   {nest: true},
	"catch_clause":                    {decision: true},
	"ternary_expression":              {decision: true},
	"binary_expression":               {boolOp: true},
	"augmented_assignment_expression": {extra: true},
}

// TRLC-LINKS: REQ-EMG-150
func newECMALanguage(name string, typed bool) *language {
	l := &language{
		name:  name,
		rules: ecmaRules,
		unitKinds: map[string]bool{
			"function_declaration": true, "generator_function_declaration": true, "method_definition": true,
			"function_expression": true, "generator_function": true, "arrow_function": true,
		},
		unit: ecmaUnit,
		scopeKinds: map[string]bool{
			"class_declaration": true, "abstract_class_declaration": true, "class": true,
			"function_expression": true, "arrow_function": true,
		},
		scope:  ecmaScope,
		params: ecmaParams,
		chained: func(n *sitter.Node) bool {
			return n.Kind() == "if_statement" && n.Parent() != nil && n.Parent().Kind() == "else_clause"
		},
		extra:       ecmaAugmented,
		boolOps:     binaryBoolOps("&&", "||", "??"),
		transparent: map[string]bool{"parenthesized_expression": true, "unary_expression": true},
		containers:  map[string]bool{"statement_block": true, "switch_case": true, "switch_default": true},
		assertion:   ecmaAssertion,
		fileIsTest:  func(c *fileCtx, _ *sitter.Node) bool { return ecmaTestPath(c.path) },
		primitives:  primitiveUnsupported,
	}
	if typed {
		l.primitives = primitiveSupported
	}
	return l
}

// ecmaTestPath: *.test.* and *.spec.* files, and files below test/, tests/ or
// __tests__/.
// TRLC-LINKS: REQ-EMG-150
func ecmaTestPath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") || pathHasDir(path, "test", "tests", "__tests__")
}

// ecmaAugmented counts the logical assignments &&= ||= ??=.
// TRLC-LINKS: REQ-EMG-150
func ecmaAugmented(_ *fileCtx, n *sitter.Node) int {
	if n.Kind() != "augmented_assignment_expression" {
		return 0
	}
	switch operatorOf(n) {
	case "&&=", "||=", "??=":
		return 1
	}
	return 0
}

// TRLC-LINKS: REQ-EMG-150
func ecmaUnit(c *fileCtx, n *sitter.Node) (unitInfo, bool) {
	body := n.ChildByFieldName("body")
	if body == nil {
		return unitInfo{}, false
	}
	switch n.Kind() {
	case "function_declaration", "generator_function_declaration":
		return unitInfo{name: ecmaDeclName(c, n), kind: "function", body: body}, true
	case "method_definition":
		return unitInfo{name: text(n.ChildByFieldName("name"), c.src), kind: "method", body: body}, true
	}
	if c.isTest {
		if role, title := ecmaTestCall(c, n); role == "test" {
			return unitInfo{name: title, kind: "function", body: body, test: true}, true
		}
	}
	if name, kind := ecmaAssignedName(c, n); name != "" {
		return unitInfo{name: name, kind: kind, body: body}, true
	}
	return unitInfo{}, false
}

// ecmaDeclName names a function declaration; export default function () {} has
// no name and is called default.
// TRLC-LINKS: REQ-EMG-150
func ecmaDeclName(c *fileCtx, n *sitter.Node) string {
	if name := text(n.ChildByFieldName("name"), c.src); name != "" {
		return name
	}
	return "default"
}

// ecmaAssignedName names a function expression or arrow function by the
// variable, property, class field or export it is bound to; an anonymous one
// returns "" and counts toward its enclosing function.
// TRLC-LINKS: REQ-EMG-150
func ecmaAssignedName(c *fileCtx, n *sitter.Node) (string, string) {
	p := n.Parent()
	if p == nil {
		return "", ""
	}
	switch p.Kind() {
	case "variable_declarator":
		if name := p.ChildByFieldName("name"); isFieldOf(n, "value") && name != nil && name.Kind() == "identifier" {
			return text(name, c.src), "function"
		}
	case "assignment_expression":
		if left := p.ChildByFieldName("left"); isFieldOf(n, "right") && left != nil && (left.Kind() == "identifier" || left.Kind() == "member_expression") {
			return strings.Join(strings.Fields(text(left, c.src)), ""), "function"
		}
	case "pair":
		if key := p.ChildByFieldName("key"); isFieldOf(n, "value") && key != nil {
			return strings.Trim(text(key, c.src), "\"'`"), "function"
		}
	case "public_field_definition", "field_definition":
		name := p.ChildByFieldName("name")
		if name == nil {
			name = p.ChildByFieldName("property")
		}
		if name != nil && isFieldOf(n, "value") {
			return text(name, c.src), "method"
		}
	case "export_statement":
		return "default", "function"
	}
	return "", ""
}

var (
	ecmaTestWords     = map[string]bool{"it": true, "test": true, "specify": true}
	ecmaDescribeWords = map[string]bool{"describe": true, "suite": true, "context": true}
	ecmaModifiers     = map[string]bool{"only": true, "skip": true, "todo": true, "each": true, "concurrent": true, "failing": true, "fails": true, "serial": true, "parallel": true}
)

// ecmaTestCall classifies a callback passed to it/test/describe (also .only,
// .skip, .each(table)(...) and test.describe): role is "test", "describe" or
// "", and title is the first argument when it is a string.
// TRLC-LINKS: REQ-EMG-150
func ecmaTestCall(c *fileCtx, n *sitter.Node) (role, title string) {
	args := n.Parent()
	if args == nil || args.Kind() != "arguments" || args.Parent() == nil || args.Parent().Kind() != "call_expression" {
		return "", ""
	}
	root, props := callRoot(args.Parent().ChildByFieldName("function"), c.src)
	switch {
	case ecmaTestWords[root]:
		role = "test"
		for _, prop := range props {
			switch {
			case ecmaDescribeWords[prop]:
				role = "describe"
			case !ecmaModifiers[prop]:
				return "", ""
			}
		}
	case ecmaDescribeWords[root]:
		role = "describe"
		for _, prop := range props {
			if !ecmaModifiers[prop] {
				return "", ""
			}
		}
	default:
		return "", ""
	}
	title = ecmaTitle(c, args)
	if title == "" {
		title = root + "@" + strconv.Itoa(int(n.StartPosition().Row)+1)
	}
	return role, title
}

// TRLC-LINKS: REQ-EMG-150
func ecmaTitle(c *fileCtx, args *sitter.Node) string {
	first := args.NamedChild(0)
	if first == nil || (first.Kind() != "string" && first.Kind() != "template_string") {
		return ""
	}
	return strings.Trim(text(first, c.src), "\"'`")
}

// ecmaScope names the class or the describe block above a function.
// TRLC-LINKS: REQ-EMG-150
func ecmaScope(c *fileCtx, n *sitter.Node) string {
	if strings.Contains(n.Kind(), "class") {
		return text(n.ChildByFieldName("name"), c.src)
	}
	if !c.isTest {
		return ""
	}
	if role, title := ecmaTestCall(c, n); role == "describe" {
		return title
	}
	return ""
}

var ecmaPrimitiveTypes = map[string]bool{"string": true, "number": true, "boolean": true, "bigint": true}

// ecmaParams counts formal parameters (this excluded). JavaScript parameters
// are bare identifiers and patterns; TypeScript ones are required_parameter and
// optional_parameter with an optional type annotation.
// TRLC-LINKS: REQ-EMG-150
func ecmaParams(c *fileCtx, n *sitter.Node, _ bool) paramStats {
	var ps paramStats
	if single := n.ChildByFieldName("parameter"); single != nil {
		ps.count = 1
		return ps
	}
	list := n.ChildByFieldName("parameters")
	if list == nil {
		return ps
	}
	for _, param := range namedChildren(list) {
		switch param.Kind() {
		case "required_parameter", "optional_parameter":
			if pattern := param.ChildByFieldName("pattern"); pattern != nil && pattern.Kind() == "this" {
				continue
			}
			ps.count++
			if t := param.ChildByFieldName("type"); t != nil {
				ps.annotated = true
				if inner := t.NamedChild(0); inner != nil && inner.Kind() == "predefined_type" && ecmaPrimitiveTypes[text(inner, c.src)] {
					ps.primitive++
				}
			}
		default:
			ps.count++
		}
	}
	return ps
}

var ecmaTestMethodsNotAssertions = map[string]bool{
	"test": true, "describe": true, "it": true, "before": true, "after": true, "beforeEach": true, "afterEach": true,
	"diagnostic": true, "skip": true, "todo": true, "plan": true, "mock": true, "runOnly": true, "name": true,
}

// ecmaAssertion: an expression statement that is an expect(...) chain, an
// assert call (assert(x), assert.equal(a, b)) or a node:test t.* call.
// TRLC-LINKS: REQ-EMG-150
func ecmaAssertion(c *fileCtx, n *sitter.Node) bool {
	if n.Kind() != "expression_statement" || n.NamedChildCount() == 0 {
		return false
	}
	root, props := callRoot(n.NamedChild(0), c.src)
	switch root {
	case "expect", "assert":
		return true
	case "t":
		return len(props) > 0 && !ecmaTestMethodsNotAssertions[props[0]]
	}
	return false
}
