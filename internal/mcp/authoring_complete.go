// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// model.complete: scope resolution, candidate collection and filtering.

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// modelComplete returns the candidates that match prefix for field.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DO-MODEL-AUTHORING-CONTRACT, FU-SYSTEM-COMPOSITION
func (s *Server) modelComplete(args map[string]any) map[string]any {
	prefix := firstNonEmptyArg(args, "prefix")
	field := strings.TrimSpace(toString(args["field"]))
	max := maxArg(args, defaultCompletionMax)
	if max <= 0 {
		max = defaultCompletionMax
	}
	rules := validate.AuthoringRules()
	scope := completionScope(field, rules)
	candidates, notes := s.scopeCandidates(scope, rules, field)
	matched := filterCompletions(candidates, prefix, scope.kinds)
	total := len(matched)
	truncated := total > max
	if truncated {
		matched = matched[:max]
	}
	out := map[string]any{"prefix": prefix, "field": field, "candidates": completionRows(matched), "total": total, "truncated": truncated}
	if len(notes) > 0 {
		out["note"] = strings.Join(notes, "; ")
	}
	return out
}

// scopeCandidates collects the candidates of every source the scope draws from,
// with the notes explaining what was unavailable or widened.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DO-MODEL-AUTHORING-CONTRACT, FU-SYSTEM-COMPOSITION
func (s *Server) scopeCandidates(scope completionScopeResult, rules validate.Rules, field string) ([]completion, []string) {
	notes := []string{}
	var candidates []completion
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
	return candidates, notes
}

// completionRows renders the matched candidates as tool result rows.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DO-MODEL-AUTHORING-CONTRACT, FU-SYSTEM-COMPOSITION
func completionRows(matched []completion) []map[string]any {
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
	return rows
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
	if keys := matchingEnumKeys(rules, lower); len(keys) > 0 {
		return completionScopeResult{enums: true, enumKeys: keys}
	}
	if kinds, ok := idFieldKindsFor(lower); ok {
		return completionScopeResult{ids: true, kinds: kinds}
	}
	return completionScopeResult{enums: true, catalog: true, ids: true, unknown: true}
}

// matchingEnumKeys returns the sorted enum keys equal to lower or ending in
// ".lower".
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func matchingEnumKeys(rules validate.Rules, lower string) []string {
	keys := []string{}
	for key := range rules.Enums {
		k := strings.ToLower(key)
		if k == lower || strings.HasSuffix(k, "."+lower) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// idFieldKindsFor returns the entity kinds an ID field accepts, trying the
// full field name and then its last segment.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func idFieldKindsFor(lower string) ([]string, bool) {
	kinds, ok := idFieldKinds[lower]
	if !ok {
		kinds, ok = idFieldKinds[lower[strings.LastIndex(lower, ".")+1:]]
	}
	return kinds, ok
}

// enumAccumulator collects enum completions in first-seen order.
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
type enumAccumulator struct {
	pairs   map[string][]validate.RelationshipPair
	byValue map[string]*completion
	order   []string
}

// add records one enum value of key; relationship.type values carry their
// allowed from/to pairs, other values merge the keys that allow them.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func (a *enumAccumulator) add(key, value string) {
	if key == "relationship.type" {
		id := "relationship.type\x00" + value
		a.byValue[id] = &completion{Value: value, Kind: "relationship.type", Source: "rules", Title: "relationship type", Fields: []string{key}, Pairs: a.pairs[value]}
		a.order = append(a.order, id)
		return
	}
	id := "enum\x00" + value
	if c, ok := a.byValue[id]; ok {
		c.Fields = append(c.Fields, key)
		return
	}
	a.byValue[id] = &completion{Value: value, Kind: "enum", Source: "rules", Fields: []string{key}}
	a.order = append(a.order, id)
}

// completions renders the accumulated completions in first-seen order.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func (a *enumAccumulator) completions() []completion {
	out := make([]completion, 0, len(a.order))
	for _, id := range a.order {
		c := a.byValue[id]
		if c.Kind == "enum" {
			c.Title = "allowed value of " + strings.Join(c.Fields, ", ")
		}
		out = append(out, *c)
	}
	return out
}

// enumCompletions lists allowed enum values (all keys when keys is empty);
// relationship.type values carry their allowed from/to kind pairs.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func enumCompletions(rules validate.Rules, keys []string) []completion {
	if len(keys) == 0 {
		keys = sortedEnumKeys(rules)
	}
	acc := &enumAccumulator{pairs: relationshipPairsByType(rules), byValue: map[string]*completion{}}
	for _, key := range keys {
		for _, value := range rules.Enums[key] {
			acc.add(key, value)
		}
	}
	return acc.completions()
}

// sortedEnumKeys returns every enum key in sorted order.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func sortedEnumKeys(rules validate.Rules) []string {
	var keys []string
	for key := range rules.Enums {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// relationshipPairsByType groups the allowed relationship from/to pairs by type.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func relationshipPairsByType(rules validate.Rules) map[string][]validate.RelationshipPair {
	pairs := map[string][]validate.RelationshipPair{}
	for _, p := range rules.RelationshipPairs {
		pairs[p.Type] = append(pairs[p.Type], p)
	}
	return pairs
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
			out = append(out, catalogEntryCompletions("catalog."+name, e)...)
		}
	}
	return out
}

// catalogEntryCompletions lists the name and aliases of one catalog entry.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func catalogEntryCompletions(kind string, e model.CatalogEntry) []completion {
	var out []completion
	title := strings.TrimSpace(e.ID + " " + oneLine(e.Definition, 80))
	if n := strings.TrimSpace(e.Name); n != "" {
		out = append(out, completion{Value: n, Kind: kind, Source: "local", Title: title})
	}
	for _, alias := range e.Aliases {
		if a := strings.TrimSpace(alias); a != "" {
			out = append(out, completion{Value: a, Kind: kind, Source: "local", Title: "alias of " + strings.TrimSpace(e.Name) + " (" + e.ID + ")"})
		}
	}
	return out
}

// idCompletions accumulates locally declared entity ID candidates.
// ENGMODEL-LINKS: FU-MCP-SERVER
type idCompletions struct {
	out []completion
}

// add records one entity ID with its kind and a one-line title; blank IDs are skipped.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func (c *idCompletions) add(kind, id, title string) {
	if id = strings.TrimSpace(id); id != "" {
		c.out = append(c.out, completion{Value: id, Kind: kind, Source: "local", Title: oneLine(title, 100)})
	}
}

// localIDCompletions lists every locally declared entity ID with its kind.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func (s *Server) localIDCompletions() []completion {
	if s.bundle == nil {
		return nil
	}
	c := &idCompletions{}
	s.addRequirementAndGraphIDs(c)
	s.addArchitectureIDs(c)
	s.addCompositionIDs(c)
	return c.out
}

// addRequirementAndGraphIDs adds requirement IDs and canonical graph node IDs.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func (s *Server) addRequirementAndGraphIDs(c *idCompletions) {
	if s.requirements != nil {
		for _, r := range s.requirements.Requirements {
			c.add("requirement", r.ID, firstNonBlank(r.Title, r.Text))
		}
	}
	for _, n := range s.graphNodes("", 1<<30) {
		c.add(toString(n["kind"]), toString(n["id"]), toString(n["name"]))
	}
}

// addArchitectureIDs adds view, decision and compliance IDs.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func (s *Server) addArchitectureIDs(c *idCompletions) {
	for _, v := range s.bundle.Architecture.Views {
		c.add("view", v.ID, v.Kind)
	}
	for _, d := range s.bundle.Architecture.Decisions {
		c.add("decision", d.ID, d.Title)
	}
	for _, p := range s.bundle.Architecture.Compliance.Profiles {
		c.add("compliance_profile", p.ID, firstNonBlank(p.Href, p.CatalogHref))
	}
	for _, m := range s.bundle.Architecture.Compliance.Mappings {
		c.add("compliance_mapping", m.ID, m.ModelControlRef)
	}
}

// addCompositionIDs adds subsystem and contract entry IDs.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func (s *Server) addCompositionIDs(c *idCompletions) {
	for _, sub := range s.bundle.Architecture.Composition.Subsystems {
		c.add("subsystem", sub.ID, firstNonBlank(sub.Name, sub.Dependency+"::"+sub.Publication))
	}
	for _, e := range s.bundle.Architecture.Contract.Provides {
		c.add("contract_entry", e.ID, e.Kind)
	}
	for _, e := range s.bundle.Architecture.Contract.Requires {
		c.add("contract_entry", e.ID, e.Kind)
	}
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
	if reason := compositionFailureReason(result, err); reason != "" {
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

// compositionFailureReason explains why the composition cannot expose its
// published entities, or returns "" when it can; lock diagnostics do not count.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION, DEP-LOCAL-WORKSPACE
func compositionFailureReason(result engmodel.CompositionResult, err error) string {
	reason := ""
	if err != nil {
		reason = err.Error()
	}
	for _, d := range result.Diagnostics {
		if reason == "" && d.Severity == validate.SeverityError && !strings.HasPrefix(d.Code, "composition.lock_") {
			reason = d.Code + ": " + d.Message
		}
	}
	return reason
}

// publishedKinds maps the Go type of a published entity value to its entity kind.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
var publishedKinds = map[reflect.Type]string{
	reflect.TypeOf(model.FunctionalGroup{}):     "functional_group",
	reflect.TypeOf(model.FunctionalUnit{}):      "functional_unit",
	reflect.TypeOf(model.Actor{}):               "actor",
	reflect.TypeOf(model.ReferencedElement{}):   "referenced_element",
	reflect.TypeOf(model.Interface{}):           "interface",
	reflect.TypeOf(model.DataObject{}):          "data_object",
	reflect.TypeOf(model.DeploymentTarget{}):    "deployment_target",
	reflect.TypeOf(model.HardwareItem{}):        "hardware_item",
	reflect.TypeOf(model.HardwareInterface{}):   "hardware_interface",
	reflect.TypeOf(model.ContractEntry{}):       "contract_entry",
	reflect.TypeOf(model.MachineState{}):        "state",
	reflect.TypeOf(model.Machine{}):             "machine",
	reflect.TypeOf(model.Event{}):               "event",
	reflect.TypeOf(model.Flow{}):                "flow",
	reflect.TypeOf(model.AttackVector{}):        "attack_vector",
	reflect.TypeOf(model.Control{}):             "control",
	reflect.TypeOf(model.Risk{}):                "risk",
	reflect.TypeOf(model.POAMItem{}):            "poam_item",
	reflect.TypeOf(model.TrustBoundary{}):       "trust_boundary",
	reflect.TypeOf(model.ThreatScenario{}):      "threat_scenario",
	reflect.TypeOf(model.ThreatAssumption{}):    "threat_assumption",
	reflect.TypeOf(model.ThreatOutOfScope{}):    "threat_out_of_scope",
	reflect.TypeOf(model.ThreatMitigation{}):    "threat_mitigation",
	reflect.TypeOf(model.ControlVerification{}): "control_verification",
	reflect.TypeOf(model.ComplianceProfile{}):   "compliance_profile",
	reflect.TypeOf(model.ComplianceMapping{}):   "compliance_mapping",
	reflect.TypeOf(model.Requirement{}):         "requirement",
	reflect.TypeOf(model.View{}):                "view",
}

// importedKindAndTitle derives the entity kind and display title of a published entity.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func importedKindAndTitle(e engmodel.PublishedProjectionEntity) (string, string) {
	kind, ok := publishedKinds[reflect.TypeOf(e.Value)]
	if !ok {
		kind = e.Domain
	}
	return kind, publishedTitle(e.Value)
}

// publishedTitle returns the first non-blank Name, Title or Kind field of a
// struct value as a one-line title.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func publishedTitle(value any) string {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range []string{"Name", "Title", "Kind"} {
		if f := v.FieldByName(name); f.IsValid() && f.Kind() == reflect.String && strings.TrimSpace(f.String()) != "" {
			return oneLine(f.String(), 100)
		}
	}
	return ""
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
		if !completionMatchesPrefix(c.Value, lowerPrefix) {
			continue
		}
		key := c.Value + "\x00" + c.Kind + "\x00" + c.Source
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool { return completionLess(out[i], out[j]) })
	return out
}

// completionMatchesPrefix reports whether value, or the ID of an alias::ID
// value, starts with the lower-cased prefix.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func completionMatchesPrefix(value, lowerPrefix string) bool {
	lowerValue := strings.ToLower(value)
	if strings.HasPrefix(lowerValue, lowerPrefix) {
		return true
	}
	_, tail, ok := strings.Cut(lowerValue, "::")
	return ok && strings.HasPrefix(tail, lowerPrefix)
}

// completionLess orders candidates by case-insensitive value, then value, kind
// and source.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func completionLess(x, y completion) bool {
	a, b := strings.ToLower(x.Value), strings.ToLower(y.Value)
	if a != b {
		return a < b
	}
	if x.Value != y.Value {
		return x.Value < y.Value
	}
	if x.Kind != y.Kind {
		return x.Kind < y.Kind
	}
	return x.Source < y.Source
}
