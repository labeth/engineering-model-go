// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package scan

// The delta check pipeline (ADR-EMG-053, ADR-EMG-059): it loads the model, builds
// the engine configuration of each file exactly as Scan does, compares the
// changed files with a base revision and sets the level of health.regression at
// the project tier. It needs no roll-ups, so it never runs the whole-module
// code scan that Scan uses for the owners and requirements: only the model, git
// and the changed files are read, which is what makes it fast on a one-file
// change.

import (
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// CheckOptions select what Check compares.
type CheckOptions struct {
	// ManifestPath is the engmod.yml of the model.
	ManifestPath string
	// Tier replaces the life-cycle tier of the model; empty keeps it.
	Tier string
	// Base is the revision to compare with ("base...head" is accepted); empty
	// means HEAD, that is the uncommitted work.
	Base string
	// Head is the revision to check; empty means the working tree.
	Head string
	// Paths limit the check to files and directories inside the project root.
	Paths []string
	// NoCache turns the result cache off. Otherwise results are cached under
	// <root>/.engmod/cache/health, except where the environment variable CI is
	// set (unless UseCache) and while git tracks a file under .engmod
	// (health.OpenCache).
	NoCache bool
	// UseCache keeps the cache on where CI is set (--cache).
	UseCache bool
	// Notes receives the notes of the run that are not part of the report, for
	// example that the cache was refused; nil drops them.
	Notes io.Writer
	// Engine defaults to the tree-sitter engine.
	Engine health.Engine
}

// checkSetup loads the model and resolves the tier, the project root and the
// scope of the paths. It is every step of Check that can be a setup error
// before git is asked anything.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func checkSetup(opts CheckOptions) (bundle model.Bundle, tier, root string, scope []string, err error) {
	bundle, err = model.LoadBundle(opts.ManifestPath)
	if err != nil {
		return bundle, "", "", nil, fmt.Errorf("%w: load model: %v", ErrSetup, err)
	}
	if t := strings.ToLower(strings.TrimSpace(opts.Tier)); t != "" {
		if model.TierRank(t) == 0 {
			return bundle, "", "", nil, fmt.Errorf("%w: unknown tier %q; use one of %s", ErrSetup, opts.Tier, strings.Join(model.Tiers, ", "))
		}
		bundle = health.WithTier(bundle, t)
	}
	tier, _, _ = validate.EffectiveTier(bundle)
	if root, err = filepath.Abs(codescan.WithDefaultCodeRoot(bundle, nil)[0]); err != nil {
		return bundle, "", "", nil, fmt.Errorf("%w: resolve project root: %v", ErrSetup, err)
	}
	if scope, err = resolveScope(root, opts.Paths); err != nil {
		return bundle, "", "", nil, err
	}
	if slices.Contains(scope, ".") { // the whole project
		scope = nil
	}
	return bundle, tier, root, scope, nil
}

// checkEngine builds the engine of a check: the tree-sitter engine (or the one
// in the options), behind the result cache unless it is off, behind the waivers
// of the manifest.
//
// TRLC-LINKS: REQ-EMG-154, REQ-EMG-159
func checkEngine(opts CheckOptions, bundle model.Bundle, root string) *health.WaivingEngine {
	engine := cachedEngine(Options{Engine: opts.Engine, NoCache: opts.NoCache, UseCache: opts.UseCache, Notes: opts.Notes}, root)
	waiving := &health.WaivingEngine{Engine: engine}
	if h := bundle.Manifest.Health; h != nil {
		waiving.Waivers = validate.AcceptedHealthWaivers(bundle, h.Waivers)
	}
	return waiving
}

// Check compares the changed files of the project with a base revision and
// returns the delta report with the level of health.regression at the tier. A
// problem of the setup (the model does not load, an unknown tier, a path
// outside the project, no git work tree, an invalid or unavailable revision) is
// an error and never a result: the caller must not treat it as a pass.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158, REQ-EMG-159
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func Check(opts CheckOptions) (health.CheckReport, error) {
	bundle, tier, root, scope, err := checkSetup(opts)
	if err != nil {
		return health.CheckReport{}, err
	}
	repo := health.NewGitRepo(root)
	if err := health.RequireWorkTree(repo); err != nil {
		return health.CheckReport{}, err
	}
	h := bundle.Manifest.Health
	waiving := checkEngine(opts, bundle, root)
	delta, err := health.Check(health.CheckOptions{
		Repo: repo, Base: opts.Base, Head: opts.Head, Paths: scope,
		Engine:    waiving,
		ConfigFor: func(p string) health.Config { return health.ConfigFor(tier, h, p) },
		Scorer:    health.ScoreFile,
		Exclude:   outOfScope(bundle, root),
		Tier:      tier,
	})
	if err != nil {
		return health.CheckReport{}, err
	}
	verdict, err := policyVerdict(opts, bundle, tier, delta.Base)
	if err != nil {
		return health.CheckReport{}, err
	}
	return health.NewCheckReport(delta, health.CheckLevels(bundle), waiving.Waived(), verdict), nil
}

// policyVerdict compares the health policy of the model with the one of its
// manifest at the base revision. The manifest is read with git (no checkout);
// one that is not there, or a base that is not a commit, has the policy of an
// empty manifest, which is the default limits of the tier. The weakening is
// recorded when the head names an accepted policyDecision that the base did not
// already name. A manifest of the base that git cannot read or that is not
// YAML is a setup error: the comparison is never skipped.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func policyVerdict(opts CheckOptions, bundle model.Bundle, tier, baseRev string) (health.PolicyVerdict, error) {
	override := ""
	if strings.TrimSpace(opts.Tier) != "" {
		override = tier
	}
	basePolicy, err := baseManifestPolicy(opts.ManifestPath, baseRev)
	if err != nil {
		return health.PolicyVerdict{}, err
	}
	modelChanges, err := modelPolicyChanges(opts.ManifestPath, baseRev, opts.Head)
	if err != nil {
		return health.PolicyVerdict{}, err
	}
	changes := append(health.PolicyWeakenings(health.PolicyFromManifest(basePolicy, override), health.PolicyOf(bundle)), modelChanges...)
	health.SortPolicyChanges(changes)
	verdict := health.PolicyVerdict{Changes: changes, Level: health.PolicyLevelOf(bundle, baseLifecycle(basePolicy, override))}
	if len(changes) > 0 {
		verdict.Decision = recordingDecision(basePolicy, bundle)
	}
	return verdict, nil
}

// baseLifecycle is the life-cycle block of the base, at the tier override of
// the run when there is one; nil without a block.
//
// TRLC-LINKS: REQ-EMG-155
func baseLifecycle(basePolicy model.HealthPolicy, override string) *model.ManifestLifecycle {
	if basePolicy.Lifecycle == nil {
		return nil
	}
	copied := *basePolicy.Lifecycle
	if override != "" {
		copied.Tier = override
	}
	return &copied
}

// recordingDecision is the accepted policyDecision of the head that the base
// did not already name, or "".
//
// TRLC-LINKS: REQ-EMG-155
func recordingDecision(basePolicy model.HealthPolicy, bundle model.Bundle) string {
	named := ""
	if basePolicy.Health != nil {
		named = strings.TrimSpace(basePolicy.Health.PolicyDecision)
	}
	if decision := validate.AcceptedHealthPolicyDecision(bundle); decision != named {
		return decision
	}
	return ""
}

// baseManifestPolicy reads the lifecycle and health blocks of the manifest at
// the base revision; the zero policy when there is no base commit or the
// manifest was not in it.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func baseManifestPolicy(manifestPath, baseRev string) (model.HealthPolicy, error) {
	if baseRev == "" {
		return model.HealthPolicy{}, nil
	}
	abs, err := filepath.Abs(manifestPath)
	if err != nil {
		return model.HealthPolicy{}, fmt.Errorf("%w: resolve the manifest: %v", ErrSetup, err)
	}
	source, err := health.NewGitRepo(filepath.Dir(abs)).ShowFile(baseRev, filepath.Base(abs))
	if errors.Is(err, health.ErrNotInRev) {
		return model.HealthPolicy{}, nil
	}
	if err != nil {
		return model.HealthPolicy{}, fmt.Errorf("%w: read the manifest at the base revision: %v", ErrSetup, err)
	}
	policy, err := model.ParseHealthPolicy(source)
	if err != nil {
		return model.HealthPolicy{}, fmt.Errorf("%w: the manifest at the base revision: %v", ErrSetup, err)
	}
	return policy, nil
}

// modelPolicyChanges compares the model-level policy of the working tree (the
// life-cycle bundles and tier decision, the strictness switches, the health
// block, the POA&M defers, the tier and health decisions) and the health
// baseline with the manifest and documents at the base revision, read with git.
// With a head revision the head side is read at that revision, else from the
// working tree. Without a base commit, or with a manifest that was not in it, there is
// nothing to compare. A base that cannot be read is a setup error, never a
// skipped comparison.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func modelPolicyChanges(manifestPath, baseRev, headRev string) ([]health.PolicyChange, error) {
	if baseRev == "" {
		return nil, nil
	}
	abs, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve the manifest: %v", ErrSetup, err)
	}
	repo := health.NewGitRepo(filepath.Dir(abs))
	base, found, err := policyAtRevision(repo, baseRev, filepath.Base(abs))
	if err != nil || !found {
		return nil, err
	}
	head, err := policyToCheck(repo, headRev, abs)
	if err != nil {
		return nil, err
	}
	return health.ModelPolicyChanges(base, head), nil
}

// policyAtRevision reads the model policy at a revision; found is false when
// the manifest was not in it.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func policyAtRevision(repo health.GitRepo, rev, manifest string) (policy health.ModelPolicy, found bool, err error) {
	source, err := repo.ShowFile(rev, manifest)
	if errors.Is(err, health.ErrNotInRev) {
		return health.ModelPolicy{}, false, nil
	}
	if err != nil {
		return health.ModelPolicy{}, false, fmt.Errorf("%w: read the manifest at %s: %v", ErrSetup, rev, err)
	}
	policy, err = health.ParseModelPolicy(source, health.GitPolicyReader(repo, rev))
	if err != nil {
		return health.ModelPolicy{}, false, fmt.Errorf("%w: the model policy at %s: %v", ErrSetup, rev, err)
	}
	return policy, true, nil
}

// policyToCheck reads the model policy of the side that is checked: the
// working tree, or the revision when there is one.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func policyToCheck(repo health.GitRepo, headRev, manifestAbs string) (health.ModelPolicy, error) {
	if headRev != "" {
		policy, _, err := policyAtRevision(repo, headRev, filepath.Base(manifestAbs))
		return policy, err
	}
	source, err := health.ReadWorkTreeFile(manifestAbs)
	if err != nil {
		return health.ModelPolicy{}, fmt.Errorf("%w: read the manifest to check: %v", ErrSetup, err)
	}
	policy, err := health.ParseModelPolicy(source, health.WorkTreePolicyReader(repo.Dir))
	if err != nil {
		return health.ModelPolicy{}, fmt.Errorf("%w: the model policy of the working tree: %v", ErrSetup, err)
	}
	return policy, nil
}

// outOfScope returns the exclusion of the changed files: the code scope Scan
// uses (git-ignored files, nested modules, dot directories, node_modules,
// vendor, inferenceHints.exclude, health.exclude and the default excludes),
// applied to one root-relative path and its directories.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-156
func outOfScope(bundle model.Bundle, root string) func(rel string) bool {
	skip := codescan.CodeScopeSkip(bundle)
	h := bundle.Manifest.Health
	return func(rel string) bool {
		if health.Excluded(h, rel) {
			return true
		}
		dir := path.Dir(rel)
		for _, part := range ancestors(dir) {
			if codescan.SkipScanDir(path.Base(part)) || skip(filepath.Join(root, filepath.FromSlash(part)), true) {
				return true
			}
		}
		return skip(filepath.Join(root, filepath.FromSlash(rel)), false)
	}
}

// ancestors lists the directories of a root-relative directory path from the
// top: a/b/c gives a, a/b, a/b/c; "." gives none.
//
// TRLC-LINKS: REQ-EMG-156
func ancestors(dir string) []string {
	if dir == "." || dir == "" {
		return nil
	}
	var out []string
	parts := strings.Split(dir, "/")
	for i := range parts {
		out = append(out, strings.Join(parts[:i+1], "/"))
	}
	return out
}
