// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/internal/model"
)

// toolCall is the prepared input of one tool call: the tool name, the raw
// arguments and the arguments most tools read.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
type toolCall struct {
	name        string
	args        map[string]any
	reqID       string
	entityID    string
	interfaceID string
	env         string
	query       string
}

// toolArg is one named argument value a handler requires.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
type toolArg struct {
	name  string
	value string
}

// toolHandler computes the payload of one tool; callTool stamps it.
type toolHandler func(*Server, toolCall) (map[string]any, error)

// newToolCall prepares the arguments every handler may read.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func newToolCall(name string, args map[string]any) toolCall {
	return toolCall{
		name:        name,
		args:        args,
		reqID:       firstNonEmptyArg(args, "requirementId"),
		entityID:    firstNonEmptyArg(args, "entityId"),
		interfaceID: firstNonEmptyArg(args, "interfaceId"),
		env:         firstNonEmptyArg(args, "environment"),
		query:       strings.ToLower(firstNonEmptyArg(args, "query")),
	}
}

// arg is the trimmed first non-empty value of the named argument.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (c toolCall) arg(key string) string {
	return firstNonEmptyArg(c.args, key)
}

// needArg fails when the required argument is empty.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (c toolCall) needArg(a toolArg) error {
	if a.value == "" {
		return fmt.Errorf("%s requires %s", c.name, a.name)
	}
	return nil
}

// needStable fails when the required argument is empty or not a stable ID
// with one of the prefixes.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (c toolCall) needStable(a toolArg, prefixes ...string) error {
	if err := c.needArg(a); err != nil {
		return err
	}
	return requireStableID(a.value, prefixes...)
}

// needWellFormed fails when the required argument is empty or not a
// well-formed stable ID.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (c toolCall) needWellFormed(a toolArg) error {
	if err := c.needArg(a); err != nil {
		return err
	}
	return requireWellFormedStableID(a.value)
}

// needReqID requires the requirementId argument as a REQ- ID.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (c toolCall) needReqID() error {
	return c.needStable(toolArg{"requirementId", c.reqID}, "REQ-")
}

// stampTool marks a tool payload as an ok response of the named tool.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func stampTool(name string, data map[string]any) map[string]any {
	data["ok"] = true
	data["tool"] = name
	data["schemaVersion"] = toolSchemaVersion
	data["generatedAt"] = time.Now().UTC().Format(time.RFC3339)
	return data
}

// authored is the authored architecture of the loaded bundle.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) authored() *model.AuthoredArchitecture {
	return &s.bundle.Architecture.AuthoredArchitecture
}

// callTool runs one tool by name through the handler table.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) callTool(name string, args map[string]any) (map[string]any, error) {
	if s.bundle == nil {
		return map[string]any{"ok": false, "tool": name, "error": "model not loaded; pass initializationOptions.modelPath"}, nil
	}
	if s.canonical == nil {
		return nil, fmt.Errorf("canonical semantic model not loaded")
	}
	handler, ok := toolHandlers[name]
	if !ok {
		return nil, fmt.Errorf("unsupported tool: %s", name)
	}
	data, err := handler(s, newToolCall(name, args))
	if err != nil {
		return nil, err
	}
	return stampTool(name, data), nil
}

// toolHandlers maps every tool name to its handler.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
var toolHandlers = map[string]toolHandler{
	"requirements.get":             (*Server).toolRequirementsGet,
	"requirements.parents":         (*Server).toolRequirementsParents,
	"links.list":                   (*Server).toolLinksList,
	"links.suspect":                (*Server).toolLinksSuspect,
	"links.review":                 (*Server).toolLinksReview,
	"requirements.impact":          (*Server).toolRequirementsImpact,
	"requirements.supportPath":     (*Server).toolRequirementsImpact,
	"requirements.suggestEditPlan": (*Server).toolRequirementsImpact,
	"files.forRequirement":         (*Server).toolFilesForRequirement,
	"files.forControl":             (*Server).toolFilesForControl,
	"files.forThreat":              (*Server).toolFilesForThreat,
	"files.owner":                  (*Server).toolFilesOwner,
	"verification.status":          (*Server).toolVerificationStatus,
	"verification.gaps":            (*Server).toolVerificationGaps,
	"verification.recommend":       (*Server).toolVerificationRecommend,
	"gemara.controlCatalog":        (*Server).toolGemaraArtifact,
	"gemara.threatCatalog":         (*Server).toolGemaraArtifact,
	"gemara.riskCatalog":           (*Server).toolGemaraArtifact,
	"gemara.vectorCatalog":         (*Server).toolGemaraArtifact,
	"gemara.capabilityCatalog":     (*Server).toolGemaraArtifact,
	"gemara.principleCatalog":      (*Server).toolGemaraArtifact,
	"gemara.guidanceCatalog":       (*Server).toolGemaraArtifact,
	"gemara.policy":                (*Server).toolGemaraArtifact,
	"gemara.lexicon":               (*Server).toolGemaraArtifact,
	"gemara.mappingDocument":       (*Server).toolGemaraArtifact,
	"gemara.auditLog":              (*Server).toolGemaraArtifact,
	"gemara.enforcementLog":        (*Server).toolGemaraArtifact,
	"gemara.evaluationLog":         (*Server).toolGemaraArtifact,
	"gemara.validate":              (*Server).toolGemaraValidate,
	"threats.forRequirement":       (*Server).toolThreatsForRequirement,
	"threats.coverage":             (*Server).toolThreatsCoverage,
	"threats.unmitigated":          (*Server).toolThreatsUnmitigated,
	"flows.forRequirement":         (*Server).toolFlowsForRequirement,
	"flows.diff":                   (*Server).toolFlowsDiff,
	"graph.neighborhood":           (*Server).toolGraphNeighborhood,
	"graph.explainEdge":            (*Server).toolGraphExplainEdge,
	"graph.search":                 (*Server).toolGraphSearch,
	"model.list":                   (*Server).toolModelList,
	"model.authoringContract":      (*Server).toolModelAuthoringContract,
	"model.complete":               (*Server).toolModelComplete,
	"model.patch":                  (*Server).toolModelPatch,
	"model.diagnose":               (*Server).toolModelDiagnose,
	"model.entity":                 (*Server).toolModelEntity,
	"model.implementations":        (*Server).toolModelImplementations,
	"code.contextForTask":          (*Server).toolCodeContextForTask,
	"views.recommend":              (*Server).toolViewsRecommend,
	"views.renderContext":          (*Server).toolViewsRenderContext,
	"generation.plan":              (*Server).toolGenerationPlan,
	"generation.status":            (*Server).toolGenerationStatus,
	"composition.resolve":          (*Server).toolCompositionResolve,
	"trace.matrix":                 (*Server).toolTraceMatrix,
	"tier.gap":                     (*Server).toolTierGap,
	"governance.policy":            (*Server).toolGovernancePolicy,
	"governance.checkPatch":        (*Server).toolGovernanceCheckPatch,
	"tasks.entryPoints":            (*Server).toolTasksEntryPoints,
	"tasks.nextBestActions":        (*Server).toolTasksNextBestActions,
	"interfaces.resolve":           (*Server).toolInterfacesResolve,
	"endpoints.resolve":            (*Server).toolInterfacesResolve,
	"interfaces.implementations":   (*Server).toolInterfacesImplementations,
	"interfaces.matchFromCode":     (*Server).toolInterfacesMatchFromCode,
	"interfaces.ambiguities":       (*Server).toolInterfacesAmbiguities,
	"environments.resolve":         (*Server).toolEnvironmentsResolve,
	"identity.resolve":             (*Server).toolIdentityResolve,
	"policy.resolve":               (*Server).toolPolicyResolve,
	"schema.resolve":               (*Server).toolSchemaResolve,
	"schema.diff":                  (*Server).toolSchemaDiff,
	"flow.detail":                  (*Server).toolFlowDetail,
	"flow.implementations":         (*Server).toolFlowImplementations,
	"tests.forRequirement":         (*Server).toolTestsForRequirement,
	"tests.forEntity":              (*Server).toolTestsForEntity,
	"coverage.strictStatus":        (*Server).toolCoverageStrictStatus,
	"ownership.resolve":            (*Server).toolOwnershipResolve,
	"runtime.resolve":              (*Server).toolRuntimeResolve,
	"confidence.explain":           (*Server).toolConfidenceExplain,
	"staleness.check":              (*Server).toolStalenessCheck,
	"changes.preflight":            (*Server).toolChangesPreflight,
}
