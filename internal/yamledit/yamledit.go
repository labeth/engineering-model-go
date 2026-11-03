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
	if value.Kind == yaml.ScalarNode && value.Line == keyNode.Line && !strings.HasPrefix(raw, "\n") && (value.Style == 0 || value.Style == yaml.DoubleQuotedStyle || value.Style == yaml.SingleQuotedStyle) {
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
