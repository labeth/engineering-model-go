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
		if d.Severity != SeverityError || !strings.Contains(d.Message, "acceptedBy") || !strings.Contains(d.Message, "actor") || !strings.Contains(d.Message, "statement") {
			t.Fatalf("diagnostic must be an error naming acceptedBy actor, date and statement: %+v", d)
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

// A partial record names the missing parts, and an acceptedBy actor, date or
// decisionRef that does not resolve is model.invalid_acceptance.
// Mutation: not checking the actor against the actors reports no
// model.invalid_acceptance for ACT-NOBODY.
// TRLC-LINKS: REQ-EMG-085
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestAcceptanceRecordFieldsResolve(t *testing.T) {
	b := acceptanceTestBundle(
		[]model.Risk{
			{ID: "RISK-PARTIAL", Title: "p", Statement: "p", Status: "accepted", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-OWNER"}},
			{ID: "RISK-BAD", Title: "b", Statement: "b", Status: "accepted", AcceptedBy: &model.AcceptanceRecord{Actor: "ACT-NOBODY", Date: "1 Nov", DecisionRef: "DEC-NONE"}},
		}, nil)
	unrecorded := behaviorDiags(b, "risk.acceptance_unrecorded")
	if len(unrecorded) != 1 || unrecorded[0].Path != "authoredArchitecture.risks[0].acceptedBy" || !strings.Contains(unrecorded[0].Message, "date") || strings.Contains(unrecorded[0].Message, "actor,") {
		t.Fatalf("want one risk.acceptance_unrecorded naming date and statement for RISK-PARTIAL, got %+v", unrecorded)
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
