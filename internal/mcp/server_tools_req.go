// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"fmt"
	"os"

	"github.com/labeth/engineering-model-go/internal/model"
)

// toolRequirementsGet is the requirements.get card with its trace.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolRequirementsGet(c toolCall) (map[string]any, error) {
	if err := c.needReqID(); err != nil {
		return nil, err
	}
	r := s.findRequirement(c.reqID)
	card := map[string]any{"requirement": r, "exists": r != nil}
	if r == nil {
		return card, nil
	}
	_, parents, links, children, err := s.requirementTrace(*r)
	if err != nil {
		return nil, err
	}
	card["parents"], card["links"], card["children"] = parents, links, children
	return card, nil
}

// toolRequirementsParents is requirements.parents.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolRequirementsParents(c toolCall) (map[string]any, error) {
	if err := c.needReqID(); err != nil {
		return nil, err
	}
	return s.requirementsParents(c.reqID)
}

// toolLinksList is links.list.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolLinksList(toolCall) (map[string]any, error) {
	return s.linksList()
}

// toolLinksSuspect is links.suspect.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolLinksSuspect(toolCall) (map[string]any, error) {
	return s.linksSuspect()
}

// toolLinksReview is links.review.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolLinksReview(c toolCall) (map[string]any, error) {
	linkID, by := c.arg("linkId"), c.arg("by")
	if linkID == "" || by == "" {
		return nil, fmt.Errorf("links.review requires linkId and by")
	}
	return s.linksReview(linkID, by, c.arg("rationale"))
}

// toolRequirementsImpact serves requirements.impact, requirements.supportPath
// and requirements.suggestEditPlan.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolRequirementsImpact(c toolCall) (map[string]any, error) {
	if err := c.needReqID(); err != nil {
		return nil, err
	}
	r := s.findRequirement(c.reqID)
	flows := s.flowsForRequirement(c.reqID)
	threats := s.threatsForRequirement(c.reqID)
	files := s.filesForRequirement(c.reqID)
	payload := map[string]any{"requirement": r, "impactedFlows": flows, "impactedThreats": threats, "relevantFiles": files}
	if c.name == "requirements.supportPath" {
		payload["supportPath"] = s.requirementSupportPath(c.reqID)
	}
	if c.name == "requirements.suggestEditPlan" {
		payload["editPlan"] = s.requirementEditPlan(c.reqID)
	}
	return payload, nil
}

// toolFilesForRequirement is files.forRequirement.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolFilesForRequirement(c toolCall) (map[string]any, error) {
	if err := c.needReqID(); err != nil {
		return nil, err
	}
	return map[string]any{"files": s.filesForRequirement(c.reqID)}, nil
}

// toolFilesForControl is files.forControl.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolFilesForControl(c toolCall) (map[string]any, error) {
	controlID := c.arg("controlId")
	if err := c.needStable(toolArg{"controlId", controlID}, "CTRL-"); err != nil {
		return nil, err
	}
	return map[string]any{"files": s.filesContaining(controlID)}, nil
}

// toolFilesForThreat is files.forThreat.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolFilesForThreat(c toolCall) (map[string]any, error) {
	threatID := c.arg("threatId")
	if err := c.needStable(toolArg{"threatId", threatID}, "TS-", "THREAT-"); err != nil {
		return nil, err
	}
	return map[string]any{"files": s.filesContaining(threatID)}, nil
}

// toolFilesOwner is files.owner.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolFilesOwner(c toolCall) (map[string]any, error) {
	path := c.arg("path")
	if err := c.needArg(toolArg{"path", path}); err != nil {
		return nil, err
	}
	safePath, ok := s.resolvePathInRepo(path)
	if !ok {
		return nil, fmt.Errorf("files.owner path must be inside repoRoot")
	}
	content, _ := os.ReadFile(safePath)
	ownerFU, markerLine := ownerUnitMarker(safePath, content)
	ownerKnown := s.functionalUnitKnown(ownerFU)
	return map[string]any{"path": safePath, "ownerFunctionalUnit": ownerFU, "ownerKnown": ownerKnown, "markerLine": markerLine}, nil
}

// functionalUnitKnown reports whether the model has a functional unit with
// the ID; an empty ID is never known.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) functionalUnitKnown(id string) bool {
	if id == "" {
		return false
	}
	for _, fu := range s.authored().FunctionalUnits {
		if fu.ID == id {
			return true
		}
	}
	return false
}

// formalEntry is the output field a formal (Quint) result adds to a
// verification tool payload: the error when the results could not be
// computed (never a clean status), else the rows when there are any.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func formalEntry(key string, rows []formalRequirement, err error) (string, any, bool) {
	if err != nil {
		return "formalError", err.Error(), true
	}
	return key, rows, len(rows) > 0
}

// toolVerificationStatus is verification.status.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolVerificationStatus(c toolCall) (map[string]any, error) {
	status := []model.ControlVerification{}
	for _, cv := range s.authored().ControlVerifications {
		if c.entityID == "" || cv.ID == c.entityID || cv.ControlRef == c.entityID {
			status = append(status, cv)
		}
	}
	out := map[string]any{"controlVerifications": status}
	formal, formalErr := s.formalStatus(c.entityID)
	if key, value, ok := formalEntry("formalResults", formal, formalErr); ok {
		out[key] = value
	}
	return out, nil
}

// toolVerificationGaps is verification.gaps.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolVerificationGaps(toolCall) (map[string]any, error) {
	gaps := []string{}
	for _, r := range s.requirementIDs() {
		if len(s.filesForRequirement(r)) == 0 {
			gaps = append(gaps, r)
		}
	}
	out := map[string]any{"requirementsWithoutFiles": gaps}
	formal, formalErr := s.formalGaps()
	if key, value, ok := formalEntry("formalGaps", formal, formalErr); ok {
		out[key] = value
	}
	return out, nil
}

// toolVerificationRecommend is verification.recommend.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolVerificationRecommend(toolCall) (map[string]any, error) {
	return map[string]any{"recommendations": s.verificationRecommendations()}, nil
}

// toolGemaraArtifact serves the Gemara artifact tools.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolGemaraArtifact(c toolCall) (map[string]any, error) {
	return s.gemaraArtifact(c.name)
}

// toolGemaraValidate is gemara.validate.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolGemaraValidate(toolCall) (map[string]any, error) {
	return s.gemaraValidate()
}

// toolThreatsForRequirement is threats.forRequirement.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolThreatsForRequirement(c toolCall) (map[string]any, error) {
	if err := c.needReqID(); err != nil {
		return nil, err
	}
	return map[string]any{"threatScenarios": s.threatsForRequirement(c.reqID)}, nil
}

// toolThreatsCoverage is threats.coverage.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolThreatsCoverage(toolCall) (map[string]any, error) {
	rows := []map[string]any{}
	for _, ts := range s.authored().ThreatScenarios {
		rows = append(rows, map[string]any{"threatScenario": ts.ID, "controls": ts.RelatedControls, "mitigations": ts.MitigationRefs, "verifications": ts.VerificationRefs})
	}
	return map[string]any{"coverage": rows}, nil
}

// toolThreatsUnmitigated is threats.unmitigated.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolThreatsUnmitigated(toolCall) (map[string]any, error) {
	rows := []string{}
	for _, ts := range s.authored().ThreatScenarios {
		if len(ts.MitigationRefs) == 0 {
			rows = append(rows, ts.ID)
		}
	}
	return map[string]any{"threatScenarioIds": rows}, nil
}

// toolFlowsForRequirement is flows.forRequirement.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolFlowsForRequirement(c toolCall) (map[string]any, error) {
	if err := c.needReqID(); err != nil {
		return nil, err
	}
	return map[string]any{"flows": s.flowsForRequirement(c.reqID)}, nil
}

// toolFlowsDiff is flows.diff.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolFlowsDiff(c toolCall) (map[string]any, error) {
	left, right := c.arg("fromFlowId"), c.arg("toFlowId")
	if left == "" || right == "" {
		return nil, fmt.Errorf("flows.diff requires fromFlowId and toFlowId")
	}
	for _, id := range []string{left, right} {
		if err := requireStableID(id, "FLOW-"); err != nil {
			return nil, err
		}
	}
	return s.flowDiff(left, right)
}
