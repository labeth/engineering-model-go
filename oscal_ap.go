// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package engmodel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

type OSCALAPOptions struct {
	SSPHref      string
	ProfileHref  string
	CatalogHref  string
	LastModified string
}

type OSCALAPResult struct {
	JSON        string
	Document    OSCALAPDocument
	Diagnostics []validate.Diagnostic
}

type OSCALAPDocument struct {
	AssessmentPlan oscalAssessmentPlan `json:"assessment-plan"`
}

type oscalAssessmentPlan struct {
	UUID             string                `json:"uuid"`
	Metadata         oscalMetadata         `json:"metadata"`
	ImportSSP        oscalImportSSP        `json:"import-ssp"`
	ReviewedControls oscalReviewedControls `json:"reviewed-controls"`
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateOSCALAssessmentPlanFromFile(architecturePath string, options OSCALAPOptions) (OSCALAPResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return OSCALAPResult{}, err
	}
	bundle, err = enrichBundleFromComposition(bundle, "architecture", "assurance", "compliance")
	if err != nil {
		return OSCALAPResult{}, err
	}
	return GenerateOSCALAssessmentPlan(bundle, options)
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE
func GenerateOSCALAssessmentPlan(bundle model.Bundle, options OSCALAPOptions) (OSCALAPResult, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return OSCALAPResult{}, err
	}
	bundle = canonical.Documents()
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return OSCALAPResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}

	compliance, complianceDiags := resolveOSCALCompliance(bundle, oscalResolveOptions{
		ProfileHref: options.ProfileHref,
		CatalogHref: options.CatalogHref,
	})
	diags = append(diags, complianceDiags...)
	if validate.HasErrors(diags) {
		return OSCALAPResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}
	controlIDs := oscalMappedControlIDs(compliance)
	if len(controlIDs) == 0 {
		return OSCALAPResult{Diagnostics: validate.SortDiagnostics(diags)}, ErrNoOSCALCompliance
	}

	lastModified, err := resolveOSCALTimestamp(bundle, options.LastModified)
	if err != nil {
		return OSCALAPResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	sspHref := strings.TrimSpace(options.SSPHref)
	if sspHref == "" {
		sspHref = "./OSCAL-SSP.json"
	}
	includeControls := make([]oscalIncludeControlRef, 0, len(controlIDs))
	for _, controlID := range controlIDs {
		includeControls = append(includeControls, oscalIncludeControlRef{ControlID: controlID})
	}

	doc := OSCALAPDocument{AssessmentPlan: oscalAssessmentPlan{
		UUID: deterministicUUID("assessment-plan|" + bundle.Architecture.Model.ID),
		Metadata: oscalMetadata{
			Title:        nonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID)) + " Assessment Plan",
			LastModified: lastModified,
			Version:      "0.1.0",
			OSCALVersion: "1.1.2",
		},
		ImportSSP: oscalImportSSP{Href: sspHref},
		ReviewedControls: oscalReviewedControls{ControlSelections: []oscalControlSelection{{
			Description:     "Controls selected from authored compliance mappings.",
			IncludeControls: includeControls,
		}}},
	}}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return OSCALAPResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	return OSCALAPResult{JSON: string(data), Document: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

// TRLC-LINKS: REQ-EMG-013
func oscalMappedControlIDs(compliance oscalComplianceContext) []string {
	controlSet := map[string]bool{}
	for _, mapping := range compliance.Mappings {
		for _, controlID := range mapping.ControlIDs {
			if normalized := normalizeOSCALControlID(controlID); normalized != "" {
				controlSet[normalized] = true
			}
		}
	}
	controlIDs := make([]string, 0, len(controlSet))
	for controlID := range controlSet {
		controlIDs = append(controlIDs, controlID)
	}
	sort.Strings(controlIDs)
	return controlIDs
}
