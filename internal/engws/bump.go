// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// BumpResult describes what bump changed.
type BumpResult struct {
	Module      string // dependency model module path
	From, To    string
	GoModule    string // Go module required, empty when not changed
	LockWritten bool
	Diagnostics []validate.Diagnostic
}

// Bump sets the version of dependency alias in root/<repo>/engmod.yml by
// editing only that YAML line, updates the go.mod require and go.sum of the
// Go module of the same repository family (go get with GOWORK=off) when the
// dependency's checkout has go.mod, and regenerates engmod.lock.yml with the
// workspace off and the module cache refreshed. Any failure restores
// engmod.yml, go.mod and go.sum.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func Bump(root, repo, alias, version string) (BumpResult, error) {
	if !semver.IsValid(version) || semver.Canonical(version) != version {
		return BumpResult{}, fmt.Errorf("version %q must be an exact semantic version such as v1.2.3", version)
	}
	checkouts, err := Scan(root)
	if err != nil {
		return BumpResult{}, err
	}
	var consumer *Checkout
	for i := range checkouts {
		if checkouts[i].Name == repo {
			consumer = &checkouts[i]
		}
	}
	if consumer == nil {
		return BumpResult{}, fmt.Errorf("no checkout %q with engmod.yml under %s", repo, root)
	}
	var dep *model.ManifestDependency
	for i := range consumer.Manifest.Dependencies {
		if consumer.Manifest.Dependencies[i].Alias == alias {
			dep = &consumer.Manifest.Dependencies[i]
		}
	}
	if dep == nil {
		return BumpResult{}, fmt.Errorf("%s/engmod.yml has no dependency alias %q", repo, alias)
	}
	result := BumpResult{Module: dep.Path, From: dep.Version, To: version}

	manifestPath := filepath.Join(consumer.Dir, "engmod.yml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return result, err
	}
	edited, err := setDependencyVersion(data, alias, version)
	if err != nil {
		return result, fmt.Errorf("%s: %w", manifestPath, err)
	}

	// The Go module of the dependency's repository family: from the go.mod of
	// the dependency's checkout, when it is in the workspace.
	if consumer.GoModule != "" {
		for _, c := range checkouts {
			if c.Manifest.Module.Path == strings.TrimSpace(dep.Path) && c.GoModule != "" {
				result.GoModule = c.GoModule
			}
		}
	}
	if result.GoModule != "" {
		goMod, err := os.ReadFile(filepath.Join(consumer.Dir, "go.mod"))
		if err != nil {
			return result, err
		}
		if !requires(goMod, result.GoModule) {
			// Only update an existing require; adding one is a code decision.
			result.GoModule = ""
		}
	}

	goModPath := filepath.Join(consumer.Dir, "go.mod")
	goSumPath := filepath.Join(consumer.Dir, "go.sum")
	goModBefore, goModErr := os.ReadFile(goModPath)
	goSumBefore, goSumErr := os.ReadFile(goSumPath)
	rollback := func(cause error) (BumpResult, error) {
		// engmod.yml, go.mod, go.sum and the lock change together or not at all.
		if err := writeFileAtomic(manifestPath, data); err != nil {
			return result, fmt.Errorf("%w; restoring engmod.yml also failed: %v", cause, err)
		}
		if result.GoModule != "" && goModErr == nil {
			if err := writeFileAtomic(goModPath, goModBefore); err != nil {
				return result, fmt.Errorf("%w; restoring go.mod also failed: %v", cause, err)
			}
			var err error
			if goSumErr == nil {
				err = writeFileAtomic(goSumPath, goSumBefore)
			} else if rmErr := os.Remove(goSumPath); rmErr != nil && !os.IsNotExist(rmErr) {
				err = rmErr
			}
			if err != nil {
				return result, fmt.Errorf("%w; restoring go.sum also failed: %v", cause, err)
			}
		}
		result.LockWritten = false
		return result, fmt.Errorf("%w; engmod.yml, go.mod and go.sum left unchanged", cause)
	}
	if err := writeFileAtomic(manifestPath, edited); err != nil {
		return result, err
	}
	if result.GoModule != "" {
		// go get updates the require and go.sum, so GOWORK=off go build and
		// go test work after the bump. It runs outside any go.work, with the
		// user's GOPROXY, GOFLAGS and credentials from the environment.
		cmd := exec.Command("go", "get", result.GoModule+"@"+version)
		cmd.Dir = consumer.Dir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			return rollback(fmt.Errorf("go get %s@%s: %v: %s", result.GoModule, version, err, strings.TrimSpace(string(out))))
		}
	}

	// Lock: compose against the registry only, so the lock pins published
	// content. GenerateCompositionFromFile writes engmod.lock.yml when the
	// composition has no errors.
	lockBefore, lockErr := os.ReadFile(filepath.Join(consumer.Dir, engmodel.CompositionLockFileName))
	restore := setEnv("ENGMOD_WORK", "off")
	defer restore()
	comp, err := engmodel.GenerateCompositionFromFile(manifestPath)
	result.Diagnostics = comp.Diagnostics
	if err != nil {
		return rollback(fmt.Errorf("regenerate lock: %w", err))
	}
	if validate.HasErrors(comp.Diagnostics) {
		return rollback(fmt.Errorf("regenerate lock: composition has errors, %s not written (is %s@%s published? an engine without ENGMOD_WORK support also applies the root %s)",
			engmodel.CompositionLockFileName, dep.Path, version, model.WorkspaceFileName))
	}
	if comp.Root != nil {
		for _, child := range comp.Root.Children {
			if child.Dependency == alias && !underModuleCache(child.Dir) {
				// The engine applied a workspace replacement despite
				// ENGMOD_WORK=off (an engine without REQ-EMG-063): the lock
				// would pin local content.
				_ = os.Remove(filepath.Join(consumer.Dir, engmodel.CompositionLockFileName))
				if lockErr == nil {
					_ = writeFileAtomic(filepath.Join(consumer.Dir, engmodel.CompositionLockFileName), lockBefore)
				}
				return rollback(fmt.Errorf("regenerate lock: %s resolved from local %s, not the registry; this engine ignores ENGMOD_WORK=off, run engws sync without that checkout local and retry", alias, child.Dir))
			}
		}
	}
	result.LockWritten = true
	return result, nil
}

// underModuleCache reports whether dir lies in an .engmod/modules cache,
// that is, the dependency was fetched from the registry.
// TRLC-LINKS: REQ-EMG-068
func underModuleCache(dir string) bool {
	sep := string(filepath.Separator)
	return strings.Contains(filepath.Clean(dir)+sep, sep+".engmod"+sep+"modules"+sep)
}

// requires reports whether go.mod data has a require for path.
// TRLC-LINKS: REQ-EMG-068
func requires(data []byte, path string) bool {
	file, err := modfile.ParseLax("go.mod", data, nil)
	if err != nil {
		return false
	}
	for _, r := range file.Require {
		if r.Mod.Path == path {
			return true
		}
	}
	return false
}

// setEnv sets an environment variable and returns a function restoring it.
// TRLC-LINKS: REQ-EMG-068
func setEnv(key, value string) func() {
	old, had := os.LookupEnv(key)
	os.Setenv(key, value)
	return func() {
		if had {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	}
}

// setDependencyVersion rewrites the version line of the dependency with the
// given alias and leaves every other byte of the manifest unchanged. It finds
// the line through the YAML node tree, so flow style, quoting and key order
// do not matter, and refuses a value on another line than its key.
// TRLC-LINKS: REQ-EMG-068
func setDependencyVersion(data []byte, alias, version string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("manifest is not a mapping")
	}
	deps := mappingValue(doc.Content[0], "dependencies")
	if deps == nil || deps.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("no dependencies list")
	}
	for _, item := range deps.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		a := mappingValue(item, "alias")
		if a == nil || a.Value != alias {
			continue
		}
		key, value := mappingEntry(item, "version")
		if value == nil {
			return nil, fmt.Errorf("dependency %q has no version", alias)
		}
		if key.Line != value.Line || value.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("dependency %q version is not a single-line scalar", alias)
		}
		return replaceScalarOnLine(data, value.Line, value.Column, value.Value, version)
	}
	return nil, fmt.Errorf("no dependency alias %q", alias)
}

// replaceScalarOnLine replaces the scalar old that starts at (line, column),
// both 1-based, with new, keeping its quotes.
// TRLC-LINKS: REQ-EMG-068
func replaceScalarOnLine(data []byte, line, column int, old, new string) ([]byte, error) {
	lines := bytes.SplitAfter(data, []byte("\n"))
	if line < 1 || line > len(lines) {
		return nil, fmt.Errorf("line %d out of range", line)
	}
	text := string(lines[line-1])
	start := column - 1
	if start < 0 || start >= len(text) {
		return nil, fmt.Errorf("column %d out of range on line %d", column, line)
	}
	if q := text[start]; q == '"' || q == '\'' {
		start++
	}
	if !strings.HasPrefix(text[start:], old) {
		return nil, fmt.Errorf("line %d does not hold %q at column %d", line, old, column)
	}
	lines[line-1] = []byte(text[:start] + new + text[start+len(old):])
	return bytes.Join(lines, nil), nil
}

// mappingValue returns the value node for key in a mapping node.
// TRLC-LINKS: REQ-EMG-068
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	_, value := mappingEntry(node, key)
	return value
}

// mappingEntry returns the key and value nodes for key in a mapping node.
// TRLC-LINKS: REQ-EMG-068
func mappingEntry(node *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i], node.Content[i+1]
		}
	}
	return nil, nil
}
