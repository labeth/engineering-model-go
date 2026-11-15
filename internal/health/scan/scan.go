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
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-151, REQ-EMG-152, REQ-EMG-153, REQ-EMG-154, REQ-EMG-155, REQ-EMG-157, REQ-EMG-158, REQ-EMG-159, REQ-EMG-160, REQ-EMG-209
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH, FLOW-HEALTH-SCAN, EVT-HEALTH-SCAN-REQUESTED
func Scan(opts Options) (health.Report, error) {
	run, err := setupRun(opts.ManifestPath, opts.Tier, opts.Paths)
	if err != nil {
		return health.Report{}, err
	}
	engine := cachedEngine(opts, run.root)
	levels := health.Levels(run.bundle)
	sc := run.analyzeSources(engine, levels)
	rollups := run.attribution(len(sc.results) > 0, levels[health.CodeRollupUnavailable])
	baseline, baselineDiags, err := loadBaseline(run.bundle, run.scope, opts.IgnoreBaseline)
	if err != nil {
		return health.Report{}, err
	}
	h := run.bundle.Manifest.Health
	var bundles []string
	if run.bundle.Manifest.Lifecycle != nil {
		bundles = run.bundle.Manifest.Lifecycle.EffectiveBundles()
	}
	return health.BuildReport(sc.results, health.Options{
		EngineName: engine.Name(), EngineVersion: engine.Version(),
		Tier: run.tier, Bundles: bundles, Levels: levels, Floor: model.EffectiveHealthFloor(h),
		Waivers: run.waivers(sc.files), Baseline: baseline, Owners: rollups.owners, Links: rollups.links,
		Diagnostics: append(append(sc.diags, rollups.diags...), baselineDiags...),
	}), nil
}

// projectRun is what a scan or a check resolves before it reads any source:
// the model at its tier, the project root and the scope of the paths.
type projectRun struct {
	bundle model.Bundle
	tier   string
	root   string
	scope  []string
}

// setupRun loads the model, applies the tier override and resolves the root
// and the scope of the paths; every failure is a setup error.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func setupRun(manifestPath, tierOverride string, paths []string) (projectRun, error) {
	bundle, err := model.LoadBundle(manifestPath)
	if err != nil {
		return projectRun{}, fmt.Errorf("%w: load model: %v", ErrSetup, err)
	}
	if t := strings.ToLower(strings.TrimSpace(tierOverride)); t != "" {
		if model.TierRank(t) == 0 {
			return projectRun{}, fmt.Errorf("%w: unknown tier %q; use one of %s", ErrSetup, tierOverride, strings.Join(model.Tiers, ", "))
		}
		bundle = health.WithTier(bundle, t)
	}
	tier, _, _ := validate.EffectiveTier(bundle)
	root, err := filepath.Abs(codescan.WithDefaultCodeRoot(bundle, nil)[0])
	if err != nil {
		return projectRun{}, fmt.Errorf("%w: resolve project root: %v", ErrSetup, err)
	}
	scope, err := resolveScope(root, paths)
	if err != nil {
		return projectRun{}, err
	}
	return projectRun{bundle: bundle, tier: tier, root: root, scope: scope}, nil
}

// scanned is what the sources of a scan yield: the files listed, the results
// of the ones analysed and the diagnostics of the ones that were not.
type scanned struct {
	files   []string
	results []health.FileResult
	diags   []health.Diagnostic
}

// analyzeSources lists, reads and analyses the sources of the scope.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-153
func (r projectRun) analyzeSources(engine health.Engine, levels map[string]string) scanned {
	files, diags := listSources(r, levels[CodeFileSkipped])
	inputs, readDiags := readSources(r.root, files, levels[CodeFileSkipped])
	sc := scanned{files: files, diags: append(diags, readDiags...), results: make([]health.FileResult, 0, len(inputs))}
	h := r.bundle.Manifest.Health
	for _, in := range inputs {
		res, err := analyzeFile(engine, in, health.ConfigFor(r.tier, h, in.Path))
		if err != nil {
			sc.diags = append(sc.diags, fileDiagnostic(health.Diagnostic{
				Code: CodeFileFailed, Level: levels[CodeFileFailed], Path: in.Path,
				Message: fmt.Sprintf("%s was not analysed: %v", shownPath(in.Path), err),
			}))
			continue
		}
		sc.results = append(sc.results, res)
	}
	return sc
}

// analyzeFile runs the engine on one file; an engine that does not return one
// result for it is an error.
//
// TRLC-LINKS: REQ-EMG-150
func analyzeFile(engine health.Engine, in health.FileInput, cfg health.Config) (health.FileResult, error) {
	res, err := engine.Analyze([]health.FileInput{in}, cfg)
	if err != nil {
		return health.FileResult{}, err
	}
	if len(res) != 1 {
		return health.FileResult{}, fmt.Errorf("engine returned %d results for one file", len(res))
	}
	return res[0], nil
}

// waivers are the accepted waivers of the manifest, limited to the files of a
// scan restricted to some paths.
//
// TRLC-LINKS: REQ-EMG-154
func (r projectRun) waivers(files []string) []model.HealthWaiver {
	h := r.bundle.Manifest.Health
	if h == nil {
		return nil
	}
	waivers := validate.AcceptedHealthWaivers(r.bundle, h.Waivers)
	if r.scope != nil {
		waivers = health.WaiversForFiles(waivers, files)
	}
	return waivers
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
	return entriesInScope(file.Entries, scope), nil, nil
}

// entriesInScope keeps the baseline entries of the files inside the scope.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func entriesInScope(entries []model.HealthBaselineEntry, scope []string) []model.HealthBaselineEntry {
	if scope == nil {
		return entries
	}
	var kept []model.HealthBaselineEntry
	for _, e := range entries {
		if inScope(scope, e.Path) {
			kept = append(kept, e)
		}
	}
	return kept
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

// fileDiagnostic is the diagnostic of a file that was not analysed: the one
// given, at the level of a warning when it names none.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func fileDiagnostic(d health.Diagnostic) health.Diagnostic {
	if d.Level == "" {
		d.Level = health.LevelWarning
	}
	return d
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
		rel, err := scopeEntry(root, p)
		if err != nil {
			return nil, err
		}
		scope = append(scope, rel)
	}
	return scope, nil
}

// scopeEntry is the root-relative, slash-separated form of one path option.
//
// TRLC-LINKS: REQ-EMG-158
func scopeEntry(root, p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("%w: path %q: %v", ErrSetup, p, err)
	}
	rel, err := filepath.Rel(root, abs)
	if outsideRoot(rel, err) {
		return "", fmt.Errorf("%w: path %q is outside the project root", ErrSetup, p)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("%w: path %q: %v", ErrSetup, p, err)
	}
	if info.Mode().IsRegular() && !supportedSource(abs) {
		return "", fmt.Errorf("%w: path %q is not a supported source file", ErrSetup, p)
	}
	return filepath.ToSlash(rel), nil
}

// outsideRoot reports a relative path that could not be made or leaves the
// root.
//
// TRLC-LINKS: REQ-EMG-158
func outsideRoot(rel string, err error) bool {
	return err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// supportedSource reports whether the file is in a language the scan reads.
//
// TRLC-LINKS: REQ-EMG-158
func supportedSource(p string) bool {
	_, ok := codemap.LanguageForPath(p)
	return ok
}

// underScope reports whether a root-relative path lies under one scope prefix.
//
// TRLC-LINKS: REQ-EMG-158
func underScope(rel, prefix string) bool {
	return prefix == "." || rel == prefix || strings.HasPrefix(rel, prefix+"/")
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
		if underScope(rel, s) {
			return true
		}
	}
	return false
}

// sourceWalk collects the source files of the code scope while the tree is
// walked.
type sourceWalk struct {
	skip  func(path string, isDir bool) bool
	h     *model.ManifestHealth
	root  string
	scope []string
	level string
	files []string
	diags []health.Diagnostic
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
func listSources(r projectRun, skippedLevel string) ([]string, []health.Diagnostic) {
	w := &sourceWalk{skip: codescan.CodeScopeSkip(r.bundle), h: r.bundle.Manifest.Health, root: r.root, scope: r.scope, level: skippedLevel}
	_ = filepath.WalkDir(r.root, w.visit)
	sort.Strings(w.files)
	return w.files, w.diags
}

// visit handles one entry of the walk.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-153
func (w *sourceWalk) visit(p string, d fs.DirEntry, err error) error {
	rel, relErr := filepath.Rel(w.root, p)
	if relErr != nil {
		return nil
	}
	rel = filepath.ToSlash(rel)
	if err != nil {
		w.unreadable(rel, err)
		return nil
	}
	if d.IsDir() {
		if p != w.root && w.skipDir(p, d, rel) {
			return filepath.SkipDir
		}
		return nil
	}
	w.addFile(p, d, rel)
	return nil
}

// unreadable reports a path the walk could not read, when the scope touches it.
//
// TRLC-LINKS: REQ-EMG-150
func (w *sourceWalk) unreadable(rel string, err error) {
	if scopeTouches(w.scope, rel) {
		w.diags = append(w.diags, fileDiagnostic(health.Diagnostic{
			Code: CodeFileSkipped, Level: w.level, Path: rel,
			Message: fmt.Sprintf("%s was not analysed: %s", shownPath(rel), walkProblem(err)),
		}))
	}
}

// skipDir reports a directory that is left out of the walk.
//
// TRLC-LINKS: REQ-EMG-153
func (w *sourceWalk) skipDir(p string, d fs.DirEntry, rel string) bool {
	return codescan.SkipScanDir(d.Name()) || w.skip(p, true) || health.Excluded(w.h, rel)
}

// wantFile reports a regular source file of a supported language that is in
// scope and not excluded.
//
// TRLC-LINKS: REQ-EMG-153
func (w *sourceWalk) wantFile(p string, d fs.DirEntry, rel string) bool {
	if !d.Type().IsRegular() || w.skip(p, false) {
		return false
	}
	if health.Excluded(w.h, rel) || !inScope(w.scope, rel) {
		return false
	}
	return supportedSource(rel)
}

// addFile lists a source file, or reports it when its path holds a control
// character.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-153
func (w *sourceWalk) addFile(p string, d fs.DirEntry, rel string) {
	if !w.wantFile(p, d, rel) {
		return
	}
	if health.HasControlChar(rel) {
		w.diags = append(w.diags, fileDiagnostic(health.Diagnostic{
			Code: CodeFileSkipped, Level: w.level, Path: rel,
			Message: fmt.Sprintf("%s was not analysed: the path holds a control character", shownPath(rel)),
		}))
		return
	}
	w.files = append(w.files, rel)
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
		if underScope(rel, s) || strings.HasPrefix(s, rel+"/") {
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
		diags = append(diags, fileDiagnostic(health.Diagnostic{
			Code: CodeFileSkipped, Level: skippedLevel, Path: rel,
			Message: fmt.Sprintf("%s was not analysed: %s", shownPath(rel), why),
		}))
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
func (r projectRun) attribution(have bool, level string) attributed {
	if !have {
		return attributed{}
	}
	items, scanDiags := inferCodeItems(r.bundle, "")
	owners, links := ownersAndLinks(items)
	return attributed{owners: owners, links: links, diags: rollupDiagnostics(scanDiags, r.root, level)}
}

// attributed is what the code scan says about ownership: the owner unit of
// each file, the marker-linked symbols, and the diagnostics of a scan that
// failed.
type attributed struct {
	owners map[string]string
	links  []health.SymbolLink
	diags  []health.Diagnostic
}

// rollupDiagnostics turns a failure of the code scan into a diagnostic of the
// given level, with the project root made relative.
//
// TRLC-LINKS: REQ-EMG-152, REQ-EMG-155
func rollupDiagnostics(scanDiags []validate.Diagnostic, root, level string) []health.Diagnostic {
	var diags []health.Diagnostic
	for _, d := range scanDiags {
		if d.Code == "code.scan_failed" || d.Code == "code.relative_path_failed" {
			msg := strings.ReplaceAll(d.Message, root, ".")
			diags = append(diags, fileDiagnostic(health.Diagnostic{
				Code: health.CodeRollupUnavailable, Level: level,
				Message: fmt.Sprintf("the unit and requirement roll-ups are unavailable or incomplete: %s: %s", d.Code, msg),
			}))
		}
	}
	return diags
}

// ownersAndLinks reads the owner unit of each source file and the linked
// symbols of production code from the items of the code scan.
//
// TRLC-LINKS: REQ-EMG-152
func ownersAndLinks(items []codescan.CodeItem) (map[string]string, []health.SymbolLink) {
	owners := map[string]string{}
	var links []health.SymbolLink
	for _, item := range items {
		if item.Kind == "source_file" {
			owners[filepath.ToSlash(item.Element)] = item.Owner
			continue
		}
		if link, ok := symbolLink(item); ok {
			links = append(links, link)
		}
	}
	return owners, links
}

// symbolLink is the marker-linked symbol of an item; false for any other item
// and for a symbol of test code, which verifies requirements and does not
// implement them.
//
// TRLC-LINKS: REQ-EMG-152
func symbolLink(item codescan.CodeItem) (health.SymbolLink, bool) {
	if item.Kind != "symbol" || len(item.Implements) == 0 || codescan.IsVerificationCodeItem(item) {
		return health.SymbolLink{}, false
	}
	p, line := codescan.SplitSourceLine(item.Source)
	return health.SymbolLink{Path: filepath.ToSlash(p), Line: line, Requirements: item.Implements}, true
}
