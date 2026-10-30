// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel

// EARS preflight diagnostics.
// Thin facade over internal/engine; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/engine"

// SuppressWaivedDiagnostics removes requirement-scoped diagnostics a model has
// explicitly waived. waivers maps a requirement id to the diagnostic codes
// waived for it (for example "lint.multiple_responses"). Only warnings are
// suppressed: an error is never waived. Diagnostics whose path is not a
// requirement path are kept.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func SuppressWaivedDiagnostics(diags []Diagnostic, waivers map[string][]string) []Diagnostic {
	return engine.SuppressWaivedDiagnostics(diags, waivers)
}
