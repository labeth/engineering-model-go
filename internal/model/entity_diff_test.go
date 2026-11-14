// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"reflect"
	"testing"
)

func ent(id, kind, parent string, fields map[string]any) Entity {
	return Entity{ID: id, Kind: kind, Documents: []string{"requirements"}, Parent: parent, Title: id, Fields: fields}
}

// TRLC-LINKS: REQ-EMG-190
func TestDiffEntityIndexesAddedRemovedChangedAndReferences(t *testing.T) {
	from := EntityIndex{
		Entities: []Entity{
			ent("REQ-1", "requirement", "", map[string]any{"text": "a", "status": "proposed"}),
			ent("REQ-2", "requirement", "", map[string]any{"text": "b"}),
			ent("REQ-3", "requirement", "", map[string]any{"text": "c"}),
			ent("FU-A", "functional_unit", "", nil),
		},
		References: []EntityReference{
			{From: "REQ-1", To: "FU-A", Via: "appliesTo", Type: EdgeField},
			{From: "REQ-3", To: "FU-A", Via: "appliesTo", Type: EdgeField},
		},
	}
	to := EntityIndex{
		Entities: []Entity{
			ent("REQ-1", "requirement", "", map[string]any{"text": "a", "status": "approved", "priority": "high"}),
			ent("REQ-2", "requirement", "", map[string]any{"text": "b"}),
			ent("REQ-4", "requirement", "", map[string]any{"text": "d"}),
			ent("FU-A", "functional_unit", "", nil),
		},
		References: []EntityReference{
			{From: "REQ-1", To: "FU-A", Via: "appliesTo", Type: EdgeField},
			{From: "REQ-4", To: "FU-A", Via: "appliesTo", Type: EdgeField},
		},
	}
	d := DiffEntityIndexes(from, to)
	ids := func(l []EntityChange) []string {
		out := []string{}
		for _, c := range l {
			out = append(out, c.ID)
		}
		return out
	}
	if got := ids(d.Added); !reflect.DeepEqual(got, []string{"REQ-4"}) {
		t.Errorf("added %v", got)
	}
	if got := ids(d.Removed); !reflect.DeepEqual(got, []string{"REQ-3"}) {
		t.Errorf("removed %v", got)
	}
	if len(d.Changed) != 1 || d.Changed[0].ID != "REQ-1" || !reflect.DeepEqual(d.Changed[0].Fields, []string{"priority", "status"}) {
		t.Errorf("changed %+v: a field that is added and one that is edited both count, sorted", d.Changed)
	}
	if len(d.ReferencesAdded) != 1 || d.ReferencesAdded[0].From != "REQ-4" {
		t.Errorf("references added %+v", d.ReferencesAdded)
	}
	// REQ-3 -> FU-A went with REQ-3: a removed entity takes its links with it, they are not reported again.
	if len(d.ReferencesRemoved) != 0 {
		t.Errorf("references removed %+v, want none", d.ReferencesRemoved)
	}
}

// TRLC-LINKS: REQ-EMG-190
func TestDiffEntityIndexesReportsAMoveAndALostLinkBetweenSurvivors(t *testing.T) {
	from := EntityIndex{
		Entities:   []Entity{ent("S-1", "state", "SM-A", nil), ent("SM-A", "machine", "", nil), ent("SM-B", "machine", "", nil)},
		References: []EntityReference{{From: "SM-A", To: "SM-B", Via: "depends_on", Type: EdgeRelationship}},
	}
	to := EntityIndex{Entities: []Entity{ent("S-1", "transition", "SM-B", nil), ent("SM-A", "machine", "", nil), ent("SM-B", "machine", "", nil)}}
	d := DiffEntityIndexes(from, to)
	if len(d.Changed) != 1 || !reflect.DeepEqual(d.Changed[0].Fields, []string{"(kind)", "(parent)"}) {
		t.Errorf("a changed kind and parent are reported as (kind) and (parent): %+v", d.Changed)
	}
	if len(d.ReferencesRemoved) != 1 {
		t.Errorf("a link between two surviving entities that is gone must be reported: %+v", d.ReferencesRemoved)
	}
}

// TRLC-LINKS: REQ-EMG-190
func TestDiffOfAnIndexWithItselfIsEmptyAndNeverNil(t *testing.T) {
	idx, err := BuildEntityIndex(loadSelf(t))
	if err != nil {
		t.Fatal(err)
	}
	d := DiffEntityIndexes(idx, idx)
	if len(d.Added)+len(d.Removed)+len(d.Changed)+len(d.ReferencesAdded)+len(d.ReferencesRemoved) != 0 {
		t.Fatalf("an index differs from itself: %+v", d)
	}
	if d.Added == nil || d.Removed == nil || d.Changed == nil || d.ReferencesAdded == nil || d.ReferencesRemoved == nil {
		t.Fatal("empty lists encode as [], never null")
	}
}
