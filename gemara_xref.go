// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

// Cross-reference check over a rendered Gemara artifact set. The SDK type
// discriminator only proves each document is a recognizable Gemara type; this
// check proves the documents agree with each other: every reference-id names a
// declared mapping reference that is rendered in the set, and every entry id
// it points at exists in that artifact.

import (
	"fmt"
	"strings"

	goyaml "github.com/goccy/go-yaml"
)

// GemaraCrossReferenceError reports one reference that does not resolve within
// the rendered artifact set.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type GemaraCrossReferenceError struct {
	Artifact    string `json:"artifact"`
	Path        string `json:"path"`
	ReferenceId string `json:"referenceId"`
	EntryId     string `json:"entryId,omitempty"`
	Message     string `json:"message"`
}

// gemaraXrefDoc is one parsed artifact with its resolvable ids.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type gemaraXrefDoc struct {
	name     string
	id       string
	typ      string
	root     map[string]any
	declared map[string]bool
	entries  map[string]bool
}

// CheckGemaraCrossReferences parses the artifacts (short-name -> YAML) and
// returns every EntryMapping, MultiEntryMapping, artifact import and mapping
// document source/target that does not resolve within the set. Documents that
// fail to parse are reported as errors too.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func CheckGemaraCrossReferences(docs map[string]string) []GemaraCrossReferenceError {
	var errs []GemaraCrossReferenceError
	byID := map[string]*gemaraXrefDoc{}
	var parsed []*gemaraXrefDoc
	for _, name := range sortedKeys(docs) {
		content := docs[name]
		if strings.TrimSpace(content) == "" {
			continue
		}
		var root map[string]any
		if err := goyaml.Unmarshal([]byte(content), &root); err != nil {
			errs = append(errs, GemaraCrossReferenceError{Artifact: name, Message: "parse: " + err.Error()})
			continue
		}
		d := &gemaraXrefDoc{name: name, root: root, declared: map[string]bool{}, entries: map[string]bool{}}
		meta, _ := root["metadata"].(map[string]any)
		d.id = xrefString(meta["id"])
		d.typ = xrefString(meta["type"])
		refs, _ := meta["mapping-references"].([]any)
		for _, r := range refs {
			if m, ok := r.(map[string]any); ok {
				d.declared[xrefString(m["id"])] = true
			}
		}
		collectXrefEntries(d)
		parsed = append(parsed, d)
		if d.id != "" {
			byID[d.id] = d
		}
	}
	for _, d := range parsed {
		errs = append(errs, checkXrefDoc(d, byID)...)
	}
	return errs
}

// collectXrefEntries gathers the ids other artifacts may cite in d: every `id`
// outside metadata, and for an Evaluation Log the control and requirement
// entry ids that key its assessment logs.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func collectXrefEntries(d *gemaraXrefDoc) {
	if d.typ == "EvaluationLog" {
		evals, _ := d.root["evaluations"].([]any)
		for _, e := range evals {
			ce, _ := e.(map[string]any)
			if c, ok := ce["control"].(map[string]any); ok {
				d.entries[xrefString(c["entry-id"])] = true
			}
			logs, _ := ce["assessment-logs"].([]any)
			for _, l := range logs {
				al, _ := l.(map[string]any)
				if r, ok := al["requirement"].(map[string]any); ok {
					d.entries[xrefString(r["entry-id"])] = true
				}
			}
		}
		return
	}
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if id := xrefString(t["id"]); id != "" {
				d.entries[id] = true
			}
			for _, child := range t {
				walk(child)
			}
		case []any:
			for _, child := range t {
				walk(child)
			}
		}
	}
	for k, v := range d.root {
		if k != "metadata" {
			walk(v)
		}
	}
}

// checkXrefDoc walks d's body and resolves each reference it makes.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func checkXrefDoc(d *gemaraXrefDoc, byID map[string]*gemaraXrefDoc) []GemaraCrossReferenceError {
	var errs []GemaraCrossReferenceError
	fail := func(path, ref, entry, msg string) {
		errs = append(errs, GemaraCrossReferenceError{Artifact: d.name, Path: path, ReferenceId: ref, EntryId: entry, Message: msg})
	}
	// target resolves a reference-id to a rendered artifact, reporting why not.
	target := func(path, ref string) *gemaraXrefDoc {
		if ref == d.id {
			return d
		}
		if !d.declared[ref] {
			fail(path, ref, "", fmt.Sprintf("reference-id %q is not a declared mapping reference", ref))
			return nil
		}
		t := byID[ref]
		if t == nil {
			fail(path, ref, "", fmt.Sprintf("reference-id %q names no artifact in the rendered set", ref))
		}
		return t
	}
	entry := func(path, ref, id string, t *gemaraXrefDoc) {
		if t != nil && !t.entries[id] {
			fail(path, ref, id, fmt.Sprintf("entry %q does not exist in %s", id, t.name))
		}
	}

	if d.typ == "MappingDocument" {
		src := target("source-reference", xrefString(xrefMap(d.root["source-reference"])["reference-id"]))
		dst := target("target-reference", xrefString(xrefMap(d.root["target-reference"])["reference-id"]))
		mappings, _ := d.root["mappings"].([]any)
		for i, m := range mappings {
			mm := xrefMap(m)
			p := fmt.Sprintf("mappings[%d]", i)
			if src != nil {
				entry(p+".source", src.id, xrefString(mm["source"]), src)
			}
			targets, _ := mm["targets"].([]any)
			for j, tg := range targets {
				if dst != nil {
					entry(fmt.Sprintf("%s.targets[%d]", p, j), dst.id, xrefString(xrefMap(tg)["entry-id"]), dst)
				}
			}
		}
		return errs
	}

	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch t := v.(type) {
		case map[string]any:
			ref := xrefString(t["reference-id"])
			_, hasEntry := t["entry-id"]
			entries, hasEntries := t["entries"].([]any)
			switch {
			case ref != "" && hasEntry:
				entry(path, ref, xrefString(t["entry-id"]), target(path, ref))
			case ref != "" && hasEntries:
				tg := target(path, ref)
				for i, e := range entries {
					entry(fmt.Sprintf("%s.entries[%d]", path, i), ref, xrefString(xrefMap(e)["reference-id"]), tg)
				}
			case ref != "":
				// A bare artifact mapping (import, criteria) names an artifact.
				target(path, ref)
			}
			for _, k := range sortedKeys(t) {
				if k == "entries" && hasEntries {
					continue
				}
				walk(path+"."+k, t[k])
			}
		case []any:
			for i, child := range t {
				walk(fmt.Sprintf("%s[%d]", path, i), child)
			}
		}
	}
	for _, k := range sortedKeys(d.root) {
		if k != "metadata" {
			walk(k, d.root[k])
		}
	}
	return errs
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func xrefMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func xrefString(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
