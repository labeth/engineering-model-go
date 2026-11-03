// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixedClock makes reviewedOn deterministic.
func fixedClock(t *testing.T) {
	t.Helper()
	old := now
	now = func() time.Time { return time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = old })
}

// changedLines returns the lines of after that differ from the line at the
// same position in before; with a different line count it returns every
// line of after, so a test that expects in-place edits fails.
func changedLines(before, after string) []string {
	b, a := strings.Split(before, "\n"), strings.Split(after, "\n")
	if len(a) != len(b) {
		return a
	}
	var out []string
	for i := range a {
		if a[i] != b[i] {
			out = append(out, a[i])
		}
	}
	return out
}

// addedLines returns the lines of after that are not part of a longest
// common subsequence with before: the lines an edit inserted or rewrote.
func addedLines(before, after string) []string {
	b, a := strings.Split(before, "\n"), strings.Split(after, "\n")
	lcs := make([][]int, len(b)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(a)+1)
	}
	for i := len(b) - 1; i >= 0; i-- {
		for j := len(a) - 1; j >= 0; j-- {
			if b[i] == a[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out []string
	i, j := 0, 0
	for j < len(a) {
		switch {
		case i < len(b) && b[i] == a[j]:
			i++
			j++
		case i < len(b) && lcs[i+1][j] >= lcs[i][j+1]:
			i++
		default:
			out = append(out, a[j])
			j++
		}
	}
	return out
}

// Acceptance 3 end to end: after a bump with --allow-suspect, review-links
// lists the suspect link with the parent requirement text and the child
// item before (at the reviewed version, from the module cache) and after;
// --accept by a model actor writes the current fingerprint, the dependency
// pin, the reviewer, the date, the rationale and status accepted, changing
// only that link's lines, and the system check passes. An unknown actor is
// refused with no change.
// Mutation: not resolving the before version, writing the old pin or the
// recorded fingerprint, skipping the actor check or rewriting the whole file
// fails this test.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-100
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestReviewLinksShowsAndAcceptsSuspectLink(t *testing.T) {
	fixedClock(t)
	publishEncoderReleases(t)
	root, system := systemWorkspace(t)
	links := filepath.Join(system, "model", "req-links.yml")
	// A comment the edit must keep.
	replaceIn(t, links, "reqLinks:\n", "reqLinks:\n# reviewed links of the position simulator\n")
	for _, args := range [][]string{{"bump", "system", "encoder", "v0.2.0"}, {"bump", "system", "encoder", "v0.3.0", "--allow-suspect"}} {
		if code, out, errOut := runMain(t, NewCUERegistry(), append([]string{"--root", root}, args...)...); code != 0 {
			t.Fatalf("%v: %d\n%s\n%s", args, code, out, errOut)
		}
	}

	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system)
	if code != 1 {
		t.Fatalf("review-links with a suspect link: %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{
		"LNK-SYS-001-ENCODER encoder::CAP-A-1: suspect, version_mismatch (pinned v0.2.0, dependency v0.3.0)",
		"parent REQ-SYS-001: When simulation epoch elapses, the position simulator shall publish ownship position on the lab bus.",
		"before (v0.2.0):", "after (v0.3.0):",
		"- REQ-A-001 [", "+ REQ-A-001 [",
		"shall publish one position sentence.", "shall publish two position sentences.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("review-links output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "LNK-SYS-002-ENCODER") || strings.Contains(out, "LNK-SYS-001-BRIDGE") {
		t.Fatalf("review-links lists a link that needs no review:\n%s", out)
	}
	code, jsonOut, _ := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--json")
	var reviews []LinkReview
	if err := json.Unmarshal([]byte(jsonOut), &reviews); code != 1 || err != nil || len(reviews) != 1 || reviews[0].Before == nil || reviews[0].After == nil ||
		!strings.Contains(reviews[0].Before.Realizers[0].Text, "one position sentence") || !strings.Contains(reviews[0].After.Realizers[0].Text, "two position sentences") {
		t.Fatalf("review-links --json: %d %v\n%s", code, err, jsonOut)
	}

	before := read(t, links)
	code, out, errOut = runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-ENCODER", "--by", "ACT-NOBODY", "--rationale", "Two sentences still satisfy the epoch requirement.")
	if code != 1 || !strings.Contains(errOut, "ACT-NOBODY") || read(t, links) != before {
		t.Fatalf("unknown actor accepted: %d\n%s\n%s", code, out, errOut)
	}
	code, out, errOut = runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-ENCODER", "--by", "ACT-SYSTEM-OWNER", "--rationale", "Two sentences per epoch: still one position per epoch on the bus.")
	if code != 0 {
		t.Fatalf("accept: %d\n%s\n%s", code, out, errOut)
	}
	after := read(t, links)
	current := reviews[0].Current
	wantChanged := []string{
		"    version: v0.3.0",
		"    fingerprint: " + current,
		"  rationale: 'Two sentences per epoch: still one position per epoch on the bus.'",
		`  reviewedOn: "2026-11-03"`,
	}
	if got := changedLines(before, after); strings.Join(got, "\n") != strings.Join(wantChanged, "\n") {
		t.Fatalf("accept changed\n%s\nwant\n%s\nfile:\n%s", strings.Join(got, "\n"), strings.Join(wantChanged, "\n"), after)
	}
	if !strings.Contains(after, "# reviewed links of the position simulator\n") {
		t.Fatalf("comment lost:\n%s", after)
	}
	if v, fp := pinOf(t, links, "LNK-SYS-001-ENCODER"); v != "v0.3.0" || fp != current || !strings.HasPrefix(current, "sha256:") {
		t.Fatalf("accepted pin %s %s, want v0.3.0 %s", v, fp, current)
	}
	if errs := checkErrors(t, system); len(errs) > 0 {
		t.Fatalf("system check after accept:\n%s", strings.Join(errs, "\n"))
	}
	if code, out, _ := runMain(t, NewCUERegistry(), "review-links", "--repo", system); code != 0 || !strings.Contains(out, "no requirement links need review") {
		t.Fatalf("review-links after accept: %d\n%s", code, out)
	}
}

// A broken link is listed but cannot be accepted, and accept on a claimed
// link sets every review field. In a workspace whose replacement provides the
// child, the reviewed version cannot be resolved offline: review-links says
// so and shows the current content only. Accept records the published
// dependency at its pin (D9: composed with the workspace off): with the
// registry unreachable it is refused with nothing written, and with the
// registry up it writes the fingerprint of the published release, not that
// of the edited workspace checkout.
// Mutation: accepting a broken link, failing instead of reporting the
// unavailable before version, or accepting from the workspace checkout
// fails this test.
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestReviewLinksBrokenAndOffline(t *testing.T) {
	fixedClock(t)
	registry := newLinksRegistry(t)
	dropCapA2 := func(dir string) {
		replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "    - id: CAP-A-2\n      kind: capability\n      note: Protect each position sentence with a checksum.\n", "")
		replaceIn(t, filepath.Join(dir, "engmod.yml"), "  - CAP-A-2\n", "")
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), "  realizes: [CAP-A-2]\n", "  derived: true\n  derivedRationale: Integrity of the encoder output.\n")
	}
	// The published v0.1.0 differs from the workspace checkout below.
	registry.publish("child-a", "v0.1.0", func(dir string) {
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), "shall publish one position sentence.", "shall publish three position sentences.")
		dropCapA2(dir)
	})
	registry.publish("child-b", "v0.1.0", nil)
	t.Setenv("CUE_REGISTRY", "models.example=127.0.0.1:1+insecure")
	root := t.TempDir()
	copyTree(t, linksExample, root)
	child := filepath.Join(root, "repos", "child-a")
	system := filepath.Join(root, "repos", "system")
	links := filepath.Join(system, "model", "req-links.yml")
	replaceIn(t, filepath.Join(child, "model", "requirements.yml"), "shall publish one position sentence.", "shall publish two position sentences.")
	dropCapA2(child)
	replaceIn(t, links, "  status: accepted\n  rationale: The bridge forwards each sentence onto the lab bus.\n  reviewedBy: ACT-SYSTEM-OWNER\n  reviewedOn: \"2026-11-02\"\n", "  status: claimed\n")

	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system)
	if code != 1 || !strings.Contains(out, "LNK-SYS-002-ENCODER encoder::CAP-A-2: broken") ||
		!strings.Contains(out, "LNK-SYS-001-ENCODER encoder::CAP-A-1: suspect") ||
		!strings.Contains(out, "before: version v0.1.0 not available offline (") || !strings.Contains(out, "shall publish two position sentences.") {
		t.Fatalf("review-links: %d\n%s\n%s", code, out, errOut)
	}
	before := read(t, links)
	code, out, errOut = runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-BRIDGE", "--by", "ACT-SYSTEM-OWNER", "--rationale", "Reviewed.")
	if code != 1 || !strings.Contains(errOut, "workspace off") || read(t, links) != before {
		t.Fatalf("accept without the published dependency: %d\n%s\n%s", code, out, errOut)
	}

	registry.serve()
	code, out, errOut = runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-002-ENCODER", "--by", "ACT-SYSTEM-OWNER", "--rationale", "x")
	if code != 1 || !strings.Contains(errOut, "is broken: encoder::CAP-A-2 no longer exists") || read(t, links) != before {
		t.Fatalf("broken link accepted: %d\n%s\n%s", code, out, errOut)
	}
	code, out, errOut = runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-BRIDGE", "--by", "ACT-SYSTEM-OWNER", "--rationale", "Reviewed.")
	if code != 0 {
		t.Fatalf("accept claimed link: %d\n%s\n%s", code, out, errOut)
	}
	got := addedLines(before, read(t, links))
	want := []string{"  status: accepted", "  rationale: Reviewed.", "  reviewedBy: ACT-SYSTEM-OWNER", `  reviewedOn: "2026-11-03"`}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("accept of a claimed link changed\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// The suspect link: accepted at the published content (three
	// sentences), not the workspace checkout (two).
	reviews, err := ReviewLinks(system)
	if err != nil {
		t.Fatal(err)
	}
	workspaceFingerprint := ""
	for _, r := range reviews {
		if r.ID == "LNK-SYS-001-ENCODER" {
			workspaceFingerprint = r.Current
		}
	}
	if code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-ENCODER", "--by", "ACT-SYSTEM-OWNER", "--rationale", "Reviewed."); code != 0 {
		t.Fatalf("accept suspect link: %d\n%s\n%s", code, out, errOut)
	}
	_, fp := pinOf(t, links, "LNK-SYS-001-ENCODER")
	if fp == workspaceFingerprint || workspaceFingerprint == "" || !strings.HasPrefix(fp, "sha256:v1:") {
		t.Fatalf("accepted fingerprint %s; the workspace checkout's is %s", fp, workspaceFingerprint)
	}
	if code, _, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-BRIDGE"); code != 2 || !strings.Contains(errOut, "--by") {
		t.Fatalf("--accept without --by and --rationale: %d %s", code, errOut)
	}
}
