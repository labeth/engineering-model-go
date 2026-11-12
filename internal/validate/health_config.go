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
//   - model.invalid_health_override (error) at an override without a path or
//     with a path the exclude dialect rejects.
//   - model.invalid_health_waiver (error) at a waiver without code, path,
//     function or decision, with a code that is no health smell code or with
//     a path that is absolute or leaves the module root.
//   - health.waiver_unjustified (error, untiered) at a waiver whose decision
//     is no decision or not accepted.
//
// health.waiver_unused needs the report and is emitted by the health package.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func healthDiagnostics(b model.Bundle) []Diagnostic {
	h := b.Manifest.Health
	if h == nil {
		return nil
	}
	var diags []Diagnostic
	add := func(code, path, format string, args ...any) {
		diags = append(diags, Diagnostic{Code: code, Severity: SeverityError, Message: fmt.Sprintf(format, args...), Path: path})
	}
	for i, pattern := range h.Exclude {
		if problem := ExcludePatternProblem(pattern); problem != "" {
			add("model.invalid_health_exclude", fmt.Sprintf("health.exclude[%d]", i),
				"health.exclude pattern %q %s; use %s", pattern, problem, ExcludeAcceptedForms)
		}
	}
	diags = append(diags, healthThresholdDiagnostics("health.thresholds", h.Thresholds)...)
	for i, o := range h.Overrides {
		path := fmt.Sprintf("health.overrides[%d]", i)
		switch problem := ExcludePatternProblem(o.Path); {
		case strings.TrimSpace(o.Path) == "":
			add("model.invalid_health_override", path+".path", "health.overrides[%d] needs a path pattern; use %s", i, ExcludeAcceptedForms)
		case problem != "":
			add("model.invalid_health_override", path+".path", "health.overrides[%d] path %q %s; use %s", i, o.Path, problem, ExcludeAcceptedForms)
		}
		diags = append(diags, healthThresholdDiagnostics(path+".thresholds", o.Thresholds)...)
	}
	refs := NewAcceptanceRefs(b)
	for i, w := range h.Waivers {
		path := fmt.Sprintf("health.waivers[%d]", i)
		for _, f := range []struct{ name, value string }{{"code", w.Code}, {"path", w.Path}, {"function", w.Function}, {"decision", w.Decision}} {
			if strings.TrimSpace(f.value) == "" {
				add("model.invalid_health_waiver", path+"."+f.name, "health.waivers[%d] needs %s (a waiver names code, path, function and decision)", i, f.name)
			}
		}
		if code := strings.TrimSpace(w.Code); code != "" && !model.IsHealthSmellCode(code) {
			add("model.invalid_health_waiver", path+".code", "health.waivers[%d] code %q is not a health smell code; use one of %s", i, w.Code, strings.Join(model.HealthSmellCodes(), ", "))
		}
		if strings.TrimSpace(w.Path) != "" {
			if problem := ExcludePatternProblem(w.Path); problem != "" {
				add("model.invalid_health_waiver", path+".path", "health.waivers[%d] path %q %s; name the file relative to the module root", i, w.Path, problem)
			}
		}
		if strings.TrimSpace(w.Decision) == "" {
			continue
		}
		if problem := waiverDecisionProblem(refs, strings.TrimSpace(w.Decision)); problem != "" {
			add(model.HealthWaiverUnjustifiedCode, path+".decision",
				"health waiver %s for %s in %s is unjustified: decision %s %s; a waiver needs an accepted decision",
				w.Code, w.Function, w.Path, w.Decision, problem)
		}
	}
	return diags
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
