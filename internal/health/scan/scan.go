// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package scan

// The scan pipeline of the code health analysis (ADR-EMG-046, ADR-EMG-051):
// it loads the model, resolves the code scope the other tools use, runs the
// tree-sitter engine on every source file with the configuration of the
// manifest, scores, applies the waivers and the baseline, rolls up and sets the level of each
// finding at the project tier. It lives in its own package because it
// constructs the engine, which internal/health must never import.

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/health/engine/treesitter"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// MaxFileBytes is the size above which a source file is skipped with a
// health.file_skipped diagnostic: a file this large is generated or data.
const MaxFileBytes = 4 << 20

// Diagnostic codes of the pipeline. A file that cannot be analysed is visible
// in the report and never counts as a clean file: its diagnostic has the level
// of health.regression at the tier (an error from product), so it counts in
// the errors of the summary and fails scan --fail-on-error there.
const (
	CodeFileSkipped = health.CodeFileSkipped
	CodeFileFailed  = health.CodeFileFailed
)

// ErrSetup marks an error of the setup (the model does not load, a bad tier or
// path), as opposed to a finding. The CLI exits 2 on it.
var ErrSetup = errors.New("health scan setup")

// Options select what Scan analyses.
type Options struct {
	// ManifestPath is the engmod.yml of the model.
	ManifestPath string
	// Tier replaces the life-cycle tier of the model for the limits and the
	// levels; empty keeps the tier the manifest declares (or none).
	Tier string
	// Paths limit the scan to files and directories inside the project root;
	// empty scans the whole code scope.
	Paths []string
	// IgnoreBaseline scans as if the manifest had no health.baseline; it is how
	// enghealth baseline write and --check see every current finding.
	IgnoreBaseline bool
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

// Scan analyses the code of a model. A health block is optional: without one
// the default limits of the tier apply and nothing is excluded but the default
// excludes. The report is deterministic: sorted by path, no timestamp and no
// absolute path.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-151, REQ-EMG-152, REQ-EMG-153, REQ-EMG-154, REQ-EMG-155, REQ-EMG-157, REQ-EMG-158, REQ-EMG-159, REQ-EMG-160
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func Scan(opts Options) (health.Report, error) {
	bundle, err := model.LoadBundle(opts.ManifestPath)
	if err != nil {
		return health.Report{}, fmt.Errorf("%w: load model: %v", ErrSetup, err)
	}
	if tier := strings.ToLower(strings.TrimSpace(opts.Tier)); tier != "" {
		if model.TierRank(tier) == 0 {
			return health.Report{}, fmt.Errorf("%w: unknown tier %q; use one of %s", ErrSetup, opts.Tier, strings.Join(model.Tiers, ", "))
		}
		bundle = health.WithTier(bundle, tier)
	}
	tier, _, _ := validate.EffectiveTier(bundle)
	root, err := filepath.Abs(codescan.WithDefaultCodeRoot(bundle, nil)[0])
	if err != nil {
		return health.Report{}, fmt.Errorf("%w: resolve project root: %v", ErrSetup, err)
	}
	scope, err := resolveScope(root, opts.Paths)
	if err != nil {
		return health.Report{}, err
	}
	engine := cachedEngine(opts, root)

	h := bundle.Manifest.Health
	levels := health.Levels(bundle)
	files, diags := listSources(bundle, root, scope, levels[CodeFileSkipped])
	inputs, readDiags := readSources(root, files, levels[CodeFileSkipped])
	diags = append(diags, readDiags...)
	results := make([]health.FileResult, 0, len(inputs))
	for _, in := range inputs {
		res, err := engine.Analyze([]health.FileInput{in}, health.ConfigFor(tier, h, in.Path))
		if err != nil || len(res) != 1 {
			if err == nil {
				err = fmt.Errorf("engine returned %d results for one file", len(res))
			}
			diags = append(diags, fileDiagnostic(CodeFileFailed, levels[CodeFileFailed], in.Path, fmt.Sprintf("%s was not analysed: %v", shownPath(in.Path), err)))
			continue
		}
		results = append(results, res[0])
	}

	owners, links, rollupDiags := attribution(bundle, root, len(results) > 0, levels[health.CodeRollupUnavailable])
	diags = append(diags, rollupDiags...)
	var waivers []model.HealthWaiver
	if h != nil {
		waivers = validate.AcceptedHealthWaivers(bundle, h.Waivers)
		if scope != nil {
			waivers = health.WaiversForFiles(waivers, files)
		}
	}
	baseline, baselineDiags, err := loadBaseline(bundle, scope, opts.IgnoreBaseline)
	if err != nil {
		return health.Report{}, err
	}
	diags = append(diags, baselineDiags...)
	var bundles []string
	if bundle.Manifest.Lifecycle != nil {
		bundles = bundle.Manifest.Lifecycle.EffectiveBundles()
	}
	return health.BuildReport(results, health.Options{
		EngineName: engine.Name(), EngineVersion: engine.Version(),
		Tier: tier, Bundles: bundles, Levels: levels,
		Waivers: waivers, Baseline: baseline, Owners: owners, Links: links, Diagnostics: diags,
	}), nil
}

// loadBaseline reads the baseline file of the manifest (ADR-EMG-059). It
// returns nothing when the manifest has no health.baseline or the caller asked
// to ignore it. A baseline whose decision is missing or not accepted
// suppresses nothing: the file is not read and a health.waiver_unjustified
// diagnostic (an error at every tier) names the baseline. A missing or
// malformed file is a setup error. For a scan restricted to some paths only the
// entries inside them are kept, so an entry of a file that was not scanned is
// never reported as stale.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-155, REQ-EMG-158, REQ-EMG-161
func loadBaseline(bundle model.Bundle, scope []string, ignore bool) ([]model.HealthBaselineEntry, []health.Diagnostic, error) {
	h := bundle.Manifest.Health
	if ignore || h == nil || h.Baseline == nil {
		return nil, nil, nil
	}
	if problem := validate.HealthBaselineDecisionProblem(bundle); problem != "" {
		return nil, []health.Diagnostic{{
			Code: model.HealthWaiverUnjustifiedCode, Level: health.LevelError, Path: h.Baseline.File,
			Message: fmt.Sprintf("health baseline %s is unjustified: %s; a baseline needs an accepted decision, so it suppresses nothing", h.Baseline.File, problem),
		}}, nil
	}
	file, err := model.LoadHealthBaseline(bundle.ManifestPath, h.Baseline.File)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrSetup, err)
	}
	if scope == nil {
		return file.Entries, nil, nil
	}
	var kept []model.HealthBaselineEntry
	for _, e := range file.Entries {
		if inScope(scope, e.Path) {
			kept = append(kept, e)
		}
	}
	return kept, nil, nil
}

// cachedEngine returns the engine of a run: the tree-sitter engine (or the one
// in the options), behind the result cache when health.OpenCache allows it. A
// refusal that is not obvious is noted on opts.Notes.
//
// TRLC-LINKS: REQ-EMG-159
func cachedEngine(opts Options, root string) health.Engine {
	engine := opts.Engine
	if engine == nil {
		engine = treesitter.New()
	}
	cache, note := health.OpenCache(root, health.CachePolicy{Disabled: opts.NoCache, Force: opts.UseCache})
	if note != "" && opts.Notes != nil {
		fmt.Fprintln(opts.Notes, "enghealth: note:", note)
	}
	if cache == nil {
		return engine
	}
	return health.CachedEngine{Engine: engine, Cache: cache}
}

// shownPath is a path for a message: quoted when it holds a control character.
//
// TRLC-LINKS: REQ-EMG-150
func shownPath(rel string) string {
	if health.HasControlChar(rel) {
		return strconv.Quote(rel)
	}
	return rel
}

// fileDiagnostic is the diagnostic of a file that was not analysed.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func fileDiagnostic(code, level, rel, message string) health.Diagnostic {
	if level == "" {
		level = health.LevelWarning
	}
	return health.Diagnostic{Code: code, Level: level, Path: rel, Message: message}
}

// resolveScope turns the Paths option into root-relative path prefixes. It
// returns nil for a scan of the whole project. A path outside the root, or
// one that does not exist, is a setup error.
//
// TRLC-LINKS: REQ-EMG-158
func resolveScope(root string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	scope := make([]string, 0, len(paths))
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("%w: path %q: %v", ErrSetup, p, err)
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("%w: path %q is outside the project root", ErrSetup, p)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("%w: path %q: %v", ErrSetup, p, err)
		}
		if info.Mode().IsRegular() {
			if _, ok := codemap.LanguageForPath(abs); !ok {
				return nil, fmt.Errorf("%w: path %q is not a supported source file", ErrSetup, p)
			}
		}
		scope = append(scope, filepath.ToSlash(rel))
	}
	return scope, nil
}

// inScope reports whether a root-relative file path lies under one of the
// scope prefixes; a nil scope holds everything.
//
// TRLC-LINKS: REQ-EMG-158
func inScope(scope []string, rel string) bool {
	if scope == nil {
		return true
	}
	for _, s := range scope {
		if s == "." || rel == s || strings.HasPrefix(rel, s+"/") {
			return true
		}
	}
	return false
}

// listSources lists the root-relative paths (sorted) of the source files of
// the code scope: the walk of the code scan (git-ignored files, nested modules,
// inferenceHints.exclude, dot directories, node_modules and vendor left out)
// minus health.exclude and the default excludes, in supported languages. A
// file whose path holds a control character is not listed but reported as a
// health.file_skipped diagnostic of the given level, since such a name would
// reach logs and terminals. A directory that cannot be read is reported the
// same way, so the files in it never read as clean.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-153
func listSources(bundle model.Bundle, root string, scope []string, skippedLevel string) ([]string, []health.Diagnostic) {
	skip := codescan.CodeScopeSkip(bundle)
	h := bundle.Manifest.Health
	var files []string
	var diags []health.Diagnostic
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if err != nil {
			if scopeTouches(scope, rel) {
				diags = append(diags, fileDiagnostic(CodeFileSkipped, skippedLevel, rel, fmt.Sprintf("%s was not analysed: %s", shownPath(rel), walkProblem(err))))
			}
			return nil
		}
		if d.IsDir() {
			if p != root && (codescan.SkipScanDir(d.Name()) || skip(p, true) || health.Excluded(h, rel)) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || skip(p, false) || health.Excluded(h, rel) || !inScope(scope, rel) {
			return nil
		}
		if _, ok := codemap.LanguageForPath(rel); !ok {
			return nil
		}
		if health.HasControlChar(rel) {
			diags = append(diags, fileDiagnostic(CodeFileSkipped, skippedLevel, rel, fmt.Sprintf("%s was not analysed: the path holds a control character", shownPath(rel))))
			return nil
		}
		files = append(files, rel)
		return nil
	})
	sort.Strings(files)
	return files, diags
}

// scopeTouches reports whether a directory or file lies under a scope prefix or
// holds one, so a problem with it can matter to a scan restricted to the scope.
//
// TRLC-LINKS: REQ-EMG-158
func scopeTouches(scope []string, rel string) bool {
	if scope == nil || rel == "." {
		return true
	}
	for _, s := range scope {
		if s == "." || rel == s || strings.HasPrefix(rel, s+"/") || strings.HasPrefix(s, rel+"/") {
			return true
		}
	}
	return false
}

// walkProblem says why the walk could not read a path, without the absolute
// path the system error holds.
//
// TRLC-LINKS: REQ-EMG-150
func walkProblem(err error) string {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return "cannot be read: " + err.Error()
}

// readSources reads the files; one that is too large, not a regular file or
// unreadable becomes a diagnostic of the given level and is not analysed.
//
// TRLC-LINKS: REQ-EMG-150
func readSources(root string, files []string, skippedLevel string) ([]health.FileInput, []health.Diagnostic) {
	inputs := make([]health.FileInput, 0, len(files))
	var diags []health.Diagnostic
	skipped := func(rel, why string) {
		diags = append(diags, fileDiagnostic(CodeFileSkipped, skippedLevel, rel, fmt.Sprintf("%s was not analysed: %s", shownPath(rel), why)))
	}
	for _, rel := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(abs)
		switch {
		case err != nil:
			skipped(rel, "cannot be read")
			continue
		case !info.Mode().IsRegular():
			skipped(rel, "is not a regular file")
			continue
		case info.Size() > MaxFileBytes:
			skipped(rel, fmt.Sprintf("larger than %d bytes", MaxFileBytes))
			continue
		}
		src, err := readCapped(abs)
		if err != nil {
			skipped(rel, err.Error())
			continue
		}
		lang, _ := codemap.LanguageForPath(rel)
		inputs = append(inputs, health.FileInput{Path: rel, Language: lang, Source: src})
	}
	return inputs, diags
}

// readCapped reads a file but never more than MaxFileBytes, so a file that
// grew after its size was checked is skipped and not read whole.
//
// TRLC-LINKS: REQ-EMG-150
func readCapped(abs string) ([]byte, error) {
	f, err := os.Open(abs)
	if err != nil {
		return nil, errors.New("cannot be read")
	}
	defer f.Close()
	src, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	switch {
	case err != nil:
		return nil, errors.New("cannot be read")
	case len(src) > MaxFileBytes:
		return nil, fmt.Errorf("larger than %d bytes", MaxFileBytes)
	}
	return src, nil
}

// inferCodeItems is the code scan of the model; a test replaces it to make the
// scan fail.
var inferCodeItems = codescan.InferCodeItems

// attribution reads the traceability the roll-ups follow from the code scan of
// the model: the file-level ENGMODEL-OWNER-UNIT of each file and the
// marker-linked symbols with the requirements of their TRLC-LINKS. Symbols of
// test code are left out; they verify requirements, they do not implement
// them. When the code scan of the model reports that it failed, the roll-ups
// are unavailable or incomplete and a health.rollup_unavailable diagnostic of
// the given level says so, instead of empty roll-ups that look like a project
// without ownership.
//
// TRLC-LINKS: REQ-EMG-152, REQ-EMG-155
func attribution(bundle model.Bundle, root string, have bool, level string) (map[string]string, []health.SymbolLink, []health.Diagnostic) {
	if !have {
		return nil, nil, nil
	}
	items, scanDiags := inferCodeItems(bundle, "")
	var diags []health.Diagnostic
	for _, d := range scanDiags {
		if d.Code == "code.scan_failed" || d.Code == "code.relative_path_failed" {
			msg := strings.ReplaceAll(d.Message, root, ".")
			diags = append(diags, fileDiagnostic(health.CodeRollupUnavailable, level, "", fmt.Sprintf("the unit and requirement roll-ups are unavailable or incomplete: %s: %s", d.Code, msg)))
		}
	}
	owners := map[string]string{}
	var links []health.SymbolLink
	for _, item := range items {
		switch item.Kind {
		case "source_file":
			owners[filepath.ToSlash(item.Element)] = item.Owner
		case "symbol":
			if len(item.Implements) == 0 || codescan.IsVerificationCodeItem(item) {
				continue
			}
			p, line := codescan.SplitSourceLine(item.Source)
			links = append(links, health.SymbolLink{Path: filepath.ToSlash(p), Line: line, Requirements: item.Implements})
		}
	}
	return owners, links, diags
}
