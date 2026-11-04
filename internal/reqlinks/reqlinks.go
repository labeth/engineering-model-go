// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE

// Package reqlinks evaluates the requirement links of a using system
// against the dependency content it resolved, and compares the meaning of a
// contract item between two versions. A contract item is a
// contract.provides capability (CAP-*) or a contract.assumptions assumption
// of use (AOU-*), together with the requirements that realize it. A link
// records the dependency version it was reviewed at (decision D23):
// published versions are immutable, so the pinned version is the review
// record, and engws bump compares the item's meaning between the pinned and
// the new version.
package reqlinks

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
)

// Kind is the kind of a contract item.
type Kind string

const (
	// KindCapability is a contract.provides entry of kind capability.
	KindCapability Kind = "capability"
	// KindAssumption is a contract.assumptions assumption of use.
	KindAssumption Kind = "assumption"
)

// Realizer is a requirement that names a contract item in realizes (or, for
// one release, the requirement a capability names in its deprecated ref).
// Text and VerificationCriteria are its meaning; the other fields are shown
// in reviews and the trace but never compared (D23).
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type Realizer struct {
	ID                   string
	Text                 string
	Notes                string
	VerificationCriteria string
	VerificationMethods  []string
	Criticality          string
	Status               string
}

// Item is a contract item with its realizing requirements.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type Item struct {
	ID                   string
	Kind                 Kind
	Text                 string
	VerificationCriteria string
	Realizers            []Realizer
}

// meaningEntry and meaningRealizer fix the field order and names of the
// meaning serialization.
type meaningEntry struct {
	ID                   string `json:"id"`
	Kind                 string `json:"kind"`
	Text                 string `json:"text"`
	VerificationCriteria string `json:"verificationCriteria"`
}

type meaningRealizer struct {
	ID                   string `json:"id"`
	Text                 string `json:"text"`
	VerificationCriteria string `json:"verificationCriteria"`
}

type meaningDocument struct {
	Item      meaningEntry      `json:"item"`
	Realizers []meaningRealizer `json:"realizers"`
}

// normalize collapses every run of whitespace to one space and trims; case
// is kept.
// TRLC-LINKS: REQ-EMG-095
func normalize(s string) string { return textutil.CollapseSpace(s) }

// meaningOf returns the normalized meaning of an item, realizers sorted by ID.
// TRLC-LINKS: REQ-EMG-095
func meaningOf(item Item) meaningDocument {
	doc := meaningDocument{
		Item: meaningEntry{
			ID: normalize(item.ID), Kind: normalize(string(item.Kind)),
			Text: normalize(item.Text), VerificationCriteria: normalize(item.VerificationCriteria),
		},
		Realizers: make([]meaningRealizer, 0, len(item.Realizers)),
	}
	for _, r := range item.Realizers {
		doc.Realizers = append(doc.Realizers, meaningRealizer{ID: normalize(r.ID), Text: normalize(r.Text), VerificationCriteria: normalize(r.VerificationCriteria)})
	}
	sort.SliceStable(doc.Realizers, func(i, j int) bool { return doc.Realizers[i].ID < doc.Realizers[j].ID })
	return doc
}

// ItemMeaning returns the meaning of a contract item as compact JSON
// {"item":{id,kind,text,verificationCriteria},"realizers":[{id,text,verificationCriteria}...]}
// with realizers sorted by id, every string whitespace-collapsed and
// trimmed, no HTML escaping and no trailing newline. The text of a
// capability is its note. A realizer's status, notes, criticality,
// verification methods, title, rationale, appliesTo, priority and tags are
// not part of it: they do not change what the item obliges (D23).
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ItemMeaning(item Item) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(meaningOf(item)) // a struct of strings always encodes
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}

// SameMeaning reports whether two versions of a contract item have the same
// meaning (ItemMeaning).
// TRLC-LINKS: REQ-EMG-095, REQ-EMG-100
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func SameMeaning(before, after Item) bool {
	return bytes.Equal(ItemMeaning(before), ItemMeaning(after))
}

// MeaningChanges names each difference in meaning between two versions of
// an item: "item kind", "item text", "item verificationCriteria", then per
// realizer by ID "<ID> added", "<ID> removed", "<ID> text" or
// "<ID> verificationCriteria"; nil when the meaning is the same.
// TRLC-LINKS: REQ-EMG-095, REQ-EMG-100
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func MeaningChanges(before, after Item) []string {
	b, a := meaningOf(before), meaningOf(after)
	var out []string
	if b.Item.Kind != a.Item.Kind {
		out = append(out, "item kind")
	}
	if b.Item.Text != a.Item.Text {
		out = append(out, "item text")
	}
	if b.Item.VerificationCriteria != a.Item.VerificationCriteria {
		out = append(out, "item verificationCriteria")
	}
	old := map[string]meaningRealizer{}
	for _, r := range b.Realizers {
		old[r.ID] = r
	}
	ids := map[string]bool{}
	for _, r := range a.Realizers {
		ids[r.ID] = true
	}
	for _, r := range b.Realizers {
		ids[r.ID] = true
	}
	sorted := make([]string, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)
	now := map[string]meaningRealizer{}
	for _, r := range a.Realizers {
		now[r.ID] = r
	}
	for _, id := range sorted {
		was, had := old[id]
		is, has := now[id]
		switch {
		case !had:
			out = append(out, id+" added")
		case !has:
			out = append(out, id+" removed")
		default:
			if was.Text != is.Text {
				out = append(out, id+" text")
			}
			if was.VerificationCriteria != is.VerificationCriteria {
				out = append(out, id+" verificationCriteria")
			}
		}
	}
	return out
}

// SortedRealizers returns a copy of realizers sorted by ID.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-105, REQ-EMG-107
func SortedRealizers(in []Realizer) []Realizer {
	out := append([]Realizer(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// CapabilityRefs maps each requirement a capability names in its deprecated
// contract ref to that capability, as RealizedItems expects. Imported
// alias::ID entries are skipped.
// TRLC-LINKS: REQ-EMG-091, REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func CapabilityRefs(bundle model.Bundle) map[string]string {
	refs := map[string]string{}
	for _, entry := range bundle.Architecture.Contract.Provides {
		id := strings.TrimSpace(entry.ID)
		if ref := strings.TrimSpace(entry.Ref); ref != "" && id != "" && !strings.Contains(id, "::") && strings.EqualFold(strings.TrimSpace(entry.Kind), string(KindCapability)) {
			refs[ref] = id
		}
	}
	return refs
}

// ItemsFromBundle returns every contract item of a model by local ID: each
// contract.provides capability and each assumption of use, with the
// requirements that realize it in requirement order. Interface entries are
// not contract items. A capability's deprecated ref counts as a realizer.
// TRLC-LINKS: REQ-EMG-091, REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ItemsFromBundle(bundle model.Bundle) map[string]Item {
	items := map[string]Item{}
	for _, entry := range bundle.Architecture.Contract.Provides {
		id := strings.TrimSpace(entry.ID)
		if id == "" || !strings.EqualFold(strings.TrimSpace(entry.Kind), string(KindCapability)) {
			continue
		}
		items[id] = Item{ID: id, Kind: KindCapability, Text: entry.Note}
	}
	for _, a := range bundle.Architecture.Contract.Assumptions {
		if id := strings.TrimSpace(a.ID); id != "" {
			items[id] = Item{ID: id, Kind: KindAssumption, Text: a.Text, VerificationCriteria: a.VerificationCriteria}
		}
	}
	refs := CapabilityRefs(bundle)
	for _, r := range bundle.Requirements.Requirements {
		realizer := Realizer{
			ID: strings.TrimSpace(r.ID), Text: r.Text, Notes: r.Notes, VerificationCriteria: r.VerificationCriteria,
			VerificationMethods: r.VerificationMethods, Criticality: r.Criticality, Status: r.Status,
		}
		for _, target := range RealizedItems(r, refs) {
			if item, ok := items[target]; ok {
				item.Realizers = append(item.Realizers, realizer)
				items[target] = item
			}
		}
	}
	return items
}

// RealizedItems returns the contract item IDs a requirement realizes: its
// realizes entries plus the capability whose deprecated ref names it
// (refs maps a requirement ID to that capability), without duplicates.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func RealizedItems(r model.Requirement, refs map[string]string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(id string) {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, target := range r.Realizes {
		add(target)
	}
	add(refs[strings.TrimSpace(r.ID)])
	return out
}

// PublishedItems is ItemsFromBundle limited to what the publications expose:
// items selected in a publication's architecture list, each with only the
// realizing requirements selected in a publication's requirements list. A
// using system fingerprints this view, so it sees what it resolved and
// nothing the child keeps private.
// TRLC-LINKS: REQ-EMG-092, REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func PublishedItems(bundle model.Bundle, publications ...model.ManifestPublication) map[string]Item {
	architecture, requirements := map[string]bool{}, map[string]bool{}
	for _, p := range publications {
		for _, id := range p.Architecture {
			architecture[strings.TrimSpace(id)] = true
		}
		for _, id := range p.Requirements {
			requirements[strings.TrimSpace(id)] = true
		}
	}
	out := map[string]Item{}
	for id, item := range ItemsFromBundle(bundle) {
		if !architecture[id] {
			continue
		}
		realizers := item.Realizers[:0:0]
		for _, r := range item.Realizers {
			if requirements[r.ID] {
				realizers = append(realizers, r)
			}
		}
		item.Realizers = realizers
		out[id] = item
	}
	return out
}
