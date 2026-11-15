// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// aiAgentNames are the agents engineering-flow registers (init/installer.mjs
// reviewers and workers lists, engineering-flow v0.6). They produce input
// for people; a DO-178C reviewer, verifier, producer or approver names a
// person (ADR-EMG-035). Kept in the engine so the check needs no flow
// install; extend it when the flow registers a new agent.
var aiAgentNames = []string{
	"architecture-review", "classification-review", "ef-experimenter", "ef-implementer", "ef-modeler",
	"export-control-review", "governance-review", "operability-review", "performance-review",
	"quality-review", "safety-review", "security-review", "verification-review",
}

// AIAgentNames returns the registered engineering-flow agent names that
// aviation.ai_actor_as_credit rejects as DO-178C credit holders, sorted.
// TRLC-LINKS: REQ-EMG-133
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-AIRBORNE-ASSURANCE-EXPORTER
func AIAgentNames() []string {
	return append([]string(nil), aiAgentNames...)
}

// evidenceStateRank orders the evidence states for the tier floors.
var evidenceStateRank = map[string]int{
	"declared": 1, "planned": 2, "executed": 3, "reviewed": 4, "accepted": 5, "authority-accepted": 6,
}

// planFamilies are the five DO-178C plans (lifecycle-data categories).
var planFamilies = map[string]bool{
	"PLAN-SOFTWARE-ASPECTS-CERTIFICATION": true, "SOFTWARE-DEVELOPMENT-PLAN": true, "SOFTWARE-VERIFICATION-PLAN": true,
	"SOFTWARE-CONFIGURATION-MANAGEMENT-PLAN": true, "SOFTWARE-QUALITY-ASSURANCE-PLAN": true,
}

// levelCoverageMetric is the structural coverage metric each software level
// needs from tier product (A MC/DC, B decision, C statement).
var levelCoverageMetric = map[string]string{"A": "mcdc", "B": "decision", "C": "statement"}

// aviationTierDiagnostics reports, in a model with lifecycle and an aviation
// document: the state codes the load no longer fails on (they are tier
// keyed); aviation.authority_state_unevidenced (error) for an
// authority-accepted state without a liaison authority record; and, with the
// safety bundle, aviation.state_below_tier (warning; tier keyed) for each
// record below the tier floor and aviation.ai_actor_as_credit (error) for an
// AI agent as reviewer, producer, verifier or approver.
// TRLC-LINKS: REQ-EMG-132, REQ-EMG-133
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-AIRBORNE-ASSURANCE-EXPORTER
func aviationTierDiagnostics(b model.Bundle) []Diagnostic {
	_, rank, ok := EffectiveTier(b)
	if !ok || b.Aviation == nil {
		return nil
	}
	a := b.Aviation.Aviation
	var diags []Diagnostic
	_, state := model.SplitAviationDiagnostics(model.ValidateAviation(*b.Aviation, b.Requirements))
	for _, d := range state {
		diags = append(diags, Diagnostic{Code: d.Code, Severity: SeverityWarning, Message: d.Message, Path: d.Path})
	}
	diags = append(diags, authorityStateDiagnostics(a)...)
	if !b.Manifest.Lifecycle.HasBundle(model.BundleSafety) {
		return diags
	}
	diags = append(diags, aviationFloorDiagnostics(a, rank)...)
	return append(diags, aiCreditDiagnostics(a)...)
}

// authorityStateDiagnostics reports each record in state authority-accepted
// with no liaison entry that is authority-accepted, has an authorityRecord
// and covers the record (no submissionRefs, or ones naming it).
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func authorityStateDiagnostics(a model.AviationProfile) []Diagnostic {
	var diags []Diagnostic
	for _, r := range aviationRecords(a) {
		if r.state == "authority-accepted" && !authorityEvidenced(a, r.id) {
			diags = append(diags, Diagnostic{Code: "aviation.authority_state_unevidenced", Severity: SeverityError, Path: r.path,
				Message: fmt.Sprintf("%s is authority-accepted but no liaison entry records the authority's acceptance (authorityState authority-accepted with an authorityRecord); record the authority record or lower the state", r.id)})
		}
	}
	return diags
}

// authorityEvidenced reports whether a liaison entry is authority-accepted,
// has an authorityRecord and covers the record id (no submissionRefs, or
// ones naming it).
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func authorityEvidenced(a model.AviationProfile, id string) bool {
	for _, l := range a.Liaison {
		if l.AuthorityState == "authority-accepted" && strings.TrimSpace(l.AuthorityRecord.ID) != "" && liaisonCovers(l, id) {
			return true
		}
	}
	return false
}

// liaisonCovers reports whether a liaison entry has no submissionRefs or names the record id.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func liaisonCovers(l model.CertificationLiaison, id string) bool {
	return len(l.SubmissionRefs) == 0 || containsTrimmed(l.SubmissionRefs, id)
}

// aviationRecord is one aviation evidence record with a state.
type aviationRecord struct {
	id, path, state, family, rationale string
	kind                               string // lifecycleData, execution, other
}

// recordList collects aviation records in document order.
type recordList struct {
	out []aviationRecord
}

// recordSection names the kind of the records of one section of the aviation document and the section.
type recordSection struct{ kind, name string }

// add appends a record of a section with its id and state.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (l *recordList) add(sec recordSection, id, state string) {
	l.out = append(l.out, aviationRecord{id: id, path: "aviation." + sec.name + "." + id, state: strings.TrimSpace(state), kind: sec.kind})
}

// aviationRecords lists the aviation records that carry an evidence state,
// in document order.
// TRLC-LINKS: REQ-EMG-132
func aviationRecords(a model.AviationProfile) []aviationRecord {
	var l recordList
	l.addLifecycleAndObjectives(a)
	l.addPlanningRecords(a)
	l.addBuildRecords(a)
	l.addReviewRecords(a)
	return l.out
}

// addLifecycleAndObjectives adds the lifecycle data and the applicable objectives.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (l *recordList) addLifecycleAndObjectives(a model.AviationProfile) {
	for _, v := range a.LifecycleData {
		l.out = append(l.out, aviationRecord{id: v.ID, path: "aviation.lifecycleData." + v.ID, state: strings.TrimSpace(v.State), family: v.CategoryCode, rationale: v.Rationale, kind: "lifecycleData"})
	}
	for _, v := range a.Objectives {
		if strings.EqualFold(strings.TrimSpace(v.Applicability), "not-applicable") {
			// Applicability comes from the level's objectives, not the tier.
			continue
		}
		l.out = append(l.out, aviationRecord{id: v.ID, path: "aviation.objectives." + v.ID, state: strings.TrimSpace(v.State), rationale: v.Rationale, kind: "other"})
	}
}

// addPlanningRecords adds the requirement assurance, verification cases, procedures, executions and baselines.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (l *recordList) addPlanningRecords(a model.AviationProfile) {
	other := func(name string) recordSection { return recordSection{"other", name} }
	for _, v := range a.RequirementAssurance {
		l.add(other("requirementAssurance"), v.RequirementID, v.State)
	}
	for _, v := range a.VerificationCases {
		l.add(other("verificationCases"), v.ID, v.State)
	}
	for _, v := range a.Procedures {
		l.add(other("procedures"), v.ID, v.State)
	}
	for _, v := range a.Executions {
		l.add(recordSection{"execution", "executions"}, v.ID, v.Status)
	}
	for _, v := range a.Baselines {
		l.add(other("baselines"), v.ID, v.State)
	}
}

// addBuildRecords adds the builds, coverage and tools.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (l *recordList) addBuildRecords(a model.AviationProfile) {
	for _, v := range a.Builds {
		l.add(recordSection{"other", "builds"}, v.ID, v.Status)
	}
	for _, v := range a.Coverage {
		l.add(recordSection{"coverage", "coverage"}, v.ID, v.Status)
	}
	for _, v := range a.Tools {
		l.add(recordSection{"other", "tools"}, v.ID, v.Status)
	}
}

// addReviewRecords adds the reviews, approvals, independence records, problem reports and SQA audits.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (l *recordList) addReviewRecords(a model.AviationProfile) {
	other := func(name string) recordSection { return recordSection{"other", name} }
	for _, v := range a.Reviews {
		l.add(other("reviews"), v.ID, v.State)
	}
	for _, v := range a.Approvals {
		l.add(other("approvals"), v.ID, v.State)
	}
	for _, v := range a.Independence {
		l.add(other("independence"), v.ID, v.State)
	}
	for _, v := range a.ProblemReports {
		l.add(other("problemReports"), v.ID, v.State)
	}
	for _, v := range a.SQAAudits {
		l.add(other("sqaAudits"), v.ID, v.State)
	}
}

// floorFor returns the minimum evidence state a record needs at a tier
// rank, or "" when there is none: plans planned at mvp and reviewed from
// product; executions executed at product; every record accepted at
// operational (ADR-EMG-035).
// TRLC-LINKS: REQ-EMG-132
func floorFor(r aviationRecord, rank int) string {
	if rank >= model.TierRank(model.TierOperational) {
		return "accepted"
	}
	if rank >= model.TierRank(model.TierProduct) {
		if floor := productFloor(r); floor != "" {
			return floor
		}
	}
	if rank >= model.TierRank(model.TierMVP) && isPlanRecord(r) {
		return "planned"
	}
	return ""
}

// productFloor is the floor of a record at tier product: reviewed for a plan, executed for an execution.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func productFloor(r aviationRecord) string {
	switch {
	case isPlanRecord(r):
		return "reviewed"
	case r.kind == "execution":
		return "executed"
	}
	return ""
}

// isPlanRecord reports whether a record is lifecycle data of one of the five plans.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func isPlanRecord(r aviationRecord) bool {
	return r.kind == "lifecycleData" && planFamilies[r.family]
}

// AviationFloorGaps returns the aviation records below the evidence floor of
// a tier rank: path, record ID, state and floor; plus, from product, a gap
// for a software level A, B or C without structural coverage of its metric.
// not-applicable satisfies a floor only with a rationale.
// TRLC-LINKS: REQ-EMG-132, REQ-EMG-134
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func AviationFloorGaps(a model.AviationProfile, rank int) []AviationFloorGap {
	var out []AviationFloorGap
	for _, r := range aviationRecords(a) {
		if gap, ok := recordFloorGap(r, rank); ok {
			out = append(out, gap)
		}
	}
	if gap, ok := coverageFloorGap(a, rank); ok {
		out = append(out, gap)
	}
	return out
}

// recordFloorGap is the gap of a record below its floor at the rank; none for
// a record without a floor or one that is not-applicable with a rationale.
// TRLC-LINKS: REQ-EMG-132, REQ-EMG-134
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func recordFloorGap(r aviationRecord, rank int) (AviationFloorGap, bool) {
	floor := floorFor(r, rank)
	if floor == "" || r.state == "not-applicable" && strings.TrimSpace(r.rationale) != "" {
		return AviationFloorGap{}, false
	}
	if evidenceStateRank[r.state] >= evidenceStateRank[floor] {
		return AviationFloorGap{}, false
	}
	return AviationFloorGap{Path: r.path, ID: r.id, State: r.state, Floor: floor}, true
}

// coverageFloorGap is the missing structural coverage of the metric a software
// level needs from tier product.
// TRLC-LINKS: REQ-EMG-132, REQ-EMG-134
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func coverageFloorGap(a model.AviationProfile, rank int) (AviationFloorGap, bool) {
	metric := levelCoverageMetric[strings.TrimSpace(a.Profile.SoftwareLevel)]
	if metric == "" || rank < model.TierRank(model.TierProduct) {
		return AviationFloorGap{}, false
	}
	for _, c := range a.Coverage {
		if strings.EqualFold(strings.TrimSpace(c.Metric), metric) {
			return AviationFloorGap{}, false
		}
	}
	return AviationFloorGap{Path: "aviation.coverage", ID: "coverage:" + metric, State: "missing", Floor: metric}, true
}

// AviationFloorGap is one aviation record below the evidence floor of a
// tier, or (ID coverage:<metric>) a missing coverage metric.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
type AviationFloorGap struct {
	Path  string `json:"path"`
	ID    string `json:"id"`
	State string `json:"state"`
	Floor string `json:"floor"`
}

// aviationFloorDiagnostics reports aviation.state_below_tier for each floor
// gap at the project tier.
// TRLC-LINKS: REQ-EMG-132
func aviationFloorDiagnostics(a model.AviationProfile, rank int) []Diagnostic {
	var diags []Diagnostic
	tier := model.Tiers[rank-1]
	for _, g := range AviationFloorGaps(a, rank) {
		msg := fmt.Sprintf("%s is %s, below the %s floor of tier %s; work done before it reaches the floor earns no credit", g.ID, g.State, g.Floor, tier)
		if strings.HasPrefix(g.ID, "coverage:") {
			msg = fmt.Sprintf("software level %s needs %s structural coverage from tier product; add a coverage record with metric %s", a.Profile.SoftwareLevel, g.Floor, g.Floor)
		}
		diags = append(diags, Diagnostic{Code: "aviation.state_below_tier", Severity: SeverityWarning, Path: g.Path, Message: msg})
	}
	return diags
}

// aiCreditDiagnostics reports aviation.ai_actor_as_credit at each review
// reviewer, independence producer or verifier, execution producer or
// verifier and approval approver that names a registered engineering-flow
// agent (compared trimmed and case-insensitively), sorted by path.
// TRLC-LINKS: REQ-EMG-133
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func aiCreditDiagnostics(a model.AviationProfile) []Diagnostic {
	agents := setOf(aiAgentNames)
	var diags []Diagnostic
	check := func(path, field, value string) {
		name := strings.ToLower(strings.TrimSpace(value))
		if !agents[name] {
			return
		}
		diags = append(diags, Diagnostic{Code: "aviation.ai_actor_as_credit", Severity: SeverityError, Path: path + "." + field,
			Message: fmt.Sprintf("%s names the AI agent %s: AI output is input to people and never holds assurance credit; name the person who reviewed, verified, produced (directed and accepted the agent's output) or approved, and record the agent as a tool with creditClaimed: false", field, name)})
	}
	for _, v := range a.Reviews {
		check("aviation.reviews."+v.ID, "reviewer", v.Reviewer)
	}
	for _, v := range a.Independence {
		check("aviation.independence."+v.ID, "producer", v.Producer)
		check("aviation.independence."+v.ID, "verifier", v.Verifier)
	}
	for _, v := range a.Executions {
		check("aviation.executions."+v.ID, "producer", v.Producer)
		check("aviation.executions."+v.ID, "verifier", v.Verifier)
	}
	for _, v := range a.Approvals {
		check("aviation.approvals."+v.ID, "approver", v.Approver)
	}
	sort.SliceStable(diags, func(i, j int) bool { return diags[i].Path < diags[j].Path })
	return diags
}
