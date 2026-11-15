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
		replaceIn(t, reqs, textEdit{Old: "shall write one log line.", New: "shall write one structured log line."})
		replaceIn(t, reqs, textEdit{Old: "  realizes: [CAP-A-1]\n- id: REQ-A-002", New: "  realizes: [CAP-A-1]\n  status: verified\n  notes: Measured on the bench.\n- id: REQ-A-002"})
	}
	registry.publish("child-a", "v0.2.0", v2)
	registry.publish("child-a", "v0.3.0", func(dir string) {
		v2(dir)
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), textEdit{Old: "shall publish one position sentence.", New: "shall publish two position sentences."})
	})
	registry.serve()
	return registry
}

// pinOf returns the pinned version of a link in the file and whether the
// link still records a (removed) fingerprint.
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

// encoderBump is the system workspace the encoder releases are bumped in.
type encoderBump struct{ root, system string }

// bump runs engws bump system encoder version with flags.
func (b encoderBump) bump(t *testing.T, version string, flags ...string) (int, string, string) {
	t.Helper()
	args := append([]string{"--root", b.root, "bump", "system", "encoder", version}, flags...)
	return runMain(t, NewCUERegistry(), args...)
}

// requirePin requires the link to be pinned at want.
func (b encoderBump) requirePin(t *testing.T, id, want string) {
	t.Helper()
	if v, _ := pinOf(t, filepath.Join(b.system, "model", "req-links.yml"), id); v != want {
		t.Fatalf("link %s pinned %s, want %s", id, v, want)
	}
}

// outputHasAll reports whether out contains every part.
func outputHasAll(out string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(out, part) {
			return false
		}
	}
	return true
}

// requireUnchangedMeaningBump bumps to v0.2.0, where no linked item changed
// in meaning: every link on the alias moves, one on another dependency does
// not, and the system check passes.
func (b encoderBump) requireUnchangedMeaningBump(t *testing.T) {
	t.Helper()
	code, out, errOut := b.bump(t, "v0.2.0")
	if code != 0 {
		t.Fatalf("bump v0.2.0: %d\n%s\n%s", code, out, errOut)
	}
	for _, id := range []string{"LNK-SYS-001-ENCODER", "LNK-SYS-002-ENCODER"} {
		if !strings.Contains(out, "moved "+id+" ") {
			t.Fatalf("bump does not print %s moved:\n%s", id, out)
		}
		b.requirePin(t, id, "v0.2.0")
	}
	b.requirePin(t, "LNK-SYS-001-BRIDGE", "v0.1.0")
	if errs := checkErrors(t, b.system); len(errs) > 0 {
		t.Fatalf("system check after an unchanged bump:\n%s", strings.Join(errs, "\n"))
	}
}

// requireChangedRealizerBump bumps to v0.3.0, where a realizer's text
// changed: the changed link keeps its pin and is printed for review by ID
// only, the bump is written and the system check fails with
// link.version_mismatch only.
func (b encoderBump) requireChangedRealizerBump(t *testing.T) {
	t.Helper()
	manifest := filepath.Join(b.system, "engmod.yml")
	code, out, errOut := b.bump(t, "v0.3.0")
	if code != 1 || !outputHasAll(out, "review LNK-SYS-001-ENCODER (encoder::CAP-A-1: REQ-A-001 text)", "moved LNK-SYS-002-ENCODER") || strings.Contains(out, "two position sentences") {
		t.Fatalf("bump with a changed realizer: %d\n%s\n%s", code, out, errOut)
	}
	if !strings.Contains(read(t, manifest), "  version: v0.3.0\n") {
		t.Fatalf("bump not written:\n%s", read(t, manifest))
	}
	b.requirePin(t, "LNK-SYS-001-ENCODER", "v0.2.0")
	b.requirePin(t, "LNK-SYS-002-ENCODER", "v0.3.0")
	errs := checkErrors(t, b.system)
	if !hasLine(errs, "link.version_mismatch", "LNK-SYS-001-ENCODER") || len(errs) != 1 {
		t.Fatalf("system check must fail with link.version_mismatch only:\n%s", strings.Join(errs, "\n"))
	}
}

// requireShowAllowReviewBump bumps again with --show, which prints the text
// before and after, and --allow-review, which exits 0.
func (b encoderBump) requireShowAllowReviewBump(t *testing.T) {
	t.Helper()
	replaceIn(t, filepath.Join(b.system, "engmod.yml"), textEdit{Old: "  version: v0.3.0\n", New: "  version: v0.2.0\n"})
	code, out, errOut := b.bump(t, "v0.3.0", "--allow-review", "--show")
	if code != 0 || !outputHasAll(out, "review LNK-SYS-001-ENCODER",
		"- REQ-A-001: When epoch period elapses, the position encoder shall publish one position sentence.",
		"+ REQ-A-001: When epoch period elapses, the position encoder shall publish two position sentences.") {
		t.Fatalf("bump --allow-review --show: %d\n%s\n%s", code, out, errOut)
	}
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
	b := encoderBump{root: root, system: system}
	b.requireUnchangedMeaningBump(t)
	b.requireChangedRealizerBump(t)
	b.requireShowAllowReviewBump(t)
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
		replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), textEdit{Old: "    - id: CAP-A-2\n      kind: capability\n      note: Protect each position sentence with a checksum.\n", New: ""})
		replaceIn(t, filepath.Join(dir, "engmod.yml"), textEdit{Old: "  - CAP-A-2\n", New: ""})
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), textEdit{Old: "  realizes: [CAP-A-2]\n", New: "  derived: true\n  derivedRationale: Integrity of the encoder output.\n"})
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
