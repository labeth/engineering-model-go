package engmodel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

type OSCALPOAMOptions struct {
	SSPHref string
}

type OSCALPOAMResult struct {
	JSON        string
	Document    OSCALPOAMDocument
	Diagnostics []validate.Diagnostic
}

type OSCALPOAMDocument struct {
	PlanOfActionAndMilestones oscalPOAMRoot `json:"plan-of-action-and-milestones"`
}

type oscalPOAMRoot struct {
	UUID      string          `json:"uuid"`
	Metadata  oscalMetadata   `json:"metadata"`
	ImportSSP oscalImportSSP  `json:"import-ssp"`
	POAMItems []oscalPOAMItem `json:"poam-items"`
	Risks     []oscalPOAMRisk `json:"risks,omitempty"`
}

type oscalImportSSP struct {
	Href string `json:"href"`
}

type oscalPOAMItem struct {
	UUID         string                `json:"uuid"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Props        []oscalProperty       `json:"props,omitempty"`
	RelatedRisks []oscalRelatedRiskRef `json:"related-risks,omitempty"`
	Remarks      string                `json:"remarks,omitempty"`
}

type oscalRelatedRiskRef struct {
	RiskUUID string `json:"risk-uuid"`
}

type oscalPOAMRisk struct {
	UUID        string          `json:"uuid"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Statement   string          `json:"statement"`
	Props       []oscalProperty `json:"props,omitempty"`
}

func GenerateOSCALPOAMFromFile(architecturePath string, options OSCALPOAMOptions) (OSCALPOAMResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return OSCALPOAMResult{}, err
	}
	return GenerateOSCALPOAM(bundle, options)
}

func GenerateOSCALPOAM(bundle model.Bundle, options OSCALPOAMOptions) (OSCALPOAMResult, error) {
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return OSCALPOAMResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}
	sspHref := strings.TrimSpace(options.SSPHref)
	if sspHref == "" {
		sspHref = "./ARCHITECTURE.ssp.json"
	}
	riskUUIDByID := map[string]string{}
	risks := []oscalPOAMRisk{}
	for _, r := range bundle.Architecture.AuthoredArchitecture.Risks {
		riskUUID := deterministicUUID("poam-risk|" + bundle.Architecture.Model.ID + "|" + strings.TrimSpace(r.ID))
		riskUUIDByID[strings.TrimSpace(r.ID)] = riskUUID
		state := strings.ToLower(strings.TrimSpace(r.Status))
		if state == "" {
			state = "open"
		}
		props := []oscalProperty{}
		risks = append(risks, oscalPOAMRisk{
			UUID:        riskUUID,
			Title:       nonEmpty(strings.TrimSpace(r.Title), strings.TrimSpace(r.ID)),
			Description: nonEmpty(strings.TrimSpace(r.Statement), "Authored risk statement."),
			Status:      state,
			Statement:   nonEmpty(strings.TrimSpace(r.Rationale), strings.TrimSpace(r.Statement)),
			Props:       props,
		})
	}
	sort.SliceStable(risks, func(i, j int) bool { return risks[i].UUID < risks[j].UUID })

	items := []oscalPOAMItem{}
	for _, p := range bundle.Architecture.AuthoredArchitecture.POAMItems {
		riskID := strings.TrimSpace(p.RiskRef)
		related := []oscalRelatedRiskRef{}
		if riskUUID := riskUUIDByID[riskID]; riskUUID != "" {
			related = append(related, oscalRelatedRiskRef{RiskUUID: riskUUID})
		}
		props := []oscalProperty{}
		if due := strings.TrimSpace(p.DueDate); due != "" {
			props = append(props, oscalProperty{Name: "due-date", Value: due})
		}
		if st := strings.TrimSpace(p.Status); st != "" {
			props = append(props, oscalProperty{Name: "status", Value: st})
		}
		if role := strings.TrimSpace(p.ResponsibleRole); role != "" {
			props = append(props, oscalProperty{Name: "responsible-role", Value: role})
		}
		artifacts := []string{}
		for _, a := range p.Artifacts {
			if x := strings.TrimSpace(a.Path); x != "" {
				artifacts = append(artifacts, x)
			}
		}
		remarks := ""
		if len(artifacts) > 0 {
			remarks = "Artifacts: " + strings.Join(artifacts, ", ")
		}
		items = append(items, oscalPOAMItem{
			UUID:         deterministicUUID("poam-item|" + bundle.Architecture.Model.ID + "|" + strings.TrimSpace(p.ID)),
			Title:        nonEmpty(strings.TrimSpace(p.Milestone), strings.TrimSpace(p.ID)),
			Description:  nonEmpty(strings.TrimSpace(p.Milestone), "POA&M action item."),
			Props:        props,
			RelatedRisks: related,
			Remarks:      remarks,
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].UUID < items[j].UUID })

	now := time.Now().UTC().Format(time.RFC3339)
	doc := OSCALPOAMDocument{PlanOfActionAndMilestones: oscalPOAMRoot{
		UUID: deterministicUUID("poam|" + bundle.Architecture.Model.ID),
		Metadata: oscalMetadata{
			Title:        nonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID)) + " POA&M",
			LastModified: now,
			Version:      "0.1.0",
			OSCALVersion: "1.1.2",
		},
		ImportSSP: oscalImportSSP{Href: sspHref},
		POAMItems: items,
		Risks:     risks,
	}}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return OSCALPOAMResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	return OSCALPOAMResult{JSON: string(b), Document: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}
