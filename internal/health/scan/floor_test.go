// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package scan

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/model"
)

// floorManifest is a product model with a baseline; the fixture has big.go at
// 8.0 (a high complex_method) and clean.go at 10.0, skipme is excluded.
const floorManifest = baselineManifest

// scanFloor scans a fixture with the manifest and the entries of the baseline.
func scanFloor(t *testing.T, manifest string, entries ...model.HealthBaselineEntry) health.Report {
	t.Helper()
	dir, path := fixture(t, manifest)
	writeBaseline(t, dir, entries...)
	r, err := Scan(Options{ManifestPath: path, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// floorPaths lists "path level" of each health.file_below_floor diagnostic.
func floorPaths(r health.Report) []string {
	out := []string{}
	for _, d := range r.Diagnostics {
		if d.Code == health.CodeFileBelowFloor {
			out = append(out, d.Path+" "+d.Level)
		}
	}
	return out
}

// floorCounts is the files below the floor, the covered ones and the errors.
func floorCounts(r health.Report) [3]int {
	return [3]int{r.Summary.BelowFloor, r.Summary.FloorCovered, r.Summary.Errors}
}

// coverCase is a baseline and the files reported below the floor.
type coverCase struct {
	name    string
	entries []model.HealthBaselineEntry
	want    []string
}

// coverCases are an empty baseline, the baseline of another file, the baseline
// of the file and an entry that the finding outgrew.
func coverCases() []coverCase {
	other := model.HealthBaselineEntry{Path: "pkg/clean.go", Function: "Clean", Code: health.CodeLargeMethod, Value: 60}
	grown := model.HealthBaselineEntry{Path: "pkg/big.go", Function: "Big", Code: health.CodeComplexMethod, Value: 5}
	return []coverCase{
		{"an empty baseline covers nothing", nil, []string{"pkg/big.go error"}},
		{"the entry of another file covers nothing", []model.HealthBaselineEntry{other}, []string{"pkg/big.go error"}},
		{"an entry of the file covers it", []model.HealthBaselineEntry{bigEntry()}, []string{}},
		{"an entry the finding outgrew still covers the file", []model.HealthBaselineEntry{grown}, []string{}},
	}
}

// The scan reports health.file_below_floor (ADR-EMG-087) for big.go, whose score
// is 8.0, unless an entry of the baseline names a finding of it; a finding that
// outgrew its entry is its own error and the entry still covers the file.
//
// Mutation: not covering by the baseline, covering by the entries of other
// files, or not covering when the finding outgrew its entry fails.
//
// TRLC-LINKS: REQ-EMG-209
func TestScanReportsFilesBelowTheFloorUnlessCovered(t *testing.T) {
	for _, c := range coverCases() {
		t.Run(c.name, func(t *testing.T) {
			r := scanFloor(t, floorManifest, c.entries...)
			if got := floorPaths(r); !reflect.DeepEqual(got, c.want) {
				t.Errorf("health.file_below_floor %v, want %v", got, c.want)
			}
		})
	}
}

// The summary counts the covered file as the burn-down and the report names the
// floor.
//
// Mutation: not counting the covered file, or a report without its floor, fails.
//
// TRLC-LINKS: REQ-EMG-209
func TestScanFloorCountsTheBurnDown(t *testing.T) {
	covered := scanFloor(t, floorManifest, bigEntry())
	if got := floorCounts(covered); got != [3]int{1, 1, 0} || covered.Floor != 10.0 {
		t.Errorf("covered: floor %v counts %v, want one file below the floor, covered, no error", covered.Floor, got)
	}
	if got := floorCounts(scanFloor(t, floorManifest)); got != [3]int{1, 0, 2} {
		t.Errorf("uncovered: counts %v, want the file and its finding as errors", got)
	}
}

// health.floor of the manifest lowers the floor: 8.0 and below pass, above
// fails.
//
// Mutation: not reading the floor of the manifest fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-209
func TestScanFloorOfTheManifest(t *testing.T) {
	for floor, want := range map[string]int{"7.5": 0, "8.0": 0, "8.1": 1, "10.0": 1} {
		r := scanFloor(t, "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n  floor: "+floor+"\n")
		if got := floorPaths(r); len(got) != want || r.Floor == 0 {
			t.Errorf("floor %s: %v (report floor %v), want %d diagnostics", floor, got, r.Floor, want)
		}
	}
}

// The tier decides the level of the diagnostic and the --tier option replaces it.
//
// Mutation: a level that ignores the tier fails.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-209
func TestScanFloorLevelFollowsTheTier(t *testing.T) {
	dir, manifest := fixture(t, floorManifest)
	writeBaseline(t, dir)
	for tier, want := range map[string]string{"poc": "warning", "mvp": "warning", "product": "error", "operational": "error"} {
		r, err := Scan(Options{ManifestPath: manifest, Tier: tier, NoCache: true})
		if err != nil {
			t.Fatal(err)
		}
		if got := floorPaths(r); !reflect.DeepEqual(got, []string{"pkg/big.go " + want}) {
			t.Errorf("tier %s: %v, want big.go as a %s", tier, got, want)
		}
	}
}

// A scan restricted to a clean file reports nothing, and two scans of one tree
// are byte-identical.
//
// Mutation: reporting a file outside the scope, or an unstable order, fails.
//
// TRLC-LINKS: REQ-EMG-158, REQ-EMG-209
func TestScanFloorRestrictedAndDeterministic(t *testing.T) {
	dir, manifest := fixture(t, floorManifest)
	writeBaseline(t, dir)
	clean, err := Scan(Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg", "clean.go")}, NoCache: true})
	if err != nil || len(floorPaths(clean)) != 0 || clean.Summary.BelowFloor != 0 {
		t.Errorf("restricted to a clean file: %v %+v %v", floorPaths(clean), clean.Summary, err)
	}
	first, _ := Scan(Options{ManifestPath: manifest, NoCache: true})
	second, _ := Scan(Options{ManifestPath: manifest, NoCache: true})
	a, _ := health.Encode(first)
	b, _ := health.Encode(second)
	if string(a) != string(b) {
		t.Errorf("two scans of one tree differ:\n%s\n%s", a, b)
	}
}
