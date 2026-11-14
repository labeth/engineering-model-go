// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"reflect"
	"sort"
)

// EntityChange is one entity that differs between two states of a model.
type EntityChange struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	// Document is the first document that defines the entity.
	Document string `json:"document"`
	// Fields lists, for a changed entity, the authored fields that differ (a field that is added or removed counts),
	// plus "(kind)" and "(parent)" when the entity moved to another kind or another enclosing entity.
	Fields []string `json:"fields,omitempty"`
}

// EntityDiff is what changed between two entity indexes of one model.
type EntityDiff struct {
	Added   []EntityChange `json:"added"`
	Removed []EntityChange `json:"removed"`
	Changed []EntityChange `json:"changed"`
	// ReferencesAdded and ReferencesRemoved are the links that appeared and disappeared between entities that
	// exist in the later state (a link of a removed entity goes with it).
	ReferencesAdded   []EntityReference `json:"referencesAdded"`
	ReferencesRemoved []EntityReference `json:"referencesRemoved"`
}

// DiffEntityIndexes compares two indexes by stable ID. Every list is sorted, so the result is deterministic.
//
// TRLC-LINKS: REQ-EMG-190
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
func DiffEntityIndexes(from, to EntityIndex) EntityDiff {
	d := EntityDiff{Added: []EntityChange{}, Removed: []EntityChange{}, Changed: []EntityChange{}, ReferencesAdded: []EntityReference{}, ReferencesRemoved: []EntityReference{}}
	d.compareEntities(from.Entities, to.Entities)
	d.compareReferences(from.References, to.References, entitySet(to.Entities))
	for _, list := range [][]EntityChange{d.Added, d.Removed, d.Changed} {
		sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	}
	return d
}

// TRLC-LINKS: REQ-EMG-190
func entitiesByID(list []Entity) map[string]Entity {
	out := make(map[string]Entity, len(list))
	for _, e := range list {
		out[e.ID] = e
	}
	return out
}

// TRLC-LINKS: REQ-EMG-190
func entitySet(list []Entity) map[string]struct{} {
	out := make(map[string]struct{}, len(list))
	for _, e := range list {
		out[e.ID] = struct{}{}
	}
	return out
}

// compareEntities fills Added, Removed and Changed.
// TRLC-LINKS: REQ-EMG-190
func (d *EntityDiff) compareEntities(from, to []Entity) {
	d.addedAndChanged(entitiesByID(from), to)
	d.removed(from, entitySet(to))
}

// TRLC-LINKS: REQ-EMG-190
func (d *EntityDiff) addedAndChanged(before map[string]Entity, to []Entity) {
	for _, e := range to {
		old, existed := before[e.ID]
		switch fields := differingFields(old, e); {
		case !existed:
			d.Added = append(d.Added, changeOf(e, nil))
		case len(fields) > 0:
			d.Changed = append(d.Changed, changeOf(e, fields))
		}
	}
}

// TRLC-LINKS: REQ-EMG-190
func (d *EntityDiff) removed(from []Entity, after map[string]struct{}) {
	for _, e := range from {
		if _, kept := after[e.ID]; !kept {
			d.Removed = append(d.Removed, changeOf(e, nil))
		}
	}
}

// compareReferences fills the reference lists. A link of a removed entity goes with the entity, so a removed link is
// reported only when both of its ends exist in the later state.
// TRLC-LINKS: REQ-EMG-190
func (d *EntityDiff) compareReferences(from, to []EntityReference, existing map[string]struct{}) {
	d.ReferencesAdded = append(d.ReferencesAdded, referencesNotIn(to, referenceSet(from), nil)...)
	d.ReferencesRemoved = append(d.ReferencesRemoved, referencesNotIn(from, referenceSet(to), existing)...)
}

// referencesNotIn lists the references of list that are not in the set, and, when existing is given, whose two ends exist.
// TRLC-LINKS: REQ-EMG-190
func referencesNotIn(list []EntityReference, set map[EntityReference]struct{}, existing map[string]struct{}) []EntityReference {
	var out []EntityReference
	for _, r := range list {
		if _, in := set[r]; !in && (existing == nil || bothExist(existing, r)) {
			out = append(out, r)
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-190
func referenceSet(list []EntityReference) map[EntityReference]struct{} {
	out := make(map[EntityReference]struct{}, len(list))
	for _, r := range list {
		out[r] = struct{}{}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-190
func bothExist(existing map[string]struct{}, r EntityReference) bool {
	_, from := existing[r.From]
	_, to := existing[r.To]
	return from && to
}

// TRLC-LINKS: REQ-EMG-190
func changeOf(e Entity, fields []string) EntityChange {
	doc := ""
	if len(e.Documents) > 0 {
		doc = e.Documents[0]
	}
	return EntityChange{ID: e.ID, Kind: e.Kind, Title: e.Title, Document: doc, Fields: fields}
}

// TRLC-LINKS: REQ-EMG-190
func differingFields(a, b Entity) []string {
	var out []string
	if a.Kind != b.Kind {
		out = append(out, "(kind)")
	}
	if a.Parent != b.Parent {
		out = append(out, "(parent)")
	}
	keys := map[string]struct{}{}
	for k := range a.Fields {
		keys[k] = struct{}{}
	}
	for k := range b.Fields {
		keys[k] = struct{}{}
	}
	for k := range keys {
		if !reflect.DeepEqual(a.Fields[k], b.Fields[k]) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
