// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// LoadBundle loads a model bundle and attaches its lazily built CODE identifier index,
// so validation can resolve authored references to in-repository files and declarations.
// TRLC-LINKS: REQ-EMG-059
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-CODEMAP-INFERENCE
func LoadBundle(manifestPath string) (model.Bundle, error) {
	bundle, err := model.LoadBundle(manifestPath)
	if err != nil {
		return bundle, err
	}
	return WithCodeIndex(bundle), nil
}

// WithCodeIndex attaches the lazily built CODE identifier index unless the bundle has one,
// so bundles loaded by callers with model.LoadBundle resolve CODE references too. An
// in-memory bundle without a manifest or architecture path has no module root and gets none.
// TRLC-LINKS: REQ-EMG-059
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-CODEMAP-INFERENCE
func WithCodeIndex(bundle model.Bundle) model.Bundle {
	if bundle.CodeIndex != nil || (bundle.ManifestPath == "" && bundle.ArchitecturePath == "") {
		return bundle
	}
	var once sync.Once
	var index map[string]int
	indexed := bundle
	bundle.CodeIndex = func() map[string]int {
		once.Do(func() { index = buildCodeIndex(indexed) })
		return index
	}
	return bundle
}

// moduleCodeRoot is the directory a model owns for code: bundle.CodeScopeRoot when set (a
// resolved dependency's module directory), else the git working tree root, or the manifest
// directory when another engmod.yml encloses the model below that root.
// TRLC-LINKS: REQ-EMG-057, REQ-EMG-067
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func moduleCodeRoot(bundle model.Bundle) string {
	if root := strings.TrimSpace(bundle.CodeScopeRoot); root != "" {
		return absOrSelf(root)
	}
	manifestDir, err := filepath.Abs(ModelRootDir(bundle))
	if err != nil {
		return ModelRootDir(bundle)
	}
	gitRoot := gitWorkTreeRoot(manifestDir)
	if gitRoot == "" || hasEnclosingModule(manifestDir, gitRoot) {
		return manifestDir
	}
	return gitRoot
}

// absOrSelf is the absolute form of a path, or the path itself when that fails.
// TRLC-LINKS: REQ-EMG-057, REQ-EMG-067
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func absOrSelf(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// hasEnclosingModule reports whether another engmod.yml sits in a directory
// above manifestDir, up to the git root.
// TRLC-LINKS: REQ-EMG-057, REQ-EMG-067
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func hasEnclosingModule(manifestDir, gitRoot string) bool {
	for dir := filepath.Dir(manifestDir); composition.PathWithin(gitRoot, dir); dir = filepath.Dir(dir) {
		if isRegularFile(filepath.Join(dir, "engmod.yml")) {
			return true
		}
		if dir == gitRoot || filepath.Dir(dir) == dir {
			break
		}
	}
	return false
}

// gitWorkTreeRoot returns the nearest ancestor holding a .git entry (a directory, or a
// file for linked worktrees and submodules), or "" outside a repository.
// TRLC-LINKS: REQ-EMG-057
func gitWorkTreeRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// WithDefaultCodeRoot falls back to the module root when no existing code root is given.
// TRLC-LINKS: REQ-EMG-057
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func WithDefaultCodeRoot(bundle model.Bundle, roots []string) []string {
	if len(roots) > 0 {
		return roots
	}
	return []string{moduleCodeRoot(bundle)}
}

// codeScanSkipper reports whether a path is outside the model's code scope: a directory of
// another module, a path matching inferenceHints.exclude, or a file ignored by git.
// Exclusions apply only inside the module root, where the patterns are anchored.
// TRLC-LINKS: REQ-EMG-057
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func codeScanSkipper(bundle model.Bundle) func(path string, isDir bool) bool {
	moduleRoot := moduleCodeRoot(bundle)
	manifestDir, _ := filepath.Abs(ModelRootDir(bundle))
	scope := codeScope{
		moduleRoot:  moduleRoot,
		manifestDir: manifestDir,
		patterns:    bundle.Architecture.InferenceHints.Exclude,
		visible:     gitVisibleFiles(moduleRoot),
	}
	return scope.skip
}

// codeScope holds what decides whether a path is outside the model's code scope.
type codeScope struct {
	moduleRoot  string
	manifestDir string
	patterns    []string
	visible     map[string]bool
}

// skip reports whether the path is outside the code scope.
// TRLC-LINKS: REQ-EMG-057
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func (c codeScope) skip(p string, isDir bool) bool {
	if isDir && c.isNestedModule(p) {
		return true
	}
	rel, ok := c.relativePath(p)
	if !ok {
		return false
	}
	if excludedPath(rel, c.patterns) {
		return true
	}
	return !isDir && c.visible != nil && !c.visible[rel]
}

// isNestedModule reports whether a directory other than the model's own holds an engmod.yml.
// TRLC-LINKS: REQ-EMG-057
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func (c codeScope) isNestedModule(p string) bool {
	return p != c.manifestDir && isRegularFile(filepath.Join(p, "engmod.yml"))
}

// relativePath is the slashed path relative to the module root; false when
// the path lies outside it.
// TRLC-LINKS: REQ-EMG-057
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func (c codeScope) relativePath(p string) (string, bool) {
	rel, err := filepath.Rel(c.moduleRoot, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// excludedPath matches a module-relative path against exclude globs. A
// pattern excludes a path it matches and everything below a directory it
// matches, so dir, dir/, dir/* and dir/** all leave the whole directory out;
// a leading ./ is ignored. Patterns ValidateExcludePattern refuses match
// nothing.
// TRLC-LINKS: REQ-EMG-057, REQ-EMG-114
func excludedPath(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern, ok := normalizeExcludePattern(pattern)
		if ok && matchesSelfOrAncestor(pattern, rel) {
			return true
		}
	}
	return false
}

// matchesSelfOrAncestor reports whether the glob matches the path or one of its ancestors.
// TRLC-LINKS: REQ-EMG-057, REQ-EMG-114
func matchesSelfOrAncestor(pattern, rel string) bool {
	for prefix := rel; prefix != "." && prefix != "/" && prefix != ""; prefix = path.Dir(prefix) {
		if matched, _ := path.Match(pattern, prefix); matched {
			return true
		}
	}
	return false
}

// normalizeExcludePattern returns the glob an exclude pattern matches
// against a path and its ancestors: dir/** and dir/ become dir, ./ is
// dropped. It reports false for a pattern validation refuses.
// TRLC-LINKS: REQ-EMG-114
func normalizeExcludePattern(pattern string) (string, bool) {
	if validate.ExcludePatternProblem(pattern) != "" {
		return "", false
	}
	p := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(pattern)), "./")
	p = strings.TrimSuffix(p, "/**")
	p = strings.TrimSuffix(p, "/")
	return p, p != ""
}

// CodeScopeSkip returns the skip function of the model's code scope (see
// codeScanSkipper): the check, the trace and the MCP tools share it, so they
// scan the same files.
// TRLC-LINKS: REQ-EMG-114
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-MCP-SERVER
func CodeScopeSkip(bundle model.Bundle) func(path string, isDir bool) bool {
	return codeScanSkipper(bundle)
}

// ScanCodeScope scans root for trace markers with the model's code scope:
// git-ignored files, inferenceHints.exclude paths and nested modules are
// left out, identifiers are module-relative and paths root-relative.
// TRLC-LINKS: REQ-EMG-114
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-MCP-SERVER
func ScanCodeScope(bundle model.Bundle, root string) ([]codemap.Symbol, []validate.Diagnostic, error) {
	return codemap.ScanWithOptions(root, codemap.ScanOptions{IDBase: moduleCodeRoot(bundle), Skip: codeScanSkipper(bundle)})
}

// ListDeclarations lists every named declaration under root with the model's code scope (see ScanCodeScope), marked
// or not, for tools that jump from a use to its definition.
// TRLC-LINKS: REQ-EMG-192
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func ListDeclarations(bundle model.Bundle, root string) ([]codemap.Declaration, error) {
	return codemap.ListDeclarations(root, codemap.ScanOptions{IDBase: moduleCodeRoot(bundle), Skip: codeScanSkipper(bundle)})
}

// gitVisibleFiles lists files git tracks or would track (not ignored) under root, relative to
// root. It returns nil when root is not in a git repository, git is unavailable, or root
// itself is ignored (a dependency in a consumer's git-ignored module cache), which disables
// ignore filtering.
// TRLC-LINKS: REQ-EMG-057, REQ-EMG-067
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func gitVisibleFiles(root string) map[string]bool {
	if gitWorkTreeRoot(root) == "" {
		return nil
	}
	if exec.Command("git", "-C", root, "check-ignore", "-q", ".").Run() == nil {
		return nil
	}
	cmd := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	visible := map[string]bool{}
	for _, name := range bytes.Split(out, []byte{0}) {
		if len(name) > 0 {
			visible[string(name)] = true
		}
	}
	return visible
}

// buildCodeIndex counts the distinct files and declarations producing each CODE identifier
// in the model's code scope. Only paths inside the module root get file identifiers.
// TRLC-LINKS: REQ-EMG-058, REQ-EMG-059, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-VALIDATION-ENGINE
func buildCodeIndex(bundle model.Bundle) map[string]int {
	b := &codeIndexBuilder{
		moduleRoot: moduleCodeRoot(bundle),
		skip:       codeScanSkipper(bundle),
		locations:  map[string]map[string]bool{},
	}
	for _, root := range UniqueExistingSources(WithDefaultCodeRoot(bundle, nil)) {
		_ = filepath.WalkDir(root, b.walker(root))
		b.addSymbols(root)
	}
	index := make(map[string]int, len(b.locations))
	for id, locs := range b.locations {
		index[id] = len(locs)
	}
	return index
}

// codeIndexBuilder collects the locations producing each CODE identifier.
type codeIndexBuilder struct {
	moduleRoot string
	skip       func(path string, isDir bool) bool
	locations  map[string]map[string]bool
}

// add records one location of an identifier.
// TRLC-LINKS: REQ-EMG-058, REQ-EMG-059, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-VALIDATION-ENGINE
func (b *codeIndexBuilder) add(id, location string) {
	if b.locations[id] == nil {
		b.locations[id] = map[string]bool{}
	}
	b.locations[id][location] = true
}

// walker is the WalkDir callback of one root: it skips unreadable entries.
// TRLC-LINKS: REQ-EMG-058, REQ-EMG-059, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-VALIDATION-ENGINE
func (b *codeIndexBuilder) walker(root string) fs.WalkDirFunc {
	return func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		return b.visit(root, p, d)
	}
}

// visit prunes skipped directories and gives every other in-scope file inside
// the module root its file identifier.
// TRLC-LINKS: REQ-EMG-058, REQ-EMG-059, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-VALIDATION-ENGINE
func (b *codeIndexBuilder) visit(root, p string, d fs.DirEntry) error {
	if d.IsDir() {
		if p != root && (SkipScanDir(d.Name()) || b.skip(p, true)) {
			return filepath.SkipDir
		}
		return nil
	}
	if !b.skip(p, false) {
		b.addFile(p)
	}
	return nil
}

// addFile gives a file inside the module root its file identifier.
// TRLC-LINKS: REQ-EMG-058, REQ-EMG-059, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-VALIDATION-ENGINE
func (b *codeIndexBuilder) addFile(p string) {
	if rel, err := filepath.Rel(b.moduleRoot, p); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		b.add(codemap.FileID(rel), p)
	}
}

// addSymbols records the CODE identifiers of the trace-marked symbols under root.
// TRLC-LINKS: REQ-EMG-058, REQ-EMG-059, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-VALIDATION-ENGINE
func (b *codeIndexBuilder) addSymbols(root string) {
	symbols, _, err := codemap.ScanWithOptions(root, codemap.ScanOptions{IDBase: b.moduleRoot, Skip: b.skip})
	if err != nil {
		return
	}
	for _, s := range symbols {
		if strings.HasPrefix(s.TraceID, "CODE-") {
			b.add(s.TraceID, symbolLocation(root, s))
		}
	}
}

// symbolLocation is the file of a symbol, plus its line unless the symbol is
// an artifact (the file itself, not a second declaration).
// TRLC-LINKS: REQ-EMG-058, REQ-EMG-059, REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-VALIDATION-ENGINE
func symbolLocation(root string, s codemap.Symbol) string {
	location := filepath.Join(sourceRootDir(root), filepath.FromSlash(s.Path))
	if !codemap.IsArtifactFile(s.Path) {
		location += ":" + strconv.Itoa(s.Line)
	}
	return location
}

// TRLC-LINKS: REQ-EMG-057
func isRegularFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}
