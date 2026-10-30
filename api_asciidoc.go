// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

// AsciiDoc architecture document generation.
// Thin facade over internal/engine; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/engine"

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type AsciiDocOptions = engine.AsciiDocOptions

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type AsciiDocResult = engine.AsciiDocResult

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014
func GenerateAsciiDocFromFiles(architecturePath, requirementsPath, designPath string, options AsciiDocOptions) (AsciiDocResult, error) {
	return engine.GenerateAsciiDocFromFiles(architecturePath, requirementsPath, designPath, options)
}

// GenerateAsciiDoc validates and renders the model. With composition it
// resolves the dependencies once, validates against the bundle enriched with
// their published entities, and returns every composition diagnostic; a
// composition error fails the call like any other validation error.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066
func GenerateAsciiDoc(bundle Bundle, requirements RequirementsDocument, design DesignDocument, options AsciiDocOptions) (AsciiDocResult, error) {
	return engine.GenerateAsciiDoc(bundle, requirements, design, options)
}
