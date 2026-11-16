// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"reflect"
	"testing"
)

func loadSelf(t *testing.T) Bundle {
	t.Helper()
	b, err := LoadBundle("../../engmod.yml")
	if err != nil {
		t.Fatalf("load own model: %v", err)
	}
	return b
}

func selfIndex(t *testing.T) EntityIndex {
	t.Helper()
	idx, err := BuildEntityIndex(loadSelf(t))
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

// expect fails the test when ok is false. what is a format string and its arguments.
func expect(t *testing.T, ok bool, what ...any) {
	t.Helper()
	if !ok {
		t.Errorf(what[0].(string), what[1:]...)
	}
}

func requireEntity(t *testing.T, idx EntityIndex, id string) Entity {
	t.Helper()
	e, ok := idx.Find(id)
	if !ok {
		t.Fatalf("%s is not in the index", id)
	}
	return e
}

// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexKindsAndTitles(t *testing.T) {
	idx := selfIndex(t)
	cases := map[string]string{
		"REQ-EMG-001":             "requirement",
		"FU-MODEL-LOADER":         "functional_unit",
		"FG-MODEL-AUTHORING":      "functional_group",
		"CAP-EMG-MODEL-AUTHORING": "capability",
		"SM-MODEL-VALIDATION":     "machine",
		"STATE-MODEL-VALID":       "state",
		"ADR-EMG-001":             "decision",

		"TS-ARTIFACT-PARSE-ABUSE":         "threat_scenario",
		"VER-ARTIFACT-FRESHNESS-GATE":     "control_verification",
		"POAM-CONSISTENCY-BATCH1":         "poam_item",
		"COND-CATALOG-REFERENCES-RESOLVE": "catalog_term",
	}
	for id, kind := range cases {
		e := requireEntity(t, idx, id)
		expect(t, e.Kind == kind, "%s: kind %q, want %q", id, e.Kind, kind)
		expect(t, e.Title != "", "%s has no title", id)
	}
}

// A unit that the architecture defines and the catalog names is one entity in two documents; a state belongs to its
// machine and nested entities are not fields of their parent.
// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexDocumentsAndParents(t *testing.T) {
	idx := selfIndex(t)
	unit := requireEntity(t, idx, "FU-MODEL-LOADER")
	expect(t, reflect.DeepEqual([]string{unit.Documents[0], unit.Documents[len(unit.Documents)-1]}, []string{"architecture", "catalog"}), "documents %v", unit.Documents)
	expect(t, requireEntity(t, idx, "STATE-MODEL-VALID").Parent == "SM-MODEL-VALIDATION", "a state's parent is its machine")
	machine := requireEntity(t, idx, "SM-MODEL-VALIDATION")
	_, nested := machine.Fields["states"]
	expect(t, !nested, "nested entities are entities of their own, not fields")
}

// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexEntitiesAreUniqueAndNumerous(t *testing.T) {
	idx := selfIndex(t)
	expect(t, len(idx.Entities) >= 400, "the engine's own model has far more than %d entities", len(idx.Entities))
	seen := map[string]bool{}
	for _, e := range idx.Entities {
		expect(t, !seen[e.ID], "duplicate entity %s", e.ID)
		seen[e.ID] = true
	}
}

func hasReference(idx EntityIndex, want EntityReference) bool {
	for _, r := range idx.References {
		r.Description = want.Description
		if r == want {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexReferencesAreBetweenKnownEntities(t *testing.T) {
	idx := selfIndex(t)
	for _, r := range idx.References {
		_, from := idx.Find(r.From)
		_, to := idx.Find(r.To)
		expect(t, from && to && r.From != r.To, "reference %+v has an unknown end or points at itself", r)
	}
}

// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexHasAllFourEdgeTypes(t *testing.T) {
	idx := selfIndex(t)
	types := map[string]int{}
	for _, r := range idx.References {
		types[r.Type]++
	}
	for _, want := range []string{EdgeField, EdgeMention, EdgeContains, EdgeRelationship} {
		expect(t, types[want] > 0, "no %s edge in the engine's own model: %v", want, types)
	}
	expect(t, types[EdgeRelationship] > 0 && hasVia(idx, "depends_on"), "the model states depends_on relationships")
}

func hasVia(idx EntityIndex, via string) bool {
	for _, r := range idx.References {
		if r.Type == EdgeRelationship && r.Via == via {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexKnownReferences(t *testing.T) {
	idx := selfIndex(t)
	field := EntityReference{From: "REQ-EMG-001", To: "FU-MODEL-LOADER", Via: "appliesTo", Type: EdgeField}
	contains := EntityReference{From: "SM-MODEL-VALIDATION", To: "STATE-MODEL-VALID", Via: EdgeContains, Type: EdgeContains}
	expect(t, hasReference(idx, field), "a requirement refers to its unit through appliesTo")
	expect(t, hasReference(idx, contains), "a machine contains its states")
}

// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexIsDeterministic(t *testing.T) {
	b := loadSelf(t)
	a, err := BuildEntityIndex(b)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		c, _ := BuildEntityIndex(b)
		expect(t, reflect.DeepEqual(a, c), "run %d produced a different index", i)
	}
}
