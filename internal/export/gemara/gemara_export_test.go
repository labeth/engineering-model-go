// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package gemara

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gemaraproj "github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/fetcher"

	"github.com/labeth/engineering-model-go/internal/model"
)

var gemaraExampleModels = []string{
	filepath.Join("examples", "payments-engineering-sample", "engmod.yml"),
	filepath.Join("examples", "bedrock-pr-review-github-app-sample", "engmod.yml"),
	filepath.Join("examples", "coffee-fleet-ota-cloud-sample", "engmod.yml"),
}

// TestGemaraArtifactsLoadThroughSDK generates every Gemara catalog for each
// example model and loads it back through the official go-gemara SDK loader,
// proving the output is structurally correct and SDK-consumable.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraArtifactsLoadThroughSDK(t *testing.T) {
	opts := Options{Version: "1.0.0", Date: "2026-06-26T00:00:00Z"}

	for _, modelPath := range gemaraExampleModels {
		modelPath := modelPath
		t.Run(filepath.Base(filepath.Dir(modelPath)), func(t *testing.T) {
			bundle, err := model.LoadBundle(modelPath)
			if err != nil {
				t.Fatalf("load bundle: %v", err)
			}
			res, err := Generate(bundle, opts)
			if err != nil {
				t.Fatalf("generate gemara: %v", err)
			}
			loader := sdkLoader{ctx: context.Background(), f: &fetcher.File{}, dir: writeAndDetectArtifacts(t, res)}
			cc := loadRequiredArtifacts(t, loader)
			loadOptionalArtifacts(t, loader)
			assertGeneratedInvariants(t, res, *cc)
			assertApplicabilityReachable(t, cc)
		})
	}
}

// sdkLoader locates the artifacts written for the SDK loader.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type sdkLoader struct {
	ctx context.Context
	f   *fetcher.File
	dir string
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (l sdkLoader) path(name string) string { return filepath.Join(l.dir, name+".yaml") }

// mustLoadSDK loads a required artifact through the SDK's typed loader.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func mustLoadSDK[T any](t *testing.T, l sdkLoader, name string) *T {
	t.Helper()
	v, err := gemaraproj.Load[T](l.ctx, l.f, l.path(name))
	if err != nil {
		t.Fatalf("SDK load %s: %v", name, err)
	}
	return v
}

// sdkLoadFn is the typed SDK load of an optional artifact.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sdkLoadFn[T any](l sdkLoader) func(string) error {
	return func(p string) error {
		_, err := gemaraproj.Load[T](l.ctx, l.f, p)
		return err
	}
}

// wantArtifactTypes is the metadata.type discriminator of every artifact.
var wantArtifactTypes = map[string]string{
	"vector-catalog":         "VectorCatalog",
	"capability-catalog":     "CapabilityCatalog",
	"control-catalog":        "ControlCatalog",
	"threat-catalog":         "ThreatCatalog",
	"risk-catalog":           "RiskCatalog",
	"principle-catalog":      "PrincipleCatalog",
	"guidance-catalog":       "GuidanceCatalog",
	"policy":                 "Policy",
	"lexicon":                "Lexicon",
	"control-threat-mapping": "MappingDocument",
	"audit-log":              "AuditLog",
	"enforcement-log":        "EnforcementLog",
}

// writeAndDetectArtifacts writes every artifact to a temp dir and checks that
// metadata.type discrimination matches; it returns the directory.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func writeAndDetectArtifacts(t *testing.T, res Result) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range res.YAML {
		if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		at, err := gemaraproj.DetectType([]byte(content))
		if err != nil {
			t.Fatalf("%s: DetectType: %v", name, err)
		}
		if at.String() != wantArtifactTypes[name] {
			t.Fatalf("%s: DetectType = %q, want %q", name, at.String(), wantArtifactTypes[name])
		}
	}
	return dir
}

// loadRequiredArtifacts loads each always-present artifact back through the
// SDK's typed loader and returns the control catalog.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func loadRequiredArtifacts(t *testing.T, l sdkLoader) *gemaraproj.ControlCatalog {
	t.Helper()
	mustLoadSDK[gemaraproj.VectorCatalog](t, l, "vector-catalog")
	mustLoadSDK[gemaraproj.CapabilityCatalog](t, l, "capability-catalog")
	mustLoadSDK[gemaraproj.ThreatCatalog](t, l, "threat-catalog")
	mustLoadSDK[gemaraproj.RiskCatalog](t, l, "risk-catalog")
	return mustLoadSDK[gemaraproj.ControlCatalog](t, l, "control-catalog")
}

// loadOptionalArtifacts round-trips the remaining artifacts through the SDK
// loader when present.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func loadOptionalArtifacts(t *testing.T, l sdkLoader) {
	t.Helper()
	optional := []struct {
		name string
		load func(string) error
	}{
		{"principle-catalog", sdkLoadFn[gemaraproj.PrincipleCatalog](l)},
		{"guidance-catalog", sdkLoadFn[gemaraproj.GuidanceCatalog](l)},
		{"policy", sdkLoadFn[gemaraproj.Policy](l)},
		{"lexicon", sdkLoadFn[gemaraproj.Lexicon](l)},
		{"control-threat-mapping", sdkLoadFn[gemaraproj.MappingDocument](l)},
		{"audit-log", sdkLoadFn[gemaraproj.AuditLog](l)},
		{"enforcement-log", sdkLoadFn[gemaraproj.EnforcementLog](l)},
	}
	for _, o := range optional {
		if _, statErr := os.Stat(l.path(o.name)); statErr != nil {
			continue
		}
		if err := o.load(l.path(o.name)); err != nil {
			t.Fatalf("SDK load %s: %v", o.name, err)
		}
	}
}

// assertGeneratedInvariants checks the structural invariants of the catalogs.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertGeneratedInvariants(t *testing.T, res Result, cc gemaraproj.ControlCatalog) {
	t.Helper()
	assertControlCatalogInvariants(t, cc)
	assertThreatCatalogInvariants(t, res.ThreatCatalog)
	assertRiskCatalogInvariants(t, res.RiskCatalog)
	assertCapabilityCatalogInvariants(t, res.CapabilityCatalog)
	if res.HasAudit {
		assertEvidencePayloads(t, res.AuditLog.Results)
	}
}

// assertApplicabilityReachable exercises an SDK consumer helper on the
// round-tripped catalog: every assessment requirement must be reachable via
// some applicability group.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertApplicabilityReachable(t *testing.T, cc *gemaraproj.ControlCatalog) {
	t.Helper()
	if len(cc.Controls) == 0 {
		return
	}
	sugar := cc.Sugar()
	total := 0
	for _, g := range cc.Metadata.ApplicabilityGroups {
		total += len(sugar.GetRequirementForApplicability(g.Id))
	}
	if total == 0 {
		t.Fatalf("no assessment requirements reachable via any applicability group")
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraDerivesMeaningfulAuthoredDate(t *testing.T) {
	res, err := GenerateFromFile("engmod.yml", Options{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("generate gemara: %v", err)
	}
	for name, content := range res.YAML {
		if strings.Contains(content, "1970-01-01T00:00:00Z") {
			t.Errorf("%s contains epoch placeholder instead of an authored date", name)
		}
	}
}

// TestGemaraEvaluationLogLoadsThroughSDK validates the L5 Evaluation Log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraEvaluationLogLoadsThroughSDK(t *testing.T) {
	opts := Options{Version: "1.0.0", Date: "2026-06-26T00:00:00Z"}
	for _, modelPath := range gemaraExampleModels {
		modelPath := modelPath
		t.Run(filepath.Base(filepath.Dir(modelPath)), func(t *testing.T) {
			res := generateExampleEvaluationLog(t, modelPath, opts)
			if !res.HasContent {
				t.Skip("no controls to evaluate")
			}
			// The Evaluation Log is validated by the SDK's type discriminator and by
			// cue vet against the schema (scripts/validate-gemara.sh). It is not
			// round-tripped through gemaraproj.Load because AssessmentStep is a func type
			// that serializes to a name string but cannot be unmarshaled back; the
			// SDK consumes evaluation logs in-memory (e.g. for OSCAL conversion).
			at, err := gemaraproj.DetectType([]byte(res.YAML))
			if err != nil || at.String() != "EvaluationLog" {
				t.Fatalf("DetectType = %q (err %v), want EvaluationLog", at.String(), err)
			}
			for _, ce := range res.EvaluationLog.Evaluations {
				assertAssessmentLogsMatchControl(t, ce)
			}
		})
	}
}

// generateExampleEvaluationLog loads an example model and renders its Evaluation Log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func generateExampleEvaluationLog(t *testing.T, modelPath string, opts Options) EvaluationResult {
	t.Helper()
	bundle, err := model.LoadBundle(modelPath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	res, err := GenerateEvaluationLog(bundle, bundle.Requirements, EvalRun{CodeRoot: "", Options: opts})
	if err != nil {
		t.Fatalf("generate evaluation log: %v", err)
	}
	return res
}

// assertAssessmentLogsMatchControl checks that assessment-logs reference the
// same control as their evaluation and carry evidence payloads.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertAssessmentLogsMatchControl(t *testing.T, ce *gemaraproj.ControlEvaluation) {
	t.Helper()
	for _, al := range ce.AssessmentLogs {
		if al.Requirement.ReferenceId != ce.Control.ReferenceId {
			t.Fatalf("assessment-log requirement ref %q != control ref %q", al.Requirement.ReferenceId, ce.Control.ReferenceId)
		}
		assertPayloads(t, "assessment-log", al.Evidence)
	}
}

// assertPayloads requires every evidence entry to carry a payload.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertPayloads(t *testing.T, kind string, evidence []gemaraproj.Evidence) {
	t.Helper()
	for _, e := range evidence {
		if e.Payload == nil {
			t.Fatalf("%s evidence %q missing payload", kind, e.Id)
		}
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertEvidencePayloads(t *testing.T, results []*gemaraproj.AuditResult) {
	t.Helper()
	for _, result := range results {
		assertPayloads(t, "audit", result.Evidence)
	}
}

// controlCatalogIDs are the group and applicability ids a control may reference.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type controlCatalogIDs struct {
	groups map[string]bool
	apps   map[string]bool
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertControlCatalogInvariants(t *testing.T, c gemaraproj.ControlCatalog) {
	t.Helper()
	ids := controlCatalogIDs{groups: map[string]bool{}, apps: map[string]bool{}}
	for _, g := range c.Groups {
		ids.groups[g.Id] = true
	}
	for _, g := range c.Metadata.ApplicabilityGroups {
		ids.apps[g.Id] = true
	}
	seen := map[string]bool{}
	for _, ctrl := range c.Controls {
		if seen[ctrl.Id] {
			t.Fatalf("duplicate control id %q", ctrl.Id)
		}
		seen[ctrl.Id] = true
		assertControlInvariants(t, ctrl, ids)
	}
}

// assertControlInvariants checks one control against the catalog's ids.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertControlInvariants(t *testing.T, ctrl gemaraproj.Control, ids controlCatalogIDs) {
	t.Helper()
	if ctrl.Objective == "" {
		t.Fatalf("control %q missing objective", ctrl.Id)
	}
	if !ids.groups[ctrl.Group] {
		t.Fatalf("control %q references unknown group %q", ctrl.Id, ctrl.Group)
	}
	if len(ctrl.AssessmentRequirements) == 0 {
		t.Fatalf("control %q has no assessment requirements", ctrl.Id)
	}
	for _, ar := range ctrl.AssessmentRequirements {
		assertRequirementApplicability(t, ar, ids.apps)
	}
}

// assertRequirementApplicability checks that a requirement names known applicability groups.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertRequirementApplicability(t *testing.T, ar gemaraproj.AssessmentRequirement, apps map[string]bool) {
	t.Helper()
	if len(ar.Applicability) == 0 {
		t.Fatalf("assessment requirement %q has empty applicability", ar.Id)
	}
	for _, a := range ar.Applicability {
		if !apps[a] {
			t.Fatalf("assessment requirement %q references unknown applicability group %q", ar.Id, a)
		}
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertThreatCatalogInvariants(t *testing.T, c gemaraproj.ThreatCatalog) {
	t.Helper()
	groupIDs := map[string]bool{}
	for _, g := range c.Groups {
		groupIDs[g.Id] = true
	}
	for _, th := range c.Threats {
		if !groupIDs[th.Group] {
			t.Fatalf("threat %q references unknown group %q", th.Id, th.Group)
		}
		if len(th.Capabilities) == 0 {
			t.Fatalf("threat %q has no capabilities (required)", th.Id)
		}
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertRiskCatalogInvariants(t *testing.T, c gemaraproj.RiskCatalog) {
	t.Helper()
	groupIDs := map[string]bool{}
	for _, g := range c.Groups {
		groupIDs[g.Id] = true
	}
	for _, r := range c.Risks {
		if !groupIDs[r.Group] {
			t.Fatalf("risk %q references unknown category %q", r.Id, r.Group)
		}
		if r.Severity.String() == "" || r.Severity == gemaraproj.InvalidSeverity {
			t.Fatalf("risk %q has invalid severity", r.Id)
		}
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertCapabilityCatalogInvariants(t *testing.T, c gemaraproj.CapabilityCatalog) {
	t.Helper()
	groupIDs := map[string]bool{}
	for _, g := range c.Groups {
		groupIDs[g.Id] = true
	}
	for _, cap := range c.Capabilities {
		if !groupIDs[cap.Group] {
			t.Fatalf("capability %q references unknown group %q", cap.Id, cap.Group)
		}
	}
}
