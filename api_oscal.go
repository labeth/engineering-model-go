// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package engmodel

// OSCAL SSP, assessment plan, assessment results, POA&M and profile export.
// Thin facade over internal/export/oscal; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/export/oscal"

type OSCALAPOptions = oscal.APOptions

type OSCALAPResult = oscal.APResult

type OSCALAPDocument = oscal.APDocument

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALAROptions = oscal.AROptions

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type OSCALARResult = oscal.ARResult

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALARDocument = oscal.ARDocument

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALPOAMOptions = oscal.POAMOptions

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type OSCALPOAMResult = oscal.POAMResult

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALPOAMDocument = oscal.POAMDocument

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALProfileOptions = oscal.ProfileOptions

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALSSPOptions = oscal.SSPOptions

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type OSCALSSPResult = oscal.SSPResult

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALSSPDocument = oscal.SSPDocument

var ErrNoPOAMItems = oscal.ErrNoPOAMItems

// ErrNoOSCALProfiles indicates that the model has no compliance profile to
// publish as an aggregate OSCAL profile.
var ErrNoOSCALProfiles = oscal.ErrNoProfiles

var ErrNoOSCALCompliance = oscal.ErrNoCompliance

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateOSCALAssessmentPlanFromFile(architecturePath string, options OSCALAPOptions) (OSCALAPResult, error) {
	return oscal.GenerateAssessmentPlanFromFile(architecturePath, options)
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE
func GenerateOSCALAssessmentPlan(bundle Bundle, options OSCALAPOptions) (OSCALAPResult, error) {
	return oscal.GenerateAssessmentPlan(bundle, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateOSCALAssessmentResultsFromFile(architecturePath string, options OSCALAROptions) (OSCALARResult, error) {
	return oscal.GenerateAssessmentResultsFromFile(architecturePath, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-THREAT-EXPORTER, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateOSCALAssessmentResults(bundle Bundle, requirements RequirementsDocument, options OSCALAROptions) (OSCALARResult, error) {
	return oscal.GenerateAssessmentResults(bundle, requirements, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateOSCALPOAMFromFile(architecturePath string, options OSCALPOAMOptions) (OSCALPOAMResult, error) {
	return oscal.GeneratePOAMFromFile(architecturePath, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-THREAT-EXPORTER, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateOSCALPOAM(bundle Bundle, options OSCALPOAMOptions) (OSCALPOAMResult, error) {
	return oscal.GeneratePOAM(bundle, options)
}

// GenerateOSCALProfileFromFile loads a model and emits one profile importing
// every authored compliance profile.
// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALProfileFromFile(path string, options OSCALProfileOptions) (string, error) {
	return oscal.GenerateProfileFromFile(path, options)
}

// GenerateOSCALProfile emits a deterministic OSCAL 1.1.2 aggregate profile so
// multi-framework SSPs import every framework they implement.
// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALProfile(bundle Bundle, options OSCALProfileOptions) (string, error) {
	return oscal.GenerateProfile(bundle, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateOSCALSSPFromFile(architecturePath string, options OSCALSSPOptions) (OSCALSSPResult, error) {
	return oscal.GenerateSSPFromFile(architecturePath, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateOSCALSSP(bundle Bundle, options OSCALSSPOptions) (OSCALSSPResult, error) {
	return oscal.GenerateSSP(bundle, options)
}
