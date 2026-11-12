// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package view

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-003
func TestBuild_AppliesViewFiltersAndDepth(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group"}},
			FunctionalUnits: []model.FunctionalUnit{
				{ID: "FU-A", Name: "A", Group: "FG-A"},
				{ID: "FU-B", Name: "B", Group: "FG-A"},
			},
			Interfaces: []model.Interface{{ID: "IF-A", Name: "Public API", Owner: "FU-A"}},
			Mappings: []model.Mapping{
				{Type: "calls", From: "FU-A", To: "IF-A"},
				{Type: "depends_on", From: "FU-A", To: "FU-B"},
			},
		},
		Views: []model.View{{
			ID:              "V-COMM",
			Kind:            "communication",
			Roots:           []string{"FU-A"},
			IncludeMappings: []string{"calls"},
			MaxDepth:        1,
		}},
	}}

	v, diags := Build(b, "V-COMM")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if len(v.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d (%+v)", len(v.Edges), v.Edges)
	}
	if v.Edges[0].Type != "calls" {
		t.Fatalf("expected calls edge, got %+v", v.Edges[0])
	}
	for _, n := range v.Nodes {
		if n.ID == "FU-B" {
			t.Fatalf("expected FU-B to be excluded by mapping filter/depth")
		}
	}
}

func stateLifecycleBundle(roots ...string) model.Bundle {
	return model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalUnits: []model.FunctionalUnit{{ID: "FU-A", Name: "Unit"}},
			Machines: []model.Machine{{
				ID: "SM-A", Name: "Update", Owner: "FU-A", Initial: "STATE-A",
				Variables: []model.MachineVariable{{Name: "tries", Type: "Integer", Initial: 0}},
				States:    []model.MachineState{{ID: "STATE-A", Name: "Ready"}, {ID: "STATE-B", Name: "Applied"}, {ID: "STATE-C", Name: "Failed"}},
				Transitions: []model.MachineTransition{
					{ID: "TR-A", From: "STATE-A", To: "STATE-B", On: "EVT-A", Guard: "tries < 3", Effect: "tries := tries + 1"},
					{ID: "TR-B", From: "STATE-A", To: "STATE-C", On: "EVT-A", Guard: "tries >= 3"},
				},
			}},
			Events:   []model.Event{{ID: "EVT-A", Name: "Requested"}},
			Controls: []model.Control{{ID: "CTRL-A", Name: "Gate"}},
			Mappings: []model.Mapping{{Type: "guarded_by", From: "STATE-B", To: "CTRL-A"}},
		},
		Views: []model.View{{ID: "V-STATE", Kind: "state-lifecycle", Roots: roots}},
	}}
}

// The state lifecycle view is driven by machines: every transition is a
// "transition" edge between its states with the transition ID, event, guard
// and effect in the label, the event a state waits for is an "on_event"
// edge (once per state and event), and guarded_by edges stay.
// Mutation: dropping the machine mappings from the view leaves no edges.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-139
func TestBuild_StateLifecycleDefaults(t *testing.T) {
	v, diags := Build(stateLifecycleBundle("STATE-A"), "V-STATE")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	type edge struct{ from, to, typ string }
	got := map[edge]string{}
	for _, e := range v.Edges {
		got[edge{e.From, e.To, e.Type}] = e.Label
	}
	want := map[edge]string{
		{"STATE-A", "STATE-B", "transition"}: "transition: TR-A on EVT-A [tries < 3] / tries := tries + 1",
		{"STATE-A", "STATE-C", "transition"}: "transition: TR-B on EVT-A [tries >= 3]",
		{"STATE-A", "EVT-A", "on_event"}:     "on_event",
		{"STATE-B", "CTRL-A", "guarded_by"}:  "guarded_by",
	}
	if len(v.Edges) != len(want) {
		t.Fatalf("want %d lifecycle edges, got %d: %+v", len(want), len(v.Edges), v.Edges)
	}
	for e, label := range want {
		if got[e] != label {
			t.Fatalf("edge %+v label = %q, want %q (all: %+v)", e, got[e], label, got)
		}
	}
	kinds := map[string]string{}
	for _, n := range v.Nodes {
		kinds[n.ID] = n.Kind
	}
	for id, kind := range map[string]string{"STATE-A": "state", "STATE-B": "state", "STATE-C": "state", "EVT-A": "event", "CTRL-A": "control"} {
		if kinds[id] != kind {
			t.Fatalf("node %s kind = %q, want %q (all: %+v)", id, kinds[id], kind, kinds)
		}
	}
	if _, ok := kinds["SM-A"]; ok {
		t.Fatalf("the machine itself is not a lifecycle node: %+v", v.Nodes)
	}
}

// A machine ID as view root selects all of its states, so a view needs no
// list of state IDs.
// Mutation: not expanding a machine root yields an empty view.
// TRLC-LINKS: REQ-EMG-139
func TestBuild_StateLifecycleMachineRootExpandsToStates(t *testing.T) {
	v, diags := Build(stateLifecycleBundle("SM-A"), "V-STATE")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	states := 0
	for _, n := range v.Nodes {
		if n.Kind == "state" {
			states++
		}
	}
	if states != 3 || len(v.Edges) != 4 {
		t.Fatalf("machine root must project 3 states and 4 edges, got %d states, edges %+v", states, v.Edges)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestBuild_InteractionFlowProjection(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group"}},
			FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Name: "Core", Group: "FG-A"}},
			Actors:           []model.Actor{{ID: "ACT-A", Name: "User"}},
			Interfaces:       []model.Interface{{ID: "IF-A", Name: "Control API", Owner: "FU-A"}},
			DataObjects:      []model.DataObject{{ID: "DO-A", Name: "Selection"}},
			Flows: []model.Flow{{
				ID:    "FLOW-A",
				Title: "Input flow",
				Entry: []string{"submit"},
				Steps: []model.FlowStep{
					{ID: "submit", Kind: "user_action", Ref: "ACT-A", Action: "Submit", Next: []string{"ingest"}},
					{ID: "ingest", Kind: "system_action", Ref: "IF-A", Action: "Ingest", DataOut: []string{"selection"}, Next: []string{"store"}, OnError: []string{"store"}, Async: true},
					{ID: "store", Kind: "data_move", Ref: "DO-A", Action: "Store"},
				},
			}},
		},
		Views: []model.View{{ID: "V-FLOW", Kind: "interaction-flow", Roots: []string{"FLOW-A"}}},
	}}

	v, diags := Build(b, "V-FLOW")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if len(v.Nodes) == 0 || len(v.Edges) == 0 {
		t.Fatalf("expected non-empty interaction-flow projection")
	}
	hasFlowStep := false
	hasAsync := false
	hasRef := false
	for _, n := range v.Nodes {
		if n.Kind == "flow_step" {
			hasFlowStep = true
		}
	}
	for _, e := range v.Edges {
		if e.Type == "flow_async" {
			hasAsync = true
		}
		if e.Type == "flow_ref" {
			hasRef = true
		}
	}
	if !hasFlowStep || !hasAsync || !hasRef {
		t.Fatalf("expected flow_step nodes and flow_async/flow_ref edges, got nodes=%+v edges=%+v", v.Nodes, v.Edges)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestBuild_InteractionFlowRespectsIncludeMappings(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group"}},
			FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Name: "Core", Group: "FG-A"}},
			Actors:           []model.Actor{{ID: "ACT-A", Name: "User"}},
			Flows: []model.Flow{{
				ID:    "FLOW-A",
				Title: "Input flow",
				Entry: []string{"submit"},
				Steps: []model.FlowStep{
					{ID: "submit", Kind: "user_action", Ref: "ACT-A", Action: "Submit", Next: []string{"process"}},
					{ID: "process", Kind: "system_action", Ref: "FU-A", Action: "Process", OnError: []string{"submit"}},
				},
			}},
		},
		Views: []model.View{{ID: "V-FLOW", Kind: "interaction-flow", Roots: []string{"FLOW-A"}, IncludeMappings: []string{"flow_error"}}},
	}}

	v, diags := Build(b, "V-FLOW")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	for _, e := range v.Edges {
		if e.Type != "flow_error" {
			t.Fatalf("expected only flow_error edges with includeMappings filter, got %+v", v.Edges)
		}
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestBuild_InteractionFlowIncludesActivityEdgesWhenConfigured(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalUnits: []model.FunctionalUnit{{ID: "FU-A", Name: "Core"}},
			Interfaces:      []model.Interface{{ID: "IF-A", Name: "Ingress API", Owner: "FU-A"}},
			DataObjects:     []model.DataObject{{ID: "DO-A", Name: "Payload"}},
			Flows: []model.Flow{{
				ID:    "FLOW-A",
				Title: "Flow",
				Entry: []string{"start"},
				Steps: []model.FlowStep{
					{ID: "start", Kind: "system_action", Ref: "FU-A", Action: "Handle request"},
				},
			}},
			Mappings: []model.Mapping{
				{Type: "calls", From: "FU-A", To: "IF-A"},
				{Type: "writes", From: "FU-A", To: "DO-A"},
			},
		},
		Views: []model.View{{
			ID:              "V-FLOW",
			Kind:            "interaction-flow",
			Roots:           []string{"FLOW-A"},
			IncludeKinds:    []string{"flow", "flow_step", "functional_unit", "interface", "data_object"},
			IncludeMappings: []string{"flow_next", "flow_ref", "calls", "writes"},
		}},
	}}

	v, diags := Build(b, "V-FLOW")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	hasCalls := false
	hasWrites := false
	for _, e := range v.Edges {
		if e.Type == "calls" && e.From == "FU-A" && e.To == "IF-A" {
			hasCalls = true
		}
		if e.Type == "writes" && e.From == "FU-A" && e.To == "DO-A" {
			hasWrites = true
		}
	}
	if !hasCalls || !hasWrites {
		t.Fatalf("expected interaction-flow activity edges, got %+v", v.Edges)
	}
}
