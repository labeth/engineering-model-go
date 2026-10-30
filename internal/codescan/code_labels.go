// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

// Display names and evidence labels for scanned code items and code element
// references (path:line lists), shared by verification inference and the
// generated documents.

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/internal/textutil"
)

// TRLC-LINKS: REQ-EMG-003
func SanitizeNode(s string) string {
	repl := strings.NewReplacer("-", "_", " ", "_", ".", "_", "/", "_", ":", "_", ",", "_", "\\", "_", "(", "_", ")", "_", "[", "_", "]", "_")
	out := repl.Replace(strings.ToUpper(strings.TrimSpace(s)))
	if out == "" {
		out = "NODE"
	}
	return out
}

// TRLC-LINKS: REQ-EMG-003
func ItemDisplayName(c CodeItem) string {
	elem := strings.TrimSpace(c.Element)
	source := SanitizeSourcePath(c.Source)
	switch strings.TrimSpace(c.Kind) {
	case "symbol":
		if label := codeItemEvidenceLabel(c); label != "" {
			return label
		}
		return textutil.NonEmpty(elem, "code symbol")
	case "source_file":
		if label := codeItemEvidenceLabel(c); label != "" {
			return label
		}
		return textutil.NonEmpty(source, elem)
	default:
		if source == "" {
			return elem
		}
		sourceLabel := EvidenceLabel(source)
		if sourceLabel == "" {
			sourceLabel = source
		}
		return elem + " (" + sourceLabel + ")"
	}
}

// TRLC-LINKS: REQ-EMG-003
func codeItemEvidenceLabel(c CodeItem) string {
	return EvidenceLabel(ItemEvidenceElement(c))
}

// TRLC-LINKS: REQ-EMG-003
func ItemEvidenceElement(c CodeItem) string {
	switch strings.TrimSpace(c.Kind) {
	case "source_file":
		return ItemPath(c)
	case "symbol":
		if strings.TrimSpace(c.AbsPath) != "" {
			_, lines, _ := SplitEvidencePathLines(c.Source)
			path := c.AbsPath
			if c.EvidencePath != "" {
				path = c.EvidencePath
			}
			path = filepath.ToSlash(filepath.Clean(path))
			if len(lines) > 0 {
				path += ":" + JoinLineNumbers(lines)
			}
			return path
		}
		return filepath.ToSlash(strings.TrimSpace(c.Source))
	default:
		return strings.TrimSpace(c.Element)
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func EvidenceLabel(elem string) string {
	elem = SanitizeSourcePath(elem)
	if elem == "" {
		return ""
	}
	path := elem
	line := ""
	if idx := strings.LastIndex(path, ":"); idx > 0 && idx < len(path)-1 {
		suffix := path[idx+1:]
		if lines, ok := ParseLineNumberList(suffix); ok {
			path = path[:idx]
			line = ":" + JoinLineNumbers(lines)
		}
	}
	if filepath.Ext(path) != "" {
		return filepath.Base(path) + line
	}
	return elem
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func GroupedEvidenceLabels(elems []string, lookup map[string]string) []string {
	if lookup == nil {
		lookup = EvidenceLabelLookup(elems)
	}
	out := []string{}
	for _, elem := range elems {
		key := EvidenceFileKey(elem)
		if key != "" {
			if label := strings.TrimSpace(lookup[key]); label != "" {
				out = textutil.AppendUnique(out, label)
				continue
			}
		}
		label := EvidenceLabel(elem)
		if strings.TrimSpace(label) == "" {
			label = strings.TrimSpace(elem)
		}
		if label != "" {
			out = textutil.AppendUnique(out, label)
		}
	}
	return textutil.UniqueSorted(out)
}

// TRLC-LINKS: REQ-EMG-003
func EvidenceLabelLookup(elems []string) map[string]string {
	type bucket struct {
		base  string
		lines map[int]bool
	}
	buckets := map[string]*bucket{}
	baseCounts := map[string]int{}
	for _, elem := range elems {
		path, lines, ok := SplitEvidencePathLines(elem)
		if !ok {
			continue
		}
		base := filepath.Base(path)
		key := filepath.ToSlash(filepath.Clean(path))
		b, ok := buckets[key]
		if !ok {
			b = &bucket{base: base, lines: map[int]bool{}}
			buckets[key] = b
			baseCounts[base]++
		}
		for _, line := range lines {
			if line > 0 {
				b.lines[line] = true
			}
		}
	}
	out := map[string]string{}
	for key, b := range buckets {
		label := b.base
		if baseCounts[b.base] > 1 {
			label = key
			parts := strings.Split(key, "/")
			for n := 2; n <= len(parts); n++ {
				candidate := strings.Join(parts[len(parts)-n:], "/")
				unique := true
				for other := range buckets {
					if other != key && (other == candidate || strings.HasSuffix(other, "/"+candidate)) {
						unique = false
						break
					}
				}
				if unique {
					label = candidate
					break
				}
			}
		}
		lines := make([]int, 0, len(b.lines))
		for line := range b.lines {
			lines = append(lines, line)
		}
		sort.Ints(lines)
		if len(lines) == 0 {
			out[key] = label
			continue
		}
		parts := make([]string, 0, len(lines))
		for _, line := range lines {
			parts = append(parts, fmt.Sprintf("%d", line))
		}
		out[key] = label + ":" + strings.Join(parts, ",")
	}
	return out
}

// SplitSourceLine splits an inferred code item Source ("path:line") into path and line.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func SplitSourceLine(source string) (string, int) {
	idx := strings.LastIndex(source, ":")
	if idx < 0 {
		return source, 0
	}
	if n, err := strconv.Atoi(source[idx+1:]); err == nil {
		return source[:idx], n
	}
	return source, 0
}

// TRLC-LINKS: REQ-EMG-003
func ItemPath(c CodeItem) string {
	src := strings.TrimSpace(c.Source)
	if src == "" {
		return ""
	}
	if idx := strings.Index(src, ":"); idx > 0 {
		src = src[:idx]
	}
	return filepath.ToSlash(strings.TrimSpace(src))
}

// TRLC-LINKS: REQ-EMG-003
func GroupedFileLabels(elems []string, lookup map[string]string) []string {
	if lookup == nil {
		lookup = EvidenceLabelLookup(elems)
	}
	out := []string{}
	for _, elem := range elems {
		label := ""
		if key := EvidenceFileKey(elem); key != "" {
			if full := strings.TrimSpace(lookup[key]); full != "" {
				label = strings.TrimSpace(strings.Split(full, ":")[0])
			}
		}
		if label == "" {
			path, _, ok := SplitEvidencePathLines(elem)
			if ok {
				label = filepath.Base(path)
			}
		}
		if label == "" {
			label = strings.TrimSpace(strings.Split(EvidenceLabel(elem), ":")[0])
		}
		if label != "" {
			out = textutil.AppendUnique(out, label)
		}
	}
	return textutil.UniqueSorted(out)
}

// TRLC-LINKS: REQ-EMG-003
func EvidenceFileKey(elem string) string {
	path, _, ok := SplitEvidencePathLines(elem)
	if !ok {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}

// TRLC-LINKS: REQ-EMG-003
func SplitEvidencePathLines(elem string) (string, []int, bool) {
	elem = filepath.ToSlash(strings.TrimSpace(elem))
	if elem == "" {
		return "", nil, false
	}
	path := elem
	lines := []int{}
	if idx := strings.LastIndex(path, ":"); idx > 0 && idx < len(path)-1 {
		suffix := path[idx+1:]
		if parsed, ok := ParseLineNumberList(suffix); ok {
			path = path[:idx]
			lines = parsed
		}
	}
	if filepath.Ext(path) == "" {
		return "", nil, false
	}
	return filepath.ToSlash(path), lines, true
}

// TRLC-LINKS: REQ-EMG-003
func ParseLineNumberList(s string) ([]int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	seen := map[int]bool{}
	out := []int{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if !allDigits(part) {
			return nil, false
		}
		n := 0
		for _, r := range part {
			n = n*10 + int(r-'0')
		}
		if n <= 0 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, false
	}
	sort.Ints(out)
	return out, true
}

// TRLC-LINKS: REQ-EMG-003
func JoinLineNumbers(lines []int) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%d", line))
	}
	return strings.Join(parts, ",")
}

// TRLC-LINKS: REQ-EMG-003
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// TRLC-LINKS: REQ-EMG-003
func SanitizeSourcePath(in string) string {
	s := filepath.ToSlash(strings.TrimSpace(in))
	if s == "" {
		return s
	}
	for _, marker := range []string{"/examples/", "/infra/", "/src/", "/tmp/"} {
		if idx := strings.Index(strings.ToLower(s), marker); idx >= 0 {
			return s[idx+1:]
		}
	}
	if filepath.IsAbs(s) {
		return filepath.Base(s)
	}
	return s
}

// TRLC-LINKS: REQ-EMG-003
func LanguageFromPath(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".rs":
		return "rust"
	case ".py":
		return "python"
	default:
		return ""
	}
}
