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

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
)

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
type Tool struct {
	Name        string
	Description string
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// ENGMODEL-LINKS: STATE-MCP-MODEL-CURRENT, STATE-MCP-MODEL-OUTDATED
type Server struct {
	tools map[string]Tool

	bundle           *model.Bundle
	canonical        *model.CanonicalBundle
	requirements     *model.RequirementsDocument
	design           *model.DesignDocument
	modelPath        string
	requirementsPath string
	designPath       string
	repoRoot         string

	indexOnce sync.Once
	indexErr  error
	repoFiles []indexedFile

	// fingerprint is the content hash of the model files at the last load
	// (modelFileFingerprint); a tools/call reloads when it changes.
	fingerprint string

	// clock is the time source of links.review (reviewedOn); nil is
	// time.Now. Tests set it.
	clock func() time.Time

	// members caches one server per family member a repo argument named,
	// by model directory; each reloads its own model per call.
	members map[string]*Server
}

// now returns the server's current time.
// TRLC-LINKS: REQ-EMG-106
func (s *Server) now() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
}

const (
	maxIndexedFileBytes = 512 * 1024
	// maxFingerprintFileBytes caps how much of one model file the reload
	// fingerprint reads; larger files are compared by size and mtime.
	maxFingerprintFileBytes = 16 * 1024 * 1024
	maxIndexedFiles         = 5000
	toolSchemaVersion       = "mcp.tool-response.v1"
)

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
var errRepoIndexLimit = errors.New("repo index file limit reached")

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
var stableIDPattern = regexp.MustCompile(`^[A-Z][A-Z0-9-]*$`)

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
var toolArgsAllowlist = map[string][]string{
	"requirements.get":             {"requirementId"},
	"requirements.parents":         {"requirementId"},
	"links.list":                   {},
	"links.suspect":                {},
	"links.review":                 {"linkId", "by", "rationale"},
	"requirements.impact":          {"requirementId"},
	"requirements.supportPath":     {"requirementId"},
	"requirements.suggestEditPlan": {"requirementId"},
	"files.forRequirement":         {"requirementId"},
	"files.forControl":             {"controlId"},
	"files.forThreat":              {"threatId"},
	"files.owner":                  {"path"},
	"verification.status":          {"entityId"},
	"verification.gaps":            {},
	"verification.recommend":       {},
	"threats.forRequirement":       {"requirementId"},
	"threats.coverage":             {},
	"threats.unmitigated":          {},
	"flows.forRequirement":         {"requirementId"},
	"flows.diff":                   {"fromFlowId", "toFlowId"},
	"graph.neighborhood":           {"query"},
	"graph.explainEdge":            {"from", "to"},
	"graph.search":                 {"query"},
	"model.list":                   {"query", "kind", "max"},
	"model.authoringContract":      {},
	"model.complete":               {"prefix", "field", "max"},
	"model.diagnose":               {"document", "content"},
	"model.patch":                  {"document", "id", "set", "unset", "append", "remove", "write"},
	"model.entity":                 {"entityId"},
	"model.implementations":        {"entityId", "groupBy", "max"},
	"code.contextForTask":          {"query", "requirementId", "entityId", "path", "max"},
	"views.recommend":              {"taskType"},
	"views.renderContext":          {"viewId"},
	"generation.plan":              {},
	"generation.status":            {},
	"composition.resolve":          {},
	"trace.matrix":                 {"summaryOnly"},
	"tier.gap":                     {"target"},
	"governance.policy":            {},
	"governance.checkPatch":        {"diff"},
	"tasks.entryPoints":            {},
	"tasks.nextBestActions":        {},
	"interfaces.resolve":           {"interfaceId", "environment"},
	"interfaces.implementations":   {"interfaceId", "groupBy", "max"},
	"interfaces.matchFromCode":     {"path"},
	"interfaces.ambiguities":       {},
	"environments.resolve":         {"environment"},
	"endpoints.resolve":            {"interfaceId", "environment"},
	"identity.resolve":             {"interfaceId"},
	"policy.resolve":               {"interfaceId"},
	"schema.resolve":               {"interfaceId"},
	"schema.diff":                  {"fromSchemaRef", "toSchemaRef", "fromInterfaceId", "toInterfaceId"},
	"flow.detail":                  {"flowId"},
	"flow.implementations":         {"flowId", "groupBy", "max"},
	"tests.forRequirement":         {"requirementId", "max"},
	"tests.forEntity":              {"entityId", "max"},
	"coverage.strictStatus":        {"path"},
	"ownership.resolve":            {"entityId"},
	"runtime.resolve":              {"entityId"},
	"confidence.explain":           {"entityId"},
	"staleness.check":              {"path"},
	"changes.preflight":            {"requirementId"},
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

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008, REQ-EMG-045, REQ-EMG-069, REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func NewServer() *Server {
	m := make(map[string]Tool, len(serverTools))
	for _, t := range serverTools {
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

// toolErrorResult is the isError tool result for err, its message scrubbed
// of machine paths.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY
func (s *Server) toolErrorResult(name string, err error) map[string]any {
	scrubber := newPathScrubber(s.repoRoot, s.modelPath)
	errPayload, _ := json.Marshal(map[string]any{
		"ok":            false,
		"tool":          name,
		"schemaVersion": toolSchemaVersion,
		"error": map[string]any{
			"code":    "INVALID_ARGUMENT",
			"message": scrubber.scrubString(err.Error()),
			"hint":    "see tools/list.inputSchema for accepted arguments",
		},
		"generatedAt": time.Now().UTC().Format(time.RFC3339),
	})
	return map[string]any{"content": []map[string]any{{"type": "text", "text": string(errPayload)}}, "isError": true}
}

// refreshModel reloads the model when the files it was loaded from changed
// since the last load, so every tool call renders the files on disk at call
// time without the client re-sending initialize. A failed reload returns its
// error and keeps the fingerprint, so the next call retries instead of
// serving the previous model.
// TRLC-LINKS: REQ-EMG-089
// ENGMODEL-LINKS: FU-MCP-SERVER, EVT-MCP-TOOL-CALL-RECEIVED
// ENGMODEL-LINKS: TR-MCP-CALL-RELOADS, TR-MCP-CALL-SERVED
func (s *Server) refreshModel() error {
	if s.modelPath == "" && s.requirementsPath == "" && s.designPath == "" {
		return nil
	}
	if s.modelFileFingerprint() == s.fingerprint {
		return nil
	}
	return s.loadContext(nil)
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
func mapKeys[V any](in map[string]V) []string {
	out := make([]string, 0, len(in))
	for k := range in {
		out = append(out, k)
	}
	sort.Strings(out)
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
		{"id": "engtrlc", "command": fmt.Sprintf("go run ./cmd/engtrlc --model %s --requirements %s --out-dir %s", shellPath(modelPath), shellPath(requirementsPath), shellPath(filepath.Join(generatedDir, "trlc"))), "outputs": []string{filepath.Join(generatedDir, "trlc", "model.rsl"), filepath.Join(generatedDir, "trlc", "requirements.trlc")}},
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
	// The check's code scope: git-ignored files, inferenceHints.exclude and
	// nested modules are left out (codescan.ScanCodeScope, REQ-EMG-114).
	symbols, diags, err := codescan.ScanCodeScope(*s.bundle, s.repoRoot)
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

// boolArg reads a flag argument sent as a JSON boolean or as the string
// "true"/"false" (case-insensitive); anything else is false.
// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
func boolArg(args map[string]any, key string) bool {
	switch v := args[key].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
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

// repoTools are the model-reading tools that take the optional repo
// argument (a family member checkout inside the workspace root): every tool
// whose answer is the model of one checkout. The file and code tools stay
// on the server's own checkout (their paths are relative to it).
var repoTools = map[string]bool{
	"model.diagnose": true, "model.patch": true, "model.list": true, "model.entity": true, "model.authoringContract": true,
	"model.complete": true, "trace.matrix": true, "tier.gap": true, "composition.resolve": true,
	"links.list": true, "links.suspect": true, "links.review": true,
	"requirements.get": true, "requirements.parents": true, "requirements.impact": true, "requirements.supportPath": true,
	"requirements.suggestEditPlan": true, "threats.coverage": true, "threats.unmitigated": true, "threats.forRequirement": true,
	"verification.status": true, "verification.gaps": true, "verification.recommend": true, "flow.detail": true,
	"flows.forRequirement": true, "graph.neighborhood": true, "graph.search": true, "graph.explainEdge": true,
	"generation.status": true, "generation.plan": true,
	"gemara.controlCatalog": true, "gemara.threatCatalog": true, "gemara.riskCatalog": true, "gemara.vectorCatalog": true,
	"gemara.capabilityCatalog": true, "gemara.principleCatalog": true, "gemara.guidanceCatalog": true, "gemara.policy": true,
	"gemara.lexicon": true, "gemara.mappingDocument": true, "gemara.auditLog": true, "gemara.enforcementLog": true,
	"gemara.evaluationLog": true, "gemara.validate": true,
}

// repoArgDescription describes the repo argument in the input schemas.
const repoArgDescription = "optional: answer for this checkout of the model family instead of the server's own: a directory with engmod.yml inside the workspace root (the directory of the trusted engmod.work.yml above the server's model, looked up on every call), relative to that root or absolute; default the server's own checkout"

// booleanToolArgs lists the arguments that are flags; their schema accepts a
// JSON boolean or the strings "true"/"false".
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
var booleanToolArgs = map[string]bool{"summaryOnly": true}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func inputSchemaForTool(name string) map[string]any {
	if name == "model.patch" {
		schema := patchInputSchema()
		schema["properties"].(map[string]any)["repo"] = map[string]any{"type": "string", "description": repoArgDescription}
		return schema
	}
	allowed := toolArgsAllowlist[name]
	props := map[string]any{}
	for _, k := range allowed {
		if booleanToolArgs[k] {
			props[k] = map[string]any{"type": []string{"boolean", "string"}, "enum": []any{true, false, "true", "false"}}
			continue
		}
		props[k] = map[string]any{"type": "string"}
	}
	if repoTools[name] {
		props["repo"] = map[string]any{"type": "string", "description": repoArgDescription}
	}
	return map[string]any{"type": "object", "properties": props, "additionalProperties": false}
}

// formerToolArgs maps an argument name that earlier versions accepted as an
// alias to the canonical names it stood for; the canonical name of a tool is
// the first one its allowlist holds. A former name is rejected like any unknown
// argument and the message names the canonical one (ADR-EMG-073).
var formerToolArgs = map[string][]string{
	"id":    {"requirementId", "entityId", "interfaceId", "linkId", "controlId", "threatId", "viewId", "flowId", "query"},
	"reqId": {"requirementId"}, "name": {"interfaceId", "query"}, "q": {"query", "prefix"},
	"query": {"prefix"}, "file": {"path"}, "env": {"environment"}, "path": {"document"},
	"left": {"fromFlowId"}, "right": {"toFlowId"}, "task": {"taskType"},
	"leftSchemaRef": {"fromSchemaRef"}, "rightSchemaRef": {"toSchemaRef"},
	"leftInterfaceId": {"fromInterfaceId"}, "rightInterfaceId": {"toInterfaceId"},
}

// canonicalArgFor returns the canonical argument of tool that a former alias
// stood for, or "" when the name was never an alias for the tool.
// TRLC-LINKS: REQ-EMG-182
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-STRICT-MCP-INPUT-SCHEMA
func canonicalArgFor(tool, former string) string {
	for _, candidate := range formerToolArgs[former] {
		if containsString(toolArgsAllowlist[tool], candidate) {
			return candidate
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func validateToolArguments(name string, args map[string]any) error {
	allowedKeys := map[string]bool{}
	for _, k := range toolArgsAllowlist[name] {
		allowedKeys[k] = true
	}
	if repoTools[name] {
		allowedKeys["repo"] = true
	}
	for k := range args {
		if !allowedKeys[k] {
			return unexpectedArgumentError(name, k)
		}
	}
	return nil
}

// unexpectedArgumentError reports an argument the tool does not accept: the
// message lists the allowed arguments and, for a former alias, names the
// canonical argument.
// TRLC-LINKS: REQ-EMG-008, REQ-EMG-182
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-STRICT-MCP-INPUT-SCHEMA
func unexpectedArgumentError(name, k string) error {
	allowed := append([]string{}, toolArgsAllowlist[name]...)
	if repoTools[name] {
		allowed = append(allowed, "repo")
	}
	sort.Strings(allowed)
	if len(allowed) == 0 {
		return fmt.Errorf("%s accepts no arguments; got %q", name, k)
	}
	if canonical := canonicalArgFor(name, k); canonical != "" {
		return fmt.Errorf("unexpected argument %q for %s: use %q (allowed: %s)", k, name, canonical, strings.Join(allowed, ", "))
	}
	return fmt.Errorf("unexpected argument %q for %s (allowed: %s)", k, name, strings.Join(allowed, ", "))
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
