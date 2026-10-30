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

// SpecDeltaOperation names the OpenSpec delta section RenderSpecDelta renders.
// ENGMODEL-LINKS: FU-MODEL-CHANGE
type SpecDeltaOperation = reqdelta.SpecDeltaOperation

// SpecDeltaAdded and SpecDeltaModified select "## ADDED Requirements" (default)
// or "## MODIFIED Requirements".
const (
	SpecDeltaAdded    = reqdelta.SpecDeltaAdded
	SpecDeltaModified = reqdelta.SpecDeltaModified
)

// SpecDeltaOptions selects the requirements to render (IDs or Prefix) and the
// optional capability heading and delta operation.
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
type SpecDeltaOptions = reqdelta.SpecDeltaOptions

// SpecDeltaResult is the rendered spec delta markdown, the rendered IDs and warnings.
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
type SpecDeltaResult = reqdelta.SpecDeltaResult

// RenderSpecDelta renders OpenSpec spec delta markdown from model requirements:
// one requirement per entry whose "<ID>: <text>" line starts with the model
// text (shall and must capitalized) and one scenario from verificationCriteria.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func RenderSpecDelta(requirements []Requirement, options SpecDeltaOptions) (SpecDeltaResult, error) {
	return reqdelta.RenderSpecDelta(requirements, options)
}

// RenderSpecDeltaFromModel loads engmod.yml and renders its selected requirements.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, DO-REQUIREMENTS-DOCUMENT
func RenderSpecDeltaFromModel(manifestPath string, options SpecDeltaOptions) (SpecDeltaResult, error) {
	return reqdelta.RenderSpecDeltaFromModel(manifestPath, options)
}
