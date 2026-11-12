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
	parts := []string{e.Operator}
	for _, o := range e.Operands {
		parts = append(parts, sexpr(o))
	}
	return "(" + strings.Join(parts, " ") + ")"
}

func mustParse(t *testing.T, src string, kind MachineExprKind) SemanticExpression {
	t.Helper()
	expr, diags := ParseMachineExpr(src, kind)
	if len(diags) != 0 {
		t.Fatalf("parse %q: unexpected diagnostics %v", src, diags)
	}
	return expr
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprBooleanTrees(t *testing.T) {
	tests := []struct {
		src, want string
	}{
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
	for _, tc := range tests {
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
	var leaves []SemanticExpression
	var walk func(SemanticExpression)
	walk = func(e SemanticExpression) {
		if e.Operator == "" {
			leaves = append(leaves, e)
		}
		if e.Operator != "" && e.Kind != MachineKindOperator {
			t.Errorf("operator node %q has kind %q", e.Operator, e.Kind)
		}
		for _, o := range e.Operands {
			walk(o)
		}
	}
	walk(expr)
	kinds := map[string]int{}
	for _, l := range leaves {
		switch {
		case l.Kind == MachineKindVariable:
			kinds["variable"]++
			if l.TypedValue != nil || l.Value != "x" {
				t.Errorf("bad variable leaf %+v", l)
			}
		case l.Kind == MachineKindReference:
			kinds["reference"]++
			if l.TypedValue == nil || l.TypedValue.Kind != "reference" || l.TypedValue.Reference != "S-1" {
				t.Errorf("bad reference leaf %+v", l)
			}
		case l.Kind == MachineKindLiteral:
			v := l.TypedValue
			if v == nil {
				t.Fatalf("literal without typed value: %+v", l)
			}
			kinds[v.Kind]++
			populated := 0
			for _, ok := range []bool{v.String != "", v.Integer != nil, v.Boolean != nil, v.Real != nil, v.Reference != "", v.Quantity != nil} {
				if ok {
					populated++
				}
			}
			if populated != 1 {
				t.Errorf("literal %+v populates %d fields", v, populated)
			}
		default:
			t.Errorf("unexpected leaf %+v", l)
		}
	}
	want := map[string]int{"variable": 1, "reference": 1, "integer": 2, "string": 2, "boolean": 1}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("leaf kinds = %v, want %v", kinds, want)
	}
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
		var diags []SemanticDiagnostic
		validateExpression(&expr, "guard", identities, &diags)
		if len(diags) != 0 {
			t.Errorf("%q: structural diagnostics %+v", tc.src, diags)
		}
	}
	expr := mustParse(t, "in_state(STATE-NOPE)", MachineExprBoolean)
	var diags []SemanticDiagnostic
	validateExpression(&expr, "guard", identities, &diags)
	if len(diags) != 1 || diags[0].Code != "semantic.dangling_reference" {
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

// TRLC-LINKS: REQ-EMG-138
func TestMachineOperatorTable(t *testing.T) {
	want := []string{"implies", "or", "and", "not", "==", "!=", "<", "<=", ">", ">=", "+", "-", "*", "neg", "in_state", ":=", "all"}
	var names []string
	for _, op := range MachineOperators() {
		names = append(names, op.Name)
		if len(op.Operands) == 0 || op.Result == "" {
			t.Errorf("operator %q has no signature: %+v", op.Name, op)
		}
		if op.Form == MachineFormInfix && len(op.Operands) != 2 || op.Form == MachineFormPrefix && len(op.Operands) != 1 {
			t.Errorf("operator %q arity does not match form %q", op.Name, op.Form)
		}
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("operators = %v, want %v", names, want)
	}
	for _, tc := range []struct {
		name     string
		operands []MachineType
		result   MachineType
		sameType bool
	}{
		{"and", []MachineType{MachineTypeBoolean, MachineTypeBoolean}, MachineTypeBoolean, false},
		{"==", []MachineType{MachineTypeAny, MachineTypeAny}, MachineTypeBoolean, true},
		{"<=", []MachineType{MachineTypeInteger, MachineTypeInteger}, MachineTypeBoolean, false},
		{"+", []MachineType{MachineTypeInteger, MachineTypeInteger}, MachineTypeInteger, false},
		{"neg", []MachineType{MachineTypeInteger}, MachineTypeInteger, false},
		{"in_state", []MachineType{MachineTypeState}, MachineTypeBoolean, false},
		{":=", []MachineType{MachineTypeAny, MachineTypeAny}, MachineTypeEffect, true},
	} {
		op, ok := LookupMachineOperator(tc.name)
		if !ok || !reflect.DeepEqual(op.Operands, tc.operands) || op.Result != tc.result || op.SameType != tc.sameType {
			t.Errorf("operator %q = %+v (found %v)", tc.name, op, ok)
		}
	}
	if all, _ := LookupMachineOperator("all"); !all.Variadic || all.Result != MachineTypeEffect {
		t.Errorf("all = %+v", all)
	}
	if imp, _ := LookupMachineOperator("implies"); !imp.RightAssociative {
		t.Errorf("implies must be right associative")
	}
	if _, ok := LookupMachineOperator("nope"); ok {
		t.Errorf("unknown operator found")
	}
	// The table must be a copy: callers cannot corrupt it.
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

// TRLC-LINKS: REQ-EMG-138
func TestMachineOperatorsMatchParserPrecedence(t *testing.T) {
	table := map[string]MachineOperator{}
	for _, op := range MachineOperators() {
		table[op.Name] = op
	}
	// Each case parses src and checks that the child of the root at operand
	// index "at" is the operator "inner". The case passes only if the parser
	// bound inner tighter than outer AND the table says the same, so swapping
	// two binding levels in the parser fails the test. Every adjacent pair of
	// levels is covered from both sides.
	for _, tc := range []struct {
		src, outer string
		at         int
		inner      string
	}{
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
	} {
		t.Run(tc.src, func(t *testing.T) {
			expr := mustParse(t, tc.src, MachineExprBoolean)
			if expr.Operator != tc.outer || len(expr.Operands) <= tc.at || expr.Operands[tc.at].Operator != tc.inner {
				t.Fatalf("parse %q = %s, want %s with %s as operand %d", tc.src, sexpr(expr), tc.outer, tc.inner, tc.at)
			}
			outer, inner := table[tc.outer], table[tc.inner]
			if outer.Name != tc.outer || inner.Name != tc.inner {
				t.Fatalf("operator table lacks %q or %q", tc.outer, tc.inner)
			}
			if tc.outer == tc.inner || outer.Precedence == inner.Precedence {
				// Same level: the table associativity must agree with the
				// side the parser nested on.
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
				return
			}
			if inner.Precedence <= outer.Precedence {
				t.Errorf("parser binds %s tighter than %s but the table has %d <= %d", tc.inner, tc.outer, inner.Precedence, outer.Precedence)
			}
		})
	}
	// Operators of one level share a precedence.
	for _, group := range [][]string{{"==", "!=", "<", "<=", ">", ">="}, {"+", "-"}} {
		for _, name := range group[1:] {
			if table[name].Precedence != table[group[0]].Precedence {
				t.Errorf("%s and %s must share a precedence level", name, group[0])
			}
		}
	}
	// Comparisons are non-associative in the parser: no pair nests, and the
	// table never marks one right associative.
	for _, a := range []string{"==", "!=", "<", "<=", ">", ">="} {
		if table[a].RightAssociative {
			t.Errorf("%s must not be right associative", a)
		}
		for _, b := range []string{"==", "<"} {
			_, diags := ParseMachineExpr("x "+a+" y "+b+" z", MachineExprBoolean)
			if len(diags) != 1 || !strings.Contains(diags[0].Message, "not associative") {
				t.Errorf("x %s y %s z: diagnostics %v, want non-associative error", a, b, diags)
			}
		}
	}
	// A comparison operand may itself be arithmetic but never a comparison;
	// parentheses restore it.
	if expr := mustParse(t, "(a < b) == (c < d)", MachineExprBoolean); expr.Operator != "==" || expr.Operands[0].Operator != "<" || expr.Operands[1].Operator != "<" {
		t.Errorf("parenthesised comparisons = %s", sexpr(expr))
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

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprDiagnostics(t *testing.T) {
	tests := []machineErrCase{
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, diags := ParseMachineExpr(tc.src, tc.kind)
			if len(diags) == 0 {
				t.Fatalf("parse %q succeeded: %s", tc.src, sexpr(expr))
			}
			if !reflect.DeepEqual(expr, SemanticExpression{}) {
				t.Errorf("expression must be zero on error, got %s", sexpr(expr))
			}
			d := diags[0]
			if d.Code != "semantic.invalid_expression" {
				t.Errorf("code = %q", d.Code)
			}
			if d.Line != tc.line || d.Column != tc.col {
				t.Errorf("position = %d:%d, want %d:%d (%s)", d.Line, d.Column, tc.line, tc.col, d.Message)
			}
			if !strings.Contains(d.Message, tc.contains) {
				t.Errorf("message %q does not contain %q", d.Message, tc.contains)
			}
			if len(diags) != 1+tc.extraDiags {
				t.Errorf("got %d diagnostics %v", len(diags), diags)
			}
		})
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

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprUnknownKind(t *testing.T) {
	expr, diags := ParseMachineExpr("a", MachineExprKind("bogus"))
	if len(diags) != 1 || diags[0].Line != 1 || diags[0].Column != 1 || !strings.Contains(diags[0].Message, `"bogus"`) {
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
	for _, k := range []string{"and", "or", "not", "implies", "true", "false", "in_state"} {
		if !IsMachineKeyword(k) {
			t.Errorf("%q must be a keyword", k)
		}
	}
	for _, k := range []string{"x", "And", "state", ""} {
		if IsMachineKeyword(k) {
			t.Errorf("%q must not be a keyword", k)
		}
	}
}

// wantMachineDiag parses src and requires exactly one diagnostic at line:col
// whose message contains contains.
func wantMachineDiag(t *testing.T, src string, kind MachineExprKind, line, col int, contains string) {
	t.Helper()
	expr, diags := ParseMachineExpr(src, kind)
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %v, want exactly one", diags)
	}
	d := diags[0]
	if d.Code != MachineExprDiagnosticCode || d.Line != line || d.Column != col || !strings.Contains(d.Message, contains) {
		t.Errorf("diagnostic = %+v, want %d:%d containing %q", d, line, col, contains)
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
		wantMachineDiag(t, src+"x", kind, 1, MaxMachineExprSourceBytes+1, "8192 bytes")
	}
	// Line and column of the first byte beyond the limit across lines.
	multi := strings.Repeat("x\n", MaxMachineExprSourceBytes/2)
	if _, diags := ParseMachineExpr(multi, MachineExprBoolean); len(diags) == 0 || diags[0].Message == "" {
		t.Fatalf("multi-line filler must fail on content, not length")
	}
	wantMachineDiag(t, multi+"y", MachineExprBoolean, MaxMachineExprSourceBytes/2+1, 1, "limit is 8192")
	// The column counts runes: the byte beyond the limit is the second byte of
	// a two-byte rune that starts at the limit boundary minus one.
	wantMachineDiag(t, strings.Repeat("x", MaxMachineExprSourceBytes-1)+"é", MachineExprBoolean, 1, MaxMachineExprSourceBytes, "limit is 8192")
	// Invalid UTF-8 counts one column per byte, like the lexer.
	wantMachineDiag(t, strings.Repeat("x", MaxMachineExprSourceBytes)+"\xff\xfe", MachineExprBoolean, 1, MaxMachineExprSourceBytes+1, "limit is 8192")
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprHugeInputIsRejectedQuickly(t *testing.T) {
	for _, filler := range []string{"(", "not ", "-", "a implies ", "x"} {
		src := strings.Repeat(filler, (1<<20)/len(filler))
		start := time.Now()
		wantMachineDiag(t, src, MachineExprBoolean, 1, MaxMachineExprSourceBytes+1, "limit is 8192")
		wantMachineDiag(t, src, MachineExprEffect, 1, MaxMachineExprSourceBytes+1, "limit is 8192")
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("rejecting 1 MiB of %q took %v", filler, elapsed)
		}
	}
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprNestingLimit(t *testing.T) {
	if MaxMachineExprDepth != 64 {
		t.Fatalf("depth limit = %d, want 64", MaxMachineExprDepth)
	}
	n := MaxMachineExprDepth
	for _, tc := range []struct {
		name string
		open func(k int) string // k nested levels around or before "x"
		tail func(k int) string
		// column of the first construct beyond the limit
		col func() int
	}{
		{"parentheses", func(k int) string { return strings.Repeat("(", k) }, func(k int) string { return strings.Repeat(")", k) }, func() int { return n + 1 }},
		{"not", func(k int) string { return strings.Repeat("not ", k) }, func(int) string { return "" }, func() int { return 4*n + 1 }},
		{"unary minus", func(k int) string { return strings.Repeat("-", k) }, func(int) string { return "" }, func() int { return n + 1 }},
		{"implies", func(k int) string { return strings.Repeat("x implies ", k) }, func(int) string { return "" }, func() int { return 10*n + 3 }},
		{"mixed", func(k int) string { return strings.Repeat("(not -", k/3) + strings.Repeat("(", k%3) }, func(k int) string { return strings.Repeat(")", k/3+k%3) }, func() int { return 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []MachineExprKind{MachineExprBoolean, MachineExprEffect} {
				prefix := ""
				if kind == MachineExprEffect {
					prefix = "v := "
				}
				ok := prefix + tc.open(n) + "x" + tc.tail(n)
				if tc.name == "implies" {
					ok = prefix + tc.open(n) + "x"
				}
				if _, diags := ParseMachineExpr(ok, kind); len(diags) != 0 {
					t.Fatalf("depth %d must parse: %v", n, diags)
				}
				tooDeep := prefix + tc.open(n+1) + "x" + tc.tail(n+1)
				if tc.name == "implies" {
					tooDeep = prefix + tc.open(n+1) + "x"
				}
				_, diags := ParseMachineExpr(tooDeep, kind)
				if len(diags) != 1 || diags[0].Code != MachineExprDiagnosticCode || diags[0].Line != 1 || !strings.Contains(diags[0].Message, "nested too deeply") || !strings.Contains(diags[0].Message, "64") {
					t.Fatalf("depth %d: diagnostics = %v", n+1, diags)
				}
				if col := tc.col(); col != 0 && diags[0].Column != len(prefix)+col {
					t.Errorf("column = %d, want %d", diags[0].Column, len(prefix)+col)
				}
			}
		})
	}
	// Levels are released: siblings never add up.
	sequential := strings.Repeat("(x) and ", 500) + "not (x) and -(1) == (2) implies (a implies b)"
	if _, diags := ParseMachineExpr(sequential, MachineExprBoolean); len(diags) != 0 {
		t.Fatalf("sequential groups must parse: %v", diags)
	}
	// Prefix operators inside parentheses count together.
	if _, diags := ParseMachineExpr(strings.Repeat("(", 32)+strings.Repeat("not ", 32)+"x"+strings.Repeat(")", 32), MachineExprBoolean); len(diags) != 0 {
		t.Fatalf("32 + 32 levels must parse: %v", diags)
	}
	wantMachineDiag(t, strings.Repeat("(", 32)+strings.Repeat("not ", 33)+"x"+strings.Repeat(")", 32), MachineExprBoolean, 1, 32+4*32+1, "nested too deeply")
	// A deep input of the maximum length fails with the nesting diagnostic.
	wantMachineDiag(t, strings.Repeat("(", MaxMachineExprSourceBytes), MachineExprBoolean, 1, MaxMachineExprDepth+1, "nested too deeply")
	// An earlier syntax error is reported before the nesting problem.
	wantMachineDiag(t, "$"+strings.Repeat("(", 100), MachineExprBoolean, 1, 1, "unexpected character")
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
	if got := MachineExprVariables(expr); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("variables = %v", got)
	}
	if printed := MachineExprString(expr); printed != strings.Repeat("a + ", height)+"a" {
		t.Errorf("printed form starts %.20q", printed)
	}
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
	wantMachineDiag(t, src+"+a", MachineExprBoolean, 1, MaxMachineExprSourceBytes+1, "limit is 8192")
}

// TRLC-LINKS: REQ-EMG-136
func TestParseMachineExprMinimumInteger(t *testing.T) {
	min := func(e SemanticExpression) bool {
		return e.Kind == MachineKindLiteral && e.Operator == "" && e.TypedValue != nil && e.TypedValue.Integer != nil && *e.TypedValue.Integer == math.MinInt64
	}
	for _, tc := range []struct{ src, want string }{
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
	} {
		expr := mustParse(t, tc.src, MachineExprBoolean)
		got := sexpr(expr)
		got = strings.ReplaceAll(got, "-9223372036854775808", "min")
		if got != tc.want {
			t.Errorf("%q = %s, want %s", tc.src, got, tc.want)
		}
		if tc.want == "min" && !min(expr) {
			t.Errorf("%q is not the int64 minimum literal: %s", tc.src, sexpr(expr))
		}
		printed := MachineExprString(expr)
		if again := mustParse(t, printed, MachineExprBoolean); !reflect.DeepEqual(expr, again) {
			t.Errorf("%q printed as %q re-parses to a different tree", tc.src, printed)
		}
	}
	effect := mustParse(t, "x := -9223372036854775808", MachineExprEffect)
	if !min(effect.Operands[0].Operands[1]) {
		t.Errorf("effect value = %s", sexpr(effect))
	}
	// The magnitude alone and anything larger stay out of range.
	wantMachineDiag(t, "9223372036854775808", MachineExprBoolean, 1, 1, "out of range")
	wantMachineDiag(t, "0009223372036854775808", MachineExprBoolean, 1, 1, "out of range")
	wantMachineDiag(t, "x - 9223372036854775808", MachineExprBoolean, 1, 5, "out of range")
	wantMachineDiag(t, "-(9223372036854775808)", MachineExprBoolean, 1, 3, "out of range")
	wantMachineDiag(t, "x := 9223372036854775808", MachineExprEffect, 1, 6, "out of range")
	wantMachineDiag(t, "-9223372036854775809", MachineExprBoolean, 1, 2, "out of range")
	wantMachineDiag(t, "9223372036854775809", MachineExprBoolean, 1, 1, "out of range")
	wantMachineDiag(t, "-99999999999999999999", MachineExprBoolean, 1, 2, "out of range")
	wantMachineDiag(t, "9223372036854775808x", MachineExprBoolean, 1, 1, "digits followed by a letter")
}

// TRLC-LINKS: REQ-EMG-136
func FuzzParseMachineExpr(f *testing.F) {
	for _, seed := range []string{
		"", "a", "a and b or c", "not (a == b)", "a implies b implies c", "-a * (b + 3) >= 0",
		`s == "x\"y\\z"`, "in_state(STATE-PAY-CLOSED) and attempts < 3", "(attempts <= 3) and not in_state(S-1)",
		`attempts := attempts + 1; status := "retry"`, "x := 1; x := 2", "((", "a :=", "a b", "\"", "in_state(", "9999999999999999999",
		"a\n and\n b", "a < b < c", "- - -1", "x := y implies z;",
		"-9223372036854775808", "x - -9223372036854775808", "9223372036854775808", "-(9223372036854775808)",
		strings.Repeat("(", 63) + "x" + strings.Repeat(")", 63), strings.Repeat("(", 100), strings.Repeat("not ", 100) + "x",
		strings.Repeat("-", 100) + "1", strings.Repeat("x implies ", 100) + "x", strings.Repeat("(not -", 40) + "x",
		strings.Repeat("a+", 4000) + "a", strings.Repeat("(", MaxMachineExprSourceBytes+1), strings.Repeat("x", MaxMachineExprSourceBytes+1),
	} {
		f.Add(seed, true)
		f.Add(seed, false)
	}
	f.Fuzz(func(t *testing.T, src string, boolean bool) {
		kind := MachineExprEffect
		if boolean {
			kind = MachineExprBoolean
		}
		expr, diags := ParseMachineExpr(src, kind)
		if len(diags) > 0 {
			for _, d := range diags {
				if d.Line < 1 || d.Column < 1 || d.Code != MachineExprDiagnosticCode || d.Message == "" {
					t.Fatalf("bad diagnostic %+v for %q", d, src)
				}
			}
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
	})
}
