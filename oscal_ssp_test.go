// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package engmodel

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-013
func TestGenerateOSCALSSP_FromBundle(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{Model: model.ModelMeta{ID: "sample-system", Title: "Sample System", Introduction: "Sample introduction."}, AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit"}},
		Actors:           []model.Actor{{ID: "ACT-A", Name: "Owner"}},
		Controls:         []model.Control{{ID: "CTRL-A", Name: "MFA"}},
		ControlAllocations: []model.ControlAllocation{{
			ID:                 "ALLOC-A",
			ControlRef:         "CTRL-A",
			OSCALControlIDs:    []string{"ac-2", "ia-2(1)"},
			AppliesTo:          []string{"FU-A"},
			ImplementationType: "technical",
			Status:             "implemented",
			Narrative:          "MFA enforced through SSO policy.",
			Evidence:           []model.ControlEvidence{{Path: "infra/identity/policy.yaml"}},
			ResponsibleRoles:   []string{"ACT-A"},
		}},
	}}}

	res, err := GenerateOSCALSSP(b, OSCALSSPOptions{})
	if err != nil {
		t.Fatalf("generate oscal ssp failed: %v", err)
	}
	if strings.TrimSpace(res.JSON) == "" {
		t.Fatalf("expected non-empty ssp json")
	}
	if !strings.Contains(res.JSON, "\"system-security-plan\"") {
		t.Fatalf("expected system-security-plan root in ssp json")
	}
	if !strings.Contains(res.JSON, "\"control-id\": \"ac-2\"") {
		t.Fatalf("expected allocated control id in ssp json")
	}
	if !strings.Contains(res.JSON, "\"component-uuid\"") {
		t.Fatalf("expected component references in ssp json")
	}

	var decoded OSCALSSPDocument
	if err := json.Unmarshal([]byte(res.JSON), &decoded); err != nil {
		t.Fatalf("expected valid json output: %v", err)
	}
	if decoded.SystemSecurityPlan.SystemCharacteristics.SystemName == "" {
		t.Fatalf("expected non-empty system-name")
	}
}

func TestGenerateOSCALSSPFromFile_PaymentsSample(t *testing.T) {
	path := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	res, err := GenerateOSCALSSPFromFile(path, OSCALSSPOptions{})
	if err != nil {
		t.Fatalf("generate oscal ssp from file failed: %v", err)
	}
	if !strings.Contains(res.JSON, "\"control-id\": \"ac-2\"") {
		t.Fatalf("expected sample control allocation in generated ssp")
	}
}
