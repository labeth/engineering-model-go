// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"github.com/labeth/engineering-model-go/model"
	"testing"
)

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-050
func TestHardwareEntitiesAreDiscoverable(t *testing.T) {
	s := NewServer()
	s.bundle = &model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		HardwareItems:      []model.HardwareItem{{ID: "HW-ADC", Name: "Converter", PartNumber: "AD9288"}},
		HardwareInterfaces: []model.HardwareInterface{{ID: "HWIF-LANES", Name: "ADC data lanes", From: "HW-ADC", To: "HW-FPGA"}},
	}}}
	for id, want := range map[string]string{"HW-ADC": "hardware_item", "HWIF-LANES": "hardware_interface"} {
		kind, entity, ok := s.modelEntity(id)
		if !ok || kind != want || entity == nil {
			t.Fatalf("entity lookup %s: %s, %v, %v", id, kind, entity, ok)
		}
		nodes := s.modelEntityNodes(id, want, 10)
		if len(nodes) != 1 || nodes[0]["id"] != id {
			t.Fatalf("entity list missing %s: %+v", id, nodes)
		}
	}
	if nodes := s.graphNodes("Converter", 10); len(nodes) != 1 || nodes[0]["id"] != "HW-ADC" {
		t.Fatalf("hardware name search: %+v", nodes)
	}
}
