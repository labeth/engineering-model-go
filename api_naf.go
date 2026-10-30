// ENGMODEL-OWNER-UNIT: FU-NAF-EXPORTER
package engmodel

// NATO Architecture Framework v4.1 export.
// Thin facade over internal/engine; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/engine"

// NAFExportResult is a deterministic NAF v4.1 architecture document and its diagnostics.
type NAFExportResult = engine.NAFExportResult

// GenerateNAFV41FromFile loads canonical YAML and generates the selected NAF products.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-042, REQ-EMG-043
// ENGMODEL-LINKS: FU-NAF-EXPORTER, IF-CLI-ENGNAF, DO-NAF-V4-ARCHITECTURE, REF-NAF-V4-1-SPECIFICATION
func GenerateNAFV41FromFile(architecturePath string) (NAFExportResult, error) {
	return engine.GenerateNAFV41FromFile(architecturePath)
}

// GenerateNAFV41 validates and projects existing Engmod views into NAF products.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-042, REQ-EMG-043
func GenerateNAFV41(bundle Bundle) (NAFExportResult, error) {
	return engine.GenerateNAFV41(bundle)
}
