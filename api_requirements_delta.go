// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package engmodel

// Requirements delta planning and application.
// Thin facade over internal/reqdelta; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/reqdelta"

// RequirementsDelta describes stable-ID operations against requirements.yml.
// Updates replace the complete requirement entity.
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
type RequirementsDelta = reqdelta.RequirementsDelta

// RequirementsDeltaOperations groups supported requirement mutations.
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA
type RequirementsDeltaOperations = reqdelta.RequirementsDeltaOperations

// RequirementsDeltaResult is the validated candidate and deterministic change summary.
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-VALIDATION-ENGINE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
type RequirementsDeltaResult = reqdelta.RequirementsDeltaResult

// PlanRequirementsDelta loads a delta, applies it in memory, and validates the complete candidate.
// It never modifies requirements.yml.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func PlanRequirementsDelta(root, deltaPath string) (RequirementsDeltaResult, error) {
	return reqdelta.PlanRequirementsDelta(root, deltaPath)
}

// ApplyRequirementsDelta validates and atomically writes a requirements delta.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT, DEP-LOCAL-WORKSPACE
func ApplyRequirementsDelta(root, deltaPath string) (result RequirementsDeltaResult, err error) {
	return reqdelta.ApplyRequirementsDelta(root, deltaPath)
}
