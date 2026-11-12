// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	"bytes"
	"sort"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// byteRange is a half-open byte range of the source.
type byteRange struct{ start, end uint }

// lineIndex answers how many logical lines (not blank, not comment) a range of
// rows holds. A row is logical when it is not blank and not wholly inside a
// comment; a row with code and a trailing comment is logical. Python
// docstrings count as comments.
type lineIndex struct {
	prefix []int // prefix[i] is the number of logical rows before row i
}

// newLineIndex classifies every row of src. comments must be sorted by start.
// TRLC-LINKS: REQ-EMG-150
func newLineIndex(src []byte, comments []byteRange) lineIndex {
	rows := bytes.Count(src, []byte{'\n'}) + 1
	prefix := make([]int, rows+1)
	offset := 0
	for r := 0; r < rows; r++ {
		line := src[offset:]
		if end := bytes.IndexByte(line, '\n'); end >= 0 {
			line = line[:end]
		}
		logical := false
		if trimmed := bytes.TrimSpace(line); len(trimmed) > 0 {
			first := offset + bytes.Index(line, trimmed)
			logical = !inComment(comments, uint(first), uint(first+len(trimmed)))
		}
		prefix[r+1] = prefix[r]
		if logical {
			prefix[r+1]++
		}
		offset += len(line) + 1
	}
	return lineIndex{prefix: prefix}
}

// inComment reports whether [first, end) lies inside one comment range.
// TRLC-LINKS: REQ-EMG-150
func inComment(comments []byteRange, first, end uint) bool {
	i := sort.Search(len(comments), func(i int) bool { return comments[i].start > first }) - 1
	return i >= 0 && comments[i].end >= end
}

// count returns the logical lines in rows first..last (zero-based, inclusive).
// TRLC-LINKS: REQ-EMG-150
func (x lineIndex) count(first, last int) int {
	first, last = max(first, 0), min(last, len(x.prefix)-2)
	if last < first {
		return 0
	}
	return x.prefix[last+1] - x.prefix[first]
}

// total returns the logical lines of the whole file.
// TRLC-LINKS: REQ-EMG-150
func (x lineIndex) total() int { return x.prefix[len(x.prefix)-1] }

// pythonDocstring reports a string statement that is the first statement of a
// module, class or function body.
// TRLC-LINKS: REQ-EMG-150
func pythonDocstring(n *sitter.Node) bool {
	if n.NamedChildCount() != 1 || n.NamedChild(0).Kind() != "string" {
		return false
	}
	parent := n.Parent()
	if parent == nil {
		return false
	}
	switch parent.Kind() {
	case "module":
	case "block":
		if owner := parent.Parent(); owner == nil || (owner.Kind() != "function_definition" && owner.Kind() != "class_definition") {
			return false
		}
	default:
		return false
	}
	siblings := namedChildren(parent)
	return len(siblings) > 0 && sameNode(siblings[0], n)
}
