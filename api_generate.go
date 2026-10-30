// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engmodel

// Single-view Mermaid generation.
// Thin facade over internal/generate; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/generate"

// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type Result = generate.Result

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION
func GenerateFromFile(architecturePath, viewID string) (Result, error) {
	return generate.GenerateFromFile(architecturePath, viewID)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func Generate(bundle Bundle, viewID string) (Result, error) {
	return generate.Generate(bundle, viewID)
}
