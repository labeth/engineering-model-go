package engmodel

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateOSCALAssessmentResultsFromFile_PaymentsSample(t *testing.T) {
	res, err := GenerateOSCALAssessmentResultsFromFile(filepath.Join("examples", "payments-engineering-sample", "architecture.yml"), OSCALAROptions{
		RequirementsPath: filepath.Join("examples", "payments-engineering-sample", "requirements.yml"),
		CodeRoot:         filepath.Join("examples", "payments-engineering-sample", "src"),
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
}

func TestGenerateOSCALPOAMFromFile_PaymentsSample(t *testing.T) {
	res, err := GenerateOSCALPOAMFromFile(filepath.Join("examples", "payments-engineering-sample", "architecture.yml"), OSCALPOAMOptions{})
	if err != nil {
		t.Fatalf("generate poam failed: %v", err)
	}
	if !strings.Contains(res.JSON, "\"plan-of-action-and-milestones\"") {
		t.Fatalf("expected poam root")
	}
	if !strings.Contains(res.JSON, "\"poam-items\"") {
		t.Fatalf("expected poam-items in poam output")
	}
}
