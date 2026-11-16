// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

// Confinement of a resolved dependency to its module root. A dependency's
// manifest is authored by another repository (a registry module or a
// workspace replacement); the paths it names must not make the engine read
// files outside that module. The root model is not confined here: its own
// layout (for example a code source in ../) is the caller's choice.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
	"gopkg.in/yaml.v3"
)

// dependencyPathEscapes reports every path of the dependency at moduleRoot
// that is absolute or resolves, after symlinks, outside moduleRoot: its
// engmod.yml and cue.mod/module.cue, each manifest document reference, each
// inferenceHints runtime source and each inferenceHints exclude glob (which
// must stay relative to the module). It reads only engmod.yml
// (never a referenced file), so no diagnostic can carry the content of a
// file outside the module.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func dependencyPathEscapes(alias, moduleRoot, manifestPath string) []validate.Diagnostic {
	c := &escapeCheck{alias: alias, moduleRoot: moduleRoot, manifestPath: manifestPath}
	c.checkFixedFiles()
	if len(c.diags) > 0 {
		return c.diags
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil // reported by the loader
	}
	var manifest model.ManifestDocument
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&manifest); err != nil {
		return nil // reported by the loader
	}
	c.checkDocuments(manifest.Documents)
	c.checkExcludes(manifest.InferenceHints.Exclude)
	for i, src := range manifest.InferenceHints.RuntimeSources {
		c.check(fmt.Sprintf("inferenceHints.runtimeSources[%d]", i), src)
	}
	return c.diags
}

// escapeCheck collects the paths of a dependency that leave its module.
type escapeCheck struct {
	alias        string
	moduleRoot   string
	manifestPath string
	diags        []validate.Diagnostic
}

// checkFixedFiles checks the files every module has: its engmod.yml and
// cue.mod/module.cue.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func (c *escapeCheck) checkFixedFiles() {
	for field, rel := range map[string]string{"engmod.yml": "engmod.yml", "cue.mod/module.cue": filepath.Join("cue.mod", "module.cue")} {
		if !resolvesWithin(c.moduleRoot, filepath.Join(c.moduleRoot, rel)) {
			c.escape(field, rel)
		}
	}
}

// checkExcludes checks the inferenceHints exclude globs. The dependency's code
// scope is its module directory (ComposedSystem.Dir); exclude globs are
// anchored there and must not name anything outside it.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func (c *escapeCheck) checkExcludes(patterns []string) {
	for i, pattern := range patterns {
		if p := strings.TrimSpace(pattern); p != "" && excludeEscapes(p) {
			c.escape(fmt.Sprintf("inferenceHints.exclude[%d]", i), p)
		}
	}
}

// escape reports a field that leaves the module.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func (c *escapeCheck) escape(field, ref string) {
	c.diags = append(c.diags, errDiag("composition.dependency_path_escape",
		fmt.Sprintf("dependency %q field %s (%q) is absolute or resolves outside the dependency module; it is not loaded", c.alias, field, ref), c.manifestPath))
}

// check reports a field whose path is absolute or leaves the module.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func (c *escapeCheck) check(field, ref string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return
	}
	if filepath.IsAbs(ref) || !resolvesWithin(c.moduleRoot, filepath.Join(c.moduleRoot, ref)) {
		c.escape(field, ref)
	}
}

// checkDocuments checks each document reference of the manifest.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func (c *escapeCheck) checkDocuments(d model.DocumentReferences) {
	for _, ref := range []struct{ field, value string }{
		{"documents.catalog", d.Catalog}, {"documents.requirements", d.Requirements},
		{"documents.architecture", d.Architecture}, {"documents.behavior", d.Behavior},
		{"documents.assurance", d.Assurance}, {"documents.compliance", d.Compliance},
		{"documents.views", d.Views}, {"documents.decisions", d.Decisions},
		{"documents.aviation", d.Aviation}, {"documents.reqLinks", d.ReqLinks},
	} {
		c.check(ref.field, ref.value)
	}
}

// excludeEscapes reports whether an exclude glob is absolute or not local to
// the module.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func excludeEscapes(p string) bool {
	return filepath.IsAbs(p) || strings.HasPrefix(p, "/") || !filepath.IsLocal(filepath.FromSlash(p))
}

// checkModuleZipEntries rejects a module zip with an entry that is neither a
// regular file nor a directory (a symlink, device or pipe). modzip.Unzip
// validates entry names (no "..", no absolute paths) but ignores the mode
// and extracts a symlink entry as a regular file holding the link target;
// a published module carrying one is malformed and is not cached.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func checkModuleZipEntries(zipPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, entry := range reader.File {
		if mode := entry.Mode(); !mode.IsRegular() && !mode.IsDir() {
			return fmt.Errorf("module zip entry %q is not a regular file (mode %v)", entry.Name, mode.Type())
		}
	}
	return nil
}

// resolvesWithin reports whether path, with every symlink resolved, lies in
// root (also symlink-resolved). A path that does not exist is judged by its
// nearest existing ancestor, so a missing file under a symlinked directory
// that points outside is still outside.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
func resolvesWithin(root, path string) bool {
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	rel, err := filepath.Rel(root, resolveExisting(filepath.Clean(path)))
	return err == nil && !leavesRoot(rel)
}

// resolveExisting resolves the symlinks of the nearest existing ancestor of
// path and appends the part below it.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
func resolveExisting(path string) string {
	rest := ""
	for {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			return filepath.Join(resolved, rest)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return path
		}
		rest = filepath.Join(filepath.Base(path), rest)
		path = parent
	}
}

// leavesRoot reports whether a path relative to a root is outside it.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
func leavesRoot(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel)
}
