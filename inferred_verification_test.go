package engmodel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

func TestExtractVerificationDescriptionMarker(t *testing.T) {
	line := "// ENGMODEL-VERIFICATION-DESCRIPTION: validates policy-only fallback behavior for unavailable model calls"
	got, ok := extractVerificationDescriptionMarker(line)
	if !ok {
		t.Fatalf("expected marker to be detected")
	}
	want := "validates policy-only fallback behavior for unavailable model calls"
	if got != want {
		t.Fatalf("unexpected marker value: got %q want %q", got, want)
	}
}

func TestInferVerificationChecks_UsesDescriptionMarkerFromTestSource(t *testing.T) {
	root := t.TempDir()
	testsDir := filepath.Join(root, "tests", "unit")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("create tests dir: %v", err)
	}

	testPath := filepath.Join(testsDir, "policy_only_fallback_test.go")
	testContent := `// ENGMODEL-VERIFICATION-DESCRIPTION: validates policy-only fallback behavior when model inference is unavailable
// REQ-PRR-004
package sample
`
	if err := os.WriteFile(testPath, []byte(testContent), 0o644); err != nil {
		t.Fatalf("write test fixture: %v", err)
	}

	bundle := model.Bundle{
		ArchitecturePath: filepath.Join(root, "architecture.yml"),
		Architecture: model.ArchitectureDocument{
			InferenceHints: model.InferenceHints{},
		},
	}
	requirements := model.RequirementsDocument{
		Requirements: []model.Requirement{
			{ID: "REQ-PRR-004", AppliesTo: []string{"FU-POLICY-CHECKS"}},
		},
	}

	checks, _ := inferVerificationChecks(bundle, requirements, nil, "")
	if len(checks) != 1 {
		t.Fatalf("expected one inferred verification check, got %d", len(checks))
	}
	want := "validates policy-only fallback behavior when model inference is unavailable"
	if checks[0].Description != want {
		t.Fatalf("unexpected check description: got %q want %q", checks[0].Description, want)
	}
}
