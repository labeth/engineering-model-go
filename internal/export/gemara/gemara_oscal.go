// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package gemara

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
	gemaraproj "github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/gemaraconv"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/export/exportutil"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
)

// gemaraOSCALControlHrefFormat is the URL template linking OSCAL controls back to
// their Gemara source (format: href(version, controlID)).
const gemaraOSCALControlHrefFormat = "https://gemara.local/controls/%s#%s"

// GenerateOSCALCatalogFromFile loads the model and emits an OSCAL Catalog
// derived from the Gemara Control Catalog.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALCatalogFromFile(architecturePath string, options Options) (string, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return "", err
	}
	bundle, err = composition.EnrichBundleFromComposition(bundle, "architecture", "assurance", "compliance")
	if err != nil {
		return "", err
	}
	return GenerateOSCALCatalog(bundle, options)
}

// GenerateOSCALCatalog converts the Gemara Control Catalog to an OSCAL
// Catalog JSON document via the go-gemara SDK. Returns "" when there are no controls.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALCatalog(bundle model.Bundle, options Options) (string, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return "", err
	}
	bundle = canonical.Documents()
	res, err := Generate(bundle, options)
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
	timestamp, err := exportutil.ResolveGeneratedTimestamp(bundle, options.Date, "Gemara OSCAL")
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

// GenerateOSCALAssessmentResultsFromFiles loads inputs and emits OSCAL
// Assessment Results derived from the Gemara evaluation log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALAssessmentResultsFromFiles(in FileInputs, options Options) (string, error) {
	bundle, err := loadEvaluationBundle(in)
	if err != nil {
		return "", err
	}
	return GenerateOSCALAssessmentResults(bundle, bundle.Requirements, EvalRun{CodeRoot: in.CodeRoot, Options: options})
}

// GenerateOSCALAssessmentResults converts the Gemara Evaluation Log to OSCAL
// Assessment Results JSON via the go-gemara SDK. Returns "" when there is nothing
// to evaluate.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALAssessmentResults(bundle model.Bundle, requirements model.RequirementsDocument, run EvalRun) (string, error) {
	bundle.Requirements = requirements
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return "", err
	}
	bundle = canonical.Documents()
	evalRes, err := GenerateEvaluationLog(bundle, bundle.Requirements, run)
	if err != nil {
		return "", err
	}
	if !evalRes.HasContent {
		return "", nil
	}
	return renderOSCALAssessmentResults(bundle, evalRes.EvaluationLog, run.Options)
}

// renderOSCALAssessmentResults converts an evaluation log to deterministic OSCAL
// Assessment Results JSON.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func renderOSCALAssessmentResults(bundle model.Bundle, evalLog gemaraproj.EvaluationLog, options Options) (string, error) {
	assessmentPlanHref := options.AssessmentPlanHref
	if assessmentPlanHref == "" {
		assessmentPlanHref = "#"
	}
	ar, err := gemaraconv.EvaluationLogToOSCALAssessmentResults(evalLog, gemaraconv.WithImportApHref(assessmentPlanHref))
	if err != nil {
		return "", err
	}
	normalizeGemaraOSCALAssessmentResults(&ar)
	timestamp, err := exportutil.ResolveGeneratedTimestamp(bundle, options.Date, "Gemara OSCAL")
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
	normalizeGemaraOSCALBackMatter(ar.BackMatter)
	for resultIndex := range ar.Results {
		normalizeGemaraOSCALResult(&ar.Results[resultIndex])
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALBackMatter(backMatter *oscalTypes.BackMatter) {
	if backMatter == nil || backMatter.Resources == nil {
		return
	}
	for resourceIndex := range *backMatter.Resources {
		normalizeGemaraOSCALResource(&(*backMatter.Resources)[resourceIndex])
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALResource(resource *oscalTypes.Resource) {
	resource.Props = nil
	if resource.Rlinks == nil {
		return
	}
	links := (*resource.Rlinks)[:0]
	for _, link := range *resource.Rlinks {
		if strings.TrimSpace(link.Href) != "" {
			links = append(links, link)
		}
	}
	if len(links) == 0 {
		resource.Rlinks = nil
		return
	}
	resource.Rlinks = &links
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALResult(result *oscalTypes.Result) {
	normalizeGemaraOSCALFindings(result.Findings)
	normalizeGemaraOSCALObservations(result.Observations)
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALFindings(findings *[]oscalTypes.Finding) {
	if findings == nil {
		return
	}
	for findingIndex := range *findings {
		finding := &(*findings)[findingIndex]
		normalizeGemaraOSCALOrigins(finding.Origins)
		finding.Target.Status.Reason = normalizeOSCALToken(finding.Target.Status.Reason)
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALObservations(observations *[]oscalTypes.Observation) {
	if observations == nil {
		return
	}
	for observationIndex := range *observations {
		normalizeGemaraOSCALOrigins((*observations)[observationIndex].Origins)
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALOrigins(origins *[]oscalTypes.Origin) {
	if origins == nil {
		return
	}
	for originIndex := range *origins {
		origin := &(*origins)[originIndex]
		for actorIndex := range origin.Actors {
			normalizeGemaraOSCALActorType(&origin.Actors[actorIndex])
		}
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeGemaraOSCALActorType(actor *oscalTypes.OriginActor) {
	switch actor.Type {
	case "assessment-platform", "party", "tool":
	default:
		actor.Type = "party"
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func normalizeOSCALToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var token strings.Builder
	previousDash := false
	for _, r := range value {
		if isOSCALTokenRune(r) {
			token.WriteRune(r)
			previousDash = r == '-'
		} else if token.Len() > 0 && !previousDash {
			token.WriteByte('-')
			previousDash = true
		}
	}
	normalized := strings.Trim(token.String(), "-")
	if startsWithDigit(normalized) {
		return "id-" + normalized
	}
	return normalized
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func isOSCALTokenRune(r rune) bool {
	return isASCIILower(r) || isASCIIDigit(r) || strings.ContainsRune("._-", r)
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-015
func startsWithDigit(text string) bool {
	return text != "" && isASCIIDigit(rune(text[0]))
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

	collector := oscalUUIDCollector{seed: seed, replacements: map[string]string{}}
	collector.collect(root, "$")
	oscalValueNormalizer{replacements: collector.replacements, timestamp: timestamp}.normalize(root)
	oscalPartyRewriter{partyUUID: firstGemaraOSCALPartyUUID(root)}.rewrite(root)
	return json.MarshalIndent(root, "", "  ")
}

// oscalUUIDCollector maps every uuid of a document to a deterministic one.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
type oscalUUIDCollector struct {
	seed         string
	replacements map[string]string
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func (c oscalUUIDCollector) collect(value any, path string) {
	switch typed := value.(type) {
	case map[string]any:
		c.collectMap(typed, path)
	case []any:
		for index, child := range typed {
			c.collect(child, fmt.Sprintf("%s[%d]", path, index))
		}
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func (c oscalUUIDCollector) collectMap(typed map[string]any, path string) {
	keys := make([]string, 0, len(typed))
	for key := range typed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key != "uuid" {
			c.collect(typed[key], path+"."+key)
			continue
		}
		if old, ok := typed[key].(string); ok && old != "" {
			c.replacements[old] = textutil.DeterministicUUID("gemara-oscal|" + c.seed + "|" + path)
		}
	}
}

// oscalIDKeys are the keys whose string values are normalized to OSCAL tokens.
var oscalIDKeys = map[string]bool{"id": true, "control-id": true, "target-id": true}

// oscalValueNormalizer replaces uuids and stamps last-modified.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
type oscalValueNormalizer struct {
	replacements map[string]string
	timestamp    string
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func (n oscalValueNormalizer) normalize(value any) {
	switch typed := value.(type) {
	case map[string]any:
		n.normalizeMap(typed)
	case []any:
		n.normalizeSlice(typed)
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func (n oscalValueNormalizer) normalizeMap(typed map[string]any) {
	for key, child := range typed {
		if key == "last-modified" {
			typed[key] = n.timestamp
			continue
		}
		if text, ok := child.(string); ok {
			typed[key] = n.normalizeText(key, text)
			continue
		}
		n.normalize(child)
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func (n oscalValueNormalizer) normalizeSlice(typed []any) {
	for index, child := range typed {
		text, ok := child.(string)
		if !ok {
			n.normalize(child)
			continue
		}
		if replacement, exists := n.replacements[text]; exists {
			typed[index] = replacement
		}
	}
}

// normalizeText is the replacement of one string value, the value itself when
// nothing applies.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func (n oscalValueNormalizer) normalizeText(key, text string) string {
	if replacement, exists := n.replacements[text]; exists {
		return replacement
	}
	if strings.HasPrefix(text, "#") {
		if replacement, exists := n.replacements[strings.TrimPrefix(text, "#")]; exists {
			return "#" + replacement
		}
		return text
	}
	if oscalIDKeys[key] {
		return normalizeOSCALToken(text)
	}
	return text
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func firstGemaraOSCALPartyUUID(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if id, ok := directPartyUUID(typed); ok {
			return id
		}
		return firstPartyUUIDInMap(typed)
	case []any:
		return firstPartyUUIDInSlice(typed)
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func firstPartyUUIDInMap(children map[string]any) string {
	for _, child := range children {
		if id := firstGemaraOSCALPartyUUID(child); id != "" {
			return id
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func firstPartyUUIDInSlice(children []any) string {
	for _, child := range children {
		if id := firstGemaraOSCALPartyUUID(child); id != "" {
			return id
		}
	}
	return ""
}

// directPartyUUID is the uuid of the first party of a "parties" list, when the
// object has one.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func directPartyUUID(object map[string]any) (string, bool) {
	parties, ok := object["parties"].([]any)
	if !ok || len(parties) == 0 {
		return "", false
	}
	party, ok := parties[0].(map[string]any)
	if !ok {
		return "", false
	}
	id, ok := party["uuid"].(string)
	return id, ok
}

// oscalPartyRewriter points every party reference at one party.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
type oscalPartyRewriter struct {
	partyUUID string
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func (r oscalPartyRewriter) rewrite(value any) {
	if r.partyUUID == "" {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			r.rewriteKey(typed, key, child)
		}
	case []any:
		for _, child := range typed {
			r.rewrite(child)
		}
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func (r oscalPartyRewriter) rewriteKey(object map[string]any, key string, child any) {
	switch key {
	case "actor-uuid", "party-uuid":
		object[key] = r.partyUUID
	case "party-uuids":
		r.setAll(child)
	default:
		r.rewrite(child)
	}
}

// setAll points every id of a list at the party.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func (r oscalPartyRewriter) setAll(list any) {
	ids, ok := list.([]any)
	if !ok {
		return
	}
	for index := range ids {
		ids[index] = r.partyUUID
	}
}
