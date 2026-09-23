// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import "strings"

// publicationTestCode gives grouped line numbers legal table wrap points without
// changing the canonical evidence labels or punctuation in source filenames.
// TRLC-LINKS: REQ-EMG-003
func publicationTestCode(elements []string) string {
	labels := append([]string(nil), elements...)
	for i, label := range labels {
		colon := strings.LastIndexByte(label, ':')
		if colon < 0 {
			continue
		}
		lines := strings.Split(label[colon+1:], ",")
		if len(lines) < 2 {
			continue
		}
		valid := true
		for _, line := range lines {
			if line == "" {
				valid = false
			}
			for _, c := range line {
				if c < '0' || c > '9' {
					valid = false
				}
			}
		}
		if valid {
			labels[i] = label[:colon+1] + strings.Join(lines, ", ")
		}
	}
	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		if trimmed := strings.TrimSpace(label); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
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
	if _, valid := parseLineNumberList(label[colon+1:]); !valid {
		return label
	}
	parts := strings.Split(label[colon+1:], ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return label[:colon+1] + " " + strings.Join(parts, ", ")
}
