// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/labeth/engineering-model-go/internal/health"
)

// duplicatedBlockMin is the number of consecutive assertion statements an
// identical block needs to be reported as duplicated.
const duplicatedBlockMin = 4

// assertStmt is one assertion statement of a test; norm is its text without
// comments and with the whitespace between tokens collapsed.
type assertStmt struct {
	norm       string
	start, end int
}

// assertRun is a maximal sequence of consecutive assertion statements in one
// statement list.
type assertRun []assertStmt

// assertionRuns finds the runs in a test body, in source order. Statement lists
// of anonymous closures (t.Run callbacks, it callbacks) are searched too;
// nested named units are not part of the test.
// TRLC-LINKS: REQ-EMG-150
func assertionRuns(l *language, c *fileCtx, body *sitter.Node) []assertRun {
	var runs []assertRun
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if c.expired() {
			return
		}
		if l.containers[n.Kind()] {
			runs = append(runs, runsIn(l, c, n)...)
		}
		for i := uint(0); i < n.NamedChildCount(); i++ {
			if child := n.NamedChild(i); child != nil && !isUnit(l, c, child) {
				visit(child)
			}
		}
	}
	visit(body)
	return runs
}

// TRLC-LINKS: REQ-EMG-150
func runsIn(l *language, c *fileCtx, container *sitter.Node) []assertRun {
	var runs []assertRun
	var cur assertRun
	flush := func() {
		if len(cur) > 0 {
			runs = append(runs, cur)
			cur = nil
		}
	}
	for i := uint(0); i < container.NamedChildCount(); i++ {
		stmt := container.NamedChild(i)
		switch {
		case stmt == nil || isComment(stmt):
		case l.assertion(c, stmt):
			cur = append(cur, assertStmt{
				norm:  normalizedText(stmt, c.src),
				start: int(stmt.StartPosition().Row) + 1,
				end:   int(stmt.EndPosition().Row) + 1,
			})
		default:
			flush()
		}
	}
	flush()
	return runs
}

// normalizedText joins the tokens of a node with single spaces, leaving out
// comments, so formatting and line endings do not matter.
// TRLC-LINKS: REQ-EMG-150
func normalizedText(n *sitter.Node, src []byte) string {
	j := &textJoiner{src: src}
	j.visit(n)
	return j.b.String()
}

// textJoiner collects the tokens of a tree.
type textJoiner struct {
	b   strings.Builder
	src []byte
}

// visit adds the tokens under n.
// TRLC-LINKS: REQ-EMG-150
func (j *textJoiner) visit(n *sitter.Node) {
	if isComment(n) {
		return
	}
	if n.ChildCount() == 0 {
		j.token(n)
		return
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		if child := n.Child(i); child != nil {
			j.visit(child)
		}
	}
}

// token adds the text of a leaf, separated by a space, unless it is blank.
// TRLC-LINKS: REQ-EMG-150
func (j *textJoiner) token(n *sitter.Node) {
	tok := strings.TrimSpace(n.Utf8Text(j.src))
	if tok == "" {
		return
	}
	if j.b.Len() > 0 {
		j.b.WriteByte(' ')
	}
	j.b.WriteString(tok)
}

// longestRun returns the length of the longest run.
// TRLC-LINKS: REQ-EMG-150
func longestRun(runs []assertRun) int {
	longest := 0
	for _, run := range runs {
		longest = max(longest, len(run))
	}
	return longest
}

// duplicateFindings reports, per test (index into runs), the blocks of at least
// duplicatedBlockMin identical consecutive assertion statements that also occur
// in another test of the same file. Overlapping windows are merged into one
// block per run. The result is ordered by test, then by position.
// TRLC-LINKS: REQ-EMG-150
func duplicateFindings(runs [][]assertRun) [][]health.Finding {
	windows := windowOwners(runs)
	out := make([][]health.Finding, len(runs))
	for test, testRuns := range runs {
		for _, run := range testRuns {
			out[test] = append(out[test], runDuplicates(run, windows)...)
		}
	}
	return out
}

// windowOwner says which test first had a window of assertions and whether
// another test has it too.
type windowOwner struct {
	test  int
	multi bool
}

// windowIndex maps the text of each window of assertions to its owner.
type windowIndex map[string]*windowOwner

// windowKey is the text of the window of duplicatedBlockMin assertions of a run
// that starts at an index.
// TRLC-LINKS: REQ-EMG-150
func windowKey(run assertRun, at int) string {
	parts := make([]string, duplicatedBlockMin)
	for i := range parts {
		parts[i] = run[at+i].norm
	}
	return strings.Join(parts, "\n")
}

// windowOwners indexes the windows of every run of every test.
// TRLC-LINKS: REQ-EMG-150
func windowOwners(runs [][]assertRun) windowIndex {
	windows := windowIndex{}
	for test, testRuns := range runs {
		for _, run := range testRuns {
			windows.addRun(test, run)
		}
	}
	return windows
}

// addRun indexes the windows of one run of a test.
// TRLC-LINKS: REQ-EMG-150
func (w windowIndex) addRun(test int, run assertRun) {
	for at := 0; at+duplicatedBlockMin <= len(run); at++ {
		k := windowKey(run, at)
		switch o := w[k]; {
		case o == nil:
			w[k] = &windowOwner{test: test}
		case o.test != test:
			o.multi = true
		}
	}
}

// runDuplicates reports the blocks of a run that also occur in another test,
// overlapping windows merged.
// TRLC-LINKS: REQ-EMG-150
func runDuplicates(run assertRun, windows windowIndex) []health.Finding {
	covered := coveredByOthers(run, windows)
	var out []health.Finding
	for at := 0; at < len(run); at++ {
		if !covered[at] {
			continue
		}
		end := blockEnd(covered, at)
		out = append(out, duplicateFinding(run, at, end))
		at = end
	}
	return out
}

// coveredByOthers marks the statements of a run that lie in a window another
// test has too.
// TRLC-LINKS: REQ-EMG-150
func coveredByOthers(run assertRun, windows windowIndex) []bool {
	covered := make([]bool, len(run))
	for at := 0; at+duplicatedBlockMin <= len(run); at++ {
		if windows[windowKey(run, at)].multi {
			markCovered(covered, at)
		}
	}
	return covered
}

// markCovered marks the window that starts at an index.
// TRLC-LINKS: REQ-EMG-150
func markCovered(covered []bool, at int) {
	for i := 0; i < duplicatedBlockMin; i++ {
		covered[at+i] = true
	}
}

// blockEnd is the index of the last covered statement of the block that starts
// at an index.
// TRLC-LINKS: REQ-EMG-150
func blockEnd(covered []bool, at int) int {
	end := at
	for end+1 < len(covered) && covered[end+1] {
		end++
	}
	return end
}

// duplicateFinding is the finding of the block of a run from one index to
// another.
// TRLC-LINKS: REQ-EMG-150
func duplicateFinding(run assertRun, at, end int) health.Finding {
	return health.Finding{
		Code: health.CodeDuplicatedAssertion, Severity: health.SeverityWarn,
		StartLine: run[at].start, EndLine: run[end].end,
		Metric: "assertions", Value: end - at + 1,
		Message: fmt.Sprintf("%d identical consecutive assertions also occur in another test of this file", end-at+1),
	}
}
