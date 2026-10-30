// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

// System-of-systems composition. A system model may reference subsystem models in
// local subdirectories; this resolves the parent->child DAG, enforces the
// workspace boundary and acyclicity, materializes parent->subsystem allocation
// traceability (without modifying any subsystem), and validates the bindings.
// References are downward-only: a subsystem never knows its parents.

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistry"
	"cuelang.org/go/mod/module"
	"cuelang.org/go/mod/modzip"
	"github.com/gofrs/flock"
	"github.com/labeth/engineering-model-go/internal/model"
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

// MaterializedAllocation is a parent->subsystem allocation with its composed-view resolution status.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type MaterializedAllocation struct {
	System            string // owning (parent) system id
	Requirement       string
	Subsystem         string
	Target            string // public id within the subsystem
	TargetRef         string // the subsystem requirement that realizes the target (its contract ref)
	Resolved          bool   // target is published in the subsystem contract
	TargetRefResolved bool   // TargetRef names a requirement that exists in the subsystem
	Note              string
}

// CompositionResult is the resolved tree plus materialized allocation trace and diagnostics.
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
type CompositionResult struct {
	Root        *ComposedSystem
	Allocations []MaterializedAllocation
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
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064, REQ-EMG-066
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
	workspaceDocument, workspacePath, err := model.FindWorkspaceWith(absTop, options.Workspace)
	if err != nil {
		return CompositionResult{}, fmt.Errorf("load model workspace: %w", err)
	}
	workspace := filepath.Dir(absTop)
	if workspacePath != "" {
		workspace = filepath.Dir(workspacePath)
	}
	resolver := &compositionResolver{
		workspace:         workspace,
		workspaceDocument: workspaceDocument,
		cacheMode:         options.cacheMode(),
		fetched:           map[string]string{},
		cacheNames:        map[string]string{},
		rootBundle:        rootBundle,
	}
	ancestry := map[string]bool{}
	root, provenance, locks, diags := resolver.resolveSystem("", "", "", absTop, ancestry, true)
	res := CompositionResult{
		Root: root, Provenance: provenance,
		Lock:        CompositionLockDocument{SchemaVersion: 1, Dependencies: locks},
		Diagnostics: diags,
		Workspace:   workspace, cacheNames: resolver.cacheNames,
	}
	if root != nil {
		res.Allocations = materializeAllocations(root)
		res.Diagnostics = append(res.Diagnostics, validateComposition(root, true)...)
	}
	resolver.nameCachePaths(res.Diagnostics)
	if hasCompositionErrors(res.Diagnostics) {
		return res, nil
	}
	dir := filepath.Dir(absTop)
	replaced := len(workspaceDocument.Replacements) > 0
	switch {
	case options.lockMode() == LockModeCheck:
		if !replaced {
			res.Diagnostics = append(res.Diagnostics, checkCompositionLock(filepath.Join(dir, CompositionLockFileName), res.Lock)...)
		} else {
			res.Diagnostics = append(res.Diagnostics, validate.Diagnostic{
				Code: "composition.lock_check_skipped", Severity: validate.SeverityWarning,
				Message: fmt.Sprintf("workspace replacements active; the committed %s is checked only with %s=off", CompositionLockFileName, model.WorkspaceEnv),
				Path:    filepath.Join(dir, CompositionLockFileName),
			})
		}
	case replaced:
		if err := WriteCompositionLock(filepath.Join(dir, CompositionLocalLockFileName), res.Lock); err != nil {
			return res, err
		}
	default:
		if err := WriteCompositionLock(filepath.Join(dir, CompositionLockFileName), res.Lock); err != nil {
			return res, err
		}
	}
	return res, nil
}

// checkCompositionLock compares the resolved lock with the committed lock and
// reports every missing, extra, moved or re-published dependency.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func checkCompositionLock(path string, resolved CompositionLockDocument) []validate.Diagnostic {
	data, err := os.ReadFile(path)
	if err != nil {
		return []validate.Diagnostic{errDiag("composition.lock_missing", fmt.Sprintf("%s is missing; resolve without %s=check and commit it", filepath.Base(path), LockEnv), path)}
	}
	var committed CompositionLockDocument
	if err := yaml.Unmarshal(data, &committed); err != nil {
		return []validate.Diagnostic{errDiag("composition.lock_invalid", fmt.Sprintf("%s cannot be read: %v", filepath.Base(path), err), path)}
	}
	byAlias := map[string]CompositionLockDependency{}
	for _, dependency := range committed.Dependencies {
		byAlias[dependency.Alias] = dependency
	}
	var diags []validate.Diagnostic
	for _, got := range resolved.Dependencies {
		want, ok := byAlias[got.Alias]
		delete(byAlias, got.Alias)
		switch {
		case !ok:
			diags = append(diags, errDiag("composition.lock_mismatch", fmt.Sprintf("dependency %q (%s) is missing from the committed lock", got.Alias, moduleAt(got.Path, got.Version)), path))
		case want.Path != got.Path || want.Version != got.Version:
			diags = append(diags, errDiag("composition.lock_mismatch", fmt.Sprintf("dependency %q: lock pins %s but the manifest resolves %s", got.Alias, moduleAt(want.Path, want.Version), moduleAt(got.Path, got.Version)), path))
		case want.Digest != got.Digest:
			diags = append(diags, errDiag("composition.lock_mismatch", fmt.Sprintf("dependency %q %s: content digest %s differs from the locked %s (tag moved or content changed)", got.Alias, moduleAt(got.Path, got.Version), got.Digest, want.Digest), path))
		}
	}
	extra := make([]string, 0, len(byAlias))
	for alias := range byAlias {
		extra = append(extra, alias)
	}
	sort.Strings(extra)
	for _, alias := range extra {
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

// resolveSystem loads the model and recursively resolves manifest dependencies.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-018, REQ-EMG-035, REQ-EMG-036, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-067
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func (r *compositionResolver) resolveSystem(subsystemID, dependencyAlias, publicationID, manifestPath string, ancestry map[string]bool, root bool) (*ComposedSystem, []ImportedEntityProvenance, []CompositionLockDependency, []validate.Diagnostic) {
	var diags []validate.Diagnostic
	if ancestry[manifestPath] {
		return nil, nil, nil, []validate.Diagnostic{{
			Code: "composition.cycle", Severity: validate.SeverityError,
			Message: fmt.Sprintf("subsystem reference cycle detected at %s", manifestPath),
			Path:    manifestPath,
		}}
	}
	ancestry[manifestPath] = true
	defer delete(ancestry, manifestPath)

	var bundle model.Bundle
	if root && r.rootBundle != nil {
		bundle = *r.rootBundle
	} else {
		canonical, err := model.LoadCanonicalBundle(manifestPath)
		if err != nil {
			return nil, nil, nil, []validate.Diagnostic{{
				Code: "composition.load_failed", Severity: validate.SeverityError,
				Message: err.Error(), Path: manifestPath,
			}}
		}
		bundle = canonical.Documents()
	}
	sys := &ComposedSystem{
		SubsystemID: subsystemID, Dependency: dependencyAlias, Publication: publicationID,
		ModulePath: bundle.Manifest.Module.Path, Version: bundle.Manifest.Module.Version,
		Dir: filepath.Dir(manifestPath), Bundle: bundle, Requirements: bundle.Requirements,
	}
	if !root {
		// A dependency's code is its own module directory (workspace checkout or
		// module cache entry), never the consumer's repository around it.
		sys.Bundle.CodeScopeRoot = sys.Dir
	}

	dependencies, ddiags := validateDependencies(bundle.Manifest, manifestPath)
	diags = append(diags, ddiags...)
	var provenance []ImportedEntityProvenance
	var locks []CompositionLockDependency
	lockedAliases := map[string]bool{}
	for _, sub := range bundle.Architecture.Composition.Subsystems {
		dependency, ok := dependencies[sub.Dependency]
		if !ok {
			diags = append(diags, errDiag("composition.unknown_dependency", fmt.Sprintf("subsystem %q references unknown dependency alias %q", sub.ID, sub.Dependency), manifestPath))
			continue
		}
		if !contains(dependency.Publications, sub.Publication) {
			diags = append(diags, errDiag("composition.publication_not_selected", fmt.Sprintf("subsystem %q publication %q is not selected by dependency %q", sub.ID, sub.Publication, sub.Dependency), manifestPath))
			continue
		}
		childManifest, resolvedDir, digest, rdiags := r.resolveDependency(dependency, manifestPath)
		diags = append(diags, rdiags...)
		if childManifest == "" {
			continue
		}
		child, childProvenance, childLocks, cdiags := r.resolveSystem(sub.ID, sub.Dependency, sub.Publication, childManifest, ancestry, false)
		diags = append(diags, cdiags...)
		if child != nil {
			if child.ModulePath != dependency.Path || child.Version != dependency.Version {
				diags = append(diags, errDiag("composition.module_identity", fmt.Sprintf("dependency %q requested %s@%s but child manifest declares %s@%s", dependency.Alias, dependency.Path, dependency.Version, child.ModulePath, child.Version), childManifest))
				continue
			}
			for _, selected := range dependency.Publications {
				selectedPublication, exists := publicationByID(child.Bundle.Manifest.Publications, selected)
				if !exists {
					diags = append(diags, errDiag("composition.unknown_publication", fmt.Sprintf("dependency %q selects publication %q which the child does not declare", dependency.Alias, selected), childManifest))
					continue
				}
				_, _, selectedDiags := validatePublication(child.Bundle, selectedPublication)
				diags = append(diags, selectedDiags...)
			}
			publication, ok := publicationByID(child.Bundle.Manifest.Publications, sub.Publication)
			if !ok {
				diags = append(diags, errDiag("composition.unknown_publication", fmt.Sprintf("dependency %q has no publication %q", sub.Dependency, sub.Publication), childManifest))
				continue
			}
			visible, facets, pdiags := validatePublication(child.Bundle, publication)
			diags = append(diags, pdiags...)
			child.PublishedIDs, child.PublishedFacet = visible, facets
			sys.Children = append(sys.Children, child)
			provenance = append(provenance, publicationProvenance(sub.Dependency, dependency, child, publication, resolvedDir)...)
			provenance = append(provenance, childProvenance...)
			locks = append(locks, childLocks...)
			if root && !lockedAliases[dependency.Alias] {
				locks = append(locks, CompositionLockDependency{
					Alias: dependency.Alias, Path: dependency.Path, Version: dependency.Version,
					Publications: append([]string(nil), dependency.Publications...), Digest: digest,
				})
				lockedAliases[dependency.Alias] = true
			}
		}
	}
	diags = append(diags, validateQualifiedReferences(bundle, sys.Children, manifestPath)...)
	return sys, provenance, locks, diags
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

// resolveDependency resolves an exact manifest dependency to its engmod.yml.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE, REF-CUE-OCI-MODULE-REGISTRY
func (r *compositionResolver) resolveDependency(dependency model.ManifestDependency, parentManifest string) (string, string, string, []validate.Diagnostic) {
	at := func(code, msg string) []validate.Diagnostic {
		return []validate.Diagnostic{{Code: code, Severity: validate.SeverityError, Message: msg, Path: parentManifest}}
	}
	mv, err := model.ParseExactModuleVersion(dependency.Path, dependency.Version)
	if err != nil {
		return "", "", "", at("composition.invalid_module", fmt.Sprintf("dependency %q: %v", dependency.Alias, err))
	}

	moduleRoot := ""
	if replacement, ok := r.workspaceDocument.ReplacementPath(mv.Path()); ok {
		moduleRoot = filepath.Clean(filepath.Join(r.workspace, replacement))
		if !pathWithin(r.workspace, moduleRoot) {
			return "", "", "", at("composition.workspace_path_escape", fmt.Sprintf("dependency %q replacement for %q resolves outside workspace", dependency.Alias, mv.Path()))
		}
	} else {
		moduleRoot, err = r.fetchModule(mv)
		if err != nil {
			return "", "", "", at("composition.module_fetch_failed", fmt.Sprintf("dependency %q module %s: %v", dependency.Alias, mv, err))
		}
	}

	// Confine the dependency before reading any file it names.
	if escapes := dependencyPathEscapes(dependency.Alias, moduleRoot, filepath.Join(moduleRoot, "engmod.yml")); len(escapes) > 0 {
		return "", "", "", escapes
	}
	actualModule, err := model.ReadCUEModulePath(moduleRoot)
	if err != nil {
		return "", "", "", at("composition.module_identity", fmt.Sprintf("dependency %q: %v", dependency.Alias, err))
	}
	if actualModule != mv.Path() {
		return "", "", "", at("composition.module_identity", fmt.Sprintf("dependency %q requested module %q but source declares %q", dependency.Alias, mv.Path(), actualModule))
	}
	digest, err := digestModuleDirectory(moduleRoot)
	if err != nil {
		return "", "", "", at("composition.module_digest", fmt.Sprintf("dependency %q: %v", dependency.Alias, err))
	}
	childManifest := filepath.Join(moduleRoot, "engmod.yml")
	if _, err := os.Stat(childManifest); err != nil {
		return "", "", "", at("composition.missing_manifest", fmt.Sprintf("dependency %q has no engmod.yml at %s", dependency.Alias, childManifest))
	}
	return childManifest, moduleRoot, digest, nil
}

// fetchModule downloads and atomically materializes a CUE module in the
// workspace cache. A cached entry is reused offline in reuse mode; in verify
// mode it is reused when the registry's manifest digest equals the one
// recorded at fetch time (or the registry is unreachable) and replaced
// otherwise; in refresh mode it is always replaced.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
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
	if cached && r.cacheMode != ModuleCacheRefresh && r.cacheMode != moduleCacheVerify {
		return dest, nil
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return "", err
	}
	lock := flock.New(dest + ".lock")
	if err := lock.Lock(); err != nil {
		return "", err
	}
	defer func() { _ = lock.Unlock() }()
	if _, err := os.Stat(dest); err == nil {
		cached = true
	}
	if cached && r.cacheMode != ModuleCacheRefresh && r.cacheMode != moduleCacheVerify {
		return dest, nil
	}

	if r.moduleClient == nil {
		resolver, err := modconfig.NewResolver(nil)
		if err != nil {
			if cached && r.cacheMode == moduleCacheVerify {
				return dest, nil
			}
			return "", err
		}
		r.moduleClient = modregistry.NewClientWithResolver(resolver)
	}
	remoteModule, err := r.moduleClient.GetModule(context.Background(), mv)
	if err != nil {
		if cached && r.cacheMode == moduleCacheVerify {
			// Offline: the committed lock still guards the cached content.
			return dest, nil
		}
		return "", err
	}
	manifestDigest := string(remoteModule.ManifestDigest())
	digestFile := dest + ".manifest-digest"
	if cached && r.cacheMode == moduleCacheVerify {
		if recorded, err := os.ReadFile(digestFile); err == nil && strings.TrimSpace(string(recorded)) == manifestDigest {
			r.fetched[dest] = dest
			return dest, nil
		}
	}
	zipReader, err := remoteModule.GetZip(context.Background())
	if err != nil {
		return "", err
	}
	zipFile, err := os.CreateTemp(cacheRoot, "."+sanitizeDirName(mv.Path())+"-*.zip")
	if err != nil {
		_ = zipReader.Close()
		return "", err
	}
	zipPath := zipFile.Name()
	defer os.Remove(zipPath)
	if _, err := io.Copy(zipFile, zipReader); err != nil {
		_ = zipFile.Close()
		_ = zipReader.Close()
		return "", err
	}
	if err := zipFile.Close(); err != nil {
		_ = zipReader.Close()
		return "", err
	}
	if err := zipReader.Close(); err != nil {
		return "", err
	}

	if err := checkModuleZipEntries(zipPath); err != nil {
		return "", err
	}
	tempDir, err := os.MkdirTemp(cacheRoot, "."+sanitizeDirName(mv.Path())+"-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)
	if err := modzip.Unzip(tempDir, mv, zipPath); err != nil {
		return "", err
	}
	if cached {
		// Replace the stale entry: move it aside, then put the new one in.
		stale, err := os.MkdirTemp(cacheRoot, "."+sanitizeDirName(mv.Path())+"-stale-")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(stale)
		if err := os.Rename(dest, filepath.Join(stale, "module")); err != nil {
			return "", err
		}
	}
	if err := os.Rename(tempDir, dest); err != nil {
		if _, statErr := os.Stat(dest); statErr != nil {
			return "", err
		}
	}
	_ = os.WriteFile(digestFile, []byte(manifestDigest+"\n"), 0o644)
	r.fetched[dest] = dest
	return dest, nil
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049
func moduleCacheDirName(mv module.Version) string {
	sum := sha256.Sum256([]byte(mv.String()))
	return sanitizeDirName(mv.Path()) + "-" + fmt.Sprintf("%x", sum[:8])
}

// TRLC-LINKS: REQ-EMG-017, REQ-EMG-049
func pathWithin(root, path string) bool {
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
		dependencies[alias] = dependency
	}
	seenPublications := map[string]bool{}
	for _, publication := range manifest.Publications {
		if seenPublications[publication.ID] {
			diags = append(diags, errDiag("composition.duplicate_publication", fmt.Sprintf("publication %q is duplicated", publication.ID), path))
		}
		seenPublications[publication.ID] = true
	}
	return dependencies, diags
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
	owned := domainOwnedIDs(bundle)
	visible := map[string]bool{}
	facets := map[string]string{}
	var diags []validate.Diagnostic
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
			if !owned[domain][id] {
				diags = append(diags, errDiag("composition.publication_unowned_id", fmt.Sprintf("publication %q exposes %s id %q not owned by that domain", publication.ID, domain, id), bundle.ManifestPath))
				continue
			}
			if previous, exists := facets[id]; exists {
				diags = append(diags, errDiag("composition.publication_duplicate_id", fmt.Sprintf("publication %q exposes id %q in both %s and %s", publication.ID, id, previous, domain), bundle.ManifestPath))
				continue
			}
			visible[id] = true
			facets[id] = domain
		}
	}
	return visible, facets, diags
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
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		collectOwnedIDs(value.Elem(), ids)
		return
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			collectOwnedIDs(value.Index(i), ids)
		}
	case reflect.Struct:
		valueType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := valueType.Field(i)
			if strings.Split(field.Tag.Get("yaml"), ",")[0] == "id" && value.Field(i).Kind() == reflect.String {
				if id := strings.TrimSpace(value.Field(i).String()); id != "" {
					ids[id] = true
				}
				continue
			}
			collectOwnedIDs(value.Field(i), ids)
		}
	}
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
func publicationProvenance(alias string, dependency model.ManifestDependency, child *ComposedSystem, publication model.ManifestPublication, resolvedDir string) []ImportedEntityProvenance {
	var out []ImportedEntityProvenance
	for id, domain := range child.PublishedFacet {
		qualified, _ := model.QualifyReference(alias, id)
		out = append(out, ImportedEntityProvenance{
			QualifiedID: qualified, Alias: alias, ModulePath: dependency.Path,
			Version: dependency.Version, SourceModel: child.Bundle.Manifest.Module.ModelID,
			Publication: publication.ID, Domain: domain, SourceID: id, ResolvedDir: resolvedDir,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].QualifiedID < out[j].QualifiedID })
	return out
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-064
func digestModuleDirectory(root string) (string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == ".engmod") {
			return filepath.SkipDir
		}
		if entry.Type().IsRegular() && filepath.Base(path) != CompositionLockFileName && filepath.Base(path) != CompositionLocalLockFileName {
			paths = append(paths, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
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

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
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

// materializeAllocations walks the tree and produces the parent->subsystem allocation
// trace, resolving each target against the referenced subsystem's published contract.
// The subsystem models are never modified.
// TRLC-LINKS: REQ-EMG-020, REQ-EMG-025
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func materializeAllocations(sys *ComposedSystem) []MaterializedAllocation {
	var out []MaterializedAllocation
	childByID := map[string]*ComposedSystem{}
	for _, c := range sys.Children {
		childByID[c.SubsystemID] = c
	}
	systemID := strings.TrimSpace(sys.Bundle.Architecture.Model.ID)
	for _, a := range sys.Bundle.Architecture.Composition.Allocations {
		ma := MaterializedAllocation{System: systemID, Requirement: a.Requirement, Subsystem: a.To, Target: a.Target}
		if child, ok := childByID[a.To]; ok {
			alias, targetID, err := model.ParseQualifiedReference(a.Target)
			if err != nil || alias != child.Dependency {
				ma.Note = "target must use the subsystem dependency alias as alias::ID"
			} else if !child.PublishedIDs[targetID] {
				ma.Note = "target is not visible through the selected publication"
			} else {
				ma.Resolved = true
				if entry, provided := contractProvidedEntry(child.Bundle, targetID); provided {
					ma.TargetRef = strings.TrimSpace(entry.Ref)
					if ma.TargetRef != "" {
						ma.TargetRefResolved = requirementExists(child.Requirements, ma.TargetRef)
					} else {
						ma.Note = "no contract ref: delegation does not resolve to a specific requirement"
					}
				}
			}
		} else if hardwareItem(sys.Bundle, a.To) {
			ma.Resolved = true
			ma.Note = "allocated to hardware item"
		} else {
			ma.Note = "unknown allocation target"
		}
		out = append(out, ma)
	}
	for _, c := range sys.Children {
		out = append(out, materializeAllocations(c)...)
	}
	return out
}

// validateComposition checks bindings across the composed tree: allocation targets must
// be public, every subsystem required interface must be satisfied, and allocations must
// reference real requirements and subsystems. top marks the root of the
// composition run: an unsatisfied required interface is an error there when
// the root's module kind is "system" (the composing system answers for its
// subsystems' needs) and a warning below it, where a parent may still bind it.
// TRLC-LINKS: REQ-EMG-019, REQ-EMG-021, REQ-EMG-022, REQ-EMG-066
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func validateComposition(sys *ComposedSystem, top bool) []validate.Diagnostic {
	var diags []validate.Diagnostic
	childByID := map[string]*ComposedSystem{}
	for _, c := range sys.Children {
		childByID[c.SubsystemID] = c
	}
	owner := strings.TrimSpace(sys.Bundle.Architecture.Model.ID)
	loc := func() string { return owner }

	// Allocations may only target identifiers visible through the subsystem's
	// selected publication, using the dependency alias as qualifier.
	for _, a := range sys.Bundle.Architecture.Composition.Allocations {
		child, ok := childByID[a.To]
		if !ok {
			if !hardwareItem(sys.Bundle, a.To) {
				diags = append(diags, errDiag("composition.unknown_subsystem", fmt.Sprintf("allocation in %s targets unknown subsystem %q", owner, a.To), loc()))
			}
			continue
		}
		alias, targetID, err := model.ParseQualifiedReference(a.Target)
		if err != nil || alias != child.Dependency {
			diags = append(diags, errDiag("composition.invalid_qualified_reference", fmt.Sprintf("allocation in %s must target %s::ID for subsystem %q", owner, child.Dependency, a.To), loc()))
			continue
		}
		if !child.PublishedIDs[targetID] {
			diags = append(diags, errDiag("composition.allocation_to_internal", fmt.Sprintf("allocation in %s targets %q which is not visible through dependency %q publication %q", owner, a.Target, child.Dependency, child.Publication), loc()))
			continue
		}
		entry, provided := contractProvidedEntry(child.Bundle, targetID)
		if !provided {
			diags = append(diags, validate.Diagnostic{
				Code: "composition.allocation_target_not_contract", Severity: validate.SeverityWarning,
				Message: fmt.Sprintf("allocation of %s in %s targets %q, which is not a contract provides entry of subsystem %q; the delegation names no subsystem requirement", a.Requirement, owner, a.Target, a.To),
				Path:    loc(),
			})
			continue
		}
		ref := strings.TrimSpace(entry.Ref)
		if ref == "" {
			diags = append(diags, validate.Diagnostic{
				Code: "composition.untraceable_delegation", Severity: validate.SeverityWarning,
				Message: fmt.Sprintf("allocation in %s to %s/%s does not resolve to a specific requirement; declare a contract ref on %s naming the subsystem requirement that satisfies it", owner, a.To, a.Target, a.Target),
				Path:    loc(),
			})
		} else if !requirementExists(child.Requirements, ref) {
			diags = append(diags, errDiag("composition.unknown_target_requirement", fmt.Sprintf("allocation in %s to %s/%s names requirement %q, which subsystem %q does not define", owner, a.To, a.Target, ref, a.To), loc()))
		}
	}

	// Satisfactions use the same publication-qualified visibility boundary.
	satisfied := map[string]bool{}
	for _, s := range sys.Bundle.Architecture.Composition.Satisfactions {
		alias, id, err := model.ParseQualifiedReference(s.Need)
		if err != nil {
			diags = append(diags, errDiag("composition.invalid_qualified_reference", fmt.Sprintf("satisfaction need %q must use alias::ID", s.Need), loc()))
			continue
		}
		for _, child := range sys.Children {
			if child.Dependency == alias && child.PublishedIDs[id] {
				satisfied[alias+"::"+id] = true
			}
		}
		if strings.Contains(s.By, "::") {
			if byAlias, byID, parseErr := model.ParseQualifiedReference(s.By); parseErr != nil || !publishedByAlias(sys.Children, byAlias, byID) {
				diags = append(diags, errDiag("composition.unpublished_reference", fmt.Sprintf("satisfaction provider %q is not visible through a selected publication", s.By), loc()))
			}
		} else if !hardwareItem(sys.Bundle, s.By) {
			diags = append(diags, errDiag("composition.invalid_qualified_reference", fmt.Sprintf("satisfaction provider %q must be alias::ID or a local hardware item", s.By), loc()))
		}
	}
	for _, c := range sys.Children {
		for _, req := range c.Bundle.Architecture.Contract.Requires {
			if !c.PublishedIDs[req.ID] {
				continue
			}
			need, _ := model.QualifyReference(c.Dependency, req.ID)
			if !satisfied[need] {
				severity := validate.SeverityWarning
				if top && strings.EqualFold(strings.TrimSpace(sys.Bundle.Manifest.Module.Kind), "system") {
					severity = validate.SeverityError
				}
				diags = append(diags, validate.Diagnostic{
					Code: "composition.unsatisfied_require", Severity: severity,
					Message: fmt.Sprintf("subsystem %q required interface %q is not satisfied by %s", c.SubsystemID, req.ID, owner),
					Path:    owner,
				})
			}
		}
	}

	for _, c := range sys.Children {
		diags = append(diags, validateComposition(c, false)...)
	}
	return diags
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
// reference field in the parent bundle, not only composition allocations.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051, REQ-EMG-065
func validateQualifiedReferences(bundle model.Bundle, children []*ComposedSystem, path string) []validate.Diagnostic {
	var diags []validate.Diagnostic
	seen := map[string]bool{}
	check := func(ref, fieldPath string) {
		ref = strings.TrimSpace(ref)
		if !strings.Contains(ref, "::") {
			return
		}
		alias, id, err := model.ParseQualifiedReference(ref)
		key := fieldPath + "\x00" + ref
		if seen[key] {
			return
		}
		seen[key] = true
		if err != nil || !publishedByAlias(children, alias, id) {
			diags = append(diags, errDiag("composition.unpublished_reference", fmt.Sprintf("reference %q is not visible through a selected dependency publication", ref), path+":"+fieldPath))
		}
	}
	var walk func(reflect.Value, string)
	walk = func(value reflect.Value, fieldPath string) {
		if !value.IsValid() {
			return
		}
		if value.Kind() == reflect.Pointer {
			if !value.IsNil() {
				walk(value.Elem(), fieldPath)
			}
			return
		}
		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < value.Len(); i++ {
				walk(value.Index(i), fmt.Sprintf("%s[%d]", fieldPath, i))
			}
		case reflect.Struct:
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
				field := value.Field(i)
				if field.Kind() == reflect.String && isReferenceField(tag) {
					check(field.String(), next)
					continue
				}
				if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.String && isReferenceField(tag) {
					for j := 0; j < field.Len(); j++ {
						check(field.Index(j).String(), fmt.Sprintf("%s[%d]", next, j))
					}
					continue
				}
				walk(field, next)
			}
		}
	}
	// Walk each authored document once. The aggregate architecture also
	// carries behavior, assurance, compliance and views content; walking it
	// whole would report one reference under two field paths.
	a := bundle.Architecture.AuthoredArchitecture
	for _, part := range []struct {
		path  string
		value any
	}{
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
	} {
		walk(reflect.ValueOf(part.value), part.path)
	}
	return diags
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

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051, REQ-EMG-065
func isReferenceField(tag string) bool {
	lower := strings.ToLower(tag)
	switch lower {
	case "schemaref", "catalogref", "basecatalogref", "href", "cataloghref":
		// File paths and URLs (an IPv6 URL holds "::"), never model IDs.
		return false
	case "group", "owner", "responsiblerole", "responsibleroles", "targets", "triggers", "namespace", "inheritedfrom", "relatedcontrols", "attackvectors", "threatscenarios", "threats":
		return true
	}
	return strings.HasSuffix(lower, "ref") || strings.HasSuffix(lower, "refs") ||
		lower == "from" || lower == "to" || lower == "target" || lower == "need" ||
		lower == "by" || lower == "requirement" || lower == "appliesto" ||
		lower == "members" || lower == "hosts" || lower == "roots" ||
		lower == "datain" || lower == "dataout" || lower == "next" || lower == "onerror"
}

// contractProvidedEntry returns the bundle's provided contract entry with the given id.
// TRLC-LINKS: REQ-EMG-022
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func contractProvidedEntry(bundle model.Bundle, id string) (model.ContractEntry, bool) {
	id = strings.TrimSpace(id)
	for _, p := range bundle.Architecture.Contract.Provides {
		if strings.TrimSpace(p.ID) == id {
			return p, true
		}
	}
	return model.ContractEntry{}, false
}

// contractProvides reports whether a bundle's contract publishes the given id.
// TRLC-LINKS: REQ-EMG-022
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func contractProvides(bundle model.Bundle, id string) bool {
	_, ok := contractProvidedEntry(bundle, id)
	return ok
}

// requirementExists reports whether a requirements document defines the given id.
// TRLC-LINKS: REQ-EMG-022, REQ-EMG-026
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func requirementExists(reqs model.RequirementsDocument, id string) bool {
	id = strings.TrimSpace(id)
	for _, r := range reqs.Requirements {
		if strings.TrimSpace(r.ID) == id {
			return true
		}
	}
	return false
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
