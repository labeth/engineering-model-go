// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// linkAddRepo is the system model engws link add writes into.
type linkAddRepo struct{ system, links string }

// add runs engws link add on the system.
func (r linkAddRepo) add(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	return runMain(t, fakeRegistry{}, append([]string{"link", "add", "--repo", r.system}, args...)...)
}

// requireAppendedLink adds a link with a rationale and requires it to be
// the only change of the link file; it returns the link file.
func (r linkAddRepo) requireAppendedLink(t *testing.T) string {
	t.Helper()
	before := read(t, r.links)
	code, out, errOut := r.add(t, "--parent", "REQ-SYS-002", "encoder::CAP-A-1", "--rationale", "The sentence carries the checksum.")
	if code != 0 || !strings.Contains(out, "added LNK-SYS-002-ENCODER-2 REQ-SYS-002 -> encoder::CAP-A-1 (v0.1.0, claimed)") {
		t.Fatalf("link add: %d\n%s\n%s", code, out, errOut)
	}
	after := read(t, r.links)
	want := "- id: LNK-SYS-002-ENCODER-2\n  parent: REQ-SYS-002\n  child: encoder::CAP-A-1\n  pinned:\n    version: v0.1.0\n  status: claimed\n  rationale: The sentence carries the checksum.\n"
	if after != before+want {
		t.Fatalf("link file:\n%s\nwant appended:\n%s", after, want)
	}
	if errs := checkErrors(t, r.system); len(errs) > 0 {
		t.Fatalf("check after link add:\n%s", strings.Join(errs, "\n"))
	}
	return after
}

// requireRefusals requires each invalid link add to fail and write nothing.
func (r linkAddRepo) requireRefusals(t *testing.T, unchanged string) {
	t.Helper()
	for name, args := range map[string][]string{
		"duplicate":           {"--parent", "REQ-SYS-002", "encoder::CAP-A-1"},
		"unknown parent":      {"--parent", "REQ-SYS-404", "encoder::CAP-A-1"},
		"unpublished item":    {"--parent", "REQ-SYS-002", "encoder::CAP-A-9"},
		"capability evidence": {"--evidence", "REQ-SYS-003", "encoder::CAP-A-1"},
		"undeclared alias":    {"--parent", "REQ-SYS-002", "nope::CAP-A-1"},
		"no parent":           {"encoder::CAP-A-1"},
	} {
		code, out, errOut := r.add(t, args...)
		if code == 0 || read(t, r.links) != unchanged {
			t.Errorf("%s: %d\n%s\n%s", name, code, out, errOut)
		}
	}
}

// requireRefusalMessages requires the duplicate and the unpublished refusal
// to say why.
func (r linkAddRepo) requireRefusalMessages(t *testing.T) {
	t.Helper()
	code, _, errOut := r.add(t, "--parent", "REQ-SYS-002", "encoder::CAP-A-1")
	if !strings.Contains(errOut, "already links REQ-SYS-002 to encoder::CAP-A-1") {
		t.Fatalf("duplicate message: %d %s", code, errOut)
	}
	_, _, errOut = r.add(t, "--parent", "REQ-SYS-002", "encoder::CAP-A-9")
	if !strings.Contains(errOut, "encoder::CAP-A-9 is not a published capability or assumption of use of encoder at v0.1.0") {
		t.Fatalf("unpublished message: %s", errOut)
	}
}

// requireCreatedLinkFile removes the link file and requires link add to
// create it and add documents.reqLinks.
func (r linkAddRepo) requireCreatedLinkFile(t *testing.T) {
	t.Helper()
	replaceIn(t, filepath.Join(r.system, "engmod.yml"), textEdit{Old: "  reqLinks: model/req-links.yml\n", New: ""})
	if err := os.Remove(r.links); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := r.add(t, "--evidence", "REQ-SYS-003", "bridge::AOU-B-1")
	if code != 0 || !strings.Contains(out, "added LNK-SYS-003-BRIDGE evidence REQ-SYS-003 -> bridge::AOU-B-1 (v0.1.0, claimed)") {
		t.Fatalf("link add without a link file: %d\n%s\n%s", code, out, errOut)
	}
	if got := read(t, r.links); got != "schemaVersion: 3\nreqLinks:\n- id: LNK-SYS-003-BRIDGE\n  child: bridge::AOU-B-1\n  evidence:\n  - REQ-SYS-003\n  pinned:\n    version: v0.1.0\n  status: claimed\n" {
		t.Fatalf("created link file:\n%s", got)
	}
	if manifest := read(t, filepath.Join(r.system, "engmod.yml")); !strings.Contains(manifest, "  reqLinks: model/req-links.yml\n") {
		t.Fatalf("documents.reqLinks not added:\n%s", manifest)
	}
}

// engws link add writes a claimed link pinned to the dependency version
// (id LNK-<REQ>-<ALIAS>, unique), appended to the link file with every
// other line kept, after checking that the parent exists and the child is
// a published contract item at that version (resolved with the workspace
// off); it refuses a duplicate, an unknown parent, an item that is not
// published and a capability link without a parent, writing nothing. In a
// model without a link file it creates model/req-links.yml and adds
// documents.reqLinks to engmod.yml. The result passes the model check apart
// from link.claimed_only.
// Mutation: skipping the publication check, the duplicate check or the
// parent check, writing a fingerprint or another pin, or rewriting the file
// fails this test.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestLinkAdd(t *testing.T) {
	registry := newLinksRegistry(t)
	registry.publish("child-a", "v0.1.0", nil)
	registry.publish("child-b", "v0.1.0", nil)
	registry.serve()
	_, system := systemWorkspace(t)
	repo := linkAddRepo{system: system, links: filepath.Join(system, "model", "req-links.yml")}

	after := repo.requireAppendedLink(t)
	repo.requireRefusals(t, after)
	repo.requireRefusalMessages(t)
	// A model without a link file: created, documents.reqLinks added.
	repo.requireCreatedLinkFile(t)
}
