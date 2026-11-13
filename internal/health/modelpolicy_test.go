// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// policyFiles is a revision of a model as a map of file name to content.
type policyFiles map[string]string

// with returns a copy of the files with the given files replaced or added.
func (f policyFiles) with(more policyFiles) policyFiles {
	out := policyFiles{}
	for k, v := range f {
		out[k] = v
	}
	for k, v := range more {
		out[k] = v
	}
	return out
}

// policyOfFiles parses the model policy of a revision held in a map.
func policyOfFiles(t *testing.T, f policyFiles) ModelPolicy {
	t.Helper()
	read := func(rel string) ([]byte, error) {
		if s, ok := f[rel]; ok {
			return []byte(s), nil
		}
		return nil, fmt.Errorf("%w: %s", os.ErrNotExist, rel)
	}
	p, err := ParseModelPolicy([]byte(f["engmod.yml"]), read)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

const (
	policyManifestBase = `schemaVersion: 3
requirementLinks: strict
lifecycle:
  tier: operational
  bundles: [foundation, safety]
  tierDecision: DEC-TIER-A
health:
  baseline: {file: model/health-baseline.json, decision: DEC-HEALTH-A}
documents:
  requirements: model/requirements.yml
  assurance: model/assurance.yml
  decisions: model/decisions.yml
`
	policyRequirementsBase = "schemaVersion: 3\nlintRun: {mode: strict}\nrequirements: []\n"
	policyAssuranceBase    = `schemaVersion: 3
assurance:
  poamItems:
  - id: POAM-1
    milestone: tier:operational
    dueDate: 2026-12-31
    status: planned
    defers:
    - {code: lifecycle.requirement_unverified, path: "requirements[1].status"}
  - id: POAM-2
    milestone: later
    dueDate: 2026-12-31
    status: completed
`
	policyDecisionsBase = `schemaVersion: 3
decisions:
- {id: DEC-TIER-A, status: accepted}
- {id: DEC-HEALTH-A, status: accepted}
- {id: ADR-X, status: accepted}
`
	policyBaselineBase = `{"schema": "engmod-health-baseline/1", "entries": [
 {"path": "a.go", "function": "F", "code": "health.complex_method", "value": 20},
 {"path": "a.go", "function": "G", "code": "health.large_method", "value": 90}]}`
)

// baseFiles is a complete model that the cases weaken.
func baseFiles() policyFiles {
	return policyFiles{
		"engmod.yml":                 policyManifestBase,
		"model/requirements.yml":     policyRequirementsBase,
		"model/assurance.yml":        policyAssuranceBase,
		"model/decisions.yml":        policyDecisionsBase,
		"model/health-baseline.json": policyBaselineBase,
	}
}

// edit replaces a text in one file of a model.
type edit struct{ file, old, repl string }

// edited returns a copy of the files with the edit applied.
func (f policyFiles) edited(t *testing.T, e edit) policyFiles {
	t.Helper()
	if !strings.Contains(f[e.file], e.old) {
		t.Fatalf("%s has no %q", e.file, e.old)
	}
	return f.with(policyFiles{e.file: strings.Replace(f[e.file], e.old, e.repl, 1)})
}

// relaxation is a head model with the aspects it must be reported with and a
// text that one of the messages has.
type relaxation struct {
	name  string
	head  policyFiles
	want  []string
	match string
}

// relaxations are the edits of the base model that weaken it, and some that do
// not.
func relaxations(t *testing.T) []relaxation {
	b := baseFiles()
	const deferEntry = `"requirements[1].status"}` + "\n"
	return []relaxation{
		{"unchanged", b, nil, ""},
		{"bundle removed", b.edited(t, edit{"engmod.yml", "[foundation, safety]", "[foundation]"}), []string{PolicyLifecycle}, "no longer has safety"},
		{"tier decision changed at the same tier", b.edited(t, edit{"engmod.yml", "DEC-TIER-A", "DEC-TIER-B"}), []string{PolicyLifecycle}, "DEC-TIER-A to DEC-TIER-B"},
		{"tier decision removed", b.edited(t, edit{"engmod.yml", "  tierDecision: DEC-TIER-A\n", ""}), []string{PolicyLifecycle}, "to none"},
		{"lint relaxed", b.edited(t, edit{"model/requirements.yml", "strict", "guided"}), []string{PolicyLint}, "from strict to guided"},
		{"lint removed", b.edited(t, edit{"model/requirements.yml", "{mode: strict}", "{}"}), []string{PolicyLint}, "from strict to none"},
		{"requirementLinks guided", b.edited(t, edit{"engmod.yml", "requirementLinks: strict", "requirementLinks: guided"}), []string{PolicyRequirementLinks}, "strict to guided"},
		{"requirementLinks removed", b.edited(t, edit{"engmod.yml", "requirementLinks: strict\n", ""}), []string{PolicyRequirementLinks}, "strict to none"},
		{"health block removed", b.edited(t, edit{"engmod.yml", "health:\n  baseline: {file: model/health-baseline.json, decision: DEC-HEALTH-A}\n", ""}), []string{PolicyHealthBlock}, "the health block is removed"},
		{"baseline removed", b.edited(t, edit{"engmod.yml", "  baseline: {file: model/health-baseline.json, decision: DEC-HEALTH-A}\n", "  exclude: [x/**]\n"}), []string{PolicyHealthBlock}, "health.baseline is removed"},
		{"new poam item that defers", b.edited(t, edit{"model/assurance.yml", "  - id: POAM-2\n", "  - id: POAM-3\n    milestone: tier:operational\n    status: planned\n    defers:\n    - {code: lifecycle.x}\n  - id: POAM-2\n"}), []string{PolicyPoam}, "POAM-3 is new and defers"},
		{"new defers entry", b.edited(t, edit{"model/assurance.yml", deferEntry, deferEntry + "    - {code: lifecycle.y}\n"}), []string{PolicyPoam}, "POAM-1 defers"},
		{"due date later", b.edited(t, edit{"model/assurance.yml", "dueDate: 2026-12-31\n    status: planned", "dueDate: 2027-03-31\n    status: planned"}), []string{PolicyPoam}, "moves dueDate from 2026-12-31 to 2027-03-31"},
		{"due date earlier is fine", b.edited(t, edit{"model/assurance.yml", "dueDate: 2026-12-31\n    status: planned", "dueDate: 2026-06-30\n    status: planned"}), nil, ""},
		{"milestone changed", b.edited(t, edit{"model/assurance.yml", "milestone: tier:operational", "milestone: tier:product"}), []string{PolicyPoam}, "changes milestone"},
		{"item without defers may move", b.edited(t, edit{"model/assurance.yml", "milestone: later\n    dueDate: 2026-12-31", "milestone: later\n    dueDate: 2030-01-01"}), nil, ""},
		{"defers removed is fine", b.edited(t, edit{"model/assurance.yml", "    defers:\n    - {code: lifecycle.requirement_unverified, path: " + deferEntry, ""}), nil, ""},
		{"tier decision accepted", b.edited(t, edit{"model/decisions.yml", "- {id: ADR-X, status: accepted}", "- {id: DEC-TIER-B, status: accepted}\n- {id: ADR-X, status: accepted}"}), []string{PolicyDecisionRecord}, "DEC-TIER-B is newly accepted"},
		{"another decision accepted is fine", b.edited(t, edit{"model/decisions.yml", "- {id: ADR-X, status: accepted}", "- {id: ADR-X, status: accepted}\n- {id: ADR-Y, status: accepted}"}), nil, ""},
	}
}

// A model that is unchanged, or only stricter, has no model-level weakening;
// each relaxation of the life-cycle bundles and tier decision, the lint and
// requirement link strictness, the health block and baseline, the POA&M defers
// and the tier and health decisions names its aspect.
//
// Mutation: deleting the call of one of the comparisons in ModelPolicyChanges,
// or comparing the head with itself, fails the matching case.
//
// TRLC-LINKS: REQ-EMG-155
func TestModelPolicyChangesNameEachRelaxation(t *testing.T) {
	base := policyOfFiles(t, baseFiles())
	for _, c := range relaxations(t) {
		got := ModelPolicyChanges(base, policyOfFiles(t, c.head))
		var aspects []string
		for _, ch := range got {
			aspects = append(aspects, ch.Aspect)
		}
		if !reflect.DeepEqual(aspects, c.want) {
			t.Errorf("%s: aspects = %v (%+v), want %v", c.name, aspects, got, c.want)
		}
		if c.match != "" && !anyMessageHas(got, c.match) {
			t.Errorf("%s: no message has %q: %+v", c.name, c.match, got)
		}
	}
}

// anyMessageHas reports whether a message of the changes has the text.
func anyMessageHas(changes []PolicyChange, text string) bool {
	for _, ch := range changes {
		if strings.Contains(ch.Message, text) {
			return true
		}
	}
	return false
}

// A higher tier may change the tier decision; a head with a new module (no
// manifest at the base) weakens nothing.
//
// Mutation: dropping the tier test of the tier decision, or the Present test
// of ModelPolicyChanges, fails.
//
// TRLC-LINKS: REQ-EMG-155
func TestModelPolicyChangesTierUpAndNewModule(t *testing.T) {
	b := baseFiles()
	lower := b.edited(t, edit{"engmod.yml", "tier: operational", "tier: product"})
	up := b.edited(t, edit{"engmod.yml", "DEC-TIER-A", "DEC-TIER-B"})
	if got := ModelPolicyChanges(policyOfFiles(t, lower), policyOfFiles(t, up)); len(got) != 0 {
		t.Errorf("tier decision at a higher tier: %+v", got)
	}
	if got := ModelPolicyChanges(ModelPolicy{}, policyOfFiles(t, b)); len(got) != 0 {
		t.Errorf("new module: %+v", got)
	}
}

// baselineFile is the baseline file of a model with the given entries.
func baselineFile(entries ...model.HealthBaselineEntry) policyFiles {
	data, err := json.Marshal(model.HealthBaselineFile{Schema: model.HealthBaselineSchema, Entries: entries})
	if err != nil {
		panic(err)
	}
	return policyFiles{"model/health-baseline.json": string(data)}
}

// The entries of the base baseline.
var (
	entryF = model.HealthBaselineEntry{Path: "a.go", Function: "F", Code: "health.complex_method", Value: 20}
	entryG = model.HealthBaselineEntry{Path: "a.go", Function: "G", Code: "health.large_method", Value: 90}
)

// The baseline can only shrink: a new finding and a higher value are reported,
// a removed entry or a lower value is not.
//
// Mutation: ignoring the value, or ignoring new entries, fails the matching
// case.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func TestBaselineWeakenings(t *testing.T) {
	b := baseFiles()
	lowerF, higherF := entryF, entryF
	lowerF.Value, higherF.Value = 19, 21
	added := model.HealthBaselineEntry{Path: "b.go", Function: "H", Code: "health.deep_nesting", Value: 5}
	cases := []struct {
		name  string
		head  policyFiles
		want  int
		match string
	}{
		{"unchanged", b, 0, ""},
		{"entry removed", b.with(baselineFile(entryF)), 0, ""},
		{"value lower", b.with(baselineFile(lowerF, entryG)), 0, ""},
		{"all removed", b.with(baselineFile()), 0, ""},
		{"value grown", b.with(baselineFile(higherF, entryG)), 1, "raises the accepted value of 1 finding(s) (health.complex_method F in a.go: 20 to 21)"},
		{"entry added", b.with(baselineFile(entryF, entryG, added)), 1, "accepts 1 finding(s) it did not at the base (health.deep_nesting H in b.go)"},
		{"added and grown", b.with(baselineFile(higherF, entryG, added)), 2, ""},
	}
	base := policyOfFiles(t, b)
	for _, c := range cases {
		got := BaselineWeakenings(base, policyOfFiles(t, c.head))
		if len(got) != c.want || (c.match != "" && !anyMessageHas(got, c.match)) {
			t.Errorf("%s: %+v, want %d change(s) with %q", c.name, got, c.want, c.match)
		}
	}
}

// A baseline that appears is allowed only as the first introduction (the base
// has no health block); a baseline file that is missing at the base has no
// entries; one that is unreadable at the head is not compared.
//
// Mutation: allowing a new baseline whenever the base had none, or treating a
// missing base file as a pass, fails the matching case.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func TestBaselineIntroduction(t *testing.T) {
	b := baseFiles()
	head := policyOfFiles(t, b)
	noHealth := b.edited(t, edit{"engmod.yml", "health:\n  baseline: {file: model/health-baseline.json, decision: DEC-HEALTH-A}\n", ""})
	if got := BaselineWeakenings(policyOfFiles(t, noHealth), head); len(got) != 0 {
		t.Errorf("first introduction (no health block at the base): %+v", got)
	}
	healthOnly := noHealth.edited(t, edit{"engmod.yml", "documents:", "health:\n  exclude: [x/**]\ndocuments:"})
	if got := BaselineWeakenings(policyOfFiles(t, healthOnly), head); len(got) != 1 || !anyMessageHas(got, "introduced") {
		t.Errorf("a baseline added to an existing health block: %+v", got)
	}
	missing := b.with(nil)
	delete(missing, "model/health-baseline.json")
	if got := BaselineWeakenings(policyOfFiles(t, missing), head); len(got) != 1 {
		t.Errorf("baseline file absent at the base: %+v", got)
	}
	bad := b.with(policyFiles{"model/health-baseline.json": "{"})
	if got := BaselineWeakenings(policyOfFiles(t, b), policyOfFiles(t, bad)); len(got) != 0 {
		t.Errorf("unreadable head baseline: %+v", got)
	}
}

// ParseModelPolicy reads what it needs leniently.
//
// TRLC-LINKS: REQ-EMG-155
func TestParseModelPolicy(t *testing.T) {
	p := policyOfFiles(t, baseFiles())
	got := map[string]any{"present": p.Present, "tier": p.Tier, "decision": p.TierDecision, "bundles": p.Bundles,
		"links": p.RequirementLinks, "lint": p.LintMode, "health": p.HasHealth, "baselineRead": p.BaselineRead,
		"baseline": len(p.Baseline), "poam": len(p.Poam), "accepted": p.Accepted}
	want := map[string]any{"present": true, "tier": "operational", "decision": "DEC-TIER-A", "bundles": []string{"foundation", "safety"},
		"links": "strict", "lint": "strict", "health": true, "baselineRead": true, "baseline": 2, "poam": 2,
		"accepted": map[string]bool{"DEC-TIER-A": true, "DEC-HEALTH-A": true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("policy = %+v, want %+v", got, want)
	}
}

// A document that is not there, or whose path leaves the manifest directory,
// contributes nothing; one that is not YAML is an error.
//
// TRLC-LINKS: REQ-EMG-155
func TestParseModelPolicyDocuments(t *testing.T) {
	odd := policyFiles{"engmod.yml": "documents: {requirements: ../x.yml, assurance: /etc/passwd, decisions: model/none.yml}\n"}
	if q := policyOfFiles(t, odd); !q.Present || q.LintMode != "" || len(q.Poam) != 0 {
		t.Errorf("odd documents: %+v", q)
	}
	read := func(string) ([]byte, error) { return []byte("a: [b"), nil }
	if _, err := ParseModelPolicy([]byte("documents: {decisions: d.yml}\n"), read); err == nil {
		t.Error("a decisions file that is not YAML: no error")
	}
	if _, err := ParseModelPolicy([]byte("a: [b"), read); err == nil {
		t.Error("a manifest that is not YAML: no error")
	}
}
