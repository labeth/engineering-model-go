// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Extension identities of the semantic elements and relationships a machine
// projects to. Validators read them to tell machine concepts from legacy states.
const (
	MachineExtension          = "engineering.machine"
	MachineStateExtension     = "engineering.machine_state"
	MachineInvariantExtension = "engineering.machine_invariant"
	// MachineTransitionSourceType is the SourceType of transition relationships.
	MachineTransitionSourceType = "machine_transition"
	// MachineInitialSourceType is the SourceType of the machine -> initial state succession.
	MachineInitialSourceType = "machine_initial"
	// MachineFinalSourceType is the SourceType of each final state -> machine succession.
	MachineFinalSourceType = "machine_final"
	// MachineInvariantResultFeature names the boolean feature holding an invariant expression.
	MachineInvariantResultFeature = "result"
)

// machineSink is the part of ProjectSemanticModel a machine projects into.
type machineSink struct {
	// addElement reports whether the element was added (not rejected as invalid or duplicate).
	addElement func(path string, kind SemanticElementKind, extension, id, name, owner, typeRef string, source any, features ...SemanticFeature) bool
	// addSuccession adds a deterministically identified succession relationship.
	addSuccession func(path, sourceType, source, target, owner string, value any)
	// addTransition adds a transition relationship that keeps its authored ID.
	addTransition func(path string, relationship SemanticRelationship, source any)
	// setReferences records the elements an already projected element refers to.
	setReferences func(id string, references []string)
	report        func(SemanticDiagnostic)
}

// projectMachine projects one authored machine into semantic elements and
// relationships and reports structural problems (missing or unresolved
// initial/from/to, duplicate transition or variable names, bad literals,
// unparsable expressions). Rich semantic validation is layered on top.
//
// A machine becomes a state_definition (variables as attribute features);
// each state a state_usage owned by the machine (final states carry
// final=true in their engineering.machine_state metadata and a succession to
// the machine); the initial state is reached by a machine -> state succession;
// each transition is a transition relationship keeping its authored ID; each
// invariant a constraint_usage whose "result" feature holds the expression.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func projectMachine(sink machineSink, path string, m Machine) {
	invalid := func(subpath, format string, args ...any) {
		sink.report(SemanticDiagnostic{
			Code: "semantic.invalid_machine", Severity: SemanticSeverityError,
			Message: fmt.Sprintf(format, args...), Path: subpath,
		})
	}
	id := strings.TrimSpace(m.ID)
	if id == "" {
		sink.addElement(path, ElementStateDefinition, MachineExtension, "", m.Name, m.Owner, "", nil)
		return
	}
	owner := strings.TrimSpace(m.Owner)
	if owner == "" {
		invalid(path+".owner", "machine %q requires an owner (the implementing functional unit)", id)
	}

	features := make([]SemanticFeature, 0, len(m.Variables))
	variables := map[string]bool{}
	for i, v := range m.Variables {
		vpath := indexPath(path+".variables", i)
		name := strings.TrimSpace(v.Name)
		if name == "" {
			invalid(vpath+".name", "machine %q variable requires a name", id)
			continue
		}
		if variables[name] {
			sink.report(SemanticDiagnostic{
				Code: "semantic.duplicate_identity", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("duplicate variable %q in machine %q", name, id), Path: vpath + ".name",
			})
			continue
		}
		variables[name] = true
		literal, problem := machineLiteral(v.Type, v.Initial)
		if problem != "" {
			invalid(vpath, "machine %q variable %q: %s", id, name, problem)
			continue
		}
		features = append(features, SemanticFeature{
			Name: name, Kind: FeatureAttribute, Type: strings.TrimSpace(v.Type), Value: &literal,
		})
	}

	sink.addElement(path, ElementStateDefinition, MachineExtension, id, m.Name, owner, "", map[string]string{
		"id": id, "name": m.Name, "description": m.Description, "owner": owner, "initial": strings.TrimSpace(m.Initial),
	}, features...)

	states := map[string]bool{}
	for i, s := range m.States {
		spath := indexPath(path+".states", i)
		sid := strings.TrimSpace(s.ID)
		if sid != "" {
			states[sid] = true
		}
		sink.addElement(spath, ElementStateUsage, MachineStateExtension, sid, s.Name, id, "", map[string]any{
			"id": sid, "name": s.Name, "description": s.Description, "final": s.Final,
		})
		if s.Final && sid != "" {
			sink.addSuccession(spath+".final", MachineFinalSourceType, sid, id, id, map[string]string{"machine": id, "final": sid})
		}
	}

	initial := strings.TrimSpace(m.Initial)
	switch {
	case initial == "":
		invalid(path+".initial", "machine %q requires exactly one initial state", id)
	case !states[initial]:
		invalid(path+".initial", "machine %q initial state %q is not one of its states", id, initial)
	default:
		sink.addSuccession(path+".initial", MachineInitialSourceType, id, initial, id, map[string]string{"machine": id, "initial": initial})
	}

	for i, t := range m.Transitions {
		tpath := indexPath(path+".transitions", i)
		tid, from, to := strings.TrimSpace(t.ID), strings.TrimSpace(t.From), strings.TrimSpace(t.To)
		ok := true
		if tid == "" {
			sink.report(SemanticDiagnostic{
				Code: "semantic.missing_id", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("machine %q transition requires a stable ID", id), Path: tpath,
			})
			ok = false
		}
		for _, end := range []struct{ field, value string }{{"from", from}, {"to", to}} {
			switch {
			case end.value == "":
				invalid(tpath+"."+end.field, "machine %q transition %q requires %q", id, tid, end.field)
				ok = false
			case !states[end.value]:
				invalid(tpath+"."+end.field, "machine %q transition %q %s state %q is not one of its states", id, tid, end.field, end.value)
				ok = false
			}
		}
		relationship := SemanticRelationship{
			ID: tid, Kind: RelationshipTransition, Source: from, Target: to, Owner: id,
			SourceType: MachineTransitionSourceType,
		}
		if on := strings.TrimSpace(t.On); on != "" {
			relationship.Triggers = []string{on}
		}
		if t.Guard != "" {
			relationship.Guard = parseMachineExpression(sink, tpath+".guard", t.Guard, MachineExprBoolean)
			ok = ok && relationship.Guard != nil
		}
		if t.Effect != "" {
			relationship.Effect = parseMachineExpression(sink, tpath+".effect", t.Effect, MachineExprEffect)
			ok = ok && relationship.Effect != nil
		}
		if ok {
			sink.addTransition(tpath, relationship, map[string]string{
				"id": tid, "from": from, "to": to, "on": strings.TrimSpace(t.On),
				"guard": t.Guard, "effect": t.Effect, "description": t.Description,
			})
		}
	}

	for i, inv := range m.Invariants {
		ipath := indexPath(path+".invariants", i)
		iid := strings.TrimSpace(inv.ID)
		var feature []SemanticFeature
		if strings.TrimSpace(inv.Expr) == "" {
			invalid(ipath+".expr", "machine %q invariant %q requires an expression", id, iid)
		} else if expression := parseMachineExpression(sink, ipath+".expr", inv.Expr, MachineExprBoolean); expression != nil {
			feature = []SemanticFeature{{Name: MachineInvariantResultFeature, Kind: FeatureAttribute, Type: string(MachineTypeBoolean), Value: expression}}
		}
		added := sink.addElement(ipath, ElementConstraintUsage, MachineInvariantExtension, iid, iid, id, "", map[string]any{
			"id": iid, "expr": inv.Expr, "description": inv.Description, "requirements": inv.Requirements,
		}, feature...)
		if added {
			sink.setReferences(iid, inv.Requirements)
		}
	}
}

// parseMachineExpression parses one machine expression; parse diagnostics
// become projection diagnostics and yield nil.
//
// TRLC-LINKS: REQ-EMG-137
func parseMachineExpression(sink machineSink, path, source string, kind MachineExprKind) *SemanticExpression {
	expression, problems := ParseMachineExpr(source, kind)
	if len(problems) == 0 {
		return &expression
	}
	sort.SliceStable(problems, func(i, j int) bool {
		if problems[i].Line != problems[j].Line {
			return problems[i].Line < problems[j].Line
		}
		return problems[i].Column < problems[j].Column
	})
	for _, problem := range problems {
		code := problem.Code
		if code == "" {
			code = MachineExprDiagnosticCode
		}
		sink.report(SemanticDiagnostic{
			Code: code, Severity: SemanticSeverityError, Path: path,
			Message: fmt.Sprintf("invalid machine expression at line %d, column %d: %s", problem.Line, problem.Column, problem.Message),
		})
	}
	return nil
}

// machineLiteral converts an authored variable initial value into a typed
// literal expression, or explains why it does not match the declared type.
//
// TRLC-LINKS: REQ-EMG-137
func machineLiteral(typ string, value any) (SemanticExpression, string) {
	typ = strings.TrimSpace(typ)
	switch MachineType(typ) {
	case MachineTypeInteger, MachineTypeBoolean, MachineTypeString:
	default:
		return SemanticExpression{}, fmt.Sprintf("type %q must be Integer, Boolean or String", typ)
	}
	if value == nil {
		return SemanticExpression{}, "requires an initial value"
	}
	mismatch := fmt.Sprintf("initial value %v is not a %s literal", value, typ)
	switch MachineType(typ) {
	case MachineTypeInteger:
		n, ok := machineInteger(value)
		if !ok {
			return SemanticExpression{}, mismatch
		}
		return SemanticExpression{Kind: MachineKindLiteral, TypedValue: &SemanticValue{Kind: "integer", Integer: &n}}, ""
	case MachineTypeBoolean:
		b, ok := value.(bool)
		if !ok {
			return SemanticExpression{}, mismatch
		}
		return SemanticExpression{Kind: MachineKindLiteral, TypedValue: &SemanticValue{Kind: "boolean", Boolean: &b}}, ""
	default:
		s, ok := value.(string)
		if !ok {
			return SemanticExpression{}, mismatch
		}
		if s == "" {
			// Like the expression parser: SemanticValue cannot represent an empty string.
			return SemanticExpression{}, "initial String value must not be empty"
		}
		return SemanticExpression{Kind: MachineKindLiteral, TypedValue: &SemanticValue{Kind: "string", String: s}}, ""
	}
}

// TRLC-LINKS: REQ-EMG-137
func machineInteger(value any) (int64, bool) {
	switch n := value.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), uint64(n) <= math.MaxInt64
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		return int64(n), n <= math.MaxInt64
	default:
		return 0, false
	}
}
