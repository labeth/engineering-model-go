// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"path/filepath"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// scanRepository scans this repository at tier operational with the baseline of
// its manifest.
func scanRepository(t *testing.T) engmodel.HealthReport {
	t.Helper()
	manifest, err := filepath.Abs(filepath.Join("..", "..", "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := engmodel.ScanHealth(engmodel.HealthScanOptions{ManifestPath: manifest, NoCache: true, Tier: "operational"})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

// requireCovered fails for each file below the floor that the baseline does not
// cover and when the report has any other error at the tier.
func requireCovered(t *testing.T, report engmodel.HealthReport) {
	t.Helper()
	for _, d := range report.Diagnostics {
		if d.Code == "health.file_below_floor" {
			t.Errorf("%s", d.Message)
		}
	}
	if report.Summary.Errors != 0 {
		t.Errorf("%d errors at tier operational: the repository has a finding or a file below the floor that the baseline does not cover", report.Summary.Errors)
	}
}

// The repository meets its own floor as far as it says: every file of this
// repository below 10.0 is covered by model/health-baseline.json, so the scan
// reports no file below the floor that the baseline does not cover, and the
// number of covered files is the burn-down to zero. The test does not ask for
// an empty baseline: the requirement of this repository that asks for it is
// added when the baseline is empty (ADR-EMG-087). It fails when a file is
// added or worsened below 10.0 without a baseline entry, which keeps the
// baseline the shrinking list of exceptions (ADR-EMG-087).
//
// Mutation: a scan that ignores the baseline reports every covered file and
// fails; so does a file with a finding that the baseline does not name.
//
// TRLC-LINKS: REQ-EMG-210
func TestRepositoryFilesAreAtTheFloorOrBaselined(t *testing.T) {
	report := scanRepository(t)
	requireCovered(t, report)
	t.Logf("burn-down: %d of %d files are below 10.0, %d of them covered by the baseline", report.Summary.BelowFloor, report.Summary.Files, report.Summary.FloorCovered)
}
