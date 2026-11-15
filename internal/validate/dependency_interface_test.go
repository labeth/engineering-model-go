// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

const (
	codeNoInterface = "architecture.dependency_without_interface"
	codeNoData      = "architecture.interface_without_data"
)

// interfaceBundle is a valid model of two groups, three units, an actor, a
// referenced element, three hardware items (HW-1 and HW-2 joined by a hardware
// interface), a data object and the interfaces IF-B (owned by FU-B), IF-C
// (contained by FU-C, no owner field), IF-ORPHAN (no owner) and IF-REF (owned
// by REF-X), with the given relationships and flows.
func interfaceBundle(mappings []model.Mapping, flows []model.Flow) model.Bundle {
	b := behaviorTestBundle(mappings, flows)
	a := &b.Architecture.AuthoredArchitecture
	a.FunctionalGroups = []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}, {ID: "FG-B", Name: "Group B"}}
	a.FunctionalUnits = []model.FunctionalUnit{
		{ID: "FU-A", Group: "FG-A", Name: "A"}, {ID: "FU-B", Group: "FG-B", Name: "B"}, {ID: "FU-C", Group: "FG-A", Name: "C"},
	}
	a.Actors = []model.Actor{{ID: "ACT-X", Name: "Person"}}
	a.ReferencedElements = []model.ReferencedElement{{ID: "REF-X", Name: "External tool"}}
	a.Interfaces = []model.Interface{
		{ID: "IF-B", Name: "B api", Owner: "FU-B"}, {ID: "IF-C", Name: "C api"},
		{ID: "IF-ORPHAN", Name: "No owner"}, {ID: "IF-REF", Name: "External api", Owner: "REF-X"},
	}
	a.DataObjects = []model.DataObject{{ID: "DO-D", Name: "Data"}}
	a.HardwareItems = []model.HardwareItem{{ID: "HW-1", Name: "One"}, {ID: "HW-2", Name: "Two"}, {ID: "HW-3", Name: "Three"}}
	a.HardwareInterfaces = []model.HardwareInterface{{ID: "HWI-1", Name: "Bus", From: "HW-1", To: "HW-2"}}
	return b
}

func rel(typ, from, to string) model.Mapping { return model.Mapping{Type: typ, From: from, To: to} }

// gapPaths returns the paths of the diagnostics of a code, in order.
func gapPaths(b model.Bundle, code string) []string {
	return diagPaths(behaviorDiags(b, code))
}

// An architectural dependency has an interface when one of its ends is an
// interface (a local one needs an owner); every other dependency between two
// architectural elements is reported once, at its relationship. Containment,
// self dependencies, relationship types that are no dependency and ends that
// are data objects or deployment targets are not asked.
// Mutation: dropping a type from dependencyTypes or an end kind from
// dependencyEndKinds, accepting an ownerless interface, or reporting a self
// dependency changes the row of that case.
// TRLC-LINKS: REQ-EMG-207
func TestDependencyNeedsAnInterface(t *testing.T) {
	cases := map[string]struct {
		mapping model.Mapping
		extra   func(*model.AuthoredArchitecture)
		want    bool
	}{
		"unit to unit across groups":            {mapping: rel("calls", "FU-A", "FU-B"), want: true},
		"unit to unit inside one group":         {mapping: rel("depends_on", "FU-A", "FU-C"), want: true},
		"unit to interface of another unit":     {mapping: rel("calls", "FU-A", "IF-B")},
		"unit to interface the unit contains":   {mapping: rel("depends_on", "FU-A", "IF-C"), extra: func(a *model.AuthoredArchitecture) { a.Mappings = append(a.Mappings, rel("contains", "FU-C", "IF-C")) }},
		"unit to interface without an owner":    {mapping: rel("depends_on", "FU-A", "IF-ORPHAN"), want: true},
		"unit to referenced element":            {mapping: rel("depends_on", "FU-A", "REF-X"), want: true},
		"unit to interface of the referenced":   {mapping: rel("depends_on", "FU-A", "IF-REF")},
		"actor to unit":                         {mapping: rel("interacts_with", "ACT-X", "FU-A"), want: true},
		"actor to interface":                    {mapping: rel("interacts_with", "ACT-X", "IF-B")},
		"group to group":                        {mapping: rel("depends_on", "FG-A", "FG-B"), want: true},
		"unit implements an interface":          {mapping: rel("implements", "FU-A", "IF-B")},
		"publishes to an interface":             {mapping: rel("publishes", "FU-A", "IF-B")},
		"hardware items joined by an interface": {mapping: rel("interacts_with", "HW-2", "HW-1")},
		"hardware items joined by none":         {mapping: rel("interacts_with", "HW-1", "HW-3"), want: true},
		"composed unit":                         {mapping: rel("calls", "FU-A", "dep::FU-REMOTE"), extra: composedUnit, want: true},
		"composed published interface":          {mapping: rel("calls", "FU-A", "dep::IF-REMOTE"), extra: composedInterface},
		"composed id the bundle does not know":  {mapping: rel("calls", "FU-A", "dep::FU-REMOTE")},
		"containment":                           {mapping: rel("contains", "FG-A", "FU-A")},
		"self dependency":                       {mapping: rel("depends_on", "FU-A", "FU-A")},
		"reads a data object":                   {mapping: rel("reads", "FU-A", "DO-D")},
		"writes a data object":                  {mapping: rel("writes", "FU-A", "DO-D")},
		"deployment":                            {mapping: rel("deployed_to", "FU-A", "DEP-X"), extra: deployment},
		"allocation is no dependency":           {mapping: rel("allocated_to", "FU-A", "HW-1")},
		"depends on a deployment target":        {mapping: rel("depends_on", "FU-A", "DEP-X"), extra: deployment},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			b := interfaceBundle([]model.Mapping{c.mapping}, nil)
			if c.extra != nil {
				c.extra(&b.Architecture.AuthoredArchitecture)
			}
			got := behaviorDiags(b, codeNoInterface)
			if c.want != (len(got) == 1) || len(got) > 1 {
				t.Fatalf("got %d diagnostics %v, want reported=%v", len(got), got, c.want)
			}
			if c.want {
				requireDependencyMessage(t, got[0], c.mapping)
			}
		})
	}
}

func composedUnit(a *model.AuthoredArchitecture) {
	a.FunctionalUnits = append(a.FunctionalUnits, model.FunctionalUnit{ID: "dep::FU-REMOTE", Group: "FG-A", Name: "Remote"})
}

func composedInterface(a *model.AuthoredArchitecture) {
	a.Interfaces = append(a.Interfaces, model.Interface{ID: "dep::IF-REMOTE", Name: "Remote api"})
}

func deployment(a *model.AuthoredArchitecture) {
	a.DeploymentTargets = append(a.DeploymentTargets, model.DeploymentTarget{ID: "DEP-X", Name: "Host"})
}

// A reported dependency names the type, both ends, the relationship path and
// the fix, and carries the remedy.
// TRLC-LINKS: REQ-EMG-207
func requireDependencyMessage(t *testing.T, d Diagnostic, m model.Mapping) {
	const path = "authoredArchitecture.mappings[0]"
	t.Helper()
	for _, want := range []string{`"` + m.Type + `"`, m.From, m.To, path, "name the interface the dependency goes through"} {
		if !strings.Contains(d.Message, want) {
			t.Errorf("message %q lacks %q", d.Message, want)
		}
	}
	if d.Path != path || d.Severity != SeverityWarning || d.Remedy != remedyDependencyInterface {
		t.Errorf("got path %q severity %q remedy %q", d.Path, d.Severity, d.Remedy)
	}
}

// An interface without an owner says so instead of "names no interface".
// TRLC-LINKS: REQ-EMG-207
func TestDependencyOnOwnerlessInterfaceSaysSo(t *testing.T) {
	got := behaviorDiags(interfaceBundle([]model.Mapping{rel("calls", "FU-A", "IF-ORPHAN")}, nil), codeNoInterface)
	if len(got) != 1 || !strings.Contains(got[0].Message, "interface IF-ORPHAN, which has no owner") {
		t.Fatalf("got %v", got)
	}
}

// Several dependencies are reported in model order, each at its own path, and
// the same model gives the same diagnostics on every run.
// Mutation: iterating a map instead of the relationship list reorders the paths.
// TRLC-LINKS: REQ-EMG-207
func TestDependencyDiagnosticsAreDeterministic(t *testing.T) {
	mappings := []model.Mapping{
		rel("calls", "FU-A", "FU-B"), rel("calls", "FU-A", "IF-B"), rel("depends_on", "FU-C", "FU-B"), rel("interacts_with", "ACT-X", "FU-A"),
	}
	want := []string{"authoredArchitecture.mappings[0]", "authoredArchitecture.mappings[2]", "authoredArchitecture.mappings[3]"}
	first := fmt.Sprint(behaviorDiags(interfaceBundle(mappings, nil), codeNoInterface))
	for i := 0; i < 5; i++ {
		b := interfaceBundle(mappings, nil)
		if got := gapPaths(b, codeNoInterface); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("paths %v, want %v", got, want)
		}
		if got := fmt.Sprint(behaviorDiags(b, codeNoInterface)); got != first {
			t.Fatalf("run %d differs:\n%s\n%s", i, got, first)
		}
	}
}

// An interaction from an actor to an interface is a valid mapping pair and a
// referenced element may own an interface; an interface owned by anything else
// is still model.invalid_interface_owner.
// Mutation: dropping actor:interface from the pairs, or the references check of
// the owner, adds an error.
// TRLC-LINKS: REQ-EMG-207
func TestInterfaceOwnerAndInteractionEnds(t *testing.T) {
	b := interfaceBundle([]model.Mapping{rel("interacts_with", "ACT-X", "IF-B"), rel("depends_on", "FU-A", "IF-REF")}, nil)
	for _, code := range []string{"model.invalid_interaction", "model.invalid_mapping_pair", "model.invalid_interface_owner"} {
		if got := behaviorDiags(b, code); len(got) != 0 {
			t.Errorf("%s: %v", code, got)
		}
	}
	b.Architecture.AuthoredArchitecture.Interfaces[0].Owner = "ACT-X"
	if got := behaviorDiags(b, "model.invalid_interface_owner"); len(got) != 1 {
		t.Errorf("an actor owns an interface: %v", got)
	}
	b = interfaceBundle([]model.Mapping{rel("interacts_with", "FU-A", "IF-B")}, nil)
	if got := behaviorDiags(b, "model.invalid_interaction"); len(got) != 1 {
		t.Errorf("a unit interacts with an interface: %v", got)
	}
}

// tierDiagnostics returns the diagnostics of a code that the tier policy gives
// the model of the bundle.
func tierDiagnostics(b model.Bundle, code string) []Diagnostic {
	var got []Diagnostic
	for _, d := range ApplyTierPolicy(b, Bundle(b)) {
		if d.Code == code {
			got = append(got, d)
		}
	}
	return got
}

// tierGrade is the severity a code must have at a tier ("" is no tier).
type tierGrade struct {
	tier     string
	severity Severity
}

// requireTierGrade checks one graded diagnostic: its severity, that it is not
// deferred and, with a tier, that it is required from product.
func requireTierGrade(t *testing.T, d Diagnostic, g tierGrade) {
	t.Helper()
	if d.Severity != g.severity || d.Deferred {
		t.Errorf("at %q: %+v, want severity %s, not deferred", g.tier, d, g.severity)
	}
	if g.tier != "" && d.RequiredFrom != "product" {
		t.Errorf("at %q: %+v, want required from product", g.tier, d)
	}
}

// Both architecture codes are graded by the project tier: a warning up to mvp
// and an error at product and operational, never deferred, the same without a
// tier decision as the other tier-keyed warnings.
// Mutation: changing the row (rowTopTwoTiers) of either code, or keying it
// as hygiene (DDWWE), fails the tier it moves.
// TRLC-LINKS: REQ-EMG-207, REQ-EMG-208
func TestArchitectureCodesAreGradedByTier(t *testing.T) {
	grades := []tierGrade{
		{"poc", SeverityWarning}, {"prototype", SeverityWarning}, {"mvp", SeverityWarning},
		{"product", SeverityError}, {"operational", SeverityError}, {"", SeverityWarning},
	}
	for _, code := range []string{codeNoInterface, codeNoData} {
		for _, g := range grades {
			t.Run(code+"/"+g.tier, func(t *testing.T) { requireGradedAt(t, code, g) })
		}
	}
}

// requireGradedAt reports the code in a model at the tier and checks every
// diagnostic of it.
func requireGradedAt(t *testing.T, code string, g tierGrade) {
	t.Helper()
	b := interfaceBundle([]model.Mapping{rel("calls", "FU-A", "FU-B")}, nil)
	if g.tier != "" {
		b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: g.tier, Bundles: []string{"foundation"}}
	}
	got := tierDiagnostics(b, code)
	if len(got) == 0 {
		t.Fatalf("%s not reported", code)
	}
	for _, d := range got {
		requireTierGrade(t, d, g)
	}
}

// lifecycle.raise cannot lower and a POA&M item defers neither code's error
// below its tier: the code is keyed like the other tier-graded diagnostics, so
// a raise is an error at every tier.
// Mutation: not keying a code in the tier table leaves a warning at product.
// TRLC-LINKS: REQ-EMG-207, REQ-EMG-208
func TestArchitectureCodesCanBeRaised(t *testing.T) {
	b := interfaceBundle([]model.Mapping{rel("calls", "FU-A", "FU-B")}, nil)
	b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: "poc", Bundles: []string{"foundation"}, Raise: []string{"architecture.*"}}
	for _, d := range ApplyTierPolicy(b, Bundle(b)) {
		if strings.HasPrefix(d.Code, "architecture.") && d.Severity != SeverityError {
			t.Errorf("%s not raised: %+v", d.Code, d)
		}
	}
}

func dataFlow(step model.FlowStep, dataRefs ...string) []model.Flow {
	step.ID, step.Ref = "step-1", "FU-A"
	return []model.Flow{{ID: "FLOW-A", Title: "A", DataRefs: dataRefs, Steps: []model.FlowStep{step}}}
}

// An interface carries data when a flow step names it in interfaceRef together
// with a data object in dataRefs, dataIn or dataOut. Nothing else counts: not
// the flow's own dataRefs, not a data object on a step with another or no
// interface, not an ID that is no data object, not a relationship that names
// the interface or a data object, not a schemaRef.
// Mutation: counting flow dataRefs, a step without interfaceRef, any ID, or
// dropping dataIn or dataOut, flips the row of that case.
// TRLC-LINKS: REQ-EMG-208
func TestInterfaceNeedsAData(t *testing.T) {
	const paths = "authoredArchitecture.interfaces[0]"
	cases := map[string]struct {
		flows    []model.Flow
		mappings []model.Mapping
		edit     func(*model.AuthoredArchitecture)
		want     bool // IF-B (interfaces[0]) is reported
	}{
		"no flow":                               {want: true},
		"step with dataRefs":                    {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-B", DataRefs: []string{"DO-D"}})},
		"step with dataIn":                      {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-B", DataIn: []string{"DO-D"}})},
		"step with dataOut":                     {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-B", DataOut: []string{"DO-D"}})},
		"data only on the flow":                 {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-B"}, "DO-D"), want: true},
		"step without interfaceRef":             {flows: dataFlow(model.FlowStep{DataRefs: []string{"DO-D"}}), want: true},
		"step on another interface":             {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-C", DataRefs: []string{"DO-D"}}), want: true},
		"interfaceRef without data":             {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-B"}), want: true},
		"data refs that are no data object":     {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-B", DataRefs: []string{"FU-A", "NOPE"}}), want: true},
		"blank data ref":                        {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-B", DataRefs: []string{" "}}), want: true},
		"relationship names interface and data": {mappings: []model.Mapping{rel("calls", "FU-A", "IF-B"), rel("publishes", "FU-B", "DO-D")}, want: true},
		"schemaRef only":                        {edit: func(a *model.AuthoredArchitecture) { a.Interfaces[0].SchemaRef = "schemas/b.json" }, want: true},
		"name that says data":                   {edit: func(a *model.AuthoredArchitecture) { a.Interfaces[0].Name = "DO-D data stream" }, want: true},
		"cli interface with input and output":   {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-B", DataIn: []string{"DO-D"}, DataOut: []string{"DO-D"}}), edit: func(a *model.AuthoredArchitecture) { a.Interfaces[0].Protocol = "cli" }},
		"unresolved composed data object":       {flows: dataFlow(model.FlowStep{InterfaceRef: "IF-B", DataRefs: []string{"dep::DO-REMOTE"}})},
		"padded ids":                            {flows: dataFlow(model.FlowStep{InterfaceRef: " IF-B ", DataRefs: []string{" DO-D "}})},
		"a later flow carries the data":         {flows: append(dataFlow(model.FlowStep{InterfaceRef: "IF-B"}), model.Flow{ID: "FLOW-B", Steps: []model.FlowStep{{ID: "s2", Ref: "FU-A", InterfaceRef: "IF-B", DataIn: []string{"DO-D"}}}})},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			b := interfaceBundle(c.mappings, c.flows)
			if c.edit != nil {
				c.edit(&b.Architecture.AuthoredArchitecture)
			}
			reported := false
			for _, p := range gapPaths(b, codeNoData) {
				reported = reported || p == paths
			}
			if reported != c.want {
				t.Fatalf("IF-B reported=%v, want %v; all: %v", reported, c.want, gapPaths(b, codeNoData))
			}
		})
	}
}

// Every interface is asked, used or not; a composed interface is checked in
// its own model; the message names the interface and the fix.
// Mutation: skipping an interface no flow or relationship uses, or asking an
// alias::IF-* interface, changes the paths.
// TRLC-LINKS: REQ-EMG-208
func TestInterfaceDataIsAskedOfLocalInterfaces(t *testing.T) {
	b := interfaceBundle(nil, dataFlow(model.FlowStep{InterfaceRef: "IF-C", DataRefs: []string{"DO-D"}}))
	composedInterface(&b.Architecture.AuthoredArchitecture)
	got := behaviorDiags(b, codeNoData)
	want := []string{"authoredArchitecture.interfaces[0]", "authoredArchitecture.interfaces[2]", "authoredArchitecture.interfaces[3]"}
	if strings.Join(diagPaths(got), ",") != strings.Join(want, ",") {
		t.Fatalf("paths %v, want %v", diagPaths(got), want)
	}
	for i, id := range []string{"IF-B", "IF-ORPHAN", "IF-REF"} {
		requireInterfaceMessage(t, got[i], id)
	}
}

// requireInterfaceMessage checks that a diagnostic names the interface and the
// fix and carries the remedy.
func requireInterfaceMessage(t *testing.T, d Diagnostic, id string) {
	t.Helper()
	if !strings.Contains(d.Message, "interface "+id+" ") || !strings.Contains(d.Message, "name the data type sent over the interface") {
		t.Errorf("message %q does not name %s and the fix", d.Message, id)
	}
	if d.Remedy != remedyInterfaceData || d.Severity != SeverityWarning {
		t.Errorf("%s: %+v", id, d)
	}
}
