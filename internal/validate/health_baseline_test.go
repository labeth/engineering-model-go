// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// baselineBundle is a health bundle whose manifest lives in a temp directory
// holding the given baseline file content ("" writes no file).
func baselineBundle(t *testing.T, file, content string, decision string) model.Bundle {
	t.Helper()
	dir := t.TempDir()
	if content != "" {
		if err := os.MkdirAll(filepath.Join(dir, "model"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "model", "baseline.json"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b := healthBundle(&model.ManifestHealth{Baseline: &model.HealthBaseline{File: file, Decision: decision}})
	b.ManifestPath = filepath.Join(dir, "engmod.yml")
	return b
}

const validBaselineJSON = `{"schema":"engmod-health-baseline/1","entries":[{"path":"a.go","function":"f","code":"health.large_method","value":60}]}`

// A baseline with an accepted decision and a readable file raises nothing, and
// a model without a baseline is untouched by the rules (REQ-EMG-160).
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
func TestHealthBaselineValidIsSilent(t *testing.T) {
	b := baselineBundle(t, "model/baseline.json", validBaselineJSON, "ADR-OK")
	if got := healthCodes(b); len(got) != 0 {
		t.Fatalf("a valid baseline got %+v", got)
	}
	if got := healthCodes(healthBundle(&model.ManifestHealth{})); len(got) != 0 {
		t.Fatalf("a block without baseline got %+v", got)
	}
}

// Each baseline error reports exactly its code at its path and names the
// baseline: a decision that does not exist or is only proposed is
// health.waiver_unjustified, a file path that is absolute, leaves the manifest
// directory or is empty is model.invalid_health_baseline, and a file that is
// missing, not JSON, of another schema or with an invalid entry is
// health.baseline_unreadable.
//
// Mutation: accepting a proposed decision, skipping the path rules or not
// reading the file fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154
func TestHealthBaselineErrors(t *testing.T) {
	cases := []struct {
		name, file, content, decision string
		code, path, in                string
	}{
		{"decision missing", "model/baseline.json", validBaselineJSON, "ADR-NOPE", model.HealthWaiverUnjustifiedCode, "health.baseline.decision", "does not exist"},
		{"decision proposed", "model/baseline.json", validBaselineJSON, "ADR-DRAFT", model.HealthWaiverUnjustifiedCode, "health.baseline.decision", "not accepted"},
		{"decision empty", "model/baseline.json", validBaselineJSON, "", "model.invalid_health_baseline", "health.baseline.decision", "needs a decision"},
		{"file absolute", "/etc/baseline.json", validBaselineJSON, "ADR-OK", "model.invalid_health_baseline", "health.baseline.file", "is absolute"},
		{"file parent", "../baseline.json", validBaselineJSON, "ADR-OK", "model.invalid_health_baseline", "health.baseline.file", "leaves the manifest directory"},
		{"file empty", "", validBaselineJSON, "ADR-OK", "model.invalid_health_baseline", "health.baseline.file", "is empty"},
		{"file missing", "model/other.json", validBaselineJSON, "ADR-OK", model.HealthBaselineUnreadableCode, "health.baseline.file", "cannot be read"},
		{"file not json", "model/baseline.json", "entries: []", "ADR-OK", model.HealthBaselineUnreadableCode, "health.baseline.file", "not a valid baseline"},
		{"file other schema", "model/baseline.json", `{"schema":"engmod-health/1","entries":[]}`, "ADR-OK", model.HealthBaselineUnreadableCode, "health.baseline.file", "engmod-health-baseline/1"},
		{"file bad entry", "model/baseline.json", `{"schema":"engmod-health-baseline/1","entries":[{"path":"a.go","function":"f","code":"health.nope","value":5}]}`, "ADR-OK", model.HealthBaselineUnreadableCode, "health.baseline.file", "not a health smell code"},
		{"file entry without value", "model/baseline.json", `{"schema":"engmod-health-baseline/1","entries":[{"path":"a.go","function":"f","code":"health.large_method"}]}`, "ADR-OK", model.HealthBaselineUnreadableCode, "health.baseline.file", "value 0 must be at least 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := healthCodes(baselineBundle(t, c.file, c.content, c.decision))
			if len(got) != 1 {
				t.Fatalf("got %+v, want exactly one diagnostic", got)
			}
			d := got[0]
			if d.Code != c.code || d.Path != c.path || d.Severity != SeverityError || !strings.Contains(d.Message, c.in) {
				t.Fatalf("got %+v, want %s at %s containing %q", d, c.code, c.path, c.in)
			}
			if c.code == model.HealthWaiverUnjustifiedCode && !strings.Contains(d.Message, "baseline") {
				t.Errorf("the message must name the baseline: %s", d.Message)
			}
		})
	}
}

// A bundle built in memory has no manifest directory, so the file is not read
// and only the manifest fields are checked.
// TRLC-LINKS: REQ-EMG-153
func TestHealthBaselineWithoutManifestPathSkipsTheFile(t *testing.T) {
	b := baselineBundle(t, "model/missing.json", "", "ADR-OK")
	b.ManifestPath = ""
	if got := healthCodes(b); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

// health.waiver_unjustified for a baseline and health.baseline_unreadable are
// untiered errors: no tier, bundle or POA&M item deferral changes them.
// TRLC-LINKS: REQ-EMG-154, REQ-EMG-155
func TestHealthBaselineErrorsAreUntiered(t *testing.T) {
	for _, code := range []string{model.HealthWaiverUnjustifiedCode, model.HealthBaselineUnreadableCode} {
		if TierKeyed(code) || IsCoreCode(code) {
			t.Errorf("%s must be neither tier keyed nor core", code)
		}
	}
	for _, tier := range model.Tiers {
		b := baselineBundle(t, "model/missing.json", "", "ADR-DRAFT")
		b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier}
		b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: "POAM-1", Milestone: "tier:" + tier, Status: "open", Defers: []model.POAMDefer{{Code: "health.*"}}}}
		seen := map[string]bool{}
		for _, d := range ApplyTierPolicy(b, Bundle(b)) {
			if d.Code == model.HealthWaiverUnjustifiedCode || d.Code == model.HealthBaselineUnreadableCode {
				seen[d.Code] = true
				if d.Severity != SeverityError || d.Deferred || d.PoamRef != "" {
					t.Errorf("%s at %s changed: %+v", d.Code, tier, d)
				}
			}
		}
		if len(seen) != 2 {
			t.Errorf("%s: saw %v", tier, seen)
		}
	}
}
