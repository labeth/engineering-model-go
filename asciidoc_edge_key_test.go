// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-003
func TestPublicationEdgeKeyPreservesRelationships(t *testing.T) {
	label := "interacts_with: complete long hardware relationship description"
	graph := "flowchart LR\nN_TEXT[\"literal -->|this is a long embedded string rather than an edge|\"]\nA -->|" + label + "| B\nB -->|contains| C\nC -->|" + label + "| A\n"
	got, key := publicationEdgeKey("deployment", graph)
	if len(key) != 2 || key[0]["Description"] != label || key[1]["Description"] != label || key[1]["Number"] != "R2" {
		t.Fatalf("lost descriptions: %#v", key)
	}
	if !strings.Contains(got, `N_TEXT["literal -->|this is a long embedded string rather than an edge|"]`) {
		t.Fatal("rewrote node text")
	}
	for _, edge := range []string{"A --->|R1| B", "B -->|contains| C", "C --->|R2| A"} {
		if !strings.Contains(got, edge) {
			t.Fatal(got)
		}
	}
	unchanged, empty := publicationEdgeKey("architecture-intent", graph)
	if unchanged != graph || len(empty) != 0 {
		t.Fatal("changed unrelated view")
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestPublicationEdgeKeyAvoidsExistingLabel(t *testing.T) {
	label := "long original relationship with <br/> explicit break"
	graph, key := publicationEdgeKey("deployment", "A --->|R1| B\nB -->|"+label+"| C")
	if len(key) != 1 || key[0]["Number"] != "R2" || key[0]["Description"] != label || !strings.Contains(graph, "B --->|R2| C") {
		t.Fatalf("%s %#v", graph, key)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestPublicationEdgeKeyTableIsOutsideCompleteDiagram(t *testing.T) {
	graph, key := publicationEdgeKey("deployment", "flowchart LR\nN_A -->|long hardware relationship description retained verbatim| N_B")
	doc, err := renderAsciiDocTemplate(asciidocTemplateData{Views: []asciidocViewSection{{ID: "VIEW-HARDWARE", Kind: "deployment", Mermaid: graph, EdgeKey: key}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(doc, "N_A --->|R1| N_B") != 1 || !strings.Contains(doc, "|R1 |long hardware relationship description retained verbatim") {
		t.Fatal("missing complete graph or description table")
	}
	if strings.Index(doc, ".Relationship descriptions") < strings.Index(doc, "N_A --->|R1| N_B") {
		t.Fatal("description table precedes graph")
	}
}
