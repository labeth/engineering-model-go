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
	lcs := reviewTestLCS(b, a)
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

// reviewTestLCS returns the longest-common-subsequence lengths of the
// suffixes of b and a.
func reviewTestLCS(b, a []string) [][]int {
	lcs := make([][]int, len(b)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(a)+1)
	}
	grid := reviewTestGrid{b: b, a: a, cells: lcs}
	for i := len(b) - 1; i >= 0; i-- {
		grid.fillRow(i)
	}
	return lcs
}

// reviewTestGrid is the table of an LCS computation with the lines it compares.
type reviewTestGrid struct {
	b, a  []string
	cells [][]int
}

// fillRow fills row i of the table for the line b[i].
func (g reviewTestGrid) fillRow(i int) {
	for j := len(g.a) - 1; j >= 0; j-- {
		g.cells[i][j] = max(g.cells[i+1][j], g.cells[i][j+1])
		if g.b[i] == g.a[j] {
			g.cells[i][j] = g.cells[i+1][j+1] + 1
		}
	}
}

// Acceptance 3 end to end: after a bump that leaves a link for review,
// review-links lists it on one line, ID only (version_mismatch, the
// versions and what changed by ID), and with --show the parent text and the
// item before (at the pinned version) and after (at the dependency pin);
// --accept by a model actor writes the dependency pin, the reviewer, the
// date, the rationale and status accepted, changing only that link's
// lines, then re-evaluates it, and the system check passes. An unknown
// actor is refused with no change.
// Mutation: not resolving the before version, writing the old pin, skipping
// the actor check, printing the text without --show or rewriting the whole
// file fails this test.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-100
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestReviewLinksShowsAndAcceptsChangedLink(t *testing.T) {
	fixedClock(t)
	publishEncoderReleases(t)
	root, system := systemWorkspace(t)
	links := filepath.Join(system, "model", "req-links.yml")
	// A comment the edit must keep.
	replaceIn(t, links, textEdit{Old: "reqLinks:\n", New: "reqLinks:\n# reviewed links of the position simulator\n"})
	for _, args := range [][]string{{"bump", "system", "encoder", "v0.2.0"}, {"bump", "system", "encoder", "v0.3.0", "--allow-review"}} {
		if code, out, errOut := runMain(t, NewCUERegistry(), append([]string{"--root", root}, args...)...); code != 0 {
			t.Fatalf("%v: %d\n%s\n%s", args, code, out, errOut)
		}
	}
	reviewTestList(t, system)
	reviewTestShow(t, system)
	reviewTestJSON(t, system)
	reviewTestAccept(t, system, links)
}

// reviewTestList checks the one-line listing after the bump.
func reviewTestList(t *testing.T, system string) {
	t.Helper()
	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system)
	want := "LNK-SYS-001-ENCODER encoder::CAP-A-1: version_mismatch (pinned v0.2.0, dependency v0.3.0): changed REQ-A-001 text\n"
	if code != 1 || !strings.Contains(out, want) || strings.Contains(out, "position sentence") {
		t.Fatalf("review-links: %d\n%s\n%s", code, out, errOut)
	}
	if strings.Contains(out, "LNK-SYS-002-ENCODER") || strings.Contains(out, "LNK-SYS-001-BRIDGE") {
		t.Fatalf("review-links lists a link that needs no review:\n%s", out)
	}
}

// reviewTestShow checks --show: parent text and the item before and after.
func reviewTestShow(t *testing.T, system string) {
	t.Helper()
	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--show")
	for _, want := range []string{
		"parent REQ-SYS-001: When simulation epoch elapses, the position simulator shall publish ownship position on the lab bus.",
		"before (v0.2.0)", "after (v0.3.0)",
		"- REQ-A-001: When epoch period elapses, the position encoder shall publish one position sentence.",
		"+ REQ-A-001: When epoch period elapses, the position encoder shall publish two position sentences.",
	} {
		if code != 1 || !strings.Contains(out, want) {
			t.Fatalf("review-links --show lacks %q:\n%s\n%s", want, out, errOut)
		}
	}
}

// reviewTestJSON checks --json carries both items and the changes.
func reviewTestJSON(t *testing.T, system string) {
	t.Helper()
	code, jsonOut, _ := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--json")
	var reviews []LinkReview
	err := json.Unmarshal([]byte(jsonOut), &reviews)
	if code != 1 || err != nil || len(reviews) != 1 {
		t.Fatalf("review-links --json: %d %v\n%s", code, err, jsonOut)
	}
	reviewTestJSONItems(t, reviews[0], jsonOut)
}

// reviewTestJSONItems checks the before and after items and the changes of
// the one review in the --json output.
func reviewTestJSONItems(t *testing.T, r LinkReview, jsonOut string) {
	t.Helper()
	if r.Before == nil || r.After == nil || strings.Join(r.Changes, ",") != "REQ-A-001 text" {
		t.Fatalf("review-links --json: items or changes wrong\n%s", jsonOut)
	}
	if !strings.Contains(r.Before.Realizers[0].Text, "one position sentence") {
		t.Fatalf("review-links --json: before text wrong\n%s", jsonOut)
	}
	if !strings.Contains(r.After.Realizers[0].Text, "two position sentences") {
		t.Fatalf("review-links --json: after text wrong\n%s", jsonOut)
	}
}

// reviewTestAccept checks the actor check, the line-preserving accept and
// the clean check afterwards.
func reviewTestAccept(t *testing.T, system, links string) {
	t.Helper()
	before := read(t, links)
	reviewTestAcceptUnknownActor(t, system, links)
	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-ENCODER", "--by", "ACT-SYSTEM-OWNER", "--rationale", "Two sentences per epoch: still one position per epoch on the bus.")
	if code != 0 || !strings.Contains(out, "accepted LNK-SYS-001-ENCODER encoder::CAP-A-1 at v0.3.0") {
		t.Fatalf("accept: %d\n%s\n%s", code, out, errOut)
	}
	after := read(t, links)
	wantChanged := []string{
		"    version: v0.3.0",
		"  rationale: 'Two sentences per epoch: still one position per epoch on the bus.'",
		`  reviewedOn: "2026-11-03"`,
	}
	if got := changedLines(before, after); strings.Join(got, "\n") != strings.Join(wantChanged, "\n") {
		t.Fatalf("accept changed\n%s\nwant\n%s\nfile:\n%s", strings.Join(got, "\n"), strings.Join(wantChanged, "\n"), after)
	}
	if !strings.Contains(after, "# reviewed links of the position simulator\n") {
		t.Fatalf("comment lost:\n%s", after)
	}
	reviewTestAcceptedClean(t, system)
}

// reviewTestAcceptUnknownActor checks that an actor the model does not define
// is refused with the link file unchanged.
func reviewTestAcceptUnknownActor(t *testing.T, system, links string) {
	t.Helper()
	before := read(t, links)
	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-ENCODER", "--by", "ACT-NOBODY", "--rationale", "Two sentences still satisfy the epoch requirement.")
	if code != 1 || !strings.Contains(errOut, "ACT-NOBODY") || read(t, links) != before {
		t.Fatalf("unknown actor accepted: %d\n%s\n%s", code, out, errOut)
	}
}

// reviewTestAcceptedClean checks that the system check passes and nothing is
// left to review after the accept.
func reviewTestAcceptedClean(t *testing.T, system string) {
	t.Helper()
	if errs := checkErrors(t, system); len(errs) > 0 {
		t.Fatalf("system check after accept:\n%s", strings.Join(errs, "\n"))
	}
	if code, out, _ := runMain(t, NewCUERegistry(), "review-links", "--repo", system); code != 0 || !strings.Contains(out, "no requirement links need review") {
		t.Fatalf("review-links after accept: %d\n%s", code, out)
	}
}

// A broken link is listed but cannot be accepted, and accept on a claimed
// link sets every review field. review-links and accept resolve the
// published dependency (workspace off), never the workspace checkout: with
// the registry unreachable the review says the version is not available
// and accept is refused with nothing written.
// Mutation: accepting a broken link, failing instead of reporting the
// unavailable version, or accepting from the workspace checkout fails this
// test.
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestReviewLinksBrokenAndOffline(t *testing.T) {
	fixedClock(t)
	registry := newLinksRegistry(t)
	dropCapA2 := func(dir string) {
		replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), textEdit{Old: "    - id: CAP-A-2\n      kind: capability\n      note: Protect each position sentence with a checksum.\n", New: ""})
		replaceIn(t, filepath.Join(dir, "engmod.yml"), textEdit{Old: "  - CAP-A-2\n", New: ""})
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), textEdit{Old: "  realizes: [CAP-A-2]\n", New: "  derived: true\n  derivedRationale: Integrity of the encoder output.\n"})
	}
	registry.publish("child-a", "v0.1.0", dropCapA2)
	registry.publish("child-b", "v0.1.0", nil)
	t.Setenv("CUE_REGISTRY", "models.example=127.0.0.1:1+insecure")
	root := t.TempDir()
	copyTree(t, linksExample, root)
	system := filepath.Join(root, "repos", "system")
	links := filepath.Join(system, "model", "req-links.yml")
	// The workspace checkout still has CAP-A-2; the published v0.1.0 does not.
	replaceIn(t, links, textEdit{Old: "  status: accepted\n  rationale: The bridge forwards each sentence onto the lab bus.\n  reviewedBy: ACT-SYSTEM-OWNER\n  reviewedOn: \"2026-11-02\"\n", New: "  status: claimed\n"})

	reviewTestOffline(t, system, links)
	registry.serve()
	reviewTestPublished(t, system, links)
}

// reviewTestOffline checks the review and the refused accept while the
// registry is unreachable.
func reviewTestOffline(t *testing.T, system, links string) {
	t.Helper()
	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system)
	if code != 1 || !strings.Contains(out, "LNK-SYS-001-BRIDGE bridge::CAP-B-1: unresolved, claimed (pinned v0.1.0, dependency v0.1.0): dependency \"bridge\" was not resolved (links are reviewed against published versions") {
		t.Fatalf("review-links offline: %d\n%s\n%s", code, out, errOut)
	}
	before := read(t, links)
	code, out, errOut = runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-BRIDGE", "--by", "ACT-SYSTEM-OWNER", "--rationale", "Reviewed.")
	if code != 1 || !strings.Contains(errOut, "cannot be accepted: unresolved") || read(t, links) != before {
		t.Fatalf("accept without the published dependency: %d\n%s\n%s", code, out, errOut)
	}
}

// reviewTestPublished checks the broken link, the claimed accept and the
// usage error with the registry served.
func reviewTestPublished(t *testing.T, system, links string) {
	t.Helper()
	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system)
	if code != 1 || !strings.Contains(out, "LNK-SYS-002-ENCODER encoder::CAP-A-2: broken (pinned v0.1.0, dependency v0.1.0): the item does not exist at v0.1.0") ||
		!strings.Contains(out, "LNK-SYS-001-BRIDGE bridge::CAP-B-1: unchanged, claimed (pinned v0.1.0, dependency v0.1.0)") {
		t.Fatalf("review-links: %d\n%s\n%s", code, out, errOut)
	}
	before := read(t, links)
	code, out, errOut = runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-002-ENCODER", "--by", "ACT-SYSTEM-OWNER", "--rationale", "x")
	if code != 1 || !strings.Contains(errOut, "is broken: encoder::CAP-A-2 does not exist") || read(t, links) != before {
		t.Fatalf("broken link accepted: %d\n%s\n%s", code, out, errOut)
	}
	reviewTestAcceptClaimed(t, system, links)
}

// reviewTestAcceptClaimed checks that accepting a claimed link adds exactly
// the review lines and that --accept needs --by and --rationale.
func reviewTestAcceptClaimed(t *testing.T, system, links string) {
	t.Helper()
	before := read(t, links)
	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-BRIDGE", "--by", "ACT-SYSTEM-OWNER", "--rationale", "Reviewed.")
	if code != 0 {
		t.Fatalf("accept claimed link: %d\n%s\n%s", code, out, errOut)
	}
	got := addedLines(before, read(t, links))
	want := []string{"  status: accepted", "  rationale: Reviewed.", "  reviewedBy: ACT-SYSTEM-OWNER", `  reviewedOn: "2026-11-03"`}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("accept of a claimed link changed\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if code, _, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-BRIDGE"); code != 2 || !strings.Contains(errOut, "--by") {
		t.Fatalf("--accept without --by and --rationale: %d %s", code, errOut)
	}
}

// --accept re-evaluates the link after writing and fails when it still does
// not pass: accepting a link while the manifest pins a version whose
// realizer set no longer matches is fine, but an accept whose link file
// edit cannot clear the finding (here: the parent requirement is unknown,
// link.parent_unknown) exits 1 naming the finding.
// Mutation: dropping the re-evaluation fails this test.
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestAcceptReevaluatesTheLink(t *testing.T) {
	fixedClock(t)
	registry := newLinksRegistry(t)
	registry.publish("child-a", "v0.1.0", nil)
	registry.publish("child-b", "v0.1.0", nil)
	registry.serve()
	_, system := systemWorkspace(t)
	links := filepath.Join(system, "model", "req-links.yml")
	replaceIn(t, links, textEdit{Old: "- id: LNK-SYS-001-BRIDGE\n  parent: REQ-SYS-001\n", New: "- id: LNK-SYS-001-BRIDGE\n  parent: REQ-SYS-404\n"})
	code, out, errOut := runMain(t, NewCUERegistry(), "review-links", "--repo", system, "--accept", "LNK-SYS-001-BRIDGE", "--by", "ACT-SYSTEM-OWNER", "--rationale", "Reviewed.")
	if code != 1 || !strings.Contains(errOut, "still does not pass: link.parent_unknown") {
		t.Fatalf("accept of a link that still fails: %d\n%s\n%s", code, out, errOut)
	}
}

// A version re-staged with other content (engws stage run again at the same
// version) is refetched: review-links compares each cached module's
// manifest digest with the registry's and refetches it when they differ, so
// the review shows the re-staged content, not the stale cache.
// Mutation: reusing the module cache without the digest check fails this
// test.
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func TestReviewLinksRefetchesARestagedVersion(t *testing.T) {
	first := newLinksRegistry(t)
	first.publish("child-a", "v0.1.0", nil)
	first.publish("child-b", "v0.1.0", nil)
	first.publish("child-a", "v0.2.0", nil)
	first.serve()
	_, system := systemWorkspace(t)
	replaceIn(t, filepath.Join(system, "engmod.yml"), textEdit{Old: "  path: models.example/reqlinks/child-a@v0\n  version: v0.1.0\n", New: "  path: models.example/reqlinks/child-a@v0\n  version: v0.2.0\n"})
	reviews, err := ReviewLinks(system)
	if err != nil || len(reviews) != 2 || len(reviews[0].Changes) != 0 {
		t.Fatalf("first stage: %v %+v", err, reviews)
	}
	second := newLinksRegistry(t)
	second.publish("child-a", "v0.1.0", nil)
	second.publish("child-b", "v0.1.0", nil)
	second.publish("child-a", "v0.2.0", func(dir string) {
		replaceIn(t, filepath.Join(dir, "model", "requirements.yml"), textEdit{Old: "shall publish one position sentence.", New: "shall publish two position sentences."})
	})
	second.serve()
	reviews, err = ReviewLinks(system)
	if err != nil || len(reviews) != 2 || strings.Join(reviews[0].Changes, ",") != "REQ-A-001 text" {
		t.Fatalf("re-staged v0.2.0 not refetched: %v %+v", err, reviews)
	}
}
