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
	evidenced := func(id string) bool {
		for _, l := range a.Liaison {
			if l.AuthorityState == "authority-accepted" && strings.TrimSpace(l.AuthorityRecord.ID) != "" &&
				(len(l.SubmissionRefs) == 0 || containsTrimmed(l.SubmissionRefs, id)) {
				return true
			}
		}
		return false
	}
	for _, r := range aviationRecords(a) {
		if r.state == "authority-accepted" && !evidenced(r.id) {
			diags = append(diags, Diagnostic{Code: "aviation.authority_state_unevidenced", Severity: SeverityError, Path: r.path,
				Message: fmt.Sprintf("%s is authority-accepted but no liaison entry records the authority's acceptance (authorityState authority-accepted with an authorityRecord); record the authority record or lower the state", r.id)})
		}
	}
	return diags
}

// aviationRecord is one aviation evidence record with a state.
type aviationRecord struct {
	id, path, state, family, rationale string
	kind                               string // lifecycleData, execution, other
}

// aviationRecords lists the aviation records that carry an evidence state,
// in document order.
// TRLC-LINKS: REQ-EMG-132
func aviationRecords(a model.AviationProfile) []aviationRecord {
	var out []aviationRecord
	add := func(kind, section, id, state string) {
		out = append(out, aviationRecord{id: id, path: "aviation." + section + "." + id, state: strings.TrimSpace(state), kind: kind})
	}
	for _, v := range a.LifecycleData {
		out = append(out, aviationRecord{id: v.ID, path: "aviation.lifecycleData." + v.ID, state: strings.TrimSpace(v.State), family: v.CategoryCode, rationale: v.Rationale, kind: "lifecycleData"})
	}
	for _, v := range a.Objectives {
		if strings.EqualFold(strings.TrimSpace(v.Applicability), "not-applicable") {
			// Applicability comes from the level's objectives, not the tier.
			continue
		}
		out = append(out, aviationRecord{id: v.ID, path: "aviation.objectives." + v.ID, state: strings.TrimSpace(v.State), rationale: v.Rationale, kind: "other"})
	}
	for _, v := range a.RequirementAssurance {
		add("other", "requirementAssurance", v.RequirementID, v.State)
	}
	for _, v := range a.VerificationCases {
		add("other", "verificationCases", v.ID, v.State)
	}
	for _, v := range a.Procedures {
		add("other", "procedures", v.ID, v.State)
	}
	for _, v := range a.Executions {
		add("execution", "executions", v.ID, v.Status)
	}
	for _, v := range a.Baselines {
		add("other", "baselines", v.ID, v.State)
	}
	for _, v := range a.Builds {
		add("other", "builds", v.ID, v.Status)
	}
	for _, v := range a.Coverage {
		add("coverage", "coverage", v.ID, v.Status)
	}
	for _, v := range a.Tools {
		add("other", "tools", v.ID, v.Status)
	}
	for _, v := range a.Reviews {
		add("other", "reviews", v.ID, v.State)
	}
	for _, v := range a.Approvals {
		add("other", "approvals", v.ID, v.State)
	}
	for _, v := range a.Independence {
		add("other", "independence", v.ID, v.State)
	}
	for _, v := range a.ProblemReports {
		add("other", "problemReports", v.ID, v.State)
	}
	for _, v := range a.SQAAudits {
		add("other", "sqaAudits", v.ID, v.State)
	}
	return out
}

// floorFor returns the minimum evidence state a record needs at a tier
// rank, or "" when there is none: plans planned at mvp and reviewed from
// product; executions executed at product; every record accepted at
// operational (ADR-EMG-035).
// TRLC-LINKS: REQ-EMG-132
func floorFor(r aviationRecord, rank int) string {
	switch {
	case rank >= model.TierRank(model.TierOperational):
		return "accepted"
	case rank >= model.TierRank(model.TierProduct) && r.kind == "lifecycleData" && planFamilies[r.family]:
		return "reviewed"
	case rank >= model.TierRank(model.TierProduct) && r.kind == "execution":
		return "executed"
	case rank >= model.TierRank(model.TierMVP) && r.kind == "lifecycleData" && planFamilies[r.family]:
		return "planned"
	}
	return ""
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
		floor := floorFor(r, rank)
		if floor == "" {
			continue
		}
		if r.state == "not-applicable" && strings.TrimSpace(r.rationale) != "" {
			continue
		}
		if evidenceStateRank[r.state] < evidenceStateRank[floor] {
			out = append(out, AviationFloorGap{Path: r.path, ID: r.id, State: r.state, Floor: floor})
		}
	}
	if metric := levelCoverageMetric[strings.TrimSpace(a.Profile.SoftwareLevel)]; metric != "" && rank >= model.TierRank(model.TierProduct) {
		found := false
		for _, c := range a.Coverage {
			found = found || strings.EqualFold(strings.TrimSpace(c.Metric), metric)
		}
		if !found {
			out = append(out, AviationFloorGap{Path: "aviation.coverage", ID: "coverage:" + metric, State: "missing", Floor: metric})
		}
	}
	return out
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
