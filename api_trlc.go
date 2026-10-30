// ENGMODEL-OWNER-UNIT: FU-TRLC-EXPORTER
package engmodel

// TRLC requirements export.
// Thin facade over internal/engine; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/engine"

// ENGMODEL-LINKS: FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type TRLCExportOptions = engine.TRLCExportOptions

// ENGMODEL-LINKS: FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type TRLCExportResult = engine.TRLCExportResult

// TRLC-LINKS: REQ-EMG-006
// ENGMODEL-LINKS: FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateTRLCRequirementsFromFile(requirementsPath string, options TRLCExportOptions) (TRLCExportResult, error) {
	return engine.GenerateTRLCRequirementsFromFile(requirementsPath, options)
}

// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateTRLCRequirements(requirements RequirementsDocument, options TRLCExportOptions) (TRLCExportResult, error) {
	return engine.GenerateTRLCRequirements(requirements, options)
}
