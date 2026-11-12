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
	var b strings.Builder
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if isComment(n) {
			return
		}
		if n.ChildCount() == 0 {
			if tok := strings.TrimSpace(n.Utf8Text(src)); tok != "" {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(tok)
			}
			return
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			if child := n.Child(i); child != nil {
				visit(child)
			}
		}
	}
	visit(n)
	return b.String()
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
	type owner struct {
		test  int
		multi bool
	}
	windows := map[string]*owner{}
	key := func(run assertRun, at int) string {
		parts := make([]string, duplicatedBlockMin)
		for i := range parts {
			parts[i] = run[at+i].norm
		}
		return strings.Join(parts, "\n")
	}
	for test, testRuns := range runs {
		for _, run := range testRuns {
			for at := 0; at+duplicatedBlockMin <= len(run); at++ {
				k := key(run, at)
				switch o := windows[k]; {
				case o == nil:
					windows[k] = &owner{test: test}
				case o.test != test:
					o.multi = true
				}
			}
		}
	}
	out := make([][]health.Finding, len(runs))
	for test, testRuns := range runs {
		for _, run := range testRuns {
			covered := make([]bool, len(run))
			for at := 0; at+duplicatedBlockMin <= len(run); at++ {
				if windows[key(run, at)].multi {
					for i := 0; i < duplicatedBlockMin; i++ {
						covered[at+i] = true
					}
				}
			}
			for at := 0; at < len(run); at++ {
				if !covered[at] {
					continue
				}
				end := at
				for end+1 < len(run) && covered[end+1] {
					end++
				}
				out[test] = append(out[test], health.Finding{
					Code: health.CodeDuplicatedAssertion, Severity: health.SeverityWarn,
					StartLine: run[at].start, EndLine: run[end].end,
					Metric: "assertions", Value: end - at + 1,
					Message: fmt.Sprintf("%d identical consecutive assertions also occur in another test of this file", end-at+1),
				})
				at = end
			}
		}
	}
	return out
}
