// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// assuranceLinkDiagnostics reports assurance records whose verification links
// do not agree:
//
//   - threat.verification_missing (warning): a threat scenario whose
//     verificationRefs omit a control verification that one of its mitigations
//     (threatScenarioRef) lists in verificationRefs, or that names the threat
//     in threatScenarioRefs. Reported once at the threat's verificationRefs.
//   - mitigation.verification_control_mismatch (error): a mitigation
//     verificationRefs entry naming a control verification whose controlRef is
//     another control than the mitigation's controlRef.
//   - verification.unlinked (warning): a control verification with method
//     test and status pass or partial but no verificationRefs, so no test or
//     requirement evidences it and the trace cannot report its tests.
//   - gemara.control_needs_review (warning): a control that the Gemara
//     evaluation log gives the result Needs Review (ControlNeedsReview): no
//     control verification names it in controlRef, or its verifications need
//     review (for example partial) and none failed or is unknown. The message
//     names any verification of another control that a mitigation of this
//     control relies on, or the verifications that need review.
//
// Imported alias::ID records belong to their own model and are skipped.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func assuranceLinkDiagnostics(a model.AuthoredArchitecture) []Diagnostic {
	var diags []Diagnostic
	imported := func(id string) bool { return strings.Contains(id, "::") }
	cvControl := map[string]string{}
	statuses := map[string][]string{}
	reviewIDs := map[string][]string{}
	for _, cv := range a.ControlVerifications {
		id, ctrl := strings.TrimSpace(cv.ID), strings.TrimSpace(cv.ControlRef)
		cvControl[id] = ctrl
		statuses[ctrl] = append(statuses[ctrl], cv.Status)
		if ClassifyVerificationStatus(cv.Status) == OutcomeNeedsReview {
			reviewIDs[ctrl] = append(reviewIDs[ctrl], fmt.Sprintf("%s (%s)", id, strings.TrimSpace(cv.Status)))
		}
	}

	expected := map[string]map[string]bool{}
	want := func(ts, cv string) {
		if expected[ts] == nil {
			expected[ts] = map[string]bool{}
		}
		expected[ts][cv] = true
	}
	borrowed := map[string][]string{}
	for i, m := range a.ThreatMitigations {
		ctrl := strings.TrimSpace(m.ControlRef)
		for j, ref := range m.VerificationRefs {
			ref = strings.TrimSpace(ref)
			other, ok := cvControl[ref]
			if !ok {
				continue
			}
			want(strings.TrimSpace(m.ThreatScenarioRef), ref)
			if other == ctrl || imported(m.ID) {
				continue
			}
			borrowed[ctrl] = append(borrowed[ctrl], fmt.Sprintf("%s verifies %s", ref, other))
			diags = append(diags, Diagnostic{
				Code: "mitigation.verification_control_mismatch", Severity: SeverityError,
				Message: fmt.Sprintf("threat mitigation %s applies control %s but its verification %s verifies control %s; reference a verification of %s", m.ID, ctrl, ref, other, ctrl),
				Path:    fmt.Sprintf("authoredArchitecture.threatMitigations[%d].verificationRefs[%d]", i, j),
			})
		}
	}
	for _, cv := range a.ControlVerifications {
		for _, ts := range cv.ThreatScenarioRefs {
			want(strings.TrimSpace(ts), strings.TrimSpace(cv.ID))
		}
	}
	for i, ts := range a.ThreatScenarios {
		id := strings.TrimSpace(ts.ID)
		if imported(id) {
			continue
		}
		listed := map[string]bool{}
		for _, ref := range ts.VerificationRefs {
			listed[strings.TrimSpace(ref)] = true
		}
		var missing []string
		for cv := range expected[id] {
			if !listed[cv] {
				missing = append(missing, cv)
			}
		}
		if len(missing) == 0 {
			continue
		}
		sort.Strings(missing)
		diags = append(diags, Diagnostic{
			Code: "threat.verification_missing", Severity: SeverityWarning,
			Message: fmt.Sprintf("threat scenario %s does not list control verifications %s that its mitigations reference or that name it; add them to verificationRefs", id, strings.Join(missing, ", ")),
			Path:    fmt.Sprintf("authoredArchitecture.threatScenarios[%d].verificationRefs", i),
		})
	}
	for i, cv := range a.ControlVerifications {
		status := strings.ToLower(strings.TrimSpace(cv.Status))
		if imported(cv.ID) || len(cv.VerificationRefs) > 0 || !strings.EqualFold(strings.TrimSpace(cv.Method), "test") || (status != "pass" && status != "partial") {
			continue
		}
		diags = append(diags, Diagnostic{
			Code: "verification.unlinked", Severity: SeverityWarning,
			Message: fmt.Sprintf("control verification %s is a test with status %s but has no verificationRefs; name the test declarations or requirements that evidence it", cv.ID, status),
			Path:    fmt.Sprintf("authoredArchitecture.controlVerifications[%d].verificationRefs", i),
		})
	}
	for i, c := range a.Controls {
		id := strings.TrimSpace(c.ID)
		if imported(id) || !ControlNeedsReview(statuses[id]) {
			continue
		}
		reason := "no control verification names it in controlRef"
		fix := fmt.Sprintf("add a control verification with controlRef %s", id)
		if len(statuses[id]) > 0 {
			reason = "its verifications need review: " + strings.Join(reviewIDs[id], ", ")
			fix = "record the verification result once it is known (pass or fail)"
		} else if refs := borrowed[id]; len(refs) > 0 {
			sort.Strings(refs)
			reason = "its only evidence is another control's verification (" + strings.Join(refs, "; ") + ")"
		}
		diags = append(diags, Diagnostic{
			Code: "gemara.control_needs_review", Severity: SeverityWarning,
			Message: fmt.Sprintf("control %s is Needs Review in the Gemara evaluation log: %s; %s", id, reason, fix),
			Path:    fmt.Sprintf("authoredArchitecture.controls[%d]", i),
		})
	}
	return diags
}

// VerificationOutcome is the evaluation result a control verification status
// stands for. The Gemara evaluation log maps it to its Result values.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
type VerificationOutcome int

// Verification outcomes.
const (
	OutcomeNotRun VerificationOutcome = iota
	OutcomePassed
	OutcomeFailed
	OutcomeNeedsReview
	OutcomeNotApplicable
	OutcomeUnknown
)

// ClassifyVerificationStatus maps a control verification status (trimmed,
// any case) to its outcome; an unrecognized status is OutcomeUnknown.
// TRLC-LINKS: REQ-EMG-083, REQ-EMG-015
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
func ClassifyVerificationStatus(status string) VerificationOutcome {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass", "passed", "satisfied", "implemented", "ok", "success":
		return OutcomePassed
	case "fail", "failed", "not-satisfied", "notsatisfied", "error":
		return OutcomeFailed
	case "partial", "needs-review", "review", "in-progress", "planned":
		return OutcomeNeedsReview
	case "n/a", "na", "not-applicable", "notapplicable":
		return OutcomeNotApplicable
	case "", "unknown", "not-run", "notrun":
		return OutcomeNotRun
	default:
		return OutcomeUnknown
	}
}

// ControlNeedsReview reports whether the Gemara evaluation log gives a
// control with these verification statuses the result Needs Review: it has
// no verification, or at least one needs review and none failed or is
// unknown (Failed and Unknown outrank Needs Review in the log's aggregate).
// TRLC-LINKS: REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
func ControlNeedsReview(statuses []string) bool {
	if len(statuses) == 0 {
		return true
	}
	review := false
	for _, status := range statuses {
		switch ClassifyVerificationStatus(status) {
		case OutcomeFailed, OutcomeUnknown:
			return false
		case OutcomeNeedsReview:
			review = true
		}
	}
	return review
}
