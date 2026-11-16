// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// baselineFile is the baseline file path, its content ("" writes no file) and the decision.
type baselineFile struct{ file, content, decision string }

// baselineBundle is a health bundle whose manifest lives in a temp directory
// holding the baseline file content ("" writes no file).
func baselineBundle(t *testing.T, f baselineFile) model.Bundle {
	t.Helper()
	dir := t.TempDir()
	if f.content != "" {
		if err := os.MkdirAll(filepath.Join(dir, "model"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "model", "baseline.json"), []byte(f.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b := healthBundle(&model.ManifestHealth{Baseline: &model.HealthBaseline{File: f.file, Decision: f.decision}})
	b.ManifestPath = filepath.Join(dir, "engmod.yml")
	return b
}

const validBaselineJSON = `{"schema":"engmod-health-baseline/1","entries":[{"path":"a.go","function":"f","code":"health.large_method","value":60}]}`

// A baseline with an accepted decision and a readable file raises nothing, and
// a model without a baseline is untouched by the rules (REQ-EMG-160).
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
func TestHealthBaselineValidIsSilent(t *testing.T) {
	b := baselineBundle(t, baselineFile{"model/baseline.json", validBaselineJSON, "ADR-OK"})
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
	cases := []baselineCase{
		{"decision missing", baselineFile{"model/baseline.json", validBaselineJSON, "ADR-NOPE"}, model.HealthWaiverUnjustifiedCode, "health.baseline.decision", "does not exist"},
		{"decision proposed", baselineFile{"model/baseline.json", validBaselineJSON, "ADR-DRAFT"}, model.HealthWaiverUnjustifiedCode, "health.baseline.decision", "not accepted"},
		{"decision empty", baselineFile{"model/baseline.json", validBaselineJSON, ""}, "model.invalid_health_baseline", "health.baseline.decision", "needs a decision"},
		{"file absolute", baselineFile{"/etc/baseline.json", validBaselineJSON, "ADR-OK"}, "model.invalid_health_baseline", "health.baseline.file", "is absolute"},
		{"file parent", baselineFile{"../baseline.json", validBaselineJSON, "ADR-OK"}, "model.invalid_health_baseline", "health.baseline.file", "leaves the manifest directory"},
		{"file empty", baselineFile{"", validBaselineJSON, "ADR-OK"}, "model.invalid_health_baseline", "health.baseline.file", "is empty"},
		{"file missing", baselineFile{"model/other.json", validBaselineJSON, "ADR-OK"}, model.HealthBaselineUnreadableCode, "health.baseline.file", "cannot be read"},
		{"file not json", baselineFile{"model/baseline.json", "entries: []", "ADR-OK"}, model.HealthBaselineUnreadableCode, "health.baseline.file", "not a valid baseline"},
		{"file other schema", baselineFile{"model/baseline.json", `{"schema":"engmod-health/1","entries":[]}`, "ADR-OK"}, model.HealthBaselineUnreadableCode, "health.baseline.file", "engmod-health-baseline/1"},
		{"file bad entry", baselineFile{"model/baseline.json", `{"schema":"engmod-health-baseline/1","entries":[{"path":"a.go","function":"f","code":"health.nope","value":5}]}`, "ADR-OK"}, model.HealthBaselineUnreadableCode, "health.baseline.file", "not a health smell code"},
		{"file entry without value", baselineFile{"model/baseline.json", `{"schema":"engmod-health-baseline/1","entries":[{"path":"a.go","function":"f","code":"health.large_method"}]}`, "ADR-OK"}, model.HealthBaselineUnreadableCode, "health.baseline.file", "value 0 must be at least 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { assertBaselineCase(t, c) })
	}
}

// baselineCase is a baseline with the one diagnostic it must report.
type baselineCase struct {
	name           string
	baseline       baselineFile
	code, path, in string
}

// assertBaselineCase checks that the baseline reports exactly its code at its path.
func assertBaselineCase(t *testing.T, c baselineCase) {
	t.Helper()
	got := healthCodes(baselineBundle(t, c.baseline))
	if len(got) != 1 {
		t.Fatalf("got %+v, want exactly one diagnostic", got)
	}
	d := got[0]
	if !isErrorAt(d, c.code, c.path) || !strings.Contains(d.Message, c.in) {
		t.Fatalf("got %+v, want %s at %s containing %q", d, c.code, c.path, c.in)
	}
	if c.code == model.HealthWaiverUnjustifiedCode && !strings.Contains(d.Message, "baseline") {
		t.Errorf("the message must name the baseline: %s", d.Message)
	}
}

// A bundle built in memory has no manifest directory, so the file is not read
// and only the manifest fields are checked.
// TRLC-LINKS: REQ-EMG-153
func TestHealthBaselineWithoutManifestPathSkipsTheFile(t *testing.T) {
	b := baselineBundle(t, baselineFile{"model/missing.json", "", "ADR-OK"})
	b.ManifestPath = ""
	if got := healthCodes(b); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

// health.waiver_unjustified for a baseline and health.baseline_unreadable are
// untiered errors: no tier, bundle or POA&M item deferral changes them.
// TRLC-LINKS: REQ-EMG-154, REQ-EMG-155
func TestHealthBaselineErrorsAreUntiered(t *testing.T) {
	assertHealthCodesUntiered(t, model.HealthWaiverUnjustifiedCode, model.HealthBaselineUnreadableCode)
	for _, code := range []string{model.HealthWaiverUnjustifiedCode, model.HealthBaselineUnreadableCode} {
		if IsCoreCode(code) {
			t.Errorf("%s must not be core", code)
		}
	}
	for _, tier := range model.Tiers {
		assertBaselineErrorsHold(t, tier)
	}
}

// assertBaselineErrorsHold checks that at the tier both baseline errors stay
// plain errors despite an open POA&M item that names health.*.
func assertBaselineErrorsHold(t *testing.T, tier string) {
	t.Helper()
	b := baselineBundle(t, baselineFile{"model/missing.json", "", "ADR-DRAFT"})
	b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier}
	b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: "POAM-1", Milestone: "tier:" + tier, Status: "open", Defers: []model.POAMDefer{{Code: "health.*"}}}}
	diags := ApplyTierPolicy(b, Bundle(b))
	for _, code := range []string{model.HealthWaiverUnjustifiedCode, model.HealthBaselineUnreadableCode} {
		found := diagnosticsWithCode(diags, code)
		if len(found) == 0 {
			t.Errorf("%s: %s not reported", tier, code)
		}
		assertPlainErrors(t, found, tier)
	}
}

// assertPlainErrors checks that the diagnostics are errors, not deferred, at the tier.
func assertPlainErrors(t *testing.T, diags []Diagnostic, tier string) {
	t.Helper()
	for _, d := range diags {
		if d.Severity != SeverityError || d.Deferred || d.PoamRef != "" {
			t.Errorf("%s at %s changed: %+v", d.Code, tier, d)
		}
	}
}
