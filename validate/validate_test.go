package validate

import (
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

func TestBundleValidationNoErrors(t *testing.T) {
	p := filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml")
	b, err := model.LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	diags := Bundle(b)
	if HasErrors(diags) {
		t.Fatalf("expected no validation errors, got: %+v", diags)
	}
}

func TestViewIDIsFreeButKindIsStrict(t *testing.T) {
	p := filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml")
	b, err := model.LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	if len(b.Architecture.Views) == 0 {
		t.Fatalf("expected sample views")
	}

	// Free-form IDs should be accepted as long as kind is supported.
	b.Architecture.Views[0].ID = "run"
	b.Architecture.Views[0].Kind = "communication"
	diags := Bundle(b)
	if HasErrors(diags) {
		t.Fatalf("expected no errors for free-form view id with valid kind, got: %+v", diags)
	}

	// Unsupported kind should fail validation regardless of ID.
	b.Architecture.Views[0].Kind = "run"
	diags = Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected errors for unsupported view kind")
	}
	found := false
	for _, d := range diags {
		if d.Code == "model.unknown_view_kind" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected model.unknown_view_kind diagnostic, got: %+v", diags)
	}
}

func TestBundleValidation_ExpandedMappingTypesAndPairs(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups:   []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:    []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
		Actors:             []model.Actor{{ID: "ACT-A", Name: "Actor A"}},
		AttackVectors:      []model.AttackVector{{ID: "AV-A", Name: "Attack"}},
		Interfaces:         []model.Interface{{ID: "IF-A", Name: "API", Owner: "FU-A"}},
		DataObjects:        []model.DataObject{{ID: "DATA-A", Name: "Payload"}},
		DeploymentTargets:  []model.DeploymentTarget{{ID: "DEP-A", Name: "Prod"}},
		Controls:           []model.Control{{ID: "CTRL-A", Name: "Digest Pinning"}},
		TrustBoundaries:    []model.TrustBoundary{{ID: "TB-A", Name: "Boundary"}},
		States:             []model.State{{ID: "STATE-A", Name: "Ready"}, {ID: "STATE-B", Name: "Applied"}},
		Events:             []model.Event{{ID: "EVT-A", Name: "Deploy Requested"}},
		ReferencedElements: []model.ReferencedElement{{ID: "REF-A", Name: "Reference", Kind: "ext", Layer: "external"}},
		Mappings: []model.Mapping{
			{Type: "calls", From: "FU-A", To: "IF-A"},
			{Type: "writes", From: "FU-A", To: "DATA-A"},
			{Type: "deployed_to", From: "FU-A", To: "DEP-A"},
			{Type: "mitigated_by", From: "AV-A", To: "CTRL-A"},
			{Type: "bounded_by", From: "FU-A", To: "TB-A"},
			{Type: "triggered_by", From: "STATE-A", To: "EVT-A"},
			{Type: "transitions_to", From: "STATE-A", To: "STATE-B"},
			{Type: "guarded_by", From: "STATE-B", To: "CTRL-A"},
		},
	}, Views: []model.View{{ID: "V", Kind: "traceability", Roots: []string{"FU-A"}}}}}

	if diags := Bundle(b); HasErrors(diags) {
		t.Fatalf("expected expanded mapping set to validate, got: %+v", diags)
	}

	b.Architecture.AuthoredArchitecture.Mappings = append(b.Architecture.AuthoredArchitecture.Mappings, model.Mapping{Type: "writes", From: "ACT-A", To: "DATA-A"})
	diags := Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected invalid mapping pair to fail")
	}
	found := false
	for _, d := range diags {
		if d.Code == "model.invalid_mapping_pair" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected model.invalid_mapping_pair diagnostic, got: %+v", diags)
	}
}

func TestBundleValidation_ViewSpecFields(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
	}, Views: []model.View{{
		ID:              "V",
		Kind:            "communication",
		Roots:           []string{"FU-A"},
		IncludeKinds:    []string{"functional_unit", "interface"},
		ExcludeKinds:    []string{"attack_vector"},
		IncludeMappings: []string{"calls", "depends_on"},
		ExcludeMappings: []string{"targets"},
		MaxDepth:        2,
	}}}}

	if diags := Bundle(b); HasErrors(diags) {
		t.Fatalf("expected valid view spec fields, got: %+v", diags)
	}

	b.Architecture.Views[0].IncludeKinds = []string{"invalid-kind"}
	b.Architecture.Views[0].MaxDepth = -1
	diags := Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected invalid view spec to fail")
	}
	hasKindErr := false
	hasDepthErr := false
	for _, d := range diags {
		if d.Code == "model.unknown_view_entity_kind" {
			hasKindErr = true
		}
		if d.Code == "model.invalid_view_depth" {
			hasDepthErr = true
		}
	}
	if !hasKindErr || !hasDepthErr {
		t.Fatalf("expected unknown kind and invalid depth diagnostics, got: %+v", diags)
	}
}
