// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-003
func TestGenerateAsciiDocFromFiles_EndToEnd(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "engmod.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "model", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "model", "views.yml")

	res, err := GenerateFromFiles(FilePaths{Architecture: modelPath, Requirements: requirementsPath, Design: designPath}, Options{})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if validate.HasErrors(res.Diagnostics) {
		t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
	}
	about(t, res.Document).has(
		docCheck{"= Sample Payments Layered Design", "missing document title"},
		docCheck{"[source,mermaid]", "missing mermaid blocks"},
		docCheck{"=== Functional Groups and Units (View Scoped)", "missing view-scoped functional groups/units section"},
		docCheck{"=== Verification Inventory", "missing verification inventory section"},
		docCheck{"VER-INF-", "missing inferred verification check content"},
		docCheck{"tests/e2e/authorized_checkout_flow.yaml", "missing inferred verification evidence path"},
		docCheck{"=== Verification Result Mapping", "missing verification result mapping section"},
		docCheck{"=== Verification References", "missing verification references section"},
		docCheck{"REQ-PAY-005", "missing verification result rows"},
		docCheck{"partial", "missing verification result rows"},
	)
	about(t, res.Document).lacks(
		docCheck{"MERMAID:", "did not expect helper markers in final document"},
		docCheck{"INF:", "did not expect helper markers in final document"},
		docCheck{"=== View-Scoped Projection", "did not expect low-value view-scoped projection section"},
	)
	requireNoBlankPaddedLines(t, res.Document)
}

// TRLC-LINKS: REQ-EMG-014
func TestGenerateAsciiDoc_DecisionsDocumentAndMainDocLinks(t *testing.T) {
	bundle := model.Bundle{ArchitecturePath: filepath.Join(t.TempDir(), "engmod.yml"), Architecture: model.ArchitectureDocument{
		Model: model.ModelMeta{ID: "decisions-sample", Title: "Decisions Sample"},
		Decisions: []model.Decision{{
			ID:           "ADR-SAMPLE-001",
			Title:        "Publish ADRs separately",
			Status:       "accepted",
			Date:         "2026-04-29",
			Context:      "The main architecture document should stay compact.",
			Decision:     "Generate a separate decisions document and link it from the main document.",
			Consequences: []string{"Readers can inspect rationale without bloating the main document."},
		}},
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
			FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Name: "Unit A", Group: "FG-A"}},
		},
		Views: []model.View{{ID: "VIEW-A", Kind: "architecture-intent", Roots: []string{"FG-A"}}},
	}}

	res := mustGenerateBundle(t, bundle, Options{DecisionsDocPath: "DECISIONS.adoc"})
	about(t, res.Document).has(
		docCheck{"== Architecture Decision Records", "main document missing ADR index"},
		docCheck{"xref:DECISIONS.adoc#ADR-SAMPLE-001[ADR-SAMPLE-001]", "main document missing ADR xref: " + res.Document},
	)
	about(t, res.DecisionsDocument).has(
		docCheck{"= Architecture Decision Records", "missing decisions document title"},
		docCheck{"[[ADR-SAMPLE-001]]", "missing decision anchor"},
		docCheck{"== ADR-SAMPLE-001: Publish ADRs separately", "missing decision section"},
		docCheck{"=== Consequences", "missing consequences section"},
	)
}

// TRLC-LINKS: REQ-EMG-014
func TestEngdocCLI_EndToEnd(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "engmod.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "model", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "model", "views.yml")

	cmd := exec.Command("go", "run", "./cmd/engdoc",
		"--model", modelPath,
		"--requirements", requirementsPath,
		"--design", designPath,
	)
	cmd.Env = os.Environ() // includes the test's ENGMOD_AS_OF
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

// TRLC-LINKS: REQ-EMG-014
func TestEngdocCLI_DecisionsOut(t *testing.T) {
	modelPath := testutil.AcceptedRepoModel(t)
	requirementsPath := filepath.Join(filepath.Dir(modelPath), "model", "requirements.yml")
	designPath := filepath.Join(filepath.Dir(modelPath), "model", "views.yml")
	outDir := t.TempDir()
	mainOut := filepath.Join(outDir, "ARCHITECTURE.adoc")
	decisionsOut := filepath.Join(outDir, "DECISIONS.adoc")

	cmd := exec.Command("go", "run", "./cmd/engdoc",
		"--model", modelPath,
		"--requirements", requirementsPath,
		"--design", designPath,
		"--out", mainOut,
		"--decisions-out", decisionsOut,
	)
	cmd.Env = os.Environ() // includes the test's ENGMOD_AS_OF
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("engdoc cli failed: %v\noutput:\n%s", err, string(out))
	}
	mainDoc, err := os.ReadFile(mainOut)
	if err != nil {
		t.Fatalf("read main doc: %v", err)
	}
	decisionsDoc, err := os.ReadFile(decisionsOut)
	if err != nil {
		t.Fatalf("read decisions doc: %v", err)
	}
	if !strings.Contains(string(mainDoc), "xref:DECISIONS.adoc#ADR-EMG-001[ADR-EMG-001]") {
		t.Fatalf("main document missing root ADR link")
	}
	if !strings.Contains(string(decisionsDoc), "== ADR-EMG-001:") {
		t.Fatalf("decisions document missing root ADR")
	}
}

// TRLC-LINKS: REQ-EMG-014
func TestGenerateAsciiDoc_EARSLintStrictFailure(t *testing.T) {
	sample := loadSample(t, filepath.Join("examples", "payments-engineering-sample"))
	sample.requirements.LintRun.Mode = "strict"
	sample.requirements.Requirements[0].Text = "The door control system lock the door."

	res, err := sample.generate()
	if err == nil {
		t.Fatalf("expected strict EARS lint failure")
	}
	if !validate.HasErrors(res.Diagnostics) {
		t.Fatalf("expected error diagnostics, got: %+v", res.Diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-014
func TestGenerateAsciiDoc_FailsWhenCatalogDescriptionMissing(t *testing.T) {
	sample := loadSample(t, filepath.Join("examples", "payments-engineering-sample"))
	if len(sample.bundle.Catalog.Catalog.FunctionalGroups) == 0 {
		t.Fatalf("expected functionalGroups in sample catalog")
	}
	sample.bundle.Catalog.Catalog.FunctionalGroups[0].Definition = ""

	res, err := sample.generate()
	if err == nil {
		t.Fatalf("expected validation failure for missing catalog description")
	}
	if !hasDiagnostic(res.Diagnostics, "catalog.missing_description") {
		t.Fatalf("expected catalog.missing_description diagnostic, got: %+v", res.Diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-014
func TestGenerateOutputs_VerificationStatusConsistentForTestAndResultNameMismatch(t *testing.T) {
	sample := loadSample(t, filepath.Join("examples", "payments-engineering-sample"))
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"tests/validation.test.js":       "// TRLC-LINKS: REQ-PAY-001\n",
		"tests/e2e-requirements.test.js": "// TRLC-LINKS: REQ-PAY-001\n",
		"test-results/validation.json":   `{"results":[{"requirement":"REQ-PAY-001","status":"pass"}]}`,
	})
	sample.bundle.ManifestPath = filepath.Join(root, "engmod.yml")

	checks, _ := codescan.InferVerificationChecks(codescan.VerificationScan{Bundle: sample.bundle, Requirements: sample.requirements, InferredCode: nil, CodeRoot: ""})
	validationCheck, ok := findCheckByEvidence(checks, filepath.ToSlash(filepath.Join("tests", "validation.test.js")))
	if !ok {
		t.Fatalf("missing validation inferred verification check")
	}
	if validationCheck.Status != "pass" {
		t.Fatalf("expected inferred verification status pass for %s, got %q", validationCheck.ID, validationCheck.Status)
	}

	adocRes := sample.mustGenerate(t)
	about(t, adocRes.Document).has(
		docCheck{"=== Verification Inventory", "asciidoc output missing verification inventory"},
		docCheck{"=== Verification Result Mapping", "asciidoc output missing verification result mapping"},
		docCheck{"REQ-PAY-001", "asciidoc output missing mapped requirement"},
		docCheck{"|Status |pass", "expected asciidoc verification status pass for " + validationCheck.ID},
	)
}

// TRLC-LINKS: REQ-EMG-003
func TestGenerateOutputs_DeploymentEvidenceAppearsInRequirementCoverage(t *testing.T) {
	sample := loadSample(t, filepath.Join("examples", "payments-engineering-sample"))
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"deploy/oci/checkout-images.lock.json":                `{"images":[{"name":"checkout","digest":"sha256:abc"}]}`,
		"deploy/oci/compose.yaml":                             "services:\n  checkout:\n    image: checkout@sha256:abc\n",
		"deploy/kairos/payment-authorization-profile.yaml":    "digestPolicy: immutable-only\nrequireDigestPinning: true\n",
		"infra/flux/payment-authorization-kustomization.yaml": "imagesLockFile: ../../deploy/oci/checkout-images.lock.json\n",
	})
	sample.bundle.ManifestPath = filepath.Join(root, "engmod.yml")
	sample.bundle.Architecture.InferenceHints.RuntimeSources = []string{"."}

	adocRes := sample.mustGenerate(t)
	about(t, adocRes.Document).has(
		docCheck{"REQ-PAY-001", "expected requirement section for REQ-PAY-001"},
		docCheck{"deploy/oci/checkout-images.lock.json", "expected deployment evidence label in requirement coverage graph"},
		docCheck{"runtime evidence", "expected runtime evidence edge in requirement coverage graph"},
	)
}

// mediaChestStructure is the part of the media chest fixture that its name says.
func mediaChestStructure() model.AuthoredArchitecture {
	return model.AuthoredArchitecture{
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
	}
}

// mediaChestThreats is the part of the media chest fixture that its name says.
func mediaChestThreats() model.AuthoredArchitecture {
	return model.AuthoredArchitecture{
		ThreatAssumptions: []model.ThreatAssumption{
			{ID: "ASM-MEDIACHESTV-DEVICE-ROOT", Title: "Device root trust", Statement: "Device root of trust is provisioned securely", Status: "accepted", Owner: "ACT-MEDIACHESTV-OPERATOR", AppliesTo: []string{"FU-MEDIACHESTV-CORE"}},
		},
		ThreatOutOfScope: []model.ThreatOutOfScope{
			{ID: "OOS-MEDIACHESTV-HW", Title: "Hardware tamper internals", Reason: "Physical tamper model owned by hardware team", Status: "approved", Owner: "ACT-MEDIACHESTV-OPERATOR", AppliesTo: []string{"DEP-MEDIACHESTV-KAIROS-NODE"}, ExpiresOn: "2027-01-01"},
		},
		ThreatScenarios: []model.ThreatScenario{
			{ID: "TS-MEDIACHESTV-INPUT-SPOOF", Title: "Spoofed update input", AttackVectorRef: "AV-MEDIACHESTV-INPUT-SPOOF", AppliesTo: []string{"FU-MEDIACHESTV-CORE"}, Likelihood: "medium", Impact: "high", Severity: "high", Status: "mitigating", Owner: "ACT-MEDIACHESTV-OPERATOR", RelatedControls: []string{"CTRL-MEDIACHESTV-DIGEST-PINNING"}, AssumptionRefs: []string{"ASM-MEDIACHESTV-DEVICE-ROOT"}, OutOfScopeRefs: []string{"OOS-MEDIACHESTV-HW"}, MitigationRefs: []string{"MIT-MEDIACHESTV-DIGEST"}, VerificationRefs: []string{"CV-MEDIACHESTV-DIGEST"}},
		},
		ThreatMitigations: []model.ThreatMitigation{
			{ID: "MIT-MEDIACHESTV-DIGEST", ThreatScenarioRef: "TS-MEDIACHESTV-INPUT-SPOOF", ControlRef: "CTRL-MEDIACHESTV-DIGEST-PINNING", Status: "implemented", Effectiveness: "high", Owner: "ACT-MEDIACHESTV-OPERATOR", VerificationRefs: []string{"CV-MEDIACHESTV-DIGEST"}},
		},
		ControlVerifications: []model.ControlVerification{
			{ID: "CV-MEDIACHESTV-DIGEST", ControlRef: "CTRL-MEDIACHESTV-DIGEST-PINNING", ThreatScenarioRefs: []string{"TS-MEDIACHESTV-INPUT-SPOOF"}, Method: "test", Status: "pass", Owner: "ACT-MEDIACHESTV-OPERATOR", LastTested: "2026-01-01", Findings: []string{"digest mismatch rejected"}},
		},
	}
}

// mediaChestBehavior is the part of the media chest fixture that its name says.
func mediaChestBehavior() model.AuthoredArchitecture {
	return model.AuthoredArchitecture{
		Flows: []model.Flow{{
			ID:                  "FLOW-MEDIACHESTV-SECURITY",
			Title:               "Secure update flow",
			Kind:                "security",
			Direction:           "inbound",
			Frequency:           "on-demand",
			SourceRef:           "ACT-MEDIACHESTV-OPERATOR",
			DestinationRef:      "FU-MEDIACHESTV-CORE",
			Protocol:            "https",
			Authentication:      "mfa",
			EncryptionInTransit: "tls1.3",
			IntegrityProtection: "digest",
			DataRefs:            []string{"DO-MEDIACHESTV-OCI-LOCK"},
			Threats:             []string{"TS-MEDIACHESTV-INPUT-SPOOF"},
			Entry:               []string{"verify"},
			Steps: []model.FlowStep{{
				ID:               "verify",
				Kind:             "security_check",
				FlowType:         "security",
				Ref:              "FU-MEDIACHESTV-CORE",
				BoundaryCrossing: true,
				TrustBoundaryRef: "TB-MEDIACHESTV-DEVICE",
			}},
		}},
		Machines: []model.Machine{{
			ID: "SM-MEDIACHESTV-DEPLOY", Owner: "FU-MEDIACHESTV-CORE", Initial: "STATE-MEDIACHESTV-IDLE",
			States: []model.MachineState{
				{ID: "STATE-MEDIACHESTV-IDLE", Name: "Idle"},
				{ID: "STATE-MEDIACHESTV-APPLYING", Name: "Applying"},
			},
			Transitions: []model.MachineTransition{{ID: "TR-MEDIACHESTV-APPLY", From: "STATE-MEDIACHESTV-IDLE", To: "STATE-MEDIACHESTV-APPLYING", On: "EVT-MEDIACHESTV-DEPLOY-REQUESTED"}},
		}},
		Events: []model.Event{{ID: "EVT-MEDIACHESTV-DEPLOY-REQUESTED", Name: "Deploy Requested"}},
	}
}

// mediaChestMappings is the part of the media chest fixture that its name says.
func mediaChestMappings() model.AuthoredArchitecture {
	return model.AuthoredArchitecture{
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
			{Type: "guarded_by", From: "STATE-MEDIACHESTV-APPLYING", To: "CTRL-MEDIACHESTV-DIGEST-PINNING"},
			{Type: "guarded_by", From: "STATE-MEDIACHESTV-IDLE", To: "TB-MEDIACHESTV-DEVICE"},
		},
	}
}

// mediaChestAuthored combines the parts of the media chest fixture.
func mediaChestAuthored() model.AuthoredArchitecture {
	a := mediaChestStructure()
	threats := mediaChestThreats()
	a.ThreatAssumptions, a.ThreatOutOfScope, a.ThreatScenarios = threats.ThreatAssumptions, threats.ThreatOutOfScope, threats.ThreatScenarios
	a.ThreatMitigations, a.ControlVerifications = threats.ThreatMitigations, threats.ControlVerifications
	behavior := mediaChestBehavior()
	a.Flows, a.Machines, a.Events = behavior.Flows, behavior.Machines, behavior.Events
	a.Mappings = mediaChestMappings().Mappings
	return a
}

// mediaChestViews are the views of the media chest fixture, one of each kind.
func mediaChestViews() []model.View {
	return []model.View{
		{ID: "VIEW-ARCHITECTURE-INTENT", Kind: "architecture-intent", Roots: []string{"FG-MEDIACHESTV-CORE"}},
		{ID: "VIEW-COMMUNICATION", Kind: "communication", Roots: []string{"FU-MEDIACHESTV-CORE"}, IncludeMappings: []string{"calls", "interacts_with"}},
		{ID: "VIEW-DEPLOYMENT", Kind: "deployment", Roots: []string{"FU-MEDIACHESTV-CORE"}, IncludeMappings: []string{"deployed_to", "bounded_by"}},
		{ID: "VIEW-SECURITY", Kind: "security", Roots: []string{"AV-MEDIACHESTV-INPUT-SPOOF"}},
		{ID: "VIEW-TRACEABILITY", Kind: "traceability", Roots: []string{"FU-MEDIACHESTV-CORE"}},
		{ID: "VIEW-STATE-LIFECYCLE", Kind: "state-lifecycle", Roots: []string{"STATE-MEDIACHESTV-IDLE"}, IncludeKinds: []string{"state", "event", "control", "trust_boundary"}, IncludeMappings: []string{"transition", "on_event", "guarded_by"}},
	}
}

// mediaChestBundle is a model with a view of every kind, a threat model and a state machine.
func mediaChestBundle(t *testing.T) model.Bundle {
	return model.Bundle{ArchitecturePath: filepath.Join(t.TempDir(), "engmod.yml"), Architecture: model.ArchitectureDocument{
		Model:                model.ModelMeta{ID: "m", Title: "m"},
		AuthoredArchitecture: mediaChestAuthored(),
		Views:                mediaChestViews(),
	}, Catalog: model.CatalogDocument{}}
}

// TRLC-LINKS: REQ-EMG-003
func TestGenerateAsciiDoc_RendersStateLifecycleAndNewAuthoredReferenceKinds(t *testing.T) {
	res := mustGenerateBundle(t, mediaChestBundle(t), Options{})
	about(t, res.Document).has(docCheck{"== State Lifecycle View", "missing State Lifecycle View chapter"})

	stateSection := sectionByHeading(res.Document, "== State Lifecycle View")
	about(t, stateSection).has(docCheck{"=== State Lifecycle Diagram", "state lifecycle section missing purpose-specific diagram"})
	about(t, stateSection).lacks(
		docCheck{"=== View-Scoped Projection", "state lifecycle section should not render generic view-scoped projection"},
		docCheck{"depends_on", "state lifecycle section should respect includeMappings and exclude depends_on"},
	)

	securitySection := sectionByHeading(res.Document, "== Security View")
	about(t, securitySection).hasAll("security section missing %s", "Threat Scenario Register", "Threat Mitigation Coverage", "Control Verification Status", "Security Flow Semantics", "Threat Assumptions", "Threat Model Out-of-Scope Decisions", "TS-MEDIACHESTV-INPUT-SPOOF", "MIT-MEDIACHESTV-DIGEST", "CV-MEDIACHESTV-DIGEST", "FLOW-MEDIACHESTV-SECURITY")
	about(t, securitySection).lacks(docCheck{"\n=== Control Verification Status", "control verification status should be nested under threat scenarios, not rendered as a standalone security section"})
	about(t, securitySection).has(docCheck{"==== Spoofed update input (TS-MEDIACHESTV-INPUT-SPOOF)", "threat scenario register should render one chapter per scenario"})
	scenarioSection := sectionByHeading(securitySection, "==== Spoofed update input (TS-MEDIACHESTV-INPUT-SPOOF)")
	about(t, scenarioSection).has(
		docCheck{"===== Control Verification Status", "threat scenario should include nested control verification status"},
		docCheck{"====== CV-MEDIACHESTV-DIGEST", "threat scenario should include nested control verification status"},
	)
	about(t, securitySection).hasNone("threat scenario register should not use wide table header %q", "|ID |Title |Attack Vector |Scope |Flows |Likelihood |Impact |Severity", "|ID |Status |Owner |Risk |Controls |Mitigations |Verifications |Operational Notes")
	requireReferenceEntries(t, res.Document, "IF-MEDIACHESTV-CONTROL-API", "DO-MEDIACHESTV-OCI-LOCK", "DEP-MEDIACHESTV-KAIROS-NODE", "CTRL-MEDIACHESTV-DIGEST-PINNING", "TB-MEDIACHESTV-DEVICE", "STATE-MEDIACHESTV-IDLE", "EVT-MEDIACHESTV-DEPLOY-REQUESTED")
}

// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
func TestGenerateAsciiDoc_InteractionFlowViewAndReferences(t *testing.T) {
	bundle := model.Bundle{ArchitecturePath: filepath.Join(t.TempDir(), "engmod.yml"), Architecture: model.ArchitectureDocument{
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

	res, err := Generate(Inputs{Bundle: schemaV2TestBundle(bundle), Requirements: schemaV2TestRequirements(model.RequirementsDocument{}), Design: schemaV2TestDesign(model.DesignDocument{})}, Options{})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if !strings.Contains(res.Document, "== Interaction Flow View") {
		t.Fatalf("missing Interaction Flow View chapter")
	}
	flowSection := sectionByHeading(res.Document, "== Interaction Flow View")
	if !strings.Contains(flowSection, "=== Interaction Flow Diagram") {
		t.Fatalf("interaction flow section missing purpose-specific diagram")
	}
	if strings.Contains(flowSection, "=== View-Scoped Projection") {
		t.Fatalf("interaction flow section should not render generic view-scoped projection")
	}
	for _, id := range []string{"FLOW-INPUT", "FLOW-INPUT::submit"} {
		block := referenceBlockByID(res.Document, id)
		if block == "" {
			t.Fatalf("missing reference index entry for %s", id)
		}
	}
}

// Generate resolves CODE references for a bundle a caller loaded with
// model.LoadBundle (engws check, MCP model.diagnose), not only through the
// package's own file loaders.
// Mutation: not attaching the code index in Generate fails this test.
// TRLC-LINKS: REQ-EMG-059, REQ-EMG-068, REQ-EMG-069
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-CODEMAP-INFERENCE
func TestGenerateAsciiDocResolvesCodeReferencesForLoadedBundle(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, "")
	dir := t.TempDir()
	testutil.WriteInterfaceModule(t, dir)
	testutil.WriteFile(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 3\nlintRun: {}\nrequirements:\n  - id: REQ-IF-001\n    text: Interface requirement.\n    sourceRefs: [CODE-SRC-IFACE-GO, CODE-SRC-GONE-GO]\n")
	testutil.WriteFile(t, filepath.Join(dir, "src", "iface.go"), "package iface\n\n// TRLC-LINKS: REQ-IF-001\nfunc Encode() {}\n")
	bundle, err := model.LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	result, _ := Generate(Inputs{Bundle: bundle, Requirements: bundle.Requirements, Design: bundle.Design}, Options{})
	testutil.AssertDiag(t, result.Diagnostics, testutil.DiagWant{Code: "model.invalid_requirement_source_ref", Fragment: "CODE-SRC-GONE-GO"})
	for _, d := range result.Diagnostics {
		if strings.Contains(d.Message, "CODE-SRC-IFACE-GO") {
			t.Fatalf("resolvable CODE reference rejected: %+v", d)
		}
	}
}

// TRLC-LINKS: REQ-EMG-010
func findCheckByEvidence(checks []codescan.VerificationCheck, evidence string) (codescan.VerificationCheck, bool) {
	for _, check := range checks {
		if slices.Contains(check.Evidence, evidence) {
			return check, true
		}
	}
	return codescan.VerificationCheck{}, false
}
