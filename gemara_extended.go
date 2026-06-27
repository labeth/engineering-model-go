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
)

// Mapping-reference ids for cross-artifact links in the extended documents.
const (
	gemaraRefEvaluationLog = "ENGMOD-EVALUATION-LOG"
	gemaraRefPolicy        = "ENGMOD-POLICY"
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
			ar.Evidence = append(ar.Evidence, gemara.Evidence{
				Id:          fmt.Sprintf("%s-EV-%d", ar.Id, i+1),
				Type:        gemara.EvidenceType("ControlVerification"),
				CollectedAt: gemara.Datetime(collected),
				Description: fallback(ev.Description, ev.Path),
			})
		}
		for _, f := range cv.Findings {
			ar.Recommendations = append(ar.Recommendations, gemara.Recommendation{Text: f, Required: rt == gemara.ResultFinding})
		}
		results = append(results, ar)
	}
	// Residual-risk observations.
	for _, r := range a.Risks {
		if strings.TrimSpace(r.ResidualRisk) == "" {
			continue
		}
		results = append(results, &gemara.AuditResult{
			Id:          "AUD-" + r.ID,
			Title:       "Residual risk: " + fallback(r.Title, r.ID),
			Type:        gemara.ResultObservation,
			Description: fmt.Sprintf("Residual risk after mitigation: %s", r.ResidualRisk),
			CriteriaReference: gemara.MultiEntryMapping{
				ReferenceId: gemaraRefControlCatalog,
				Entries:     []gemara.ArtifactMapping{{ReferenceId: firstNonEmpty(r.RelatedControls)}},
			},
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
	log.Metadata.MappingReferences = []gemara.MappingReference{
		{Id: gemaraRefControlCatalog, Title: "Control Catalog", Version: GemaraVersion},
		{Id: gemaraRefEvaluationLog, Title: "Evaluation Log", Version: GemaraVersion},
	}
	return log, true
}

// -------------------- L6 Enforcement Log --------------------

// buildEnforcementLog derives an enforcement log from POA&M remediation items.
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

	riskTitle := map[string]string{}
	for _, r := range a.Risks {
		riskTitle[r.ID] = fallback(r.Title, r.ID)
	}

	aggregate := gemara.DispositionClear
	var actions []*gemara.ActionResult
	for _, p := range a.POAMItems {
		disp := gemara.DispositionEnforced
		if isAcceptedStatus(p.Status) {
			disp = gemara.DispositionTolerated
		}
		if aggregate == gemara.DispositionClear || (disp == gemara.DispositionEnforced && aggregate == gemara.DispositionTolerated) {
			aggregate = disp
		}
		msg := fmt.Sprintf("Remediation for %s (milestone: %s, due: %s, status: %s).",
			fallback(p.RiskRef, p.ID), fallback(p.Milestone, "n/a"), fallback(p.DueDate, "n/a"), fallback(p.Status, "open"))
		steps := []gemara.EnforcementStep{gemara.EnforcementStep(fallback(p.ResponsibleRole, "remediation-owner"))}
		for _, art := range p.Artifacts {
			steps = append(steps, gemara.EnforcementStep(fallback(art.Path, art.Description)))
		}
		actions = append(actions, &gemara.ActionResult{
			Disposition: disp,
			Method:      gemara.EntryMapping{ReferenceId: gemaraRefPolicy, EntryId: "remediation"},
			Message:     strPtr(msg),
			Start:       gemara.Datetime(start),
			Steps:       steps,
			Justification: gemara.Justification{
				Assessments: []gemara.AssessmentFinding{{
					Result: gemara.Failed,
					Log:    gemara.EntryMapping{ReferenceId: gemaraRefEvaluationLog, EntryId: fallback(p.RiskRef, p.ID)},
				}},
			},
		})
	}

	log := gemara.EnforcementLog{
		Metadata:    cfg.newMetadata(cfg.modelID+"-ENFORCEMENT-LOG", "Remediation enforcement actions for "+cfg.modelTitle, gemara.EnforcementLogArtifact),
		Disposition: aggregate,
		Actions:     actions,
		Target:      cfg.targetResource(),
	}
	log.Metadata.MappingReferences = []gemara.MappingReference{
		{Id: gemaraRefEvaluationLog, Title: "Evaluation Log", Version: GemaraVersion},
		{Id: gemaraRefPolicy, Title: "Policy", Version: GemaraVersion},
	}
	_ = riskTitle
	return log, true
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

// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func firstNonEmpty(ss []string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func strPtr(s string) *string { return &s }
