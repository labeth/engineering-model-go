// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package engmodel

// OSCAL SSP, assessment plan, assessment results, POA&M and profile export.
// Thin facade over internal/engine; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/engine"

type OSCALAPOptions = engine.OSCALAPOptions

type OSCALAPResult = engine.OSCALAPResult

type OSCALAPDocument = engine.OSCALAPDocument

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALAROptions = engine.OSCALAROptions

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type OSCALARResult = engine.OSCALARResult

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALARDocument = engine.OSCALARDocument

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALPOAMOptions = engine.OSCALPOAMOptions

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type OSCALPOAMResult = engine.OSCALPOAMResult

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALPOAMDocument = engine.OSCALPOAMDocument

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALProfileOptions = engine.OSCALProfileOptions

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALSSPOptions = engine.OSCALSSPOptions

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type OSCALSSPResult = engine.OSCALSSPResult

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALSSPDocument = engine.OSCALSSPDocument

var ErrNoPOAMItems = engine.ErrNoPOAMItems

// ErrNoOSCALProfiles indicates that the model has no compliance profile to
// publish as an aggregate OSCAL profile.
var ErrNoOSCALProfiles = engine.ErrNoOSCALProfiles

var ErrNoOSCALCompliance = engine.ErrNoOSCALCompliance

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateOSCALAssessmentPlanFromFile(architecturePath string, options OSCALAPOptions) (OSCALAPResult, error) {
	return engine.GenerateOSCALAssessmentPlanFromFile(architecturePath, options)
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE
func GenerateOSCALAssessmentPlan(bundle Bundle, options OSCALAPOptions) (OSCALAPResult, error) {
	return engine.GenerateOSCALAssessmentPlan(bundle, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateOSCALAssessmentResultsFromFile(architecturePath string, options OSCALAROptions) (OSCALARResult, error) {
	return engine.GenerateOSCALAssessmentResultsFromFile(architecturePath, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-THREAT-EXPORTER, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateOSCALAssessmentResults(bundle Bundle, requirements RequirementsDocument, options OSCALAROptions) (OSCALARResult, error) {
	return engine.GenerateOSCALAssessmentResults(bundle, requirements, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateOSCALPOAMFromFile(architecturePath string, options OSCALPOAMOptions) (OSCALPOAMResult, error) {
	return engine.GenerateOSCALPOAMFromFile(architecturePath, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-THREAT-EXPORTER, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateOSCALPOAM(bundle Bundle, options OSCALPOAMOptions) (OSCALPOAMResult, error) {
	return engine.GenerateOSCALPOAM(bundle, options)
}

// GenerateOSCALProfileFromFile loads a model and emits one profile importing
// every authored compliance profile.
// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALProfileFromFile(path string, options OSCALProfileOptions) (string, error) {
	return engine.GenerateOSCALProfileFromFile(path, options)
}

// GenerateOSCALProfile emits a deterministic OSCAL 1.1.2 aggregate profile so
// multi-framework SSPs import every framework they implement.
// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALProfile(bundle Bundle, options OSCALProfileOptions) (string, error) {
	return engine.GenerateOSCALProfile(bundle, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateOSCALSSPFromFile(architecturePath string, options OSCALSSPOptions) (OSCALSSPResult, error) {
	return engine.GenerateOSCALSSPFromFile(architecturePath, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateOSCALSSP(bundle Bundle, options OSCALSSPOptions) (OSCALSSPResult, error) {
	return engine.GenerateOSCALSSP(bundle, options)
}
