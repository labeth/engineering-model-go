// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// The codes of the requirement links of machine elements (ADR-EMG-069 item 3).
// The three unlinked codes are warnings that the tier policy keys with the
// hygiene row; an unknown requirement ID is the error an invariant gets.
const (
	codeStateUnlinked      = "behavior.state_unlinked"
	codeTransitionUnlinked = "behavior.transition_unlinked"
	codeEventUnlinked      = "behavior.event_unlinked"
	codeDanglingReference  = "semantic.dangling_reference"
)

// linkSubject is one element that may list requirements: a state or
// transition of a machine, or an event. unlinked is the code reported when it
// lists none, empty for an element that is not asked.
type linkSubject struct {
	unlinked     string
	kind         string
	id           string
	owner        string // the machine or machines, already phrased
	path         string
	requirements []string
}

// behaviorLinkDiagnostics reports, for each state and transition of an
// authored machine and each event a machine transition triggers on, the
// requirements it lists: behavior.state_unlinked, behavior.transition_unlinked
// or behavior.event_unlinked (warnings, tier keyed) when it lists none, and
// semantic.dangling_reference (an error) for each listed ID that is no
// requirement of the model. Only the element's own requirements list counts: a
// requirement the machine's invariants list does not link its elements. A
// model without a machine is not asked, and imported alias::ID machines and
// events are skipped. requirements holds the IDs of the model's requirements.
// TRLC-LINKS: REQ-EMG-177, REQ-EMG-178
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func behaviorLinkDiagnostics(a model.AuthoredArchitecture, requirements map[string]bool) []Diagnostic {
	var diags []Diagnostic
	for _, s := range linkSubjects(a) {
		diags = append(diags, s.danglingDiagnostics(requirements)...)
		if s.unlinked != "" && len(s.listed()) == 0 {
			diags = append(diags, s.unlinkedDiagnostic())
		}
	}
	return diags
}

// linkSubjects lists the states and transitions of the local machines and
// every local event, in model order. Only events that a machine transition
// uses carry an unlinked code.
// TRLC-LINKS: REQ-EMG-177, REQ-EMG-178
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func linkSubjects(a model.AuthoredArchitecture) []linkSubject {
	var subjects []linkSubject
	users := map[string][]string{}
	for i, m := range a.Machines {
		if isImportedID(m.ID) {
			continue
		}
		subjects = append(subjects, machineLinkSubjects(m, fmt.Sprintf("authoredArchitecture.machines[%d]", i))...)
		noteEventUsers(users, m)
	}
	return append(subjects, eventLinkSubjects(a.Events, users)...)
}

// machineLinkSubjects lists the states and transitions of one machine.
// TRLC-LINKS: REQ-EMG-177, REQ-EMG-178
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func machineLinkSubjects(m model.Machine, path string) []linkSubject {
	var subjects []linkSubject
	owner := fmt.Sprintf("machine %q", strings.TrimSpace(m.ID))
	for i, s := range m.States {
		subjects = append(subjects, linkSubject{unlinked: codeStateUnlinked, kind: "state", id: strings.TrimSpace(s.ID),
			owner: owner, path: fmt.Sprintf("%s.states[%d]", path, i), requirements: s.Requirements})
	}
	for i, t := range m.Transitions {
		subjects = append(subjects, linkSubject{unlinked: codeTransitionUnlinked, kind: "transition", id: strings.TrimSpace(t.ID),
			owner: owner, path: fmt.Sprintf("%s.transitions[%d]", path, i), requirements: t.Requirements})
	}
	return subjects
}

// noteEventUsers records m as a user of each event its transitions name in on.
// TRLC-LINKS: REQ-EMG-178
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func noteEventUsers(users map[string][]string, m model.Machine) {
	machine := strings.TrimSpace(m.ID)
	for _, t := range m.Transitions {
		on := strings.TrimSpace(t.On)
		if on != "" && !containsString(users[on], machine) {
			users[on] = append(users[on], machine)
		}
	}
}

// eventLinkSubjects lists the local events; an event that no machine
// transition triggers on is not asked for requirements.
// TRLC-LINKS: REQ-EMG-178
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func eventLinkSubjects(events []model.Event, users map[string][]string) []linkSubject {
	var subjects []linkSubject
	for i, e := range events {
		id := strings.TrimSpace(e.ID)
		if isImportedID(id) {
			continue
		}
		s := linkSubject{kind: "event", id: id, path: fmt.Sprintf("authoredArchitecture.events[%d]", i), requirements: e.Requirements}
		if machines := users[id]; len(machines) > 0 {
			s.unlinked, s.owner = codeEventUnlinked, phraseMachines(machines)
		}
		subjects = append(subjects, s)
	}
	return subjects
}

// phraseMachines names one machine or several, in ID order.
// TRLC-LINKS: REQ-EMG-178
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func phraseMachines(machines []string) string {
	sort.Strings(machines)
	if len(machines) == 1 {
		return fmt.Sprintf("machine %q", machines[0])
	}
	return `machines "` + strings.Join(machines, `", "`) + `"`
}

// listed returns the non-blank requirement IDs of the element.
// TRLC-LINKS: REQ-EMG-177
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s linkSubject) listed() []string {
	var ids []string
	for _, r := range s.requirements {
		if r = strings.TrimSpace(r); r != "" {
			ids = append(ids, r)
		}
	}
	return ids
}

// danglingDiagnostics reports each listed ID that is no requirement.
// TRLC-LINKS: REQ-EMG-177
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s linkSubject) danglingDiagnostics(requirements map[string]bool) []Diagnostic {
	var diags []Diagnostic
	for i, r := range s.requirements {
		if r = strings.TrimSpace(r); r == "" || requirements[r] {
			continue
		}
		diags = append(diags, Diagnostic{Code: codeDanglingReference, Severity: SeverityError,
			Message: fmt.Sprintf("%s %q lists requirement %q, which does not exist; list a REQ-* of the model", s.kind, s.id, r),
			Path:    fmt.Sprintf("%s.requirements[%d]", s.path, i)})
	}
	return diags
}

// unlinkedDiagnostic reports an element that lists no requirement.
// TRLC-LINKS: REQ-EMG-178
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s linkSubject) unlinkedDiagnostic() Diagnostic {
	where := ""
	if s.owner != "" {
		where = " of " + s.owner
	}
	return Diagnostic{Code: s.unlinked, Severity: SeverityWarning, Path: s.path + ".requirements",
		Message: fmt.Sprintf("%s %q%s lists no requirement; list the REQ-* it realizes in requirements", s.kind, s.id, where),
		Remedy:  ConsistencyRemedy(s.unlinked)}
}

// isImportedID reports whether an ID names an imported alias::ID entity,
// which belongs to its own model.
// TRLC-LINKS: REQ-EMG-177
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func isImportedID(id string) bool {
	return strings.Contains(id, "::")
}

// containsString reports whether list holds value.
// TRLC-LINKS: REQ-EMG-178
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func containsString(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
