// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package gemara

// Extended Gemara artifacts derived from existing model data (no new authored
// inputs required):
//   - Lexicon         <- catalog terms (controlled vocabulary)
//   - Mapping Document <- control -> threat relationships
//   - L7 Audit Log     <- control verifications + risks (point-in-time review)
//   - L6 Enforcement Log <- POA&M items (remediative actions)

import (
	"fmt"
	"strings"

	gemaraproj "github.com/gemaraproj/go-gemara"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func (c gemaraConfig) targetResource() gemaraproj.Resource {
	return gemaraproj.Resource{
		Id:          c.modelID,
		Name:        c.modelTitle,
		Type:        gemaraproj.Software,
		Description: "System under governance: " + c.modelTitle,
	}
}

// -------------------- Lexicon --------------------

// buildLexicon derives a controlled-vocabulary Lexicon from the catalog terms.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildLexicon(bundle model.Bundle, cfg gemaraConfig) (gemaraproj.Lexicon, bool) {
	cat := bundle.Catalog.Catalog
	lex := gemaraproj.Lexicon{
		Title:    cfg.modelTitle + " - Lexicon",
		Metadata: cfg.newMetadata(cfg.modelID+"-LEXICON", "Controlled vocabulary for "+cfg.modelTitle, gemaraproj.LexiconArtifact),
	}
	seen := map[string]bool{}
	add := func(entries []model.CatalogEntry) {
		for _, e := range entries {
			if e.ID == "" || seen[e.ID] {
				continue
			}
			seen[e.ID] = true
			lex.Terms = append(lex.Terms, gemaraproj.LexiconTerm{
				Id:         e.ID,
				Title:      textutil.Fallback(e.Name, e.ID),
				Definition: textutil.Fallback(e.Definition, textutil.Fallback(e.Name, e.ID)),
				Synonyms:   e.Aliases,
			})
		}
	}
	add(cat.Systems)
	add(cat.FunctionalGroups)
	add(cat.FunctionalUnits)
	add(cat.Actors)
	add(cat.AttackVectors)
	add(cat.Events)
	add(cat.States)
	add(cat.Features)
	add(cat.Modes)
	add(cat.Conditions)
	add(cat.DataTerms)
	if len(lex.Terms) == 0 {
		return gemaraproj.Lexicon{}, false
	}
	return lex, true
}

// -------------------- Mapping Document (Control -> Threat) --------------------

// buildControlThreatMapping derives a mapping document from control->threat links.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildControlThreatMapping(bundle model.Bundle, cfg gemaraConfig) (gemaraproj.MappingDocument, bool) {
	a := bundle.Architecture.AuthoredArchitecture
	byControl := mitigatedThreatsByControl(a)

	var mappings []gemaraproj.Mapping
	for _, ctrl := range a.Controls {
		if threats := byControl[ctrl.ID]; len(threats) > 0 {
			mappings = append(mappings, controlThreatMapping(ctrl, threats))
		}
	}
	if len(mappings) == 0 {
		return gemaraproj.MappingDocument{}, false
	}

	doc := gemaraproj.MappingDocument{
		Title:           cfg.modelTitle + " - Control to Threat Mapping",
		Metadata:        cfg.newMetadata(cfg.modelID+"-CONTROL-THREAT-MAPPING", "How controls relate to threats for "+cfg.modelTitle, gemaraproj.MappingDocumentArtifact),
		SourceReference: gemaraproj.TypedMapping{EntryType: gemaraproj.EntryTypeControl, ReferenceId: gemaraRefControlCatalog},
		TargetReference: gemaraproj.TypedMapping{EntryType: gemaraproj.EntryTypeThreat, ReferenceId: gemaraRefThreatCatalog},
		Mappings:        mappings,
	}
	doc.Metadata.MappingReferences = []gemaraproj.MappingReference{
		{Id: gemaraRefControlCatalog, Title: "Control Catalog", Version: Version},
		{Id: gemaraRefThreatCatalog, Title: "Threat Catalog", Version: Version},
	}
	return doc, true
}

// mitigatedThreatsByControl collects the existing threats per control from
// mitigations and related-controls.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func mitigatedThreatsByControl(a model.AuthoredArchitecture) map[string][]string {
	threatIDs := threatIDSet(a.ThreatScenarios)
	byControl := map[string][]string{}
	for _, tm := range a.ThreatMitigations {
		addKnownMitigation(byControl, tm, threatIDs)
	}
	for _, ts := range a.ThreatScenarios {
		addRelatedControls(byControl, ts)
	}
	return byControl
}

// addKnownMitigation records a mitigation whose control is named and whose
// threat exists.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func addKnownMitigation(byControl map[string][]string, tm model.ThreatMitigation, threatIDs map[string]bool) {
	if tm.ControlRef == "" || !threatIDs[tm.ThreatScenarioRef] {
		return
	}
	byControl[tm.ControlRef] = textutil.AppendUnique(byControl[tm.ControlRef], tm.ThreatScenarioRef)
}

// controlThreatMapping maps one control to the threats it mitigates.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func controlThreatMapping(ctrl model.Control, threats []string) gemaraproj.Mapping {
	var targets []gemaraproj.MappingTarget
	for _, tID := range threats {
		targets = append(targets, gemaraproj.MappingTarget{
			EntryId:   tID,
			Strength:  7,
			Rationale: fmt.Sprintf("Control %s mitigates threat %s.", ctrl.ID, tID),
		})
	}
	return gemaraproj.Mapping{
		Id:           "MAP-" + ctrl.ID,
		Source:       ctrl.ID,
		Targets:      targets,
		Relationship: gemaraproj.RelRelatesTo,
	}
}

// -------------------- L7 Audit Log --------------------

// controlVerificationEvidence retains the authored path as the raw payload
// required by Gemara's strict evidence schema.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func controlVerificationEvidence(id, collected string, ev model.ControlVerificationEvidence) gemaraproj.Evidence {
	return gemaraproj.Evidence{
		Id:          id,
		Type:        gemaraproj.EvidenceType("ControlVerification"),
		CollectedAt: gemaraproj.Datetime(collected),
		Payload:     textutil.Fallback(ev.Path, ev.Description),
		Description: textutil.Fallback(ev.Description, ev.Path),
	}
}

// buildAuditLog derives a point-in-time audit log from control verifications and risks.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildAuditLog(bundle model.Bundle, cfg gemaraConfig, date string) (gemaraproj.AuditLog, bool) {
	a := bundle.Architecture.AuthoredArchitecture
	collected := normalizeDatetime(date)
	if collected == "" {
		collected = gemaraDefaultTimestamp
	}

	var results []*gemaraproj.AuditResult
	for _, cv := range a.ControlVerifications {
		results = append(results, verificationAuditResult(cv, collected))
	}
	riskResults, usedRiskRef := residualRiskResults(a)
	results = append(results, riskResults...)
	if len(results) == 0 {
		return gemaraproj.AuditLog{}, false
	}

	log := gemaraproj.AuditLog{
		Metadata: cfg.newMetadata(cfg.modelID+"-AUDIT-LOG", "Point-in-time audit of "+cfg.modelTitle, gemaraproj.AuditLogArtifact),
		Summary:  fmt.Sprintf("Audit of %s: %d result(s) reviewed against the control catalog.", cfg.modelTitle, len(results)),
		Criteria: []gemaraproj.ArtifactMapping{{ReferenceId: gemaraRefControlCatalog, Remarks: "Control catalog defines the acceptable state."}},
		Results:  results,
		Target:   cfg.targetResource(),
	}
	log.Metadata.MappingReferences = mappingRefs(
		mappingRef(true, gemaraRefControlCatalog, "Control Catalog"),
		mappingRef(usedRiskRef, gemaraRefRiskCatalog, "Risk Catalog"),
	)
	return log, true
}

// auditResultType is the audit result type of a verification status: a pass is
// a strength, a failure a finding, anything else an observation.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func auditResultType(status string) gemaraproj.ResultType {
	switch mapVerificationResult(status) {
	case gemaraproj.Passed:
		return gemaraproj.ResultStrength
	case gemaraproj.Failed:
		return gemaraproj.ResultFinding
	default:
		return gemaraproj.ResultObservation
	}
}

// verificationAuditResult is the audit result of one control verification.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func verificationAuditResult(cv model.ControlVerification, collected string) *gemaraproj.AuditResult {
	rt := auditResultType(cv.Status)
	ar := &gemaraproj.AuditResult{
		Id:          "AUD-" + cv.ID,
		Title:       fmt.Sprintf("Verification of %s", textutil.Fallback(cv.ControlRef, cv.ID)),
		Type:        rt,
		Description: fmt.Sprintf("Control %s assessed via %s with status %s.", textutil.Fallback(cv.ControlRef, cv.ID), textutil.Fallback(cv.Method, "verification"), textutil.Fallback(cv.Status, "unknown")),
		CriteriaReference: gemaraproj.MultiEntryMapping{
			ReferenceId: gemaraRefControlCatalog,
			Entries:     []gemaraproj.ArtifactMapping{{ReferenceId: textutil.Fallback(cv.ControlRef, cv.ID)}},
		},
	}
	for i, ev := range cv.Evidence {
		ar.Evidence = append(ar.Evidence, controlVerificationEvidence(fmt.Sprintf("%s-EV-%d", ar.Id, i+1), collected, ev))
	}
	for _, f := range cv.Findings {
		ar.Recommendations = append(ar.Recommendations, gemaraproj.Recommendation{Text: f, Required: rt == gemaraproj.ResultFinding})
	}
	return ar
}

// residualRiskResults are the residual-risk observations, one per risk that
// states a residual risk, and whether any cites the risk catalog. The criterion
// is the first related control that exists in the control catalog, otherwise
// the risk itself.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func residualRiskResults(a model.AuthoredArchitecture) ([]*gemaraproj.AuditResult, bool) {
	controlIDs := controlIDSet(a)
	usedRiskRef := false
	var results []*gemaraproj.AuditResult
	for _, r := range a.Risks {
		if strings.TrimSpace(r.ResidualRisk) == "" {
			continue
		}
		result, citesRisk := residualRiskResult(r, controlIDs)
		usedRiskRef = usedRiskRef || citesRisk
		results = append(results, result)
	}
	return results, usedRiskRef
}

// residualRiskResult is the observation of one risk and whether it cites the
// risk catalog rather than a control.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func residualRiskResult(r model.Risk, controlIDs map[string]bool) (*gemaraproj.AuditResult, bool) {
	criteria := gemaraproj.MultiEntryMapping{ReferenceId: gemaraRefRiskCatalog, Entries: []gemaraproj.ArtifactMapping{{ReferenceId: r.ID}}}
	ctrl := firstKnown(r.RelatedControls, controlIDs)
	if ctrl != "" {
		criteria = gemaraproj.MultiEntryMapping{ReferenceId: gemaraRefControlCatalog, Entries: []gemaraproj.ArtifactMapping{{ReferenceId: ctrl}}}
	}
	return &gemaraproj.AuditResult{
		Id:                "AUD-" + r.ID,
		Title:             "Residual risk: " + textutil.Fallback(r.Title, r.ID),
		Type:              gemaraproj.ResultObservation,
		Description:       fmt.Sprintf("Residual risk after mitigation: %s", r.ResidualRisk),
		CriteriaReference: criteria,
	}, ctrl == ""
}

// controlIDSet is the set of control ids.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func controlIDSet(a model.AuthoredArchitecture) map[string]bool {
	ids := map[string]bool{}
	for _, c := range a.Controls {
		ids[c.ID] = true
	}
	return ids
}

// -------------------- L6 Enforcement Log --------------------

// buildEnforcementLog derives an enforcement log from POA&M remediation items.
// An action cites an evaluation-log entry only when a control verification of
// the POA&M item's risk is rendered in the evaluation log; otherwise the action
// carries no assessment reference rather than a fabricated one.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildEnforcementLog(bundle model.Bundle, cfg gemaraConfig, date string) (gemaraproj.EnforcementLog, bool) {
	a := bundle.Architecture.AuthoredArchitecture
	if len(a.POAMItems) == 0 {
		return gemaraproj.EnforcementLog{}, false
	}
	start := normalizeDatetime(date)
	if start == "" {
		start = gemaraDefaultTimestamp
	}
	findings := evaluationFindingsByRisk(a)
	builder := enforcementBuilder{cfg: cfg, findings: findings, start: start}

	usedEvalRef := false
	var actions []*gemaraproj.ActionResult
	dispositions := make([]gemaraproj.Disposition, 0, len(a.POAMItems))
	for _, p := range justifiedPOAMItems(a, findings) {
		action, citesEvaluation := builder.action(p)
		dispositions = append(dispositions, action.Disposition)
		usedEvalRef = usedEvalRef || citesEvaluation
		actions = append(actions, action)
	}

	reportUnjustifiedPOAM(cfg, a, findings)
	if len(actions) == 0 {
		// Gemara requires at least one action; with none justified there is no log.
		return gemaraproj.EnforcementLog{}, false
	}
	log := gemaraproj.EnforcementLog{
		Metadata:    cfg.newMetadata(cfg.modelID+"-ENFORCEMENT-LOG", "Remediation enforcement actions for "+cfg.modelTitle, gemaraproj.EnforcementLogArtifact),
		Disposition: aggregateDisposition(dispositions),
		Actions:     actions,
		Target:      cfg.targetResource(),
	}
	log.Metadata.MappingReferences = mappingRefs(
		mappingRef(usedEvalRef, cfg.evaluationLogID(), "Evaluation Log"),
		mappingRef(true, cfg.policyID(), "Policy"),
	)
	return log, true
}

// enforcementBuilder renders the enforcement actions of POA&M items.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type enforcementBuilder struct {
	cfg      gemaraConfig
	findings map[string][]evaluationFinding
	start    string
}

// action renders one POA&M item and reports whether it cites an evaluation-log entry.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func (b enforcementBuilder) action(p model.POAMItem) (*gemaraproj.ActionResult, bool) {
	msg := fmt.Sprintf("Remediation for %s (milestone: %s, due: %s, status: %s).",
		textutil.Fallback(p.RiskRef, p.ID), textutil.Fallback(p.Milestone, "not set"), textutil.Fallback(p.DueDate, "not set"), textutil.Fallback(p.Status, "not set"))
	steps := []gemaraproj.EnforcementStep{gemaraproj.EnforcementStep(textutil.Fallback(p.ResponsibleRole, "responsible role "+gemaraNotSet))}
	for _, art := range p.Artifacts {
		steps = append(steps, gemaraproj.EnforcementStep(textutil.Fallback(art.Path, art.Description)))
	}
	var assessments []gemaraproj.AssessmentFinding
	for _, f := range b.findings[strings.TrimSpace(p.RiskRef)] {
		assessments = append(assessments, gemaraproj.AssessmentFinding{
			Result: f.result,
			Log:    gemaraproj.EntryMapping{ReferenceId: b.cfg.evaluationLogID(), EntryId: f.entryID},
		})
	}
	return &gemaraproj.ActionResult{
		Disposition:   poamDisposition(p.Status),
		Method:        gemaraproj.EntryMapping{ReferenceId: b.cfg.policyID(), EntryId: policyRemediationMethodID},
		Message:       strPtr(msg),
		Start:         gemaraproj.Datetime(b.start),
		Steps:         steps,
		Justification: gemaraproj.Justification{Assessments: assessments},
	}, len(assessments) > 0
}

// reportUnjustifiedPOAM reports every POA&M item left out of the enforcement log.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func reportUnjustifiedPOAM(cfg gemaraConfig, a model.AuthoredArchitecture, findings map[string][]evaluationFinding) {
	for _, p := range a.POAMItems {
		if len(findings[strings.TrimSpace(p.RiskRef)]) > 0 {
			continue
		}
		cfg.addDiagnostic(validate.Diagnostic{
			Code:     "gemara.enforcement_unjustified",
			Severity: validate.SeverityWarning,
			Message:  fmt.Sprintf("POA&M item %s has no control verification of risk %q in the evaluation log; Gemara requires an assessment for every enforcement action, so the item is left out of the enforcement log", p.ID, p.RiskRef),
			Path:     "poamItems[" + p.ID + "]",
		})
	}
}

// justifiedPOAMItems returns the POA&M items whose risk has at least one
// control verification in the evaluation log. Gemara requires every
// enforcement action to cite an assessment finding, so only these become
// actions; the others are reported as gemara.enforcement_unjustified.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
func justifiedPOAMItems(a model.AuthoredArchitecture, findings map[string][]evaluationFinding) []model.POAMItem {
	out := make([]model.POAMItem, 0, len(a.POAMItems))
	for _, p := range a.POAMItems {
		if len(findings[strings.TrimSpace(p.RiskRef)]) > 0 {
			out = append(out, p)
		}
	}
	return out
}

// evaluationFinding is an evaluation-log assessment entry an enforcement action may cite.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type evaluationFinding struct {
	entryID string
	result  gemaraproj.Result
}

// evaluationFindingsByRisk indexes, per risk id, the control verifications the
// evaluation log renders (those whose control exists) with their result.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func evaluationFindingsByRisk(a model.AuthoredArchitecture) map[string][]evaluationFinding {
	controlIDs := controlIDSet(a)
	out := map[string][]evaluationFinding{}
	for _, cv := range a.ControlVerifications {
		if !controlIDs[cv.ControlRef] {
			continue
		}
		for _, r := range nonEmptyTrimmed(cv.RiskRefs) {
			out[r] = append(out[r], evaluationFinding{entryID: cv.ID, result: mapVerificationResult(cv.Status)})
		}
	}
	return out
}

// nonEmptyTrimmed returns the trimmed, non-empty values.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func nonEmptyTrimmed(values []string) []string {
	var out []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// poamDisposition maps a POA&M status to the enforcement disposition it
// supports: only a completed item was enforced and only an accepted one is
// tolerated. Planned, in-progress, deferred or unset items have no enforcement
// outcome yet, so they are Undetermined rather than claimed as Enforced.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
func poamDisposition(status string) gemaraproj.Disposition {
	switch {
	case strings.EqualFold(strings.TrimSpace(status), "completed"):
		return gemaraproj.DispositionEnforced
	case isAcceptedStatus(status):
		return gemaraproj.DispositionTolerated
	default:
		return gemaraproj.DispositionUndetermined
	}
}

// aggregateDisposition rolls action dispositions up for the log: Undetermined
// while any item is unresolved, otherwise Enforced if any item was enforced,
// otherwise Tolerated.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
func aggregateDisposition(ds []gemaraproj.Disposition) gemaraproj.Disposition {
	out := gemaraproj.DispositionTolerated
	for _, d := range ds {
		switch d {
		case gemaraproj.DispositionUndetermined:
			return gemaraproj.DispositionUndetermined
		case gemaraproj.DispositionEnforced:
			out = gemaraproj.DispositionEnforced
		}
	}
	if len(ds) == 0 {
		return gemaraproj.DispositionUndetermined
	}
	return out
}

// -------------------- small helpers --------------------

// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func isAcceptedStatus(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "accepted", "risk-accepted", "accept", "waived", "tolerated":
		return true
	default:
		return false
	}
}

// firstKnown returns the first id in ss that is in known, or "".
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func firstKnown(ss []string, known map[string]bool) string {
	for _, s := range ss {
		if s = strings.TrimSpace(s); s != "" && known[s] {
			return s
		}
	}
	return ""
}

// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func strPtr(s string) *string { return &s }
