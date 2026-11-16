// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/health"
)

// setManifest replaces a block of the manifest of the project (the model must
// have it) and returns the path.
//
// TRLC-LINKS: REQ-EMG-155
func (r *checkRepo) setManifest(old, replacement string) {
	r.t.Helper()
	data, err := os.ReadFile(r.manifest)
	if err != nil {
		r.t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		r.t.Fatalf("the manifest has no %q:\n%s", old, data)
	}
	if err := os.WriteFile(r.manifest, []byte(strings.Replace(string(data), old, replacement, 1)), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// setFile replaces a text in a file of the project.
//
// TRLC-LINKS: REQ-EMG-155
func (r *checkRepo) setFile(rel, old, replacement string) {
	r.t.Helper()
	data, err := os.ReadFile(filepath.Join(r.dir, filepath.FromSlash(rel)))
	if err != nil {
		r.t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		r.t.Fatalf("%s has no %q:\n%s", rel, old, data)
	}
	r.write(rel, strings.Replace(string(data), old, replacement, 1))
}

// An edit that weakens the policy it is measured by is health.policy_weakened:
// a higher limit, a new exclude, an override that raises a limit and a lower
// tier each make the check exit 1 where the rule is an error (product and
// operational), name the aspect in the text and the JSON, and only warn below
// product. The base manifest is the one at the base revision, so the edit is
// seen even though it is uncommitted work and the code did not change.
//
// Mutation: comparing the head with itself, ignoring one aspect or reading the
// level at the head tier only (a lowered tier would make its own weakening a
// warning) fails the matching case.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158
func TestCheckReportsAWeakenedPolicy(t *testing.T) {
	const product = "lifecycle:\n  tier: product\n"
	cases := []weakening{
		{"limit raised", product + "health:\n  exclude: [gen/**]\n", product + "health:\n  exclude: [gen/**]\n  thresholds:\n    health.complex_method: {warn: 12, high: 25}\n", "threshold"},
		{"exclude widened", product + "health:\n  exclude: [gen/**]\n", product + "health:\n  exclude: [gen/**, pkg/**]\n", "exclude"},
		{"override raises", product + "health:\n  exclude: [gen/**]\n", product + "health:\n  exclude: [gen/**]\n  overrides:\n    - path: pkg/**\n      thresholds:\n        health.large_method: {warn: 300, high: 600}\n", "override"},
		{"tier lowered", product + "health:\n  exclude: [gen/**]\n", "lifecycle:\n  tier: mvp\nhealth:\n  exclude: [gen/**]\n", "tier"},
		{"lifecycle removed", product + "health:\n  exclude: [gen/**]\n", "health:\n  exclude: [gen/**]\n", "tier"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { requireWeakened(t, c) })
	}
}

// weakening is an edit of the manifest that weakens the policy: the manifest
// at the base, the manifest at the head and the aspect the check names.
type weakening struct {
	name, base, head, aspect string
}

// requireWeakened checks that the check of an uncommitted edit fails on the
// weakening and names its aspect in the JSON, on stderr and in the text.
func requireWeakened(t *testing.T, c weakening) {
	t.Helper()
	r := newCheckRepo(t, c.base, true)
	r.setManifest(c.base, c.head)
	code, doc, stderr := r.jsonCheck()
	wantEqual(t, [4]any{code, doc.Failed, doc.PolicyLevel, len(doc.PolicyWeakened)}, [4]any{exitFail, true, "error", 1})
	mustEqual(t, len(doc.PolicyWeakened), 1)
	wantEqual(t, doc.PolicyWeakened[0].Aspect, c.aspect)
	wantContains(t, stderr, "weakening(s) of the health policy against the base revision are not recorded by a policyDecision")
	_, stdout, _ := r.check()
	wantAll(t, stdout, "policy weakened ("+c.aspect+")", "result: FAIL")
}

// Below product the same edit is a warning at both tiers (exit 0); with the
// safety bundle the level is an error from mvp; a tier given on the command
// line applies to both sides, so it is no tier weakening; and an unchanged or
// stricter policy is fine.
//
// Mutation: reading the level at the head tier only, or ignoring the tier flag,
// fails the matching case.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158
func TestCheckPolicyWeakeningLevelFollowsTheTier(t *testing.T) {
	const mvp = "lifecycle:\n  tier: mvp\n"
	const product = "lifecycle:\n  tier: product\nhealth:\n  exclude: [gen/**]\n"
	cases := []struct {
		name, base, old, edit string
		args                  []string
		want                  policyOutcome
	}{
		{"mvp warns", mvp + "health:\n  exclude: [gen/**]\n", "exclude: [gen/**]", "exclude: [gen/**, pkg/**]", nil, policyOutcome{exitOK, "warning", 1}},
		{"mvp with safety fails", mvp + "  bundles: [safety]\nhealth:\n  exclude: [gen/**]\n", "exclude: [gen/**]", "exclude: [gen/**, pkg/**]", nil, policyOutcome{exitFail, "error", 1}},
		{"tier on both sides", product, "tier: product", "tier: mvp", []string{"--tier", "product"}, policyOutcome{exitOK, "", 0}},
		{"stricter limit", product, "exclude: [gen/**]", "exclude: [gen/**]\n  thresholds:\n    health.complex_method: {warn: 5, high: 10}", nil, policyOutcome{exitOK, "", 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newCheckRepo(t, c.base, true)
			r.setManifest(c.old, c.edit)
			code, doc, _ := r.jsonCheck(c.args...)
			wantEqual(t, [2]int{code, len(doc.PolicyWeakened)}, [2]int{c.want.code, c.want.weakened})
			wantEqual(t, c.want.level == "" || doc.PolicyLevel == c.want.level, true)
		})
	}
}

// policyOutcome is what the check decides about an edit of the policy: the exit
// code, the level of the weakening (empty: not checked) and how many there are.
type policyOutcome struct {
	code     int
	level    string
	weakened int
}

// A weakening is recorded, and no longer fails the check, when the head names
// a policyDecision that is an accepted decision and that the base did not
// already name. A proposed or unknown decision records nothing; a decision the
// base already named cannot excuse another weakening; the report still lists
// the weakening and the decision.
//
// Mutation: accepting any policyDecision (or none), or one already in the base,
// fails the matching case.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158
func TestCheckAcceptsARecordedPolicyWeakening(t *testing.T) {
	cases := []struct {
		name, decision string
		want           int
	}{
		{"accepted decision", "ADR-POLICY", exitOK},
		{"proposed decision", "ADR-DRAFT", exitFail},
		{"unknown decision", "ADR-NOPE", exitFail},
		{"no decision", "", exitFail},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { requireDecision(t, c.decision, c.want) })
	}
	// The decision was already named at the base: it records the earlier
	// weakening, not this one.
	r := newCheckRepo(t, policyBase+"  policyDecision: ADR-POLICY\n", true)
	r.setManifest("exclude: [gen/**]", "exclude: [gen/**, pkg/**]")
	code, doc, _ := r.jsonCheck()
	wantEqual(t, [2]any{code, doc.PolicyDecision}, [2]any{exitFail, ""})
}

// policyBase is the manifest at the base of the policy decision tests.
const policyBase = "lifecycle:\n  tier: product\nhealth:\n  exclude: [gen/**]\n"

// weakerPolicy is policyBase with a wider exclude and the policy decision.
func weakerPolicy(decision string) string {
	out := "lifecycle:\n  tier: product\nhealth:\n  exclude: [gen/**, pkg/**]\n"
	if decision != "" {
		out += "  policyDecision: " + decision + "\n"
	}
	return out
}

// requireDecision checks the exit code of a weakening that names a decision
// and, when the decision records it, the report and the text.
func requireDecision(t *testing.T, decision string, want int) {
	t.Helper()
	r := newCheckRepo(t, policyBase, true)
	r.setManifest(policyBase, weakerPolicy(decision))
	code, doc, _ := r.jsonCheck()
	wantEqual(t, [3]any{code, doc.Failed, len(doc.PolicyWeakened)}, [3]any{want, want == exitFail, 1})
	if want != exitOK {
		return
	}
	wantEqual(t, doc.PolicyDecision, "ADR-POLICY")
	_, stdout, _ := r.check()
	wantContains(t, stdout, "the weakening is recorded by decision ADR-POLICY")
}

// A changed source file above 4 MiB is skipped; where a regression is an error
// the check exits 1 and says which file, and where it is a warning it exits 0.
// The cap is the same for the working tree and for --head REV.
//
// Mutation: treating a skipped file as clean, or leaving the revision read
// without a cap, fails.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func TestCheckFailsOnASkippedChangedFile(t *testing.T) {
	for tier, want := range map[string]int{"product": exitFail, "poc": exitOK} {
		t.Run(tier, func(t *testing.T) { requireSkipped(t, tier, want) })
	}
}

// requireSkipped checks that a changed file above the cap is skipped, in the
// working tree and in --head REV, with the exit code the tier gives.
func requireSkipped(t *testing.T, tier string, want int) {
	t.Helper()
	huge := "package pkg\n\n// " + strings.Repeat("x", 4<<20) + "\nfunc Huge() {}\n"
	r := newCheckRepo(t, "lifecycle:\n  tier: "+tier+"\n", true)
	base := strings.TrimSpace(gitIn(t, r.dir, "rev-parse", "HEAD"))
	r.write("pkg/huge.go", huge)
	code, doc, stderr := r.jsonCheck()
	mustEqual(t, [3]any{code, doc.Failed, len(doc.Skipped)}, [3]any{want, want == exitFail, 1})
	wantEqual(t, doc.Skipped[0].Path, "pkg/huge.go")
	if want == exitFail {
		wantContains(t, stderr, "1 changed file(s) were skipped")
	}
	r.commit("huge")
	code, doc, _ = r.jsonCheck("--base", base, "--head", "HEAD")
	wantEqual(t, [2]int{code, len(doc.Skipped)}, [2]int{want, 1})
}

// --cache and --no-cache cannot be combined; where CI is set the check does not
// write the cache unless --cache is given; a tracked file under .engmod refuses
// the cache with a note on stderr and the run goes on.
//
// Mutation: ignoring CI, or the tracked-file rule, fails the matching case.
//
// TRLC-LINKS: REQ-EMG-159, REQ-EMG-158
func TestCheckCacheFlagsAndRules(t *testing.T) {
	t.Setenv("CI", "")
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n", true)
	r.write("pkg/svc.go", svcSource(3))
	r.expect(cliCase{args: []string{"--cache", "--no-cache"}, code: exitUsage, stderr: "cannot be combined"})
	r.expect(cliCase{args: []string{"--no-cache"}, code: exitOK})
	wantEqual(t, cacheWritten(r), false)
	t.Setenv("CI", "true")
	r.expect(cliCase{code: exitOK})
	wantEqual(t, cacheWritten(r), false) // CI is set
	r.expect(cliCase{args: []string{"--cache"}, code: exitOK})
	wantEqual(t, cacheWritten(r), true)
	// Track a file under .engmod: the cache is refused, with a note, even with --cache.
	r.write(".engmod/cache/health/"+strings.Repeat("0a", 32)+".json", "{}")
	gitIn(t, r.dir, "add", "-f", ".engmod")
	_, stderr := r.expect(cliCase{args: []string{"--cache"}, code: exitOK, stderr: "the result cache is not used"})
	wantContains(t, stderr, "tracked by git")
}

// cacheWritten reports whether the project has a result cache directory.
func cacheWritten(r *checkRepo) bool {
	_, err := os.Stat(filepath.Join(r.dir, ".engmod", "cache", "health"))
	return err == nil
}

// The output file is written through a temporary file and a rename, never
// through a symbolic link: an existing symlink (or any non-regular target) is
// refused with exit 2, the file it points to is not touched, and no temporary
// file is left behind. A regular file is replaced.
//
// Mutation: os.WriteFile on the target follows the link and fails this test.
//
// TRLC-LINKS: REQ-EMG-158
func TestOutputIsNotWrittenThroughSymbolicLinks(t *testing.T) {
	manifest := project(t, "lifecycle:\n  tier: mvp\n")
	dir := filepath.Dir(manifest)
	victim := filepath.Join(dir, "victim.txt")
	writeFileOrFail(t, victim, "precious")
	if err := os.Symlink(victim, filepath.Join(dir, "report.json")); err != nil {
		t.Skip("symbolic links are not available")
	}
	requireCLI(t, cliCase{args: []string{"scan", "--model", manifest, "--format", "json", "--out", filepath.Join(dir, "report.json")}, code: exitUsage, stderr: "not a regular file"})
	requireUntouched(t, victim)
	r := newCheckRepo(t, "lifecycle:\n  tier: mvp\n", true)
	rl := filepath.Join(r.dir, "out.json")
	if err := os.Symlink(victim, rl); err != nil {
		t.Fatal(err)
	}
	r.expect(cliCase{args: []string{"--format", "json", "--out", rl}, code: exitUsage, stderr: "not a regular file"})
	requireUntouched(t, victim)
	requireCLI(t, cliCase{args: []string{"scan", "--model", manifest, "--format", "json", "--out", filepath.Join(dir, "sub-nope", "x.json")}, code: exitUsage})
	requireRegularFileReplaced(t, manifest)
}

// writeFileOrFail writes a file (mode 0644) or stops the test.
func writeFileOrFail(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// requireUntouched checks that the file the links point to still holds its
// content.
func requireUntouched(t *testing.T, victim string) {
	t.Helper()
	data, _ := os.ReadFile(victim)
	wantEqual(t, string(data), "precious")
}

// requireRegularFileReplaced checks that scan --out replaces a regular file with
// the report and leaves no temporary file behind.
func requireRegularFileReplaced(t *testing.T, manifest string) {
	t.Helper()
	dir := filepath.Dir(manifest)
	regular := filepath.Join(dir, "regular.json")
	writeFileOrFail(t, regular, "old")
	requireCLI(t, cliCase{args: []string{"scan", "--model", manifest, "--format", "json", "--out", regular}, code: exitOK})
	data, _ := os.ReadFile(regular)
	wantEqual(t, bytes.HasPrefix(data, []byte("{")), true)
	leftovers, _ := filepath.Glob(filepath.Join(dir, ".enghealth-*"))
	wantEqual(t, len(leftovers), 0)
}

// safeText turns every control character into a visible escape and leaves text
// without one untouched.
//
// TRLC-LINKS: REQ-EMG-158
func TestSafeText(t *testing.T) {
	for in, want := range map[string]string{
		"plain/path.go":                 "plain/path.go",
		"naïve ünï.go":                  "naïve ünï.go",
		"a\nb":                          `a\x0ab`,
		"::error::x\n::stop-commands::": `::error::x\x0a::stop-commands::`,
		"\x1b[2J":                       `\x1b[2J`,
		"del\x7f":                       `del\x7f`,
		"c1\u009b":                      `c1\x9b`,
		"sep\u2028x":                    `sep\u2028x`,
		`back\slash`:                    `back\slash`,
		"":                              "",
	} {
		if got := safeText(in); got != want {
			t.Errorf("safeText(%q) = %q, want %q", in, got, want)
		}
	}
}

// Text output never prints a control character from the repository: a function
// name or a message with a newline or an escape sequence is escaped in the scan
// report, in the check summary and in the lines of the regressions, so nothing
// can start a workflow command (::error::) in a log or move a terminal cursor.
//
// Mutation: printing the raw strings fails.
//
// TRLC-LINKS: REQ-EMG-158
func TestTextOutputEscapesControlCharacters(t *testing.T) {
	var report engmodel.HealthReport
	report.Engine.Name, report.Engine.Version = "treesitter", "1"
	report.Rollups.Project.Worst = &health.WorstFile{Path: "pkg/a\nb.go", Score: 3}
	report.Diagnostics = []engmodel.HealthDiagnostic{{Code: "health.file_skipped", Level: "warning", Message: "pkg/x\n::error::y was not analysed"}}
	text := renderText(report)
	wantEqual(t, leaksControl(text), false)
	wantAll(t, text, `pkg/a\x0ab.go`, `pkg/x\x0a::error::y`)
}

// leaksControl reports an escape character or a line that starts a workflow
// command in a text.
func leaksControl(out string) bool {
	return strings.Contains(out, "\x1b") || strings.Contains(out, "\n::") || strings.Contains(out, "\n      ::")
}

// The check summary and the lines of the regressions and failures escape the
// control characters of the repository too.
//
// Mutation: printing the raw strings fails.
//
// TRLC-LINKS: REQ-EMG-158
func TestCheckTextOutputEscapesControlCharacters(t *testing.T) {
	evil := "Big\n::error::injected\x1b[2J"
	var check engmodel.HealthCheckReport
	check.Notes = []string{"skipped a\n::error::b: too large"}
	check.Blocking = []engmodel.HealthDeltaEntry{{Path: "pkg/a\nb.go", Function: evil, Code: "health.complex_method", Change: "new", Severity: "high", Metric: "cyclomatic", ValueAfter: 21, StartLine: 3, Message: "m\n::error::z"}}
	check.Skipped = []engmodel.HealthSkippedFile{{Path: "gen/\n::error::s.go", Reason: "r\x1b[0m"}}
	check.Policy = []engmodel.HealthPolicyChange{{Aspect: "exclude", Message: "pattern \"\n::error::p\""}}
	for name, out := range map[string]string{"summary": renderCheckText(check), "lines": regressionLines(check) + failureLines(check)} {
		wantEqual(t, [2]any{name, leaksControl(out)}, [2]any{name, false})
	}
}

// scan --import trusts the levels of the file it reads unless a model or a
// tier says otherwise, so --fail-on-error with an import and neither is a usage
// error (exit 2), not a pass; with --tier or --model the levels are recomputed
// and the flag works.
//
// Mutation: letting --fail-on-error read the imported summary fails.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func TestImportedReportCannotDecideTheGate(t *testing.T) {
	manifest := project(t, "lifecycle:\n  tier: product\n")
	forged := forgedReport(t, manifest)
	requireCLI(t, cliCase{args: []string{"scan", "--import", forged, "--fail-on-error"}, code: exitUsage, stderr: "levels of the imported report are not trusted"})
	requireCLI(t, cliCase{args: []string{"scan", "--import", forged, "--tier", "product", "--fail-on-error"}, code: exitFail}) // levels recomputed
	requireCLI(t, cliCase{args: []string{"scan", "--import", forged, "--model", manifest, "--fail-on-error"}, code: exitFail})
	requireCLI(t, cliCase{args: []string{"scan", "--import", forged}, code: exitOK}) // without the flag it renders
}

// forgedReport scans the project and writes a copy of the report that claims
// no errors; it returns its path.
func forgedReport(t *testing.T, manifest string) string {
	t.Helper()
	dir := filepath.Dir(manifest)
	report := filepath.Join(dir, "report.json")
	requireCLI(t, cliCase{args: []string{"scan", "--model", manifest, "--format", "json", "--out", report}, code: exitOK})
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	forged := strings.ReplaceAll(string(data), `"complex_method": "error"`, `"complex_method": "warning"`)
	forged = strings.Replace(forged, `"errors": 1`, `"errors": 0`, 1)
	path := filepath.Join(dir, "forged.json")
	writeFileOrFail(t, path, forged)
	return path
}
