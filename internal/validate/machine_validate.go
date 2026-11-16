// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// Machine diagnostic codes. The structural codes are always errors and never
// tier keyed; the advisory codes are warnings that the tier policy keys with
// the hygiene row (ADR-EMG-043).
const (
	codeTransitionUnknownEvent  = "behavior.transition_unknown_event"
	codeExpressionUnknownVar    = "behavior.expression_unknown_variable"
	codeExpressionType          = "behavior.expression_type"
	codeInStateForeign          = "behavior.in_state_foreign"
	codeFinalStateTransitions   = "behavior.final_state_has_transitions"
	codeMachineDeadlock         = "behavior.machine_deadlock"
	codeTransitionNondetermined = "behavior.transition_nondeterministic"
	codeTransitionNoTrigger     = "behavior.transition_no_trigger"
	codeInvariantUnlinked       = "behavior.invariant_unlinked"
	codeStateUnreachable        = "behavior.state_unreachable"
)

// machineDiagnostics validates every authored machine of a model. Projection
// (model.ProjectSemanticModel) already reports a missing or unresolved
// initial state, unresolved transition ends, duplicate variables, bad
// literals and expression syntax errors as semantic.invalid_machine and
// semantic.invalid_expression; this reports only what projection does not.
//
// Structural errors (always errors, never tier keyed):
//   - behavior.transition_unknown_event: on names no behavior event.
//   - behavior.expression_unknown_variable: a guard, effect or invariant
//     reads or assigns an undeclared variable.
//   - behavior.expression_type: an operand or result type that the shared
//     operator table (model.LookupMachineOperator) does not allow; guards
//     and invariants must be Boolean and := needs the variable's type.
//   - behavior.in_state_foreign: in_state names a state of another machine
//     or no state.
//   - behavior.final_state_has_transitions: a final state has an outgoing
//     transition.
//
// Advisory warnings (tier keyed with the hygiene row): behavior.machine_deadlock,
// behavior.transition_nondeterministic, behavior.transition_no_trigger,
// behavior.invariant_unlinked and behavior.state_unreachable.
//
// TRLC-LINKS: REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func machineDiagnostics(a model.AuthoredArchitecture, requirements map[string]bool) []Diagnostic {
	var diags []Diagnostic
	events := map[string]bool{}
	for _, e := range a.Events {
		events[strings.TrimSpace(e.ID)] = true
	}
	ctx := machineContext{events: events, owners: machineStateOwners(a.Machines)}
	for i, m := range a.Machines {
		path := fmt.Sprintf("authoredArchitecture.machines[%d]", i)
		diags = append(diags, machineStructureDiagnostics(m, path, ctx)...)
		diags = append(diags, machineAdvisoryDiagnostics(m, path, requirements)...)
	}
	return append(diags, behaviorLinkDiagnostics(a, requirements)...)
}

// machineContext is what the structure checks of every machine share: the
// behavior events and the declaring machine of each state ID.
type machineContext struct {
	events map[string]bool
	owners map[string]string
}

// machineStateOwners maps each state ID to the ID of the machine that
// declares it.
// TRLC-LINKS: REQ-EMG-138
func machineStateOwners(machines []model.Machine) map[string]string {
	owners := map[string]string{}
	for _, m := range machines {
		addMachineStateOwners(owners, m)
	}
	return owners
}

// addMachineStateOwners records the machine as the owner of each of its states that no earlier machine declares.
// TRLC-LINKS: REQ-EMG-138
func addMachineStateOwners(owners map[string]string, m model.Machine) {
	for _, s := range m.States {
		id := strings.TrimSpace(s.ID)
		if _, taken := owners[id]; id != "" && !taken {
			owners[id] = strings.TrimSpace(m.ID)
		}
	}
}

// machineScope is what an expression of one machine may refer to.
type machineScope struct {
	machine   string
	states    map[string]bool
	variables map[string]model.MachineType // "" for a variable with an invalid type
	owners    map[string]string            // state ID -> machine ID, all machines
}

// TRLC-LINKS: REQ-EMG-138
func newMachineScope(m model.Machine, owners map[string]string) machineScope {
	scope := machineScope{
		machine: strings.TrimSpace(m.ID), states: map[string]bool{},
		variables: map[string]model.MachineType{}, owners: owners,
	}
	for _, s := range m.States {
		scope.states[strings.TrimSpace(s.ID)] = true
	}
	for _, v := range m.Variables {
		name := strings.TrimSpace(v.Name)
		if _, dup := scope.variables[name]; dup || name == "" {
			continue
		}
		switch t := model.MachineType(strings.TrimSpace(v.Type)); t {
		case model.MachineTypeInteger, model.MachineTypeBoolean, model.MachineTypeString:
			scope.variables[name] = t
		default:
			scope.variables[name] = ""
		}
	}
	return scope
}

// machineStructureDiagnostics reports the structural errors of one machine.
// TRLC-LINKS: REQ-EMG-138
func machineStructureDiagnostics(m model.Machine, path string, ctx machineContext) []Diagnostic {
	var diags []Diagnostic
	scope := newMachineScope(m, ctx.owners)
	checker := transitionChecker{scope: scope, events: ctx.events, final: finalStates(m)}
	for i, t := range m.Transitions {
		diags = append(diags, checker.check(t, fmt.Sprintf("%s.transitions[%d]", path, i))...)
	}
	for i, inv := range m.Invariants {
		if strings.TrimSpace(inv.Expr) != "" {
			diags = append(diags, scope.checkSource(fmt.Sprintf("%s.invariants[%d].expr", path, i), inv.Expr, model.MachineExprBoolean)...)
		}
	}
	return diags
}

// finalStates is the set of the trimmed ids of the final states of a machine.
// TRLC-LINKS: REQ-EMG-138
func finalStates(m model.Machine) map[string]bool {
	final := map[string]bool{}
	for _, s := range m.States {
		if s.Final {
			final[strings.TrimSpace(s.ID)] = true
		}
	}
	return final
}

// transitionChecker checks the transitions of one machine.
type transitionChecker struct {
	scope  machineScope
	events map[string]bool
	final  map[string]bool
}

// check reports a transition triggered by an unknown event, one leaving a
// final state and the type errors of its guard and effect.
// TRLC-LINKS: REQ-EMG-138
func (c transitionChecker) check(t model.MachineTransition, tpath string) []Diagnostic {
	var diags []Diagnostic
	id := strings.TrimSpace(t.ID)
	if on := strings.TrimSpace(t.On); on != "" && !c.events[on] {
		diags = append(diags, Diagnostic{Code: codeTransitionUnknownEvent, Severity: SeverityError,
			Message: fmt.Sprintf("machine %q transition %q is triggered by %q, which is not a behavior event; declare it in behavior.events", c.scope.machine, id, on),
			Path:    tpath + ".on"})
	}
	if from := strings.TrimSpace(t.From); c.final[from] {
		diags = append(diags, Diagnostic{Code: codeFinalStateTransitions, Severity: SeverityError,
			Message: fmt.Sprintf("machine %q transition %q leaves final state %q; a final state ends the machine, so remove the transition or the final flag", c.scope.machine, id, from),
			Path:    tpath + ".from"})
	}
	if strings.TrimSpace(t.Guard) != "" {
		diags = append(diags, c.scope.checkSource(tpath+".guard", t.Guard, model.MachineExprBoolean)...)
	}
	if strings.TrimSpace(t.Effect) != "" {
		diags = append(diags, c.scope.checkSource(tpath+".effect", t.Effect, model.MachineExprEffect)...)
	}
	return diags
}

// checkSource parses one guard, effect or invariant and type checks it. A
// syntax error is reported by projection, so an unparsable string yields no
// diagnostic here.
// TRLC-LINKS: REQ-EMG-138
func (s machineScope) checkSource(path, source string, kind model.MachineExprKind) []Diagnostic {
	expr, problems := model.ParseMachineExpr(source, kind)
	if len(problems) > 0 {
		return nil
	}
	var diags []Diagnostic
	got := s.check(expr, path, &diags)
	switch {
	case got == "":
	case kind == model.MachineExprBoolean && got != model.MachineTypeBoolean:
		diags = append(diags, s.typeError(path, "a guard or invariant must be Boolean, got %s", got))
	case kind == model.MachineExprEffect && got != model.MachineTypeEffect:
		diags = append(diags, s.typeError(path, "an effect must be a list of assignments, got %s", got))
	}
	return diags
}

// TRLC-LINKS: REQ-EMG-138
func (s machineScope) typeError(path, format string, args ...any) Diagnostic {
	return Diagnostic{Code: codeExpressionType, Severity: SeverityError, Path: path,
		Message: fmt.Sprintf("machine %q: %s", s.machine, fmt.Sprintf(format, args...))}
}

// exprSite is where an expression sits: its path and the diagnostics sink.
type exprSite struct {
	path  string
	diags *[]Diagnostic
}

// check returns the type of an expression, or "" when it is ill typed or
// refers to something unknown (already reported, so enclosing operators stay
// quiet).
// TRLC-LINKS: REQ-EMG-138
func (s machineScope) check(e model.SemanticExpression, path string, diags *[]Diagnostic) model.MachineType {
	switch e.Kind {
	case model.MachineKindLiteral:
		return literalType(e.TypedValue)
	case model.MachineKindVariable:
		t, declared := s.variables[e.Value]
		if !declared {
			*diags = append(*diags, s.unknownVariable(path, e.Value, "reads"))
			return ""
		}
		return t
	case model.MachineKindOperator:
		return s.checkOperator(e, path, diags)
	}
	return ""
}

// literalType is the machine type of a typed literal value, "" for none or an unknown kind.
// TRLC-LINKS: REQ-EMG-138
func literalType(v *model.SemanticValue) model.MachineType {
	if v == nil {
		return ""
	}
	switch v.Kind {
	case "integer":
		return model.MachineTypeInteger
	case "boolean":
		return model.MachineTypeBoolean
	case "string":
		return model.MachineTypeString
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-138
func (s machineScope) unknownVariable(path, name, verb string) Diagnostic {
	return Diagnostic{Code: codeExpressionUnknownVar, Severity: SeverityError, Path: path,
		Message: fmt.Sprintf("machine %q %s undeclared variable %q; declare it in the machine variables", s.machine, verb, name)}
}

// TRLC-LINKS: REQ-EMG-138
func (s machineScope) checkOperator(e model.SemanticExpression, path string, diags *[]Diagnostic) model.MachineType {
	op, known := model.LookupMachineOperator(e.Operator)
	if !known {
		*diags = append(*diags, s.typeError(path, "operator %q is not in the operator table", e.Operator))
		return ""
	}
	site := exprSite{path: path, diags: diags}
	switch op.Name {
	case "in_state":
		s.checkInState(e, path, diags)
		return op.Result
	case ":=":
		return s.checkAssignment(op, e, site)
	}
	if arityMismatch(op, len(e.Operands)) {
		*diags = append(*diags, s.typeError(path, "operator %q takes %d operands, got %d", op.Name, len(op.Operands), len(e.Operands)))
		return ""
	}
	check := operandCheck{scope: s, op: op, site: site, ok: true}
	if !check.run(e.Operands) {
		return ""
	}
	return op.Result
}

// arityMismatch reports whether an operator does not take n operands: a
// variadic operator takes at least one.
// TRLC-LINKS: REQ-EMG-138
func arityMismatch(op model.MachineOperator, n int) bool {
	if op.Variadic {
		return n == 0
	}
	return n != len(op.Operands)
}

// operandCheck type checks the operands of one operator application, keeping
// the type of the first operand of the wildcard type.
type operandCheck struct {
	scope   machineScope
	op      model.MachineOperator
	site    exprSite
	ok      bool
	anyType model.MachineType
}

// run checks every operand and reports whether all were well typed.
// TRLC-LINKS: REQ-EMG-138
func (c *operandCheck) run(operands []model.SemanticExpression) bool {
	for i, operand := range operands {
		c.operand(i, operand)
	}
	return c.ok
}

// operand checks the i-th operand against the type the operator expects there.
// TRLC-LINKS: REQ-EMG-138
func (c *operandCheck) operand(i int, operand model.SemanticExpression) {
	expected := c.op.Operands[0]
	if !c.op.Variadic {
		expected = c.op.Operands[i]
	}
	got := c.scope.check(operand, c.site.path, c.site.diags)
	switch {
	case got == "":
		c.ok = false
	case expected == model.MachineTypeAny:
		c.wildcard(got)
	case got != expected:
		*c.site.diags = append(*c.site.diags, c.scope.typeError(c.site.path, "operand %d of %q must be %s, got %s", i+1, c.op.Name, expected, got))
		c.ok = false
	}
}

// wildcard checks an operand of the wildcard type: with a same-type operator it
// must match the first one.
// TRLC-LINKS: REQ-EMG-138
func (c *operandCheck) wildcard(got model.MachineType) {
	if c.op.SameType && c.anyType != "" && got != c.anyType {
		*c.site.diags = append(*c.site.diags, c.scope.typeError(c.site.path, "operands of %q must have the same type, got %s and %s", c.op.Name, c.anyType, got))
		c.ok = false
	}
	if c.anyType == "" {
		c.anyType = got
	}
}

// checkAssignment checks variable := expression: the target must be a
// declared variable and the right side must have the variable's type.
// TRLC-LINKS: REQ-EMG-138
func (s machineScope) checkAssignment(op model.MachineOperator, e model.SemanticExpression, site exprSite) model.MachineType {
	if len(e.Operands) != 2 || e.Operands[0].Kind != model.MachineKindVariable {
		*site.diags = append(*site.diags, s.typeError(site.path, "an assignment must be variable := expression"))
		return ""
	}
	name := e.Operands[0].Value
	target, declared := s.variables[name]
	if !declared {
		*site.diags = append(*site.diags, s.unknownVariable(site.path, name, "assigns"))
	}
	value := s.check(e.Operands[1], site.path, site.diags)
	if target == "" || value == "" {
		return ""
	}
	if value != target {
		*site.diags = append(*site.diags, s.typeError(site.path, "cannot assign %s to %s variable %q", value, target, name))
		return ""
	}
	return op.Result
}

// checkInState checks that in_state names a state of this machine.
// TRLC-LINKS: REQ-EMG-138
func (s machineScope) checkInState(e model.SemanticExpression, path string, diags *[]Diagnostic) {
	if len(e.Operands) != 1 || e.Operands[0].TypedValue == nil {
		*diags = append(*diags, s.typeError(path, "in_state takes one state ID"))
		return
	}
	id := e.Operands[0].TypedValue.Reference
	if s.states[id] {
		return
	}
	message := fmt.Sprintf("machine %q in_state(%s) names no state; use a state of this machine", s.machine, id)
	if owner, ok := s.owners[id]; ok {
		message = fmt.Sprintf("machine %q in_state(%s) names a state of machine %q; in_state may only name states of the same machine", s.machine, id, owner)
	}
	*diags = append(*diags, Diagnostic{Code: codeInStateForeign, Severity: SeverityError, Message: message, Path: path})
}

// machineAdvisoryDiagnostics reports the tier-keyed advisory warnings of one
// machine: deadlock, nondeterminism, missing trigger, unlinked invariant and
// unreachable state.
// TRLC-LINKS: REQ-EMG-138, REQ-EMG-140
func machineAdvisoryDiagnostics(m model.Machine, path string, requirements map[string]bool) []Diagnostic {
	diags := machineDeadlockDiagnostics(m, path)
	diags = append(diags, machineTriggerDiagnostics(m, path)...)
	diags = append(diags, machineInvariantDiagnostics(m, path, requirements)...)
	return append(diags, machineUnreachableDiagnostics(m, path)...)
}

// machineDeadlockDiagnostics reports each non-final state without an outgoing transition.
// TRLC-LINKS: REQ-EMG-138, REQ-EMG-140
func machineDeadlockDiagnostics(m model.Machine, path string) []Diagnostic {
	var diags []Diagnostic
	outgoing := map[string]int{}
	for _, t := range m.Transitions {
		outgoing[strings.TrimSpace(t.From)]++
	}
	for i, s := range m.States {
		id := strings.TrimSpace(s.ID)
		if id == "" || s.Final || outgoing[id] > 0 {
			continue
		}
		diags = append(diags, Diagnostic{Code: codeMachineDeadlock, Severity: SeverityWarning,
			Message: fmt.Sprintf("machine %q state %q is not final and has no outgoing transition; add a transition or mark it final: true", strings.TrimSpace(m.ID), id),
			Path:    fmt.Sprintf("%s.states[%d]", path, i)})
	}
	return diags
}

// machineTriggerDiagnostics reports each transition with neither event nor
// guard and each transition that overlaps an earlier one.
// TRLC-LINKS: REQ-EMG-138, REQ-EMG-140
func machineTriggerDiagnostics(m model.Machine, path string) []Diagnostic {
	var diags []Diagnostic
	machine := strings.TrimSpace(m.ID)
	for j, t := range m.Transitions {
		tpath := fmt.Sprintf("%s.transitions[%d]", path, j)
		if strings.TrimSpace(t.On) == "" && strings.TrimSpace(t.Guard) == "" {
			diags = append(diags, Diagnostic{Code: codeTransitionNoTrigger, Severity: SeverityWarning,
				Message: fmt.Sprintf("machine %q transition %q has neither on nor guard, so it can fire at any time; name the event or add a guard", machine, strings.TrimSpace(t.ID)),
				Path:    tpath})
		}
		if i := firstOverlap(m.Transitions, j); i >= 0 {
			diags = append(diags, Diagnostic{Code: codeTransitionNondetermined, Severity: SeverityWarning,
				Message: fmt.Sprintf("machine %q transitions %q and %q both leave state %q on the same trigger with no distinguishing guard; add guards that exclude each other", machine, strings.TrimSpace(m.Transitions[i].ID), strings.TrimSpace(t.ID), strings.TrimSpace(t.From)),
				Path:    tpath})
		}
	}
	return diags
}

// firstOverlap is the index of the first transition before j that overlaps
// transition j, or -1.
// TRLC-LINKS: REQ-EMG-138
func firstOverlap(transitions []model.MachineTransition, j int) int {
	for i := 0; i < j; i++ {
		if machineTransitionsOverlap(transitions[i], transitions[j]) {
			return i
		}
	}
	return -1
}

// machineInvariantDiagnostics reports each invariant linked to no existing requirement.
// TRLC-LINKS: REQ-EMG-138, REQ-EMG-140
func machineInvariantDiagnostics(m model.Machine, path string, requirements map[string]bool) []Diagnostic {
	var diags []Diagnostic
	for i, inv := range m.Invariants {
		if !anyKnown(inv.Requirements, requirements) {
			diags = append(diags, Diagnostic{Code: codeInvariantUnlinked, Severity: SeverityWarning,
				Message: fmt.Sprintf("machine %q invariant %q is linked to no existing requirement; list the REQ-* it protects in requirements", strings.TrimSpace(m.ID), strings.TrimSpace(inv.ID)),
				Path:    fmt.Sprintf("%s.invariants[%d].requirements", path, i)})
		}
	}
	return diags
}

// anyKnown reports whether any trimmed name is in the set.
// TRLC-LINKS: REQ-EMG-138
func anyKnown(names []string, known map[string]bool) bool {
	for _, n := range names {
		if known[strings.TrimSpace(n)] {
			return true
		}
	}
	return false
}

// machineTransitionsOverlap reports whether two transitions leave one state
// on the same event (or both without one) and are not told apart by their
// guards: neither has a guard, or both have the same guard text.
// TRLC-LINKS: REQ-EMG-138
func machineTransitionsOverlap(a, b model.MachineTransition) bool {
	if strings.TrimSpace(a.From) != strings.TrimSpace(b.From) || strings.TrimSpace(a.On) != strings.TrimSpace(b.On) {
		return false
	}
	return machineGuardKey(a.Guard) == machineGuardKey(b.Guard)
}

// machineGuardKey normalizes a guard to its canonical text, so formatting
// differences do not hide identical guards; an empty guard has the empty key.
// TRLC-LINKS: REQ-EMG-138
func machineGuardKey(guard string) string {
	guard = strings.TrimSpace(guard)
	if guard == "" {
		return ""
	}
	if expr, problems := model.ParseMachineExpr(guard, model.MachineExprBoolean); len(problems) == 0 {
		return model.MachineExprString(expr)
	}
	return strings.Join(strings.Fields(guard), " ")
}

// machineUnreachableDiagnostics reports each state of a machine that no
// path of transitions reaches from the machine's initial state. A machine
// whose initial state does not resolve is skipped: projection reports it.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138
func machineUnreachableDiagnostics(m model.Machine, path string) []Diagnostic {
	declared := map[string]bool{}
	for _, s := range m.States {
		declared[strings.TrimSpace(s.ID)] = true
	}
	initial := strings.TrimSpace(m.Initial)
	if !declared[initial] {
		return nil
	}
	reached := reachableStates(m, declared, initial)
	var diags []Diagnostic
	for i, s := range m.States {
		id := strings.TrimSpace(s.ID)
		if id == "" || reached[id] {
			continue
		}
		diags = append(diags, Diagnostic{Code: codeStateUnreachable, Severity: SeverityWarning,
			Message: fmt.Sprintf("machine %q state %q is unreachable: no path of transitions leads to it from the initial state %q", strings.TrimSpace(m.ID), id, initial),
			Path:    fmt.Sprintf("%s.states[%d]", path, i)})
	}
	return diags
}

// reachableStates is the set of declared states a path of transitions leads to from the initial state.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138
func reachableStates(m model.Machine, declared map[string]bool, initial string) map[string]bool {
	next := map[string][]string{}
	for _, t := range m.Transitions {
		from, to := strings.TrimSpace(t.From), strings.TrimSpace(t.To)
		if declared[from] && declared[to] {
			next[from] = append(next[from], to)
		}
	}
	reached := map[string]bool{initial: true}
	queue := []string{initial}
	for len(queue) > 0 {
		current := queue[0]
		queue = markReached(reached, next[current], queue[1:])
	}
	return reached
}

// markReached adds the targets not yet reached to the reached set and the queue.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138
func markReached(reached map[string]bool, targets, queue []string) []string {
	for _, to := range targets {
		if !reached[to] {
			reached[to] = true
			queue = append(queue, to)
		}
	}
	return queue
}
