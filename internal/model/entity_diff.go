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
	before := make(map[string]Entity, len(from.Entities))
	for _, e := range from.Entities {
		before[e.ID] = e
	}
	after := make(map[string]Entity, len(to.Entities))
	for _, e := range to.Entities {
		after[e.ID] = e
		old, existed := before[e.ID]
		if !existed {
			d.Added = append(d.Added, changeOf(e, nil))
			continue
		}
		if fields := differingFields(old, e); len(fields) > 0 {
			d.Changed = append(d.Changed, changeOf(e, fields))
		}
	}
	for _, e := range from.Entities {
		if _, kept := after[e.ID]; !kept {
			d.Removed = append(d.Removed, changeOf(e, nil))
		}
	}
	oldRefs := make(map[EntityReference]struct{}, len(from.References))
	for _, r := range from.References {
		oldRefs[r] = struct{}{}
	}
	newRefs := make(map[EntityReference]struct{}, len(to.References))
	for _, r := range to.References {
		newRefs[r] = struct{}{}
		if _, had := oldRefs[r]; !had {
			d.ReferencesAdded = append(d.ReferencesAdded, r)
		}
	}
	for _, r := range from.References {
		if _, has := newRefs[r]; has {
			continue
		}
		if _, ok := after[r.From]; !ok {
			continue
		}
		if _, ok := after[r.To]; !ok {
			continue
		}
		d.ReferencesRemoved = append(d.ReferencesRemoved, r)
	}
	for _, list := range [][]EntityChange{d.Added, d.Removed, d.Changed} {
		sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	}
	return d
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
