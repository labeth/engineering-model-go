// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package engmodel

import (
	"encoding/json"
	"os"
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
	}, Compliance: model.ComplianceModel{
		Profiles: []model.ComplianceProfile{{ID: "PROF-A", Href: "https://example.invalid/oscal-profile.json"}},
		Mappings: []model.ComplianceMapping{{
			ID:                   "MAP-A",
			ProfileRef:           "PROF-A",
			ModelControlRef:      "CTRL-A",
			ControlIDs:           []string{"ac-2", "ia-2.1"},
			AppliesTo:            []string{"FU-A"},
			ImplementationType:   "technical",
			ImplementationStatus: "implemented",
			Narrative:            "MFA enforced through SSO policy.",
			Evidence:             []model.ControlEvidence{{Path: "infra/identity/policy.yaml"}},
			ResponsibleRoles:     []string{"ACT-A"},
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

// TRLC-LINKS: REQ-EMG-013
func TestGenerateOSCALSSPFromFile_PaymentsSample(t *testing.T) {
	path := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	res, err := GenerateOSCALSSPFromFile(path, OSCALSSPOptions{})
	if err != nil {
		t.Fatalf("generate oscal ssp from file failed: %v", err)
	}
	if !strings.Contains(res.JSON, "\"control-id\": \"ac-2\"") {
		t.Fatalf("expected sample compliance mapping in generated ssp")
	}
	for _, id := range []string{"cm-6", "ia-2", "ia-2.1"} {
		if !strings.Contains(res.JSON, "\"control-id\": \""+id+"\"") {
			t.Fatalf("expected NIST 800-53 Low sample control %s in generated ssp", id)
		}
	}
	if strings.Contains(res.JSON, "\"control-id\": \"si-10\"") {
		t.Fatalf("did not expect out-of-scope SI-10 control in Low profile generated ssp")
	}
}

// TRLC-LINKS: REQ-EMG-013
func TestGenerateOSCALSSP_UsesComplianceProfileMappings(t *testing.T) {
	tmp := t.TempDir()
	writeOSCALFixtures(t, tmp)
	b := complianceFixtureBundle(tmp, []string{"ac-2"})

	res, err := GenerateOSCALSSP(b, OSCALSSPOptions{})
	if err != nil {
		t.Fatalf("generate oscal ssp failed: %v\n%+v", err, res.Diagnostics)
	}
	if !strings.Contains(res.JSON, "\"control-id\": \"ac-2\"") {
		t.Fatalf("expected profile-selected control in ssp json")
	}
	if strings.Contains(res.JSON, "\"control-id\": \"ia-2.1\"") {
		t.Fatalf("did not expect unselected profile control in ssp json")
	}
	if !strings.Contains(res.JSON, "\"name\": \"model-control-ref\"") || !strings.Contains(res.JSON, "\"value\": \"CTRL-A\"") {
		t.Fatalf("expected model control reference prop in ssp json")
	}
}

// TRLC-LINKS: REQ-EMG-013
func TestGenerateOSCALSSP_RejectsComplianceMappingOutsideProfile(t *testing.T) {
	tmp := t.TempDir()
	writeOSCALFixtures(t, tmp)
	b := complianceFixtureBundle(tmp, []string{"ia-2.1"})

	res, err := GenerateOSCALSSP(b, OSCALSSPOptions{})
	if err == nil {
		t.Fatalf("expected profile validation failure")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "oscal.control_not_in_profile" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected oscal.control_not_in_profile diagnostic, got %+v", res.Diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-013
func writeOSCALFixtures(t *testing.T, dir string) {
	t.Helper()
	catalog := `{
  "catalog": {
    "uuid": "00000000-0000-4000-8000-000000000001",
    "metadata": {"title": "Test Catalog", "last-modified": "2026-01-01T00:00:00Z", "version": "1.0.0", "oscal-version": "1.1.2"},
    "groups": [{
      "id": "ac",
      "title": "Access Control",
      "controls": [
        {"id": "ac-2", "title": "Account Management"},
        {"id": "ia-2.1", "title": "Multi-Factor Authentication"}
      ]
    }]
  }
}`
	profile := `{
  "profile": {
    "uuid": "00000000-0000-4000-8000-000000000002",
    "metadata": {"title": "Test Profile", "last-modified": "2026-01-01T00:00:00Z", "version": "1.0.0", "oscal-version": "1.1.2"},
    "imports": [{
      "href": "catalog.json",
      "include-controls": [{"with-ids": ["ac-2"]}]
    }]
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(catalog), 0o644); err != nil {
		t.Fatalf("write catalog fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.json"), []byte(profile), 0o644); err != nil {
		t.Fatalf("write profile fixture: %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-013
func complianceFixtureBundle(dir string, controlIDs []string) model.Bundle {
	return model.Bundle{ArchitecturePath: filepath.Join(dir, "architecture.yml"), Architecture: model.ArchitectureDocument{
		Model: model.ModelMeta{ID: "sample-system", Title: "Sample System", Introduction: "Sample introduction."},
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group"}},
			FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit"}},
			Actors:           []model.Actor{{ID: "ACT-A", Name: "Owner"}},
			Controls:         []model.Control{{ID: "CTRL-A", Name: "SSO MFA"}},
		},
		Compliance: model.ComplianceModel{
			Profiles: []model.ComplianceProfile{{ID: "PROF-TEST", Href: "profile.json", CatalogHref: "catalog.json"}},
			Mappings: []model.ComplianceMapping{{
				ID:                   "MAP-A",
				ProfileRef:           "PROF-TEST",
				ControlIDs:           controlIDs,
				ModelControlRef:      "CTRL-A",
				AppliesTo:            []string{"FU-A"},
				ImplementationType:   "technical",
				ImplementationStatus: "implemented",
				Narrative:            "SSO with MFA implements account access controls.",
				Evidence:             []model.ControlEvidence{{Path: "infra/identity/policy.yaml"}},
				ResponsibleRoles:     []string{"ACT-A"},
			}},
		},
	}}
}
