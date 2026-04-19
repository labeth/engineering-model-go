package view

import (
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

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

func TestBuild_StateLifecycleDefaults(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			States:   []model.State{{ID: "STATE-A", Name: "Ready"}, {ID: "STATE-B", Name: "Applied"}},
			Events:   []model.Event{{ID: "EVT-A", Name: "Requested"}},
			Controls: []model.Control{{ID: "CTRL-A", Name: "Gate"}},
			Mappings: []model.Mapping{
				{Type: "triggered_by", From: "STATE-A", To: "EVT-A"},
				{Type: "transitions_to", From: "STATE-A", To: "STATE-B"},
				{Type: "guarded_by", From: "STATE-B", To: "CTRL-A"},
			},
		},
		Views: []model.View{{ID: "V-STATE", Kind: "state-lifecycle", Roots: []string{"STATE-A"}}},
	}}

	v, diags := Build(b, "V-STATE")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if len(v.Edges) != 3 {
		t.Fatalf("expected 3 lifecycle edges, got %d", len(v.Edges))
	}
	hasTransition := false
	for _, e := range v.Edges {
		if e.Type == "transitions_to" {
			hasTransition = true
		}
	}
	if !hasTransition {
		t.Fatalf("expected transitions_to edge in lifecycle view")
	}
}

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
