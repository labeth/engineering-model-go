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
	last := start + 1
	for i := start + 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r\n")
		trimmed := strings.TrimLeft(line, " ")
		depth := len(line) - len(trimmed)
		switch {
		case trimmed == "" || strings.HasPrefix(trimmed, "#"):
			continue
		case depth > indent:
		case depth == indent && value.Kind == yaml.SequenceNode && value.Style&yaml.FlowStyle == 0 && (trimmed == "-" || strings.HasPrefix(trimmed, "- ")):
		default:
			return start, last
		}
		last = i + 1
	}
	return start, last
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

// SpliceLines replaces lines [start, end) with replacement lines.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func SpliceLines(lines []string, start, end int, replacement ...string) []byte {
	var b strings.Builder
	for _, l := range lines[:start] {
		b.WriteString(l)
	}
	if start > 0 && !strings.HasSuffix(lines[start-1], "\n") {
		b.WriteString("\n")
	}
	for _, l := range replacement {
		b.WriteString(l)
		if !strings.HasSuffix(l, "\n") {
			b.WriteString("\n")
		}
	}
	for _, l := range lines[end:] {
		b.WriteString(l)
	}
	return []byte(b.String())
}

// SetKey sets key of the block mapping found by find to the encoded YAML
// value raw (a single-line scalar or flow collection, or a block starting
// with a newline). An existing single-line scalar is replaced in place,
// keeping a trailing comment; any other existing value is replaced with its
// lines; a missing key is added after the mapping's last key.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func SetKey(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key, raw string) ([]byte, error) {
	root, err := Parse(data)
	if err != nil {
		return nil, err
	}
	mapping, err := find(root)
	if err != nil {
		return nil, err
	}
	if mapping.Kind != yaml.MappingNode || len(mapping.Content) == 0 {
		return nil, fmt.Errorf("line %d: %s can only be set in a non-empty mapping", mapping.Line, key)
	}
	lines := SplitLines(data)
	keyNode, value := MappingEntry(mapping, key)
	flow := mapping.Style&yaml.FlowStyle != 0
	indent := strings.Repeat(" ", mapping.Content[0].Column-1)
	if keyNode != nil {
		// Keep a "- " item prefix when the key is the first of a list item.
		indent = lines[keyNode.Line-1][:keyNode.Column-1]
	}
	entry := func() []string {
		if strings.HasPrefix(raw, "\n") {
			return append([]string{indent + key + ":"}, strings.Split(strings.TrimPrefix(raw, "\n"), "\n")...)
		}
		return []string{indent + key + ": " + raw}
	}
	if flow && (keyNode == nil || value.Kind != yaml.ScalarNode || strings.HasPrefix(raw, "\n")) {
		return nil, fmt.Errorf("line %d: %s cannot be set in a flow mapping; rewrite it as a block mapping first", mapping.Line, key)
	}
	if keyNode == nil {
		lastKey := mapping.Content[len(mapping.Content)-2]
		_, end := KeyRange(lines, lastKey, mapping.Content[len(mapping.Content)-1])
		return SpliceLines(lines, end, end, entry()...), nil
	}
	if value.Kind == yaml.ScalarNode && value.Line == keyNode.Line && !strings.Contains(raw, "\n") && (value.Style == 0 || value.Style == yaml.DoubleQuotedStyle || value.Style == yaml.SingleQuotedStyle) {
		line := lines[value.Line-1]
		startCol := value.Column - 1
		endCol, ok := ScalarEnd(line, startCol, value)
		if ok {
			lines[value.Line-1] = line[:startCol] + raw + line[endCol:]
			return []byte(strings.Join(lines, "")), nil
		}
	}
	if flow {
		return nil, fmt.Errorf("line %d: %s cannot be set in place in a flow mapping", mapping.Line, key)
	}
	start, end := KeyRange(lines, keyNode, value)
	return SpliceLines(lines, start, end, entry()...), nil
}

// SetScalarInPlace replaces the single-line scalar value of key in the
// mapping found by find with value, keeping its quote style (plain, single
// or double) and every other byte of data, in block and flow mappings. It
// refuses a missing key, a value that is no scalar or not on its key's line
// (a block or multi-line scalar), and a value that its quote style cannot
// hold unescaped.
// TRLC-LINKS: REQ-EMG-068
func SetScalarInPlace(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key, value string) ([]byte, error) {
	root, err := Parse(data)
	if err != nil {
		return nil, err
	}
	mapping, err := find(root)
	if err != nil {
		return nil, err
	}
	keyNode, node := MappingEntry(mapping, key)
	switch {
	case keyNode == nil:
		return nil, fmt.Errorf("no %s", key)
	case node.Kind != yaml.ScalarNode || node.Line != keyNode.Line || (node.Style != 0 && node.Style != yaml.SingleQuotedStyle && node.Style != yaml.DoubleQuotedStyle):
		return nil, fmt.Errorf("line %d: %s is not a single-line scalar", keyNode.Line, key)
	}
	raw := value
	switch node.Style {
	case yaml.SingleQuotedStyle:
		if strings.ContainsAny(value, "'\n") {
			return nil, fmt.Errorf("line %d: %q cannot be written single-quoted in place", node.Line, value)
		}
		raw = "'" + value + "'"
	case yaml.DoubleQuotedStyle:
		if strings.ContainsAny(value, "\"\\\n") {
			return nil, fmt.Errorf("line %d: %q cannot be written double-quoted in place", node.Line, value)
		}
		raw = "\"" + value + "\""
	default:
		if Scalar(value) != value {
			return nil, fmt.Errorf("line %d: %q is not a plain scalar", node.Line, value)
		}
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

// ScalarEnd returns the column just after a single-line scalar that starts
// at col of line.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101
func ScalarEnd(line string, col int, value *yaml.Node) (int, bool) {
	if col >= len(line) {
		return 0, false
	}
	switch value.Style {
	case yaml.DoubleQuotedStyle:
		for i := col + 1; i < len(line); i++ {
			switch line[i] {
			case '\\':
				i++
			case '"':
				return i + 1, true
			}
		}
	case yaml.SingleQuotedStyle:
		for i := col + 1; i < len(line); i++ {
			if line[i] == '\'' {
				if i+1 < len(line) && line[i+1] == '\'' {
					i++
					continue
				}
				return i + 1, true
			}
		}
	default:
		if strings.HasPrefix(line[col:], value.Value) {
			return col + len(value.Value), true
		}
	}
	return 0, false
}

// RemoveKey removes key and its value lines from the block mapping found by
// find; a missing key is no change.
// TRLC-LINKS: REQ-EMG-104
func RemoveKey(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key string) ([]byte, bool, error) {
	root, err := Parse(data)
	if err != nil {
		return nil, false, err
	}
	mapping, err := find(root)
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
	return SpliceLines(lines, start, end), true, nil
}

// AppendSequenceItems appends the block lines items (each item already
// formatted as "- ..." with continuation lines indented by two spaces,
// relative to column 0) to the sequence under key of the mapping found by
// find, at the indentation of its existing items. An empty or flow sequence
// is replaced by a block sequence.
// TRLC-LINKS: REQ-EMG-104
func AppendSequenceItems(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key string, items []string) ([]byte, error) {
	root, err := Parse(data)
	if err != nil {
		return nil, err
	}
	mapping, err := find(root)
	if err != nil {
		return nil, err
	}
	keyNode, value := MappingEntry(mapping, key)
	if keyNode == nil || value.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("no %s sequence", key)
	}
	lines := SplitLines(data)
	start, end := KeyRange(lines, keyNode, value)
	indent := keyNode.Column - 1
	if len(value.Content) > 0 && value.Style&yaml.FlowStyle == 0 {
		// The dash sits two columns before the first item's first key.
		indent = value.Content[0].Column - 3
		if indent < 0 {
			indent = 0
		}
	}
	pad := strings.Repeat(" ", indent)
	var block []string
	for _, item := range items {
		for _, l := range strings.Split(strings.TrimSuffix(item, "\n"), "\n") {
			block = append(block, pad+l)
		}
	}
	if len(value.Content) == 0 || value.Style&yaml.FlowStyle != 0 {
		if len(value.Content) > 0 {
			return nil, fmt.Errorf("line %d: %s is a flow sequence; rewrite it as a block sequence first", keyNode.Line, key)
		}
		return SpliceLines(lines, start, end, append([]string{strings.Repeat(" ", keyNode.Column-1) + key + ":"}, block...)...), nil
	}
	return SpliceLines(lines, end, end, block...), nil
}

// RemoveSequenceItems removes the items at indexes (0-based, any order) of
// the block sequence under key of the mapping found by find, each with its
// lines up to the next item; the other items keep their lines.
// TRLC-LINKS: REQ-EMG-104
func RemoveSequenceItems(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key string, indexes []int) ([]byte, error) {
	root, err := Parse(data)
	if err != nil {
		return nil, err
	}
	mapping, err := find(root)
	if err != nil {
		return nil, err
	}
	keyNode, value := MappingEntry(mapping, key)
	if keyNode == nil || value.Kind != yaml.SequenceNode || value.Style&yaml.FlowStyle != 0 {
		return nil, fmt.Errorf("no %s block sequence", key)
	}
	lines := SplitLines(data)
	_, end := KeyRange(lines, keyNode, value)
	remove := map[int]bool{}
	for _, i := range indexes {
		remove[i] = true
	}
	var out []string
	cursor := 0
	for i, item := range value.Content {
		start := item.Line - 1
		stop := end
		if i+1 < len(value.Content) {
			stop = value.Content[i+1].Line - 1
		}
		out = append(out, lines[cursor:start]...)
		if !remove[i] {
			out = append(out, lines[start:stop]...)
		}
		cursor = stop
	}
	out = append(out, lines[cursor:]...)
	return []byte(strings.Join(out, "")), nil
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
func FindListItem(keys []string, idKey, id string, sub ...string) func(*yaml.Node) (*yaml.Node, error) {
	return func(root *yaml.Node) (*yaml.Node, error) {
		seq, err := FindPath(keys...)(root)
		if err != nil {
			return nil, err
		}
		if seq.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("%s is not a list", strings.Join(keys, "."))
		}
		for _, item := range seq.Content {
			if item.Kind == yaml.MappingNode {
				if v := MappingValue(item, idKey); v != nil && strings.TrimSpace(v.Value) == id {
					return FindPath(sub...)(item)
				}
			}
		}
		return nil, fmt.Errorf("no %s entry with %s %s", strings.Join(keys, "."), idKey, id)
	}
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
	case nil:
		return nil, fmt.Errorf("a null value cannot be set; unset the field instead")
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: x}, nil
	case bool, int, int64, float64:
		var n yaml.Node
		if err := n.Encode(x); err != nil {
			return nil, err
		}
		return &n, nil
	case []string:
		items := make([]any, len(x))
		for i, s := range x {
			items[i] = s
		}
		return valueNode(items)
	case []any:
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range x {
			n, err := valueNode(item)
			if err != nil {
				return nil, err
			}
			seq.Content = append(seq.Content, n)
		}
		return seq, nil
	case map[string]any:
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
		m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, k := range keys {
			n, err := valueNode(x[k])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}, n)
		}
		return m, nil
	default:
		return nil, fmt.Errorf("unsupported value type %T", v)
	}
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
func SetPath(data []byte, find func(*yaml.Node) (*yaml.Node, error), path []string, value any) ([]byte, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty field path")
	}
	root, err := Parse(data)
	if err != nil {
		return nil, err
	}
	node, err := find(root)
	if err != nil {
		return nil, err
	}
	depth := 0
	for ; depth < len(path)-1; depth++ {
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("line %d: %s is not a mapping", node.Line, strings.Join(path[:depth], "."))
		}
		next := MappingValue(node, path[depth])
		if next == nil {
			break
		}
		node = next
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("line %d: %s is not a mapping", node.Line, strings.Join(path[:depth], "."))
	}
	// Build the missing part of the path as nested objects.
	for i := len(path) - 1; i > depth; i-- {
		value = map[string]any{path[i]: value}
	}
	raw, err := rawValue(path[depth], value, keyIndent(node))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.Join(path, "."), err)
	}
	return SetKey(data, within(find, path[:depth]), path[depth], raw)
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
	if key != nil && value.Kind != yaml.SequenceNode && !(value.Kind == yaml.ScalarNode && value.Tag == "!!null") {
		return nil, nil, nil, fmt.Errorf("line %d: %s is not a list", key.Line, strings.Join(path, "."))
	}
	return mapping, key, value, nil
}

// flowItemsRange returns the column range [start, end) of the one-line flow
// sequence value on its line, brackets included.
// TRLC-LINKS: REQ-EMG-113
func flowItemsRange(line string, value *yaml.Node) (int, int, bool) {
	start := value.Column - 1
	if start >= len(line) || line[start] != '[' {
		return 0, 0, false
	}
	for _, item := range value.Content {
		if item.Kind != yaml.ScalarNode || item.Line != value.Line {
			return 0, 0, false
		}
	}
	quote := byte(0)
	for i := start + 1; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else if c == '\\' && quote == '"' {
				i++
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '[' || c == '{':
			return 0, 0, false
		case c == ']':
			return start, i + 1, true
		}
	}
	return 0, 0, false
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

// AppendPath appends items to the list at path, creating the list when the
// field is missing. A block list gets block items at the indentation of its
// items; a one-line flow list of scalars stays a flow list.
// TRLC-LINKS: REQ-EMG-113
func AppendPath(data []byte, find func(*yaml.Node) (*yaml.Node, error), path []string, items []any) ([]byte, error) {
	if len(items) == 0 {
		return data, nil
	}
	_, key, value, err := sequenceAt(data, find, path)
	if err != nil {
		return nil, err
	}
	if key == nil || value.Kind != yaml.SequenceNode {
		return SetPath(data, find, path, items)
	}
	if value.Style&yaml.FlowStyle != 0 && len(value.Content) > 0 {
		raw, err := flowRaw(SplitLines(data)[value.Line-1], value)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			n, err := valueNode(item)
			if err != nil {
				return nil, err
			}
			if n.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("line %d: %s is a flow list; only scalars can be appended to it", value.Line, strings.Join(path, "."))
			}
			lines, err := encodeLines(n)
			if err != nil {
				return nil, err
			}
			raw = append(raw, lines[0])
		}
		return rewriteFlow(data, value, raw)
	}
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
	return AppendSequenceItems(data, within(find, path[:len(path)-1]), path[len(path)-1], block)
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
func RemovePath(data []byte, find func(*yaml.Node) (*yaml.Node, error), path []string, values []any) ([]byte, error) {
	_, key, value, err := sequenceAt(data, find, path)
	if err != nil {
		return nil, err
	}
	if key == nil || value.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("no list %s", strings.Join(path, "."))
	}
	remove := map[int]bool{}
	for _, v := range values {
		s, ok := v.(string)
		if !ok {
			n, err := valueNode(v)
			if err != nil || n.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("remove %s: name an item by its scalar value or id", strings.Join(path, "."))
			}
			s = n.Value
		}
		matched := false
		for i, item := range value.Content {
			if itemMatches(item, s) {
				remove[i], matched = true, true
			}
		}
		if !matched {
			return nil, fmt.Errorf("list %s has no item %q", strings.Join(path, "."), s)
		}
	}
	if len(remove) == len(value.Content) {
		return SetKey(data, within(find, path[:len(path)-1]), path[len(path)-1], "[]")
	}
	if value.Style&yaml.FlowStyle != 0 {
		raw, err := flowRaw(SplitLines(data)[value.Line-1], value)
		if err != nil {
			return nil, err
		}
		kept := []string{}
		for i, r := range raw {
			if !remove[i] {
				kept = append(kept, r)
			}
		}
		return rewriteFlow(data, value, kept)
	}
	indexes := make([]int, 0, len(remove))
	for i := range remove {
		indexes = append(indexes, i)
	}
	return RemoveSequenceItems(data, within(find, path[:len(path)-1]), path[len(path)-1], indexes)
}

// UnifiedDiff returns the unified diff (three lines of context, git apply
// format) of before and after under name, or "" when they are equal.
// TRLC-LINKS: REQ-EMG-113
func UnifiedDiff(name string, before, after []byte) string {
	if bytes.Equal(before, after) {
		return ""
	}
	split := func(b []byte) []string {
		s := strings.TrimSuffix(string(b), "\n")
		if s == "" {
			return nil
		}
		return strings.Split(s, "\n")
	}
	a, b := split(before), split(after)
	// Trim the common prefix and suffix before the quadratic LCS.
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	type op struct {
		kind byte // ' ', '-', '+'
		ai   int  // index in a (for ' ' and '-'), else position
		bi   int
	}
	var ops []op
	for i := 0; i < pre; i++ {
		ops = append(ops, op{' ', i, i})
	}
	ma, mb := a[pre:len(a)-suf], b[pre:len(b)-suf]
	lcs := make([][]int, len(ma)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(mb)+1)
	}
	for i := len(ma) - 1; i >= 0; i-- {
		for j := len(mb) - 1; j >= 0; j-- {
			if ma[i] == mb[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	i, j := 0, 0
	for i < len(ma) || j < len(mb) {
		switch {
		case i < len(ma) && j < len(mb) && ma[i] == mb[j]:
			ops = append(ops, op{' ', pre + i, pre + j})
			i++
			j++
		case j == len(mb) || (i < len(ma) && lcs[i+1][j] >= lcs[i][j+1]):
			ops = append(ops, op{'-', pre + i, pre + j})
			i++
		default:
			ops = append(ops, op{'+', pre + i, pre + j})
			j++
		}
	}
	for k := 0; k < suf; k++ {
		ops = append(ops, op{' ', len(a) - suf + k, len(b) - suf + k})
	}
	const context = 3
	var out strings.Builder
	fmt.Fprintf(&out, "--- a/%s\n+++ b/%s\n", name, name)
	rng := func(start, count int) string {
		if count == 0 {
			return fmt.Sprintf("%d,0", start)
		}
		if count == 1 {
			return fmt.Sprint(start + 1)
		}
		return fmt.Sprintf("%d,%d", start+1, count)
	}
	for k := 0; k < len(ops); {
		if ops[k].kind == ' ' {
			k++
			continue
		}
		start := max(k-context, 0)
		end := k
		for end < len(ops) {
			if ops[end].kind != ' ' {
				end++
				continue
			}
			run := end
			for run < len(ops) && ops[run].kind == ' ' {
				run++
			}
			if run == len(ops) || run-end > 2*context {
				end = min(end+context, len(ops))
				break
			}
			end = run
		}
		aCount, bCount := 0, 0
		for _, o := range ops[start:end] {
			if o.kind != '+' {
				aCount++
			}
			if o.kind != '-' {
				bCount++
			}
		}
		fmt.Fprintf(&out, "@@ -%s +%s @@\n", rng(ops[start].ai, aCount), rng(ops[start].bi, bCount))
		for _, o := range ops[start:end] {
			switch o.kind {
			case ' ':
				out.WriteString(" " + a[o.ai] + "\n")
			case '-':
				out.WriteString("-" + a[o.ai] + "\n")
			default:
				out.WriteString("+" + b[o.bi] + "\n")
			}
		}
		k = end
	}
	return out.String()
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
	name := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		return fail(err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Sync(); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}
