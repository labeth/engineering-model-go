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
// summary counts suspect, broken and claimed links and unparented
// requirements.
// Mutation: dropping the children, the parents, the satisfiedBy roll-up or
// one of the summary counters fails this test.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func TestRequirementLinksTraceRollUp(t *testing.T) {
	root := copyRequirementLinksExample(t)
	childA := filepath.Join(root, "repos", "child-a")
	addChildCode(t, childA, "REQ-A-001", "REQ-A-002", "REQ-A-003")
	addChildCode(t, filepath.Join(root, "repos", "child-b"), "REQ-B-001")

	m := systemMatrix(t, root, "system")
	sys1 := traceRow(t, m, "REQ-SYS-001")
	if want := []string{"bridge::CAP-B-1", "bridge::REQ-B-001", "encoder::CAP-A-1", "encoder::REQ-A-001", "encoder::REQ-A-002"}; !slices.Equal(sortedCopy(sys1.Children), want) {
		t.Fatalf("REQ-SYS-001 children %v, want %v", sys1.Children, want)
	}
	if sys1.Status != "implemented" || sys1.SatisfiedBy != "links" || len(sys1.Parents) != 0 || len(sys1.Links) != 2 {
		t.Fatalf("REQ-SYS-001 must be implemented through its two links: %+v", sys1)
	}
	if r := traceRow(t, m, "encoder::REQ-A-001"); !slices.Equal(r.Parents, []string{"encoder::CAP-A-1"}) || len(r.Children) != 0 ||
		r.DelegatedStatus == nil || !r.DelegatedStatus.Implemented || !r.DelegatedStatus.TestLinked {
		t.Fatalf("encoder::REQ-A-001: %+v (status %+v)", r, r.DelegatedStatus)
	}
	if m.Summary.Implemented != 2 || m.Summary.TestLinked != 2 || m.Summary.Orphan != 1 || slices.Contains(m.Orphans, "REQ-SYS-001") ||
		m.Summary.LinksSuspect != 0 || m.Summary.LinksBroken != 0 || m.Summary.LinksClaimed != 0 || m.Summary.Unparented != 0 {
		t.Fatalf("summary %+v orphans %v", m.Summary, m.Orphans)
	}

	// A realizing requirement without a test: implemented, not test linked.
	testutil.WriteFile(t, filepath.Join(childA, "src", "child_test.go"), "package child\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-A-001\nfunc TestA001(t *testing.T) { FA001() }\n\n// TRLC-LINKS: REQ-A-003\nfunc TestA003(t *testing.T) { FA003() }\n")
	m = systemMatrix(t, root, "system")
	if r := traceRow(t, m, "REQ-SYS-001"); r.Status != "implemented" || m.Summary.TestLinked != 1 || len(r.Verifications) != 0 {
		t.Fatalf("REQ-A-002 untested: REQ-SYS-001 %+v, summary %+v", r, m.Summary)
	}
	// Without code or tests for its realizer the requirement is not satisfied.
	if err := os.RemoveAll(filepath.Join(childA, "src")); err != nil {
		t.Fatal(err)
	}
	m = systemMatrix(t, root, "system")
	if r := traceRow(t, m, "REQ-SYS-002"); r.Status != "orphan" || r.SatisfiedBy != "" || len(r.Children) != 2 {
		t.Fatalf("REQ-SYS-002 without implemented realizer: %+v", r)
	}

	// Suspect, broken and claimed links are counted.
	editFile(t, filepath.Join(childA, "model", "requirements.yml"), "shall publish one position sentence.", "shall publish two position sentences.")
	editFile(t, filepath.Join(childA, "model", "architecture.yml"), "    - id: CAP-A-2\n      kind: capability\n      note: Protect each position sentence with a checksum.\n", "")
	editFile(t, filepath.Join(childA, "engmod.yml"), "  - CAP-A-2\n", "")
	editFile(t, filepath.Join(childA, "model", "requirements.yml"), "  realizes: [CAP-A-2]\n", "")
	editFile(t, filepath.Join(root, "repos", "system", "model", "req-links.yml"), "  status: accepted\n  rationale: The bridge", "  status: claimed\n  rationale: The bridge")
	m = systemMatrix(t, root, "system")
	if m.Summary.LinksSuspect != 1 || m.Summary.LinksBroken != 1 || m.Summary.LinksClaimed != 1 {
		t.Fatalf("want 1 suspect, 1 broken, 1 claimed: %+v", m.Summary)
	}
	// The child alone: REQ-A-003 now traces to nothing.
	child := systemMatrix(t, root, "child-a")
	if r := traceRow(t, child, "REQ-A-001"); !slices.Equal(r.Parents, []string{"CAP-A-1"}) || len(r.Children) != 0 || child.Summary.Unparented != 1 {
		t.Fatalf("child-a: REQ-A-001 %+v, summary %+v", r, child.Summary)
	}
}

// sortedCopy returns a sorted copy of list.
func sortedCopy(list []string) []string {
	out := append([]string(nil), list...)
	slices.Sort(out)
	return out
}
