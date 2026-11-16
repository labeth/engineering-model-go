// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Entity is one authored thing with a stable ID, as a reader of the model sees it: a requirement,
// a functional unit, a threat, a decision, a state of a machine, a step of a flow.
//
// Fields are the authored fields exactly as the document spells them (YAML scalars, lists and
// mappings), without the entities nested inside this one: those are entities of their own whose
// Parent is this entity's ID.
type Entity struct {
	ID string `json:"id"`
	// Kind is the snake_case kind: requirement, functional_unit, threat_scenario, control_verification, ...
	Kind string `json:"kind"`
	// Documents lists the model documents that define this ID (an ID that the catalog names and the
	// architecture defines is one entity in two documents), in document order.
	Documents []string `json:"documents"`
	// Parent is the ID of the enclosing entity (the machine of a state, the flow of a step), or empty.
	Parent string `json:"parent,omitempty"`
	// Title is the name, else the title, else the ID.
	Title  string         `json:"title"`
	Fields map[string]any `json:"fields,omitempty"`
}

// Edge types of an EntityReference.
const (
	// EdgeField: a field of the entity holds exactly the ID of another entity (appliesTo, owner, next, ...).
	EdgeField = "field"
	// EdgeMention: the ID appears inside prose (a requirement's rationale, a decision's context).
	EdgeMention = "mention"
	// EdgeContains: the entity encloses the other (a machine contains its states).
	EdgeContains = "contains"
	// EdgeRelationship: a typed relationship of behavior.relationships (depends_on, reads, writes, ...).
	EdgeRelationship = "relationship"
)

// EntityReference is a directed link from one entity to another that the model states.
type EntityReference struct {
	From string `json:"from"`
	To   string `json:"to"`
	// Via is the field name for field and mention edges, the relationship type for relationship edges,
	// and "contains" for containment.
	Via         string `json:"via"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// EntityIndex is every authored entity of a bundle and every link between them, in a deterministic
// order, so two indexes of the same model are equal and two of a changed model differ exactly where
// the model changed.
type EntityIndex struct {
	Entities   []Entity          `json:"entities"`
	References []EntityReference `json:"references"`
}

// Find returns the entity with the ID.
// TRLC-LINKS: REQ-EMG-189
func (x EntityIndex) Find(id string) (Entity, bool) {
	i := sort.Search(len(x.Entities), func(i int) bool { return x.Entities[i].ID >= id })
	if i < len(x.Entities) && x.Entities[i].ID == id {
		return x.Entities[i], true
	}
	return Entity{}, false
}

// listKinds maps the key a list of entities sits under to the kind of its items. The kinds are the
// ones the MCP tools name (model.list), so a reader can move between the two.
var listKinds = map[string]string{
	"requirements": "requirement", "functionalGroups": "functional_group", "functionalUnits": "functional_unit",
	"actors": "actor", "interfaces": "interface", "dataObjects": "data_object", "deploymentTargets": "deployment_target",
	"hardwareItems": "hardware_item", "hardwareInterfaces": "hardware_interface", "referencedElements": "referenced_element",
	"attackVectors": "attack_vector", "controls": "control", "trustBoundaries": "trust_boundary",
	"provides": "contract_item", "assumptions": "assumption",
	"machines": "machine", "states": "state", "transitions": "transition", "invariants": "invariant",
	"events": "event", "flows": "flow", "steps": "flow_step",
	"threatScenarios": "threat_scenario", "threatAssumptions": "threat_assumption", "threatOutOfScope": "threat_out_of_scope",
	"threatMitigations": "threat_mitigation", "controlVerifications": "control_verification", "risks": "risk", "poamItems": "poam_item",
	"decisions": "decision", "profiles": "compliance_profile", "mappings": "compliance_mapping",
	"views": "view", "stakeholders": "naf_stakeholder", "concerns": "naf_concern", "products": "naf_product",
	"reqLinks": "requirement_link", "systems": "system",
	"features": "catalog_term", "modes": "catalog_term", "conditions": "catalog_term", "dataTerms": "catalog_term",
}

// documentOrder is the order documents are read in. The catalog is last, so a catalog entry for an ID
// that another document defines joins that entity instead of defining a second one.
var documentOrder = []string{"requirements", "architecture", "behavior", "assurance", "compliance", "views", "decisions", "reqLinks", "aviation", "catalog"}

var idToken = regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+\b`)

// BuildEntityIndex reads every authored document of the bundle into entities and references. It
// never reads the code and never changes the bundle.
//
// TRLC-LINKS: REQ-EMG-189
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
func BuildEntityIndex(b Bundle) (EntityIndex, error) {
	docs := map[string]any{
		"requirements": b.Requirements, "architecture": b.Architecture, "behavior": b.Behavior, "assurance": b.Assurance,
		"compliance": b.Compliance, "views": b.Views, "decisions": b.Decisions, "reqLinks": b.ReqLinks, "catalog": b.Catalog,
	}
	if b.Aviation != nil {
		docs["aviation"] = *b.Aviation
	}
	w := &entityWalker{byID: map[string]*Entity{}, edges: map[EntityReference]struct{}{}}
	for _, name := range documentOrder {
		v, ok := docs[name]
		if !ok {
			continue
		}
		raw, err := yaml.Marshal(v)
		if err != nil {
			return EntityIndex{}, fmt.Errorf("entity index: %s: %w", name, err)
		}
		var generic any
		if err := yaml.Unmarshal(raw, &generic); err != nil {
			return EntityIndex{}, fmt.Errorf("entity index: %s: %w", name, err)
		}
		w.document = name
		w.walk(generic, "", "")
	}
	idx := EntityIndex{Entities: make([]Entity, 0, len(w.byID))}
	for _, e := range w.byID {
		idx.Entities = append(idx.Entities, *e)
	}
	sort.Slice(idx.Entities, func(i, j int) bool { return idx.Entities[i].ID < idx.Entities[j].ID })
	w.references(&idx)
	return idx, nil
}

type entityWalker struct {
	document string
	byID     map[string]*Entity
	rels     []EntityReference
	edges    map[EntityReference]struct{}
}

// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) walk(v any, key, parent string) {
	switch x := v.(type) {
	case map[string]any:
		w.walkMap(x, parent)
	case []any:
		w.walkList(x, key, parent)
	}
}

// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) walkMap(m map[string]any, parent string) {
	for _, k := range sortedKeys(m) {
		if k == "relationships" {
			w.relationships(m[k])
			continue
		}
		w.walk(m[k], k, parent)
	}
}

// walkList reads a list of entities, or looks inside each element of any other list.
// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) walkList(list []any, key, parent string) {
	items, ok := entityItems(list)
	if !ok {
		for _, e := range list {
			w.walk(e, key, parent)
		}
		return
	}
	for _, it := range items {
		w.add(it, key, parent)
	}
}

// entityItems reports whether every element is a mapping with a string id.
// TRLC-LINKS: REQ-EMG-189
func entityItems(list []any) ([]map[string]any, bool) {
	if len(list) == 0 {
		return nil, false
	}
	out := make([]map[string]any, 0, len(list))
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok || !hasID(m) {
			return nil, false
		}
		out = append(out, m)
	}
	return out, true
}

// TRLC-LINKS: REQ-EMG-189
func hasID(m map[string]any) bool {
	id, ok := m["id"].(string)
	return ok && strings.TrimSpace(id) != ""
}

// isEntityList reports whether v is a list of entities (so it holds children, not a field).
// TRLC-LINKS: REQ-EMG-189
func isEntityList(v any) bool {
	list, ok := v.([]any)
	if !ok {
		return false
	}
	_, nested := entityItems(list)
	return nested
}

// splitEntityFields separates the fields of an entity mapping from the keys that hold entities nested inside it.
// TRLC-LINKS: REQ-EMG-189
func splitEntityFields(m map[string]any) (fields map[string]any, children []string) {
	fields = map[string]any{}
	for _, k := range sortedKeys(m) {
		switch {
		case k == "id":
		case isEntityList(m[k]):
			children = append(children, k)
		default:
			fields[k] = m[k]
		}
	}
	return fields, children
}

// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) add(m map[string]any, key, parent string) {
	id := m["id"].(string)
	fields, children := splitEntityFields(m)
	if e, ok := w.byID[id]; ok {
		w.join(e, fields, parent)
	} else {
		e := &Entity{ID: id, Kind: kindFor(key, m), Documents: []string{w.document}, Parent: parent, Fields: fields}
		e.Title = titleOf(id, fields)
		w.byID[id] = e
	}
	if parent != "" {
		w.edges[EntityReference{From: parent, To: id, Via: EdgeContains, Type: EdgeContains}] = struct{}{}
	}
	for _, k := range children {
		w.walk(m[k], k, id)
	}
}

// join adds what a later document says about an entity that an earlier one defined: it adds the document and the
// fields the first did not say, and never overrides.
// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) join(e *Entity, fields map[string]any, parent string) {
	if !containsString(e.Documents, w.document) {
		e.Documents = append(e.Documents, w.document)
	}
	for k, v := range fields {
		if _, has := e.Fields[k]; !has {
			e.Fields[k] = v
		}
	}
	if e.Parent == "" {
		e.Parent = parent
	}
	e.Title = titleOf(e.ID, e.Fields)
}

// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) relationships(v any) {
	list, _ := v.([]any)
	for _, e := range list {
		if r, ok := relationshipOf(e); ok {
			w.rels = append(w.rels, r)
		}
	}
}

// relationshipOf reads one behavior.relationships item; an item without both ends is not a relationship.
// TRLC-LINKS: REQ-EMG-189
func relationshipOf(v any) (EntityReference, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return EntityReference{}, false
	}
	from, _ := m["from"].(string)
	to, _ := m["to"].(string)
	typ, _ := m["type"].(string)
	desc, _ := m["description"].(string)
	return EntityReference{From: from, To: to, Via: typ, Type: EdgeRelationship, Description: desc}, from != "" && to != ""
}

// references derives the links: what each entity's fields hold, what its prose mentions, what encloses what, and
// the typed relationships. Only links between entities of the index are kept.
// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) references(idx *EntityIndex) {
	for _, e := range idx.Entities {
		w.scanFields(e)
	}
	w.keepRelationships()
	idx.References = sortedReferences(w.edges)
}

// scanFields adds the references an entity's field values make.
// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) scanFields(e Entity) {
	s := fieldScanner{from: e.ID, known: w.byID, set: w.edges}
	for _, k := range sortedKeys(e.Fields) {
		s.scan(k, e.Fields[k])
	}
}

// keepRelationships adds the typed relationships whose two ends are entities of the index.
// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) keepRelationships() {
	for _, r := range w.rels {
		if w.known(r.From) && w.known(r.To) {
			w.edges[r] = struct{}{}
		}
	}
}

// TRLC-LINKS: REQ-EMG-189
func (w *entityWalker) known(id string) bool {
	_, ok := w.byID[id]
	return ok
}

// TRLC-LINKS: REQ-EMG-189
func sortedReferences(set map[EntityReference]struct{}) []EntityReference {
	out := make([]EntityReference, 0, len(set))
	for r := range set {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return referenceLess(out[i], out[j]) })
	return out
}

// TRLC-LINKS: REQ-EMG-189
func referenceLess(a, b EntityReference) bool {
	for _, p := range [][2]string{{a.From, b.From}, {a.To, b.To}, {a.Via, b.Via}} {
		if p[0] != p[1] {
			return p[0] < p[1]
		}
	}
	return a.Type < b.Type
}

// fieldScanner finds the references one entity's field values make.
type fieldScanner struct {
	from  string
	known map[string]*Entity
	set   map[EntityReference]struct{}
}

// TRLC-LINKS: REQ-EMG-189
func (s fieldScanner) link(via, to, typ string) {
	if _, ok := s.known[to]; ok && to != s.from {
		s.set[EntityReference{From: s.from, To: to, Via: via, Type: typ}] = struct{}{}
	}
}

// TRLC-LINKS: REQ-EMG-189
func (s fieldScanner) scan(via string, v any) {
	switch x := v.(type) {
	case string:
		s.scanString(via, x)
	case []any:
		for _, e := range x {
			s.scan(via, e)
		}
	case map[string]any:
		for _, k := range sortedKeys(x) {
			s.scan(via, x[k])
		}
	}
}

// scanString reads a string that is exactly another entity's ID as a field reference, any other string for IDs
// inside its prose.
// TRLC-LINKS: REQ-EMG-189
func (s fieldScanner) scanString(via, x string) {
	if _, ok := s.known[x]; ok && x != s.from {
		s.link(via, x, EdgeField)
		return
	}
	for _, t := range idToken.FindAllString(x, -1) {
		s.link(via, t, EdgeMention)
	}
}

// TRLC-LINKS: REQ-EMG-189
func kindFor(key string, m map[string]any) string {
	if key == "provides" {
		if k, _ := m["kind"].(string); k == "capability" {
			return "capability"
		}
		return "contract_interface"
	}
	if k, ok := listKinds[key]; ok {
		return k
	}
	return singular(snake(key))
}

// TRLC-LINKS: REQ-EMG-189
func titleOf(id string, fields map[string]any) string {
	for _, k := range []string{"name", "title"} {
		if s, ok := fields[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return id
}

// snake turns a camelCase key into snake_case.
// TRLC-LINKS: REQ-EMG-189
func snake(key string) string {
	var b strings.Builder
	for i, r := range key {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// singular drops a plural ending: ies becomes y, a single s goes, ss stays.
// TRLC-LINKS: REQ-EMG-189
func singular(s string) string {
	switch {
	case strings.HasSuffix(s, "ies"):
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss"):
		return s[:len(s)-1]
	}
	return s
}

// TRLC-LINKS: REQ-EMG-189
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
