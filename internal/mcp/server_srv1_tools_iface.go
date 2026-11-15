// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// requireInterface requires the interfaceId argument as an IF- ID that names
// an interface of the model.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) requireInterface(c toolCall) (*model.Interface, error) {
	if err := c.needStable(toolArg{"interfaceId", c.interfaceID}, "IF-"); err != nil {
		return nil, err
	}
	iface := s.findInterface(c.interfaceID)
	if iface == nil {
		return nil, fmt.Errorf("%s interface not found: %s", c.name, c.interfaceID)
	}
	return iface, nil
}

// toolInterfacesResolve serves interfaces.resolve and endpoints.resolve.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolInterfacesResolve(c toolCall) (map[string]any, error) {
	iface, err := s.requireInterface(c)
	if err != nil {
		return nil, err
	}
	return map[string]any{"interface": iface, "environment": c.env, "resolvedEndpoint": endpointForEnv(iface, c.env)}, nil
}

// toolInterfacesImplementations is interfaces.implementations.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolInterfacesImplementations(c toolCall) (map[string]any, error) {
	iface, err := s.requireInterface(c)
	if err != nil {
		return nil, err
	}
	implementations, diagnostics := s.implementationsForModelLink(c.interfaceID)
	implementations = limitRows(implementations, maxArg(c.args, 0))
	return map[string]any{"interface": iface, "implementations": implementations, "files": groupImplementationsByFile(implementations), "diagnostics": diagnostics}, nil
}

// toolInterfacesMatchFromCode is interfaces.matchFromCode.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolInterfacesMatchFromCode(c toolCall) (map[string]any, error) {
	path := c.arg("path")
	if err := c.needArg(toolArg{"path", path}); err != nil {
		return nil, err
	}
	safePath, ok := s.resolvePathInRepo(path)
	if !ok {
		return nil, fmt.Errorf("interfaces.matchFromCode path must be inside repoRoot")
	}
	content, _ := os.ReadFile(safePath)
	ownerFU, _ := ownerUnitMarker(safePath, content)
	matched := []string{}
	for _, i := range s.authored().Interfaces {
		if strings.Contains(string(content), i.ID) || strings.Contains(string(content), i.Endpoint) {
			matched = append(matched, i.ID)
		}
	}
	return map[string]any{"path": safePath, "interfaces": matched, "ownerFunctionalUnit": ownerFU}, nil
}

// toolInterfacesAmbiguities is interfaces.ambiguities.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolInterfacesAmbiguities(toolCall) (map[string]any, error) {
	byName := map[string]int{}
	for _, i := range s.authored().Interfaces {
		byName[strings.ToLower(strings.TrimSpace(i.Name))]++
	}
	amb := []string{}
	for n, count := range byName {
		if count > 1 {
			amb = append(amb, n)
		}
	}
	sort.Strings(amb)
	return map[string]any{"ambiguousInterfaceNames": amb}, nil
}

// toolEnvironmentsResolve is environments.resolve.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolEnvironmentsResolve(c toolCall) (map[string]any, error) {
	rows := []model.DeploymentTarget{}
	for _, d := range s.authored().DeploymentTargets {
		if c.env == "" || strings.EqualFold(d.Environment, c.env) {
			rows = append(rows, d)
		}
	}
	return map[string]any{"deploymentTargets": rows}, nil
}

// toolIdentityResolve is identity.resolve.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolIdentityResolve(c toolCall) (map[string]any, error) {
	iface, err := s.requireInterface(c)
	if err != nil {
		return nil, err
	}
	return map[string]any{"interface": iface, "authHints": s.authHintsForInterface(c.interfaceID)}, nil
}

// toolPolicyResolve is policy.resolve.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolPolicyResolve(c toolCall) (map[string]any, error) {
	if err := c.needArg(toolArg{"interfaceId", c.interfaceID}); err != nil {
		return nil, err
	}
	return s.policyForEntity(c.interfaceID), nil
}

// toolSchemaResolve is schema.resolve.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolSchemaResolve(c toolCall) (map[string]any, error) {
	iface, err := s.requireInterface(c)
	if err != nil {
		return nil, err
	}
	return map[string]any{"schemaRef": iface.SchemaRef, "protocol": iface.Protocol}, nil
}

// toolSchemaDiff is schema.diff.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolSchemaDiff(c toolCall) (map[string]any, error) {
	left, right := c.arg("fromSchemaRef"), c.arg("toSchemaRef")
	if left == "" {
		left = s.schemaRefForInterface(c.arg("fromInterfaceId"))
	}
	if right == "" {
		right = s.schemaRefForInterface(c.arg("toInterfaceId"))
	}
	if left == "" && right == "" {
		return nil, fmt.Errorf("schema.diff requires schema refs or interface ids")
	}
	status, note, leftMeta, rightMeta := s.diffSchemaRefs(left, right)
	return map[string]any{"fromSchemaRef": left, "toSchemaRef": right, "status": status, "message": note, "fromSchema": leftMeta, "toSchema": rightMeta}, nil
}

// requireFlowID requires the flowId argument as a FLOW- ID.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (c toolCall) requireFlowID() (string, error) {
	flowID := c.arg("flowId")
	return flowID, c.needStable(toolArg{"flowId", flowID}, "FLOW-")
}

// toolFlowDetail is flow.detail.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolFlowDetail(c toolCall) (map[string]any, error) {
	flowID, err := c.requireFlowID()
	if err != nil {
		return nil, err
	}
	flow := s.findFlow(flowID)
	return map[string]any{"exists": flow != nil, "flow": flow, "implementations": s.implementationFilesForModelLink(flowID)}, nil
}

// toolFlowImplementations is flow.implementations.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolFlowImplementations(c toolCall) (map[string]any, error) {
	flowID, err := c.requireFlowID()
	if err != nil {
		return nil, err
	}
	if s.findFlow(flowID) == nil {
		return nil, fmt.Errorf("flow.implementations flow not found: %s", flowID)
	}
	implementations, diagnostics := s.implementationsForModelLink(flowID)
	implementations = limitRows(implementations, maxArg(c.args, 0))
	return map[string]any{"flowId": flowID, "implementations": implementations, "files": groupImplementationsByFile(implementations), "diagnostics": diagnostics}, nil
}

// toolTestsForRequirement is tests.forRequirement.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolTestsForRequirement(c toolCall) (map[string]any, error) {
	if err := c.needReqID(); err != nil {
		return nil, err
	}
	tests, diagnostics := s.testsForRequirement(c.reqID)
	tests = limitRows(tests, maxArg(c.args, 0))
	return map[string]any{"requirementId": c.reqID, "tests": tests, "files": groupImplementationsByFile(tests), "diagnostics": diagnostics}, nil
}

// toolTestsForEntity is tests.forEntity.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolTestsForEntity(c toolCall) (map[string]any, error) {
	if err := c.needWellFormed(toolArg{"entityId", c.entityID}); err != nil {
		return nil, err
	}
	if s.modelEntityKind(c.entityID) == "" {
		return nil, fmt.Errorf("tests.forEntity entity not found: %s", c.entityID)
	}
	tests, diagnostics := s.testsForModelLink(c.entityID)
	tests = limitRows(tests, maxArg(c.args, 0))
	return map[string]any{"entityId": c.entityID, "tests": tests, "files": groupImplementationsByFile(tests), "diagnostics": diagnostics}, nil
}

// toolCoverageStrictStatus is coverage.strictStatus.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolCoverageStrictStatus(c toolCall) (map[string]any, error) {
	return s.strictCoverageStatus(c.arg("path")), nil
}

// toolOwnershipResolve is ownership.resolve.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolOwnershipResolve(c toolCall) (map[string]any, error) {
	if err := c.needStable(toolArg{"entityId", c.entityID}, "REQ-", "IF-", "FU-", "CTRL-", "TS-", "RISK-"); err != nil {
		return nil, err
	}
	return map[string]any{"id": c.entityID, "owner": s.ownerFor(c.entityID)}, nil
}

// toolRuntimeResolve is runtime.resolve.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolRuntimeResolve(c toolCall) (map[string]any, error) {
	return s.runtimeForEntity(c.entityID), nil
}

// toolConfidenceExplain is confidence.explain.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolConfidenceExplain(c toolCall) (map[string]any, error) {
	if err := c.needStable(toolArg{"entityId", c.entityID}, "REQ-", "IF-", "FU-", "CTRL-", "TS-", "RISK-", "FLOW-", "VIEW-", "DO-", "DEP-", "TB-", "EVT-", "STATE-"); err != nil {
		return nil, err
	}
	return s.confidenceForEntity(c.entityID), nil
}

// toolStalenessCheck is staleness.check.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolStalenessCheck(c toolCall) (map[string]any, error) {
	path := c.arg("path")
	if path == "" {
		path = s.modelPath
	}
	return s.stalenessForPath(path), nil
}

// toolChangesPreflight is changes.preflight.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) toolChangesPreflight(c toolCall) (map[string]any, error) {
	if err := c.needReqID(); err != nil {
		return nil, err
	}
	return map[string]any{"requirementId": c.reqID, "impactedFiles": s.filesForRequirement(c.reqID), "rerun": []string{"go test ./...", "engdoc", "engdragon", "engstruct", "engtrlc"}}, nil
}
