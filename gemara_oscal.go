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
	"fmt"
	"sort"
	"strings"

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
	bundle, err = enrichBundleFromComposition(bundle, "architecture", "assurance", "compliance")
	if err != nil {
		return "", err
	}
	return GenerateGemaraOSCALCatalog(bundle, options)
}

// GenerateGemaraOSCALCatalog converts the Gemara Control Catalog to an OSCAL
// Catalog JSON document via the go-gemara SDK. Returns "" when there are no controls.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALCatalog(bundle model.Bundle, options GemaraExportOptions) (string, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return "", err
	}
	bundle = canonical.Documents()
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
	timestamp, err := resolveGeneratedTimestamp(bundle, options.Date, "Gemara OSCAL")
	if err != nil {
		return "", err
	}
	out, err := marshalDeterministicGemaraOSCAL(
		oscalTypes.OscalModels{Catalog: &oscalCatalog},
		"catalog|"+bundle.Architecture.Model.ID,
		timestamp,
	)
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
		bundle.Requirements = requirements
	}
	bundle, err = enrichBundleFromComposition(bundle, "architecture", "assurance", "compliance", "requirements")
	if err != nil {
		return "", err
	}
	requirements = bundle.Requirements
	return GenerateGemaraOSCALAssessmentResults(bundle, requirements, codeRoot, options)
}

// GenerateGemaraOSCALAssessmentResults converts the Gemara Evaluation Log to OSCAL
// Assessment Results JSON via the go-gemara SDK. Returns "" when there is nothing
// to evaluate.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraOSCALAssessmentResults(bundle model.Bundle, requirements model.RequirementsDocument, codeRoot string, options GemaraExportOptions) (string, error) {
	bundle.Requirements = requirements
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return "", err
	}
	bundle = canonical.Documents()
	requirements = bundle.Requirements
	evalRes, err := GenerateGemaraEvaluationLog(bundle, requirements, codeRoot, options)
	if err != nil {
		return "", err
	}
	if !evalRes.HasContent {
		return "", nil
	}
	assessmentPlanHref := options.AssessmentPlanHref
	if assessmentPlanHref == "" {
		assessmentPlanHref = "#"
	}
	ar, err := gemaraconv.EvaluationLogToOSCALAssessmentResults(evalRes.EvaluationLog, gemaraconv.WithImportApHref(assessmentPlanHref))
	if err != nil {
		return "", err
	}
	normalizeGemaraOSCALAssessmentResults(&ar)
	timestamp, err := resolveGeneratedTimestamp(bundle, options.Date, "Gemara OSCAL")
	if err != nil {
		return "", err
	}
	out, err := marshalDeterministicGemaraOSCAL(
		oscalTypes.OscalModels{AssessmentResults: &ar},
		"assessment-results|"+bundle.Architecture.Model.ID,
		timestamp,
	)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALAssessmentResults(ar *oscalTypes.AssessmentResults) {
	if ar == nil {
		return
	}
	if ar.BackMatter != nil && ar.BackMatter.Resources != nil {
		for resourceIndex := range *ar.BackMatter.Resources {
			resource := &(*ar.BackMatter.Resources)[resourceIndex]
			resource.Props = nil
			if resource.Rlinks != nil {
				links := (*resource.Rlinks)[:0]
				for _, link := range *resource.Rlinks {
					if strings.TrimSpace(link.Href) != "" {
						links = append(links, link)
					}
				}
				if len(links) == 0 {
					resource.Rlinks = nil
				} else {
					resource.Rlinks = &links
				}
			}
		}
	}
	for resultIndex := range ar.Results {
		result := &ar.Results[resultIndex]
		if result.Findings != nil {
			for findingIndex := range *result.Findings {
				finding := &(*result.Findings)[findingIndex]
				normalizeGemaraOSCALOrigins(finding.Origins)
				finding.Target.Status.Reason = normalizeOSCALToken(finding.Target.Status.Reason)
			}
		}
		if result.Observations != nil {
			for observationIndex := range *result.Observations {
				normalizeGemaraOSCALOrigins((*result.Observations)[observationIndex].Origins)
			}
		}
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALOrigins(origins *[]oscalTypes.Origin) {
	if origins == nil {
		return
	}
	for originIndex := range *origins {
		for actorIndex := range (*origins)[originIndex].Actors {
			actor := &(*origins)[originIndex].Actors[actorIndex]
			switch actor.Type {
			case "assessment-platform", "party", "tool":
			default:
				actor.Type = "party"
			}
		}
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeOSCALToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var token strings.Builder
	previousDash := false
	for _, r := range value {
		valid := r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-'
		if valid {
			token.WriteRune(r)
			previousDash = r == '-'
		} else if token.Len() > 0 && !previousDash {
			token.WriteByte('-')
			previousDash = true
		}
	}
	normalized := strings.Trim(token.String(), "-")
	if normalized != "" && normalized[0] >= '0' && normalized[0] <= '9' {
		return "id-" + normalized
	}
	return normalized
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func marshalDeterministicGemaraOSCAL(document any, seed, timestamp string) ([]byte, error) {
	data, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	replacements := map[string]string{}
	collectGemaraOSCALUUIDs(root, "$", seed, replacements)
	normalizeGemaraOSCALValues(root, replacements, timestamp)
	normalizeGemaraOSCALPartyReferences(root, firstGemaraOSCALPartyUUID(root))
	return json.MarshalIndent(root, "", "  ")
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func collectGemaraOSCALUUIDs(value any, path, seed string, replacements map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			childPath := path + "." + key
			if key == "uuid" {
				if old, ok := typed[key].(string); ok && old != "" {
					replacements[old] = deterministicUUID("gemara-oscal|" + seed + "|" + path)
				}
				continue
			}
			collectGemaraOSCALUUIDs(typed[key], childPath, seed, replacements)
		}
	case []any:
		for index, child := range typed {
			collectGemaraOSCALUUIDs(child, fmt.Sprintf("%s[%d]", path, index), seed, replacements)
		}
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALValues(value any, replacements map[string]string, timestamp string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "last-modified" {
				typed[key] = timestamp
				continue
			}
			if text, ok := child.(string); ok {
				if replacement, exists := replacements[text]; exists {
					typed[key] = replacement
				} else if strings.HasPrefix(text, "#") {
					if replacement, exists := replacements[strings.TrimPrefix(text, "#")]; exists {
						typed[key] = "#" + replacement
					}
				} else if key == "id" || key == "control-id" || key == "target-id" {
					typed[key] = normalizeOSCALToken(text)
				}
				continue
			}
			normalizeGemaraOSCALValues(child, replacements, timestamp)
		}
	case []any:
		for index, child := range typed {
			if text, ok := child.(string); ok {
				if replacement, exists := replacements[text]; exists {
					typed[index] = replacement
				}
				continue
			}
			normalizeGemaraOSCALValues(child, replacements, timestamp)
		}
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func firstGemaraOSCALPartyUUID(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if parties, ok := typed["parties"].([]any); ok && len(parties) > 0 {
			if party, ok := parties[0].(map[string]any); ok {
				if id, ok := party["uuid"].(string); ok {
					return id
				}
			}
		}
		for _, child := range typed {
			if id := firstGemaraOSCALPartyUUID(child); id != "" {
				return id
			}
		}
	case []any:
		for _, child := range typed {
			if id := firstGemaraOSCALPartyUUID(child); id != "" {
				return id
			}
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALPartyReferences(value any, partyUUID string) {
	if partyUUID == "" {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			switch key {
			case "actor-uuid", "party-uuid":
				typed[key] = partyUUID
			case "party-uuids":
				if ids, ok := child.([]any); ok {
					for index := range ids {
						ids[index] = partyUUID
					}
				}
			default:
				normalizeGemaraOSCALPartyReferences(child, partyUUID)
			}
		}
	case []any:
		for _, child := range typed {
			normalizeGemaraOSCALPartyReferences(child, partyUUID)
		}
	}
}
