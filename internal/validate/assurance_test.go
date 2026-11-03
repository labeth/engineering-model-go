// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// assuranceTestBundle is a valid bundle with two controls, one threat scenario
// and the given mitigations and control verifications.
func assuranceTestBundle(ts model.ThreatScenario, mitigations []model.ThreatMitigation, cvs []model.ControlVerification) model.Bundle {
	b := behaviorTestBundle(nil, nil)
	a := &b.Architecture.AuthoredArchitecture
	a.Controls = []model.Control{{ID: "CTRL-A", Name: "A"}, {ID: "CTRL-B", Name: "B"}}
	a.ThreatScenarios = []model.ThreatScenario{ts}
	a.ThreatMitigations = mitigations
	a.ControlVerifications = cvs
	b.Requirements.Requirements = []model.Requirement{{ID: "REQ-A", Text: "x"}}
	return b
}

// A threat scenario must list every control verification that its
// mitigations reference or that names it in threatScenarioRefs; a missing one
// is a warning at the threat's verificationRefs naming the verification.
// Mutation: ignoring mitigation verificationRefs (only CV threatScenarioRefs)
// misses CV-B.
// TRLC-LINKS: REQ-EMG-082
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func TestThreatVerificationMissingWarnsAtTheThreat(t *testing.T) {
	ts := model.ThreatScenario{ID: "TS-A", Title: "T", AppliesTo: []string{"FU-A"}, MitigationRefs: []string{"MIT-A"}}
	b := assuranceTestBundle(ts,
		[]model.ThreatMitigation{{ID: "MIT-A", ThreatScenarioRef: "TS-A", ControlRef: "CTRL-B", VerificationRefs: []string{"CV-B"}}},
		[]model.ControlVerification{
			{ID: "CV-A", ControlRef: "CTRL-A", ThreatScenarioRefs: []string{"TS-A"}, Method: "analysis", Status: "not-run"},
			{ID: "CV-B", ControlRef: "CTRL-B", Method: "analysis", Status: "not-run"},
		})
	got := behaviorDiags(b, "threat.verification_missing")
	if len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Path != "authoredArchitecture.threatScenarios[0].verificationRefs" ||
		!strings.Contains(got[0].Message, "TS-A") || !strings.Contains(got[0].Message, "CV-A") || !strings.Contains(got[0].Message, "CV-B") {
		t.Fatalf("want one threat.verification_missing for TS-A naming CV-A and CV-B, got %+v", got)
	}

	b.Architecture.AuthoredArchitecture.ThreatScenarios[0].VerificationRefs = []string{"CV-A", "CV-B"}
	if got := behaviorDiags(b, "threat.verification_missing"); len(got) != 0 {
		t.Fatalf("a threat that lists its verifications must be quiet, got %+v", got)
	}
}

// A mitigation verificationRefs entry naming a control verification of
// another control is an error at that entry; a matching one is not.
// Mutation: comparing the mitigation's control with itself accepts CV-B.
// TRLC-LINKS: REQ-EMG-082
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func TestMitigationVerificationControlMismatchIsAnError(t *testing.T) {
	ts := model.ThreatScenario{ID: "TS-A", Title: "T", AppliesTo: []string{"FU-A"}, VerificationRefs: []string{"CV-A", "CV-B"}}
	b := assuranceTestBundle(ts,
		[]model.ThreatMitigation{{ID: "MIT-A", ThreatScenarioRef: "TS-A", ControlRef: "CTRL-A", VerificationRefs: []string{"CV-A", "CV-B"}}},
		[]model.ControlVerification{
			{ID: "CV-A", ControlRef: "CTRL-A", Method: "analysis", Status: "not-run"},
			{ID: "CV-B", ControlRef: "CTRL-B", Method: "analysis", Status: "not-run"},
		})
	got := behaviorDiags(b, "mitigation.verification_control_mismatch")
	if len(got) != 1 || got[0].Severity != SeverityError || got[0].Path != "authoredArchitecture.threatMitigations[0].verificationRefs[1]" ||
		!strings.Contains(got[0].Message, "MIT-A") || !strings.Contains(got[0].Message, "CTRL-B") || !strings.Contains(got[0].Message, "CTRL-A") {
		t.Fatalf("want one mitigation.verification_control_mismatch error at verificationRefs[1], got %+v", got)
	}
}

// A test-method control verification with status pass or partial and no
// verificationRefs is a warning at the verification; one with verificationRefs,
// a not-run one and an inspection are quiet.
// Mutation: dropping the method test condition reports CV-INSPECT.
// TRLC-LINKS: REQ-EMG-082
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func TestVerificationUnlinkedWarnsForPassingTestWithoutRefs(t *testing.T) {
	ts := model.ThreatScenario{ID: "TS-A", Title: "T", AppliesTo: []string{"FU-A"}}
	b := assuranceTestBundle(ts, nil, []model.ControlVerification{
		{ID: "CV-PASS", ControlRef: "CTRL-A", Method: "test", Status: "pass"},
		{ID: "CV-LINKED", ControlRef: "CTRL-A", Method: "test", Status: "pass", VerificationRefs: []string{"REQ-A"}},
		{ID: "CV-PARTIAL", ControlRef: "CTRL-B", Method: "Test", Status: "partial"},
		{ID: "CV-NOTRUN", ControlRef: "CTRL-B", Method: "test", Status: "not-run"},
		{ID: "CV-INSPECT", ControlRef: "CTRL-B", Method: "inspection", Status: "pass"},
	})
	got := behaviorDiags(b, "verification.unlinked")
	if len(got) != 2 || got[0].Path != "authoredArchitecture.controlVerifications[0].verificationRefs" ||
		got[1].Path != "authoredArchitecture.controlVerifications[2].verificationRefs" || got[0].Severity != SeverityWarning ||
		!strings.Contains(got[0].Message, "CV-PASS") || !strings.Contains(got[1].Message, "CV-PARTIAL") {
		t.Fatalf("want verification.unlinked for CV-PASS and CV-PARTIAL, got %+v", got)
	}
}

// A control that no control verification names gets a Needs Review result in
// the Gemara evaluation log, so it is a warning at the control; the message
// says when only another control's verification evidences it.
// Mutation: counting mitigation verificationRefs as the control's own
// verification hides CTRL-B.
// TRLC-LINKS: REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraControlNeedsReviewWarnsAtUnverifiedControl(t *testing.T) {
	ts := model.ThreatScenario{ID: "TS-A", Title: "T", AppliesTo: []string{"FU-A"}, VerificationRefs: []string{"CV-A"}}
	b := assuranceTestBundle(ts,
		[]model.ThreatMitigation{{ID: "MIT-B", ThreatScenarioRef: "TS-A", ControlRef: "CTRL-B", VerificationRefs: []string{"CV-A"}}},
		[]model.ControlVerification{{ID: "CV-A", ControlRef: "CTRL-A", Method: "analysis", Status: "pass"}})
	got := behaviorDiags(b, "gemara.control_needs_review")
	if len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Path != "authoredArchitecture.controls[1]" ||
		!strings.Contains(got[0].Message, "CTRL-B") || !strings.Contains(got[0].Message, "CV-A verifies CTRL-A") {
		t.Fatalf("want one gemara.control_needs_review for CTRL-B naming CV-A of CTRL-A, got %+v", got)
	}
}

// A control whose only verifications need review (partial) is reported like
// an unverified control, naming the verification; a control with a partial
// and a failed verification (the log says Failed) and one with a passing
// verification are not.
// Mutation: keeping the old rule (a control is reported only without any
// verification) misses CTRL-A.
// TRLC-LINKS: REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-GEMARA-EXPORTER
func TestGemaraControlNeedsReviewWarnsForPartialOnlyControl(t *testing.T) {
	ts := model.ThreatScenario{ID: "TS-A", Title: "T", AppliesTo: []string{"FU-A"}}
	b := assuranceTestBundle(ts, nil, []model.ControlVerification{
		{ID: "CV-A", ControlRef: "CTRL-A", Method: "analysis", Status: "partial"},
		{ID: "CV-B1", ControlRef: "CTRL-B", Method: "analysis", Status: "partial"},
		{ID: "CV-B2", ControlRef: "CTRL-B", Method: "analysis", Status: "fail"},
	})
	b.Architecture.AuthoredArchitecture.Controls = append(b.Architecture.AuthoredArchitecture.Controls, model.Control{ID: "CTRL-C", Name: "C", Category: "c", Description: "c"})
	b.Architecture.AuthoredArchitecture.ControlVerifications = append(b.Architecture.AuthoredArchitecture.ControlVerifications, model.ControlVerification{ID: "CV-C", ControlRef: "CTRL-C", Method: "analysis", Status: "pass"})
	got := behaviorDiags(b, "gemara.control_needs_review")
	if len(got) != 1 || got[0].Path != "authoredArchitecture.controls[0]" || !strings.Contains(got[0].Message, "CV-A (partial)") {
		t.Fatalf("want one gemara.control_needs_review at CTRL-A naming CV-A (partial), got %+v", got)
	}
	for statuses, want := range map[string]bool{"": true, "pass": false, "partial": true, "partial,pass": true, "partial,fail": false, "partial,weird": false, "not-run": false} {
		var list []string
		if statuses != "" {
			list = strings.Split(statuses, ",")
		}
		if ControlNeedsReview(list) != want {
			t.Fatalf("ControlNeedsReview(%v) = %v, want %v", list, !want, want)
		}
	}
}

// One (likelihood, impact) pair must map to one severity across the model's
// threat scenarios, compared case-insensitively: each scenario of a pair that
// maps to two severities is a warning at its severity naming the others.
// Scenarios without all three values and imported ones are skipped.
// Mutation: comparing case-sensitively, or reporting only the first
// scenario of a conflicting pair, fails this test.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestSeverityInconsistentWarnsPerScenario(t *testing.T) {
	b := assuranceTestBundle(model.ThreatScenario{ID: "TS-XSS", Title: "x", AppliesTo: []string{"FU-A"}, Likelihood: "low", Impact: "medium", Severity: "low"}, nil, nil)
	a := &b.Architecture.AuthoredArchitecture
	a.ThreatScenarios = append(a.ThreatScenarios,
		model.ThreatScenario{ID: "TS-REMOTE", Title: "r", AppliesTo: []string{"FU-A"}, Likelihood: "Low", Impact: "medium", Severity: "medium"},
		model.ThreatScenario{ID: "TS-SAME", Title: "s", AppliesTo: []string{"FU-A"}, Likelihood: "low", Impact: "MEDIUM", Severity: "Low"},
		model.ThreatScenario{ID: "TS-OTHER", Title: "o", AppliesTo: []string{"FU-A"}, Likelihood: "high", Impact: "high", Severity: "high"},
		model.ThreatScenario{ID: "TS-PARTIAL", Title: "p", AppliesTo: []string{"FU-A"}, Likelihood: "low", Impact: "medium"},
	)
	var paths []string
	for _, d := range behaviorDiags(b, "risk.severity_inconsistent") {
		if d.Severity != SeverityWarning || !strings.Contains(d.Message, "likelihood low and impact medium") {
			t.Fatalf("must be a warning naming the pair: %+v", d)
		}
		paths = append(paths, d.Path)
	}
	want := "authoredArchitecture.threatScenarios[0].severity,authoredArchitecture.threatScenarios[1].severity,authoredArchitecture.threatScenarios[2].severity"
	if strings.Join(paths, ",") != want {
		t.Fatalf("risk.severity_inconsistent paths = %v, want %s", paths, want)
	}
}

// A control verification that names a risk in riskRefs whose
// relatedControls omit the verification's control is a warning at that
// riskRefs entry; a risk that lists the control, an unknown risk and an
// imported one are not reported.
// Mutation: checking the risk's relatedControls against the wrong control,
// or skipping the riskRefs loop, fails this test.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func TestRiskControlUnlinkedWarnsAtTheVerification(t *testing.T) {
	b := assuranceTestBundle(model.ThreatScenario{ID: "TS-A", Title: "t", AppliesTo: []string{"FU-A"}}, nil, []model.ControlVerification{
		{ID: "CV-A", ControlRef: "CTRL-A", RiskRefs: []string{"RISK-LISTS", "RISK-OMITS", "dep::RISK-X"}, Method: "analysis", Status: "not-run"},
	})
	b.Architecture.AuthoredArchitecture.Risks = []model.Risk{
		{ID: "RISK-LISTS", Title: "l", Statement: "l", Status: "open", RelatedControls: []string{"CTRL-A"}},
		{ID: "RISK-OMITS", Title: "o", Statement: "o", Status: "open", RelatedControls: []string{"CTRL-B"}},
	}
	got := behaviorDiags(b, "controlVerification.risk_control_unlinked")
	if len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Path != "authoredArchitecture.controlVerifications[0].riskRefs[1]" ||
		!strings.Contains(got[0].Message, "RISK-OMITS") || !strings.Contains(got[0].Message, "CTRL-A") {
		t.Fatalf("want one warning at riskRefs[1] naming RISK-OMITS and CTRL-A, got %+v", got)
	}
}

// Where at least one local threat scenario names its attackVectorRef, each
// one without it is a warning at attackVectorRef; a model where none names
// one reports nothing.
// Mutation: reporting regardless of the other scenarios, or never, fails
// this test.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestAttackVectorMissingWhenOthersHaveOne(t *testing.T) {
	b := assuranceTestBundle(model.ThreatScenario{ID: "TS-NONE", Title: "n", AppliesTo: []string{"FU-A"}}, nil, nil)
	if got := behaviorDiags(b, "threat.attack_vector_missing"); len(got) != 0 {
		t.Fatalf("no scenario names a vector, so none is missing: %+v", got)
	}
	a := &b.Architecture.AuthoredArchitecture
	a.AttackVectors = []model.AttackVector{{ID: "AV-A", Name: "A"}}
	a.ThreatScenarios = append(a.ThreatScenarios, model.ThreatScenario{ID: "TS-WITH", Title: "w", AppliesTo: []string{"FU-A"}, AttackVectorRef: "AV-A"})
	got := behaviorDiags(b, "threat.attack_vector_missing")
	if len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Path != "authoredArchitecture.threatScenarios[0].attackVectorRef" || !strings.Contains(got[0].Message, "TS-NONE") {
		t.Fatalf("want one warning at TS-NONE attackVectorRef, got %+v", got)
	}
}
