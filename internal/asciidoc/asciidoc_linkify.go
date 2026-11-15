// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

// linkifier turns the model terms in the text of a section into links.
type linkifier struct {
	targets map[string]linkTarget
}

// all links each of the strings in place.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) all(fields ...*string) {
	for _, f := range fields {
		*f = linkifyText(*f, l.targets)
	}
}

// linkEach applies link to every item of a slice, in place.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func linkEach[T any](items []T, link func(*T)) {
	for i := range items {
		link(&items[i])
	}
}

// linkifySections links the model terms in every section of the document, in place.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (g *generation) linkifySections() {
	l := linkifier{targets: g.linkTargets}
	linkEach(g.views, l.view)
	linkEach(g.reqSections, l.requirement)
	linkEach(g.verification, l.verification)
	linkEach(g.verResults, l.verificationResult)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) view(v *asciidocViewSection) {
	linkEach(v.Groups, l.group)
	linkEach(v.Units, l.unit)
	linkEach(v.SecurityAttackChapters, l.attackChapter)
	l.securityRows(v)
	l.all(&v.Inf)
	l.each(v.ViewQuestions)
	l.each(v.CoverageGaps)
	l.each(v.NextActions)
	linkEach(v.ProjectedNodes, l.projectedNode)
	linkEach(v.ProjectedMappings, l.projectedMapping)
	l.all(&v.CoverageSummary)
}

// securityRows links the threat model rows of a security view.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) securityRows(v *asciidocViewSection) {
	linkEach(v.SecurityThreatScenarios, l.threatScenario)
	linkEach(v.SecurityThreatAssumptions, l.threatAssumption)
	linkEach(v.SecurityThreatOutOfScope, l.threatOutOfScope)
	linkEach(v.SecurityThreatMitigations, l.threatMitigation)
	linkEach(v.SecurityControlChecks, l.controlVerification)
	linkEach(v.SecurityFlowRows, l.securityFlow)
}

// each links every string of a slice.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) each(items []string) {
	for j := range items {
		l.all(&items[j])
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) details(details []asciidocDesignDetail) {
	for k := range details {
		l.all(&details[k].Narrative)
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) group(g *asciidocEntitySection) {
	l.all(&g.Description, &g.Intro)
	l.details(g.Details)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) unit(u *asciidocUnitSection) {
	l.all(&u.Intro, &u.WhatOwns, &u.Triggers, &u.Consumes, &u.Produces, &u.DependsOn, &u.Threats, &u.Evidence)
	l.details(u.Details)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) attackChapter(c *asciidocSecurityAttackChapter) {
	l.all(&c.Name, &c.Description)
	linkEach(c.Units, l.unit)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) threatScenario(s *asciidocThreatScenarioRow) {
	l.all(&s.ID, &s.Title, &s.Summary, &s.AttackVector, &s.Scope, &s.Flows, &s.Likelihood, &s.Impact,
		&s.Severity, &s.Status, &s.Owner, &s.Risk, &s.Controls, &s.Mitigations, &s.Verifications)
	linkEach(s.ControlVerifications, l.controlVerification)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) threatAssumption(a *asciidocThreatAssumptionRow) {
	l.all(&a.ID, &a.Title, &a.Status, &a.Owner, &a.AppliesTo, &a.Rationale)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) threatOutOfScope(o *asciidocThreatOutOfScopeRow) {
	l.all(&o.ID, &o.Title, &o.Status, &o.Owner, &o.AppliesTo, &o.ExpiresOn, &o.Reason)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) threatMitigation(m *asciidocThreatMitigationRow) {
	l.all(&m.ID, &m.ThreatScenario, &m.Control, &m.Status, &m.Effectiveness, &m.Owner, &m.Verifications, &m.Notes)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) controlVerification(c *asciidocControlVerificationRow) {
	l.all(&c.ID, &c.Control, &c.ThreatScenarios, &c.Risks, &c.Method, &c.Status, &c.Owner, &c.LastTested, &c.Findings)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) securityFlow(f *asciidocSecurityFlowRow) {
	l.all(&f.ID, &f.Title, &f.Kind, &f.Direction, &f.Frequency, &f.Source, &f.Destination, &f.Protocol,
		&f.Authentication, &f.Encryption, &f.Integrity, &f.TrustBoundary, &f.BoundaryCrossing, &f.Threats, &f.Data)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) projectedNode(n *asciidocProjectedNode) {
	l.all(&n.ID, &n.Kind, &n.Label)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) projectedMapping(m *asciidocMappingSection) {
	l.all(&m.Type, &m.From, &m.To, &m.Description)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) requirement(r *asciidocRequirementSection) {
	l.all(&r.Text, &r.Notes)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) verification(v *asciidocVerificationSection) {
	l.all(&v.Kind, &v.Status, &v.Verifies, &v.TestCode, &v.DerivedOwners, &v.Evidence, &v.ResultSummary, &v.Description)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) verificationResult(r *asciidocVerificationResultRow) {
	l.all(&r.CheckID, &r.CheckName, &r.Requirement, &r.Status, &r.Evidence, &r.Notes)
}
