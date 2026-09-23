// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

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
	for _, id := range hosts {
		if strings.Contains(graph, id) || !strings.Contains(chapter, id) {
			t.Fatalf("allocation %s must remain in the table, outside the graph label", id)
		}
	}
	for _, want := range []string{`["HW-FPGA: Acquisition FPGA"]`, `["HW-RAM"]`, " -->|parallel| "} {
		if !strings.Contains(graph, want) {
			t.Fatalf("hardware identity or interface missing: %q\n%s", want, graph)
		}
	}
	if !strings.Contains(chapter, "Hardware Items table lists every hosted unit") {
		t.Fatal("diagram must direct readers to full allocation details")
	}
	if len(bundle.Architecture.AuthoredArchitecture.HardwareItems[0].Hosts) != 80 {
		t.Fatal("rendering changed authored allocations")
	}
}
