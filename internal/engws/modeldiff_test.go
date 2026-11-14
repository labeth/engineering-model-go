// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// modelDiffRepo is child-a at v1.0.0, then at v1.1.0 with one requirement added, one removed, one edited, and
// a source file and a test file that carry trace markers for REQ-A-001.
func modelDiffRepo(t *testing.T) string {
	dir := gitModel(t, "child-a", "v1.0.0")
	req := filepath.Join(dir, "model", "requirements.yml")
	data, err := os.ReadFile(req)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	start := strings.Index(text, "- id: REQ-A-004")
	if start < 0 {
		t.Fatal("fixture has no REQ-A-004")
	}
	text = text[:start] // REQ-A-004 is the last requirement: removed
	text = strings.Replace(text, "verificationCriteria: A published sentence carries latitude and longitude.", "verificationCriteria: A published sentence carries latitude and longitude, both in degrees.", 1)
	text += "- id: REQ-A-005\n  text: The sentence encoder shall append a checksum to each position sentence.\n  verificationMethods: [test]\n  verificationCriteria: The checksum matches.\n  appliesTo: [FU-ENCODER]\n  realizes: [CAP-A-2]\n"
	write(t, req, text)
	write(t, filepath.Join(dir, "src", "encoder.go"), "// ENGMODEL-OWNER-UNIT: FU-ENCODER\npackage src\n\n// TRLC-LINKS: REQ-A-001\nfunc Publish() {}\n")
	write(t, filepath.Join(dir, "src", "encoder_test.go"), "// ENGMODEL-OWNER-UNIT: FU-ENCODER\npackage src\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-A-001\nfunc TestPublish(t *testing.T) {}\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "second")
	runGit(t, dir, "tag", "v1.1.0")
	return dir
}

// runModelDiffJSON runs engws model-diff with --json and decodes the result.
func runModelDiffJSON(t *testing.T, args ...string) engmodel.ModelDiff {
	t.Helper()
	code, out, errOut := runMain(t, fakeRegistry{}, append([]string{"model-diff", "--json"}, args...)...)
	var d engmodel.ModelDiff
	if err := json.Unmarshal([]byte(out), &d); code != 0 || err != nil {
		t.Fatalf("exit %d, json error %v: %s\n%s", code, err, errOut, out)
	}
	return d
}

func tagDiff(t *testing.T) engmodel.ModelDiff {
	t.Helper()
	return runModelDiffJSON(t, "--repo", modelDiffRepo(t), "v1.0.0", "v1.1.0")
}

// The whole authored model is compared by stable ID between two tags: a requirement added, one removed and one
// edited in a single field.
// Mutation: comparing only the requirements document, or reading the head twice, fails this test.
// TRLC-LINKS: REQ-EMG-190
func TestModelDiffBetweenTagsReportsAddedRemovedAndChangedEntities(t *testing.T) {
	d := tagDiff(t)
	if d.From != "v1.0.0" || d.To != "v1.1.0" {
		t.Errorf("refs %q..%q", d.From, d.To)
	}
	if got := changeIDs(d.Entities.Added); !reflect.DeepEqual(got, []string{"REQ-A-005"}) {
		t.Errorf("added %v", got)
	}
	if got := changeIDs(d.Entities.Removed); !reflect.DeepEqual(got, []string{"REQ-A-004"}) {
		t.Errorf("removed %v", got)
	}
}

// TRLC-LINKS: REQ-EMG-190
func TestModelDiffNamesTheFieldThatChanged(t *testing.T) {
	changed := tagDiff(t).Entities.Changed
	if len(changed) != 1 || changed[0].ID != "REQ-A-002" || !reflect.DeepEqual(changed[0].Fields, []string{"verificationCriteria"}) {
		t.Errorf("only REQ-A-002 changed, in verificationCriteria only: %+v", changed)
	}
}

func changeIDs(list []engmodel.EntityChange) []string {
	out := []string{}
	for _, c := range list {
		out = append(out, c.ID)
	}
	return out
}

// The trace delta shows which requirement gained code and a test and which became or stopped being an orphan.
// Mutation: reading the trace of the head twice fails this test.
// TRLC-LINKS: REQ-EMG-191
func TestModelDiffTraceMovesWithCodeAndTests(t *testing.T) {
	d := tagDiff(t)
	if d.Trace == nil {
		t.Fatalf("no trace delta: %s", d.TraceError)
	}
	for name, got := range map[string][]string{"gained code": d.Trace.Gained, "gained tests": d.Trace.TestsGained, "resolved orphans": d.Trace.ResolvedOrphans} {
		if !slices.Contains(got, "REQ-A-001") {
			t.Errorf("REQ-A-001 is among the %s: %v", name, got)
		}
	}
	if !slices.Contains(d.Trace.NewOrphans, "REQ-A-005") {
		t.Errorf("REQ-A-005 is a new orphan: %v", d.Trace.NewOrphans)
	}
}

// Leaving out the second ref compares with the working tree, so an uncommitted edit shows.
// TRLC-LINKS: REQ-EMG-190
func TestModelDiffWithoutASecondRefReadsTheWorkingTree(t *testing.T) {
	dir := modelDiffRepo(t)
	write(t, filepath.Join(dir, "src", "encoder.go"), "// ENGMODEL-OWNER-UNIT: FU-ENCODER\npackage src\n\n// TRLC-LINKS: REQ-A-001, REQ-A-003\nfunc Publish() {}\n")
	d := runModelDiffJSON(t, "--repo", dir, "v1.1.0")
	if d.To != "" || len(d.Entities.Added)+len(d.Entities.Removed)+len(d.Entities.Changed) != 0 {
		t.Errorf("the model did not change, only code did: %+v", d.Entities)
	}
	if d.Trace == nil || !slices.Contains(d.Trace.Gained, "REQ-A-003") {
		t.Fatalf("REQ-A-003 gained code in the working tree: %+v (%s)", d.Trace, d.TraceError)
	}
}

// --no-trace skips the code scan.
// TRLC-LINKS: REQ-EMG-190
func TestModelDiffNoTraceSkipsTheCodeScan(t *testing.T) {
	d := runModelDiffJSON(t, "--repo", modelDiffRepo(t), "--no-trace", "v1.0.0", "v1.1.0")
	if d.Trace != nil || d.TraceError != "" {
		t.Errorf("--no-trace: %+v", d)
	}
}

// The text form says what changed.
// TRLC-LINKS: REQ-EMG-190
func TestModelDiffTextForm(t *testing.T) {
	_, out, _ := runMain(t, fakeRegistry{}, "model-diff", "--repo", modelDiffRepo(t), "v1.0.0", "v1.1.0")
	for _, want := range []string{"model v1.0.0..v1.1.0", "added    requirement REQ-A-005", "removed  requirement REQ-A-004", "changed  requirement REQ-A-002 (verificationCriteria)", "trace:"} {
		if !strings.Contains(out, want) {
			t.Errorf("text form lacks %q:\n%s", want, out)
		}
	}
}

// An unknown ref, a ref that git would read as an option and a missing ref exit 2, and the option writes nothing.
// TRLC-LINKS: REQ-EMG-190
func TestModelDiffRefusesBadRefs(t *testing.T) {
	dir := modelDiffRepo(t)
	pwned := filepath.Join(dir, "pwned")
	for _, args := range [][]string{{"model-diff", "--repo", dir, "v9.9.9"}, {"model-diff", "--repo", dir, "--output=" + pwned}, {"model-diff", "--repo", dir}} {
		if code, _, _ := runMain(t, fakeRegistry{}, args...); code != 2 {
			t.Errorf("%v exited %d, want 2", args, code)
		}
	}
	if _, err := os.Stat(pwned); err == nil {
		t.Fatal("a ref option made git write a file")
	}
}
