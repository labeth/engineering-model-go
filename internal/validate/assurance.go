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
	l := newAssuranceLinks(a)
	diags := l.mitigationDiagnostics()
	l.expectFromVerifications()
	diags = append(diags, l.threatDiagnostics()...)
	diags = append(diags, l.unlinkedDiagnostics()...)
	return append(diags, l.reviewDiagnostics()...)
}

// isImported reports whether an id is an imported alias::ID record of another model.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func isImported(id string) bool {
	return strings.Contains(id, "::")
}

// assuranceLinks is the index of the control verifications and what the
// threat and mitigation records expect of them.
type assuranceLinks struct {
	a         model.AuthoredArchitecture
	cvControl map[string]string
	statuses  map[string][]string
	reviewIDs map[string][]string
	expected  map[string]map[string]bool
	borrowed  map[string][]string
}

// newAssuranceLinks indexes the control verifications by id and by control.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func newAssuranceLinks(a model.AuthoredArchitecture) *assuranceLinks {
	l := &assuranceLinks{
		a: a, cvControl: map[string]string{}, statuses: map[string][]string{}, reviewIDs: map[string][]string{},
		expected: map[string]map[string]bool{}, borrowed: map[string][]string{},
	}
	for _, cv := range a.ControlVerifications {
		id, ctrl := strings.TrimSpace(cv.ID), strings.TrimSpace(cv.ControlRef)
		l.cvControl[id] = ctrl
		l.statuses[ctrl] = append(l.statuses[ctrl], cv.Status)
		if ClassifyVerificationStatus(cv.Status) == OutcomeNeedsReview {
			l.reviewIDs[ctrl] = append(l.reviewIDs[ctrl], fmt.Sprintf("%s (%s)", id, strings.TrimSpace(cv.Status)))
		}
	}
	return l
}

// want records that a threat scenario should list a control verification.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (l *assuranceLinks) want(ts, cv string) {
	if l.expected[ts] == nil {
		l.expected[ts] = map[string]bool{}
	}
	l.expected[ts][cv] = true
}

// mitigationDiagnostics reports each mitigation verification that verifies another control than the mitigation's.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (l *assuranceLinks) mitigationDiagnostics() []Diagnostic {
	var diags []Diagnostic
	for i, m := range l.a.ThreatMitigations {
		diags = append(diags, l.mitigation(i, m)...)
	}
	return diags
}

// mitigation reports the verification refs of one mitigation that verify another control, and records the verifications it expects of its threat.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (l *assuranceLinks) mitigation(i int, m model.ThreatMitigation) []Diagnostic {
	var diags []Diagnostic
	ctrl := strings.TrimSpace(m.ControlRef)
	for j, raw := range m.VerificationRefs {
		ref := strings.TrimSpace(raw)
		other, ok := l.cvControl[ref]
		if !ok {
			continue
		}
		l.want(strings.TrimSpace(m.ThreatScenarioRef), ref)
		if other == ctrl || isImported(m.ID) {
			continue
		}
		l.borrowed[ctrl] = append(l.borrowed[ctrl], fmt.Sprintf("%s verifies %s", ref, other))
		diags = append(diags, Diagnostic{
			Code: "mitigation.verification_control_mismatch", Severity: SeverityError,
			Message: fmt.Sprintf("threat mitigation %s applies control %s but its verification %s verifies control %s; reference a verification of %s", m.ID, ctrl, ref, other, ctrl),
			Path:    fmt.Sprintf("authoredArchitecture.threatMitigations[%d].verificationRefs[%d]", i, j),
		})
	}
	return diags
}

// expectFromVerifications records that a threat scenario a verification names should list that verification.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (l *assuranceLinks) expectFromVerifications() {
	for _, cv := range l.a.ControlVerifications {
		for _, ts := range cv.ThreatScenarioRefs {
			l.want(strings.TrimSpace(ts), strings.TrimSpace(cv.ID))
		}
	}
}

// threatDiagnostics warns about each threat scenario that omits expected verifications.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (l *assuranceLinks) threatDiagnostics() []Diagnostic {
	var diags []Diagnostic
	for i, ts := range l.a.ThreatScenarios {
		id := strings.TrimSpace(ts.ID)
		if isImported(id) {
			continue
		}
		missing := l.missingVerifications(id, ts)
		if len(missing) == 0 {
			continue
		}
		diags = append(diags, Diagnostic{
			Code: "threat.verification_missing", Severity: SeverityWarning,
			Message: fmt.Sprintf("threat scenario %s does not list control verifications %s that its mitigations reference or that name it; add them to verificationRefs", id, strings.Join(missing, ", ")),
			Path:    fmt.Sprintf("authoredArchitecture.threatScenarios[%d].verificationRefs", i),
		})
	}
	return diags
}

// missingVerifications lists, sorted, the verifications a threat scenario is expected to list and does not.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (l *assuranceLinks) missingVerifications(id string, ts model.ThreatScenario) []string {
	listed := map[string]bool{}
	for _, ref := range ts.VerificationRefs {
		listed[strings.TrimSpace(ref)] = true
	}
	var missing []string
	for cv := range l.expected[id] {
		if !listed[cv] {
			missing = append(missing, cv)
		}
	}
	sort.Strings(missing)
	return missing
}

// unlinkedDiagnostics warns about each passing test verification without verificationRefs.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (l *assuranceLinks) unlinkedDiagnostics() []Diagnostic {
	var diags []Diagnostic
	for i, cv := range l.a.ControlVerifications {
		status, unlinked := unlinkedTestStatus(cv)
		if !unlinked {
			continue
		}
		diags = append(diags, Diagnostic{
			Code: "verification.unlinked", Severity: SeverityWarning,
			Message: fmt.Sprintf("control verification %s is a test with status %s but has no verificationRefs; name the test declarations or requirements that evidence it", cv.ID, status),
			Path:    fmt.Sprintf("authoredArchitecture.controlVerifications[%d].verificationRefs", i),
		})
	}
	return diags
}

// unlinkedTestStatus returns the lowercase status of a local control
// verification that is a test with status pass or partial and no
// verificationRefs.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func unlinkedTestStatus(cv model.ControlVerification) (string, bool) {
	status := strings.ToLower(strings.TrimSpace(cv.Status))
	if isImported(cv.ID) || len(cv.VerificationRefs) > 0 || !strings.EqualFold(strings.TrimSpace(cv.Method), "test") {
		return status, false
	}
	return status, status == "pass" || status == "partial"
}

// reviewDiagnostics warns about each control the Gemara evaluation log gives the result Needs Review.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (l *assuranceLinks) reviewDiagnostics() []Diagnostic {
	var diags []Diagnostic
	for i, c := range l.a.Controls {
		id := strings.TrimSpace(c.ID)
		if isImported(id) || !ControlNeedsReview(l.statuses[id]) {
			continue
		}
		reason, fix := l.reviewReason(id)
		diags = append(diags, Diagnostic{
			Code: "gemara.control_needs_review", Severity: SeverityWarning,
			Message: fmt.Sprintf("control %s is Needs Review in the Gemara evaluation log: %s; %s", id, reason, fix),
			Path:    fmt.Sprintf("authoredArchitecture.controls[%d]", i),
		})
	}
	return diags
}

// reviewReason says why a control needs review and how to fix it.
// TRLC-LINKS: REQ-EMG-082, REQ-EMG-083
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (l *assuranceLinks) reviewReason(id string) (string, string) {
	if len(l.statuses[id]) > 0 {
		return "its verifications need review: " + strings.Join(l.reviewIDs[id], ", "), "record the verification result once it is known (pass or fail)"
	}
	if refs := l.borrowed[id]; len(refs) > 0 {
		sort.Strings(refs)
		return "its only evidence is another control's verification (" + strings.Join(refs, "; ") + ")", fmt.Sprintf("add a control verification with controlRef %s", id)
	}
	return "no control verification names it in controlRef", fmt.Sprintf("add a control verification with controlRef %s", id)
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

// assuranceConsistencyDiagnostics reports assurance records that disagree
// with the rest of the model (REQ-EMG-117):
//
//   - risk.severity_inconsistent (warning) at the severity of each threat
//     scenario whose (likelihood, impact) pair, compared case-insensitively,
//     maps to more than one severity among the model's threat scenarios. Only
//     threat scenarios carry likelihood, impact and severity together (a
//     risk has residualRisk, not a severity).
//   - controlVerification.risk_control_unlinked (warning) at a riskRefs entry
//     of a control verification naming a local risk whose relatedControls
//     omit the verification's controlRef: the trace must hold both ways.
//   - threat.attack_vector_missing (warning) at the attackVectorRef of a
//     threat scenario without one, when another local threat scenario names
//     its attack vector.
//
// Imported alias::ID records are skipped.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func assuranceConsistencyDiagnostics(a model.AuthoredArchitecture) []Diagnostic {
	c := consistencyCheck{a: a, severities: severityIndex(a), withVector: countWithVector(a)}
	var diags []Diagnostic
	for i, ts := range a.ThreatScenarios {
		if !isImported(strings.TrimSpace(ts.ID)) {
			diags = append(diags, c.threat(i, ts)...)
		}
	}
	return append(diags, riskControlDiagnostics(a)...)
}

// normText is a lowercased, trimmed string.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func normText(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ratingPair is a (likelihood, impact) pair of a threat scenario.
type ratingPair struct{ likelihood, impact string }

// pairOf is the normalized (likelihood, impact) pair of a threat scenario.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func pairOf(ts model.ThreatScenario) ratingPair {
	return ratingPair{normText(ts.Likelihood), normText(ts.Impact)}
}

// severityIndex maps each (likelihood, impact) pair of the local threat
// scenarios to the ids of the scenarios rated with each severity.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func severityIndex(a model.AuthoredArchitecture) map[ratingPair]map[string][]string {
	severities := map[ratingPair]map[string][]string{}
	for _, ts := range a.ThreatScenarios {
		p, sev := pairOf(ts), normText(ts.Severity)
		rated := p.likelihood != "" && p.impact != "" && sev != ""
		if isImported(ts.ID) || !rated {
			continue
		}
		if severities[p] == nil {
			severities[p] = map[string][]string{}
		}
		severities[p][sev] = append(severities[p][sev], strings.TrimSpace(ts.ID))
	}
	return severities
}

// countWithVector counts the local threat scenarios that name an attack vector.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func countWithVector(a model.AuthoredArchitecture) int {
	n := 0
	for _, ts := range a.ThreatScenarios {
		if !isImported(ts.ID) && strings.TrimSpace(ts.AttackVectorRef) != "" {
			n++
		}
	}
	return n
}

// consistencyCheck holds what the per threat scenario consistency checks share.
type consistencyCheck struct {
	a          model.AuthoredArchitecture
	severities map[ratingPair]map[string][]string
	withVector int
}

// threat reports an inconsistent severity and a missing attack vector of one threat scenario.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (c consistencyCheck) threat(i int, ts model.ThreatScenario) []Diagnostic {
	var diags []Diagnostic
	id := strings.TrimSpace(ts.ID)
	if others := c.otherSeverities(ts); len(others) > 0 {
		p := pairOf(ts)
		diags = append(diags, Diagnostic{
			Code: "risk.severity_inconsistent", Severity: SeverityWarning,
			Message: fmt.Sprintf("threat scenario %s rates likelihood %s and impact %s as severity %s, but the same pair is rated %s elsewhere; apply one scoring method (record it as a decision) and use the severity it gives", id, p.likelihood, p.impact, normText(ts.Severity), strings.Join(others, "; ")),
			Path:    fmt.Sprintf("authoredArchitecture.threatScenarios[%d].severity", i),
		})
	}
	if c.withVector > 0 && strings.TrimSpace(ts.AttackVectorRef) == "" {
		diags = append(diags, Diagnostic{
			Code: "threat.attack_vector_missing", Severity: SeverityWarning,
			Message: fmt.Sprintf("threat scenario %s names no attackVectorRef while %d other threat scenarios name theirs; add the attack vector (AV-*) it uses", id, c.withVector),
			Path:    fmt.Sprintf("authoredArchitecture.threatScenarios[%d].attackVectorRef", i),
		})
	}
	return diags
}

// otherSeverities lists, sorted, the other severities the scenario's rating pair has, with their scenario ids;
// none when the scenario has no severity or its pair has one severity only.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (c consistencyCheck) otherSeverities(ts model.ThreatScenario) []string {
	bySeverity := c.severities[pairOf(ts)]
	if len(bySeverity) <= 1 || normText(ts.Severity) == "" {
		return nil
	}
	var others []string
	for sev, ids := range bySeverity {
		if sev != normText(ts.Severity) {
			others = append(others, fmt.Sprintf("%s (%s)", sev, strings.Join(ids, ", ")))
		}
	}
	sort.Strings(others)
	return others
}

// riskControlDiagnostics warns about each risk a control verification names whose relatedControls omit the verification's control.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func riskControlDiagnostics(a model.AuthoredArchitecture) []Diagnostic {
	riskControls := riskControlIndex(a)
	var diags []Diagnostic
	for i, cv := range a.ControlVerifications {
		if ctrl := strings.TrimSpace(cv.ControlRef); !isImported(cv.ID) && ctrl != "" {
			diags = append(diags, unlinkedRiskControls(i, cv, riskControls)...)
		}
	}
	return diags
}

// riskControlIndex maps each risk id to the set of its related controls.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func riskControlIndex(a model.AuthoredArchitecture) map[string]map[string]bool {
	riskControls := map[string]map[string]bool{}
	for _, r := range a.Risks {
		controls := map[string]bool{}
		for _, c := range r.RelatedControls {
			controls[strings.TrimSpace(c)] = true
		}
		riskControls[strings.TrimSpace(r.ID)] = controls
	}
	return riskControls
}

// unlinkedRiskControls reports the riskRefs of a control verification whose risk does not list the control.
// TRLC-LINKS: REQ-EMG-117
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func unlinkedRiskControls(i int, cv model.ControlVerification, riskControls map[string]map[string]bool) []Diagnostic {
	var diags []Diagnostic
	ctrl := strings.TrimSpace(cv.ControlRef)
	for j, ref := range cv.RiskRefs {
		ref = strings.TrimSpace(ref)
		controls, ok := riskControls[ref]
		if !ok || isImported(ref) || controls[ctrl] {
			continue
		}
		diags = append(diags, Diagnostic{
			Code: "controlVerification.risk_control_unlinked", Severity: SeverityWarning,
			Message: fmt.Sprintf("control verification %s verifies control %s for risk %s, but %s does not list %s in relatedControls; add it there, or drop the risk from riskRefs", cv.ID, ctrl, ref, ref, ctrl),
			Path:    fmt.Sprintf("authoredArchitecture.controlVerifications[%d].riskRefs[%d]", i, j),
		})
	}
	return diags
}
