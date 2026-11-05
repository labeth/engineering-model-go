// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/linkreview"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
	"github.com/labeth/engineering-model-go/internal/yamledit"
	"golang.org/x/mod/modfile"
	gomodule "golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// BumpResult describes what bump changed.
type BumpResult struct {
	Module      string // dependency model module path
	From, To    string
	GoModule    string // Go module required, empty when not changed
	LockWritten bool
	Diagnostics []validate.Diagnostic
	// Links is the outcome of every requirement link of the consumer on the
	// bumped alias, in link order.
	Links []BumpLink
	// Repinned lists the links whose pinned.version moved to the new
	// version because the item means the same at both versions (D23).
	Repinned []string
	// NeedReview reports that a link was left at its pin for review: its
	// item changed in meaning, is gone, or its pinned version cannot be
	// resolved.
	NeedReview bool
	// Warnings are notes the bump prints: a dependency release that uses a
	// manifest key the engine the consumer's own check pins rejects.
	Warnings []string
}

// lifecycleEngine is the first engineering-model-go version that reads
// engmod.yml lifecycle; an older engine rejects the key on load.
const lifecycleEngine = "v0.10.0"

// engineModule is the Go module of this engine.
const engineModule = "github.com/labeth/engineering-model-go"

// pinnedEngine returns the engineering-model-go version the consumer's
// .github/engmod-check go.mod requires, or "" without one.
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func pinnedEngine(consumerDir string) string {
	data, err := os.ReadFile(filepath.Join(consumerDir, ".github", "engmod-check", "go.mod"))
	if err != nil {
		return ""
	}
	file, err := modfile.ParseLax("go.mod", data, nil)
	if err != nil {
		return ""
	}
	for _, r := range file.Require {
		if r.Mod.Path == engineModule {
			return r.Mod.Version
		}
	}
	return ""
}

// rejectedKeyWarnings returns a warning when the dependency alias of comp
// declares lifecycle and the engine of the consumer's check is older than
// v0.10.0: that engine fails to load the dependency (composition.load_failed).
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func rejectedKeyWarnings(comp engmodel.CompositionResult, consumerDir, alias, version string) []string {
	engine := pinnedEngine(consumerDir)
	if comp.Root == nil || engine == "" || !semver.IsValid(engine) || semver.Compare(engine, lifecycleEngine) >= 0 {
		return nil
	}
	for _, child := range comp.Root.Children {
		if child.Dependency == alias && child.Bundle.Manifest.Lifecycle != nil {
			return []string{fmt.Sprintf("%s %s declares lifecycle in its engmod.yml, which .github/engmod-check (engineering-model-go %s) rejects: its check fails with composition.load_failed until you bump that engine to %s or later", alias, version, engine, lifecycleEngine)}
		}
	}
	return nil
}

// Bump link outcomes.
const (
	// BumpLinkMoved: same meaning at both versions; pinned.version moved.
	BumpLinkMoved = "moved"
	// BumpLinkCurrent: the link is already pinned to the new version.
	BumpLinkCurrent = "current"
	// BumpLinkReview: the meaning changed or the pinned version cannot be
	// resolved; the pin is kept for review.
	BumpLinkReview = "review"
	// BumpLinkBroken: the item does not exist at the new version.
	BumpLinkBroken = "broken"
)

// BumpLink is the outcome of one requirement link on the bumped alias: the
// item at its pinned version (Before) and at the new version (After), the
// difference in meaning (Changes) and, for a link left for review, Note.
// TRLC-LINKS: REQ-EMG-100
type BumpLink struct {
	ID, Child, From, To string
	Outcome             string
	Changes             []string
	Note                string
	Before, After       *reqlinks.Item
}

// BumpOptions modify Bump. AllowReview makes a bump that leaves links for
// review succeed (exit 0); the pins are written either way.
// TRLC-LINKS: REQ-EMG-100
type BumpOptions struct {
	AllowReview bool
}

// Bump sets the version of dependency alias in root/<repo>/engmod.yml by
// editing only that YAML line, updates the go.mod require and go.sum of the
// Go module of the same repository family (go get with GOWORK=off) when the
// dependency's checkout has go.mod, and regenerates engmod.lock.yml with the
// workspace off and the module cache refreshed. It then compares each
// requirement link of the consumer on the alias between its pinned version
// and the new one, both published versions resolved with the workspace off
// (decision D23): a link whose item means the same at both
// (reqlinks.SameMeaning) moves pinned.version to the new version; one whose
// meaning changed, whose item is gone or whose pinned version cannot be
// resolved keeps its pin, so the model check reports link.version_mismatch
// (or link.broken) until engws review-links accepts it. Any failure restores
// engmod.yml, go.mod, go.sum, the lock and the requirement link file.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
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
	result.Warnings = rejectedKeyWarnings(comp, consumer.Dir, alias, version)

	// Requirement links on the alias: compared between versions.
	if comp.Root == nil || len(comp.Root.Bundle.ReqLinks.ReqLinks) == 0 {
		return result, nil
	}
	result.Links = compareBumpLinks(comp.Root.Bundle, engmodel.LinkDependencies(comp.Root), alias, version)
	for _, l := range result.Links {
		switch l.Outcome {
		case BumpLinkMoved:
			result.Repinned = append(result.Repinned, l.ID)
		case BumpLinkReview, BumpLinkBroken:
			result.NeedReview = true
		}
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

// compareBumpLinks compares each link of consumer on alias between the
// item at its pinned version (resolved with the workspace off, module cache
// else registry) and at version (deps, the composition at the new pin).
// TRLC-LINKS: REQ-EMG-100
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func compareBumpLinks(consumer model.Bundle, deps []reqlinks.Dependency, alias, version string) []BumpLink {
	var after map[string]reqlinks.Item
	for _, d := range deps {
		if d.Alias == alias {
			after = reqlinks.PublishedItems(d.Bundle, d.Publications...)
		}
	}
	cache := linkreview.VersionCache{}
	// Lock check verifies each cached module against the registry's manifest
	// digest, so a re-staged version is refetched.
	options := composition.ResolutionOptions{Lock: composition.LockModeCheck}
	var out []BumpLink
	for _, link := range consumer.ReqLinks.ReqLinks {
		a, id, err := model.ParseQualifiedReference(strings.TrimSpace(link.Child))
		if err != nil || a != alias {
			continue
		}
		l := BumpLink{ID: strings.TrimSpace(link.ID), Child: strings.TrimSpace(link.Child), From: strings.TrimSpace(link.Pinned.Version), To: version}
		if item, ok := after[id]; ok {
			l.After = &item
		}
		switch {
		case l.From == version:
			l.Outcome = BumpLinkCurrent
		case l.After == nil:
			l.Outcome, l.Note = BumpLinkBroken, fmt.Sprintf("%s does not exist at %s", l.Child, version)
		default:
			var note string
			l.Before, note = cache.Item(consumer, alias, id, l.From, options)
			switch {
			case l.Before == nil:
				l.Outcome, l.Note = BumpLinkReview, fmt.Sprintf("%s: %s", l.Child, strings.TrimPrefix(strings.TrimPrefix(note, "before: "), "version "))
			case reqlinks.SameMeaning(*l.Before, *l.After):
				l.Outcome = BumpLinkMoved
			default:
				l.Outcome, l.Changes = BumpLinkReview, reqlinks.MeaningChanges(*l.Before, *l.After)
				l.Note = l.Child + ": " + strings.Join(l.Changes, ", ")
			}
		}
		out = append(out, l)
	}
	return out
}

// hasCompositionErrors reports composition errors other than the link.*
// diagnostics: a link finding does not stop the lock (bump compares the
// links itself).
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

// setDependencyVersion rewrites the version scalar of the dependency with
// the given alias and leaves every other byte of the manifest unchanged
// (yamledit.SetScalarInPlace: found through the YAML node tree, so flow
// style, quoting and key order do not matter); it refuses a value on
// another line than its key.
// TRLC-LINKS: REQ-EMG-068
func setDependencyVersion(data []byte, alias, version string) ([]byte, error) {
	out, err := yamledit.SetScalarInPlace(data, yamledit.FindListItem([]string{"dependencies"}, "alias", alias), "version", version)
	if err != nil {
		return nil, fmt.Errorf("dependency %q version: %w", alias, err)
	}
	return out, nil
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
