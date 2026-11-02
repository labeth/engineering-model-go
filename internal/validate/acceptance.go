// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"
	"time"

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

// AcceptanceRefs resolves the references of acceptance records against one
// model: its actors and its decisions with their status. Validation and the
// Gemara policy share it, so a record the policy lists as accepted is one
// validation accepts.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
type AcceptanceRefs struct {
	actors    map[string]bool
	decisions map[string]string // decision ID to lowercased status
}

// AcceptanceProblem is one acceptedBy field that does not resolve.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
type AcceptanceProblem struct {
	Field   string // actor, date or decisionRef
	Message string
}

// NewAcceptanceRefs collects the actors and decisions of a bundle.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
func NewAcceptanceRefs(b model.Bundle) AcceptanceRefs {
	refs := AcceptanceRefs{actors: map[string]bool{}, decisions: map[string]string{}}
	for _, a := range b.Architecture.AuthoredArchitecture.Actors {
		refs.actors[strings.TrimSpace(a.ID)] = true
	}
	for _, d := range b.Architecture.Decisions {
		refs.decisions[strings.TrimSpace(d.ID)] = strings.ToLower(strings.TrimSpace(d.Status))
	}
	return refs
}

// Problems returns the fields of a record that are set but do not resolve:
// an actor that is no actor, a date that is no calendar date in YYYY-MM-DD,
// and a decisionRef that is no decision or a decision whose status is not
// accepted.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
func (r AcceptanceRefs) Problems(rec *model.AcceptanceRecord) []AcceptanceProblem {
	if rec == nil {
		return nil
	}
	var out []AcceptanceProblem
	if actor := strings.TrimSpace(rec.Actor); actor != "" && !r.actors[actor] {
		out = append(out, AcceptanceProblem{"actor", fmt.Sprintf("acceptedBy actor %q is not an actor", actor)})
	}
	if date := strings.TrimSpace(rec.Date); date != "" {
		if _, err := time.Parse(time.DateOnly, date); err != nil {
			out = append(out, AcceptanceProblem{"date", fmt.Sprintf("acceptedBy date %q is not a YYYY-MM-DD calendar date", date)})
		}
	}
	if ref := strings.TrimSpace(rec.DecisionRef); ref != "" {
		status, ok := r.decisions[ref]
		switch {
		case !ok:
			out = append(out, AcceptanceProblem{"decisionRef", fmt.Sprintf("acceptedBy decisionRef %q is not a decision", ref)})
		case status != "accepted":
			out = append(out, AcceptanceProblem{"decisionRef", fmt.Sprintf("acceptedBy decisionRef %q is a decision with status %q, not accepted", ref, status)})
		}
	}
	return out
}

// Resolves reports whether a record is complete and every field resolves:
// the condition for listing a risk as accepted.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
func (r AcceptanceRefs) Resolves(rec *model.AcceptanceRecord) bool {
	return len(AcceptanceMissing(rec)) == 0 && len(r.Problems(rec)) == 0
}

// acceptanceDiagnostics reports accepted risks and threat scenarios without an
// owner acceptance record:
//
//   - risk.acceptance_unrecorded (error) at acceptedBy of a risk with status
//     accepted or response accept, or a threat scenario with status accepted,
//     whose acceptedBy is missing or lacks actor, date, or statement and
//     decisionRef.
//   - model.invalid_acceptance (error) at an acceptedBy actor that is no
//     actor, a date that is not a YYYY-MM-DD calendar date or a decisionRef
//     that is no decision or a decision that is not accepted.
//   - risk.acceptance_not_owner (warning) at the acceptedBy actor of a risk
//     with an owner when the actor is another actor than the owner.
//   - risk.accept_while_open (warning) at the status of a risk with response
//     accept and status open.
//
// Imported alias::ID records belong to their own model and are skipped; the
// schema rejects :: in authored risk and threat scenario IDs, so only
// composition enrichment produces them.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func acceptanceDiagnostics(b model.Bundle) []Diagnostic {
	a := b.Architecture.AuthoredArchitecture
	refs := NewAcceptanceRefs(b)
	var diags []Diagnostic
	check := func(kind, id, path string, accepted bool, rec *model.AcceptanceRecord) {
		if missing := AcceptanceMissing(rec); accepted && len(missing) > 0 {
			diags = append(diags, Diagnostic{
				Code: "risk.acceptance_unrecorded", Severity: SeverityError,
				Message: fmt.Sprintf("%s %s is accepted without an owner acceptance record; add acceptedBy with %s (actor: the ACT-* owner who accepted, date: YYYY-MM-DD, statement: the owner's own words, or decisionRef: an accepted decision quoting the owner), or set the status the risk really has", kind, id, strings.Join(missing, ", ")),
				Path:    path + ".acceptedBy",
			})
		}
		for _, p := range refs.Problems(rec) {
			diags = append(diags, Diagnostic{Code: "model.invalid_acceptance", Severity: SeverityError, Message: p.Message, Path: path + ".acceptedBy." + p.Field})
		}
	}
	for i, r := range a.Risks {
		id := strings.TrimSpace(r.ID)
		if strings.Contains(id, "::") {
			continue
		}
		path := fmt.Sprintf("authoredArchitecture.risks[%d]", i)
		check("risk", id, path, riskRecordedAccepted(r), r.AcceptedBy)
		if owner := strings.TrimSpace(r.Owner); owner != "" && r.AcceptedBy != nil {
			if actor := strings.TrimSpace(r.AcceptedBy.Actor); actor != "" && actor != owner {
				diags = append(diags, Diagnostic{
					Code: "risk.acceptance_not_owner", Severity: SeverityWarning,
					Message: fmt.Sprintf("risk %s was accepted by %s, but its owner is %s; record the owner's acceptance, or make the accepting actor the owner", id, actor, owner),
					Path:    path + ".acceptedBy.actor",
				})
			}
		}
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
