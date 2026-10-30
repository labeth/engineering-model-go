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
	tests := map[string]bool{}
	for _, it := range scopedCode {
		if it.Kind == "symbol" && model.IsCodeID(it.Element) && codescan.IsVerificationCodeItem(it) {
			tests[strings.TrimSpace(it.Element)] = true
		}
	}
	for _, v := range verification {
		for _, r := range v.Verifies {
			tests[strings.TrimSpace(r)] = true
		}
	}
	var diags []validate.Diagnostic
	existing := map[string][]string{}
	a := bundle.Architecture.AuthoredArchitecture
	for i, cv := range a.ControlVerifications {
		id := strings.TrimSpace(cv.ID)
		var found []string
		for _, ref := range cv.VerificationRefs {
			if ref = strings.TrimSpace(ref); tests[ref] {
				found = append(found, ref)
			}
		}
		found = textutil.UniqueSorted(found)
		existing[id] = found
		status := strings.ToLower(strings.TrimSpace(cv.Status))
		if len(found) == 0 || strings.Contains(id, "::") || (status != "" && status != "not-run") {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "verification.status_stale", Severity: validate.SeverityWarning,
			Message: fmt.Sprintf("control verification %s has status %q but tests exist: %s; run them and update the status", id, textutil.NonEmpty(status, "unset"), strings.Join(found, ", ")),
			Path:    fmt.Sprintf("authoredArchitecture.controlVerifications[%d].status", i),
		})
	}
	for i, m := range a.ThreatMitigations {
		id := strings.TrimSpace(m.ID)
		status := strings.ToLower(strings.TrimSpace(m.Status))
		if strings.Contains(id, "::") || (status != "" && status != "planned") {
			continue
		}
		var tested []string
		for _, ref := range m.VerificationRefs {
			if ref = strings.TrimSpace(ref); len(existing[ref]) > 0 {
				tested = append(tested, fmt.Sprintf("%s (%s)", ref, strings.Join(existing[ref], ", ")))
			}
		}
		if len(tested) == 0 {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "verification.status_stale", Severity: validate.SeverityWarning,
			Message: fmt.Sprintf("threat mitigation %s has status %q but tests exist: %s; update the status", id, textutil.NonEmpty(status, "unset"), strings.Join(tested, "; ")),
			Path:    fmt.Sprintf("authoredArchitecture.threatMitigations[%d].status", i),
		})
	}
	return diags
}
