// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package scan

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

const baselineManifest = "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n  baseline: {file: model/health-baseline.json, decision: ADR-DEBT}\n"

// writeBaseline writes the baseline file of the fixture.
func writeBaseline(t *testing.T, dir string, entries ...model.HealthBaselineEntry) {
	t.Helper()
	data, err := model.EncodeHealthBaseline(entries)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(dir, "model", "health-baseline.json"), string(data))
}

// bigEntry is the baseline entry of the complex function of the fixture.
func bigEntry() model.HealthBaselineEntry {
	return model.HealthBaselineEntry{Path: "pkg/big.go", Function: "Big", Code: health.CodeComplexMethod, Value: 17}
}

// A scan applies the baseline of the manifest: the baselined error is listed
// under baselined and is not an error of the summary, the file keeps its score
// (the finding still counts), nothing is stale, and scanning at the tier with
// the baseline ignored gives the report of a model without a baseline.
//
// Mutation: dropping the finding from the function (score 10.0) or counting it
// as an error fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155, REQ-EMG-160
func TestScanAppliesTheBaseline(t *testing.T) {
	dir, manifest := fixture(t, baselineManifest)
	writeBaseline(t, dir, bigEntry())
	r, err := Scan(Options{ManifestPath: manifest})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Baselined) != 1 || r.Baselined[0].Path != "pkg/big.go" || r.Baselined[0].Function != "Big" || r.Baselined[0].Code != health.CodeComplexMethod {
		t.Fatalf("baselined = %+v", r.Baselined)
	}
	if r.Summary.Errors != 0 || r.Summary.Baselined != 1 || r.Summary.Findings != 1 || len(r.Diagnostics) != 0 {
		t.Errorf("summary = %+v diagnostics %+v, want no error and one baselined finding", r.Summary, r.Diagnostics)
	}
	if r.Files[0].Path != "pkg/big.go" || r.Files[0].Score != 8.0 || len(findingsOf(r, "pkg/big.go", "Big")) != 1 {
		t.Errorf("big.go = %v with %d findings, want 8.0: a baselined finding still counts", r.Files[0].Score, len(findingsOf(r, "pkg/big.go", "Big")))
	}

	_, plainManifest := fixture(t, "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n")
	plain, err := Scan(Options{ManifestPath: plainManifest})
	if err != nil {
		t.Fatal(err)
	}
	ignored, err := Scan(Options{ManifestPath: manifest, IgnoreBaseline: true})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := health.Encode(plain)
	b, _ := health.Encode(ignored)
	if string(a) != string(b) {
		t.Errorf("ignoring the baseline must give the report of a model without one:\n%s\n%s", a, b)
	}
}

// A baseline entry that matches no finding is a stale diagnostic, a warning at
// product (an error at operational), and a scan restricted to some paths
// reports stale only for entries inside them: the entry of big.go is not stale
// when only clean.go is scanned, an entry of clean.go is.
//
// TRLC-LINKS: REQ-EMG-155
func TestScanReportsStaleEntriesOnlyInScope(t *testing.T) {
	dir, manifest := fixture(t, baselineManifest)
	stale := model.HealthBaselineEntry{Path: "pkg/clean.go", Function: "Clean", Code: health.CodeLargeMethod, Value: 60}
	writeBaseline(t, dir, bigEntry(), stale)

	r, err := Scan(Options{ManifestPath: manifest})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != model.HealthBaselineStaleCode || r.Diagnostics[0].Level != health.LevelWarning ||
		r.Diagnostics[0].Path != "pkg/clean.go" || r.Summary.Errors != 0 || r.Summary.Warnings != 1 {
		t.Errorf("diagnostics %+v summary %+v", r.Diagnostics, r.Summary)
	}

	op, err := Scan(Options{ManifestPath: manifest, Tier: "operational"})
	if err != nil {
		t.Fatal(err)
	}
	if len(op.Diagnostics) != 1 || op.Diagnostics[0].Level != health.LevelError {
		t.Errorf("operational diagnostics = %+v, want an error", op.Diagnostics)
	}

	onlyClean := Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg", "clean.go")}}
	r, err = Scan(onlyClean)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Baselined) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Path != "pkg/clean.go" {
		t.Errorf("restricted to clean.go: baselined %+v diagnostics %+v", r.Baselined, r.Diagnostics)
	}

	writeBaseline(t, dir, bigEntry())
	r, err = Scan(onlyClean)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Diagnostics) != 0 || len(r.Baselined) != 0 {
		t.Errorf("an entry outside the scan must not be reported: %+v %+v", r.Diagnostics, r.Baselined)
	}
}

// A baseline that is named but missing, not JSON, of another schema or that
// leaves the manifest directory is a setup error (exit 2 in enghealth), never a
// clean scan.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func TestScanBaselineFileProblemsAreSetupErrors(t *testing.T) {
	dir, manifest := fixture(t, baselineManifest)
	if _, err := Scan(Options{ManifestPath: manifest}); !errors.Is(err, ErrSetup) {
		t.Errorf("missing file: err = %v, want ErrSetup", err)
	}
	for name, content := range map[string]string{"not json": "entries:", "other schema": `{"schema":"engmod-health/1","entries":[]}`} {
		testutil.WriteFile(t, filepath.Join(dir, "model", "health-baseline.json"), content)
		if _, err := Scan(Options{ManifestPath: manifest}); !errors.Is(err, ErrSetup) {
			t.Errorf("%s: err = %v, want ErrSetup", name, err)
		}
	}
	if _, err := Scan(Options{ManifestPath: manifest, IgnoreBaseline: true}); err != nil {
		t.Errorf("ignoring the baseline must not read it: %v", err)
	}
	_, outside := fixture(t, "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n  baseline: {file: ../x.json, decision: ADR-DEBT}\n")
	if _, err := Scan(Options{ManifestPath: outside}); !errors.Is(err, ErrSetup) {
		t.Errorf("a file outside the manifest directory: err = %v, want ErrSetup", err)
	}
}

// A baseline whose decision does not exist or is not accepted suppresses
// nothing: the scan does not read the file, the finding stays an ordinary error
// at product and a health.waiver_unjustified diagnostic (an error at every
// tier, naming the baseline) is reported and counted, so scan --fail-on-error
// fails. With an accepted decision the same baseline suppresses the finding.
//
// Mutation: loading the baseline without asking for the decision, or giving
// the diagnostic a warning level, fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-155
func TestScanBaselineNeedsAnAcceptedDecision(t *testing.T) {
	for name, decision := range map[string]string{"missing decision": "ADR-NOPE", "proposed decision": "ADR-DRAFT"} {
		manifest := "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n  baseline: {file: model/health-baseline.json, decision: " + decision + "}\n"
		dir, path := fixture(t, manifest)
		writeBaseline(t, dir, bigEntry())
		r, err := Scan(Options{ManifestPath: path})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(r.Baselined) != 0 || r.Summary.Baselined != 0 || r.Summary.Errors != 2 {
			t.Errorf("%s: baselined %d, errors %d; want nothing suppressed and the finding plus the diagnostic as errors", name, len(r.Baselined), r.Summary.Errors)
		}
		var found []health.Diagnostic
		for _, d := range r.Diagnostics {
			if d.Code == model.HealthWaiverUnjustifiedCode {
				found = append(found, d)
			}
		}
		if len(found) != 1 || found[0].Level != health.LevelError || !strings.Contains(found[0].Message, "model/health-baseline.json") ||
			!strings.Contains(found[0].Message, decision) || !strings.Contains(found[0].Message, "suppresses nothing") {
			t.Errorf("%s: diagnostics = %+v", name, r.Diagnostics)
		}
		// The diagnostic is an error at every tier, even a warning-only one.
		low, err := Scan(Options{ManifestPath: path, Tier: "poc"})
		if err != nil || low.Summary.Errors != 1 {
			t.Errorf("%s: at poc errors = %d (%v), want the diagnostic only", name, low.Summary.Errors, err)
		}
		// Ignoring the baseline adds no diagnostic.
		ignored, err := Scan(Options{ManifestPath: path, IgnoreBaseline: true})
		if err != nil || len(ignored.Diagnostics) != 0 {
			t.Errorf("%s: ignored baseline: %+v %v", name, ignored.Diagnostics, err)
		}
	}
	// An unreadable file with an unaccepted decision is reported by validation, not
	// read by the scan.
	_, path := fixture(t, "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n  baseline: {file: model/absent.json, decision: ADR-DRAFT}\n")
	if r, err := Scan(Options{ManifestPath: path}); err != nil || r.Summary.Baselined != 0 {
		t.Errorf("a baseline with an unaccepted decision is not read: %v %+v", err, r.Summary)
	}
}

// An entry whose recorded value is below the current value does not baseline
// the finding: the function that grew counts as an error at product, the entry
// is not stale, and the message says what the baseline accepted. An entry at or
// above the current value baselines it.
//
// Mutation: ignoring Value in the comparison fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
func TestScanDebtThatGrewPastTheBaselineIsAnError(t *testing.T) {
	dir, manifest := fixture(t, baselineManifest)
	grown := bigEntry()
	grown.Value = 16
	writeBaseline(t, dir, grown)
	r, err := Scan(Options{ManifestPath: manifest})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Baselined) != 0 || r.Summary.Errors != 1 || r.Summary.Baselined != 0 || len(r.Diagnostics) != 0 {
		t.Fatalf("grown: baselined %+v summary %+v diagnostics %+v", r.Baselined, r.Summary, r.Diagnostics)
	}
	fs := findingsOf(r, "pkg/big.go", "Big")
	if len(fs) != 1 || !strings.Contains(fs[0].Message, "the baseline accepts cyclomatic 16, now 17") {
		t.Errorf("findings = %+v", fs)
	}
	grown.Value = 17
	writeBaseline(t, dir, grown)
	if r, err = Scan(Options{ManifestPath: manifest}); err != nil || len(r.Baselined) != 1 || r.Summary.Errors != 0 {
		t.Errorf("equal value: %v baselined %+v summary %+v", err, r.Baselined, r.Summary)
	}
}
