// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-057
func InferCodeItems(bundle model.Bundle, codeRootOption string) ([]CodeItem, []validate.Diagnostic) {
	baseDir := ModelRootDir(bundle)
	roots := []string{}
	if strings.TrimSpace(codeRootOption) != "" {
		roots = append(roots, codeRootOption)
	}
	roots = WithDefaultCodeRoot(bundle, UniqueExistingSources(roots))
	if len(roots) == 0 {
		return nil, nil
	}
	inf := &codeInference{
		moduleRoot: moduleCodeRoot(bundle),
		skip:       codeScanSkipper(bundle),
		seen:       map[string]bool{},
		items:      []CodeItem{},
		diags:      []validate.Diagnostic{},
	}
	for _, root := range roots {
		inf.scanRoot(root)
	}
	if failed := relativizeCodeItems(inf.items, baseDir); failed != nil {
		return nil, failed
	}
	sortCodeItems(inf.items)
	return inf.items, validate.SortDiagnostics(inf.diags)
}

// codeInference accumulates the code items and diagnostics of the scanned roots.
type codeInference struct {
	moduleRoot string
	skip       func(path string, isDir bool) bool
	seen       map[string]bool
	items      []CodeItem
	diags      []validate.Diagnostic
}

// rootScan is what one scanned root yields: the base for its relative paths,
// the owner and description of each source file and its trace-marked symbols.
type rootScan struct {
	base         string
	owners       map[string]string
	descriptions map[string]string
	symbols      []codemap.Symbol
}

// scanRoot scans one code root; a failed symbol scan is a warning and skips the root.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-057
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (c *codeInference) scanRoot(root string) {
	abs := root
	if !filepath.IsAbs(abs) {
		abs, _ = filepath.Abs(root)
	}
	rs := rootScan{base: sourceRootDir(abs), owners: map[string]string{}, descriptions: map[string]string{}}
	for rel, m := range scanCodeMetadata(abs, c.skip) {
		rs.owners[rel] = m.Owner
		if x := strings.TrimSpace(m.Description); x != "" {
			rs.descriptions[rel] = x
		}
	}
	symbols, sDiags, err := codemap.ScanWithOptions(abs, codemap.ScanOptions{IDBase: c.moduleRoot, Skip: c.skip})
	if err != nil {
		c.diags = append(c.diags, validate.Diagnostic{
			Code:     "code.scan_failed",
			Severity: validate.SeverityWarning,
			Message:  err.Error(),
			Path:     abs,
		})
		return
	}
	c.diags = append(c.diags, sDiags...)
	rs.symbols = symbols
	c.addFileItems(rs)
	c.addSymbolItems(rs)
}

// addOnce appends the item unless its key was seen; it reports whether it did.
// TRLC-LINKS: REQ-EMG-010
func (c *codeInference) addOnce(key string, item CodeItem) bool {
	if c.seen[key] {
		return false
	}
	c.seen[key] = true
	c.items = append(c.items, item)
	return true
}

// addFileItems adds one source-file item with its dependencies per owned file.
// TRLC-LINKS: REQ-EMG-010
func (c *codeInference) addFileItems(rs rootScan) {
	for rel, owner := range rs.owners {
		c.addFileItem(rs, rel, owner)
	}
}

// addFileItem adds a source-file item and, when it is new, the dependencies it imports.
// TRLC-LINKS: REQ-EMG-010
func (c *codeInference) addFileItem(rs rootScan, rel, owner string) {
	absPath := filepath.Join(rs.base, rel)
	item := CodeItem{
		Element:     rel,
		Kind:        "source_file",
		AbsPath:     absPath,
		Owner:       owner,
		Description: strings.TrimSpace(rs.descriptions[rel]),
		Source:      rel,
	}
	if !c.addOnce("source_file|"+absPath+"|"+owner, item) {
		return
	}
	deps, depDiags := parseCodeDependencies(rs.base, rel, owner)
	c.diags = append(c.diags, depDiags...)
	for _, dep := range deps {
		dep.AbsPath = absPath
		c.addOnce(dep.Kind+"|"+dep.Element+"|"+dep.Owner+"|"+filepath.Join(rs.base, dep.Source), dep)
	}
}

// addSymbolItems adds one item per trace-marked symbol, owned by the nearest owned path.
// TRLC-LINKS: REQ-EMG-010
func (c *codeInference) addSymbolItems(rs rootScan) {
	for _, s := range rs.symbols {
		label := s.TraceID
		if strings.TrimSpace(label) == "" {
			label = s.Signature
		}
		absPath := filepath.Join(rs.base, s.Path)
		c.addOnce("symbol|"+label+"|"+absPath+fmt.Sprintf("|%d", s.Line), CodeItem{
			Element:    label,
			Kind:       "symbol",
			Owner:      ownerForPath(s.Path, rs.owners),
			Source:     fmt.Sprintf("%s:%d", s.Path, s.Line),
			AbsPath:    absPath,
			Implements: textutil.UniqueSorted(s.Implements),
			ModelLinks: textutil.UniqueSorted(s.ModelLinks),
		})
	}
}

// relativizeCodeItems gives symbol items their model-relative evidence path and,
// where roots produce the same source label for different files, a
// model-relative source. Publication and trace matrices also use Source as
// identity, so those collisions must be disambiguated. It returns the failure
// diagnostics when a path cannot be made relative.
// TRLC-LINKS: REQ-EMG-010
func relativizeCodeItems(items []CodeItem, baseDir string) []validate.Diagnostic {
	ambiguous := ambiguousSources(items)
	for i := range items {
		if failed := setEvidencePath(&items[i], baseDir); failed != nil {
			return failed
		}
		if ambiguous[items[i].Source] {
			rebaseSource(&items[i], baseDir)
		}
	}
	return nil
}

// ambiguousSources lists the Source labels shared by items of different files.
// Compact scan-relative display paths are preserved otherwise.
// TRLC-LINKS: REQ-EMG-010
func ambiguousSources(items []CodeItem) map[string]bool {
	pathsBySource := map[string]string{}
	ambiguous := map[string]bool{}
	for _, item := range items {
		if previous, ok := pathsBySource[item.Source]; ok && previous != item.AbsPath {
			ambiguous[item.Source] = true
		}
		pathsBySource[item.Source] = item.AbsPath
	}
	return ambiguous
}

// setEvidencePath sets the model-relative evidence path of a symbol inside baseDir.
// TRLC-LINKS: REQ-EMG-010
func setEvidencePath(item *CodeItem, baseDir string) []validate.Diagnostic {
	if item.Kind != "symbol" || !composition.PathWithin(baseDir, item.AbsPath) {
		return nil
	}
	relative, err := filepath.Rel(baseDir, item.AbsPath)
	if err != nil {
		return []validate.Diagnostic{{Code: "code.relative_path_failed", Severity: validate.SeverityError, Message: err.Error(), Path: item.AbsPath}}
	}
	item.EvidencePath = filepath.ToSlash(relative)
	return nil
}

// rebaseSource replaces the item's source path with its path relative to
// baseDir (the absolute path when it cannot be made relative), keeping the line.
// TRLC-LINKS: REQ-EMG-010
func rebaseSource(item *CodeItem, baseDir string) {
	_, line := SplitSourceLine(item.Source)
	path, err := filepath.Rel(baseDir, item.AbsPath)
	if err != nil {
		path = item.AbsPath
	}
	item.Source = filepath.ToSlash(path)
	if line > 0 {
		item.Source += fmt.Sprintf(":%d", line)
	}
}

// sortCodeItems orders items by kind, element and source.
// TRLC-LINKS: REQ-EMG-010
func sortCodeItems(items []CodeItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		if items[i].Element != items[j].Element {
			return items[i].Element < items[j].Element
		}
		return items[i].Source < items[j].Source
	})
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// SkipScanDir reports whether a directory must be excluded from code/metadata walks.
// Dependency caches and dot-directories (tool caches like .opencode, the .engmod
// composition cache, .git) are not source; scanning them makes generated artifacts
// depend on the local environment and breaks reproducibility.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-012
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func SkipScanDir(name string) bool {
	if name == "node_modules" || name == "vendor" {
		return true
	}
	return len(name) > 1 && name[0] == '.'
}

// TRLC-LINKS: REQ-EMG-010
func parseCodeDependencies(root, rel, owner string) ([]CodeItem, []validate.Diagnostic) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, []validate.Diagnostic{{
			Code:     "code.dependency_read_failed",
			Severity: validate.SeverityWarning,
			Message:  err.Error(),
			Path:     path,
		}}
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go":
		return parseGoDependencies(rel, data, owner)
	case ".ts", ".tsx":
		return parseTypeScriptDependencies(rel, string(data), owner), nil
	case ".rs":
		return parseRustDependencies(rel, string(data), owner), nil
	default:
		return nil, nil
	}
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type codeFileMetadata struct {
	Owner       string
	Description string
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func parseGoDependencies(rel string, src []byte, owner string) ([]CodeItem, []validate.Diagnostic) {
	full := filepath.ToSlash(filepath.Clean(rel))
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, full, src, parser.ImportsOnly)
	if err != nil {
		return nil, []validate.Diagnostic{{
			Code:     "code.go_dependency_parse_failed",
			Severity: validate.SeverityWarning,
			Message:  err.Error(),
			Path:     rel,
		}}
	}
	out := []CodeItem{}
	for _, im := range f.Imports {
		path := strings.TrimSpace(strings.Trim(im.Path.Value, "\""))
		if path == "" {
			continue
		}
		kind := goImportKind(path)
		out = append(out, CodeItem{
			Element: path,
			Kind:    kind,
			Owner:   owner,
			Source:  rel,
		})
	}
	return out, nil
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func goImportKind(path string) string {
	switch {
	case strings.HasPrefix(path, "github.com/labeth/engineering-model-go/"):
		return "library_first_party"
	case strings.Contains(path, "."):
		return "library_external"
	default:
		return "library_stdlib"
	}
}

var (
	tsImportRe  = regexp.MustCompile(`(?m)^\s*import\s+(?:[^'"]+?\s+from\s+)?['"]([^'"]+)['"]`)
	tsRequireRe = regexp.MustCompile(`(?m)require\(\s*['"]([^'"]+)['"]\s*\)`)
	rsUseRe     = regexp.MustCompile(`(?m)^\s*use\s+([A-Za-z0-9_:\{\}]+)\s*;`)
)

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func parseTypeScriptDependencies(rel, content, owner string) []CodeItem {
	out := []CodeItem{}
	specs := append(moduleSpecs(tsImportRe, content), moduleSpecs(tsRequireRe, content)...)
	for _, dep := range specs {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		kind := "library_external"
		if strings.HasPrefix(dep, ".") || strings.HasPrefix(dep, "/") {
			kind = "library_first_party"
		}
		out = append(out, CodeItem{
			Element: dep,
			Kind:    kind,
			Owner:   owner,
			Source:  rel,
		})
	}
	return out
}

// moduleSpecs lists the first capture group of each match of an import pattern.
// TRLC-LINKS: REQ-EMG-010
func moduleSpecs(re *regexp.Regexp, content string) []string {
	var specs []string
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			specs = append(specs, m[1])
		}
	}
	return specs
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func parseRustDependencies(rel, content, owner string) []CodeItem {
	out := []CodeItem{}
	for _, m := range rsUseRe.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		dep := strings.TrimSpace(m[1])
		if dep == "" {
			continue
		}
		kind := "library_external"
		if strings.HasPrefix(dep, "crate::") || strings.HasPrefix(dep, "self::") || strings.HasPrefix(dep, "super::") {
			kind = "library_first_party"
		}
		out = append(out, CodeItem{
			Element: dep,
			Kind:    kind,
			Owner:   owner,
			Source:  rel,
		})
	}
	return out
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-084
func scanCodeMetadata(root string, skip func(path string, isDir bool) bool) map[string]codeFileMetadata {
	scan := metadataScan{root: root, skip: skip, out: map[string]codeFileMetadata{}}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		return scan.visit(path, d)
	})
	return scan.out
}

// metadataScan collects the owner and description of the source files under root.
type metadataScan struct {
	root string
	skip func(path string, isDir bool) bool
	out  map[string]codeFileMetadata
}

// visit prunes skipped directories and records the metadata of each in-scope file.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (m metadataScan) visit(path string, d fs.DirEntry) error {
	if d.IsDir() {
		if m.skipDir(path, d.Name()) {
			return filepath.SkipDir
		}
		return nil
	}
	if m.skip != nil && m.skip(path, false) {
		return nil
	}
	if rel, meta, ok := m.fileMetadata(path); ok {
		m.out[rel] = meta
	}
	return nil
}

// skipDir reports whether a directory is left out of the walk.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (m metadataScan) skipDir(path, name string) bool {
	if SkipScanDir(name) {
		return true
	}
	return path != m.root && m.skip != nil && m.skip(path, true)
}

// metadataExtensions are the source file extensions scanned for ownership markers.
var metadataExtensions = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".rs": true, ".v": true,
	".js": true, ".mjs": true, ".cjs": true, ".py": true,
}

// fileMetadata reads one file: the source-relative path and its metadata, or false
// when the file is not a source file, is unreadable or an artifact without markers.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (m metadataScan) fileMetadata(path string) (string, codeFileMetadata, bool) {
	if codemap.IsArtifactFile(path) {
		return m.artifactMetadata(path)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if !metadataExtensions[ext] {
		return "", codeFileMetadata{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", codeFileMetadata{}, false
	}
	return m.relativePath(path), sourceFileMetadata(path, ext, data), true
}

// artifactMetadata reads a container file or Markdown document, which is a
// source file only when it carries trace markers; its owner comes from its comments.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (m metadataScan) artifactMetadata(path string) (string, codeFileMetadata, bool) {
	if !codemap.ReadableArtifact(path) {
		return "", codeFileMetadata{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", codeFileMetadata{}, false
	}
	owner, _, links, modelLinks := codemap.ArtifactMarkers(path, data)
	if owner == "" && links == 0 && modelLinks == 0 {
		return "", codeFileMetadata{}, false
	}
	return m.relativePath(path), codeFileMetadata{Owner: textutil.NonEmpty(owner, "unresolved")}, true
}

// relativePath is the slashed path relative to the source root, else the path itself.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (m metadataScan) relativePath(path string) string {
	rel, err := filepath.Rel(sourceRootDir(m.root), path)
	if err != nil {
		rel = path
	}
	return filepath.ToSlash(rel)
}

// sourceFileMetadata finds the owner and the first description marker in a source
// file; Python files count only their comment lines.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func sourceFileMetadata(path, ext string, data []byte) codeFileMetadata {
	include := func(int) bool { return true }
	if ext == ".py" {
		comments, _ := codemap.CommentLines(path, data)
		include = func(line int) bool { return comments[line] }
	}
	meta := codeFileMetadata{Owner: "unresolved"}
	for index, line := range strings.Split(string(data), "\n") {
		if !include(index + 1) {
			continue
		}
		if owner := ownerMarkerValue(line); owner != "" {
			meta.Owner = owner
		}
		if meta.Description == "" {
			meta.Description, _ = extractCodeDescriptionMarker(line)
		}
	}
	return meta
}

// ownerMarkerValue is the trimmed ENGMODEL-OWNER-UNIT value of a line, or "".
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func ownerMarkerValue(line string) string {
	v, ok := extractMarkerValue(strings.TrimSpace(line), "ENGMODEL-OWNER-UNIT:")
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func extractCodeDescriptionMarker(line string) (string, bool) {
	markers := []string{
		"ENGMODEL-CODE-DESCRIPTION:",
		"engmodel:code-description:",
		"engmodel:code-description",
		"engmodel.code.description:",
		"engmodel.code.description",
	}
	for _, marker := range markers {
		if v := markerDescription(line, marker); v != "" {
			return v, true
		}
	}
	return "", false
}

// markerDescription is the description following the marker on the line, or "".
// TRLC-LINKS: REQ-EMG-010
func markerDescription(line, marker string) string {
	v, ok := extractMarkerValue(line, marker)
	if !ok {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v), ":"))
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func ownerForPath(path string, owners map[string]string) string {
	if x := strings.TrimSpace(owners[path]); x != "" && x != "unresolved" {
		return x
	}
	if best := prefixOwner(path, owners); best != "" {
		return best
	}
	return "unresolved"
}

// prefixOwner is the resolved owner of a path prefix of path, preferring the
// owner whose prefix is longer than the owner chosen so far.
// TRLC-LINKS: REQ-EMG-010
func prefixOwner(path string, owners map[string]string) string {
	best := ""
	for p, owner := range owners {
		if owner == "" || owner == "unresolved" {
			continue
		}
		if strings.HasPrefix(path, p) && len(p) > len(best) {
			best = owner
		}
	}
	return best
}

// sourceRootDir resolves source paths consistently for directory and single-file roots.
// TRLC-LINKS: REQ-EMG-010
func sourceRootDir(root string) string {
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		return filepath.Dir(root)
	}
	return root
}

// TRLC-LINKS: REQ-EMG-010
func UniqueExistingSources(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, path := range in {
		abs, err := filepath.Abs(path)
		if err != nil || seen[abs] {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || (!info.IsDir() && !info.Mode().IsRegular()) {
			continue
		}
		seen[abs] = true
		out = append(out, abs)
	}
	return out
}

// IsVerificationCodeItem reports whether a code item is test code rather than
// production implementation: it lives in a test file, or it is a Rust item
// compiled only for tests (#[cfg(test)] module or #![cfg(test)] file). A
// container build file or Markdown document is never test code, even under
// tests/: its trace markers record implementation evidence.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030, REQ-EMG-084
func IsVerificationCodeItem(item CodeItem) bool {
	path, line := SplitSourceLine(item.Source)
	if isArtifactItem(item, path) {
		return false
	}
	if isVerificationTestPath(item.AbsPath) || isVerificationTestPath(path) {
		return true
	}
	return isRustTestOnlyItem(item, line)
}

// isArtifactItem reports whether the item's source or absolute path is a
// container build file or Markdown document.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030, REQ-EMG-084
func isArtifactItem(item CodeItem, path string) bool {
	return codemap.IsArtifactFile(path) || item.AbsPath != "" && codemap.IsArtifactFile(item.AbsPath)
}

// isRustTestOnlyItem reports whether a symbol item sits on a Rust declaration line
// compiled only for tests.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func isRustTestOnlyItem(item CodeItem, line int) bool {
	if item.Kind != "symbol" || line <= 0 || !strings.EqualFold(filepath.Ext(item.AbsPath), ".rs") {
		return false
	}
	return rustTestOnlyLines(item.AbsPath)[line]
}

// rustTestOnlyCache memoizes the test-only declaration lines of Rust files, keyed by
// absolute path and invalidated by size and modification time.
var rustTestOnlyCache = struct {
	sync.Mutex
	entries map[string]rustTestOnlyEntry
}{entries: map[string]rustTestOnlyEntry{}}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type rustTestOnlyEntry struct {
	size    int64
	modTime int64
	lines   map[int]bool
}

// rustTestOnlyLines returns the declaration lines of a Rust file that codemap marks
// TestOnly. Unreadable files yield no lines, so their code stays production code.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func rustTestOnlyLines(absPath string) map[int]bool {
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return nil
	}
	rustTestOnlyCache.Lock()
	defer rustTestOnlyCache.Unlock()
	if e, ok := rustTestOnlyCache.entries[absPath]; ok && e.current(info) {
		return e.lines
	}
	symbols, _, err := codemap.Scan(absPath)
	if err != nil {
		return nil
	}
	lines := testOnlyLines(symbols)
	rustTestOnlyCache.entries[absPath] = rustTestOnlyEntry{size: info.Size(), modTime: info.ModTime().UnixNano(), lines: lines}
	return lines
}

// current reports whether the entry was taken from a file of this size and modification time.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func (e rustTestOnlyEntry) current(info os.FileInfo) bool {
	return e.size == info.Size() && e.modTime == info.ModTime().UnixNano()
}

// testOnlyLines is the set of declaration lines of the test-only symbols.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func testOnlyLines(symbols []codemap.Symbol) map[int]bool {
	lines := map[int]bool{}
	for _, s := range symbols {
		if s.TestOnly {
			lines[s.Line] = true
		}
	}
	return lines
}
