package engmodel

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

type OSCALAROptions struct {
	AssessmentPlanHref string
	RequirementsPath   string
	CodeRoot           string
}

type OSCALARResult struct {
	JSON        string
	Document    OSCALARDocument
	Diagnostics []validate.Diagnostic
}

type OSCALARDocument struct {
	AssessmentResults oscalAssessmentResults `json:"assessment-results"`
}

type oscalAssessmentResults struct {
	UUID     string          `json:"uuid"`
	Metadata oscalMetadata   `json:"metadata"`
	ImportAP oscalImportAP   `json:"import-ap"`
	Results  []oscalARResult `json:"results"`
}

type oscalImportAP struct {
	Href string `json:"href"`
}

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

type oscalReviewedControls struct {
	ControlSelections []oscalControlSelection `json:"control-selections"`
}

type oscalControlSelection struct {
	Description     string                   `json:"description,omitempty"`
	IncludeControls []oscalIncludeControlRef `json:"include-controls,omitempty"`
}

type oscalIncludeControlRef struct {
	ControlID string `json:"control-id"`
}

type oscalFinding struct {
	UUID        string             `json:"uuid"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Target      oscalFindingTarget `json:"target"`
	Props       []oscalProperty    `json:"props,omitempty"`
}

type oscalFindingTarget struct {
	Type     string                 `json:"type"`
	TargetID string                 `json:"target-id"`
	Status   oscalOperationalStatus `json:"status"`
}

type oscalARRisk struct {
	UUID        string          `json:"uuid"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Statement   string          `json:"statement"`
	Props       []oscalProperty `json:"props,omitempty"`
}

func GenerateOSCALAssessmentResultsFromFile(architecturePath string, options OSCALAROptions) (OSCALARResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return OSCALARResult{}, err
	}
	var req model.RequirementsDocument
	if strings.TrimSpace(options.RequirementsPath) != "" {
		req, err = model.LoadRequirements(options.RequirementsPath)
		if err != nil {
			return OSCALARResult{}, err
		}
	}
	if strings.TrimSpace(options.CodeRoot) != "" && !filepath.IsAbs(options.CodeRoot) {
		baseDir := filepath.Dir(architecturePath)
		options.CodeRoot = filepath.Join(baseDir, options.CodeRoot)
	}
	return GenerateOSCALAssessmentResults(bundle, req, options)
}

func GenerateOSCALAssessmentResults(bundle model.Bundle, requirements model.RequirementsDocument, options OSCALAROptions) (OSCALARResult, error) {
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return OSCALARResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}

	apHref := strings.TrimSpace(options.AssessmentPlanHref)
	if apHref == "" {
		apHref = "./ASSESSMENT-PLAN.json"
	}

	inferredCode, codeDiags := inferCodeItems(bundle, options.CodeRoot)
	diags = append(diags, codeDiags...)
	inferredVerification, verDiags := inferVerificationChecks(bundle, requirements, inferredCode, options.CodeRoot)
	diags = append(diags, verDiags...)

	now := time.Now().UTC().Format(time.RFC3339)
	controlSet := map[string]bool{}
	for _, a := range bundle.Architecture.AuthoredArchitecture.ControlAllocations {
		for _, cid := range a.OSCALControlIDs {
			if c := normalizeOSCALControlID(cid); c != "" {
				controlSet[c] = true
			}
		}
	}
	controlIDs := make([]string, 0, len(controlSet))
	for cid := range controlSet {
		controlIDs = append(controlIDs, cid)
	}
	sort.Strings(controlIDs)
	controlRefs := make([]oscalIncludeControlRef, 0, len(controlIDs))
	for _, cid := range controlIDs {
		controlRefs = append(controlRefs, oscalIncludeControlRef{ControlID: cid})
	}

	findings := []oscalFinding{}
	for _, v := range inferredVerification {
		status := strings.ToLower(strings.TrimSpace(v.Status))
		if status == "pass" {
			continue
		}
		target := "verification"
		if len(v.Verifies) > 0 {
			target = strings.TrimSpace(v.Verifies[0])
		}
		findings = append(findings, oscalFinding{
			UUID:        deterministicUUID("finding|" + bundle.Architecture.Model.ID + "|" + v.ID),
			Title:       nonEmpty(strings.TrimSpace(v.Name), strings.TrimSpace(v.ID)),
			Description: nonEmpty(strings.TrimSpace(v.Description), fmt.Sprintf("Verification status %s.", nonEmpty(status, "unknown"))),
			Target:      oscalFindingTarget{Type: "objective-id", TargetID: target, Status: oscalOperationalStatus{State: "not-satisfied"}},
			Props:       []oscalProperty{{Name: "verification-id", Value: strings.TrimSpace(v.ID)}, {Name: "verification-status", Value: nonEmpty(status, "unknown")}},
		})
	}
	sort.SliceStable(findings, func(i, j int) bool { return findings[i].UUID < findings[j].UUID })

	risks := []oscalARRisk{}
	for _, r := range bundle.Architecture.AuthoredArchitecture.Risks {
		state := strings.TrimSpace(strings.ToLower(r.Status))
		if state == "" {
			state = "open"
		}
		props := []oscalProperty{}
		risks = append(risks, oscalARRisk{
			UUID:        deterministicUUID("ar-risk|" + bundle.Architecture.Model.ID + "|" + strings.TrimSpace(r.ID)),
			Title:       nonEmpty(strings.TrimSpace(r.Title), strings.TrimSpace(r.ID)),
			Description: nonEmpty(strings.TrimSpace(r.Statement), "Authored risk statement."),
			Status:      state,
			Statement:   nonEmpty(strings.TrimSpace(r.Rationale), strings.TrimSpace(r.Statement)),
			Props:       props,
		})
	}
	sort.SliceStable(risks, func(i, j int) bool { return risks[i].UUID < risks[j].UUID })

	doc := OSCALARDocument{AssessmentResults: oscalAssessmentResults{
		UUID: deterministicUUID("assessment-results|" + bundle.Architecture.Model.ID),
		Metadata: oscalMetadata{
			Title:        nonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID)) + " Assessment Results",
			LastModified: now,
			Version:      "0.1.0",
			OSCALVersion: "1.1.2",
		},
		ImportAP: oscalImportAP{Href: apHref},
		Results: []oscalARResult{{
			UUID:        deterministicUUID("result|" + bundle.Architecture.Model.ID),
			Title:       "Automated architecture assessment",
			Description: "Assessment results generated from verification evidence, authored risks, and control allocations.",
			Start:       now,
			End:         now,
			ReviewedControls: oscalReviewedControls{ControlSelections: []oscalControlSelection{{
				Description:     "Controls reviewed through architecture-derived allocations.",
				IncludeControls: controlRefs,
			}}},
			Findings: findings,
			Risks:    risks,
		}},
	}}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return OSCALARResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	return OSCALARResult{JSON: string(b), Document: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}
