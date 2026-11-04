// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// statusFixture is child-a of the links fixture as its own git repository
// with code for REQ-A-001 to REQ-A-003, an ungated test for REQ-A-001 and
// REQ-A-003, a gated test for REQ-A-002 and a control verification over
// REQ-A-001 and REQ-A-003; it returns the directory.
func statusFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "child-a")
	copyTree(t, filepath.Join(linksExample, "repos", "child-a"), dir)
	write(t, filepath.Join(dir, "src", "enc.go"), "package enc\n\n// TRLC-LINKS: REQ-A-001\nfunc Publish() {}\n\n// TRLC-LINKS: REQ-A-002\nfunc Format() {}\n\n// TRLC-LINKS: REQ-A-003\nfunc Checksum() {}\n")
	write(t, filepath.Join(dir, "src", "enc_test.go"), `package enc

import (
	"os"
	"testing"
)

// TRLC-LINKS: REQ-A-001, REQ-A-003
func TestPublish(t *testing.T) { Publish() }

// TRLC-LINKS: REQ-A-002
func TestFormatLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE")
	}
}
`)
	write(t, filepath.Join(dir, "model", "assurance.yml"), `schemaVersion: 2
assurance:
  controls:
  - id: CTRL-ENC-INTEGRITY
    name: Encoder integrity
    category: integrity
    description: Sentences are complete and checksummed.
  controlVerifications:
  - id: VER-ENC-INTEGRITY
    controlRef: CTRL-ENC-INTEGRITY
    method: test
    status: not-run
    verificationRefs:
    - REQ-A-001
    - REQ-A-003
  - id: VER-ENC-FORMAT
    controlRef: CTRL-ENC-INTEGRITY
    method: test
    status: not-run
    verificationRefs:
    - REQ-A-002
`)
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "init")
	return dir
}

// redLine is one .engflow/red-checks.jsonl line.
func redLine(tree, verdict string, reqs ...string) string {
	data, _ := json.Marshal(map[string]any{"time": "2026-11-04T00:00:00Z", "tree": tree, "file": "src/enc.go", "find": "x", "command": []string{"go", "test"}, "verdict": verdict, "reqs": reqs})
	return string(data)
}

// engws status-suggest prints the status each requirement and control
// verification earns from the tree's evidence and writes nothing: verified
// for a requirement with an ungated, non-benchmark test link and a RED line
// of .engflow/red-checks.jsonl for this tree (git write-tree of the working
// tree with uncommitted and untracked files, or --tree) naming it; implemented for one with code only (a gated
// test or a RED on another tree, or a GREEN, earns no more); a derived
// requirement without code nothing; a control verification pass when every
// requirement it names earns verified, else it keeps its status. --json
// gives the same rows.
// Mutation: crediting a gated test, a RED of another tree or a GREEN line,
// passing a verification with an unverified requirement, or writing a file
// fails this test.
// TRLC-LINKS: REQ-EMG-122
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestStatusSuggest(t *testing.T) {
	newLinksRegistry(t)
	dir := statusFixture(t)
	// An uncommitted edit: the tree is the working tree's, not HEAD's.
	write(t, filepath.Join(dir, "NOTES.md"), "work in progress\n")
	headTree := runGit(t, dir, "rev-parse", "HEAD^{tree}")
	runGit(t, dir, "add", "NOTES.md")
	tree := runGit(t, dir, "write-tree")
	runGit(t, dir, "reset", "-q", "NOTES.md")
	if tree == headTree {
		t.Fatal("fixture: the working tree must differ from HEAD")
	}
	if err := os.MkdirAll(filepath.Join(dir, ".engflow"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, ".engflow", ".gitignore"), "*\n")
	write(t, filepath.Join(dir, ".engflow", "red-checks.jsonl"), strings.Join([]string{
		redLine(tree, "RED", "REQ-A-001"),
		redLine(tree, "GREEN", "REQ-A-003"),
		redLine("0000000000000000000000000000000000000000", "RED", "REQ-A-003"),
		redLine(tree, "RED", "REQ-A-002"),
		"not json",
	}, "\n")+"\n")
	before := runGit(t, dir, "status", "--porcelain")

	code, out, errOut := runMain(t, fakeRegistry{}, "status-suggest", "--repo", dir, "--json")
	if code != 0 {
		t.Fatalf("status-suggest: %d\n%s\n%s", code, out, errOut)
	}
	var got StatusSuggestion
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if got.Tree != tree {
		t.Fatalf("tree %s, want %s", got.Tree, tree)
	}
	want := map[string]string{"REQ-A-001": "verified", "REQ-A-002": "implemented", "REQ-A-003": "implemented", "REQ-A-004": ""}
	for _, r := range got.Requirements {
		if w, ok := want[r.ID]; !ok || r.Suggested != w {
			t.Errorf("%s suggested %q, want %q (%+v)", r.ID, r.Suggested, w, r)
		}
		delete(want, r.ID)
	}
	if len(want) > 0 {
		t.Fatalf("missing rows %v", want)
	}
	cv := map[string]string{}
	for _, v := range got.ControlVerifications {
		cv[v.ID] = v.Suggested
	}
	if cv["VER-ENC-INTEGRITY"] != "not-run" || cv["VER-ENC-FORMAT"] != "not-run" {
		t.Fatalf("control verifications %v", cv)
	}

	// The RED for REQ-A-003 on this tree makes the integrity verification pass.
	write(t, filepath.Join(dir, ".engflow", "red-checks.jsonl"), redLine(tree, "RED", "REQ-A-001", "REQ-A-003")+"\n")
	code, out, _ = runMain(t, fakeRegistry{}, "status-suggest", "--repo", dir)
	if code != 0 || !regexpLine(out, `REQ-A-003\s+-\s+verified`) || !regexpLine(out, `VER-ENC-INTEGRITY\s+not-run\s+pass`) || !regexpLine(out, `VER-ENC-FORMAT\s+not-run\s+not-run`) {
		t.Fatalf("table:\n%s", out)
	}
	// --tree names another tree: nothing on it is RED.
	code, out, _ = runMain(t, fakeRegistry{}, "status-suggest", "--repo", dir, "--tree", strings.Repeat("1", 40))
	if code != 0 || regexpLine(out, `REQ-A-001\s+-\s+verified`) || !regexpLine(out, `REQ-A-001\s+-\s+implemented`) {
		t.Fatalf("--tree:\n%s", out)
	}
	if after := runGit(t, dir, "status", "--porcelain"); after != before {
		t.Fatalf("status-suggest changed the tree:\n%s", after)
	}
}
