// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"fmt"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// hardwarePublication is the graph and the chapter of a hardware publication.
type hardwarePublication struct {
	graph   string
	chapter string
}

// requireAllocationsOutsideGraph fails the test for a hosted unit that appears
// in the graph or is missing from the chapter table.
func (h hardwarePublication) requireAllocationsOutsideGraph(t *testing.T, hosts []string) {
	t.Helper()
	for _, id := range hosts {
		if strings.Contains(h.graph, id) || !strings.Contains(h.chapter, id) {
			t.Fatalf("allocation %s must remain in the table, outside the graph label", id)
		}
	}
}

// TRLC-LINKS: REQ-EMG-024
func TestHardwarePublicationPreservesLargeAllocationsOutsideGraphLabels(t *testing.T) {
	hosts := make([]string, 80)
	for i := range hosts {
		hosts[i] = fmt.Sprintf("FU-HOSTED-%03d", i)
	}
	bundle := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		HardwareItems: []model.HardwareItem{
			{ID: "HW-FPGA", Name: "Acquisition FPGA", Hosts: hosts},
			{ID: "HW-RAM"},
		},
		HardwareInterfaces: []model.HardwareInterface{{ID: "HWIF-DATA", From: "HW-FPGA", To: "HW-RAM", BusType: "parallel"}},
	}}}
	graph := compositionHardwareMermaid(bundle)
	chapter := renderCompositionAsciiDocChapter(bundle)
	hardwarePublication{graph: graph, chapter: chapter}.requireAllocationsOutsideGraph(t, hosts)
	about(t, graph).hasShowing("hardware identity or interface missing: %q\n%s", `["HW-FPGA: Acquisition FPGA"]`, `["HW-RAM"]`, " -->|parallel| ")
	about(t, chapter).has(docCheck{"Hardware Items table lists every hosted unit", "diagram must direct readers to full allocation details"})
	if len(bundle.Architecture.AuthoredArchitecture.HardwareItems[0].Hosts) != 80 {
		t.Fatal("rendering changed authored allocations")
	}
}
