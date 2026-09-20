// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
)

const WorkspaceFileName = "engmod.work.yml"

// LoadWorkspace loads and validates a local model workspace replacement file.
// TRLC-LINKS: REQ-EMG-048, REQ-EMG-049
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
func LoadWorkspace(path string) (WorkspaceDocument, error) {
	var workspace WorkspaceDocument
	if err := decodeYAMLFile(path, &workspace); err != nil {
		return WorkspaceDocument{}, err
	}
	version, err := EffectiveSchemaVersion(workspace.SchemaVersion)
	if err != nil {
		return WorkspaceDocument{}, fmt.Errorf("workspace schema: %w", err)
	}
	workspace.SchemaVersion = version

	seen := make(map[string]bool, len(workspace.Replacements))
	for i, replacement := range workspace.Replacements {
		modulePath := strings.TrimSpace(replacement.Module)
		if err := module.CheckPath(modulePath); err != nil {
			return WorkspaceDocument{}, fmt.Errorf("replacements[%d].module: %w", i, err)
		}
		if seen[modulePath] {
			return WorkspaceDocument{}, fmt.Errorf("replacements[%d].module %q is duplicated", i, modulePath)
		}
		seen[modulePath] = true
		if strings.TrimSpace(replacement.Path) == "" {
			return WorkspaceDocument{}, fmt.Errorf("replacements[%d].path is required", i)
		}
		workspace.Replacements[i].Module = modulePath
	}
	return workspace, nil
}

// FindWorkspace finds the nearest engmod.work.yml at or above start. Like other
// workspace mechanisms, it may intentionally sit above multiple sibling Git
// repositories.
// TRLC-LINKS: REQ-EMG-048
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
func FindWorkspace(start string) (WorkspaceDocument, string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return WorkspaceDocument{}, "", err
	}
	if info, statErr := os.Stat(dir); statErr == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		path := filepath.Join(dir, WorkspaceFileName)
		if _, err := os.Stat(path); err == nil {
			workspace, err := LoadWorkspace(path)
			return workspace, path, err
		} else if !os.IsNotExist(err) {
			return WorkspaceDocument{}, "", fmt.Errorf("stat workspace %s: %w", path, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return WorkspaceDocument{}, "", nil
		}
		dir = parent
	}
}

// ReplacementPath returns the authored local path for a CUE module identity.
// TRLC-LINKS: REQ-EMG-048
func (w WorkspaceDocument) ReplacementPath(modulePath string) (string, bool) {
	for _, replacement := range w.Replacements {
		if replacement.Module == modulePath {
			return strings.TrimSpace(replacement.Path), true
		}
	}
	return "", false
}

// ReadCUEModulePath reads the canonical module identity from cue.mod/module.cue.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
func ReadCUEModulePath(dir string) (string, error) {
	path := filepath.Join(dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	file, err := modfile.Parse(data, path)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", path, err)
	}
	return file.QualifiedModule(), nil
}
