// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-003
func TestLargePublicationDiagramsRemainComplete(t *testing.T) {
	var code []inferredCodeItem
	var checks []inferredVerificationCheck
	for i := 0; i < 20; i++ {
		code = append(code, inferredCodeItem{Kind: "symbol", Owner: "FU-A", Source: fmt.Sprintf("src/file%d.go:%d", i, 100+i), Implements: []string{"REQ-A"}})
		checks = append(checks, inferredVerificationCheck{ID: fmt.Sprintf("VER-%d", i), Status: "partial", Verifies: []string{"REQ-A"}, CodeElements: []string{fmt.Sprintf("tests/file%d_test.go:10", i)}})
	}
	graph := buildRequirementCoverageMermaid([]model.Requirement{{ID: "REQ-A", AppliesTo: []string{"FU-A"}}}, nil, code, checks, nil, nil)
	for _, kind := range []string{"architecture-intent", "state-lifecycle", "interaction-flow"} {
		doc, err := renderAsciiDocTemplate(asciidocTemplateData{
			Views:        []asciidocViewSection{{ID: "VIEW-LARGE", Kind: kind, Mermaid: graph}},
			Requirements: []asciidocRequirementSection{{ID: "REQ-A", CoverageMermaid: graph}},
		})
		if err != nil {
			t.Fatal(err)
		}
		block := "[source,mermaid]\n----\n" + graph + "\n----"
		if strings.Count(doc, block) != 2 {
			t.Fatalf("%s: view and requirement must each retain exactly one complete graph", kind)
		}
		for _, unwanted := range []string{"Implementation files 1", "Evidence part", "Diagram part", "divided into numbered parts"} {
			if strings.Contains(doc, unwanted) {
				t.Fatalf("split publication returned: %s", unwanted)
			}
		}
	}
}
