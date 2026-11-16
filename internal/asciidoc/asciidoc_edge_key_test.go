// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"strings"
	"testing"
)

// edgeKeyHasDescriptions reports whether the key has two entries with the
// description, the second numbered R2.
func edgeKeyHasDescriptions(key []map[string]string, label string) bool {
	if len(key) != 2 {
		return false
	}
	return key[0]["Description"] == label && key[1]["Description"] == label && key[1]["Number"] == "R2"
}

// TRLC-LINKS: REQ-EMG-003
func TestPublicationEdgeKeyPreservesRelationships(t *testing.T) {
	label := "interacts_with: complete long hardware relationship description"
	graph := "flowchart LR\nN_TEXT[\"literal -->|this is a long embedded string rather than an edge|\"]\nA -->|" + label + "| B\nB -->|contains| C\nC -->|" + label + "| A\n"
	got, key := publicationEdgeKey("deployment", graph)
	if !edgeKeyHasDescriptions(key, label) {
		t.Fatalf("lost descriptions: %#v", key)
	}
	about(t, got).has(docCheck{`N_TEXT["literal -->|this is a long embedded string rather than an edge|"]`, "rewrote node text"})
	inside(t, got, got).hasShowing("%[2]s", "A --->|R1| B", "B -->|contains| C", "C --->|R2| A")
	unchanged, empty := publicationEdgeKey("architecture-intent", graph)
	if unchanged != graph || len(empty) != 0 {
		t.Fatal("changed unrelated view")
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestPublicationEdgeKeyAvoidsExistingLabel(t *testing.T) {
	label := "long original relationship with <br/> explicit break"
	graph, key := publicationEdgeKey("deployment", "A --->|R1| B\nB -->|"+label+"| C")
	if len(key) != 1 || key[0]["Number"] != "R2" || key[0]["Description"] != label {
		t.Fatalf("%s %#v", graph, key)
	}
	if !strings.Contains(graph, "B --->|R2| C") {
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
