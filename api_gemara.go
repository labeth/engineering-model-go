// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

// OpenSSF Gemara artifacts and their OSCAL forms.
// Thin facade over internal/export/gemara; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/export/gemara"

// GemaraEvaluationResult holds the L5 Evaluation Log and its serialization.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type GemaraEvaluationResult = gemara.EvaluationResult

// GemaraExportOptions configures the deterministic metadata stamped into artifacts.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type GemaraExportOptions = gemara.Options

// GemaraExportResult holds the typed Gemara documents and their YAML serializations.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type GemaraExportResult = gemara.Result

// GemaraCrossReferenceError reports one reference that does not resolve within
// the rendered artifact set.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type GemaraCrossReferenceError = gemara.CrossReferenceError

// GemaraVersion is the Gemara specification version stamped into every artifact's metadata.
const GemaraVersion = gemara.Version

// GenerateGemaraEvaluationLogFromFiles loads inputs and renders the Evaluation Log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraEvaluationLogFromFiles(architecturePath, requirementsPath, codeRoot string, options GemaraExportOptions) (GemaraEvaluationResult, error) {
	return gemara.GenerateEvaluationLogFromFiles(architecturePath, requirementsPath, codeRoot, options)
}

// GenerateGemaraEvaluationLog builds an L5 Evaluation Log from the model bundle.
// requirements and codeRoot are reserved for inferred-verification augmentation.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraEvaluationLog(bundle Bundle, requirements RequirementsDocument, codeRoot string, options GemaraExportOptions) (GemaraEvaluationResult, error) {
	return gemara.GenerateEvaluationLog(bundle, requirements, codeRoot, options)
}

// GenerateGemaraFromFile loads the model bundle and renders the Gemara catalogs.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraFromFile(architecturePath string, options GemaraExportOptions) (GemaraExportResult, error) {
	return gemara.GenerateFromFile(architecturePath, options)
}

// GenerateGemara renders the engineering model bundle into Gemara L1-L3 documents.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemara(bundle Bundle, options GemaraExportOptions) (GemaraExportResult, error) {
	return gemara.Generate(bundle, options)
}

// GenerateGemaraOSCALCatalogFromFile loads the model and emits an OSCAL Catalog
// derived from the Gemara Control Catalog.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALCatalogFromFile(architecturePath string, options GemaraExportOptions) (string, error) {
	return gemara.GenerateOSCALCatalogFromFile(architecturePath, options)
}

// GenerateGemaraOSCALCatalog converts the Gemara Control Catalog to an OSCAL
// Catalog JSON document via the go-gemara SDK. Returns "" when there are no controls.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALCatalog(bundle Bundle, options GemaraExportOptions) (string, error) {
	return gemara.GenerateOSCALCatalog(bundle, options)
}

// GenerateGemaraOSCALAssessmentResultsFromFiles loads inputs and emits OSCAL
// Assessment Results derived from the Gemara evaluation log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALAssessmentResultsFromFiles(architecturePath, requirementsPath, codeRoot string, options GemaraExportOptions) (string, error) {
	return gemara.GenerateOSCALAssessmentResultsFromFiles(architecturePath, requirementsPath, codeRoot, options)
}

// GenerateGemaraOSCALAssessmentResults converts the Gemara Evaluation Log to OSCAL
// Assessment Results JSON via the go-gemara SDK. Returns "" when there is nothing
// to evaluate.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALAssessmentResults(bundle Bundle, requirements RequirementsDocument, codeRoot string, options GemaraExportOptions) (string, error) {
	return gemara.GenerateOSCALAssessmentResults(bundle, requirements, codeRoot, options)
}

// CheckGemaraCrossReferences parses the artifacts (short-name -> YAML) and
// returns every EntryMapping, MultiEntryMapping, artifact import and mapping
// document source/target that does not resolve within the set. Documents that
// fail to parse are reported as errors too.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func CheckGemaraCrossReferences(docs map[string]string) []GemaraCrossReferenceError {
	return gemara.CheckCrossReferences(docs)
}

// CheckGemaraModelConsistency compares the treatment claims in a rendered
// artifact set (short-name -> YAML) with the model they were rendered from: an
// enforcement action's disposition must follow its POA&M item's status (only
// completed is Enforced, only accepted is Tolerated), the log's aggregate must
// follow the actions, the policy may list a risk as mitigated only when it is
// closed and as accepted only when it is accepted. It reports each claim the
// model does not support with the same error shape as the cross-reference check.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func CheckGemaraModelConsistency(bundle Bundle, docs map[string]string) []GemaraCrossReferenceError {
	return gemara.CheckModelConsistency(bundle, docs)
}
