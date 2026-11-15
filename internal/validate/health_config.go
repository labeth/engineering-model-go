// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// healthDiagnostics checks engmod.yml health (ADR-EMG-051, ADR-EMG-052) and
// is the only health rule that needs no report data. A manifest without a
// health block gets no diagnostic at all (REQ-EMG-160).
//
//   - model.invalid_health_exclude (error) at an exclude pattern the
//     inferenceHints.exclude dialect rejects.
//   - model.invalid_health_threshold (error) at a threshold, top-level or in
//     an override, whose key is no smell code with numeric limits, whose warn
//     is below 1 or whose high is below warn.
//   - model.invalid_health_floor (error) at a floor outside 1.0 to 10.0
//     (ADR-EMG-087).
//   - model.invalid_health_override (error) at an override without a path or
//     with a path the exclude dialect rejects.
//   - model.invalid_health_waiver (error) at a waiver without code, path,
//     function or decision, with a code that is no health smell code or with
//     a path that is absolute or leaves the module root.
//   - health.waiver_unjustified (error, untiered) at a waiver whose decision
//     is no decision or not accepted.
//   - health.waiver_unjustified (error, untiered) at a policyDecision that is
//     no decision or not accepted; the message names the policy decision.
//   - the baseline rules of healthBaselineDiagnostics (ADR-EMG-059).
//
// health.waiver_unused and health.baseline_stale need the report and are
// emitted by the health package.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func healthDiagnostics(b model.Bundle) []Diagnostic {
	h := b.Manifest.Health
	if h == nil {
		return nil
	}
	diags := healthExcludeDiagnostics(h)
	diags = append(append(diags, healthFloorDiagnostics(h)...), healthThresholdDiagnostics("health.thresholds", h.Thresholds)...)
	diags = append(diags, healthOverrideDiagnostics(h)...)
	refs := NewAcceptanceRefs(b)
	diags = append(diags, healthWaiverDiagnostics(h, refs)...)
	diags = append(diags, healthPolicyDecisionDiagnostics(h, refs)...)
	return append(diags, healthBaselineDiagnostics(b, refs)...)
}

// healthExcludeDiagnostics rejects the exclude patterns the inferenceHints.exclude dialect rejects.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func healthExcludeDiagnostics(h *model.ManifestHealth) []Diagnostic {
	var s diagSink
	for i, pattern := range h.Exclude {
		if problem := ExcludePatternProblem(pattern); problem != "" {
			s.fail("model.invalid_health_exclude", indexedPath("health.exclude", i),
				fmt.Sprintf("health.exclude pattern %q %s; use %s", pattern, problem, ExcludeAcceptedForms))
		}
	}
	return s.diags
}

// healthOverrideDiagnostics requires a path pattern of each override and checks its thresholds.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func healthOverrideDiagnostics(h *model.ManifestHealth) []Diagnostic {
	var s diagSink
	for i, o := range h.Overrides {
		path := indexedPath("health.overrides", i)
		switch problem := ExcludePatternProblem(o.Path); {
		case strings.TrimSpace(o.Path) == "":
			s.fail("model.invalid_health_override", path+".path", fmt.Sprintf("%s needs a path pattern; use %s", path, ExcludeAcceptedForms))
		case problem != "":
			s.fail("model.invalid_health_override", path+".path", fmt.Sprintf("%s path %q %s; use %s", path, o.Path, problem, ExcludeAcceptedForms))
		}
		s.diags = append(s.diags, healthThresholdDiagnostics(path+".thresholds", o.Thresholds)...)
	}
	return s.diags
}

// healthWaiverDiagnostics checks each waiver of the health block.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func healthWaiverDiagnostics(h *model.ManifestHealth, refs AcceptanceRefs) []Diagnostic {
	var diags []Diagnostic
	for i, w := range h.Waivers {
		diags = append(diags, waiverDiagnostics(indexedPath("health.waivers", i), w, refs)...)
	}
	return diags
}

// waiverDiagnostics requires a waiver to name code, path, function and an accepted decision.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func waiverDiagnostics(path string, w model.HealthWaiver, refs AcceptanceRefs) []Diagnostic {
	var s diagSink
	for _, f := range []struct{ name, value string }{{"code", w.Code}, {"path", w.Path}, {"function", w.Function}, {"decision", w.Decision}} {
		if strings.TrimSpace(f.value) == "" {
			s.fail("model.invalid_health_waiver", path+"."+f.name, fmt.Sprintf("%s needs %s (a waiver names code, path, function and decision)", path, f.name))
		}
	}
	s.waiverCodeAndPath(path, w)
	s.waiverDecision(path, w, refs)
	return s.diags
}

// waiverCodeAndPath rejects a waiver code that is no health smell code and a path the exclude dialect rejects.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s *diagSink) waiverCodeAndPath(path string, w model.HealthWaiver) {
	if code := strings.TrimSpace(w.Code); code != "" && !model.IsHealthSmellCode(code) {
		s.fail("model.invalid_health_waiver", path+".code", fmt.Sprintf("%s code %q is not a health smell code; use one of %s", path, w.Code, strings.Join(model.HealthSmellCodes(), ", ")))
	}
	if strings.TrimSpace(w.Path) == "" {
		return
	}
	if problem := ExcludePatternProblem(w.Path); problem != "" {
		s.fail("model.invalid_health_waiver", path+".path", fmt.Sprintf("%s path %q %s; name the file relative to the module root", path, w.Path, problem))
	}
}

// waiverDecision requires the decision of a waiver, when it names one, to be accepted.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s *diagSink) waiverDecision(path string, w model.HealthWaiver, refs AcceptanceRefs) {
	decision := strings.TrimSpace(w.Decision)
	if decision == "" {
		return
	}
	if problem := waiverDecisionProblem(refs, decision); problem != "" {
		s.fail(model.HealthWaiverUnjustifiedCode, path+".decision",
			fmt.Sprintf("health waiver %s for %s in %s is unjustified: decision %s %s; a waiver needs an accepted decision",
				w.Code, w.Function, w.Path, w.Decision, problem))
	}
}

// healthPolicyDecisionDiagnostics requires the policyDecision of the health block, when set, to be accepted.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func healthPolicyDecisionDiagnostics(h *model.ManifestHealth, refs AcceptanceRefs) []Diagnostic {
	decision := strings.TrimSpace(h.PolicyDecision)
	if decision == "" {
		return nil
	}
	var s diagSink
	if problem := waiverDecisionProblem(refs, decision); problem != "" {
		s.fail(model.HealthWaiverUnjustifiedCode, "health.policyDecision",
			fmt.Sprintf("health policy decision %s is unjustified: it %s; a recorded policy weakening needs an accepted decision", decision, problem))
	}
	return s.diags
}

// AcceptedHealthPolicyDecision returns the policyDecision of the health block
// when it names an accepted decision, else "". A policyDecision that does not
// exist or is not accepted records nothing (health.waiver_unjustified reports
// it in validation), so a weakening of the policy it was meant to cover stays
// a health.policy_weakened finding.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func AcceptedHealthPolicyDecision(b model.Bundle) string {
	h := b.Manifest.Health
	if h == nil {
		return ""
	}
	decision := strings.TrimSpace(h.PolicyDecision)
	if decision == "" || waiverDecisionProblem(NewAcceptanceRefs(b), decision) != "" {
		return ""
	}
	return decision
}

// HealthBaselineDecisionProblem returns why the decision of health.baseline
// cannot justify the baseline (it is empty, does not exist or is not
// accepted), or "" when it can or the manifest has no baseline. The scan calls
// it so that a baseline without an accepted decision suppresses nothing.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-161
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func HealthBaselineDecisionProblem(b model.Bundle) string {
	h := b.Manifest.Health
	if h == nil || h.Baseline == nil {
		return ""
	}
	decision := strings.TrimSpace(h.Baseline.Decision)
	if decision == "" {
		return "has no decision"
	}
	if problem := waiverDecisionProblem(NewAcceptanceRefs(b), decision); problem != "" {
		return "decision " + decision + " " + problem
	}
	return ""
}

// healthFloorDiagnostics checks health.floor: a number from the lowest to the
// highest file score (ADR-EMG-087), else model.invalid_health_floor.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-209
func healthFloorDiagnostics(h *model.ManifestHealth) []Diagnostic {
	if h.Floor == nil || (*h.Floor >= model.MinHealthFloor && *h.Floor <= model.DefaultHealthFloor) {
		return nil
	}
	return []Diagnostic{{Code: "model.invalid_health_floor", Severity: SeverityError, Path: "health.floor",
		Message: fmt.Sprintf("health.floor %v is outside %.1f to %.1f, the lowest and the highest file score; the floor is %.1f unless the manifest lowers it (a lower floor is a weakening that enghealth check reports as health.policy_weakened)",
			*h.Floor, model.MinHealthFloor, model.DefaultHealthFloor, model.DefaultHealthFloor)}}
}

// healthBaselineDiagnostics checks health.baseline (ADR-EMG-059):
//
//   - model.invalid_health_baseline (error) at a missing file or decision and
//     at a file path that is absolute, leaves the manifest directory or is not
//     a file name.
//   - health.waiver_unjustified (error, untiered, the code of a waiver) at a
//     baseline whose decision is no decision or not accepted; the message
//     names the baseline.
//   - health.baseline_unreadable (error, untiered) when the file, read
//     relative to the manifest directory, is missing, is not valid JSON, is not
//     engmod-health-baseline/1 or holds an invalid entry. The file format
//     lives in internal/model, so this layer reads it without importing the
//     health package; which entries are stale needs a scan and is reported
//     by the health package (health.baseline_stale). A bundle without a
//     manifest path (built in memory) has no directory to read from, so only
//     the manifest fields are checked.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-161
func healthBaselineDiagnostics(b model.Bundle, refs AcceptanceRefs) []Diagnostic {
	base := b.Manifest.Health.Baseline
	if base == nil {
		return nil
	}
	var s diagSink
	file := strings.TrimSpace(base.File)
	fileProblem := model.HealthBaselinePathProblem(file)
	if fileProblem != "" {
		s.fail("model.invalid_health_baseline", "health.baseline.file", fmt.Sprintf("health.baseline file %q %s; name a file relative to the manifest directory", base.File, fileProblem))
	}
	s.baselineDecision(base, refs)
	if fileProblem == "" && b.ManifestPath != "" {
		s.baselineReadable(b.ManifestPath, base)
	}
	return s.diags
}

// baselineDecision requires the baseline to name an accepted decision.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-161
func (s *diagSink) baselineDecision(base *model.HealthBaseline, refs AcceptanceRefs) {
	decision := strings.TrimSpace(base.Decision)
	if decision == "" {
		s.fail("model.invalid_health_baseline", "health.baseline.decision", "health.baseline needs a decision (an accepted decision that justifies the baseline)")
	} else if problem := waiverDecisionProblem(refs, decision); problem != "" {
		s.fail(model.HealthWaiverUnjustifiedCode, "health.baseline.decision",
			fmt.Sprintf("health baseline %s is unjustified: decision %s %s; a baseline needs an accepted decision", base.File, base.Decision, problem))
	}
}

// baselineReadable reports a baseline file that cannot be read from the manifest directory.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-161
func (s *diagSink) baselineReadable(manifestPath string, base *model.HealthBaseline) {
	if _, err := model.LoadHealthBaseline(manifestPath, strings.TrimSpace(base.File)); err != nil {
		s.fail(model.HealthBaselineUnreadableCode, "health.baseline.file", fmt.Sprintf("health baseline %s is unreadable: %s", base.File, strings.TrimPrefix(err.Error(), model.ErrHealthBaseline.Error()+": ")))
	}
}

// waiverDecisionProblem returns why a decision cannot justify a waiver, or ""
// when it exists and is accepted (the test acceptance records apply to
// decisionRef).
// TRLC-LINKS: REQ-EMG-154
func waiverDecisionProblem(refs AcceptanceRefs, decision string) string {
	status, ok := refs.decisions[decision]
	switch {
	case !ok:
		return "does not exist"
	case status != "accepted":
		return fmt.Sprintf("has status %q, not accepted", status)
	}
	return ""
}

// healthThresholdDiagnostics checks one thresholds map in sorted key order.
// TRLC-LINKS: REQ-EMG-153
func healthThresholdDiagnostics(path string, thresholds map[string]model.HealthThreshold) []Diagnostic {
	keys := make([]string, 0, len(thresholds))
	for code := range thresholds {
		keys = append(keys, code)
	}
	sort.Strings(keys)
	var diags []Diagnostic
	for _, code := range keys {
		t := thresholds[code]
		at := fmt.Sprintf("%s[%q]", path, code)
		add := func(format string, args ...any) {
			diags = append(diags, Diagnostic{Code: "model.invalid_health_threshold", Severity: SeverityError, Message: fmt.Sprintf(format, args...), Path: at})
		}
		switch {
		case !model.IsHealthThresholdCode(code):
			add("%s threshold code %q takes no numeric limits; use one of %s", path, code, strings.Join(model.HealthThresholdCodes(), ", "))
		case t.Warn < 1:
			add("%s %q warn %d must be at least 1", path, code, t.Warn)
		case t.High < t.Warn:
			add("%s %q high %d must be at least warn %d", path, code, t.High, t.Warn)
		}
	}
	return diags
}

// AcceptedHealthWaivers keeps the waivers of the manifest health block whose
// decision exists and is accepted. A waiver with any other decision waives
// nothing (health.waiver_unjustified reports it in validation), so a waiver
// that was proposed but not yet accepted cannot hide a finding in a scan or a
// delta check.
//
// TRLC-LINKS: REQ-EMG-154
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func AcceptedHealthWaivers(b model.Bundle, waivers []model.HealthWaiver) []model.HealthWaiver {
	refs := NewAcceptanceRefs(b)
	kept := make([]model.HealthWaiver, 0, len(waivers))
	for _, w := range waivers {
		decision := strings.TrimSpace(w.Decision)
		if decision != "" && waiverDecisionProblem(refs, decision) == "" {
			kept = append(kept, w)
		}
	}
	return kept
}

// CodeExemptionDecisionProblem returns why the decision of a codeCoverage
// exemption cannot justify it (it is empty, does not exist or is not
// accepted), or "" when it can. It is the rule of a health waiver decision.
//
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func CodeExemptionDecisionProblem(b model.Bundle, decision string) string {
	decision = strings.TrimSpace(decision)
	if decision == "" {
		return "has no decision"
	}
	return waiverDecisionProblem(NewAcceptanceRefs(b), decision)
}
