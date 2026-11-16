// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// repositoryManifest is the manifest of this repository.
func repositoryManifest(t *testing.T) string {
	t.Helper()
	manifest, err := filepath.Abs(filepath.Join("..", "..", "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

// scanRepository scans this repository at tier operational with the baseline of
// its manifest.
func scanRepository(t *testing.T) engmodel.HealthReport {
	t.Helper()
	report, err := engmodel.ScanHealth(engmodel.HealthScanOptions{ManifestPath: repositoryManifest(t), NoCache: true, Tier: "operational"})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

// baselineEntries counts the entries of the baseline file of this repository.
func baselineEntries(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(filepath.Dir(repositoryManifest(t)), "model", "health-baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	var baseline struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatal(err)
	}
	return len(baseline.Entries)
}

// requireEveryFileAtTen fails for each file whose score is below 10.0, read
// from the score itself and not from a diagnostic, so that a lower floor or a
// wider waiver cannot hide it.
func requireEveryFileAtTen(t *testing.T, report engmodel.HealthReport) {
	t.Helper()
	if len(report.Files) == 0 {
		t.Fatal("the scan found no source file")
	}
	for _, f := range report.Files {
		if f.Score < 10.0 {
			t.Errorf("%s scores %.1f, not 10.0", f.Path, f.Score)
		}
	}
}

// debtCount adds the findings, the errors, the warnings, the waived findings,
// the baselined findings and the files below the floor or covered by a baseline
// that the summary of a report counts.
func debtCount(s engmodel.HealthSummary) int {
	return s.Findings + s.Errors + s.Warnings + s.Waived + s.Baselined + s.BelowFloor + s.FloorCovered
}

// requireNoDebt fails for each diagnostic, finding, waiver and covered file of
// the report: the repository has nothing left to burn down.
func requireNoDebt(t *testing.T, report engmodel.HealthReport) {
	t.Helper()
	for _, d := range report.Diagnostics {
		t.Errorf("%s: %s", d.Code, d.Message)
	}
	if debtCount(report.Summary) != 0 {
		t.Errorf("the scan reports debt at tier operational: %+v", report.Summary)
	}
}

// Every source file of this repository scores 10.0 in the scan at tier
// operational (ADR-EMG-087, ADR-EMG-088), tests included: the scan reports no
// finding, no waived finding, no file below the floor and no file the baseline
// covers, and model/health-baseline.json has no entry. It fails when a file is
// added or changed below 10.0, and when a baseline entry or a waiver is added
// to hide it; the floor itself is held at 10.0 by the tests of
// health.file_below_floor.
//
// Mutation: a function with ten branches appended to a source file or to a test
// file scores below 10.0 and fails the test; so does a baseline entry.
//
// TRLC-LINKS: REQ-EMG-210, REQ-EMG-211
func TestEveryRepositoryFileScoresTen(t *testing.T) {
	report := scanRepository(t)
	requireEveryFileAtTen(t, report)
	requireNoDebt(t, report)
	if n := baselineEntries(t); n != 0 {
		t.Errorf("model/health-baseline.json has %d entries, want none", n)
	}
	t.Logf("%d files, every one at 10.0", report.Summary.Files)
}
