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
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	goyaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
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
	"interface.owner":      {"functional_unit"},
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

// modelComplete returns the candidates that match prefix for field.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DO-MODEL-AUTHORING-CONTRACT, FU-SYSTEM-COMPOSITION
func (s *Server) modelComplete(args map[string]any) map[string]any {
	prefix := firstNonEmptyArg(args, "prefix", "query", "q")
	field := strings.TrimSpace(toString(args["field"]))
	max := maxArg(args, defaultCompletionMax)
	if max <= 0 {
		max = defaultCompletionMax
	}
	rules := validate.AuthoringRules()
	notes := []string{}
	var candidates []completion
	scope := completionScope(field, rules)
	if scope.unknown {
		notes = append(notes, fmt.Sprintf("field %q is not a known enum or reference field; returning every candidate kind", field))
	}
	if scope.enums {
		candidates = append(candidates, enumCompletions(rules, scope.enumKeys)...)
	}
	if scope.catalog {
		candidates = append(candidates, s.catalogCompletions(scope.catalogGroup)...)
	}
	if scope.ids {
		candidates = append(candidates, s.localIDCompletions()...)
		imported, note := s.importedIDCompletions()
		candidates = append(candidates, imported...)
		if note != "" {
			notes = append(notes, note)
		}
	}
	matched := filterCompletions(candidates, prefix, scope.kinds)
	total := len(matched)
	truncated := total > max
	if truncated {
		matched = matched[:max]
	}
	rows := make([]map[string]any, 0, len(matched))
	for _, c := range matched {
		row := map[string]any{"value": c.Value, "kind": c.Kind, "source": c.Source, "title": c.Title}
		if len(c.Fields) > 0 {
			row["fields"] = c.Fields
		}
		if len(c.Pairs) > 0 {
			row["pairs"] = c.Pairs
		}
		rows = append(rows, row)
	}
	out := map[string]any{"prefix": prefix, "field": field, "candidates": rows, "total": total, "truncated": truncated}
	if len(notes) > 0 {
		out["note"] = strings.Join(notes, "; ")
	}
	return out
}

// completionScopeResult says which candidate sources a field draws from.
// ENGMODEL-LINKS: FU-MCP-SERVER
type completionScopeResult struct {
	enums, catalog, ids, unknown bool
	enumKeys                     []string
	catalogGroup                 string
	kinds                        []string
}

// completionScope maps a field name to candidate sources: an enum key (or the
// last segment of one), catalog, or an ID field with its accepted kinds.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func completionScope(field string, rules validate.Rules) completionScopeResult {
	lower := strings.ToLower(field)
	if lower == "" {
		return completionScopeResult{enums: true, catalog: true, ids: true}
	}
	if lower == "catalog" || strings.HasPrefix(lower, "catalog.") {
		return completionScopeResult{catalog: true, catalogGroup: strings.TrimPrefix(strings.TrimPrefix(field, "catalog"), ".")}
	}
	keys := []string{}
	for key := range rules.Enums {
		k := strings.ToLower(key)
		if k == lower || strings.HasSuffix(k, "."+lower) {
			keys = append(keys, key)
		}
	}
	if len(keys) > 0 {
		sort.Strings(keys)
		return completionScopeResult{enums: true, enumKeys: keys}
	}
	kinds, ok := idFieldKinds[lower]
	if !ok {
		kinds, ok = idFieldKinds[lower[strings.LastIndex(lower, ".")+1:]]
	}
	if ok {
		return completionScopeResult{ids: true, kinds: kinds}
	}
	return completionScopeResult{enums: true, catalog: true, ids: true, unknown: true}
}

// enumCompletions lists allowed enum values (all keys when keys is empty);
// relationship.type values carry their allowed from/to kind pairs.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func enumCompletions(rules validate.Rules, keys []string) []completion {
	if len(keys) == 0 {
		for key := range rules.Enums {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	pairs := map[string][]validate.RelationshipPair{}
	for _, p := range rules.RelationshipPairs {
		pairs[p.Type] = append(pairs[p.Type], p)
	}
	byValue := map[string]*completion{}
	var order []string
	for _, key := range keys {
		for _, value := range rules.Enums[key] {
			if key == "relationship.type" {
				id := "relationship.type\x00" + value
				byValue[id] = &completion{Value: value, Kind: "relationship.type", Source: "rules", Title: "relationship type", Fields: []string{key}, Pairs: pairs[value]}
				order = append(order, id)
				continue
			}
			id := "enum\x00" + value
			if c, ok := byValue[id]; ok {
				c.Fields = append(c.Fields, key)
				continue
			}
			byValue[id] = &completion{Value: value, Kind: "enum", Source: "rules", Fields: []string{key}}
			order = append(order, id)
		}
	}
	out := make([]completion, 0, len(order))
	for _, id := range order {
		c := byValue[id]
		if c.Kind == "enum" {
			c.Title = "allowed value of " + strings.Join(c.Fields, ", ")
		}
		out = append(out, *c)
	}
	return out
}

// catalogCompletions lists catalog term names and aliases, optionally of one group.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func (s *Server) catalogCompletions(group string) []completion {
	if s.bundle == nil {
		return nil
	}
	var out []completion
	value := reflect.ValueOf(s.bundle.Catalog.Catalog)
	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("yaml"), ",")[0]
		if group != "" && !strings.EqualFold(group, name) {
			continue
		}
		entries, ok := value.Field(i).Interface().([]model.CatalogEntry)
		if !ok {
			continue
		}
		for _, e := range entries {
			title := strings.TrimSpace(e.ID + " " + oneLine(e.Definition, 80))
			if n := strings.TrimSpace(e.Name); n != "" {
				out = append(out, completion{Value: n, Kind: "catalog." + name, Source: "local", Title: title})
			}
			for _, alias := range e.Aliases {
				if a := strings.TrimSpace(alias); a != "" {
					out = append(out, completion{Value: a, Kind: "catalog." + name, Source: "local", Title: "alias of " + strings.TrimSpace(e.Name) + " (" + e.ID + ")"})
				}
			}
		}
	}
	return out
}

// localIDCompletions lists every locally declared entity ID with its kind.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func (s *Server) localIDCompletions() []completion {
	if s.bundle == nil {
		return nil
	}
	var out []completion
	add := func(kind, id, title string) {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, completion{Value: id, Kind: kind, Source: "local", Title: oneLine(title, 100)})
		}
	}
	if s.requirements != nil {
		for _, r := range s.requirements.Requirements {
			add("requirement", r.ID, firstNonBlank(r.Title, r.Text))
		}
	}
	for _, n := range s.graphNodes("", 1<<30) {
		add(toString(n["kind"]), toString(n["id"]), toString(n["name"]))
	}
	for _, v := range s.bundle.Architecture.Views {
		add("view", v.ID, v.Kind)
	}
	for _, d := range s.bundle.Architecture.Decisions {
		add("decision", d.ID, d.Title)
	}
	for _, p := range s.bundle.Architecture.Compliance.Profiles {
		add("compliance_profile", p.ID, firstNonBlank(p.Href, p.CatalogHref))
	}
	for _, m := range s.bundle.Architecture.Compliance.Mappings {
		add("compliance_mapping", m.ID, m.ModelControlRef)
	}
	for _, sub := range s.bundle.Architecture.Composition.Subsystems {
		add("subsystem", sub.ID, firstNonBlank(sub.Name, sub.Dependency+"::"+sub.Publication))
	}
	for _, c := range s.bundle.Architecture.Contract.Provides {
		add("contract_entry", c.ID, c.Kind)
	}
	for _, c := range s.bundle.Architecture.Contract.Requires {
		add("contract_entry", c.ID, c.Kind)
	}
	return out
}

// importedIDCompletions lists alias::ID for every entity the selected
// dependency publications expose. Resolution failure returns a note, not an error.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION, DEP-LOCAL-WORKSPACE
func (s *Server) importedIDCompletions() ([]completion, string) {
	if s.bundle == nil || !engmodel.HasComposition(*s.bundle) {
		return nil, ""
	}
	// The lock state (missing or stale engmod.lock.yml) does not hide the
	// published entities; composition.resolve and model.diagnose report it.
	result, err := engmodel.GenerateCompositionFromFileWith(s.bundle.ManifestPath, resolution())
	reason := ""
	if err != nil {
		reason = err.Error()
	}
	for _, d := range result.Diagnostics {
		if reason == "" && d.Severity == validate.SeverityError && !strings.HasPrefix(d.Code, "composition.lock_") {
			reason = d.Code + ": " + d.Message
		}
	}
	if reason != "" {
		return nil, "imported IDs unavailable (dependencies are not in a workspace replacement or the local module cache, or composition is invalid): " + reason + "; returning local candidates only"
	}
	projection := engmodel.BuildCompositionProjection(result)
	out := make([]completion, 0, len(projection.Entities))
	for _, e := range projection.Entities {
		kind, title := importedKindAndTitle(e)
		alias, _, _ := model.ParseQualifiedReference(e.QualifiedID)
		out = append(out, completion{Value: e.QualifiedID, Kind: kind, Source: alias, Title: title})
	}
	return out, ""
}

// importedKindAndTitle derives the entity kind and display title of a published entity.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func importedKindAndTitle(e engmodel.PublishedProjectionEntity) (string, string) {
	kind := e.Domain
	switch e.Value.(type) {
	case model.FunctionalGroup:
		kind = "functional_group"
	case model.FunctionalUnit:
		kind = "functional_unit"
	case model.Actor:
		kind = "actor"
	case model.ReferencedElement:
		kind = "referenced_element"
	case model.Interface:
		kind = "interface"
	case model.DataObject:
		kind = "data_object"
	case model.DeploymentTarget:
		kind = "deployment_target"
	case model.HardwareItem:
		kind = "hardware_item"
	case model.HardwareInterface:
		kind = "hardware_interface"
	case model.ContractEntry:
		kind = "contract_entry"
	case model.State:
		kind = "state"
	case model.Event:
		kind = "event"
	case model.Flow:
		kind = "flow"
	case model.AttackVector:
		kind = "attack_vector"
	case model.Control:
		kind = "control"
	case model.Risk:
		kind = "risk"
	case model.POAMItem:
		kind = "poam_item"
	case model.TrustBoundary:
		kind = "trust_boundary"
	case model.ThreatScenario:
		kind = "threat_scenario"
	case model.ThreatAssumption:
		kind = "threat_assumption"
	case model.ThreatOutOfScope:
		kind = "threat_out_of_scope"
	case model.ThreatMitigation:
		kind = "threat_mitigation"
	case model.ControlVerification:
		kind = "control_verification"
	case model.ComplianceProfile:
		kind = "compliance_profile"
	case model.ComplianceMapping:
		kind = "compliance_mapping"
	case model.Requirement:
		kind = "requirement"
	case model.View:
		kind = "view"
	}
	title := ""
	if v := reflect.ValueOf(e.Value); v.Kind() == reflect.Struct {
		for _, name := range []string{"Name", "Title", "Kind"} {
			if f := v.FieldByName(name); f.IsValid() && f.Kind() == reflect.String && strings.TrimSpace(f.String()) != "" {
				title = oneLine(f.String(), 100)
				break
			}
		}
	}
	return kind, title
}

// filterCompletions keeps candidates whose value (or, for alias::ID, the ID)
// starts with prefix case-insensitively and whose kind is accepted, de-duplicated
// and sorted deterministically.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func filterCompletions(in []completion, prefix string, kinds []string) []completion {
	lowerPrefix := strings.ToLower(prefix)
	accepted := map[string]bool{}
	for _, k := range kinds {
		accepted[k] = true
	}
	seen := map[string]bool{}
	out := []completion{}
	for _, c := range in {
		if len(accepted) > 0 && !accepted[c.Kind] {
			continue
		}
		lowerValue := strings.ToLower(c.Value)
		match := strings.HasPrefix(lowerValue, lowerPrefix)
		if !match {
			if _, tail, ok := strings.Cut(lowerValue, "::"); ok {
				match = strings.HasPrefix(tail, lowerPrefix)
			}
		}
		if !match {
			continue
		}
		key := c.Value + "\x00" + c.Kind + "\x00" + c.Source
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].Value), strings.ToLower(out[j].Value)
		if a != b {
			return a < b
		}
		if out[i].Value != out[j].Value {
			return out[i].Value < out[j].Value
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Source < out[j].Source
	})
	return out
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
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, FU-VALIDATION-ENGINE
func (s *Server) modelDiagnose(args map[string]any) (map[string]any, error) {
	document := firstNonEmptyArg(args, "document", "path")
	if document == "" {
		return nil, fmt.Errorf("model.diagnose requires document")
	}
	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("model.diagnose requires content (the full YAML text of the document)")
	}
	if s.bundle == nil || strings.TrimSpace(s.bundle.ManifestPath) == "" {
		return nil, fmt.Errorf("model.diagnose requires a model loaded from engmod.yml")
	}
	root := filepath.Dir(s.bundle.ManifestPath)
	manifestBytes, err := os.ReadFile(s.bundle.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var manifest model.ManifestDocument
	if err := goyaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	documents := manifestDocuments(manifest.Documents)
	key, rel, err := resolveDiagnoseDocument(documents, document)
	if err != nil {
		return nil, err
	}
	if err := confineDocuments(root, documents); err != nil {
		return nil, err
	}

	out := map[string]any{"document": key, "path": rel}
	if parseErrors := yamlParseErrors(content); len(parseErrors) > 0 {
		out["valid"] = false
		out["parseErrors"] = parseErrors
		out["diagnostics"] = []map[string]any{}
		out["errors"] = len(parseErrors)
		out["warnings"] = 0
		out["otherDiagnostics"] = map[string]any{"total": 0, "errors": 0, "warnings": 0}
		out["note"] = "the content is not valid YAML; fix the parse errors to see validation diagnostics"
		return out, nil
	}

	diags, err := diagnoseWithContent(root, manifestBytes, documents, rel, content)
	if err != nil {
		return nil, err
	}
	docRows := []map[string]any{}
	other := map[string]int{"total": 0, "errors": 0, "warnings": 0}
	otherByCode := map[string]int{}
	docErrors, docWarnings, allErrors := 0, 0, 0
	tree, _ := parser.ParseBytes([]byte(content), 0)
	manifestPrefix := s.bundle.ManifestPath + ":"
	for _, d := range diags {
		// Composition reports root-model fields as <engmod.yml>:<field path>;
		// the field path names the document.
		d.Path = strings.TrimPrefix(d.Path, manifestPrefix)
		if d.Severity == validate.SeverityError {
			allErrors++
		}
		if diagnosticDocument(d.Path, root, documents) != key {
			other["total"]++
			if d.Severity == validate.SeverityError {
				other["errors"]++
			} else {
				other["warnings"]++
			}
			otherByCode[d.Code]++
			continue
		}
		row := map[string]any{"code": d.Code, "severity": string(d.Severity), "message": d.Message, "path": d.Path}
		if line, column := messagePosition(d.Message, rel); line > 0 {
			row["line"] = line
			row["column"] = column
		} else if line, column := diagnosticLine(tree, key, d, requirementIDsForLines(key, content)); line > 0 {
			row["line"] = line
			row["column"] = column
		}
		if d.Severity == validate.SeverityError {
			docErrors++
		} else {
			docWarnings++
		}
		docRows = append(docRows, row)
	}
	otherSummary := map[string]any{"total": other["total"], "errors": other["errors"], "warnings": other["warnings"]}
	if len(otherByCode) > 0 {
		otherSummary["byCode"] = otherByCode
	}
	out["valid"] = allErrors == 0
	out["errors"] = docErrors
	out["warnings"] = docWarnings
	out["diagnostics"] = docRows
	out["otherDiagnostics"] = otherSummary
	out["parseErrors"] = []map[string]any{}
	return out, nil
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

// diagnoseWithContent copies the manifest and its documents into a private
// temporary directory, replaces one document, loads and validates the model
// with the same pipeline as the model check, and removes the copy. The loaded
// bundle is re-rooted at the real model directory so code, trace and
// composition analysis see the real project.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func diagnoseWithContent(root string, manifest []byte, documents map[string]string, target, content string) ([]validate.Diagnostic, error) {
	tmp, err := os.MkdirTemp("", "engmod-diagnose-")
	if err != nil {
		return nil, fmt.Errorf("create temporary model copy: %w", err)
	}
	defer os.RemoveAll(tmp)
	write := func(rel string, data []byte) error {
		path := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		return os.WriteFile(path, data, 0o600)
	}
	if err := write("engmod.yml", manifest); err != nil {
		return nil, fmt.Errorf("copy manifest: %w", err)
	}
	for _, rel := range documents {
		data := []byte(content)
		if rel != target {
			if data, err = os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
				return nil, fmt.Errorf("read %s: %w", rel, err)
			}
		}
		if err := write(rel, data); err != nil {
			return nil, fmt.Errorf("copy %s: %w", rel, err)
		}
	}
	scrub := func(msg string) string {
		return strings.ReplaceAll(strings.ReplaceAll(msg, tmp+string(filepath.Separator), ""), tmp, ".")
	}
	bundle, err := model.LoadBundle(filepath.Join(tmp, "engmod.yml"))
	if err != nil {
		var aviation *model.AviationValidationError
		if errors.As(err, &aviation) {
			out := []validate.Diagnostic{}
			for _, d := range aviation.Diagnostics {
				out = append(out, validate.Diagnostic{Code: d.Code, Severity: validate.SeverityError, Message: d.Message, Path: documents["aviation"] + "#" + d.Path})
			}
			return out, nil
		}
		return []validate.Diagnostic{{Code: "model.load_failed", Severity: validate.SeverityError, Message: scrub(err.Error()), Path: loadErrorPath(scrub(err.Error()), documents, target)}}, nil
	}
	rebase := func(p string) string {
		if rel, rerr := filepath.Rel(tmp, p); rerr == nil && filepath.IsLocal(rel) {
			return filepath.Join(root, rel)
		}
		return p
	}
	bundle.ManifestPath = rebase(bundle.ManifestPath)
	for _, p := range []*string{&bundle.ArchitecturePath, &bundle.BehaviorPath, &bundle.AssurancePath, &bundle.CompliancePath, &bundle.ViewsPath, &bundle.CatalogPath, &bundle.DecisionsPath, &bundle.RequirementsPath, &bundle.DesignPath, &bundle.AviationPath} {
		if *p != "" {
			*p = rebase(*p)
		}
	}

	// Dependencies resolve from the real model root (its workspace and module
	// cache); the root system is the in-memory bundle with the submitted
	// content, so its alias::ID references and composition are checked as
	// submitted. Like every MCP read tool it writes no lock and reuses the
	// module cache offline (see resolution).
	options := resolution()
	var diags []validate.Diagnostic
	canonical, cerr := model.NewCanonicalBundle(bundle)
	for _, d := range canonical.Diagnostics() {
		diags = append(diags, validate.Diagnostic{Code: d.Code, Severity: validate.Severity(d.Severity), Message: d.Message, Path: d.Path})
	}
	if cerr != nil {
		diags = append(diags, validate.Bundle(bundle)...)
	} else {
		result, _ := engmodel.GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, engmodel.AsciiDocOptions{Resolution: options})
		diags = append(diags, result.Diagnostics...)
	}
	for i := range diags {
		diags[i].Message = scrub(diags[i].Message)
		diags[i].Path = scrub(diags[i].Path)
	}
	return validate.SortDiagnostics(diags), nil
}

// loadErrorPath attributes a load error to the document its message names.
// TRLC-LINKS: REQ-EMG-069
func loadErrorPath(msg string, documents map[string]string, target string) string {
	for _, rel := range documents {
		if strings.Contains(msg, rel) && rel != target {
			if !strings.Contains(msg, target) {
				return rel
			}
		}
	}
	return target
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
	"states":             {"behavior", "behavior.states"}, "events": {"behavior", "behavior.events"},
	"flows": {"behavior", "behavior.flows"}, "mappings": {"behavior", "behavior.relationships"},
	"attackVectors": {"assurance", "assurance.attackVectors"}, "controls": {"assurance", "assurance.controls"},
	"risks": {"assurance", "assurance.risks"}, "poamItems": {"assurance", "assurance.poamItems"},
	"trustBoundaries": {"assurance", "assurance.trustBoundaries"}, "threatScenarios": {"assurance", "assurance.threatScenarios"},
	"threatAssumptions": {"assurance", "assurance.threatAssumptions"}, "threatOutOfScope": {"assurance", "assurance.threatOutOfScope"},
	"threatMitigations": {"assurance", "assurance.threatMitigations"}, "controlVerifications": {"assurance", "assurance.controlVerifications"},
}

// diagnosticDocument returns the manifest document key a diagnostic path
// belongs to, or "" when it cannot be attributed (code, canonical graph).
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func diagnosticDocument(path, root string, documents map[string]string) string {
	document, _, _ := splitDiagnosticPath(path)
	if document != "" {
		return document
	}
	file := diagnosticFilePath(path)
	if file == "" {
		return ""
	}
	if filepath.IsAbs(file) {
		if rel, err := filepath.Rel(root, file); err == nil {
			file = rel
		}
	}
	file = filepath.ToSlash(filepath.Clean(file))
	for key, rel := range documents {
		if rel == file || strings.HasPrefix(path, rel+"#") {
			return key
		}
	}
	return ""
}

// splitDiagnosticPath returns the document key, its YAML root path and the
// remainder (".x[0]" or "[REQ-ID]") of a model diagnostic path.
// TRLC-LINKS: REQ-EMG-069
func splitDiagnosticPath(path string) (string, string, string) {
	path = strings.TrimSpace(path)
	if rest, ok := strings.CutPrefix(path, "authoredArchitecture."); ok {
		name := rest
		if i := strings.IndexAny(rest, ".["); i >= 0 {
			name = rest[:i]
		}
		if target, ok := authoredArchitectureDocuments[name]; ok {
			return target[0], target[1], rest[len(name):]
		}
		return "", "", ""
	}
	for _, p := range diagnosticPathDocuments {
		if rest, ok := strings.CutPrefix(path, p.prefix); ok && (rest == "" || rest[0] == '.' || rest[0] == '[') {
			return p.document, p.yamlRoot, rest
		}
	}
	return "", "", ""
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

// TRLC-LINKS: REQ-EMG-069
var diagnosticIndexRe = regexp.MustCompile(`\[([^\]]+)\]`)

// diagnosticLine locates a diagnostic in the document's YAML and returns its
// line and column, trimming path segments until a node is found.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func diagnosticLine(tree *ast.File, key string, d validate.Diagnostic, requirementIDs []string) (int, int) {
	if tree == nil {
		return 0, 0
	}
	document, yamlRoot, rest := splitDiagnosticPath(d.Path)
	if document != key {
		return 0, 0
	}
	if at := strings.Index(rest, "@"); at >= 0 {
		rest = rest[:at]
	}
	rest = diagnosticIndexRe.ReplaceAllStringFunc(rest, func(m string) string {
		inner := m[1 : len(m)-1]
		if _, err := strconv.Atoi(inner); err == nil {
			return m
		}
		for i, id := range requirementIDs {
			if id == inner {
				return fmt.Sprintf("[%d]", i)
			}
		}
		return ""
	})
	full := "$." + yamlRoot + rest
	for candidate := full; strings.HasPrefix(candidate, "$."); {
		if p, err := goyaml.PathString(candidate); err == nil {
			if node, err := p.FilterFile(tree); err == nil && node != nil {
				if tok := node.GetToken(); tok != nil && tok.Position != nil {
					return tok.Position.Line, tok.Position.Column
				}
			}
		}
		cut := strings.LastIndexAny(candidate, ".[")
		if cut <= 1 {
			break
		}
		candidate = candidate[:cut]
	}
	return 0, 0
}
