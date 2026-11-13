// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// gitIn runs git in dir with a fixed identity and no machine configuration and
// returns its stdout.
//
// TRLC-LINKS: REQ-EMG-156
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{
		"-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always", "-c", "init.defaultBranch=main",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_DATE=2026-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

// checkRepo is a model project that is a git repository.
type checkRepo struct {
	t        *testing.T
	dir      string
	manifest string
}

// newCheckRepo writes a model project (extra is inserted into the manifest)
// with a small clean pkg/svc.go and pkg/ok.go in a new git repository. With
// commit it commits them on main; the working tree is then clean.
//
// TRLC-LINKS: REQ-EMG-156
func newCheckRepo(t *testing.T, extra string, commit bool) *checkRepo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	t.Setenv("HOME", t.TempDir())
	manifest := project(t, extra)
	dir := filepath.Dir(manifest)
	// project() writes a function with a finding; replace it with clean code.
	if err := os.Remove(filepath.Join(dir, "pkg", "big.go")); err != nil {
		t.Fatal(err)
	}
	r := &checkRepo{t: t, dir: dir, manifest: manifest}
	r.write("pkg/svc.go", svcSource(0))
	gitIn(t, dir, "init", "-q", "-b", "main")
	if commit {
		r.commit("base")
	}
	return r
}

// write writes a file of the project.
//
// TRLC-LINKS: REQ-EMG-156
func (r *checkRepo) write(rel, content string) {
	r.t.Helper()
	testutil.WriteFile(r.t, filepath.Join(r.dir, filepath.FromSlash(rel)), content)
}

// commit stages everything, commits and returns the commit id.
//
// TRLC-LINKS: REQ-EMG-156
func (r *checkRepo) commit(msg string) string {
	r.t.Helper()
	gitIn(r.t, r.dir, "add", "-A")
	gitIn(r.t, r.dir, "commit", "-q", "--allow-empty", "-m", msg)
	return strings.TrimSpace(gitIn(r.t, r.dir, "rev-parse", "HEAD"))
}

// check runs enghealth check on the project with extra arguments.
//
// TRLC-LINKS: REQ-EMG-158
func (r *checkRepo) check(args ...string) (int, string, string) {
	full := append([]string{"check", "--model", r.manifest}, args...)
	return runCLI(full...)
}

// svcSource is pkg/svc.go: Svc with n sequential ifs has cyclomatic complexity
// n + 1. The tests use 0 (clean), 9 (cyclomatic 10: a warn at product, limit
// 8, and at the default limit 10) and 20 (cyclomatic 21: a high complex_method
// at the default limit 20 and at the product limit 15).
//
// TRLC-LINKS: REQ-EMG-156
func svcSource(n int) string {
	var b strings.Builder
	b.WriteString("package pkg\n\n// Svc decides.\nfunc Svc(a int) int {\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "\tif a == %d { return %d }\n", i, i)
	}
	b.WriteString("\treturn 0\n}\n")
	return b.String()
}

// checkJSON is the part of the JSON output the tests read.
type checkJSON struct {
	Schema          string `json:"schema"`
	Tier            string `json:"tier"`
	Base            string `json:"base"`
	Head            string `json:"head"`
	RegressionLevel string `json:"regressionLevel"`
	Failed          bool   `json:"failed"`
	Notes           []string
	Counts          struct{ New, Worsened, Existing int }
	Levels          map[string]string
	Waived          []struct{ Path, Function, Code, Decision string }
	Files           []struct {
		Path                    string
		ScoreBefore, ScoreAfter *float64
	}
	Blocking []struct {
		Path, Function, Code, Change string
		StartLine                    int
	}
	Skipped        []struct{ Path, Reason string }
	PolicyWeakened []struct{ Aspect, Message string } `json:"policyWeakened"`
	PolicyLevel    string                             `json:"policyLevel"`
	PolicyDecision string                             `json:"policyDecision"`
}

// jsonCheck runs the check with --format json and decodes the output.
//
// TRLC-LINKS: REQ-EMG-158
func (r *checkRepo) jsonCheck(args ...string) (int, checkJSON, string) {
	r.t.Helper()
	code, stdout, stderr := r.check(append([]string{"--format", "json"}, args...)...)
	var doc checkJSON
	if code != exitUsage {
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			r.t.Fatalf("exit %d, output is not JSON: %v\n%s\n%s", code, err, stdout, stderr)
		}
	}
	return code, doc, stderr
}

// A clean tree has nothing to compare: exit 0, no changed files, no
// regressions, whatever the tier.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func TestCheckCleanTreeExitsZero(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: operational\n", true)
	code, stdout, stderr := r.check()
	if code != exitOK || !strings.Contains(stdout, "no changed source files") || !strings.Contains(stdout, "result: ok, no regressions") {
		t.Fatalf("exit %d\n%s\n%s", code, stdout, stderr)
	}
	code, doc, _ := r.jsonCheck()
	if code != exitOK || doc.Failed || len(doc.Blocking) != 0 || len(doc.Files) != 0 || doc.Schema != "engmod-health-delta/1" {
		t.Errorf("json: exit %d %+v", code, doc)
	}
}

// A new function with a high finding is a regression. At product the level of
// health.regression is error: exit 1, and the text names each regression with
// file:line, function, code, before -> after and message. At poc the same edit
// is only a warning: exit 0, and the verdict says so. From mvp with the safety
// bundle it is an error again.
//
// Mutation: ignoring the tier (always failing, or never) fails one of the cases.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158
func TestCheckRegressionFollowsTheTier(t *testing.T) {
	r := newCheckRepo(t, "", true)
	r.write("pkg/svc.go", svcSource(20)) // cyclomatic 21: high at every tier

	code, stdout, stderr := r.check("--tier", "product")
	for _, want := range []string{
		"tier product", "changed files (1)", "pkg/svc.go", "10.0 green -> ", "regressions (1, error at tier product)",
		"pkg/svc.go:4 Svc: health.complex_method (new) (none) -> cyclomatic 21 high", "cyclomatic 21", "result: FAIL",
		"1 new, 0 worsened, 0 existing, 0 waived",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("product text lacks %q:\n%s", want, stdout)
		}
	}
	if code != exitFail || !strings.Contains(stderr, "1 regression(s) are errors at tier product") {
		t.Errorf("product: exit %d, stderr %q, want 1", code, stderr)
	}

	code, stdout, stderr = r.check("--tier", "poc")
	if code != exitOK || !strings.Contains(stdout, "regressions (1, warning at tier poc)") || !strings.Contains(stdout, "result: ok, the regressions only warn at this tier") || stderr != "" {
		t.Errorf("poc: exit %d\n%s\n%s", code, stdout, stderr)
	}
	for tier, want := range map[string]int{"prototype": exitOK, "mvp": exitOK, "product": exitFail, "operational": exitFail} {
		if code, _, _ := r.check("--tier", tier); code != want {
			t.Errorf("tier %s: exit %d, want %d", tier, code, want)
		}
	}

	safety := newCheckRepo(t, "lifecycle:\n  tier: mvp\n  bundles: [safety]\n", true)
	safety.write("pkg/svc.go", svcSource(20))
	if code, doc, _ := safety.jsonCheck(); code != exitFail || doc.RegressionLevel != "error" || doc.Levels["health.regression"] != "error" {
		t.Errorf("mvp with the safety bundle: exit %d %+v, want 1 and error", code, doc)
	}
	plain := newCheckRepo(t, "lifecycle:\n  tier: mvp\n", true)
	plain.write("pkg/svc.go", svcSource(20))
	if code, doc, _ := plain.jsonCheck(); code != exitOK || doc.RegressionLevel != "warning" || len(doc.Blocking) != 1 {
		t.Errorf("mvp without the bundle: exit %d %+v, want 0 and a warning", code, doc)
	}
	raised := newCheckRepo(t, "lifecycle:\n  tier: poc\n  raise: [health.regression]\n", true)
	raised.write("pkg/svc.go", svcSource(20))
	if code, _, _ := raised.jsonCheck(); code != exitFail {
		t.Errorf("poc with lifecycle.raise: exit %d, want 1", code)
	}
}

// Debt that was already there never fails the gate: a function that has a high
// finding at base and keeps it while the file changes elsewhere is existing.
// A finding that grows from warn to high is worsened and fails at product.
//
// Mutation: treating every finding in a changed file as new fails the first
// case.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func TestCheckExistingDebtPassesAndWorsenedFails(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n", false)
	r.write("pkg/svc.go", svcSource(20))
	r.commit("base with debt")
	r.write("pkg/svc.go", svcSource(20)+"\n// Other is new and clean.\nfunc Other() int { return 1 }\n")
	code, doc, stderr := r.jsonCheck()
	if code != exitOK || doc.Failed || doc.Counts.Existing != 1 || doc.Counts.New != 0 || len(doc.Blocking) != 0 {
		t.Fatalf("existing debt: exit %d %+v %s", code, doc, stderr)
	}

	r.write("pkg/svc.go", svcSource(9)) // warn at product (cc 10 >= 8)
	r.commit("warn")
	r.write("pkg/svc.go", svcSource(20)) // warn -> high
	code, doc, _ = r.jsonCheck()
	if code != exitFail || doc.Counts.Worsened != 1 || len(doc.Blocking) != 1 || doc.Blocking[0].Change != "worsened" {
		t.Errorf("worsened: exit %d %+v, want 1 with one worsened regression", code, doc)
	}
	_, stdout, _ := r.check()
	if !strings.Contains(stdout, "cyclomatic 10 warn -> cyclomatic 21 high") {
		t.Errorf("the text lacks the before -> after of the worsened finding:\n%s", stdout)
	}
}

// A finding that a waiver covers is existing for the check: it is not a
// regression, it is counted as waived, and the gate passes at product. The same
// edit without the waiver fails.
//
// Mutation: ignoring the waivers of the manifest fails.
//
// TRLC-LINKS: REQ-EMG-154, REQ-EMG-156, REQ-EMG-158
func TestCheckWaivedFindingIsNeverARegression(t *testing.T) {
	waiver := "health:\n  waivers:\n  - {code: health.complex_method, path: pkg/svc.go, function: Svc, decision: ADR-SVC}\n"
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n"+waiver, true)
	r.write("pkg/svc.go", svcSource(20))
	code, doc, stderr := r.jsonCheck()
	if code != exitOK || doc.Failed || len(doc.Blocking) != 0 || doc.Counts.New != 0 {
		t.Fatalf("waived: exit %d %+v %s", code, doc, stderr)
	}
	if len(doc.Waived) != 1 || doc.Waived[0].Code != "health.complex_method" || doc.Waived[0].Function != "Svc" || doc.Waived[0].Decision != "ADR-SVC" {
		t.Errorf("waived = %+v, want the one waived finding with its decision", doc.Waived)
	}
	if _, stdout, _ := r.check(); !strings.Contains(stdout, "0 new, 0 worsened, 0 existing, 1 waived") {
		t.Errorf("text does not count the waived finding:\n%s", stdout)
	}
	other := newCheckRepo(t, "lifecycle:\n  tier: product\nhealth:\n  waivers:\n  - {code: health.complex_method, path: pkg/svc.go, function: NotSvc, decision: ADR-SVC}\n", true)
	other.write("pkg/svc.go", svcSource(20))
	if code, _, _ := other.jsonCheck(); code != exitFail {
		t.Errorf("a waiver for another function must not cover Svc: exit %d, want 1", code)
	}
}

// The configuration of the manifest applies as in scan: a threshold raises the
// limit (no finding, so no regression), and health.exclude leaves a file out.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-156
func TestCheckUsesThresholdsAndExcludesOfTheManifest(t *testing.T) {
	lifted := newCheckRepo(t, "lifecycle:\n  tier: product\nhealth:\n  thresholds:\n    health.complex_method: {warn: 30, high: 40}\n", true)
	lifted.write("pkg/svc.go", svcSource(20))
	if code, doc, _ := lifted.jsonCheck(); code != exitOK || len(doc.Blocking) != 0 {
		t.Errorf("threshold 30/40: exit %d %+v, want no regression", code, doc)
	}
	excluded := newCheckRepo(t, "lifecycle:\n  tier: product\nhealth:\n  exclude: [pkg/svc.go]\n", true)
	excluded.write("pkg/svc.go", svcSource(20))
	if code, doc, _ := excluded.jsonCheck(); code != exitOK || len(doc.Files) != 0 {
		t.Errorf("excluded file: exit %d %+v, want it left out", code, doc)
	}
	generated := newCheckRepo(t, "lifecycle:\n  tier: product\n", true)
	generated.write("generated/gen.go", svcSource(20))
	generated.write("vendor/v/v.go", svcSource(20))
	generated.write(".hidden/h.go", svcSource(20))
	if code, doc, _ := generated.jsonCheck(); code != exitOK || len(doc.Files) != 0 {
		t.Errorf("default excludes and dot directories: exit %d %+v, want them left out", code, doc)
	}
}

// In a repository without a commit there is no base: every finding is new, the
// check says so in a note, and the tier decides as usual.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func TestCheckEmptyRepository(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n", false)
	r.write("pkg/svc.go", svcSource(20))
	code, doc, _ := r.jsonCheck()
	if code != exitFail || doc.Base != "" || len(doc.Blocking) != 1 || len(doc.Notes) != 1 || !strings.Contains(doc.Notes[0], "no commits") {
		t.Errorf("product: exit %d %+v, want 1 with the note", code, doc)
	}
	if code, stdout, _ := r.check("--tier", "poc"); code != exitOK || !strings.Contains(stdout, "note: the repository has no commits") || !strings.Contains(stdout, "(no commits)") {
		t.Errorf("poc: exit %d\n%s", code, stdout)
	}
}

// Everything that is not a verdict exits 2: an unknown base, an invalid
// revision, a shallow clone without the base (the message names the base and
// says how to fetch it), a project that is not a git repository, a model that
// does not load, an unknown tier, a path outside the project and a bad flag or
// format. Never 0, never 1.
//
// Mutation: treating an unavailable base as a pass (exit 0) fails.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func TestCheckSetupErrorsExitTwo(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n", true)
	r.write("pkg/svc.go", svcSource(20))
	t.Chdir(r.dir) // paths are relative to the working directory
	cases := map[string][]string{
		"unknown base":     {"--base", "no-such-branch"},
		"invalid base":     {"--base", "-x"},
		"invalid head":     {"--base", "HEAD", "--head", "a:b"},
		"unknown range":    {"--base", "main...nope"},
		"range and head":   {"--base", "main...main", "--head", "main"},
		"unknown tier":     {"--tier", "gold"},
		"path outside":     {"--", "../elsewhere"},
		"missing path":     {"--", "pkg/none.go"},
		"unknown format":   {"--format", "xml"},
		"bad flag":         {"--no-such-flag"},
		"model not loaded": {"--model", filepath.Join(r.dir, "nope.yml")},
	}
	for name, args := range cases {
		full := append([]string{"check", "--model", r.manifest}, args...)
		if name == "model not loaded" {
			full = append([]string{"check"}, args...)
		}
		if code, stdout, stderr := runCLI(full...); code != exitUsage {
			t.Errorf("%s: exit %d, want 2\n%s\n%s", name, code, stdout, stderr)
		}
	}
	if code, _, _ := runCLI("check"); code != exitUsage {
		t.Errorf("no --model: exit %d, want 2", code)
	}
	if code, _, stderr := r.check("--base", "no-such-branch"); !strings.Contains(stderr, "base revision") || !strings.Contains(stderr, "no-such-branch") {
		t.Errorf("unknown base message = %q, want it to name the base revision (exit %d)", stderr, code)
	}

	// Not a git repository.
	plain := project(t, "lifecycle:\n  tier: product\n")
	if code, _, stderr := runCLI("check", "--model", plain); code != exitUsage || !strings.Contains(stderr, "not a git repository") {
		t.Errorf("no repository: exit %d %q, want 2 naming the repository", code, stderr)
	}
}

// A shallow clone that lacks the base revision is a setup error whose message
// says to fetch more history; a silent pass would hide regressions.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func TestCheckShallowCloneWithoutTheBaseExitsTwo(t *testing.T) {
	src := newCheckRepo(t, "lifecycle:\n  tier: product\n", true)
	first := strings.TrimSpace(gitIn(t, src.dir, "rev-parse", "HEAD"))
	src.write("pkg/svc.go", svcSource(1))
	src.commit("second")
	clone := filepath.Join(t.TempDir(), "clone")
	gitIn(t, filepath.Dir(clone), "clone", "-q", "--depth", "1", "file://"+src.dir, clone)
	manifest := filepath.Join(clone, "engmod.yml")
	code, _, stderr := runCLI("check", "--model", manifest, "--base", first)
	if code != exitUsage || !strings.Contains(stderr, "shallow clone") || !strings.Contains(stderr, "fetch") || !strings.Contains(stderr, first) {
		t.Errorf("shallow: exit %d %q, want 2 with the base, the shallow clone and how to fetch", code, stderr)
	}
	if code, _, _ := runCLI("check", "--model", manifest); code != exitOK {
		t.Errorf("shallow clone against HEAD: exit %d, want 0", code)
	}
}

// base...head compares two commits through their merge base, whatever the
// working tree holds; --head with --base does the same.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func TestCheckRangeBetweenCommits(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n", true)
	base := strings.TrimSpace(gitIn(t, r.dir, "rev-parse", "HEAD"))
	r.write("pkg/svc.go", svcSource(20))
	head := r.commit("regression")
	r.write("pkg/svc.go", svcSource(0)) // the working tree is clean again

	code, doc, _ := r.jsonCheck("--base", base+"..."+head)
	if code != exitFail || doc.Base != base || doc.Head != head || len(doc.Blocking) != 1 {
		t.Fatalf("range: exit %d %+v, want 1 between %s and %s", code, doc, base, head)
	}
	code, doc, _ = r.jsonCheck("--base", base, "--head", head)
	if code != exitFail || doc.Head != head {
		t.Errorf("--base and --head: exit %d %+v, want 1", code, doc)
	}
	// The other direction: the regression is gone.
	if code, doc, _ := r.jsonCheck("--base", head+"..."+base); code != exitOK || len(doc.Blocking) != 0 {
		t.Errorf("reversed range: exit %d %+v, want 0", code, doc)
	}
	// Against HEAD the working tree is the clean version of the base.
	if code, doc, _ := r.jsonCheck("--base", head); code != exitOK || doc.Counts.New != 0 {
		t.Errorf("working tree against the regressing commit: exit %d %+v, want 0", code, doc)
	}
}

// The JSON output is deterministic: two runs, with and without the cache, give
// the same bytes, which hold no absolute path and no timestamp; with --out the
// file has the output and stdout is empty. The failure is still on stderr.
//
// Mutation: map-order or time dependent output fails the comparison.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158, REQ-EMG-159
func TestCheckJSONIsDeterministic(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n", true)
	r.write("pkg/svc.go", svcSource(20))
	r.write("pkg/extra.go", "package pkg\n\nfunc Extra() int { return 2 }\n")
	_, first, _ := r.check("--format", "json")
	_, second, _ := r.check("--format", "json")
	_, uncached, _ := r.check("--format", "json", "--no-cache")
	if first != second || first != uncached {
		t.Fatalf("the output differs between runs:\n%s\n---\n%s\n---\n%s", first, second, uncached)
	}
	if strings.Contains(first, r.dir) {
		t.Error("the output holds an absolute path")
	}
	for _, key := range []string{`"schema": "engmod-health-delta/1"`, `"levels"`, `"regressionLevel": "error"`, `"counts"`, `"failed": true`, `"blocking"`, `"entries"`} {
		if !strings.Contains(first, key) {
			t.Errorf("the JSON lacks %s:\n%s", key, first)
		}
	}
	out := filepath.Join(t.TempDir(), "check.json")
	code, stdout, stderr := r.check("--format", "json", "--out", out)
	data, err := os.ReadFile(out)
	if err != nil || string(data) != first || stdout != "" || code != exitFail {
		t.Errorf("--out: exit %d stdout %q err %v identical %v", code, stdout, err, string(data) == first)
	}
	if !strings.Contains(stderr, "pkg/svc.go:4 Svc: health.complex_method") || !strings.Contains(stderr, "regression(s) are errors") {
		t.Errorf("the failure is not on stderr when the output goes to a file: %q", stderr)
	}
}

// The SARIF output holds only the regressions, as results of the rule
// health.regression with the smell code in the properties and the level the
// rule has at the tier; existing debt is not a result.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func TestCheckSARIFHoldsOnlyRegressions(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n", false)
	r.write("pkg/old.go", "package pkg\n\n"+strings.TrimPrefix(strings.Replace(svcSource(20), "func Svc", "func Old", 1), "package pkg\n\n"))
	r.commit("debt")
	r.write("pkg/svc.go", svcSource(20))
	code, stdout, _ := r.check("--format", "sarif")
	var doc struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name  string `json:"name"`
					Rules []struct {
						ID string `json:"id"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID     string         `json:"ruleId"`
				Level      string         `json:"level"`
				Properties map[string]any `json:"properties"`
				Locations  []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	if code != exitFail || doc.Version != "2.1.0" || len(doc.Runs) != 1 || doc.Runs[0].Tool.Driver.Name != "enghealth" ||
		len(doc.Runs[0].Tool.Driver.Rules) != 1 || doc.Runs[0].Tool.Driver.Rules[0].ID != "health.regression" || len(doc.Runs[0].Results) != 1 {
		t.Fatalf("exit %d, sarif = %s", code, stdout)
	}
	res := doc.Runs[0].Results[0]
	loc := res.Locations[0].PhysicalLocation
	if res.RuleID != "health.regression" || res.Level != "error" || res.Properties["code"] != "health.complex_method" ||
		res.Properties["function"] != "Svc" || loc.ArtifactLocation.URI != "pkg/svc.go" || loc.Region.StartLine != 4 {
		t.Errorf("result = %+v", res)
	}
	if code, stdout, _ := r.check("--format", "sarif", "--tier", "poc"); code != exitOK || !strings.Contains(stdout, `"level": "warning"`) {
		t.Errorf("sarif at poc: exit %d\n%s", code, stdout)
	}
}

// Paths restrict the check to files and directories of the project, and
// --no-cache leaves no cache directory behind while the default run writes one
// under .engmod/cache/health.
//
// TRLC-LINKS: REQ-EMG-158, REQ-EMG-159
func TestCheckPathsAndCache(t *testing.T) {
	t.Setenv("CI", "") // the cache is off when CI is set; this test needs it on
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n", true)
	r.write("pkg/svc.go", svcSource(20))
	r.write("other/o.go", strings.Replace(svcSource(20), "func Svc", "func O", 1))
	t.Chdir(r.dir) // paths are relative to the working directory
	if code, doc, _ := r.jsonCheck("--no-cache"); code != exitFail || len(doc.Blocking) != 2 {
		t.Fatalf("both files: exit %d %+v", code, doc)
	}
	if _, err := os.Stat(filepath.Join(r.dir, ".engmod", "cache", "health")); err == nil {
		t.Error("--no-cache wrote a cache")
	}
	if code, doc, _ := r.jsonCheck("other"); code != exitFail || len(doc.Blocking) != 1 || doc.Blocking[0].Path != "other/o.go" {
		t.Errorf("path other: exit %d %+v, want only other/o.go", code, doc)
	}
	if code, doc, _ := r.jsonCheck("."); code != exitFail || len(doc.Blocking) != 2 {
		t.Errorf("path .: exit %d %+v, want both files", code, doc)
	}
	if code, doc, _ := r.jsonCheck(filepath.Join(r.dir, "pkg", "svc.go")); code != exitFail || len(doc.Blocking) != 1 {
		t.Errorf("absolute path inside the project: exit %d %+v", code, doc)
	}
	entries, err := os.ReadDir(filepath.Join(r.dir, ".engmod", "cache", "health"))
	if err != nil || len(entries) == 0 {
		t.Errorf("the cached run wrote no cache: %v", err)
	}
	if code, doc, _ := r.jsonCheck("other"); code != exitFail || len(doc.Blocking) != 1 {
		t.Errorf("a second cached run changed the result: exit %d %+v", code, doc)
	}
}
