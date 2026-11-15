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
type machineSink interface {
	// addElement reports whether the element was added (not rejected as invalid or duplicate).
	addElement(spec elementSpec) bool
	// addRelationship adds a deterministically identified relationship.
	addRelationship(spec relationshipSpec)
	// addTransition adds a transition relationship that keeps its authored ID.
	addTransition(path string, relationship SemanticRelationship, source any)
	// setReferences records the elements an already projected element refers to.
	setReferences(id string, references []string)
	report(SemanticDiagnostic)
}

// machineProjection projects the parts of one authored machine into a sink.
type machineProjection struct {
	sink machineSink
	path string
	id   string
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
	mp := machineProjection{sink: sink, path: path, id: strings.TrimSpace(m.ID)}
	if mp.id == "" {
		sink.addElement(elementSpec{path: path, kind: ElementStateDefinition, extension: MachineExtension, name: m.Name, owner: m.Owner})
		return
	}
	owner := strings.TrimSpace(m.Owner)
	if owner == "" {
		mp.invalid(path+".owner", "machine %q requires an owner (the implementing functional unit)", mp.id)
	}
	features := mp.variableFeatures(m.Variables)
	sink.addElement(elementSpec{
		path: path, kind: ElementStateDefinition, extension: MachineExtension, id: mp.id, name: m.Name, owner: owner,
		source: map[string]string{
			"id": mp.id, "name": m.Name, "description": m.Description, "owner": owner, "initial": strings.TrimSpace(m.Initial),
		},
		features: features,
	})
	states := mp.projectStates(m.States)
	mp.projectInitial(m.Initial, states)
	mp.projectTransitions(m.Transitions, states)
	mp.projectInvariants(m.Invariants)
}

// invalid reports a semantic.invalid_machine problem.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) invalid(subpath, format string, args ...any) {
	mp.sink.report(semanticError("semantic.invalid_machine", subpath, fmt.Sprintf(format, args...)))
}

// variableFeatures turns the machine variables into attribute features and
// reports the variables that cannot be projected.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) variableFeatures(variables []MachineVariable) []SemanticFeature {
	features := make([]SemanticFeature, 0, len(variables))
	seen := map[string]bool{}
	for i, v := range variables {
		if feature, ok := mp.variableFeature(indexPath(mp.path+".variables", i), v, seen); ok {
			features = append(features, feature)
		}
	}
	return features
}

// variableFeature projects one variable; seen holds the names already declared.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) variableFeature(vpath string, v MachineVariable, seen map[string]bool) (SemanticFeature, bool) {
	name := strings.TrimSpace(v.Name)
	if name == "" {
		mp.invalid(vpath+".name", "machine %q variable requires a name", mp.id)
		return SemanticFeature{}, false
	}
	if seen[name] {
		mp.sink.report(semanticError("semantic.duplicate_identity", vpath+".name",
			fmt.Sprintf("duplicate variable %q in machine %q", name, mp.id)))
		return SemanticFeature{}, false
	}
	seen[name] = true
	literal, problem := machineLiteral(v.Type, v.Initial)
	if problem != "" {
		mp.invalid(vpath, "machine %q variable %q: %s", mp.id, name, problem)
		return SemanticFeature{}, false
	}
	return SemanticFeature{Name: name, Kind: FeatureAttribute, Type: strings.TrimSpace(v.Type), Value: &literal}, true
}

// projectStates projects the machine states and returns the set of state IDs.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) projectStates(machineStates []MachineState) map[string]bool {
	states := map[string]bool{}
	for i, s := range machineStates {
		spath := indexPath(mp.path+".states", i)
		sid := strings.TrimSpace(s.ID)
		if sid != "" {
			states[sid] = true
		}
		mp.sink.addElement(elementSpec{
			path: spath, kind: ElementStateUsage, extension: MachineStateExtension, id: sid, name: s.Name, owner: mp.id,
			source: map[string]any{"id": sid, "name": s.Name, "description": s.Description, "final": s.Final},
		})
		if s.Final && sid != "" {
			mp.sink.addRelationship(relationshipSpec{
				path: spath + ".final", kind: RelationshipSuccession, sourceType: MachineFinalSourceType,
				source: sid, target: mp.id, owner: mp.id, value: map[string]string{"machine": mp.id, "final": sid},
			})
		}
	}
	return states
}

// projectInitial projects the machine -> initial state succession.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) projectInitial(declared string, states map[string]bool) {
	initial := strings.TrimSpace(declared)
	switch {
	case initial == "":
		mp.invalid(mp.path+".initial", "machine %q requires exactly one initial state", mp.id)
	case !states[initial]:
		mp.invalid(mp.path+".initial", "machine %q initial state %q is not one of its states", mp.id, initial)
	default:
		mp.sink.addRelationship(relationshipSpec{
			path: mp.path + ".initial", kind: RelationshipSuccession, sourceType: MachineInitialSourceType,
			source: mp.id, target: initial, owner: mp.id, value: map[string]string{"machine": mp.id, "initial": initial},
		})
	}
}

// projectTransitions projects the machine transitions.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) projectTransitions(transitions []MachineTransition, states map[string]bool) {
	for i, t := range transitions {
		mp.projectTransition(indexPath(mp.path+".transitions", i), t, states)
	}
}

// projectTransition projects one transition unless one of its ends, its ID
// or one of its expressions is invalid.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) projectTransition(tpath string, t MachineTransition, states map[string]bool) {
	tid, from, to := strings.TrimSpace(t.ID), strings.TrimSpace(t.From), strings.TrimSpace(t.To)
	ok := mp.checkTransitionEnds(tpath, t, states)
	if tid == "" {
		mp.sink.report(semanticError("semantic.missing_id", tpath, fmt.Sprintf("machine %q transition requires a stable ID", mp.id)))
		ok = false
	}
	relationship := SemanticRelationship{
		ID: tid, Kind: RelationshipTransition, Source: from, Target: to, Owner: mp.id,
		SourceType: MachineTransitionSourceType,
	}
	if on := strings.TrimSpace(t.On); on != "" {
		relationship.Triggers = []string{on}
	}
	ok = mp.parseTransitionExpressions(tpath, t, &relationship) && ok
	if ok {
		mp.sink.addTransition(tpath, relationship, map[string]string{
			"id": tid, "from": from, "to": to, "on": strings.TrimSpace(t.On),
			"guard": t.Guard, "effect": t.Effect, "description": t.Description,
		})
	}
}

// checkTransitionEnds reports the ends of a transition that are missing or
// not one of the machine states and says whether both are valid.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) checkTransitionEnds(tpath string, t MachineTransition, states map[string]bool) bool {
	tid := strings.TrimSpace(t.ID)
	ok := true
	for _, end := range []struct{ field, value string }{{"from", strings.TrimSpace(t.From)}, {"to", strings.TrimSpace(t.To)}} {
		switch {
		case end.value == "":
			mp.invalid(tpath+"."+end.field, "machine %q transition %q requires %q", mp.id, tid, end.field)
			ok = false
		case !states[end.value]:
			mp.invalid(tpath+"."+end.field, "machine %q transition %q %s state %q is not one of its states", mp.id, tid, end.field, end.value)
			ok = false
		}
	}
	return ok
}

// parseTransitionExpressions parses the guard and effect of a transition into
// relationship and says whether every present expression parsed.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) parseTransitionExpressions(tpath string, t MachineTransition, relationship *SemanticRelationship) bool {
	ok := true
	if t.Guard != "" {
		relationship.Guard = mp.parseExpression(tpath+".guard", t.Guard, MachineExprBoolean)
		ok = relationship.Guard != nil
	}
	if t.Effect != "" {
		relationship.Effect = mp.parseExpression(tpath+".effect", t.Effect, MachineExprEffect)
		ok = ok && relationship.Effect != nil
	}
	return ok
}

// projectInvariants projects the machine invariants as constraint usages.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) projectInvariants(invariants []MachineInvariant) {
	for i, inv := range invariants {
		ipath := indexPath(mp.path+".invariants", i)
		iid := strings.TrimSpace(inv.ID)
		added := mp.sink.addElement(elementSpec{
			path: ipath, kind: ElementConstraintUsage, extension: MachineInvariantExtension, id: iid, name: iid, owner: mp.id,
			source: map[string]any{
				"id": iid, "expr": inv.Expr, "description": inv.Description, "requirements": inv.Requirements,
			},
			features: mp.invariantFeature(ipath, iid, inv),
		})
		if added {
			mp.sink.setReferences(iid, inv.Requirements)
		}
	}
}

// invariantFeature returns the "result" feature holding the invariant expression.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) invariantFeature(ipath, iid string, inv MachineInvariant) []SemanticFeature {
	if strings.TrimSpace(inv.Expr) == "" {
		mp.invalid(ipath+".expr", "machine %q invariant %q requires an expression", mp.id, iid)
		return nil
	}
	expression := mp.parseExpression(ipath+".expr", inv.Expr, MachineExprBoolean)
	if expression == nil {
		return nil
	}
	return []SemanticFeature{{Name: MachineInvariantResultFeature, Kind: FeatureAttribute, Type: string(MachineTypeBoolean), Value: expression}}
}

// parseExpression parses one machine expression; parse diagnostics become
// projection diagnostics and yield nil.
//
// TRLC-LINKS: REQ-EMG-137
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
func (mp machineProjection) parseExpression(path, source string, kind MachineExprKind) *SemanticExpression {
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
		mp.sink.report(semanticError(code, path,
			fmt.Sprintf("invalid machine expression at line %d, column %d: %s", problem.Line, problem.Column, problem.Message)))
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
	typed, ok := machineTypedValue(MachineType(typ), value)
	if !ok {
		return SemanticExpression{}, fmt.Sprintf("initial value %v is not a %s literal", value, typ)
	}
	if typed.Kind == "string" && typed.String == "" {
		// Like the expression parser: SemanticValue cannot represent an empty string.
		return SemanticExpression{}, "initial String value must not be empty"
	}
	return SemanticExpression{Kind: MachineKindLiteral, TypedValue: &typed}, ""
}

// machineTypedValue converts value to the typed value of the declared machine
// type and says whether value has that type.
//
// TRLC-LINKS: REQ-EMG-137
func machineTypedValue(typ MachineType, value any) (SemanticValue, bool) {
	switch typ {
	case MachineTypeInteger:
		n, ok := machineInteger(value)
		return SemanticValue{Kind: "integer", Integer: &n}, ok
	case MachineTypeBoolean:
		b, ok := value.(bool)
		return SemanticValue{Kind: "boolean", Boolean: &b}, ok
	default:
		s, ok := value.(string)
		return SemanticValue{Kind: "string", String: s}, ok
	}
}

// machineInteger converts a decoded integer of any width to int64; ok is
// false for other values and for unsigned values beyond int64.
//
// TRLC-LINKS: REQ-EMG-137
func machineInteger(value any) (int64, bool) {
	if n, ok := machineSignedInteger(value); ok {
		return n, true
	}
	return machineUnsignedInteger(value)
}

// TRLC-LINKS: REQ-EMG-137
func machineSignedInteger(value any) (int64, bool) {
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
	default:
		return 0, false
	}
}

// TRLC-LINKS: REQ-EMG-137
func machineUnsignedInteger(value any) (int64, bool) {
	switch n := value.(type) {
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
