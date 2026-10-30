// ENGMODEL-OWNER-UNIT: FU-THREAT-EXPORTER
package engmodel

// Threat Dragon v2 and Open Threat Model export.
// Thin facade over internal/export/threatmodel; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/export/threatmodel"

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type ThreatModelFormat = threatmodel.Format

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type ThreatModelExportOptions = threatmodel.Options

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type ThreatModelExportResult = threatmodel.Result

const ThreatModelFormatThreatDragonV2 = threatmodel.FormatThreatDragonV2

const ThreatModelFormatOpenOTM = threatmodel.FormatOpenOTM

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateThreatModelExportFromFile(architecturePath string, options ThreatModelExportOptions) (ThreatModelExportResult, error) {
	return threatmodel.GenerateFromFile(architecturePath, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-004, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateThreatModelExport(bundle Bundle, options ThreatModelExportOptions) (ThreatModelExportResult, error) {
	return threatmodel.Generate(bundle, options)
}
