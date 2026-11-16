// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

const baselineExtra = "lifecycle:\n  tier: product\nhealth:\n  baseline: {file: model/health-baseline.json, decision: ADR-DEBT}\n"

// runBaselineCLI runs enghealth baseline with args and returns the exit code
// and the outputs.
func runBaselineCLI(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runBaseline(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// complexGo is a Go file with one function of cyclomatic complexity 14, a
// finding at the strict limits of product.
func complexGo(name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// ENGMODEL-OWNER-UNIT: FU-A\npackage pkg\n\n// TRLC-LINKS: REQ-X\nfunc %s(a int) int {\n", name)
	for i := 0; i < 13; i++ {
		fmt.Fprintf(&b, "\tif a == %d { return %d }\n", i, i)
	}
	b.WriteString("\treturn 0\n}\n")
	return b.String()
}

// loadBaseline reads the baseline file of the project.
func loadBaseline(t *testing.T, manifest string) model.HealthBaselineFile {
	t.Helper()
	f, err := model.LoadHealthBaseline(manifest, "model/health-baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// write creates the baseline of the current debt: exit 0, the canonical file
// holds the one finding of the project, a second write is byte-identical, and
// no temporary file is left behind.
//
// Mutation: writing without sorting, a timestamp in the file, or leaving the
// temporary file fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func TestBaselineWriteIsCanonicalAndIdempotent(t *testing.T) {
	p := newBaselineProject(t)
	requireBaseline(t, cliCase{args: []string{"write", "--model", p.manifest}, code: exitOK, stdout: "wrote 1 entries to model/health-baseline.json"})
	first := mustRead(t, p.file())
	want, _ := model.EncodeHealthBaseline([]model.HealthBaselineEntry{{Path: "pkg/big.go", Function: "Big", Code: "health.complex_method", Value: 17}})
	mustEqual(t, string(first), string(want))
	p.write(t)
	wantEqual(t, string(mustRead(t, p.file())), string(first)) // a second write changes nothing
	requireNoTempFiles(t, filepath.Dir(p.file()))
}

// baselineProject is a project whose manifest has a health baseline.
type baselineProject struct {
	manifest, dir string
}

// newBaselineProject writes the project of the baseline tests.
func newBaselineProject(t *testing.T) baselineProject {
	t.Helper()
	manifest := project(t, baselineExtra)
	return baselineProject{manifest: manifest, dir: filepath.Dir(manifest)}
}

// file is the baseline file of the manifest.
func (p baselineProject) file() string { return filepath.Join(p.dir, "model", "health-baseline.json") }

// write runs baseline write and requires exit 0.
func (p baselineProject) write(t *testing.T) {
	t.Helper()
	requireBaseline(t, cliCase{args: []string{"write", "--model", p.manifest}, code: exitOK})
}

// check runs baseline write --check, with the extra arguments of the case, and
// checks the case; it returns the outputs.
func (p baselineProject) check(t *testing.T, c cliCase) (stdout, stderr string) {
	t.Helper()
	c.args = append([]string{"write", "--check", "--model", p.manifest}, c.args...)
	return requireBaseline(t, c)
}

// scan runs scan --fail-on-error and checks the case.
func (p baselineProject) scan(t *testing.T, c cliCase) (stdout, stderr string) {
	t.Helper()
	c.args = []string{"scan", "--model", p.manifest, "--fail-on-error"}
	return requireCLI(t, c)
}

// requireBaseline runs enghealth baseline and checks the case.
func requireBaseline(t *testing.T, c cliCase) (stdout, stderr string) {
	t.Helper()
	code, stdout, stderr := runBaselineCLI(c.args...)
	wantEqual(t, code, c.code)
	wantContains(t, stdout, c.stdout)
	wantContains(t, stderr, c.stderr)
	return stdout, stderr
}

// mustRead reads a file or stops the test.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// requireNoTempFiles checks that a directory holds no temporary file.
func requireNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		wantEqual(t, strings.HasSuffix(e.Name(), ".tmp"), false)
	}
}

// With the baseline written, scan --fail-on-error passes where it failed before
// (the baselined error is not counted), and write ignores the baseline it is
// replacing: it still lists the finding.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155, REQ-EMG-158
func TestBaselineTurnsTheGateGreen(t *testing.T) {
	p := newBaselineProject(t)
	requireBaseline(t, cliCase{args: []string{"write", "--model", p.manifest, "--out", filepath.Join(p.dir, "elsewhere.json")}, code: exitOK})
	// The manifest names model/health-baseline.json, which does not exist yet:
	// a scan is a setup error rather than a pass.
	p.scan(t, cliCase{code: exitUsage})
	p.write(t)
	p.scan(t, cliCase{code: exitOK, stdout: "1 baselined"}) // the text report says how many are baselined
	wantEqual(t, len(loadBaseline(t, p.manifest).Entries), 1)
	// Writing again, with the baseline in place, still sees the finding.
	requireBaseline(t, cliCase{args: []string{"write", "--model", p.manifest}, code: exitOK, stdout: "wrote 1 entries"})
}

// --check exits 0 only when the file is exactly what write would produce: new
// debt is reported as missing and paid debt as stale (exit 1), a file that is
// not canonical differs, a missing or malformed file differs, and write repairs
// each.
//
// Mutation: a check that compares only the entries and not the bytes, or that
// ignores stale entries, fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func TestBaselineCheckVerifiesTheFileIsTheCurrentDebt(t *testing.T) {
	p := newBaselineProject(t)
	p.check(t, cliCase{code: exitFail, stdout: "does not exist"})
	p.write(t)
	p.check(t, cliCase{code: exitOK, stdout: "exactly the current debt"})

	extra := filepath.Join(p.dir, "pkg", "extra.go")
	testutil.WriteFile(t, extra, complexGo("Extra"))
	p.check(t, cliCase{code: exitFail, stdout: "missing: health.complex_method Extra in pkg/extra.go", stderr: "1 missing"})
	p.write(t)
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	p.check(t, cliCase{code: exitFail, stdout: "stale: health.complex_method Extra in pkg/extra.go"})
	p.write(t)

	compact := `{"schema":"engmod-health-baseline/1","entries":[{"path":"pkg/big.go","function":"Big","code":"health.complex_method","value":17}]}`
	testutil.WriteFile(t, p.file(), compact)
	p.check(t, cliCase{code: exitFail, stdout: "not in canonical form"})
	testutil.WriteFile(t, p.file(), "{")
	p.check(t, cliCase{code: exitFail, stdout: "unusable"})
	p.write(t)
	p.check(t, cliCase{code: exitOK})
}

// A baseline checked out with CRLF line ends (git autocrlf on Windows) is still
// exactly the current debt: the comparison reads it as LF (Q7 of the second
// review pass). A file with real differences still fails.
// Mutation: comparing the raw bytes fails the CRLF check.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func TestBaselineCheckAcceptsCRLFCheckout(t *testing.T) {
	p := newBaselineProject(t)
	p.write(t)
	lf := mustRead(t, p.file())
	testutil.WriteFile(t, p.file(), strings.ReplaceAll(string(lf), "\n", "\r\n"))
	p.check(t, cliCase{code: exitOK, stdout: "exactly the current debt"})
	// The scan itself also reads the CRLF file.
	wantEqual(t, len(loadBaseline(t, p.manifest).Entries), 1)
	// A real difference in a CRLF file still fails.
	testutil.WriteFile(t, filepath.Join(p.dir, "pkg", "extra.go"), complexGo("Extra"))
	p.check(t, cliCase{code: exitFail, stdout: "missing: health.complex_method Extra"})
}

// --tier writes only the findings that are errors at that tier: at mvp the same
// function is a warning, so the baseline is empty; at product it is an error.
// Without --tier every finding of the scan is listed.
//
// TRLC-LINKS: REQ-EMG-153
func TestBaselineTierFilter(t *testing.T) {
	manifest := project(t, baselineExtra)
	count := func(args ...string) int {
		t.Helper()
		if code, _, stderr := runBaselineCLI(append([]string{"write", "--model", manifest}, args...)...); code != exitOK {
			t.Fatalf("write %v: %d %q", args, code, stderr)
		}
		return len(loadBaseline(t, manifest).Entries)
	}
	if got := count("--tier", "mvp"); got != 0 {
		t.Errorf("mvp: %d entries, want 0 (a warning at mvp)", got)
	}
	if got := count("--tier", "product"); got != 1 {
		t.Errorf("product: %d entries, want 1", got)
	}
	if got := count(); got != 1 {
		t.Errorf("no tier: %d entries, want 1", got)
	}
}

// A write restricted to paths replaces the entries of the files it scanned and
// keeps the others, and --check with the same paths compares the same merge.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func TestBaselineWriteWithPathsKeepsTheOtherEntries(t *testing.T) {
	p := newBaselineProject(t)
	p.write(t)
	other := filepath.Join(p.dir, "pkg", "other.go")
	testutil.WriteFile(t, other, complexGo("Other"))
	requireBaseline(t, cliCase{args: []string{"write", "--model", p.manifest, other}, code: exitOK})
	var got []string
	for _, e := range loadBaseline(t, p.manifest).Entries {
		got = append(got, e.Path+":"+e.Function)
	}
	mustEqual(t, strings.Join(got, ","), "pkg/big.go:Big,pkg/other.go:Other")
	p.check(t, cliCase{args: []string{other}, code: exitOK})
	p.check(t, cliCase{code: exitOK}) // the full check agrees with the merged file
	testutil.WriteFile(t, p.file(), "{")
	requireBaseline(t, cliCase{args: []string{"write", "--model", p.manifest, other}, code: exitUsage, stderr: "full write"}) // refused over a malformed file
}

// Setup and usage errors exit 2 and write nothing: no subcommand, no model, an
// unknown tier or a model that does not load, no baseline in the manifest and
// no --out, and an --out outside the manifest directory, also through a
// symbolic link.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func TestBaselineSetupErrorsExitTwoAndWriteNothing(t *testing.T) {
	p := newBaselineProject(t)
	manifest, dir := p.manifest, p.dir
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Skip("symbolic links are not available:", err)
	}
	plain := project(t, "lifecycle:\n  tier: product\n")
	cases := map[string][]string{
		"no subcommand":     {},
		"wrong subcommand":  {"read"},
		"no model":          {"write"},
		"unknown tier":      {"write", "--model", manifest, "--tier", "gold"},
		"model missing":     {"write", "--model", filepath.Join(dir, "nope.yml")},
		"no baseline block": {"write", "--model", plain},
		"out parent":        {"write", "--model", manifest, "--out", filepath.Join(outside, "b.json")},
		"out dotdot":        {"write", "--model", manifest, "--out", filepath.Join(dir, "..", "b.json")},
		"out via symlink":   {"write", "--model", manifest, "--out", filepath.Join(dir, "link", "b.json")},
		"bad flag":          {"write", "--model", manifest, "--nope"},
	}
	for name, args := range cases {
		code, _, _ := runBaselineCLI(args...)
		wantEqual(t, [2]any{name, code}, [2]any{name, exitUsage})
	}
	requireNotWritten(t, filepath.Join(outside, "b.json"), filepath.Join(dir, "..", "b.json"), p.file())
	requireBaseline(t, cliCase{args: []string{"--help"}, code: exitOK, stdout: "usage: enghealth baseline write"})
}

// requireNotWritten checks that none of the paths exists.
func requireNotWritten(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		_, err := os.Stat(path)
		wantEqual(t, [2]any{path, err == nil}, [2]any{path, false})
	}
}

// --out inside the manifest directory writes there, creating directories, and
// the manifest baseline file is left alone.
//
// TRLC-LINKS: REQ-EMG-153
func TestBaselineWriteOut(t *testing.T) {
	manifest := project(t, baselineExtra)
	dir := filepath.Dir(manifest)
	out := filepath.Join(dir, "debt", "baseline.json")
	if code, stdout, stderr := runBaselineCLI("write", "--model", manifest, "--out", out); code != exitOK || !strings.Contains(stdout, "debt/baseline.json") {
		t.Fatalf("write --out: %d %q %q", code, stdout, stderr)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "model", "health-baseline.json")); err == nil {
		t.Error("--out must not write the manifest baseline file")
	}
}

// The baseline records the metric value of each finding. Debt that grows past
// its recorded value is no longer accepted: scan --fail-on-error fails again
// although the function is listed, and baseline write --check reports the
// entry as stale and missing with the two values until the file is rewritten;
// debt that shrinks stays accepted but also makes --check fail, so the file only
// ever records the current, lower value.
//
// Mutation: baselining by key alone lets the growth pass.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155, REQ-EMG-158
func TestBaselineValuesStopSilentGrowth(t *testing.T) {
	p := newBaselineProject(t)
	p.write(t)
	f := loadBaseline(t, p.manifest)
	mustEqual(t, len(f.Entries), 1)
	wantEqual(t, f.Entries[0].Value, 17) // Big with the value 17
	p.scan(t, cliCase{code: exitOK})

	// One more branch: cyclomatic 18 against the recorded 17.
	grown := strings.Replace(complexGo("Big"), "\treturn 0\n}\n", "\tif a == 99 { return 99 }\n\tif a == 98 { return 98 }\n\tif a == 97 { return 97 }\n\tif a == 96 { return 96 }\n\tif a == 95 { return 95 }\n\treturn 0\n}\n", 1)
	testutil.WriteFile(t, filepath.Join(p.dir, "pkg", "big.go"), grown)
	p.scan(t, cliCase{code: exitFail, stdout: "the baseline accepts cyclomatic"})
	stdout, _ := p.check(t, cliCase{code: exitFail, stdout: "stale: health.complex_method Big in pkg/big.go (value 17)"})
	wantContains(t, stdout, "missing: health.complex_method Big in pkg/big.go (value 19)")
	p.write(t)
	p.scan(t, cliCase{code: exitOK}) // after the rewrite the scan passes again

	// Paid down: still baselined, but the file no longer matches the debt.
	testutil.WriteFile(t, filepath.Join(p.dir, "pkg", "big.go"), complexGo("Big"))
	p.scan(t, cliCase{code: exitOK})
	stdout, _ = p.check(t, cliCase{code: exitFail, stdout: "(value 19)"})
	wantContains(t, stdout, "(value 14)")
}

// A baseline whose decision is not accepted suppresses nothing, so the scan
// fails on the finding and on the unjustified baseline.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-158
func TestBaselineWithoutAnAcceptedDecisionSuppressesNothing(t *testing.T) {
	p := baselineProject{manifest: project(t, strings.Replace(baselineExtra, "ADR-DEBT", "ADR-DRAFT", 1))}
	p.write(t)
	stdout, _ := p.scan(t, cliCase{code: exitFail, stdout: "health.waiver_unjustified"})
	wantContains(t, stdout, "suppresses nothing")
	wantNone(t, stdout, "1 baselined")
}
