// ENGMODEL-OWNER-UNIT: FU-STRUCTURIZR-EXPORTER
package engmodel

// Structurizr DSL export.
// Thin facade over internal/export/structurizr; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/export/structurizr"

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type StructurizrExportResult = structurizr.Result

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateStructurizrDSLFromFile(architecturePath string) (StructurizrExportResult, error) {
	return structurizr.GenerateDSLFromFile(architecturePath)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-005, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateStructurizrDSL(bundle Bundle) (StructurizrExportResult, error) {
	return structurizr.GenerateDSL(bundle)
}
