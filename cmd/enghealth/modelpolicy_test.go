// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"fmt"
	"strings"
	"testing"
)

// modelPolicyRepo is a committed project at tier product with a strict lint
// mode, a health block with a baseline of two findings and a POA&M item that
// defers a diagnostic.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func modelPolicyRepo(t *testing.T) *checkRepo {
	t.Helper()
	const extra = "requirementLinks: strict\nlifecycle:\n  tier: product\n  bundles: [foundation]\n  tierDecision: ADR-POLICY\n" +
		"health:\n  baseline:\n    file: model/health-baseline.json\n    decision: ADR-DEBT\n"
	r := newCheckRepo(t, extra, false)
	r.write("model/requirements.yml", "schemaVersion: 2\nlintRun: {mode: strict}\nrequirements:\n  - id: REQ-X\n    text: The unit shall decide.\n")
	r.write("model/health-baseline.json", baselineJSON(20, 90))
	r.write("model/assurance.yml", "schemaVersion: 2\nassurance:\n  poamItems:\n  - id: POAM-1\n    milestone: tier:product\n    dueDate: 2026-12-31\n    status: planned\n    defers:\n    - {code: lifecycle.x}\n")
	r.commit("base")
	return r
}

// baselineJSON is a baseline of two findings with the given values.
func baselineJSON(f, g int) string {
	return fmt.Sprintf(`{"schema": "engmod-health-baseline/1", "entries": [
 {"path": "pkg/svc.go", "function": "F", "code": "health.complex_method", "value": %d},
 {"path": "pkg/svc.go", "function": "G", "code": "health.large_method", "value": %d}]}
`, f, g)
}

// modelEdit is an edit of a file of the project that weakens the model policy,
// with the aspect it is reported as.
type modelEdit struct{ file, old, repl, aspect string }

// modelEdits are the weakenings the cases make.
var modelEdits = map[string]modelEdit{
	"lint relaxed":            {"model/requirements.yml", "mode: strict", "mode: guided", "lint"},
	"requirementLinks guided": {"engmod.yml", "requirementLinks: strict", "requirementLinks: guided", "requirementLinks"},
	"tier decision swapped":   {"engmod.yml", "tierDecision: ADR-POLICY", "tierDecision: ADR-W", "lifecycle"},
	"deferral added":          {"model/assurance.yml", "    - {code: lifecycle.x}\n", "    - {code: lifecycle.x}\n    - {code: lifecycle.y}\n", "poam"},
	"due date moved":          {"model/assurance.yml", "2026-12-31", "2027-12-31", "poam"},
	"tier decision accepted":  {"model/decisions.yml", "decisions:\n", "decisions:\n  - {id: DEC-TIER-NEW, title: t, status: accepted, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n", "decision"},
}

// onlyWeakening reports whether a check document holds exactly one weakening of
// an aspect, an error at the tier.
func onlyWeakening(doc checkJSON, aspect string) bool {
	return len(doc.PolicyWeakened) == 1 && doc.PolicyWeakened[0].Aspect == aspect && doc.PolicyLevel == "error"
}

// The model-level policy is compared with the base revision: a weaker lint
// mode, requirementLinks, tier decision, a new POA&M deferral, a later due
// date and a newly accepted tier decision each make the check exit 1 at
// product with health.policy_weakened naming the aspect; the same edit with an
// accepted policyDecision of the head exits 0 and still lists the weakening.
//
// Mutation: not calling modelPolicyChanges from policyVerdict fails every case.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158
func TestCheckReportsAWeakenedModelPolicy(t *testing.T) {
	for name, e := range modelEdits {
		r := modelPolicyRepo(t)
		r.setFile(e.file, e.old, e.repl)
		wantUnrecordedWeakening(r, name, e.aspect)
		r.setFile("engmod.yml", "health:\n", "health:\n  policyDecision: ADR-W\n")
		if code, doc, stderr := r.jsonCheck(); code != exitOK || doc.PolicyDecision != "ADR-W" || len(doc.PolicyWeakened) == 0 {
			t.Errorf("%s with a policyDecision: exit %d %+v %q", name, code, doc, stderr)
		}
	}
}

// wantUnrecordedWeakening checks that the check of the project exits 1 with one
// weakening of the aspect, in the JSON and in the text.
func wantUnrecordedWeakening(r *checkRepo, name, aspect string) {
	t := r.t
	t.Helper()
	code, doc, stderr := r.jsonCheck()
	if code != exitFail || !doc.Failed || !onlyWeakening(doc, aspect) {
		t.Errorf("%s: exit %d, %+v, stderr %q; want 1 with one %s weakening", name, code, doc.PolicyWeakened, stderr, aspect)
	}
	if _, stdout, _ := r.check(); !strings.Contains(stdout, "policy weakened ("+aspect+")") {
		t.Errorf("%s: text lacks the weakening:\n%s", name, stdout)
	}
}

// A removed health block is reported; an unchanged model passes.
//
// Mutation: not calling healthBlockWeakenings fails the first case.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func TestCheckHealthBlockRemovedAndUnchanged(t *testing.T) {
	r := modelPolicyRepo(t)
	r.setFile("engmod.yml", "health:\n  baseline:\n    file: model/health-baseline.json\n    decision: ADR-DEBT\n", "")
	if code, doc, _ := r.jsonCheck(); code != exitFail || !onlyWeakening(doc, "health") {
		t.Errorf("health block removed: exit %d %+v", code, doc.PolicyWeakened)
	}
	r = modelPolicyRepo(t)
	if code, doc, stderr := r.jsonCheck(); code != exitOK || len(doc.PolicyWeakened) != 0 {
		t.Errorf("unchanged: exit %d %+v %q", code, doc.PolicyWeakened, stderr)
	}
}

// baselineEntryJSON is one entry of a baseline file.
func baselineEntryJSON(fn, code string, value int) string {
	return fmt.Sprintf(`{"path": "pkg/svc.go", "function": %q, "code": %q, "value": %d}`, fn, code, value)
}

// baselineOf is a baseline file with the given entries.
func baselineOf(entries ...string) string {
	return `{"schema": "engmod-health-baseline/1", "entries": [` + strings.Join(entries, ",") + "]}\n"
}

// A baseline that grows fails the check (health.policy_weakened, aspect
// baseline): a value above the base's, a finding the base did not have. One that
// shrinks passes.
//
// Mutation: not calling BaselineWeakenings, or ignoring the value, fails the
// matching case.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func TestCheckBaselineCanOnlyShrink(t *testing.T) {
	cases := []struct {
		name, file, match string
	}{
		{"value grown", baselineOf(baselineEntryJSON("F", "health.complex_method", 21), baselineEntryJSON("G", "health.large_method", 90)), "20 to 21"},
		{"entry added", baselineOf(baselineEntryJSON("F", "health.complex_method", 20), baselineEntryJSON("G", "health.large_method", 90), baselineEntryJSON("H", "health.deep_nesting", 5)), "health.deep_nesting H in pkg/svc.go"},
		{"shrunk", baselineOf(baselineEntryJSON("F", "health.complex_method", 19)), ""},
	}
	for _, c := range cases {
		r := modelPolicyRepo(t)
		r.write("model/health-baseline.json", c.file)
		code, doc, stderr := r.jsonCheck()
		if !baselineVerdictIs(code, doc, c.match) {
			t.Errorf("%s: exit %d %+v %q", c.name, code, doc.PolicyWeakened, stderr)
		}
	}
}

// baselineVerdictIs reports whether a check passed without a weakening (match
// empty) or failed with one baseline weakening whose message has match.
func baselineVerdictIs(code int, doc checkJSON, match string) bool {
	if match == "" {
		return code == exitOK && len(doc.PolicyWeakened) == 0
	}
	return code == exitFail && onlyWeakening(doc, "baseline") && strings.Contains(doc.PolicyWeakened[0].Message, match)
}

// A baseline that is new where the base had no health block is the first
// introduction and passes; one added to an existing health block fails.
//
// Mutation: letting a new baseline pass whenever the base had none fails the
// second case.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func TestCheckBaselineFirstIntroduction(t *testing.T) {
	const intro = "lifecycle:\n  tier: product\n  bundles: [foundation]\n  tierDecision: ADR-POLICY\n"
	const block = "  baseline:\n    file: model/health-baseline.json\n    decision: ADR-DEBT\n"
	r := newCheckRepo(t, intro, true)
	r.setFile("engmod.yml", "documents:", "health:\n"+block+"documents:")
	r.write("model/health-baseline.json", baselineJSON(20, 90))
	if code, doc, stderr := r.jsonCheck(); code != exitOK || len(doc.PolicyWeakened) != 0 {
		t.Errorf("first introduction: exit %d %+v %q", code, doc.PolicyWeakened, stderr)
	}
	r = newCheckRepo(t, intro+"health:\n  exclude: [gen/**]\n", true)
	r.setFile("engmod.yml", "  exclude: [gen/**]\n", "  exclude: [gen/**]\n"+block)
	r.write("model/health-baseline.json", baselineJSON(20, 90))
	if code, doc, _ := r.jsonCheck(); code != exitFail || !onlyWeakening(doc, "baseline") {
		t.Errorf("baseline added to an existing health block: exit %d %+v", code, doc.PolicyWeakened)
	}
}

// With --base and --head the model policy is read at the head revision, not
// from the working tree.
//
// Mutation: reading the head side from the working tree fails.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func TestCheckModelPolicyAtAHeadRevision(t *testing.T) {
	r := modelPolicyRepo(t)
	base := strings.TrimSpace(gitIn(t, r.dir, "rev-parse", "HEAD"))
	r.setFile("model/requirements.yml", "mode: strict", "mode: guided")
	r.commit("relax")
	// The working tree is restored, the commit still relaxes.
	r.setFile("model/requirements.yml", "mode: guided", "mode: strict")
	code, doc, _ := r.jsonCheck("--base", base, "--head", "HEAD")
	if code != exitFail || !onlyWeakening(doc, "lint") {
		t.Errorf("head revision: exit %d %+v", code, doc.PolicyWeakened)
	}
}
