// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// behaviorTestBundle is a valid bundle with two states, two events and the
// given relationships and flows.
func behaviorTestBundle(mappings []model.Mapping, flows []model.Flow) model.Bundle {
	return schemaV2TestBundle(model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit"}},
		TrustBoundaries:  []model.TrustBoundary{{ID: "TB-A", Name: "Boundary"}},
		States:           []model.State{{ID: "STATE-IDLE", Name: "Idle"}, {ID: "STATE-RUN", Name: "Running"}, {ID: "STATE-STOP", Name: "Stopped"}},
		Events:           []model.Event{{ID: "EVT-START", Name: "Start"}, {ID: "EVT-TERM", Name: "Termination signal"}},
		Flows:            flows,
		Mappings:         mappings,
	}}})
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

// An authored behavior event that no triggered_by relationship, semantic
// transition trigger or flow step names is a warning at the event; a used
// event and an imported alias::EVT are not reported.
// Mutation: not marking triggered_by targets as used reports EVT-START too.
// TRLC-LINKS: REQ-EMG-070
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestBehaviorEventUnusedWarnsAtTheEvent(t *testing.T) {
	b := behaviorTestBundle([]model.Mapping{
		{Type: "transitions_to", From: "STATE-IDLE", To: "STATE-RUN"},
		{Type: "transitions_to", From: "STATE-RUN", To: "STATE-STOP"},
		{Type: "triggered_by", From: "STATE-RUN", To: "EVT-START"},
	}, nil)
	got := behaviorDiags(b, "behavior.event_unused")
	if len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Path != "authoredArchitecture.events[1]" || !strings.Contains(got[0].Message, "EVT-TERM") {
		t.Fatalf("want one behavior.event_unused warning for EVT-TERM at authoredArchitecture.events[1], got %+v", got)
	}

	b.Architecture.AuthoredArchitecture.Mappings = append(b.Architecture.AuthoredArchitecture.Mappings, model.Mapping{Type: "triggered_by", From: "STATE-STOP", To: "EVT-TERM"})
	b.Architecture.AuthoredArchitecture.Events = append(b.Architecture.AuthoredArchitecture.Events, model.Event{ID: "dep::EVT-REMOTE", Name: "Remote"})
	if got := behaviorDiags(b, "behavior.event_unused"); len(got) != 0 {
		t.Fatalf("used and imported events must not be reported, got %+v", got)
	}
}

// With transitions in the model, a state other than the first declared one
// (the initial state) that no transition enters is a warning at the state;
// a model without any transition reports nothing.
// Mutation: treating every transition source as reached, or dropping the
// no-transitions guard, fails this test.
// TRLC-LINKS: REQ-EMG-070
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestBehaviorStateUnreachableWarnsAtTheState(t *testing.T) {
	b := behaviorTestBundle([]model.Mapping{
		{Type: "transitions_to", From: "STATE-RUN", To: "STATE-IDLE"},
		{Type: "triggered_by", From: "STATE-RUN", To: "EVT-START"},
		{Type: "triggered_by", From: "STATE-STOP", To: "EVT-TERM"},
	}, nil)
	got := behaviorDiags(b, "behavior.state_unreachable")
	if len(got) != 2 || got[0].Path != "authoredArchitecture.states[1]" || got[1].Path != "authoredArchitecture.states[2]" ||
		got[0].Severity != SeverityWarning || !strings.Contains(got[0].Message, "STATE-RUN") || !strings.Contains(got[1].Message, "STATE-STOP") {
		t.Fatalf("want behavior.state_unreachable for STATE-RUN and STATE-STOP, got %+v", got)
	}

	b.Architecture.AuthoredArchitecture.Mappings = append(b.Architecture.AuthoredArchitecture.Mappings,
		model.Mapping{Type: "transitions_to", From: "STATE-IDLE", To: "STATE-RUN"},
		model.Mapping{Type: "transitions_to", From: "STATE-RUN", To: "STATE-STOP"})
	if got := behaviorDiags(b, "behavior.state_unreachable"); len(got) != 0 {
		t.Fatalf("every state is reached, got %+v", got)
	}

	none := behaviorTestBundle([]model.Mapping{{Type: "triggered_by", From: "STATE-RUN", To: "EVT-START"}}, nil)
	if got := behaviorDiags(none, "behavior.state_unreachable"); len(got) != 0 {
		t.Fatalf("a model without transitions must not report unreachable states, got %+v", got)
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
