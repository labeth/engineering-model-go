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
	manifest := project(t, baselineExtra)
	file := filepath.Join(filepath.Dir(manifest), "model", "health-baseline.json")
	code, stdout, stderr := runBaselineCLI("write", "--model", manifest)
	if code != exitOK || !strings.Contains(stdout, "wrote 1 entries to model/health-baseline.json") {
		t.Fatalf("write: %d %q %q", code, stdout, stderr)
	}
	first, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := model.EncodeHealthBaseline([]model.HealthBaselineEntry{{Path: "pkg/big.go", Function: "Big", Code: "health.complex_method", Value: 17}})
	if !bytes.Equal(first, want) {
		t.Fatalf("file =\n%s\nwant\n%s", first, want)
	}
	if code, _, stderr := runBaselineCLI("write", "--model", manifest); code != exitOK {
		t.Fatalf("second write: %d %q", code, stderr)
	}
	if second, _ := os.ReadFile(file); !bytes.Equal(first, second) {
		t.Error("a second write changed the file")
	}
	entries, err := os.ReadDir(filepath.Dir(file))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temporary file %s left behind", e.Name())
		}
	}
}

// With the baseline written, scan --fail-on-error passes where it failed before
// (the baselined error is not counted), and write ignores the baseline it is
// replacing: it still lists the finding.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155, REQ-EMG-158
func TestBaselineTurnsTheGateGreen(t *testing.T) {
	manifest := project(t, baselineExtra)
	if code, _, _ := runBaselineCLI("write", "--model", manifest, "--out", filepath.Join(filepath.Dir(manifest), "elsewhere.json")); code != exitOK {
		t.Fatal("write --out failed")
	}
	// The manifest names model/health-baseline.json, which does not exist yet:
	// a scan is a setup error rather than a pass.
	if code, _, _ := runCLI("scan", "--model", manifest, "--fail-on-error"); code != exitUsage {
		t.Fatalf("scan with a missing baseline file: %d, want 2", code)
	}
	if code, _, stderr := runBaselineCLI("write", "--model", manifest); code != exitOK {
		t.Fatalf("write: %d %q", code, stderr)
	}
	code, stdout, stderr := runCLI("scan", "--model", manifest, "--fail-on-error")
	if code != exitOK {
		t.Fatalf("scan with the baseline: %d %q %q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "1 baselined") {
		t.Errorf("the text report must say how many findings are baselined:\n%s", stdout)
	}
	if f := loadBaseline(t, manifest); len(f.Entries) != 1 {
		t.Errorf("entries = %+v", f.Entries)
	}
	// Writing again, with the baseline in place, still sees the finding.
	if code, stdout, _ := runBaselineCLI("write", "--model", manifest); code != exitOK || !strings.Contains(stdout, "wrote 1 entries") {
		t.Errorf("write ignores the existing baseline: %d %q", code, stdout)
	}
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
	manifest := project(t, baselineExtra)
	dir := filepath.Dir(manifest)
	file := filepath.Join(dir, "model", "health-baseline.json")
	check := func() (int, string, string) { return runBaselineCLI("write", "--check", "--model", manifest) }

	if code, stdout, _ := check(); code != exitFail || !strings.Contains(stdout, "does not exist") {
		t.Fatalf("missing file: %d %q", code, stdout)
	}
	runBaselineCLI("write", "--model", manifest)
	if code, stdout, stderr := check(); code != exitOK || !strings.Contains(stdout, "exactly the current debt") {
		t.Fatalf("equal: %d %q %q", code, stdout, stderr)
	}

	testutil.WriteFile(t, filepath.Join(dir, "pkg", "extra.go"), complexGo("Extra"))
	code, stdout, stderr := check()
	if code != exitFail || !strings.Contains(stdout, "missing: health.complex_method Extra in pkg/extra.go") || !strings.Contains(stderr, "1 missing") {
		t.Fatalf("new debt: %d %q %q", code, stdout, stderr)
	}
	runBaselineCLI("write", "--model", manifest)
	if err := os.Remove(filepath.Join(dir, "pkg", "extra.go")); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ = check()
	if code != exitFail || !strings.Contains(stdout, "stale: health.complex_method Extra in pkg/extra.go") {
		t.Fatalf("paid debt: %d %q", code, stdout)
	}
	runBaselineCLI("write", "--model", manifest)

	compact := `{"schema":"engmod-health-baseline/1","entries":[{"path":"pkg/big.go","function":"Big","code":"health.complex_method","value":17}]}`
	testutil.WriteFile(t, file, compact)
	if code, stdout, _ := check(); code != exitFail || !strings.Contains(stdout, "not in canonical form") {
		t.Fatalf("non-canonical: %d %q", code, stdout)
	}
	testutil.WriteFile(t, file, "{")
	if code, stdout, _ := check(); code != exitFail || !strings.Contains(stdout, "unusable") {
		t.Fatalf("malformed: %d %q", code, stdout)
	}
	runBaselineCLI("write", "--model", manifest)
	if code, _, _ := check(); code != exitOK {
		t.Fatalf("after write: %d", code)
	}
}

// A baseline checked out with CRLF line ends (git autocrlf on Windows) is still
// exactly the current debt: the comparison reads it as LF (Q7 of the second
// review pass). A file with real differences still fails.
// Mutation: comparing the raw bytes fails the CRLF check.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func TestBaselineCheckAcceptsCRLFCheckout(t *testing.T) {
	manifest := project(t, baselineExtra)
	file := filepath.Join(filepath.Dir(manifest), "model", "health-baseline.json")
	if code, _, stderr := runBaselineCLI("write", "--model", manifest); code != exitOK {
		t.Fatalf("write: %d %s", code, stderr)
	}
	lf, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, file, strings.ReplaceAll(string(lf), "\n", "\r\n"))
	if code, stdout, stderr := runBaselineCLI("write", "--check", "--model", manifest); code != exitOK || !strings.Contains(stdout, "exactly the current debt") {
		t.Fatalf("CRLF file: %d %q %q", code, stdout, stderr)
	}
	// The scan itself also reads the CRLF file.
	if loaded := loadBaseline(t, manifest); len(loaded.Entries) != 1 {
		t.Errorf("entries = %+v", loaded.Entries)
	}
	// A real difference in a CRLF file still fails.
	testutil.WriteFile(t, filepath.Join(filepath.Dir(manifest), "pkg", "extra.go"), complexGo("Extra"))
	if code, stdout, _ := runBaselineCLI("write", "--check", "--model", manifest); code != exitFail || !strings.Contains(stdout, "missing: health.complex_method Extra") {
		t.Fatalf("CRLF file with new debt: %d %q", code, stdout)
	}
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
	manifest := project(t, baselineExtra)
	dir := filepath.Dir(manifest)
	runBaselineCLI("write", "--model", manifest)
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "other.go"), complexGo("Other"))
	other := filepath.Join(dir, "pkg", "other.go")
	if code, _, stderr := runBaselineCLI("write", "--model", manifest, other); code != exitOK {
		t.Fatalf("restricted write: %d %q", code, stderr)
	}
	var got []string
	for _, e := range loadBaseline(t, manifest).Entries {
		got = append(got, e.Path+":"+e.Function)
	}
	if strings.Join(got, ",") != "pkg/big.go:Big,pkg/other.go:Other" {
		t.Fatalf("entries = %v", got)
	}
	if code, _, stderr := runBaselineCLI("write", "--check", "--model", manifest, other); code != exitOK {
		t.Fatalf("restricted check: %d %q", code, stderr)
	}
	if code, _, _ := runBaselineCLI("write", "--check", "--model", manifest); code != exitOK {
		t.Fatal("the full check must agree with the merged file")
	}
	testutil.WriteFile(t, filepath.Join(dir, "model", "health-baseline.json"), "{")
	if code, _, stderr := runBaselineCLI("write", "--model", manifest, other); code != exitUsage || !strings.Contains(stderr, "full write") {
		t.Errorf("a restricted write over a malformed file must be refused: %d %q", code, stderr)
	}
}

// Setup and usage errors exit 2 and write nothing: no subcommand, no model, an
// unknown tier or a model that does not load, no baseline in the manifest and
// no --out, and an --out outside the manifest directory, also through a
// symbolic link.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func TestBaselineSetupErrorsExitTwoAndWriteNothing(t *testing.T) {
	manifest := project(t, baselineExtra)
	dir := filepath.Dir(manifest)
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
		if code, _, _ := runBaselineCLI(args...); code != exitUsage {
			t.Errorf("%s: exit %d, want 2", name, code)
		}
	}
	for _, p := range []string{filepath.Join(outside, "b.json"), filepath.Join(dir, "..", "b.json"), filepath.Join(dir, "model", "health-baseline.json")} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s was written by a refused command", p)
		}
	}
	if code, stdout, _ := runBaselineCLI("--help"); code != exitOK || !strings.Contains(stdout, "usage: enghealth baseline write") {
		t.Errorf("help: %d %q", code, stdout)
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
	manifest := project(t, baselineExtra)
	dir := filepath.Dir(manifest)
	if code, _, stderr := runBaselineCLI("write", "--model", manifest); code != exitOK {
		t.Fatalf("write: %d %q", code, stderr)
	}
	if f := loadBaseline(t, manifest); len(f.Entries) != 1 || f.Entries[0].Value != 17 {
		t.Fatalf("entries = %+v, want Big with the value 17", f.Entries)
	}
	if code, _, _ := runCLI("scan", "--model", manifest, "--fail-on-error"); code != exitOK {
		t.Fatalf("the baselined scan must pass: %d", code)
	}

	// One more branch: cyclomatic 18 against the recorded 17.
	grown := strings.Replace(complexGo("Big"), "\treturn 0\n}\n", "\tif a == 99 { return 99 }\n\tif a == 98 { return 98 }\n\tif a == 97 { return 97 }\n\tif a == 96 { return 96 }\n\tif a == 95 { return 95 }\n\treturn 0\n}\n", 1)
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "big.go"), grown)
	code, stdout, stderr := runCLI("scan", "--model", manifest, "--fail-on-error")
	if code != exitFail || !strings.Contains(stdout, "the baseline accepts cyclomatic") {
		t.Errorf("a grown baselined finding: %d\n%s\n%s", code, stdout, stderr)
	}
	code, stdout, _ = runBaselineCLI("write", "--model", manifest, "--check")
	if code != exitFail || !strings.Contains(stdout, "stale: health.complex_method Big in pkg/big.go (value 17)") ||
		!strings.Contains(stdout, "missing: health.complex_method Big in pkg/big.go (value 19)") {
		t.Errorf("check after growth: %d %q", code, stdout)
	}
	if code, _, _ := runBaselineCLI("write", "--model", manifest); code != exitOK {
		t.Fatal("rewrite failed")
	}
	if code, _, _ := runCLI("scan", "--model", manifest, "--fail-on-error"); code != exitOK {
		t.Errorf("after the rewrite the scan passes again: %d", code)
	}

	// Paid down: still baselined, but the file no longer matches the debt.
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "big.go"), complexGo("Big"))
	if code, _, _ := runCLI("scan", "--model", manifest, "--fail-on-error"); code != exitOK {
		t.Errorf("a finding below its recorded value stays baselined: %d", code)
	}
	if code, stdout, _ := runBaselineCLI("write", "--model", manifest, "--check"); code != exitFail || !strings.Contains(stdout, "(value 19)") || !strings.Contains(stdout, "(value 14)") {
		t.Errorf("check after payment: %d %q", code, stdout)
	}
}

// A baseline whose decision is not accepted suppresses nothing, so the scan
// fails on the finding and on the unjustified baseline.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-158
func TestBaselineWithoutAnAcceptedDecisionSuppressesNothing(t *testing.T) {
	manifest := project(t, strings.Replace(baselineExtra, "ADR-DEBT", "ADR-DRAFT", 1))
	if code, _, stderr := runBaselineCLI("write", "--model", manifest); code != exitOK {
		t.Fatalf("write: %d %q", code, stderr)
	}
	code, stdout, stderr := runCLI("scan", "--model", manifest, "--fail-on-error")
	if code != exitFail || !strings.Contains(stdout, "health.waiver_unjustified") || !strings.Contains(stdout, "suppresses nothing") || strings.Contains(stdout, "1 baselined") {
		t.Errorf("scan: %d\n%s\n%s", code, stdout, stderr)
	}
}
