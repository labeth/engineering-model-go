// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package oscal

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

type APOptions struct {
	SSPHref      string
	ProfileHref  string
	CatalogHref  string
	LastModified string
}

type APResult struct {
	JSON        string
	Document    APDocument
	Diagnostics []validate.Diagnostic
}

type APDocument struct {
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
func GenerateAssessmentPlanFromFile(architecturePath string, options APOptions) (APResult, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return APResult{}, err
	}
	bundle, err = composition.EnrichBundleFromComposition(bundle, "architecture", "assurance", "compliance")
	if err != nil {
		return APResult{}, err
	}
	return GenerateAssessmentPlan(bundle, options)
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE
func GenerateAssessmentPlan(bundle model.Bundle, options APOptions) (APResult, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return APResult{}, err
	}
	bundle = canonical.Documents()
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return APResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}

	compliance, complianceDiags := resolveOSCALCompliance(bundle, oscalResolveOptions{
		ProfileHref: options.ProfileHref,
		CatalogHref: options.CatalogHref,
	})
	diags = append(diags, complianceDiags...)
	if validate.HasErrors(diags) {
		return APResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}
	controlIDs := oscalMappedControlIDs(compliance)
	if len(controlIDs) == 0 {
		return APResult{Diagnostics: validate.SortDiagnostics(diags)}, ErrNoCompliance
	}

	lastModified, err := resolveOSCALTimestamp(bundle, options.LastModified)
	if err != nil {
		return APResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	doc := APDocument{AssessmentPlan: buildAssessmentPlan(bundle, apHeader{LastModified: lastModified, SSPHref: apSSPHref(options)}, controlIDs)}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return APResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	return APResult{JSON: string(data), Document: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func apSSPHref(options APOptions) string {
	sspHref := strings.TrimSpace(options.SSPHref)
	if sspHref == "" {
		sspHref = "./OSCAL-SSP.json"
	}
	return sspHref
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type apHeader struct {
	LastModified string
	SSPHref      string
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildAssessmentPlan(bundle model.Bundle, header apHeader, controlIDs []string) oscalAssessmentPlan {
	return oscalAssessmentPlan{
		UUID: textutil.DeterministicUUID("assessment-plan|" + bundle.Architecture.Model.ID),
		Metadata: oscalMetadata{
			Title:        textutil.NonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID)) + " Assessment Plan",
			LastModified: header.LastModified,
			Version:      "0.1.0",
			OSCALVersion: "1.1.2",
		},
		ImportSSP: oscalImportSSP{Href: header.SSPHref},
		ReviewedControls: oscalReviewedControls{ControlSelections: []oscalControlSelection{{
			Description:     "Controls selected from authored compliance mappings.",
			IncludeControls: includeControlRefs(controlIDs),
		}}},
	}
}

// TRLC-LINKS: REQ-EMG-013
func oscalMappedControlIDs(compliance oscalComplianceContext) []string {
	controlSet := map[string]bool{}
	for _, mapping := range compliance.Mappings {
		addNormalizedControlIDs(controlSet, mapping.ControlIDs)
	}
	controlIDs := make([]string, 0, len(controlSet))
	for controlID := range controlSet {
		controlIDs = append(controlIDs, controlID)
	}
	sort.Strings(controlIDs)
	return controlIDs
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func addNormalizedControlIDs(set map[string]bool, rawIDs []string) {
	for _, controlID := range rawIDs {
		if normalized := normalizeOSCALControlID(controlID); normalized != "" {
			set[normalized] = true
		}
	}
}
