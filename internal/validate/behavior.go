// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// behaviorDiagnostics reports behavior that is declared but not wired, as
// warnings:
//
//   - behavior.event_unused: an authored behavior event that no triggered_by
//     relationship targets, no semantic transition lists as a trigger and no
//     semantic relationship, flow or flow step references. Catalog-only events
//     are EARS vocabulary and are covered by catalog.term_unreferenced.
//   - behavior.state_unreachable: in a model with at least one transition
//     (transitions_to relationship or semantic transition), an authored state
//     other than the first declared state that no transition enters. The
//     schema has no initial-state marker, so the first declared state is the
//     initial state; without any transition nothing is reported.
//   - behavior.flow_step_no_onerror: a flow step with boundaryCrossing: true
//     and no onError target.
//
// Imported alias::ID entities belong to their own model and are skipped.
// TRLC-LINKS: REQ-EMG-070
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func behaviorDiagnostics(a model.AuthoredArchitecture, semantics model.SemanticContent) []Diagnostic {
	var diags []Diagnostic
	usedEvents := map[string]bool{}
	entered := map[string]bool{}
	transitions := 0
	for _, m := range a.Mappings {
		switch strings.TrimSpace(m.Type) {
		case "triggered_by":
			usedEvents[strings.TrimSpace(m.To)] = true
		case "transitions_to":
			transitions++
			entered[strings.TrimSpace(m.To)] = true
		}
	}
	for _, r := range semantics.Relationships {
		usedEvents[strings.TrimSpace(r.Source)] = true
		usedEvents[strings.TrimSpace(r.Target)] = true
		for _, trigger := range r.Triggers {
			usedEvents[strings.TrimSpace(trigger)] = true
		}
		if r.Kind == model.RelationshipTransition {
			transitions++
			entered[strings.TrimSpace(r.Target)] = true
		}
	}
	for _, f := range a.Flows {
		usedEvents[strings.TrimSpace(f.SourceRef)] = true
		usedEvents[strings.TrimSpace(f.DestinationRef)] = true
		for _, s := range f.Steps {
			usedEvents[strings.TrimSpace(s.Ref)] = true
			usedEvents[strings.TrimSpace(s.SourceRef)] = true
			usedEvents[strings.TrimSpace(s.DestinationRef)] = true
		}
	}

	for i, e := range a.Events {
		id := strings.TrimSpace(e.ID)
		if id == "" || strings.Contains(id, "::") || usedEvents[id] {
			continue
		}
		diags = append(diags, Diagnostic{Code: "behavior.event_unused", Severity: SeverityWarning,
			Message: fmt.Sprintf("event %q is in no triggered_by relationship, transition trigger or flow; add the state it triggers or remove the event", id),
			Path:    fmt.Sprintf("authoredArchitecture.events[%d]", i)})
	}

	if transitions > 0 {
		initial := true
		for i, s := range a.States {
			id := strings.TrimSpace(s.ID)
			if id == "" || strings.Contains(id, "::") {
				continue
			}
			if initial {
				initial = false
				continue
			}
			if entered[id] {
				continue
			}
			diags = append(diags, Diagnostic{Code: "behavior.state_unreachable", Severity: SeverityWarning,
				Message: fmt.Sprintf("state %q is the target of no transition and is not the initial (first declared) state; add a transitions_to into it", id),
				Path:    fmt.Sprintf("authoredArchitecture.states[%d]", i)})
		}
	}

	for i, f := range a.Flows {
		for j, s := range f.Steps {
			if !s.BoundaryCrossing || len(s.OnError) > 0 {
				continue
			}
			diags = append(diags, Diagnostic{Code: "behavior.flow_step_no_onerror", Severity: SeverityWarning,
				Message: fmt.Sprintf("flow %q step %q crosses a trust boundary but has no onError step; model the failure path", strings.TrimSpace(f.ID), strings.TrimSpace(s.ID)),
				Path:    fmt.Sprintf("authoredArchitecture.flows[%d].steps[%d].onError", i, j)})
		}
	}
	return diags
}
