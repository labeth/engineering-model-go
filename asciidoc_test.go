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

func TestGenerateAsciiDoc_RendersStateLifecycleAndNewAuthoredReferenceKinds(t *testing.T) {
	bundle := model.Bundle{ArchitecturePath: filepath.Join(t.TempDir(), "architecture.yml"), Architecture: model.ArchitectureDocument{
		Model: model.ModelMeta{ID: "m", Title: "m"},
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups:   []model.FunctionalGroup{{ID: "FG-MEDIACHESTV-CORE", Name: "Core"}},
			FunctionalUnits:    []model.FunctionalUnit{{ID: "FU-MEDIACHESTV-CORE", Name: "Core Unit", Group: "FG-MEDIACHESTV-CORE"}},
			Actors:             []model.Actor{{ID: "ACT-MEDIACHESTV-OPERATOR", Name: "Operator"}},
			AttackVectors:      []model.AttackVector{{ID: "AV-MEDIACHESTV-INPUT-SPOOF", Name: "Input spoof"}},
			ReferencedElements: []model.ReferencedElement{{ID: "REF-MEDIACHESTV-UPSTREAM", Name: "Upstream", Kind: "service", Layer: "external"}},
			Interfaces: []model.Interface{
				{ID: "IF-MEDIACHESTV-CONTROL-API", Name: "Control API", Protocol: "https", Endpoint: "/control", SchemaRef: "schemas/control.json", Owner: "FU-MEDIACHESTV-CORE"},
			},
			DataObjects: []model.DataObject{
				{ID: "DO-MEDIACHESTV-OCI-LOCK", Name: "OCI Lock", SchemaRef: "schemas/oci-lock.json", Sensitivity: "internal"},
			},
			DeploymentTargets: []model.DeploymentTarget{
				{ID: "DEP-MEDIACHESTV-KAIROS-NODE", Name: "Kairos Node", Environment: "prod", Cluster: "kairos", Namespace: "mediachest", TrustZone: "device"},
			},
			Controls: []model.Control{
				{ID: "CTRL-MEDIACHESTV-DIGEST-PINNING", Name: "Digest pinning", Category: "supply-chain", Description: "digest only"},
			},
			TrustBoundaries: []model.TrustBoundary{
				{ID: "TB-MEDIACHESTV-DEVICE", Name: "Device", Description: "device trust boundary"},
			},
			States: []model.State{
				{ID: "STATE-MEDIACHESTV-IDLE", Name: "Idle"},
				{ID: "STATE-MEDIACHESTV-APPLYING", Name: "Applying"},
			},
			Events: []model.Event{{ID: "EVT-MEDIACHESTV-DEPLOY-REQUESTED", Name: "Deploy Requested"}},
			Mappings: []model.Mapping{
				{Type: "contains", From: "FG-MEDIACHESTV-CORE", To: "FU-MEDIACHESTV-CORE"},
				{Type: "interacts_with", From: "ACT-MEDIACHESTV-OPERATOR", To: "FU-MEDIACHESTV-CORE"},
				{Type: "depends_on", From: "FU-MEDIACHESTV-CORE", To: "REF-MEDIACHESTV-UPSTREAM"},
				{Type: "calls", From: "FU-MEDIACHESTV-CORE", To: "IF-MEDIACHESTV-CONTROL-API"},
				{Type: "writes", From: "FU-MEDIACHESTV-CORE", To: "DO-MEDIACHESTV-OCI-LOCK"},
				{Type: "deployed_to", From: "FU-MEDIACHESTV-CORE", To: "DEP-MEDIACHESTV-KAIROS-NODE"},
				{Type: "targets", From: "AV-MEDIACHESTV-INPUT-SPOOF", To: "FU-MEDIACHESTV-CORE"},
				{Type: "mitigated_by", From: "AV-MEDIACHESTV-INPUT-SPOOF", To: "CTRL-MEDIACHESTV-DIGEST-PINNING"},
				{Type: "bounded_by", From: "FU-MEDIACHESTV-CORE", To: "TB-MEDIACHESTV-DEVICE"},
				{Type: "transitions_to", From: "STATE-MEDIACHESTV-IDLE", To: "STATE-MEDIACHESTV-APPLYING"},
				{Type: "triggered_by", From: "STATE-MEDIACHESTV-IDLE", To: "EVT-MEDIACHESTV-DEPLOY-REQUESTED"},
				{Type: "guarded_by", From: "STATE-MEDIACHESTV-APPLYING", To: "CTRL-MEDIACHESTV-DIGEST-PINNING"},
				{Type: "guarded_by", From: "STATE-MEDIACHESTV-IDLE", To: "TB-MEDIACHESTV-DEVICE"},
			},
		},
		Views: []model.View{
			{ID: "VIEW-ARCHITECTURE-INTENT", Kind: "architecture-intent", Roots: []string{"FG-MEDIACHESTV-CORE"}},
			{ID: "VIEW-COMMUNICATION", Kind: "communication", Roots: []string{"FU-MEDIACHESTV-CORE"}, IncludeMappings: []string{"calls", "interacts_with"}},
			{ID: "VIEW-DEPLOYMENT", Kind: "deployment", Roots: []string{"FU-MEDIACHESTV-CORE"}, IncludeMappings: []string{"deployed_to", "bounded_by"}},
			{ID: "VIEW-SECURITY", Kind: "security", Roots: []string{"AV-MEDIACHESTV-INPUT-SPOOF"}},
			{ID: "VIEW-TRACEABILITY", Kind: "traceability", Roots: []string{"FU-MEDIACHESTV-CORE"}},
			{ID: "VIEW-STATE-LIFECYCLE", Kind: "state-lifecycle", Roots: []string{"STATE-MEDIACHESTV-IDLE"}, IncludeKinds: []string{"state", "event", "control", "trust_boundary"}, IncludeMappings: []string{"transitions_to", "triggered_by", "guarded_by"}},
		},
	}, Catalog: model.CatalogDocument{}}

	res, err := GenerateAsciiDoc(bundle, model.RequirementsDocument{}, model.DesignDocument{}, AsciiDocOptions{})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}

	if !strings.Contains(res.Document, "== State Lifecycle View") {
		t.Fatalf("missing State Lifecycle View chapter")
	}

	stateSection := sectionByHeading(res.Document, "== State Lifecycle View")
	for _, want := range []string{"STATE-MEDIACHESTV-IDLE", "EVT-MEDIACHESTV-DEPLOY-REQUESTED", "CTRL-MEDIACHESTV-DIGEST-PINNING", "TB-MEDIACHESTV-DEVICE"} {
		if !strings.Contains(stateSection, want) {
			t.Fatalf("state lifecycle section missing node %s", want)
		}
	}
	for _, want := range []string{"transitions_to", "triggered_by", "guarded_by"} {
		if !strings.Contains(stateSection, want) {
			t.Fatalf("state lifecycle projection missing mapping type %s", want)
		}
	}
	if strings.Contains(stateSection, "depends_on") {
		t.Fatalf("state lifecycle section should respect includeMappings and exclude depends_on")
	}

	for _, id := range []string{"IF-MEDIACHESTV-CONTROL-API", "DO-MEDIACHESTV-OCI-LOCK", "DEP-MEDIACHESTV-KAIROS-NODE", "CTRL-MEDIACHESTV-DIGEST-PINNING", "TB-MEDIACHESTV-DEVICE", "STATE-MEDIACHESTV-IDLE", "EVT-MEDIACHESTV-DEPLOY-REQUESTED"} {
		block := referenceBlockByID(res.Document, id)
		if block == "" {
			t.Fatalf("missing reference index entry for %s", id)
		}
		if !strings.Contains(block, "|Mentioned In |<<") {
			t.Fatalf("reference entry for %s missing backlinks", id)
		}
	}
}

func sectionByHeading(doc, heading string) string {
	idx := strings.Index(doc, heading)
	if idx < 0 {
		return ""
	}
	section := doc[idx:]
	next := strings.Index(section[len(heading):], "\n== ")
	if next < 0 {
		return section
	}
	return section[:len(heading)+next]
}

func referenceBlockByID(doc, id string) string {
	marker := "==== " + id
	idx := strings.Index(doc, marker)
	if idx < 0 {
		return ""
	}
	block := doc[idx:]
	next := strings.Index(block[len(marker):], "\n[discrete]\n==== ")
	if next < 0 {
		return block
	}
	return block[:len(marker)+next]
}

func TestGenerateAsciiDoc_InteractionFlowViewAndReferences(t *testing.T) {
	bundle := model.Bundle{ArchitecturePath: filepath.Join(t.TempDir(), "architecture.yml"), Architecture: model.ArchitectureDocument{
		Model: model.ModelMeta{ID: "m", Title: "m"},
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group"}},
			FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Name: "Unit", Group: "FG-A"}},
			Actors:           []model.Actor{{ID: "ACT-A", Name: "User"}},
			Interfaces:       []model.Interface{{ID: "IF-A", Name: "Control API", Owner: "FU-A"}},
			Flows: []model.Flow{{
				ID:    "FLOW-INPUT",
				Title: "Input Selection Flow",
				Entry: []string{"submit"},
				Exits: []string{"ack"},
				Steps: []model.FlowStep{
					{ID: "submit", Kind: "user_action", Ref: "ACT-A", Action: "Submit input", Next: []string{"call-api"}},
					{ID: "call-api", Kind: "system_action", Ref: "IF-A", Action: "Call API", Async: true, Next: []string{"ack"}},
					{ID: "ack", Kind: "system_action", Ref: "FU-A", Action: "Acknowledge"},
				},
			}},
		},
		Views: []model.View{{ID: "VIEW-FLOW", Kind: "interaction-flow", Roots: []string{"FLOW-INPUT"}}},
	}}

	res, err := GenerateAsciiDoc(bundle, model.RequirementsDocument{}, model.DesignDocument{}, AsciiDocOptions{})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if !strings.Contains(res.Document, "== Interaction Flow View") {
		t.Fatalf("missing Interaction Flow View chapter")
	}
	flowSection := sectionByHeading(res.Document, "== Interaction Flow View")
	for _, want := range []string{"FLOW-INPUT", "FLOW-INPUT::submit", "flow_next", "flow_async", "flow_ref"} {
		if !strings.Contains(flowSection, want) {
			t.Fatalf("interaction flow section missing %s", want)
		}
	}
	for _, id := range []string{"FLOW-INPUT", "FLOW-INPUT::submit"} {
		block := referenceBlockByID(res.Document, id)
		if block == "" {
			t.Fatalf("missing reference index entry for %s", id)
		}
		if !strings.Contains(block, "|Mentioned In |<<") {
			t.Fatalf("reference entry for %s missing backlinks", id)
		}
	}
}
