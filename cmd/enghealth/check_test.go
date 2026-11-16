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
	FloorLevel     string                             `json:"floorLevel"`
	BelowFloor     []struct {
		Path         string
		Score, Floor float64
		Causes       []struct {
			Code, Function string
			Line           int
		}
	} `json:"belowFloor"`
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
	code, stdout, _ := r.check()
	wantEqual(t, code, exitOK)
	wantAll(t, stdout, "no changed source files", "result: ok, no regressions")
	code, doc, _ := r.jsonCheck()
	wantEqual(t, outcomeOf(code, doc), checkOutcome{Schema: "engmod-health-delta/1"})
}

// checkOutcome is what a run of the check decides, from its exit code and its
// JSON report: whether it failed and how many regressions and changed files it
// holds.
type checkOutcome struct {
	Code     int
	Failed   bool
	Blocking int
	Files    int
	Schema   string
}

// outcomeOf summarizes a run of the check.
func outcomeOf(code int, doc checkJSON) checkOutcome {
	return checkOutcome{Code: code, Failed: doc.Failed, Blocking: len(doc.Blocking), Files: len(doc.Files), Schema: doc.Schema}
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
	requireProductRegression(t, r)
	code, stdout, stderr := r.check("--tier", "poc")
	wantEqual(t, code, exitOK)
	wantAll(t, stdout, "regressions (1, warning at tier poc)", "result: ok, the regressions and the files below the floor only warn at this tier")
	wantEqual(t, stderr, "")
	for tier, want := range map[string]int{"prototype": exitOK, "mvp": exitOK, "product": exitFail, "operational": exitFail} {
		got, _, _ := r.check("--tier", tier)
		wantEqual(t, [2]any{tier, got}, [2]any{tier, want})
	}
}

// requireProductRegression checks the text and the exit code of the check at
// product, where the regression of r is an error.
func requireProductRegression(t *testing.T, r *checkRepo) {
	t.Helper()
	code, stdout, stderr := r.check("--tier", "product")
	wantAll(t, stdout, "tier product", "changed files (1)", "pkg/svc.go", "10.0 green -> ", "regressions (1, error at tier product)")
	wantAll(t, stdout, "pkg/svc.go:4 Svc: health.complex_method (new) (none) -> cyclomatic 21 high", "cyclomatic 21", "result: FAIL", "1 new, 0 worsened, 0 existing, 0 waived")
	wantEqual(t, code, exitFail)
	wantContains(t, stderr, "1 regression(s) are errors at tier product")
}

// regressionLevels is what the check decides for a regression: the exit code
// and the level of health.regression in the report.
type regressionLevels struct {
	Code           int
	Level, InLevel string
	Blocking       int
}

// The level of health.regression follows the bundles of the life-cycle block:
// from mvp the safety bundle makes it an error, lifecycle.raise raises it at poc
// and without either at mvp it only warns.
//
// Mutation: ignoring the bundles or raise fails one of the cases.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158
func TestCheckRegressionLevelFollowsTheBundles(t *testing.T) {
	cases := map[string]struct {
		extra string
		want  regressionLevels
	}{
		"mvp with the safety bundle": {"lifecycle:\n  tier: mvp\n  bundles: [safety]\n", regressionLevels{exitFail, "error", "error", 1}},
		"mvp without the bundle":     {"lifecycle:\n  tier: mvp\n", regressionLevels{exitOK, "warning", "warning", 1}},
		"poc with lifecycle.raise":   {"lifecycle:\n  tier: poc\n  raise: [health.regression]\n", regressionLevels{exitFail, "error", "error", 1}},
	}
	for name, c := range cases {
		r := newCheckRepo(t, c.extra, true)
		r.write("pkg/svc.go", svcSource(20))
		code, doc, _ := r.jsonCheck()
		got := regressionLevels{code, doc.RegressionLevel, doc.Levels["health.regression"], len(doc.Blocking)}
		wantEqual(t, [2]any{name, got}, [2]any{name, c.want})
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
	// The debt is in the baseline, so the file below the floor is covered
	// (ADR-EMG-087); without the baseline the same change fails on the floor, see
	// TestCheckFileBelowFloor.
	r := newCheckRepo(t, "lifecycle:\n  tier: product\nhealth:\n  baseline: {file: model/health-baseline.json, decision: ADR-DEBT}\n", false)
	r.write("model/health-baseline.json", svcBaseline)
	r.write("pkg/svc.go", svcSource(20))
	r.commit("base with debt")
	r.write("pkg/svc.go", svcSource(20)+"\n// Other is new and clean.\nfunc Other() int { return 1 }\n")
	code, doc, _ := r.jsonCheck()
	wantEqual(t, outcomeOf(code, doc), checkOutcome{Files: 1, Schema: "engmod-health-delta/1"})
	wantEqual(t, [2]int{doc.Counts.Existing, doc.Counts.New}, [2]int{1, 0})

	r.write("pkg/svc.go", svcSource(9)) // warn at product (cc 10 >= 8)
	r.commit("warn")
	r.write("pkg/svc.go", svcSource(20)) // warn -> high
	requireWorsened(t, r)
}

// requireWorsened checks that the finding of r grew from warn to high and fails
// the check, in the counts, the entry and the text.
func requireWorsened(t *testing.T, r *checkRepo) {
	t.Helper()
	code, doc, _ := r.jsonCheck()
	wantEqual(t, [3]int{code, doc.Counts.Worsened, len(doc.Blocking)}, [3]int{exitFail, 1, 1})
	wantEqual(t, doc.Blocking[0].Change, "worsened")
	_, stdout, _ := r.check()
	wantContains(t, stdout, "cyclomatic 10 warn -> cyclomatic 21 high")
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
	code, doc, _ := r.jsonCheck()
	wantEqual(t, [3]int{code, len(doc.Blocking), doc.Counts.New}, [3]int{exitOK, 0, 0})
	wantEqual(t, doc.Failed, false)
	wantEqual(t, len(doc.Waived), 1)
	wantEqual(t, doc.Waived[0], struct{ Path, Function, Code, Decision string }{"pkg/svc.go", "Svc", "health.complex_method", "ADR-SVC"})
	_, stdout, _ := r.check()
	wantContains(t, stdout, "0 new, 0 worsened, 0 existing, 1 waived")
	other := newCheckRepo(t, "lifecycle:\n  tier: product\nhealth:\n  waivers:\n  - {code: health.complex_method, path: pkg/svc.go, function: NotSvc, decision: ADR-SVC}\n", true)
	other.write("pkg/svc.go", svcSource(20))
	code, _, _ = other.jsonCheck()
	wantEqual(t, code, exitFail) // a waiver for another function must not cover Svc
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
	wantEqual(t, [3]any{code, doc.Base, len(doc.Blocking)}, [3]any{exitFail, "", 1})
	mustEqual(t, len(doc.Notes), 1)
	wantContains(t, doc.Notes[0], "no commits")
	code, stdout, _ := r.check("--tier", "poc")
	wantEqual(t, code, exitOK)
	wantAll(t, stdout, "note: the repository has no commits", "(no commits)")
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
		code, _, _ := runCLI(full...)
		wantEqual(t, [2]any{name, code}, [2]any{name, exitUsage})
	}
	code, _, _ := runCLI("check")
	wantEqual(t, code, exitUsage) // no --model
	_, _, stderr := r.check("--base", "no-such-branch")
	wantAll(t, stderr, "base revision", "no-such-branch")
	requireNotARepository(t)
}

// requireNotARepository checks that the check of a project that is not a git
// repository exits 2 and names the repository.
func requireNotARepository(t *testing.T) {
	t.Helper()
	plain := project(t, "lifecycle:\n  tier: product\n")
	code, _, stderr := runCLI("check", "--model", plain)
	wantEqual(t, code, exitUsage)
	wantContains(t, stderr, "not a git repository")
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
	wantEqual(t, code, exitUsage)
	wantAll(t, stderr, "shallow clone", "fetch", first)
	code, _, _ = runCLI("check", "--model", manifest)
	wantEqual(t, code, exitOK) // against HEAD
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
	mustEqual(t, [4]any{code, doc.Base, doc.Head, len(doc.Blocking)}, [4]any{exitFail, base, head, 1})
	code, doc, _ = r.jsonCheck("--base", base, "--head", head)
	wantEqual(t, [2]any{code, doc.Head}, [2]any{exitFail, head})
	// The other direction: the regression is gone.
	code, doc, _ = r.jsonCheck("--base", head+"..."+base)
	wantEqual(t, [2]int{code, len(doc.Blocking)}, [2]int{exitOK, 0})
	// Against HEAD the working tree is the clean version of the base.
	code, doc, _ = r.jsonCheck("--base", head)
	wantEqual(t, [2]int{code, doc.Counts.New}, [2]int{exitOK, 0})
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
	mustEqual(t, [2]string{second, uncached}, [2]string{first, first})
	wantNone(t, first, r.dir)
	wantAll(t, first, `"schema": "engmod-health-delta/1"`, `"levels"`, `"regressionLevel": "error"`, `"counts"`, `"failed": true`, `"blocking"`, `"entries"`)
	requireOutFile(t, r, first)
}

// requireOutFile checks that the check with --out writes the same JSON to the
// file, writes nothing to stdout and still reports the failure on stderr.
func requireOutFile(t *testing.T, r *checkRepo, want string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "check.json")
	code, stdout, stderr := r.check("--format", "json", "--out", out)
	data, err := os.ReadFile(out)
	wantEqual(t, err, nil)
	wantEqual(t, string(data), want)
	wantEqual(t, [2]any{stdout, code}, [2]any{"", exitFail})
	wantAll(t, stderr, "pkg/svc.go:4 Svc: health.complex_method", "regression(s) are errors")
}

// The SARIF output holds only the regressions, as results of the rule
// health.regression with the smell code in the properties and the level the
// rule has at the tier; existing debt is not a result.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func TestCheckSARIFHoldsOnlyRegressions(t *testing.T) {
	// The baseline covers svc.go, so the floor adds no result (ADR-EMG-087).
	r := newBaselinedRepo(t, "product")
	r.write("pkg/old.go", "package pkg\n\n"+strings.TrimPrefix(strings.Replace(svcSource(20), "func Svc", "func Old", 1), "package pkg\n\n"))
	r.commit("debt")
	r.write("pkg/svc.go", svcSource(20))
	code, stdout, _ := r.check("--format", "sarif")
	var doc checkSARIF
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	mustEqual(t, [3]any{code, doc.Version, len(doc.Runs)}, [3]any{exitFail, "2.1.0", 1})
	run := doc.Runs[0]
	mustEqual(t, [3]any{run.Tool.Driver.Name, len(run.Tool.Driver.Rules), len(run.Results)}, [3]any{"enghealth", 1, 1})
	wantEqual(t, run.Tool.Driver.Rules[0].ID, "health.regression")
	res := run.Results[0]
	wantEqual(t, [4]any{res.RuleID, res.Level, res.Properties["code"], res.Properties["function"]}, [4]any{"health.regression", "error", "health.complex_method", "Svc"})
	loc := res.Locations[0].PhysicalLocation
	wantEqual(t, [2]any{loc.ArtifactLocation.URI, loc.Region.StartLine}, [2]any{"pkg/svc.go", 4})
	code, stdout, _ = r.check("--format", "sarif", "--tier", "poc")
	wantEqual(t, code, exitOK)
	wantContains(t, stdout, `"level": "warning"`)
}

// checkSARIF is the part of the SARIF output of the check the tests read.
type checkSARIF struct {
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
	code, doc, _ := r.jsonCheck("--no-cache")
	mustEqual(t, [2]int{code, len(doc.Blocking)}, [2]int{exitFail, 2})
	cacheDir := filepath.Join(r.dir, ".engmod", "cache", "health")
	_, err := os.Stat(cacheDir)
	wantEqual(t, err != nil, true) // --no-cache wrote no cache
	requirePathScopes(t, r)
	entries, err := os.ReadDir(cacheDir)
	wantEqual(t, err == nil && len(entries) > 0, true) // the cached run wrote a cache
	code, doc, _ = r.jsonCheck("other")
	wantEqual(t, [2]int{code, len(doc.Blocking)}, [2]int{exitFail, 1}) // a second cached run keeps the result
}

// requirePathScopes checks which regressions each form of the path argument
// leaves in the report of r.
func requirePathScopes(t *testing.T, r *checkRepo) {
	t.Helper()
	cases := map[string]struct {
		arg  string
		want int
	}{
		"other":                  {"other", 1},
		"the project":            {".", 2},
		"absolute path in scope": {filepath.Join(r.dir, "pkg", "svc.go"), 1},
	}
	for name, c := range cases {
		code, doc, _ := r.jsonCheck(c.arg)
		wantEqual(t, [3]any{name, code, len(doc.Blocking)}, [3]any{name, exitFail, c.want})
	}
	_, doc, _ := r.jsonCheck("other")
	mustEqual(t, len(doc.Blocking), 1)
	wantEqual(t, doc.Blocking[0].Path, "other/o.go")
}

// svcBaseline is a baseline file that accepts the high complex_method finding of
// Svc in pkg/svc.go (cyclomatic 21), the debt of svcSource(20).
const svcBaseline = `{"schema":"engmod-health-baseline/1","entries":[{"path":"pkg/svc.go","function":"Svc","code":"health.complex_method","value":21}]}`
