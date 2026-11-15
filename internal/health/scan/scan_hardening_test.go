// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package scan

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

// hugeGo is a Go file just above the size limit.
func hugeGo() string {
	return "package pkg\n\n// " + strings.Repeat("x", MaxFileBytes) + "\nfunc Huge() {}\n"
}

// diagnosticFor returns the diagnostic of a path and code.
func diagnosticFor(r health.Report, path, code string) (health.Diagnostic, bool) {
	for _, d := range r.Diagnostics {
		if d.Path == path && d.Code == code {
			return d, true
		}
	}
	return health.Diagnostic{}, false
}

// A source file that was not analysed (too large, or its analysis failed) is no
// clean file: its diagnostic has the level of health.regression at the tier
// (computed by the tier policy), so it is an error from product and from mvp
// with the safety bundle and a warning below, and it counts in the errors of
// the summary, which is what scan --fail-on-error tests. A padded file above
// 4 MiB therefore blocks at product and only warns at poc.
//
// Mutation: a fixed warning level for the two diagnostics fails the product and
// safety rows.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155, REQ-EMG-158
func TestScanSkippedAndFailedFilesBlockWhereARegressionBlocks(t *testing.T) {
	cases := []levelCase{
		{"poc", "lifecycle:\n  tier: poc\n", health.LevelWarning},
		{"mvp", "lifecycle:\n  tier: mvp\n", health.LevelWarning},
		{"product", "lifecycle:\n  tier: product\n", health.LevelError},
		{"operational", "lifecycle:\n  tier: operational\n", health.LevelError},
		{"mvp with safety", "lifecycle:\n  tier: mvp\n  bundles: [safety]\n", health.LevelError},
		{"no lifecycle", "", health.LevelWarning},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { requireBlockingLevel(t, c) })
	}
}

// levelCase is a manifest block and the level the skipped and failed files have
// under it.
type levelCase struct {
	name, lifecycle, want string
}

// requireBlockingLevel scans a padded file and a file the engine fails on, and
// checks the level of their diagnostics, the error count of the summary and the
// levels of the report.
func requireBlockingLevel(t *testing.T, c levelCase) {
	t.Helper()
	dir, manifest := fixture(t, c.lifecycle)
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "huge.go"), hugeGo())
	r := mustScan(t, Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, Engine: failingEngine{Engine: realEngine()}, NoCache: true})
	skipped, ok1 := diagnosticFor(r, "pkg/huge.go", CodeFileSkipped)
	failed, ok2 := diagnosticFor(r, "pkg/big.go", CodeFileFailed)
	wantEqual(t, [4]any{ok1, ok2, skipped.Level, failed.Level}, [4]any{true, true, c.want, c.want})
	// The padded file also breaks the code scan of the model (a line above the
	// scanner limit), so the roll-ups are unavailable: a third diagnostic of the
	// same level.
	errors := 0
	if c.want == health.LevelError {
		errors = 3
	}
	_, unavailable := diagnosticFor(r, "", health.CodeRollupUnavailable)
	wantEqual(t, unavailable, true)
	wantEqual(t, countErrorDiagnostics(r), errors)
	wantEqual(t, r.Summary.Errors >= errors, true)
	wantEqual(t, [2]string{r.Levels[CodeFileSkipped], r.Levels[CodeFileFailed]}, [2]string{c.want, c.want})
}

// countErrorDiagnostics counts the diagnostics of a report that are errors.
func countErrorDiagnostics(r health.Report) int {
	counted := 0
	for _, d := range r.Diagnostics {
		if d.Level == health.LevelError {
			counted++
		}
	}
	return counted
}

// A path with a control character is not scanned: it is a health.file_skipped
// diagnostic at the level of a regression whose message quotes the name, so a
// newline or an escape sequence of a file name never reaches a log raw.
//
// Mutation: listing the file anyway, or printing the name unquoted, fails.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-158
func TestScanSkipsPathsWithControlCharacters(t *testing.T) {
	dir, manifest := fixture(t, "lifecycle:\n  tier: product\n")
	odd := "pkg/evil\n::error::injected.go"
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(odd)), []byte("package pkg\n"), 0o644); err != nil {
		t.Skipf("the file system does not accept the name: %v", err)
	}
	r := mustScan(t, Options{ManifestPath: manifest, NoCache: true})
	for _, f := range r.Files {
		wantNone(t, f.Path, "\n")
	}
	d, ok := diagnosticFor(r, odd, CodeFileSkipped)
	wantEqual(t, [2]any{ok, d.Level}, [2]any{true, health.LevelError})
	wantNone(t, d.Message, "\n") // the message quotes the name
	wantContains(t, d.Message, `\n`)
}

// countingEngine counts the files its inner engine analysed.
type countingEngine struct {
	health.Engine
	files *atomic.Int64
}

// Analyze counts and delegates.
func (e countingEngine) Analyze(inputs []health.FileInput, cfg health.Config) ([]health.FileResult, error) {
	e.files.Add(int64(len(inputs)))
	return e.Engine.Analyze(inputs, cfg)
}

// gitIn runs git in dir for a test.
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1"}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, stderr.String())
	}
}

// cacheDirOf is the cache directory of a fixture.
func cacheDirOf(dir string) string { return filepath.Join(dir, ".engmod", "cache", "health") }

// The scan uses the result cache: the second run on unchanged files analyses
// nothing and gives the byte-identical report, a changed file is analysed
// again, and --no-cache neither reads nor writes. Where CI is set the cache is
// off unless UseCache is set; a tracked file under .engmod refuses it with a
// note on Notes (the run goes on, uncached).
//
// Mutation: a Scan that does not wrap its engine in the cache fails the hit
// count; ignoring CI or the tracked-file rule fails the matching case.
//
// TRLC-LINKS: REQ-EMG-159
func TestScanUsesTheResultCacheUnderTheRules(t *testing.T) {
	t.Setenv("CI", "")
	dir, manifest := fixture(t, "lifecycle:\n  tier: product\n")
	c := newCacheFixture(t)
	first, analysed := c.run(manifest, Options{})
	mustEqual(t, analysed > 0, true)
	_, err := os.Stat(cacheDirOf(dir))
	mustEqual(t, err, nil) // the first run wrote a cache
	second, analysed2 := c.run(manifest, Options{})
	wantEqual(t, [2]any{analysed2, bytes.Equal(first, second)}, [2]any{int64(0), true}) // a full cache hit
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "clean.go"), "// ENGMODEL-OWNER-UNIT: FU-B\npackage pkg\n\n// TRLC-LINKS: REQ-X\nfunc Clean() int { return 2 }\n")
	_, analysed3 := c.run(manifest, Options{})
	wantEqual(t, analysed3, int64(1)) // one edited file
	_, analysedNo := c.run(manifest, Options{NoCache: true})
	wantEqual(t, analysedNo > 0, true) // --no-cache analyses every file
}

// cacheFixture counts the files the engine of its scans analysed.
type cacheFixture struct {
	t      *testing.T
	files  *atomic.Int64
	engine countingEngine
}

// newCacheFixture returns a fixture with a real engine behind the counter.
func newCacheFixture(t *testing.T) *cacheFixture {
	t.Helper()
	var files atomic.Int64
	return &cacheFixture{t: t, files: &files, engine: countingEngine{Engine: realEngine(), files: &files}}
}

// run scans the manifest and returns the encoded report and the number of
// files that were analysed.
func (c *cacheFixture) run(manifest string, opts Options) ([]byte, int64) {
	c.t.Helper()
	before := c.files.Load()
	opts.ManifestPath, opts.Engine = manifest, c.engine
	data := mustEncode(c.t, mustScan(c.t, opts))
	return data, c.files.Load() - before
}

// Where CI is set the scan does not use the cache unless UseCache is set.
//
// Mutation: ignoring CI fails.
//
// TRLC-LINKS: REQ-EMG-159
func TestScanCacheIsOffWhereCIIsSet(t *testing.T) {
	t.Setenv("CI", "")
	dir, manifest := fixture(t, "lifecycle:\n  tier: product\n")
	c := newCacheFixture(t)
	_, analysed := c.run(manifest, Options{NoCache: true})
	t.Setenv("CI", "true")
	_, n1 := c.run(manifest, Options{})
	_, n2 := c.run(manifest, Options{})
	_, err := os.Stat(cacheDirOf(dir))
	wantEqual(t, [2]any{err != nil, n1 + n2}, [2]any{true, 2 * analysed}) // no cache, every file analysed twice
	_, m1 := c.run(manifest, Options{UseCache: true})
	_, m2 := c.run(manifest, Options{UseCache: true})
	_, err = os.Stat(cacheDirOf(dir))
	wantEqual(t, [2]any{err == nil, m1 + m2}, [2]any{true, analysed}) // --cache in CI keeps it on
}

// A tracked file under .engmod refuses the cache with a note; the run goes on,
// uncached.
//
// Mutation: ignoring the tracked-file rule fails.
//
// TRLC-LINKS: REQ-EMG-159
func TestScanTrackedEngmodFileRefusesTheCache(t *testing.T) {
	t.Setenv("CI", "")
	dir, manifest := fixture(t, "lifecycle:\n  tier: product\n")
	gitIn(t, dir, "init", "-q", "-b", "main")
	testutil.WriteFile(t, filepath.Join(dir, ".engmod", "cache", "health", strings.Repeat("ab", 32)+".json"), "{}")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "x")
	c := newCacheFixture(t)
	for i := 0; i < 2; i++ {
		var notes bytes.Buffer
		c.run(manifest, Options{Notes: &notes})
		wantContains(t, notes.String(), "tracked by git")
	}
	entries, _ := os.ReadDir(cacheDirOf(dir))
	wantEqual(t, [2]any{len(entries), c.files.Load() > 0}, [2]any{1, true}) // a refused cache is neither read nor written
}
