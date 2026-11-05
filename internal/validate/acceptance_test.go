// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// acceptanceTestBundle is a valid bundle with an owner actor, one decision and
// the given risks and threat scenarios.
func acceptanceTestBundle(risks []model.Risk, threats []model.ThreatScenario) model.Bundle {
	b := behaviorTestBundle(nil, nil)
	a := &b.Architecture.AuthoredArchitecture
	a.Actors = []model.Actor{{ID: "ACT-OWNER", Name: "Owner"}}
	a.Risks = risks
	a.ThreatScenarios = threats
	b.Architecture.Decisions = []model.Decision{{ID: "DEC-ACCEPT", Title: "Accept", Status: "accepted", Date: "2026-11-01", Context: "The owner said: accept it.", Decision: "Accept.", Consequences: []string{"Risk accepted."}}}
	return b
}

// An accepted risk (status accepted or response accept) and an accepted threat
// scenario without acceptedBy are errors at their acceptedBy field that name
// the fields to add; a risk with response accept and status open is also a
// warning at its status. A complete record (actor, date, statement) or a
// decisionRef clears the error.
// Mutation: dropping the response accept condition misses RISK-RESPONSE.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func TestAcceptedRecordsNeedAnOwnerAcceptance(t *testing.T) {
	b := acceptanceTestBundle(
		[]model.Risk{
			{ID: "RISK-STATUS", Title: "s", Statement: "s", Status: "accepted", Response: "mitigate"},
			{ID: "RISK-RESPONSE", Title: "r", Statement: "r", Status: "open", Response: "accept"},
			{ID: "RISK-MITIGATING", Title: "m", Statement: "m", Status: "mitigating", Response: "mitigate"},
		},
		[]model.ThreatScenario{{ID: "TS-A", Title: "t", AppliesTo: []string{"FU-A"}, Status: "accepted"}},
	)
	got := behaviorDiags(b, "risk.acceptance_unrecorded")
	paths := []string{}
	for _, d := range got {
		if d.Severity != SeverityError || !strings.Contains(d.Message, "add acceptedBy with actor, date, statement or decisionRef (") {
			t.Fatalf("diagnostic must be an error naming the missing actor, date, statement or decisionRef: %+v", d)
		}
		paths = append(paths, d.Path)
	}
	want := "authoredArchitecture.risks[0].acceptedBy,authoredArchitecture.risks[1].acceptedBy,authoredArchitecture.threatScenarios[0].acceptedBy"
	if strings.Join(paths, ",") != want {
		t.Fatalf("risk.acceptance_unrecorded paths = %v, want %s", paths, want)
	}
	open := behaviorDiags(b, "risk.accept_while_open")
	if len(open) != 1 || open[0].Severity != SeverityWarning || open[0].Path != "authoredArchitecture.risks[1].status" {
		t.Fatalf("want one risk.accept_while_open warning at risks[1].status, got %+v", open)
	}

	a := &b.Architecture.AuthoredArchitecture
	a.Risks[0].AcceptedBy = &model.AcceptanceRecord{Actor: "ACT-OWNER", Date: "2026-11-01", Statement: "I accept this risk."}
	a.Risks[1].AcceptedBy = &model.AcceptanceRecord{Actor: "ACT-OWNER", Date: "2026-11-01", DecisionRef: "DEC-ACCEPT"}
	a.ThreatScenarios[0].AcceptedBy = &model.AcceptanceRecord{Actor: "ACT-OWNER", Date: "2026-11-01", Statement: "Accepted."}
	if got := behaviorDiags(b, "risk.acceptance_unrecorded"); len(got) != 0 {
		t.Fatalf("complete records must clear the error, got %+v", got)
	}
	if HasErrors(Bundle(b)) {
		t.Fatalf("bundle with complete acceptance records has errors: %+v", Bundle(b))
	}
}

// A partial record names exactly the missing parts, and an acceptedBy actor,
// date or decisionRef that does not resolve is model.invalid_acceptance.
// Mutation: not checking the actor against the actors reports no
// model.invalid_acceptance for ACT-NOBODY; dropping the actor check from
// AcceptanceMissing changes the missing-parts list.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestAcceptanceRecordFieldsResolve(t *testing.T) {
	b := acceptanceTestBundle(
		[]model.Risk{
			{ID: "RISK-PARTIAL", Title: "p", Statement: "p", Status: "accepted", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OWNER"}},
			{ID: "RISK-BAD", Title: "b", Statement: "b", Status: "accepted", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-NOBODY", Date: "1 Nov", DecisionRef: "DEC-NONE"}},
		}, nil)
	unrecorded := behaviorDiags(b, "risk.acceptance_unrecorded")
	if len(unrecorded) != 1 || unrecorded[0].Path != "authoredArchitecture.risks[0].acceptedBy" || !strings.Contains(unrecorded[0].Message, "add acceptedBy with date, statement or decisionRef (") {
		t.Fatalf("want one risk.acceptance_unrecorded naming only date, statement or decisionRef for RISK-PARTIAL, got %+v", unrecorded)
	}
	if got := AcceptanceMissing(&model.AcceptanceRecord{Actor: "ACT-OWNER"}); strings.Join(got, "|") != "date|statement or decisionRef" {
		t.Fatalf("AcceptanceMissing = %v", got)
	}
	if got := AcceptanceMissing(&model.AcceptanceRecord{Date: "2026-11-01", DecisionRef: "DEC-ACCEPT"}); strings.Join(got, "|") != "actor" {
		t.Fatalf("AcceptanceMissing without actor = %v, want [actor]", got)
	}
	if got := AcceptanceMissing(nil); strings.Join(got, "|") != "actor|date|statement or decisionRef" {
		t.Fatalf("AcceptanceMissing(nil) = %v", got)
	}
	invalid := behaviorDiags(b, "model.invalid_acceptance")
	paths := []string{}
	for _, d := range invalid {
		if d.Severity != SeverityError {
			t.Fatalf("model.invalid_acceptance must be an error: %+v", d)
		}
		paths = append(paths, d.Path)
	}
	want := "authoredArchitecture.risks[1].acceptedBy.actor,authoredArchitecture.risks[1].acceptedBy.date,authoredArchitecture.risks[1].acceptedBy.decisionRef"
	if strings.Join(paths, ",") != want {
		t.Fatalf("model.invalid_acceptance paths = %v, want %s", paths, want)
	}
}

// A record resolves only when the date is a real calendar date (2026-13-45 is
// not), and a decisionRef names a decision with status accepted; a proposed
// decision is model.invalid_acceptance. A risk accepted by another actor than
// its owner is the warning risk.acceptance_not_owner at the actor; a risk
// without an owner, or accepted by its owner, is not. Resolves agrees.
// Mutation: matching the date with the YYYY-MM-DD pattern instead of
// time.Parse accepts 2026-13-45; not checking the decision status accepts
// DEC-PROPOSED; dropping the owner comparison reports no
// risk.acceptance_not_owner.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
func TestAcceptanceRecordNeedsCalendarDateAcceptedDecisionAndOwner(t *testing.T) {
	b := acceptanceTestBundle(
		[]model.Risk{
			{ID: "RISK-DATE", Title: "d", Statement: "d", Status: "accepted", Owner: "ACT-OWNER", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OWNER", Date: "2026-13-45", Statement: "ok"}},
			{ID: "RISK-PROPOSED", Title: "p", Statement: "p", Status: "accepted", Owner: "ACT-OWNER", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OWNER", Date: "2026-11-01", DecisionRef: "DEC-PROPOSED"}},
			{ID: "RISK-OTHER", Title: "o", Statement: "o", Status: "accepted", Owner: "ACT-OWNER", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OTHER", Date: "2026-11-01", Statement: "I accept it."}},
			{ID: "RISK-NO-OWNER", Title: "n", Statement: "n", Status: "accepted", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OTHER", Date: "2026-11-01", DecisionRef: "DEC-ACCEPT"}},
		}, nil)
	a := &b.Architecture.AuthoredArchitecture
	a.Actors = append(a.Actors, model.Actor{ID: "ACT-OTHER", Name: "Other"})
	b.Architecture.Decisions = append(b.Architecture.Decisions, model.Decision{ID: "DEC-PROPOSED", Title: "Maybe", Status: "proposed", Date: "2026-11-01", Context: "c", Decision: "d", Consequences: []string{"c"}})
	invalid := behaviorDiags(b, "model.invalid_acceptance")
	paths := []string{}
	for _, d := range invalid {
		paths = append(paths, d.Path)
	}
	want := "authoredArchitecture.risks[0].acceptedBy.date,authoredArchitecture.risks[1].acceptedBy.decisionRef"
	if strings.Join(paths, ",") != want {
		t.Fatalf("model.invalid_acceptance paths = %v, want %s (%+v)", paths, want, invalid)
	}
	if !strings.Contains(invalid[1].Message, `status "proposed"`) {
		t.Fatalf("decisionRef message must name the status: %+v", invalid[1])
	}
	notOwner := behaviorDiags(b, "risk.acceptance_not_owner")
	if len(notOwner) != 1 || notOwner[0].Severity != SeverityWarning || notOwner[0].Path != "authoredArchitecture.risks[2].acceptedBy.actor" ||
		!strings.Contains(notOwner[0].Message, "ACT-OTHER") || !strings.Contains(notOwner[0].Message, "ACT-OWNER") {
		t.Fatalf("want one risk.acceptance_not_owner warning at risks[2].acceptedBy.actor, got %+v", notOwner)
	}
	refs := NewAcceptanceRefs(b)
	var resolving []string
	for _, r := range a.Risks {
		if refs.Resolves(r.AcceptedBy) {
			resolving = append(resolving, r.ID)
		}
	}
	if strings.Join(resolving, ",") != "RISK-OTHER,RISK-NO-OWNER" {
		t.Fatalf("Resolves = %v, want RISK-OTHER,RISK-NO-OWNER", resolving)
	}
}

// An acceptance record with a decisionRef but no statement is a warning at
// its statement, for a risk and for a threat scenario: the owner's words
// belong on the risk itself. A record with both, and one with a statement
// only, report nothing.
// Mutation: checking only risks misses TS-DEC; reporting a record that also
// has a statement reports RISK-BOTH.
// TRLC-LINKS: REQ-EMG-116
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestAcceptanceByDecisionOnlyWarns(t *testing.T) {
	b := acceptanceTestBundle(
		[]model.Risk{
			{ID: "RISK-DEC", Title: "d", Statement: "d", Status: "accepted", Owner: "ACT-OWNER", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OWNER", Date: "2026-11-01", DecisionRef: "DEC-ACCEPT"}},
			{ID: "RISK-BOTH", Title: "b", Statement: "b", Status: "accepted", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OWNER", Date: "2026-11-01", DecisionRef: "DEC-ACCEPT", Statement: "I accept it."}},
			{ID: "RISK-WORDS", Title: "w", Statement: "w", Status: "accepted", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OWNER", Date: "2026-11-01", Statement: "I accept it."}},
		},
		[]model.ThreatScenario{{ID: "TS-DEC", Title: "t", AppliesTo: []string{"FU-A"}, Status: "accepted", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OWNER", Date: "2026-11-01", DecisionRef: "DEC-ACCEPT"}}},
	)
	var paths []string
	for _, d := range behaviorDiags(b, "risk.acceptance_by_decision_only") {
		if d.Severity != SeverityWarning || !strings.Contains(d.Message, "DEC-ACCEPT") || !strings.Contains(d.Message, "statement") {
			t.Fatalf("must be a warning naming the decision and statement: %+v", d)
		}
		paths = append(paths, d.Path)
	}
	if want := "authoredArchitecture.risks[0].acceptedBy.statement,authoredArchitecture.threatScenarios[0].acceptedBy.statement"; strings.Join(paths, ",") != want {
		t.Fatalf("risk.acceptance_by_decision_only paths = %v, want %s", paths, want)
	}
	if HasErrors(Bundle(b)) {
		t.Fatalf("decision-only acceptance must stay a warning: %+v", Bundle(b))
	}
}

// A risk dependencies entry must be a dependency alias declared in
// engmod.yml: an unknown or empty alias is model.invalid_risk_dependency
// (error) at the entry, naming the declared aliases; a declared alias reports
// nothing.
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestRiskDependencyMustBeADeclaredAlias(t *testing.T) {
	b := acceptanceTestBundle([]model.Risk{
		{ID: "RISK-DEP", Title: "d", Statement: "d", Status: "open", Dependencies: []string{"encoder", "nosuch"}},
	}, nil)
	b.Manifest.Dependencies = []model.ManifestDependency{{Alias: "encoder", Path: "models.example/encoder@v0", Version: "v0.1.0"}}
	got := behaviorDiags(b, "model.invalid_risk_dependency")
	if len(got) != 1 || got[0].Severity != SeverityError || got[0].Path != "authoredArchitecture.risks[0].dependencies[1]" ||
		!strings.Contains(got[0].Message, `"nosuch"`) || !strings.Contains(got[0].Message, "encoder") {
		t.Fatalf("model.invalid_risk_dependency = %+v, want one error at dependencies[1] naming nosuch and the alias encoder", got)
	}
	b.Architecture.AuthoredArchitecture.Risks[0].Dependencies = []string{"encoder"}
	if got := behaviorDiags(b, "model.invalid_risk_dependency"); len(got) != 0 {
		t.Fatalf("a declared alias must not be reported: %+v", got)
	}
}
