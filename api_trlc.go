// ENGMODEL-OWNER-UNIT: FU-TRLC-EXPORTER
package engmodel

// TRLC requirements export.
// Thin facade over internal/export/trlc; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/export/trlc"

// ENGMODEL-LINKS: FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type TRLCExportOptions = trlc.Options

// ENGMODEL-LINKS: FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type TRLCExportResult = trlc.Result

// TRLC-LINKS: REQ-EMG-006
// ENGMODEL-LINKS: FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateTRLCRequirementsFromFile(requirementsPath string, options TRLCExportOptions) (TRLCExportResult, error) {
	return trlc.GenerateRequirementsFromFile(requirementsPath, options)
}

// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateTRLCRequirements(requirements RequirementsDocument, options TRLCExportOptions) (TRLCExportResult, error) {
	return trlc.GenerateRequirements(requirements, options)
}
