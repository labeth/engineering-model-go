// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-003
func TestBuildSecurityAttackChapters_IncludesMitigationsAndBoundaries(t *testing.T) {
	a := model.AuthoredArchitecture{
		AttackVectors: []model.AttackVector{{ID: "AV-A", Name: "Attack A"}},
		Mappings: []model.Mapping{
			{Type: "targets", From: "AV-A", To: "FU-A"},
			{Type: "mitigated_by", From: "AV-A", To: "CTRL-A"},
			{Type: "bounded_by", From: "FU-A", To: "TB-A"},
		},
	}
	units := []asciidocUnitSection{{ID: "FU-A", Name: "Unit A"}}
	nodeSet := map[string]bool{"FU-A": true}
	rows := []asciidocSecurityPathRow{{
		AttackVectorID: "AV-A",
		AttackVector:   "Attack A",
		TargetID:       "FU-A",
		Target:         "Unit A",
	}}
	labels := map[string]string{
		"CTRL-A": "Control A",
		"TB-A":   "Boundary A",
	}

	got := buildSecurityAttackChapters(a, units, nodeSet, rows, labels, nil, nil)

	if len(got) != 1 {
		t.Fatalf("expected one attack chapter, got %d", len(got))
	}
	if got[0].MitigatedBy != "Control A" {
		t.Fatalf("expected mitigated_by control in attack chapter, got %q", got[0].MitigatedBy)
	}
	if got[0].TrustBoundaries != "Boundary A" {
		t.Fatalf("expected bounded_by trust boundary in attack chapter, got %q", got[0].TrustBoundaries)
	}
}
