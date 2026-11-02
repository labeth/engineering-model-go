// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bytes"
	"fmt"
	"github.com/labeth/engineering-model-go/internal/yamledit"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
	"golang.org/x/mod/modfile"
	gomodule "golang.org/x/mod/module"
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
	// Links is the evaluation of every requirement link of the consumer on
	// the bumped alias at the new version (Content: unchanged, suspect or
	// broken; empty when the item is not a published contract item).
	Links []reqlinks.LinkStatus
	// Repinned lists the unchanged links whose pinned.version moved to the
	// new version (decision D8).
	Repinned []string
	// Blocked reports that a suspect or broken link aborted the bump.
	Blocked bool
}

// BumpOptions modify Bump. AllowSuspect writes the bump although a link is
// suspect or broken; the model check then fails until engws review-links.
// TRLC-LINKS: REQ-EMG-100
type BumpOptions struct {
	AllowSuspect bool
}

// NeedsReview reports whether a link evaluated by bump needs a review: its
// item changed (suspect), is gone (broken) or is no published contract item.
// TRLC-LINKS: REQ-EMG-100
func NeedsReview(s reqlinks.LinkStatus) bool {
	return s.Content != reqlinks.StateUnchanged
}

// Bump sets the version of dependency alias in root/<repo>/engmod.yml by
// editing only that YAML line, updates the go.mod require and go.sum of the
// Go module of the same repository family (go get with GOWORK=off) when the
// dependency's checkout has go.mod, and regenerates engmod.lock.yml with the
// workspace off and the module cache refreshed. It then evaluates the
// consumer's requirement links on the alias at the new version: unchanged
// links move pinned.version to the new version, suspect and broken ones keep
// their pin and fingerprint (decision D8), and without opts.AllowSuspect a
// suspect or broken link aborts the bump. Any failure or abort restores
// engmod.yml, go.mod, go.sum, the lock and the requirement link file.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func Bump(root, repo, alias, version string, opts BumpOptions) (BumpResult, error) {
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
		// go get takes the path as an argument and has no "--": a path
		// starting with "-" would be read as a flag.
		if err := gomodule.CheckPath(result.GoModule); err != nil {
			return result, fmt.Errorf("invalid Go module path %q in the go.mod of %s's checkout: %v", result.GoModule, dep.Path, err)
		}
		goMod, err := os.ReadFile(filepath.Join(consumer.Dir, "go.mod"))
		if err != nil {
			return result, err
		}
		if !requires(goMod, result.GoModule) {
			// Only update an existing require; adding one is a code decision.
			result.GoModule = ""
		}
	}

	lockPath := filepath.Join(consumer.Dir, engmodel.CompositionLockFileName)
	lockBefore, lockErr := os.ReadFile(lockPath)
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
		if result.LockWritten {
			var err error
			if lockErr == nil {
				err = writeFileAtomic(lockPath, lockBefore)
			} else if rmErr := os.Remove(lockPath); rmErr != nil && !os.IsNotExist(rmErr) {
				err = rmErr
			}
			if err != nil {
				return result, fmt.Errorf("%w; restoring %s also failed: %v", cause, engmodel.CompositionLockFileName, err)
			}
		}
		result.LockWritten = false
		return result, fmt.Errorf("%w; engmod.yml, go.mod, go.sum and the lock left unchanged", cause)
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
	restore := setEnv("ENGMOD_WORK", "off")
	defer restore()
	comp, err := engmodel.GenerateCompositionFromFile(manifestPath)
	result.Diagnostics = comp.Diagnostics
	if err != nil {
		return rollback(fmt.Errorf("regenerate lock: %w", err))
	}
	if hasCompositionErrors(comp.Diagnostics) {
		return rollback(fmt.Errorf("regenerate lock: %s does not compose against %s (%d errors above); adapt it to that version first, then bump again",
			repo, depModuleAt(dep.Path, version), countErrors(comp.Diagnostics)))
	}
	if comp.Root != nil {
		for _, child := range comp.Root.Children {
			if child.Dependency == alias && !underModuleCache(child.Dir) {
				// The engine applied a workspace replacement despite
				// ENGMOD_WORK=off (an engine without REQ-EMG-063): the lock
				// would pin local content.
				result.LockWritten = true
				return rollback(fmt.Errorf("regenerate lock: %s resolved from local %s, not the registry; this engine ignores ENGMOD_WORK=off, run engws sync without that checkout local and retry", alias, child.Dir))
			}
		}
	}
	result.LockWritten = true

	// Requirement links on the alias, evaluated at the new version.
	if comp.Root == nil || len(comp.Root.Bundle.ReqLinks.ReqLinks) == 0 {
		return result, nil
	}
	result.Links = bumpLinkStatuses(comp.Root.Bundle, engmodel.LinkDependencies(comp.Root), alias, version)
	for _, s := range result.Links {
		if NeedsReview(s) {
			result.Blocked = true
		} else if s.Pinned != version {
			result.Repinned = append(result.Repinned, s.ID)
		}
	}
	if result.Blocked && !opts.AllowSuspect {
		result.Repinned = nil
		return rollback(fmt.Errorf("%s has requirement links that need review at %s (above); review them first, or bump with --allow-suspect and run engws review-links", repo, depModuleAt(dep.Path, version)))
	}
	if len(result.Repinned) > 0 {
		linksPath := comp.Root.Bundle.ReqLinksPath
		data, err := os.ReadFile(linksPath)
		if err != nil {
			return rollback(err)
		}
		edited := data
		for _, id := range result.Repinned {
			if edited, err = yamledit.SetKey(edited, yamledit.FindListItem([]string{"reqLinks"}, "id", id, "pinned"), "version", yamledit.Scalar(version)); err != nil {
				return rollback(fmt.Errorf("%s: link %s: %w", linksPath, id, err))
			}
		}
		if err := writeFileAtomic(linksPath, edited); err != nil {
			return rollback(err)
		}
	}
	return result, nil
}

// bumpLinkStatuses evaluates the consumer's links on alias as if each were
// pinned to version, so Content tells whether the item changed against the
// recorded fingerprint.
// TRLC-LINKS: REQ-EMG-100
func bumpLinkStatuses(consumer model.Bundle, deps []reqlinks.Dependency, alias, version string) []reqlinks.LinkStatus {
	moved := consumer
	moved.ReqLinks.ReqLinks = nil
	pinned := map[string]string{}
	for _, link := range consumer.ReqLinks.ReqLinks {
		if a, _, err := model.ParseQualifiedReference(strings.TrimSpace(link.Child)); err != nil || a != alias {
			continue
		}
		pinned[strings.TrimSpace(link.ID)] = strings.TrimSpace(link.Pinned.Version)
		link.Pinned.Version = version
		moved.ReqLinks.ReqLinks = append(moved.ReqLinks.ReqLinks, link)
	}
	statuses := reqlinks.EvaluateLinks(moved, deps)
	for i := range statuses {
		statuses[i].Pinned = pinned[statuses[i].ID]
		if statuses[i].Content == "" {
			statuses[i].Content = statuses[i].State
		}
	}
	return statuses
}

// hasCompositionErrors reports composition errors other than the link.*
// diagnostics: a suspect or broken link does not stop the lock (bump
// evaluates the links itself).
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100
func hasCompositionErrors(diags []validate.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == validate.SeverityError && !strings.HasPrefix(d.Code, "link.") {
			return true
		}
	}
	return false
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
	deps := yamledit.MappingValue(doc.Content[0], "dependencies")
	if deps == nil || deps.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("no dependencies list")
	}
	for _, item := range deps.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		a := yamledit.MappingValue(item, "alias")
		if a == nil || a.Value != alias {
			continue
		}
		key, value := yamledit.MappingEntry(item, "version")
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

// depModuleAt names a module version without repeating the major suffix.
// TRLC-LINKS: REQ-EMG-068
func depModuleAt(path, version string) string {
	if base, major, ok := strings.Cut(path, "@"); ok && (version == major || strings.HasPrefix(version, major+".")) {
		path = base
	}
	return path + "@" + version
}

// TRLC-LINKS: REQ-EMG-068
func countErrors(diags []validate.Diagnostic) int {
	n := 0
	for _, d := range diags {
		if d.Severity == validate.SeverityError && !strings.HasPrefix(d.Code, "link.") {
			n++
		}
	}
	return n
}
