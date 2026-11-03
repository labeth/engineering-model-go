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
	"fmt"
	"regexp"
	"sort"
	"strconv"
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
// one release, the requirement a capability names in its deprecated ref),
// with the requirement fields the fingerprint covers.
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

// CanonicalVersion is the version of the canonical form CanonicalItem
// writes and Fingerprint names in its prefix (decision D17). Any change to
// the canonical form, a field added, removed, renamed or normalized
// differently, is a new version: a fingerprint of another version is
// reported as link.fingerprint_outdated, never compared.
const CanonicalVersion = 1

// canonicalEntry and canonicalRealizer fix the field order and names of the
// canonical serialization (version 1).
type canonicalEntry struct {
	ID                   string `json:"id"`
	Kind                 string `json:"kind"`
	Text                 string `json:"text"`
	VerificationCriteria string `json:"verificationCriteria"`
}

type canonicalRealizer struct {
	ID                   string   `json:"id"`
	Text                 string   `json:"text"`
	Notes                string   `json:"notes"`
	VerificationCriteria string   `json:"verificationCriteria"`
	VerificationMethods  []string `json:"verificationMethods"`
	Criticality          string   `json:"criticality"`
	Status               string   `json:"status"`
}

type canonicalDocument struct {
	Item      canonicalEntry      `json:"item"`
	Realizers []canonicalRealizer `json:"realizers"`
}

// normalize collapses every run of whitespace to one space and trims; case
// is kept.
// TRLC-LINKS: REQ-EMG-095
func normalize(s string) string { return textutil.CollapseSpace(s) }

// normalizedMethods returns the verification methods normalized, without
// empty entries, sorted ([] when there are none).
// TRLC-LINKS: REQ-EMG-095
func normalizedMethods(in []string) []string {
	out := []string{}
	for _, m := range in {
		if m = normalize(m); m != "" {
			out = append(out, m)
		}
	}
	sort.Strings(out)
	return out
}

// CanonicalItem returns the canonical serialization (version 1) the
// fingerprint hashes: compact JSON
// {"item":{id,kind,text,verificationCriteria},"realizers":[...]} with
// realizers sorted by id, each
// {id,text,notes,verificationCriteria,verificationMethods,criticality,status},
// verificationMethods sorted, every string whitespace-collapsed and trimmed,
// no HTML escaping and no trailing newline. Tools diff two items through
// this form.
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
			ID: normalize(r.ID), Text: normalize(r.Text), Notes: normalize(r.Notes),
			VerificationCriteria: normalize(r.VerificationCriteria), VerificationMethods: normalizedMethods(r.VerificationMethods),
			Criticality: normalize(r.Criticality), Status: normalize(r.Status),
		})
	}
	sort.SliceStable(doc.Realizers, func(i, j int) bool { return doc.Realizers[i].ID < doc.Realizers[j].ID })
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(doc) // a struct of strings always encodes
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}

// Fingerprint returns "sha256:v<CanonicalVersion>:<hex>" of
// CanonicalItem(item).
// TRLC-LINKS: REQ-EMG-095, REQ-EMG-109
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func Fingerprint(item Item) string {
	sum := sha256.Sum256(CanonicalItem(item))
	return fmt.Sprintf("sha256:v%d:%s", CanonicalVersion, hex.EncodeToString(sum[:]))
}

var fingerprintPattern = regexp.MustCompile(`^sha256:(?:v([0-9]+):)?[0-9a-f]{64}$`)

// FingerprintVersion returns the canonical-form version a recorded
// fingerprint names: N for "sha256:vN:<hex>", 0 for the unversioned
// "sha256:<hex>" of the v0.8.0 pre-releases, -1 when it is no fingerprint.
// TRLC-LINKS: REQ-EMG-109
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func FingerprintVersion(fingerprint string) int {
	m := fingerprintPattern.FindStringSubmatch(strings.TrimSpace(fingerprint))
	switch {
	case m == nil:
		return -1
	case m[1] == "":
		return 0
	}
	v, err := strconv.Atoi(m[1])
	if err != nil {
		return -1
	}
	return v
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
