// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package oscal

import (
	"path/filepath"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013
func TestGenerateOSCALAssessmentPlanFromFile_PaymentsSample(t *testing.T) {
	const lastModified = "2026-09-19T00:00:00Z"
	res, err := GenerateAssessmentPlanFromFile(filepath.Join("examples", "payments-engineering-sample", "engmod.yml"), APOptions{
		SSPHref:      "./ARCHITECTURE.ssp.json",
		LastModified: lastModified,
	})
	if err != nil {
		t.Fatalf("generate assessment plan failed: %v", err)
	}
	if !strings.Contains(res.JSON, "\"assessment-plan\"") {
		t.Fatalf("expected assessment-plan root")
	}
	if !strings.Contains(res.JSON, "\"href\": \"./ARCHITECTURE.ssp.json\"") {
		t.Fatalf("expected assessment plan to import the sibling SSP")
	}
	if !strings.Contains(res.JSON, "\"last-modified\": \""+lastModified+"\"") {
		t.Fatalf("expected deterministic last-modified timestamp")
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-050
func TestGenerateOSCALAssessmentResultsFromFile_PaymentsSample(t *testing.T) {
	const lastModified = "2026-09-19T00:00:00Z"
	res, err := GenerateAssessmentResultsFromFile(filepath.Join("examples", "payments-engineering-sample", "engmod.yml"), AROptions{
		RequirementsPath: filepath.Join("examples", "payments-engineering-sample", "model", "requirements.yml"),
		CodeRoot:         filepath.Join("examples", "payments-engineering-sample", "src"),
		LastModified:     lastModified,
	})
	if err != nil {
		t.Fatalf("generate assessment results failed: %v", err)
	}
	if !strings.Contains(res.JSON, "\"assessment-results\"") {
		t.Fatalf("expected assessment-results root")
	}
	if !strings.Contains(res.JSON, "\"reviewed-controls\"") {
		t.Fatalf("expected reviewed-controls in assessment results")
	}
	if !strings.Contains(res.JSON, "\"last-modified\": \""+lastModified+"\"") {
		t.Fatalf("expected deterministic last-modified timestamp")
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-050
func TestGenerateOSCALPOAMFromFile_PaymentsSample(t *testing.T) {
	const lastModified = "2026-09-19T00:00:00Z"
	res, err := GeneratePOAMFromFile(filepath.Join("examples", "payments-engineering-sample", "engmod.yml"), POAMOptions{LastModified: lastModified})
	if err != nil {
		t.Fatalf("generate poam failed: %v", err)
	}
	if !strings.Contains(res.JSON, "\"plan-of-action-and-milestones\"") {
		t.Fatalf("expected poam root")
	}
	if !strings.Contains(res.JSON, "\"poam-items\"") {
		t.Fatalf("expected poam-items in poam output")
	}
	if !strings.Contains(res.JSON, "\"last-modified\": \""+lastModified+"\"") {
		t.Fatalf("expected deterministic last-modified timestamp")
	}
	if !strings.Contains(res.JSON, "\"ns\": \""+engineeringModelOSCALNamespace+"\"") {
		t.Fatalf("expected custom POA&M properties to use the engineering-model namespace")
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-050
func TestGenerateOSCALPOAM_IncludesPublishedDependencyItems(t *testing.T) {
	result, err := GeneratePOAMFromFile(
		filepath.Join("examples", "atlas-industries", "repos", "aegis-sentinel", "engmod.yml"),
		POAMOptions{LastModified: "2026-09-19T00:00:00Z"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.JSON, "compliance::POAM-COMP-001") ||
		!strings.Contains(result.JSON, "compliance::ACT-COMP-OFFICER") {
		t.Fatalf("expected qualified published compliance content:\n%s", result.JSON)
	}
}

// Without an explicit last-modified the documents carry the latest authored
// decision or verification date, so regeneration needs no wall clock.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013
func TestGenerateOSCALPOAMDefaultsToAuthoredTimestamp(t *testing.T) {
	res, err := GeneratePOAMFromFile(filepath.Join("examples", "payments-engineering-sample", "engmod.yml"), POAMOptions{})
	if err != nil {
		t.Fatalf("generate poam failed: %v", err)
	}
	if !strings.Contains(res.JSON, "\"last-modified\": \"2026-04-29T00:00:00Z\"") {
		t.Fatalf("expected the latest authored date as last-modified:\n%.600s", res.JSON)
	}
}
