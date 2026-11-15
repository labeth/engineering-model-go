// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package gemara

// Cross-reference check over a rendered Gemara artifact set. The SDK type
// discriminator only proves each document is a recognizable Gemara type; this
// check proves the documents agree with each other: every reference-id names a
// declared mapping reference that is rendered in the set, and every entry id
// it points at exists in that artifact.

import (
	"fmt"
	"strings"

	gemaraproj "github.com/gemaraproj/go-gemara"
	goyaml "github.com/goccy/go-yaml"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// CrossReferenceError reports one reference that does not resolve within
// the rendered artifact set.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type CrossReferenceError struct {
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

// CheckCrossReferences parses the artifacts (short-name -> YAML) and
// returns every EntryMapping, MultiEntryMapping, artifact import and mapping
// document source/target that does not resolve within the set. Documents that
// fail to parse are reported as errors too.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func CheckCrossReferences(docs map[string]string) []CrossReferenceError {
	var errs []CrossReferenceError
	byID := map[string]*gemaraXrefDoc{}
	var parsed []*gemaraXrefDoc
	for _, name := range textutil.SortedKeys(docs) {
		d, err := parseXrefDoc(name, docs[name])
		if err != nil {
			errs = append(errs, CrossReferenceError{Artifact: name, Message: "parse: " + err.Error()})
			continue
		}
		if d == nil {
			continue
		}
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

// parseXrefDoc parses one artifact; a blank document yields nil.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func parseXrefDoc(name, content string) (*gemaraXrefDoc, error) {
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}
	var root map[string]any
	if err := goyaml.Unmarshal([]byte(content), &root); err != nil {
		return nil, err
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
	return d, nil
}

// collectXrefEntries gathers the ids other artifacts may cite in d: every `id`
// outside metadata, and for an Evaluation Log the control and requirement
// entry ids that key its assessment logs.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func collectXrefEntries(d *gemaraXrefDoc) {
	if d.typ == "EvaluationLog" {
		d.collectEvaluationEntries()
		return
	}
	for k, v := range d.root {
		if k != "metadata" {
			d.walkEntries(v)
		}
	}
}

// collectEvaluationEntries cites the entries of every control evaluation.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (d *gemaraXrefDoc) collectEvaluationEntries() {
	evals, _ := d.root["evaluations"].([]any)
	for _, e := range evals {
		d.addEvaluationEntries(xrefMap(e))
	}
}

// addEvaluationEntries cites the control and the requirement entry ids of one
// control evaluation.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (d *gemaraXrefDoc) addEvaluationEntries(ce map[string]any) {
	d.addEntryOf(ce, "control")
	logs, _ := ce["assessment-logs"].([]any)
	for _, l := range logs {
		d.addEntryOf(xrefMap(l), "requirement")
	}
}

// addEntryOf cites the entry-id of the mapping stored under key.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (d *gemaraXrefDoc) addEntryOf(parent map[string]any, key string) {
	if m, ok := parent[key].(map[string]any); ok {
		d.entries[xrefString(m["entry-id"])] = true
	}
}

// walkEntries cites every `id` below v.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (d *gemaraXrefDoc) walkEntries(v any) {
	switch t := v.(type) {
	case map[string]any:
		if id := xrefString(t["id"]); id != "" {
			d.entries[id] = true
		}
		for _, child := range t {
			d.walkEntries(child)
		}
	case []any:
		for _, child := range t {
			d.walkEntries(child)
		}
	}
}

// xrefChecker resolves the references of one artifact against the rendered set.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type xrefChecker struct {
	d    *gemaraXrefDoc
	byID map[string]*gemaraXrefDoc
	errs []CrossReferenceError
}

// xrefLoc is a reference-id at a path of the artifact.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type xrefLoc struct {
	Path string
	Ref  string
}

// checkXrefDoc walks d's body and resolves each reference it makes. The
// declared ENGMOD-REQUIREMENTS reference of the formal evaluation entries is
// external and is not resolved against the set.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func checkXrefDoc(d *gemaraXrefDoc, byID map[string]*gemaraXrefDoc) []CrossReferenceError {
	c := &xrefChecker{d: d, byID: byID}
	if d.typ == "MappingDocument" {
		c.checkMappingDocument()
		return c.errs
	}
	for _, k := range textutil.SortedKeys(d.root) {
		if k != "metadata" {
			c.walk(k, d.root[k])
		}
	}
	return c.errs
}

// fail records one unresolved reference of the artifact.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c *xrefChecker) fail(e CrossReferenceError) {
	e.Artifact = c.d.name
	c.errs = append(c.errs, e)
}

// target resolves a reference-id to a rendered artifact, reporting why not.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c *xrefChecker) target(path, ref string) *gemaraXrefDoc {
	if ref == c.d.id {
		return c.d
	}
	if !c.d.declared[ref] {
		c.fail(CrossReferenceError{Path: path, ReferenceId: ref, Message: fmt.Sprintf("reference-id %q is not a declared mapping reference", ref)})
		return nil
	}
	t := c.byID[ref]
	if t == nil {
		c.fail(CrossReferenceError{Path: path, ReferenceId: ref, Message: fmt.Sprintf("reference-id %q names no artifact in the rendered set", ref)})
	}
	return t
}

// entry reports an entry id that does not exist in the resolved artifact.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c *xrefChecker) entry(loc xrefLoc, id string, t *gemaraXrefDoc) {
	if t != nil && !t.entries[id] {
		c.fail(CrossReferenceError{Path: loc.Path, ReferenceId: loc.Ref, EntryId: id, Message: fmt.Sprintf("entry %q does not exist in %s", id, t.name)})
	}
}

// checkMappingDocument resolves the source and target entries of a mapping document.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c *xrefChecker) checkMappingDocument() {
	root := c.d.root
	src := c.target("source-reference", xrefString(xrefMap(root["source-reference"])["reference-id"]))
	dst := c.target("target-reference", xrefString(xrefMap(root["target-reference"])["reference-id"]))
	mappings, _ := root["mappings"].([]any)
	for i, m := range mappings {
		mm := xrefMap(m)
		p := fmt.Sprintf("mappings[%d]", i)
		if src != nil {
			c.entry(xrefLoc{Path: p + ".source", Ref: src.id}, xrefString(mm["source"]), src)
		}
		targets, _ := mm["targets"].([]any)
		if dst != nil {
			c.checkMappingTargets(p, targets, dst)
		}
	}
}

// checkMappingTargets resolves the target entries of one mapping.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c *xrefChecker) checkMappingTargets(path string, targets []any, dst *gemaraXrefDoc) {
	for j, tg := range targets {
		c.entry(xrefLoc{Path: fmt.Sprintf("%s.targets[%d]", path, j), Ref: dst.id}, xrefString(xrefMap(tg)["entry-id"]), dst)
	}
}

// walk resolves the references of every mapping below v.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c *xrefChecker) walk(path string, v any) {
	switch t := v.(type) {
	case map[string]any:
		c.checkReference(path, t)
		c.walkChildren(path, t)
	case []any:
		for i, child := range t {
			c.walk(fmt.Sprintf("%s[%d]", path, i), child)
		}
	}
}

// checkReference resolves the reference-id of one mapping object, with its
// entry-id or entries.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c *xrefChecker) checkReference(path string, t map[string]any) {
	ref := xrefString(t["reference-id"])
	if ref == "" {
		return
	}
	_, hasEntry := t["entry-id"]
	entries, hasEntries := t["entries"].([]any)
	switch {
	case ref == gemaraRefRequirements && c.d.declared[ref]:
		// The requirement-keyed formal entries of the Evaluation Log
		// name the model's requirements, which are not a rendered
		// Gemara artifact (gemara_formal.go): a declared external
		// reference, resolved by the model, not by the set.
	case hasEntry:
		c.entry(xrefLoc{Path: path, Ref: ref}, xrefString(t["entry-id"]), c.target(path, ref))
	case hasEntries:
		tg := c.target(path, ref)
		for i, e := range entries {
			c.entry(xrefLoc{Path: fmt.Sprintf("%s.entries[%d]", path, i), Ref: ref}, xrefString(xrefMap(e)["reference-id"]), tg)
		}
	default:
		// A bare artifact mapping (import, criteria) names an artifact.
		c.target(path, ref)
	}
}

// walkChildren walks the members of a mapping object in key order; the entries
// of a mapping were resolved by checkReference.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c *xrefChecker) walkChildren(path string, t map[string]any) {
	_, hasEntries := t["entries"].([]any)
	for _, k := range textutil.SortedKeys(t) {
		if k == "entries" && hasEntries {
			continue
		}
		c.walk(path+"."+k, t[k])
	}
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

// CheckModelConsistency compares the treatment claims in a rendered
// artifact set (short-name -> YAML) with the model they were rendered from: an
// enforcement action's disposition must follow its POA&M item's status (only
// completed is Enforced, only accepted is Tolerated), the log's aggregate must
// follow the actions, the policy may list a risk as mitigated only when it is
// closed and as accepted only when riskAccepted holds (a resolving owner
// acceptance record). It reports each claim the model does not support with
// the same error shape as the cross-reference check.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062, REQ-EMG-085
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func CheckModelConsistency(bundle model.Bundle, docs map[string]string) []CrossReferenceError {
	checks := []struct {
		name  string
		check func(model.Bundle, map[string]any) []CrossReferenceError
	}{
		{"enforcement-log", checkEnforcementConsistency},
		{"policy", checkPolicyConsistency},
	}
	var errs []CrossReferenceError
	for _, c := range checks {
		root, parseErrs := parseConsistencyDoc(c.name, docs[c.name])
		errs = append(errs, parseErrs...)
		if root != nil {
			errs = append(errs, c.check(bundle, root)...)
		}
	}
	return errs
}

// parseConsistencyDoc parses one artifact; a blank one yields no document and
// no error, a parse failure yields the error.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062, REQ-EMG-085
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func parseConsistencyDoc(name, content string) (map[string]any, []CrossReferenceError) {
	var root map[string]any
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}
	if err := goyaml.Unmarshal([]byte(content), &root); err != nil {
		return nil, []CrossReferenceError{{Artifact: name, Message: "parse: " + err.Error()}}
	}
	return root, nil
}

// checkEnforcementConsistency checks the action dispositions and the log
// aggregate against the POA&M items of the model.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062, REQ-EMG-085
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func checkEnforcementConsistency(bundle model.Bundle, root map[string]any) []CrossReferenceError {
	a := bundle.Architecture.AuthoredArchitecture
	actions, _ := root["actions"].([]any)
	items := justifiedPOAMItems(a, evaluationFindingsByRisk(a))
	if len(actions) != len(items) {
		return []CrossReferenceError{{Artifact: "enforcement-log", Path: "actions", Message: fmt.Sprintf("%d actions for %d POA&M items with an evaluated risk in the model", len(actions), len(items))}}
	}
	var errs []CrossReferenceError
	want := make([]gemaraproj.Disposition, 0, len(items))
	for i, p := range items {
		want = append(want, poamDisposition(p.Status))
		if e := actionDispositionMismatch(i, actions[i], p); e != nil {
			errs = append(errs, *e)
		}
	}
	if got, expected := xrefString(root["disposition"]), aggregateDisposition(want); got != expected.String() {
		errs = append(errs, CrossReferenceError{Artifact: "enforcement-log", Path: "disposition", Message: fmt.Sprintf("aggregate disposition %s is not supported by the actions; expected %s", got, expected)})
	}
	return errs
}

// actionDispositionMismatch reports an action whose disposition its POA&M item
// status does not support.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062, REQ-EMG-085
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func actionDispositionMismatch(i int, action any, p model.POAMItem) *CrossReferenceError {
	expected := poamDisposition(p.Status)
	got := xrefString(xrefMap(action)["disposition"])
	if got == expected.String() {
		return nil
	}
	return &CrossReferenceError{
		Artifact: "enforcement-log",
		Path:     fmt.Sprintf("actions[%d].disposition", i),
		EntryId:  p.ID,
		Message:  fmt.Sprintf("POA&M item %s has status %q, which supports disposition %s, not %s", p.ID, textutil.Fallback(p.Status, "not set"), expected, got),
	}
}

// policyBucket is a risk list of the policy and the claim listing a risk in it makes.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type policyBucket struct {
	key   string
	ok    func(model.Risk) bool
	claim string
}

// checkPolicyConsistency checks that every risk the policy lists as mitigated
// or accepted is supported by the model.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062, REQ-EMG-085
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func checkPolicyConsistency(bundle model.Bundle, root map[string]any) []CrossReferenceError {
	refs := validate.NewAcceptanceRefs(bundle)
	risksByID := map[string]model.Risk{}
	for _, r := range bundle.Architecture.AuthoredArchitecture.Risks {
		risksByID[strings.TrimSpace(r.ID)] = r
	}
	risks := xrefMap(root["risks"])
	buckets := []policyBucket{
		{"mitigated", riskMitigated, "mitigated, but its status is not closed"},
		{"accepted", func(r model.Risk) bool { return riskAccepted(r, refs) }, "accepted, but it has no complete, resolving acceptedBy record (existing actor, calendar date, accepted decisionRef if set) with status accepted (or response accept while not open)"},
	}
	var errs []CrossReferenceError
	for _, bucket := range buckets {
		entries, _ := risks[bucket.key].([]any)
		errs = append(errs, unsupportedPolicyRisks(bucket, entries, risksByID)...)
	}
	return errs
}

// unsupportedPolicyRisks reports the listed risks of a bucket the model does
// not support.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062, REQ-EMG-085
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func unsupportedPolicyRisks(bucket policyBucket, entries []any, risksByID map[string]model.Risk) []CrossReferenceError {
	var errs []CrossReferenceError
	for i, e := range entries {
		id := xrefString(xrefMap(xrefMap(e)["risk"])["entry-id"])
		r, known := risksByID[id]
		if known && !bucket.ok(r) {
			errs = append(errs, CrossReferenceError{
				Artifact: "policy",
				Path:     fmt.Sprintf("risks.%s[%d]", bucket.key, i),
				EntryId:  id,
				Message:  fmt.Sprintf("risk %s (status %q) is listed as %s", id, textutil.Fallback(r.Status, "not set"), bucket.claim),
			})
		}
	}
	return errs
}
