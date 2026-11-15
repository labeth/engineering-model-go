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
		skip := func(string, bool) bool { return false }
		if s.bundle != nil {
			skip = codescan.CodeScopeSkip(*s.bundle)
		}
		s.indexErr = filepath.Walk(s.repoRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				switch info.Name() {
				case ".git", "node_modules", ".opencode", ".idea", ".vscode", ".engflow":
					return filepath.SkipDir
				}
				if path != s.repoRoot && skip(path, true) {
					return filepath.SkipDir
				}
				return nil
			}
			if skip(path, false) {
				return nil
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
	for _, x := range a.HardwareItems {
		if x.ID == id {
			return "hardware_item", x, true
		}
	}
	for _, x := range a.HardwareInterfaces {
		if x.ID == id {
			return "hardware_interface", x, true
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
	for _, x := range a.Machines {
		if x.ID == id {
			return "machine", x, true
		}
	}
	for _, x := range a.MachineStates() {
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
	reqID := firstNonEmptyArg(args, "requirementId")
	entityID := firstNonEmptyArg(args, "entityId")
	path := firstNonEmptyArg(args, "path")
	query := strings.ToLower(firstNonEmptyArg(args, "query"))
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
	for _, x := range a.HardwareItems {
		add("hardware_item", x.ID, x.Name)
	}
	for _, x := range a.HardwareInterfaces {
		add("hardware_interface", x.ID, x.Name)
	}
	for _, x := range a.Controls {
		add("control", x.ID, x.Name)
	}
	for _, x := range a.TrustBoundaries {
		add("trust_boundary", x.ID, x.Name)
	}
	for _, x := range a.Machines {
		add("machine", x.ID, x.Name)
	}
	for _, x := range a.MachineStates() {
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

// ownerUnitMarker returns the value of the first ENGMODEL-OWNER-UNIT comment
// marker and its 1-based line, or "" and 0 when the file has none. Mentions of
// FU IDs elsewhere in the file are ignored; Python markers count only in comments.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-010
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func ownerUnitMarker(path string, content []byte) (string, int) {
	const marker = "ENGMODEL-OWNER-UNIT:"
	var pythonComments map[int]bool
	isPython := strings.EqualFold(filepath.Ext(path), ".py")
	if isPython {
		pythonComments, _ = codemap.CommentLines(path, content)
	}
	for index, line := range strings.Split(string(content), "\n") {
		if isPython && !pythonComments[index+1] {
			continue
		}
		x := strings.TrimSpace(line)
		isComment := false
		for _, prefix := range []string{"//", "#", "/*", "*", "--", ";", "%"} {
			if strings.HasPrefix(x, prefix) {
				x = strings.TrimSpace(strings.TrimPrefix(x, prefix))
				isComment = true
				break
			}
		}
		if !isComment || !strings.HasPrefix(strings.ToUpper(x), marker) {
			continue
		}
		value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(x[len(marker):]), "*/"))
		if fields := strings.Fields(value); len(fields) > 0 {
			return fields[0], index + 1
		}
	}
	return "", 0
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
