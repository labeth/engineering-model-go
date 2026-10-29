// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

// Extended Gemara artifacts derived from existing model data (no new authored
// inputs required):
//   - Lexicon         <- catalog terms (controlled vocabulary)
//   - Mapping Document <- control -> threat relationships
//   - L7 Audit Log     <- control verifications + risks (point-in-time review)
//   - L6 Enforcement Log <- POA&M items (remediative actions)

import (
	"fmt"
	"strings"

	gemara "github.com/gemaraproj/go-gemara"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func (c gemaraConfig) targetResource() gemara.Resource {
	return gemara.Resource{
		Id:          c.modelID,
		Name:        c.modelTitle,
		Type:        gemara.Software,
		Description: "System under governance: " + c.modelTitle,
	}
}

// -------------------- Lexicon --------------------

// buildLexicon derives a controlled-vocabulary Lexicon from the catalog terms.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildLexicon(bundle model.Bundle, cfg gemaraConfig) (gemara.Lexicon, bool) {
	cat := bundle.Catalog.Catalog
	lex := gemara.Lexicon{
		Title:    cfg.modelTitle + " - Lexicon",
		Metadata: cfg.newMetadata(cfg.modelID+"-LEXICON", "Controlled vocabulary for "+cfg.modelTitle, gemara.LexiconArtifact),
	}
	seen := map[string]bool{}
	add := func(entries []model.CatalogEntry) {
		for _, e := range entries {
			if e.ID == "" || seen[e.ID] {
				continue
			}
			seen[e.ID] = true
			lex.Terms = append(lex.Terms, gemara.LexiconTerm{
				Id:         e.ID,
				Title:      fallback(e.Name, e.ID),
				Definition: fallback(e.Definition, fallback(e.Name, e.ID)),
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
		return gemara.Lexicon{}, false
	}
	return lex, true
}

// -------------------- Mapping Document (Control -> Threat) --------------------

// buildControlThreatMapping derives a mapping document from control->threat links.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildControlThreatMapping(bundle model.Bundle, cfg gemaraConfig) (gemara.MappingDocument, bool) {
	a := bundle.Architecture.AuthoredArchitecture
	threatIDs := map[string]bool{}
	for _, ts := range a.ThreatScenarios {
		threatIDs[ts.ID] = true
	}
	// Collect threats per control from mitigations and related-controls.
	threatsByControl := map[string][]string{}
	for _, tm := range a.ThreatMitigations {
		if tm.ControlRef != "" && threatIDs[tm.ThreatScenarioRef] {
			threatsByControl[tm.ControlRef] = appendUnique(threatsByControl[tm.ControlRef], tm.ThreatScenarioRef)
		}
	}
	for _, ts := range a.ThreatScenarios {
		for _, ctrl := range ts.RelatedControls {
			threatsByControl[ctrl] = appendUnique(threatsByControl[ctrl], ts.ID)
		}
	}

	var mappings []gemara.Mapping
	for _, ctrl := range a.Controls {
		threats := threatsByControl[ctrl.ID]
		if len(threats) == 0 {
			continue
		}
		var targets []gemara.MappingTarget
		for _, tID := range threats {
			targets = append(targets, gemara.MappingTarget{
				EntryId:   tID,
				Strength:  7,
				Rationale: fmt.Sprintf("Control %s mitigates threat %s.", ctrl.ID, tID),
			})
		}
		mappings = append(mappings, gemara.Mapping{
			Id:           "MAP-" + ctrl.ID,
			Source:       ctrl.ID,
			Targets:      targets,
			Relationship: gemara.RelRelatesTo,
		})
	}
	if len(mappings) == 0 {
		return gemara.MappingDocument{}, false
	}

	doc := gemara.MappingDocument{
		Title:           cfg.modelTitle + " - Control to Threat Mapping",
		Metadata:        cfg.newMetadata(cfg.modelID+"-CONTROL-THREAT-MAPPING", "How controls relate to threats for "+cfg.modelTitle, gemara.MappingDocumentArtifact),
		SourceReference: gemara.TypedMapping{EntryType: gemara.EntryTypeControl, ReferenceId: gemaraRefControlCatalog},
		TargetReference: gemara.TypedMapping{EntryType: gemara.EntryTypeThreat, ReferenceId: gemaraRefThreatCatalog},
		Mappings:        mappings,
	}
	doc.Metadata.MappingReferences = []gemara.MappingReference{
		{Id: gemaraRefControlCatalog, Title: "Control Catalog", Version: GemaraVersion},
		{Id: gemaraRefThreatCatalog, Title: "Threat Catalog", Version: GemaraVersion},
	}
	return doc, true
}

// -------------------- L7 Audit Log --------------------

// controlVerificationEvidence retains the authored path as the raw payload
// required by Gemara's strict evidence schema.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func controlVerificationEvidence(id, collected string, ev model.ControlVerificationEvidence) gemara.Evidence {
	return gemara.Evidence{
		Id:          id,
		Type:        gemara.EvidenceType("ControlVerification"),
		CollectedAt: gemara.Datetime(collected),
		Payload:     fallback(ev.Path, ev.Description),
		Description: fallback(ev.Description, ev.Path),
	}
}

// buildAuditLog derives a point-in-time audit log from control verifications and risks.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildAuditLog(bundle model.Bundle, cfg gemaraConfig, date string) (gemara.AuditLog, bool) {
	a := bundle.Architecture.AuthoredArchitecture
	collected := normalizeDatetime(date)
	if collected == "" {
		collected = gemaraDefaultTimestamp
	}

	var results []*gemara.AuditResult
	for _, cv := range a.ControlVerifications {
		rt := gemara.ResultObservation
		switch mapVerificationResult(cv.Status) {
		case gemara.Passed:
			rt = gemara.ResultStrength
		case gemara.Failed:
			rt = gemara.ResultFinding
		case gemara.NeedsReview, gemara.Unknown:
			rt = gemara.ResultObservation
		}
		ar := &gemara.AuditResult{
			Id:          "AUD-" + cv.ID,
			Title:       fmt.Sprintf("Verification of %s", fallback(cv.ControlRef, cv.ID)),
			Type:        rt,
			Description: fmt.Sprintf("Control %s assessed via %s with status %s.", fallback(cv.ControlRef, cv.ID), fallback(cv.Method, "verification"), fallback(cv.Status, "unknown")),
			CriteriaReference: gemara.MultiEntryMapping{
				ReferenceId: gemaraRefControlCatalog,
				Entries:     []gemara.ArtifactMapping{{ReferenceId: fallback(cv.ControlRef, cv.ID)}},
			},
		}
		for i, ev := range cv.Evidence {
			ar.Evidence = append(ar.Evidence, controlVerificationEvidence(
				fmt.Sprintf("%s-EV-%d", ar.Id, i+1),
				collected,
				ev,
			))
		}
		for _, f := range cv.Findings {
			ar.Recommendations = append(ar.Recommendations, gemara.Recommendation{Text: f, Required: rt == gemara.ResultFinding})
		}
		results = append(results, ar)
	}
	// Residual-risk observations. The criterion is the first related control
	// that exists in the control catalog, otherwise the risk itself.
	controlIDs := map[string]bool{}
	for _, c := range a.Controls {
		controlIDs[c.ID] = true
	}
	usedRiskRef := false
	for _, r := range a.Risks {
		if strings.TrimSpace(r.ResidualRisk) == "" {
			continue
		}
		criteria := gemara.MultiEntryMapping{ReferenceId: gemaraRefRiskCatalog, Entries: []gemara.ArtifactMapping{{ReferenceId: r.ID}}}
		if ctrl := firstKnown(r.RelatedControls, controlIDs); ctrl != "" {
			criteria = gemara.MultiEntryMapping{ReferenceId: gemaraRefControlCatalog, Entries: []gemara.ArtifactMapping{{ReferenceId: ctrl}}}
		} else {
			usedRiskRef = true
		}
		results = append(results, &gemara.AuditResult{
			Id:                "AUD-" + r.ID,
			Title:             "Residual risk: " + fallback(r.Title, r.ID),
			Type:              gemara.ResultObservation,
			Description:       fmt.Sprintf("Residual risk after mitigation: %s", r.ResidualRisk),
			CriteriaReference: criteria,
		})
	}
	if len(results) == 0 {
		return gemara.AuditLog{}, false
	}

	log := gemara.AuditLog{
		Metadata: cfg.newMetadata(cfg.modelID+"-AUDIT-LOG", "Point-in-time audit of "+cfg.modelTitle, gemara.AuditLogArtifact),
		Summary:  fmt.Sprintf("Audit of %s: %d result(s) reviewed against the control catalog.", cfg.modelTitle, len(results)),
		Criteria: []gemara.ArtifactMapping{{ReferenceId: gemaraRefControlCatalog, Remarks: "Control catalog defines the acceptable state."}},
		Results:  results,
		Target:   cfg.targetResource(),
	}
	log.Metadata.MappingReferences = mappingRefs(
		mappingRef(true, gemaraRefControlCatalog, "Control Catalog"),
		mappingRef(usedRiskRef, gemaraRefRiskCatalog, "Risk Catalog"),
	)
	return log, true
}

// -------------------- L6 Enforcement Log --------------------

// buildEnforcementLog derives an enforcement log from POA&M remediation items.
// An action cites an evaluation-log entry only when a control verification of
// the POA&M item's risk is rendered in the evaluation log; otherwise the action
// carries no assessment reference rather than a fabricated one.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildEnforcementLog(bundle model.Bundle, cfg gemaraConfig, date string) (gemara.EnforcementLog, bool) {
	a := bundle.Architecture.AuthoredArchitecture
	if len(a.POAMItems) == 0 {
		return gemara.EnforcementLog{}, false
	}
	start := normalizeDatetime(date)
	if start == "" {
		start = gemaraDefaultTimestamp
	}
	findings := evaluationFindingsByRisk(a)

	usedEvalRef := false
	var actions []*gemara.ActionResult
	dispositions := make([]gemara.Disposition, 0, len(a.POAMItems))
	for _, p := range a.POAMItems {
		disp := poamDisposition(p.Status)
		dispositions = append(dispositions, disp)
		msg := fmt.Sprintf("Remediation for %s (milestone: %s, due: %s, status: %s).",
			fallback(p.RiskRef, p.ID), fallback(p.Milestone, "not set"), fallback(p.DueDate, "not set"), fallback(p.Status, "not set"))
		steps := []gemara.EnforcementStep{gemara.EnforcementStep(fallback(p.ResponsibleRole, "responsible role "+gemaraNotSet))}
		for _, art := range p.Artifacts {
			steps = append(steps, gemara.EnforcementStep(fallback(art.Path, art.Description)))
		}
		var assessments []gemara.AssessmentFinding
		for _, f := range findings[strings.TrimSpace(p.RiskRef)] {
			assessments = append(assessments, gemara.AssessmentFinding{
				Result: f.result,
				Log:    gemara.EntryMapping{ReferenceId: cfg.evaluationLogID(), EntryId: f.entryID},
			})
			usedEvalRef = true
		}
		if len(assessments) == 0 {
			cfg.addDiagnostic(validate.Diagnostic{
				Code:     "gemara.enforcement_unjustified",
				Severity: validate.SeverityWarning,
				Message:  fmt.Sprintf("POA&M item %s has no control verification of risk %q in the evaluation log; the enforcement action cites no assessment", p.ID, p.RiskRef),
				Path:     "poamItems[" + p.ID + "]",
			})
		}
		actions = append(actions, &gemara.ActionResult{
			Disposition:   disp,
			Method:        gemara.EntryMapping{ReferenceId: cfg.policyID(), EntryId: policyRemediationMethodID},
			Message:       strPtr(msg),
			Start:         gemara.Datetime(start),
			Steps:         steps,
			Justification: gemara.Justification{Assessments: assessments},
		})
	}

	log := gemara.EnforcementLog{
		Metadata:    cfg.newMetadata(cfg.modelID+"-ENFORCEMENT-LOG", "Remediation enforcement actions for "+cfg.modelTitle, gemara.EnforcementLogArtifact),
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

// evaluationFinding is an evaluation-log assessment entry an enforcement action may cite.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type evaluationFinding struct {
	entryID string
	result  gemara.Result
}

// evaluationFindingsByRisk indexes, per risk id, the control verifications the
// evaluation log renders (those whose control exists) with their result.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func evaluationFindingsByRisk(a model.AuthoredArchitecture) map[string][]evaluationFinding {
	controlIDs := map[string]bool{}
	for _, c := range a.Controls {
		controlIDs[c.ID] = true
	}
	out := map[string][]evaluationFinding{}
	for _, cv := range a.ControlVerifications {
		if !controlIDs[cv.ControlRef] {
			continue
		}
		for _, r := range cv.RiskRefs {
			r = strings.TrimSpace(r)
			if r != "" {
				out[r] = append(out[r], evaluationFinding{entryID: cv.ID, result: mapVerificationResult(cv.Status)})
			}
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
func poamDisposition(status string) gemara.Disposition {
	switch {
	case strings.EqualFold(strings.TrimSpace(status), "completed"):
		return gemara.DispositionEnforced
	case isAcceptedStatus(status):
		return gemara.DispositionTolerated
	default:
		return gemara.DispositionUndetermined
	}
}

// aggregateDisposition rolls action dispositions up for the log: Undetermined
// while any item is unresolved, otherwise Enforced if any item was enforced,
// otherwise Tolerated.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
func aggregateDisposition(ds []gemara.Disposition) gemara.Disposition {
	out := gemara.DispositionTolerated
	for _, d := range ds {
		switch d {
		case gemara.DispositionUndetermined:
			return gemara.DispositionUndetermined
		case gemara.DispositionEnforced:
			out = gemara.DispositionEnforced
		}
	}
	if len(ds) == 0 {
		return gemara.DispositionUndetermined
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
