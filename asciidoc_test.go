package engmodel

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

func TestGenerateAsciiDocFromFiles_EndToEnd(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")

	res, err := GenerateAsciiDocFromFiles(modelPath, requirementsPath, designPath, AsciiDocOptions{})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if validate.HasErrors(res.Diagnostics) {
		t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
	}
	if !strings.Contains(res.Document, "= Sample Payments Layered Design") {
		t.Fatalf("missing document title")
	}
	if !strings.Contains(res.Document, "[source,mermaid]") {
		t.Fatalf("missing mermaid blocks")
	}
	if strings.Contains(res.Document, "MERMAID:") || strings.Contains(res.Document, "INF:") {
		t.Fatalf("did not expect helper markers in final document")
	}
	if !strings.Contains(res.Document, "=== Functional Groups and Units (View Scoped)") {
		t.Fatalf("missing view-scoped functional groups/units section")
	}
	if !strings.Contains(res.Document, "=== Verification Inventory") {
		t.Fatalf("missing verification inventory section")
	}
	if !strings.Contains(res.Document, "VER-INF-") {
		t.Fatalf("missing inferred verification check content")
	}
	if !strings.Contains(res.Document, "tests/e2e/authorized_checkout_flow.yaml") {
		t.Fatalf("missing inferred verification evidence path")
	}
	if !strings.Contains(res.Document, "=== Verification Result Mapping") {
		t.Fatalf("missing verification result mapping section")
	}
	if !strings.Contains(res.Document, "=== Verification References") {
		t.Fatalf("missing verification references section")
	}
	if !strings.Contains(res.Document, "REQ-PAY-005") || !strings.Contains(res.Document, "partial") {
		t.Fatalf("missing verification result rows")
	}
}

func TestEngdocCLI_EndToEnd(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")

	cmd := exec.Command("go", "run", "./cmd/engdoc",
		"--model", modelPath,
		"--requirements", requirementsPath,
		"--design", designPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("engdoc cli failed: %v\noutput:\n%s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "== Architecture Intent View") {
		t.Fatalf("cli output missing architecture intent view chapter")
	}
	if !strings.Contains(text, "== Deployment View") {
		t.Fatalf("cli output missing deployment view chapter")
	}
}

func TestGenerateAsciiDoc_EARSLintStrictFailure(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")

	bundle, err := model.LoadBundle(modelPath)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		t.Fatalf("load requirements failed: %v", err)
	}
	design, err := model.LoadDesign(designPath)
	if err != nil {
		t.Fatalf("load design failed: %v", err)
	}

	requirements.LintRun.Mode = "strict"
	requirements.Requirements[0].Text = "The door control system lock the door."

	res, err := GenerateAsciiDoc(bundle, requirements, design, AsciiDocOptions{})
	if err == nil {
		t.Fatalf("expected strict EARS lint failure")
	}
	if !validate.HasErrors(res.Diagnostics) {
		t.Fatalf("expected error diagnostics, got: %+v", res.Diagnostics)
	}
}

func TestGenerateAsciiDoc_FailsWhenCatalogDescriptionMissing(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")

	bundle, err := model.LoadBundle(modelPath)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		t.Fatalf("load requirements failed: %v", err)
	}
	design, err := model.LoadDesign(designPath)
	if err != nil {
		t.Fatalf("load design failed: %v", err)
	}

	if len(bundle.Catalog.Catalog.FunctionalGroups) == 0 {
		t.Fatalf("expected functionalGroups in sample catalog")
	}
	bundle.Catalog.Catalog.FunctionalGroups[0].Definition = ""

	res, err := GenerateAsciiDoc(bundle, requirements, design, AsciiDocOptions{})
	if err == nil {
		t.Fatalf("expected validation failure for missing catalog description")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "catalog.missing_description" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected catalog.missing_description diagnostic, got: %+v", res.Diagnostics)
	}
}

func TestGenerateOutputs_VerificationStatusConsistentForTestAndResultNameMismatch(t *testing.T) {
	sample := filepath.Join("examples", "payments-engineering-sample")
	bundle, err := model.LoadBundle(filepath.Join(sample, "architecture.yml"))
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	requirements, err := model.LoadRequirements(filepath.Join(sample, "requirements.yml"))
	if err != nil {
		t.Fatalf("load requirements failed: %v", err)
	}
	design, err := model.LoadDesign(filepath.Join(sample, "design.yml"))
	if err != nil {
		t.Fatalf("load design failed: %v", err)
	}

	root := t.TempDir()
	testsDir := filepath.Join(root, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("create tests dir: %v", err)
	}
	resultsDir := filepath.Join(root, "test-results")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatalf("create test-results dir: %v", err)
	}

	validationTest := filepath.Join(testsDir, "validation.test.js")
	if err := os.WriteFile(validationTest, []byte("// TRACE-REQS: REQ-PAY-001\n"), 0o644); err != nil {
		t.Fatalf("write validation test fixture: %v", err)
	}
	otherTest := filepath.Join(testsDir, "e2e-requirements.test.js")
	if err := os.WriteFile(otherTest, []byte("// TRACE-REQS: REQ-PAY-001\n"), 0o644); err != nil {
		t.Fatalf("write e2e test fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resultsDir, "validation.json"), []byte(`{"results":[{"requirement":"REQ-PAY-001","status":"pass"}]}`), 0o644); err != nil {
		t.Fatalf("write validation result fixture: %v", err)
	}

	bundle.ArchitecturePath = filepath.Join(root, "architecture.yml")

	checks, _ := inferVerificationChecks(bundle, requirements, nil, "")
	validationCheck, ok := findCheckByEvidence(checks, filepath.ToSlash(filepath.Join("tests", "validation.test.js")))
	if !ok {
		t.Fatalf("missing validation inferred verification check")
	}

	aiRes, err := GenerateAIView(bundle, requirements, design, AIViewOptions{})
	if err != nil {
		t.Fatalf("generate ai view failed: %v", err)
	}
	foundAIStatus := false
	for _, e := range aiRes.Document.Entities {
		if e.Kind == "verification" && e.ID == validationCheck.ID {
			if e.Status != "pass" {
				t.Fatalf("expected ai verification status pass for %s, got %q", e.ID, e.Status)
			}
			foundAIStatus = true
			break
		}
	}
	if !foundAIStatus {
		t.Fatalf("missing ai verification entity for %s", validationCheck.ID)
	}

	adocRes, err := GenerateAsciiDoc(bundle, requirements, design, AsciiDocOptions{})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if !strings.Contains(adocRes.Document, "=== Verification Inventory") {
		t.Fatalf("asciidoc output missing verification inventory")
	}
	if !strings.Contains(adocRes.Document, "=== Verification Result Mapping") {
		t.Fatalf("asciidoc output missing verification result mapping")
	}
	if !strings.Contains(adocRes.Document, "REQ-PAY-001") {
		t.Fatalf("asciidoc output missing mapped requirement")
	}
	if !strings.Contains(adocRes.Document, "|Status |pass") {
		t.Fatalf("expected asciidoc verification status pass for %s", validationCheck.ID)
	}
}

func TestGenerateOutputs_DeploymentEvidenceAppearsInRequirementCoverage(t *testing.T) {
	sample := filepath.Join("examples", "payments-engineering-sample")
	bundle, err := model.LoadBundle(filepath.Join(sample, "architecture.yml"))
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	requirements, err := model.LoadRequirements(filepath.Join(sample, "requirements.yml"))
	if err != nil {
		t.Fatalf("load requirements failed: %v", err)
	}
	design, err := model.LoadDesign(filepath.Join(sample, "design.yml"))
	if err != nil {
		t.Fatalf("load design failed: %v", err)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "deploy", "oci"), 0o755); err != nil {
		t.Fatalf("create deploy/oci: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "deploy", "kairos"), 0o755); err != nil {
		t.Fatalf("create deploy/kairos: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "infra", "flux"), 0o755); err != nil {
		t.Fatalf("create infra/flux: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "deploy", "oci", "checkout-images.lock.json"), []byte(`{"images":[{"name":"checkout","digest":"sha256:abc"}]}`), 0o644); err != nil {
		t.Fatalf("write checkout lock file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "deploy", "oci", "compose.yaml"), []byte("services:\n  checkout:\n    image: checkout@sha256:abc\n"), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "deploy", "kairos", "payment-authorization-profile.yaml"), []byte("digestPolicy: immutable-only\nrequireDigestPinning: true\n"), 0o644); err != nil {
		t.Fatalf("write kairos profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "infra", "flux", "payment-authorization-kustomization.yaml"), []byte("imagesLockFile: ../../deploy/oci/checkout-images.lock.json\n"), 0o644); err != nil {
		t.Fatalf("write flux kustomization: %v", err)
	}

	bundle.ArchitecturePath = filepath.Join(root, "architecture.yml")
	bundle.Architecture.InferenceHints.RuntimeSources = []string{"."}

	aiRes, err := GenerateAIView(bundle, requirements, design, AIViewOptions{})
	if err != nil {
		t.Fatalf("generate ai view failed: %v", err)
	}

	deploymentRuntimeIDs := map[string]bool{}
	for _, e := range aiRes.Document.Entities {
		if e.Kind != "runtime_element" {
			continue
		}
		title := strings.TrimSpace(e.Title)
		if strings.Contains(title, "deploy/oci/") || strings.Contains(title, "deploy/kairos/") || strings.Contains(title, "infra/flux/") {
			deploymentRuntimeIDs[e.ID] = true
		}
	}
	if len(deploymentRuntimeIDs) == 0 {
		t.Fatalf("expected deployment runtime evidence entities in ai view")
	}

	foundSupportPath := false
	for _, sp := range aiRes.Document.SupportPaths {
		if strings.TrimSpace(sp.FromID) != "REQ-PAY-001" {
			continue
		}
		for _, id := range sp.Path {
			if deploymentRuntimeIDs[id] {
				foundSupportPath = true
				break
			}
		}
	}
	if !foundSupportPath {
		t.Fatalf("expected REQ-PAY-001 support path to include deployment runtime evidence")
	}

	adocRes, err := GenerateAsciiDoc(bundle, requirements, design, AsciiDocOptions{})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if !strings.Contains(adocRes.Document, "REQ-PAY-001") {
		t.Fatalf("expected requirement section for REQ-PAY-001")
	}
	if !strings.Contains(adocRes.Document, "deploy/oci/checkout-images.lock.json") {
		t.Fatalf("expected deployment evidence label in requirement coverage graph")
	}
	if !strings.Contains(adocRes.Document, "runtime evidence") {
		t.Fatalf("expected runtime evidence edge in requirement coverage graph")
	}
}
