// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE

// Package yamledit edits YAML model files line by line: each edit parses the
// document with yaml.v3, finds the node through the node tree and rewrites
// only the lines of that node, so comments, key order, quoting and
// indentation of every other line are kept. Callers apply one edit at a
// time and re-parse.
package yamledit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse returns the root mapping node of a YAML document.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func Parse(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("document is not a mapping")
	}
	return doc.Content[0], nil
}

// Scalar encodes s as a single-line YAML scalar (plain when safe,
// quoted otherwise); runs of whitespace are collapsed.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func Scalar(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	out, err := yaml.Marshal(map[string]string{"k": s})
	if err != nil {
		return fmt.Sprintf("%q", s)
	}
	return strings.TrimSuffix(strings.TrimPrefix(string(out), "k: "), "\n")
}

// KeyRange returns the 0-based line range [start, end) a mapping key and
// its value occupy: the key line and every following line that is blank,
// indented deeper than the key, or (for a sequence value written without
// indentation) a "-" item at the key's indentation. Trailing blank and
// comment lines are not part of the range.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func KeyRange(lines []string, key, value *yaml.Node) (int, int) {
	start := key.Line - 1
	indent := key.Column - 1
	seqItems := value.Kind == yaml.SequenceNode && value.Style&yaml.FlowStyle == 0
	last := start + 1
	for i := start + 1; i < len(lines); i++ {
		blank, inside := rangeLine(lines[i], indent, seqItems)
		if blank {
			continue
		}
		if !inside {
			return start, last
		}
		last = i + 1
	}
	return start, last
}

// rangeLine classifies a line for KeyRange: blank (blank or comment, never
// ends a range by itself) or inside the range of a key at indent.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func rangeLine(line string, indent int, seqItems bool) (blank, inside bool) {
	line = strings.TrimRight(line, "\r\n")
	trimmed := strings.TrimLeft(line, " ")
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return true, false
	}
	depth := len(line) - len(trimmed)
	if depth > indent {
		return false, true
	}
	isDash := trimmed == "-" || strings.HasPrefix(trimmed, "- ")
	return false, depth == indent && seqItems && isDash
}

// SplitLines splits data into lines that keep their newline.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func SplitLines(data []byte) []string {
	var out []string
	for _, l := range bytes.SplitAfter(data, []byte("\n")) {
		if len(l) > 0 {
			out = append(out, string(l))
		}
	}
	return out
}

// Span is the line range [Start, End) of a document.
type Span struct{ Start, End int }

// Finder selects a node of a parsed document.
type Finder func(root *yaml.Node) (*yaml.Node, error)

// KeyAt names Key of the mapping Find selects.
type KeyAt struct {
	Find Finder
	Key  string
}

// FieldAt names the field at Path: keys below the entity Find selects.
type FieldAt struct {
	Find Finder
	Path []string
}

// ListItem names the mapping item with ID under IDKey in the sequence at
// Keys; Sub are keys to walk inside the item.
type ListItem struct {
	Keys  []string
	IDKey string
	ID    string
	Sub   []string
}

// SpliceLines replaces the lines of span with replacement lines.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func SpliceLines(lines []string, span Span, replacement ...string) []byte {
	start, end := span.Start, span.End
	var b strings.Builder
	b.WriteString(strings.Join(lines[:start], ""))
	if start > 0 && !strings.HasSuffix(lines[start-1], "\n") {
		b.WriteString("\n")
	}
	for _, l := range replacement {
		b.WriteString(withNewline(l))
	}
	b.WriteString(strings.Join(lines[end:], ""))
	return []byte(b.String())
}

// withNewline returns l ended by a newline.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func withNewline(l string) string {
	if strings.HasSuffix(l, "\n") {
		return l
	}
	return l + "\n"
}

// locate parses data and returns the node find selects.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func locate(data []byte, find func(root *yaml.Node) (*yaml.Node, error)) (*yaml.Node, error) {
	root, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return find(root)
}

// entryOf returns the key and value nodes of key in the mapping find selects
// (nil nodes when the key is missing).
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func entryOf(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key string) (*yaml.Node, *yaml.Node, error) {
	mapping, err := locate(data, find)
	if err != nil {
		return nil, nil, err
	}
	keyNode, value := MappingEntry(mapping, key)
	return keyNode, value, nil
}

// keyEdit is one SetKey edit: the mapping, the existing entry of the key (nil
// when missing) and the encoded value.
type keyEdit struct {
	lines   []string
	mapping *yaml.Node
	keyNode *yaml.Node
	value   *yaml.Node
	key     string
	raw     string
	flow    bool
	indent  string
}

// prepared fills the entry, flow and indent fields of an edit that has its
// lines, mapping, key and raw value.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func (e keyEdit) prepared() keyEdit {
	e.keyNode, e.value = MappingEntry(e.mapping, e.key)
	e.flow = e.mapping.Style&yaml.FlowStyle != 0
	e.indent = strings.Repeat(" ", e.mapping.Content[0].Column-1)
	if e.keyNode != nil {
		// Keep a "- " item prefix when the key is the first of a list item.
		e.indent = e.lines[e.keyNode.Line-1][:e.keyNode.Column-1]
	}
	return e
}

// entry returns the lines of the key entry.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func (e keyEdit) entry() []string {
	if strings.HasPrefix(e.raw, "\n") {
		return append([]string{e.indent + e.key + ":"}, strings.Split(strings.TrimPrefix(e.raw, "\n"), "\n")...)
	}
	return []string{e.indent + e.key + ": " + e.raw}
}

// needsBlock reports whether the edit needs block syntax: a new key, a value
// that is no scalar or a block value.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func (e keyEdit) needsBlock() bool {
	return e.keyNode == nil || e.value.Kind != yaml.ScalarNode || strings.HasPrefix(e.raw, "\n")
}

// inPlace replaces the single-line scalar value of the key within its line.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func (e keyEdit) inPlace() ([]byte, bool) {
	if strings.Contains(e.raw, "\n") || !singleLineScalar(e.keyNode, e.value) {
		return nil, false
	}
	line := e.lines[e.value.Line-1]
	startCol := e.value.Column - 1
	endCol, ok := ScalarEnd(line, startCol, e.value)
	if !ok {
		return nil, false
	}
	e.lines[e.value.Line-1] = line[:startCol] + e.raw + line[endCol:]
	return []byte(strings.Join(e.lines, "")), true
}

// singleLineScalar reports whether value is a plain or quoted scalar that
// starts on the line of its key.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-101
func singleLineScalar(keyNode, value *yaml.Node) bool {
	if value.Kind != yaml.ScalarNode || value.Line != keyNode.Line {
		return false
	}
	switch value.Style {
	case 0, yaml.DoubleQuotedStyle, yaml.SingleQuotedStyle:
		return true
	}
	return false
}

// apply performs the edit.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func (e keyEdit) apply() ([]byte, error) {
	if e.flow && e.needsBlock() {
		return nil, fmt.Errorf("line %d: %s cannot be set in a flow mapping; rewrite it as a block mapping first", e.mapping.Line, e.key)
	}
	if e.keyNode == nil {
		lastKey := e.mapping.Content[len(e.mapping.Content)-2]
		_, end := KeyRange(e.lines, lastKey, e.mapping.Content[len(e.mapping.Content)-1])
		return SpliceLines(e.lines, Span{end, end}, e.entry()...), nil
	}
	if out, ok := e.inPlace(); ok {
		return out, nil
	}
	if e.flow {
		return nil, fmt.Errorf("line %d: %s cannot be set in place in a flow mapping", e.mapping.Line, e.key)
	}
	start, end := KeyRange(e.lines, e.keyNode, e.value)
	return SpliceLines(e.lines, Span{start, end}, e.entry()...), nil
}

// SetKey sets key of the block mapping found by find to the encoded YAML
// value raw (a single-line scalar or flow collection, or a block starting
// with a newline). An existing single-line scalar is replaced in place,
// keeping a trailing comment; any other existing value is replaced with its
// lines; a missing key is added after the mapping's last key.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func SetKey(data []byte, at KeyAt, raw string) ([]byte, error) {
	key := at.Key
	mapping, err := locate(data, at.Find)
	if err != nil {
		return nil, err
	}
	if mapping.Kind != yaml.MappingNode || len(mapping.Content) == 0 {
		return nil, fmt.Errorf("line %d: %s can only be set in a non-empty mapping", mapping.Line, key)
	}
	return keyEdit{lines: SplitLines(data), mapping: mapping, key: key, raw: raw}.prepared().apply()
}

// SetScalarInPlace replaces the single-line scalar value of key in the
// mapping found by find with value, keeping its quote style (plain, single
// or double) and every other byte of data, in block and flow mappings. It
// refuses a missing key, a value that is no scalar or not on its key's line
// (a block or multi-line scalar), and a value that its quote style cannot
// hold unescaped.
// TRLC-LINKS: REQ-EMG-068
func SetScalarInPlace(data []byte, at KeyAt, value string) ([]byte, error) {
	key := at.Key
	keyNode, node, err := entryOf(data, at.Find, key)
	if err != nil {
		return nil, err
	}
	if keyNode == nil {
		return nil, fmt.Errorf("no %s", key)
	}
	if !singleLineScalar(keyNode, node) {
		return nil, fmt.Errorf("line %d: %s is not a single-line scalar", keyNode.Line, key)
	}
	raw, err := quoteInPlace(node, value)
	if err != nil {
		return nil, err
	}
	lines := SplitLines(data)
	line := lines[node.Line-1]
	end, ok := ScalarEnd(line, node.Column-1, node)
	if !ok {
		return nil, fmt.Errorf("line %d: %s value not found in place", node.Line, key)
	}
	lines[node.Line-1] = line[:node.Column-1] + raw + line[end:]
	return []byte(strings.Join(lines, "")), nil
}

// quoteInPlace returns value written in the quote style of node, or an error
// when the style cannot hold it unescaped.
// TRLC-LINKS: REQ-EMG-068
func quoteInPlace(node *yaml.Node, value string) (string, error) {
	switch node.Style {
	case yaml.SingleQuotedStyle:
		if strings.ContainsAny(value, "'\n") {
			return "", fmt.Errorf("line %d: %q cannot be written single-quoted in place", node.Line, value)
		}
		return "'" + value + "'", nil
	case yaml.DoubleQuotedStyle:
		if strings.ContainsAny(value, "\"\\\n") {
			return "", fmt.Errorf("line %d: %q cannot be written double-quoted in place", node.Line, value)
		}
		return "\"" + value + "\"", nil
	}
	if Scalar(value) != value {
		return "", fmt.Errorf("line %d: %q is not a plain scalar", node.Line, value)
	}
	return value, nil
}

// ScalarEnd returns the column just after a single-line scalar that starts
// at col of line.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101
func ScalarEnd(line string, col int, value *yaml.Node) (int, bool) {
	if col >= len(line) {
		return 0, false
	}
	switch value.Style {
	case yaml.DoubleQuotedStyle:
		return doubleQuotedEnd(line, col)
	case yaml.SingleQuotedStyle:
		return singleQuotedEnd(line, col)
	}
	if strings.HasPrefix(line[col:], value.Value) {
		return col + len(value.Value), true
	}
	return 0, false
}

// doubleQuotedEnd returns the column after the closing quote of the
// double-quoted scalar at col.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101
func doubleQuotedEnd(line string, col int) (int, bool) {
	for i := col + 1; i < len(line); i++ {
		if line[i] == '\\' {
			i++
		} else if line[i] == '"' {
			return i + 1, true
		}
	}
	return 0, false
}

// singleQuotedEnd returns the column after the closing quote of the
// single-quoted scalar at col.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101
func singleQuotedEnd(line string, col int) (int, bool) {
	for i := col + 1; i < len(line); i++ {
		if line[i] != '\'' {
			continue
		}
		if i+1 < len(line) && line[i+1] == '\'' {
			i++
			continue
		}
		return i + 1, true
	}
	return 0, false
}

// RemoveKey removes key and its value lines from the block mapping found by
// find; a missing key is no change.
// TRLC-LINKS: REQ-EMG-104
func RemoveKey(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key string) ([]byte, bool, error) {
	mapping, err := locate(data, find)
	if err != nil {
		return nil, false, err
	}
	keyNode, value := MappingEntry(mapping, key)
	if keyNode == nil {
		return data, false, nil
	}
	if mapping.Style&yaml.FlowStyle != 0 {
		return nil, false, fmt.Errorf("line %d: cannot remove %s from a flow mapping", mapping.Line, key)
	}
	lines := SplitLines(data)
	if prefix := lines[keyNode.Line-1][:keyNode.Column-1]; strings.Contains(prefix, "-") {
		return nil, false, fmt.Errorf("line %d: %s is the first key of a list item; move it below another key first", keyNode.Line, key)
	}
	start, end := KeyRange(lines, keyNode, value)
	return SpliceLines(lines, Span{start, end}), true, nil
}

// AppendSequenceItems appends the block lines items (each item already
// formatted as "- ..." with continuation lines indented by two spaces,
// relative to column 0) to the sequence under key of the mapping found by
// find, at the indentation of its existing items. An empty or flow sequence
// is replaced by a block sequence.
// TRLC-LINKS: REQ-EMG-104
func AppendSequenceItems(data []byte, at KeyAt, items []string) ([]byte, error) {
	key := at.Key
	keyNode, value, err := entryOf(data, at.Find, key)
	if err != nil {
		return nil, err
	}
	if keyNode == nil || value.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("no %s sequence", key)
	}
	if len(value.Content) > 0 && value.Style&yaml.FlowStyle != 0 {
		return nil, fmt.Errorf("line %d: %s is a flow sequence; rewrite it as a block sequence first", keyNode.Line, key)
	}
	lines := SplitLines(data)
	block := indentItems(items, strings.Repeat(" ", itemIndent(keyNode, value)))
	start, end := KeyRange(lines, keyNode, value)
	if len(value.Content) == 0 {
		header := strings.Repeat(" ", keyNode.Column-1) + key + ":"
		return SpliceLines(lines, Span{start, end}, append([]string{header}, block...)...), nil
	}
	return SpliceLines(lines, Span{end, end}, block...), nil
}

// itemIndent returns the indentation of the items of the sequence value of
// keyNode: that of its existing block items, else the key's.
// TRLC-LINKS: REQ-EMG-104
func itemIndent(keyNode, value *yaml.Node) int {
	if len(value.Content) == 0 || value.Style&yaml.FlowStyle != 0 {
		return keyNode.Column - 1
	}
	// The dash sits two columns before the first item's first key.
	return max(value.Content[0].Column-3, 0)
}

// indentItems splits the items into lines prefixed with pad.
// TRLC-LINKS: REQ-EMG-104
func indentItems(items []string, pad string) []string {
	var block []string
	for _, item := range items {
		for _, l := range strings.Split(strings.TrimSuffix(item, "\n"), "\n") {
			block = append(block, pad+l)
		}
	}
	return block
}

// RemoveSequenceItems removes the items at indexes (0-based, any order) of
// the block sequence under key of the mapping found by find, each with its
// lines up to the next item; the other items keep their lines.
// TRLC-LINKS: REQ-EMG-104
func RemoveSequenceItems(data []byte, at KeyAt, indexes []int) ([]byte, error) {
	keyNode, value, err := entryOf(data, at.Find, at.Key)
	if err != nil {
		return nil, err
	}
	if keyNode == nil || !isBlockSequence(value) {
		return nil, fmt.Errorf("no %s block sequence", at.Key)
	}
	lines := SplitLines(data)
	_, end := KeyRange(lines, keyNode, value)
	remove := map[int]bool{}
	for _, i := range indexes {
		remove[i] = true
	}
	kept := itemLines{lines: lines, items: value.Content, end: end}.without(remove)
	return []byte(strings.Join(kept, "")), nil
}

// isBlockSequence reports whether value is a block-style sequence.
// TRLC-LINKS: REQ-EMG-104
func isBlockSequence(value *yaml.Node) bool {
	return value.Kind == yaml.SequenceNode && value.Style&yaml.FlowStyle == 0
}

// itemLines is the lines of a block sequence with its items; end is the line
// after the last item.
type itemLines struct {
	lines []string
	items []*yaml.Node
	end   int
}

// without returns the lines without the items whose index is in remove.
// TRLC-LINKS: REQ-EMG-104
func (il itemLines) without(remove map[int]bool) []string {
	var out []string
	cursor := 0
	for i, item := range il.items {
		start := item.Line - 1
		stop := il.end
		if i+1 < len(il.items) {
			stop = il.items[i+1].Line - 1
		}
		out = append(out, il.lines[cursor:start]...)
		if !remove[i] {
			out = append(out, il.lines[start:stop]...)
		}
		cursor = stop
	}
	return append(out, il.lines[cursor:]...)
}

// FindPath returns a function that walks mapping keys from the root.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func FindPath(keys ...string) func(*yaml.Node) (*yaml.Node, error) {
	return func(node *yaml.Node) (*yaml.Node, error) {
		for _, k := range keys {
			next := MappingValue(node, k)
			if next == nil {
				return nil, fmt.Errorf("no %s", strings.Join(keys, "."))
			}
			node = next
		}
		return node, nil
	}
}

// FindListItem returns a function that finds the mapping item with id in
// the sequence at keys, then walks sub keys inside it.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func FindListItem(at ListItem) Finder {
	keys, idKey, id, sub := at.Keys, at.IDKey, at.ID, at.Sub
	return func(root *yaml.Node) (*yaml.Node, error) {
		seq, err := FindPath(keys...)(root)
		if err != nil {
			return nil, err
		}
		if seq.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("%s is not a list", strings.Join(keys, "."))
		}
		if item := itemWithID(seq, idKey, id); item != nil {
			return FindPath(sub...)(item)
		}
		return nil, fmt.Errorf("no %s entry with %s %s", strings.Join(keys, "."), idKey, id)
	}
}

// itemWithID returns the mapping item of seq whose idKey is id, or nil.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func itemWithID(seq *yaml.Node, idKey, id string) *yaml.Node {
	for _, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		if v := MappingValue(item, idKey); v != nil && strings.TrimSpace(v.Value) == id {
			return item
		}
	}
	return nil
}

// MappingValue returns the value node for key in a mapping node.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func MappingValue(node *yaml.Node, key string) *yaml.Node {
	_, value := MappingEntry(node, key)
	return value
}

// MappingEntry returns the key and value nodes for key in a mapping node.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func MappingEntry(node *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i], node.Content[i+1]
		}
	}
	return nil, nil
}

// FindByID returns a function that finds the one mapping, an item of any
// sequence in the document, whose id is id: a top-level entity (a risk, a
// requirement) or a nested one (a flow step). No match and more than one
// match are errors.
// TRLC-LINKS: REQ-EMG-113
func FindByID(id string) func(*yaml.Node) (*yaml.Node, error) {
	return func(root *yaml.Node) (*yaml.Node, error) {
		var found []*yaml.Node
		var walk func(n *yaml.Node, inSequence bool)
		walk = func(n *yaml.Node, inSequence bool) {
			if n.Kind == yaml.MappingNode && inSequence {
				if v := MappingValue(n, "id"); v != nil && v.Kind == yaml.ScalarNode && strings.TrimSpace(v.Value) == id {
					found = append(found, n)
				}
			}
			for _, c := range n.Content {
				walk(c, n.Kind == yaml.SequenceNode)
			}
		}
		walk(root, false)
		switch len(found) {
		case 0:
			return nil, fmt.Errorf("no entity with id %s in the document", id)
		case 1:
			return found[0], nil
		default:
			lines := make([]string, len(found))
			for i, n := range found {
				lines[i] = fmt.Sprint(n.Line)
			}
			return nil, fmt.Errorf("id %s is not unique in the document (lines %s)", id, strings.Join(lines, ", "))
		}
	}
}

// within returns a function that walks keys (mappings only) below the node
// find returns.
// TRLC-LINKS: REQ-EMG-113
func within(find func(*yaml.Node) (*yaml.Node, error), keys []string) func(*yaml.Node) (*yaml.Node, error) {
	return func(root *yaml.Node) (*yaml.Node, error) {
		node, err := find(root)
		if err != nil {
			return nil, err
		}
		for i, k := range keys {
			if node.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("line %d: %s is not a mapping", node.Line, strings.Join(keys[:i], "."))
			}
			next := MappingValue(node, k)
			if next == nil {
				return nil, fmt.Errorf("no %s", strings.Join(keys[:i+1], "."))
			}
			node = next
		}
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("line %d: %s is not a mapping", node.Line, strings.Join(keys, "."))
		}
		return node, nil
	}
}

// valueNode converts a JSON-decoded value (string, bool, number, list,
// object) to a YAML node; object keys are written id first, then sorted.
// TRLC-LINKS: REQ-EMG-113
func valueNode(v any) (*yaml.Node, error) {
	switch x := v.(type) {
	case []string:
		return sequenceNode(stringsAsAny(x))
	case []any:
		return sequenceNode(x)
	case map[string]any:
		return mappingNode(x)
	}
	return scalarNode(v)
}

// scalarNode converts a JSON-decoded scalar (string, bool, number) to a YAML
// node.
// TRLC-LINKS: REQ-EMG-113
func scalarNode(v any) (*yaml.Node, error) {
	switch x := v.(type) {
	case nil:
		return nil, fmt.Errorf("a null value cannot be set; unset the field instead")
	case string:
		return stringNode(x), nil
	case bool, int, int64, float64:
		var n yaml.Node
		if err := n.Encode(x); err != nil {
			return nil, err
		}
		return &n, nil
	}
	return nil, fmt.Errorf("unsupported value type %T", v)
}

// objectKeys returns the keys of x: id first when present, then the others
// sorted.
// TRLC-LINKS: REQ-EMG-113
func objectKeys(x map[string]any) []string {
	keys := make([]string, 0, len(x))
	for k := range x {
		if k != "id" {
			keys = append(keys, k)
		}
	}
	sortStrings(keys)
	if _, ok := x["id"]; ok {
		keys = append([]string{"id"}, keys...)
	}
	return keys
}

// stringNode returns a string scalar node.
// TRLC-LINKS: REQ-EMG-113
func stringNode(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
}

// stringsAsAny converts a string list to a list of any.
// TRLC-LINKS: REQ-EMG-113
func stringsAsAny(x []string) []any {
	items := make([]any, len(x))
	for i, s := range x {
		items[i] = s
	}
	return items
}

// sequenceNode converts a list to a YAML sequence node.
// TRLC-LINKS: REQ-EMG-113
func sequenceNode(x []any) (*yaml.Node, error) {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, item := range x {
		n, err := valueNode(item)
		if err != nil {
			return nil, err
		}
		seq.Content = append(seq.Content, n)
	}
	return seq, nil
}

// mappingNode converts an object to a YAML mapping node, id key first, then
// the other keys sorted.
// TRLC-LINKS: REQ-EMG-113
func mappingNode(x map[string]any) (*yaml.Node, error) {
	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, k := range objectKeys(x) {
		n, err := valueNode(x[k])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
		m.Content = append(m.Content, stringNode(k), n)
	}
	return m, nil
}

// TRLC-LINKS: REQ-EMG-113
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// encodeLines renders node as YAML with two-space indentation, one string
// per line without newline.
// TRLC-LINKS: REQ-EMG-113
func encodeLines(node *yaml.Node) ([]string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n"), nil
}

// rawValue encodes value for SetKey under a key whose mapping keys are
// indented by indent columns: a single-line scalar or flow value, a block
// scalar header with its lines, or (starting with a newline) a block.
// TRLC-LINKS: REQ-EMG-113
func rawValue(key string, value any, indent int) (string, error) {
	node, err := valueNode(value)
	if err != nil {
		return "", err
	}
	lines, err := encodeLines(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, node}})
	if err != nil {
		return "", err
	}
	head := strings.TrimPrefix(strings.TrimPrefix(lines[0], Scalar(key)), ":")
	pad := strings.Repeat(" ", indent)
	rest := make([]string, 0, len(lines)-1)
	for _, l := range lines[1:] {
		rest = append(rest, pad+l)
	}
	if head == "" {
		return "\n" + strings.Join(rest, "\n"), nil
	}
	return strings.Join(append([]string{strings.TrimPrefix(head, " ")}, rest...), "\n"), nil
}

// keyIndent returns the indentation of the keys of a block mapping.
// TRLC-LINKS: REQ-EMG-113
func keyIndent(mapping *yaml.Node) int {
	if len(mapping.Content) > 0 {
		return mapping.Content[0].Column - 1
	}
	return mapping.Column - 1
}

// SetPath sets the field at path (keys below the entity find returns, for
// example acceptedBy.statement) to value, a JSON-decoded string, bool,
// number, list or object. A single-line scalar is replaced in place; a
// missing key is added after the last key of its mapping; a missing
// intermediate mapping is created with the rest of the path.
// TRLC-LINKS: REQ-EMG-113
func SetPath(data []byte, at FieldAt, value any) ([]byte, error) {
	find, path := at.Find, at.Path
	if len(path) == 0 {
		return nil, fmt.Errorf("empty field path")
	}
	node, err := locate(data, find)
	if err != nil {
		return nil, err
	}
	node, depth, err := existingPrefix(node, path)
	if err != nil {
		return nil, err
	}
	// Build the missing part of the path as nested objects.
	for i := len(path) - 1; i > depth; i-- {
		value = map[string]any{path[i]: value}
	}
	raw, err := rawValue(path[depth], value, keyIndent(node))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.Join(path, "."), err)
	}
	return SetKey(data, KeyAt{within(find, path[:depth]), path[depth]}, raw)
}

// existingPrefix walks the mappings of path below node as far as they exist
// and returns the mapping that holds the first missing key (or the last key)
// with its depth in path.
// TRLC-LINKS: REQ-EMG-113
func existingPrefix(node *yaml.Node, path []string) (*yaml.Node, int, error) {
	depth := 0
	for ; depth < len(path)-1; depth++ {
		if node.Kind != yaml.MappingNode {
			return nil, 0, fmt.Errorf("line %d: %s is not a mapping", node.Line, strings.Join(path[:depth], "."))
		}
		next := MappingValue(node, path[depth])
		if next == nil {
			break
		}
		node = next
	}
	if node.Kind != yaml.MappingNode {
		return nil, 0, fmt.Errorf("line %d: %s is not a mapping", node.Line, strings.Join(path[:depth], "."))
	}
	return node, depth, nil
}

// UnsetPath removes the field at path with its lines; a missing field is an
// error.
// TRLC-LINKS: REQ-EMG-113
func UnsetPath(data []byte, find func(*yaml.Node) (*yaml.Node, error), path []string) ([]byte, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty field path")
	}
	out, changed, err := RemoveKey(data, within(find, path[:len(path)-1]), path[len(path)-1])
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, fmt.Errorf("no field %s to unset", strings.Join(path, "."))
	}
	return out, nil
}

// sequenceAt returns the mapping holding the last key of path, and the key
// and value nodes of that key (nil when the key is missing).
// TRLC-LINKS: REQ-EMG-113
func sequenceAt(data []byte, find func(*yaml.Node) (*yaml.Node, error), path []string) (*yaml.Node, *yaml.Node, *yaml.Node, error) {
	if len(path) == 0 {
		return nil, nil, nil, fmt.Errorf("empty field path")
	}
	root, err := Parse(data)
	if err != nil {
		return nil, nil, nil, err
	}
	mapping, err := within(find, path[:len(path)-1])(root)
	if err != nil {
		return nil, nil, nil, err
	}
	key, value := MappingEntry(mapping, path[len(path)-1])
	if key != nil && !listOrNull(value) {
		return nil, nil, nil, fmt.Errorf("line %d: %s is not a list", key.Line, strings.Join(path, "."))
	}
	return mapping, key, value, nil
}

// listOrNull reports whether n is a sequence or an empty (null) value.
// TRLC-LINKS: REQ-EMG-113
func listOrNull(n *yaml.Node) bool {
	return n.Kind == yaml.SequenceNode || (n.Kind == yaml.ScalarNode && n.Tag == "!!null")
}

// flowItemsRange returns the column range [start, end) of the one-line flow
// sequence value on its line, brackets included.
// TRLC-LINKS: REQ-EMG-113
func flowItemsRange(line string, value *yaml.Node) (int, int, bool) {
	start := value.Column - 1
	if start >= len(line) || line[start] != '[' || !scalarsOnLine(value) {
		return 0, 0, false
	}
	end, ok := flowClose(line, start)
	if !ok {
		return 0, 0, false
	}
	return start, end, true
}

// scalarsOnLine reports whether every item of the sequence value is a scalar
// on the line of the sequence start.
// TRLC-LINKS: REQ-EMG-113
func scalarsOnLine(value *yaml.Node) bool {
	for _, item := range value.Content {
		if item.Kind != yaml.ScalarNode || item.Line != value.Line {
			return false
		}
	}
	return true
}

// flowClose returns the column after the "]" that closes the flow sequence
// opened at start, false when it holds a nested collection or does not
// close on the line.
// TRLC-LINKS: REQ-EMG-113
func flowClose(line string, start int) (int, bool) {
	quote := byte(0)
	for i := start + 1; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			quote, i = stepQuoted(quote, c, i)
		case isQuote(c):
			quote = c
		case isOpener(c):
			return 0, false
		case c == ']':
			return i + 1, true
		}
	}
	return 0, false
}

// isQuote reports whether c opens a quoted flow item.
// TRLC-LINKS: REQ-EMG-113
func isQuote(c byte) bool { return c == '"' || c == '\'' }

// isOpener reports whether c opens a nested flow collection.
// TRLC-LINKS: REQ-EMG-113
func isOpener(c byte) bool { return c == '[' || c == '{' }

// stepQuoted advances inside a quoted flow item: it returns the open quote
// (0 once closed) and the index of the last character consumed.
// TRLC-LINKS: REQ-EMG-113
func stepQuoted(quote, c byte, i int) (byte, int) {
	if c == quote {
		return 0, i
	}
	if c == '\\' && quote == '"' {
		return quote, i + 1
	}
	return quote, i
}

// rewriteFlow replaces a one-line flow sequence with items (raw scalars).
// TRLC-LINKS: REQ-EMG-113
func rewriteFlow(data []byte, value *yaml.Node, items []string) ([]byte, error) {
	lines := SplitLines(data)
	line := lines[value.Line-1]
	start, end, ok := flowItemsRange(line, value)
	if !ok {
		return nil, fmt.Errorf("line %d: the flow list spans lines or holds collections; rewrite it as a block list first", value.Line)
	}
	lines[value.Line-1] = line[:start] + "[" + strings.Join(items, ", ") + "]" + line[end:]
	return []byte(strings.Join(lines, "")), nil
}

// flowRaw returns the raw text of each scalar item of a one-line flow
// sequence.
// TRLC-LINKS: REQ-EMG-113
func flowRaw(line string, value *yaml.Node) ([]string, error) {
	out := make([]string, 0, len(value.Content))
	for _, item := range value.Content {
		end, ok := ScalarEnd(line, item.Column-1, item)
		if !ok {
			return nil, fmt.Errorf("line %d: flow list item %q not found in place", item.Line, item.Value)
		}
		out = append(out, line[item.Column-1:end])
	}
	return out, nil
}

// listRef is a flow list in a document: the data, the list node and its
// dotted field path (for messages).
type listRef struct {
	data  []byte
	value *yaml.Node
	name  string
}

// raw returns the raw text of the items of the one-line flow list.
// TRLC-LINKS: REQ-EMG-113
func (r listRef) raw() ([]string, error) {
	return flowRaw(SplitLines(r.data)[r.value.Line-1], r.value)
}

// AppendPath appends items to the list at path, creating the list when the
// field is missing. A block list gets block items at the indentation of its
// items; a one-line flow list of scalars stays a flow list.
// TRLC-LINKS: REQ-EMG-113
func AppendPath(data []byte, at FieldAt, items []any) ([]byte, error) {
	find, path := at.Find, at.Path
	if len(items) == 0 {
		return data, nil
	}
	_, key, value, err := sequenceAt(data, find, path)
	if err != nil {
		return nil, err
	}
	if key == nil || value.Kind != yaml.SequenceNode {
		return SetPath(data, at, items)
	}
	if isFlowList(value) {
		return appendFlow(listRef{data: data, value: value, name: strings.Join(path, ".")}, items)
	}
	block, err := blockItems(items)
	if err != nil {
		return nil, err
	}
	return AppendSequenceItems(data, KeyAt{within(find, path[:len(path)-1]), path[len(path)-1]}, block)
}

// isFlowList reports whether value is a non-empty flow sequence.
// TRLC-LINKS: REQ-EMG-113
func isFlowList(value *yaml.Node) bool {
	return value.Style&yaml.FlowStyle != 0 && len(value.Content) > 0
}

// appendFlow appends scalar items to a one-line flow list.
// TRLC-LINKS: REQ-EMG-113
func appendFlow(ref listRef, items []any) ([]byte, error) {
	raw, err := ref.raw()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		text, err := flowItemText(ref, item)
		if err != nil {
			return nil, err
		}
		raw = append(raw, text)
	}
	return rewriteFlow(ref.data, ref.value, raw)
}

// flowItemText encodes item as the text of a flow list item; only scalars
// can be one.
// TRLC-LINKS: REQ-EMG-113
func flowItemText(ref listRef, item any) (string, error) {
	n, err := valueNode(item)
	if err != nil {
		return "", err
	}
	if n.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("line %d: %s is a flow list; only scalars can be appended to it", ref.value.Line, ref.name)
	}
	lines, err := encodeLines(n)
	if err != nil {
		return "", err
	}
	return lines[0], nil
}

// blockItems encodes each item as the "- " block lines of a sequence item.
// TRLC-LINKS: REQ-EMG-113
func blockItems(items []any) ([]string, error) {
	block := make([]string, 0, len(items))
	for _, item := range items {
		n, err := valueNode(item)
		if err != nil {
			return nil, err
		}
		lines, err := encodeLines(&yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{n}})
		if err != nil {
			return nil, err
		}
		block = append(block, strings.Join(lines, "\n"))
	}
	return block, nil
}

// itemMatches reports whether a list item is value: a scalar equal to it, or
// a mapping whose id is it.
// TRLC-LINKS: REQ-EMG-113
func itemMatches(item *yaml.Node, value string) bool {
	switch item.Kind {
	case yaml.ScalarNode:
		return item.Value == value
	case yaml.MappingNode:
		id := MappingValue(item, "id")
		return id != nil && id.Kind == yaml.ScalarNode && id.Value == value
	}
	return false
}

// RemovePath removes from the list at path every item equal to one of
// values (a scalar item, or an object item by its id). A value no item
// matches is an error; a list left empty is written as [].
// TRLC-LINKS: REQ-EMG-113
func RemovePath(data []byte, at FieldAt, values []any) ([]byte, error) {
	find, path := at.Find, at.Path
	value, err := existingList(data, find, path)
	if err != nil {
		return nil, err
	}
	name := strings.Join(path, ".")
	remove, err := matchItems(value, values, name)
	if err != nil {
		return nil, err
	}
	parent := KeyAt{within(find, path[:len(path)-1]), path[len(path)-1]}
	if len(remove) == len(value.Content) {
		return SetKey(data, parent, "[]")
	}
	if value.Style&yaml.FlowStyle != 0 {
		return removeFlow(listRef{data: data, value: value, name: name}, remove)
	}
	return RemoveSequenceItems(data, parent, indexesOf(remove))
}

// existingList returns the sequence node at path; a missing field or a null
// value is an error.
// TRLC-LINKS: REQ-EMG-113
func existingList(data []byte, find func(*yaml.Node) (*yaml.Node, error), path []string) (*yaml.Node, error) {
	_, key, value, err := sequenceAt(data, find, path)
	if err != nil {
		return nil, err
	}
	if key == nil || value.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("no list %s", strings.Join(path, "."))
	}
	return value, nil
}

// indexesOf returns the indexes in set.
// TRLC-LINKS: REQ-EMG-113
func indexesOf(set map[int]bool) []int {
	indexes := make([]int, 0, len(set))
	for i := range set {
		indexes = append(indexes, i)
	}
	return indexes
}

// matchItems returns the indexes of the items of the list value that equal
// one of values; a value no item matches is an error.
// TRLC-LINKS: REQ-EMG-113
func matchItems(value *yaml.Node, values []any, name string) (map[int]bool, error) {
	remove := map[int]bool{}
	for _, v := range values {
		s, err := itemName(v, name)
		if err != nil {
			return nil, err
		}
		matched := matchingIndexes(value.Content, s)
		if len(matched) == 0 {
			return nil, fmt.Errorf("list %s has no item %q", name, s)
		}
		for _, i := range matched {
			remove[i] = true
		}
	}
	return remove, nil
}

// itemName returns the scalar value or id that names a list item to remove.
// TRLC-LINKS: REQ-EMG-113
func itemName(v any, name string) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	n, err := valueNode(v)
	if err != nil || n.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("remove %s: name an item by its scalar value or id", name)
	}
	return n.Value, nil
}

// matchingIndexes returns the indexes of the items that are value.
// TRLC-LINKS: REQ-EMG-113
func matchingIndexes(items []*yaml.Node, value string) []int {
	var out []int
	for i, item := range items {
		if itemMatches(item, value) {
			out = append(out, i)
		}
	}
	return out
}

// removeFlow rewrites a one-line flow list without the items in remove.
// TRLC-LINKS: REQ-EMG-113
func removeFlow(ref listRef, remove map[int]bool) ([]byte, error) {
	raw, err := ref.raw()
	if err != nil {
		return nil, err
	}
	kept := []string{}
	for i, r := range raw {
		if !remove[i] {
			kept = append(kept, r)
		}
	}
	return rewriteFlow(ref.data, ref.value, kept)
}

// diffOp is one line of a diff: kind is ' ' (kept), '-' (removed) or '+'
// (added); ai and bi index the lines of before and after.
type diffOp struct {
	kind byte
	ai   int
	bi   int
}

// diffContext is the number of context lines around a change.
const diffContext = 3

// UnifiedDiff returns the unified diff (three lines of context, git apply
// format) of before and after under name, or "" when they are equal.
// TRLC-LINKS: REQ-EMG-113
func UnifiedDiff(name string, before, after []byte) string {
	if bytes.Equal(before, after) {
		return ""
	}
	d := diffText{a: diffLines(before), b: diffLines(after)}
	ops := d.ops()
	var out strings.Builder
	fmt.Fprintf(&out, "--- a/%s\n+++ b/%s\n", name, name)
	for k := nextChange(ops, 0); k < len(ops); {
		end := hunkEnd(ops, k)
		out.WriteString(d.hunk(ops[max(k-diffContext, 0):end]))
		k = nextChange(ops, end)
	}
	return out.String()
}

// diffLines splits b into lines for diffing.
// TRLC-LINKS: REQ-EMG-113
func diffLines(b []byte) []string {
	s := strings.TrimSuffix(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// nextChange returns the index of the first changed op at or after from.
// TRLC-LINKS: REQ-EMG-113
func nextChange(ops []diffOp, from int) int {
	for from < len(ops) && ops[from].kind == ' ' {
		from++
	}
	return from
}

// spaceRun returns the index after the run of kept ops starting at from.
// TRLC-LINKS: REQ-EMG-113
func spaceRun(ops []diffOp, from int) int {
	for from < len(ops) && ops[from].kind == ' ' {
		from++
	}
	return from
}

// hunkEnd returns the end of the hunk whose first change is at k: changes
// separated by at most twice the context share a hunk.
// TRLC-LINKS: REQ-EMG-113
func hunkEnd(ops []diffOp, k int) int {
	end := k
	for end < len(ops) {
		if ops[end].kind != ' ' {
			end++
			continue
		}
		run := spaceRun(ops, end)
		if run == len(ops) || run-end > 2*diffContext {
			return min(end+diffContext, len(ops))
		}
		end = run
	}
	return end
}

// hunkRange formats the start and count of one side of a hunk header.
// TRLC-LINKS: REQ-EMG-113
func hunkRange(start, count int) string {
	switch count {
	case 0:
		return fmt.Sprintf("%d,0", start)
	case 1:
		return fmt.Sprint(start + 1)
	}
	return fmt.Sprintf("%d,%d", start+1, count)
}

// diffText is the two line lists a diff compares.
type diffText struct {
	a []string
	b []string
}

// ops returns the line operations that turn a into b: the common prefix and
// suffix are kept and the middle is diffed by the longest common subsequence.
// TRLC-LINKS: REQ-EMG-113
func (d diffText) ops() []diffOp {
	a, b := d.a, d.b
	// Trim the common prefix and suffix before the quadratic LCS.
	pre := d.commonPrefix()
	suf := d.commonSuffix(pre)
	var ops []diffOp
	for i := 0; i < pre; i++ {
		ops = append(ops, diffOp{' ', i, i})
	}
	ops = append(ops, newLCS(a[pre:len(a)-suf], b[pre:len(b)-suf]).ops(pre)...)
	for k := 0; k < suf; k++ {
		ops = append(ops, diffOp{' ', len(a) - suf + k, len(b) - suf + k})
	}
	return ops
}

// commonPrefix returns the number of leading lines a and b share.
// TRLC-LINKS: REQ-EMG-113
func (d diffText) commonPrefix() int {
	pre := 0
	for pre < len(d.a) && pre < len(d.b) && d.a[pre] == d.b[pre] {
		pre++
	}
	return pre
}

// commonSuffix returns the number of trailing lines a and b share after
// their first pre lines.
// TRLC-LINKS: REQ-EMG-113
func (d diffText) commonSuffix(pre int) int {
	suf := 0
	for suf < len(d.a)-pre && suf < len(d.b)-pre && d.a[len(d.a)-1-suf] == d.b[len(d.b)-1-suf] {
		suf++
	}
	return suf
}

// hunk formats one hunk: its header and lines.
// TRLC-LINKS: REQ-EMG-113
func (d diffText) hunk(ops []diffOp) string {
	aCount, bCount := 0, 0
	for _, o := range ops {
		if o.kind != '+' {
			aCount++
		}
		if o.kind != '-' {
			bCount++
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, "@@ -%s +%s @@\n", hunkRange(ops[0].ai, aCount), hunkRange(ops[0].bi, bCount))
	for _, o := range ops {
		out.WriteString(d.line(o))
	}
	return out.String()
}

// line formats one op as a diff line.
// TRLC-LINKS: REQ-EMG-113
func (d diffText) line(o diffOp) string {
	switch o.kind {
	case ' ':
		return " " + d.a[o.ai] + "\n"
	case '-':
		return "-" + d.a[o.ai] + "\n"
	}
	return "+" + d.b[o.bi] + "\n"
}

// lcsTable holds the longest-common-subsequence lengths of the suffixes of
// two line lists.
type lcsTable struct {
	a    []string
	b    []string
	size [][]int
}

// newLCS fills the table for a and b.
// TRLC-LINKS: REQ-EMG-113
func newLCS(a, b []string) lcsTable {
	t := lcsTable{a: a, b: b, size: make([][]int, len(a)+1)}
	for i := range t.size {
		t.size[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			t.size[i][j] = t.cell(i, j)
		}
	}
	return t
}

// cell returns the common subsequence length of a[i:] and b[j:].
// TRLC-LINKS: REQ-EMG-113
func (t lcsTable) cell(i, j int) int {
	if t.a[i] == t.b[j] {
		return t.size[i+1][j+1] + 1
	}
	return max(t.size[i+1][j], t.size[i][j+1])
}

// same reports whether the lines at i and j are both present and equal.
// TRLC-LINKS: REQ-EMG-113
func (t lcsTable) same(i, j int) bool {
	return i < len(t.a) && j < len(t.b) && t.a[i] == t.b[j]
}

// removes reports whether the line at i is removed rather than the line at
// j added.
// TRLC-LINKS: REQ-EMG-113
func (t lcsTable) removes(i, j int) bool {
	return j == len(t.b) || (i < len(t.a) && t.size[i+1][j] >= t.size[i][j+1])
}

// ops walks the table into line operations; offset is the number of lines
// before a and b in the full lists.
// TRLC-LINKS: REQ-EMG-113
func (t lcsTable) ops(offset int) []diffOp {
	var ops []diffOp
	i, j := 0, 0
	for i < len(t.a) || j < len(t.b) {
		switch {
		case t.same(i, j):
			ops = append(ops, diffOp{' ', offset + i, offset + j})
			i++
			j++
		case t.removes(i, j):
			ops = append(ops, diffOp{'-', offset + i, offset + j})
			i++
		default:
			ops = append(ops, diffOp{'+', offset + i, offset + j})
			j++
		}
	}
	return ops
}

// WriteFileAtomic replaces the existing file at path with data through a
// temporary file in the same directory and a rename, keeping the file's
// permission bits; a missing file is an error.
// TRLC-LINKS: REQ-EMG-113
func WriteFileAtomic(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	if err := fillTemp(tmp, data, info.Mode().Perm()); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// fillTemp sets the permission bits of tmp, writes data to it and syncs it.
// TRLC-LINKS: REQ-EMG-113
func fillTemp(tmp *os.File, data []byte, perm os.FileMode) error {
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	return tmp.Sync()
}
