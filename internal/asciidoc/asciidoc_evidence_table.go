// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
)

// publicationTestCode gives grouped line numbers legal table wrap points without
// changing the canonical evidence labels or punctuation in source filenames.
// TRLC-LINKS: REQ-EMG-003
func publicationTestCode(elements []string) string {
	parts := make([]string, 0, len(elements))
	for _, label := range elements {
		if trimmed := strings.TrimSpace(spaceLineNumberList(label)); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// spaceLineNumberList puts a space after each comma of the line numbers that
// end a label, when there are several and all are numbers.
// TRLC-LINKS: REQ-EMG-003
func spaceLineNumberList(label string) string {
	colon := strings.LastIndexByte(label, ':')
	if colon < 0 {
		return label
	}
	lines := strings.Split(label[colon+1:], ",")
	if len(lines) < 2 || !allDigits(lines) {
		return label
	}
	return label[:colon+1] + strings.Join(lines, ", ")
}

// allDigits reports whether every item is a non-empty string of digits.
// TRLC-LINKS: REQ-EMG-003
func allDigits(items []string) bool {
	for _, item := range items {
		if item == "" || strings.Trim(item, "0123456789") != "" {
			return false
		}
	}
	return true
}

// diagramCodeEvidenceLabel lets Mermaid wrap between source coordinates rather
// than treating a filename and all its line numbers as one unbroken token.
// Canonical labels and node identity continue to use the original evidence.
// TRLC-LINKS: REQ-EMG-003
func diagramCodeEvidenceLabel(label string) string {
	colon := strings.LastIndexByte(label, ':')
	if colon < 0 {
		return label
	}
	if _, valid := codescan.ParseLineNumberList(label[colon+1:]); !valid {
		return label
	}
	parts := strings.Split(label[colon+1:], ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return label[:colon+1] + " " + strings.Join(parts, ", ")
}
