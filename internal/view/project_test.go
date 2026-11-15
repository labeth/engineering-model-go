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

	v := mustBuild(t, b, "V-COMM")
	if len(v.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d (%+v)", len(v.Edges), v.Edges)
	}
	if v.Edges[0].Type != "calls" {
		t.Fatalf("expected calls edge, got %+v", v.Edges[0])
	}
	if hasNodeID(v, "FU-B") {
		t.Fatalf("expected FU-B to be excluded by mapping filter/depth")
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
	v := mustBuild(t, stateLifecycleBundle("STATE-A"), "V-STATE")
	requireEdgeLabels(t, v, map[edgeKey]string{
		{"STATE-A", "STATE-B", "transition"}: "transition: TR-A on EVT-A [tries < 3] / tries := tries + 1",
		{"STATE-A", "STATE-C", "transition"}: "transition: TR-B on EVT-A [tries >= 3]",
		{"STATE-A", "EVT-A", "on_event"}:     "on_event",
		{"STATE-B", "CTRL-A", "guarded_by"}:  "guarded_by",
	})
	requireNodeKinds(t, v, map[string]string{"STATE-A": "state", "STATE-B": "state", "STATE-C": "state", "EVT-A": "event", "CTRL-A": "control"})
	if hasNodeID(v, "SM-A") {
		t.Fatalf("the machine itself is not a lifecycle node: %+v", v.Nodes)
	}
}

// A machine ID as view root selects all of its states, so a view needs no
// list of state IDs.
// Mutation: not expanding a machine root yields an empty view.
// TRLC-LINKS: REQ-EMG-139
func TestBuild_StateLifecycleMachineRootExpandsToStates(t *testing.T) {
	v := mustBuild(t, stateLifecycleBundle("SM-A"), "V-STATE")
	if states := countNodeKind(v, "state"); states != 3 || len(v.Edges) != 4 {
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

	v := mustBuild(t, b, "V-FLOW")
	if len(v.Nodes) == 0 || len(v.Edges) == 0 {
		t.Fatalf("expected non-empty interaction-flow projection")
	}
	if countNodeKind(v, "flow_step") == 0 || !hasEdgeType(v, "flow_async") || !hasEdgeType(v, "flow_ref") {
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

	requireOnlyEdgeType(t, mustBuild(t, b, "V-FLOW"), "flow_error")
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

	v := mustBuild(t, b, "V-FLOW")
	if !hasEdge(v, "calls", "FU-A", "IF-A") || !hasEdge(v, "writes", "FU-A", "DO-A") {
		t.Fatalf("expected interaction-flow activity edges, got %+v", v.Edges)
	}
}
