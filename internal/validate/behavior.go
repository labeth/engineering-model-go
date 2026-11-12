// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// behaviorDiagnostics reports behavior that is declared but not wired, as
// warnings, and validates the machines (machineDiagnostics):
//
//   - behavior.event_unused: an authored behavior event that no machine
//     transition names in on, no semantic transition lists as a trigger and no
//     semantic relationship, flow or flow step references. Catalog-only events
//     are EARS vocabulary and are covered by catalog.term_unreferenced.
//   - behavior.state_unreachable: a machine state that no path of transitions
//     leads to from the machine's initial state.
//   - behavior.flow_step_no_onerror: a flow step with boundaryCrossing: true
//     and no onError target.
//
// Imported alias::ID entities belong to their own model and are skipped.
// requirements holds the IDs of the model's requirements, for invariant links.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func behaviorDiagnostics(a model.AuthoredArchitecture, semantics model.SemanticContent, requirements map[string]bool) []Diagnostic {
	var diags []Diagnostic
	usedEvents := map[string]bool{}
	for _, m := range a.Machines {
		for _, t := range m.Transitions {
			usedEvents[strings.TrimSpace(t.On)] = true
		}
	}
	for _, r := range semantics.Relationships {
		usedEvents[strings.TrimSpace(r.Source)] = true
		usedEvents[strings.TrimSpace(r.Target)] = true
		for _, trigger := range r.Triggers {
			usedEvents[strings.TrimSpace(trigger)] = true
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
			Message: fmt.Sprintf("event %q is in no machine transition (on), semantic transition trigger or flow; name it in the on of the machine transition it causes or remove the event", id),
			Path:    fmt.Sprintf("authoredArchitecture.events[%d]", i)})
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
	return append(diags, machineDiagnostics(a, requirements)...)
}
