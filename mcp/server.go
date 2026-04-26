package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/labeth/engineering-model-go/model"
)

type Tool struct {
	Name        string
	Description string
}

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
)

var errRepoIndexLimit = errors.New("repo index file limit reached")

type indexedFile struct {
	Path    string
	Content string
}

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
		{Name: "views.recommend", Description: "Recommend views for task"},
		{Name: "views.renderContext", Description: "Render compact view context"},
		{Name: "generation.plan", Description: "Plan artifact regeneration"},
		{Name: "generation.status", Description: "Get artifact freshness status"},
		{Name: "governance.policy", Description: "Get governance policy"},
		{Name: "governance.checkPatch", Description: "Check patch against governance"},
		{Name: "tasks.entryPoints", Description: "List task entry points"},
		{Name: "tasks.nextBestActions", Description: "Suggest next best actions"},
		{Name: "interfaces.resolve", Description: "Resolve interface endpoint target"},
		{Name: "interfaces.matchFromCode", Description: "Match interface from code path"},
		{Name: "interfaces.ambiguities", Description: "List interface ambiguities"},
		{Name: "environments.resolve", Description: "Resolve environment metadata"},
		{Name: "endpoints.resolve", Description: "Resolve endpoint per environment"},
		{Name: "identity.resolve", Description: "Resolve identity/auth requirements"},
		{Name: "policy.resolve", Description: "Resolve policy for interface"},
		{Name: "schema.resolve", Description: "Resolve schema for interface"},
		{Name: "schema.diff", Description: "Diff schema versions"},
		{Name: "ownership.resolve", Description: "Resolve ownership metadata"},
		{Name: "runtime.resolve", Description: "Resolve runtime target"},
		{Name: "confidence.explain", Description: "Explain confidence for entity"},
		{Name: "staleness.check", Description: "Check evidence staleness"},
		{Name: "changes.preflight", Description: "Preflight change impact"},
	}
	m := map[string]Tool{}
	for _, t := range all {
		m[t.Name] = t
	}
	return &Server{tools: m}
}

func (s *Server) ToolNames() []string {
	names := make([]string, 0, len(s.tools))
	for n := range s.tools {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

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
			tools = append(tools, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": map[string]any{"type": "object", "additionalProperties": true}})
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
		payload, err := s.callTool(name, args)
		if err != nil {
			return map[string]any{"content": []map[string]any{{"type": "text", "text": err.Error()}}, "isError": true}, 0, nil
		}
		buf, _ := json.Marshal(payload)
		return map[string]any{"content": []map[string]any{{"type": "text", "text": string(buf)}}, "isError": false}, 0, nil
	default:
		return nil, -32601, fmt.Errorf("unsupported method: %s", method)
	}
}

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
		data["generatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return data, nil
	}

	switch name {
	case "requirements.get":
		if reqID == "" {
			return nil, fmt.Errorf("requirements.get requires requirementId")
		}
		r := s.findRequirement(reqID)
		return simple(map[string]any{"requirement": r, "exists": r != nil})
	case "requirements.impact", "requirements.supportPath", "requirements.suggestEditPlan":
		if reqID == "" {
			return nil, fmt.Errorf("%s requires requirementId", name)
		}
		r := s.findRequirement(reqID)
		flows := s.flowsForRequirement(reqID)
		threats := s.threatsForRequirement(reqID)
		files := s.filesForRequirement(reqID)
		return simple(map[string]any{"requirement": r, "impactedFlows": flows, "impactedThreats": threats, "relevantFiles": files})
	case "files.forRequirement":
		if reqID == "" {
			return nil, fmt.Errorf("files.forRequirement requires requirementId")
		}
		return simple(map[string]any{"files": s.filesForRequirement(reqID)})
	case "files.forControl":
		controlID := strings.TrimSpace(firstNonEmptyArg(args, "controlId", "id"))
		if controlID == "" {
			return nil, fmt.Errorf("files.forControl requires controlId")
		}
		return simple(map[string]any{"files": s.filesContaining(controlID)})
	case "files.forThreat":
		threatID := strings.TrimSpace(firstNonEmptyArg(args, "threatId", "id"))
		if threatID == "" {
			return nil, fmt.Errorf("files.forThreat requires threatId")
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
		return simple(map[string]any{"recommendations": []string{"add TRLC-LINKS markers in tests for uncovered REQ IDs", "add control verification evidence for partial controls"}})
	case "threats.forRequirement":
		if reqID == "" {
			return nil, fmt.Errorf("threats.forRequirement requires requirementId")
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
		return simple(map[string]any{"flows": s.flowsForRequirement(reqID)})
	case "flows.diff":
		left := firstNonEmptyArg(args, "fromFlowId", "left")
		right := firstNonEmptyArg(args, "toFlowId", "right")
		if left == "" || right == "" {
			return nil, fmt.Errorf("flows.diff requires fromFlowId and toFlowId")
		}
		return simple(map[string]any{"from": left, "to": right, "note": "flow diff not yet semantic; compare ids and metadata"})
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
	case "views.recommend":
		taskType := strings.ToLower(firstNonEmptyArg(args, "taskType", "task"))
		recommended := []string{"VIEW-TRACE"}
		if strings.Contains(taskType, "security") {
			recommended = []string{"VIEW-SEC"}
		} else if strings.Contains(taskType, "deploy") {
			recommended = []string{"VIEW-DEPLOY"}
		}
		return simple(map[string]any{"recommendedViews": recommended})
	case "views.renderContext":
		viewID := firstNonEmptyArg(args, "viewId", "id")
		if viewID == "" {
			return nil, fmt.Errorf("views.renderContext requires viewId")
		}
		for _, v := range s.bundle.Architecture.Views {
			if v.ID == viewID {
				return simple(map[string]any{"view": v, "rootCount": len(v.Roots)})
			}
		}
		return simple(map[string]any{"view": nil})
	case "generation.plan":
		return simple(map[string]any{"commands": []string{"go run ./cmd/engdoc ...", "go run ./cmd/engdragon ...", "go run ./cmd/engstruct ...", "go run ./cmd/engtrlc ...", "go run ./cmd/englobster ..."}})
	case "generation.status":
		return simple(map[string]any{"modelPath": s.modelPath, "requirementsPath": s.requirementsPath, "designPath": s.designPath})
	case "governance.policy":
		return simple(map[string]any{"requiredMarkers": []string{"TRLC-LINKS"}, "rules": []string{"include requirement links in tests", "prefer stable IDs", "regenerate affected artifacts"}})
	case "governance.checkPatch":
		diff := firstNonEmptyArg(args, "diff")
		issues := []string{}
		if strings.Contains(diff, "tests/") && !strings.Contains(diff, "TRLC-LINKS") {
			issues = append(issues, "test changes without TRLC-LINKS markers")
		}
		return simple(map[string]any{"issues": issues})
	case "tasks.entryPoints":
		ids := s.requirementIDs()
		return simple(map[string]any{"entryPoints": []map[string]any{{"id": "EP-REQ", "requirements": ids}, {"id": "EP-VERIFICATION-GAPS", "count": len(ids)}}})
	case "tasks.nextBestActions":
		return simple(map[string]any{"actions": []string{"Update requirement text", "Adjust mapped flow", "Update tests with TRLC-LINKS", "Regenerate docs/exports"}})
	case "interfaces.resolve", "endpoints.resolve":
		if interfaceID == "" {
			return nil, fmt.Errorf("%s requires interfaceId", name)
		}
		iface := s.findInterface(interfaceID)
		if iface == nil {
			return nil, fmt.Errorf("%s interface not found: %s", name, interfaceID)
		}
		return simple(map[string]any{"interface": iface, "environment": env, "resolvedEndpoint": endpointForEnv(iface, env)})
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
		iface := s.findInterface(interfaceID)
		if iface == nil {
			return nil, fmt.Errorf("identity.resolve interface not found: %s", interfaceID)
		}
		return simple(map[string]any{"interface": iface, "authHints": s.authHintsForInterface(interfaceID)})
	case "policy.resolve":
		if interfaceID == "" && firstNonEmptyArg(args, "id") == "" {
			return nil, fmt.Errorf("policy.resolve requires interfaceId or id")
		}
		controls := []string{}
		for _, m := range a.Mappings {
			if m.Type == "guarded_by" && (m.From == interfaceID || m.From == firstNonEmptyArg(args, "id")) {
				controls = append(controls, m.To)
			}
		}
		return simple(map[string]any{"controls": controls})
	case "schema.resolve":
		if interfaceID == "" {
			return nil, fmt.Errorf("schema.resolve requires interfaceId")
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
		status := "same"
		note := "schema references match"
		switch {
		case left == "" || right == "":
			status = "unknown"
			note = "missing schema reference"
		case left != right:
			status = "changed"
			note = "schema references differ"
		}
		return simple(map[string]any{"fromSchemaRef": left, "toSchemaRef": right, "status": status, "message": note})
	case "ownership.resolve":
		if entityID == "" {
			return nil, fmt.Errorf("ownership.resolve requires entityId")
		}
		return simple(map[string]any{"id": entityID, "owner": s.ownerFor(entityID)})
	case "runtime.resolve":
		targets := []string{}
		for _, m := range a.Mappings {
			if m.Type == "deployed_to" && (entityID == "" || m.From == entityID) {
				targets = append(targets, m.To)
			}
		}
		sort.Strings(targets)
		return simple(map[string]any{"runtimeTargets": targets})
	case "confidence.explain":
		if entityID == "" {
			return nil, fmt.Errorf("confidence.explain requires entityId")
		}
		return simple(map[string]any{"id": entityID, "confidence": "high", "basis": "authored model links"})
	case "staleness.check":
		path := firstNonEmptyArg(args, "path")
		if path == "" {
			path = s.modelPath
		}
		st := "unknown"
		if info, err := os.Stat(path); err == nil {
			age := time.Since(info.ModTime())
			if age < 24*time.Hour {
				st = "fresh"
			} else if age < 7*24*time.Hour {
				st = "stale-soon"
			} else {
				st = "stale"
			}
		}
		return simple(map[string]any{"path": path, "staleness": st})
	case "changes.preflight":
		r := firstNonEmptyArg(args, "requirementId", "id")
		if r == "" {
			return nil, fmt.Errorf("changes.preflight requires requirementId")
		}
		return simple(map[string]any{"requirementId": r, "impactedFiles": s.filesForRequirement(r), "rerun": []string{"go test ./...", "engdoc", "engdragon", "engstruct", "engtrlc"}})
	default:
		return simple(map[string]any{"message": "tool not implemented"})
	}
}

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

func (s *Server) filesForRequirement(reqID string) []string {
	if strings.TrimSpace(reqID) == "" {
		return nil
	}
	return s.filesContaining(reqID)
}

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

func (s *Server) schemaRefForInterface(interfaceID string) string {
	iface := s.findInterface(interfaceID)
	if iface == nil {
		return ""
	}
	return strings.TrimSpace(iface.SchemaRef)
}

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

func endpointForEnv(i *model.Interface, env string) string {
	if i == nil {
		return ""
	}
	if env == "" {
		return i.Endpoint
	}
	return fmt.Sprintf("%s [%s]", i.Endpoint, env)
}

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
	for _, x := range a.Interfaces {
		add("interface", x.ID, x.Name)
	}
	for _, x := range a.DataObjects {
		add("data_object", x.ID, x.Name)
	}
	for _, x := range a.Controls {
		add("control", x.ID, x.Name)
	}
	for _, x := range a.ThreatScenarios {
		add("threat_scenario", x.ID, x.Title)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
	if len(out) > max {
		out = out[:max]
	}
	return out
}

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

func nonEmptyString(v any, fallback string) string {
	s := strings.TrimSpace(toString(v))
	if s == "" {
		return fallback
	}
	return s
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}

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
