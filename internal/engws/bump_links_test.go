// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// publishEncoderReleases publishes child-b v0.1.0 and child-a v0.1.0, v0.2.0
// (only the derived REQ-A-004 changed, and the status of the realizing
// REQ-A-001 became verified) and v0.3.0 (also the text of REQ-A-001, a
// realizer of the linked CAP-A-1, changed).
func publishEncoderReleases(t *testing.T) *linksRegistry {
	t.Helper()
	registry := newLinksRegistry(t)
	registry.publish("child-b", "v0.1.0", nil)
	registry.publish("child-a", "v0.1.0", nil)
	v2 := func(dir string) {
		reqs := filepath.Join(dir, "model", "requirements.yml")
		replaceIn(t, reqs, "shall write one log line.", "shall write one structured log line.")
		replaceIn(t, reqs, "  realizes: [CAP-A-1]\n- id: REQ-A-002", "  realizes: [CAP-A-1]\n  status: verified\n  notes: Measured on the bench.\n- id: REQ-A-002")
	}
	registry.publish("child-a", "v0.2.0", v2)
	registry.publish("child-a", "v0.3.0", func(dir string) {
		v2(dir)
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), "shall publish one position sentence.", "shall publish two position sentences.")
	})
	registry.serve()
	return registry
}

// pinOf returns the pinned version of a link in the file and whether the
// link still records a (deprecated) fingerprint.
func pinOf(t *testing.T, path, id string) (string, bool) {
	t.Helper()
	var doc model.ReqLinksDocument
	if err := decodeStrict([]byte(read(t, path)), &doc); err != nil {
		t.Fatal(err)
	}
	for _, l := range doc.ReqLinks {
		if l.ID == id {
			return l.Pinned.Version, l.Pinned.Fingerprint != ""
		}
	}
	t.Fatalf("no link %s in %s", id, path)
	return "", false
}

// bump compares each link on the bumped alias between its pinned version
// and the new one by meaning (D23): a release that changes only an
// unrelated requirement and a realizer's status and notes moves every pin;
// a release that changes a realizer's text leaves that link at its old pin,
// prints it as "review LNK-ID (encoder::CAP-A-1: REQ-A-001 text)" (ID only,
// --show adds the text before and after) and exits 1 with the bump written
// and no other change undone; --allow-review exits 0. The system check then
// fails with link.version_mismatch until the link is reviewed.
// Mutation: comparing status or notes, moving a changed link's pin,
// printing the text without --show or not failing the check fails this
// test.
// TRLC-LINKS: REQ-EMG-100
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestBumpEvaluatesRequirementLinks(t *testing.T) {
	publishEncoderReleases(t)
	root, system := systemWorkspace(t)
	links := filepath.Join(system, "model", "req-links.yml")
	manifest := filepath.Join(system, "engmod.yml")

	code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "system", "encoder", "v0.2.0")
	if code != 0 {
		t.Fatalf("bump v0.2.0: %d\n%s\n%s", code, out, errOut)
	}
	for _, id := range []string{"LNK-SYS-001-ENCODER", "LNK-SYS-002-ENCODER"} {
		if !strings.Contains(out, "moved "+id+" ") {
			t.Fatalf("bump does not print %s moved:\n%s", id, out)
		}
		if v, _ := pinOf(t, links, id); v != "v0.2.0" {
			t.Fatalf("%s pinned %s after a bump without a change in meaning", id, v)
		}
	}
	if v, _ := pinOf(t, links, "LNK-SYS-001-BRIDGE"); v != "v0.1.0" {
		t.Fatalf("a link on another dependency moved to %s", v)
	}
	if errs := checkErrors(t, system); len(errs) > 0 {
		t.Fatalf("system check after an unchanged bump:\n%s", strings.Join(errs, "\n"))
	}

	code, out, errOut = runMain(t, NewCUERegistry(), "--root", root, "bump", "system", "encoder", "v0.3.0")
	if code != 1 || !strings.Contains(out, "review LNK-SYS-001-ENCODER (encoder::CAP-A-1: REQ-A-001 text)") ||
		!strings.Contains(out, "moved LNK-SYS-002-ENCODER") || strings.Contains(out, "two position sentences") {
		t.Fatalf("bump with a changed realizer: %d\n%s\n%s", code, out, errOut)
	}
	if !strings.Contains(read(t, manifest), "  version: v0.3.0\n") {
		t.Fatalf("bump not written:\n%s", read(t, manifest))
	}
	if v, _ := pinOf(t, links, "LNK-SYS-001-ENCODER"); v != "v0.2.0" {
		t.Fatalf("a changed link moved to %s", v)
	}
	if v, _ := pinOf(t, links, "LNK-SYS-002-ENCODER"); v != "v0.3.0" {
		t.Fatalf("an unchanged link stayed at %s", v)
	}
	errs := checkErrors(t, system)
	if !hasLine(errs, "link.version_mismatch", "LNK-SYS-001-ENCODER") || len(errs) != 1 {
		t.Fatalf("system check must fail with link.version_mismatch only:\n%s", strings.Join(errs, "\n"))
	}

	// --show prints the text before and after; --allow-review exits 0.
	replaceIn(t, manifest, "  version: v0.3.0\n", "  version: v0.2.0\n")
	code, out, errOut = runMain(t, NewCUERegistry(), "--root", root, "bump", "system", "encoder", "v0.3.0", "--allow-review", "--show")
	if code != 0 || !strings.Contains(out, "review LNK-SYS-001-ENCODER") || !strings.Contains(out, "- REQ-A-001: When epoch period elapses, the position encoder shall publish one position sentence.") ||
		!strings.Contains(out, "+ REQ-A-001: When epoch period elapses, the position encoder shall publish two position sentences.") {
		t.Fatalf("bump --allow-review --show: %d\n%s\n%s", code, out, errOut)
	}
}

// When the pinned version of a link cannot be resolved (not published), bump
// says so and leaves the link at its pin for review; with the item gone at
// the new version the link is broken and stays too.
// Mutation: moving a link whose old version is unknown, or treating a
// removed item as unchanged, fails this test.
// TRLC-LINKS: REQ-EMG-100
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestBumpUnresolvedAndRemovedItems(t *testing.T) {
	registry := newLinksRegistry(t)
	registry.publish("child-b", "v0.1.0", nil)
	registry.publish("child-a", "v0.2.0", func(dir string) {
		replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "    - id: CAP-A-2\n      kind: capability\n      note: Protect each position sentence with a checksum.\n", "")
		replaceIn(t, filepath.Join(dir, "engmod.yml"), "  - CAP-A-2\n", "")
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), "  realizes: [CAP-A-2]\n", "  derived: true\n  derivedRationale: Integrity of the encoder output.\n")
	})
	registry.serve()
	root, system := systemWorkspace(t)
	links := filepath.Join(system, "model", "req-links.yml")
	code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "system", "encoder", "v0.2.0", "--allow-review")
	if code != 0 || !strings.Contains(out, "review LNK-SYS-001-ENCODER (encoder::CAP-A-1: v0.1.0 not available") ||
		!strings.Contains(out, "broken LNK-SYS-002-ENCODER (encoder::CAP-A-2 does not exist at v0.2.0)") {
		t.Fatalf("bump: %d\n%s\n%s", code, out, errOut)
	}
	for _, id := range []string{"LNK-SYS-001-ENCODER", "LNK-SYS-002-ENCODER"} {
		if v, _ := pinOf(t, links, id); v != "v0.1.0" {
			t.Fatalf("%s moved to %s", id, v)
		}
	}
}

// A bump to a dependency release that declares lifecycle warns when the
// consumer's own check pins an engine older than v0.10.0, which rejects the
// key (composition.load_failed); a consumer on v0.10.0 gets no warning.
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestBumpWarnsOfManifestKeysTheEngineRejects(t *testing.T) {
	registry := newLinksRegistry(t)
	registry.publish("child-b", "v0.1.0", nil)
	registry.publish("child-a", "v0.1.0", nil)
	// The example child-a declares lifecycle (every model must, ADR-EMG-061).
	registry.publish("child-a", "v0.2.0", nil)
	registry.serve()
	for engine, want := range map[string]bool{"v0.9.0": true, "v0.10.0": false} {
		root, system := systemWorkspace(t)
		write(t, filepath.Join(system, ".github", "engmod-check", "go.mod"), "module check\n\ngo 1.25\n\nrequire github.com/labeth/engineering-model-go "+engine+"\n")
		code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "system", "encoder", "v0.2.0")
		if code != 0 {
			t.Fatalf("bump with engine %s: %d\n%s\n%s", engine, code, out, errOut)
		}
		got := strings.Contains(out, "warning: encoder v0.2.0 declares lifecycle") && strings.Contains(out, "engineering-model-go "+engine)
		if got != want {
			t.Fatalf("engine %s: warning %v, want %v:\n%s", engine, got, want, out)
		}
	}
}

// The deprecated flag --allow-suspect is gone: bump rejects it as an unknown
// flag (ADR-EMG-073). Mutation: registering the flag again fails this test.
// TRLC-LINKS: REQ-EMG-182
func TestBumpRejectsTheRemovedAllowSuspectFlag(t *testing.T) {
	code, _, errOut := runMain(t, NewCUERegistry(), "bump", "system", "encoder", "v0.3.0", "--allow-suspect")
	if code == 0 || !strings.Contains(errOut, "allow-suspect") {
		t.Fatalf("bump --allow-suspect must be rejected as an unknown flag: %d\n%s", code, errOut)
	}
}
