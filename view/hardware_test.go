// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package view

import (
	"github.com/labeth/engineering-model-go/model"
	"testing"
)

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-050
func TestHardwareViewPreservesLabelsKindsAndConnections(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			HardwareItems:      []model.HardwareItem{{ID: "HW-BOARD", Name: "Instrument board"}, {ID: "HW-ADC", Name: "Converter"}},
			HardwareInterfaces: []model.HardwareInterface{{ID: "HWIF-DATA", Name: "Parallel samples", From: "HW-ADC", To: "HW-BOARD"}},
			Mappings:           []model.Mapping{{Type: "contains", From: "HW-BOARD", To: "HW-ADC"}, {Type: "contains", From: "HW-BOARD", To: "HWIF-DATA"}, {Type: "interacts_with", From: "HW-ADC", To: "HW-BOARD"}},
		}, Views: []model.View{{ID: "VIEW-HW", Kind: "deployment", Roots: []string{"HW-BOARD"}, IncludeKinds: []string{"hardware_item", "hardware_interface"}, IncludeMappings: []string{"contains", "interacts_with"}}},
	}}
	v, d := Build(b, "VIEW-HW")
	if len(d) != 0 {
		t.Fatalf("diagnostics: %+v", d)
	}
	if len(v.Nodes) != 3 || len(v.Edges) != 3 {
		t.Fatalf("incomplete hardware view: %+v", v)
	}
	for _, n := range v.Nodes {
		if n.Kind == "unknown" || n.Label == n.ID {
			t.Fatalf("hardware identity lost: %+v", n)
		}
	}
}
