// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// riskRecordedAccepted reports whether a risk is recorded as accepted: status
// accepted or response accept.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
func riskRecordedAccepted(r model.Risk) bool {
	return strings.EqualFold(strings.TrimSpace(r.Status), "accepted") || strings.EqualFold(strings.TrimSpace(r.Response), "accept")
}

// AcceptanceMissing returns the parts an acceptance record lacks: actor,
// date, and statement or decisionRef. It is empty for a complete record.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
func AcceptanceMissing(rec *model.AcceptanceRecord) []string {
	if rec == nil {
		return []string{"actor", "date", "statement or decisionRef"}
	}
	var missing []string
	if strings.TrimSpace(rec.Actor) == "" {
		missing = append(missing, "actor")
	}
	if strings.TrimSpace(rec.Date) == "" {
		missing = append(missing, "date")
	}
	if strings.TrimSpace(rec.Statement) == "" && strings.TrimSpace(rec.DecisionRef) == "" {
		missing = append(missing, "statement or decisionRef")
	}
	return missing
}

// acceptanceDiagnostics reports accepted risks and threat scenarios without an
// owner acceptance record:
//
//   - risk.acceptance_unrecorded (error) at acceptedBy of a risk with status
//     accepted or response accept, or a threat scenario with status accepted,
//     whose acceptedBy is missing or lacks actor, date, or statement and
//     decisionRef.
//   - model.invalid_acceptance (error) at an acceptedBy actor that is no
//     actor, a date that is not YYYY-MM-DD or a decisionRef that is no
//     decision.
//   - risk.accept_while_open (warning) at the status of a risk with response
//     accept and status open.
//
// Imported alias::ID records belong to their own model and are skipped.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func acceptanceDiagnostics(a model.AuthoredArchitecture, actors map[string]bool, decisions map[string]bool) []Diagnostic {
	var diags []Diagnostic
	check := func(kind, id, path string, accepted bool, rec *model.AcceptanceRecord) {
		if missing := AcceptanceMissing(rec); accepted && len(missing) > 0 {
			diags = append(diags, Diagnostic{
				Code: "risk.acceptance_unrecorded", Severity: SeverityError,
				Message: fmt.Sprintf("%s %s is accepted without an owner acceptance record; add acceptedBy with %s (actor: the ACT-* owner who accepted, date: YYYY-MM-DD, statement: the owner's own words, or decisionRef: a decision quoting the owner), or set the status the risk really has", kind, id, strings.Join(missing, ", ")),
				Path:    path + ".acceptedBy",
			})
		}
		if rec == nil {
			return
		}
		if actor := strings.TrimSpace(rec.Actor); actor != "" && !actors[actor] {
			diags = append(diags, Diagnostic{Code: "model.invalid_acceptance", Severity: SeverityError, Message: fmt.Sprintf("acceptedBy actor %q is not an actor", actor), Path: path + ".acceptedBy.actor"})
		}
		if date := strings.TrimSpace(rec.Date); date != "" && !isoDateRe.MatchString(date) {
			diags = append(diags, Diagnostic{Code: "model.invalid_acceptance", Severity: SeverityError, Message: fmt.Sprintf("acceptedBy date %q is not YYYY-MM-DD", date), Path: path + ".acceptedBy.date"})
		}
		if ref := strings.TrimSpace(rec.DecisionRef); ref != "" && !decisions[ref] {
			diags = append(diags, Diagnostic{Code: "model.invalid_acceptance", Severity: SeverityError, Message: fmt.Sprintf("acceptedBy decisionRef %q is not a decision", ref), Path: path + ".acceptedBy.decisionRef"})
		}
	}
	for i, r := range a.Risks {
		id := strings.TrimSpace(r.ID)
		if strings.Contains(id, "::") {
			continue
		}
		path := fmt.Sprintf("authoredArchitecture.risks[%d]", i)
		check("risk", id, path, riskRecordedAccepted(r), r.AcceptedBy)
		if strings.EqualFold(strings.TrimSpace(r.Response), "accept") && strings.EqualFold(strings.TrimSpace(r.Status), "open") {
			diags = append(diags, Diagnostic{
				Code: "risk.accept_while_open", Severity: SeverityWarning,
				Message: fmt.Sprintf("risk %s has response accept but status open; set status accepted once the owner has accepted it, or the response it is waiting for", id),
				Path:    path + ".status",
			})
		}
	}
	for i, ts := range a.ThreatScenarios {
		id := strings.TrimSpace(ts.ID)
		if strings.Contains(id, "::") {
			continue
		}
		check("threat scenario", id, fmt.Sprintf("authoredArchitecture.threatScenarios[%d]", i), strings.EqualFold(strings.TrimSpace(ts.Status), "accepted"), ts.AcceptedBy)
	}
	return diags
}
