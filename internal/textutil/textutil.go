// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION

// Package textutil holds small, model-independent string and collection
// helpers shared by the generators.
package textutil

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// NonEmpty returns v unless it is blank (whitespace only), then fallback.
// v is returned untrimmed.
// TRLC-LINKS: REQ-EMG-003
func NonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// Fallback returns s unless it is blank (whitespace only), then def.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func Fallback(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// UniqueSorted trims each value, drops blanks and duplicates and returns the
// rest sorted. The result is never nil.
// TRLC-LINKS: REQ-EMG-003
func UniqueSorted(in []string) []string {
	set := map[string]bool{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			set[s] = true
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// AppendUnique appends the trimmed x to in unless it is blank or already
// present. It may share in's backing array.
// TRLC-LINKS: REQ-EMG-010
func AppendUnique(in []string, x string) []string {
	x = strings.TrimSpace(x)
	if x == "" {
		return in
	}
	for _, e := range in {
		if e == x {
			return in
		}
	}
	return append(in, x)
}

// AppendUniqueSlice returns a copy of in with each item appended by AppendUnique.
// TRLC-LINKS: REQ-EMG-010
func AppendUniqueSlice(in []string, items []string) []string {
	out := append([]string(nil), in...)
	for _, x := range items {
		out = AppendUnique(out, x)
	}
	return out
}

// SortedKeys returns map keys in deterministic order (used by serialization helpers).
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func SortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// DeterministicUUID derives a version-5-style UUID string from the SHA-1 of seed.
// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func DeterministicUUID(seed string) string {
	h := sha1.Sum([]byte(seed))
	b := h[:16]
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80
	x := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", x[0:8], x[8:12], x[12:16], x[16:20], x[20:32])
}
