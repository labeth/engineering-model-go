// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

func has(list []engmodel.EntityChange, id string) bool {
	for _, c := range list {
		if c.ID == id {
			return true
		}
	}
	return false
}

func contains(list []string, id string) bool {
	for _, s := range list {
		if s == id {
			return true
		}
	}
	return false
}

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

// The whole authored model is compared by stable ID between two tags, and the trace delta shows which requirement
// gained code and a test and which became or stopped being an orphan.
// Mutation: comparing only the requirements document, or reading the trace of the head twice, fails this test.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-191
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestModelDiffBetweenTagsReportsEntitiesAndTraceMovement(t *testing.T) {
	dir := modelDiffRepo(t)
	code, out, errOut := runMain(t, fakeRegistry{}, "model-diff", "--repo", dir, "--json", "v1.0.0", "v1.1.0")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var d engmodel.ModelDiff
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if d.From != "v1.0.0" || d.To != "v1.1.0" {
		t.Errorf("refs %q..%q", d.From, d.To)
	}
	if !has(d.Entities.Added, "REQ-A-005") || !has(d.Entities.Removed, "REQ-A-004") {
		t.Errorf("added %v removed %v", d.Entities.Added, d.Entities.Removed)
	}
	var edited *engmodel.EntityChange
	for i := range d.Entities.Changed {
		if d.Entities.Changed[i].ID == "REQ-A-002" {
			edited = &d.Entities.Changed[i]
		}
	}
	if edited == nil || len(edited.Fields) != 1 || edited.Fields[0] != "verificationCriteria" {
		t.Errorf("REQ-A-002 changed in verificationCriteria only: %+v", d.Entities.Changed)
	}
	if has(d.Entities.Changed, "REQ-A-001") || has(d.Entities.Changed, "REQ-A-003") {
		t.Errorf("an untouched requirement is reported as changed: %+v", d.Entities.Changed)
	}
	if d.Trace == nil {
		t.Fatalf("no trace delta: %s", d.TraceError)
	}
	if !contains(d.Trace.Gained, "REQ-A-001") || !contains(d.Trace.TestsGained, "REQ-A-001") {
		t.Errorf("REQ-A-001 gained code and a test: %+v", d.Trace)
	}
	if !contains(d.Trace.ResolvedOrphans, "REQ-A-001") || !contains(d.Trace.NewOrphans, "REQ-A-005") {
		t.Errorf("orphans: resolved %v new %v", d.Trace.ResolvedOrphans, d.Trace.NewOrphans)
	}
}

// Leaving out the second ref compares with the working tree, so an uncommitted edit shows.
// TRLC-LINKS: REQ-EMG-190
func TestModelDiffWithoutASecondRefReadsTheWorkingTree(t *testing.T) {
	dir := modelDiffRepo(t)
	write(t, filepath.Join(dir, "src", "encoder.go"), "// ENGMODEL-OWNER-UNIT: FU-ENCODER\npackage src\n\n// TRLC-LINKS: REQ-A-001, REQ-A-003\nfunc Publish() {}\n")
	code, out, errOut := runMain(t, fakeRegistry{}, "model-diff", "--repo", dir, "--json", "v1.1.0")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var d engmodel.ModelDiff
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		t.Fatal(err)
	}
	if d.To != "" || len(d.Entities.Added)+len(d.Entities.Removed)+len(d.Entities.Changed) != 0 {
		t.Errorf("the model did not change, only code did: %+v", d.Entities)
	}
	if d.Trace == nil || !contains(d.Trace.Gained, "REQ-A-003") {
		t.Fatalf("REQ-A-003 gained code in the working tree: %+v (%s)", d.Trace, d.TraceError)
	}
}

// --no-trace skips the code scan, an unknown ref and a ref that git would read as an option exit 2, and the text form
// says what changed.
// TRLC-LINKS: REQ-EMG-190
func TestModelDiffFlagsErrorsAndTextForm(t *testing.T) {
	dir := modelDiffRepo(t)
	code, out, _ := runMain(t, fakeRegistry{}, "model-diff", "--repo", dir, "--json", "--no-trace", "v1.0.0", "v1.1.0")
	var d engmodel.ModelDiff
	if err := json.Unmarshal([]byte(out), &d); code != 0 || err != nil || d.Trace != nil || d.TraceError != "" {
		t.Fatalf("--no-trace: %d %v %+v", code, err, d)
	}
	code, out, _ = runMain(t, fakeRegistry{}, "model-diff", "--repo", dir, "v1.0.0", "v1.1.0")
	for _, want := range []string{"model v1.0.0..v1.1.0", "added    requirement REQ-A-005", "removed  requirement REQ-A-004", "changed  requirement REQ-A-002 (verificationCriteria)", "trace:"} {
		if !strings.Contains(out, want) {
			t.Errorf("text form lacks %q:\n%s", want, out)
		}
	}
	for _, args := range [][]string{{"model-diff", "--repo", dir, "v9.9.9"}, {"model-diff", "--repo", dir, "--output=" + filepath.Join(dir, "pwned")}, {"model-diff", "--repo", dir}} {
		if code, _, _ := runMain(t, fakeRegistry{}, args...); code != 2 {
			t.Errorf("%v exited %d, want 2", args, code)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
		t.Fatal("a ref option made git write a file")
	}
}
