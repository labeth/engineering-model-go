// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"regexp"
)

// TRLC-LINKS: REQ-EMG-003
func publicationEdgeKey(kind, graph string) (string, []map[string]string) {
	if kind != "deployment" {
		return graph, nil
	}
	pattern := regexp.MustCompile(`(?m)^([A-Za-z0-9_]+[ \t]+)--+>\|([^|\n]+)\|`)
	reserved := map[string]bool{}
	for _, match := range pattern.FindAllStringSubmatch(graph, -1) {
		reserved[match[2]] = true
	}
	rows := []map[string]string{}
	next := 1
	graph = pattern.ReplaceAllStringFunc(graph, func(edge string) string {
		match := pattern.FindStringSubmatch(edge)
		label := match[2]
		if len(label) <= 32 {
			return edge
		}
		number := fmt.Sprintf("R%d", next)
		for reserved[number] {
			next++
			number = fmt.Sprintf("R%d", next)
		}
		next++
		rows = append(rows, map[string]string{"Number": number, "Description": label})
		return match[1] + "--->|" + number + "|"
	})
	return graph, rows
}
