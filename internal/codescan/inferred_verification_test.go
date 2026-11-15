// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
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
		"package sample\n\n" +
		"// TRLC-" + "LINKS: REQ-PRR-004\n" +
		"func TestPolicyOnlyFallback() {}\n"
	if err := os.WriteFile(testPath, []byte(testContent), 0o644); err != nil {
		t.Fatalf("write test fixture: %v", err)
	}

	bundle := model.Bundle{
		ArchitecturePath: filepath.Join(root, "engmod.yml"),
		Architecture: model.ArchitectureDocument{
			InferenceHints: model.InferenceHints{},
		},
	}
	requirements := model.RequirementsDocument{
		Requirements: []model.Requirement{
			{ID: "REQ-PRR-004", AppliesTo: []string{"FU-POLICY-CHECKS"}},
		},
	}

	checks, _ := InferVerificationChecks(VerificationScan{Bundle: bundle, Requirements: requirements, InferredCode: nil, CodeRoot: ""})
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

	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml")}
	requirements := model.RequirementsDocument{
		Requirements: []model.Requirement{{ID: "REQ-ROOT-001", AppliesTo: []string{"FU-ROOT"}}},
	}

	checks, diags := InferVerificationChecks(VerificationScan{Bundle: bundle, Requirements: requirements, InferredCode: nil, CodeRoot: ""})
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

	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml")}
	requirements := model.RequirementsDocument{
		Requirements: []model.Requirement{{ID: "REQ-FEATURE-001", AppliesTo: []string{"FU-FEATURE"}}},
	}

	checks, diags := InferVerificationChecks(VerificationScan{Bundle: bundle, Requirements: requirements, InferredCode: nil, CodeRoot: ""})
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
	writeFixtureFiles(t, root, map[string]string{
		"tests/validation.test.js":       "// TRLC-LINKS: REQ-PRR-004\n",
		"tests/e2e-requirements.test.js": "// TRLC-LINKS: REQ-PRR-004\n",
		"test-results/validation.json":   `{"results":[{"requirement":"REQ-PRR-004","status":"pass"}]}`,
	})

	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml")}
	requirements := model.RequirementsDocument{
		Requirements: []model.Requirement{{ID: "REQ-PRR-004", AppliesTo: []string{"FU-POLICY-CHECKS"}}},
	}

	checks, _ := InferVerificationChecks(VerificationScan{Bundle: bundle, Requirements: requirements, InferredCode: nil, CodeRoot: ""})

	validationRel := filepath.ToSlash(filepath.Join("tests", "validation.test.js"))
	validationCheck, ok := findCheckByEvidence(checks, validationRel)
	if !ok {
		t.Fatalf("missing inferred verification check for %s", validationRel)
	}
	if validationCheck.Status != "pass" {
		t.Fatalf("expected validation test check status pass, got %q", validationCheck.Status)
	}

	resultRel := filepath.ToSlash(filepath.Join("test-results", "validation.json"))
	if !hasPassEvidence(validationCheck, "REQ-PRR-004", resultRel) {
		t.Fatalf("expected pass result evidence from %s in validation check", resultRel)
	}
}

// writeFixtureFiles writes each slash-separated path under root, creating its directories.
func writeFixtureFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, text := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create dir of %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", rel, err)
		}
	}
}

// hasPassEvidence reports whether the check holds a passing result for the
// requirement taken from the evidence file.
func hasPassEvidence(c VerificationCheck, req, evidence string) bool {
	for _, r := range c.Results {
		if r.Requirement == req && r.Status == "pass" && r.Evidence == evidence {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-010
func TestBuildVerificationCodeElementIndex_PrefersSymbolsAndSkipsDependencies(t *testing.T) {
	items := []CodeItem{
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
func findCheckByEvidence(checks []VerificationCheck, evidence string) (VerificationCheck, bool) {
	for _, check := range checks {
		if slices.Contains(check.Evidence, evidence) {
			return check, true
		}
	}
	return VerificationCheck{}, false
}

// TRLC-LINKS: REQ-EMG-010
func TestVerilogTestbenchPathClassification(t *testing.T) {
	assertTestPaths(t, []string{"fpga/sram_bench/sim/tb.v", "fpga/sim/tb_capture.v", "rtl/capture_tb.v"}, true)
	assertTestPaths(t, []string{"fpga/sram_bench/bench.v", "rtl/tb_controller.vh", "rtl/table.v"}, false)
}

// assertTestPaths checks the test-path classification of each path against want.
func assertTestPaths(t *testing.T, paths []string, want bool) {
	t.Helper()
	for _, path := range paths {
		if isVerificationTestPath(path) != want {
			t.Errorf("%s: test path = %v, want %v", path, !want, want)
		}
	}
}
