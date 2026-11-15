// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// checkRequirements validates each requirement's waivers, fields and targets.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkRequirements() {
	for i, r := range v.b.Requirements.Requirements {
		path := indexedPath("requirements", i)
		v.checkLintWaivers(path, r)
		v.checkRequirementFields(path, r)
		v.checkAppliesTo(path, r)
		v.refList(path, r.SourceRefs, refSpec{ok: v.validID, code: "model.invalid_requirement_source_ref", format: "requirement %q references unknown source %q", field: "sourceRefs", lead: []any{r.ID}})
	}
}

// checkLintWaivers requires each lint waiver to name a lint.* code and give a reason.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkLintWaivers(path string, r model.Requirement) {
	for j, waiver := range r.LintWaivers {
		wpath := indexedPath(path+".lintWaivers", j)
		if !strings.HasPrefix(strings.TrimSpace(waiver.Code), "lint.") {
			v.fail("model.invalid_lint_waiver_code", wpath+".code", fmt.Sprintf("lint waiver code %q must name a lint.* warning", waiver.Code))
		}
		v.requireText(wpath+".reason", waiver.Reason, textSpec{"model.missing_lint_waiver_reason", fmt.Sprintf("lint waiver %q on %q needs a reason", waiver.Code, r.ID)})
	}
}

// checkRequirementFields checks a requirement's rationale, status and verification fields.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkRequirementFields(path string, r model.Requirement) {
	if r.Derived {
		v.requireText(path+".derivedRationale", r.DerivedRationale, textSpec{"model.missing_derived_requirement_rationale", fmt.Sprintf("derived requirement %q must include derivedRationale", r.ID)})
	}
	v.enumValue(path+".status", r.Status, enumSpec{allowedRequirementStatus, "model.invalid_requirement_status", "unknown requirement status %q", nil})
	for j, method := range r.VerificationMethods {
		if !allowedRequirementVerificationMethods[strings.ToLower(strings.TrimSpace(method))] {
			v.fail("model.invalid_requirement_verification_method", indexedPath(path+".verificationMethods", j), fmt.Sprintf("unknown verification method %q", method))
		}
	}
	if len(r.VerificationMethods) > 0 {
		v.requireText(path+".verificationCriteria", r.VerificationCriteria, textSpec{"model.missing_requirement_verification_criteria", fmt.Sprintf("requirement %q with verification methods must include verificationCriteria", r.ID)})
	}
}

// checkAppliesTo requires each appliesTo target of a requirement to be an
// authored id, a CODE id or a catalog, subsystem or contract id; an
// alias::ID entry is checked against the dependencies' publications by
// composition (composition.unpublished_reference).
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkAppliesTo(path string, r model.Requirement) {
	for j, rawTarget := range r.AppliesTo {
		target := strings.TrimSpace(rawTarget)
		if strings.Contains(target, "::") {
			continue
		}
		if target == "" || !(v.validID(target) || v.appliesToTargets[target]) {
			v.fail("requirement.invalid_applies_to", indexedPath(path+".appliesTo", j), fmt.Sprintf("requirement %q appliesTo unknown id %q", r.ID, target))
		}
	}
}

// checkDecisions validates each decision.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkDecisions() {
	for i, d := range v.b.Architecture.Decisions {
		v.checkDecision(indexedPath("decisions", i), d)
	}
}

// checkDecision requires a decision's title, status, date, context, text and consequences.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkDecision(path string, d model.Decision) {
	v.requireText(path, d.Title, textSpec{"model.missing_decision_title", "decision title is required"})
	v.checkDecisionStatus(path, d)
	v.checkDecisionDate(path, d)
	v.requireText(path, d.Context, textSpec{"model.missing_decision_context", "decision context is required"})
	v.requireText(path, d.Decision, textSpec{"model.missing_decision_text", "decision text is required"})
	if len(d.Consequences) == 0 {
		v.fail("model.empty_decision_consequences", path, "decision must include at least one consequence")
	}
	for j, consequence := range d.Consequences {
		v.requireText(indexedPath(path+".consequences", j), consequence, textSpec{"model.empty_decision_consequence", "decision consequence must not be empty"})
	}
}

// checkDecisionStatus requires a decision status from the vocabulary.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkDecisionStatus(path string, d model.Decision) {
	status := strings.ToLower(strings.TrimSpace(d.Status))
	if status == "" {
		v.fail("model.missing_decision_status", path, "decision status is required")
	} else if !allowedDecisionStatus[status] {
		v.fail("model.invalid_decision_status", path, fmt.Sprintf("unknown decision status %q", d.Status))
	}
}

// checkDecisionDate requires a decision date in YYYY-MM-DD form.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkDecisionDate(path string, d model.Decision) {
	date := strings.TrimSpace(d.Date)
	if date == "" {
		v.fail("model.missing_decision_date", path, "decision date is required")
	} else if !isoDateRe.MatchString(date) {
		v.fail("model.invalid_decision_date", path, fmt.Sprintf("invalid decision date %q", d.Date))
	}
}
