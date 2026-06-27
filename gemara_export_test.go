// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	gemara "github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/fetcher"

	"github.com/labeth/engineering-model-go/model"
)

var gemaraExampleModels = []string{
	filepath.Join("examples", "payments-engineering-sample", "architecture.yml"),
	filepath.Join("examples", "bedrock-pr-review-github-app-sample", "architecture.yml"),
	filepath.Join("examples", "coffee-fleet-ota-cloud-sample", "architecture.yml"),
}

// TestGemaraArtifactsLoadThroughSDK generates every Gemara catalog for each
// example model and loads it back through the official go-gemara SDK loader,
// proving the output is structurally correct and SDK-consumable.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraArtifactsLoadThroughSDK(t *testing.T) {
	opts := GemaraExportOptions{Version: "1.0.0", Date: "2026-06-26T00:00:00Z"}

	for _, modelPath := range gemaraExampleModels {
		modelPath := modelPath
		t.Run(filepath.Base(filepath.Dir(modelPath)), func(t *testing.T) {
			bundle, err := model.LoadBundle(modelPath)
			if err != nil {
				t.Fatalf("load bundle: %v", err)
			}
			res, err := GenerateGemara(bundle, opts)
			if err != nil {
				t.Fatalf("generate gemara: %v", err)
			}

			dir := t.TempDir()
			f := &fetcher.File{}
			ctx := context.Background()

			// metadata.type discrimination must match for every artifact.
			wantType := map[string]string{
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
			for name, content := range res.YAML {
				path := filepath.Join(dir, name+".yaml")
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("write %s: %v", name, err)
				}
				at, err := gemara.DetectType([]byte(content))
				if err != nil {
					t.Fatalf("%s: DetectType: %v", name, err)
				}
				if at.String() != wantType[name] {
					t.Fatalf("%s: DetectType = %q, want %q", name, at.String(), wantType[name])
				}
			}

			// Load each artifact back through the SDK's typed loader.
			if _, err := gemara.Load[gemara.VectorCatalog](ctx, f, filepath.Join(dir, "vector-catalog.yaml")); err != nil {
				t.Fatalf("SDK load vector catalog: %v", err)
			}
			if _, err := gemara.Load[gemara.CapabilityCatalog](ctx, f, filepath.Join(dir, "capability-catalog.yaml")); err != nil {
				t.Fatalf("SDK load capability catalog: %v", err)
			}
			if _, err := gemara.Load[gemara.ThreatCatalog](ctx, f, filepath.Join(dir, "threat-catalog.yaml")); err != nil {
				t.Fatalf("SDK load threat catalog: %v", err)
			}
			if _, err := gemara.Load[gemara.RiskCatalog](ctx, f, filepath.Join(dir, "risk-catalog.yaml")); err != nil {
				t.Fatalf("SDK load risk catalog: %v", err)
			}
			cc, err := gemara.Load[gemara.ControlCatalog](ctx, f, filepath.Join(dir, "control-catalog.yaml"))
			if err != nil {
				t.Fatalf("SDK load control catalog: %v", err)
			}

			// Round-trip the remaining artifacts through the SDK loader when present.
			loadIfPresent := func(name string, load func(string) error) {
				p := filepath.Join(dir, name+".yaml")
				if _, statErr := os.Stat(p); statErr == nil {
					if err := load(p); err != nil {
						t.Fatalf("SDK load %s: %v", name, err)
					}
				}
			}
			loadIfPresent("principle-catalog", func(p string) error { _, e := gemara.Load[gemara.PrincipleCatalog](ctx, f, p); return e })
			loadIfPresent("guidance-catalog", func(p string) error { _, e := gemara.Load[gemara.GuidanceCatalog](ctx, f, p); return e })
			loadIfPresent("policy", func(p string) error { _, e := gemara.Load[gemara.Policy](ctx, f, p); return e })
			loadIfPresent("lexicon", func(p string) error { _, e := gemara.Load[gemara.Lexicon](ctx, f, p); return e })
			loadIfPresent("control-threat-mapping", func(p string) error { _, e := gemara.Load[gemara.MappingDocument](ctx, f, p); return e })
			loadIfPresent("audit-log", func(p string) error { _, e := gemara.Load[gemara.AuditLog](ctx, f, p); return e })
			loadIfPresent("enforcement-log", func(p string) error { _, e := gemara.Load[gemara.EnforcementLog](ctx, f, p); return e })

			assertControlCatalogInvariants(t, *cc)
			assertThreatCatalogInvariants(t, res.ThreatCatalog)
			assertRiskCatalogInvariants(t, res.RiskCatalog)
			assertCapabilityCatalogInvariants(t, res.CapabilityCatalog)

			// Exercise an SDK consumer helper on the round-tripped catalog: every
			// assessment requirement must be reachable via some applicability group.
			if len(cc.Controls) > 0 {
				sugar := cc.Sugar()
				total := 0
				for _, g := range cc.Metadata.ApplicabilityGroups {
					total += len(sugar.GetRequirementForApplicability(g.Id))
				}
				if total == 0 {
					t.Fatalf("no assessment requirements reachable via any applicability group")
				}
			}
		})
	}
}

// TestGemaraEvaluationLogLoadsThroughSDK validates the L5 Evaluation Log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraEvaluationLogLoadsThroughSDK(t *testing.T) {
	opts := GemaraExportOptions{Version: "1.0.0", Date: "2026-06-26T00:00:00Z"}
	for _, modelPath := range gemaraExampleModels {
		modelPath := modelPath
		t.Run(filepath.Base(filepath.Dir(modelPath)), func(t *testing.T) {
			bundle, err := model.LoadBundle(modelPath)
			if err != nil {
				t.Fatalf("load bundle: %v", err)
			}
			res, err := GenerateGemaraEvaluationLog(bundle, model.RequirementsDocument{}, "", opts)
			if err != nil {
				t.Fatalf("generate evaluation log: %v", err)
			}
			if !res.HasContent {
				t.Skip("no controls to evaluate")
			}
			// The Evaluation Log is validated by the SDK's type discriminator and by
			// cue vet against the schema (scripts/validate-gemara.sh). It is not
			// round-tripped through gemara.Load because AssessmentStep is a func type
			// that serializes to a name string but cannot be unmarshaled back; the
			// SDK consumes evaluation logs in-memory (e.g. for OSCAL conversion).
			at, err := gemara.DetectType([]byte(res.YAML))
			if err != nil || at.String() != "EvaluationLog" {
				t.Fatalf("DetectType = %q (err %v), want EvaluationLog", at.String(), err)
			}
			// assessment-logs must reference the same control as their evaluation.
			for _, ce := range res.EvaluationLog.Evaluations {
				for _, al := range ce.AssessmentLogs {
					if al.Requirement.ReferenceId != ce.Control.ReferenceId {
						t.Fatalf("assessment-log requirement ref %q != control ref %q", al.Requirement.ReferenceId, ce.Control.ReferenceId)
					}
				}
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertControlCatalogInvariants(t *testing.T, c gemara.ControlCatalog) {
	t.Helper()
	groupIDs := map[string]bool{}
	for _, g := range c.Groups {
		groupIDs[g.Id] = true
	}
	appIDs := map[string]bool{}
	for _, g := range c.Metadata.ApplicabilityGroups {
		appIDs[g.Id] = true
	}
	seen := map[string]bool{}
	for _, ctrl := range c.Controls {
		if seen[ctrl.Id] {
			t.Fatalf("duplicate control id %q", ctrl.Id)
		}
		seen[ctrl.Id] = true
		if ctrl.Objective == "" {
			t.Fatalf("control %q missing objective", ctrl.Id)
		}
		if !groupIDs[ctrl.Group] {
			t.Fatalf("control %q references unknown group %q", ctrl.Id, ctrl.Group)
		}
		if len(ctrl.AssessmentRequirements) == 0 {
			t.Fatalf("control %q has no assessment requirements", ctrl.Id)
		}
		for _, ar := range ctrl.AssessmentRequirements {
			if len(ar.Applicability) == 0 {
				t.Fatalf("assessment requirement %q has empty applicability", ar.Id)
			}
			for _, a := range ar.Applicability {
				if !appIDs[a] {
					t.Fatalf("assessment requirement %q references unknown applicability group %q", ar.Id, a)
				}
			}
		}
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertThreatCatalogInvariants(t *testing.T, c gemara.ThreatCatalog) {
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
func assertRiskCatalogInvariants(t *testing.T, c gemara.RiskCatalog) {
	t.Helper()
	groupIDs := map[string]bool{}
	for _, g := range c.Groups {
		groupIDs[g.Id] = true
	}
	for _, r := range c.Risks {
		if !groupIDs[r.Group] {
			t.Fatalf("risk %q references unknown category %q", r.Id, r.Group)
		}
		if r.Severity.String() == "" || r.Severity == gemara.InvalidSeverity {
			t.Fatalf("risk %q has invalid severity", r.Id)
		}
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func assertCapabilityCatalogInvariants(t *testing.T, c gemara.CapabilityCatalog) {
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
