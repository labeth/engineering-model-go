// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// fixtureBundle loads a copy of a fixture child after edit.
func fixtureBundle(t *testing.T, child string, edit func(dir string)) model.Bundle {
	t.Helper()
	dir := filepath.Join(t.TempDir(), child)
	copyTree(t, filepath.Join(linksExample, "repos", child), dir)
	if edit != nil {
		edit(dir)
	}
	b, err := model.LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// removeCapA2 removes the published CAP-A-2 from a child-a copy; REQ-A-003
// becomes derived.
func removeCapA2(t *testing.T, dir string) {
	t.Helper()
	replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "    - id: CAP-A-2\n      kind: capability\n      note: Protect each position sentence with a checksum.\n", "")
	replaceIn(t, filepath.Join(dir, "engmod.yml"), "  - CAP-A-2\n", "")
	replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), "  realizes: [CAP-A-2]\n", "  derived: true\n  derivedRationale: Integrity of the encoder output.\n")
}

// changes returns "ID:change:impact" per changed item and the overall impact.
func changes(d ContractDiffResult) (string, string) {
	var out []string
	for _, c := range d.Items {
		if c.Change != ChangeUnchanged {
			out = append(out, c.ID+":"+string(c.Change)+":"+string(c.Impact))
		}
	}
	return strings.Join(out, ","), string(d.Impact)
}

// The contract diff classifies each published contract item and derives the
// semver impact: a removed capability, a changed capability text, an added
// assumption and a changed assumption are major; a removed assumption, an
// added capability and realizer-only changes are minor; anything else
// (an unrelated requirement) is patch.
// Mutation: classifying a removal or a capability text change as minor, an
// added assumption as minor, a realizer change as patch or ignoring the
// realizers fails this test.
// TRLC-LINKS: REQ-EMG-102
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestDiffContractsImpact(t *testing.T) {
	baseA := fixtureBundle(t, "child-a", nil)
	baseB := fixtureBundle(t, "child-b", nil)
	reqs := func(old, new string) func(string) {
		return func(dir string) { replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), old, new) }
	}
	cases := []struct {
		name        string
		from, to    model.Bundle
		want, level string
	}{
		{"unrelated", baseA, fixtureBundle(t, "child-a", reqs("shall write one log line.", "shall write one structured log line.")), "", "patch"},
		{"removed capability", baseA, fixtureBundle(t, "child-a", func(dir string) { removeCapA2(t, dir) }), "CAP-A-2:removed:major", "major"},
		{"capability text", baseA, fixtureBundle(t, "child-a", func(dir string) {
			replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "note: Protect each position sentence with a checksum.", "note: Protect each position sentence with a CRC.")
		}), "CAP-A-2:changed:major", "major"},
		{"realizer only", baseA, fixtureBundle(t, "child-a", reqs("shall publish one position sentence.", "shall publish two position sentences.")), "CAP-A-1:changed:minor", "minor"},
		{"added capability", baseA, fixtureBundle(t, "child-a", func(dir string) {
			replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "      note: Protect each position sentence with a checksum.\n", "      note: Protect each position sentence with a checksum.\n    - id: CAP-A-3\n      kind: capability\n      note: Report the encoder version.\n")
			replaceIn(t, filepath.Join(dir, "engmod.yml"), "  - CAP-A-2\n", "  - CAP-A-2\n  - CAP-A-3\n")
			replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), "  realizes: [CAP-A-2]\n", "  realizes: [CAP-A-2, CAP-A-3]\n")
		}), "CAP-A-3:added:minor", "minor"},
		{"added assumption", baseB, fixtureBundle(t, "child-b", func(dir string) {
			replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "      verificationCriteria: The using system shows the lab bus is isolated from other networks.\n",
				"      verificationCriteria: The using system shows the lab bus is isolated from other networks.\n    - id: AOU-B-2\n      text: Position sentences arrive at most once per epoch.\n")
			replaceIn(t, filepath.Join(dir, "engmod.yml"), "  - AOU-B-1\n", "  - AOU-B-1\n  - AOU-B-2\n")
		}), "AOU-B-2:added:major", "major"},
		{"tightened assumption", baseB, fixtureBundle(t, "child-b", func(dir string) {
			replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "shows the lab bus is isolated from other networks.", "shows the lab bus is isolated from other networks and the internet.")
		}), "AOU-B-1:changed:major", "major"},
	}
	for _, c := range cases {
		got, level := changes(DiffContracts(c.from, c.to))
		if got != c.want || level != c.level {
			t.Errorf("%s: got %q %s, want %q %s", c.name, got, level, c.want, c.level)
		}
	}
	// Removing an assumption is minor (users relied on less).
	got, level := changes(DiffContracts(fixtureBundle(t, "child-b", func(dir string) {
		replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "      verificationCriteria: The using system shows the lab bus is isolated from other networks.\n",
			"      verificationCriteria: The using system shows the lab bus is isolated from other networks.\n    - id: AOU-B-2\n      text: Position sentences arrive at most once per epoch.\n")
		replaceIn(t, filepath.Join(dir, "engmod.yml"), "  - AOU-B-1\n", "  - AOU-B-1\n  - AOU-B-2\n")
	}), baseB))
	if got != "AOU-B-2:removed:minor" || level != "minor" {
		t.Fatalf("removed assumption: %q %s", got, level)
	}
}

// gitModel puts a copy of a fixture child in its own git repository,
// committed and tagged.
func gitModel(t *testing.T, child, tag string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), child)
	copyTree(t, filepath.Join(linksExample, "repos", child), dir)
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")
	runGit(t, dir, "tag", tag)
	return dir
}

// Acceptance 6, first half: contract-diff reads the model at two git refs
// (not the working tree) and reports the removed capability, the text
// change note and the major impact, in text and JSON.
// Mutation: reading the working tree instead of the refs, or dropping the
// "treated as major" note, fails this test.
// TRLC-LINKS: REQ-EMG-102
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestContractDiffBetweenTags(t *testing.T) {
	dir := gitModel(t, "child-a", "v1.0.0")
	removeCapA2(t, dir)
	replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "note: Publish ownship position sentences at the configured rate.", "note: Publish ownship position sentences at the configured epoch rate.")
	runGit(t, dir, "commit", "-q", "-am", "drop the checksum")
	runGit(t, dir, "tag", "v1.1.0")
	// The working tree differs from both refs.
	replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "at the configured epoch rate.", "at the working tree rate.")

	code, out, errOut := runMain(t, fakeRegistry{}, "contract-diff", "v1.0.0", "v1.1.0", "--repo", dir)
	if code != 0 {
		t.Fatalf("contract-diff: %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{
		"removed   capability CAP-A-2: Protect each position sentence with a checksum. [major]",
		"changed   capability CAP-A-1 (text): text changed: treated as major [major]",
		"impact: major",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("contract-diff lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "working tree") {
		t.Fatalf("contract-diff read the working tree:\n%s", out)
	}
	code, out, _ = runMain(t, fakeRegistry{}, "contract-diff", "--repo", dir, "--json", "v1.0.0", "v1.1.0")
	var result ContractDiffResult
	if err := json.Unmarshal([]byte(out), &result); code != 0 || err != nil || result.Impact != ImpactMajor || result.From != "v1.0.0" || len(result.Items) != 2 {
		t.Fatalf("contract-diff --json: %d %v\n%s", code, err, out)
	}
	if code, _, errOut := runMain(t, fakeRegistry{}, "contract-diff", "v1.0.0", "v9.9.9", "--repo", dir); code != 2 || !strings.Contains(errOut, "v9.9.9") {
		t.Fatalf("unknown ref: %d %s", code, errOut)
	}
	if code, _, _ := runMain(t, fakeRegistry{}, "contract-diff", "v1.0.0", "--repo", dir); code != 2 {
		t.Fatalf("one ref accepted: %d", code)
	}
}

// A git ref is never read as a git option: a ref starting with "-" or an
// empty ref is refused before git runs, and a ref that names no commit is
// refused by rev-parse --verify, so contract-diff and release-check cannot
// be steered into another git command (for example --output=FILE).
// Mutation: dropping the "-" prefix check of LoadBundleAtRef fails this
// test.
// TRLC-LINKS: REQ-EMG-102
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestLoadBundleAtRefRejectsOptionRefs(t *testing.T) {
	dir := gitModel(t, "child-a", "v1.0.0")
	for _, ref := range []string{"--output=" + filepath.Join(dir, "pwned"), "-p", "", " "} {
		_, err := LoadBundleAtRef(dir, ref)
		if err == nil || !strings.Contains(err.Error(), "invalid git ref") {
			t.Errorf("ref %q: err = %v, want invalid git ref", ref, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
		t.Fatal("a ref option made git write a file")
	}
	if _, err := LoadBundleAtRef(dir, "v1.0.0"); err != nil {
		t.Fatalf("a tag ref: %v", err)
	}
}
