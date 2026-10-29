// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

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

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
	"gopkg.in/yaml.v3"
)

// dependencyPathEscapes reports every path of the dependency at moduleRoot
// that is absolute or resolves, after symlinks, outside moduleRoot: its
// engmod.yml and cue.mod/module.cue, each manifest document reference and
// each inferenceHints code and runtime source. It reads only engmod.yml
// (never a referenced file), so no diagnostic can carry the content of a
// file outside the module.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func dependencyPathEscapes(alias, moduleRoot, manifestPath string) []validate.Diagnostic {
	var diags []validate.Diagnostic
	escape := func(field, ref string) {
		diags = append(diags, errDiag("composition.dependency_path_escape",
			fmt.Sprintf("dependency %q field %s (%q) is absolute or resolves outside the dependency module; it is not loaded", alias, field, ref), manifestPath))
	}
	for field, rel := range map[string]string{"engmod.yml": "engmod.yml", "cue.mod/module.cue": filepath.Join("cue.mod", "module.cue")} {
		if !resolvesWithin(moduleRoot, filepath.Join(moduleRoot, rel)) {
			escape(field, rel)
		}
	}
	if len(diags) > 0 {
		return diags
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil // reported by the loader
	}
	var manifest model.ManifestDocument
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&manifest); err != nil {
		return nil // reported by the loader
	}
	check := func(field, ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		if filepath.IsAbs(ref) || !resolvesWithin(moduleRoot, filepath.Join(moduleRoot, ref)) {
			escape(field, ref)
		}
	}
	d := manifest.Documents
	for _, ref := range []struct{ field, value string }{
		{"documents.catalog", d.Catalog}, {"documents.requirements", d.Requirements},
		{"documents.architecture", d.Architecture}, {"documents.behavior", d.Behavior},
		{"documents.assurance", d.Assurance}, {"documents.compliance", d.Compliance},
		{"documents.views", d.Views}, {"documents.decisions", d.Decisions},
		{"documents.aviation", d.Aviation},
	} {
		check(ref.field, ref.value)
	}
	for i, src := range manifest.InferenceHints.CodeSources {
		check(fmt.Sprintf("inferenceHints.codeSources[%d]", i), src)
	}
	for i, src := range manifest.InferenceHints.RuntimeSources {
		check(fmt.Sprintf("inferenceHints.runtimeSources[%d]", i), src)
	}
	return diags
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
	path = filepath.Clean(path)
	rest := ""
	for {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = filepath.Join(resolved, rest)
			break
		}
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		rest = filepath.Join(filepath.Base(path), rest)
		path = parent
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
