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

// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexKindsDocumentsAndParents(t *testing.T) {
	idx, err := BuildEntityIndex(loadSelf(t))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ id, kind string }{
		{"REQ-EMG-001", "requirement"},
		{"FU-MODEL-LOADER", "functional_unit"},
		{"FG-MODEL-AUTHORING", "functional_group"},
		{"CAP-EMG-MODEL-AUTHORING", "capability"},
		{"SM-MODEL-VALIDATION", "machine"},
		{"STATE-MODEL-VALID", "state"},
		{"ADR-EMG-001", "decision"},
	}
	for _, c := range cases {
		e, ok := idx.Find(c.id)
		if !ok {
			t.Fatalf("%s is not in the index", c.id)
		}
		if e.Kind != c.kind {
			t.Errorf("%s: kind %q, want %q", c.id, e.Kind, c.kind)
		}
		if e.Title == "" {
			t.Errorf("%s has no title", c.id)
		}
	}
	unit, _ := idx.Find("FU-MODEL-LOADER")
	if len(unit.Documents) < 2 || unit.Documents[0] != "architecture" || unit.Documents[len(unit.Documents)-1] != "catalog" {
		t.Errorf("a unit the architecture defines and the catalog names is one entity in both documents, got %v", unit.Documents)
	}
	state, _ := idx.Find("STATE-MODEL-VALID")
	if state.Parent != "SM-MODEL-VALIDATION" {
		t.Errorf("a state's parent is its machine, got %q", state.Parent)
	}
	if _, nested := unit.Fields["states"]; nested {
		t.Error("nested entities are entities of their own, not fields")
	}
}

// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexReferencesAreBetweenKnownEntitiesAndTyped(t *testing.T) {
	idx, err := BuildEntityIndex(loadSelf(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Entities) < 400 {
		t.Fatalf("the engine's own model has far more than %d entities", len(idx.Entities))
	}
	seen := map[string]bool{}
	for _, e := range idx.Entities {
		if seen[e.ID] {
			t.Fatalf("duplicate entity %s", e.ID)
		}
		seen[e.ID] = true
	}
	types := map[string]int{}
	for _, r := range idx.References {
		if !seen[r.From] || !seen[r.To] {
			t.Fatalf("reference %+v has an unknown end", r)
		}
		if r.From == r.To {
			t.Fatalf("self reference %+v", r)
		}
		types[r.Type]++
	}
	for _, want := range []string{EdgeField, EdgeMention, EdgeContains, EdgeRelationship} {
		if types[want] == 0 {
			t.Errorf("no %s edge in the engine's own model: %v", want, types)
		}
	}
	has := func(from, to, via, typ string) bool {
		for _, r := range idx.References {
			if r.From == from && r.To == to && r.Via == via && r.Type == typ {
				return true
			}
		}
		return false
	}
	if !has("REQ-EMG-001", "FU-MODEL-LOADER", "appliesTo", EdgeField) {
		t.Error("a requirement refers to its unit through appliesTo")
	}
	if !has("SM-MODEL-VALIDATION", "STATE-MODEL-VALID", EdgeContains, EdgeContains) {
		t.Error("a machine contains its states")
	}
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
		if !reflect.DeepEqual(a, c) {
			t.Fatalf("run %d produced a different index", i)
		}
	}
}

// TRLC-LINKS: REQ-EMG-189
func TestEntityIndexTypedRelationshipsKeepTheirDescription(t *testing.T) {
	idx, err := BuildEntityIndex(loadSelf(t))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, r := range idx.References {
		if r.Type == EdgeRelationship && r.Via == "depends_on" {
			n++
		}
	}
	if n == 0 {
		t.Fatal("the engine's own model states depends_on relationships")
	}
}
