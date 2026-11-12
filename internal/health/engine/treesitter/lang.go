// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

// The language description the engine walks with. Every language provides one
// table keyed by node kind (decision point, nesting control structure) and a
// few hooks for what a table cannot say (function units, parameters, boolean
// operators, assertion statements). The walker itself knows no language.

import (
	"path/filepath"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// rule says what a node kind contributes to the metrics of the function that
// contains it.
type rule struct {
	// decision adds one decision point to the cyclomatic complexity.
	decision bool
	// nest makes the node a control structure that raises the nesting depth of
	// everything below it (an else-if continuation excepted, see language.chained).
	nest bool
	// extra and boolOp make the walker ask language.extra and language.boolOps
	// about nodes of this kind; other kinds are never asked, which keeps the
	// number of calls into the parser small.
	extra  bool
	boolOp bool
}

// Whether the primitive obsession smell can be computed for a language.
const (
	primitiveSupported   = iota // types are written in the source
	primitiveIfAnnotated        // only when the file annotates parameters
	primitiveUnsupported        // the language has no parameter types
)

// fileCtx is the per-file state hooks need.
type fileCtx struct {
	src    []byte
	path   string
	isTest bool
}

// unitInfo describes one function unit found by language.unit.
type unitInfo struct {
	name string // own name; Go methods are Type.Method
	kind string // function or method
	body *sitter.Node
	test bool
}

// paramStats is the parameter summary of one function.
type paramStats struct {
	count     int
	primitive int
	annotated bool
}

// language is the table and the hooks of one source language.
type language struct {
	name  string
	rules map[string]rule
	// unitKinds lists the node kinds language.unit is asked about.
	unitKinds map[string]bool
	unit      func(c *fileCtx, n *sitter.Node) (unitInfo, bool)
	// scopeKinds lists the node kinds language.scope is asked about; the scope
	// name qualifies the functions below the node (class, impl, describe).
	scopeKinds map[string]bool
	scope      func(c *fileCtx, n *sitter.Node) string
	params     func(c *fileCtx, n *sitter.Node, method bool) paramStats
	// chained reports an else-if continuation: a nesting node that stays on the
	// level of the if it continues.
	chained func(n *sitter.Node) bool
	// extra returns the decision points a node adds beyond its rule.
	extra func(c *fileCtx, n *sitter.Node) int
	// boolOps returns the boolean operators the node itself carries.
	boolOps func(c *fileCtx, n *sitter.Node) int
	// transparent kinds (parentheses, negation) do not end a boolean expression.
	transparent map[string]bool
	// containers hold statement sequences in which assertion runs are searched.
	containers map[string]bool
	assertion  func(c *fileCtx, n *sitter.Node) bool
	// fileIsTest reports whether the whole file is test code.
	fileIsTest func(c *fileCtx, root *sitter.Node) bool
	primitives int
}

// languageFor returns the table of a language name.
// TRLC-LINKS: REQ-EMG-150
func languageFor(name string) (*language, bool) {
	switch name {
	case "go":
		return goLanguage, true
	case "rust":
		return rustLanguage, true
	case "typescript":
		return typescriptLanguage, true
	case "tsx":
		return tsxLanguage, true
	case "javascript":
		return jsLanguage, true
	case "python":
		return pythonLanguage, true
	}
	return nil, false
}

// TRLC-LINKS: REQ-EMG-150
func text(n *sitter.Node, src []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(src)
}

// TRLC-LINKS: REQ-EMG-150
func isComment(n *sitter.Node) bool { return isCommentKind(n.Kind()) }

// TRLC-LINKS: REQ-EMG-150
func isCommentKind(kind string) bool { return strings.Contains(kind, "comment") }

// TRLC-LINKS: REQ-EMG-150
func sameNode(a, b *sitter.Node) bool { return a != nil && b != nil && a.Id() == b.Id() }

// isFieldOf reports whether n is the child of its parent under field.
// TRLC-LINKS: REQ-EMG-150
func isFieldOf(n *sitter.Node, field string) bool {
	p := n.Parent()
	return p != nil && sameNode(p.ChildByFieldName(field), n)
}

// namedChildren returns the named children of n without comments.
// TRLC-LINKS: REQ-EMG-150
func namedChildren(n *sitter.Node) []*sitter.Node {
	count := n.NamedChildCount()
	out := make([]*sitter.Node, 0, count)
	for i := uint(0); i < count; i++ {
		if child := n.NamedChild(i); child != nil && !isComment(child) {
			out = append(out, child)
		}
	}
	return out
}

// operatorOf returns the operator kind of a binary-like node.
// TRLC-LINKS: REQ-EMG-150
func operatorOf(n *sitter.Node) string {
	if op := n.ChildByFieldName("operator"); op != nil {
		return op.Kind()
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-150
func countChildKind(n *sitter.Node, kind string) int {
	total := 0
	for i := uint(0); i < n.ChildCount(); i++ {
		if child := n.Child(i); child != nil && child.Kind() == kind {
			total++
		}
	}
	return total
}

// pathHasDir reports whether a directory component of path equals one of names
// (case-insensitive); the file name itself is not a directory.
// TRLC-LINKS: REQ-EMG-150
func pathHasDir(path string, names ...string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts[:max(len(parts)-1, 0)] {
		for _, name := range names {
			if strings.EqualFold(part, name) {
				return true
			}
		}
	}
	return false
}

// callRoot returns the identifier a call chain starts at (expect(a).not.toBe(b)
// starts at expect) and the property names after it, or "" when the chain does
// not start at an identifier.
// TRLC-LINKS: REQ-EMG-150
func callRoot(n *sitter.Node, src []byte) (string, []string) {
	var props []string
	for cur := n; cur != nil; {
		switch cur.Kind() {
		case "identifier":
			for i, j := 0, len(props)-1; i < j; i, j = i+1, j-1 {
				props[i], props[j] = props[j], props[i]
			}
			return text(cur, src), props
		case "call_expression":
			cur = cur.ChildByFieldName("function")
		case "member_expression":
			props = append(props, text(cur.ChildByFieldName("property"), src))
			cur = cur.ChildByFieldName("object")
		case "await_expression", "parenthesized_expression", "non_null_expression":
			if cur.NamedChildCount() == 0 {
				return "", nil
			}
			cur = cur.NamedChild(0)
		default:
			return "", nil
		}
	}
	return "", nil
}
