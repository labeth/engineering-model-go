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
		return symbolEvidenceElement(c)
	default:
		return strings.TrimSpace(c.Element)
	}
}

// symbolEvidenceElement is the evidence element of a symbol item: its absolute
// (or evidence) path with the source line list, else the slashed source.
// TRLC-LINKS: REQ-EMG-003
func symbolEvidenceElement(c CodeItem) string {
	if strings.TrimSpace(c.AbsPath) == "" {
		return filepath.ToSlash(strings.TrimSpace(c.Source))
	}
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
		if label := groupedEvidenceLabel(elem, lookup); label != "" {
			out = textutil.AppendUnique(out, label)
		}
	}
	return textutil.UniqueSorted(out)
}

// groupedEvidenceLabel is the lookup label of an element, else its own
// evidence label, else the trimmed element.
// TRLC-LINKS: REQ-EMG-003
func groupedEvidenceLabel(elem string, lookup map[string]string) string {
	if key := EvidenceFileKey(elem); key != "" {
		if label := strings.TrimSpace(lookup[key]); label != "" {
			return label
		}
	}
	label := EvidenceLabel(elem)
	if strings.TrimSpace(label) == "" {
		label = strings.TrimSpace(elem)
	}
	return label
}

// evidenceBucket collects the line numbers named for one evidence file.
type evidenceBucket struct {
	base  string
	lines map[int]bool
}

// TRLC-LINKS: REQ-EMG-003
func EvidenceLabelLookup(elems []string) map[string]string {
	buckets, baseCounts := collectEvidenceBuckets(elems)
	out := map[string]string{}
	for key, b := range buckets {
		label := b.base
		if baseCounts[b.base] > 1 {
			label = uniqueEvidenceLabel(key, buckets)
		}
		out[key] = label + evidenceLineSuffix(b.lines)
	}
	return out
}

// collectEvidenceBuckets groups the elements by cleaned path and counts the
// distinct paths per base name.
// TRLC-LINKS: REQ-EMG-003
func collectEvidenceBuckets(elems []string) (map[string]*evidenceBucket, map[string]int) {
	buckets := map[string]*evidenceBucket{}
	baseCounts := map[string]int{}
	for _, elem := range elems {
		path, lines, ok := SplitEvidencePathLines(elem)
		if !ok {
			continue
		}
		key := filepath.ToSlash(filepath.Clean(path))
		b, ok := buckets[key]
		if !ok {
			base := filepath.Base(path)
			b = &evidenceBucket{base: base, lines: map[int]bool{}}
			buckets[key] = b
			baseCounts[base]++
		}
		b.addLines(lines)
	}
	return buckets, baseCounts
}

// addLines records the positive line numbers.
// TRLC-LINKS: REQ-EMG-003
func (b *evidenceBucket) addLines(lines []int) {
	for _, line := range lines {
		if line > 0 {
			b.lines[line] = true
		}
	}
}

// uniqueEvidenceLabel is the shortest trailing path (two segments or more) no
// other bucket ends with, else the full key.
// TRLC-LINKS: REQ-EMG-003
func uniqueEvidenceLabel(key string, buckets map[string]*evidenceBucket) string {
	parts := strings.Split(key, "/")
	for n := 2; n <= len(parts); n++ {
		candidate := strings.Join(parts[len(parts)-n:], "/")
		if !evidenceSuffixTaken(key, candidate, buckets) {
			return candidate
		}
	}
	return key
}

// evidenceSuffixTaken reports whether a bucket other than key is, or ends
// with, the candidate path.
// TRLC-LINKS: REQ-EMG-003
func evidenceSuffixTaken(key, candidate string, buckets map[string]*evidenceBucket) bool {
	for other := range buckets {
		if other != key && (other == candidate || strings.HasSuffix(other, "/"+candidate)) {
			return true
		}
	}
	return false
}

// evidenceLineSuffix is ":1,2,3" for the sorted line set, empty for none.
// TRLC-LINKS: REQ-EMG-003
func evidenceLineSuffix(set map[int]bool) string {
	if len(set) == 0 {
		return ""
	}
	lines := make([]int, 0, len(set))
	for line := range set {
		lines = append(lines, line)
	}
	sort.Ints(lines)
	return ":" + JoinLineNumbers(lines)
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
		if label := groupedFileLabel(elem, lookup); label != "" {
			out = textutil.AppendUnique(out, label)
		}
	}
	return textutil.UniqueSorted(out)
}

// lookupFileLabel is the lookup label of the element's file without its lines.
// TRLC-LINKS: REQ-EMG-003
func lookupFileLabel(elem string, lookup map[string]string) string {
	key := EvidenceFileKey(elem)
	if key == "" {
		return ""
	}
	full := strings.TrimSpace(lookup[key])
	if full == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(full, ":")[0])
}

// groupedFileLabel is the file label of an element: the lookup label without
// its lines, else the base name, else the evidence label without its lines.
// TRLC-LINKS: REQ-EMG-003
func groupedFileLabel(elem string, lookup map[string]string) string {
	if label := lookupFileLabel(elem, lookup); label != "" {
		return label
	}
	if path, _, ok := SplitEvidencePathLines(elem); ok {
		return filepath.Base(path)
	}
	return strings.TrimSpace(strings.Split(EvidenceLabel(elem), ":")[0])
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
		n := digitsValue(part)
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

// digitsValue is the decimal value of an all-digit string.
// TRLC-LINKS: REQ-EMG-003
func digitsValue(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
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
