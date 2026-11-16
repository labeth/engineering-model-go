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
		return ecmaDeclaratorName(c, n, p)
	case "assignment_expression":
		return ecmaAssignmentName(c, n, p)
	case "pair":
		return ecmaPairName(c, n, p)
	case "public_field_definition", "field_definition":
		return ecmaFieldName(c, n, p)
	case "export_statement":
		return "default", "function"
	}
	return "", ""
}

// ecmaDeclaratorName names a function that is the value of a declarator.
// TRLC-LINKS: REQ-EMG-150
func ecmaDeclaratorName(c *fileCtx, n, p *sitter.Node) (string, string) {
	if name := p.ChildByFieldName("name"); isFieldOf(n, "value") && name != nil && name.Kind() == "identifier" {
		return text(name, c.src), "function"
	}
	return "", ""
}

// ecmaAssignmentName names a function that is assigned to a name or a member.
// TRLC-LINKS: REQ-EMG-150
func ecmaAssignmentName(c *fileCtx, n, p *sitter.Node) (string, string) {
	if left := p.ChildByFieldName("left"); isFieldOf(n, "right") && isIdentifierOrMember(left) {
		return strings.Join(strings.Fields(text(left, c.src)), ""), "function"
	}
	return "", ""
}

// isIdentifierOrMember reports an identifier or a member expression.
// TRLC-LINKS: REQ-EMG-150
func isIdentifierOrMember(n *sitter.Node) bool {
	return n != nil && (n.Kind() == "identifier" || n.Kind() == "member_expression")
}

// ecmaPairName names a function that is the value of an object property.
// TRLC-LINKS: REQ-EMG-150
func ecmaPairName(c *fileCtx, n, p *sitter.Node) (string, string) {
	if key := p.ChildByFieldName("key"); isFieldOf(n, "value") && key != nil {
		return strings.Trim(text(key, c.src), "\"'`"), "function"
	}
	return "", ""
}

// ecmaFieldName names a function that is the value of a class field.
// TRLC-LINKS: REQ-EMG-150
func ecmaFieldName(c *fileCtx, n, p *sitter.Node) (string, string) {
	name := p.ChildByFieldName("name")
	if name == nil {
		name = p.ChildByFieldName("property")
	}
	if name != nil && isFieldOf(n, "value") {
		return text(name, c.src), "method"
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
	call := ecmaCallOfArguments(args)
	if call == nil {
		return "", ""
	}
	root, props := callRoot(call.ChildByFieldName("function"), c.src)
	role, ok := ecmaRole(root, props)
	if !ok {
		return "", ""
	}
	title = ecmaTitle(c, args)
	if title == "" {
		title = root + "@" + strconv.Itoa(int(n.StartPosition().Row)+1)
	}
	return role, title
}

// ecmaCallOfArguments returns the call a node of arguments belongs to; nil for
// any other node.
// TRLC-LINKS: REQ-EMG-150
func ecmaCallOfArguments(args *sitter.Node) *sitter.Node {
	if args == nil || args.Kind() != "arguments" {
		return nil
	}
	call := args.Parent()
	if call == nil || call.Kind() != "call_expression" {
		return nil
	}
	return call
}

// ecmaRole is the role of the call chain that starts at root with the given
// properties: "test", "describe", or false when it is neither.
// TRLC-LINKS: REQ-EMG-150
func ecmaRole(root string, props []string) (string, bool) {
	switch {
	case ecmaTestWords[root]:
		return ecmaTestRole(props)
	case ecmaDescribeWords[root]:
		return "describe", onlyModifiers(props)
	}
	return "", false
}

// ecmaTestRole is the role of a call chain of it/test: a describe property
// turns it into a describe block, a property that is no modifier is no test.
// TRLC-LINKS: REQ-EMG-150
func ecmaTestRole(props []string) (string, bool) {
	role := "test"
	for _, prop := range props {
		switch {
		case ecmaDescribeWords[prop]:
			role = "describe"
		case !ecmaModifiers[prop]:
			return "", false
		}
	}
	return role, true
}

// onlyModifiers reports whether every property is a modifier such as only or
// skip.
// TRLC-LINKS: REQ-EMG-150
func onlyModifiers(props []string) bool {
	for _, prop := range props {
		if !ecmaModifiers[prop] {
			return false
		}
	}
	return true
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
		ps.addECMAParam(c, param)
	}
	return ps
}

// addECMAParam counts one formal parameter.
// TRLC-LINKS: REQ-EMG-150
func (ps *paramStats) addECMAParam(c *fileCtx, param *sitter.Node) {
	switch param.Kind() {
	case "required_parameter", "optional_parameter":
		ps.addTypedParam(c, param)
	default:
		ps.count++
	}
}

// addTypedParam counts a TypeScript parameter, unless it is the this
// parameter, and what its annotation says.
// TRLC-LINKS: REQ-EMG-150
func (ps *paramStats) addTypedParam(c *fileCtx, param *sitter.Node) {
	if pattern := param.ChildByFieldName("pattern"); pattern != nil && pattern.Kind() == "this" {
		return
	}
	ps.count++
	t := param.ChildByFieldName("type")
	if t == nil {
		return
	}
	ps.annotated = true
	if ecmaPrimitiveType(c, t) {
		ps.primitive++
	}
}

// ecmaPrimitiveType reports a type annotation that is a predefined primitive.
// TRLC-LINKS: REQ-EMG-150
func ecmaPrimitiveType(c *fileCtx, t *sitter.Node) bool {
	inner := t.NamedChild(0)
	return inner != nil && inner.Kind() == "predefined_type" && ecmaPrimitiveTypes[text(inner, c.src)]
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
