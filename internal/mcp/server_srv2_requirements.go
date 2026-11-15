// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) flowsForRequirement(reqID string) []model.Flow {
	if s.bundle == nil {
		return nil
	}
	applies := s.requirementAppliesSet(reqID)
	out := []model.Flow{}
	for _, f := range s.bundle.Architecture.AuthoredArchitecture.Flows {
		if reqID == "" || flowTouches(f, applies) {
			out = append(out, f)
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) requirementAppliesSet(reqID string) map[string]bool {
	applies := map[string]bool{}
	if req := s.findRequirement(reqID); req != nil {
		for _, x := range req.AppliesTo {
			applies[x] = true
		}
	}
	return applies
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func flowTouches(f model.Flow, applies map[string]bool) bool {
	if anyApplies(applies, f.SourceRef, f.DestinationRef) {
		return true
	}
	for _, st := range f.Steps {
		if anyApplies(applies, st.Ref, st.SourceRef, st.DestinationRef, st.InterfaceRef) {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func anyApplies(applies map[string]bool, refs ...string) bool {
	for _, ref := range refs {
		if applies[ref] {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER
func (s *Server) threatsForRequirement(reqID string) []model.ThreatScenario {
	if s.bundle == nil {
		return nil
	}
	applies := s.requirementAppliesSet(reqID)
	out := []model.ThreatScenario{}
	for _, ts := range s.bundle.Architecture.AuthoredArchitecture.ThreatScenarios {
		if reqID == "" || anyApplies(applies, ts.AppliesTo...) {
			out = append(out, ts)
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) verificationRecommendations() []map[string]any {
	out := s.untestedRequirementActions()
	out = append(out, s.controlVerificationActions()...)
	status := s.strictCoverageStatus("")
	if status["status"] == "fail" {
		out = append(out, map[string]any{"action": "fix_code_linking_diagnostics", "reason": "strict code-linking diagnostics contain errors", "summary": status["summary"]})
	}
	if len(out) == 0 {
		out = append(out, map[string]any{"action": "maintain_current_coverage", "reason": "requirements, code links, and control verifications have no immediate MCP-detected gaps"})
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) untestedRequirementActions() []map[string]any {
	out := []map[string]any{}
	for _, reqID := range s.requirementIDs() {
		tests, _ := s.testsForRequirement(reqID)
		if len(tests) == 0 {
			out = append(out, map[string]any{"action": "add_requirement_test_links", "requirementId": reqID, "reason": "no test declarations with TRLC-LINKS were found for this requirement"})
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) controlVerificationActions() []map[string]any {
	out := []map[string]any{}
	if s.bundle == nil {
		return out
	}
	for _, cv := range s.bundle.Architecture.AuthoredArchitecture.ControlVerifications {
		if !strings.EqualFold(cv.Status, "pass") {
			out = append(out, map[string]any{"action": "complete_control_verification", "controlVerificationId": cv.ID, "controlId": cv.ControlRef, "status": cv.Status, "reason": "control verification is not passing"})
		}
		if len(cv.Evidence) == 0 {
			out = append(out, map[string]any{"action": "add_control_verification_evidence", "controlVerificationId": cv.ID, "controlId": cv.ControlRef, "reason": "control verification has no evidence paths"})
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func (s *Server) flowDiff(leftID, rightID string) (map[string]any, error) {
	left := s.findFlow(leftID)
	right := s.findFlow(rightID)
	if left == nil {
		return nil, fmt.Errorf("flows.diff flow not found: %s", leftID)
	}
	if right == nil {
		return nil, fmt.Errorf("flows.diff flow not found: %s", rightID)
	}
	return map[string]any{
		"from":         leftID,
		"to":           rightID,
		"fromFlow":     flowEnds(left),
		"toFlow":       flowEnds(right),
		"fieldChanges": flowFieldChanges(left, right),
		"setChanges":   flowSetChanges(left, right),
		"stepChanges":  flowStepChanges(left, right),
	}, nil
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func flowEnds(f *model.Flow) map[string]any {
	return map[string]any{
		"id": f.ID, "title": f.Title, "sourceRef": f.SourceRef, "destinationRef": f.DestinationRef,
	}
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
type flowFieldPair struct {
	field string
	from  string
	to    string
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func flowFieldChanges(left, right *model.Flow) []map[string]any {
	pairs := []flowFieldPair{
		{"title", left.Title, right.Title},
		{"kind", left.Kind, right.Kind},
		{"methodology", left.Methodology, right.Methodology},
		{"direction", left.Direction, right.Direction},
		{"frequency", left.Frequency, right.Frequency},
		{"sourceRef", left.SourceRef, right.SourceRef},
		{"destinationRef", left.DestinationRef, right.DestinationRef},
		{"protocol", left.Protocol, right.Protocol},
		{"channel", left.Channel, right.Channel},
		{"authentication", left.Authentication, right.Authentication},
		{"encryptionInTransit", left.EncryptionInTransit, right.EncryptionInTransit},
		{"integrityProtection", left.IntegrityProtection, right.IntegrityProtection},
		{"criticality", left.Criticality, right.Criticality},
	}
	fields := []map[string]any{}
	for _, p := range pairs {
		if strings.TrimSpace(p.from) != strings.TrimSpace(p.to) {
			fields = append(fields, map[string]any{"field": p.field, "from": p.from, "to": p.to})
		}
	}
	return fields
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
type flowSlicePair struct {
	field string
	from  []string
	to    []string
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func flowSetChanges(left, right *model.Flow) []map[string]any {
	pairs := []flowSlicePair{
		{"dataRefs", left.DataRefs, right.DataRefs},
		{"threats", left.Threats, right.Threats},
		{"entry", left.Entry, right.Entry},
		{"exits", left.Exits, right.Exits},
	}
	slices := []map[string]any{}
	for _, p := range pairs {
		added, removed, common := sliceSetDiff(p.from, p.to)
		if len(added) > 0 || len(removed) > 0 {
			slices = append(slices, map[string]any{"field": p.field, "added": added, "removed": removed, "common": common})
		}
	}
	return slices
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func flowStepChanges(left, right *model.Flow) map[string]any {
	leftSteps := flowStepsByID(left.Steps)
	rightSteps := flowStepsByID(right.Steps)
	addedSteps, removedSteps, commonSteps := sliceSetDiff(mapKeys(leftSteps), mapKeys(rightSteps))
	changedSteps := []string{}
	for _, id := range commonSteps {
		lb, _ := json.Marshal(leftSteps[id])
		rb, _ := json.Marshal(rightSteps[id])
		if string(lb) != string(rb) {
			changedSteps = append(changedSteps, id)
		}
	}
	return map[string]any{"added": addedSteps, "removed": removedSteps, "changed": changedSteps, "common": commonSteps}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func sliceSetDiff(left, right []string) (added, removed, common []string) {
	leftSeen := trimmedSet(left)
	rightSeen := trimmedSet(right)
	added = sortedSetMembers(rightSeen, func(x string) bool { return !leftSeen[x] })
	common = sortedSetMembers(rightSeen, func(x string) bool { return leftSeen[x] })
	removed = sortedSetMembers(leftSeen, func(x string) bool { return !rightSeen[x] })
	return added, removed, common
}

// trimmedSet collects the non-empty trimmed entries of in.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func trimmedSet(in []string) map[string]bool {
	seen := map[string]bool{}
	for _, x := range in {
		if x = strings.TrimSpace(x); x != "" {
			seen[x] = true
		}
	}
	return seen
}

// sortedSetMembers returns the sorted members of set accepted by keep; the
// result stays nil when none qualify.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func sortedSetMembers(set map[string]bool, keep func(string) bool) []string {
	var out []string
	for x := range set {
		if keep(x) {
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-VIEW-PROJECTION
type viewCandidate struct {
	kind   string
	reason string
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-VIEW-PROJECTION
type viewRule struct {
	words      []string
	candidates []viewCandidate
}

var defaultViewCandidates = []viewCandidate{{kind: "traceability", reason: "default view for implementation and requirement impact work"}}

var viewRules = []viewRule{
	{
		words:      []string{"security", "threat", "control"},
		candidates: []viewCandidate{{kind: "security", reason: "security tasks need controls, threats, and trust boundaries"}, {kind: "traceability", reason: "traceability shows requirements and verification links"}},
	},
	{
		words:      []string{"deploy", "runtime", "environment"},
		candidates: []viewCandidate{{kind: "deployment", reason: "deployment tasks need runtime targets and environment boundaries"}, {kind: "communication", reason: "communication shows runtime interaction surfaces"}},
	},
	{
		words:      []string{"flow", "api", "interface"},
		candidates: []viewCandidate{{kind: "communication", reason: "interface work needs service and data interaction context"}, {kind: "traceability", reason: "traceability links interfaces back to requirements"}},
	},
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-VIEW-PROJECTION
func viewCandidatesFor(taskType string) []viewCandidate {
	for _, rule := range viewRules {
		if anyString(rule.words, func(word string) bool { return strings.Contains(taskType, word) }) {
			return rule.candidates
		}
	}
	return defaultViewCandidates
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-VIEW-PROJECTION
func (s *Server) recommendedViews(taskType string) []map[string]any {
	taskType = strings.ToLower(strings.TrimSpace(taskType))
	out := []map[string]any{}
	for _, c := range viewCandidatesFor(taskType) {
		out = append(out, s.viewsOfKind(c)...)
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-VIEW-PROJECTION
func (s *Server) viewsOfKind(c viewCandidate) []map[string]any {
	out := []map[string]any{}
	for _, v := range s.bundle.Architecture.Views {
		if strings.EqualFold(v.Kind, c.kind) {
			out = append(out, map[string]any{"id": v.ID, "kind": v.Kind, "reason": c.reason, "roots": v.Roots, "audience": v.Audience})
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) nextBestActions() []map[string]any {
	out := s.strictLinkingActions()
	out = append(out, s.unlinkedRequirementActions()...)
	out = append(out, s.openRiskActions()...)
	out = append(out, s.failingVerificationActions()...)
	if len(out) == 0 {
		out = append(out, map[string]any{"action": "inspect_task_context", "tool": "code.contextForTask", "reason": "no immediate gaps found; use task context before editing"})
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) strictLinkingActions() []map[string]any {
	out := []map[string]any{}
	status := s.strictCoverageStatus("")
	if status["status"] == "fail" {
		out = append(out, map[string]any{"action": "fix_strict_code_linking", "tool": "coverage.strictStatus", "reason": "source diagnostics include strict coverage failures", "summary": status["summary"]})
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) unlinkedRequirementActions() []map[string]any {
	out := []map[string]any{}
	for _, reqID := range s.requirementIDs() {
		if len(s.filesForRequirement(reqID)) == 0 {
			out = append(out, map[string]any{"action": "link_requirement_to_code_or_tests", "tool": "files.forRequirement", "requirementId": reqID, "reason": "requirement has no indexed file mentions"})
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) openRiskActions() []map[string]any {
	out := []map[string]any{}
	for _, risk := range s.bundle.Architecture.AuthoredArchitecture.Risks {
		if strings.EqualFold(risk.Status, "open") {
			out = append(out, map[string]any{"action": "reduce_open_risk", "tool": "threats.coverage", "riskId": risk.ID, "reason": risk.Rationale})
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) failingVerificationActions() []map[string]any {
	out := []map[string]any{}
	for _, cv := range s.bundle.Architecture.AuthoredArchitecture.ControlVerifications {
		if !strings.EqualFold(cv.Status, "pass") {
			out = append(out, map[string]any{"action": "complete_control_verification", "tool": "verification.status", "controlVerificationId": cv.ID, "status": cv.Status, "reason": "verification is not passing"})
		}
	}
	return out
}
