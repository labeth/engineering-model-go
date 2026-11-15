// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package oscal

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type AROptions struct {
	AssessmentPlanHref string
	RequirementsPath   string
	CodeRoot           string
	ProfileHref        string
	CatalogHref        string
	LastModified       string
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type ARResult struct {
	JSON        string
	Document    ARDocument
	Diagnostics []validate.Diagnostic
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type ARDocument struct {
	AssessmentResults oscalAssessmentResults `json:"assessment-results"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER
type oscalAssessmentResults struct {
	UUID     string          `json:"uuid"`
	Metadata oscalMetadata   `json:"metadata"`
	ImportAP oscalImportAP   `json:"import-ap"`
	Results  []oscalARResult `json:"results"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalImportAP struct {
	Href string `json:"href"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER
type oscalARResult struct {
	UUID             string                `json:"uuid"`
	Title            string                `json:"title"`
	Description      string                `json:"description"`
	Start            string                `json:"start"`
	End              string                `json:"end"`
	ReviewedControls oscalReviewedControls `json:"reviewed-controls"`
	Findings         []oscalFinding        `json:"findings,omitempty"`
	Risks            []oscalARRisk         `json:"risks,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalReviewedControls struct {
	ControlSelections []oscalControlSelection `json:"control-selections"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalControlSelection struct {
	Description     string                   `json:"description,omitempty"`
	IncludeControls []oscalIncludeControlRef `json:"include-controls,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalIncludeControlRef struct {
	ControlID string `json:"control-id"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalFinding struct {
	UUID        string             `json:"uuid"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Target      oscalFindingTarget `json:"target"`
	Props       []oscalProperty    `json:"props,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalFindingTarget struct {
	Type     string                 `json:"type"`
	TargetID string                 `json:"target-id"`
	Status   oscalOperationalStatus `json:"status"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER
type oscalARRisk struct {
	UUID        string          `json:"uuid"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Statement   string          `json:"statement"`
	Props       []oscalProperty `json:"props,omitempty"`
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateAssessmentResultsFromFile(architecturePath string, options AROptions) (ARResult, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return ARResult{}, err
	}
	var req model.RequirementsDocument
	if strings.TrimSpace(options.RequirementsPath) != "" {
		req, err = model.LoadRequirements(options.RequirementsPath)
		if err != nil {
			return ARResult{}, err
		}
		bundle.Requirements = req
	}
	bundle, err = composition.EnrichBundleFromComposition(bundle, "architecture", "assurance", "compliance", "requirements")
	if err != nil {
		return ARResult{}, err
	}
	req = bundle.Requirements
	if strings.TrimSpace(options.CodeRoot) != "" && !filepath.IsAbs(options.CodeRoot) {
		baseDir := filepath.Dir(architecturePath)
		options.CodeRoot = filepath.Join(baseDir, options.CodeRoot)
	}
	return GenerateAssessmentResults(bundle, req, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-THREAT-EXPORTER, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateAssessmentResults(bundle model.Bundle, requirements model.RequirementsDocument, options AROptions) (ARResult, error) {
	bundle.Requirements = requirements
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return ARResult{}, err
	}
	bundle = canonical.Documents()
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return arValidationFailure(diags)
	}

	evidence := collectAREvidence(bundle, options)
	diags = append(diags, evidence.Diagnostics...)
	if validate.HasErrors(diags) {
		return arValidationFailure(diags)
	}
	if len(evidence.Compliance.Mappings) == 0 {
		return ARResult{Diagnostics: validate.SortDiagnostics(diags)}, ErrNoCompliance
	}

	now, err := resolveOSCALTimestamp(bundle, options.LastModified)
	if err != nil {
		return ARResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	doc := ARDocument{AssessmentResults: buildAssessmentResults(bundle, evidence, arHeader{Now: now, AssessmentPlanHref: arAssessmentPlanHref(options)})}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return ARResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	return ARResult{JSON: string(b), Document: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func arValidationFailure(diags []validate.Diagnostic) (ARResult, error) {
	return ARResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func arAssessmentPlanHref(options AROptions) string {
	apHref := strings.TrimSpace(options.AssessmentPlanHref)
	if apHref == "" {
		apHref = "./ASSESSMENT-PLAN.json"
	}
	return apHref
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type arEvidence struct {
	Verification []codescan.VerificationCheck
	Compliance   oscalComplianceContext
	Diagnostics  []validate.Diagnostic
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type arHeader struct {
	Now                string
	AssessmentPlanHref string
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func collectAREvidence(bundle model.Bundle, options AROptions) arEvidence {
	inferredCode, codeDiags := codescan.InferCodeItems(bundle, options.CodeRoot)
	inferredVerification, verDiags := codescan.InferVerificationChecks(codescan.VerificationScan{Bundle: bundle, Requirements: bundle.Requirements, InferredCode: inferredCode, CodeRoot: options.CodeRoot})
	compliance, complianceDiags := resolveOSCALCompliance(bundle, oscalResolveOptions{ProfileHref: options.ProfileHref, CatalogHref: options.CatalogHref})
	diags := append(append(append([]validate.Diagnostic{}, codeDiags...), verDiags...), complianceDiags...)
	return arEvidence{Verification: inferredVerification, Compliance: compliance, Diagnostics: diags}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildAssessmentResults(bundle model.Bundle, evidence arEvidence, header arHeader) oscalAssessmentResults {
	modelID := bundle.Architecture.Model.ID
	return oscalAssessmentResults{
		UUID: textutil.DeterministicUUID("assessment-results|" + modelID),
		Metadata: oscalMetadata{
			Title:        textutil.NonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(modelID)) + " Assessment Results",
			LastModified: header.Now,
			Version:      "0.1.0",
			OSCALVersion: "1.1.2",
		},
		ImportAP: oscalImportAP{Href: header.AssessmentPlanHref},
		Results: []oscalARResult{{
			UUID:        textutil.DeterministicUUID("result|" + modelID),
			Title:       "Automated architecture assessment",
			Description: "Assessment results generated from verification evidence, authored risks, and compliance mappings.",
			Start:       header.Now,
			End:         header.Now,
			ReviewedControls: oscalReviewedControls{ControlSelections: []oscalControlSelection{{
				Description:     "Controls reviewed through architecture-derived compliance mappings.",
				IncludeControls: includeControlRefs(oscalMappedControlIDs(evidence.Compliance)),
			}}},
			Findings: arFindings(modelID, evidence.Verification),
			Risks:    arRisks(bundle),
		}},
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func includeControlRefs(controlIDs []string) []oscalIncludeControlRef {
	refs := make([]oscalIncludeControlRef, 0, len(controlIDs))
	for _, cid := range controlIDs {
		refs = append(refs, oscalIncludeControlRef{ControlID: cid})
	}
	return refs
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func arFindings(modelID string, checks []codescan.VerificationCheck) []oscalFinding {
	findings := []oscalFinding{}
	for _, v := range checks {
		if strings.ToLower(strings.TrimSpace(v.Status)) == "pass" {
			continue
		}
		findings = append(findings, arFinding(modelID, v))
	}
	sort.SliceStable(findings, func(i, j int) bool { return findings[i].UUID < findings[j].UUID })
	return findings
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func arFinding(modelID string, v codescan.VerificationCheck) oscalFinding {
	status := strings.ToLower(strings.TrimSpace(v.Status))
	target := "verification"
	if len(v.Verifies) > 0 {
		target = strings.TrimSpace(v.Verifies[0])
	}
	return oscalFinding{
		UUID:        textutil.DeterministicUUID("finding|" + modelID + "|" + v.ID),
		Title:       textutil.NonEmpty(strings.TrimSpace(v.Name), strings.TrimSpace(v.ID)),
		Description: textutil.NonEmpty(strings.TrimSpace(v.Description), fmt.Sprintf("Verification status %s.", textutil.NonEmpty(status, "unknown"))),
		Target:      oscalFindingTarget{Type: "objective-id", TargetID: target, Status: oscalOperationalStatus{State: "not-satisfied"}},
		Props: []oscalProperty{
			engineeringModelOSCALProperty("verification-id", strings.TrimSpace(v.ID)),
			engineeringModelOSCALProperty("verification-status", textutil.NonEmpty(status, "unknown")),
		},
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func arRisks(bundle model.Bundle) []oscalARRisk {
	risks := []oscalARRisk{}
	for _, r := range bundle.Architecture.AuthoredArchitecture.Risks {
		state := strings.TrimSpace(strings.ToLower(r.Status))
		if state == "" {
			state = "open"
		}
		risks = append(risks, oscalARRisk{
			UUID:        textutil.DeterministicUUID("ar-risk|" + bundle.Architecture.Model.ID + "|" + strings.TrimSpace(r.ID)),
			Title:       textutil.NonEmpty(strings.TrimSpace(r.Title), strings.TrimSpace(r.ID)),
			Description: textutil.NonEmpty(strings.TrimSpace(r.Statement), "Authored risk statement."),
			Status:      state,
			Statement:   textutil.NonEmpty(strings.TrimSpace(r.Rationale), strings.TrimSpace(r.Statement)),
			Props:       []oscalProperty{},
		})
	}
	sort.SliceStable(risks, func(i, j int) bool { return risks[i].UUID < risks[j].UUID })
	return risks
}
