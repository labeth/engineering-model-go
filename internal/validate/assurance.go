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
//   - gemara.control_needs_review (warning): a control that no control
//     verification names in controlRef; the Gemara evaluation log gives it the
//     result Needs Review. The message names any verification of another
//     control that a mitigation of this control relies on.
//
// Imported alias::ID records belong to their own model and are skipped.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func assuranceLinkDiagnostics(a model.AuthoredArchitecture) []Diagnostic {
	var diags []Diagnostic
	imported := func(id string) bool { return strings.Contains(id, "::") }
	cvControl := map[string]string{}
	verified := map[string]bool{}
	for _, cv := range a.ControlVerifications {
		id, ctrl := strings.TrimSpace(cv.ID), strings.TrimSpace(cv.ControlRef)
		cvControl[id] = ctrl
		verified[ctrl] = true
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
		if imported(id) || verified[id] {
			continue
		}
		reason := "no control verification names it in controlRef"
		if refs := borrowed[id]; len(refs) > 0 {
			sort.Strings(refs)
			reason = "its only evidence is another control's verification (" + strings.Join(refs, "; ") + ")"
		}
		diags = append(diags, Diagnostic{
			Code: "gemara.control_needs_review", Severity: SeverityWarning,
			Message: fmt.Sprintf("control %s is Needs Review in the Gemara evaluation log: %s; add a control verification with controlRef %s", id, reason, id),
			Path:    fmt.Sprintf("authoredArchitecture.controls[%d]", i),
		})
	}
	return diags
}
