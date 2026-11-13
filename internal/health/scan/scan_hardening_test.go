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
	cases := []struct {
		name, lifecycle string
		want            string
	}{
		{"poc", "lifecycle:\n  tier: poc\n", health.LevelWarning},
		{"mvp", "lifecycle:\n  tier: mvp\n", health.LevelWarning},
		{"product", "lifecycle:\n  tier: product\n", health.LevelError},
		{"operational", "lifecycle:\n  tier: operational\n", health.LevelError},
		{"mvp with safety", "lifecycle:\n  tier: mvp\n  bundles: [safety]\n", health.LevelError},
		{"no lifecycle", "", health.LevelWarning},
	}
	for _, c := range cases {
		dir, manifest := fixture(t, c.lifecycle)
		testutil.WriteFile(t, filepath.Join(dir, "pkg", "huge.go"), hugeGo())
		scope := []string{filepath.Join(dir, "pkg")}
		r, err := Scan(Options{ManifestPath: manifest, Paths: scope, Engine: failingEngine{Engine: realEngine()}, NoCache: true})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		skipped, ok1 := diagnosticFor(r, "pkg/huge.go", CodeFileSkipped)
		failed, ok2 := diagnosticFor(r, "pkg/big.go", CodeFileFailed)
		if !ok1 || !ok2 || skipped.Level != c.want || failed.Level != c.want {
			t.Errorf("%s: skipped %+v failed %+v, want level %s", c.name, skipped, failed, c.want)
		}
		errors := 0
		if c.want == health.LevelError {
			// The padded file also breaks the code scan of the model (a line
			// above the scanner limit), so the roll-ups are unavailable: a third
			// diagnostic of the same level.
			errors = 3
		}
		if _, ok := diagnosticFor(r, "", health.CodeRollupUnavailable); !ok {
			t.Errorf("%s: the report does not say that the roll-ups are unavailable: %+v", c.name, r.Diagnostics)
		}
		counted := 0
		for _, d := range r.Diagnostics {
			if d.Level == health.LevelError {
				counted++
			}
		}
		if counted != errors || r.Summary.Errors < errors {
			t.Errorf("%s: %d error diagnostics %+v, summary %+v, want %d errors in the summary", c.name, counted, r.Diagnostics, r.Summary, errors)
		}
		if r.Levels[CodeFileSkipped] != c.want || r.Levels[CodeFileFailed] != c.want {
			t.Errorf("%s: levels %v", c.name, r.Levels)
		}
	}
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
	r, err := Scan(Options{ManifestPath: manifest, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.Files {
		if strings.ContainsAny(f.Path, "\n") {
			t.Errorf("scanned %q", f.Path)
		}
	}
	d, ok := diagnosticFor(r, odd, CodeFileSkipped)
	if !ok || d.Level != health.LevelError || strings.Contains(d.Message, "\n") || !strings.Contains(d.Message, `\n`) {
		t.Errorf("diagnostic = %+v (found %v), want an error whose message quotes the name", d, ok)
	}
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
	var files atomic.Int64
	engine := countingEngine{Engine: realEngine(), files: &files}
	run := func(opts Options) ([]byte, int64) {
		t.Helper()
		before := files.Load()
		opts.ManifestPath, opts.Engine = manifest, engine
		r, err := Scan(opts)
		if err != nil {
			t.Fatal(err)
		}
		data, err := health.Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		return data, files.Load() - before
	}

	first, analysed := run(Options{})
	if analysed == 0 {
		t.Fatal("the first run analysed nothing")
	}
	if _, err := os.Stat(cacheDirOf(dir)); err != nil {
		t.Fatalf("the first run wrote no cache: %v", err)
	}
	second, analysed2 := run(Options{})
	if analysed2 != 0 || !bytes.Equal(first, second) {
		t.Errorf("second run analysed %d files, identical output %v; want a full cache hit", analysed2, bytes.Equal(first, second))
	}
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "clean.go"), "// ENGMODEL-OWNER-UNIT: FU-B\npackage pkg\n\n// TRLC-LINKS: REQ-X\nfunc Clean() int { return 2 }\n")
	if _, analysed3 := run(Options{}); analysed3 != 1 {
		t.Errorf("after editing one file %d were analysed, want 1", analysed3)
	}
	if _, analysedNo := run(Options{NoCache: true}); analysedNo == 0 {
		t.Error("--no-cache must analyse every file")
	}

	// CI set: no cache unless forced.
	dirCI, manifestCI := fixture(t, "lifecycle:\n  tier: product\n")
	t.Setenv("CI", "true")
	files.Store(0)
	for i := 0; i < 2; i++ {
		if _, err := Scan(Options{ManifestPath: manifestCI, Engine: engine}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(cacheDirOf(dirCI)); err == nil || files.Load() != 2*analysed {
		t.Errorf("CI set: cache dir error %v, %d files analysed in two runs, want no cache and %d", err, files.Load(), 2*analysed)
	}
	files.Store(0)
	for i := 0; i < 2; i++ {
		if _, err := Scan(Options{ManifestPath: manifestCI, Engine: engine, UseCache: true}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(cacheDirOf(dirCI)); err != nil || files.Load() != analysed {
		t.Errorf("--cache in CI: cache dir error %v, %d files analysed in two runs, want the cache and %d", err, files.Load(), analysed)
	}

	// A tracked file under .engmod refuses the cache, with a note.
	t.Setenv("CI", "")
	dirGit, manifestGit := fixture(t, "lifecycle:\n  tier: product\n")
	gitIn(t, dirGit, "init", "-q", "-b", "main")
	testutil.WriteFile(t, filepath.Join(dirGit, ".engmod", "cache", "health", strings.Repeat("ab", 32)+".json"), "{}")
	gitIn(t, dirGit, "add", "-A")
	gitIn(t, dirGit, "commit", "-q", "-m", "x")
	var notes bytes.Buffer
	files.Store(0)
	for i := 0; i < 2; i++ {
		notes.Reset()
		if _, err := Scan(Options{ManifestPath: manifestGit, Engine: engine, Notes: &notes}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(notes.String(), "tracked by git") {
			t.Errorf("run %d: notes = %q, want the refusal", i, notes.String())
		}
	}
	entries, _ := os.ReadDir(cacheDirOf(dirGit))
	if len(entries) != 1 || files.Load() == 0 {
		t.Errorf("a refused cache must not be read or written: %d entries, %d files analysed", len(entries), files.Load())
	}
}
