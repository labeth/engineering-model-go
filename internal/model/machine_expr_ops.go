// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// MachineType is a value type of the machine expression language. The parser
// does not check types; the validator and generators use the operator table.
type MachineType string

const (
	MachineTypeInteger MachineType = "Integer"
	MachineTypeBoolean MachineType = "Boolean"
	MachineTypeString  MachineType = "String"
	// MachineTypeAny is any value type; with SameType every Any operand of
	// one operator application must have the same concrete type.
	MachineTypeAny MachineType = "Any"
	// MachineTypeState is a state ID reference (the in_state operand).
	MachineTypeState MachineType = "State"
	// MachineTypeEffect is the result of an assignment or of "all".
	MachineTypeEffect MachineType = "Effect"
)

// MachineOperatorForm tells how an operator is written in source text.
type MachineOperatorForm string

const (
	MachineFormInfix    MachineOperatorForm = "infix"    // a op b
	MachineFormPrefix   MachineOperatorForm = "prefix"   // op a
	MachineFormCall     MachineOperatorForm = "call"     // op(a)
	MachineFormAssign   MachineOperatorForm = "assign"   // variable := a
	MachineFormSequence MachineOperatorForm = "sequence" // a; b; c
)

// MachineOperator describes one operator of the machine expression tree.
// The Quint generator and the validator share this table.
//
// TRLC-LINKS: REQ-EMG-138
type MachineOperator struct {
	// Name is the SemanticExpression.Operator value.
	Name string
	// Syntax is the source token ("and", "<=", ...); empty for "all", which
	// is written as ";"-separated assignments.
	Syntax string
	Form   MachineOperatorForm
	// Operands lists the operand types. For a variadic operator it holds the
	// type of every operand.
	Operands []MachineType
	Variadic bool
	// SameType requires all MachineTypeAny operands to have one concrete
	// type. For ":=" the first operand is the assigned variable and the
	// second its new value.
	SameType bool
	Result   MachineType
	// Precedence is higher for tighter binding; 0 for non-expression forms.
	Precedence int
	// RightAssociative is set for implies; comparisons are non-associative.
	RightAssociative bool
}

// Precedence levels, low to high.
const (
	machinePrecImplies = 1 + iota
	machinePrecOr
	machinePrecAnd
	machinePrecNot
	machinePrecComparison
	machinePrecAdditive
	machinePrecMultiplicative
	machinePrecUnary
	machinePrecPrimary
)

var machineOperatorTable = buildMachineOperatorTable()

// TRLC-LINKS: REQ-EMG-138
func buildMachineOperatorTable() []MachineOperator {
	bool2 := []MachineType{MachineTypeBoolean, MachineTypeBoolean}
	int2 := []MachineType{MachineTypeInteger, MachineTypeInteger}
	table := []MachineOperator{
		{Name: "implies", Syntax: "implies", Form: MachineFormInfix, Operands: bool2, Result: MachineTypeBoolean, Precedence: machinePrecImplies, RightAssociative: true},
		{Name: "or", Syntax: "or", Form: MachineFormInfix, Operands: bool2, Result: MachineTypeBoolean, Precedence: machinePrecOr},
		{Name: "and", Syntax: "and", Form: MachineFormInfix, Operands: bool2, Result: MachineTypeBoolean, Precedence: machinePrecAnd},
		{Name: "not", Syntax: "not", Form: MachineFormPrefix, Operands: []MachineType{MachineTypeBoolean}, Result: MachineTypeBoolean, Precedence: machinePrecNot},
		{Name: "==", Syntax: "==", Form: MachineFormInfix, Operands: []MachineType{MachineTypeAny, MachineTypeAny}, SameType: true, Result: MachineTypeBoolean, Precedence: machinePrecComparison},
		{Name: "!=", Syntax: "!=", Form: MachineFormInfix, Operands: []MachineType{MachineTypeAny, MachineTypeAny}, SameType: true, Result: MachineTypeBoolean, Precedence: machinePrecComparison},
		{Name: "<", Syntax: "<", Form: MachineFormInfix, Operands: int2, Result: MachineTypeBoolean, Precedence: machinePrecComparison},
		{Name: "<=", Syntax: "<=", Form: MachineFormInfix, Operands: int2, Result: MachineTypeBoolean, Precedence: machinePrecComparison},
		{Name: ">", Syntax: ">", Form: MachineFormInfix, Operands: int2, Result: MachineTypeBoolean, Precedence: machinePrecComparison},
		{Name: ">=", Syntax: ">=", Form: MachineFormInfix, Operands: int2, Result: MachineTypeBoolean, Precedence: machinePrecComparison},
		{Name: "+", Syntax: "+", Form: MachineFormInfix, Operands: int2, Result: MachineTypeInteger, Precedence: machinePrecAdditive},
		{Name: "-", Syntax: "-", Form: MachineFormInfix, Operands: int2, Result: MachineTypeInteger, Precedence: machinePrecAdditive},
		{Name: "*", Syntax: "*", Form: MachineFormInfix, Operands: int2, Result: MachineTypeInteger, Precedence: machinePrecMultiplicative},
		{Name: "neg", Syntax: "-", Form: MachineFormPrefix, Operands: []MachineType{MachineTypeInteger}, Result: MachineTypeInteger, Precedence: machinePrecUnary},
		{Name: "in_state", Syntax: "in_state", Form: MachineFormCall, Operands: []MachineType{MachineTypeState}, Result: MachineTypeBoolean, Precedence: machinePrecPrimary},
		{Name: ":=", Syntax: ":=", Form: MachineFormAssign, Operands: []MachineType{MachineTypeAny, MachineTypeAny}, SameType: true, Result: MachineTypeEffect},
		{Name: "all", Form: MachineFormSequence, Operands: []MachineType{MachineTypeEffect}, Variadic: true, Result: MachineTypeEffect},
	}
	return table
}

// MachineOperators returns a copy of the operator table in precedence order,
// loosest binding first, with the effect operators last.
//
// TRLC-LINKS: REQ-EMG-138
func MachineOperators() []MachineOperator {
	out := make([]MachineOperator, len(machineOperatorTable))
	for i, op := range machineOperatorTable {
		op.Operands = append([]MachineType(nil), op.Operands...)
		out[i] = op
	}
	return out
}

// LookupMachineOperator returns the table entry for a SemanticExpression
// operator name. Only the found entry is copied, so a lookup per tree node does
// not copy the whole table.
//
// TRLC-LINKS: REQ-EMG-138
func LookupMachineOperator(name string) (MachineOperator, bool) {
	for _, op := range machineOperatorTable {
		if op.Name == name {
			op.Operands = append([]MachineType(nil), op.Operands...)
			return op, true
		}
	}
	return MachineOperator{}, false
}

// MachineExprVariables returns the sorted, de-duplicated names of every
// variable leaf in expr, whether read or assigned.
//
// TRLC-LINKS: REQ-EMG-136
func MachineExprVariables(expr SemanticExpression) []string {
	return collectMachineLeaves(expr, func(e SemanticExpression) string {
		if e.Kind == MachineKindVariable {
			return e.Value
		}
		return ""
	})
}

// MachineExprStateRefs returns the sorted, de-duplicated state IDs that expr
// tests with in_state.
//
// TRLC-LINKS: REQ-EMG-136
func MachineExprStateRefs(expr SemanticExpression) []string {
	return collectMachineLeaves(expr, func(e SemanticExpression) string {
		if e.Kind == MachineKindReference && e.TypedValue != nil {
			return e.TypedValue.Reference
		}
		return ""
	})
}

// TRLC-LINKS: REQ-EMG-138
func collectMachineLeaves(expr SemanticExpression, pick func(SemanticExpression) string) []string {
	set := map[string]bool{}
	var walk func(SemanticExpression)
	walk = func(e SemanticExpression) {
		if v := pick(e); v != "" {
			set[v] = true
		}
		for _, operand := range e.Operands {
			walk(operand)
		}
	}
	walk(expr)
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// MachineExprString prints expr as canonical source text with the fewest
// parentheses the precedence rules allow, so that parsing the result yields
// the same tree. An "all" node prints as "a := 1; b := 2" (no trailing
// semicolon). Malformed trees print on a best-effort basis and are not
// guaranteed to re-parse.
//
// TRLC-LINKS: REQ-EMG-136
func MachineExprString(expr SemanticExpression) string {
	var sb strings.Builder
	printMachineExpr(&sb, expr, 0)
	return sb.String()
}

// TRLC-LINKS: REQ-EMG-138
func printMachineExpr(sb *strings.Builder, e SemanticExpression, minPrec int) {
	if e.Operator == "" {
		printMachineLeaf(sb, e)
		return
	}
	op, known := LookupMachineOperator(e.Operator)
	if !known {
		sb.WriteString("?" + e.Operator + "(")
		printMachineList(sb, e.Operands, ", ")
		sb.WriteString(")")
		return
	}
	wrap := op.Precedence != 0 && op.Precedence < minPrec
	if wrap {
		sb.WriteString("(")
	}
	printMachineForm(sb, op, e)
	if wrap {
		sb.WriteString(")")
	}
}

// machineOperand returns operand i of e, or a "?" placeholder when missing.
//
// TRLC-LINKS: REQ-EMG-138
func machineOperand(e SemanticExpression, i int) SemanticExpression {
	if i < len(e.Operands) {
		return e.Operands[i]
	}
	return SemanticExpression{Kind: MachineKindVariable, Value: "?"}
}

// printMachineList prints operands separated by sep.
//
// TRLC-LINKS: REQ-EMG-138
func printMachineList(sb *strings.Builder, operands []SemanticExpression, sep string) {
	for i, o := range operands {
		if i > 0 {
			sb.WriteString(sep)
		}
		printMachineExpr(sb, o, 0)
	}
}

// printMachineForm prints the known operator op applied to the operands of e
// in the source form of the operator.
//
// TRLC-LINKS: REQ-EMG-138
func printMachineForm(sb *strings.Builder, op MachineOperator, e SemanticExpression) {
	switch op.Form {
	case MachineFormInfix:
		leftMin, rightMin := machineInfixMinimums(op)
		printMachineExpr(sb, machineOperand(e, 0), leftMin)
		sb.WriteString(" " + op.Syntax + " ")
		printMachineExpr(sb, machineOperand(e, 1), rightMin)
	case MachineFormPrefix:
		sb.WriteString(op.Syntax)
		if op.Name == "not" {
			sb.WriteString(" ")
		}
		printMachineExpr(sb, machineOperand(e, 0), op.Precedence)
	case MachineFormCall:
		sb.WriteString(op.Syntax + "(")
		printMachineLeaf(sb, machineOperand(e, 0))
		sb.WriteString(")")
	case MachineFormAssign:
		printMachineLeaf(sb, machineOperand(e, 0))
		sb.WriteString(" := ")
		printMachineExpr(sb, machineOperand(e, 1), machinePrecImplies)
	case MachineFormSequence:
		printMachineList(sb, e.Operands, "; ")
	}
}

// machineInfixMinimums returns the minimum precedence of the left and right
// operand of an infix operator that needs no parentheses.
//
// TRLC-LINKS: REQ-EMG-138
func machineInfixMinimums(op MachineOperator) (left, right int) {
	switch {
	case op.RightAssociative:
		return op.Precedence + 1, op.Precedence
	case op.Precedence == machinePrecComparison:
		return op.Precedence + 1, op.Precedence + 1
	}
	return op.Precedence, op.Precedence + 1
}

// TRLC-LINKS: REQ-EMG-138
func printMachineLeaf(sb *strings.Builder, e SemanticExpression) {
	switch {
	case e.Operator != "":
		printMachineExpr(sb, e, 0)
	case e.Kind == MachineKindVariable:
		sb.WriteString(e.Value)
	case e.TypedValue == nil:
		sb.WriteString(fmt.Sprintf("?%s", e.Value))
	default:
		sb.WriteString(machineValueText(e.TypedValue))
	}
}

// machineValueText renders a typed literal; the first populated field wins.
//
// TRLC-LINKS: REQ-EMG-138
func machineValueText(v *SemanticValue) string {
	switch {
	case v.Reference != "":
		return v.Reference
	case v.Integer != nil:
		return strconv.FormatInt(*v.Integer, 10)
	case v.Boolean != nil:
		return strconv.FormatBool(*v.Boolean)
	case v.Real != nil:
		return strconv.FormatFloat(*v.Real, 'g', -1, 64)
	}
	return quoteMachineString(v.String)
}

// TRLC-LINKS: REQ-EMG-138
func quoteMachineString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
