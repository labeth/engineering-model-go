// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labeth/engineering-model-go/codemap"
	"github.com/labeth/engineering-model-go/model"
)

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
type Tool struct {
	Name        string
	Description string
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type Server struct {
	tools map[string]Tool

	bundle           *model.Bundle
	requirements     *model.RequirementsDocument
	design           *model.DesignDocument
	modelPath        string
	requirementsPath string
	designPath       string
	repoRoot         string

	indexOnce sync.Once
	indexErr  error
	repoFiles []indexedFile
}

const (
	maxIndexedFileBytes = 512 * 1024
	maxIndexedFiles     = 5000
	toolSchemaVersion   = "mcp.tool-response.v1"
)

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
var errRepoIndexLimit = errors.New("repo index file limit reached")

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
var stableIDPattern = regexp.MustCompile(`^[A-Z][A-Z0-9-]*$`)

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
var toolArgsAllowlist = map[string][]string{
	"requirements.get":             {"requirementId", "id", "reqId"},
	"requirements.impact":          {"requirementId", "id", "reqId"},
	"requirements.supportPath":     {"requirementId", "id", "reqId"},
	"requirements.suggestEditPlan": {"requirementId", "id", "reqId"},
	"files.forRequirement":         {"requirementId", "id", "reqId"},
	"files.forControl":             {"controlId", "id"},
	"files.forThreat":              {"threatId", "id"},
	"files.owner":                  {"path", "file"},
	"verification.status":          {"entityId", "id"},
	"verification.gaps":            {},
	"verification.recommend":       {},
	"threats.forRequirement":       {"requirementId", "id", "reqId"},
	"threats.coverage":             {},
	"threats.unmitigated":          {},
	"flows.forRequirement":         {"requirementId", "id", "reqId"},
	"flows.diff":                   {"fromFlowId", "toFlowId", "left", "right"},
	"graph.neighborhood":           {"query", "q", "name", "id"},
	"graph.explainEdge":            {"from", "to"},
	"graph.search":                 {"query", "q", "name", "id"},
	"model.list":                   {"query", "q", "kind", "max"},
	"entities.list":                {"query", "q", "kind", "max"},
	"model.entity":                 {"entityId", "id"},
	"model.implementations":        {"entityId", "id", "groupBy", "max"},
	"code.contextForTask":          {"query", "q", "requirementId", "reqId", "entityId", "id", "path", "file", "max"},
	"views.recommend":              {"taskType", "task"},
	"views.renderContext":          {"viewId", "id"},
	"generation.plan":              {},
	"generation.status":            {},
	"composition.resolve":          {},
	"trace.matrix":                 {},
	"governance.policy":            {},
	"governance.checkPatch":        {"diff"},
	"tasks.entryPoints":            {},
	"tasks.nextBestActions":        {},
	"interfaces.resolve":           {"interfaceId", "id", "name", "environment", "env"},
	"interfaces.implementations":   {"interfaceId", "id", "name", "groupBy", "max"},
	"interfaces.matchFromCode":     {"path", "file"},
	"interfaces.ambiguities":       {},
	"environments.resolve":         {"environment", "env"},
	"endpoints.resolve":            {"interfaceId", "id", "name", "environment", "env"},
	"identity.resolve":             {"interfaceId", "id", "name"},
	"policy.resolve":               {"interfaceId", "id", "name"},
	"schema.resolve":               {"interfaceId", "id", "name"},
	"schema.diff":                  {"fromSchemaRef", "leftSchemaRef", "toSchemaRef", "rightSchemaRef", "fromInterfaceId", "leftInterfaceId", "toInterfaceId", "rightInterfaceId"},
	"flow.detail":                  {"flowId", "id"},
	"flow.implementations":         {"flowId", "id", "groupBy", "max"},
	"tests.forRequirement":         {"requirementId", "id", "reqId", "max"},
	"tests.forEntity":              {"entityId", "id", "max"},
	"coverage.strictStatus":        {"path", "file"},
	"ownership.resolve":            {"entityId", "id"},
	"runtime.resolve":              {"entityId", "id"},
	"confidence.explain":           {"entityId", "id"},
	"staleness.check":              {"path"},
	"changes.preflight":            {"requirementId", "id", "reqId"},
	"gemara.controlCatalog":        {},
	"gemara.threatCatalog":         {},
	"gemara.riskCatalog":           {},
	"gemara.vectorCatalog":         {},
	"gemara.capabilityCatalog":     {},
	"gemara.principleCatalog":      {},
	"gemara.guidanceCatalog":       {},
	"gemara.policy":                {},
	"gemara.lexicon":               {},
	"gemara.mappingDocument":       {},
	"gemara.auditLog":              {},
	"gemara.enforcementLog":        {},
	"gemara.evaluationLog":         {},
	"gemara.validate":              {},
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type indexedFile struct {
	Path    string
	Content string
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func NewServer() *Server {
	all := []Tool{
		{Name: "requirements.get", Description: "Get requirement detail card"},
		{Name: "requirements.impact", Description: "Get requirement impact map"},
		{Name: "requirements.supportPath", Description: "Get requirement support chain"},
		{Name: "requirements.suggestEditPlan", Description: "Suggest requirement edit plan"},
		{Name: "files.forRequirement", Description: "List files for requirement"},
		{Name: "files.forControl", Description: "List files for control"},
		{Name: "files.forThreat", Description: "List files for threat"},
		{Name: "files.owner", Description: "Resolve owner for file"},
		{Name: "verification.status", Description: "Get verification status"},
		{Name: "verification.gaps", Description: "List verification gaps"},
		{Name: "verification.recommend", Description: "Recommend verification actions"},
		{Name: "threats.forRequirement", Description: "Get threats linked to requirement"},
		{Name: "threats.coverage", Description: "Get threat-control-verification coverage"},
		{Name: "threats.unmitigated", Description: "List unmitigated threats"},
		{Name: "flows.forRequirement", Description: "Get flows for requirement"},
		{Name: "flows.diff", Description: "Diff flow impact"},
		{Name: "graph.neighborhood", Description: "Get graph neighborhood"},
		{Name: "graph.explainEdge", Description: "Explain graph edge"},
		{Name: "graph.search", Description: "Search graph entities"},
		{Name: "model.list", Description: "List model entities"},
		{Name: "entities.list", Description: "List model entities"},
		{Name: "model.entity", Description: "Get model entity detail"},
		{Name: "model.implementations", Description: "List source declarations linked to any model entity"},
		{Name: "code.contextForTask", Description: "Assemble compact code and model context for a task"},
		{Name: "views.recommend", Description: "Recommend views for task"},
		{Name: "views.renderContext", Description: "Render compact view context"},
		{Name: "generation.plan", Description: "Plan artifact regeneration"},
		{Name: "generation.status", Description: "Get artifact freshness status"},
		{Name: "composition.resolve", Description: "Resolve the system-of-systems: subsystems, their contracts, requirement allocations with the delegated subsystem requirement, and composition diagnostics"},
		{Name: "trace.matrix", Description: "Traceability matrix: per-requirement implemented/verified/delegated/orphan status, code references, and dangling code trace links"},
		{Name: "governance.policy", Description: "Get governance policy"},
		{Name: "governance.checkPatch", Description: "Check patch against governance"},
		{Name: "tasks.entryPoints", Description: "List task entry points"},
		{Name: "tasks.nextBestActions", Description: "Suggest next best actions"},
		{Name: "interfaces.resolve", Description: "Resolve interface endpoint target"},
		{Name: "interfaces.implementations", Description: "List source declarations linked to an interface"},
		{Name: "interfaces.matchFromCode", Description: "Match interface from code path"},
		{Name: "interfaces.ambiguities", Description: "List interface ambiguities"},
		{Name: "environments.resolve", Description: "Resolve environment metadata"},
		{Name: "endpoints.resolve", Description: "Resolve endpoint per environment"},
		{Name: "identity.resolve", Description: "Resolve identity/auth requirements"},
		{Name: "policy.resolve", Description: "Resolve policy for interface"},
		{Name: "schema.resolve", Description: "Resolve schema for interface"},
		{Name: "schema.diff", Description: "Diff schema versions"},
		{Name: "flow.detail", Description: "Get flow detail"},
		{Name: "flow.implementations", Description: "List source declarations linked to a flow"},
		{Name: "tests.forRequirement", Description: "List test declarations linked to a requirement"},
		{Name: "tests.forEntity", Description: "List test declarations linked to a model entity"},
		{Name: "coverage.strictStatus", Description: "Report strict code-linking coverage diagnostics"},
		{Name: "ownership.resolve", Description: "Resolve ownership metadata"},
		{Name: "runtime.resolve", Description: "Resolve runtime target"},
		{Name: "confidence.explain", Description: "Explain confidence for entity"},
		{Name: "staleness.check", Description: "Check evidence staleness"},
		{Name: "changes.preflight", Description: "Preflight change impact"},
		{Name: "gemara.controlCatalog", Description: "Render the OpenSSF Gemara L2 Control Catalog (YAML) from the model"},
		{Name: "gemara.threatCatalog", Description: "Render the OpenSSF Gemara L2 Threat Catalog (YAML) from the model"},
		{Name: "gemara.riskCatalog", Description: "Render the OpenSSF Gemara L3 Risk Catalog (YAML) from the model"},
		{Name: "gemara.vectorCatalog", Description: "Render the OpenSSF Gemara L1 Vector Catalog (YAML) from the model"},
		{Name: "gemara.capabilityCatalog", Description: "Render the OpenSSF Gemara L2 Capability Catalog (YAML) from the model"},
		{Name: "gemara.principleCatalog", Description: "Render the OpenSSF Gemara L1 Principle Catalog (YAML) from the model"},
		{Name: "gemara.guidanceCatalog", Description: "Render the OpenSSF Gemara L1 Guidance Catalog (YAML) from the model"},
		{Name: "gemara.policy", Description: "Render the OpenSSF Gemara L3 Policy (YAML) from the model"},
		{Name: "gemara.lexicon", Description: "Render the OpenSSF Gemara Lexicon (YAML) from the catalog terms"},
		{Name: "gemara.mappingDocument", Description: "Render the OpenSSF Gemara control-to-threat Mapping Document (YAML)"},
		{Name: "gemara.auditLog", Description: "Render the OpenSSF Gemara L7 Audit Log (YAML) from the model"},
		{Name: "gemara.enforcementLog", Description: "Render the OpenSSF Gemara L6 Enforcement Log (YAML) from POA&M items"},
		{Name: "gemara.evaluationLog", Description: "Render the OpenSSF Gemara L5 Evaluation Log (YAML) from the model"},
		{Name: "gemara.validate", Description: "Validate all generated Gemara artifacts via the go-gemara SDK type discriminator"},
	}
	m := map[string]Tool{}
	for _, t := range all {
		m[t.Name] = t
	}
	return &Server{tools: m}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) ToolNames() []string {
	names := make([]string, 0, len(s.tools))
	for n := range s.tools {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) Handle(raw []byte) (resp []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			resp, err = json.Marshal(map[string]any{"jsonrpc": "2.0", "id": nil, "error": map[string]any{"code": -32603, "message": "internal server error"}})
		}
	}()
	var req map[string]any
	if err := json.Unmarshal(raw, &req); err != nil {
		return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": nil, "error": map[string]any{"code": -32700, "message": "parse error"}})
	}
	if version, _ := req["jsonrpc"].(string); version != "2.0" {
		return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req["id"], "error": map[string]any{"code": -32600, "message": "invalid request: jsonrpc must be 2.0"}})
	}
	method, _ := req["method"].(string)
	id, hasID := req["id"]
	if strings.TrimSpace(method) == "" {
		return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32600, "message": "invalid request: missing method"}})
	}
	if method == "notifications/initialized" || !hasID {
		return nil, nil
	}
	result, code, rpcErr := s.dispatch(method, req["params"])
	if rpcErr != nil {
		if code == 0 {
			code = -32602
		}
		return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": rpcErr.Error()}})
	}
	return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) dispatch(method string, params any) (any, int, error) {
	switch method {
	case "initialize":
		if params != nil {
			if _, ok := params.(map[string]any); !ok {
				return nil, -32602, fmt.Errorf("initialize params must be an object")
			}
		}
		if err := s.loadContext(params); err != nil {
			return nil, -32602, err
		}
		return map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "engineering-model-mcp", "version": "0.2.0"},
		}, 0, nil
	case "ping":
		return map[string]any{}, 0, nil
	case "tools/list":
		tools := make([]map[string]any, 0, len(s.tools))
		for _, name := range s.ToolNames() {
			t := s.tools[name]
			tools = append(tools, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": inputSchemaForTool(t.Name)})
		}
		return map[string]any{"tools": tools}, 0, nil
	case "tools/call":
		p, ok := params.(map[string]any)
		if !ok {
			return nil, -32602, fmt.Errorf("tools/call params must be an object")
		}
		name := strings.TrimSpace(toString(p["name"]))
		if name == "" {
			return nil, -32602, fmt.Errorf("tools/call name is required")
		}
		if _, ok := s.tools[name]; !ok {
			return nil, -32602, fmt.Errorf("unknown tool: %s", name)
		}
		if s.bundle == nil {
			return nil, -32001, fmt.Errorf("server not initialized with modelPath")
		}
		args := map[string]any{}
		if rawArgs, exists := p["arguments"]; exists && rawArgs != nil {
			typedArgs, ok := rawArgs.(map[string]any)
			if !ok {
				return nil, -32602, fmt.Errorf("tools/call arguments must be an object")
			}
			args = typedArgs
		}
		if err := validateToolArguments(name, args); err != nil {
			return nil, -32602, err
		}
		payload, err := s.callTool(name, args)
		if err != nil {
			errPayload, _ := json.Marshal(map[string]any{
				"ok":            false,
				"tool":          name,
				"schemaVersion": toolSchemaVersion,
				"error": map[string]any{
					"code":    "INVALID_ARGUMENT",
					"message": err.Error(),
					"hint":    "see tools/list.inputSchema for accepted arguments",
				},
				"generatedAt": time.Now().UTC().Format(time.RFC3339),
			})
			return map[string]any{"content": []map[string]any{{"type": "text", "text": string(errPayload)}}, "isError": true}, 0, nil
		}
		buf, _ := json.Marshal(payload)
		return map[string]any{"content": []map[string]any{{"type": "text", "text": string(buf)}}, "isError": false}, 0, nil
	default:
		return nil, -32601, fmt.Errorf("unsupported method: %s", method)
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) loadContext(params any) error {
	p, _ := params.(map[string]any)
	init, _ := p["initializationOptions"].(map[string]any)
	if init == nil {
		init = map[string]any{}
	}

	s.modelPath = nonEmptyString(init["modelPath"], s.modelPath)
	s.requirementsPath = nonEmptyString(init["requirementsPath"], s.requirementsPath)
	s.designPath = nonEmptyString(init["designPath"], s.designPath)
	s.repoRoot = nonEmptyString(init["repoRoot"], s.repoRoot)

	if s.modelPath != "" {
		absModel, err := filepath.Abs(s.modelPath)
		if err == nil {
			s.modelPath = absModel
		}
		b, err := model.LoadBundle(s.modelPath)
		if err != nil {
			return fmt.Errorf("load model bundle: %w", err)
		}
		s.bundle = &b
		if s.repoRoot == "" {
			s.repoRoot = filepath.Dir(filepath.Dir(b.ArchitecturePath))
		}
		if s.requirementsPath == "" {
			cand := filepath.Join(filepath.Dir(b.ArchitecturePath), "requirements.yml")
			if _, err := os.Stat(cand); err == nil {
				s.requirementsPath = cand
			}
		}
		if s.designPath == "" {
			cand := filepath.Join(filepath.Dir(b.ArchitecturePath), "design.yml")
			if _, err := os.Stat(cand); err == nil {
				s.designPath = cand
			}
		}
	}
	if s.requirementsPath != "" {
		if absReq, err := filepath.Abs(s.requirementsPath); err == nil {
			s.requirementsPath = absReq
		}
		r, err := model.LoadRequirements(s.requirementsPath)
		if err != nil {
			return fmt.Errorf("load requirements: %w", err)
		}
		s.requirements = &r
	}
	if s.designPath != "" {
		if absDesign, err := filepath.Abs(s.designPath); err == nil {
			s.designPath = absDesign
		}
		d, err := model.LoadDesign(s.designPath)
		if err != nil {
			return fmt.Errorf("load design: %w", err)
		}
		s.design = &d
	}
	if s.repoRoot != "" {
		if absRoot, err := filepath.Abs(s.repoRoot); err == nil {
			s.repoRoot = absRoot
		}
	}
	// reset file index if context changed
	s.indexOnce = sync.Once{}
	s.repoFiles = nil
	s.indexErr = nil
	return nil
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION, FU-CODEMAP-INFERENCE
func (s *Server) callTool(name string, args map[string]any) (map[string]any, error) {
	if s.bundle == nil {
		return map[string]any{"ok": false, "tool": name, "error": "model not loaded; pass initializationOptions.modelPath"}, nil
	}
	a := s.bundle.Architecture.AuthoredArchitecture
	reqID := firstNonEmptyArg(args, "requirementId", "id", "reqId")
	entityID := firstNonEmptyArg(args, "entityId", "id")
	interfaceID := firstNonEmptyArg(args, "interfaceId", "id", "name")
	env := firstNonEmptyArg(args, "environment", "env")
	query := strings.ToLower(firstNonEmptyArg(args, "query", "q", "name", "id"))

	simple := func(data map[string]any) (map[string]any, error) {
		data["ok"] = true
		data["tool"] = name
		data["schemaVersion"] = toolSchemaVersion
		data["generatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return data, nil
	}

	switch name {
	case "requirements.get":
		if reqID == "" {
			return nil, fmt.Errorf("requirements.get requires requirementId")
		}
		if err := requireStableID(reqID, "REQ-"); err != nil {
			return nil, err
		}
		r := s.findRequirement(reqID)
		return simple(map[string]any{"requirement": r, "exists": r != nil})
	case "requirements.impact", "requirements.supportPath", "requirements.suggestEditPlan":
		if reqID == "" {
			return nil, fmt.Errorf("%s requires requirementId", name)
		}
		if err := requireStableID(reqID, "REQ-"); err != nil {
			return nil, err
		}
		r := s.findRequirement(reqID)
		flows := s.flowsForRequirement(reqID)
		threats := s.threatsForRequirement(reqID)
		files := s.filesForRequirement(reqID)
		payload := map[string]any{"requirement": r, "impactedFlows": flows, "impactedThreats": threats, "relevantFiles": files}
		if name == "requirements.supportPath" {
			payload["supportPath"] = s.requirementSupportPath(reqID)
		}
		if name == "requirements.suggestEditPlan" {
			payload["editPlan"] = s.requirementEditPlan(reqID)
		}
		return simple(payload)
	case "files.forRequirement":
		if reqID == "" {
			return nil, fmt.Errorf("files.forRequirement requires requirementId")
		}
		if err := requireStableID(reqID, "REQ-"); err != nil {
			return nil, err
		}
		return simple(map[string]any{"files": s.filesForRequirement(reqID)})
	case "files.forControl":
		controlID := strings.TrimSpace(firstNonEmptyArg(args, "controlId", "id"))
		if controlID == "" {
			return nil, fmt.Errorf("files.forControl requires controlId")
		}
		if err := requireStableID(controlID, "CTRL-"); err != nil {
			return nil, err
		}
		return simple(map[string]any{"files": s.filesContaining(controlID)})
	case "files.forThreat":
		threatID := strings.TrimSpace(firstNonEmptyArg(args, "threatId", "id"))
		if threatID == "" {
			return nil, fmt.Errorf("files.forThreat requires threatId")
		}
		if err := requireStableID(threatID, "TS-", "THREAT-"); err != nil {
			return nil, err
		}
		return simple(map[string]any{"files": s.filesContaining(threatID)})
	case "files.owner":
		path := firstNonEmptyArg(args, "path", "file")
		if path == "" {
			return nil, fmt.Errorf("files.owner requires path")
		}
		ownerFU := ""
		safePath, ok := s.resolvePathInRepo(path)
		if !ok {
			return nil, fmt.Errorf("files.owner path must be inside repoRoot")
		}
		content, _ := os.ReadFile(safePath)
		for _, fu := range a.FunctionalUnits {
			if strings.Contains(string(content), fu.ID) {
				ownerFU = fu.ID
				break
			}
		}
		return simple(map[string]any{"path": safePath, "ownerFunctionalUnit": ownerFU})
	case "verification.status":
		status := []model.ControlVerification{}
		for _, cv := range a.ControlVerifications {
			if entityID == "" || cv.ID == entityID || cv.ControlRef == entityID {
				status = append(status, cv)
			}
		}
		return simple(map[string]any{"controlVerifications": status})
	case "verification.gaps":
		gaps := []string{}
		for _, r := range s.requirementIDs() {
			if len(s.filesForRequirement(r)) == 0 {
				gaps = append(gaps, r)
			}
		}
		return simple(map[string]any{"requirementsWithoutFiles": gaps})
	case "verification.recommend":
		return simple(map[string]any{"recommendations": s.verificationRecommendations()})
	case "gemara.controlCatalog", "gemara.threatCatalog", "gemara.riskCatalog",
		"gemara.vectorCatalog", "gemara.capabilityCatalog", "gemara.principleCatalog",
		"gemara.guidanceCatalog", "gemara.policy", "gemara.lexicon", "gemara.mappingDocument",
		"gemara.auditLog", "gemara.enforcementLog", "gemara.evaluationLog":
		data, err := s.gemaraArtifact(name)
		if err != nil {
			return nil, err
		}
		return simple(data)
	case "gemara.validate":
		data, err := s.gemaraValidate()
		if err != nil {
			return nil, err
		}
		return simple(data)
	case "threats.forRequirement":
		if reqID == "" {
			return nil, fmt.Errorf("threats.forRequirement requires requirementId")
		}
		if err := requireStableID(reqID, "REQ-"); err != nil {
			return nil, err
		}
		return simple(map[string]any{"threatScenarios": s.threatsForRequirement(reqID)})
	case "threats.coverage":
		rows := []map[string]any{}
		for _, ts := range a.ThreatScenarios {
			rows = append(rows, map[string]any{"threatScenario": ts.ID, "controls": ts.RelatedControls, "mitigations": ts.MitigationRefs, "verifications": ts.VerificationRefs})
		}
		return simple(map[string]any{"coverage": rows})
	case "threats.unmitigated":
		rows := []string{}
		for _, ts := range a.ThreatScenarios {
			if len(ts.MitigationRefs) == 0 {
				rows = append(rows, ts.ID)
			}
		}
		return simple(map[string]any{"threatScenarioIds": rows})
	case "flows.forRequirement":
		if reqID == "" {
			return nil, fmt.Errorf("flows.forRequirement requires requirementId")
		}
		if err := requireStableID(reqID, "REQ-"); err != nil {
			return nil, err
		}
		return simple(map[string]any{"flows": s.flowsForRequirement(reqID)})
	case "flows.diff":
		left := firstNonEmptyArg(args, "fromFlowId", "left")
		right := firstNonEmptyArg(args, "toFlowId", "right")
		if left == "" || right == "" {
			return nil, fmt.Errorf("flows.diff requires fromFlowId and toFlowId")
		}
		if err := requireStableID(left, "FLOW-"); err != nil {
			return nil, err
		}
		if err := requireStableID(right, "FLOW-"); err != nil {
			return nil, err
		}
		diff, err := s.flowDiff(left, right)
		if err != nil {
			return nil, err
		}
		return simple(diff)
	case "graph.neighborhood":
		return simple(map[string]any{"nodes": s.graphNodes(query, 50), "edges": s.graphEdges(query, 100)})
	case "graph.explainEdge":
		from := firstNonEmptyArg(args, "from")
		to := firstNonEmptyArg(args, "to")
		for _, m := range a.Mappings {
			if m.From == from && m.To == to {
				return simple(map[string]any{"exists": true, "mapping": m, "explanation": "Authored mapping exists"})
			}
		}
		return simple(map[string]any{"exists": false, "explanation": "No authored mapping found"})
	case "graph.search":
		return simple(map[string]any{"nodes": s.graphNodes(query, 200)})
	case "model.list", "entities.list":
		return simple(map[string]any{"entities": s.modelEntityNodes(query, firstNonEmptyArg(args, "kind"), maxArg(args, 500))})
	case "model.entity":
		id := firstNonEmptyArg(args, "entityId", "id")
		if id == "" {
			return nil, fmt.Errorf("model.entity requires entityId")
		}
		if err := requireWellFormedStableID(id); err != nil {
			return nil, err
		}
		kind, entity, ok := s.modelEntity(id)
		return simple(map[string]any{"exists": ok, "kind": kind, "entity": entity})
	case "model.implementations":
		id := firstNonEmptyArg(args, "entityId", "id")
		if id == "" {
			return nil, fmt.Errorf("model.implementations requires entityId")
		}
		if err := requireWellFormedStableID(id); err != nil {
			return nil, err
		}
		kind := s.modelEntityKind(id)
		if kind == "" {
			return nil, fmt.Errorf("model.implementations entity not found: %s", id)
		}
		implementations, diagnostics := s.implementationsForModelLink(id)
		implementations = limitRows(implementations, maxArg(args, 0))
		return simple(map[string]any{"entity": map[string]any{"id": id, "kind": kind}, "implementations": implementations, "files": groupImplementationsByFile(implementations), "diagnostics": diagnostics})
	case "code.contextForTask":
		return simple(s.contextForTask(args))
	case "views.recommend":
		taskType := strings.ToLower(firstNonEmptyArg(args, "taskType", "task"))
		return simple(map[string]any{"recommendedViews": s.recommendedViews(taskType)})
	case "views.renderContext":
		viewID := firstNonEmptyArg(args, "viewId", "id")
		if viewID == "" {
			return nil, fmt.Errorf("views.renderContext requires viewId")
		}
		if err := requireStableID(viewID, "VIEW-"); err != nil {
			return nil, err
		}
		for _, v := range s.bundle.Architecture.Views {
			if v.ID == viewID {
				return simple(map[string]any{"view": v, "rootCount": len(v.Roots)})
			}
		}
		return simple(map[string]any{"view": nil})
	case "generation.plan":
		return simple(map[string]any{"commands": s.generationCommands(), "artifacts": s.generationArtifacts()})
	case "generation.status":
		return simple(map[string]any{"modelPath": s.modelPath, "requirementsPath": s.requirementsPath, "designPath": s.designPath, "artifacts": s.generationArtifactStatus()})
	case "composition.resolve":
		data, err := s.compositionResolve()
		if err != nil {
			return nil, err
		}
		return simple(data)
	case "trace.matrix":
		data, err := s.traceMatrix()
		if err != nil {
			return nil, err
		}
		return simple(data)
	case "governance.policy":
		return simple(map[string]any{"policy": s.governancePolicy()})
	case "governance.checkPatch":
		diff := firstNonEmptyArg(args, "diff")
		issues := []string{}
		if strings.Contains(diff, "tests/") && !strings.Contains(diff, "TRLC-LINKS") {
			issues = append(issues, "test changes without TRLC-LINKS markers")
		}
		return simple(map[string]any{"issues": issues})
	case "tasks.entryPoints":
		return simple(map[string]any{"entryPoints": s.taskEntryPoints()})
	case "tasks.nextBestActions":
		return simple(map[string]any{"actions": s.nextBestActions()})
	case "interfaces.resolve", "endpoints.resolve":
		if interfaceID == "" {
			return nil, fmt.Errorf("%s requires interfaceId", name)
		}
		if err := requireStableID(interfaceID, "IF-"); err != nil {
			return nil, err
		}
		iface := s.findInterface(interfaceID)
		if iface == nil {
			return nil, fmt.Errorf("%s interface not found: %s", name, interfaceID)
		}
		return simple(map[string]any{"interface": iface, "environment": env, "resolvedEndpoint": endpointForEnv(iface, env)})
	case "interfaces.implementations":
		if interfaceID == "" {
			return nil, fmt.Errorf("interfaces.implementations requires interfaceId")
		}
		if err := requireStableID(interfaceID, "IF-"); err != nil {
			return nil, err
		}
		iface := s.findInterface(interfaceID)
		if iface == nil {
			return nil, fmt.Errorf("interfaces.implementations interface not found: %s", interfaceID)
		}
		implementations, diagnostics := s.implementationsForModelLink(interfaceID)
		implementations = limitRows(implementations, maxArg(args, 0))
		return simple(map[string]any{"interface": iface, "implementations": implementations, "files": groupImplementationsByFile(implementations), "diagnostics": diagnostics})
	case "interfaces.matchFromCode":
		path := firstNonEmptyArg(args, "path", "file")
		if path == "" {
			return nil, fmt.Errorf("interfaces.matchFromCode requires path")
		}
		safePath, ok := s.resolvePathInRepo(path)
		if !ok {
			return nil, fmt.Errorf("interfaces.matchFromCode path must be inside repoRoot")
		}
		content, _ := os.ReadFile(safePath)
		ownerFU := ""
		for _, fu := range a.FunctionalUnits {
			if strings.Contains(string(content), fu.ID) {
				ownerFU = fu.ID
				break
			}
		}
		matched := []string{}
		for _, i := range a.Interfaces {
			if strings.Contains(string(content), i.ID) || strings.Contains(string(content), i.Endpoint) {
				matched = append(matched, i.ID)
			}
		}
		return simple(map[string]any{"path": safePath, "interfaces": matched, "ownerFunctionalUnit": ownerFU})
	case "interfaces.ambiguities":
		byName := map[string]int{}
		for _, i := range a.Interfaces {
			byName[strings.ToLower(strings.TrimSpace(i.Name))]++
		}
		amb := []string{}
		for n, c := range byName {
			if c > 1 {
				amb = append(amb, n)
			}
		}
		sort.Strings(amb)
		return simple(map[string]any{"ambiguousInterfaceNames": amb})
	case "environments.resolve":
		rows := []model.DeploymentTarget{}
		for _, d := range a.DeploymentTargets {
			if env == "" || strings.EqualFold(d.Environment, env) {
				rows = append(rows, d)
			}
		}
		return simple(map[string]any{"deploymentTargets": rows})
	case "identity.resolve":
		if interfaceID == "" {
			return nil, fmt.Errorf("identity.resolve requires interfaceId")
		}
		if err := requireStableID(interfaceID, "IF-"); err != nil {
			return nil, err
		}
		iface := s.findInterface(interfaceID)
		if iface == nil {
			return nil, fmt.Errorf("identity.resolve interface not found: %s", interfaceID)
		}
		return simple(map[string]any{"interface": iface, "authHints": s.authHintsForInterface(interfaceID)})
	case "policy.resolve":
		if interfaceID == "" && firstNonEmptyArg(args, "id") == "" {
			return nil, fmt.Errorf("policy.resolve requires interfaceId or id")
		}
		id := firstNonEmptyArg(args, "id")
		if id == "" {
			id = interfaceID
		}
		return simple(s.policyForEntity(id))
	case "schema.resolve":
		if interfaceID == "" {
			return nil, fmt.Errorf("schema.resolve requires interfaceId")
		}
		if err := requireStableID(interfaceID, "IF-"); err != nil {
			return nil, err
		}
		iface := s.findInterface(interfaceID)
		if iface == nil {
			return nil, fmt.Errorf("schema.resolve interface not found: %s", interfaceID)
		}
		return simple(map[string]any{"schemaRef": iface.SchemaRef, "protocol": iface.Protocol})
	case "schema.diff":
		left := firstNonEmptyArg(args, "fromSchemaRef", "leftSchemaRef")
		right := firstNonEmptyArg(args, "toSchemaRef", "rightSchemaRef")
		if left == "" {
			left = s.schemaRefForInterface(firstNonEmptyArg(args, "fromInterfaceId", "leftInterfaceId"))
		}
		if right == "" {
			right = s.schemaRefForInterface(firstNonEmptyArg(args, "toInterfaceId", "rightInterfaceId"))
		}
		if left == "" && right == "" {
			return nil, fmt.Errorf("schema.diff requires schema refs or interface ids")
		}
		status, note, leftMeta, rightMeta := s.diffSchemaRefs(left, right)
		return simple(map[string]any{"fromSchemaRef": left, "toSchemaRef": right, "status": status, "message": note, "fromSchema": leftMeta, "toSchema": rightMeta})
	case "flow.detail":
		flowID := firstNonEmptyArg(args, "flowId", "id")
		if flowID == "" {
			return nil, fmt.Errorf("flow.detail requires flowId")
		}
		if err := requireStableID(flowID, "FLOW-"); err != nil {
			return nil, err
		}
		flow := s.findFlow(flowID)
		return simple(map[string]any{"exists": flow != nil, "flow": flow, "implementations": s.implementationFilesForModelLink(flowID)})
	case "flow.implementations":
		flowID := firstNonEmptyArg(args, "flowId", "id")
		if flowID == "" {
			return nil, fmt.Errorf("flow.implementations requires flowId")
		}
		if err := requireStableID(flowID, "FLOW-"); err != nil {
			return nil, err
		}
		if s.findFlow(flowID) == nil {
			return nil, fmt.Errorf("flow.implementations flow not found: %s", flowID)
		}
		implementations, diagnostics := s.implementationsForModelLink(flowID)
		implementations = limitRows(implementations, maxArg(args, 0))
		return simple(map[string]any{"flowId": flowID, "implementations": implementations, "files": groupImplementationsByFile(implementations), "diagnostics": diagnostics})
	case "tests.forRequirement":
		r := firstNonEmptyArg(args, "requirementId", "id", "reqId")
		if r == "" {
			return nil, fmt.Errorf("tests.forRequirement requires requirementId")
		}
		if err := requireStableID(r, "REQ-"); err != nil {
			return nil, err
		}
		tests, diagnostics := s.testsForRequirement(r)
		tests = limitRows(tests, maxArg(args, 0))
		return simple(map[string]any{"requirementId": r, "tests": tests, "files": groupImplementationsByFile(tests), "diagnostics": diagnostics})
	case "tests.forEntity":
		id := firstNonEmptyArg(args, "entityId", "id")
		if id == "" {
			return nil, fmt.Errorf("tests.forEntity requires entityId")
		}
		if err := requireWellFormedStableID(id); err != nil {
			return nil, err
		}
		if s.modelEntityKind(id) == "" {
			return nil, fmt.Errorf("tests.forEntity entity not found: %s", id)
		}
		tests, diagnostics := s.testsForModelLink(id)
		tests = limitRows(tests, maxArg(args, 0))
		return simple(map[string]any{"entityId": id, "tests": tests, "files": groupImplementationsByFile(tests), "diagnostics": diagnostics})
	case "coverage.strictStatus":
		path := firstNonEmptyArg(args, "path", "file")
		status := s.strictCoverageStatus(path)
		return simple(status)
	case "ownership.resolve":
		if entityID == "" {
			return nil, fmt.Errorf("ownership.resolve requires entityId")
		}
		if err := requireStableID(entityID, "REQ-", "IF-", "FU-", "CTRL-", "TS-", "RISK-"); err != nil {
			return nil, err
		}
		return simple(map[string]any{"id": entityID, "owner": s.ownerFor(entityID)})
	case "runtime.resolve":
		return simple(s.runtimeForEntity(entityID))
	case "confidence.explain":
		if entityID == "" {
			return nil, fmt.Errorf("confidence.explain requires entityId")
		}
		if err := requireStableID(entityID, "REQ-", "IF-", "FU-", "CTRL-", "TS-", "RISK-", "FLOW-", "VIEW-", "DO-", "DEP-", "TB-", "EVT-", "STATE-"); err != nil {
			return nil, err
		}
		return simple(s.confidenceForEntity(entityID))
	case "staleness.check":
		path := firstNonEmptyArg(args, "path")
		if path == "" {
			path = s.modelPath
		}
		return simple(s.stalenessForPath(path))
	case "changes.preflight":
		r := firstNonEmptyArg(args, "requirementId", "id")
		if r == "" {
			return nil, fmt.Errorf("changes.preflight requires requirementId")
		}
		if err := requireStableID(r, "REQ-"); err != nil {
			return nil, err
		}
		return simple(map[string]any{"requirementId": r, "impactedFiles": s.filesForRequirement(r), "rerun": []string{"go test ./...", "engdoc", "engdragon", "engstruct", "engtrlc"}})
	default:
		return nil, fmt.Errorf("unsupported tool: %s", name)
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) findRequirement(id string) *model.Requirement {
	if s.requirements == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	for i := range s.requirements.Requirements {
		if s.requirements.Requirements[i].ID == id {
			return &s.requirements.Requirements[i]
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) requirementIDs() []string {
	if s.requirements == nil {
		return nil
	}
	out := make([]string, 0, len(s.requirements.Requirements))
	for _, r := range s.requirements.Requirements {
		out = append(out, r.ID)
	}
	sort.Strings(out)
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) flowsForRequirement(reqID string) []model.Flow {
	if s.bundle == nil {
		return nil
	}
	a := s.bundle.Architecture.AuthoredArchitecture
	req := s.findRequirement(reqID)
	applies := map[string]bool{}
	if req != nil {
		for _, x := range req.AppliesTo {
			applies[x] = true
		}
	}
	out := []model.Flow{}
	for _, f := range a.Flows {
		if reqID == "" {
			out = append(out, f)
			continue
		}
		if applies[f.SourceRef] || applies[f.DestinationRef] {
			out = append(out, f)
			continue
		}
		for _, st := range f.Steps {
			if applies[st.Ref] || applies[st.SourceRef] || applies[st.DestinationRef] || applies[st.InterfaceRef] {
				out = append(out, f)
				break
			}
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER
func (s *Server) threatsForRequirement(reqID string) []model.ThreatScenario {
	if s.bundle == nil {
		return nil
	}
	a := s.bundle.Architecture.AuthoredArchitecture
	req := s.findRequirement(reqID)
	applies := map[string]bool{}
	if req != nil {
		for _, x := range req.AppliesTo {
			applies[x] = true
		}
	}
	out := []model.ThreatScenario{}
	for _, ts := range a.ThreatScenarios {
		if reqID == "" {
			out = append(out, ts)
			continue
		}
		for _, x := range ts.AppliesTo {
			if applies[x] {
				out = append(out, ts)
				break
			}
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) filesForRequirement(reqID string) []string {
	if strings.TrimSpace(reqID) == "" {
		return nil
	}
	return s.filesContaining(reqID)
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) filesContaining(token string) []string {
	token = strings.TrimSpace(token)
	if token == "" || s.repoRoot == "" {
		return nil
	}
	if err := s.ensureRepoIndex(); err != nil {
		return nil
	}
	out := []string{}
	for _, f := range s.repoFiles {
		if strings.Contains(f.Content, token) {
			out = append(out, f.Path)
		}
	}
	sort.Strings(out)
	if len(out) > 50 {
		out = out[:50]
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) ensureRepoIndex() error {
	s.indexOnce.Do(func() {
		s.indexErr = filepath.Walk(s.repoRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				switch info.Name() {
				case ".git", "node_modules", ".opencode", ".idea", ".vscode":
					return filepath.SkipDir
				default:
					return nil
				}
			}
			if len(s.repoFiles) >= maxIndexedFiles {
				return errRepoIndexLimit
			}
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".go", ".ts", ".tsx", ".rs", ".py", ".js", ".yaml", ".yml", ".md":
			default:
				return nil
			}
			if info.Size() > maxIndexedFileBytes {
				return nil
			}
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			s.repoFiles = append(s.repoFiles, indexedFile{Path: path, Content: string(b)})
			return nil
		})
		if errors.Is(s.indexErr, errRepoIndexLimit) {
			s.indexErr = nil
		}
	})
	return s.indexErr
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) findInterface(idOrName string) *model.Interface {
	idOrName = strings.TrimSpace(idOrName)
	if s.bundle == nil || idOrName == "" {
		return nil
	}
	for i := range s.bundle.Architecture.AuthoredArchitecture.Interfaces {
		x := &s.bundle.Architecture.AuthoredArchitecture.Interfaces[i]
		if x.ID == idOrName || strings.EqualFold(x.Name, idOrName) {
			return x
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) requirementSupportPath(reqID string) []map[string]any {
	req := s.findRequirement(reqID)
	if req == nil {
		return nil
	}
	out := []map[string]any{{"kind": "requirement", "id": req.ID, "text": req.Text, "appliesTo": req.AppliesTo}}
	for _, appliesTo := range req.AppliesTo {
		kind := s.modelEntityKind(appliesTo)
		out = append(out, map[string]any{"kind": kind, "id": appliesTo, "relationship": "applies_to", "implementations": s.implementationFilesForModelLink(appliesTo)})
	}
	for _, flow := range s.flowsForRequirement(reqID) {
		out = append(out, map[string]any{"kind": "flow", "id": flow.ID, "relationship": "impacted_flow", "title": flow.Title, "implementations": s.implementationFilesForModelLink(flow.ID)})
	}
	for _, threat := range s.threatsForRequirement(reqID) {
		out = append(out, map[string]any{"kind": "threat_scenario", "id": threat.ID, "relationship": "impacted_threat", "title": threat.Title, "controls": threat.RelatedControls, "verifications": threat.VerificationRefs})
	}
	for _, file := range s.filesForRequirement(reqID) {
		out = append(out, map[string]any{"kind": "file", "path": file, "relationship": "mentions_requirement"})
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) requirementEditPlan(reqID string) []map[string]any {
	req := s.findRequirement(reqID)
	if req == nil {
		return []map[string]any{{"step": "check_requirement_id", "reason": "requirement was not found", "requirementId": reqID}}
	}
	steps := []map[string]any{
		{"step": "update_requirement", "reason": "change the authored requirement text and appliesTo list first", "files": []string{s.requirementsPath}, "ids": []string{req.ID}},
	}
	if len(req.AppliesTo) > 0 {
		steps = append(steps, map[string]any{"step": "review_model_entities", "reason": "requirement appliesTo entities define the design surface to inspect", "ids": req.AppliesTo})
	}
	flows := s.flowsForRequirement(reqID)
	if len(flows) > 0 {
		ids := []string{}
		for _, f := range flows {
			ids = append(ids, f.ID)
		}
		steps = append(steps, map[string]any{"step": "review_flows", "reason": "flows touch the requirement appliesTo entities", "ids": ids})
	}
	files := s.filesForRequirement(reqID)
	if len(files) > 0 {
		steps = append(steps, map[string]any{"step": "update_linked_code_and_tests", "reason": "source files mention or verify the requirement", "files": files})
	}
	steps = append(steps, map[string]any{"step": "run_verification", "reason": "validate model loading, code links, and generated MCP output", "commands": []string{"go test ./...", "go run ./cmd/engmcp"}})
	return steps
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) verificationRecommendations() []map[string]any {
	out := []map[string]any{}
	for _, reqID := range s.requirementIDs() {
		tests, _ := s.testsForRequirement(reqID)
		if len(tests) == 0 {
			out = append(out, map[string]any{"action": "add_requirement_test_links", "requirementId": reqID, "reason": "no test declarations with TRLC-LINKS were found for this requirement"})
		}
	}
	if s.bundle != nil {
		for _, cv := range s.bundle.Architecture.AuthoredArchitecture.ControlVerifications {
			if !strings.EqualFold(cv.Status, "pass") {
				out = append(out, map[string]any{"action": "complete_control_verification", "controlVerificationId": cv.ID, "controlId": cv.ControlRef, "status": cv.Status, "reason": "control verification is not passing"})
			}
			if len(cv.Evidence) == 0 {
				out = append(out, map[string]any{"action": "add_control_verification_evidence", "controlVerificationId": cv.ID, "controlId": cv.ControlRef, "reason": "control verification has no evidence paths"})
			}
		}
	}
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
	fields := []map[string]any{}
	addField := func(field, l, r string) {
		if strings.TrimSpace(l) != strings.TrimSpace(r) {
			fields = append(fields, map[string]any{"field": field, "from": l, "to": r})
		}
	}
	addField("title", left.Title, right.Title)
	addField("kind", left.Kind, right.Kind)
	addField("methodology", left.Methodology, right.Methodology)
	addField("direction", left.Direction, right.Direction)
	addField("frequency", left.Frequency, right.Frequency)
	addField("sourceRef", left.SourceRef, right.SourceRef)
	addField("destinationRef", left.DestinationRef, right.DestinationRef)
	addField("protocol", left.Protocol, right.Protocol)
	addField("channel", left.Channel, right.Channel)
	addField("authentication", left.Authentication, right.Authentication)
	addField("encryptionInTransit", left.EncryptionInTransit, right.EncryptionInTransit)
	addField("integrityProtection", left.IntegrityProtection, right.IntegrityProtection)
	addField("criticality", left.Criticality, right.Criticality)

	slices := []map[string]any{}
	addSliceDiff := func(field string, l, r []string) {
		added, removed, common := sliceSetDiff(l, r)
		if len(added) > 0 || len(removed) > 0 {
			slices = append(slices, map[string]any{"field": field, "added": added, "removed": removed, "common": common})
		}
	}
	addSliceDiff("dataRefs", left.DataRefs, right.DataRefs)
	addSliceDiff("threats", left.Threats, right.Threats)
	addSliceDiff("entry", left.Entry, right.Entry)
	addSliceDiff("exits", left.Exits, right.Exits)

	leftSteps := flowStepsByID(left.Steps)
	rightSteps := flowStepsByID(right.Steps)
	leftStepIDs := mapKeys(leftSteps)
	rightStepIDs := mapKeys(rightSteps)
	addedSteps, removedSteps, commonSteps := sliceSetDiff(leftStepIDs, rightStepIDs)
	changedSteps := []string{}
	for _, id := range commonSteps {
		lb, _ := json.Marshal(leftSteps[id])
		rb, _ := json.Marshal(rightSteps[id])
		if string(lb) != string(rb) {
			changedSteps = append(changedSteps, id)
		}
	}
	return map[string]any{
		"from": leftID,
		"to":   rightID,
		"fromFlow": map[string]any{
			"id": left.ID, "title": left.Title, "sourceRef": left.SourceRef, "destinationRef": left.DestinationRef,
		},
		"toFlow": map[string]any{
			"id": right.ID, "title": right.Title, "sourceRef": right.SourceRef, "destinationRef": right.DestinationRef,
		},
		"fieldChanges": fields,
		"setChanges":   slices,
		"stepChanges":  map[string]any{"added": addedSteps, "removed": removedSteps, "changed": changedSteps, "common": commonSteps},
	}, nil
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func flowStepsByID(steps []model.FlowStep) map[string]model.FlowStep {
	out := map[string]model.FlowStep{}
	for _, step := range steps {
		out[step.ID] = step
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func sliceSetDiff(left, right []string) (added, removed, common []string) {
	leftSeen := map[string]bool{}
	rightSeen := map[string]bool{}
	for _, x := range left {
		x = strings.TrimSpace(x)
		if x != "" {
			leftSeen[x] = true
		}
	}
	for _, x := range right {
		x = strings.TrimSpace(x)
		if x != "" {
			rightSeen[x] = true
		}
	}
	for x := range rightSeen {
		if !leftSeen[x] {
			added = append(added, x)
		} else {
			common = append(common, x)
		}
	}
	for x := range leftSeen {
		if !rightSeen[x] {
			removed = append(removed, x)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(common)
	return added, removed, common
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func mapKeys[V any](in map[string]V) []string {
	out := make([]string, 0, len(in))
	for k := range in {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-VIEW-PROJECTION
func (s *Server) recommendedViews(taskType string) []map[string]any {
	taskType = strings.ToLower(strings.TrimSpace(taskType))
	type candidate struct {
		kind   string
		reason string
	}
	candidates := []candidate{{kind: "traceability", reason: "default view for implementation and requirement impact work"}}
	switch {
	case strings.Contains(taskType, "security") || strings.Contains(taskType, "threat") || strings.Contains(taskType, "control"):
		candidates = []candidate{{kind: "security", reason: "security tasks need controls, threats, and trust boundaries"}, {kind: "traceability", reason: "traceability shows requirements and verification links"}}
	case strings.Contains(taskType, "deploy") || strings.Contains(taskType, "runtime") || strings.Contains(taskType, "environment"):
		candidates = []candidate{{kind: "deployment", reason: "deployment tasks need runtime targets and environment boundaries"}, {kind: "communication", reason: "communication shows runtime interaction surfaces"}}
	case strings.Contains(taskType, "flow") || strings.Contains(taskType, "api") || strings.Contains(taskType, "interface"):
		candidates = []candidate{{kind: "communication", reason: "interface work needs service and data interaction context"}, {kind: "traceability", reason: "traceability links interfaces back to requirements"}}
	}
	out := []map[string]any{}
	for _, c := range candidates {
		for _, v := range s.bundle.Architecture.Views {
			if strings.EqualFold(v.Kind, c.kind) {
				out = append(out, map[string]any{"id": v.ID, "kind": v.Kind, "reason": c.reason, "roots": v.Roots, "audience": v.Audience})
			}
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
func (s *Server) generationCommands() []map[string]any {
	modelPath := s.modelPath
	requirementsPath := s.requirementsPath
	designPath := s.designPath
	generatedDir := s.generatedDir()
	codeRoot := s.repoRoot
	if codeRoot == "" {
		codeRoot = filepath.Dir(filepath.Dir(modelPath))
	}
	commands := []map[string]any{
		{"id": "engdoc.asciidoc", "command": fmt.Sprintf("go run ./cmd/engdoc --model %s --requirements %s --design %s --code-root %s --out %s --decisions-out %s", shellPath(modelPath), shellPath(requirementsPath), shellPath(designPath), shellPath(codeRoot), shellPath(filepath.Join(generatedDir, "ARCHITECTURE.adoc")), shellPath(filepath.Join(generatedDir, "DECISIONS.adoc"))), "outputs": []string{filepath.Join(generatedDir, "ARCHITECTURE.adoc"), filepath.Join(generatedDir, "DECISIONS.adoc")}},
		{"id": "engdragon.threatdragon", "command": fmt.Sprintf("go run ./cmd/engdragon --model %s --format threat-dragon-v2 --out %s", shellPath(modelPath), shellPath(filepath.Join(generatedDir, "threat-dragon-v2.json"))), "outputs": []string{filepath.Join(generatedDir, "threat-dragon-v2.json")}},
		{"id": "engdragon.openotm", "command": fmt.Sprintf("go run ./cmd/engdragon --model %s --format open-otm --out %s", shellPath(modelPath), shellPath(filepath.Join(generatedDir, "open-threat-model.json"))), "outputs": []string{filepath.Join(generatedDir, "open-threat-model.json")}},
		{"id": "engstruct", "command": fmt.Sprintf("go run ./cmd/engstruct --model %s --out %s", shellPath(modelPath), shellPath(filepath.Join(generatedDir, "STRUCTURIZR.dsl"))), "outputs": []string{filepath.Join(generatedDir, "STRUCTURIZR.dsl")}},
		{"id": "engtrlc", "command": fmt.Sprintf("go run ./cmd/engtrlc --requirements %s --out-dir %s", shellPath(requirementsPath), shellPath(filepath.Join(generatedDir, "trlc"))), "outputs": []string{filepath.Join(generatedDir, "trlc", "model.rsl"), filepath.Join(generatedDir, "trlc", "requirements.trlc")}},
		{"id": "engoscal", "command": fmt.Sprintf("go run ./cmd/engoscal --model %s --requirements %s --code-root %s --ssp-out %s --ar-out %s --poam-out %s", shellPath(modelPath), shellPath(requirementsPath), shellPath(codeRoot), shellPath(filepath.Join(generatedDir, "ARCHITECTURE.ssp.json")), shellPath(filepath.Join(generatedDir, "ARCHITECTURE.ar.json")), shellPath(filepath.Join(generatedDir, "ARCHITECTURE.poam.json"))), "outputs": []string{filepath.Join(generatedDir, "ARCHITECTURE.ssp.json"), filepath.Join(generatedDir, "ARCHITECTURE.ar.json"), filepath.Join(generatedDir, "ARCHITECTURE.poam.json")}},
	}
	for _, view := range s.bundle.Architecture.Views {
		out := filepath.Join(generatedDir, view.ID+".mmd")
		commands = append(commands, map[string]any{"id": "engview." + view.ID, "command": fmt.Sprintf("go run ./cmd/engview --model %s --view %s --out %s", shellPath(modelPath), shellPath(view.ID), shellPath(out)), "outputs": []string{out}})
	}
	return commands
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func shellPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "<missing>"
	}
	if strings.ContainsAny(path, " \t\n\"'") {
		return strconv.Quote(path)
	}
	return path
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) generatedDir() string {
	if s.modelPath == "" {
		return "generated"
	}
	return filepath.Join(filepath.Dir(s.modelPath), "generated")
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) generationArtifacts() []map[string]any {
	generatedDir := s.generatedDir()
	out := []map[string]any{
		{"id": "ARCHITECTURE.adoc", "path": filepath.Join(generatedDir, "ARCHITECTURE.adoc"), "producer": "engdoc.asciidoc"},
		{"id": "DECISIONS.adoc", "path": filepath.Join(generatedDir, "DECISIONS.adoc"), "producer": "engdoc.asciidoc"},
		{"id": "threat-dragon-v2.json", "path": filepath.Join(generatedDir, "threat-dragon-v2.json"), "producer": "engdragon.threatdragon"},
		{"id": "open-threat-model.json", "path": filepath.Join(generatedDir, "open-threat-model.json"), "producer": "engdragon.openotm"},
		{"id": "STRUCTURIZR.dsl", "path": filepath.Join(generatedDir, "STRUCTURIZR.dsl"), "producer": "engstruct"},
		{"id": "ARCHITECTURE.ssp.json", "path": filepath.Join(generatedDir, "ARCHITECTURE.ssp.json"), "producer": "engoscal"},
		{"id": "ARCHITECTURE.ar.json", "path": filepath.Join(generatedDir, "ARCHITECTURE.ar.json"), "producer": "engoscal"},
		{"id": "ARCHITECTURE.poam.json", "path": filepath.Join(generatedDir, "ARCHITECTURE.poam.json"), "producer": "engoscal"},
		{"id": "trlc.model", "path": filepath.Join(generatedDir, "trlc", "model.rsl"), "producer": "engtrlc"},
		{"id": "trlc.requirements", "path": filepath.Join(generatedDir, "trlc", "requirements.trlc"), "producer": "engtrlc"},
	}
	for _, view := range s.bundle.Architecture.Views {
		out = append(out, map[string]any{"id": view.ID + ".mmd", "path": filepath.Join(generatedDir, view.ID+".mmd"), "producer": "engview", "viewId": view.ID})
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) generationArtifactStatus() []map[string]any {
	out := []map[string]any{}
	for _, artifact := range s.generationArtifacts() {
		path, _ := artifact["path"].(string)
		status := s.fileStatus(path)
		for k, v := range artifact {
			status[k] = v
		}
		out = append(out, status)
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) fileStatus(path string) map[string]any {
	out := map[string]any{"path": path, "exists": false}
	if info, err := os.Stat(path); err == nil {
		out["exists"] = true
		out["bytes"] = info.Size()
		out["modifiedAt"] = info.ModTime().UTC().Format(time.RFC3339)
		out["ageHours"] = int(time.Since(info.ModTime()).Hours())
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-STRICT-MCP-INPUT-SCHEMA, CTRL-TRACEABILITY-COVERAGE, DEP-CI-PIPELINE
func (s *Server) governancePolicy() map[string]any {
	mode := ""
	if s.requirements != nil {
		mode = s.requirements.LintRun.Mode
	}
	return map[string]any{
		"lintMode":           mode,
		"requiredMarkers":    []string{"ENGMODEL-OWNER-UNIT", "ENGMODEL-LINKS", "TRLC-LINKS"},
		"supportedCodeGlobs": []string{"**/*.go", "**/*.ts", "**/*.tsx", "**/*.rs"},
		"rules": []map[string]any{
			{"id": "concrete_model_links", "description": "ENGMODEL-LINKS values must point to concrete authored model entities, not generic aliases"},
			{"id": "function_requirement_links", "description": "function and method declarations need TRLC-LINKS requirement IDs when strict coverage is enforced"},
			{"id": "repo_path_boundary", "description": "MCP file tools only resolve paths inside repoRoot"},
			{"id": "artifact_regeneration", "description": "model, requirements, design, and code-link changes require regenerating affected documentation and exports"},
		},
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE
func (s *Server) taskEntryPoints() []map[string]any {
	a := s.bundle.Architecture.AuthoredArchitecture
	return []map[string]any{
		{"id": "requirements", "count": len(s.requirementIDs()), "ids": s.requirementIDs(), "use": []string{"requirements.get", "requirements.impact", "code.contextForTask"}},
		{"id": "functional_units", "count": len(a.FunctionalUnits), "use": []string{"model.entity", "model.implementations", "ownership.resolve"}},
		{"id": "interfaces", "count": len(a.Interfaces), "use": []string{"interfaces.resolve", "interfaces.implementations", "schema.resolve", "policy.resolve"}},
		{"id": "flows", "count": len(a.Flows), "use": []string{"flow.detail", "flow.implementations", "flows.diff"}},
		{"id": "security", "count": len(a.ThreatScenarios), "use": []string{"threats.coverage", "policy.resolve", "verification.status"}},
		{"id": "coverage", "use": []string{"coverage.strictStatus", "verification.gaps", "verification.recommend"}},
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) nextBestActions() []map[string]any {
	out := []map[string]any{}
	status := s.strictCoverageStatus("")
	if status["status"] == "fail" {
		out = append(out, map[string]any{"action": "fix_strict_code_linking", "tool": "coverage.strictStatus", "reason": "source diagnostics include strict coverage failures", "summary": status["summary"]})
	}
	for _, reqID := range s.requirementIDs() {
		if len(s.filesForRequirement(reqID)) == 0 {
			out = append(out, map[string]any{"action": "link_requirement_to_code_or_tests", "tool": "files.forRequirement", "requirementId": reqID, "reason": "requirement has no indexed file mentions"})
		}
	}
	for _, risk := range s.bundle.Architecture.AuthoredArchitecture.Risks {
		if strings.EqualFold(risk.Status, "open") {
			out = append(out, map[string]any{"action": "reduce_open_risk", "tool": "threats.coverage", "riskId": risk.ID, "reason": risk.Rationale})
		}
	}
	for _, cv := range s.bundle.Architecture.AuthoredArchitecture.ControlVerifications {
		if !strings.EqualFold(cv.Status, "pass") {
			out = append(out, map[string]any{"action": "complete_control_verification", "tool": "verification.status", "controlVerificationId": cv.ID, "status": cv.Status, "reason": "verification is not passing"})
		}
	}
	if len(out) == 0 {
		out = append(out, map[string]any{"action": "inspect_task_context", "tool": "code.contextForTask", "reason": "no immediate gaps found; use task context before editing"})
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE
func (s *Server) policyForEntity(id string) map[string]any {
	a := s.bundle.Architecture.AuthoredArchitecture
	controls := []string{}
	mappings := []model.Mapping{}
	complianceMappings := []model.ComplianceMapping{}
	threats := []string{}
	risks := []string{}
	for _, m := range a.Mappings {
		if (m.Type == "guarded_by" || m.Type == "mitigated_by") && m.From == id {
			controls = append(controls, m.To)
			mappings = append(mappings, m)
		}
	}
	for _, cm := range s.bundle.Architecture.Compliance.Mappings {
		if containsString(cm.AppliesTo, id) {
			controls = append(controls, cm.ModelControlRef)
			complianceMappings = append(complianceMappings, cm)
		}
	}
	for _, ts := range a.ThreatScenarios {
		if containsString(ts.AppliesTo, id) || ts.EntryPoint == id || containsString(ts.FlowRefs, id) {
			threats = append(threats, ts.ID)
			controls = append(controls, ts.RelatedControls...)
		}
	}
	for _, risk := range a.Risks {
		if containsString(risk.AppliesTo, id) || containsString(risk.ThreatScenarios, id) {
			risks = append(risks, risk.ID)
			controls = append(controls, risk.RelatedControls...)
		}
	}
	sort.Strings(threats)
	sort.Strings(risks)
	return map[string]any{"entityId": id, "controls": uniqueStrings(controls), "complianceMappings": complianceMappings, "threatScenarios": uniqueStrings(threats), "risks": uniqueStrings(risks), "mappings": mappings}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) runtimeForEntity(id string) map[string]any {
	a := s.bundle.Architecture.AuthoredArchitecture
	targetIDs := []string{}
	mappings := []model.Mapping{}
	idsToCheck := []string{id}
	if iface := s.findInterface(id); iface != nil && iface.Owner != "" {
		idsToCheck = append(idsToCheck, iface.Owner)
	}
	for _, m := range a.Mappings {
		if m.Type == "deployed_to" && (id == "" || containsString(idsToCheck, m.From)) {
			targetIDs = append(targetIDs, m.To)
			mappings = append(mappings, m)
		}
	}
	targetIDs = uniqueStrings(targetIDs)
	targets := []model.DeploymentTarget{}
	for _, dep := range a.DeploymentTargets {
		if containsString(targetIDs, dep.ID) {
			targets = append(targets, dep)
		}
	}
	return map[string]any{"entityId": id, "runtimeTargets": targetIDs, "deploymentTargets": targets, "mappings": mappings, "runtimeSources": s.bundle.Architecture.InferenceHints.RuntimeSources}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) confidenceForEntity(id string) map[string]any {
	kind, _, exists := s.modelEntity(id)
	if !exists {
		for _, view := range s.bundle.Architecture.Views {
			if view.ID == id {
				kind = "view"
				exists = true
				break
			}
		}
	}
	implementations, diagnostics := s.implementationsForModelLink(id)
	mappings := []model.Mapping{}
	for _, m := range s.bundle.Architecture.AuthoredArchitecture.Mappings {
		if m.From == id || m.To == id {
			mappings = append(mappings, m)
		}
	}
	reqs := s.requirementsForEntity(id)
	basis := []string{}
	if exists {
		basis = append(basis, "authored model entity")
	}
	if len(mappings) > 0 {
		basis = append(basis, "authored mappings")
	}
	if len(reqs) > 0 {
		basis = append(basis, "requirement appliesTo links")
	}
	if len(implementations) > 0 {
		basis = append(basis, "source ENGMODEL-LINKS")
	}
	confidence := "low"
	if exists && (len(mappings) > 0 || len(implementations) > 0 || len(reqs) > 0) {
		confidence = "high"
	} else if exists {
		confidence = "medium"
	}
	return map[string]any{"id": id, "kind": kind, "exists": exists, "confidence": confidence, "basis": basis, "mappingCount": len(mappings), "requirementIds": reqs, "implementationFiles": groupImplementationsByFile(implementations), "diagnostics": diagnostics}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE
func (s *Server) requirementsForEntity(id string) []string {
	out := []string{}
	if s.requirements == nil {
		return out
	}
	for _, req := range s.requirements.Requirements {
		if containsString(req.AppliesTo, id) {
			out = append(out, req.ID)
		}
	}
	sort.Strings(out)
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) stalenessForPath(path string) map[string]any {
	originalPath := strings.TrimSpace(path)
	checkPath := originalPath
	if checkPath != "" && s.repoRoot != "" {
		if resolved, ok := s.resolvePathInRepo(checkPath); ok {
			checkPath = resolved
		}
	}
	out := s.fileStatus(checkPath)
	out["requestedPath"] = originalPath
	staleness := "missing"
	if out["exists"] == true {
		ageHours, _ := out["ageHours"].(int)
		switch {
		case ageHours < 24:
			staleness = "fresh"
		case ageHours < 168:
			staleness = "stale-soon"
		default:
			staleness = "stale"
		}
	}
	out["staleness"] = staleness
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) implementationsForModelLink(modelID string) ([]map[string]any, []map[string]any) {
	symbols, diagnostics := s.scanCodeSymbols()
	return symbolRows(symbols, func(sym codemap.Symbol) bool {
		return containsString(sym.ModelLinks, modelID)
	}), diagnostics
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) implementationFilesForModelLink(modelID string) []map[string]any {
	implementations, _ := s.implementationsForModelLink(modelID)
	return groupImplementationsByFile(implementations)
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) testsForRequirement(reqID string) ([]map[string]any, []map[string]any) {
	symbols, diagnostics := s.scanCodeSymbols()
	return symbolRows(symbols, func(sym codemap.Symbol) bool {
		return isTestPath(sym.Path) && containsString(sym.Implements, reqID)
	}), diagnostics
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) testsForModelLink(modelID string) ([]map[string]any, []map[string]any) {
	symbols, diagnostics := s.scanCodeSymbols()
	return symbolRows(symbols, func(sym codemap.Symbol) bool {
		return isTestPath(sym.Path) && containsString(sym.ModelLinks, modelID)
	}), diagnostics
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) scanCodeSymbols() ([]codemap.Symbol, []map[string]any) {
	if strings.TrimSpace(s.repoRoot) == "" {
		return nil, []map[string]any{{"code": "repo_root_missing", "message": "server not initialized with repoRoot"}}
	}
	symbols, diags, err := codemap.Scan(s.repoRoot)
	diagnostics := []map[string]any{}
	if err != nil {
		diagnostics = append(diagnostics, map[string]any{"code": "scan_failed", "message": err.Error()})
	}
	for _, d := range diags {
		diagnostics = append(diagnostics, map[string]any{"code": d.Code, "severity": string(d.Severity), "message": d.Message, "path": d.Path})
	}
	return symbols, diagnostics
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func symbolRows(symbols []codemap.Symbol, include func(codemap.Symbol) bool) []map[string]any {
	out := []map[string]any{}
	seen := map[string]bool{}
	for _, sym := range symbols {
		if !include(sym) {
			continue
		}
		key := fmt.Sprintf("%s:%d:%s", sym.Path, sym.Line, sym.Signature)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, map[string]any{
			"path":       sym.Path,
			"line":       sym.Line,
			"signature":  sym.Signature,
			"traceId":    sym.TraceID,
			"trlcLinks":  sym.Implements,
			"modelLinks": sym.ModelLinks,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i]["path"].(string) != out[j]["path"].(string) {
			return out[i]["path"].(string) < out[j]["path"].(string)
		}
		return out[i]["line"].(int) < out[j]["line"].(int)
	})
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func groupImplementationsByFile(rows []map[string]any) []map[string]any {
	type group struct {
		path      string
		lines     []int
		symbols   []map[string]any
		lineSeen  map[int]bool
		traceSeen map[string]bool
	}
	byPath := map[string]*group{}
	paths := []string{}
	for _, row := range rows {
		path, _ := row["path"].(string)
		if path == "" {
			continue
		}
		g := byPath[path]
		if g == nil {
			g = &group{path: path, lineSeen: map[int]bool{}, traceSeen: map[string]bool{}}
			byPath[path] = g
			paths = append(paths, path)
		}
		line, ok := row["line"].(int)
		if ok && line > 0 && !g.lineSeen[line] {
			g.lineSeen[line] = true
			g.lines = append(g.lines, line)
		}
		trace, _ := row["traceId"].(string)
		key := fmt.Sprintf("%s:%d", trace, line)
		if !g.traceSeen[key] {
			g.traceSeen[key] = true
			g.symbols = append(g.symbols, map[string]any{
				"line":      line,
				"signature": row["signature"],
				"traceId":   row["traceId"],
			})
		}
	}
	sort.Strings(paths)
	out := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		g := byPath[path]
		sort.Ints(g.lines)
		sort.SliceStable(g.symbols, func(i, j int) bool {
			li, _ := iLine(g.symbols[i])
			lj, _ := iLine(g.symbols[j])
			return li < lj
		})
		out = append(out, map[string]any{
			"path":     g.path,
			"lines":    g.lines,
			"lineList": joinLineNumbers(g.lines),
			"symbols":  g.symbols,
		})
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func joinLineNumbers(lines []int) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%d", line))
	}
	return strings.Join(parts, ",")
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func iLine(row map[string]any) (int, bool) {
	line, ok := row["line"].(int)
	return line, ok
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func limitRows(rows []map[string]any, max int) []map[string]any {
	if max <= 0 || len(rows) <= max {
		return rows
	}
	return rows[:max]
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func isTestPath(path string) bool {
	p := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(p)
	return strings.Contains(p, "/test/") ||
		strings.Contains(p, "/tests/") ||
		strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, ".spec.tsx")
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func maxArg(args map[string]any, fallback int) int {
	raw := firstNonEmptyArg(args, "max")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) modelEntity(id string) (string, any, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil, false
	}
	if s.requirements != nil {
		for _, x := range s.requirements.Requirements {
			if x.ID == id {
				return "requirement", x, true
			}
		}
	}
	if s.bundle == nil {
		return "", nil, false
	}
	a := s.bundle.Architecture.AuthoredArchitecture
	for _, x := range a.FunctionalGroups {
		if x.ID == id {
			return "functional_group", x, true
		}
	}
	for _, x := range a.FunctionalUnits {
		if x.ID == id {
			return "functional_unit", x, true
		}
	}
	for _, x := range a.Actors {
		if x.ID == id {
			return "actor", x, true
		}
	}
	for _, x := range a.AttackVectors {
		if x.ID == id {
			return "attack_vector", x, true
		}
	}
	for _, x := range a.ReferencedElements {
		if x.ID == id {
			return "referenced_element", x, true
		}
	}
	for _, x := range a.Interfaces {
		if x.ID == id {
			return "interface", x, true
		}
	}
	for _, x := range a.DataObjects {
		if x.ID == id {
			return "data_object", x, true
		}
	}
	for _, x := range a.DeploymentTargets {
		if x.ID == id {
			return "deployment_target", x, true
		}
	}
	for _, x := range a.Controls {
		if x.ID == id {
			return "control", x, true
		}
	}
	for _, x := range a.TrustBoundaries {
		if x.ID == id {
			return "trust_boundary", x, true
		}
	}
	for _, x := range a.States {
		if x.ID == id {
			return "state", x, true
		}
	}
	for _, x := range a.Events {
		if x.ID == id {
			return "event", x, true
		}
	}
	for _, x := range a.Flows {
		if x.ID == id {
			return "flow", x, true
		}
	}
	for _, x := range a.ThreatScenarios {
		if x.ID == id {
			return "threat_scenario", x, true
		}
	}
	for _, x := range a.ThreatAssumptions {
		if x.ID == id {
			return "threat_assumption", x, true
		}
	}
	for _, x := range a.ThreatOutOfScope {
		if x.ID == id {
			return "threat_out_of_scope", x, true
		}
	}
	for _, x := range a.ThreatMitigations {
		if x.ID == id {
			return "threat_mitigation", x, true
		}
	}
	for _, x := range a.ControlVerifications {
		if x.ID == id {
			return "control_verification", x, true
		}
	}
	for _, x := range a.Risks {
		if x.ID == id {
			return "risk", x, true
		}
	}
	for _, x := range a.POAMItems {
		if x.ID == id {
			return "poam_item", x, true
		}
	}
	return "", nil, false
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) modelEntityKind(id string) string {
	kind, _, _ := s.modelEntity(id)
	return kind
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) modelEntityNodes(query, kind string, max int) []map[string]any {
	query = strings.ToLower(strings.TrimSpace(query))
	kind = strings.ToLower(strings.TrimSpace(kind))
	out := []map[string]any{}
	add := func(entityKind, id, name string) {
		if kind != "" && entityKind != kind {
			return
		}
		if query != "" && !strings.Contains(strings.ToLower(id), query) && !strings.Contains(strings.ToLower(name), query) {
			return
		}
		out = append(out, map[string]any{"kind": entityKind, "id": id, "name": name})
	}
	if s.requirements != nil {
		for _, x := range s.requirements.Requirements {
			add("requirement", x.ID, x.Text)
		}
	}
	for _, n := range s.graphNodes("", 10000) {
		entityKind, _ := n["kind"].(string)
		id, _ := n["id"].(string)
		name, _ := n["name"].(string)
		add(entityKind, id, name)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
	if max <= 0 {
		max = 500
	}
	if len(out) > max {
		out = out[:max]
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) findFlow(id string) *model.Flow {
	id = strings.TrimSpace(id)
	if s.bundle == nil || id == "" {
		return nil
	}
	for i := range s.bundle.Architecture.AuthoredArchitecture.Flows {
		if s.bundle.Architecture.AuthoredArchitecture.Flows[i].ID == id {
			return &s.bundle.Architecture.AuthoredArchitecture.Flows[i]
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) contextForTask(args map[string]any) map[string]any {
	reqID := firstNonEmptyArg(args, "requirementId", "reqId")
	entityID := firstNonEmptyArg(args, "entityId", "id")
	path := firstNonEmptyArg(args, "path", "file")
	query := strings.ToLower(firstNonEmptyArg(args, "query", "q"))
	max := maxArg(args, 25)

	entities := []map[string]any{}
	requirements := []model.Requirement{}
	flows := []model.Flow{}
	threats := []model.ThreatScenario{}
	files := []string{}
	implementations := []map[string]any{}
	tests := []map[string]any{}
	diagnostics := []map[string]any{}

	if reqID != "" {
		if r := s.findRequirement(reqID); r != nil {
			requirements = append(requirements, *r)
		}
		flows = append(flows, s.flowsForRequirement(reqID)...)
		threats = append(threats, s.threatsForRequirement(reqID)...)
		files = append(files, s.filesForRequirement(reqID)...)
		reqTests, reqDiags := s.testsForRequirement(reqID)
		tests = append(tests, reqTests...)
		diagnostics = append(diagnostics, reqDiags...)
	}
	if entityID != "" {
		if kind, entity, ok := s.modelEntity(entityID); ok {
			entities = append(entities, map[string]any{"kind": kind, "id": entityID, "entity": entity})
			entityImpls, entityDiags := s.implementationsForModelLink(entityID)
			implementations = append(implementations, entityImpls...)
			diagnostics = append(diagnostics, entityDiags...)
			entityTests, entityTestDiags := s.testsForModelLink(entityID)
			tests = append(tests, entityTests...)
			diagnostics = append(diagnostics, entityTestDiags...)
			files = append(files, s.filesContaining(entityID)...)
		}
	}
	if query != "" {
		entities = append(entities, s.modelEntityNodes(query, "", max)...)
		for _, f := range s.filesContaining(strings.ToUpper(query)) {
			files = append(files, f)
		}
	}
	if path != "" {
		if safePath, ok := s.resolvePathInRepo(path); ok {
			files = append(files, safePath)
		}
	}

	implementations = limitRows(implementations, max)
	tests = limitRows(tests, max)
	return map[string]any{
		"query":               query,
		"requirementId":       reqID,
		"entityId":            entityID,
		"entities":            limitRows(entities, max),
		"requirements":        requirements,
		"flows":               flows,
		"threatScenarios":     threats,
		"files":               uniqueStrings(files),
		"implementations":     implementations,
		"implementationFiles": groupImplementationsByFile(implementations),
		"tests":               tests,
		"testFiles":           groupImplementationsByFile(tests),
		"diagnostics":         uniqueDiagnosticRows(diagnostics),
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) strictCoverageStatus(path string) map[string]any {
	_, diagnostics := s.scanCodeSymbols()
	path = strings.TrimSpace(path)
	relPath := ""
	if path != "" {
		safePath, ok := s.resolvePathInRepo(path)
		if !ok {
			return map[string]any{"status": "error", "diagnostics": []map[string]any{{"code": "path_outside_repo", "message": "coverage.strictStatus path must be inside repoRoot"}}}
		}
		if rel, err := filepath.Rel(s.repoRoot, safePath); err == nil {
			relPath = filepath.ToSlash(rel)
		}
	}
	filtered := []map[string]any{}
	counts := map[string]int{}
	for _, d := range diagnostics {
		diagPath, _ := d["path"].(string)
		if relPath != "" && diagnosticFilePath(diagPath) != relPath {
			continue
		}
		filtered = append(filtered, d)
		code, _ := d["code"].(string)
		if code != "" {
			counts[code]++
		}
	}
	status := "pass"
	if counts["code.missing_trlc_link"] > 0 || counts["code.invalid_trlc_link"] > 0 || counts["scan_failed"] > 0 {
		status = "fail"
	}
	return map[string]any{"status": status, "path": relPath, "summary": counts, "diagnostics": filtered}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func uniqueDiagnosticRows(in []map[string]any) []map[string]any {
	seen := map[string]bool{}
	out := []map[string]any{}
	for _, row := range in {
		key := fmt.Sprintf("%v|%v|%v", row["code"], row["path"], row["message"])
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, row)
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func diagnosticFilePath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return ""
	}
	if idx := strings.LastIndex(path, ":"); idx >= 0 {
		tail := path[idx+1:]
		if tail != "" && strings.Trim(tail, "0123456789,") == "" {
			return path[:idx]
		}
	}
	return path
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) schemaRefForInterface(interfaceID string) string {
	iface := s.findInterface(interfaceID)
	if iface == nil {
		return ""
	}
	return strings.TrimSpace(iface.SchemaRef)
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) diffSchemaRefs(left, right string) (string, string, map[string]any, map[string]any) {
	leftMeta := s.schemaFileMeta(left)
	rightMeta := s.schemaFileMeta(right)
	switch {
	case left == "" || right == "":
		return "unknown", "missing schema reference", leftMeta, rightMeta
	case leftMeta["exists"] == true && rightMeta["exists"] == true:
		if leftMeta["sha256"] == rightMeta["sha256"] {
			return "same", "schema file contents match", leftMeta, rightMeta
		}
		return "changed", "schema file contents differ", leftMeta, rightMeta
	case left == right:
		return "same", "schema references match; schema files were not found", leftMeta, rightMeta
	default:
		return "changed", "schema references differ; schema files were not found", leftMeta, rightMeta
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE, CTRL-MCP-PATH-BOUNDARY
func (s *Server) schemaFileMeta(ref string) map[string]any {
	ref = strings.TrimSpace(ref)
	out := map[string]any{"ref": ref, "exists": false}
	if ref == "" {
		return out
	}
	candidates := []string{}
	if s.modelPath != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(s.modelPath), ref))
	}
	if s.repoRoot != "" {
		candidates = append(candidates, filepath.Join(s.repoRoot, ref))
	}
	for _, cand := range candidates {
		clean, ok := s.resolvePathInRepo(cand)
		if !ok {
			continue
		}
		b, err := os.ReadFile(clean)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(b)
		out["exists"] = true
		out["path"] = clean
		out["bytes"] = len(b)
		out["sha256"] = fmt.Sprintf("%x", sum[:])
		return out
	}
	return out
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) resolvePathInRepo(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" || strings.TrimSpace(s.repoRoot) == "" {
		return "", false
	}
	root, err := filepath.Abs(s.repoRoot)
	if err != nil {
		return "", false
	}
	target := path
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	cleanTarget, err := filepath.Abs(target)
	if err != nil {
		return "", false
	}
	cleanTarget = filepath.Clean(cleanTarget)
	root = filepath.Clean(root)
	if !strings.HasPrefix(cleanTarget, root+string(os.PathSeparator)) && cleanTarget != root {
		return "", false
	}
	return cleanTarget, true
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
func endpointForEnv(i *model.Interface, env string) string {
	if i == nil {
		return ""
	}
	if env == "" {
		return i.Endpoint
	}
	return fmt.Sprintf("%s [%s]", i.Endpoint, env)
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) authHintsForInterface(interfaceID string) []string {
	if s.bundle == nil {
		return nil
	}
	hints := []string{}
	for _, f := range s.bundle.Architecture.AuthoredArchitecture.Flows {
		for _, st := range f.Steps {
			if st.InterfaceRef == interfaceID {
				if st.Authentication != "" {
					hints = append(hints, st.Authentication)
				}
			}
		}
	}
	sort.Strings(hints)
	return uniqueStrings(hints)
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (s *Server) ownerFor(id string) string {
	if s.bundle == nil || strings.TrimSpace(id) == "" {
		return ""
	}
	a := s.bundle.Architecture.AuthoredArchitecture
	for _, i := range a.Interfaces {
		if i.ID == id {
			return i.Owner
		}
	}
	for _, r := range a.Risks {
		if r.ID == id {
			return r.Owner
		}
	}
	for _, t := range a.ThreatScenarios {
		if t.ID == id {
			return t.Owner
		}
	}
	for _, c := range a.ControlVerifications {
		if c.ID == id {
			return c.Owner
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) graphNodes(query string, max int) []map[string]any {
	if s.bundle == nil {
		return nil
	}
	a := s.bundle.Architecture.AuthoredArchitecture
	out := []map[string]any{}
	add := func(kind, id, name string) {
		if query != "" {
			q := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(id), q) && !strings.Contains(strings.ToLower(name), q) {
				return
			}
		}
		out = append(out, map[string]any{"kind": kind, "id": id, "name": name})
	}
	for _, x := range a.FunctionalGroups {
		add("functional_group", x.ID, x.Name)
	}
	for _, x := range a.FunctionalUnits {
		add("functional_unit", x.ID, x.Name)
	}
	for _, x := range a.Actors {
		add("actor", x.ID, x.Name)
	}
	for _, x := range a.AttackVectors {
		add("attack_vector", x.ID, x.Name)
	}
	for _, x := range a.ReferencedElements {
		add("referenced_element", x.ID, x.Name)
	}
	for _, x := range a.Interfaces {
		add("interface", x.ID, x.Name)
	}
	for _, x := range a.DataObjects {
		add("data_object", x.ID, x.Name)
	}
	for _, x := range a.DeploymentTargets {
		add("deployment_target", x.ID, x.Name)
	}
	for _, x := range a.Controls {
		add("control", x.ID, x.Name)
	}
	for _, x := range a.TrustBoundaries {
		add("trust_boundary", x.ID, x.Name)
	}
	for _, x := range a.States {
		add("state", x.ID, x.Name)
	}
	for _, x := range a.Events {
		add("event", x.ID, x.Name)
	}
	for _, x := range a.Flows {
		add("flow", x.ID, x.Title)
	}
	for _, x := range a.ThreatScenarios {
		add("threat_scenario", x.ID, x.Title)
	}
	for _, x := range a.ThreatAssumptions {
		add("threat_assumption", x.ID, x.Title)
	}
	for _, x := range a.ThreatOutOfScope {
		add("threat_out_of_scope", x.ID, x.Title)
	}
	for _, x := range a.ThreatMitigations {
		add("threat_mitigation", x.ID, strings.TrimSpace(x.ThreatScenarioRef+" "+x.ControlRef))
	}
	for _, x := range a.ControlVerifications {
		add("control_verification", x.ID, strings.TrimSpace(x.ControlRef+" "+x.Method))
	}
	for _, x := range a.Risks {
		add("risk", x.ID, x.Title)
	}
	for _, x := range a.POAMItems {
		add("poam_item", x.ID, x.RiskRef)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
	if len(out) > max {
		out = out[:max]
	}
	return out
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) graphEdges(query string, max int) []map[string]any {
	if s.bundle == nil {
		return nil
	}
	out := []map[string]any{}
	for _, m := range s.bundle.Architecture.AuthoredArchitecture.Mappings {
		if query != "" {
			q := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(m.From), q) && !strings.Contains(strings.ToLower(m.To), q) && !strings.Contains(strings.ToLower(m.Type), q) {
				continue
			}
		}
		out = append(out, map[string]any{"type": m.Type, "from": m.From, "to": m.To, "description": m.Description})
	}
	if len(out) > max {
		out = out[:max]
	}
	return out
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func firstNonEmptyArg(args map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := args[k]; ok {
			s := strings.TrimSpace(toString(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func nonEmptyString(v any, fallback string) string {
	s := strings.TrimSpace(toString(v))
	if s == "" {
		return fallback
	}
	return s
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func toString(v any) string {
	s, _ := v.(string)
	return s
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range in {
		x = strings.TrimSpace(x)
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	return out
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func containsString(in []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, x := range in {
		if strings.TrimSpace(x) == target {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func inputSchemaForTool(name string) map[string]any {
	allowed := toolArgsAllowlist[name]
	props := map[string]any{}
	for _, k := range allowed {
		props[k] = map[string]any{"type": "string"}
	}
	return map[string]any{"type": "object", "properties": props, "additionalProperties": false}
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func validateToolArguments(name string, args map[string]any) error {
	allowedKeys := map[string]bool{}
	for _, k := range toolArgsAllowlist[name] {
		allowedKeys[k] = true
	}
	for k := range args {
		if !allowedKeys[k] {
			allowed := append([]string{}, toolArgsAllowlist[name]...)
			sort.Strings(allowed)
			if len(allowed) == 0 {
				return fmt.Errorf("%s accepts no arguments; got %q", name, k)
			}
			return fmt.Errorf("unexpected argument %q for %s (allowed: %s)", k, name, strings.Join(allowed, ", "))
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func requireStableID(id string, allowedPrefixes ...string) error {
	id = strings.TrimSpace(id)
	if err := requireWellFormedStableID(id); err != nil {
		return err
	}
	for _, p := range allowedPrefixes {
		if strings.HasPrefix(id, p) {
			return nil
		}
	}
	return fmt.Errorf("ID %s must start with one of: %s", id, strings.Join(allowedPrefixes, ", "))
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func requireWellFormedStableID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("missing stable ID")
	}
	if !stableIDPattern.MatchString(id) {
		return fmt.Errorf("invalid stable ID format: %s", id)
	}
	return nil
}
