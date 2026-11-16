// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// staleVerificationStatus warns (verification.status_stale) when an assurance
// record still says its verification has not happened although the tests it
// names exist in the code: a control verification whose status is unset or
// not-run while a verificationRefs entry is a test declaration found in the
// scanned code (a CODE identifier of a declaration in a test file) or a
// requirement that a test entry point links; and a threat mitigation whose
// status is unset or planned while a control verification it references has
// such tests. The status stays authored; the warning only says to update it.
// Imported alias::ID records belong to their own model and are skipped.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func staleVerificationStatus(bundle model.Bundle, scopedCode []codescan.CodeItem, verification []codescan.VerificationCheck) []validate.Diagnostic {
	tests := existingTests(scopedCode, verification)
	a := bundle.Architecture.AuthoredArchitecture
	diags, existing := staleControlVerifications(a.ControlVerifications, tests)
	return append(diags, staleThreatMitigations(a.ThreatMitigations, existing)...)
}

// staleControlVerifications reports the control verifications with a stale
// status and returns, by id, the tests each names.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func staleControlVerifications(cvs []model.ControlVerification, tests map[string]bool) ([]validate.Diagnostic, map[string][]string) {
	var diags []validate.Diagnostic
	existing := map[string][]string{}
	for i, cv := range cvs {
		found := textutil.UniqueSorted(foundTests(cv.VerificationRefs, tests))
		existing[strings.TrimSpace(cv.ID)] = found
		if d, stale := staleControlVerification(i, cv, found); stale {
			diags = append(diags, d)
		}
	}
	return diags, existing
}

// staleThreatMitigations reports the threat mitigations with a stale status.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func staleThreatMitigations(ms []model.ThreatMitigation, existing map[string][]string) []validate.Diagnostic {
	var diags []validate.Diagnostic
	for i, m := range ms {
		if d, stale := staleThreatMitigation(i, m, existing); stale {
			diags = append(diags, d)
		}
	}
	return diags
}

// existingTests is the tests in the code: the CODE identifiers of
// declarations in test files and the requirements a verification check links.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func existingTests(scopedCode []codescan.CodeItem, verification []codescan.VerificationCheck) map[string]bool {
	tests := verifiedRequirements(verification)
	for _, it := range scopedCode {
		if it.Kind == "symbol" && model.IsCodeID(it.Element) && codescan.IsVerificationCodeItem(it) {
			tests[strings.TrimSpace(it.Element)] = true
		}
	}
	return tests
}

// foundTests is the trimmed refs that name an existing test.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func foundTests(refs []string, tests map[string]bool) []string {
	var found []string
	for _, ref := range refs {
		if ref = strings.TrimSpace(ref); tests[ref] {
			found = append(found, ref)
		}
	}
	return found
}

// staleControlVerification reports a control verification whose status is
// unset or not-run while tests it names exist (found); imported alias::ID
// records are skipped.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func staleControlVerification(i int, cv model.ControlVerification, found []string) (validate.Diagnostic, bool) {
	id := strings.TrimSpace(cv.ID)
	status := strings.ToLower(strings.TrimSpace(cv.Status))
	if len(found) == 0 || strings.Contains(id, "::") || !statusLags(status, "not-run") {
		return validate.Diagnostic{}, false
	}
	return validate.Diagnostic{
		Code: "verification.status_stale", Severity: validate.SeverityWarning,
		Message: fmt.Sprintf("control verification %s has status %q but tests exist: %s; run them and update the status", id, textutil.NonEmpty(status, "unset"), strings.Join(found, ", ")),
		Path:    fmt.Sprintf("authoredArchitecture.controlVerifications[%d].status", i),
	}, true
}

// staleThreatMitigation reports a threat mitigation whose status is unset or
// planned while a control verification it references has tests (existing,
// by control verification id); imported alias::ID records are skipped.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func staleThreatMitigation(i int, m model.ThreatMitigation, existing map[string][]string) (validate.Diagnostic, bool) {
	id := strings.TrimSpace(m.ID)
	status := strings.ToLower(strings.TrimSpace(m.Status))
	if strings.Contains(id, "::") || !statusLags(status, "planned") {
		return validate.Diagnostic{}, false
	}
	var tested []string
	for _, ref := range m.VerificationRefs {
		if ref = strings.TrimSpace(ref); len(existing[ref]) > 0 {
			tested = append(tested, fmt.Sprintf("%s (%s)", ref, strings.Join(existing[ref], ", ")))
		}
	}
	if len(tested) == 0 {
		return validate.Diagnostic{}, false
	}
	return validate.Diagnostic{
		Code: "verification.status_stale", Severity: validate.SeverityWarning,
		Message: fmt.Sprintf("threat mitigation %s has status %q but tests exist: %s; update the status", id, textutil.NonEmpty(status, "unset"), strings.Join(tested, "; ")),
		Path:    fmt.Sprintf("authoredArchitecture.threatMitigations[%d].status", i),
	}, true
}

// statusLags reports whether a status is unset or still the initial one.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func statusLags(status, initial string) bool {
	return status == "" || status == initial
}
