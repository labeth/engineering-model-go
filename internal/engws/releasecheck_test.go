// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"path/filepath"
	"strings"
	"testing"
)

// Acceptance 6: removing a capability needs a major bump. release-check
// compares the tag with the highest release tag below it: a minor tag over
// v1.0.0 is contract.semver_too_small (exit 1), a major tag passes; without
// --tag the release tag at HEAD is checked. In v0.x a breaking change needs
// a minor bump: v0.2.0 passes and v0.1.1 does not.
// Mutation: comparing with the lowest instead of the highest previous tag,
// treating a v0.x minor bump as minor-capable only, or not failing on a too
// small bump fails this test.
// TRLC-LINKS: REQ-EMG-103, REQ-EMG-102
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestReleaseCheckSemverGate(t *testing.T) {
	dir := gitModel(t, "child-a", "v0.9.0")
	runGit(t, dir, "tag", "v1.0.0")
	removeCapA2(t, dir)
	runGit(t, dir, "commit", "-q", "-am", "drop the checksum")

	assertReleaseCheck(t, releaseCheckCase{name: "major tag", dir: dir, args: []string{"--tag", "v2.0.0"}, code: 0,
		out: []string{"contract v1.0.0..HEAD: impact major, tag v2.0.0 is a major bump"}})
	assertReleaseCheck(t, releaseCheckCase{name: "minor tag", dir: dir, args: []string{"--tag", "v1.1.0"}, code: 1,
		out: []string{"contract.semver_too_small [error]", "v1.1.0 is a minor bump over v1.0.0 but the contract change is major", "removed   capability CAP-A-2"}})
	runGit(t, dir, "tag", "v1.1.0")
	assertReleaseCheck(t, releaseCheckCase{name: "tag at HEAD", dir: dir, code: 1,
		out: []string{"contract v1.0.0..v1.1.0", "contract.semver_too_small"}})

	zero := gitModel(t, "child-a", "v0.1.0")
	removeCapA2(t, zero)
	runGit(t, zero, "commit", "-q", "-am", "drop the checksum")
	assertReleaseCheck(t, releaseCheckCase{name: "v0.x minor bump with a breaking change", dir: zero, args: []string{"--tag", "v0.2.0"}, code: 0, out: []string{"v0.x"}})
	assertReleaseCheck(t, releaseCheckCase{name: "v0.x patch bump with a breaking change", dir: zero, args: []string{"--tag", "v0.1.1"}, code: 1, out: []string{"contract.semver_too_small"}})
	assertReleaseCheck(t, releaseCheckCase{name: "invalid tag", dir: zero, args: []string{"--tag", "latest"}, code: 2})
	assertReleaseCheck(t, releaseCheckCase{name: "first release", dir: zero, args: []string{"--tag", "v0.1.0"}, code: 0, out: []string{"no release tag below v0.1.0"}})
}

// releaseCheckCase is one release-check run: the arguments after --repo, the exit code and the output it must contain.
type releaseCheckCase struct {
	name string
	dir  string
	args []string
	code int
	out  []string
}

// assertReleaseCheck runs release-check and checks the exit code and that the output contains every wanted text.
func assertReleaseCheck(t *testing.T, c releaseCheckCase) {
	t.Helper()
	code, out, errOut := runMain(t, fakeRegistry{}, append([]string{"release-check", "--repo", c.dir}, c.args...)...)
	if code != c.code {
		t.Fatalf("%s: exit %d, want %d\n%s\n%s", c.name, code, c.code, out, errOut)
	}
	for _, want := range c.out {
		if !strings.Contains(out, want) {
			t.Fatalf("%s: output lacks %q\n%s\n%s", c.name, want, out, errOut)
		}
	}
}

// A release pins only release tags: a pseudo-version dependency and a
// pre-release lock entry are each dependency.unreleased_pin (exit 1).
// Mutation: checking only the manifest, only the lock, or treating a
// pseudo-version as released fails this test.
// TRLC-LINKS: REQ-EMG-103, REQ-EMG-096
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func TestReleaseCheckUnreleasedPins(t *testing.T) {
	_, system := systemWorkspace(t)
	if code, out, errOut := runMain(t, fakeRegistry{}, "release-check", "--repo", system); code != 0 || !strings.Contains(out, "semver gate skipped") {
		t.Fatalf("released pins: %d\n%s\n%s", code, out, errOut)
	}
	pseudo := "v0.1.1-0.20261102103400-abcdef123456"
	replaceIn(t, filepath.Join(system, "engmod.yml"), "  path: models.example/reqlinks/child-a@v0\n  version: v0.1.0\n", "  path: models.example/reqlinks/child-a@v0\n  version: "+pseudo+"\n")
	write(t, filepath.Join(system, "engmod.lock.yml"), "schemaVersion: 1\ndependencies:\n    - alias: bridge\n      path: models.example/reqlinks/child-b@v0\n      version: v0.2.0-rc.1\n      publications:\n        - public\n      digest: sha256:00\n")
	code, out, errOut := runMain(t, fakeRegistry{}, "release-check", "--repo", system)
	if code != 1 || !strings.Contains(out, "dependency.unreleased_pin [error] dependency encoder pins "+pseudo+" (engmod.yml)") ||
		!strings.Contains(out, "dependency.unreleased_pin [error] dependency bridge pins v0.2.0-rc.1 (engmod.lock.yml)") {
		t.Fatalf("unreleased pins: %d\n%s\n%s", code, out, errOut)
	}
}

// The previous release is the nearest release tag below the tag in the
// commit's own history (git tag --merged), not the highest release tag of
// the repository: a v1.5.0 on a side branch that already dropped the
// capability does not hide the removal from a v1.6.0 cut from main, whose
// last release is v1.0.0.
// Mutation: taking the previous tag from every release tag instead of the
// tags merged into the commit fails this test.
// TRLC-LINKS: REQ-EMG-103
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestReleaseCheckPreviousTagIsAnAncestor(t *testing.T) {
	dir := gitModel(t, "child-a", "v1.0.0")
	runGit(t, dir, "checkout", "-q", "-b", "side")
	removeCapA2(t, dir)
	runGit(t, dir, "commit", "-q", "-am", "side: drop the checksum")
	runGit(t, dir, "tag", "v1.5.0")
	runGit(t, dir, "checkout", "-q", "-")
	removeCapA2(t, dir)
	runGit(t, dir, "commit", "-q", "-am", "main: drop the checksum")
	code, out, errOut := runMain(t, fakeRegistry{}, "release-check", "--repo", dir, "--tag", "v1.6.0")
	if code != 1 || !strings.Contains(out, "contract v1.0.0..HEAD: impact major") || !strings.Contains(out, "contract.semver_too_small") {
		t.Fatalf("release-check from main: %d\n%s\n%s", code, out, errOut)
	}
	runGit(t, dir, "tag", "v2.0.0")
	if code, out, _ := runMain(t, fakeRegistry{}, "release-check", "--repo", dir); code != 0 || !strings.Contains(out, "contract v1.0.0..v2.0.0") {
		t.Fatalf("release-check at the tag: %d\n%s", code, out)
	}
}
