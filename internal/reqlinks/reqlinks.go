// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE

// Package reqlinks computes contract item fingerprints and evaluates the
// requirement links of a using system against the dependency content it
// resolved. A contract item is a contract.provides capability (CAP-*) or a
// contract.assumptions assumption of use (AOU-*), together with the
// requirements that realize it. The fingerprint covers both, so any change
// to the item or a realizing requirement makes every link on it suspect,
// without the child knowing who links it.
package reqlinks

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
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
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type Realizer struct {
	ID                   string
	Text                 string
	VerificationCriteria string
	Status               string
}

// Item is a contract item with its realizing requirements: the content a
// fingerprint covers.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type Item struct {
	ID                   string
	Kind                 Kind
	Text                 string
	VerificationCriteria string
	Realizers            []Realizer
}

// canonicalEntry and canonicalRealizer fix the field order and names of the
// canonical serialization.
type canonicalEntry struct {
	ID                   string `json:"id"`
	Kind                 string `json:"kind"`
	Text                 string `json:"text"`
	VerificationCriteria string `json:"verificationCriteria"`
}

type canonicalRealizer struct {
	ID                   string `json:"id"`
	Text                 string `json:"text"`
	VerificationCriteria string `json:"verificationCriteria"`
	Status               string `json:"status"`
}

type canonicalDocument struct {
	Item      canonicalEntry      `json:"item"`
	Realizers []canonicalRealizer `json:"realizers"`
}

// normalize collapses every run of whitespace to one space and trims; case
// is kept.
// TRLC-LINKS: REQ-EMG-095
func normalize(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// CanonicalItem returns the canonical serialization the fingerprint hashes:
// compact JSON {"item":{id,kind,text,verificationCriteria},"realizers":[...]}
// with realizers sorted by id, each {id,text,verificationCriteria,status},
// every string whitespace-collapsed and trimmed, no HTML escaping and no
// trailing newline. Tools diff two items through this form.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func CanonicalItem(item Item) []byte {
	doc := canonicalDocument{
		Item: canonicalEntry{
			ID: normalize(item.ID), Kind: normalize(string(item.Kind)),
			Text: normalize(item.Text), VerificationCriteria: normalize(item.VerificationCriteria),
		},
		Realizers: make([]canonicalRealizer, 0, len(item.Realizers)),
	}
	for _, r := range item.Realizers {
		doc.Realizers = append(doc.Realizers, canonicalRealizer{
			ID: normalize(r.ID), Text: normalize(r.Text),
			VerificationCriteria: normalize(r.VerificationCriteria), Status: normalize(r.Status),
		})
	}
	sort.SliceStable(doc.Realizers, func(i, j int) bool { return doc.Realizers[i].ID < doc.Realizers[j].ID })
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(doc) // a struct of strings always encodes
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}

// Fingerprint returns "sha256:<hex>" of CanonicalItem(item).
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func Fingerprint(item Item) string {
	sum := sha256.Sum256(CanonicalItem(item))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ItemsFromBundle returns every contract item of a model by local ID: each
// contract.provides capability and each assumption of use, with the
// requirements that realize it in requirement order. Interface entries are
// not contract items. A capability's deprecated ref counts as a realizer.
// TRLC-LINKS: REQ-EMG-091, REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ItemsFromBundle(bundle model.Bundle) map[string]Item {
	items := map[string]Item{}
	refs := map[string]string{}
	for _, entry := range bundle.Architecture.Contract.Provides {
		id := strings.TrimSpace(entry.ID)
		if id == "" || !strings.EqualFold(strings.TrimSpace(entry.Kind), string(KindCapability)) {
			continue
		}
		items[id] = Item{ID: id, Kind: KindCapability, Text: entry.Note}
		if ref := strings.TrimSpace(entry.Ref); ref != "" {
			refs[ref] = id
		}
	}
	for _, a := range bundle.Architecture.Contract.Assumptions {
		if id := strings.TrimSpace(a.ID); id != "" {
			items[id] = Item{ID: id, Kind: KindAssumption, Text: a.Text, VerificationCriteria: a.VerificationCriteria}
		}
	}
	for _, r := range bundle.Requirements.Requirements {
		realizer := Realizer{ID: strings.TrimSpace(r.ID), Text: r.Text, VerificationCriteria: r.VerificationCriteria, Status: r.Status}
		targets := RealizedItems(r, refs)
		for _, target := range targets {
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
