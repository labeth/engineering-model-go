// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

// System-of-systems composition across model modules.
// Thin facade over internal/composition; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/composition"

// ComposedSystem is a resolved system together with its resolved subsystems.
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, DO-ARCHITECTURE-MODEL
type ComposedSystem = composition.ComposedSystem

// ImportedEntityProvenance records where each visible imported identifier came
// from without leaking composition metadata into generic target properties.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
type ImportedEntityProvenance = composition.ImportedEntityProvenance

// CompositionLockDocument is deterministic generated dependency state.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
type CompositionLockDocument = composition.CompositionLockDocument

type CompositionLockDependency = composition.CompositionLockDependency

// MaterializedAllocation is a parent->subsystem allocation with its composed-view resolution status.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type MaterializedAllocation = composition.MaterializedAllocation

// CompositionResult is the resolved tree plus materialized allocation trace and diagnostics.
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
type CompositionResult = composition.CompositionResult

// ResolutionOptions overrides the ENGMOD_WORK and ENGMOD_LOCK environment for
// one call. Empty fields fall back to the environment. Workspace is "off" or a
// workspace file path; Lock is "check" or "write".
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
type ResolutionOptions = composition.ResolutionOptions

// PublishedProjectionEntity is one explicitly published dependency entity,
// retaining its typed value, qualified identity, source bundle, and provenance.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
type PublishedProjectionEntity = composition.PublishedProjectionEntity

// PublishedProjection groups the entities selected from one dependency publication.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
type PublishedProjection = composition.PublishedProjection

// CompositionProjection is the reusable, read-only projection boundary used by
// every output format. It never mutates an authored bundle.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
type CompositionProjection = composition.CompositionProjection

const CompositionLockFileName = composition.CompositionLockFileName

// CompositionLocalLockFileName receives the resolved lock while workspace
// replacements are active, so the committed lock never pins unpublished content.
const CompositionLocalLockFileName = composition.CompositionLocalLockFileName

// LockEnv selects the lock mode: "check" compares with the committed lock and
// writes nothing; "write" (or unset) writes the lock.
const LockEnv = composition.LockEnv

// LockModeWrite writes engmod.lock.yml (or the local lock in workspace mode).
const LockModeWrite = composition.LockModeWrite

// LockModeCheck compares the resolved lock with engmod.lock.yml and writes nothing.
const LockModeCheck = composition.LockModeCheck

// ModuleCacheEnv selects the module cache mode: "refresh" re-fetches every
// registry dependency; "reuse" reuses cached modules offline; unset applies
// the defaults of cacheMode.
const ModuleCacheEnv = composition.ModuleCacheEnv

// ModuleCacheReuse reuses a cached module without contacting the
// registry. Set explicitly it applies in every lock mode; it is also the
// default except for lock check mode and lock regeneration (see cacheMode).
const ModuleCacheReuse = composition.ModuleCacheReuse

// ModuleCacheRefresh re-fetches every registry dependency.
const ModuleCacheRefresh = composition.ModuleCacheRefresh

// HasComposition reports whether the model declares any subsystems.
// TRLC-LINKS: REQ-EMG-016
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func HasComposition(bundle Bundle) bool {
	return composition.HasComposition(bundle)
}

// GenerateCompositionFromFile resolves the system-of-systems rooted at engmod.yml
// with the workspace and lock modes taken from ENGMOD_WORK and ENGMOD_LOCK.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateCompositionFromFile(manifestPath string) (CompositionResult, error) {
	return composition.GenerateCompositionFromFile(manifestPath)
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
	return composition.GenerateCompositionFromFileWith(manifestPath, options)
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
func GenerateCompositionForBundle(bundle Bundle, options ResolutionOptions) (CompositionResult, error) {
	return composition.GenerateCompositionForBundle(bundle, options)
}

// WriteCompositionLock writes deterministic generated dependency state.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func WriteCompositionLock(path string, lock CompositionLockDocument) error {
	return composition.WriteCompositionLock(path, lock)
}

// QualifiedReferenceDiagnostics reports every alias::ID reference in the
// authored model that no selected dependency publication makes visible, once
// per field path (composition.unpublished_reference). A model without
// composition has no visible imports, so each of its qualified references is
// reported.
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func QualifiedReferenceDiagnostics(bundle Bundle, result CompositionResult) []Diagnostic {
	return composition.QualifiedReferenceDiagnostics(bundle, result)
}

// BuildCompositionProjection materializes only direct, explicitly selected
// dependency publications. Transitive or unexported child data is not exposed.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func BuildCompositionProjection(result CompositionResult) CompositionProjection {
	return composition.BuildCompositionProjection(result)
}

// ResolveCompositionProjection resolves and validates composition for a bundle.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func ResolveCompositionProjection(bundle Bundle) (CompositionProjection, CompositionResult, error) {
	return composition.ResolveCompositionProjection(bundle)
}

// ResolveCompositionProjectionWith is ResolveCompositionProjection under
// explicit workspace and lock modes.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051, REQ-EMG-063, REQ-EMG-064
func ResolveCompositionProjectionWith(bundle Bundle, options ResolutionOptions) (CompositionProjection, CompositionResult, error) {
	return composition.ResolveCompositionProjectionWith(bundle, options)
}
