// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"errors"
	"testing"
)

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func TestNewCanonicalBundleRejectsProjectionErrors(t *testing.T) {
	bundle := Bundle{
		Architecture: ArchitectureDocument{
			Model: ModelMeta{ID: "MODEL-TEST", Title: "Test"},
			Semantics: SemanticContent{
				Elements: []SemanticElement{
					{ID: "PART-DUPLICATE", Kind: ElementPartDefinition},
					{ID: "PART-DUPLICATE", Kind: ElementPartDefinition},
				},
			},
		},
	}

	canonical, err := NewCanonicalBundle(bundle)
	if err == nil {
		t.Fatal("expected canonical projection error")
	}
	var projectionError *SemanticProjectionError
	if !errors.As(err, &projectionError) {
		t.Fatalf("expected SemanticProjectionError, got %T", err)
	}
	if !HasSemanticErrors(canonical.Diagnostics()) {
		t.Fatalf("expected error diagnostics, got %+v", canonical.Diagnostics())
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func TestNewCanonicalRequirementsPreservesRequirementConcepts(t *testing.T) {
	requirements := RequirementsDocument{
		LintRun: LintRun{ID: "requirements"},
		Requirements: []Requirement{
			{ID: "REQ-TEST-001", Text: "The system shall preserve canonical identity."},
		},
	}

	canonical, err := NewCanonicalRequirements(requirements)
	if err != nil {
		t.Fatalf("project requirements: %v", err)
	}
	if got := canonical.Documents().Requirements.Requirements[0].ID; got != "REQ-TEST-001" {
		t.Fatalf("unexpected compatibility document requirement: %q", got)
	}
	found := false
	for _, element := range canonical.Semantic().Elements {
		if element.ID == "REQ-TEST-001" && element.Kind == ElementRequirementDefinition {
			found = true
		}
	}
	if !found {
		t.Fatal("canonical requirement concept not found")
	}
}
