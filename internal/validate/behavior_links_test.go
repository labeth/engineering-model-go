// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// unlinkedCodes are the codes of the requirement links of machine elements.
var unlinkedCodes = []string{"behavior.state_unlinked", "behavior.transition_unlinked", "behavior.event_unlinked"}

// linkDiags returns the unlinked diagnostics of a bundle as "code path".
func linkDiags(b model.Bundle) []string {
	var out []string
	for _, code := range unlinkedCodes {
		for _, d := range behaviorDiags(b, code) {
			out = append(out, code+" "+d.Path)
		}
	}
	return out
}

// An element that lists a requirement in its own requirements is linked, and
// the three codes name each of the others when they list none: initial and
// final states are asked too, and an event no transition uses is not.
// Mutation: treating an empty list as linked, or skipping a kind, drops a row.
// TRLC-LINKS: REQ-EMG-178
func TestBehaviorElementsWithoutRequirementsAreReported(t *testing.T) {
	cases := map[string]struct {
		edit func(*model.Bundle)
		want []string
	}{
		"all linked": {func(*model.Bundle) {}, nil},
		"initial state": {func(b *model.Bundle) {
			b.Architecture.AuthoredArchitecture.Machines[0].States[0].Requirements = nil
		}, []string{"behavior.state_unlinked " + machinePath + ".states[0].requirements"}},
		"final state": {func(b *model.Bundle) {
			b.Architecture.AuthoredArchitecture.Machines[0].States[2].Requirements = []string{" "}
		}, []string{"behavior.state_unlinked " + machinePath + ".states[2].requirements"}},
		"transition": {func(b *model.Bundle) {
			b.Architecture.AuthoredArchitecture.Machines[0].Transitions[1].Requirements = []string{}
		}, []string{"behavior.transition_unlinked " + machinePath + ".transitions[1].requirements"}},
		"event used by a transition": {func(b *model.Bundle) {
			b.Architecture.AuthoredArchitecture.Events[1].Requirements = nil
		}, []string{"behavior.event_unlinked authoredArchitecture.events[1].requirements"}},
		"event no transition uses": {func(b *model.Bundle) {
			a := &b.Architecture.AuthoredArchitecture
			a.Events = append(a.Events, model.Event{ID: "EVT-IDLE", Name: "Unused"})
		}, nil},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			b := machineBundle(nil)
			c.edit(&b)
			got := linkDiags(b)
			if strings.Join(got, ";") != strings.Join(c.want, ";") {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

// The message names the element and its machine, the diagnostic is a warning
// that carries the ef-behavior remedy, and an event names every machine that
// uses it.
// Mutation: dropping the machine from the message or the remedy fails it.
// TRLC-LINKS: REQ-EMG-178
func TestBehaviorUnlinkedMessageAndRemedy(t *testing.T) {
	b := machineBundle(func(m *model.Machine) {
		m.States[1].Requirements = nil
		m.Transitions[0].Requirements = nil
	})
	other := baseTestMachine()
	other.ID = "SM-OTHER"
	other.States = []model.MachineState{{ID: "STATE-O1", Requirements: []string{"REQ-A"}}}
	other.Transitions = []model.MachineTransition{{ID: "TR-O1", From: "STATE-O1", To: "STATE-O1", On: "EVT-START", Requirements: []string{"REQ-A"}}}
	other.Initial = "STATE-O1"
	a := &b.Architecture.AuthoredArchitecture
	a.Machines = append(a.Machines, other)
	a.Events[0].Requirements = nil
	for code, fragments := range map[string][]string{
		"behavior.state_unlinked":      {`state "STATE-RUN"`, `machine "SM-RUN"`},
		"behavior.transition_unlinked": {`transition "TR-START"`, `machine "SM-RUN"`},
		"behavior.event_unlinked":      {`event "EVT-START"`, `"SM-OTHER"`, `"SM-RUN"`},
	} {
		wantUnlinkedWarning(t, behaviorDiags(b, code), fragments)
	}
}

// wantUnlinkedWarning fails unless diags holds one warning with the
// ef-behavior remedy and a message that contains every fragment.
// TRLC-LINKS: REQ-EMG-178
func wantUnlinkedWarning(t *testing.T, diags []Diagnostic, fragments []string) {
	t.Helper()
	if len(diags) != 1 || diags[0].Severity != SeverityWarning || !strings.Contains(diags[0].Remedy, "ef-behavior") {
		t.Fatalf("want one warning with the ef-behavior remedy, got %+v", diags)
	}
	for _, f := range fragments {
		if !strings.Contains(diags[0].Message, f) {
			t.Fatalf("message %q lacks %q", diags[0].Message, f)
		}
	}
}

// Only the element's own list links it: an invariant of the machine that
// lists a requirement does not link the machine's states or transitions.
// Mutation: counting machine-level invariant requirements as links hides
// these two diagnostics.
// TRLC-LINKS: REQ-EMG-178
func TestBehaviorInvariantRequirementsDoNotLinkElements(t *testing.T) {
	b := machineBundle(func(m *model.Machine) {
		m.Invariants = []model.MachineInvariant{{ID: "INV-A", Requirements: []string{"REQ-A"}, Expr: "count >= 0"}}
		m.States[0].Requirements = nil
		m.Transitions[0].Requirements = nil
	})
	want := "behavior.state_unlinked " + machinePath + ".states[0].requirements;behavior.transition_unlinked " + machinePath + ".transitions[0].requirements"
	if got := strings.Join(linkDiags(b), ";"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// A model that declares no machine is not asked, and imported alias::ID
// machines and events are skipped.
// Mutation: dropping the alias skip or asking events without a machine adds a
// diagnostic.
// TRLC-LINKS: REQ-EMG-177, REQ-EMG-178
func TestBehaviorLinksSkipModelsWithoutMachinesAndAliases(t *testing.T) {
	bare := model.Machine{ID: "SM-X", States: []model.MachineState{{ID: "STATE-X"}},
		Transitions: []model.MachineTransition{{ID: "TR-X", From: "STATE-X", To: "STATE-X", On: "EVT-X"}}}
	imported := bare
	imported.ID = "alias::SM-X"
	cases := map[string]struct {
		a    model.AuthoredArchitecture
		want int
	}{
		"no machines":        {model.AuthoredArchitecture{Events: []model.Event{{ID: "EVT-X"}}}, 0},
		"local machine":      {model.AuthoredArchitecture{Machines: []model.Machine{bare}, Events: []model.Event{{ID: "EVT-X"}}}, 3},
		"alias machine":      {model.AuthoredArchitecture{Machines: []model.Machine{imported}, Events: []model.Event{{ID: "EVT-X"}}}, 0},
		"alias event":        {model.AuthoredArchitecture{Machines: []model.Machine{bare}, Events: []model.Event{{ID: "alias::EVT-X"}}}, 2},
		"alias event listed": {model.AuthoredArchitecture{Events: []model.Event{{ID: "alias::EVT-X", Requirements: []string{"REQ-NOPE"}}}}, 0},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := behaviorLinkDiagnostics(c.a, map[string]bool{}); len(got) != c.want {
				t.Fatalf("want %d diagnostics, got %+v", c.want, got)
			}
		})
	}
}

// A listed requirement ID that is no requirement of the model is a
// semantic.dangling_reference error at that entry, for a state, a transition
// and an event, and it also counts as listed (no unlinked warning); a real ID
// is accepted.
// Mutation: skipping the existence lookup accepts REQ-NOPE.
// TRLC-LINKS: REQ-EMG-177
func TestBehaviorElementRequirementMustExist(t *testing.T) {
	cases := map[string]struct {
		edit func(*model.Bundle)
		path string
	}{
		"state": {func(b *model.Bundle) {
			b.Architecture.AuthoredArchitecture.Machines[0].States[1].Requirements = []string{"REQ-A", "REQ-NOPE"}
		}, machinePath + ".states[1].requirements[1]"},
		"transition": {func(b *model.Bundle) {
			b.Architecture.AuthoredArchitecture.Machines[0].Transitions[0].Requirements = []string{"REQ-NOPE"}
		}, machinePath + ".transitions[0].requirements[0]"},
		"event": {func(b *model.Bundle) {
			b.Architecture.AuthoredArchitecture.Events[0].Requirements = []string{"REQ-NOPE"}
		}, "authoredArchitecture.events[0].requirements[0]"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			b := machineBundle(nil)
			c.edit(&b)
			wantOne(t, behaviorDiags(b, "semantic.dangling_reference"), SeverityError, c.path, "REQ-NOPE", name)
			if got := linkDiags(b); len(got) != 0 {
				t.Fatalf("a listed requirement is not unlinked, got %v", got)
			}
		})
	}
	if got := behaviorDiags(machineBundle(nil), "semantic.dangling_reference"); len(got) != 0 {
		t.Fatalf("existing requirements must be accepted, got %+v", got)
	}
}
