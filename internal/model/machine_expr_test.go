// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// sexpr renders a tree in fully parenthesised prefix form so precedence and
// associativity assertions do not depend on the printer.
func sexpr(e SemanticExpression) string {
	if e.Operator == "" {
		return sexprLeaf(e)
	}
	parts := []string{e.Operator}
	for _, o := range e.Operands {
		parts = append(parts, sexpr(o))
	}
	return "(" + strings.Join(parts, " ") + ")"
}

// sexprLeaf renders a variable or literal leaf of a tree.
func sexprLeaf(e SemanticExpression) string {
	switch {
	case e.Kind == MachineKindVariable:
		return e.Value
	case e.TypedValue == nil:
		return "<?>"
	case e.TypedValue.Reference != "":
		return "@" + e.TypedValue.Reference
	case e.TypedValue.Integer != nil:
		return strconv.FormatInt(*e.TypedValue.Integer, 10)
	case e.TypedValue.Boolean != nil:
		return strconv.FormatBool(*e.TypedValue.Boolean)
	default:
		return strconv.Quote(e.TypedValue.String)
	}
}

func mustParse(t *testing.T, src string, kind MachineExprKind) SemanticExpression {
	t.Helper()
	expr, diags := ParseMachineExpr(src, kind)
	if len(diags) != 0 {
		t.Fatalf("parse %q: unexpected diagnostics %v", src, diags)
	}
	return expr
}

// machineTreeCase is a source and the prefix form its tree must render to.
type machineTreeCase struct{ src, want string }

var machineBooleanTreeCases = []machineTreeCase{
	{"x", "x"},
	{"42", "42"},
	{"0", "0"},
	{"true", "true"},
	{"false", "false"},
	{`"hello"`, `"hello"`},
	{`"a\"b\\c"`, `"a\"b\\c"`},
	{"_a1", "_a1"},
	{"in_state(STATE-PAY-CLOSED)", "(in_state @STATE-PAY-CLOSED)"},
	{"in_state( 9_X-1 )", "(in_state @9_X-1)"},
	{"(x)", "x"},
	{"((x))", "x"},
	{"  \n x \t ", "x"},
	// every operator
	{"a implies b", "(implies a b)"},
	{"a or b", "(or a b)"},
	{"a and b", "(and a b)"},
	{"not a", "(not a)"},
	{"a == b", "(== a b)"},
	{"a != b", "(!= a b)"},
	{"a < b", "(< a b)"},
	{"a <= b", "(<= a b)"},
	{"a > b", "(> a b)"},
	{"a >= b", "(>= a b)"},
	{"a + b", "(+ a b)"},
	{"a - b", "(- a b)"},
	{"a * b", "(* a b)"},
	{"-a", "(neg a)"},
	// precedence, loosest first
	{"a implies b or c", "(implies a (or b c))"},
	{"a or b and c", "(or a (and b c))"},
	{"a and b or c", "(or (and a b) c)"},
	{"a and not b", "(and a (not b))"},
	{"not a and b", "(and (not a) b)"},
	{"not a == b", "(not (== a b))"},
	{"a == b + c", "(== a (+ b c))"},
	{"a + b * c", "(+ a (* b c))"},
	{"a * b + c", "(+ (* a b) c)"},
	{"-a * b", "(* (neg a) b)"},
	{"-a + b", "(+ (neg a) b)"},
	{"a - -b", "(- a (neg b))"},
	{"- - a", "(neg (neg a))"},
	{"-5", "(neg 5)"},
	{"not not a", "(not (not a))"},
	{"(a or b) and c", "(and (or a b) c)"},
	{"not (a and b)", "(not (and a b))"},
	{"(a + b) * c", "(* (+ a b) c)"},
	{"(a == b) == c", "(== (== a b) c)"},
	// associativity
	{"a implies b implies c", "(implies a (implies b c))"},
	{"(a implies b) implies c", "(implies (implies a b) c)"},
	{"a or b or c", "(or (or a b) c)"},
	{"a and b and c", "(and (and a b) c)"},
	{"a - b - c", "(- (- a b) c)"},
	{"a - b + c", "(+ (- a b) c)"},
	{"a * b * c", "(* (* a b) c)"},
	// realistic
	{"attempts < 3", "(< attempts 3)"},
	{"(attempts <= 3) and not in_state(STATE-PAY-CLOSED)", "(and (<= attempts 3) (not (in_state @STATE-PAY-CLOSED)))"},
	{`status == "retry"`, `(== status "retry")`},
	{"007", "7"},
	{"9223372036854775807", "9223372036854775807"},
	// types are not the parser's concern
	{"(a and b) + 1", "(+ (and a b) 1)"},
	{"1 and 2", "(and 1 2)"},
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprBooleanTrees(t *testing.T) {
	for _, tc := range machineBooleanTreeCases {
		t.Run(tc.src, func(t *testing.T) {
			expr := mustParse(t, tc.src, MachineExprBoolean)
			if got := sexpr(expr); got != tc.want {
				t.Fatalf("parse %q = %s, want %s", tc.src, got, tc.want)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprEffectTrees(t *testing.T) {
	tests := []struct {
		src, want string
	}{
		{"x := 1", "(all (:= x 1))"},
		{"x := 1;", "(all (:= x 1))"},
		{"x := 1 ;  y := x + 1 ;", "(all (:= x 1) (:= y (+ x 1)))"},
		{`attempts := attempts + 1; status := "retry"`, `(all (:= attempts (+ attempts 1)) (:= status "retry"))`},
		{"a := b implies c", "(all (:= a (implies b c)))"},
	}
	for _, tc := range tests {
		t.Run(tc.src, func(t *testing.T) {
			expr := mustParse(t, tc.src, MachineExprEffect)
			if got := sexpr(expr); got != tc.want {
				t.Fatalf("parse %q = %s, want %s", tc.src, got, tc.want)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprLeafEncoding(t *testing.T) {
	expr := mustParse(t, `x + 1 == 2 and in_state(S-1) and "s" != "t" and true`, MachineExprBoolean)
	kinds := map[string]int{}
	for _, l := range leavesOfTestTree(t, expr) {
		kinds[checkMachineLeaf(t, l)]++
	}
	want := map[string]int{"variable": 1, "reference": 1, "integer": 2, "string": 2, "boolean": 1}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("leaf kinds = %v, want %v", kinds, want)
	}
}

// leavesOfTestTree returns the leaves of a tree and reports an operator
// node whose kind is not the operator kind.
func leavesOfTestTree(t *testing.T, e SemanticExpression) []SemanticExpression {
	t.Helper()
	if e.Operator == "" {
		return []SemanticExpression{e}
	}
	if e.Kind != MachineKindOperator {
		t.Errorf("operator node %q has kind %q", e.Operator, e.Kind)
	}
	var leaves []SemanticExpression
	for _, o := range e.Operands {
		leaves = append(leaves, leavesOfTestTree(t, o)...)
	}
	return leaves
}

// checkMachineLeaf validates the encoding of one leaf and returns its kind
// name, or "other" for a leaf of no known kind.
func checkMachineLeaf(t *testing.T, l SemanticExpression) string {
	t.Helper()
	switch l.Kind {
	case MachineKindVariable:
		if l.TypedValue != nil || l.Value != "x" {
			t.Errorf("bad variable leaf %+v", l)
		}
		return "variable"
	case MachineKindReference:
		checkMachineReferenceLeaf(t, l)
		return "reference"
	case MachineKindLiteral:
		return checkMachineLiteralLeaf(t, l)
	}
	t.Errorf("unexpected leaf %+v", l)
	return "other"
}

// checkMachineReferenceLeaf requires a reference leaf to the state S-1.
func checkMachineReferenceLeaf(t *testing.T, l SemanticExpression) {
	t.Helper()
	if l.TypedValue == nil || l.TypedValue.Kind != "reference" || l.TypedValue.Reference != "S-1" {
		t.Errorf("bad reference leaf %+v", l)
	}
}

// checkMachineLiteralLeaf requires a typed value with exactly one populated
// field and returns its kind.
func checkMachineLiteralLeaf(t *testing.T, l SemanticExpression) string {
	t.Helper()
	v := l.TypedValue
	if v == nil {
		t.Fatalf("literal without typed value: %+v", l)
	}
	populated := 0
	for _, ok := range []bool{v.String != "", v.Integer != nil, v.Boolean != nil, v.Real != nil, v.Reference != "", v.Quantity != nil} {
		if ok {
			populated++
		}
	}
	if populated != 1 {
		t.Errorf("literal %+v populates %d fields", v, populated)
	}
	return v.Kind
}

// Parsed trees must pass the existing structural validators once the state
// IDs they reference exist in the model.
//
// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprPassesStructuralValidation(t *testing.T) {
	identities := map[string]string{"STATE-PAY-CLOSED": "state"}
	for _, tc := range []struct {
		src  string
		kind MachineExprKind
	}{
		{`attempts < 3 and not in_state(STATE-PAY-CLOSED) and false != (x == 0)`, MachineExprBoolean},
		{`attempts := attempts + 1; status := "retry"; open := true; n := -1`, MachineExprEffect},
	} {
		expr := mustParse(t, tc.src, tc.kind)
		validator := semanticValidator{identities: identities}
		validator.validateExpression(&expr, "guard")
		if len(validator.diagnostics) != 0 {
			t.Errorf("%q: structural diagnostics %+v", tc.src, validator.diagnostics)
		}
	}
	expr := mustParse(t, "in_state(STATE-NOPE)", MachineExprBoolean)
	validator := semanticValidator{identities: identities}
	validator.validateExpression(&expr, "guard")
	if diags := validator.diagnostics; len(diags) != 1 || diags[0].Code != "semantic.dangling_reference" {
		t.Errorf("unknown state must dangle, got %+v", diags)
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprTreeSurvivesJSON(t *testing.T) {
	expr := mustParse(t, `x := 0; y := false; z := "q"`, MachineExprEffect)
	data, err := json.Marshal(expr)
	if err != nil {
		t.Fatal(err)
	}
	var back SemanticExpression
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expr, back) {
		t.Fatalf("JSON round trip changed the tree:\n%s", data)
	}
	if got := MachineExprString(back); got != `x := 0; y := false; z := "q"` {
		t.Fatalf("zero and false must be preserved, got %q", got)
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestMachineExprStringCanonicalAndRoundTrip(t *testing.T) {
	tests := []struct {
		src, want string
		kind      MachineExprKind
	}{
		{"((a))", "a", MachineExprBoolean},
		{"( a  or b )  and c", "(a or b) and c", MachineExprBoolean},
		{"a or (b and c)", "a or b and c", MachineExprBoolean},
		{"(a and b) and c", "a and b and c", MachineExprBoolean},
		{"a and (b and c)", "a and (b and c)", MachineExprBoolean},
		{"a - (b - c)", "a - (b - c)", MachineExprBoolean},
		{"(a - b) - c", "a - b - c", MachineExprBoolean},
		{"a implies (b implies c)", "a implies b implies c", MachineExprBoolean},
		{"(a implies b) implies c", "(a implies b) implies c", MachineExprBoolean},
		{"not (a == b)", "not a == b", MachineExprBoolean},
		{"(not a) == b", "(not a) == b", MachineExprBoolean},
		{"(a == b) == c", "(a == b) == c", MachineExprBoolean},
		{"-(a + b)", "-(a + b)", MachineExprBoolean},
		{"-(a * b)", "-(a * b)", MachineExprBoolean},
		{"- - a", "--a", MachineExprBoolean},
		{"a - -1", "a - -1", MachineExprBoolean},
		{"(-a) * b", "-a * b", MachineExprBoolean},
		{"not (a and b) or c", "not (a and b) or c", MachineExprBoolean},
		{"a   and\n  not   in_state( S-1 )", "a and not in_state(S-1)", MachineExprBoolean},
		{`"a\\b\"c" == s`, `"a\\b\"c" == s`, MachineExprBoolean},
		{"a := (1 + 2) * 3;", "a := (1 + 2) * 3", MachineExprEffect},
		{"a:=b implies c;b:=-1", "a := b implies c; b := -1", MachineExprEffect},
	}
	for _, tc := range tests {
		t.Run(tc.src, func(t *testing.T) {
			first := mustParse(t, tc.src, tc.kind)
			printed := MachineExprString(first)
			if printed != tc.want {
				t.Fatalf("print = %q, want %q", printed, tc.want)
			}
			second := mustParse(t, printed, tc.kind)
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("round trip changed the tree: %s vs %s", sexpr(first), sexpr(second))
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestMachineExprStringMalformedTreesDoNotPanic(t *testing.T) {
	one := int64(1)
	real := 2.5
	for _, tc := range []struct {
		expr SemanticExpression
		want string
	}{
		{SemanticExpression{Kind: MachineKindOperator, Operator: "and", Operands: []SemanticExpression{machineVariable("a")}}, "a and ?"},
		{SemanticExpression{Kind: MachineKindOperator, Operator: "bogus", Operands: []SemanticExpression{machineVariable("a"), machineVariable("b")}}, "?bogus(a, b)"},
		{SemanticExpression{Kind: MachineKindOperator, Operator: "bogus"}, "?bogus()"},
		{SemanticExpression{Kind: MachineKindOperator, Operator: "in_state"}, "in_state(?)"},
		{SemanticExpression{Kind: "literal", Value: "raw"}, "?raw"},
		{SemanticExpression{Kind: "literal", TypedValue: &SemanticValue{Kind: "real", Real: &real}}, "2.5"},
		{SemanticExpression{Kind: "literal", TypedValue: &SemanticValue{Kind: "integer", Integer: &one}}, "1"},
		{SemanticExpression{}, "?"},
	} {
		if got := MachineExprString(tc.expr); got != tc.want {
			t.Errorf("MachineExprString(%+v) = %q, want %q", tc.expr, got, tc.want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestMachineExprLeafCollection(t *testing.T) {
	expr := mustParse(t, "(b + a > c) and (in_state(S-2) or in_state(S-1)) and in_state(S-2) and a == a", MachineExprBoolean)
	if got, want := MachineExprVariables(expr), []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("variables = %v, want %v", got, want)
	}
	if got, want := MachineExprStateRefs(expr), []string{"S-1", "S-2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("state refs = %v, want %v", got, want)
	}
	effect := mustParse(t, "x := y + 1; z := 2", MachineExprEffect)
	if got, want := MachineExprVariables(effect), []string{"x", "y", "z"}; !reflect.DeepEqual(got, want) {
		t.Errorf("effect variables = %v, want %v", got, want)
	}
	if got := MachineExprStateRefs(effect); len(got) != 0 {
		t.Errorf("effect state refs = %v, want none", got)
	}
}

// machineOperatorSignature is the expected signature of one operator.
type machineOperatorSignature struct {
	name     string
	operands []MachineType
	result   MachineType
	sameType bool
}

var machineOperatorSignatures = []machineOperatorSignature{
	{"and", []MachineType{MachineTypeBoolean, MachineTypeBoolean}, MachineTypeBoolean, false},
	{"==", []MachineType{MachineTypeAny, MachineTypeAny}, MachineTypeBoolean, true},
	{"<=", []MachineType{MachineTypeInteger, MachineTypeInteger}, MachineTypeBoolean, false},
	{"+", []MachineType{MachineTypeInteger, MachineTypeInteger}, MachineTypeInteger, false},
	{"neg", []MachineType{MachineTypeInteger}, MachineTypeInteger, false},
	{"in_state", []MachineType{MachineTypeState}, MachineTypeBoolean, false},
	{":=", []MachineType{MachineTypeAny, MachineTypeAny}, MachineTypeEffect, true},
}

// TRLC-LINKS: REQ-EMG-138
func TestMachineOperatorTable(t *testing.T) {
	want := []string{"implies", "or", "and", "not", "==", "!=", "<", "<=", ">", ">=", "+", "-", "*", "neg", "in_state", ":=", "all"}
	var names []string
	for _, op := range MachineOperators() {
		names = append(names, op.Name)
		checkMachineOperatorShape(t, op)
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("operators = %v, want %v", names, want)
	}
	for _, tc := range machineOperatorSignatures {
		checkMachineOperatorSignature(t, tc)
	}
	checkMachineOperatorLookups(t)
	checkMachineOperatorTableIsCopy(t)
}

// checkMachineOperatorSignature requires the looked-up operator to have the
// expected signature.
func checkMachineOperatorSignature(t *testing.T, tc machineOperatorSignature) {
	t.Helper()
	op, ok := LookupMachineOperator(tc.name)
	if !ok || !reflect.DeepEqual(op.Operands, tc.operands) {
		t.Errorf("operator %q = %+v (found %v)", tc.name, op, ok)
	}
	if op.Result != tc.result || op.SameType != tc.sameType {
		t.Errorf("operator %q = %+v (found %v)", tc.name, op, ok)
	}
}

// checkMachineOperatorShape requires a signature and an arity that matches the
// operator form.
func checkMachineOperatorShape(t *testing.T, op MachineOperator) {
	t.Helper()
	if len(op.Operands) == 0 || op.Result == "" {
		t.Errorf("operator %q has no signature: %+v", op.Name, op)
	}
	infixArity := op.Form == MachineFormInfix && len(op.Operands) != 2
	prefixArity := op.Form == MachineFormPrefix && len(op.Operands) != 1
	if infixArity || prefixArity {
		t.Errorf("operator %q arity does not match form %q", op.Name, op.Form)
	}
}

// checkMachineOperatorLookups checks the variadic, associativity and unknown
// operator lookups.
func checkMachineOperatorLookups(t *testing.T) {
	t.Helper()
	if all, _ := LookupMachineOperator("all"); !all.Variadic || all.Result != MachineTypeEffect {
		t.Errorf("all = %+v", all)
	}
	if imp, _ := LookupMachineOperator("implies"); !imp.RightAssociative {
		t.Errorf("implies must be right associative")
	}
	if _, ok := LookupMachineOperator("nope"); ok {
		t.Errorf("unknown operator found")
	}
}

// checkMachineOperatorTableIsCopy requires that callers cannot corrupt the
// operator table through the returned slice.
func checkMachineOperatorTableIsCopy(t *testing.T) {
	t.Helper()
	ops := MachineOperators()
	ops[0].Name = "mutated"
	ops[4].Operands[0] = MachineTypeString
	if again, _ := LookupMachineOperator("implies"); again.Name != "implies" {
		t.Errorf("table was mutated through MachineOperators")
	}
	if eq, _ := LookupMachineOperator("=="); eq.Operands[0] != MachineTypeAny {
		t.Errorf("operand slice was mutated through MachineOperators")
	}
}

// machinePrecedenceCase parses src and expects the child of the root at
// operand index at to be the operator inner under the operator outer. A case
// passes only if the parser bound inner tighter than outer AND the table says
// the same, so swapping two binding levels in the parser fails the test. Every
// adjacent pair of levels is covered from both sides.
type machinePrecedenceCase struct {
	src, outer string
	at         int
	inner      string
}

var machinePrecedenceCases = []machinePrecedenceCase{
	{"a implies b or c", "implies", 1, "or"},
	{"a or b implies c", "implies", 0, "or"},
	{"a or b and c", "or", 1, "and"},
	{"a and b or c", "or", 0, "and"},
	{"a and not b", "and", 1, "not"},
	{"not a and b", "and", 0, "not"},
	{"not a == b", "not", 0, "=="},
	{"not a < b", "not", 0, "<"},
	{"a == b + c", "==", 1, "+"},
	{"a + b == c", "==", 0, "+"},
	{"a >= b - c", ">=", 1, "-"},
	{"a + b * c", "+", 1, "*"},
	{"a * b + c", "+", 0, "*"},
	{"a - b * c", "-", 1, "*"},
	{"-a * b", "*", 0, "neg"},
	{"a * -b", "*", 1, "neg"},
	{"a - -b", "-", 1, "neg"},
	{"-in_state(S)", "neg", 0, "in_state"},
	// associativity: right for implies, left for every other binary level
	{"a implies b implies c", "implies", 1, "implies"},
	{"a or b or c", "or", 0, "or"},
	{"a and b and c", "and", 0, "and"},
	{"a + b + c", "+", 0, "+"},
	{"a - b - c", "-", 0, "-"},
	{"a + b - c", "-", 0, "+"},
	{"a - b + c", "+", 0, "-"},
	{"a * b * c", "*", 0, "*"},
	{"not not a", "not", 0, "not"},
	{"- - a", "neg", 0, "neg"},
}

// machineOperatorsByName indexes the operator table by name.
func machineOperatorsByName() map[string]MachineOperator {
	table := map[string]MachineOperator{}
	for _, op := range MachineOperators() {
		table[op.Name] = op
	}
	return table
}

// TRLC-LINKS: REQ-EMG-138
func TestMachineOperatorsMatchParserPrecedence(t *testing.T) {
	table := machineOperatorsByName()
	for _, tc := range machinePrecedenceCases {
		t.Run(tc.src, func(t *testing.T) {
			checkMachinePrecedenceCase(t, tc, table)
		})
	}
	checkMachineSharedPrecedence(t, table)
	checkMachineComparisonsAreNonAssociative(t, table)
	// A comparison operand may itself be arithmetic but never a comparison;
	// parentheses restore it.
	if expr := mustParse(t, "(a < b) == (c < d)", MachineExprBoolean); expr.Operator != "==" || expr.Operands[0].Operator != "<" || expr.Operands[1].Operator != "<" {
		t.Errorf("parenthesised comparisons = %s", sexpr(expr))
	}
}

// checkMachinePrecedenceCase requires that the parser nests the case as
// described and that the operator table agrees with it.
func checkMachinePrecedenceCase(t *testing.T, tc machinePrecedenceCase, table map[string]MachineOperator) {
	t.Helper()
	requireMachineNesting(t, tc)
	outer, inner := table[tc.outer], table[tc.inner]
	if outer.Name != tc.outer || inner.Name != tc.inner {
		t.Fatalf("operator table lacks %q or %q", tc.outer, tc.inner)
	}
	if tc.outer == tc.inner || outer.Precedence == inner.Precedence {
		checkMachineSameLevelNesting(t, tc, outer)
		return
	}
	if inner.Precedence <= outer.Precedence {
		t.Errorf("parser binds %s tighter than %s but the table has %d <= %d", tc.inner, tc.outer, inner.Precedence, outer.Precedence)
	}
}

// requireMachineNesting requires that the parser nested the case's inner
// operator at the case's operand index of the outer one.
func requireMachineNesting(t *testing.T, tc machinePrecedenceCase) {
	t.Helper()
	expr := mustParse(t, tc.src, MachineExprBoolean)
	if expr.Operator != tc.outer || len(expr.Operands) <= tc.at || expr.Operands[tc.at].Operator != tc.inner {
		t.Fatalf("parse %q = %s, want %s with %s as operand %d", tc.src, sexpr(expr), tc.outer, tc.inner, tc.at)
	}
}

// checkMachineSameLevelNesting requires that the table associativity agrees
// with the side the parser nested on at one precedence level.
func checkMachineSameLevelNesting(t *testing.T, tc machinePrecedenceCase, outer MachineOperator) {
	t.Helper()
	wantAt := 0
	if outer.RightAssociative {
		wantAt = 1
	}
	if outer.Form == MachineFormPrefix {
		wantAt = 0
	}
	if tc.at != wantAt {
		t.Errorf("%s nests on operand %d but the table says RightAssociative=%v", tc.outer, tc.at, outer.RightAssociative)
	}
}

// checkMachineSharedPrecedence requires that operators of one level share a
// precedence.
func checkMachineSharedPrecedence(t *testing.T, table map[string]MachineOperator) {
	t.Helper()
	for _, pair := range machineSharedLevelPairs() {
		if table[pair[0]].Precedence != table[pair[1]].Precedence {
			t.Errorf("%s and %s must share a precedence level", pair[0], pair[1])
		}
	}
}

// machineSharedLevelPairs pairs every operator of a shared precedence level
// with the first operator of its level.
func machineSharedLevelPairs() [][2]string {
	var pairs [][2]string
	for _, group := range [][]string{{"==", "!=", "<", "<=", ">", ">="}, {"+", "-"}} {
		for _, name := range group[1:] {
			pairs = append(pairs, [2]string{name, group[0]})
		}
	}
	return pairs
}

// checkMachineComparisonsAreNonAssociative requires that no comparison pair
// nests in the parser and that the table never marks one right associative.
func checkMachineComparisonsAreNonAssociative(t *testing.T, table map[string]MachineOperator) {
	t.Helper()
	for _, a := range []string{"==", "!=", "<", "<=", ">", ">="} {
		if table[a].RightAssociative {
			t.Errorf("%s must not be right associative", a)
		}
		requireChainedComparisonRejected(t, a, "==")
		requireChainedComparisonRejected(t, a, "<")
	}
}

// requireChainedComparisonRejected requires "x a y b z" to fail as a chained
// comparison.
func requireChainedComparisonRejected(t *testing.T, a, b string) {
	t.Helper()
	_, diags := ParseMachineExpr("x "+a+" y "+b+" z", MachineExprBoolean)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "not associative") {
		t.Errorf("x %s y %s z: diagnostics %v, want non-associative error", a, b, diags)
	}
}

type machineErrCase struct {
	name       string
	src        string
	kind       MachineExprKind
	line, col  int
	contains   string
	extraDiags int
}

var machineDiagnosticCases = []machineErrCase{
	{name: "empty", src: "", kind: MachineExprBoolean, line: 1, col: 1, contains: "expected an expression"},
	{name: "blank", src: "  \n  ", kind: MachineExprBoolean, line: 2, col: 3, contains: "end of input"},
	{name: "empty effect", src: "", kind: MachineExprEffect, line: 1, col: 1, contains: "empty effect"},
	{name: "lone semicolon effect", src: " ;", kind: MachineExprEffect, line: 1, col: 2, contains: "expected a variable name"},
	{name: "unexpected character", src: "a < 3 && b", kind: MachineExprBoolean, line: 1, col: 7, contains: `unexpected character '&'`},
	{name: "unexpected unicode character", src: "ä", kind: MachineExprBoolean, line: 1, col: 1, contains: "unexpected character"},
	{name: "column counts runes", src: `"é" == é`, kind: MachineExprBoolean, line: 1, col: 8, contains: "unexpected character"},
	{name: "single equals", src: "a = 1", kind: MachineExprBoolean, line: 1, col: 3, contains: "did you mean"},
	{name: "single bang", src: "!a", kind: MachineExprBoolean, line: 1, col: 1, contains: "did you mean"},
	{name: "lone colon", src: "a : 1", kind: MachineExprEffect, line: 1, col: 3, contains: "unexpected character ':'"},
	{name: "unbalanced open", src: "(a and b", kind: MachineExprBoolean, line: 1, col: 9, contains: "unbalanced parenthesis"},
	{name: "unbalanced close", src: "a and b)", kind: MachineExprBoolean, line: 1, col: 8, contains: "unbalanced parenthesis"},
	{name: "empty parens", src: "()", kind: MachineExprBoolean, line: 1, col: 2, contains: `unexpected ")"`},
	{name: "trailing token", src: "a b", kind: MachineExprBoolean, line: 1, col: 3, contains: `unexpected "b" after`},
	{name: "trailing integer", src: "a 1", kind: MachineExprBoolean, line: 1, col: 3, contains: "unexpected integer 1"},
	{name: "trailing string", src: `a "s"`, kind: MachineExprBoolean, line: 1, col: 3, contains: "unexpected string literal"},
	{name: "trailing semicolon in guard", src: "a;", kind: MachineExprBoolean, line: 1, col: 2, contains: `unexpected ";"`},
	{name: "assignment in guard", src: "x := 1", kind: MachineExprBoolean, line: 1, col: 3, contains: "only allowed in effects"},
	{name: "assignment in invariant parens", src: "(x := 1)", kind: MachineExprBoolean, line: 1, col: 4, contains: "unbalanced parenthesis: expected ')'"},
	{name: "dangling operator", src: "a and", kind: MachineExprBoolean, line: 1, col: 6, contains: "expected an expression"},
	{name: "dangling not", src: "not", kind: MachineExprBoolean, line: 1, col: 4, contains: "expected an expression"},
	{name: "dangling minus", src: "a - ", kind: MachineExprBoolean, line: 1, col: 5, contains: "expected an expression"},
	{name: "dangling implies", src: "a implies", kind: MachineExprBoolean, line: 1, col: 10, contains: "expected an expression"},
	{name: "dangling comparison", src: "a <", kind: MachineExprBoolean, line: 1, col: 4, contains: "expected an expression"},
	{name: "leading operator", src: "* a", kind: MachineExprBoolean, line: 1, col: 1, contains: `unexpected "*"`},
	{name: "chained comparison", src: "a < b < c", kind: MachineExprBoolean, line: 1, col: 7, contains: "not associative"},
	{name: "chained equality", src: "a == b != c", kind: MachineExprBoolean, line: 1, col: 8, contains: "not associative"},
	{name: "not inside comparison", src: "a == not b", kind: MachineExprBoolean, line: 1, col: 6, contains: "wrap it in parentheses"},
	{name: "keyword and as variable", src: "x and and", kind: MachineExprBoolean, line: 1, col: 7, contains: `keyword "and" cannot be used as a variable`},
	{name: "keyword or as operand", src: "or", kind: MachineExprBoolean, line: 1, col: 1, contains: `keyword "or" cannot be used`},
	{name: "keyword implies as variable", src: "1 + implies", kind: MachineExprBoolean, line: 1, col: 5, contains: `keyword "implies"`},
	{name: "keyword assigned", src: "and := 1", kind: MachineExprEffect, line: 1, col: 1, contains: `keyword "and" cannot be used as a variable name`},
	{name: "true assigned", src: "x := 1; true := 2", kind: MachineExprEffect, line: 1, col: 9, contains: `keyword "true"`},
	{name: "in_state assigned", src: "in_state := 1", kind: MachineExprEffect, line: 1, col: 1, contains: `keyword "in_state"`},
	{name: "in_state without paren", src: "in_state", kind: MachineExprBoolean, line: 1, col: 9, contains: "expected '(' after in_state"},
	{name: "in_state empty id", src: "in_state()", kind: MachineExprBoolean, line: 1, col: 10, contains: "expected a state ID"},
	{name: "in_state eof", src: "in_state(", kind: MachineExprBoolean, line: 1, col: 10, contains: "expected a state ID"},
	{name: "in_state quoted id", src: `in_state("S")`, kind: MachineExprBoolean, line: 1, col: 10, contains: "expected a state ID"},
	{name: "in_state bad id char", src: "in_state(A.B)", kind: MachineExprBoolean, line: 1, col: 11, contains: "unexpected character '.'"},
	{name: "in_state unclosed", src: "in_state(S-1", kind: MachineExprBoolean, line: 1, col: 13, contains: "expected ')' after the state ID"},
	{name: "in_state two ids", src: "in_state(A B)", kind: MachineExprBoolean, line: 1, col: 12, contains: "expected ')' after the state ID"},
	{name: "unterminated string", src: `s == "abc`, kind: MachineExprBoolean, line: 1, col: 6, contains: "unterminated string"},
	{name: "unterminated string at newline", src: "s == \"abc\ndef\"", kind: MachineExprBoolean, line: 1, col: 6, contains: "unterminated string"},
	{name: "lone quote", src: `"`, kind: MachineExprBoolean, line: 1, col: 1, contains: "unterminated string"},
	{name: "trailing backslash", src: `"a\`, kind: MachineExprBoolean, line: 1, col: 3, contains: "invalid escape"},
	{name: "invalid escape", src: `"a\n"`, kind: MachineExprBoolean, line: 1, col: 3, contains: "invalid escape"},
	{name: "empty string literal", src: `s == ""`, kind: MachineExprBoolean, line: 1, col: 6, contains: "empty string"},
	{name: "integer then letter", src: "3abc", kind: MachineExprBoolean, line: 1, col: 1, contains: "invalid integer literal"},
	{name: "integer overflow", src: "9223372036854775808", kind: MachineExprBoolean, line: 1, col: 1, contains: "out of range"},
	{name: "float", src: "1.5", kind: MachineExprBoolean, line: 1, col: 2, contains: "unexpected character '.'"},
	{name: "duplicate assignment", src: "x := 1; y := 2; x := 3", kind: MachineExprEffect, line: 1, col: 17, contains: `variable "x" is assigned more than once`},
	{name: "duplicate assignment without trailing", src: "x := 1;\nx := x + 1;", kind: MachineExprEffect, line: 2, col: 1, contains: `variable "x" is assigned more than once`},
	{name: "effect missing assign", src: "x + 1", kind: MachineExprEffect, line: 1, col: 3, contains: "expected ':=' after the variable name"},
	{name: "effect missing value", src: "x :=", kind: MachineExprEffect, line: 1, col: 5, contains: "expected an expression"},
	{name: "effect missing separator", src: "x := 1 y := 2", kind: MachineExprEffect, line: 1, col: 8, contains: "expected ';' or the end of the effect"},
	{name: "effect nested assignment", src: "x := y := 1", kind: MachineExprEffect, line: 1, col: 8, contains: "expected ';' or the end of the effect"},
	{name: "effect starts with literal", src: "1 := 2", kind: MachineExprEffect, line: 1, col: 1, contains: "expected a variable name"},
	{name: "effect lexical error before syntax", src: "x := 1 $ y", kind: MachineExprEffect, line: 1, col: 8, contains: "unexpected character '$'"},
	{name: "multi-line position", src: "a and\n  b or\n  (c <\n   )", kind: MachineExprBoolean, line: 4, col: 4, contains: `unexpected ")"`},
	{name: "multi-line unexpected character", src: "a\n\n  and  #", kind: MachineExprBoolean, line: 3, col: 8, contains: "unexpected character '#'"},
	{name: "multi-line effect duplicate", src: "x := 1;\n  y := 2;\n  y := 3;", kind: MachineExprEffect, line: 3, col: 3, contains: `variable "y"`},
	{name: "crlf position", src: "a\r\nand\r\n)", kind: MachineExprBoolean, line: 3, col: 1, contains: "expected an expression"},
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprDiagnostics(t *testing.T) {
	for _, tc := range machineDiagnosticCases {
		t.Run(tc.name, func(t *testing.T) {
			checkMachineErrCase(t, tc)
		})
	}
}

// checkMachineErrCase requires that the source fails with the case position,
// message and number of diagnostics and yields a zero expression.
func checkMachineErrCase(t *testing.T, tc machineErrCase) {
	t.Helper()
	expr, diags := ParseMachineExpr(tc.src, tc.kind)
	if len(diags) == 0 {
		t.Fatalf("parse %q succeeded: %s", tc.src, sexpr(expr))
	}
	if !reflect.DeepEqual(expr, SemanticExpression{}) {
		t.Errorf("expression must be zero on error, got %s", sexpr(expr))
	}
	checkMachineErrDiagnostic(t, tc, diags[0])
	if len(diags) != 1+tc.extraDiags {
		t.Errorf("got %d diagnostics %v", len(diags), diags)
	}
}

// checkMachineErrDiagnostic requires the first diagnostic to carry the code,
// position and message fragment of the case.
func checkMachineErrDiagnostic(t *testing.T, tc machineErrCase, d MachineExprDiagnostic) {
	t.Helper()
	if d.Code != "semantic.invalid_expression" {
		t.Errorf("code = %q", d.Code)
	}
	if !isMachineDiagAt(d, tc.line, tc.col) {
		t.Errorf("position = %d:%d, want %d:%d (%s)", d.Line, d.Column, tc.line, tc.col, d.Message)
	}
	if !strings.Contains(d.Message, tc.contains) {
		t.Errorf("message %q does not contain %q", d.Message, tc.contains)
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprReportsEveryDuplicateAssignment(t *testing.T) {
	_, diags := ParseMachineExpr("x := 1; x := 2; y := 1; x := 3; y := 2", MachineExprEffect)
	if len(diags) != 3 {
		t.Fatalf("diagnostics = %v, want 3", diags)
	}
	for i, want := range []string{"9", "25", "33"} {
		if got := strconv.Itoa(diags[i].Column); got != want {
			t.Errorf("diagnostic %d column = %s, want %s", i, got, want)
		}
	}
	// A syntax error after a duplicate is appended after it.
	_, diags = ParseMachineExpr("x := 1; x := 2; y := )", MachineExprEffect)
	if len(diags) != 2 || diags[0].Column != 9 || diags[1].Column != 22 {
		t.Fatalf("diagnostics = %v", diags)
	}
}

// isMachineDiagAt reports whether the diagnostic is at line:col.
func isMachineDiagAt(d MachineExprDiagnostic, line, col int) bool {
	return d.Line == line && d.Column == col
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprUnknownKind(t *testing.T) {
	expr, diags := ParseMachineExpr("a", MachineExprKind("bogus"))
	if len(diags) != 1 || !isMachineDiagAt(diags[0], 1, 1) || !strings.Contains(diags[0].Message, `"bogus"`) {
		t.Fatalf("diags = %v", diags)
	}
	if !reflect.DeepEqual(expr, SemanticExpression{}) {
		t.Fatalf("expression must be zero")
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestMachineExprDiagnosticString(t *testing.T) {
	_, diags := ParseMachineExpr("a )", MachineExprBoolean)
	if len(diags) != 1 || !strings.HasPrefix(diags[0].String(), "1:3: ") {
		t.Fatalf("diags = %v", diags)
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestIsMachineKeyword(t *testing.T) {
	requireMachineKeywords(t, []string{"and", "or", "not", "implies", "true", "false", "in_state"}, true)
	requireMachineKeywords(t, []string{"x", "And", "state", ""}, false)
}

// requireMachineKeywords reports every word whose keyword status differs from
// want.
func requireMachineKeywords(t *testing.T, words []string, want bool) {
	t.Helper()
	for _, k := range words {
		if IsMachineKeyword(k) != want {
			t.Errorf("%q must be a keyword: %v", k, want)
		}
	}
}

// machineDiagWant is the one diagnostic a source must produce: its kind, the
// position and a message fragment.
type machineDiagWant struct {
	kind      MachineExprKind
	line, col int
	contains  string
}

// wantMachineDiag parses src and requires exactly one diagnostic at the wanted
// position whose message contains the wanted fragment.
func wantMachineDiag(t *testing.T, src string, want machineDiagWant) {
	t.Helper()
	expr, diags := ParseMachineExpr(src, want.kind)
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %v, want exactly one", diags)
	}
	d := diags[0]
	if d.Code != MachineExprDiagnosticCode || !isMachineDiagAt(d, want.line, want.col) || !strings.Contains(d.Message, want.contains) {
		t.Errorf("diagnostic = %+v, want %d:%d containing %q", d, want.line, want.col, want.contains)
	}
	if !reflect.DeepEqual(expr, SemanticExpression{}) {
		t.Errorf("expression must be zero on error")
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprSourceLengthLimit(t *testing.T) {
	if MaxMachineExprSourceBytes != 8192 {
		t.Fatalf("limit = %d, want 8 KiB", MaxMachineExprSourceBytes)
	}
	atLimit := strings.Repeat("x", MaxMachineExprSourceBytes)
	for _, kind := range []MachineExprKind{MachineExprBoolean, MachineExprEffect} {
		src := atLimit
		if kind == MachineExprEffect {
			src = "v := " + atLimit[5:]
		}
		if _, diags := ParseMachineExpr(src, kind); len(diags) != 0 {
			t.Fatalf("%s source of exactly the limit must parse: %v", kind, diags)
		}
		wantMachineDiag(t, src+"x", machineDiagWant{kind, 1, MaxMachineExprSourceBytes + 1, "8192 bytes"})
	}
	// Line and column of the first byte beyond the limit across lines.
	multi := strings.Repeat("x\n", MaxMachineExprSourceBytes/2)
	if _, diags := ParseMachineExpr(multi, MachineExprBoolean); len(diags) == 0 || diags[0].Message == "" {
		t.Fatalf("multi-line filler must fail on content, not length")
	}
	wantMachineDiag(t, multi+"y", machineDiagWant{MachineExprBoolean, MaxMachineExprSourceBytes/2 + 1, 1, "limit is 8192"})
	// The column counts runes: the byte beyond the limit is the second byte of
	// a two-byte rune that starts at the limit boundary minus one.
	wantMachineDiag(t, strings.Repeat("x", MaxMachineExprSourceBytes-1)+"é", machineDiagWant{MachineExprBoolean, 1, MaxMachineExprSourceBytes, "limit is 8192"})
	// Invalid UTF-8 counts one column per byte, like the lexer.
	wantMachineDiag(t, strings.Repeat("x", MaxMachineExprSourceBytes)+"\xff\xfe", machineDiagWant{MachineExprBoolean, 1, MaxMachineExprSourceBytes + 1, "limit is 8192"})
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprHugeInputIsRejectedQuickly(t *testing.T) {
	for _, filler := range []string{"(", "not ", "-", "a implies ", "x"} {
		src := strings.Repeat(filler, (1<<20)/len(filler))
		start := time.Now()
		wantMachineDiag(t, src, machineDiagWant{MachineExprBoolean, 1, MaxMachineExprSourceBytes + 1, "limit is 8192"})
		wantMachineDiag(t, src, machineDiagWant{MachineExprEffect, 1, MaxMachineExprSourceBytes + 1, "limit is 8192"})
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("rejecting 1 MiB of %q took %v", filler, elapsed)
		}
	}
}

// machineNestingShape builds sources that nest k levels deep around or before
// "x"; col is the column of the first construct beyond the limit (0 when not
// checked) and openOnly marks shapes with no closing tail.
type machineNestingShape struct {
	name     string
	open     func(k int) string
	tail     func(k int) string
	col      int
	openOnly bool
}

func machineNestingShapes() []machineNestingShape {
	n := MaxMachineExprDepth
	none := func(int) string { return "" }
	return []machineNestingShape{
		{"parentheses", func(k int) string { return strings.Repeat("(", k) }, func(k int) string { return strings.Repeat(")", k) }, n + 1, false},
		{"not", func(k int) string { return strings.Repeat("not ", k) }, none, 4*n + 1, false},
		{"unary minus", func(k int) string { return strings.Repeat("-", k) }, none, n + 1, false},
		{"implies", func(k int) string { return strings.Repeat("x implies ", k) }, none, 10*n + 3, true},
		{"mixed", func(k int) string { return strings.Repeat("(not -", k/3) + strings.Repeat("(", k%3) }, func(k int) string { return strings.Repeat(")", k/3+k%3) }, 0, false},
	}
}

// source returns the source nested k levels deep for the expression kind.
func (shape machineNestingShape) source(kind MachineExprKind, k int) string {
	prefix := machineNestingPrefix(kind)
	if shape.openOnly {
		return prefix + shape.open(k) + "x"
	}
	return prefix + shape.open(k) + "x" + shape.tail(k)
}

// machineNestingPrefix is the text before the expression of a source of the
// given kind: an assignment for an effect.
func machineNestingPrefix(kind MachineExprKind) string {
	if kind == MachineExprEffect {
		return "v := "
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprNestingLimit(t *testing.T) {
	if MaxMachineExprDepth != 64 {
		t.Fatalf("depth limit = %d, want 64", MaxMachineExprDepth)
	}
	for _, shape := range machineNestingShapes() {
		t.Run(shape.name, func(t *testing.T) {
			for _, kind := range []MachineExprKind{MachineExprBoolean, MachineExprEffect} {
				checkMachineNestingLimit(t, shape, kind)
			}
		})
	}
	checkMachineNestingSiblingsAndMixes(t)
}

// checkMachineNestingLimit requires the maximum depth to parse and one more
// level to fail with the nesting diagnostic at the expected column.
func checkMachineNestingLimit(t *testing.T, shape machineNestingShape, kind MachineExprKind) {
	t.Helper()
	n := MaxMachineExprDepth
	if _, diags := ParseMachineExpr(shape.source(kind, n), kind); len(diags) != 0 {
		t.Fatalf("depth %d must parse: %v", n, diags)
	}
	_, diags := ParseMachineExpr(shape.source(kind, n+1), kind)
	requireMachineNestingDiagnostic(t, diags)
	want := len(machineNestingPrefix(kind)) + shape.col
	if shape.col != 0 && diags[0].Column != want {
		t.Errorf("column = %d, want %d", diags[0].Column, want)
	}
}

// requireMachineNestingDiagnostic requires exactly one nesting diagnostic on
// line 1 that names the depth limit.
func requireMachineNestingDiagnostic(t *testing.T, diags []MachineExprDiagnostic) {
	t.Helper()
	if len(diags) != 1 || diags[0].Code != MachineExprDiagnosticCode || diags[0].Line != 1 {
		t.Fatalf("depth %d: diagnostics = %v", MaxMachineExprDepth+1, diags)
	}
	if !strings.Contains(diags[0].Message, "nested too deeply") || !strings.Contains(diags[0].Message, "64") {
		t.Fatalf("depth %d: diagnostics = %v", MaxMachineExprDepth+1, diags)
	}
}

// checkMachineNestingSiblingsAndMixes covers released levels, prefix operators
// inside parentheses and the order of nesting and syntax errors.
func checkMachineNestingSiblingsAndMixes(t *testing.T) {
	t.Helper()
	// Levels are released: siblings never add up.
	sequential := strings.Repeat("(x) and ", 500) + "not (x) and -(1) == (2) implies (a implies b)"
	if _, diags := ParseMachineExpr(sequential, MachineExprBoolean); len(diags) != 0 {
		t.Fatalf("sequential groups must parse: %v", diags)
	}
	// Prefix operators inside parentheses count together.
	if _, diags := ParseMachineExpr(strings.Repeat("(", 32)+strings.Repeat("not ", 32)+"x"+strings.Repeat(")", 32), MachineExprBoolean); len(diags) != 0 {
		t.Fatalf("32 + 32 levels must parse: %v", diags)
	}
	tooDeep := strings.Repeat("(", 32) + strings.Repeat("not ", 33) + "x" + strings.Repeat(")", 32)
	wantMachineDiag(t, tooDeep, machineDiagWant{MachineExprBoolean, 1, 32 + 4*32 + 1, "nested too deeply"})
	// A deep input of the maximum length fails with the nesting diagnostic.
	wantMachineDiag(t, strings.Repeat("(", MaxMachineExprSourceBytes), machineDiagWant{MachineExprBoolean, 1, MaxMachineExprDepth + 1, "nested too deeply"})
	// An earlier syntax error is reported before the nesting problem.
	wantMachineDiag(t, "$"+strings.Repeat("(", 100), machineDiagWant{MachineExprBoolean, 1, 1, "unexpected character"})
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprLongLeftChainsAreBounded(t *testing.T) {
	// "a+a+...+a" of exactly the limit: 4096 operands, a left-nested tree of
	// height 4095. Every consumer handles it.
	src := strings.Repeat("a+", (MaxMachineExprSourceBytes-2)/2) + "a"
	if len(src) >= MaxMachineExprSourceBytes {
		t.Fatalf("test source has %d bytes", len(src))
	}
	expr := mustParse(t, src, MachineExprBoolean)
	height := 0
	for e := expr; e.Operator == "+"; e = e.Operands[0] {
		height++
	}
	if height != (MaxMachineExprSourceBytes-2)/2 {
		t.Fatalf("height = %d", height)
	}
	checkMachineChainConsumers(t, expr, height)
	checkMachineChainJSON(t, expr)
	wantMachineDiag(t, src+"+a", machineDiagWant{MachineExprBoolean, 1, MaxMachineExprSourceBytes + 1, "limit is 8192"})
}

// checkMachineChainConsumers requires the variable collector and the printer
// to handle a left chain of the given height.
func checkMachineChainConsumers(t *testing.T, expr SemanticExpression, height int) {
	t.Helper()
	if got := MachineExprVariables(expr); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("variables = %v", got)
	}
	if printed := MachineExprString(expr); printed != strings.Repeat("a + ", height)+"a" {
		t.Errorf("printed form starts %.20q", printed)
	}
}

// checkMachineChainJSON requires the longest chain to survive a JSON round trip.
func checkMachineChainJSON(t *testing.T, expr SemanticExpression) {
	t.Helper()
	raw, err := json.Marshal(expr)
	if err != nil {
		t.Fatal(err)
	}
	var back SemanticExpression
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("JSON round trip of the longest chain: %v", err)
	}
	if !reflect.DeepEqual(expr, back) {
		t.Errorf("JSON round trip changed the longest chain")
	}
}

// isMachineMinimumInteger reports whether e is the int64 minimum literal.
func isMachineMinimumInteger(e SemanticExpression) bool {
	if e.Kind != MachineKindLiteral || e.Operator != "" {
		return false
	}
	return e.TypedValue != nil && e.TypedValue.Integer != nil && *e.TypedValue.Integer == math.MinInt64
}

var machineMinimumIntegerCases = []machineTreeCase{
	{"-9223372036854775808", "min"},
	{"- 9223372036854775808", "min"},
	{"-0009223372036854775808", "min"},
	{"x == -9223372036854775808", "(== x min)"},
	{"x - -9223372036854775808", "(- x min)"},
	{"-9223372036854775808 * 2", "(* min 2)"},
	{"--9223372036854775808", "(neg min)"},
	{"-(-9223372036854775808)", "(neg min)"},
	{"-9223372036854775807", "(neg 9223372036854775807)"},
	{"9223372036854775807", "9223372036854775807"},
}

// checkMachineMinimumIntegerCase parses the case, requires its prefix form and
// that its printed form re-parses to the same tree.
func checkMachineMinimumIntegerCase(t *testing.T, tc machineTreeCase) {
	t.Helper()
	expr := mustParse(t, tc.src, MachineExprBoolean)
	got := strings.ReplaceAll(sexpr(expr), "-9223372036854775808", "min")
	if got != tc.want {
		t.Errorf("%q = %s, want %s", tc.src, got, tc.want)
	}
	if tc.want == "min" && !isMachineMinimumInteger(expr) {
		t.Errorf("%q is not the int64 minimum literal: %s", tc.src, sexpr(expr))
	}
	printed := MachineExprString(expr)
	if again := mustParse(t, printed, MachineExprBoolean); !reflect.DeepEqual(expr, again) {
		t.Errorf("%q printed as %q re-parses to a different tree", tc.src, printed)
	}
}

// machineOutOfRangeCases are sources whose integer is out of range at the
// given column.
var machineOutOfRangeCases = []struct {
	src  string
	kind MachineExprKind
	col  int
	msg  string
}{
	{"9223372036854775808", MachineExprBoolean, 1, "out of range"},
	{"0009223372036854775808", MachineExprBoolean, 1, "out of range"},
	{"x - 9223372036854775808", MachineExprBoolean, 5, "out of range"},
	{"-(9223372036854775808)", MachineExprBoolean, 3, "out of range"},
	{"x := 9223372036854775808", MachineExprEffect, 6, "out of range"},
	{"-9223372036854775809", MachineExprBoolean, 2, "out of range"},
	{"9223372036854775809", MachineExprBoolean, 1, "out of range"},
	{"-99999999999999999999", MachineExprBoolean, 2, "out of range"},
	{"9223372036854775808x", MachineExprBoolean, 1, "digits followed by a letter"},
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprMinimumInteger(t *testing.T) {
	for _, tc := range machineMinimumIntegerCases {
		checkMachineMinimumIntegerCase(t, tc)
	}
	effect := mustParse(t, "x := -9223372036854775808", MachineExprEffect)
	if !isMachineMinimumInteger(effect.Operands[0].Operands[1]) {
		t.Errorf("effect value = %s", sexpr(effect))
	}
	// The magnitude alone and anything larger stay out of range.
	for _, tc := range machineOutOfRangeCases {
		wantMachineDiag(t, tc.src, machineDiagWant{tc.kind, 1, tc.col, tc.msg})
	}
}

// machineExprFuzzSeeds are the seed sources of FuzzParseMachineExpr.
func machineExprFuzzSeeds() []string {
	return []string{
		"", "a", "a and b or c", "not (a == b)", "a implies b implies c", "-a * (b + 3) >= 0",
		`s == "x\"y\\z"`, "in_state(STATE-PAY-CLOSED) and attempts < 3", "(attempts <= 3) and not in_state(S-1)",
		`attempts := attempts + 1; status := "retry"`, "x := 1; x := 2", "((", "a :=", "a b", "\"", "in_state(", "9999999999999999999",
		"a\n and\n b", "a < b < c", "- - -1", "x := y implies z;",
		"-9223372036854775808", "x - -9223372036854775808", "9223372036854775808", "-(9223372036854775808)",
		strings.Repeat("(", 63) + "x" + strings.Repeat(")", 63), strings.Repeat("(", 100), strings.Repeat("not ", 100) + "x",
		strings.Repeat("-", 100) + "1", strings.Repeat("x implies ", 100) + "x", strings.Repeat("(not -", 40) + "x",
		strings.Repeat("a+", 4000) + "a", strings.Repeat("(", MaxMachineExprSourceBytes+1), strings.Repeat("x", MaxMachineExprSourceBytes+1),
	}
}

// TRLC-LINKS: REQ-EMG-136
func FuzzParseMachineExpr(f *testing.F) {
	for _, seed := range machineExprFuzzSeeds() {
		f.Add(seed, true)
		f.Add(seed, false)
	}
	f.Fuzz(func(t *testing.T, src string, boolean bool) {
		kind := MachineExprEffect
		if boolean {
			kind = MachineExprBoolean
		}
		checkMachineExprProperties(t, src, kind)
	})
}

// checkMachineExprProperties requires well-formed diagnostics for a source
// that fails and a stable print-then-parse cycle for one that parses.
func checkMachineExprProperties(t *testing.T, src string, kind MachineExprKind) {
	t.Helper()
	expr, diags := ParseMachineExpr(src, kind)
	if len(diags) > 0 {
		checkMachineDiagnosticsWellFormed(t, src, diags)
		return
	}
	printed := MachineExprString(expr)
	if len(printed) > MaxMachineExprSourceBytes {
		// Canonical spacing can push a valid source past the length limit.
		return
	}
	again, diags := ParseMachineExpr(printed, kind)
	if len(diags) > 0 {
		t.Fatalf("printed form %q of %q does not re-parse: %v", printed, src, diags)
	}
	if !reflect.DeepEqual(expr, again) {
		t.Fatalf("print-then-parse changed the tree of %q (printed %q)", src, printed)
	}
	if second := MachineExprString(again); second != printed {
		t.Fatalf("printing is not idempotent: %q vs %q", printed, second)
	}
	_ = MachineExprVariables(expr)
	_ = MachineExprStateRefs(expr)
}

// checkMachineDiagnosticsWellFormed requires a position, the code and a
// message on every diagnostic.
func checkMachineDiagnosticsWellFormed(t *testing.T, src string, diags []MachineExprDiagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.Line < 1 || d.Column < 1 {
			t.Fatalf("bad diagnostic %+v for %q", d, src)
		}
		if d.Code != MachineExprDiagnosticCode || d.Message == "" {
			t.Fatalf("bad diagnostic %+v for %q", d, src)
		}
	}
}
