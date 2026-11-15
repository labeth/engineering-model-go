// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// rpcError is the code and message of a JSON-RPC error response.
type rpcError struct {
	code    int
	message string
}

// rpcErrorResponse is the JSON-RPC error response for the request id.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func rpcErrorResponse(id any, e rpcError) ([]byte, error) {
	return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": e.code, "message": e.message}})
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) Handle(raw []byte) (resp []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			resp, err = rpcErrorResponse(nil, rpcError{-32603, "internal server error"})
		}
	}()
	var req map[string]any
	if err := json.Unmarshal(raw, &req); err != nil {
		return rpcErrorResponse(nil, rpcError{-32700, "parse error"})
	}
	return s.handleRequest(req)
}

// handleRequest validates a decoded JSON-RPC request and answers it; a
// notification gets no response.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) handleRequest(req map[string]any) ([]byte, error) {
	if version, _ := req["jsonrpc"].(string); version != "2.0" {
		return rpcErrorResponse(req["id"], rpcError{-32600, "invalid request: jsonrpc must be 2.0"})
	}
	method, _ := req["method"].(string)
	id, hasID := req["id"]
	if strings.TrimSpace(method) == "" {
		return rpcErrorResponse(id, rpcError{-32600, "invalid request: missing method"})
	}
	if method == "notifications/initialized" || !hasID {
		return nil, nil
	}
	return s.respond(id, method, req["params"])
}

// respond dispatches the method and wraps its result or error as a JSON-RPC
// response, with machine paths scrubbed from an error message.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) respond(id any, method string, params any) ([]byte, error) {
	result, code, rpcErr := s.dispatch(method, params)
	if rpcErr == nil {
		return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	}
	if code == 0 {
		code = -32602
	}
	message := newPathScrubber(s.repoRoot, s.modelPath).scrubString(rpcErr.Error())
	return rpcErrorResponse(id, rpcError{code, message})
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
// ENGMODEL-LINKS: TR-MCP-RELOAD-FAILS
func (s *Server) dispatch(method string, params any) (any, int, error) {
	switch method {
	case "initialize":
		return s.dispatchInitialize(params)
	case "ping":
		return map[string]any{}, 0, nil
	case "tools/list":
		return s.toolsList(), 0, nil
	case "tools/call":
		return s.dispatchToolsCall(params)
	default:
		return nil, -32601, fmt.Errorf("unsupported method: %s", method)
	}
}

// dispatchInitialize loads the model context of an initialize request.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
// ENGMODEL-LINKS: TR-MCP-RELOAD-FAILS
func (s *Server) dispatchInitialize(params any) (any, int, error) {
	if _, ok := params.(map[string]any); params != nil && !ok {
		return nil, -32602, fmt.Errorf("initialize params must be an object")
	}
	if err := s.loadContext(params); err != nil {
		return nil, -32602, err
	}
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":      map[string]any{"name": "engineering-model-mcp", "version": "0.2.0"},
	}, 0, nil
}

// toolsList is the tools/list result.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) toolsList() map[string]any {
	tools := make([]map[string]any, 0, len(s.tools))
	for _, name := range s.ToolNames() {
		t := s.tools[name]
		tools = append(tools, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": inputSchemaForTool(t.Name)})
	}
	return map[string]any{"tools": tools}
}

// toolsCallRequest is a validated tools/call request.
type toolsCallRequest struct {
	name string
	args map[string]any
}

// parseToolsCall validates the params of a tools/call request: the tool is
// known, the server is initialized and the arguments are an accepted object.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) parseToolsCall(params any) (toolsCallRequest, int, error) {
	p, ok := params.(map[string]any)
	if !ok {
		return toolsCallRequest{}, -32602, fmt.Errorf("tools/call params must be an object")
	}
	name := strings.TrimSpace(toString(p["name"]))
	if name == "" {
		return toolsCallRequest{}, -32602, fmt.Errorf("tools/call name is required")
	}
	if _, ok := s.tools[name]; !ok {
		return toolsCallRequest{}, -32602, fmt.Errorf("unknown tool: %s", name)
	}
	if s.bundle == nil {
		return toolsCallRequest{}, -32001, fmt.Errorf("server not initialized with modelPath")
	}
	args, err := toolCallArguments(p)
	if err != nil {
		return toolsCallRequest{}, -32602, err
	}
	if err := validateToolArguments(name, args); err != nil {
		return toolsCallRequest{}, -32602, err
	}
	return toolsCallRequest{name: name, args: args}, 0, nil
}

// toolCallArguments is the arguments object of a tools/call request; absent
// or null arguments are empty.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func toolCallArguments(p map[string]any) (map[string]any, error) {
	rawArgs, exists := p["arguments"]
	if !exists || rawArgs == nil {
		return map[string]any{}, nil
	}
	typedArgs, ok := rawArgs.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tools/call arguments must be an object")
	}
	return typedArgs, nil
}

// targetServer is the server that answers the call: s, or the family member
// a repo argument names (removed from the arguments).
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) targetServer(req toolsCallRequest) (*Server, error) {
	repo, given := req.args["repo"]
	if !given {
		return s, nil
	}
	delete(req.args, "repo")
	return s.memberServer(toString(repo))
}

// dispatchToolsCall runs a tools/call request and wraps the payload as a
// tool result, scrubbed of machine paths.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
// ENGMODEL-LINKS: TR-MCP-RELOAD-FAILS
func (s *Server) dispatchToolsCall(params any) (any, int, error) {
	req, code, err := s.parseToolsCall(params)
	if err != nil {
		return nil, code, err
	}
	target, err := s.targetServer(req)
	if err != nil {
		return s.toolErrorResult(req.name, err), 0, nil
	}
	if err := target.refreshModel(); err != nil {
		return nil, -32001, fmt.Errorf("reload changed model: %w", err)
	}
	return target.runTool(req), 0, nil
}

// runTool calls the tool and returns its tool result: the scrubbed payload,
// or the error result when the tool failed.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) runTool(req toolsCallRequest) map[string]any {
	payload, err := s.callTool(req.name, req.args)
	if err != nil {
		return s.toolErrorResult(req.name, err)
	}
	// Round-trip through JSON so every string, typed structs included, is
	// scrubbed of absolute machine paths before it leaves the server.
	buf, _ := json.Marshal(payload)
	var generic any
	if json.Unmarshal(buf, &generic) == nil {
		buf, _ = json.Marshal(newPathScrubber(s.repoRoot, s.modelPath).scrub(generic))
	}
	return map[string]any{"content": []map[string]any{{"type": "text", "text": string(buf)}}, "isError": false}
}
