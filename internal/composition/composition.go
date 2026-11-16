// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

// System-of-systems composition. A system model may reference subsystem models in
// local subdirectories; this resolves the parent->child DAG, enforces the
// workspace boundary and acyclicity, and validates the bindings (without
// modifying any subsystem).
// References are downward-only: a subsystem never knows its parents.

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync/atomic"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistry"
	"cuelang.org/go/mod/module"
	"cuelang.org/go/mod/modzip"
	"github.com/gofrs/flock"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
	"gopkg.in/yaml.v3"
)

// ComposedSystem is a resolved system together with its resolved subsystems.
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, DO-ARCHITECTURE-MODEL
type ComposedSystem struct {
	SubsystemID    string
	Dependency     string
	Publication    string
	ModulePath     string
	Version        string
	Dir            string
	Bundle         model.Bundle
	Requirements   model.RequirementsDocument
	PublishedIDs   map[string]bool
	PublishedFacet map[string]string
	Children       []*ComposedSystem
	// Local is set when the system resolved from a workspace replacement (a
	// checkout), not from the registry.
	Local bool
}

// ImportedEntityProvenance records where each visible imported identifier came
// from without leaking composition metadata into generic target properties.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
type ImportedEntityProvenance struct {
	QualifiedID string `yaml:"qualifiedId"`
	Alias       string `yaml:"alias"`
	ModulePath  string `yaml:"path"`
	Version     string `yaml:"version"`
	SourceModel string `yaml:"sourceModelId"`
	Publication string `yaml:"publicationId"`
	Domain      string `yaml:"domain"`
	SourceID    string `yaml:"sourceId"`
	ResolvedDir string `yaml:"resolvedDirectory"`
}

const CompositionLockFileName = "engmod.lock.yml"

// CompositionLockSchemaVersion is the schemaVersion the engine writes into
// engmod.lock.yml and requires when it reads one (ADR-EMG-072): a lock with
// another version, or none, is composition.lock_schema_outdated.
const CompositionLockSchemaVersion = 1

// CompositionLockDocument is deterministic generated dependency state.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
type CompositionLockDocument struct {
	SchemaVersion int                         `yaml:"schemaVersion"`
	Dependencies  []CompositionLockDependency `yaml:"dependencies"`
}

type CompositionLockDependency struct {
	Alias        string   `yaml:"alias"`
	Path         string   `yaml:"path"`
	Version      string   `yaml:"version"`
	Publications []string `yaml:"publications"`
	Digest       string   `yaml:"digest"`
}

// CompositionResult is the resolved tree plus provenance, lock and diagnostics.
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
type CompositionResult struct {
	Root        *ComposedSystem
	Provenance  []ImportedEntityProvenance
	Lock        CompositionLockDocument
	Diagnostics []validate.Diagnostic
	// Workspace is the workspace root the dependencies resolved from.
	Workspace string
	// cacheNames maps each module cache directory used to module@version.
	cacheNames map[string]string
}

// DisplayDir names a resolved directory for output that may leave the
// machine (MCP responses): a module cache entry as module@version, a
// directory under the workspace root relative to it, anything else by its
// base name. It never returns an absolute path.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-066
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY
func (r CompositionResult) DisplayDir(dir string) string {
	if dir == "" {
		return ""
	}
	dir = filepath.Clean(dir)
	if name, ok := r.cacheNames[dir]; ok {
		return name
	}
	if r.Workspace != "" {
		if rel, err := filepath.Rel(r.Workspace, dir); err == nil && filepath.IsLocal(rel) {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.Base(dir)
}

// HasComposition reports whether the model declares any subsystems.
// TRLC-LINKS: REQ-EMG-016
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func HasComposition(bundle model.Bundle) bool {
	return len(bundle.Architecture.Composition.Subsystems) > 0
}

// CompositionLocalLockFileName receives the resolved lock while workspace
// replacements are active, so the committed lock never pins unpublished content.
const CompositionLocalLockFileName = "engmod.lock.local.yml"

// LockEnv selects the lock mode: "check" compares with the committed lock and
// writes nothing; "write" (or unset) writes the lock.
const LockEnv = "ENGMOD_LOCK"

const (
	// LockModeWrite writes engmod.lock.yml (or the local lock in workspace mode).
	LockModeWrite = "write"
	// LockModeCheck compares the resolved lock with engmod.lock.yml and writes nothing.
	LockModeCheck = "check"
)

// ResolutionOptions overrides the ENGMOD_WORK and ENGMOD_LOCK environment for
// one call. Empty fields fall back to the environment. Workspace is "off" or a
// workspace file path; Lock is "check" or "write".
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
type ResolutionOptions struct {
	Workspace string
	Lock      string
	// ModuleCache is "refresh" to re-fetch every registry dependency into
	// the module cache, or "reuse"; empty falls back to ENGMOD_MODULE_CACHE.
	ModuleCache string
}

// ModuleCacheEnv selects the module cache mode: "refresh" re-fetches every
// registry dependency; "reuse" reuses cached modules offline; unset applies
// the defaults of cacheMode.
const ModuleCacheEnv = "ENGMOD_MODULE_CACHE"

const (
	// ModuleCacheReuse reuses a cached module without contacting the
	// registry. Set explicitly it applies in every lock mode; it is also the
	// default except for lock check mode and lock regeneration (see cacheMode).
	ModuleCacheReuse = "reuse"
	// ModuleCacheRefresh re-fetches every registry dependency.
	ModuleCacheRefresh = "refresh"
	// moduleCacheVerify compares a cached module with the registry's current
	// manifest and re-fetches it when the tag moved; offline it reuses it.
	moduleCacheVerify = "verify"
)

// cacheMode returns how cached registry modules are used. Explicit refresh
// (option or ENGMOD_MODULE_CACHE) re-fetches; explicit reuse reuses the
// cache offline in every lock mode (the MCP server's read tools). Unset:
// lock regeneration - write mode with the workspace off - re-fetches, so a
// regenerated lock never records a stale cached digest, and lock check mode
// (the CLI and CI check path) verifies each cached module against the
// registry (one manifest lookup per dependency), so a moved tag is reported
// as composition.lock_mismatch even with a warm cache.
// TRLC-LINKS: REQ-EMG-064, REQ-EMG-069
func (o ResolutionOptions) cacheMode() string {
	mode := strings.TrimSpace(o.ModuleCache)
	if mode == "" {
		mode = strings.TrimSpace(os.Getenv(ModuleCacheEnv))
	}
	switch mode {
	case ModuleCacheRefresh:
		return ModuleCacheRefresh
	case ModuleCacheReuse:
		return ModuleCacheReuse
	}
	if o.lockMode() == LockModeCheck {
		return moduleCacheVerify
	}
	workspace := strings.TrimSpace(o.Workspace)
	if workspace == "" {
		workspace = strings.TrimSpace(os.Getenv(model.WorkspaceEnv))
	}
	if workspace == model.WorkspaceOff {
		return ModuleCacheRefresh
	}
	return ModuleCacheReuse
}

// lockMode returns the effective lock mode for the options.
// TRLC-LINKS: REQ-EMG-064
func (o ResolutionOptions) lockMode() string {
	mode := strings.TrimSpace(o.Lock)
	if mode == "" {
		mode = strings.TrimSpace(os.Getenv(LockEnv))
	}
	if mode == LockModeCheck {
		return LockModeCheck
	}
	return LockModeWrite
}

// compositionRuns counts composition resolutions; tests use it to prove that a
// generation call composes once.
var compositionRuns atomic.Int64

// ResolutionCount returns how many composition resolutions this process has
// run. Engine tests use it to prove that one generation call composes once.
// TRLC-LINKS: REQ-EMG-066
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func ResolutionCount() int64 {
	return compositionRuns.Load()
}

// GenerateCompositionFromFile resolves the system-of-systems rooted at engmod.yml
// with the workspace and lock modes taken from ENGMOD_WORK and ENGMOD_LOCK.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateCompositionFromFile(manifestPath string) (CompositionResult, error) {
	return GenerateCompositionFromFileWith(manifestPath, ResolutionOptions{})
}

// GenerateCompositionFromFileWith resolves the system-of-systems rooted at
// engmod.yml under explicit workspace and lock modes. Lock policy: while
// workspace replacements are active, engmod.lock.yml is never touched and the
// resolved lock is written to engmod.lock.local.yml; in check mode nothing is
// written and, without replacements, the resolved lock is compared with the
// committed one (composition.lock_missing, lock_invalid, lock_mismatch);
// otherwise engmod.lock.yml is written.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064, REQ-EMG-066
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateCompositionFromFileWith(manifestPath string, options ResolutionOptions) (CompositionResult, error) {
	return generateComposition(manifestPath, nil, options)
}

// GenerateCompositionForBundle resolves the composition of an already loaded
// bundle: the dependencies are resolved from bundle.ManifestPath's location
// (workspace, module cache, lock) as in GenerateCompositionFromFileWith, but
// the root system - its manifest dependencies, composition, contract and
// every alias::ID reference - is the given bundle, not the files on disk. A
// caller validating edited content in memory thus gets the diagnostics of
// that content.
// TRLC-LINKS: REQ-EMG-065, REQ-EMG-066, REQ-EMG-069
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func GenerateCompositionForBundle(bundle model.Bundle, options ResolutionOptions) (CompositionResult, error) {
	return generateComposition(bundle.ManifestPath, &bundle, options)
}

// generateComposition resolves the composition rooted at manifestPath, with
// rootBundle (when set) as the root system instead of the manifest's files.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064, REQ-EMG-066, REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func generateComposition(manifestPath string, rootBundle *model.Bundle, options ResolutionOptions) (CompositionResult, error) {
	if filepath.Base(filepath.Clean(manifestPath)) != "engmod.yml" {
		return CompositionResult{}, fmt.Errorf("engmod.yml is the only supported composition entry point")
	}
	absTop, err := filepath.Abs(manifestPath)
	if err != nil {
		return CompositionResult{}, err
	}
	compositionRuns.Add(1)
	resolver, err := newCompositionResolver(absTop, rootBundle, options)
	if err != nil {
		return CompositionResult{}, err
	}
	res := resolver.resolveRoot(absTop)
	if hasCompositionErrors(res.Diagnostics) {
		return res, nil
	}
	// Requirement link diagnostics need the resolved dependencies but do not
	// make the resolution itself invalid: a suspect link still writes the lock.
	var linkDiags []validate.Diagnostic
	if res.Root != nil {
		linkDiags = LinkDiagnostics(res.Root.Bundle, LinkDependencies(res.Root))
	}
	replaced := len(resolver.workspaceDocument.Replacements) > 0
	if err := res.settleLock(filepath.Dir(absTop), replaced, options.lockMode()); err != nil {
		return res, err
	}
	res.Diagnostics = append(res.Diagnostics, linkDiags...)
	return res, nil
}

// newCompositionResolver finds the workspace of the manifest and returns the
// resolver of its dependencies.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064, REQ-EMG-066, REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func newCompositionResolver(absTop string, rootBundle *model.Bundle, options ResolutionOptions) (*compositionResolver, error) {
	workspaceDocument, workspacePath, err := model.FindWorkspaceWith(absTop, options.Workspace)
	if err != nil {
		return nil, fmt.Errorf("load model workspace: %w", err)
	}
	workspace := filepath.Dir(absTop)
	if workspacePath != "" {
		workspace = filepath.Dir(workspacePath)
	}
	return &compositionResolver{
		workspace:         workspace,
		workspaceDocument: workspaceDocument,
		cacheMode:         options.cacheMode(),
		fetched:           map[string]string{},
		cacheNames:        map[string]string{},
		rootBundle:        rootBundle,
	}, nil
}

// resolveRoot resolves the system at absTop and its dependencies and returns
// the result with the validation diagnostics of the composed tree.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064, REQ-EMG-066, REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func (r *compositionResolver) resolveRoot(absTop string) CompositionResult {
	resolved := r.resolveSystem(systemRef{manifestPath: absTop}, map[string]bool{}, true)
	res := CompositionResult{
		Root: resolved.system, Provenance: resolved.provenance,
		Lock:        CompositionLockDocument{SchemaVersion: CompositionLockSchemaVersion, Dependencies: resolved.locks},
		Diagnostics: resolved.diags,
		Workspace:   r.workspace, cacheNames: r.cacheNames,
	}
	if resolved.system != nil {
		res.Diagnostics = append(res.Diagnostics, validateComposition(resolved.system, true)...)
		res.Diagnostics = append(res.Diagnostics, TierDiagnostics(resolved.system)...)
	}
	r.nameCachePaths(res.Diagnostics)
	return res
}

// settleLock handles the lock file of a resolution in dir. While workspace
// replacements are active engmod.lock.yml is never touched and the resolved
// lock goes to engmod.lock.local.yml; in check mode nothing is written and,
// without replacements, the resolved lock is compared with the committed one.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064, REQ-EMG-066, REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func (res *CompositionResult) settleLock(dir string, replaced bool, mode string) error {
	if mode == LockModeCheck {
		res.Diagnostics = append(res.Diagnostics, lockCheckDiagnostics(dir, res.Lock, replaced)...)
		return nil
	}
	name := CompositionLockFileName
	if replaced {
		name = CompositionLocalLockFileName
	}
	return WriteCompositionLock(filepath.Join(dir, name), res.Lock)
}

// lockCheckDiagnostics compares the resolved lock with the committed one, or
// warns that the check is skipped while workspace replacements are active.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064, REQ-EMG-066, REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func lockCheckDiagnostics(dir string, lock CompositionLockDocument, replaced bool) []validate.Diagnostic {
	path := filepath.Join(dir, CompositionLockFileName)
	if !replaced {
		return checkCompositionLock(path, lock)
	}
	return []validate.Diagnostic{{
		Code: "composition.lock_check_skipped", Severity: validate.SeverityWarning,
		Message: fmt.Sprintf("workspace replacements active; the committed %s is checked only with %s=off", CompositionLockFileName, model.WorkspaceEnv),
		Path:    path,
	}}
}

// checkCompositionLock compares the resolved lock with the committed lock and
// reports every missing, extra, moved or re-published dependency.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func checkCompositionLock(path string, resolved CompositionLockDocument) []validate.Diagnostic {
	committed, diags := readCommittedLock(path)
	if diags != nil {
		return diags
	}
	byAlias := map[string]CompositionLockDependency{}
	for _, dependency := range committed.Dependencies {
		byAlias[dependency.Alias] = dependency
	}
	for _, got := range resolved.Dependencies {
		var want *CompositionLockDependency
		if locked, ok := byAlias[got.Alias]; ok {
			want = &locked
		}
		delete(byAlias, got.Alias)
		diags = append(diags, lockMismatch(got, want, path)...)
	}
	return append(diags, extraLockDiagnostics(byAlias, path)...)
}

// readCommittedLock reads the committed lock, or returns the diagnostic of a
// lock that is missing, unreadable or at another schema version.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func readCommittedLock(path string) (CompositionLockDocument, []validate.Diagnostic) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CompositionLockDocument{}, []validate.Diagnostic{errDiag("composition.lock_missing", fmt.Sprintf("%s is missing; resolve without %s=check and commit it", filepath.Base(path), LockEnv), path)}
	}
	var committed CompositionLockDocument
	if err := yaml.Unmarshal(data, &committed); err != nil {
		return committed, []validate.Diagnostic{errDiag("composition.lock_invalid", fmt.Sprintf("%s cannot be read: %v", filepath.Base(path), err), path)}
	}
	return committed, lockSchemaDiagnostics(path, committed.SchemaVersion)
}

// lockMismatch reports a resolved dependency that the committed lock (want,
// nil when it has none) does not pin the same way.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func lockMismatch(got CompositionLockDependency, want *CompositionLockDependency, path string) []validate.Diagnostic {
	switch {
	case want == nil:
		return []validate.Diagnostic{errDiag("composition.lock_mismatch", fmt.Sprintf("dependency %q (%s) is missing from the committed lock", got.Alias, moduleAt(got.Path, got.Version)), path)}
	case want.Path != got.Path || want.Version != got.Version:
		return []validate.Diagnostic{errDiag("composition.lock_mismatch", fmt.Sprintf("dependency %q: lock pins %s but the manifest resolves %s", got.Alias, moduleAt(want.Path, want.Version), moduleAt(got.Path, got.Version)), path)}
	case want.Digest != got.Digest:
		return []validate.Diagnostic{errDiag("composition.lock_mismatch", fmt.Sprintf("dependency %q %s: content digest %s differs from the locked %s (tag moved or content changed)", got.Alias, moduleAt(got.Path, got.Version), got.Digest, want.Digest), path)}
	}
	return nil
}

// lockSchemaDiagnostics reports a committed lock whose schemaVersion is not
// the one the engine writes (ADR-EMG-072, REQ-EMG-179): a missing key reads
// as 0. It returns nil for a lock at the current version.
// TRLC-LINKS: REQ-EMG-179
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func lockSchemaDiagnostics(path string, version int) []validate.Diagnostic {
	if version == CompositionLockSchemaVersion {
		return nil
	}
	msg := fmt.Sprintf("%s has schemaVersion %d but the engine writes %d (0 means the key is missing)", filepath.Base(path), version, CompositionLockSchemaVersion)
	d := errDiag("composition.lock_schema_outdated", msg, path)
	d.Remedy = validate.RequiredRemedy(d.Code)
	return []validate.Diagnostic{d}
}

// extraLockDiagnostics reports the committed dependencies the manifest no
// longer resolves, in alias order.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func extraLockDiagnostics(extra map[string]CompositionLockDependency, path string) []validate.Diagnostic {
	aliases := make([]string, 0, len(extra))
	for alias := range extra {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	diags := make([]validate.Diagnostic, 0, len(aliases))
	for _, alias := range aliases {
		diags = append(diags, errDiag("composition.lock_mismatch", fmt.Sprintf("committed lock lists dependency %q that the manifest no longer resolves", alias), path))
	}
	return diags
}

// moduleAt names a module version for messages: the major-qualified path
// example.com/m@v0 at v0.1.0 is example.com/m@v0.1.0.
// TRLC-LINKS: REQ-EMG-064
func moduleAt(path, version string) string {
	path, version = strings.TrimSpace(path), strings.TrimSpace(version)
	if base, major, ok := strings.Cut(path, "@"); ok && (version == major || strings.HasPrefix(version, major+".")) {
		path = base
	}
	return path + "@" + version
}

// WriteCompositionLock writes deterministic generated dependency state.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func WriteCompositionLock(path string, lock CompositionLockDocument) error {
	sort.Slice(lock.Dependencies, func(i, j int) bool {
		return lock.Dependencies[i].Alias < lock.Dependencies[j].Alias
	})
	for i := range lock.Dependencies {
		sort.Strings(lock.Dependencies[i].Publications)
	}
	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("encode composition lock: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// systemRef names a system to resolve: the subsystem that selects it, the
// dependency and publication it comes through, and its manifest.
type systemRef struct {
	subsystemID  string
	dependency   string
	publication  string
	manifestPath string
}

// resolvedSystem is a resolved system with the provenance, locks and
// diagnostics of it and its dependencies. The system is nil when it could not
// be resolved.
type resolvedSystem struct {
	system     *ComposedSystem
	provenance []ImportedEntityProvenance
	locks      []CompositionLockDependency
	diags      []validate.Diagnostic
}

// resolveSystem loads the model and recursively resolves manifest dependencies.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-018, REQ-EMG-035, REQ-EMG-036, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-067
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func (r *compositionResolver) resolveSystem(ref systemRef, ancestry map[string]bool, root bool) resolvedSystem {
	if ancestry[ref.manifestPath] {
		return resolvedSystem{diags: []validate.Diagnostic{{
			Code: "composition.cycle", Severity: validate.SeverityError,
			Message: fmt.Sprintf("subsystem reference cycle detected at %s", ref.manifestPath),
			Path:    ref.manifestPath,
		}}}
	}
	ancestry[ref.manifestPath] = true
	defer delete(ancestry, ref.manifestPath)

	bundle, loadDiags := r.loadBundle(ref.manifestPath, root)
	if loadDiags != nil {
		return resolvedSystem{diags: loadDiags}
	}
	sys := &ComposedSystem{
		SubsystemID: ref.subsystemID, Dependency: ref.dependency, Publication: ref.publication,
		ModulePath: bundle.Manifest.Module.Path, Version: bundle.Manifest.Module.Version,
		Dir: filepath.Dir(ref.manifestPath), Bundle: bundle, Requirements: bundle.Requirements,
	}
	if !root {
		// A dependency's code is its own module directory (workspace checkout or
		// module cache entry), never the consumer's repository around it.
		sys.Bundle.CodeScopeRoot = sys.Dir
	}
	dependencies, ddiags := validateDependencies(bundle.Manifest, ref.manifestPath)
	sr := &systemResolution{
		resolver: r, ref: ref, ancestry: ancestry, root: root, dependencies: dependencies,
		lockedAliases: map[string]bool{}, out: resolvedSystem{system: sys, diags: ddiags},
	}
	for _, sub := range bundle.Architecture.Composition.Subsystems {
		sr.resolveSubsystem(sub)
	}
	sr.out.diags = append(sr.out.diags, validateQualifiedReferences(bundle, sys.Children, ref.manifestPath)...)
	return sr.out
}

// loadBundle loads the model of a system: the root system is the given bundle
// when the resolver has one; a failure is the load_failed diagnostic.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-018, REQ-EMG-035, REQ-EMG-036, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-067
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func (r *compositionResolver) loadBundle(manifestPath string, root bool) (model.Bundle, []validate.Diagnostic) {
	if root && r.rootBundle != nil {
		return *r.rootBundle, nil
	}
	canonical, err := model.LoadCanonicalBundle(manifestPath)
	if err != nil {
		return model.Bundle{}, []validate.Diagnostic{{
			Code: "composition.load_failed", Severity: validate.SeverityError,
			Message: err.Error(), Path: manifestPath,
		}}
	}
	return canonical.Documents(), nil
}

// systemResolution is the state of resolving the subsystems of one system.
type systemResolution struct {
	resolver      *compositionResolver
	ref           systemRef
	ancestry      map[string]bool
	root          bool
	dependencies  map[string]model.ManifestDependency
	lockedAliases map[string]bool
	out           resolvedSystem
}

// resolvedDependency is a manifest dependency located on disk: the manifest
// of its module, the module directory and the content digest.
type resolvedDependency struct {
	dependency model.ManifestDependency
	manifest   string
	dir        string
	digest     string
}

// fail records an error diagnostic at the manifest of the system.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-018, REQ-EMG-035, REQ-EMG-036, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-067
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func (s *systemResolution) fail(code, message string) {
	s.out.diags = append(s.out.diags, errDiag(code, message, s.ref.manifestPath))
}

// resolveSubsystem resolves the dependency a subsystem selects and adopts it
// as a child of the system.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-018, REQ-EMG-035, REQ-EMG-036, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-067
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func (s *systemResolution) resolveSubsystem(sub model.Subsystem) {
	located, ok := s.locate(sub)
	if !ok {
		return
	}
	child := s.resolver.resolveSystem(systemRef{subsystemID: sub.ID, dependency: sub.Dependency, publication: sub.Publication, manifestPath: located.manifest}, s.ancestry, false)
	s.out.diags = append(s.out.diags, child.diags...)
	if child.system != nil {
		s.adopt(sub, located, child)
	}
}

// locate finds the module of the dependency a subsystem selects on disk, with
// the diagnostics of a dependency that is unknown, does not select the
// publication or cannot be resolved.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-018, REQ-EMG-035, REQ-EMG-036, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-067
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func (s *systemResolution) locate(sub model.Subsystem) (resolvedDependency, bool) {
	dependency, ok := s.dependencies[sub.Dependency]
	if !ok {
		s.fail("composition.unknown_dependency", fmt.Sprintf("subsystem %q references unknown dependency alias %q", sub.ID, sub.Dependency))
		return resolvedDependency{}, false
	}
	if !contains(dependency.Publications, sub.Publication) {
		s.fail("composition.publication_not_selected", fmt.Sprintf("subsystem %q publication %q is not selected by dependency %q", sub.ID, sub.Publication, sub.Dependency))
		return resolvedDependency{}, false
	}
	manifest, dir, digest, diags := s.resolver.resolveDependency(dependency, s.ref.manifestPath)
	s.out.diags = append(s.out.diags, diags...)
	return resolvedDependency{dependency: dependency, manifest: manifest, dir: dir, digest: digest}, manifest != ""
}

// adopt checks a resolved child against what the dependency requested and
// publishes, then makes it a child of the system with its provenance and, for
// the root system, the lock entry of the dependency.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-018, REQ-EMG-035, REQ-EMG-036, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-067
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func (s *systemResolution) adopt(sub model.Subsystem, located resolvedDependency, resolved resolvedSystem) {
	child, dependency := resolved.system, located.dependency
	child.Local = s.resolver.replaced(dependency)
	if child.ModulePath != dependency.Path || child.Version != dependency.Version {
		s.out.diags = append(s.out.diags, errDiag("composition.module_identity", fmt.Sprintf("dependency %q requested %s@%s but child manifest declares %s@%s", dependency.Alias, dependency.Path, dependency.Version, child.ModulePath, child.Version), located.manifest))
		return
	}
	s.out.diags = append(s.out.diags, selectedPublicationDiagnostics(child, dependency, located.manifest)...)
	publication, ok := publicationByID(child.Bundle.Manifest.Publications, sub.Publication)
	if !ok {
		s.out.diags = append(s.out.diags, errDiag("composition.unknown_publication", fmt.Sprintf("dependency %q has no publication %q", sub.Dependency, sub.Publication), located.manifest))
		return
	}
	visible, facets, pdiags := validatePublication(child.Bundle, publication)
	s.out.diags = append(s.out.diags, pdiags...)
	child.PublishedIDs, child.PublishedFacet = visible, facets
	s.out.system.Children = append(s.out.system.Children, child)
	s.out.provenance = append(s.out.provenance, publicationProvenance(sub.Dependency, located, publishedChild{child: child, publication: publication})...)
	s.out.provenance = append(s.out.provenance, resolved.provenance...)
	s.out.locks = append(s.out.locks, resolved.locks...)
	if s.root && !s.lockedAliases[dependency.Alias] {
		s.out.locks = append(s.out.locks, CompositionLockDependency{
			Alias: dependency.Alias, Path: dependency.Path, Version: dependency.Version,
			Publications: append([]string(nil), dependency.Publications...), Digest: located.digest,
		})
		s.lockedAliases[dependency.Alias] = true
	}
}

// selectedPublicationDiagnostics validates every publication the dependency
// selects against the child, which must declare it.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-018, REQ-EMG-035, REQ-EMG-036, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-067
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func selectedPublicationDiagnostics(child *ComposedSystem, dependency model.ManifestDependency, manifest string) []validate.Diagnostic {
	var diags []validate.Diagnostic
	for _, selected := range dependency.Publications {
		selectedPublication, exists := publicationByID(child.Bundle.Manifest.Publications, selected)
		if !exists {
			diags = append(diags, errDiag("composition.unknown_publication", fmt.Sprintf("dependency %q selects publication %q which the child does not declare", dependency.Alias, selected), manifest))
			continue
		}
		_, _, selectedDiags := validatePublication(child.Bundle, selectedPublication)
		diags = append(diags, selectedDiags...)
	}
	return diags
}

type compositionResolver struct {
	workspace         string
	workspaceDocument model.WorkspaceDocument
	moduleClient      *modregistry.Client
	cacheMode         string
	// fetched holds the modules refreshed or verified in this run, so a
	// module shared by several dependents is fetched once.
	fetched map[string]string
	// cacheNames maps each module cache directory used in this run to the
	// module@version it holds.
	cacheNames map[string]string
	// rootBundle, when set, is the root system instead of its files on disk.
	rootBundle *model.Bundle
}

// nameCachePaths rewrites diagnostic paths and messages that point into the
// module cache (.engmod/modules/<entry>/file) to module@version:file, so a
// reader sees which dependency version is at fault, not a cache hash.
// TRLC-LINKS: REQ-EMG-066
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func (r *compositionResolver) nameCachePaths(diags []validate.Diagnostic) {
	if len(r.cacheNames) == 0 {
		return
	}
	dirs := make([]string, 0, len(r.cacheNames))
	for dir := range r.cacheNames {
		dirs = append(dirs, dir)
	}
	// Longest first, so no directory is replaced inside a longer one.
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	rename := func(text string) string {
		for _, dir := range dirs {
			name := r.cacheNames[dir]
			text = strings.ReplaceAll(text, dir+string(filepath.Separator), name+":")
			text = strings.ReplaceAll(text, dir, name)
		}
		return text
	}
	for i := range diags {
		diags[i].Path = rename(diags[i].Path)
		diags[i].Message = rename(diags[i].Message)
	}
}

// replaced reports whether a workspace replacement provides dependency.
// TRLC-LINKS: REQ-EMG-097
func (r *compositionResolver) replaced(dependency model.ManifestDependency) bool {
	mv, err := model.ParseExactModuleVersion(dependency.Path, dependency.Version)
	if err != nil {
		return false
	}
	_, ok := r.workspaceDocument.ReplacementPath(mv.Path())
	return ok
}

// resolveDependency resolves an exact manifest dependency to its engmod.yml.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func (r *compositionResolver) resolveDependency(dependency model.ManifestDependency, parentManifest string) (string, string, string, []validate.Diagnostic) {
	mv, err := model.ParseExactModuleVersion(dependency.Path, dependency.Version)
	if err != nil {
		return "", "", "", errAt("composition.invalid_module", fmt.Sprintf("dependency %q: %v", dependency.Alias, err), parentManifest)
	}
	moduleRoot, diags := r.locateModule(dependency, mv, parentManifest)
	if diags != nil {
		return "", "", "", diags
	}
	return verifyModule(moduleLocation{dependency: dependency, modulePath: mv.Path(), root: moduleRoot, parent: parentManifest})
}

// errAt is a one-element list of the error diagnostic.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func errAt(code, msg, path string) []validate.Diagnostic {
	return []validate.Diagnostic{errDiag(code, msg, path)}
}

// locateModule finds the directory of a dependency's module: its workspace
// replacement or the module cache entry, fetched when needed. The directory is
// confined before any file it names is read.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func (r *compositionResolver) locateModule(dependency model.ManifestDependency, mv module.Version, parentManifest string) (string, []validate.Diagnostic) {
	moduleRoot := ""
	if replacement, ok := r.workspaceDocument.ReplacementPath(mv.Path()); ok {
		moduleRoot = filepath.Clean(filepath.Join(r.workspace, replacement))
		if !PathWithin(r.workspace, moduleRoot) {
			return "", errAt("composition.workspace_path_escape", fmt.Sprintf("dependency %q replacement for %q resolves outside workspace", dependency.Alias, mv.Path()), parentManifest)
		}
	} else {
		var err error
		if moduleRoot, err = r.fetchModule(mv); err != nil {
			return "", errAt("composition.module_fetch_failed", fmt.Sprintf("dependency %q module %s: %v", dependency.Alias, mv, err), parentManifest)
		}
	}
	// Confine the dependency before reading any file it names.
	if escapes := dependencyPathEscapes(dependency.Alias, moduleRoot, filepath.Join(moduleRoot, "engmod.yml")); len(escapes) > 0 {
		return "", escapes
	}
	return moduleRoot, nil
}

// moduleLocation is a dependency's module directory with the module path the
// dependency requested and the manifest that requested it.
type moduleLocation struct {
	dependency model.ManifestDependency
	modulePath string
	root       string
	parent     string
}

// verifyModule checks that a module directory holds the requested module and
// returns its manifest, directory and content digest.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func verifyModule(loc moduleLocation) (string, string, string, []validate.Diagnostic) {
	alias := loc.dependency.Alias
	actualModule, err := model.ReadCUEModulePath(loc.root)
	if err != nil {
		return "", "", "", errAt("composition.module_identity", fmt.Sprintf("dependency %q: %v", alias, err), loc.parent)
	}
	if actualModule != loc.modulePath {
		return "", "", "", errAt("composition.module_identity", fmt.Sprintf("dependency %q requested module %q but source declares %q", alias, loc.modulePath, actualModule), loc.parent)
	}
	digest, err := digestModuleDirectory(loc.root)
	if err != nil {
		return "", "", "", errAt("composition.module_digest", fmt.Sprintf("dependency %q: %v", alias, err), loc.parent)
	}
	childManifest := filepath.Join(loc.root, "engmod.yml")
	if _, err := os.Stat(childManifest); err != nil {
		return "", "", "", errAt("composition.missing_manifest", fmt.Sprintf("dependency %q has no engmod.yml at %s", alias, childManifest), loc.parent)
	}
	return childManifest, loc.root, digest, nil
}

// moduleEntry is a module to materialize in the workspace cache: its cache
// directory and whether an entry is already there.
type moduleEntry struct {
	mv        module.Version
	cacheRoot string
	dest      string
	cached    bool
}

// fetchModule downloads and atomically materializes a CUE module in the
// workspace cache. A cached entry is reused offline in reuse mode; in verify
// mode it is reused when the registry's manifest digest equals the one
// recorded at fetch time (or the registry is unreachable) and replaced
// otherwise; in refresh mode it is always replaced.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func (r *compositionResolver) fetchModule(mv module.Version) (string, error) {
	cacheRoot := filepath.Join(r.workspace, ".engmod", "modules")
	dest := filepath.Join(cacheRoot, moduleCacheDirName(mv))
	r.cacheNames[dest] = mv.String()
	if done, ok := r.fetched[dest]; ok {
		return done, nil
	}
	cached := false
	if _, err := os.Stat(dest); err == nil {
		cached = true
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if cached && !r.refetches() {
		return dest, nil
	}
	return r.fetchLocked(moduleEntry{mv: mv, cacheRoot: cacheRoot, dest: dest, cached: cached})
}

// refetches reports whether a cached module is fetched again: in refresh
// and verify mode.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func (r *compositionResolver) refetches() bool {
	return r.cacheMode == ModuleCacheRefresh || r.cacheMode == moduleCacheVerify
}

// fetchLocked fetches the module under the lock of its cache entry, which
// another process may have filled meanwhile.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func (r *compositionResolver) fetchLocked(e moduleEntry) (string, error) {
	unlock, err := lockCacheEntry(e)
	if err != nil {
		return "", err
	}
	defer unlock()
	if _, err := os.Stat(e.dest); err == nil {
		e.cached = true
	}
	if e.cached && !r.refetches() {
		return e.dest, nil
	}
	return r.refresh(e)
}

// lockCacheEntry creates the cache directory and takes the lock of the entry.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func lockCacheEntry(e moduleEntry) (func(), error) {
	if err := os.MkdirAll(e.cacheRoot, 0o755); err != nil {
		return nil, err
	}
	lock := flock.New(e.dest + ".lock")
	if err := lock.Lock(); err != nil {
		return nil, err
	}
	return func() { _ = lock.Unlock() }, nil
}

// refresh replaces the cache entry with the module from the registry, or
// keeps a cached one in verify mode when the registry is unreachable or its
// manifest digest is the recorded one.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func (r *compositionResolver) refresh(e moduleEntry) (string, error) {
	remoteModule, err := r.remoteModule(e.mv)
	if err != nil {
		return r.keepVerified(e, err)
	}
	manifestDigest := string(remoteModule.ManifestDigest())
	digestFile := e.dest + ".manifest-digest"
	if e.cached && r.cacheMode == moduleCacheVerify && recordedDigestMatches(digestFile, manifestDigest) {
		r.fetched[e.dest] = e.dest
		return e.dest, nil
	}
	if err := installModule(remoteModule, e); err != nil {
		return "", err
	}
	_ = os.WriteFile(digestFile, []byte(manifestDigest+"\n"), 0o644)
	r.fetched[e.dest] = e.dest
	return e.dest, nil
}

// keepVerified is the result when the registry cannot be reached: in verify
// mode a cached entry stays, as the committed lock still guards its content;
// otherwise the error stands.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func (r *compositionResolver) keepVerified(e moduleEntry, err error) (string, error) {
	if e.cached && r.cacheMode == moduleCacheVerify {
		return e.dest, nil
	}
	return "", err
}

// remoteModule looks the module up in the registry, creating the registry
// client on first use.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func (r *compositionResolver) remoteModule(mv module.Version) (*modregistry.Module, error) {
	if r.moduleClient == nil {
		resolver, err := modconfig.NewResolver(nil)
		if err != nil {
			return nil, err
		}
		r.moduleClient = modregistry.NewClientWithResolver(resolver)
	}
	return r.moduleClient.GetModule(context.Background(), mv)
}

// recordedDigestMatches reports whether the digest file holds the registry's
// manifest digest.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func recordedDigestMatches(digestFile, manifestDigest string) bool {
	recorded, err := os.ReadFile(digestFile)
	return err == nil && strings.TrimSpace(string(recorded)) == manifestDigest
}

// installModule downloads the module, checks and unpacks it and moves it into
// its cache entry, replacing a stale one.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func installModule(remoteModule *modregistry.Module, e moduleEntry) error {
	zipPath, err := downloadModuleZip(remoteModule, e)
	if err != nil {
		return err
	}
	defer os.Remove(zipPath)
	if err := checkModuleZipEntries(zipPath); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(e.cacheRoot, "."+sanitizeDirName(e.mv.Path())+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	if err := modzip.Unzip(tempDir, e.mv, zipPath); err != nil {
		return err
	}
	if e.cached {
		// Replace the stale entry: move it aside, then put the new one in.
		stale, err := moveAsideStale(e)
		defer os.RemoveAll(stale)
		if err != nil {
			return err
		}
	}
	return moveIntoPlace(tempDir, e.dest)
}

// moveAsideStale moves the cached entry into a new temporary directory and
// returns that directory.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func moveAsideStale(e moduleEntry) (string, error) {
	stale, err := os.MkdirTemp(e.cacheRoot, "."+sanitizeDirName(e.mv.Path())+"-stale-")
	if err != nil {
		return "", err
	}
	return stale, os.Rename(e.dest, filepath.Join(stale, "module"))
}

// moveIntoPlace renames the unpacked module to its cache entry; a rename that
// fails is no error when another process has filled the entry meanwhile.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func moveIntoPlace(tempDir, dest string) error {
	err := os.Rename(tempDir, dest)
	if _, statErr := os.Stat(dest); err != nil && statErr != nil {
		return err
	}
	return nil
}

// downloadModuleZip downloads the module archive into a temporary file in the
// cache and returns its path; the caller removes it.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func downloadModuleZip(remoteModule *modregistry.Module, e moduleEntry) (string, error) {
	zipReader, err := remoteModule.GetZip(context.Background())
	if err != nil {
		return "", err
	}
	zipFile, err := os.CreateTemp(e.cacheRoot, "."+sanitizeDirName(e.mv.Path())+"-*.zip")
	if err != nil {
		_ = zipReader.Close()
		return "", err
	}
	if err := copyAndClose(zipFile, zipReader); err != nil {
		os.Remove(zipFile.Name())
		return "", err
	}
	return zipFile.Name(), nil
}

// copyAndClose copies src into dst and closes both, returning the first of
// the copy, dst close and src close errors.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY, DO-MODEL-MODULE-ARCHIVE, FLOW-ENGINE-INTEGRATIONS
func copyAndClose(dst *os.File, src io.ReadCloser) error {
	_, copyErr := io.Copy(dst, src)
	dstErr := dst.Close()
	srcErr := src.Close()
	for _, err := range []error{copyErr, dstErr, srcErr} {
		if err != nil {
			return err
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049
func moduleCacheDirName(mv module.Version) string {
	sum := sha256.Sum256([]byte(mv.String()))
	return sanitizeDirName(mv.Path()) + "-" + fmt.Sprintf("%x", sum[:8])
}

// PathWithin reports whether path, after resolving symlinks where possible,
// lies inside root (root itself included).
// TRLC-LINKS: REQ-EMG-017, REQ-EMG-049
func PathWithin(root, path string) bool {
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// sanitizeDirName reduces an id to a safe directory name for the .engmod cache.
// TRLC-LINKS: REQ-EMG-047
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func sanitizeDirName(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "subsystem"
	}
	return b.String()
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
func validateDependencies(manifest model.ManifestDocument, path string) (map[string]model.ManifestDependency, []validate.Diagnostic) {
	dependencies := make(map[string]model.ManifestDependency, len(manifest.Dependencies))
	var diags []validate.Diagnostic
	for _, dependency := range manifest.Dependencies {
		alias := strings.TrimSpace(dependency.Alias)
		if _, exists := dependencies[alias]; exists {
			diags = append(diags, errDiag("composition.duplicate_dependency_alias", fmt.Sprintf("dependency alias %q is duplicated", alias), path))
			continue
		}
		diags = append(diags, dependencyProblems(dependency, alias, path)...)
		dependencies[alias] = dependency
	}
	return dependencies, append(diags, duplicatePublications(manifest.Publications, path)...)
}

// duplicatePublications reports each publication id the manifest declares
// more than once.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
func duplicatePublications(publications []model.ManifestPublication, path string) []validate.Diagnostic {
	var diags []validate.Diagnostic
	seenPublications := map[string]bool{}
	for _, publication := range publications {
		if seenPublications[publication.ID] {
			diags = append(diags, errDiag("composition.duplicate_publication", fmt.Sprintf("publication %q is duplicated", publication.ID), path))
		}
		seenPublications[publication.ID] = true
	}
	return diags
}

// dependencyProblems reports a dependency that is not an exact module
// version, selects no publication or selects one twice.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
func dependencyProblems(dependency model.ManifestDependency, alias, path string) []validate.Diagnostic {
	var diags []validate.Diagnostic
	if _, err := model.ParseExactModuleVersion(strings.TrimSpace(dependency.Path), strings.TrimSpace(dependency.Version)); err != nil {
		diags = append(diags, errDiag("composition.invalid_module", fmt.Sprintf("dependency %q: %v", alias, err), path))
	}
	if len(dependency.Publications) == 0 {
		diags = append(diags, errDiag("composition.missing_publication", fmt.Sprintf("dependency %q must select at least one publication", alias), path))
	}
	seenPublications := map[string]bool{}
	for _, publication := range dependency.Publications {
		if seenPublications[publication] {
			diags = append(diags, errDiag("composition.duplicate_dependency_publication", fmt.Sprintf("dependency %q selects publication %q more than once", alias, publication), path))
		}
		seenPublications[publication] = true
	}
	return diags
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func publicationByID(publications []model.ManifestPublication, id string) (model.ManifestPublication, bool) {
	for _, publication := range publications {
		if publication.ID == id {
			return publication, true
		}
	}
	return model.ManifestPublication{}, false
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func validatePublication(bundle model.Bundle, publication model.ManifestPublication) (map[string]bool, map[string]string, []validate.Diagnostic) {
	check := publicationCheck{
		bundle: bundle, publication: publication, owned: domainOwnedIDs(bundle),
		visible: map[string]bool{}, facets: map[string]string{},
	}
	selected := map[string][]string{
		"architecture": publication.Architecture,
		"behavior":     publication.Behavior,
		"assurance":    publication.Assurance,
		"compliance":   publication.Compliance,
		"requirements": publication.Requirements,
		"views":        publication.Views,
	}
	for domain, ids := range selected {
		for _, id := range ids {
			check.expose(domain, id)
		}
	}
	return check.visible, check.facets, check.diags
}

// publicationCheck is the state of validating the ids a publication exposes.
type publicationCheck struct {
	bundle      model.Bundle
	publication model.ManifestPublication
	owned       map[string]map[string]bool
	visible     map[string]bool
	facets      map[string]string
	diags       []validate.Diagnostic
}

// expose makes an id visible when its domain owns it and no other domain
// exposes it already; otherwise it reports the problem.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func (c *publicationCheck) expose(domain, id string) {
	if !c.owned[domain][id] {
		c.diags = append(c.diags, errDiag("composition.publication_unowned_id", fmt.Sprintf("publication %q exposes %s id %q not owned by that domain", c.publication.ID, domain, id), c.bundle.ManifestPath))
		return
	}
	if previous, exists := c.facets[id]; exists {
		c.diags = append(c.diags, errDiag("composition.publication_duplicate_id", fmt.Sprintf("publication %q exposes id %q in both %s and %s", c.publication.ID, id, previous, domain), c.bundle.ManifestPath))
		return
	}
	c.visible[id] = true
	c.facets[id] = domain
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func domainOwnedIDs(bundle model.Bundle) map[string]map[string]bool {
	architecture := map[string]bool{}
	for _, value := range []any{
		bundle.Architecture.AuthoredArchitecture.FunctionalGroups,
		bundle.Architecture.AuthoredArchitecture.FunctionalUnits,
		bundle.Architecture.AuthoredArchitecture.Actors,
		bundle.Architecture.AuthoredArchitecture.ReferencedElements,
		bundle.Architecture.AuthoredArchitecture.Interfaces,
		bundle.Architecture.AuthoredArchitecture.DataObjects,
		bundle.Architecture.AuthoredArchitecture.DeploymentTargets,
		bundle.Architecture.AuthoredArchitecture.HardwareItems,
		bundle.Architecture.AuthoredArchitecture.HardwareInterfaces,
		bundle.Architecture.Contract,
	} {
		collectOwnedIDs(reflect.ValueOf(value), architecture)
	}
	result := map[string]map[string]bool{
		"architecture": architecture,
		"behavior":     {}, "assurance": {}, "compliance": {}, "requirements": {}, "views": {},
	}
	collectOwnedIDs(reflect.ValueOf(bundle.Behavior.Behavior), result["behavior"])
	collectOwnedIDs(reflect.ValueOf(bundle.Assurance.Assurance), result["assurance"])
	collectOwnedIDs(reflect.ValueOf(bundle.Compliance.Compliance), result["compliance"])
	collectOwnedIDs(reflect.ValueOf(bundle.Requirements.Requirements), result["requirements"])
	collectOwnedIDs(reflect.ValueOf(bundle.Views), result["views"])
	return result
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func collectOwnedIDs(value reflect.Value, ids map[string]bool) {
	if !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.Pointer:
		if !value.IsNil() {
			collectOwnedIDs(value.Elem(), ids)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			collectOwnedIDs(value.Index(i), ids)
		}
	case reflect.Struct:
		collectStructIDs(value, ids)
	}
}

// collectStructIDs adds the id fields of a struct and walks its other fields.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func collectStructIDs(value reflect.Value, ids map[string]bool) {
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if strings.Split(valueType.Field(i).Tag.Get("yaml"), ",")[0] != "id" || field.Kind() != reflect.String {
			collectOwnedIDs(field, ids)
		} else if id := strings.TrimSpace(field.String()); id != "" {
			ids[id] = true
		}
	}
}

// publishedChild is a resolved child with the publication it is published through.
type publishedChild struct {
	child       *ComposedSystem
	publication model.ManifestPublication
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
func publicationProvenance(alias string, located resolvedDependency, published publishedChild) []ImportedEntityProvenance {
	var out []ImportedEntityProvenance
	for id, domain := range published.child.PublishedFacet {
		qualified, _ := model.QualifyReference(alias, id)
		out = append(out, ImportedEntityProvenance{
			QualifiedID: qualified, Alias: alias, ModulePath: located.dependency.Path,
			Version: located.dependency.Version, SourceModel: published.child.Bundle.Manifest.Module.ModelID,
			Publication: published.publication.ID, Domain: domain, SourceID: id, ResolvedDir: located.dir,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].QualifiedID < out[j].QualifiedID })
	return out
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-064, REQ-EMG-134
func digestModuleDirectory(root string) (string, error) {
	paths, err := moduleFilePaths(root)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(hash, rel)
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

// moduleFilePaths lists the regular files of a module directory, slash
// separated and sorted, leaving out the lock files and the directories that
// hold tool state. .engflow holds engineering-flow's local experiment state
// and variant worktrees, never module content.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-064, REQ-EMG-134
func moduleFilePaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() && slices.Contains([]string{".git", ".engmod", ".engflow"}, entry.Name()) {
			return filepath.SkipDir
		}
		if entry.Type().IsRegular() && !slices.Contains([]string{CompositionLockFileName, CompositionLocalLockFileName}, filepath.Base(path)) {
			paths = append(paths, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// LinkDependencies returns the resolved direct dependencies of a composed
// system as requirement link evaluation sees them: alias, resolved version,
// model and the publication each subsystem selects.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func LinkDependencies(sys *ComposedSystem) []reqlinks.Dependency {
	if sys == nil {
		return nil
	}
	deps := make([]reqlinks.Dependency, 0, len(sys.Children))
	for _, child := range sys.Children {
		dep := reqlinks.Dependency{Alias: child.Dependency, Version: child.Version, Bundle: child.Bundle, Local: child.Local}
		if publication, ok := publicationByID(child.Bundle.Manifest.Publications, child.Publication); ok {
			dep.Publications = []model.ManifestPublication{publication}
		}
		deps = append(deps, dep)
	}
	return deps
}

// TRLC-LINKS: REQ-EMG-049
func hasCompositionErrors(diags []validate.Diagnostic) bool {
	for _, diagnostic := range diags {
		if diagnostic.Severity == validate.SeverityError {
			return true
		}
	}
	return false
}

// validateComposition checks bindings across the composed tree: every subsystem
// required interface must be satisfied and satisfactions must name published
// identifiers. top marks the root of the
// composition run: an unsatisfied required interface is an error there when
// the root's module kind is "system" (the composing system answers for its
// subsystems' needs) and a warning below it, where a parent may still bind it.
// TRLC-LINKS: REQ-EMG-019, REQ-EMG-021, REQ-EMG-022, REQ-EMG-066
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func validateComposition(sys *ComposedSystem, top bool) []validate.Diagnostic {
	satisfied, diags := satisfactionDiagnostics(sys)
	diags = append(diags, unsatisfiedRequireDiagnostics(sys, satisfied, top)...)
	for _, c := range sys.Children {
		diags = append(diags, validateComposition(c, false)...)
	}
	return diags
}

// satisfactionDiagnostics reports the satisfactions of a system that name an
// unpublished need or provider, and returns the published needs they satisfy.
// Satisfactions use the same publication-qualified visibility boundary.
// TRLC-LINKS: REQ-EMG-019, REQ-EMG-021, REQ-EMG-022, REQ-EMG-066
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func satisfactionDiagnostics(sys *ComposedSystem) (map[string]bool, []validate.Diagnostic) {
	owner := strings.TrimSpace(sys.Bundle.Architecture.Model.ID)
	satisfied := map[string]bool{}
	var diags []validate.Diagnostic
	for _, s := range sys.Bundle.Architecture.Composition.Satisfactions {
		alias, id, err := model.ParseQualifiedReference(s.Need)
		if err != nil {
			diags = append(diags, errDiag("composition.invalid_qualified_reference", fmt.Sprintf("satisfaction need %q must use alias::ID", s.Need), owner))
			continue
		}
		if publishedByAlias(sys.Children, alias, id) {
			satisfied[alias+"::"+id] = true
		}
		diags = append(diags, providerDiagnostics(sys, s.By, owner)...)
	}
	return satisfied, diags
}

// providerDiagnostics reports a satisfaction provider that is neither
// visible through a selected publication (alias::ID) nor a local hardware
// item.
// TRLC-LINKS: REQ-EMG-019, REQ-EMG-021, REQ-EMG-022, REQ-EMG-066
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func providerDiagnostics(sys *ComposedSystem, by, owner string) []validate.Diagnostic {
	if !strings.Contains(by, "::") {
		if hardwareItem(sys.Bundle, by) {
			return nil
		}
		return []validate.Diagnostic{errDiag("composition.invalid_qualified_reference", fmt.Sprintf("satisfaction provider %q must be alias::ID or a local hardware item", by), owner)}
	}
	byAlias, byID, parseErr := model.ParseQualifiedReference(by)
	if parseErr == nil && publishedByAlias(sys.Children, byAlias, byID) {
		return nil
	}
	return []validate.Diagnostic{errDiag("composition.unpublished_reference", fmt.Sprintf("satisfaction provider %q is not visible through a selected publication", by), owner)}
}

// unsatisfiedRequireDiagnostics reports each published required interface of
// a subsystem that no satisfaction binds: an error in the root of a "system"
// module, a warning elsewhere.
// TRLC-LINKS: REQ-EMG-019, REQ-EMG-021, REQ-EMG-022, REQ-EMG-066
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func unsatisfiedRequireDiagnostics(sys *ComposedSystem, satisfied map[string]bool, top bool) []validate.Diagnostic {
	owner := strings.TrimSpace(sys.Bundle.Architecture.Model.ID)
	severity := validate.SeverityWarning
	if top && strings.EqualFold(strings.TrimSpace(sys.Bundle.Manifest.Module.Kind), "system") {
		severity = validate.SeverityError
	}
	var diags []validate.Diagnostic
	for _, c := range sys.Children {
		for _, id := range unsatisfiedNeeds(c, satisfied) {
			diags = append(diags, validate.Diagnostic{
				Code: "composition.unsatisfied_require", Severity: severity,
				Message: fmt.Sprintf("subsystem %q required interface %q is not satisfied by %s", c.SubsystemID, id, owner),
				Path:    owner,
			})
		}
	}
	return diags
}

// unsatisfiedNeeds is the ids of the interfaces a child requires and
// publishes that the satisfied needs do not include.
// TRLC-LINKS: REQ-EMG-019, REQ-EMG-021, REQ-EMG-022, REQ-EMG-066
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func unsatisfiedNeeds(c *ComposedSystem, satisfied map[string]bool) []string {
	var ids []string
	for _, req := range c.Bundle.Architecture.Contract.Requires {
		need, _ := model.QualifyReference(c.Dependency, req.ID)
		if c.PublishedIDs[req.ID] && !satisfied[need] {
			ids = append(ids, req.ID)
		}
	}
	return ids
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func publishedByAlias(children []*ComposedSystem, alias, id string) bool {
	for _, child := range children {
		if child.Dependency == alias && child.PublishedIDs[id] {
			return true
		}
	}
	return false
}

// validateQualifiedReferences applies publication visibility to every typed
// reference field in the parent bundle.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051, REQ-EMG-065
func validateQualifiedReferences(bundle model.Bundle, children []*ComposedSystem, path string) []validate.Diagnostic {
	walker := &referenceWalker{children: children, path: path, seen: map[string]bool{}}
	for _, part := range referenceRoots(bundle) {
		walker.walk(reflect.ValueOf(part.value), part.path)
	}
	return walker.diags
}

// referencePart is a part of the authored model to search for references,
// with the field path it is reported under.
type referencePart struct {
	path  string
	value any
}

// referenceRoots lists the parts of the bundle to walk, each authored
// document once. The aggregate architecture also carries behavior,
// assurance, compliance and views content; walking it whole would report one
// reference under two field paths.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051, REQ-EMG-065
func referenceRoots(bundle model.Bundle) []referencePart {
	a := bundle.Architecture.AuthoredArchitecture
	return []referencePart{
		{"architecture.functionalGroups", a.FunctionalGroups},
		{"architecture.functionalUnits", a.FunctionalUnits},
		{"architecture.actors", a.Actors},
		{"architecture.referencedElements", a.ReferencedElements},
		{"architecture.interfaces", a.Interfaces},
		{"architecture.dataObjects", a.DataObjects},
		{"architecture.deploymentTargets", a.DeploymentTargets},
		{"architecture.hardwareItems", a.HardwareItems},
		{"architecture.hardwareInterfaces", a.HardwareInterfaces},
		{"architecture.contract", bundle.Architecture.Contract},
		{"architecture.composition", bundle.Architecture.Composition},
		{"architecture.semantics", bundle.Architecture.Semantics},
		{"", bundle.Requirements},
		{"", bundle.Behavior},
		{"", bundle.Assurance},
		{"", bundle.Compliance},
		{"", bundle.Views},
	}
}

// referenceWalker checks the qualified references of a model once per field.
type referenceWalker struct {
	children []*ComposedSystem
	path     string
	seen     map[string]bool
	diags    []validate.Diagnostic
}

// check reports a qualified reference no selected publication makes visible.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051, REQ-EMG-065
func (w *referenceWalker) check(ref, fieldPath string) {
	ref = strings.TrimSpace(ref)
	if !strings.Contains(ref, "::") {
		return
	}
	alias, id, err := model.ParseQualifiedReference(ref)
	key := fieldPath + "\x00" + ref
	if w.seen[key] {
		return
	}
	w.seen[key] = true
	if err != nil || !publishedByAlias(w.children, alias, id) {
		w.diags = append(w.diags, errDiag("composition.unpublished_reference", fmt.Sprintf("reference %q is not visible through a selected dependency publication", ref), w.path+":"+fieldPath))
	}
}

// walk visits a value and the values below it.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051, REQ-EMG-065
func (w *referenceWalker) walk(value reflect.Value, fieldPath string) {
	if !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.Pointer:
		if !value.IsNil() {
			w.walk(value.Elem(), fieldPath)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			w.walk(value.Index(i), fmt.Sprintf("%s[%d]", fieldPath, i))
		}
	case reflect.Struct:
		w.walkStruct(value, fieldPath)
	}
}

// walkStruct checks the reference fields of a struct and walks its other
// fields.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051, REQ-EMG-065
func (w *referenceWalker) walkStruct(value reflect.Value, fieldPath string) {
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		tag := strings.Split(valueType.Field(i).Tag.Get("yaml"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		next := tag
		if fieldPath != "" {
			next = fieldPath + "." + tag
		}
		if !w.checkReferenceField(value.Field(i), tag, next) {
			w.walk(value.Field(i), next)
		}
	}
}

// checkReferenceField checks a field that holds a reference or a list of
// references and reports whether it was one.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051, REQ-EMG-065
func (w *referenceWalker) checkReferenceField(field reflect.Value, tag, fieldPath string) bool {
	if !isReferenceField(tag) {
		return false
	}
	switch {
	case field.Kind() == reflect.String:
		w.check(field.String(), fieldPath)
	case field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.String:
		for j := 0; j < field.Len(); j++ {
			w.check(field.Index(j).String(), fmt.Sprintf("%s[%d]", fieldPath, j))
		}
	default:
		return false
	}
	return true
}

// QualifiedReferenceDiagnostics reports every alias::ID reference in the
// authored model that no selected dependency publication makes visible, once
// per field path (composition.unpublished_reference). A model without
// composition has no visible imports, so each of its qualified references is
// reported.
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func QualifiedReferenceDiagnostics(bundle model.Bundle, result CompositionResult) []validate.Diagnostic {
	var children []*ComposedSystem
	if result.Root != nil {
		children = result.Root.Children
	}
	return validateQualifiedReferences(bundle, children, bundle.ManifestPath)
}

// notReferenceFields are field names that hold file paths and URLs (an IPv6
// URL holds "::"), never model IDs, though they end in "ref".
var notReferenceFields = []string{"schemaref", "catalogref", "basecatalogref", "href", "cataloghref"}

// referenceFields are the field names, lower case, that hold model IDs
// beyond those ending in "ref" or "refs".
var referenceFields = []string{
	"group", "owner", "responsiblerole", "responsibleroles", "targets", "triggers", "namespace", "inheritedfrom",
	"relatedcontrols", "attackvectors", "threatscenarios", "threats",
	"from", "to", "target", "need", "by", "requirement", "appliesto", "members", "hosts", "roots",
	"datain", "dataout", "next", "onerror",
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051, REQ-EMG-065
func isReferenceField(tag string) bool {
	lower := strings.ToLower(tag)
	if slices.Contains(notReferenceFields, lower) {
		return false
	}
	return slices.Contains(referenceFields, lower) || strings.HasSuffix(lower, "ref") || strings.HasSuffix(lower, "refs")
}

// hardwareItem reports whether the bundle declares a hardware item with the given id.
// TRLC-LINKS: REQ-EMG-024
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func hardwareItem(bundle model.Bundle, id string) bool {
	for _, h := range bundle.Architecture.AuthoredArchitecture.HardwareItems {
		if h.ID == id {
			return true
		}
	}
	return false
}

// errDiag builds an error-severity diagnostic.
// TRLC-LINKS: REQ-EMG-019
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func errDiag(code, msg, path string) validate.Diagnostic {
	return validate.Diagnostic{Code: code, Severity: validate.SeverityError, Message: msg, Path: path}
}

// ComposeForGeneration returns the composition for a bundle: the caller's
// result when given, otherwise one fresh resolution with the bundle itself as
// the root system (so in-memory content, not the files on disk, is checked).
// A model without composition returns an empty result.
// TRLC-LINKS: REQ-EMG-066
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-SYSTEM-COMPOSITION
func ComposeForGeneration(bundle model.Bundle, given *CompositionResult, options ResolutionOptions) (CompositionResult, error) {
	if !HasComposition(bundle) {
		return CompositionResult{}, nil
	}
	if given != nil {
		return *given, nil
	}
	return GenerateCompositionForBundle(bundle, options)
}
