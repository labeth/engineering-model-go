// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Line-preserving YAML edits. Each edit parses the document with yaml.v3,
// finds the node to change through the node tree and rewrites only the lines
// of that node, so comments, key order, quoting and indentation of every
// other line are kept. Callers apply one edit at a time and re-parse.

// parseYAML returns the root mapping node of a YAML document.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func parseYAML(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("document is not a mapping")
	}
	return doc.Content[0], nil
}

// yamlScalar encodes s as a single-line YAML scalar (plain when safe,
// quoted otherwise); runs of whitespace are collapsed.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func yamlScalar(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	out, err := yaml.Marshal(map[string]string{"k": s})
	if err != nil {
		return fmt.Sprintf("%q", s)
	}
	return strings.TrimSuffix(strings.TrimPrefix(string(out), "k: "), "\n")
}

// keyRange returns the 0-based line range [start, end) a mapping key and
// its value occupy: the key line and every following line that is blank,
// indented deeper than the key, or (for a sequence value written without
// indentation) a "-" item at the key's indentation. Trailing blank and
// comment lines are not part of the range.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func keyRange(lines []string, key, value *yaml.Node) (int, int) {
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

// splitLines splits data into lines that keep their newline.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func splitLines(data []byte) []string {
	var out []string
	for _, l := range bytes.SplitAfter(data, []byte("\n")) {
		if len(l) > 0 {
			out = append(out, string(l))
		}
	}
	return out
}

// spliceLines replaces lines [start, end) with replacement lines.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
func spliceLines(lines []string, start, end int, replacement ...string) []byte {
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

// setKey sets key of the block mapping found by find to the encoded YAML
// value raw (a single-line scalar or flow collection, or a block starting
// with a newline). An existing single-line scalar is replaced in place,
// keeping a trailing comment; any other existing value is replaced with its
// lines; a missing key is added after the mapping's last key.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func setKey(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key, raw string) ([]byte, error) {
	root, err := parseYAML(data)
	if err != nil {
		return nil, err
	}
	mapping, err := find(root)
	if err != nil {
		return nil, err
	}
	if mapping.Kind != yaml.MappingNode || mapping.Style&yaml.FlowStyle != 0 || len(mapping.Content) == 0 {
		return nil, fmt.Errorf("line %d: %s can only be set in a non-empty block mapping", mapping.Line, key)
	}
	lines := splitLines(data)
	keyNode, value := mappingEntry(mapping, key)
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
	if keyNode == nil {
		lastKey := mapping.Content[len(mapping.Content)-2]
		_, end := keyRange(lines, lastKey, mapping.Content[len(mapping.Content)-1])
		return spliceLines(lines, end, end, entry()...), nil
	}
	if value.Kind == yaml.ScalarNode && value.Line == keyNode.Line && !strings.HasPrefix(raw, "\n") && (value.Style == 0 || value.Style == yaml.DoubleQuotedStyle || value.Style == yaml.SingleQuotedStyle) {
		line := lines[value.Line-1]
		startCol := value.Column - 1
		endCol, ok := scalarEnd(line, startCol, value)
		if ok {
			lines[value.Line-1] = line[:startCol] + raw + line[endCol:]
			return []byte(strings.Join(lines, "")), nil
		}
	}
	start, end := keyRange(lines, keyNode, value)
	return spliceLines(lines, start, end, entry()...), nil
}

// scalarEnd returns the column just after a single-line scalar that starts
// at col of line.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101
func scalarEnd(line string, col int, value *yaml.Node) (int, bool) {
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

// removeKey removes key and its value lines from the block mapping found by
// find; a missing key is no change.
// TRLC-LINKS: REQ-EMG-104
func removeKey(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key string) ([]byte, bool, error) {
	root, err := parseYAML(data)
	if err != nil {
		return nil, false, err
	}
	mapping, err := find(root)
	if err != nil {
		return nil, false, err
	}
	keyNode, value := mappingEntry(mapping, key)
	if keyNode == nil {
		return data, false, nil
	}
	if mapping.Style&yaml.FlowStyle != 0 {
		return nil, false, fmt.Errorf("line %d: cannot remove %s from a flow mapping", mapping.Line, key)
	}
	lines := splitLines(data)
	if prefix := lines[keyNode.Line-1][:keyNode.Column-1]; strings.Contains(prefix, "-") {
		return nil, false, fmt.Errorf("line %d: %s is the first key of a list item; move it below another key first", keyNode.Line, key)
	}
	start, end := keyRange(lines, keyNode, value)
	return spliceLines(lines, start, end), true, nil
}

// appendSequenceItems appends the block lines items (each item already
// formatted as "- ..." with continuation lines indented by two spaces,
// relative to column 0) to the sequence under key of the mapping found by
// find, at the indentation of its existing items. An empty or flow sequence
// is replaced by a block sequence.
// TRLC-LINKS: REQ-EMG-104
func appendSequenceItems(data []byte, find func(root *yaml.Node) (*yaml.Node, error), key string, items []string) ([]byte, error) {
	root, err := parseYAML(data)
	if err != nil {
		return nil, err
	}
	mapping, err := find(root)
	if err != nil {
		return nil, err
	}
	keyNode, value := mappingEntry(mapping, key)
	if keyNode == nil || value.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("no %s sequence", key)
	}
	lines := splitLines(data)
	start, end := keyRange(lines, keyNode, value)
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
		return spliceLines(lines, start, end, append([]string{strings.Repeat(" ", keyNode.Column-1) + key + ":"}, block...)...), nil
	}
	return spliceLines(lines, end, end, block...), nil
}

// findPath returns a function that walks mapping keys from the root.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func findPath(keys ...string) func(*yaml.Node) (*yaml.Node, error) {
	return func(node *yaml.Node) (*yaml.Node, error) {
		for _, k := range keys {
			next := mappingValue(node, k)
			if next == nil {
				return nil, fmt.Errorf("no %s", strings.Join(keys, "."))
			}
			node = next
		}
		return node, nil
	}
}

// findListItem returns a function that finds the mapping item with id in
// the sequence at keys, then walks sub keys inside it.
// TRLC-LINKS: REQ-EMG-100, REQ-EMG-101, REQ-EMG-104
func findListItem(keys []string, idKey, id string, sub ...string) func(*yaml.Node) (*yaml.Node, error) {
	return func(root *yaml.Node) (*yaml.Node, error) {
		seq, err := findPath(keys...)(root)
		if err != nil {
			return nil, err
		}
		if seq.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("%s is not a list", strings.Join(keys, "."))
		}
		for _, item := range seq.Content {
			if item.Kind == yaml.MappingNode {
				if v := mappingValue(item, idKey); v != nil && strings.TrimSpace(v.Value) == id {
					return findPath(sub...)(item)
				}
			}
		}
		return nil, fmt.Errorf("no %s entry with %s %s", strings.Join(keys, "."), idKey, id)
	}
}
