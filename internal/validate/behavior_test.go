// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// baseTestMachine is a well-formed machine: Idle -> Running on EVT-START and
// Running -> Stopped (final) on EVT-TERM, with an Integer, a Boolean and a
// String variable and no invariant.
func baseTestMachine() model.Machine {
	return model.Machine{
		ID: "SM-RUN", Name: "Run", Owner: "FU-A", Initial: "STATE-IDLE",
		Variables: []model.MachineVariable{
			{Name: "count", Type: "Integer", Initial: 0},
			{Name: "flag", Type: "Boolean", Initial: false},
			{Name: "label", Type: "String", Initial: "x"},
		},
		States: []model.MachineState{
			{ID: "STATE-IDLE", Name: "Idle"}, {ID: "STATE-RUN", Name: "Running"}, {ID: "STATE-STOP", Name: "Stopped", Final: true},
		},
		Transitions: []model.MachineTransition{
			{ID: "TR-START", From: "STATE-IDLE", To: "STATE-RUN", On: "EVT-START"},
			{ID: "TR-TERM", From: "STATE-RUN", To: "STATE-STOP", On: "EVT-TERM"},
		},
	}
}

// behaviorTestBundle is a valid bundle with the machine SM-RUN, two events,
// requirement REQ-A and the given relationships and flows.
func behaviorTestBundle(mappings []model.Mapping, flows []model.Flow) model.Bundle {
	b := schemaV2TestBundle(model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit"}},
		TrustBoundaries:  []model.TrustBoundary{{ID: "TB-A", Name: "Boundary"}},
		Machines:         []model.Machine{baseTestMachine()},
		Events:           []model.Event{{ID: "EVT-START", Name: "Start"}, {ID: "EVT-TERM", Name: "Termination signal"}},
		Flows:            flows,
		Mappings:         mappings,
	}}})
	b.Requirements.Requirements = []model.Requirement{{ID: "REQ-A", Text: "x"}}
	return b
}

// machineBundle is behaviorTestBundle with SM-RUN edited by edit.
func machineBundle(edit func(*model.Machine)) model.Bundle {
	b := behaviorTestBundle(nil, nil)
	m := baseTestMachine()
	if edit != nil {
		edit(&m)
	}
	b.Architecture.AuthoredArchitecture.Machines = []model.Machine{m}
	return b
}

func behaviorDiags(b model.Bundle, code string) []Diagnostic {
	var out []Diagnostic
	for _, d := range Bundle(b) {
		if d.Code == code {
			out = append(out, d)
		}
	}
	return out
}

// paths returns the paths of diagnostics, in order.
func diagPaths(diags []Diagnostic) []string {
	out := make([]string, len(diags))
	for i, d := range diags {
		out[i] = d.Path
	}
	return out
}

const machinePath = "authoredArchitecture.machines[0]"

// wantOne fails unless diags holds exactly one diagnostic of the severity at
// path whose message contains every fragment.
func wantOne(t *testing.T, diags []Diagnostic, severity Severity, path string, fragments ...string) {
	t.Helper()
	if len(diags) != 1 || diags[0].Severity != severity || diags[0].Path != path {
		t.Fatalf("want one %s at %s, got %+v", severity, path, diags)
	}
	for _, fragment := range fragments {
		if !strings.Contains(diags[0].Message, fragment) {
			t.Fatalf("message %q lacks %q", diags[0].Message, fragment)
		}
	}
}

// A well-formed machine reports no behavior diagnostic and no error.
// Mutation: any machine rule that fires on the baseline fails this test.
// TRLC-LINKS: REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineBaselineReportsNothing(t *testing.T) {
	diags := Bundle(machineBundle(nil))
	for _, d := range diags {
		if strings.HasPrefix(d.Code, "behavior.") || (d.Severity == SeverityError && d.Code != "lifecycle.missing") {
			t.Fatalf("well-formed machine must be clean, got %+v", d)
		}
	}
}

// A transition whose on is not a behavior event is an error at the on field.
// Mutation: skipping the event lookup accepts EVT-NOPE.
// TRLC-LINKS: REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineTransitionOnMustBeABehaviorEvent(t *testing.T) {
	b := machineBundle(func(m *model.Machine) { m.Transitions[0].On = "EVT-NOPE" })
	wantOne(t, behaviorDiags(b, "behavior.transition_unknown_event"), SeverityError, machinePath+".transitions[0].on", "EVT-NOPE", "TR-START")
	if got := behaviorDiags(machineBundle(nil), "behavior.transition_unknown_event"); len(got) != 0 {
		t.Fatalf("declared events must be accepted, got %+v", got)
	}
}

// A guard, effect or invariant that reads or assigns an undeclared variable
// is an error at that field; declared variables are accepted.
// Mutation: not checking the assignment target accepts ghost := 1.
// TRLC-LINKS: REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineExpressionUnknownVariable(t *testing.T) {
	cases := map[string]struct {
		edit func(*model.Machine)
		path string
	}{
		"guard":      {func(m *model.Machine) { m.Transitions[0].Guard = "ghost < 3" }, machinePath + ".transitions[0].guard"},
		"effect":     {func(m *model.Machine) { m.Transitions[0].Effect = "ghost := 1" }, machinePath + ".transitions[0].effect"},
		"effect rhs": {func(m *model.Machine) { m.Transitions[0].Effect = "count := ghost + 1" }, machinePath + ".transitions[0].effect"},
		"invariant": {func(m *model.Machine) {
			m.Invariants = []model.MachineInvariant{{ID: "INV-A", Requirements: []string{"REQ-A"}, Expr: "ghost == 1"}}
		}, machinePath + ".invariants[0].expr"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			wantOne(t, behaviorDiags(machineBundle(c.edit), "behavior.expression_unknown_variable"), SeverityError, c.path, "ghost")
		})
	}
	clean := machineBundle(func(m *model.Machine) {
		m.Transitions[0].Guard = "count < 3"
		m.Transitions[0].Effect = "count := count + 1"
		m.Invariants = []model.MachineInvariant{{ID: "INV-A", Requirements: []string{"REQ-A"}, Expr: "count >= 0"}}
	})
	if got := behaviorDiags(clean, "behavior.expression_unknown_variable"); len(got) != 0 {
		t.Fatalf("declared variables must be accepted, got %+v", got)
	}
}

// Operand and result types follow the shared operator table: a guard and an
// invariant are Boolean, ordering needs Integer operands, == needs equal
// types, := needs the variable's type, and, or and not need Boolean. Each
// violation is behavior.expression_type at the field; well-typed use of every
// operator is accepted.
// Mutation: treating String as ordered or skipping the := check fails a case.
// TRLC-LINKS: REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineExpressionTypeMismatch(t *testing.T) {
	guard := func(source string) func(*model.Machine) {
		return func(m *model.Machine) { m.Transitions[0].Guard = source }
	}
	effect := func(source string) func(*model.Machine) {
		return func(m *model.Machine) { m.Transitions[0].Effect = source }
	}
	invariant := func(source string) func(*model.Machine) {
		return func(m *model.Machine) {
			m.Invariants = []model.MachineInvariant{{ID: "INV-A", Requirements: []string{"REQ-A"}, Expr: source}}
		}
	}
	guardPath, effectPath, invariantPath := machinePath+".transitions[0].guard", machinePath+".transitions[0].effect", machinePath+".invariants[0].expr"
	bad := []struct {
		name, path, fragment string
		edit                 func(*model.Machine)
	}{
		{"guard is Integer", guardPath, "must be Boolean, got Integer", guard("count + 1")},
		{"guard is String", guardPath, "must be Boolean, got String", guard("label")},
		{"invariant is Integer", invariantPath, "must be Boolean, got Integer", invariant("count")},
		{"string compared with <", guardPath, `operand 1 of "<" must be Integer, got String`, guard("label < 3")},
		{"boolean compared with >=", guardPath, `operand 1 of ">=" must be Integer, got Boolean`, guard("flag >= 1")},
		{"== of different types", guardPath, `operands of "==" must have the same type, got Integer and Boolean`, guard("count == flag")},
		{"and of Integers", guardPath, `operand 1 of "and" must be Boolean, got Integer`, guard("count and flag")},
		{"not of Integer", guardPath, `operand 1 of "not" must be Boolean, got Integer`, guard("(not count)")},
		{"arithmetic on Boolean", guardPath, `operand 1 of "+" must be Integer, got Boolean`, guard("flag + 1 > 0")},
		{"assign Boolean to Integer", effectPath, `cannot assign Boolean to Integer variable "count"`, effect("count := flag")},
		{"assign Integer to String", effectPath, `cannot assign Integer to String variable "label"`, effect("label := 1")},
		{"assign Integer to Boolean", effectPath, `cannot assign Integer to Boolean variable "flag"`, effect("flag := count + 1")},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			wantOne(t, behaviorDiags(machineBundle(c.edit), "behavior.expression_type"), SeverityError, c.path, c.fragment)
		})
	}
	good := machineBundle(func(m *model.Machine) {
		m.Transitions[0].Guard = `count < 3 and not flag or label == "x" and count != 2 implies count * 2 - 1 <= 9`
		m.Transitions[0].Effect = `count := count * 2 - 1; flag := count > 0; label := "y"`
		m.Transitions[1].Guard = "in_state(STATE-RUN) and flag == true"
		m.Invariants = []model.MachineInvariant{{ID: "INV-A", Requirements: []string{"REQ-A"}, Expr: "flag implies count >= 0"}}
	})
	if got := behaviorDiags(good, "behavior.expression_type"); len(got) != 0 {
		t.Fatalf("well-typed expressions must be accepted, got %+v", got)
	}
	// An unparsable expression is a projection error, not a type error.
	syntax := machineBundle(guard("count <"))
	if got := behaviorDiags(syntax, "behavior.expression_type"); len(got) != 0 {
		t.Fatalf("syntax errors belong to projection, got %+v", got)
	}
}

// in_state must name a state of the same machine: a state of another machine
// and an unknown state are errors at the field, naming which.
// Mutation: checking only that the state exists somewhere accepts STATE-OTHER.
// TRLC-LINKS: REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineInStateMustNameAStateOfTheSameMachine(t *testing.T) {
	with := func(guard string) model.Bundle {
		b := machineBundle(func(m *model.Machine) { m.Transitions[0].Guard = guard })
		b.Architecture.AuthoredArchitecture.Machines = append(b.Architecture.AuthoredArchitecture.Machines, model.Machine{
			ID: "SM-OTHER", Owner: "FU-A", Initial: "STATE-OTHER",
			States: []model.MachineState{{ID: "STATE-OTHER", Name: "Other", Final: true}},
		})
		return b
	}
	path := machinePath + ".transitions[0].guard"
	wantOne(t, behaviorDiags(with("in_state(STATE-OTHER)"), "behavior.in_state_foreign"), SeverityError, path, "STATE-OTHER", `machine "SM-OTHER"`)
	wantOne(t, behaviorDiags(with("in_state(STATE-NOPE)"), "behavior.in_state_foreign"), SeverityError, path, "STATE-NOPE", "names no state")
	if got := behaviorDiags(with("in_state(STATE-RUN)"), "behavior.in_state_foreign"); len(got) != 0 {
		t.Fatalf("a state of the same machine must be accepted, got %+v", got)
	}
}

// A final state with an outgoing transition is an error at the transition's
// from field; a final state without one is fine.
// Mutation: dropping the final check accepts STATE-RUN final with TR-TERM.
// TRLC-LINKS: REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineFinalStateMustNotHaveTransitions(t *testing.T) {
	b := machineBundle(func(m *model.Machine) { m.States[1].Final = true })
	wantOne(t, behaviorDiags(b, "behavior.final_state_has_transitions"), SeverityError, machinePath+".transitions[1].from", "STATE-RUN", "TR-TERM")
	if got := behaviorDiags(machineBundle(nil), "behavior.final_state_has_transitions"); len(got) != 0 {
		t.Fatalf("the final state without transitions must be accepted, got %+v", got)
	}
}

// A non-final state without an outgoing transition is a warning at the
// state; a final state without one is not reported.
// Mutation: ignoring the final flag reports STATE-STOP in the baseline.
// TRLC-LINKS: REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineDeadlockWarnsAtTheState(t *testing.T) {
	b := machineBundle(func(m *model.Machine) { m.States[2].Final = false })
	wantOne(t, behaviorDiags(b, "behavior.machine_deadlock"), SeverityWarning, machinePath+".states[2]", "STATE-STOP", "final: true")
	if got := behaviorDiags(machineBundle(nil), "behavior.machine_deadlock"); len(got) != 0 {
		t.Fatalf("a final state and states with transitions are not deadlocks, got %+v", got)
	}
}

// Two transitions from one state on one event are nondeterministic when
// neither has a guard or both have the same guard text (in any formatting);
// two without an event count as the same trigger; different events or
// different guards are fine. The finding is at the later transition.
// Mutation: comparing raw guard text reports the same-guard case only when
// the spelling matches; comparing only on reports guarded pairs.
// TRLC-LINKS: REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineTransitionNondeterministicWarns(t *testing.T) {
	add := func(t machineTransitionEdit) func(*model.Machine) {
		return func(m *model.Machine) {
			m.Transitions = append(m.Transitions, model.MachineTransition{ID: "TR-DUP", From: "STATE-IDLE", To: "STATE-STOP", On: t.on, Guard: t.guard})
			m.Transitions[0].Guard = t.first
			m.Transitions[0].On = t.firstOn
		}
	}
	path := machinePath + ".transitions[2]"
	positive := map[string]machineTransitionEdit{
		"both unguarded, same event":  {on: "EVT-START", firstOn: "EVT-START"},
		"identical guard text":        {on: "EVT-START", firstOn: "EVT-START", guard: "count < 3", first: "count < 3"},
		"same guard other formatting": {on: "EVT-START", firstOn: "EVT-START", guard: "count<3", first: "count  <  3"},
		"both without event, guards":  {first: "count < 3", guard: "count < 3"},
	}
	for name, c := range positive {
		t.Run(name, func(t *testing.T) {
			wantOne(t, behaviorDiags(machineBundle(add(c)), "behavior.transition_nondeterministic"), SeverityWarning, path, "TR-START", "TR-DUP", "STATE-IDLE")
		})
	}
	negative := map[string]machineTransitionEdit{
		"different events":    {on: "EVT-TERM", firstOn: "EVT-START"},
		"different guards":    {on: "EVT-START", firstOn: "EVT-START", guard: "count >= 3", first: "count < 3"},
		"guarded and not":     {on: "EVT-START", firstOn: "EVT-START", guard: "count >= 3"},
		"event and no event":  {firstOn: "EVT-START"},
		"other source states": {on: "EVT-START", firstOn: "EVT-START", guard: ""},
	}
	for name, c := range negative {
		t.Run("not "+name, func(t *testing.T) {
			b := machineBundle(add(c))
			if name == "other source states" {
				b.Architecture.AuthoredArchitecture.Machines[0].Transitions[2].From = "STATE-RUN"
			}
			if got := behaviorDiags(b, "behavior.transition_nondeterministic"); len(got) != 0 {
				t.Fatalf("distinguishable transitions must not be reported, got %+v", got)
			}
		})
	}
}

type machineTransitionEdit struct{ on, firstOn, guard, first string }

// A transition without on and without guard is a warning at the transition;
// a transition with only an event or only a guard is not.
// Mutation: requiring both on and guard reports every transition.
// TRLC-LINKS: REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineTransitionWithoutTriggerWarns(t *testing.T) {
	b := machineBundle(func(m *model.Machine) { m.Transitions[1].On = "" })
	wantOne(t, behaviorDiags(b, "behavior.transition_no_trigger"), SeverityWarning, machinePath+".transitions[1]", "TR-TERM")
	guarded := machineBundle(func(m *model.Machine) { m.Transitions[1].On, m.Transitions[1].Guard = "", "flag" })
	if got := behaviorDiags(guarded, "behavior.transition_no_trigger"); len(got) != 0 {
		t.Fatalf("a guard alone is a trigger, got %+v", got)
	}
	if got := behaviorDiags(machineBundle(nil), "behavior.transition_no_trigger"); len(got) != 0 {
		t.Fatalf("an event alone is a trigger, got %+v", got)
	}
}

// An invariant with no requirement, or only requirements that do not exist,
// is a warning at its requirements field; one existing requirement links it.
// Mutation: counting unknown IDs as links accepts REQ-NOPE.
// TRLC-LINKS: REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineInvariantUnlinkedWarns(t *testing.T) {
	with := func(requirements ...string) model.Bundle {
		return machineBundle(func(m *model.Machine) {
			m.Invariants = []model.MachineInvariant{{ID: "INV-A", Requirements: requirements, Expr: "count >= 0"}}
		})
	}
	path := machinePath + ".invariants[0].requirements"
	wantOne(t, behaviorDiags(with(), "behavior.invariant_unlinked"), SeverityWarning, path, "INV-A")
	wantOne(t, behaviorDiags(with("REQ-NOPE"), "behavior.invariant_unlinked"), SeverityWarning, path, "INV-A")
	for _, linked := range [][]string{{"REQ-A"}, {"REQ-NOPE", "REQ-A"}} {
		if got := behaviorDiags(with(linked...), "behavior.invariant_unlinked"); len(got) != 0 {
			t.Fatalf("an existing requirement links the invariant, got %+v for %v", got, linked)
		}
	}
}

// A state is unreachable when no path of transitions leads to it from the
// machine's initial state: the first declared state gets no exemption, a
// cycle that nothing enters is unreachable, and a state reached through
// several transitions is reachable. A machine without a resolvable initial
// state reports nothing here (projection reports it).
// Mutation: restoring the first-declared-state exemption drops STATE-IDLE
// from the second case; counting any incoming transition as reaching drops
// the cycle states.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestBehaviorStateUnreachableWorksOnMachines(t *testing.T) {
	if got := behaviorDiags(machineBundle(nil), "behavior.state_unreachable"); len(got) != 0 {
		t.Fatalf("every state is reached from the initial state, got %+v", got)
	}

	initialIsNotFirst := machineBundle(func(m *model.Machine) {
		m.Initial = "STATE-RUN"
		m.Transitions = []model.MachineTransition{{ID: "TR-TERM", From: "STATE-RUN", To: "STATE-STOP", On: "EVT-TERM"}}
	})
	got := behaviorDiags(initialIsNotFirst, "behavior.state_unreachable")
	wantOne(t, got, SeverityWarning, machinePath+".states[0]", "STATE-IDLE", "STATE-RUN")

	cycle := machineBundle(func(m *model.Machine) {
		m.States = append(m.States, model.MachineState{ID: "STATE-X"}, model.MachineState{ID: "STATE-Y"})
		m.Transitions = append(m.Transitions,
			model.MachineTransition{ID: "TR-XY", From: "STATE-X", To: "STATE-Y", On: "EVT-START"},
			model.MachineTransition{ID: "TR-YX", From: "STATE-Y", To: "STATE-X", On: "EVT-TERM"})
	})
	got = behaviorDiags(cycle, "behavior.state_unreachable")
	if len(got) != 2 || strings.Join(diagPaths(got), " ") != machinePath+".states[3] "+machinePath+".states[4]" {
		t.Fatalf("want the entered-only-by-each-other states X and Y unreachable, got %+v", got)
	}

	noInitial := machineBundle(func(m *model.Machine) { m.Initial = "STATE-NOPE" })
	if got := behaviorDiags(noInitial, "behavior.state_unreachable"); len(got) != 0 {
		t.Fatalf("an unresolved initial is reported by projection, got %+v", got)
	}
}

// Machine, state, transition and invariant IDs join the model-wide
// duplicate ID check, at the path of the second declaration.
// Mutation: not registering transitions accepts a repeated TR-START.
// TRLC-LINKS: REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineIDsAreRegisteredForDuplicateDetection(t *testing.T) {
	cases := map[string]struct {
		edit func(*model.Machine)
		path string
	}{
		"state repeats a state":      {func(m *model.Machine) { m.States[2].ID = "STATE-IDLE" }, machinePath + ".states[2]"},
		"state repeats a unit":       {func(m *model.Machine) { m.States[2].ID = "FU-A" }, machinePath + ".states[2]"},
		"transition repeats":         {func(m *model.Machine) { m.Transitions[1].ID = "TR-START" }, machinePath + ".transitions[1]"},
		"transition repeats a state": {func(m *model.Machine) { m.Transitions[1].ID = "STATE-RUN" }, machinePath + ".transitions[1]"},
		"invariant repeats": {func(m *model.Machine) {
			m.Invariants = []model.MachineInvariant{{ID: "INV-A", Requirements: []string{"REQ-A"}, Expr: "true"}, {ID: "INV-A", Requirements: []string{"REQ-A"}, Expr: "true"}}
		}, machinePath + ".invariants[1]"},
		"machine repeats a unit": {func(m *model.Machine) { m.ID = "FU-A" }, machinePath},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var got []Diagnostic
			for _, d := range behaviorDiags(machineBundle(c.edit), "model.duplicate_id") {
				got = append(got, d)
			}
			wantOne(t, got, SeverityError, c.path)
		})
	}
	if got := behaviorDiags(machineBundle(nil), "model.duplicate_id"); len(got) != 0 {
		t.Fatalf("distinct IDs must be accepted, got %+v", got)
	}
}

// guarded_by may start at a machine or a machine state and target a control,
// trust boundary or referenced element; the legacy state relationship types
// are rejected with the removal message.
// Mutation: dropping the machine pairs rejects the machine source.
// TRLC-LINKS: REQ-EMG-139
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestGuardedByAcceptsMachinesAndRejectsRemovedStateRelationships(t *testing.T) {
	b := behaviorTestBundle([]model.Mapping{
		{Type: "guarded_by", From: "STATE-RUN", To: "TB-A"},
		{Type: "guarded_by", From: "SM-RUN", To: "TB-A"},
	}, nil)
	b.Architecture.AuthoredArchitecture.Controls = []model.Control{{ID: "CTRL-A", Name: "Gate"}}
	if got := behaviorDiags(b, "model.invalid_mapping_pair"); len(got) != 0 {
		t.Fatalf("machine and machine state are guarded_by sources, got %+v", got)
	}
	b.Architecture.AuthoredArchitecture.Mappings = append(b.Architecture.AuthoredArchitecture.Mappings,
		model.Mapping{Type: "guarded_by", From: "CTRL-A", To: "TB-A"},
		model.Mapping{Type: "transitions_to", From: "STATE-IDLE", To: "STATE-RUN"},
		model.Mapping{Type: "triggered_by", From: "STATE-RUN", To: "EVT-START"})
	pairs := behaviorDiags(b, "model.invalid_mapping_pair")
	if len(pairs) != 1 || pairs[0].Path != "authoredArchitecture.mappings[2]" {
		t.Fatalf("a control is not a guarded_by source, got %+v", pairs)
	}
	removed := behaviorDiags(b, "model.unknown_mapping_type")
	if len(removed) != 2 || removed[0].Path != "authoredArchitecture.mappings[3]" || removed[1].Path != "authoredArchitecture.mappings[4]" {
		t.Fatalf("want both removed types rejected, got %+v", removed)
	}
	for i, typ := range []string{"transitions_to", "triggered_by"} {
		want := fmt.Sprintf("mapping type %q was removed in v0.11.0: model state behavior with behavior.machines (see ADR-EMG-038)", typ)
		if removed[i].Message != want {
			t.Fatalf("message = %q, want %q", removed[i].Message, want)
		}
	}
}

// An authored behavior event that no machine transition, semantic
// transition trigger or flow step names is a warning at the event; an event
// a machine transition names in on, and an imported alias::EVT, are not
// reported.
// Mutation: not counting machine transition triggers as uses reports EVT-START too.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestBehaviorEventUnusedWarnsAtTheEvent(t *testing.T) {
	b := machineBundle(func(m *model.Machine) { m.Transitions[1].On = ""; m.Transitions[1].Guard = "flag" })
	got := behaviorDiags(b, "behavior.event_unused")
	if len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Path != "authoredArchitecture.events[1]" || !strings.Contains(got[0].Message, "EVT-TERM") {
		t.Fatalf("want one behavior.event_unused warning for EVT-TERM at authoredArchitecture.events[1], got %+v", got)
	}

	b.Architecture.AuthoredArchitecture.Machines[0].Transitions[1].On = "EVT-TERM"
	b.Architecture.AuthoredArchitecture.Events = append(b.Architecture.AuthoredArchitecture.Events, model.Event{ID: "dep::EVT-REMOTE", Name: "Remote"})
	if got := behaviorDiags(b, "behavior.event_unused"); len(got) != 0 {
		t.Fatalf("used and imported events must not be reported, got %+v", got)
	}
}

// A flow step with boundaryCrossing: true and no onError target is a warning
// at the step's onError field; a crossing step with onError and a local step
// without one are not reported.
// Mutation: dropping the boundaryCrossing condition reports the local step.
// TRLC-LINKS: REQ-EMG-070
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestBehaviorFlowStepBoundaryCrossingWithoutOnErrorWarns(t *testing.T) {
	flow := model.Flow{ID: "FLOW-A", Title: "A", Kind: "data", Entry: []string{"local"}, Steps: []model.FlowStep{
		{ID: "local", Ref: "FU-A", Next: []string{"send"}},
		{ID: "send", Ref: "FU-A", BoundaryCrossing: true, TrustBoundaryRef: "TB-A", Next: []string{"guarded"}},
		{ID: "guarded", Ref: "FU-A", BoundaryCrossing: true, TrustBoundaryRef: "TB-A", OnError: []string{"local"}},
	}}
	got := behaviorDiags(behaviorTestBundle(nil, []model.Flow{flow}), "behavior.flow_step_no_onerror")
	if len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Path != "authoredArchitecture.flows[0].steps[1].onError" ||
		!strings.Contains(got[0].Message, "FLOW-A") || !strings.Contains(got[0].Message, `"send"`) {
		t.Fatalf("want one behavior.flow_step_no_onerror for FLOW-A step send, got %+v", got)
	}
}

// A control verification verificationRefs entry must name a requirement or an
// inferred CODE identifier of the model's code; anything else is an error at
// the entry.
// Mutation: dropping the kind check accepts the functional unit FU-A.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestControlVerificationRefsMustNameTestsOrRequirements(t *testing.T) {
	b := behaviorTestBundle(nil, nil)
	b.Architecture.AuthoredArchitecture.Controls = []model.Control{{ID: "CTRL-A", Name: "Control"}}
	b.Requirements.Requirements = []model.Requirement{{ID: "REQ-A", Text: "x"}}
	b.CodeIndex = func() map[string]int { return map[string]int{"CODE-A-TEST-GO--TESTA": 1} }
	b.Architecture.AuthoredArchitecture.ControlVerifications = []model.ControlVerification{{
		ID: "CV-A", ControlRef: "CTRL-A", VerificationRefs: []string{"REQ-A", "CODE-A-TEST-GO--TESTA", "FU-A", "CODE-GONE"},
	}}
	var got []string
	for _, d := range Bundle(b) {
		if d.Code == "model.invalid_control_verification_ref" && d.Severity == SeverityError {
			got = append(got, d.Path)
		}
	}
	want := []string{"authoredArchitecture.controlVerifications[0].verificationRefs[2]", "authoredArchitecture.controlVerifications[0].verificationRefs[3]"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("want model.invalid_control_verification_ref at %v, got %v", want, got)
	}
}
