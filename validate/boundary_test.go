// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"github.com/labeth/engineering-model-go/model"
	"testing"
)

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-050
func TestBoundaryMembershipAcrossSoftwareHardwareAndActors(t *testing.T) {
	b := schemaV2TestBundle(model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			Actors:          []model.Actor{{ID: "ACT-HOST", Name: "Operator host"}},
			HardwareItems:   []model.HardwareItem{{ID: "HW-FPGA", Name: "FPGA"}},
			DataObjects:     []model.DataObject{{ID: "DO-RECORD", Name: "Record"}},
			TrustBoundaries: []model.TrustBoundary{{ID: "TB-DEVICE", Name: "Device", Members: []string{"ACT-HOST", "HW-FPGA", "DO-RECORD"}}},
			Mappings: []model.Mapping{
				{Type: "bounded_by", From: "ACT-HOST", To: "TB-DEVICE"},
				{Type: "bounded_by", From: "HW-FPGA", To: "TB-DEVICE"},
				{Type: "bounded_by", From: "DO-RECORD", To: "TB-DEVICE"},
			},
		}, Views: []model.View{{ID: "VIEW-SEC", Kind: "security", Roots: []string{"TB-DEVICE"}}},
	}})
	if d := Bundle(b); HasErrors(d) {
		t.Fatalf("valid boundary memberships rejected: %+v", d)
	}
	// Supporting more member kinds must not permit a non-boundary target.
	b.Architecture.AuthoredArchitecture.Mappings[0].To = "HW-FPGA"
	for _, d := range Bundle(b) {
		if d.Code == "model.invalid_mapping_pair" {
			return
		}
	}
	t.Fatal("missing invalid_mapping_pair for non-boundary target")
}
