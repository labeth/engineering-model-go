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

	code, out, errOut := runMain(t, fakeRegistry{}, "release-check", "--repo", dir, "--tag", "v2.0.0")
	if code != 0 || !strings.Contains(out, "contract v1.0.0..HEAD: impact major, tag v2.0.0 is a major bump") {
		t.Fatalf("major tag: %d\n%s\n%s", code, out, errOut)
	}
	code, out, errOut = runMain(t, fakeRegistry{}, "release-check", "--repo", dir, "--tag", "v1.1.0")
	if code != 1 || !strings.Contains(out, "contract.semver_too_small [error]") || !strings.Contains(out, "v1.1.0 is a minor bump over v1.0.0 but the contract change is major") ||
		!strings.Contains(out, "removed   capability CAP-A-2") {
		t.Fatalf("minor tag: %d\n%s\n%s", code, out, errOut)
	}
	runGit(t, dir, "tag", "v1.1.0")
	if code, out, _ := runMain(t, fakeRegistry{}, "release-check", "--repo", dir); code != 1 || !strings.Contains(out, "contract v1.0.0..v1.1.0") || !strings.Contains(out, "contract.semver_too_small") {
		t.Fatalf("tag at HEAD: %d\n%s", code, out)
	}

	zero := gitModel(t, "child-a", "v0.1.0")
	removeCapA2(t, zero)
	runGit(t, zero, "commit", "-q", "-am", "drop the checksum")
	if code, out, errOut := runMain(t, fakeRegistry{}, "release-check", "--repo", zero, "--tag", "v0.2.0"); code != 0 || !strings.Contains(out, "v0.x") {
		t.Fatalf("v0.x minor bump with a breaking change: %d\n%s\n%s", code, out, errOut)
	}
	if code, out, _ := runMain(t, fakeRegistry{}, "release-check", "--repo", zero, "--tag", "v0.1.1"); code != 1 || !strings.Contains(out, "contract.semver_too_small") {
		t.Fatalf("v0.x patch bump with a breaking change: %d\n%s", code, out)
	}
	if code, _, _ := runMain(t, fakeRegistry{}, "release-check", "--repo", zero, "--tag", "latest"); code != 2 {
		t.Fatalf("invalid tag: %d", code)
	}
	if code, out, _ := runMain(t, fakeRegistry{}, "release-check", "--repo", zero, "--tag", "v0.1.0"); code != 0 || !strings.Contains(out, "no release tag below v0.1.0") {
		t.Fatalf("first release: %d\n%s", code, out)
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
