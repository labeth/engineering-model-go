// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// publishEncoderReleases publishes child-b v0.1.0 and child-a v0.1.0, v0.2.0
// (only the derived REQ-A-004 changed) and v0.3.0 (also the realizing
// REQ-A-001 of the linked CAP-A-1 changed).
func publishEncoderReleases(t *testing.T) *linksRegistry {
	t.Helper()
	registry := newLinksRegistry(t)
	registry.publish("child-b", "v0.1.0", nil)
	registry.publish("child-a", "v0.1.0", nil)
	derived := func(dir string) {
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), "shall write one log line.", "shall write one structured log line.")
	}
	registry.publish("child-a", "v0.2.0", derived)
	registry.publish("child-a", "v0.3.0", func(dir string) {
		derived(dir)
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), "shall publish one position sentence.", "shall publish two position sentences.")
	})
	registry.serve()
	return registry
}

// pinOf returns the pinned version and fingerprint of a link in the file.
func pinOf(t *testing.T, path, id string) (string, string) {
	t.Helper()
	var doc model.ReqLinksDocument
	if err := decodeStrict([]byte(read(t, path)), &doc); err != nil {
		t.Fatal(err)
	}
	for _, l := range doc.ReqLinks {
		if l.ID == id {
			return l.Pinned.Version, l.Pinned.Fingerprint
		}
	}
	t.Fatalf("no link %s in %s", id, path)
	return "", ""
}

// bump evaluates the bumping model's links at the new version and prints
// each as unchanged, suspect or broken. D8: an unchanged link (same
// fingerprint) moves to the new version; a suspect link keeps its pin and
// fingerprint. Without --allow-suspect a suspect link aborts the bump and no
// file changes (exit 1); with it the bump is written (exit 0) and the system
// check fails with link.suspect and link.version_mismatch.
// Mutation: not evaluating the links, re-pinning a suspect link, re-stamping
// its fingerprint, writing the bump despite a suspect link or not moving an
// unchanged link fails this test.
// TRLC-LINKS: REQ-EMG-100
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestBumpEvaluatesRequirementLinks(t *testing.T) {
	publishEncoderReleases(t)
	root, system := systemWorkspace(t)
	links := filepath.Join(system, "model", "req-links.yml")
	manifest := filepath.Join(system, "engmod.yml")
	_, fp001 := pinOf(t, links, "LNK-SYS-001-ENCODER")
	_, fp002 := pinOf(t, links, "LNK-SYS-002-ENCODER")

	// A release that changes only a derived requirement: every link stays
	// unchanged and moves to the new version.
	code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "system", "encoder", "v0.2.0")
	if code != 0 {
		t.Fatalf("bump v0.2.0: %d\n%s\n%s", code, out, errOut)
	}
	for _, id := range []string{"LNK-SYS-001-ENCODER", "LNK-SYS-002-ENCODER"} {
		if !strings.Contains(out, "unchanged "+id+" encoder::") {
			t.Fatalf("bump does not print %s unchanged:\n%s", id, out)
		}
		if v, _ := pinOf(t, links, id); v != "v0.2.0" {
			t.Fatalf("%s pinned %s after an unchanged bump", id, v)
		}
	}
	if v, _ := pinOf(t, links, "LNK-SYS-001-BRIDGE"); v != "v0.1.0" {
		t.Fatalf("a link on another dependency moved to %s", v)
	}
	if errs := checkErrors(t, system); len(errs) > 0 {
		t.Fatalf("system check after an unchanged bump:\n%s", strings.Join(errs, "\n"))
	}

	// A release that changes a realizing requirement of CAP-A-1: suspect,
	// exit 1, nothing written.
	before := map[string]string{}
	for _, f := range []string{manifest, links, filepath.Join(system, "engmod.lock.yml")} {
		before[f] = read(t, f)
	}
	code, out, errOut = runMain(t, NewCUERegistry(), "--root", root, "bump", "system", "encoder", "v0.3.0")
	if code != 1 || !strings.Contains(out, "suspect LNK-SYS-001-ENCODER encoder::CAP-A-1") || !strings.Contains(out, "shall publish two position sentences.") ||
		!strings.Contains(out, "unchanged LNK-SYS-002-ENCODER") || !strings.Contains(errOut, "--allow-suspect") {
		t.Fatalf("suspect bump: %d\n%s\n%s", code, out, errOut)
	}
	for f, want := range before {
		if got := read(t, f); got != want {
			t.Fatalf("aborted bump changed %s:\n%s", f, got)
		}
	}

	// --allow-suspect writes the bump; the suspect link keeps its pin and
	// fingerprint, the unchanged one moves, and the check fails.
	code, out, errOut = runMain(t, NewCUERegistry(), "--root", root, "bump", "system", "encoder", "v0.3.0", "--allow-suspect")
	if code != 0 || !strings.Contains(out, "suspect LNK-SYS-001-ENCODER") {
		t.Fatalf("bump --allow-suspect: %d\n%s\n%s", code, out, errOut)
	}
	if !strings.Contains(read(t, manifest), "  version: v0.3.0\n") {
		t.Fatalf("bump not written:\n%s", read(t, manifest))
	}
	if v, fp := pinOf(t, links, "LNK-SYS-001-ENCODER"); v != "v0.2.0" || fp != fp001 {
		t.Fatalf("suspect link re-pinned or re-stamped: %s %s", v, fp)
	}
	if v, fp := pinOf(t, links, "LNK-SYS-002-ENCODER"); v != "v0.3.0" || fp != fp002 {
		t.Fatalf("unchanged link: %s %s", v, fp)
	}
	errs := checkErrors(t, system)
	if !hasLine(errs, "link.suspect", "LNK-SYS-001-ENCODER") || !hasLine(errs, "link.version_mismatch", "LNK-SYS-001-ENCODER") {
		t.Fatalf("system check must fail with link.suspect and link.version_mismatch:\n%s", strings.Join(errs, "\n"))
	}
}
