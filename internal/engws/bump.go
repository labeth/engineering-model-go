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
func rejectedKeyWarnings(comp engmodel.CompositionResult, consumerDir string, target BumpTarget) []string {
	engine := pinnedEngine(consumerDir)
	if comp.Root == nil || !engineRejectsLifecycle(engine) {
		return nil
	}
	for _, child := range comp.Root.Children {
		if child.Dependency == target.Alias && child.Bundle.Manifest.Lifecycle != nil {
			return []string{fmt.Sprintf("%s %s declares lifecycle in its engmod.yml, which .github/engmod-check (engineering-model-go %s) rejects: its check fails with composition.load_failed until you bump that engine to %s or later", target.Alias, target.Version, engine, lifecycleEngine)}
		}
	}
	return nil
}

// engineRejectsLifecycle reports whether a pinned engine version is valid
// and older than the first engine that reads lifecycle.
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func engineRejectsLifecycle(engine string) bool {
	return engine != "" && semver.IsValid(engine) && semver.Compare(engine, lifecycleEngine) < 0
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

// BumpTarget names what Bump moves: the dependency Alias of checkout Repo
// to Version.
// TRLC-LINKS: REQ-EMG-100
type BumpTarget struct{ Repo, Alias, Version string }

// Bump sets the version of dependency target.Alias in root/<repo>/engmod.yml by
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
func Bump(root string, target BumpTarget, _ BumpOptions) (BumpResult, error) {
	run, err := newBumpRun(root, target)
	if err != nil {
		return BumpResult{}, err
	}
	if err := run.prepare(); err != nil {
		return run.result, err
	}
	return run.execute()
}

// bumpFile is a file as it was before the bump: its content, or the error
// reading it (a missing file is removed again on restore).
type bumpFile struct {
	path string
	data []byte
	err  error
}

// bumpRead records the file at path.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func bumpRead(path string) bumpFile {
	data, err := os.ReadFile(path)
	return bumpFile{path: path, data: data, err: err}
}

// restore writes the recorded content back, or removes the file when it did
// not exist.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (f bumpFile) restore() error {
	if f.err == nil {
		return writeFileAtomic(f.path, f.data)
	}
	if rmErr := os.Remove(f.path); rmErr != nil && !os.IsNotExist(rmErr) {
		return rmErr
	}
	return nil
}

// bumpRun is the state of one Bump: what is bumped, what was found and the
// files to restore on failure.
type bumpRun struct {
	target       BumpTarget
	checkouts    []Checkout
	consumer     *Checkout
	dep          *model.ManifestDependency
	result       BumpResult
	manifestPath string
	original     []byte
	edited       []byte
	goMod, goSum bumpFile
	lock         bumpFile
}

// newBumpRun finds the consumer checkout and its dependency alias.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func newBumpRun(root string, target BumpTarget) (*bumpRun, error) {
	if !semver.IsValid(target.Version) || semver.Canonical(target.Version) != target.Version {
		return nil, fmt.Errorf("version %q must be an exact semantic version such as v1.2.3", target.Version)
	}
	checkouts, err := Scan(root)
	if err != nil {
		return nil, err
	}
	consumer := bumpFindConsumer(checkouts, target.Repo)
	if consumer == nil {
		return nil, fmt.Errorf("no checkout %q with engmod.yml under %s", target.Repo, root)
	}
	dep := bumpFindDependency(consumer, target.Alias)
	if dep == nil {
		return nil, fmt.Errorf("%s/engmod.yml has no dependency alias %q", target.Repo, target.Alias)
	}
	return &bumpRun{target: target, checkouts: checkouts, consumer: consumer, dep: dep,
		result: BumpResult{Module: dep.Path, From: dep.Version, To: target.Version}}, nil
}

// bumpFindConsumer returns the last checkout named repo, or nil.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func bumpFindConsumer(checkouts []Checkout, repo string) *Checkout {
	var consumer *Checkout
	for i := range checkouts {
		if checkouts[i].Name == repo {
			consumer = &checkouts[i]
		}
	}
	return consumer
}

// bumpFindDependency returns the last dependency of the consumer with the
// alias, or nil.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func bumpFindDependency(consumer *Checkout, alias string) *model.ManifestDependency {
	var dep *model.ManifestDependency
	for i := range consumer.Manifest.Dependencies {
		if consumer.Manifest.Dependencies[i].Alias == alias {
			dep = &consumer.Manifest.Dependencies[i]
		}
	}
	return dep
}

// prepare edits the manifest in memory, decides whether go.mod is updated
// and records the files that change.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) prepare() error {
	r.manifestPath = filepath.Join(r.consumer.Dir, "engmod.yml")
	data, err := os.ReadFile(r.manifestPath)
	if err != nil {
		return err
	}
	r.original = data
	r.edited, err = setDependencyVersion(data, r.target.Alias, r.target.Version)
	if err != nil {
		return fmt.Errorf("%s: %w", r.manifestPath, err)
	}
	if err := r.resolveGoModule(); err != nil {
		return err
	}
	r.lock = bumpRead(filepath.Join(r.consumer.Dir, engmodel.CompositionLockFileName))
	r.goMod = bumpRead(filepath.Join(r.consumer.Dir, "go.mod"))
	r.goSum = bumpRead(filepath.Join(r.consumer.Dir, "go.sum"))
	return nil
}

// familyGoModule returns the Go module of the dependency's repository
// family: from the go.mod of the dependency's checkout, when it is in the
// workspace.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) familyGoModule() string {
	found := ""
	if r.consumer.GoModule == "" {
		return found
	}
	for _, c := range r.checkouts {
		if c.Manifest.Module.Path == strings.TrimSpace(r.dep.Path) && c.GoModule != "" {
			found = c.GoModule
		}
	}
	return found
}

// resolveGoModule sets result.GoModule to the Go module whose require is
// updated, or "" when none is.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) resolveGoModule() error {
	r.result.GoModule = r.familyGoModule()
	if r.result.GoModule == "" {
		return nil
	}
	// go get takes the path as an argument and has no "--": a path
	// starting with "-" would be read as a flag.
	if err := gomodule.CheckPath(r.result.GoModule); err != nil {
		return fmt.Errorf("invalid Go module path %q in the go.mod of %s's checkout: %v", r.result.GoModule, r.dep.Path, err)
	}
	goMod, err := os.ReadFile(filepath.Join(r.consumer.Dir, "go.mod"))
	if err != nil {
		return err
	}
	if !requires(goMod, r.result.GoModule) {
		// Only update an existing require; adding one is a code decision.
		r.result.GoModule = ""
	}
	return nil
}

// execute writes the manifest, updates go.mod, regenerates the lock and
// re-pins the links; a failure restores every file.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) execute() (BumpResult, error) {
	if err := writeFileAtomic(r.manifestPath, r.edited); err != nil {
		return r.result, err
	}
	if err := r.goGet(); err != nil {
		return r.rollback(err)
	}
	// Lock: compose against the registry only, so the lock pins published
	// content. GenerateCompositionFromFile writes engmod.lock.yml when the
	// composition has no errors.
	restore := setEnv("ENGMOD_WORK", "off")
	defer restore()
	comp, err := r.regenerateLock()
	if err != nil {
		return r.rollback(err)
	}
	r.result.LockWritten = true
	r.result.Warnings = rejectedKeyWarnings(comp, r.consumer.Dir, r.target)
	return r.repinLinks(comp)
}

// goGet updates the require and go.sum, so GOWORK=off go build and go test
// work after the bump. It runs outside any go.work, with the user's GOPROXY,
// GOFLAGS and credentials from the environment.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) goGet() error {
	if r.result.GoModule == "" {
		return nil
	}
	cmd := exec.Command("go", "get", r.result.GoModule+"@"+r.target.Version)
	cmd.Dir = r.consumer.Dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go get %s@%s: %v: %s", r.result.GoModule, r.target.Version, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// regenerateLock composes the consumer against the registry only, which
// writes the lock when the composition has no errors.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) regenerateLock() (engmodel.CompositionResult, error) {
	comp, err := engmodel.GenerateCompositionFromFile(r.manifestPath)
	r.result.Diagnostics = comp.Diagnostics
	if err != nil {
		return comp, fmt.Errorf("regenerate lock: %w", err)
	}
	if hasCompositionErrors(comp.Diagnostics) {
		return comp, fmt.Errorf("regenerate lock: %s does not compose against %s (%d errors above); adapt it to that version first, then bump again",
			r.target.Repo, depModuleAt(r.dep.Path, r.target.Version), countErrors(comp.Diagnostics))
	}
	if dir, local := r.localChild(comp); local {
		// The engine applied a workspace replacement despite
		// ENGMOD_WORK=off (an engine without REQ-EMG-063): the lock
		// would pin local content.
		r.result.LockWritten = true
		return comp, fmt.Errorf("regenerate lock: %s resolved from local %s, not the registry; this engine ignores ENGMOD_WORK=off, run engws sync without that checkout local and retry", r.target.Alias, dir)
	}
	return comp, nil
}

// localChild returns the directory of the first dependency alias child that
// was not fetched from the module cache.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) localChild(comp engmodel.CompositionResult) (string, bool) {
	if comp.Root == nil {
		return "", false
	}
	for _, child := range comp.Root.Children {
		if child.Dependency == r.target.Alias && !underModuleCache(child.Dir) {
			return child.Dir, true
		}
	}
	return "", false
}

// repinLinks compares the requirement links on the alias between versions
// and moves the pin of each link whose item means the same.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) repinLinks(comp engmodel.CompositionResult) (BumpResult, error) {
	if comp.Root == nil || len(comp.Root.Bundle.ReqLinks.ReqLinks) == 0 {
		return r.result, nil
	}
	r.result.Links = compareBumpLinks(comp.Root.Bundle, engmodel.LinkDependencies(comp.Root), r.target)
	r.classifyLinks()
	if len(r.result.Repinned) == 0 {
		return r.result, nil
	}
	if err := r.writeRepins(comp.Root.Bundle.ReqLinksPath); err != nil {
		return r.rollback(err)
	}
	return r.result, nil
}

// classifyLinks lists the moved links and flags the links left for review.
// TRLC-LINKS: REQ-EMG-100
func (r *bumpRun) classifyLinks() {
	for _, l := range r.result.Links {
		switch l.Outcome {
		case BumpLinkMoved:
			r.result.Repinned = append(r.result.Repinned, l.ID)
		case BumpLinkReview, BumpLinkBroken:
			r.result.NeedReview = true
		}
	}
}

// writeRepins sets pinned.version of every moved link in the link file.
// TRLC-LINKS: REQ-EMG-100
func (r *bumpRun) writeRepins(linksPath string) error {
	data, err := os.ReadFile(linksPath)
	if err != nil {
		return err
	}
	edited := data
	for _, id := range r.result.Repinned {
		if edited, err = yamledit.SetKey(edited, yamledit.KeyAt{Find: yamledit.FindListItem(yamledit.ListItem{Keys: []string{"reqLinks"}, IDKey: "id", ID: id, Sub: []string{"pinned"}}), Key: "version"}, yamledit.Scalar(r.target.Version)); err != nil {
			return fmt.Errorf("%s: link %s: %w", linksPath, id, err)
		}
	}
	return writeFileAtomic(linksPath, edited)
}

// rollback restores engmod.yml, go.mod, go.sum and the lock together, so
// they change together or not at all, and returns cause with the outcome.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) rollback(cause error) (BumpResult, error) {
	if err := writeFileAtomic(r.manifestPath, r.original); err != nil {
		return r.result, fmt.Errorf("%w; restoring engmod.yml also failed: %v", cause, err)
	}
	if err := r.restoreGoFiles(cause); err != nil {
		return r.result, err
	}
	if err := r.restoreLock(cause); err != nil {
		return r.result, err
	}
	r.result.LockWritten = false
	return r.result, fmt.Errorf("%w; engmod.yml, go.mod, go.sum and the lock left unchanged", cause)
}

// restoreGoFiles restores go.mod and go.sum when the bump changed them.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) restoreGoFiles(cause error) error {
	if r.result.GoModule == "" || r.goMod.err != nil {
		return nil
	}
	if err := r.goMod.restore(); err != nil {
		return fmt.Errorf("%w; restoring go.mod also failed: %v", cause, err)
	}
	if err := r.goSum.restore(); err != nil {
		return fmt.Errorf("%w; restoring go.sum also failed: %v", cause, err)
	}
	return nil
}

// restoreLock restores the lock when the bump wrote it.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func (r *bumpRun) restoreLock(cause error) error {
	if !r.result.LockWritten {
		return nil
	}
	if err := r.lock.restore(); err != nil {
		return fmt.Errorf("%w; restoring %s also failed: %v", cause, engmodel.CompositionLockFileName, err)
	}
	return nil
}

// bumpComparer compares the links of a consumer on one alias.
type bumpComparer struct {
	consumer model.Bundle
	target   BumpTarget
	after    map[string]reqlinks.Item
	cache    linkreview.VersionCache
	options  composition.ResolutionOptions
}

// compareBumpLinks compares each link of consumer on the alias of target
// between the item at its pinned version (resolved with the workspace off,
// module cache else registry) and at the target version (deps, the
// composition at the new pin).
// TRLC-LINKS: REQ-EMG-100
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-MISMATCH
func compareBumpLinks(consumer model.Bundle, deps []reqlinks.Dependency, target BumpTarget) []BumpLink {
	cmp := bumpComparer{consumer: consumer, target: target, cache: linkreview.VersionCache{},
		// Lock check verifies each cached module against the registry's manifest
		// digest, so a re-staged version is refetched.
		options: composition.ResolutionOptions{Lock: composition.LockModeCheck}}
	cmp.after = bumpPublishedItems(deps, target.Alias)
	var out []BumpLink
	for _, link := range consumer.ReqLinks.ReqLinks {
		a, id, err := model.ParseQualifiedReference(strings.TrimSpace(link.Child))
		if err != nil || a != target.Alias {
			continue
		}
		out = append(out, cmp.link(link, id))
	}
	return out
}

// bumpPublishedItems returns the published items of dependency alias.
// TRLC-LINKS: REQ-EMG-100
func bumpPublishedItems(deps []reqlinks.Dependency, alias string) map[string]reqlinks.Item {
	var items map[string]reqlinks.Item
	for _, d := range deps {
		if d.Alias == alias {
			items = reqlinks.PublishedItems(d.Bundle, d.Publications...)
		}
	}
	return items
}

// link compares one link, whose child item is id on the alias.
// TRLC-LINKS: REQ-EMG-100
func (c bumpComparer) link(link model.ReqLink, id string) BumpLink {
	l := BumpLink{ID: strings.TrimSpace(link.ID), Child: strings.TrimSpace(link.Child), From: strings.TrimSpace(link.Pinned.Version), To: c.target.Version}
	if item, ok := c.after[id]; ok {
		l.After = &item
	}
	switch {
	case l.From == c.target.Version:
		l.Outcome = BumpLinkCurrent
	case l.After == nil:
		l.Outcome, l.Note = BumpLinkBroken, fmt.Sprintf("%s does not exist at %s", l.Child, c.target.Version)
	default:
		c.compareVersions(&l, id)
	}
	return l
}

// compareVersions sets the outcome of a link whose item exists at the new
// version and is pinned to another one.
// TRLC-LINKS: REQ-EMG-100
func (c bumpComparer) compareVersions(l *BumpLink, id string) {
	var note string
	l.Before, note = c.cache.Item(c.consumer, linkreview.ItemRef{Alias: c.target.Alias, ID: id, Version: l.From}, c.options)
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
	out, err := yamledit.SetScalarInPlace(data, yamledit.KeyAt{Find: yamledit.FindListItem(yamledit.ListItem{Keys: []string{"dependencies"}, IDKey: "alias", ID: alias}), Key: "version"}, version)
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
