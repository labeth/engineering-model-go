// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package quint

import (
	"fmt"
	"strconv"

	"github.com/labeth/engineering-model-go/internal/model"
)

// problem is one failure while rendering an expression; code is the
// diagnostic code the caller reports it under.
type problem struct {
	code string
	msg  string
}

// Error is part of the Quint module generation.
//
// TRLC-LINKS: REQ-EMG-141
func (p *problem) Error() string { return p.msg }

const (
	codeUnresolved = "quint.unresolved_reference"
	codeType       = "quint.type_mismatch"
	codeExpression = "quint.invalid_expression"
	codeString     = "quint.unsupported_string"
)

// exprRenderer renders machine expression trees as Quint source. variables
// maps a model variable name to its type; stateCtor maps a state ID to its
// Quint constructor. Both are read-only.
type exprRenderer struct {
	variables map[string]model.MachineType
	stateCtor map[string]string
}

// varIdent is the Quint name of a model variable.
//
// TRLC-LINKS: REQ-EMG-141
func varIdent(name string) string { return "v_" + name }

// boolean renders a guard or invariant and checks that it is Boolean.
//
// TRLC-LINKS: REQ-EMG-141
func (r exprRenderer) boolean(e model.SemanticExpression) (string, *problem) {
	text, typ, err := r.expr(e)
	if err != nil {
		return "", err
	}
	if typ != model.MachineTypeBoolean {
		return "", &problem{codeType, fmt.Sprintf("expression %q is %s, expected Boolean", model.MachineExprString(e), typ)}
	}
	return text, nil
}

// assignment is one rendered "variable := value" of an effect.
type assignment struct {
	variable string
	value    string
}

// effect renders the assignments of an "all" node in authored order and checks
// each against the declared variable type.
//
// TRLC-LINKS: REQ-EMG-141
func (r exprRenderer) effect(e model.SemanticExpression) ([]assignment, *problem) {
	if e.Operator != "all" {
		return nil, &problem{codeExpression, fmt.Sprintf("effect %q is not an assignment list", model.MachineExprString(e))}
	}
	out := make([]assignment, 0, len(e.Operands))
	for _, assign := range e.Operands {
		if assign.Operator != ":=" || len(assign.Operands) != 2 || assign.Operands[0].Kind != model.MachineKindVariable {
			return nil, &problem{codeExpression, fmt.Sprintf("effect part %q is not 'variable := expression'", model.MachineExprString(assign))}
		}
		name := assign.Operands[0].Value
		want, ok := r.variables[name]
		if !ok {
			return nil, &problem{codeUnresolved, fmt.Sprintf("effect assigns undeclared variable %q", name)}
		}
		text, got, err := r.expr(assign.Operands[1])
		if err != nil {
			return nil, err
		}
		if got != want {
			return nil, &problem{codeType, fmt.Sprintf("effect assigns a %s value to %s variable %q", got, want, name)}
		}
		out = append(out, assignment{variable: name, value: text})
	}
	return out, nil
}

// expr renders e and returns its type, applying the typing rules of the shared
// operator table (model.MachineOperators).
//
// TRLC-LINKS: REQ-EMG-141
func (r exprRenderer) expr(e model.SemanticExpression) (string, model.MachineType, *problem) {
	if e.Operator == "" {
		return r.leaf(e)
	}
	op, known := model.LookupMachineOperator(e.Operator)
	if !known {
		return "", "", &problem{codeExpression, fmt.Sprintf("unknown operator %q", e.Operator)}
	}
	if op.Name == "in_state" {
		return r.inState(e)
	}
	if op.Form == model.MachineFormAssign || op.Form == model.MachineFormSequence {
		return "", "", &problem{codeExpression, fmt.Sprintf("operator %q is only allowed in effects", op.Name)}
	}
	if len(e.Operands) != len(op.Operands) {
		return "", "", &problem{codeExpression, fmt.Sprintf("operator %q takes %d operand(s), got %d", op.Name, len(op.Operands), len(e.Operands))}
	}
	texts := make([]string, len(e.Operands))
	var same model.MachineType
	for i, operand := range e.Operands {
		text, typ, err := r.expr(operand)
		if err != nil {
			return "", "", err
		}
		want := op.Operands[i]
		if want == model.MachineTypeAny {
			if same == "" {
				same = typ
			}
			want = same
		}
		if typ != want {
			return "", "", &problem{codeType, fmt.Sprintf("operator %q needs %s for operand %d, got %s in %q", op.Name, want, i+1, typ, model.MachineExprString(operand))}
		}
		texts[i] = text
		if op.Form == model.MachineFormInfix && isCompound(operand) {
			texts[i] = "(" + text + ")"
		}
	}
	switch op.Form {
	case model.MachineFormInfix:
		return texts[0] + " " + op.Syntax + " " + texts[1], op.Result, nil
	case model.MachineFormPrefix:
		if op.Name == "not" {
			return "not(" + texts[0] + ")", op.Result, nil
		}
		return "-(" + texts[0] + ")", op.Result, nil
	}
	return "", "", &problem{codeExpression, fmt.Sprintf("operator %q has no Quint rendering", op.Name)}
}

// isCompound reports whether an operand needs parentheses inside an infix
// operator: every operator application except not(...) and -(...), which
// delimit themselves, and every negative integer literal.
//
// TRLC-LINKS: REQ-EMG-141
func isCompound(e model.SemanticExpression) bool {
	if e.Operator == "" {
		return e.TypedValue != nil && e.TypedValue.Integer != nil && *e.TypedValue.Integer < 0
	}
	return e.Operator != "not" && e.Operator != "neg"
}

// inState is part of the Quint module generation.
//
// TRLC-LINKS: REQ-EMG-141
func (r exprRenderer) inState(e model.SemanticExpression) (string, model.MachineType, *problem) {
	if len(e.Operands) != 1 || e.Operands[0].Kind != model.MachineKindReference || e.Operands[0].TypedValue == nil {
		return "", "", &problem{codeExpression, "in_state needs one state ID"}
	}
	id := e.Operands[0].TypedValue.Reference
	ctor, ok := r.stateCtor[id]
	if !ok {
		return "", "", &problem{codeUnresolved, fmt.Sprintf("in_state(%s) names a state that is not in this machine", id)}
	}
	return "state == " + ctor, model.MachineTypeBoolean, nil
}

// leaf is part of the Quint module generation.
//
// TRLC-LINKS: REQ-EMG-141
func (r exprRenderer) leaf(e model.SemanticExpression) (string, model.MachineType, *problem) {
	if e.Kind == model.MachineKindVariable {
		typ, ok := r.variables[e.Value]
		if !ok {
			return "", "", &problem{codeUnresolved, fmt.Sprintf("undeclared variable %q", e.Value)}
		}
		return varIdent(e.Value), typ, nil
	}
	v := e.TypedValue
	switch {
	case v == nil:
		return "", "", &problem{codeExpression, "expression leaf has neither a variable nor a literal value"}
	case v.Integer != nil:
		return strconv.FormatInt(*v.Integer, 10), model.MachineTypeInteger, nil
	case v.Boolean != nil:
		return strconv.FormatBool(*v.Boolean), model.MachineTypeBoolean, nil
	case v.Kind == "string":
		text, ok := quintString(v.String)
		if !ok {
			return "", "", &problem{codeString, fmt.Sprintf("string literal %q cannot be written in Quint (no double quote or line break)", v.String)}
		}
		return text, model.MachineTypeString, nil
	}
	return "", "", &problem{codeExpression, fmt.Sprintf("unsupported literal of kind %q", v.Kind)}
}
