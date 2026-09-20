// ENGMODEL-OWNER-UNIT: FU-AIRBORNE-ASSURANCE-EXPORTER
package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-053, REQ-EMG-054
func TestAviationSchemaRejectsUnknownFields(t *testing.T) {
	dir, err := os.MkdirTemp(".", ".aviation-schema-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "aviation.yml")
	source := "schemaVersion: 2\naviation:\n  profile:\n    id: AVIATION-TEST\n    standard: DO-178C\n    edition: \"2011\"\n    softwareLevel: C\n    softwareItem: Test\n    applicant: Applicant\n    authority: Authority\n    certificationBasis: REF\n    safetyAllocationReference: SAFETY\n    claimsDisclaimer: DRAFT\n    unexpected: true\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	var document AviationDocument
	err = decodeYAMLFile(path, &document)
	if err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-054
func TestValidateAviationBlockingRules(t *testing.T) {
	tests := []struct {
		name string
		edit func(*AviationDocument)
		code string
	}{
		{"invalid dal", func(d *AviationDocument) { d.Aviation.Profile.SoftwareLevel = "Z" }, "aviation.profile.invalid_dal"},
		{"duplicate id", func(d *AviationDocument) { d.Aviation.Objectives[0].ID = "CASE-1" }, "aviation.duplicate_id"},
		{"requirement reference", func(d *AviationDocument) { d.Aviation.VerificationCases[0].RequirementRefs = []string{"REQ-MISSING"} }, "aviation.reference.requirement"},
		{"derived feedback", func(d *AviationDocument) { d.Aviation.RequirementAssurance[0].Derived = true }, "aviation.derived.incomplete_safety_feedback"},
		{"execution build baseline", func(d *AviationDocument) { d.Aviation.Executions[0].Baseline = "BASELINE-OTHER" }, "aviation.execution.baseline_mismatch"},
		{"baseline hash", func(d *AviationDocument) { d.Aviation.Baselines[0].Hash = "" }, "aviation.baseline.hash_required"},
		{"build hash", func(d *AviationDocument) { d.Aviation.Builds[0].Hash = "" }, "aviation.build.hash_required"},
		{"procedure chain", func(d *AviationDocument) { d.Aviation.Executions[0].ProcedureRef = "PROC-MISSING" }, "aviation.reference.procedure"},
		{"independence", func(d *AviationDocument) { d.Aviation.Executions[0].Verifier = d.Aviation.Executions[0].Producer }, "aviation.independence.missing"},
		{"coverage metric", func(d *AviationDocument) { d.Aviation.Coverage[0].Metric = "decision" }, "aviation.coverage.metric"},
		{"coverage arithmetic", func(d *AviationDocument) { d.Aviation.Coverage[0].Total++ }, "aviation.coverage.arithmetic"},
		{"coverage dispositions", func(d *AviationDocument) { d.Aviation.Coverage[0].Dispositions[0].Count = 0 }, "aviation.coverage.disposition_count"},
		{"coverage closure", func(d *AviationDocument) { d.Aviation.Coverage[0].Status = "accepted" }, "aviation.coverage.open_at_closure"},
		{"tool credit", func(d *AviationDocument) { d.Aviation.Tools[0].CreditClaimed = true }, "aviation.tool.incomplete_credit_assessment"},
		{"authority acceptance", func(d *AviationDocument) { d.Aviation.Objectives[0].State = "authority-accepted" }, "aviation.authority.acceptance_missing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, requirements := validAviationFixture()
			test.edit(&document)
			diagnostics := ValidateAviation(document, requirements)
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == test.code {
					return
				}
			}
			t.Fatalf("missing diagnostic %s: %+v", test.code, diagnostics)
		})
	}
}

// TRLC-LINKS: REQ-EMG-054
func validAviationFixture() (AviationDocument, RequirementsDocument) {
	document := AviationDocument{SchemaVersion: 2, Aviation: AviationProfile{
		Profile:              AirborneProfile{ID: "AVIATION-TEST", Standard: "DO-178C", Edition: "2011", SoftwareLevel: "C", SoftwareItem: "Test", Applicant: "Applicant", Authority: "Authority", CertificationBasis: "REF", SafetyAllocationReference: "SAFETY", ClaimsDisclaimer: "DRAFT"},
		Objectives:           []ObjectiveReference{{ID: "OBJ-1", SourceReference: "CUSTOMER-1", Applicability: "applicable", Rationale: "Customer supplied", EvidenceRefs: []string{"EXEC-1"}, State: "reviewed"}},
		RequirementAssurance: []RequirementAssurance{{RequirementID: "REQ-1", Level: "LLR", VerificationCriteria: "Expected", Baseline: "BASELINE-1", State: "reviewed"}},
		VerificationCases:    []AirborneVerificationCase{{ID: "CASE-1", Method: "test", Intent: "normal", RequirementRefs: []string{"REQ-1"}, ExpectedResult: "Expected", ProcedureRef: "PROC-1", IndependenceRequired: true, State: "reviewed"}},
		Procedures:           []VerificationProcedure{{ID: "PROC-1", CaseRefs: []string{"CASE-1"}, Steps: []string{"Run"}, Environment: "ENV", Target: "TARGET", Baseline: "BASELINE-1", State: "reviewed"}},
		Executions:           []VerificationExecution{{ID: "EXEC-1", ProcedureRef: "PROC-1", CaseRefs: []string{"CASE-1"}, Baseline: "BASELINE-1", Build: "BUILD-1", Environment: "ENV", Target: "TARGET", Producer: "A", Verifier: "B", Status: "reviewed", Result: "Passed", Evidence: []string{"OBJ-1"}, ApprovalRefs: []string{"APPROVAL-1"}}},
		Baselines:            []ConfigurationBaseline{{ID: "BASELINE-1", Version: "1", Hash: "sha256:a", Artifacts: []ConfigurationArtifact{{Path: "a", Hash: "sha256:a"}}, State: "reviewed"}},
		Builds:               []ConfigurationBuild{{ID: "BUILD-1", Baseline: "BASELINE-1", Hash: "sha256:b", Artifacts: []ConfigurationArtifact{{Path: "b", Hash: "sha256:b"}}, Compiler: "cc", Target: "target", Environment: "env", Reproducibility: "recorded", Status: "reviewed"}},
		Coverage:             []StructuralCoverage{{ID: "COVERAGE-1", Metric: "statement", Baseline: "BASELINE-1", Build: "BUILD-1", ToolRefs: []string{"TOOL-1"}, Total: 2, Covered: 1, Uncovered: 1, Dispositions: []UncoveredDisposition{{ID: "GAP-1", Disposition: "open", Count: 1, Rationale: "test pending", Reviewer: "B", Status: "planned"}}, Evidence: []string{"EXEC-1"}, Reviewer: "B", Status: "reviewed"}},
		Tools:                []AirborneTool{{ID: "TOOL-1", Name: "tool", Version: "1", IntendedUse: "measurement", Baseline: "BASELINE-1", Environment: "env", Status: "reviewed"}},
		Approvals:            []AirborneApproval{{ID: "APPROVAL-1", SubjectRefs: []string{"EXEC-1"}, Approver: "C", Evidence: []string{"EXEC-1"}, State: "accepted"}},
	}}
	return document, RequirementsDocument{Requirements: []Requirement{{ID: "REQ-1"}}}
}
