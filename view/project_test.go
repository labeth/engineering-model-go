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
