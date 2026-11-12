// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// measurer computes the structural metrics of one function unit by walking its
// body once. Nested named units are skipped (they are measured on their own);
// anonymous closures are walked and count toward the unit.
type measurer struct {
	l   *language
	c   *fileCtx
	cyc int // decision points; the cyclomatic complexity is 1 + cyc
	// maxDepth is the deepest nesting of control structures.
	maxDepth int
	// bumps counts the top-level control structures (depth 1) that contain
	// another one, i.e. reach depth 2 or more; blockMax is the deepest level
	// reached inside the current top-level structure.
	bumps    int
	blockMax int
	// maxBool is the most boolean operators in one boolean expression.
	maxBool int
}

// TRLC-LINKS: REQ-EMG-150, REQ-EMG-159
func measure(l *language, c *fileCtx, body *sitter.Node) measurer {
	m := measurer{l: l, c: c}
	m.walk(body, 0, true)
	return m
}

// isUnit reports whether a node starts a nested function unit of its own.
// TRLC-LINKS: REQ-EMG-150
func isUnit(l *language, c *fileCtx, n *sitter.Node) bool {
	return isUnitKind(l, c, n, n.Kind())
}

// TRLC-LINKS: REQ-EMG-150
func isUnitKind(l *language, c *fileCtx, n *sitter.Node, kind string) bool {
	if !l.unitKinds[kind] {
		return false
	}
	_, ok := l.unit(c, n)
	return ok
}

// TRLC-LINKS: REQ-EMG-150
func (m *measurer) walk(n *sitter.Node, depth int, top bool) {
	kind := n.Kind()
	if !top && isUnitKind(m.l, m.c, n, kind) {
		return
	}
	r := m.l.rules[kind]
	if r.decision {
		m.cyc++
	}
	if r.extra {
		m.cyc += m.l.extra(m.c, n)
	}
	if r.boolOp {
		if ops := m.l.boolOps(m.c, n); ops > 0 {
			m.cyc += ops
			m.boolChain(n)
		}
	}
	enter := r.nest && (m.l.chained == nil || !m.l.chained(n))
	if enter {
		depth++
		m.maxDepth = max(m.maxDepth, depth)
		m.blockMax = max(m.blockMax, depth)
		if depth == 1 {
			m.blockMax = 1
		}
	}
	for i, count := uint(0), n.NamedChildCount(); i < count; i++ {
		if child := n.NamedChild(i); child != nil {
			m.walk(child, depth, false)
		}
	}
	if enter && depth == 1 && m.blockMax >= 2 {
		m.bumps++
	}
}

// boolChain records the size of the boolean expression n heads, when n is its
// root: no boolean-operator ancestor above it once parentheses and negations
// are looked through.
// TRLC-LINKS: REQ-EMG-150
func (m *measurer) boolChain(n *sitter.Node) {
	p := n.Parent()
	for p != nil && m.l.transparent[p.Kind()] {
		p = p.Parent()
	}
	if p != nil && m.l.boolOps(m.c, p) > 0 {
		return
	}
	m.maxBool = max(m.maxBool, m.chainCount(n))
}

// chainCount counts the boolean operators of a boolean expression: those of the
// node and of the operand subtrees reached through boolean operators,
// parentheses and negations.
// TRLC-LINKS: REQ-EMG-150
func (m *measurer) chainCount(n *sitter.Node) int {
	total := m.l.boolOps(m.c, n)
	for i := uint(0); i < n.NamedChildCount(); i++ {
		child := n.NamedChild(i)
		if child != nil && (m.l.transparent[child.Kind()] || m.l.boolOps(m.c, child) > 0) {
			total += m.chainCount(child)
		}
	}
	return total
}
