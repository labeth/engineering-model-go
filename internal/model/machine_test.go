// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const machineBehaviorYAML = `schemaVersion: 3
behavior:
  events:
    - id: EVT-A
      name: Authorization requested
  machines:
    - id: SM-PAY
      name: Payment authorization
      description: Authorizes a payment.
      owner: FU-A
      initial: STATE-PAY-IDLE
      variables:
        - {name: attempts, type: Integer, initial: 0}
        - {name: approved, type: Boolean, initial: false}
      states:
        - {id: STATE-PAY-IDLE, name: Idle}
        - {id: STATE-PAY-AUTH, name: Authorizing}
        - {id: STATE-PAY-DONE, name: Done, final: true}
      transitions:
        - {id: TR-PAY-001, from: STATE-PAY-IDLE, to: STATE-PAY-AUTH, on: EVT-A, guard: "attempts < 3", effect: "attempts := attempts + 1", description: Start}
        - {id: TR-PAY-002, from: STATE-PAY-AUTH, to: STATE-PAY-DONE, guard: approved}
        - {id: TR-PAY-003, from: STATE-PAY-AUTH, to: STATE-PAY-IDLE}
      invariants:
        - {id: INV-PAY-001, requirements: [REQ-A], expr: "attempts <= 3", description: Bounded}
`

func loadMachineFixture(t *testing.T, behavior string) (Bundle, error) {
	t.Helper()
	dir := writeV2Fixture(t, validManifest)
	writeTestFile(t, filepath.Join(dir, "model", "behavior.yml"), behavior)
	return LoadBundle(filepath.Join(dir, "engmod.yml"))
}

// TRLC-LINKS: REQ-EMG-135
func TestLoadBundleCarriesMachines(t *testing.T) {
	bundle, err := loadMachineFixture(t, machineBehaviorYAML)
	if err != nil {
		t.Fatal(err)
	}
	machines := bundle.Architecture.AuthoredArchitecture.Machines
	if len(machines) != 1 || len(bundle.Behavior.Behavior.Machines) != 1 {
		t.Fatalf("machines were not carried into the architecture: %+v", machines)
	}
	m := machines[0]
	if m.ID != "SM-PAY" || m.Owner != "FU-A" || m.Initial != "STATE-PAY-IDLE" ||
		len(m.Variables) != 2 || len(m.States) != 3 || len(m.Transitions) != 3 || len(m.Invariants) != 1 {
		t.Fatalf("unexpected machine: %+v", m)
	}
	if got, ok := m.Variables[0].Initial.(int); !ok || got != 0 || m.Variables[0].Type != "Integer" {
		t.Fatalf("integer variable decoded wrongly: %#v", m.Variables[0])
	}
	if got, ok := m.Variables[1].Initial.(bool); !ok || got {
		t.Fatalf("boolean variable decoded wrongly: %#v", m.Variables[1])
	}
	if !m.States[2].Final || m.Transitions[0].On != "EVT-A" || m.Transitions[0].Guard != "attempts < 3" {
		t.Fatalf("state or transition decoded wrongly: %+v", m)
	}
	if m.Invariants[0].Requirements[0] != "REQ-A" || m.Invariants[0].Expr != "attempts <= 3" {
		t.Fatalf("invariant decoded wrongly: %+v", m.Invariants[0])
	}
	if _, err := NewCanonicalBundle(bundle); err != nil {
		t.Fatalf("a valid machine must project and validate: %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-135
func TestLoadBundleRejectsInvalidMachineYAML(t *testing.T) {
	cases := map[string]struct{ old, new, want string }{
		"unknown machine field":    {"      owner: FU-A\n", "      owner: FU-A\n      invented: 1\n", "invented"},
		"unknown state field":      {"{id: STATE-PAY-IDLE, name: Idle}", "{id: STATE-PAY-IDLE, name: Idle, mode: x}", "mode"},
		"unknown transition":       {"to: STATE-PAY-AUTH, on:", "to: STATE-PAY-AUTH, label: x, on:", "label"},
		"unknown invariant field":  {"expr: \"attempts <= 3\"", "expr: \"attempts <= 3\", severity: high", "severity"},
		"machine id prefix":        {"id: SM-PAY", "id: PAY", "SM-"},
		"variable type":            {"type: Boolean", "type: Real", "type"},
		"variable initial type":    {"initial: false", "initial: [1]", "initial"},
		"variable without initial": {"{name: approved, type: Boolean, initial: false}", "{name: approved, type: Boolean}", "initial"},
		"variable name":            {"name: attempts", "name: 1attempts", "name"},
		"empty guard":              {`guard: "attempts < 3"`, `guard: ""`, "guard"},
		"empty invariant expr":     {`expr: "attempts <= 3"`, `expr: ""`, "expr"},
		"machines not a list":      {"  machines:\n", "  machines: oops\n  extra:\n", "machines"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(machineBehaviorYAML, c.old) {
				t.Fatalf("fixture lacks %q", c.old)
			}
			_, err := loadMachineFixture(t, strings.Replace(machineBehaviorYAML, c.old, c.new, 1))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected rejection mentioning %q, got %v", c.want, err)
			}
		})
	}
}

func machineTestBundle(machines ...Machine) Bundle {
	return Bundle{
		Architecture: ArchitectureDocument{
			Model: ModelMeta{ID: "MODEL-A"},
			AuthoredArchitecture: AuthoredArchitecture{
				FunctionalUnits: []FunctionalUnit{{ID: "FU-A", Name: "A"}},
				Events:          []Event{{ID: "EVT-A", Name: "Requested"}},
				Machines:        machines,
			},
		},
		Requirements: RequirementsDocument{Requirements: []Requirement{{ID: "REQ-A"}}},
	}
}

func validTestMachine() Machine {
	return Machine{
		ID: "SM-PAY", Name: "Payment authorization", Description: "Authorizes a payment.",
		Owner: "FU-A", Initial: "STATE-PAY-IDLE",
		Variables: []MachineVariable{
			{Name: "attempts", Type: "Integer", Initial: 0},
			{Name: "approved", Type: "Boolean", Initial: false},
			{Name: "reason", Type: "String", Initial: "none"},
		},
		States: []MachineState{
			{ID: "STATE-PAY-IDLE", Name: "Idle"},
			{ID: "STATE-PAY-AUTH", Name: "Authorizing"},
			{ID: "STATE-PAY-DONE", Name: "Done", Final: true},
		},
		Transitions: []MachineTransition{
			{ID: "TR-PAY-001", From: "STATE-PAY-IDLE", To: "STATE-PAY-AUTH", On: "EVT-A", Guard: "attempts < 3", Effect: "attempts := attempts + 1", Description: "Start"},
			{ID: "TR-PAY-002", From: "STATE-PAY-AUTH", To: "STATE-PAY-DONE", Guard: "approved"},
		},
		Invariants: []MachineInvariant{
			{ID: "INV-PAY-001", Requirements: []string{"REQ-A"}, Expr: "attempts <= 3", Description: "Bounded"},
		},
	}
}

// machineGolden keeps only what the machine projected, so the golden file is
// independent of unrelated model content.
type machineGolden struct {
	Elements      []SemanticElement      `json:"elements"`
	Relationships []SemanticRelationship `json:"relationships"`
}

// TRLC-LINKS: REQ-EMG-137
func TestProjectSemanticModelMachineGolden(t *testing.T) {
	semantic, diagnostics := ProjectSemanticModel(machineTestBundle(validTestMachine()))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected projection diagnostics: %+v", diagnostics)
	}
	if more := ValidateSemanticModel(semantic); len(more) != 0 {
		t.Fatalf("projected machine must validate: %+v", more)
	}
	golden := machineGolden{}
	for _, element := range semantic.Elements {
		if strings.HasPrefix(element.ID, "SM-") || strings.HasPrefix(element.ID, "STATE-PAY-") || strings.HasPrefix(element.ID, "INV-") {
			golden.Elements = append(golden.Elements, element)
		}
	}
	for _, relationship := range semantic.Relationships {
		if relationship.Owner == "SM-PAY" {
			golden.Relationships = append(golden.Relationships, relationship)
		}
	}
	got, err := json.MarshalIndent(golden, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", "machine-projection.golden.json")
	if os.Getenv("ENGMOD_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("machine projection differs from %s (ENGMOD_UPDATE_GOLDEN=1 rewrites it)\n got: %s", path, got)
	}
}

// TRLC-LINKS: REQ-EMG-137
func TestProjectSemanticModelMachineShape(t *testing.T) {
	semantic, _ := ProjectSemanticModel(machineTestBundle(validTestMachine()))
	elements := map[string]SemanticElement{}
	for _, element := range semantic.Elements {
		elements[element.ID] = element
	}
	machine := elements["SM-PAY"]
	if machine.Kind != ElementStateDefinition || machine.Extension != MachineExtension || machine.Owner != "FU-A" || len(machine.Features) != 3 {
		t.Fatalf("unexpected machine element: %+v", machine)
	}
	if f := machine.Features[0]; f.Name != "attempts" || f.Type != "Integer" || f.Value == nil || *f.Value.TypedValue.Integer != 0 {
		t.Fatalf("unexpected variable feature: %+v", f)
	}
	if f := machine.Features[2]; f.Type != "String" || f.Value.TypedValue.String != "none" {
		t.Fatalf("unexpected string variable feature: %+v", f)
	}
	state := elements["STATE-PAY-DONE"]
	if state.Kind != ElementStateUsage || state.Extension != MachineStateExtension || state.Owner != "SM-PAY" {
		t.Fatalf("unexpected state element: %+v", state)
	}
	final := state.Metadata[0].Properties["final"]
	if final.Kind != "boolean" || final.Boolean == nil || !*final.Boolean {
		t.Fatalf("final state must be readable from metadata: %+v", state.Metadata)
	}
	if idle := elements["STATE-PAY-IDLE"].Metadata[0].Properties["final"]; idle.Boolean == nil || *idle.Boolean {
		t.Fatalf("non-final state must carry final=false: %+v", idle)
	}
	invariant := elements["INV-PAY-001"]
	if invariant.Kind != ElementConstraintUsage || invariant.Owner != "SM-PAY" ||
		len(invariant.References) != 1 || invariant.References[0] != "REQ-A" ||
		len(invariant.Features) != 1 || invariant.Features[0].Name != "result" ||
		invariant.Features[0].Type != "Boolean" || invariant.Features[0].Value.Operator != "<=" {
		t.Fatalf("unexpected invariant element: %+v", invariant)
	}
	if _, ok := invariant.Properties["result"]; ok {
		t.Fatalf("invariants must not use properties (result is derived in SysML): %+v", invariant.Properties)
	}
	var transitions, successions []SemanticRelationship
	for _, relationship := range semantic.Relationships {
		switch relationship.Kind {
		case RelationshipTransition:
			transitions = append(transitions, relationship)
		case RelationshipSuccession:
			successions = append(successions, relationship)
		}
	}
	if len(transitions) != 2 || len(successions) != 2 {
		t.Fatalf("expected 2 transitions and 2 successions, got %+v / %+v", transitions, successions)
	}
	first := transitions[0]
	if first.ID != "TR-PAY-001" || first.Source != "STATE-PAY-IDLE" || first.Target != "STATE-PAY-AUTH" ||
		first.Owner != "SM-PAY" || len(first.Triggers) != 1 || first.Triggers[0] != "EVT-A" ||
		first.Guard == nil || first.Effect == nil || first.Effect.Operator != "all" {
		t.Fatalf("unexpected transition: %+v", first)
	}
	if second := transitions[1]; second.Triggers != nil || second.Effect != nil {
		t.Fatalf("optional parts must stay absent: %+v", second)
	}
}

// TRLC-LINKS: REQ-EMG-135, REQ-EMG-137
func TestProjectSemanticModelMachineStructuralChecks(t *testing.T) {
	type mutation func(*Machine)
	cases := []struct {
		name string
		edit mutation
		code string
		path string
	}{
		{"missing owner", func(m *Machine) { m.Owner = "" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].owner"},
		{"missing initial", func(m *Machine) { m.Initial = "" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].initial"},
		{"initial not a state", func(m *Machine) { m.Initial = "STATE-NOPE" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].initial"},
		{"initial is an element outside the machine", func(m *Machine) { m.Initial = "FU-A" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].initial"},
		{"transition from unknown", func(m *Machine) { m.Transitions[0].From = "STATE-NOPE" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].transitions[0].from"},
		{"transition to unknown", func(m *Machine) { m.Transitions[1].To = "STATE-NOPE" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].transitions[1].to"},
		{"transition without from", func(m *Machine) { m.Transitions[0].From = "" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].transitions[0].from"},
		{"transition without to", func(m *Machine) { m.Transitions[0].To = "" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].transitions[0].to"},
		{"transition without id", func(m *Machine) { m.Transitions[0].ID = "" }, "semantic.missing_id", "authoredArchitecture.machines[0].transitions[0]"},
		{"duplicate transition id", func(m *Machine) { m.Transitions[1].ID = "TR-PAY-001" }, "semantic.duplicate_identity", "authoredArchitecture.machines[0].transitions[1]"},
		{"transition id clashes with state", func(m *Machine) { m.Transitions[1].ID = "STATE-PAY-IDLE" }, "semantic.duplicate_identity", "authoredArchitecture.machines[0].transitions[1]"},
		{"duplicate state id", func(m *Machine) { m.States[1].ID = "STATE-PAY-IDLE" }, "semantic.duplicate_identity", "authoredArchitecture.machines[0].states[1]"},
		{"state clashes with another element", func(m *Machine) { m.States[1].ID = "FU-A" }, "semantic.duplicate_identity", "authoredArchitecture.machines[0].states[1]"},
		{"state without id", func(m *Machine) { m.States[1].ID = "" }, "semantic.missing_id", "authoredArchitecture.machines[0].states[1]"},
		{"duplicate invariant id", func(m *Machine) {
			m.Invariants = append(m.Invariants, MachineInvariant{ID: "INV-PAY-001", Expr: "approved"})
		}, "semantic.duplicate_identity", "authoredArchitecture.machines[0].invariants[1]"},
		{"invariant without id", func(m *Machine) { m.Invariants[0].ID = "" }, "semantic.missing_id", "authoredArchitecture.machines[0].invariants[0]"},
		{"invariant without expression", func(m *Machine) { m.Invariants[0].Expr = " " }, "semantic.invalid_machine", "authoredArchitecture.machines[0].invariants[0].expr"},
		{"invariant requirement missing", func(m *Machine) { m.Invariants[0].Requirements = []string{"REQ-NOPE"} }, "semantic.dangling_reference", "semantic.elements"},
		{"trigger event missing", func(m *Machine) { m.Transitions[0].On = "EVT-NOPE" }, "semantic.dangling_relationship", "semantic.relationships"},
		{"owner missing from model", func(m *Machine) { m.Owner = "FU-NOPE" }, "semantic.invalid_ownership", "semantic.elements"},
		{"bad guard", func(m *Machine) { m.Transitions[0].Guard = "attempts <" }, "semantic.invalid_expression", "authoredArchitecture.machines[0].transitions[0].guard"},
		{"bad effect", func(m *Machine) { m.Transitions[0].Effect = "attempts + 1" }, "semantic.invalid_expression", "authoredArchitecture.machines[0].transitions[0].effect"},
		{"bad invariant expression", func(m *Machine) { m.Invariants[0].Expr = "attempts := 1" }, "semantic.invalid_expression", "authoredArchitecture.machines[0].invariants[0].expr"},
		{"variable type", func(m *Machine) { m.Variables[0].Type = "Real" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].variables[0]"},
		{"variable initial mismatch", func(m *Machine) { m.Variables[0].Initial = "zero" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].variables[0]"},
		{"variable initial missing", func(m *Machine) { m.Variables[1].Initial = nil }, "semantic.invalid_machine", "authoredArchitecture.machines[0].variables[1]"},
		{"variable empty string initial", func(m *Machine) { m.Variables[2].Initial = "" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].variables[2]"},
		{"variable without name", func(m *Machine) { m.Variables[0].Name = "" }, "semantic.invalid_machine", "authoredArchitecture.machines[0].variables[0].name"},
		{"duplicate variable", func(m *Machine) { m.Variables[1].Name = "attempts" }, "semantic.duplicate_identity", "authoredArchitecture.machines[0].variables[1].name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			machine := validTestMachine()
			c.edit(&machine)
			semantic, diagnostics := ProjectSemanticModel(machineTestBundle(machine))
			diagnostics = append(diagnostics, ValidateSemanticModel(semantic)...)
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == c.code && strings.HasPrefix(diagnostic.Path, c.path) && diagnostic.Severity == SemanticSeverityError {
					return
				}
			}
			t.Fatalf("expected %s at %s, got %+v", c.code, c.path, diagnostics)
		})
	}
}

// TRLC-LINKS: REQ-EMG-135
func TestProjectSemanticModelMachineMissingAndDuplicateMachineID(t *testing.T) {
	anonymous := validTestMachine()
	anonymous.ID = ""
	_, diagnostics := ProjectSemanticModel(machineTestBundle(anonymous))
	if !hasSemanticDiagnostic(diagnostics, "semantic.missing_id") {
		t.Fatalf("expected missing machine ID, got %+v", diagnostics)
	}
	twin := validTestMachine()
	_, diagnostics = ProjectSemanticModel(machineTestBundle(validTestMachine(), twin))
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "semantic.duplicate_identity" && diagnostic.Path == "authoredArchitecture.machines[1]" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate machine ID at machines[1], got %+v", diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-137
func TestProjectSemanticModelWithoutMachinesAddsNothing(t *testing.T) {
	with := machineTestBundle(validTestMachine())
	without := machineTestBundle()
	a, _ := ProjectSemanticModel(without)
	b, _ := ProjectSemanticModel(with)
	for _, element := range a.Elements {
		if element.Extension == MachineExtension || element.Extension == MachineStateExtension || element.Extension == MachineInvariantExtension {
			t.Fatalf("machine element without machines: %+v", element)
		}
	}
	if len(b.Elements) <= len(a.Elements) || len(b.Relationships) <= len(a.Relationships) {
		t.Fatalf("machines must add elements and relationships")
	}
}
