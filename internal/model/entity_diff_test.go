// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"reflect"
	"testing"
)

type entitySpec struct {
	id, kind, parent string
	fields           map[string]any
}

func entities(specs ...entitySpec) []Entity {
	out := []Entity{}
	for _, s := range specs {
		out = append(out, Entity{ID: s.id, Kind: s.kind, Documents: []string{"requirements"}, Parent: s.parent, Title: s.id, Fields: s.fields})
	}
	return out
}

func req(id string, fields map[string]any) entitySpec {
	return entitySpec{id: id, kind: "requirement", fields: fields}
}

func applies(from string) EntityReference {
	return EntityReference{From: from, To: "FU-A", Via: "appliesTo", Type: EdgeField}
}

func changeIDs(l []EntityChange) []string {
	out := []string{}
	for _, c := range l {
		out = append(out, c.ID)
	}
	return out
}

func sampleDiff() EntityDiff {
	unit := entitySpec{id: "FU-A", kind: "functional_unit"}
	from := EntityIndex{
		Entities:   entities(req("REQ-1", map[string]any{"text": "a", "status": "proposed"}), req("REQ-2", map[string]any{"text": "b"}), req("REQ-3", map[string]any{"text": "c"}), unit),
		References: []EntityReference{applies("REQ-1"), applies("REQ-3")},
	}
	to := EntityIndex{
		Entities:   entities(req("REQ-1", map[string]any{"text": "a", "status": "approved", "priority": "high"}), req("REQ-2", map[string]any{"text": "b"}), req("REQ-4", map[string]any{"text": "d"}), unit),
		References: []EntityReference{applies("REQ-1"), applies("REQ-4")},
	}
	return DiffEntityIndexes(from, to)
}

// TRLC-LINKS: REQ-EMG-190
func TestDiffEntityIndexesAddedAndRemoved(t *testing.T) {
	d := sampleDiff()
	expect(t, reflect.DeepEqual(changeIDs(d.Added), []string{"REQ-4"}), "added %v", changeIDs(d.Added))
	expect(t, reflect.DeepEqual(changeIDs(d.Removed), []string{"REQ-3"}), "removed %v", changeIDs(d.Removed))
}

// A field that is added and one that is edited both count, sorted; an untouched entity is not listed.
// TRLC-LINKS: REQ-EMG-190
func TestDiffEntityIndexesChangedFieldsAreSorted(t *testing.T) {
	d := sampleDiff()
	expect(t, reflect.DeepEqual(changeIDs(d.Changed), []string{"REQ-1"}), "changed %v", changeIDs(d.Changed))
	expect(t, reflect.DeepEqual(d.Changed[0].Fields, []string{"priority", "status"}), "fields %v", d.Changed[0].Fields)
}

// REQ-3 -> FU-A went with REQ-3: a removed entity takes its links with it, they are not reported again.
// TRLC-LINKS: REQ-EMG-190
func TestDiffEntityIndexesReferences(t *testing.T) {
	d := sampleDiff()
	expect(t, reflect.DeepEqual(d.ReferencesAdded, []EntityReference{applies("REQ-4")}), "references added %+v", d.ReferencesAdded)
	expect(t, len(d.ReferencesRemoved) == 0, "references removed %+v, want none", d.ReferencesRemoved)
}

// TRLC-LINKS: REQ-EMG-190
func TestDiffEntityIndexesReportsAMoveAndALostLinkBetweenSurvivors(t *testing.T) {
	machines := []entitySpec{{id: "SM-A", kind: "machine"}, {id: "SM-B", kind: "machine"}}
	from := EntityIndex{
		Entities:   entities(append(machines, entitySpec{id: "S-1", kind: "state", parent: "SM-A"})...),
		References: []EntityReference{{From: "SM-A", To: "SM-B", Via: "depends_on", Type: EdgeRelationship}},
	}
	to := EntityIndex{Entities: entities(append(machines, entitySpec{id: "S-1", kind: "transition", parent: "SM-B"})...)}
	d := DiffEntityIndexes(from, to)
	expect(t, reflect.DeepEqual(changeIDs(d.Changed), []string{"S-1"}), "changed %+v", d.Changed)
	expect(t, reflect.DeepEqual(d.Changed[0].Fields, []string{"(kind)", "(parent)"}), "a changed kind and parent are reported as (kind) and (parent): %+v", d.Changed)
	expect(t, len(d.ReferencesRemoved) == 1, "a link between two surviving entities that is gone must be reported: %+v", d.ReferencesRemoved)
}

// TRLC-LINKS: REQ-EMG-190
func TestDiffOfAnIndexWithItselfIsEmpty(t *testing.T) {
	idx := selfIndex(t)
	d := DiffEntityIndexes(idx, idx)
	total := len(d.Added) + len(d.Removed) + len(d.Changed) + len(d.ReferencesAdded) + len(d.ReferencesRemoved)
	expect(t, total == 0, "an index differs from itself: %+v", d)
}

// Empty lists encode as [], never null.
// TRLC-LINKS: REQ-EMG-190
func TestDiffListsAreNeverNil(t *testing.T) {
	d := DiffEntityIndexes(EntityIndex{}, EntityIndex{})
	for name, nonNil := range map[string]bool{"added": d.Added != nil, "removed": d.Removed != nil, "changed": d.Changed != nil, "referencesAdded": d.ReferencesAdded != nil, "referencesRemoved": d.ReferencesRemoved != nil} {
		expect(t, nonNil, "%s is nil", name)
	}
}
