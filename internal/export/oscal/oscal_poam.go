// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package oscal

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type POAMOptions struct {
	SSPHref      string
	LastModified string
}

var ErrNoPOAMItems = errors.New("OSCAL POA&M requires at least one authored POA&M item")

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type POAMResult struct {
	JSON        string
	Document    POAMDocument
	Diagnostics []validate.Diagnostic
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type POAMDocument struct {
	PlanOfActionAndMilestones oscalPOAMRoot `json:"plan-of-action-and-milestones"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER
type oscalPOAMRoot struct {
	UUID      string          `json:"uuid"`
	Metadata  oscalMetadata   `json:"metadata"`
	ImportSSP oscalImportSSP  `json:"import-ssp"`
	POAMItems []oscalPOAMItem `json:"poam-items"`
	Risks     []oscalPOAMRisk `json:"risks,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalImportSSP struct {
	Href string `json:"href"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER
type oscalPOAMItem struct {
	UUID         string                `json:"uuid"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Props        []oscalProperty       `json:"props,omitempty"`
	RelatedRisks []oscalRelatedRiskRef `json:"related-risks,omitempty"`
	Remarks      string                `json:"remarks,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER
type oscalRelatedRiskRef struct {
	RiskUUID string `json:"risk-uuid"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER
type oscalPOAMRisk struct {
	UUID        string          `json:"uuid"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Statement   string          `json:"statement"`
	Props       []oscalProperty `json:"props,omitempty"`
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GeneratePOAMFromFile(architecturePath string, options POAMOptions) (POAMResult, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return POAMResult{}, err
	}
	bundle, err = composition.EnrichBundleFromComposition(bundle, "architecture", "assurance", "compliance")
	if err != nil {
		return POAMResult{}, err
	}
	return GeneratePOAM(bundle, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-THREAT-EXPORTER, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GeneratePOAM(bundle model.Bundle, options POAMOptions) (POAMResult, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return POAMResult{}, err
	}
	bundle = canonical.Documents()
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return POAMResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}
	risks, riskUUIDByID := poamRisks(bundle)
	items := poamItems(bundle, riskUUIDByID)
	if len(items) == 0 {
		return POAMResult{Diagnostics: validate.SortDiagnostics(diags)}, ErrNoPOAMItems
	}

	now, err := resolveOSCALTimestamp(bundle, options.LastModified)
	if err != nil {
		return POAMResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	doc := POAMDocument{PlanOfActionAndMilestones: buildPOAMRoot(bundle, poamHeader{Now: now, SSPHref: poamSSPHref(options), Items: items, Risks: risks})}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return POAMResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}
	return POAMResult{JSON: string(b), Document: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func poamSSPHref(options POAMOptions) string {
	sspHref := strings.TrimSpace(options.SSPHref)
	if sspHref == "" {
		sspHref = "./ARCHITECTURE.ssp.json"
	}
	return sspHref
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type poamHeader struct {
	Now     string
	SSPHref string
	Items   []oscalPOAMItem
	Risks   []oscalPOAMRisk
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildPOAMRoot(bundle model.Bundle, header poamHeader) oscalPOAMRoot {
	return oscalPOAMRoot{
		UUID: textutil.DeterministicUUID("poam|" + bundle.Architecture.Model.ID),
		Metadata: oscalMetadata{
			Title:        textutil.NonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID)) + " POA&M",
			LastModified: header.Now,
			Version:      "0.1.0",
			OSCALVersion: "1.1.2",
		},
		ImportSSP: oscalImportSSP{Href: header.SSPHref},
		POAMItems: header.Items,
		Risks:     header.Risks,
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func poamRisks(bundle model.Bundle) ([]oscalPOAMRisk, map[string]string) {
	riskUUIDByID := map[string]string{}
	risks := []oscalPOAMRisk{}
	for _, r := range bundle.Architecture.AuthoredArchitecture.Risks {
		riskUUID := textutil.DeterministicUUID("poam-risk|" + bundle.Architecture.Model.ID + "|" + strings.TrimSpace(r.ID))
		riskUUIDByID[strings.TrimSpace(r.ID)] = riskUUID
		state := strings.ToLower(strings.TrimSpace(r.Status))
		if state == "" {
			state = "open"
		}
		risks = append(risks, oscalPOAMRisk{
			UUID:        riskUUID,
			Title:       textutil.NonEmpty(strings.TrimSpace(r.Title), strings.TrimSpace(r.ID)),
			Description: textutil.NonEmpty(strings.TrimSpace(r.Statement), "Authored risk statement."),
			Status:      state,
			Statement:   textutil.NonEmpty(strings.TrimSpace(r.Rationale), strings.TrimSpace(r.Statement)),
		})
	}
	sort.SliceStable(risks, func(i, j int) bool { return risks[i].UUID < risks[j].UUID })
	return risks, riskUUIDByID
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func poamItems(bundle model.Bundle, riskUUIDByID map[string]string) []oscalPOAMItem {
	items := []oscalPOAMItem{}
	for _, p := range bundle.Architecture.AuthoredArchitecture.POAMItems {
		related := []oscalRelatedRiskRef{}
		if riskUUID := riskUUIDByID[strings.TrimSpace(p.RiskRef)]; riskUUID != "" {
			related = append(related, oscalRelatedRiskRef{RiskUUID: riskUUID})
		}
		items = append(items, oscalPOAMItem{
			UUID:         textutil.DeterministicUUID("poam-item|" + bundle.Architecture.Model.ID + "|" + strings.TrimSpace(p.ID)),
			Title:        textutil.NonEmpty(strings.TrimSpace(p.Milestone), strings.TrimSpace(p.ID)),
			Description:  textutil.NonEmpty(strings.TrimSpace(p.Milestone), "POA&M action item."),
			Props:        poamItemProps(p),
			RelatedRisks: related,
			Remarks:      poamArtifactsRemark(p),
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].UUID < items[j].UUID })
	return items
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func poamItemProps(p model.POAMItem) []oscalProperty {
	props := []oscalProperty{engineeringModelOSCALProperty("model-poam-id", strings.TrimSpace(p.ID))}
	for _, f := range []struct{ name, value string }{
		{"due-date", p.DueDate},
		{"status", p.Status},
		{"responsible-role", p.ResponsibleRole},
	} {
		if v := strings.TrimSpace(f.value); v != "" {
			props = append(props, engineeringModelOSCALProperty(f.name, v))
		}
	}
	return props
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func poamArtifactsRemark(p model.POAMItem) string {
	artifacts := []string{}
	for _, a := range p.Artifacts {
		if x := strings.TrimSpace(a.Path); x != "" {
			artifacts = append(artifacts, x)
		}
	}
	if len(artifacts) == 0 {
		return ""
	}
	return "Artifacts: " + strings.Join(artifacts, ", ")
}
