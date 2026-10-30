// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engine

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
	"github.com/labeth/engineering-model-go/internal/model"
)

// loadBundle loads a model bundle and attaches its lazily built CODE identifier index,
// so validation can resolve authored references to in-repository files and declarations.
// TRLC-LINKS: REQ-EMG-059
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-CODEMAP-INFERENCE
func loadBundle(manifestPath string) (model.Bundle, error) {
	bundle, err := model.LoadBundle(manifestPath)
	if err != nil {
		return bundle, err
	}
	return withCodeIndex(bundle), nil
}

// withCodeIndex attaches the lazily built CODE identifier index unless the bundle has one,
// so bundles loaded by callers with model.LoadBundle resolve CODE references too. An
// in-memory bundle without a manifest or architecture path has no module root and gets none.
// TRLC-LINKS: REQ-EMG-059
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-CODEMAP-INFERENCE
func withCodeIndex(bundle model.Bundle) model.Bundle {
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
		if abs, err := filepath.Abs(root); err == nil {
			return abs
		}
		return root
	}
	manifestDir, err := filepath.Abs(modelRootDir(bundle))
	if err != nil {
		return modelRootDir(bundle)
	}
	gitRoot := gitWorkTreeRoot(manifestDir)
	if gitRoot == "" {
		return manifestDir
	}
	for dir := filepath.Dir(manifestDir); pathWithin(gitRoot, dir); dir = filepath.Dir(dir) {
		if isRegularFile(filepath.Join(dir, "engmod.yml")) {
			return manifestDir
		}
		if dir == gitRoot || filepath.Dir(dir) == dir {
			break
		}
	}
	return gitRoot
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

// withDefaultCodeRoot falls back to the module root when no existing code root is given.
// TRLC-LINKS: REQ-EMG-057
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func withDefaultCodeRoot(bundle model.Bundle, roots []string) []string {
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
	manifestDir, _ := filepath.Abs(modelRootDir(bundle))
	patterns := bundle.Architecture.InferenceHints.Exclude
	visible := gitVisibleFiles(moduleRoot)
	return func(p string, isDir bool) bool {
		if isDir && p != manifestDir && isRegularFile(filepath.Join(p, "engmod.yml")) {
			return true
		}
		rel, err := filepath.Rel(moduleRoot, p)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return false
		}
		rel = filepath.ToSlash(rel)
		if excludedPath(rel, patterns) {
			return true
		}
		return !isDir && visible != nil && !visible[rel]
	}
}

// excludedPath matches a module-relative path against exclude globs; "dir/**" excludes a subtree.
// TRLC-LINKS: REQ-EMG-057
func excludedPath(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.Trim(filepath.ToSlash(strings.TrimSpace(pattern)), "/")
		if pattern == "" {
			continue
		}
		if base, subtree := strings.CutSuffix(pattern, "/**"); subtree {
			for prefix := rel; prefix != "." && prefix != "/" && prefix != ""; prefix = path.Dir(prefix) {
				if ok, _ := path.Match(base, prefix); ok {
					return true
				}
			}
			continue
		}
		if ok, _ := path.Match(pattern, rel); ok {
			return true
		}
	}
	return false
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
// TRLC-LINKS: REQ-EMG-058, REQ-EMG-059
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-VALIDATION-ENGINE
func buildCodeIndex(bundle model.Bundle) map[string]int {
	moduleRoot := moduleCodeRoot(bundle)
	skip := codeScanSkipper(bundle)
	roots := uniqueExistingSources(withDefaultCodeRoot(bundle, nil))

	locations := map[string]map[string]bool{}
	add := func(id, location string) {
		if locations[id] == nil {
			locations[id] = map[string]bool{}
		}
		locations[id][location] = true
	}
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if p != root && (skipScanDir(d.Name()) || skip(p, true)) {
					return filepath.SkipDir
				}
				return nil
			}
			if skip(p, false) {
				return nil
			}
			if rel, err := filepath.Rel(moduleRoot, p); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				add(codemap.FileID(rel), p)
			}
			return nil
		})
		symbols, _, err := codemap.ScanWithOptions(root, codemap.ScanOptions{IDBase: moduleRoot, Skip: skip})
		if err != nil {
			continue
		}
		for _, s := range symbols {
			if strings.HasPrefix(s.TraceID, "CODE-") {
				add(s.TraceID, filepath.Join(sourceRootDir(root), filepath.FromSlash(s.Path))+":"+strconv.Itoa(s.Line))
			}
		}
	}
	index := make(map[string]int, len(locations))
	for id, locs := range locations {
		index[id] = len(locs)
	}
	return index
}

// TRLC-LINKS: REQ-EMG-057
func isRegularFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}
