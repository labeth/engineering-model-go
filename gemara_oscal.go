// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

// Gemara -> OSCAL bridge. The Gemara SDK can convert a Gemara Control Catalog
// into an OSCAL Catalog and a Gemara Evaluation Log into OSCAL Assessment
// Results. This is ADDITIVE: engmod keeps its hand-written OSCAL SSP/AR/POA&M
// (which encode system characteristics, POA&M items, and compliance-resolved
// controls the Gemara schema does not represent); this path adds a Gemara-sourced
// OSCAL control Catalog (which engmod did not previously emit) and demonstrates
// the Gemara assessment-results bridge.

import (
	"encoding/json"

	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/gemaraproj/go-gemara/gemaraconv"

	"github.com/labeth/engineering-model-go/model"
)

// gemaraOSCALControlHrefFormat is the URL template linking OSCAL controls back to
// their Gemara source (format: href(version, controlID)).
const gemaraOSCALControlHrefFormat = "https://gemara.local/controls/%s#%s"

// GenerateGemaraOSCALCatalogFromFile loads the model and emits an OSCAL Catalog
// derived from the Gemara Control Catalog.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALCatalogFromFile(architecturePath string, options GemaraExportOptions) (string, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return "", err
	}
	return GenerateGemaraOSCALCatalog(bundle, options)
}

// GenerateGemaraOSCALCatalog converts the Gemara Control Catalog to an OSCAL
// Catalog JSON document via the go-gemara SDK. Returns "" when there are no controls.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALCatalog(bundle model.Bundle, options GemaraExportOptions) (string, error) {
	res, err := GenerateGemara(bundle, options)
	if err != nil {
		return "", err
	}
	if len(res.ControlCatalog.Controls) == 0 {
		return "", nil
	}
	oscalCatalog, err := gemaraconv.CatalogToOSCAL(res.ControlCatalog, gemaraconv.WithControlHref(gemaraOSCALControlHrefFormat))
	if err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(oscalTypes.OscalModels{Catalog: &oscalCatalog}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GenerateGemaraOSCALAssessmentResultsFromFiles loads inputs and emits OSCAL
// Assessment Results derived from the Gemara evaluation log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALAssessmentResultsFromFiles(architecturePath, requirementsPath, codeRoot string, options GemaraExportOptions) (string, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return "", err
	}
	var requirements model.RequirementsDocument
	if requirementsPath != "" {
		requirements, err = model.LoadRequirements(requirementsPath)
		if err != nil {
			return "", err
		}
	}
	return GenerateGemaraOSCALAssessmentResults(bundle, requirements, codeRoot, options)
}

// GenerateGemaraOSCALAssessmentResults converts the Gemara Evaluation Log to OSCAL
// Assessment Results JSON via the go-gemara SDK. Returns "" when there is nothing
// to evaluate.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALAssessmentResults(bundle model.Bundle, requirements model.RequirementsDocument, codeRoot string, options GemaraExportOptions) (string, error) {
	evalRes, err := GenerateGemaraEvaluationLog(bundle, requirements, codeRoot, options)
	if err != nil {
		return "", err
	}
	if !evalRes.HasContent {
		return "", nil
	}
	ar, err := gemaraconv.EvaluationLogToOSCALAssessmentResults(evalRes.EvaluationLog, gemaraconv.WithImportApHref("#"))
	if err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(oscalTypes.OscalModels{AssessmentResults: &ar}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
