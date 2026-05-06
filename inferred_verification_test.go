// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engmodel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-010
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

// TRLC-LINKS: REQ-EMG-010
func TestInferVerificationChecks_UsesDescriptionMarkerFromTestSource(t *testing.T) {
	root := t.TempDir()
	testsDir := filepath.Join(root, "tests", "unit")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("create tests dir: %v", err)
	}

	testPath := filepath.Join(testsDir, "policy_only_fallback_test.go")
	testContent := "// ENGMODEL-VERIFICATION-DESCRIPTION: validates policy-only fallback behavior when model inference is unavailable\n" +
		"// TRLC-" + "LINKS: REQ-PRR-004\n" +
		"package sample\n"
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

// TRLC-LINKS: REQ-EMG-010
func TestInferVerificationChecks_UsesRootGoTestFilesFromCodeRoot(t *testing.T) {
	root := t.TempDir()
	testPath := filepath.Join(root, "root_feature_test.go")
	testContent := "package sample\n\n" +
		"// TRLC-" + "LINKS: REQ-ROOT-001\n" +
		"func TestRootFeature() {}\n"
	if err := os.WriteFile(testPath, []byte(testContent), 0o644); err != nil {
		t.Fatalf("write test fixture: %v", err)
	}

	bundle := model.Bundle{
		ArchitecturePath: filepath.Join(root, "architecture.yml"),
		Architecture: model.ArchitectureDocument{
			InferenceHints: model.InferenceHints{CodeSources: []string{"./"}},
		},
	}
	requirements := model.RequirementsDocument{
		Requirements: []model.Requirement{{ID: "REQ-ROOT-001", AppliesTo: []string{"FU-ROOT"}}},
	}

	checks, diags := inferVerificationChecks(bundle, requirements, nil, "")
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got: %+v", diags)
	}
	if len(checks) != 1 {
		t.Fatalf("expected one inferred verification check, got %d", len(checks))
	}
	if checks[0].Evidence[0] != "root_feature_test.go" {
		t.Fatalf("unexpected evidence: %+v", checks[0].Evidence)
	}
	if checks[0].Verifies[0] != "REQ-ROOT-001" {
		t.Fatalf("unexpected verifies: %+v", checks[0].Verifies)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestInferVerificationChecks_UsesTestFunctionCodeElement(t *testing.T) {
	root := t.TempDir()
	testsDir := filepath.Join(root, "tests", "unit")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("create tests dir: %v", err)
	}
	testPath := filepath.Join(testsDir, "feature_test.go")
	testContent := "package unit\n\n" +
		"// TRLC-" + "LINKS: REQ-FEATURE-001\n" +
		"func TestFeature() {}\n"
	if err := os.WriteFile(testPath, []byte(testContent), 0o644); err != nil {
		t.Fatalf("write test fixture: %v", err)
	}

	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "architecture.yml")}
	requirements := model.RequirementsDocument{
		Requirements: []model.Requirement{{ID: "REQ-FEATURE-001", AppliesTo: []string{"FU-FEATURE"}}},
	}

	checks, diags := inferVerificationChecks(bundle, requirements, nil, "")
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got: %+v", diags)
	}
	if len(checks) != 1 {
		t.Fatalf("expected one inferred verification check, got %d", len(checks))
	}
	want := "feature_test.go:4"
	if len(checks[0].CodeElements) != 1 || checks[0].CodeElements[0] != want {
		t.Fatalf("unexpected code elements: got %+v want %q", checks[0].CodeElements, want)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestInferVerificationChecks_MatchesResultArtifactToTestFileByNormalizedIdentity(t *testing.T) {
	root := t.TempDir()
	testsDir := filepath.Join(root, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("create tests dir: %v", err)
	}
	resultsDir := filepath.Join(root, "test-results")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatalf("create test-results dir: %v", err)
	}

	validationPath := filepath.Join(testsDir, "validation.test.js")
	validationTest := "// TRLC-LINKS: REQ-PRR-004\n"
	if err := os.WriteFile(validationPath, []byte(validationTest), 0o644); err != nil {
		t.Fatalf("write validation test fixture: %v", err)
	}

	e2ePath := filepath.Join(testsDir, "e2e-requirements.test.js")
	e2eTest := "// TRLC-LINKS: REQ-PRR-004\n"
	if err := os.WriteFile(e2ePath, []byte(e2eTest), 0o644); err != nil {
		t.Fatalf("write e2e test fixture: %v", err)
	}

	validationResultPath := filepath.Join(resultsDir, "validation.json")
	validationResult := `{"results":[{"requirement":"REQ-PRR-004","status":"pass"}]}`
	if err := os.WriteFile(validationResultPath, []byte(validationResult), 0o644); err != nil {
		t.Fatalf("write validation result fixture: %v", err)
	}

	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "architecture.yml")}
	requirements := model.RequirementsDocument{
		Requirements: []model.Requirement{{ID: "REQ-PRR-004", AppliesTo: []string{"FU-POLICY-CHECKS"}}},
	}

	checks, _ := inferVerificationChecks(bundle, requirements, nil, "")

	validationRel := filepath.ToSlash(filepath.Join("tests", "validation.test.js"))
	validationCheck, ok := findCheckByEvidence(checks, validationRel)
	if !ok {
		t.Fatalf("missing inferred verification check for %s", validationRel)
	}
	if validationCheck.Status != "pass" {
		t.Fatalf("expected validation test check status pass, got %q", validationCheck.Status)
	}

	resultRel := filepath.ToSlash(filepath.Join("test-results", "validation.json"))
	foundResultEvidence := false
	for _, r := range validationCheck.Results {
		if r.Requirement == "REQ-PRR-004" && r.Status == "pass" && r.Evidence == resultRel {
			foundResultEvidence = true
			break
		}
	}
	if !foundResultEvidence {
		t.Fatalf("expected pass result evidence from %s in validation check", resultRel)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestBuildVerificationCodeElementIndex_PrefersSymbolsAndSkipsDependencies(t *testing.T) {
	items := []inferredCodeItem{
		{Element: "tests/unit/feature_test.go", Kind: "source_file", Source: "tests/unit/feature_test.go"},
		{Element: "testing", Kind: "library_stdlib", Source: "tests/unit/feature_test.go"},
		{Element: "os", Kind: "library_stdlib", Source: "tests/unit/feature_test.go"},
		{Element: "CODE-TESTFEATURE", Kind: "symbol", Source: "tests/unit/feature_test.go:12"},
		{Element: "CODE-TESTFEATUREOTHER", Kind: "symbol", Source: "tests/unit/feature_test.go:31"},
		{Element: "path/filepath", Kind: "library_stdlib", Source: "tests/unit/feature_test.go"},
		{Element: "tests/e2e/flow.yaml", Kind: "source_file", Source: "tests/e2e/flow.yaml"},
	}

	index := buildVerificationCodeElementIndex(items)

	got := index["tests/unit/feature_test.go"]
	want := []string{"feature_test.go:12,31"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("unexpected symbol-backed code elements: got %+v want %+v", got, want)
	}
	if got := index["tests/e2e/flow.yaml"]; len(got) != 1 || got[0] != "flow.yaml" {
		t.Fatalf("expected yaml fallback to filename, got %+v", got)
	}
}

// TRLC-LINKS: REQ-EMG-010
func findCheckByEvidence(checks []inferredVerificationCheck, evidence string) (inferredVerificationCheck, bool) {
	for _, check := range checks {
		for _, ev := range check.Evidence {
			if ev == evidence {
				return check, true
			}
		}
	}
	return inferredVerificationCheck{}, false
}
