// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package view

import (
	"github.com/labeth/engineering-model-go/internal/model"
	"testing"
)

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-050
func TestSecurityViewPreservesBoundaryMembersAndExplicitFilters(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			Actors:          []model.Actor{{ID: "ACT-HOST", Name: "Operator host"}},
			HardwareItems:   []model.HardwareItem{{ID: "HW-FPGA", Name: "FPGA"}},
			DataObjects:     []model.DataObject{{ID: "DO-RECORD", Name: "Record"}},
			TrustBoundaries: []model.TrustBoundary{{ID: "TB-DEVICE", Name: "Device"}},
			Mappings: []model.Mapping{
				{Type: "bounded_by", From: "ACT-HOST", To: "TB-DEVICE"},
				{Type: "bounded_by", From: "HW-FPGA", To: "TB-DEVICE"},
				{Type: "bounded_by", From: "DO-RECORD", To: "TB-DEVICE"},
			},
		}, Views: []model.View{{ID: "VIEW-SEC", Kind: "security", Roots: []string{"TB-DEVICE"}}},
	}}
	v := mustBuild(t, b, "VIEW-SEC")
	want := map[string]string{"ACT-HOST": "actor", "HW-FPGA": "hardware_item", "DO-RECORD": "data_object", "TB-DEVICE": "trust_boundary"}
	if len(v.Nodes) != len(want) || len(v.Edges) != 3 {
		t.Fatalf("lost boundary graph: %+v", v)
	}
	requireMemberIdentity(t, v, want)
	b.Architecture.Views[0].ExcludeKinds = []string{"actor"}
	v = mustBuild(t, b, "VIEW-SEC")
	if len(v.Nodes) != 3 || len(v.Edges) != 2 {
		t.Fatalf("explicit exclusion ignored: %+v", v)
	}
	if hasNodeID(v, "ACT-HOST") {
		t.Fatal("excluded actor retained")
	}
}
