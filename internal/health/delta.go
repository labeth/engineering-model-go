// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

// The delta check (ADR-EMG-053): it classifies the findings of the changed
// files against a base revision, so that a gate can fail on a regression
// (health.regression) and never on debt that was already there.
//
// A finding is keyed by (path, qualified name, code). It is
//
//	new       absent at base;
//	worsened  present at base, and at head either of a higher severity or with
//	          a metric value that grew by at least 20 percent
//	          (value >= ceil(base * 1.2));
//	existing  everything else (it never blocks).
//
// A function renamed or moved with the same body hash (BodyHash) is not new:
// its findings are existing. Deleted files and functions produce nothing. The
// report holds no timestamp and no absolute path, and is sorted.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/model"
)

// DeltaSchema is the schema string of the delta report.
const DeltaSchema = "engmod-health-delta/1"

// WorkingTree is the Head of a report that compares against the working tree.
const WorkingTree = "working-tree"

// The change classes of a delta entry.
const (
	ChangeNew      = "new"
	ChangeWorsened = "worsened"
	ChangeExisting = "existing"
)

// maxDeltaFileBytes is the largest file the check reads, on either side and
// whether it comes from the working tree or from a revision; a larger one is
// skipped. A skipped head file is listed in the report and blocks where a
// regression blocks (S2 of the review: a file the check cannot read must not be
// a way to hide findings).
const maxDeltaFileBytes = 4 << 20

// ErrBaseUnavailable reports that the base revision cannot be read: it is
// unknown, or the clone is shallow and does not hold it. It is a setup error
// (the command exits 2), never a pass.
var ErrBaseUnavailable = errors.New("base revision unavailable")

// DeltaEntry is one finding of a changed file with its change class.
type DeltaEntry struct {
	Path     string `json:"path"`
	Function string `json:"function"`
	Code     string `json:"code"`
	// Change is new, worsened or existing.
	Change   string   `json:"change"`
	Severity Severity `json:"severity"`
	// SeverityBefore is the severity at base; empty for a new finding.
	SeverityBefore Severity `json:"severityBefore,omitempty"`
	Metric         string   `json:"metric"`
	// ValueBefore is the metric at base; 0 for a new finding.
	ValueBefore int    `json:"valueBefore"`
	ValueAfter  int    `json:"valueAfter"`
	StartLine   int    `json:"startLine"`
	EndLine     int    `json:"endLine"`
	Message     string `json:"message"`
}

// FileDelta is the score of one changed file before and after. A file that is
// new has no before; a file that is deleted is not listed.
type FileDelta struct {
	Path string `json:"path"`
	// OldPath is the path at base when the file was renamed.
	OldPath     string   `json:"oldPath,omitempty"`
	Status      string   `json:"status"`
	ScoreBefore *float64 `json:"scoreBefore,omitempty"`
	ScoreAfter  *float64 `json:"scoreAfter,omitempty"`
	BandBefore  string   `json:"bandBefore,omitempty"`
	BandAfter   string   `json:"bandAfter,omitempty"`
}

// SkippedFile is a changed file at head that the check did not analyse. Its
// findings are unknown, so it is not a clean file.
type SkippedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// DeltaReport is the result of Check.
type DeltaReport struct {
	Schema string `json:"schema"`
	Tier   string `json:"tier,omitempty"`
	// Base is the commit that was compared against: the merge base of the
	// requested base and the head (HEAD for the working tree); empty in a
	// repository without commits.
	Base string `json:"base"`
	// Head is the head commit, or "working-tree".
	Head string `json:"head"`
	// Notes are warnings the reader must see, such as an empty repository.
	Notes   []string     `json:"notes"`
	Files   []FileDelta  `json:"files"`
	Entries []DeltaEntry `json:"entries"`
	// Blocking are the entries that are new or worsened with severity high:
	// the health.regression findings.
	Blocking []DeltaEntry `json:"blocking"`
	// Skipped are the changed files at head that were not analysed because
	// they are too large or not regular files.
	Skipped []SkippedFile `json:"skipped"`
	// BelowFloor are the changed files at head whose score is below the floor
	// and that no baseline entry covers (health.file_below_floor, ADR-EMG-087);
	// absent when there is none, or when the check was given no floor.
	BelowFloor []FloorFile `json:"belowFloor,omitempty"`
}

// CheckOptions configures Check.
type CheckOptions struct {
	// Repo is the work tree; paths are relative to Repo.Dir.
	Repo GitRepo
	// Base is the revision to compare with; empty means HEAD (uncommitted
	// work). "base...head" is accepted and sets Head as well. The comparison is
	// always against the merge base of Base and the head (HEAD for the working
	// tree), never against a newer tip of Base.
	Base string
	// Head is the revision to check; empty means the working tree.
	Head string
	// Paths restricts the check to these files or directories.
	Paths []string
	// Engine analyses the files; the caller may wrap it in a CachedEngine.
	Engine Engine
	// ConfigFor gives the effective Config of a path; the same Config is used
	// for both sides. Nil means the default thresholds of Tier.
	ConfigFor func(path string) Config
	// Scorer scores the files of the report; nil leaves the scores out.
	Scorer Scorer
	// Exclude skips a path when it returns true.
	Exclude func(path string) bool
	// Tier is the life-cycle tier; it selects the default thresholds when
	// ConfigFor is nil and is echoed in the report.
	Tier string
	// Floor is the score every changed file at head needs unless the baseline
	// covers it (ADR-EMG-087); 0 or a nil Scorer asks for no floor. Baseline
	// holds the entries that cover a file; a regression is still never
	// baselined.
	Floor    float64
	Baseline []model.HealthBaselineEntry
}

// LanguageOf returns the language of a path by its extension (the mapping of
// the code scan, codemap.LanguageForPath); false when the delta check does not
// analyse such a file.
//
// TRLC-LINKS: REQ-EMG-156
func LanguageOf(p string) (string, bool) { return codemap.LanguageForPath(p) }

// Regressions returns the entries that block: new or worsened with severity
// high. They are the health.regression findings.
//
// TRLC-LINKS: REQ-EMG-156
func Regressions(r DeltaReport) []DeltaEntry {
	out := []DeltaEntry{}
	for _, e := range r.Entries {
		if e.Severity == SeverityHigh && (e.Change == ChangeNew || e.Change == ChangeWorsened) {
			out = append(out, e)
		}
	}
	return out
}

// severityRank orders the severities.
//
// TRLC-LINKS: REQ-EMG-156
func severityRank(s Severity) int {
	switch s {
	case SeverityWarn:
		return 1
	case SeverityHigh:
		return 2
	}
	return 0
}

// classify compares one finding at base with the same finding at head.
//
// TRLC-LINKS: REQ-EMG-156
func classify(before, after Finding) string {
	if severityRank(after.Severity) > severityRank(before.Severity) {
		return ChangeWorsened
	}
	if after.Value > before.Value && after.Value >= (before.Value*12+9)/10 {
		return ChangeWorsened
	}
	return ChangeExisting
}

// deltaFile is one changed file with both sides. A side that is absent has an
// empty language.
type deltaFile struct {
	status             string
	basePath, headPath string
	baseLang, headLang string
	baseSrc, headSrc   []byte
	base, head         *FileResult
}

// path is the path the file is reported under.
//
// TRLC-LINKS: REQ-EMG-156
func (f *deltaFile) path() string {
	if f.headLang != "" {
		return f.headPath
	}
	return f.basePath
}

// checker holds the state of one Check call.
type checker struct {
	opts    CheckOptions
	baseRev string // "" when there is no base commit
	headRev string // "" for the working tree
	notes   []string
	skipped []SkippedFile
}

// Check classifies the findings of the files that changed between Base and
// Head. See the package comment of delta.go for the rules.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-159
func Check(opts CheckOptions) (DeltaReport, error) {
	rep := DeltaReport{Schema: DeltaSchema, Tier: opts.Tier, Head: WorkingTree}
	if opts.Engine == nil {
		return rep, errors.New("health: the delta check needs an engine")
	}
	c := &checker{opts: opts}
	changed, err := c.plan()
	if err != nil {
		return rep, err
	}
	rep.Base = c.baseRev
	if c.headRev != "" {
		rep.Head = c.headRev
	}
	files, err := c.selectFiles(changed)
	if err != nil {
		return rep, err
	}
	for _, f := range files {
		if err := c.load(f); err != nil {
			return rep, err
		}
	}
	c.listNewSyntaxErrors(files)
	rep.Entries = compareFiles(files)
	rep.Files = c.fileDeltas(files)
	rep.Blocking = Regressions(DeltaReport{Entries: rep.Entries})
	rep.BelowFloor = c.belowFloor(files)
	sort.Strings(c.notes)
	rep.Notes = append([]string{}, c.notes...)
	sort.Slice(c.skipped, func(i, j int) bool { return c.skipped[i].Path < c.skipped[j].Path })
	rep.Skipped = append([]SkippedFile{}, c.skipped...)
	return rep, nil
}

// belowFloor lists the head files whose score is below the floor and that the
// baseline does not cover, sorted by path (the files are).
//
// TRLC-LINKS: REQ-EMG-210
func (c *checker) belowFloor(files []*deltaFile) []FloorFile {
	if c.opts.Scorer == nil || c.opts.Floor <= 0 {
		return nil
	}
	var out []FloorFile
	for _, f := range files {
		if f.head == nil {
			continue
		}
		if file, below := c.floorFile(f.head); below {
			out = append(out, file)
		}
	}
	return out
}

// floorFile describes a head file whose score is below the floor and that the
// baseline does not cover; false for any other file.
//
// TRLC-LINKS: REQ-EMG-210
func (c *checker) floorFile(head *FileResult) (FloorFile, bool) {
	score := c.opts.Scorer(*head)
	if !BelowFloor(score.Score, c.opts.Floor) || BaselineCovered([]FileResult{*head}, c.opts.Baseline)[head.Path] {
		return FloorFile{}, false
	}
	report := FileReport{Path: head.Path, Score: score.Score, Deductions: score.Deductions, Functions: head.Functions}
	return NewFloorFile(report, c.opts.Floor), true
}

// listNewSyntaxErrors lists a head file the parser only partly read, when the
// change brought the errors in (the base was clean or had fewer): the functions
// the parser lost are not measured, so the file must not read as clean. It is
// listed like a skipped file and fails where a regression fails. A file that
// was as broken at base is not listed again.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func (c *checker) listNewSyntaxErrors(files []*deltaFile) {
	for _, f := range files {
		if n := f.newSyntaxErrors(); n > 0 {
			c.skip(f.headPath, fmt.Sprintf("%d syntax errors: the functions the parser lost are not measured", n))
		}
	}
}

// newSyntaxErrors is the number of syntax errors at head that the change
// brought in; 0 when the head is clean or the base had as many.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func (f *deltaFile) newSyntaxErrors() int {
	if f.head == nil || f.head.SyntaxErrors == 0 {
		return 0
	}
	if f.base != nil && f.base.SyntaxErrors >= f.head.SyntaxErrors {
		return 0
	}
	return f.head.SyntaxErrors
}

// splitRange applies the defaults and splits "base...head".
//
// TRLC-LINKS: REQ-EMG-156
func splitRange(base, head string) (string, string, error) {
	if base == "" {
		base = "HEAD"
	}
	if i := strings.Index(base, "..."); i >= 0 {
		if head != "" {
			return "", "", fmt.Errorf("%w: %q already names a head, so Head %q must be empty", ErrInvalidRev, base, head)
		}
		base, head = base[:i], base[i+3:]
		if base == "" {
			base = "HEAD"
		}
		if head == "" {
			head = "HEAD"
		}
	}
	return base, head, nil
}

// plan resolves the revisions and lists the changed files.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) plan() ([]ChangedFile, error) {
	base, head, err := splitRange(c.opts.Base, c.opts.Head)
	if err != nil {
		return nil, err
	}
	has, err := c.opts.Repo.HasCommits()
	if err != nil {
		return nil, err
	}
	if !has {
		return c.planWithoutCommits(base, head)
	}
	baseSha, err := c.opts.Repo.Resolve(base)
	if err != nil {
		return nil, c.baseError(base, err)
	}
	c.baseRev = baseSha
	if head == "" {
		return c.planWorkTree(base, baseSha)
	}
	return c.planRange(base, baseSha, head)
}

// planWithoutCommits lists every file of a repository that has no commits.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) planWithoutCommits(base, head string) ([]ChangedFile, error) {
	if head != "" || base != "HEAD" {
		return nil, fmt.Errorf("%w: base revision %q does not exist, the repository has no commits", ErrBaseUnavailable, base)
	}
	c.notes = append(c.notes, "the repository has no commits: there is no base, so every finding is new")
	return c.opts.Repo.ChangedFiles("", "")
}

// planWorkTree lists the changes of the working tree against the base.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) planWorkTree(base, baseSha string) ([]ChangedFile, error) {
	if err := c.useMergeBaseWithHead(base, baseSha); err != nil {
		return nil, err
	}
	return c.opts.Repo.ChangedFiles(c.baseRev, "")
}

// planRange lists the changes of base...head.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) planRange(base, baseSha, head string) ([]ChangedFile, error) {
	var err error
	if c.headRev, err = c.opts.Repo.Resolve(head); err != nil {
		return nil, err
	}
	if c.baseRev, err = c.opts.Repo.MergeBase(baseSha, c.headRev); err != nil {
		return nil, c.baseError(base, err)
	}
	return c.opts.Repo.ChangedFiles(c.baseRev, c.headRev)
}

// useMergeBaseWithHead sets the base of a working-tree comparison to the merge
// base of the resolved base and HEAD, like the three-dot form does for a head
// revision: a branch that is behind its target (the target moved on and fixed a
// smell) must not see the later work of the target as a regression of its own.
// Without a common ancestor the resolved base is kept and a note says so; in a
// shallow clone the missing ancestor is a setup error, never a quiet fallback.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) useMergeBaseWithHead(base, baseSha string) error {
	mb, err := c.opts.Repo.MergeBase(baseSha, "HEAD")
	switch {
	case err == nil:
		if mb != baseSha {
			c.baseRev = mb
			c.notes = append(c.notes, fmt.Sprintf("compared against the merge base %s of %s (%s) and HEAD, not against the tip of %s", shortSha(mb), base, shortSha(baseSha), base))
		}
		return nil
	case errors.Is(err, ErrUnknownRev):
		if shallow, serr := c.opts.Repo.IsShallow(); serr == nil && shallow {
			return c.baseError(base, err)
		}
		c.notes = append(c.notes, fmt.Sprintf("%s and HEAD have no common ancestor: compared against the tip of %s (%s)", base, base, shortSha(baseSha)))
		return nil
	}
	return c.baseError(base, err)
}

// shortSha shortens an object id for a note.
//
// TRLC-LINKS: REQ-EMG-156
func shortSha(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// baseError turns a failure to reach the base into ErrBaseUnavailable with a
// message that says what to do.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) baseError(base string, err error) error {
	if errors.Is(err, ErrInvalidRev) {
		return err
	}
	if shallow, serr := c.opts.Repo.IsShallow(); serr == nil && shallow {
		return fmt.Errorf("%w: base revision %q is not in this shallow clone; fetch more history (git fetch --deepen, git fetch --unshallow, or fetch-depth: 0 in CI): %w", ErrBaseUnavailable, base, err)
	}
	return fmt.Errorf("%w: base revision %q cannot be resolved: %w", ErrBaseUnavailable, base, err)
}

// selectFiles keeps the changed files that are in scope and supported, as
// deltaFiles sorted by path.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) selectFiles(changed []ChangedFile) ([]*deltaFile, error) {
	scope, err := c.scope()
	if err != nil {
		return nil, err
	}
	var files []*deltaFile
	for _, cf := range changed {
		if f := newDeltaFile(cf); c.wanted(f, scope) {
			files = append(files, f)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].path() != files[j].path() {
			return files[i].path() < files[j].path()
		}
		return files[i].basePath < files[j].basePath
	})
	return files, nil
}

// scope is the cleaned list of the paths a check is limited to; nil for all.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) scope() ([]string, error) {
	var scope []string
	for _, p := range c.opts.Paths {
		clean, err := normalizeRepoPath(p)
		if err != nil {
			return nil, err
		}
		scope = append(scope, clean)
	}
	return scope, nil
}

// inScope reports whether a path is in a scope; every path is in a nil one.
//
// TRLC-LINKS: REQ-EMG-156
func inScope(scope []string, p string) bool {
	if scope == nil {
		return true
	}
	for _, s := range scope {
		if p == s || strings.HasPrefix(p, s+"/") {
			return true
		}
	}
	return false
}

// newDeltaFile builds the file of a changed entry, with the language of each
// side that exists.
//
// TRLC-LINKS: REQ-EMG-156
func newDeltaFile(cf ChangedFile) *deltaFile {
	f := &deltaFile{status: cf.Status, headPath: cf.Path, basePath: cf.Path}
	if cf.Status == "R" {
		f.basePath = cf.OldPath
	}
	f.headLang, _ = LanguageOf(f.headPath)
	f.baseLang, _ = LanguageOf(f.basePath)
	switch cf.Status {
	case "A":
		f.baseLang = ""
	case "D":
		f.headLang = ""
	}
	return f
}

// wanted reports whether a file is supported, in scope and not excluded.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) wanted(f *deltaFile, scope []string) bool {
	if f.headLang == "" && f.baseLang == "" {
		return false
	}
	if !(inScope(scope, f.headPath) || inScope(scope, f.basePath)) {
		return false
	}
	return c.opts.Exclude == nil || !c.opts.Exclude(f.path())
}

// config is the effective Config of a file.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) config(f *deltaFile) Config {
	if c.opts.ConfigFor != nil {
		return c.opts.ConfigFor(f.path())
	}
	return Config{Thresholds: DefaultThresholdsForTier(c.opts.Tier)}
}

// load reads and analyses both sides of f.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) load(f *deltaFile) error {
	cfg := c.config(f)
	if err := c.readHeadSide(f); err != nil {
		return err
	}
	if err := c.readBaseSide(f); err != nil {
		return err
	}
	if f.headLang != "" && c.analyzeHead(f, cfg) {
		return nil
	}
	c.analyzeBase(f, cfg)
	return nil
}

// readHeadSide reads the head source of f; a head that is not there or is
// skipped has no language afterwards.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) readHeadSide(f *deltaFile) error {
	if f.headLang == "" {
		return nil
	}
	src, err := c.readHead(f.headPath)
	if err != nil {
		return err
	}
	if src == nil {
		f.headLang = ""
	} else {
		f.headSrc = src
	}
	return nil
}

// readBaseSide reads the base source of f; a base that is not there or is
// skipped has no language afterwards.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) readBaseSide(f *deltaFile) error {
	if f.baseLang == "" {
		return nil
	}
	src, err := c.readBase(f.basePath)
	if err != nil {
		return err
	}
	if src == nil {
		f.baseLang = ""
	} else {
		f.baseSrc = src
	}
	return nil
}

// analyzeHead analyses the head of f. A file the engine cannot analyse is
// skipped like one that is too large: its findings are unknown, so the report
// lists it and fails where a regression fails, instead of ending the whole
// check. It reports whether the analysis failed.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) analyzeHead(f *deltaFile, cfg Config) bool {
	res, err := c.analyze(FileInput{Path: f.headPath, Language: f.headLang, Source: f.headSrc}, cfg)
	if err != nil {
		c.skip(f.headPath, "the analysis failed: "+err.Error())
		f.headLang = ""
		return true
	}
	f.head = res
	return false
}

// sameSides reports whether the base of f is the head, unchanged.
//
// TRLC-LINKS: REQ-EMG-156
func (f *deltaFile) sameSides() bool {
	return f.head != nil && f.sameInputs()
}

// sameInputs reports whether both sides of f have the same path, language and
// source.
//
// TRLC-LINKS: REQ-EMG-156
func (f *deltaFile) sameInputs() bool {
	return f.headPath == f.basePath && f.headLang == f.baseLang && string(f.headSrc) == string(f.baseSrc)
}

// analyzeBase analyses the base of f, or reuses the head result when both are
// the same; a base the engine cannot analyse is noted and counts as absent.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) analyzeBase(f *deltaFile, cfg Config) {
	if f.baseLang == "" {
		return
	}
	if f.sameSides() {
		f.base = f.head
		return
	}
	res, err := c.analyze(FileInput{Path: f.basePath, Language: f.baseLang, Source: f.baseSrc}, cfg)
	if err != nil {
		c.notes = append(c.notes, "could not analyse the base of "+f.basePath+": "+err.Error()+"; every finding of the file counts as new")
		f.baseLang = ""
		return
	}
	f.base = res
}

// readHead returns the head content of a file; nil when it is not there or is
// skipped (a note and an entry of the skipped list say why). The size limit
// applies to a file of the working tree and to one read from a revision alike.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) readHead(p string) ([]byte, error) {
	if c.headRev != "" {
		return c.readHeadRev(p)
	}
	return c.readHeadWorkTree(p)
}

// readHeadRev reads a head file from the head revision.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) readHeadRev(p string) ([]byte, error) {
	src, err := c.opts.Repo.ShowFile(c.headRev, p)
	if errors.Is(err, ErrNotInRev) {
		return nil, nil
	}
	if err == nil && len(src) > maxDeltaFileBytes {
		c.skip(p, "larger than "+strconv.Itoa(maxDeltaFileBytes)+" bytes")
		return nil, nil
	}
	return src, err
}

// readHeadWorkTree reads a head file from the working tree.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) readHeadWorkTree(p string) ([]byte, error) {
	full := filepath.Join(c.opts.Repo.Dir, filepath.FromSlash(p))
	info, err := os.Lstat(full)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	case !info.Mode().IsRegular():
		c.skip(p, "not a regular file")
		return nil, nil
	case info.Size() > maxDeltaFileBytes:
		c.skip(p, "larger than "+strconv.Itoa(maxDeltaFileBytes)+" bytes")
		return nil, nil
	}
	return os.ReadFile(full)
}

// skip records a head file that was not analysed.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) skip(p, reason string) {
	c.notes = append(c.notes, "skipped "+p+": "+reason)
	c.skipped = append(c.skipped, SkippedFile{Path: p, Reason: reason})
}

// readBase returns the base content of a file; nil when there is no base or
// the file was not in it.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) readBase(p string) ([]byte, error) {
	if c.baseRev == "" {
		return nil, nil
	}
	src, err := c.opts.Repo.ShowFile(c.baseRev, p)
	if errors.Is(err, ErrNotInRev) {
		return nil, nil
	}
	if err == nil && len(src) > maxDeltaFileBytes {
		c.notes = append(c.notes, "skipped the base of "+p+": larger than "+strconv.Itoa(maxDeltaFileBytes)+" bytes")
		return nil, nil
	}
	return src, err
}

// analyze runs the engine on one file.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) analyze(in FileInput, cfg Config) (*FileResult, error) {
	res, err := c.opts.Engine.Analyze([]FileInput{in}, cfg)
	if err != nil {
		return nil, fmt.Errorf("health: %s: %w", in.Path, err)
	}
	if len(res) != 1 {
		return nil, fmt.Errorf("health: engine %s returned %d results for %s", c.opts.Engine.Name(), len(res), in.Path)
	}
	return &res[0], nil
}

// fileDeltas lists the score of every file that exists at head.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) fileDeltas(files []*deltaFile) []FileDelta {
	out := []FileDelta{}
	for _, f := range files {
		if f.head != nil {
			out = append(out, c.fileDelta(f))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// fileDelta is the score change of one file that exists at head.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) fileDelta(f *deltaFile) FileDelta {
	fd := FileDelta{Path: f.headPath, Status: f.status}
	if f.status == "R" {
		fd.OldPath = f.basePath
	}
	if c.opts.Scorer != nil {
		c.addScores(&fd, f)
	}
	return fd
}

// addScores sets the scores of the head of f, and of its base when it has one.
//
// TRLC-LINKS: REQ-EMG-156
func (c *checker) addScores(fd *FileDelta, f *deltaFile) {
	after := c.opts.Scorer(*f.head)
	fd.ScoreAfter, fd.BandAfter = &after.Score, after.Band
	if f.base != nil {
		before := c.opts.Scorer(*f.base)
		fd.ScoreBefore, fd.BandBefore = &before.Score, before.Band
	}
}

// funcRef is one function of one file side.
type funcRef struct {
	file *deltaFile
	fn   *Function
	key  string
	hash string
}

// functionRefs lists the functions of a result with keys that stay unique
// when a file has several functions of one name (the n-th pairs with the n-th).
//
// TRLC-LINKS: REQ-EMG-156
func functionRefs(f *deltaFile, res *FileResult) []*funcRef {
	if res == nil {
		return nil
	}
	seen := map[string]int{}
	refs := make([]*funcRef, 0, len(res.Functions))
	for i := range res.Functions {
		fn := &res.Functions[i]
		n := seen[fn.Name]
		seen[fn.Name]++
		refs = append(refs, &funcRef{file: f, fn: fn, key: fn.Name + "#" + strconv.Itoa(n)})
	}
	return refs
}

// bodyHash computes the hash of the function once, from the side of its file.
//
// TRLC-LINKS: REQ-EMG-156
func (r *funcRef) bodyHash(headSide bool) string {
	if r.hash == "" {
		if headSide {
			r.hash = BodyHash(r.file.headSrc, *r.fn, r.file.headLang)
		} else {
			r.hash = BodyHash(r.file.baseSrc, *r.fn, r.file.baseLang)
		}
	}
	return r.hash
}

// compareFiles classifies every finding at head. Functions pair by name
// within a file (a renamed file pairs old to new); the functions left over on
// both sides pair by body hash, each base function at most once and only when
// it no longer exists at head, so a copied function stays new.
//
// TRLC-LINKS: REQ-EMG-156
func compareFiles(files []*deltaFile) []DeltaEntry {
	entries := []DeltaEntry{}
	var headLeft, baseLeft []*funcRef
	for _, f := range files {
		p := pairByName(f)
		entries = append(entries, p.entries...)
		headLeft = append(headLeft, p.headLeft...)
		baseLeft = append(baseLeft, p.baseLeft...)
	}
	entries = append(entries, pairByHash(headLeft, baseLeft)...)
	sortEntries(entries)
	return entries
}

// namePairing is the outcome of pairing the functions of one file by name:
// the entries of the pairs and the functions left on each side.
type namePairing struct {
	entries            []DeltaEntry
	headLeft, baseLeft []*funcRef
}

// pairByName pairs the functions of a file by name.
//
// TRLC-LINKS: REQ-EMG-156
func pairByName(f *deltaFile) namePairing {
	var p namePairing
	baseRefs := functionRefs(f, f.base)
	baseByKey := map[string]*funcRef{}
	for _, r := range baseRefs {
		baseByKey[r.key] = r
	}
	for _, hr := range functionRefs(f, f.head) {
		br := baseByKey[hr.key]
		if br == nil {
			p.headLeft = append(p.headLeft, hr)
			continue
		}
		delete(baseByKey, hr.key)
		p.entries = append(p.entries, functionEntries(f.headPath, hr.fn, br.fn)...)
	}
	p.baseLeft = unpaired(baseRefs, baseByKey)
	return p
}

// unpaired lists the functions of refs that are still in byKey.
//
// TRLC-LINKS: REQ-EMG-156
func unpaired(refs []*funcRef, byKey map[string]*funcRef) []*funcRef {
	var left []*funcRef
	for _, r := range refs {
		if _, ok := byKey[r.key]; ok {
			left = append(left, r)
		}
	}
	return left
}

// pairByHash pairs the functions left over on both sides by body hash and
// returns the entries of the head ones.
//
// TRLC-LINKS: REQ-EMG-156
func pairByHash(headLeft, baseLeft []*funcRef) []DeltaEntry {
	var entries []DeltaEntry
	claimed := map[*funcRef]bool{}
	for _, hr := range headLeft {
		var match *funcRef
		if len(hr.fn.Findings) > 0 {
			match = matchByHash(hr, baseLeft, claimed)
		}
		if match == nil {
			entries = append(entries, functionEntries(hr.file.headPath, hr.fn, nil)...)
			continue
		}
		claimed[match] = true
		entries = append(entries, movedEntries(hr, match.fn)...)
	}
	return entries
}

// matchByHash finds the unclaimed base function with the same body hash that
// has a finding of every code of hr; one in the same file wins.
//
// TRLC-LINKS: REQ-EMG-156
func matchByHash(hr *funcRef, baseLeft []*funcRef, claimed map[*funcRef]bool) *funcRef {
	hash := hr.bodyHash(true)
	if hash == "" {
		return nil
	}
	var other *funcRef
	for _, br := range baseLeft {
		if !br.candidate(hash, claimed) {
			continue
		}
		if br.file == hr.file {
			return br
		}
		if other == nil {
			other = br
		}
	}
	return other
}

// candidate reports whether a base function is unclaimed, has findings and
// has the body hash.
//
// TRLC-LINKS: REQ-EMG-156
func (r *funcRef) candidate(hash string, claimed map[*funcRef]bool) bool {
	return !claimed[r] && len(r.fn.Findings) > 0 && r.bodyHash(false) == hash
}

// movedEntries are the entries of a function that moved or was renamed with
// the same body: a finding that it had before is existing, any other is new.
//
// TRLC-LINKS: REQ-EMG-156
func movedEntries(hr *funcRef, before *Function) []DeltaEntry {
	var out []DeltaEntry
	for _, f := range hr.fn.Findings {
		e := newEntry(hr.file.headPath, hr.fn, f)
		if b := findingOf(before, f.Code); b != nil {
			e.Change, e.SeverityBefore, e.ValueBefore = ChangeExisting, b.Severity, b.Value
		}
		out = append(out, e)
	}
	return out
}

// functionEntries are the entries of a function; before is nil for a function
// that has no counterpart at base.
//
// TRLC-LINKS: REQ-EMG-156
func functionEntries(p string, fn, before *Function) []DeltaEntry {
	var out []DeltaEntry
	for _, f := range fn.Findings {
		e := newEntry(p, fn, f)
		if b := findingOf(before, f.Code); b != nil {
			e.Change, e.SeverityBefore, e.ValueBefore = classify(*b, f), b.Severity, b.Value
		}
		out = append(out, e)
	}
	return out
}

// newEntry is the entry of a finding that is new.
//
// TRLC-LINKS: REQ-EMG-156
func newEntry(p string, fn *Function, f Finding) DeltaEntry {
	return DeltaEntry{
		Path: p, Function: fn.Name, Code: f.Code, Change: ChangeNew, Severity: f.Severity,
		Metric: f.Metric, ValueAfter: f.Value, StartLine: f.StartLine, EndLine: f.EndLine, Message: f.Message,
	}
}

// findingOf returns the finding of a code in fn, or nil.
//
// TRLC-LINKS: REQ-EMG-156
func findingOf(fn *Function, code string) *Finding {
	if fn == nil {
		return nil
	}
	for i := range fn.Findings {
		if fn.Findings[i].Code == code {
			return &fn.Findings[i]
		}
	}
	return nil
}

// sortEntries orders entries by path, function, code, line and change.
//
// TRLC-LINKS: REQ-EMG-159
func sortEntries(entries []DeltaEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		switch {
		case a.Path != b.Path:
			return a.Path < b.Path
		case a.Function != b.Function:
			return a.Function < b.Function
		case a.Code != b.Code:
			return a.Code < b.Code
		case a.StartLine != b.StartLine:
			return a.StartLine < b.StartLine
		case a.Change != b.Change:
			return a.Change < b.Change
		}
		return a.ValueAfter < b.ValueAfter
	})
}
