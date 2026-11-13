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
	cases := []struct {
		name, base, head, aspect string
	}{
		{"limit raised", product + "health:\n  exclude: [gen/**]\n", product + "health:\n  exclude: [gen/**]\n  thresholds:\n    health.complex_method: {warn: 12, high: 25}\n", "threshold"},
		{"exclude widened", product + "health:\n  exclude: [gen/**]\n", product + "health:\n  exclude: [gen/**, pkg/**]\n", "exclude"},
		{"override raises", product + "health:\n  exclude: [gen/**]\n", product + "health:\n  exclude: [gen/**]\n  overrides:\n    - path: pkg/**\n      thresholds:\n        health.large_method: {warn: 300, high: 600}\n", "override"},
		{"tier lowered", product + "health:\n  exclude: [gen/**]\n", "lifecycle:\n  tier: mvp\nhealth:\n  exclude: [gen/**]\n", "tier"},
		{"lifecycle removed", product + "health:\n  exclude: [gen/**]\n", "health:\n  exclude: [gen/**]\n", "tier"},
	}
	for _, c := range cases {
		r := newCheckRepo(t, c.base, true)
		r.setManifest(c.base, c.head)
		code, doc, stderr := r.jsonCheck()
		if code != exitFail || !doc.Failed || doc.PolicyLevel != "error" || len(doc.PolicyWeakened) != 1 || doc.PolicyWeakened[0].Aspect != c.aspect {
			t.Errorf("%s: exit %d, %+v, stderr %q; want 1 with one %s weakening", c.name, code, doc.PolicyWeakened, stderr, c.aspect)
		}
		if !strings.Contains(stderr, "weakening(s) of the health policy against the base revision are not recorded by a policyDecision") {
			t.Errorf("%s: stderr = %q", c.name, stderr)
		}
		_, stdout, _ := r.check()
		if !strings.Contains(stdout, "policy weakened ("+c.aspect+")") || !strings.Contains(stdout, "result: FAIL") {
			t.Errorf("%s: text lacks the weakening:\n%s", c.name, stdout)
		}
	}

	// Below product the same edit is a warning at both tiers: exit 0.
	const mvp = "lifecycle:\n  tier: mvp\n"
	r := newCheckRepo(t, mvp+"health:\n  exclude: [gen/**]\n", true)
	r.setManifest("exclude: [gen/**]", "exclude: [gen/**, pkg/**]")
	if code, doc, _ := r.jsonCheck(); code != exitOK || doc.Failed || doc.PolicyLevel != "warning" || len(doc.PolicyWeakened) != 1 {
		t.Errorf("mvp: exit %d %+v, want 0 with a warning", code, doc)
	}
	// With the safety bundle the level is an error from mvp.
	r = newCheckRepo(t, mvp+"  bundles: [safety]\nhealth:\n  exclude: [gen/**]\n", true)
	r.setManifest("exclude: [gen/**]", "exclude: [gen/**, pkg/**]")
	if code, doc, _ := r.jsonCheck(); code != exitFail || doc.PolicyLevel != "error" {
		t.Errorf("mvp with safety: exit %d %+v, want 1", code, doc)
	}
	// A tier given on the command line applies to both sides: no tier weakening.
	r = newCheckRepo(t, product+"health:\n  exclude: [gen/**]\n", true)
	r.setManifest("tier: product", "tier: mvp")
	if code, doc, _ := r.jsonCheck("--tier", "product"); code != exitOK || len(doc.PolicyWeakened) != 0 {
		t.Errorf("--tier on both sides: exit %d %+v", code, doc.PolicyWeakened)
	}
	// An unchanged or stricter policy, and a new exclude that the base had, are fine.
	r = newCheckRepo(t, product+"health:\n  exclude: [gen/**]\n", true)
	r.setManifest("exclude: [gen/**]", "exclude: [gen/**]\n  thresholds:\n    health.complex_method: {warn: 5, high: 10}")
	if code, doc, _ := r.jsonCheck(); code != exitOK || len(doc.PolicyWeakened) != 0 {
		t.Errorf("a stricter limit: exit %d %+v", code, doc.PolicyWeakened)
	}
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
	const base = "lifecycle:\n  tier: product\nhealth:\n  exclude: [gen/**]\n"
	weaker := func(decision string) string {
		out := "lifecycle:\n  tier: product\nhealth:\n  exclude: [gen/**, pkg/**]\n"
		if decision != "" {
			out += "  policyDecision: " + decision + "\n"
		}
		return out
	}
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
		r := newCheckRepo(t, base, true)
		r.setManifest(base, weaker(c.decision))
		code, doc, stderr := r.jsonCheck()
		if code != c.want || doc.Failed != (c.want == exitFail) || len(doc.PolicyWeakened) != 1 {
			t.Errorf("%s: exit %d failed %v weakened %d, stderr %q; want exit %d", c.name, code, doc.Failed, len(doc.PolicyWeakened), stderr, c.want)
		}
		if c.want == exitOK && doc.PolicyDecision != "ADR-POLICY" {
			t.Errorf("%s: policyDecision = %q", c.name, doc.PolicyDecision)
		}
		if c.want == exitOK {
			if _, stdout, _ := r.check(); !strings.Contains(stdout, "the weakening is recorded by decision ADR-POLICY") {
				t.Errorf("%s: text does not name the decision:\n%s", c.name, stdout)
			}
		}
	}

	// The decision was already named at the base: it records the earlier
	// weakening, not this one.
	named := base + "  policyDecision: ADR-POLICY\n"
	r := newCheckRepo(t, named, true)
	r.setManifest("exclude: [gen/**]", "exclude: [gen/**, pkg/**]")
	if code, doc, _ := r.jsonCheck(); code != exitFail || doc.PolicyDecision != "" {
		t.Errorf("a decision the base already named: exit %d %+v, want 1", code, doc)
	}
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
	huge := "package pkg\n\n// " + strings.Repeat("x", 4<<20) + "\nfunc Huge() {}\n"
	for _, tier := range []string{"product", "poc"} {
		r := newCheckRepo(t, "lifecycle:\n  tier: "+tier+"\n", true)
		base := strings.TrimSpace(gitIn(t, r.dir, "rev-parse", "HEAD"))
		r.write("pkg/huge.go", huge)
		want := exitFail
		if tier == "poc" {
			want = exitOK
		}
		code, doc, stderr := r.jsonCheck()
		if code != want || doc.Failed != (want == exitFail) || len(doc.Skipped) != 1 || doc.Skipped[0].Path != "pkg/huge.go" {
			t.Errorf("%s working tree: exit %d %+v stderr %q; want %d", tier, code, doc.Skipped, stderr, want)
		}
		if want == exitFail && !strings.Contains(stderr, "1 changed file(s) were skipped") {
			t.Errorf("%s: stderr %q", tier, stderr)
		}
		r.commit("huge")
		code, doc, _ = r.jsonCheck("--base", base, "--head", "HEAD")
		if code != want || len(doc.Skipped) != 1 {
			t.Errorf("%s --head: exit %d %+v; want %d", tier, code, doc.Skipped, want)
		}
	}
}

// --cache and --no-cache cannot be combined; where CI is set the check does not
// write the cache unless --cache is given; a tracked file under .engmod refuses
// the cache with a note on stderr and the run goes on.
//
// Mutation: ignoring CI, or the tracked-file rule, fails the matching case.
//
// TRLC-LINKS: REQ-EMG-159, REQ-EMG-158
func TestCheckCacheFlagsAndRules(t *testing.T) {
	cacheDir := func(r *checkRepo) string { return filepath.Join(r.dir, ".engmod", "cache", "health") }
	t.Setenv("CI", "")
	r := newCheckRepo(t, "lifecycle:\n  tier: product\n", true)
	r.write("pkg/svc.go", svcSource(3))
	if code, _, stderr := r.check("--cache", "--no-cache"); code != exitUsage || !strings.Contains(stderr, "cannot be combined") {
		t.Errorf("--cache --no-cache: %d %q", code, stderr)
	}
	if code, _, _ := r.check("--no-cache"); code != exitOK {
		t.Fatal("check --no-cache failed")
	}
	if _, err := os.Stat(cacheDir(r)); err == nil {
		t.Error("--no-cache wrote the cache")
	}
	t.Setenv("CI", "true")
	if code, _, _ := r.check(); code != exitOK {
		t.Fatal("check in CI failed")
	}
	if _, err := os.Stat(cacheDir(r)); err == nil {
		t.Error("the cache was written although CI is set")
	}
	if code, _, _ := r.check("--cache"); code != exitOK {
		t.Fatal("check --cache failed")
	}
	if _, err := os.Stat(cacheDir(r)); err != nil {
		t.Errorf("--cache in CI did not write the cache: %v", err)
	}

	// Track a file under .engmod: the cache is refused, with a note, even with --cache.
	r.write(".engmod/cache/health/"+strings.Repeat("0a", 32)+".json", "{}")
	gitIn(t, r.dir, "add", "-f", ".engmod")
	code, _, stderr := r.check("--cache")
	if code != exitOK || !strings.Contains(stderr, "the result cache is not used") || !strings.Contains(stderr, "tracked by git") {
		t.Errorf("tracked .engmod: exit %d stderr %q; want a note and a normal run", code, stderr)
	}
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
	if err := os.WriteFile(victim, []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "report.json")
	if err := os.Symlink(victim, link); err != nil {
		t.Skip("symbolic links are not available")
	}
	code, _, stderr := runCLI("scan", "--model", manifest, "--format", "json", "--out", link)
	if code != exitUsage || !strings.Contains(stderr, "not a regular file") {
		t.Errorf("scan --out through a link: %d %q", code, stderr)
	}
	if data, _ := os.ReadFile(victim); string(data) != "precious" {
		t.Errorf("the target of the link was overwritten: %q", data)
	}
	r := newCheckRepo(t, "lifecycle:\n  tier: mvp\n", true)
	rl := filepath.Join(r.dir, "out.json")
	if err := os.Symlink(victim, rl); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := r.check("--format", "json", "--out", rl); code != exitUsage || !strings.Contains(stderr, "not a regular file") {
		t.Errorf("check --out through a link: %d %q", code, stderr)
	}
	if data, _ := os.ReadFile(victim); string(data) != "precious" {
		t.Errorf("the target of the link was overwritten by check: %q", data)
	}
	if code, _, _ := runCLI("scan", "--model", manifest, "--format", "json", "--out", filepath.Join(dir, "sub-nope", "x.json")); code != exitUsage {
		t.Errorf("an output in a missing directory must exit 2, got %d", code)
	}

	regular := filepath.Join(dir, "regular.json")
	if err := os.WriteFile(regular, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := runCLI("scan", "--model", manifest, "--format", "json", "--out", regular); code != exitOK {
		t.Fatalf("scan --out: %d %q", code, stderr)
	}
	if data, _ := os.ReadFile(regular); !bytes.HasPrefix(data, []byte("{")) {
		t.Errorf("the regular file was not replaced: %q", data)
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, ".enghealth-*"))
	if len(leftovers) != 0 {
		t.Errorf("temporary files left behind: %v", leftovers)
	}
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
	evil := "Big\n::error::injected\x1b[2J"
	var report engmodel.HealthReport
	report.Engine.Name, report.Engine.Version = "treesitter", "1"
	report.Rollups.Project.Worst = &health.WorstFile{Path: "pkg/a\nb.go", Score: 3}
	report.Diagnostics = []engmodel.HealthDiagnostic{{Code: "health.file_skipped", Level: "warning", Message: "pkg/x\n::error::y was not analysed"}}
	text := renderText(report)
	if strings.ContainsAny(text, "\x1b") || strings.Contains(text, "\n::error::") {
		t.Errorf("renderText leaked a control character:\n%q", text)
	}
	if !strings.Contains(text, `pkg/a\x0ab.go`) || !strings.Contains(text, `pkg/x\x0a::error::y`) {
		t.Errorf("renderText did not escape:\n%q", text)
	}

	var check engmodel.HealthCheckReport
	check.Notes = []string{"skipped a\n::error::b: too large"}
	check.Blocking = []engmodel.HealthDeltaEntry{{Path: "pkg/a\nb.go", Function: evil, Code: "health.complex_method", Change: "new", Severity: "high", Metric: "cyclomatic", ValueAfter: 21, StartLine: 3, Message: "m\n::error::z"}}
	check.Skipped = []engmodel.HealthSkippedFile{{Path: "gen/\n::error::s.go", Reason: "r\x1b[0m"}}
	check.Policy = []engmodel.HealthPolicyChange{{Aspect: "exclude", Message: "pattern \"\n::error::p\""}}
	for name, out := range map[string]string{"summary": renderCheckText(check), "lines": regressionLines(check) + failureLines(check)} {
		if strings.Contains(out, "\x1b") || strings.Contains(out, "\n::") || strings.Contains(out, "\n      ::") {
			t.Errorf("%s leaked a control character:\n%q", name, out)
		}
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
	dir := filepath.Dir(manifest)
	report := filepath.Join(dir, "report.json")
	if code, _, stderr := runCLI("scan", "--model", manifest, "--format", "json", "--out", report); code != exitOK {
		t.Fatalf("scan: %d %q", code, stderr)
	}
	// The imported report claims no errors.
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	forged := strings.ReplaceAll(string(data), `"complex_method": "error"`, `"complex_method": "warning"`)
	forged = strings.Replace(forged, `"errors": 1`, `"errors": 0`, 1)
	forgedPath := filepath.Join(dir, "forged.json")
	if err := os.WriteFile(forgedPath, []byte(forged), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := runCLI("scan", "--import", forgedPath, "--fail-on-error"); code != exitUsage || !strings.Contains(stderr, "levels of the imported report are not trusted") {
		t.Errorf("--import --fail-on-error: %d %q, want 2", code, stderr)
	}
	if code, _, stderr := runCLI("scan", "--import", forgedPath, "--tier", "product", "--fail-on-error"); code != exitFail {
		t.Errorf("--import --tier product --fail-on-error: %d %q, want 1 (levels recomputed)", code, stderr)
	}
	if code, _, stderr := runCLI("scan", "--import", forgedPath, "--model", manifest, "--fail-on-error"); code != exitFail {
		t.Errorf("--import --model --fail-on-error: %d %q, want 1", code, stderr)
	}
	if code, _, _ := runCLI("scan", "--import", forgedPath); code != exitOK {
		t.Errorf("--import without the flag renders: %d", code)
	}
}
