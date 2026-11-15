// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package integration

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/trace"
)

// addChildCode gives a fixture child repository Go code and a test for each
// requirement in reqs (REQ-A-001 -> func A001 and TestA001).
func addChildCode(t *testing.T, dir string, reqs ...string) {
	t.Helper()
	var code, tests strings.Builder
	code.WriteString("package child\n")
	tests.WriteString("package child\n\nimport \"testing\"\n")
	for _, r := range reqs {
		name := strings.ReplaceAll(strings.TrimPrefix(r, "REQ-"), "-", "")
		code.WriteString("\n// TRLC-LINKS: " + r + "\nfunc F" + name + "() {}\n")
		tests.WriteString("\n// TRLC-LINKS: " + r + "\nfunc Test" + name + "(t *testing.T) { F" + name + "() }\n")
	}
	testutil.WriteFile(t, filepath.Join(dir, "src", "child.go"), code.String())
	testutil.WriteFile(t, filepath.Join(dir, "src", "child_test.go"), tests.String())
}

// systemMatrix builds the system's trace matrix with the workspace on.
func systemMatrix(t *testing.T, root, repo string) trace.Matrix {
	t.Helper()
	dir := filepath.Join(root, "repos", repo)
	m, _, err := trace.BuildFromFilesWith(filepath.Join(dir, "engmod.yml"), filepath.Join(dir, "model", "requirements.yml"), "", trace.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// traceRow returns the row with id.
func traceRow(t *testing.T, m trace.Matrix, id string) trace.Row {
	t.Helper()
	for _, r := range m.Requirements {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no trace row %s", id)
	return trace.Row{}
}

// E1: a system requirement's trace row lists its linked child items and
// their realizing requirements as children and counts as implemented and
// test linked through the links (satisfiedBy links) when every realizing
// requirement is implemented and test linked in its own repository; a child
// requirement's row lists the contract items it realizes as parents. The
// summary counts the links that need review (version-mismatched, broken,
// claimed) and unparented requirements.
// Mutation: dropping the children, the parents, the satisfiedBy roll-up or
// one of the summary counters fails this test.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func TestRequirementLinksTraceRollUp(t *testing.T) {
	root := copyRequirementLinksExample(t)
	childA := filepath.Join(root, "repos", "child-a")
	addChildCode(t, childA, "REQ-A-001", "REQ-A-002", "REQ-A-003")
	addChildCode(t, filepath.Join(root, "repos", "child-b"), "REQ-B-001")

	assertRolledUpThroughLinks(t, systemMatrix(t, root, "system"))
	assertUntestedRealizerIsNotTestLinked(t, root)
	assertUnimplementedRealizerLeavesAnOrphan(t, root)
	assertLinksNeedingReviewAreCounted(t, root)
	// The child alone: REQ-A-003 now traces to nothing.
	child := systemMatrix(t, root, "child-a")
	if r := traceRow(t, child, "REQ-A-001"); !slices.Equal(r.Parents, []string{"CAP-A-1"}) || len(r.Children) != 0 || child.Summary.Unparented != 1 {
		t.Fatalf("child-a: REQ-A-001 %+v, summary %+v", r, child.Summary)
	}
}

// assertUntestedRealizerIsNotTestLinked checks that a realizing requirement
// without a test is implemented but not test linked.
func assertUntestedRealizerIsNotTestLinked(t *testing.T, root string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(root, "repos", "child-a", "src", "child_test.go"), "package child\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-A-001\nfunc TestA001(t *testing.T) { FA001() }\n\n// TRLC-LINKS: REQ-A-003\nfunc TestA003(t *testing.T) { FA003() }\n")
	m := systemMatrix(t, root, "system")
	if r := traceRow(t, m, "REQ-SYS-001"); r.Status != "implemented" || m.Summary.TestLinked != 1 || len(r.Verifications) != 0 {
		t.Fatalf("REQ-A-002 untested: REQ-SYS-001 %+v, summary %+v", r, m.Summary)
	}
}

// assertUnimplementedRealizerLeavesAnOrphan checks that without code or tests
// for its realizer the requirement is not satisfied.
func assertUnimplementedRealizerLeavesAnOrphan(t *testing.T, root string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(root, "repos", "child-a", "src")); err != nil {
		t.Fatal(err)
	}
	m := systemMatrix(t, root, "system")
	if r := traceRow(t, m, "REQ-SYS-002"); r.Status != "orphan" || r.SatisfiedBy != "" || len(r.Children) != 2 {
		t.Fatalf("REQ-SYS-002 without implemented realizer: %+v", r)
	}
}

// assertRolledUpThroughLinks checks that with every realizer implemented and
// test linked, REQ-SYS-001 is implemented through its two links and nothing
// needs review.
func assertRolledUpThroughLinks(t *testing.T, m trace.Matrix) {
	t.Helper()
	sys1 := traceRow(t, m, "REQ-SYS-001")
	if want := []string{"bridge::CAP-B-1", "bridge::REQ-B-001", "encoder::CAP-A-1", "encoder::REQ-A-001", "encoder::REQ-A-002"}; !slices.Equal(sortedCopy(sys1.Children), want) {
		t.Fatalf("REQ-SYS-001 children %v, want %v", sys1.Children, want)
	}
	assertSystemRequirementImplementedThroughLinks(t, sys1)
	assertRolledUpSummary(t, m)
}

// assertSystemRequirementImplementedThroughLinks checks that the row is
// implemented through its two links and is no child.
func assertSystemRequirementImplementedThroughLinks(t *testing.T, sys1 trace.Row) {
	t.Helper()
	if sys1.Status != "implemented" || sys1.SatisfiedBy != "links" {
		t.Fatalf("REQ-SYS-001 must be implemented through its two links: %+v", sys1)
	}
	if len(sys1.Parents) != 0 || len(sys1.Links) != 2 {
		t.Fatalf("REQ-SYS-001 must be implemented through its two links: %+v", sys1)
	}
}

// assertRolledUpSummary checks the delegated status of a child requirement
// and the summary counters.
func assertRolledUpSummary(t *testing.T, m trace.Matrix) {
	t.Helper()
	if r := traceRow(t, m, "encoder::REQ-A-001"); !slices.Equal(r.Parents, []string{"encoder::CAP-A-1"}) || len(r.Children) != 0 || !implementedAndTested(r.DelegatedStatus) {
		t.Fatalf("encoder::REQ-A-001: %+v (status %+v)", r, r.DelegatedStatus)
	}
	s := m.Summary
	if [3]int{s.Implemented, s.TestLinked, s.Orphan} != [3]int{2, 2, 1} || slices.Contains(m.Orphans, "REQ-SYS-001") {
		t.Fatalf("summary %+v orphans %v", s, m.Orphans)
	}
	if [5]int{s.LinksNeedReview, s.LinksVersionMismatch, s.LinksBroken, s.LinksClaimed, s.Unparented} != [5]int{} {
		t.Fatalf("summary %+v orphans %v", s, m.Orphans)
	}
}

// implementedAndTested reports whether a delegated status is implemented and
// test linked.
func implementedAndTested(status *trace.DelegatedStatus) bool {
	return status != nil && status.Implemented && status.TestLinked
}

// assertLinksNeedingReviewAreCounted checks that version-mismatched, broken
// and claimed links are counted, each link once in linksNeedReview; a
// published release without CAP-A-2 breaks its link (a workspace checkout
// would not, D23).
func assertLinksNeedingReviewAreCounted(t *testing.T, root string) {
	t.Helper()
	childA := filepath.Join(root, "repos", "child-a")
	editFile(t, filepath.Join(childA, "model", "architecture.yml"), textEdit{from: "    - id: CAP-A-2\n      kind: capability\n      note: Protect each position sentence with a checksum.\n", to: ""})
	editFile(t, filepath.Join(childA, "engmod.yml"), textEdit{from: "  - CAP-A-2\n", to: ""})
	editFile(t, filepath.Join(childA, "model", "requirements.yml"), textEdit{from: "  realizes: [CAP-A-2]\n", to: ""})
	links := filepath.Join(root, "repos", "system", "model", "req-links.yml")
	editFile(t, links, textEdit{from: "  status: accepted\n  rationale: The bridge", to: "  status: claimed\n  rationale: The bridge"})
	editFile(t, links, textEdit{from: "  child: encoder::CAP-A-1\n  pinned:\n    version: v0.1.0\n", to: "  child: encoder::CAP-A-1\n  pinned:\n    version: v0.0.9\n"})
	m := systemMatrix(t, root, "system")
	if [4]int{m.Summary.LinksBroken, m.Summary.LinksVersionMismatch, m.Summary.LinksClaimed, m.Summary.LinksNeedReview} != [4]int{0, 1, 1, 2} {
		t.Fatalf("workspace: want 0 broken, 1 version mismatch, 1 claimed, 2 to review: %+v", m.Summary)
	}
	registry := newPublishedChildren(t, root)
	registry.publish("child-a", "v0.1.0")
	registry.publish("child-b", "v0.1.0")
	registry.serve()
	m = systemMatrix(t, root, "system")
	if [4]int{m.Summary.LinksBroken, m.Summary.LinksVersionMismatch, m.Summary.LinksClaimed, m.Summary.LinksNeedReview} != [4]int{1, 1, 1, 3} {
		t.Fatalf("published: want 1 broken, 1 version mismatch, 1 claimed, 3 to review: %+v", m.Summary)
	}
}

// sortedCopy returns a sorted copy of list.
func sortedCopy(list []string) []string {
	out := append([]string(nil), list...)
	slices.Sort(out)
	return out
}

// Only reviewed, unchanged links roll a system requirement up: with every
// realizer implemented and test linked, a claimed link and a link pinned
// to another version each leave the requirement
// unsatisfied (satisfiedBy links-unreviewed, not counted as implemented or
// test linked, still an orphan); the accepted unchanged links still count.
// Mutation: rolling up a claimed or version-mismatched link fails this
// test.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func TestRequirementLinksRollUpNeedsReviewedLinks(t *testing.T) {
	for name, edit := range map[string]func(root string){
		"claimed": func(root string) {
			editFile(t, filepath.Join(root, "repos", "system", "model", "req-links.yml"), textEdit{from: "  status: accepted\n  rationale: The bridge", to: "  status: claimed\n  rationale: The bridge"})
		},
		"version mismatch": func(root string) {
			editFile(t, filepath.Join(root, "repos", "system", "model", "req-links.yml"), textEdit{from: "  child: bridge::CAP-B-1\n  pinned:\n    version: v0.1.0\n", to: "  child: bridge::CAP-B-1\n  pinned:\n    version: v0.0.9\n"})
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := copyRequirementLinksExample(t)
			addChildCode(t, filepath.Join(root, "repos", "child-a"), "REQ-A-001", "REQ-A-002", "REQ-A-003")
			addChildCode(t, filepath.Join(root, "repos", "child-b"), "REQ-B-001")
			edit(root)
			assertOnlyReviewedLinksRollUp(t, systemMatrix(t, root, "system"), name)
		})
	}
}

// assertOnlyReviewedLinksRollUp checks that REQ-SYS-001 (with the link edited
// for the case) is not satisfied while REQ-SYS-002 (accepted, unchanged) is.
func assertOnlyReviewedLinksRollUp(t *testing.T, m trace.Matrix, name string) {
	t.Helper()
	sys1 := traceRow(t, m, "REQ-SYS-001")
	if sys1.SatisfiedBy != "links-unreviewed" || sys1.Status != "orphan" || !slices.Contains(m.Orphans, "REQ-SYS-001") {
		t.Fatalf("REQ-SYS-001 with a %s link: %+v", name, sys1)
	}
	if sys1.LinksImplemented || sys1.LinksTestLinked {
		t.Fatalf("REQ-SYS-001 with a %s link: %+v", name, sys1)
	}
	assertAcceptedLinkRollsUp(t, m)
}

// assertAcceptedLinkRollsUp checks that REQ-SYS-002 is implemented through its
// accepted, unchanged link and is the only requirement counted.
func assertAcceptedLinkRollsUp(t *testing.T, m trace.Matrix) {
	t.Helper()
	sys2 := traceRow(t, m, "REQ-SYS-002")
	if sys2.SatisfiedBy != "links" || sys2.Status != "implemented" {
		t.Fatalf("REQ-SYS-002 (accepted, unchanged): %+v, summary %+v", sys2, m.Summary)
	}
	if m.Summary.Implemented != 1 || m.Summary.TestLinked != 1 {
		t.Fatalf("REQ-SYS-002 (accepted, unchanged): %+v, summary %+v", sys2, m.Summary)
	}
}
