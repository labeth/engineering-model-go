// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) ensureRepoIndex() error {
	s.indexOnce.Do(func() {
		s.indexErr = filepath.Walk(s.repoRoot, s.repoIndexVisitor(s.repoIndexSkip()))
		if errors.Is(s.indexErr, errRepoIndexLimit) {
			s.indexErr = nil
		}
	})
	return s.indexErr
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) repoIndexSkip() func(string, bool) bool {
	if s.bundle != nil {
		return codescan.CodeScopeSkip(*s.bundle)
	}
	return func(string, bool) bool { return false }
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) repoIndexVisitor(skip func(string, bool) bool) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			return s.repoIndexDir(path, info, skip)
		}
		return s.repoIndexFile(path, info, skip)
	}
}

// repoIndexIgnoredDirs are never walked by the repo index.
var repoIndexIgnoredDirs = map[string]bool{".git": true, "node_modules": true, ".opencode": true, ".idea": true, ".vscode": true, ".engflow": true}

// repoIndexExtensions are the file types the repo index reads.
var repoIndexExtensions = map[string]bool{".go": true, ".ts": true, ".tsx": true, ".rs": true, ".py": true, ".js": true, ".yaml": true, ".yml": true, ".md": true}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) repoIndexDir(path string, info os.FileInfo, skip func(string, bool) bool) error {
	if repoIndexIgnoredDirs[info.Name()] {
		return filepath.SkipDir
	}
	if path != s.repoRoot && skip(path, true) {
		return filepath.SkipDir
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) repoIndexFile(path string, info os.FileInfo, skip func(string, bool) bool) error {
	if skip(path, false) {
		return nil
	}
	if len(s.repoFiles) >= maxIndexedFiles {
		return errRepoIndexLimit
	}
	if !repoIndexExtensions[strings.ToLower(filepath.Ext(path))] || info.Size() > maxIndexedFileBytes {
		return nil
	}
	b, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil
	}
	s.repoFiles = append(s.repoFiles, indexedFile{Path: path, Content: string(b)})
	return nil
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type taskContext struct {
	s               *Server
	reqID           string
	entityID        string
	path            string
	query           string
	max             int
	entities        []map[string]any
	requirements    []model.Requirement
	flows           []model.Flow
	threats         []model.ThreatScenario
	files           []string
	implementations []map[string]any
	tests           []map[string]any
	diagnostics     []map[string]any
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) contextForTask(args map[string]any) map[string]any {
	tc := &taskContext{
		s:               s,
		reqID:           firstNonEmptyArg(args, "requirementId"),
		entityID:        firstNonEmptyArg(args, "entityId"),
		path:            firstNonEmptyArg(args, "path"),
		query:           strings.ToLower(firstNonEmptyArg(args, "query")),
		max:             maxArg(args, 25),
		entities:        []map[string]any{},
		requirements:    []model.Requirement{},
		flows:           []model.Flow{},
		threats:         []model.ThreatScenario{},
		files:           []string{},
		implementations: []map[string]any{},
		tests:           []map[string]any{},
		diagnostics:     []map[string]any{},
	}
	tc.addRequirement()
	tc.addEntity()
	tc.addQuery()
	tc.addPath()
	return tc.result()
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (tc *taskContext) addRequirement() {
	if tc.reqID == "" {
		return
	}
	s := tc.s
	if r := s.findRequirement(tc.reqID); r != nil {
		tc.requirements = append(tc.requirements, *r)
	}
	tc.flows = append(tc.flows, s.flowsForRequirement(tc.reqID)...)
	tc.threats = append(tc.threats, s.threatsForRequirement(tc.reqID)...)
	tc.files = append(tc.files, s.filesForRequirement(tc.reqID)...)
	reqTests, reqDiags := s.testsForRequirement(tc.reqID)
	tc.tests = append(tc.tests, reqTests...)
	tc.diagnostics = append(tc.diagnostics, reqDiags...)
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (tc *taskContext) addEntity() {
	if tc.entityID == "" {
		return
	}
	s := tc.s
	kind, entity, ok := s.modelEntity(tc.entityID)
	if !ok {
		return
	}
	tc.entities = append(tc.entities, map[string]any{"kind": kind, "id": tc.entityID, "entity": entity})
	entityImpls, entityDiags := s.implementationsForModelLink(tc.entityID)
	tc.implementations = append(tc.implementations, entityImpls...)
	tc.diagnostics = append(tc.diagnostics, entityDiags...)
	entityTests, entityTestDiags := s.testsForModelLink(tc.entityID)
	tc.tests = append(tc.tests, entityTests...)
	tc.diagnostics = append(tc.diagnostics, entityTestDiags...)
	tc.files = append(tc.files, s.filesContaining(tc.entityID)...)
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (tc *taskContext) addQuery() {
	if tc.query == "" {
		return
	}
	tc.entities = append(tc.entities, tc.s.modelEntityNodes(tc.query, "", tc.max)...)
	tc.files = append(tc.files, tc.s.filesContaining(strings.ToUpper(tc.query))...)
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (tc *taskContext) addPath() {
	if tc.path == "" {
		return
	}
	if safePath, ok := tc.s.resolvePathInRepo(tc.path); ok {
		tc.files = append(tc.files, safePath)
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (tc *taskContext) result() map[string]any {
	implementations := limitRows(tc.implementations, tc.max)
	tests := limitRows(tc.tests, tc.max)
	return map[string]any{
		"query":               tc.query,
		"requirementId":       tc.reqID,
		"entityId":            tc.entityID,
		"entities":            limitRows(tc.entities, tc.max),
		"requirements":        tc.requirements,
		"flows":               tc.flows,
		"threatScenarios":     tc.threats,
		"files":               uniqueStrings(tc.files),
		"implementations":     implementations,
		"implementationFiles": groupImplementationsByFile(implementations),
		"tests":               tests,
		"testFiles":           groupImplementationsByFile(tests),
		"diagnostics":         uniqueDiagnosticRows(tc.diagnostics),
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) strictCoverageStatus(path string) map[string]any {
	_, diagnostics := s.scanCodeSymbols()
	relPath, ok := s.coverageRelPath(strings.TrimSpace(path))
	if !ok {
		return map[string]any{"status": "error", "diagnostics": []map[string]any{{"code": "path_outside_repo", "message": "coverage.strictStatus path must be inside repoRoot"}}}
	}
	filtered, counts := filterCoverageDiagnostics(diagnostics, relPath)
	return map[string]any{"status": coverageStatusFor(counts), "path": relPath, "summary": counts, "diagnostics": filtered}
}

// coverageRelPath returns the repo-relative slash form of a non-empty path;
// ok is false when the path resolves outside the repo.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) coverageRelPath(path string) (string, bool) {
	if path == "" {
		return "", true
	}
	safePath, ok := s.resolvePathInRepo(path)
	if !ok {
		return "", false
	}
	rel, err := filepath.Rel(s.repoRoot, safePath)
	if err != nil {
		return "", true
	}
	return filepath.ToSlash(rel), true
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func filterCoverageDiagnostics(diagnostics []map[string]any, relPath string) ([]map[string]any, map[string]int) {
	filtered := []map[string]any{}
	counts := map[string]int{}
	for _, d := range diagnostics {
		diagPath, _ := d["path"].(string)
		if relPath != "" && diagnosticFilePath(diagPath) != relPath {
			continue
		}
		filtered = append(filtered, d)
		if code, _ := d["code"].(string); code != "" {
			counts[code]++
		}
	}
	return filtered, counts
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func coverageStatusFor(counts map[string]int) string {
	for _, code := range []string{"code.missing_trlc_link", "code.invalid_trlc_link", "scan_failed"} {
		if counts[code] > 0 {
			return "fail"
		}
	}
	return "pass"
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
	cleanTarget, err := absoluteTarget(root, path)
	if err != nil {
		return "", false
	}
	if !pathWithinRoot(cleanTarget, filepath.Clean(root)) {
		return "", false
	}
	return cleanTarget, true
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func absoluteTarget(root, path string) (string, error) {
	target := path
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	cleanTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	return filepath.Clean(cleanTarget), nil
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func pathWithinRoot(target, root string) bool {
	return strings.HasPrefix(target, root+string(os.PathSeparator)) || target == root
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) authHintsForInterface(interfaceID string) []string {
	if s.bundle == nil {
		return nil
	}
	hints := []string{}
	for _, f := range s.bundle.Architecture.AuthoredArchitecture.Flows {
		hints = append(hints, stepAuthHints(f.Steps, interfaceID)...)
	}
	sort.Strings(hints)
	return uniqueStrings(hints)
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func stepAuthHints(steps []model.FlowStep, interfaceID string) []string {
	var hints []string
	for _, st := range steps {
		if st.InterfaceRef == interfaceID && st.Authentication != "" {
			hints = append(hints, st.Authentication)
		}
	}
	return hints
}

// ownerUnitMarker returns the value of the first ENGMODEL-OWNER-UNIT comment
// marker and its 1-based line, or "" and 0 when the file has none. Mentions of
// FU IDs elsewhere in the file are ignored; Python markers count only in comments.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-010
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func ownerUnitMarker(path string, content []byte) (string, int) {
	var pythonComments map[int]bool
	isPython := strings.EqualFold(filepath.Ext(path), ".py")
	if isPython {
		pythonComments, _ = codemap.CommentLines(path, content)
	}
	for index, line := range strings.Split(string(content), "\n") {
		if isPython && !pythonComments[index+1] {
			continue
		}
		if value, ok := ownerMarkerValue(line); ok {
			return value, index + 1
		}
	}
	return "", 0
}

// commentPrefixes are the line-comment openers an ownership marker may follow.
var commentPrefixes = []string{"//", "#", "/*", "*", "--", ";", "%"}

// ownerMarkerValue returns the first field after an ENGMODEL-OWNER-UNIT
// marker on a comment line.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-010
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func ownerMarkerValue(line string) (string, bool) {
	const marker = "ENGMODEL-OWNER-UNIT:"
	x, isComment := stripCommentPrefix(line)
	if !isComment || !strings.HasPrefix(strings.ToUpper(x), marker) {
		return "", false
	}
	value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(x[len(marker):]), "*/"))
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", false
	}
	return fields[0], true
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-010
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func stripCommentPrefix(line string) (string, bool) {
	x := strings.TrimSpace(line)
	for _, prefix := range commentPrefixes {
		if strings.HasPrefix(x, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(x, prefix)), true
		}
	}
	return x, false
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func firstNonEmptyArg(args map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := strings.TrimSpace(toString(args[k])); s != "" {
			return s
		}
	}
	return ""
}
