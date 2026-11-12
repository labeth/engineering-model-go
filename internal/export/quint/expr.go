// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package quint

import (
	"fmt"
	"strconv"
	"strings"

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

// errSilent is the problem of an expression that reads a variable whose own
// declaration was already reported; it is not reported a second time.
var errSilent = &problem{code: "quint.suppressed", msg: "depends on an invalid variable declaration"}

// invalidVariableType marks a variable whose declaration failed and whose type
// is therefore unknown.
const invalidVariableType model.MachineType = "?"

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
		if want == invalidVariableType {
			return nil, errSilent
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
// operator table (model.MachineOperators). The text is written once into a
// shared builder, so rendering is linear in the size of the tree and output.
//
// TRLC-LINKS: REQ-EMG-141
func (r exprRenderer) expr(e model.SemanticExpression) (string, model.MachineType, *problem) {
	var sb strings.Builder
	typ, err := r.write(&sb, e)
	if err != nil {
		return "", "", err
	}
	return sb.String(), typ, nil
}

// write appends the Quint text of e to sb and returns its type. On a problem
// the content already written to sb is meaningless and must be discarded.
// Operands of infix operators that need it are parenthesised explicitly.
//
// TRLC-LINKS: REQ-EMG-141
func (r exprRenderer) write(sb *strings.Builder, e model.SemanticExpression) (model.MachineType, *problem) {
	if e.Operator == "" {
		return r.writeLeaf(sb, e)
	}
	op, known := model.LookupMachineOperator(e.Operator)
	if !known {
		return "", &problem{codeExpression, fmt.Sprintf("unknown operator %q", e.Operator)}
	}
	if op.Name == "in_state" {
		return r.writeInState(sb, e)
	}
	if op.Form == model.MachineFormAssign || op.Form == model.MachineFormSequence {
		return "", &problem{codeExpression, fmt.Sprintf("operator %q is only allowed in effects", op.Name)}
	}
	if len(e.Operands) != len(op.Operands) {
		return "", &problem{codeExpression, fmt.Sprintf("operator %q takes %d operand(s), got %d", op.Name, len(op.Operands), len(e.Operands))}
	}
	switch op.Form {
	case model.MachineFormInfix:
	case model.MachineFormPrefix:
		if op.Name == "not" {
			sb.WriteString("not(")
		} else {
			sb.WriteString("-(")
		}
	default:
		return "", &problem{codeExpression, fmt.Sprintf("operator %q has no Quint rendering", op.Name)}
	}
	var same model.MachineType
	for i, operand := range e.Operands {
		if i > 0 {
			sb.WriteString(" " + op.Syntax + " ")
		}
		wrap := op.Form == model.MachineFormInfix && isCompound(operand)
		if wrap {
			sb.WriteByte('(')
		}
		typ, err := r.write(sb, operand)
		if err != nil {
			return "", err
		}
		if wrap {
			sb.WriteByte(')')
		}
		want := op.Operands[i]
		if want == model.MachineTypeAny {
			if same == "" {
				same = typ
			}
			want = same
		}
		if typ != want {
			return "", &problem{codeType, fmt.Sprintf("operator %q needs %s for operand %d, got %s in %q", op.Name, want, i+1, typ, model.MachineExprString(operand))}
		}
	}
	if op.Form == model.MachineFormPrefix {
		sb.WriteByte(')')
	}
	return op.Result, nil
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

// writeInState appends "state == Ctor" for an in_state node.
//
// TRLC-LINKS: REQ-EMG-141
func (r exprRenderer) writeInState(sb *strings.Builder, e model.SemanticExpression) (model.MachineType, *problem) {
	if len(e.Operands) != 1 || e.Operands[0].Kind != model.MachineKindReference || e.Operands[0].TypedValue == nil {
		return "", &problem{codeExpression, "in_state needs one state ID"}
	}
	id := e.Operands[0].TypedValue.Reference
	ctor, ok := r.stateCtor[id]
	if !ok {
		return "", &problem{codeUnresolved, fmt.Sprintf("in_state(%s) names a state that is not in this machine", id)}
	}
	sb.WriteString("state == " + ctor)
	return model.MachineTypeBoolean, nil
}

// writeLeaf appends a variable or literal to sb.
//
// TRLC-LINKS: REQ-EMG-141
func (r exprRenderer) writeLeaf(sb *strings.Builder, e model.SemanticExpression) (model.MachineType, *problem) {
	if e.Kind == model.MachineKindVariable {
		typ, ok := r.variables[e.Value]
		if !ok {
			return "", &problem{codeUnresolved, fmt.Sprintf("undeclared variable %q", e.Value)}
		}
		if typ == invalidVariableType {
			return "", errSilent
		}
		sb.WriteString(varIdent(e.Value))
		return typ, nil
	}
	v := e.TypedValue
	switch {
	case v == nil:
		return "", &problem{codeExpression, "expression leaf has neither a variable nor a literal value"}
	case v.Integer != nil:
		sb.WriteString(strconv.FormatInt(*v.Integer, 10))
		return model.MachineTypeInteger, nil
	case v.Boolean != nil:
		sb.WriteString(strconv.FormatBool(*v.Boolean))
		return model.MachineTypeBoolean, nil
	case v.Kind == "string":
		text, ok := quintString(v.String)
		if !ok {
			return "", &problem{codeString, fmt.Sprintf("string literal %q cannot be written in Quint (no double quote or line break)", v.String)}
		}
		sb.WriteString(text)
		return model.MachineTypeString, nil
	}
	return "", &problem{codeExpression, fmt.Sprintf("unsupported literal of kind %q", v.Kind)}
}
