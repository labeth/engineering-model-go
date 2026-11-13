// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// The element to code coverage checks of ADR-EMG-069 and the codeCoverage
// exemption list: fixtures are in-memory bundles and code items.

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// coverageBundle is a model with three local units (FU-A, FU-B, FU-C), an
// imported unit, a machine SM-A owned by FU-A (STATE-1, STATE-2, TR-1 on
// EVT-1, TR-2 on EVT-2) and a machine SM-X owned by an imported unit, an
// event EVT-3 that only a transition of SM-X uses, an interface IF-A owned by FU-A and
// IF-EXT owned by an imported unit, a flow FLOW-A through IF-A carrying DAT-1,
// a flow FLOW-B that touches no interface carrying DAT-2, an actor, a risk and
// a trust boundary, and the decisions DEC-OK (accepted) and DEC-NO (proposed).
func coverageBundle() model.Bundle {
	b := model.Bundle{}
	a := &b.Architecture.AuthoredArchitecture
	a.FunctionalUnits = []model.FunctionalUnit{{ID: "FU-A"}, {ID: "FU-B"}, {ID: "FU-C"}, {ID: "dep::FU-Q"}}
	a.Actors = []model.Actor{{ID: "ACT-1"}}
	a.Risks = []model.Risk{{ID: "RISK-1"}}
	a.TrustBoundaries = []model.TrustBoundary{{ID: "TB-1"}}
	a.Machines = []model.Machine{
		{ID: "SM-A", Owner: "FU-A",
			States:      []model.MachineState{{ID: "STATE-1"}, {ID: "STATE-2"}},
			Transitions: []model.MachineTransition{{ID: "TR-1", On: "EVT-1"}, {ID: "TR-2", On: "EVT-2"}}},
		{ID: "SM-X", Owner: "dep::FU-Z", States: []model.MachineState{{ID: "STATE-X"}}, Transitions: []model.MachineTransition{{ID: "TR-X", On: "EVT-3"}}},
	}
	a.Events = []model.Event{{ID: "EVT-1"}, {ID: "EVT-2"}, {ID: "EVT-3"}}
	a.Interfaces = []model.Interface{{ID: "IF-A", Owner: "FU-A"}, {ID: "IF-EXT", Owner: "dep::FU-Z"}}
	a.DataObjects = []model.DataObject{{ID: "DAT-1"}, {ID: "DAT-2"}}
	a.Flows = []model.Flow{
		{ID: "FLOW-A", DataRefs: []string{"DAT-1"}, Steps: []model.FlowStep{{ID: "S1", InterfaceRef: "IF-A"}}},
		{ID: "FLOW-B", DataRefs: []string{"DAT-2"}, Steps: []model.FlowStep{{ID: "S1", InterfaceRef: "IF-EXT"}}},
	}
	b.Architecture.Decisions = []model.Decision{{ID: "DEC-OK", Status: "accepted"}, {ID: "DEC-NO", Status: "proposed"}}
	return b
}

// withExemptions returns the bundle with a codeCoverage.exempt list.
func withExemptions(b model.Bundle, exempt ...model.CodeExemption) model.Bundle {
	b.Manifest.CodeCoverage = &model.CodeCoverage{Exempt: exempt}
	return b
}

const goodReason = "implemented by an external system"

// ownerFile is a scanned non-test source file owned by a unit.
func ownerFile(source, owner string) codescan.CodeItem {
	return codescan.CodeItem{Kind: "source_file", Owner: owner, Source: source, Element: source}
}

// linkSymbol is a declaration in a non-test file with ENGMODEL-LINKS ids.
func linkSymbol(ids ...string) codescan.CodeItem {
	return codescan.CodeItem{Kind: "symbol", Source: "a.go:3", Element: "CODE-A", ModelLinks: ids}
}

// reported returns the sorted "code id" of each diagnostic, the id read from
// the path-independent message tail kept in the diagnostic.
func reported(diags []validate.Diagnostic) []string {
	out := []string{}
	for _, d := range diags {
		out = append(out, d.Code+" "+d.Path)
	}
	sort.Strings(out)
	return out
}

// allUnlinked is everything coverageBundle asks about when scanned source
// names only the unit FU-A (a file it owns).
var allUnlinked = []string{
	"code.data_unlinked authoredArchitecture.dataObjects[0]",
	"code.event_unlinked authoredArchitecture.events[0]",
	"code.event_unlinked authoredArchitecture.events[1]",
	"code.flow_unlinked authoredArchitecture.flows[0]",
	"code.interface_unlinked authoredArchitecture.interfaces[0]",
	"code.machine_unlinked authoredArchitecture.machines[1]",
	"code.state_unlinked authoredArchitecture.machines[0].states[0]",
	"code.state_unlinked authoredArchitecture.machines[0].states[1]",
	"code.transition_unlinked authoredArchitecture.machines[0].transitions[0]",
	"code.transition_unlinked authoredArchitecture.machines[0].transitions[1]",
	"code.unit_unlinked authoredArchitecture.functionalUnits[1]",
	"code.unit_unlinked authoredArchitecture.functionalUnits[2]",
}

// The check reports, once each, the unit, machine, state, transition, event,
// interface, flow and data object that no scanned non-test source names, and
// leaves out the named, the imported, the externally owned and the kinds that
// are never asked (actors, risks, trust boundaries, an event no transition of
// a local machine uses, a flow and data that touch no local interface).
// Mutation: dropping a builder from coverageBuilders, the local-owner filter
// or the alias:: filter changes the reported set.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestCoverageReportsUnnamedElements(t *testing.T) {
	cases := []struct {
		name  string
		items []codescan.CodeItem
		want  []string
	}{
		{"only a unit named", []codescan.CodeItem{ownerFile("a.go", "FU-A")}, allUnlinked},
		{"a test file names nothing", []codescan.CodeItem{ownerFile("a.go", "FU-A"), ownerFile("b_test.go", "FU-B"), {Kind: "symbol", Source: "b_test.go:3", ModelLinks: []string{"IF-A"}}}, allUnlinked},
		{"no scanned source", nil, []string{}},
		{"declarations without a source file", []codescan.CodeItem{linkSymbol("FU-A")}, []string{}},
		{"every element named", []codescan.CodeItem{
			ownerFile("a.go", "FU-A"), ownerFile("b.go", "FU-B"), ownerFile("c.go", "FU-C"),
			linkSymbol("SM-X", "IF-A", "FLOW-A", "DAT-1", "STATE-1", "STATE-2", "TR-1", "TR-2", "EVT-1", "EVT-2"),
		}, []string{}},
		{"a state link names its machine", []codescan.CodeItem{ownerFile("a.go", "FU-B"), ownerFile("c.go", "FU-C"), linkSymbol("STATE-1")}, []string{
			"code.data_unlinked authoredArchitecture.dataObjects[0]",
			"code.event_unlinked authoredArchitecture.events[0]",
			"code.event_unlinked authoredArchitecture.events[1]",
			"code.flow_unlinked authoredArchitecture.flows[0]",
			"code.interface_unlinked authoredArchitecture.interfaces[0]",
			"code.machine_unlinked authoredArchitecture.machines[1]",
			"code.state_unlinked authoredArchitecture.machines[0].states[1]",
			"code.transition_unlinked authoredArchitecture.machines[0].transitions[0]",
			"code.transition_unlinked authoredArchitecture.machines[0].transitions[1]",
			"code.unit_unlinked authoredArchitecture.functionalUnits[0]",
		}},
		{"an event named by a link", []codescan.CodeItem{ownerFile("a.go", "FU-A"), ownerFile("b.go", "FU-B"), ownerFile("c.go", "FU-C"), linkSymbol("EVT-1", "EVT-2", "SM-X")}, []string{
			"code.data_unlinked authoredArchitecture.dataObjects[0]",
			"code.flow_unlinked authoredArchitecture.flows[0]",
			"code.interface_unlinked authoredArchitecture.interfaces[0]",
			"code.state_unlinked authoredArchitecture.machines[0].states[0]",
			"code.state_unlinked authoredArchitecture.machines[0].states[1]",
			"code.transition_unlinked authoredArchitecture.machines[0].transitions[0]",
			"code.transition_unlinked authoredArchitecture.machines[0].transitions[1]",
		}},
	}
	for _, c := range cases {
		diags := coverageDiagnostics(coverageBundle(), c.items)
		if got := reported(diags); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: reported\n%v\nwant\n%v", c.name, got, c.want)
		}
		requireWarningsWithRemedy(t, c.name, diags)
	}
}

// requireWarningsWithRemedy fails for each diagnostic that is not a warning
// with a remedy and a message.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func requireWarningsWithRemedy(t *testing.T, name string, diags []validate.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.Severity != validate.SeverityWarning || d.Remedy == "" || d.Message == "" {
			t.Errorf("%s: want a warning with a remedy and a message: %+v", name, d)
		}
	}
}

// A machine counts as named when its owner unit is named, or its own ID, or a
// state or transition ID of it; the machine is reported only when none is.
// Mutation: dropping the owner, the state or the transition aliases of a
// machine makes the named machine reported.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestCoverageMachineNamedByAnyPart(t *testing.T) {
	machine := func(items ...codescan.CodeItem) bool {
		for _, d := range coverageDiagnostics(coverageBundle(), append([]codescan.CodeItem{ownerFile("z.go", "FU-C")}, items...)) {
			if d.Code == "code.machine_unlinked" && d.Path == "authoredArchitecture.machines[0]" {
				return true
			}
		}
		return false
	}
	cases := []struct {
		name     string
		items    []codescan.CodeItem
		reported bool
	}{
		{"nothing names it", nil, true},
		{"its owner unit", []codescan.CodeItem{ownerFile("a.go", "FU-A")}, false},
		{"its own ID", []codescan.CodeItem{linkSymbol("SM-A")}, false},
		{"one of its states", []codescan.CodeItem{linkSymbol("STATE-2")}, false},
		{"one of its transitions", []codescan.CodeItem{linkSymbol("TR-1")}, false},
		{"another machine's state", []codescan.CodeItem{linkSymbol("STATE-X")}, true},
	}
	for _, c := range cases {
		if got := machine(c.items...); got != c.reported {
			t.Errorf("%s: machine reported %v, want %v", c.name, got, c.reported)
		}
	}
}

// An exemption for an unlinked element with an accepted decision removes
// that element and is used; a machine exemption also covers its states and
// transitions; an exemption for a named element is unused.
// Mutation: ignoring the exemption list, the parent machine or the used
// bookkeeping changes the reported set or the unused entry.
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestCoverageExemptions(t *testing.T) {
	exempt := func(id string) model.CodeExemption {
		return model.CodeExemption{ID: id, Reason: goodReason, Decision: "DEC-OK"}
	}
	scan := []codescan.CodeItem{ownerFile("a.go", "FU-A")}
	cases := []struct {
		name   string
		bundle model.Bundle
		want   []string
	}{
		{"an exempt unit is not reported", withExemptions(coverageBundle(), exempt("FU-B")), without(allUnlinked, "code.unit_unlinked authoredArchitecture.functionalUnits[1]")},
		{"an exempt machine covers its parts", withExemptions(coverageBundle(), exempt("SM-A")), without(allUnlinked,
			"code.state_unlinked authoredArchitecture.machines[0].states[0]", "code.state_unlinked authoredArchitecture.machines[0].states[1]",
			"code.transition_unlinked authoredArchitecture.machines[0].transitions[0]", "code.transition_unlinked authoredArchitecture.machines[0].transitions[1]")},
		{"an exemption of a named unit is unused", withExemptions(coverageBundle(), exempt("FU-A")), append(append([]string{}, allUnlinked...), "code.exemption_unused codeCoverage.exempt[0]")},
		{"an exemption of a kind never asked is unused", withExemptions(coverageBundle(), exempt("ACT-1")), append(append([]string{}, allUnlinked...), "code.exemption_unused codeCoverage.exempt[0]")},
		{"an unaccepted decision exempts nothing", withExemptions(coverageBundle(), model.CodeExemption{ID: "FU-B", Reason: goodReason, Decision: "DEC-NO"}), append(append([]string{}, allUnlinked...), "code.exemption_unjustified codeCoverage.exempt[0].decision")},
		{"a missing decision exempts nothing", withExemptions(coverageBundle(), model.CodeExemption{ID: "FU-B", Reason: goodReason, Decision: "DEC-GONE"}), append(append([]string{}, allUnlinked...), "code.exemption_unjustified codeCoverage.exempt[0].decision")},
		{"an unknown element exempts nothing", withExemptions(coverageBundle(), exempt("FU-GHOST")), append(append([]string{}, allUnlinked...), "model.invalid_code_exemption codeCoverage.exempt[0].id")},
		{"a requirement is no element", withExemptions(coverageBundle(), exempt("REQ-A")), append(append([]string{}, allUnlinked...), "model.invalid_code_exemption codeCoverage.exempt[0].id")},
		{"a short reason exempts nothing", withExemptions(coverageBundle(), model.CodeExemption{ID: "FU-B", Reason: "  too short   ", Decision: "DEC-OK"}), append(append([]string{}, allUnlinked...), "model.invalid_code_exemption codeCoverage.exempt[0].reason")},
	}
	for _, c := range cases {
		c.bundle.Requirements = model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-A"}}}
		got := reported(coverageDiagnostics(c.bundle, scan))
		want := append([]string{}, c.want...)
		sort.Strings(want)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: reported\n%v\nwant\n%v", c.name, got, want)
		}
	}
}

// without returns the list minus the given entries.
func without(list []string, drop ...string) []string {
	skip := map[string]bool{}
	for _, d := range drop {
		skip[d] = true
	}
	out := []string{}
	for _, s := range list {
		if !skip[s] {
			out = append(out, s)
		}
	}
	return out
}

// The exemption entries are validated without scanned source, but an unused
// entry and an unlinked element are not reported there.
// Mutation: moving the scanned test before the exemption validation, or
// reporting an unused entry without a scan, changes the result.
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestCoverageExemptionsWithoutScan(t *testing.T) {
	bundle := withExemptions(coverageBundle(),
		model.CodeExemption{ID: "FU-B", Reason: goodReason, Decision: "DEC-NO"},
		model.CodeExemption{ID: "FU-C", Reason: goodReason, Decision: "DEC-OK"})
	got := reported(coverageDiagnostics(bundle, nil))
	want := []string{"code.exemption_unjustified codeCoverage.exempt[0].decision"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reported %v, want %v", got, want)
	}
}

// code.exemption_unjustified and model.invalid_code_exemption are errors with
// a remedy at the point they are emitted; the unlinked codes and the unused
// entry are warnings.
// Mutation: emitting the config problems as warnings fails the severity.
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestCoverageExemptionProblemSeverity(t *testing.T) {
	bundle := withExemptions(coverageBundle(), model.CodeExemption{ID: "FU-GHOST", Reason: "short", Decision: "DEC-NO"})
	diags := coverageDiagnostics(bundle, nil)
	if len(diags) != 3 {
		t.Fatalf("want three problems for one bad entry, got %+v", diags)
	}
	for _, d := range diags {
		if d.Severity != validate.SeverityError || d.Remedy == "" {
			t.Errorf("want an error with a remedy: %+v", d)
		}
	}
}

// ValidateIntegrity carries the coverage check: a unit no source names is
// reported through the integrity tail.
// Mutation: removing the coverageDiagnostics call from integrityTail drops
// the code.
// TRLC-LINKS: REQ-EMG-175, REQ-EMG-176
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestValidateIntegrityReportsCoverage(t *testing.T) {
	bundle := withExemptions(coverageBundle(), model.CodeExemption{ID: "FU-B", Reason: goodReason, Decision: "DEC-OK"})
	diags := ValidateIntegrity(bundle, model.RequirementsDocument{}, []codescan.CodeItem{ownerFile("a.go", "FU-A")}, nil, nil)
	got := pathsOf(diags, "code.unit_unlinked")
	if want := []string{"authoredArchitecture.functionalUnits[2]"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("code.unit_unlinked at %v, want %v", got, want)
	}
	if fmt.Sprint(pathsOf(diags, "code.exemption_unused")) != "[]" {
		t.Fatalf("the used exemption was reported unused: %v", pathsOf(diags, "code.exemption_unused"))
	}
}
