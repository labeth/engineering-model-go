// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// Authoring assistance for an agent editing model YAML. model.complete offers
// the values a field may take - local and imported qualified IDs, catalog
// terms, enum values and relationship types - and model.diagnose validates the
// model with one document replaced by supplied content, without writing the
// model. Both run offline: imported IDs come from workspace replacements or the
// local module cache, and an unresolved composition only adds a note.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	goyaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

const (
	defaultCompletionMax = 50
	// lockModeEnv is the library's composition lock mode setting (REQ-EMG-064).
)

// completion is one model.complete candidate.
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
type completion struct {
	Value  string
	Kind   string
	Source string
	Title  string
	Fields []string
	Pairs  []validate.RelationshipPair
}

// idFieldKinds maps an ID-valued field to the entity kinds it accepts; an
// empty list accepts any model element.
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
var idFieldKinds = map[string][]string{
	"appliesto":            {},
	"sourcerefs":           {},
	"roots":                {},
	"members":              {},
	"from":                 {},
	"to":                   {},
	"target":               {},
	"sourceref":            {},
	"destinationref":       {},
	"ref":                  {},
	"owner":                {"actor"},
	"responsiblerole":      {"actor"},
	"responsibleroles":     {"actor"},
	"interface.owner":      {"functional_unit", "referenced_element"},
	"group":                {"functional_group"},
	"interfaceref":         {"interface"},
	"softwareinterfaceref": {"interface"},
	"controlref":           {"control"},
	"modelcontrolref":      {"control"},
	"relatedcontrols":      {"control"},
	"threatscenarioref":    {"threat_scenario"},
	"threatscenarios":      {"threat_scenario"},
	"flowrefs":             {"flow"},
	"riskref":              {"risk"},
	"trustboundaryref":     {"trust_boundary"},
	"parent":               {"trust_boundary"},
	"datain":               {"data_object"},
	"dataout":              {"data_object"},
	"datarefs":             {"data_object"},
	"attackvectors":        {"attack_vector"},
	"verificationrefs":     {"control_verification"},
	"mitigationrefs":       {"threat_mitigation"},
	"profileref":           {"compliance_profile"},
	"requirement":          {"requirement"},
	"requirementref":       {"requirement"},
	"hosts":                {"functional_unit"},
}

// TRLC-LINKS: REQ-EMG-069
func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

// TRLC-LINKS: REQ-EMG-069
func firstNonBlank(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// manifestDocuments returns the manifest document keys and their paths
// relative to the model root.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func manifestDocuments(refs model.DocumentReferences) map[string]string {
	out := map[string]string{
		"catalog": refs.Catalog, "requirements": refs.Requirements, "architecture": refs.Architecture,
		"behavior": refs.Behavior, "assurance": refs.Assurance, "compliance": refs.Compliance,
		"views": refs.Views, "decisions": refs.Decisions, "aviation": refs.Aviation,
		"reqLinks": refs.ReqLinks,
	}
	for k, v := range out {
		if strings.TrimSpace(v) == "" {
			delete(out, k)
			continue
		}
		out[k] = filepath.ToSlash(filepath.Clean(strings.TrimSpace(v)))
	}
	return out
}

// resolveDiagnoseDocument maps a document key or manifest-relative path to the
// manifest document key and path. Anything else is rejected.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA
func resolveDiagnoseDocument(documents map[string]string, document string) (string, string, error) {
	document = strings.TrimSpace(document)
	if rel, ok := documents[document]; ok {
		return document, rel, nil
	}
	cleaned := filepath.ToSlash(filepath.Clean(document))
	for key, rel := range documents {
		if rel == cleaned {
			return key, rel, nil
		}
	}
	keys := make([]string, 0, len(documents))
	for key := range documents {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return "", "", fmt.Errorf("model.diagnose document %q is not a manifest document (use one of: %s, or its path)", document, strings.Join(keys, ", "))
}

// modelDiagnose validates the model with one document's content replaced and
// returns that document's diagnostics, the counts of the others and YAML parse
// errors. The model is copied to a private temporary directory and never written.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func (s *Server) modelDiagnose(args map[string]any) (map[string]any, error) {
	target, err := s.diagnoseTarget(args)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"document": target.key, "path": target.rel}
	if parseErrors := yamlParseErrors(target.content); len(parseErrors) > 0 {
		return parseErrorResult(out, parseErrors), nil
	}
	diags, err := diagnoseWithContent(target)
	if err != nil {
		return nil, err
	}
	tally := tallyDiagnostics(diags, target, s.bundle.ManifestPath+":")
	out["valid"] = tally.allErrors == 0
	out["errors"] = tally.docErrors
	out["warnings"] = tally.docWarnings
	out["diagnostics"] = tally.rows
	out["otherDiagnostics"] = tally.otherSummary()
	out["parseErrors"] = []map[string]any{}
	return out, nil
}

// diagnoseRequest is a validated model.diagnose request: the replaced content,
// the model root, the manifest and the document it names.
type diagnoseRequest struct {
	content       string
	root          string
	key           string
	rel           string
	manifestBytes []byte
	documents     map[string]string
}

// readManifestDocuments reads the manifest and returns its bytes and documents.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func readManifestDocuments(manifestPath string) ([]byte, map[string]string, error) {
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read manifest: %w", err)
	}
	var manifest model.ManifestDocument
	if err := goyaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, nil, fmt.Errorf("decode manifest: %w", err)
	}
	return manifestBytes, manifestDocuments(manifest.Documents), nil
}

// diagnoseArguments returns the document and the replacement content.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func diagnoseArguments(args map[string]any) (document, content string, err error) {
	document = firstNonEmptyArg(args, "document")
	if document == "" {
		return "", "", fmt.Errorf("model.diagnose requires document")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", "", fmt.Errorf("model.diagnose requires content (the full YAML text of the document)")
	}
	return document, content, nil
}

// diagnoseTarget validates the arguments and resolves the document to replace.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func (s *Server) diagnoseTarget(args map[string]any) (diagnoseRequest, error) {
	document, content, err := diagnoseArguments(args)
	if err != nil {
		return diagnoseRequest{}, err
	}
	if s.bundle == nil || strings.TrimSpace(s.bundle.ManifestPath) == "" {
		return diagnoseRequest{}, fmt.Errorf("model.diagnose requires a model loaded from engmod.yml")
	}
	root := filepath.Dir(s.bundle.ManifestPath)
	manifestBytes, documents, err := readManifestDocuments(s.bundle.ManifestPath)
	if err != nil {
		return diagnoseRequest{}, err
	}
	key, rel, err := resolveDiagnoseDocument(documents, document)
	if err != nil {
		return diagnoseRequest{}, err
	}
	if err := confineDocuments(root, documents); err != nil {
		return diagnoseRequest{}, err
	}
	return diagnoseRequest{content: content, root: root, key: key, rel: rel, manifestBytes: manifestBytes, documents: documents}, nil
}

// parseErrorResult fills the model.diagnose result for content that is not
// valid YAML.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func parseErrorResult(out map[string]any, parseErrors []map[string]any) map[string]any {
	out["valid"] = false
	out["parseErrors"] = parseErrors
	out["diagnostics"] = []map[string]any{}
	out["errors"] = len(parseErrors)
	out["warnings"] = 0
	out["otherDiagnostics"] = map[string]any{"total": 0, "errors": 0, "warnings": 0}
	out["note"] = "the content is not valid YAML; fix the parse errors to see validation diagnostics"
	return out
}

// diagnoseTally splits diagnostics into those of the replaced document and a
// summary of the others.
type diagnoseTally struct {
	rows        []map[string]any
	other       map[string]int
	otherByCode map[string]int
	remedies    map[string]string
	docErrors   int
	docWarnings int
	allErrors   int
}

// tallyDiagnostics assigns each diagnostic to the replaced document or to the
// others and counts them.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func tallyDiagnostics(diags []validate.Diagnostic, target diagnoseRequest, manifestPrefix string) *diagnoseTally {
	t := &diagnoseTally{
		rows:        []map[string]any{},
		other:       map[string]int{"total": 0, "errors": 0, "warnings": 0},
		otherByCode: map[string]int{},
		remedies:    map[string]string{},
	}
	tree, _ := parser.ParseBytes([]byte(target.content), 0)
	for _, d := range diags {
		// Composition reports root-model fields as <engmod.yml>:<field path>;
		// the field path names the document.
		d.Path = strings.TrimPrefix(d.Path, manifestPrefix)
		if d.Severity == validate.SeverityError {
			t.allErrors++
		}
		if diagnosticDocument(d.Path, target.root, target.documents) != target.key {
			t.addOther(d)
			continue
		}
		t.addDocument(d, documentRow(d, tree, target))
	}
	return t
}

// addOther counts a diagnostic of another document.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func (t *diagnoseTally) addOther(d validate.Diagnostic) {
	t.other["total"]++
	if d.Severity == validate.SeverityError {
		t.other["errors"]++
	} else {
		t.other["warnings"]++
	}
	t.otherByCode[d.Code]++
	if d.Remedy != "" {
		t.remedies[d.Code] = d.Remedy
	}
}

// addDocument records a diagnostic row of the replaced document.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func (t *diagnoseTally) addDocument(d validate.Diagnostic, row map[string]any) {
	if d.Severity == validate.SeverityError {
		t.docErrors++
	} else {
		t.docWarnings++
	}
	t.rows = append(t.rows, row)
}

// documentRow renders a diagnostic of the replaced document with its position.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func documentRow(d validate.Diagnostic, tree *ast.File, target diagnoseRequest) map[string]any {
	row := diagRows([]validate.Diagnostic{d})[0]
	line, column := messagePosition(d.Message, target.rel)
	if line == 0 {
		locator := diagnosticLocator{tree: tree, key: target.key, requirementIDs: requirementIDsForLines(target.key, target.content)}
		line, column = locator.locate(d)
	}
	if line > 0 {
		row["line"] = line
		row["column"] = column
	}
	return row
}

// otherSummary renders the counts of the diagnostics of the other documents. A
// required-switch error of the manifest (lifecycle.missing) belongs to no
// document; its remedy is reported by code.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func (t *diagnoseTally) otherSummary() map[string]any {
	summary := map[string]any{"total": t.other["total"], "errors": t.other["errors"], "warnings": t.other["warnings"]}
	if len(t.otherByCode) > 0 {
		summary["byCode"] = t.otherByCode
	}
	if len(t.remedies) > 0 {
		summary["remedies"] = t.remedies
	}
	return summary
}

// TRLC-LINKS: REQ-EMG-069
func requirementIDsForLines(key, content string) []string {
	if key != "requirements" {
		return nil
	}
	var doc model.RequirementsDocument
	if err := goyaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil
	}
	ids := make([]string, 0, len(doc.Requirements))
	for _, r := range doc.Requirements {
		ids = append(ids, strings.TrimSpace(r.ID))
	}
	return ids
}

// confineDocuments rejects manifest documents that are not local paths or
// that resolve, after symlinks, outside the model root.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func confineDocuments(root string, documents map[string]string) error {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve model root: %w", err)
	}
	for key, rel := range documents {
		if !filepath.IsLocal(filepath.FromSlash(rel)) {
			return fmt.Errorf("manifest document %s (%s) is outside the model root", key, rel)
		}
		resolved, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue // a missing document is reported by the loader
		}
		if inside, err := filepath.Rel(realRoot, resolved); err != nil || !filepath.IsLocal(inside) {
			return fmt.Errorf("manifest document %s (%s) resolves outside the model root", key, rel)
		}
	}
	return nil
}

// yamlParseErrors reports YAML syntax errors with line and column.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func yamlParseErrors(content string) []map[string]any {
	_, err := parser.ParseBytes([]byte(content), 0)
	if err == nil {
		return nil
	}
	row := map[string]any{"message": err.Error()}
	var yerr goyaml.Error
	if errors.As(err, &yerr) {
		row["message"] = yerr.GetMessage()
		if tok := yerr.GetToken(); tok != nil && tok.Position != nil {
			row["line"] = tok.Position.Line
			row["column"] = tok.Position.Column
		}
	}
	return []map[string]any{row}
}

// attribution maps diagnostic path prefixes to manifest document keys.
// ENGMODEL-LINKS: FU-MCP-SERVER
var diagnosticPathDocuments = []struct{ prefix, document, yamlRoot string }{
	{"requirements", "requirements", "requirements"},
	{"catalog", "catalog", "catalog"},
	{"decisions", "decisions", "decisions"},
	{"views", "views", "views"},
	{"documents", "views", "documents"},
	{"naf", "views", "naf"},
	{"design", "views", "design"},
	{"compliance", "compliance", "compliance"},
	{"behavior", "behavior", "behavior"},
	{"assurance", "assurance", "assurance"},
	{"architecture", "architecture", "architecture"},
	{"composition", "architecture", "architecture.composition"},
	{"contract", "architecture", "architecture.contract"},
	{"semantics", "architecture", "architecture.semantics"},
}

// authoredArchitectureDocuments maps authoredArchitecture collections to their
// document and YAML path.
// ENGMODEL-LINKS: FU-MCP-SERVER
var authoredArchitectureDocuments = map[string][2]string{
	"functionalGroups": {"architecture", "architecture.functionalGroups"}, "functionalUnits": {"architecture", "architecture.functionalUnits"},
	"actors": {"architecture", "architecture.actors"}, "referencedElements": {"architecture", "architecture.referencedElements"},
	"interfaces": {"architecture", "architecture.interfaces"}, "dataObjects": {"architecture", "architecture.dataObjects"},
	"deploymentTargets": {"architecture", "architecture.deploymentTargets"}, "hardwareItems": {"architecture", "architecture.hardwareItems"},
	"hardwareInterfaces": {"architecture", "architecture.hardwareInterfaces"},
	"events":             {"behavior", "behavior.events"},
	"flows":              {"behavior", "behavior.flows"}, "mappings": {"behavior", "behavior.relationships"},
	"machines":      {"behavior", "behavior.machines"},
	"attackVectors": {"assurance", "assurance.attackVectors"}, "controls": {"assurance", "assurance.controls"},
	"risks": {"assurance", "assurance.risks"}, "poamItems": {"assurance", "assurance.poamItems"},
	"trustBoundaries": {"assurance", "assurance.trustBoundaries"}, "threatScenarios": {"assurance", "assurance.threatScenarios"},
	"threatAssumptions": {"assurance", "assurance.threatAssumptions"}, "threatOutOfScope": {"assurance", "assurance.threatOutOfScope"},
	"threatMitigations": {"assurance", "assurance.threatMitigations"}, "controlVerifications": {"assurance", "assurance.controlVerifications"},
}

// messagePosition returns the first "<rel>:line:column" position named in a
// loader or schema error message.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func messagePosition(message, rel string) (int, int) {
	re := regexp.MustCompile(regexp.QuoteMeta(rel) + `:(\d+):(\d+)`)
	m := re.FindStringSubmatch(message)
	if m == nil {
		return 0, 0
	}
	line, _ := strconv.Atoi(m[1])
	column, _ := strconv.Atoi(m[2])
	return line, column
}
