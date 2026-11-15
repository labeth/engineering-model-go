// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// toolGraphNeighborhood is graph.neighborhood.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolGraphNeighborhood(c toolCall) (map[string]any, error) {
	return map[string]any{"nodes": s.graphNodes(c.query, 50), "edges": s.graphEdges(c.query, 100)}, nil
}

// toolGraphExplainEdge is graph.explainEdge.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolGraphExplainEdge(c toolCall) (map[string]any, error) {
	from, to := c.arg("from"), c.arg("to")
	for _, m := range s.authored().Mappings {
		if m.From == from && m.To == to {
			return map[string]any{"exists": true, "mapping": m, "explanation": "Authored mapping exists"}, nil
		}
	}
	return map[string]any{"exists": false, "explanation": "No authored mapping found"}, nil
}

// toolGraphSearch is graph.search.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolGraphSearch(c toolCall) (map[string]any, error) {
	return map[string]any{"nodes": s.graphNodes(c.query, 200)}, nil
}

// toolModelList is model.list.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolModelList(c toolCall) (map[string]any, error) {
	return map[string]any{"entities": s.modelEntityNodes(c.query, c.arg("kind"), maxArg(c.args, 500))}, nil
}

// toolModelAuthoringContract is model.authoringContract.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolModelAuthoringContract(toolCall) (map[string]any, error) {
	return map[string]any{"contract": model.BuildAuthoringContract(*s.bundle), "rules": validate.AuthoringRules()}, nil
}

// toolModelComplete is model.complete.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolModelComplete(c toolCall) (map[string]any, error) {
	return s.modelComplete(c.args), nil
}

// toolModelPatch is model.patch.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolModelPatch(c toolCall) (map[string]any, error) {
	return s.modelPatch(c.args)
}

// toolModelDiagnose is model.diagnose.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolModelDiagnose(c toolCall) (map[string]any, error) {
	return s.modelDiagnose(c.args)
}

// toolModelEntity is model.entity.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolModelEntity(c toolCall) (map[string]any, error) {
	if err := c.needWellFormed(toolArg{"entityId", c.entityID}); err != nil {
		return nil, err
	}
	kind, entity, ok := s.modelEntity(c.entityID)
	return map[string]any{"exists": ok, "kind": kind, "entity": entity}, nil
}

// toolModelImplementations is model.implementations.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolModelImplementations(c toolCall) (map[string]any, error) {
	if err := c.needWellFormed(toolArg{"entityId", c.entityID}); err != nil {
		return nil, err
	}
	kind := s.modelEntityKind(c.entityID)
	if kind == "" {
		return nil, fmt.Errorf("model.implementations entity not found: %s", c.entityID)
	}
	implementations, diagnostics := s.implementationsForModelLink(c.entityID)
	implementations = limitRows(implementations, maxArg(c.args, 0))
	return map[string]any{"entity": map[string]any{"id": c.entityID, "kind": kind}, "implementations": implementations, "files": groupImplementationsByFile(implementations), "diagnostics": diagnostics}, nil
}

// toolCodeContextForTask is code.contextForTask.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolCodeContextForTask(c toolCall) (map[string]any, error) {
	return s.contextForTask(c.args), nil
}

// toolViewsRecommend is views.recommend.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolViewsRecommend(c toolCall) (map[string]any, error) {
	taskType := strings.ToLower(c.arg("taskType"))
	return map[string]any{"recommendedViews": s.recommendedViews(taskType)}, nil
}

// toolViewsRenderContext is views.renderContext.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolViewsRenderContext(c toolCall) (map[string]any, error) {
	viewID := c.arg("viewId")
	if err := c.needStable(toolArg{"viewId", viewID}, "VIEW-"); err != nil {
		return nil, err
	}
	for _, v := range s.bundle.Architecture.Views {
		if v.ID == viewID {
			return map[string]any{"view": v, "rootCount": len(v.Roots)}, nil
		}
	}
	return map[string]any{"view": nil}, nil
}

// toolGenerationPlan is generation.plan.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolGenerationPlan(toolCall) (map[string]any, error) {
	return map[string]any{"commands": s.generationCommands(), "artifacts": s.generationArtifacts()}, nil
}

// toolGenerationStatus is generation.status.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolGenerationStatus(toolCall) (map[string]any, error) {
	return map[string]any{"modelPath": s.modelPath, "requirementsPath": s.requirementsPath, "designPath": s.designPath, "artifacts": s.generationArtifactStatus()}, nil
}

// toolCompositionResolve is composition.resolve.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolCompositionResolve(toolCall) (map[string]any, error) {
	return s.compositionResolve()
}

// toolTraceMatrix is trace.matrix.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolTraceMatrix(c toolCall) (map[string]any, error) {
	return s.traceMatrix(boolArg(c.args, "summaryOnly"))
}

// toolTierGap is tier.gap.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolTierGap(c toolCall) (map[string]any, error) {
	return s.tierGap(c.arg("target"))
}

// toolGovernancePolicy is governance.policy.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolGovernancePolicy(toolCall) (map[string]any, error) {
	return map[string]any{"policy": s.governancePolicy()}, nil
}

// toolGovernanceCheckPatch is governance.checkPatch.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolGovernanceCheckPatch(c toolCall) (map[string]any, error) {
	diff := c.arg("diff")
	issues := []string{}
	if strings.Contains(diff, "tests/") && !strings.Contains(diff, "TRLC-LINKS") {
		issues = append(issues, "test changes without TRLC-LINKS markers")
	}
	return map[string]any{"issues": issues}, nil
}

// toolTasksEntryPoints is tasks.entryPoints.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolTasksEntryPoints(toolCall) (map[string]any, error) {
	return map[string]any{"entryPoints": s.taskEntryPoints()}, nil
}

// toolTasksNextBestActions is tasks.nextBestActions.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolTasksNextBestActions(toolCall) (map[string]any, error) {
	return map[string]any{"actions": s.nextBestActions()}, nil
}
