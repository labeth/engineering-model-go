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
	diags := eventUnusedDiagnostics(a, usedEventRefs(a, semantics))
	diags = append(diags, flowStepOnErrorDiagnostics(a)...)
	return append(diags, machineDiagnostics(a, requirements)...)
}

// refSet is a set of trimmed ids.
type refSet map[string]bool

// add puts the trimmed values into the set.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s refSet) add(values ...string) {
	for _, v := range values {
		s[strings.TrimSpace(v)] = true
	}
}

// usedEventRefs is the set of ids that a machine transition, a semantic
// relationship, a flow or a flow step names.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func usedEventRefs(a model.AuthoredArchitecture, semantics model.SemanticContent) refSet {
	used := refSet{}
	for _, m := range a.Machines {
		used.addTransitionEvents(m)
	}
	for _, r := range semantics.Relationships {
		used.add(r.Source, r.Target)
		used.add(r.Triggers...)
	}
	for _, f := range a.Flows {
		used.addFlowRefs(f)
	}
	return used
}

// addTransitionEvents adds the events the transitions of a machine are triggered by.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s refSet) addTransitionEvents(m model.Machine) {
	for _, t := range m.Transitions {
		s.add(t.On)
	}
}

// addFlowRefs adds the source, destination and step references of a flow.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s refSet) addFlowRefs(f model.Flow) {
	s.add(f.SourceRef, f.DestinationRef)
	for _, st := range f.Steps {
		s.add(st.Ref, st.SourceRef, st.DestinationRef)
	}
}

// eventUnusedDiagnostics warns about each local event that is in no machine transition, trigger or flow.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func eventUnusedDiagnostics(a model.AuthoredArchitecture, used refSet) []Diagnostic {
	var diags []Diagnostic
	for i, e := range a.Events {
		id := strings.TrimSpace(e.ID)
		if id == "" || strings.Contains(id, "::") || used[id] {
			continue
		}
		diags = append(diags, Diagnostic{Code: "behavior.event_unused", Severity: SeverityWarning,
			Message: fmt.Sprintf("event %q is in no machine transition (on), semantic transition trigger or flow; name it in the on of the machine transition it causes or remove the event", id),
			Path:    fmt.Sprintf("authoredArchitecture.events[%d]", i)})
	}
	return diags
}

// flowStepOnErrorDiagnostics warns about each flow step that crosses a trust boundary without an onError step.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func flowStepOnErrorDiagnostics(a model.AuthoredArchitecture) []Diagnostic {
	var diags []Diagnostic
	for i, f := range a.Flows {
		diags = append(diags, flowOnErrorDiagnostics(i, f)...)
	}
	return diags
}

// flowOnErrorDiagnostics warns about each step of a flow that crosses a trust boundary without an onError step.
// TRLC-LINKS: REQ-EMG-070, REQ-EMG-138, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func flowOnErrorDiagnostics(i int, f model.Flow) []Diagnostic {
	var diags []Diagnostic
	for j, s := range f.Steps {
		if !s.BoundaryCrossing || len(s.OnError) > 0 {
			continue
		}
		diags = append(diags, Diagnostic{Code: "behavior.flow_step_no_onerror", Severity: SeverityWarning,
			Message: fmt.Sprintf("flow %q step %q crosses a trust boundary but has no onError step; model the failure path", strings.TrimSpace(f.ID), strings.TrimSpace(s.ID)),
			Path:    fmt.Sprintf("authoredArchitecture.flows[%d].steps[%d].onError", i, j)})
	}
	return diags
}
