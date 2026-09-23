// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"github.com/labeth/engineering-model-go/model"
	"testing"
)

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-050
func TestHardwareViewsAndEndpointValidation(t *testing.T) {
	base := func() model.Bundle {
		return schemaV2TestBundle(model.Bundle{Architecture: model.ArchitectureDocument{
			AuthoredArchitecture: model.AuthoredArchitecture{
				HardwareItems:      []model.HardwareItem{{ID: "HW-BOARD", Name: "Board"}, {ID: "HW-ADC", Name: "ADC"}},
				Interfaces:         []model.Interface{{ID: "IF-ADC", Name: "Samples"}},
				HardwareInterfaces: []model.HardwareInterface{{ID: "HWIF-ADC", From: "HW-ADC", To: "HW-BOARD", SoftwareInterfaceRef: "IF-ADC"}},
				Mappings:           []model.Mapping{{Type: "contains", From: "HW-BOARD", To: "HW-ADC"}, {Type: "interacts_with", From: "HW-ADC", To: "HW-BOARD"}},
			}, Views: []model.View{{ID: "VIEW-HW", Kind: "deployment", Roots: []string{"HW-BOARD"}, IncludeKinds: []string{"hardware_item", "hardware_interface"}}},
		}})
	}
	if d := Bundle(base()); HasErrors(d) {
		t.Fatalf("hardware graph rejected: %+v", d)
	}
	for _, tc := range []struct {
		name, code string
		edit       func(*model.Bundle)
	}{
		{"wrong endpoint kind", "model.invalid_hardware_endpoint", func(b *model.Bundle) { b.Architecture.AuthoredArchitecture.HardwareInterfaces[0].To = "IF-ADC" }},
		{"unknown software interface", "model.invalid_hardware_software_interface", func(b *model.Bundle) {
			b.Architecture.AuthoredArchitecture.HardwareInterfaces[0].SoftwareInterfaceRef = "IF-MISSING"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := base()
			tc.edit(&b)
			for _, d := range Bundle(b) {
				if d.Code == tc.code {
					return
				}
			}
			t.Fatalf("missing %s", tc.code)
		})
	}
}
