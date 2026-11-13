// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package reqlint

import (
	"reflect"
	"sort"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// modelingCatalog names the terms the fixture requirements use.
func modelingCatalog() model.CatalogDocument {
	return model.CatalogDocument{Catalog: model.CatalogGroups{
		Systems:   []model.CatalogEntry{{ID: "SYS-ACME", Name: "acme system"}},
		Events:    []model.CatalogEntry{{ID: "EVT-PING", Name: "ping event"}},
		States:    []model.CatalogEntry{{ID: "STATE-READY", Name: "ready"}},
		DataTerms: []model.CatalogEntry{{ID: "DATA-PONG", Name: "pong"}},
	}}
}

const (
	textEvent    = "When ping event is received, the acme system shall publish pong."
	textState    = "While ready is active, the acme system shall publish pong."
	textUbiq     = "The acme system shall publish pong."
	textUnwanted = "If ping event is received, then the acme system shall publish pong."
)

// requirementPatterns names the EARS pattern of each requirement.
// Mutation: dropping the pattern from the result map empties it.
// TRLC-LINKS: REQ-EMG-009, REQ-EMG-172
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestRequirementPatterns(t *testing.T) {
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{
		{ID: "REQ-EVENT", Text: textEvent}, {ID: "REQ-STATE", Text: textState}, {ID: "REQ-UBIQ", Text: textUbiq},
	}}
	got := requirementPatterns(reqs, modelingCatalog())
	want := map[string]string{"REQ-EVENT": "event-driven", "REQ-STATE": "state-driven", "REQ-UBIQ": "ubiquitous"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("patterns %v, want %v", got, want)
	}
}

// modeledMachine is the machine of the fixture: a state, a transition and an
// invariant that lists REQ-LISTED.
var modeledMachine = model.Machine{
	ID: "SM-A", States: []model.MachineState{{ID: "STATE-A"}}, Transitions: []model.MachineTransition{{ID: "TR-A"}},
	Invariants: []model.MachineInvariant{{ID: "INV-A", Requirements: []string{"REQ-LISTED"}}},
}

func withMachine(a *model.AuthoredArchitecture) { a.Machines = []model.Machine{modeledMachine} }

func withEvent(a *model.AuthoredArchitecture) {
	withMachine(a)
	a.Events = []model.Event{{ID: "EVT-A"}}
}

func withFlow(a *model.AuthoredArchitecture) {
	withMachine(a)
	a.Flows = []model.Flow{{ID: "FLOW-A"}}
}

func withUnit(a *model.AuthoredArchitecture) {
	withMachine(a)
	a.FunctionalUnits = []model.FunctionalUnit{{ID: "FU-A"}}
}

// req is a requirement of the fixture with its appliesTo.
func req(id, text string, appliesTo ...string) model.Requirement {
	return model.Requirement{ID: id, Text: text, AppliesTo: appliesTo}
}

// modeledCase is one model and the requirement paths reported on it.
type modeledCase struct {
	name  string
	setup func(*model.AuthoredArchitecture)
	reqs  []model.Requirement
	want  []string
}

// modeledCases are the models of TestLintRequirementsModeled.
func modeledCases() []modeledCase {
	retired, rejected, derived := req("REQ-1", textEvent), req("REQ-2", textEvent), req("REQ-3", textEvent)
	retired.Status, rejected.Status, derived.Derived = "Retired", "rejected", true
	none := []string{}
	return []modeledCase{
		{"no machine, no check", nil, []model.Requirement{req("REQ-1", textEvent)}, none},
		{"event and state driven are reported", withMachine, []model.Requirement{req("REQ-1", textEvent), req("REQ-2", textState)}, []string{"requirements[REQ-1]", "requirements[REQ-2]"}},
		{"ubiquitous and unwanted are not behavior", withMachine, []model.Requirement{req("REQ-1", textUbiq), req("REQ-2", textUnwanted)}, none},
		{"listed by an invariant", withMachine, []model.Requirement{req("REQ-LISTED", textEvent)}, none},
		{"appliesTo a machine", withMachine, []model.Requirement{req("REQ-1", textEvent, "SM-A")}, none},
		{"appliesTo a state", withMachine, []model.Requirement{req("REQ-1", textState, "STATE-A")}, none},
		{"appliesTo a transition", withMachine, []model.Requirement{req("REQ-1", textEvent, " TR-A ")}, none},
		{"appliesTo an event", withEvent, []model.Requirement{req("REQ-1", textEvent, "EVT-A")}, none},
		{"appliesTo a flow", withFlow, []model.Requirement{req("REQ-1", textEvent, "FLOW-A")}, none},
		{"appliesTo a unit only", withUnit, []model.Requirement{req("REQ-1", textEvent, "FU-A")}, []string{"requirements[REQ-1]"}},
		{"retired, rejected and derived", withMachine, []model.Requirement{retired, rejected, derived}, none},
	}
}

// reportedPaths runs the check on a case and returns the sorted paths of the
// diagnostics, after checking their shape.
func reportedPaths(t *testing.T, c modeledCase) []string {
	t.Helper()
	bundle := model.Bundle{Catalog: modelingCatalog()}
	if c.setup != nil {
		c.setup(&bundle.Architecture.AuthoredArchitecture)
	}
	got := []string{}
	for _, d := range lintRequirementsModeled(bundle, model.RequirementsDocument{Requirements: c.reqs}) {
		if d.Code != "behavior.requirement_unmodeled" || d.Remedy == "" || d.Severity != "warning" {
			t.Errorf("%s: %+v", c.name, d)
		}
		got = append(got, d.Path)
	}
	sort.Strings(got)
	return got
}

// lintRequirementsModeled reports an event-driven or state-driven requirement
// that no invariant lists and whose appliesTo names no behavior element, only
// in a model with a machine; retired, rejected and derived requirements and
// other patterns are exempt.
// Mutation: dropping the invariant lookup, the appliesTo lookup, the machine
// precondition or an exemption changes the reported set of its case.
// TRLC-LINKS: REQ-EMG-172
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestLintRequirementsModeled(t *testing.T) {
	for _, c := range modeledCases() {
		if got := reportedPaths(t, c); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: reported %v, want %v", c.name, got, c.want)
		}
	}
}

// LintModelRequirementsEARS, the entry the publication and the delta use,
// carries the check.
// Mutation: removing the call from LintModelRequirementsEARS drops the code.
// TRLC-LINKS: REQ-EMG-172
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestLintModelRequirementsEARSReportsUnmodeled(t *testing.T) {
	bundle := model.Bundle{Catalog: modelingCatalog()}
	bundle.Architecture.AuthoredArchitecture.Machines = []model.Machine{{ID: "SM-A"}}
	diags := LintModelRequirementsEARS(bundle, model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-1", Text: textEvent}}})
	found := false
	for _, d := range diags {
		found = found || d.Code == "behavior.requirement_unmodeled"
	}
	if !found {
		t.Fatalf("no behavior.requirement_unmodeled in %+v", diags)
	}
}
