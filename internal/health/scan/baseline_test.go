// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package scan

import (
	"errors"
	"path/filepath"
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
	r := mustScan(t, Options{ManifestPath: manifest})
	mustEqual(t, len(r.Baselined), 1)
	wantEqual(t, [3]string{r.Baselined[0].Path, r.Baselined[0].Function, r.Baselined[0].Code}, [3]string{"pkg/big.go", "Big", health.CodeComplexMethod})
	// No error, one baselined finding and nothing stale.
	wantEqual(t, [4]int{r.Summary.Errors, r.Summary.Baselined, r.Summary.Findings, len(r.Diagnostics)}, [4]int{0, 1, 1, 0})
	// A baselined finding still counts in the score.
	wantEqual(t, [3]any{r.Files[0].Path, r.Files[0].Score, len(findingsOf(r, "pkg/big.go", "Big"))}, [3]any{"pkg/big.go", 8.0, 1})

	_, plainManifest := fixture(t, "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n")
	plain := mustEncode(t, mustScan(t, Options{ManifestPath: plainManifest}))
	ignored := mustEncode(t, mustScan(t, Options{ManifestPath: manifest, IgnoreBaseline: true}))
	wantEqual(t, string(ignored), string(plain)) // ignoring the baseline gives the report of a model without one
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

	r := mustScan(t, Options{ManifestPath: manifest})
	mustEqual(t, len(r.Diagnostics), 1)
	d := r.Diagnostics[0]
	wantEqual(t, [5]any{d.Code, d.Level, d.Path, r.Summary.Errors, r.Summary.Warnings}, [5]any{model.HealthBaselineStaleCode, health.LevelWarning, "pkg/clean.go", 0, 1})

	op := mustScan(t, Options{ManifestPath: manifest, Tier: "operational"})
	mustEqual(t, len(op.Diagnostics), 1)
	wantEqual(t, op.Diagnostics[0].Level, health.LevelError)

	onlyClean := Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg", "clean.go")}}
	r = mustScan(t, onlyClean)
	mustEqual(t, len(r.Diagnostics), 1)
	wantEqual(t, [2]any{len(r.Baselined), r.Diagnostics[0].Path}, [2]any{0, "pkg/clean.go"})

	writeBaseline(t, dir, bigEntry())
	r = mustScan(t, onlyClean)
	wantEqual(t, [2]int{len(r.Diagnostics), len(r.Baselined)}, [2]int{0, 0}) // an entry outside the scan is not reported
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
		t.Run(name, func(t *testing.T) { requireUnjustifiedBaseline(t, decision) })
	}
	// An unreadable file with an unaccepted decision is reported by validation, not
	// read by the scan.
	_, path := fixture(t, "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n  baseline: {file: model/absent.json, decision: ADR-DRAFT}\n")
	wantEqual(t, mustScan(t, Options{ManifestPath: path}).Summary.Baselined, 0)
}

// requireUnjustifiedBaseline checks the scan of a baseline with a decision that
// is missing or not accepted: nothing is suppressed, the finding and the
// diagnostic are errors, the diagnostic is an error at every tier and ignoring
// the baseline adds none.
func requireUnjustifiedBaseline(t *testing.T, decision string) {
	t.Helper()
	manifest := "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n  floor: 8.0\n  baseline: {file: model/health-baseline.json, decision: " + decision + "}\n"
	dir, path := fixture(t, manifest)
	writeBaseline(t, dir, bigEntry())
	r := mustScan(t, Options{ManifestPath: path})
	wantEqual(t, [3]int{len(r.Baselined), r.Summary.Baselined, r.Summary.Errors}, [3]int{0, 0, 2})
	var found []health.Diagnostic
	for _, d := range r.Diagnostics {
		if d.Code == model.HealthWaiverUnjustifiedCode {
			found = append(found, d)
		}
	}
	mustEqual(t, len(found), 1)
	wantEqual(t, found[0].Level, health.LevelError)
	wantAll(t, found[0].Message, "model/health-baseline.json", decision, "suppresses nothing")
	low := mustScan(t, Options{ManifestPath: path, Tier: "poc"})
	wantEqual(t, low.Summary.Errors, 1) // the diagnostic only
	ignored := mustScan(t, Options{ManifestPath: path, IgnoreBaseline: true})
	wantEqual(t, len(ignored.Diagnostics), 0)
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
	r := mustScan(t, Options{ManifestPath: manifest})
	wantEqual(t, [4]int{len(r.Baselined), r.Summary.Errors, r.Summary.Baselined, len(r.Diagnostics)}, [4]int{0, 1, 0, 0})
	fs := findingsOf(r, "pkg/big.go", "Big")
	mustEqual(t, len(fs), 1)
	wantContains(t, fs[0].Message, "the baseline accepts cyclomatic 16, now 17")
	grown.Value = 17
	writeBaseline(t, dir, grown)
	r = mustScan(t, Options{ManifestPath: manifest})
	wantEqual(t, [2]int{len(r.Baselined), r.Summary.Errors}, [2]int{1, 0}) // equal value
}
