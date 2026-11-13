// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// withEngineVersion makes the running engine report v for the test.
func withEngineVersion(t *testing.T, v string) {
	t.Helper()
	old := engineVersion
	engineVersion = func() string { return v }
	t.Cleanup(func() { engineVersion = old })
}

// templateFiles is the content of a check template: a go.mod and a main.go,
// each skipped when empty.
type templateFiles struct{ gomod, source string }

// writeTemplate writes dir/.github/engmod-check with the files.
func writeTemplate(t *testing.T, dir string, files templateFiles) {
	t.Helper()
	check := filepath.Join(dir, ".github", "engmod-check")
	if files.gomod != "" {
		write(t, filepath.Join(check, "go.mod"), files.gomod)
	}
	if files.source != "" {
		write(t, filepath.Join(check, "main.go"), files.source)
	}
}

const (
	templateGoMod = "module example.com/engmod-check\n\ngo 1.25\n"
	currentSource = "package main\n\nfunc main() { _ = \"workspace-root\" }\n"
	staleSource   = "package main\n\nfunc main() { _ = \"summary\" }\n"
)

// engineRequire is a check template go.mod pinning the engine at version v.
func engineRequire(v string) string {
	return templateGoMod + "\nrequire github.com/labeth/engineering-model-go " + v + "\n"
}

// A repository whose check template cannot take the workspace root, or that
// has none, is the error engine.check_template_stale with the remedy to run
// engflow upgrade for its directory and is not checked by the library; the
// in-process option checks it with the library as before; a current template
// is checked by it.
// Mutation: falling back to routeLibrary in routeOf (the old behaviour), or
// ignoring opts.InProcess, fails a row.
// TRLC-LINKS: REQ-EMG-180
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestStaleCheckTemplateIsAnErrorNotAFallback(t *testing.T) {
	cases := []struct {
		name      string
		gomod     string
		source    string
		inProcess bool
		route     checkRoute
		stale     bool
	}{
		{"template without the workspace flag", templateGoMod, staleSource, false, routeStale, true},
		{"no template at all", "", "", false, routeStale, true},
		{"template go.mod without a source", templateGoMod, "", false, routeStale, true},
		{"source with the flag but no go.mod", "", currentSource, false, routeStale, true},
		{"stale template, in-process requested", templateGoMod, staleSource, true, routeLibrary, false},
		{"no template, in-process requested", "", "", true, routeLibrary, false},
		{"current template", templateGoMod, currentSource, false, routeExec, false},
		{"current template, in-process requested", templateGoMod, currentSource, true, routeLibrary, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			dir := checkout(t, root, "if-a")
			writeTemplate(t, dir, templateFiles{c.gomod, c.source})
			opts := CheckOptions{InProcess: c.inProcess}
			if got := routeOf(dir, opts); got != c.route {
				t.Fatalf("route %d, want %d", got, c.route)
			}
			if c.route == routeExec {
				return // a current template is exec'd; the pin tests run one
			}
			result := RunChecks(context.Background(), root, []CheckTarget{{Entry: Entry{Checkout: Checkout{Name: "if-a", Dir: dir}}, Reason: "local"}}, opts)[0]
			if c.stale {
				assertStaleResult(t, result, dir)
				return
			}
			assertLibraryResult(t, result)
		})
	}
}

// assertStaleResult checks a result is the stale error with its fix line and
// no library check.
// TRLC-LINKS: REQ-EMG-180
func assertStaleResult(t *testing.T, result CheckResult, dir string) {
	t.Helper()
	if !result.Failed || result.Errors != 1 || strings.Contains(result.Method, "library") {
		t.Fatalf("want one error and no library check, got %+v", result)
	}
	if len(result.Lines) != 2 || !strings.HasPrefix(result.Lines[0], "engine.check_template_stale [error] ") {
		t.Fatalf("lines %q", result.Lines)
	}
	if want := "  fix: run engflow upgrade --root " + dir + " "; !strings.HasPrefix(result.Lines[1], want) {
		t.Fatalf("fix line %q, want prefix %q", result.Lines[1], want)
	}
}

// assertLibraryResult checks a result is a passing library check.
// TRLC-LINKS: REQ-EMG-180
func assertLibraryResult(t *testing.T, result CheckResult) {
	t.Helper()
	if result.Method != "library" || result.Failed || result.Errors != 0 {
		t.Fatalf("want a passing library check, got %+v", result)
	}
}

// assertOutputHas fails for each substring the output lacks.
// TRLC-LINKS: REQ-EMG-180
func assertOutputHas(t *testing.T, out string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("output lacks %q:\n%s", w, out)
		}
	}
}

// The error and its fix line reach the CLI output and the exit code is 1; with
// --in-process the same repository passes.
// Mutation: dropping the stale result from RunChecks (checking it with the
// library) makes the exit code 0 and fails this test.
// TRLC-LINKS: REQ-EMG-180
func TestCheckCLIReportsStaleTemplate(t *testing.T) {
	root := t.TempDir()
	dir := checkout(t, root, "if-a")
	code, out, _ := runMain(t, allPublished("if-a"), "--root", root, "check")
	if code != 1 {
		t.Fatalf("stale check exits %d, want 1:\n%s", code, out)
	}
	assertOutputHas(t, out, "FAIL if-a (1 errors", "engine.check_template_stale [error]", "fix: run engflow upgrade --root "+dir)
	code, out, _ = runMain(t, allPublished("if-a"), "--root", root, "check", "--in-process")
	if code != 0 || strings.Contains(out, "engine.check_template_stale") {
		t.Fatalf("--in-process check: %d\n%s", code, out)
	}
	assertOutputHas(t, out, "ok   if-a")
}

// engine.pin_outdated is reported for a go.mod that pins an engine older than
// the running one and for nothing else.
// Mutation: comparing with <= instead of < (an equal pin reported), dropping
// the replace test, or not skipping a devel engine fails a row.
// TRLC-LINKS: REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestPinOutdatedDiagnostics(t *testing.T) {
	for _, c := range pinCases {
		t.Run(c.name, func(t *testing.T) { assertPinCase(t, c) })
	}
}

// pinCase is one go.mod of a check template against one running engine.
type pinCase struct {
	name    string
	running string
	gomod   string // "" writes no go.mod
	goWork  bool
	want    bool
}

var pinCases = []pinCase{
	{"older pin", "v0.12.0", engineRequire("v0.10.0"), false, true},
	{"older pin, newer pre-release engine", "v0.12.0-rc.1", engineRequire("v0.11.0"), false, true},
	{"equal pin", "v0.12.0", engineRequire("v0.12.0"), false, false},
	{"newer pin", "v0.12.0", engineRequire("v0.13.0"), false, false},
	{"older pin with a replace directive", "v0.12.0", engineRequire("v0.10.0") + "\nreplace github.com/labeth/engineering-model-go => ../../../engineering-model-go\n", false, false},
	{"older pin with a go.work next to it", "v0.12.0", engineRequire("v0.10.0"), true, false},
	{"missing go.mod", "v0.12.0", "", false, false},
	{"go.mod without the engine", "v0.12.0", templateGoMod, false, false},
	{"unparsable pin", "v0.12.0", engineRequire("banana"), false, false},
	{"devel running engine", "(devel)", engineRequire("v0.10.0"), false, false},
	{"unknown running engine", "", engineRequire("v0.10.0"), false, false},
	{"pseudo-version running engine", "v0.12.1-0.20260101000000-abcdefabcdef", engineRequire("v0.10.0"), false, false},
	{"dirty running engine", "v0.12.0+dirty", engineRequire("v0.10.0"), false, false},
}

// assertPinCase writes the template of the case and checks the diagnostics.
// TRLC-LINKS: REQ-EMG-181
func assertPinCase(t *testing.T, c pinCase) {
	t.Helper()
	withEngineVersion(t, c.running)
	dir := t.TempDir()
	writeTemplate(t, dir, templateFiles{c.gomod, currentSource})
	if c.goWork {
		write(t, filepath.Join(dir, ".github", "engmod-check", GoWorkFileName), "go 1.25\n\nuse .\n")
	}
	diags := pinOutdatedDiagnostics(dir)
	if c.want {
		assertPinOutdated(t, diags, c.running)
	} else if len(diags) != 0 {
		t.Fatalf("unexpected %+v", diags)
	}
}

// assertPinOutdated checks the one engine.pin_outdated error and its remedy.
// TRLC-LINKS: REQ-EMG-181
func assertPinOutdated(t *testing.T, diags []validate.Diagnostic, running string) {
	t.Helper()
	if len(diags) != 1 || diags[0].Code != "engine.pin_outdated" || diags[0].Severity != validate.SeverityError {
		t.Fatalf("want one engine.pin_outdated error, got %+v", diags)
	}
	want := "bump the pin: go get github.com/labeth/engineering-model-go@" + running + " in .github/engmod-check"
	if diags[0].Remedy != want {
		t.Fatalf("remedy %q, want %q", diags[0].Remedy, want)
	}
}

// A repository checked by its own template carries engine.pin_outdated in its
// result when the pin is older than the running engine, and a current pin
// adds nothing.
// Mutation: not calling pinOutdatedDiagnostics from pinnedExec fails the
// older row.
// TRLC-LINKS: REQ-EMG-181
func TestRunChecksReportsOutdatedPinOfExecCheck(t *testing.T) {
	for _, c := range []struct {
		name, pin string
		want      bool
	}{{"older pin", "v0.10.0", true}, {"equal pin", "v0.12.0", false}} {
		t.Run(c.name, func(t *testing.T) {
			withEngineVersion(t, "v0.12.0")
			root := t.TempDir()
			dir := checkout(t, root, "if-a")
			writeTemplate(t, dir, templateFiles{engineRequire(c.pin), currentSource})
			target := CheckTarget{Entry: Entry{Checkout: Checkout{Name: "if-a", Dir: dir}}, Reason: "local"}
			result := RunChecks(context.Background(), root, []CheckTarget{target}, CheckOptions{})[0]
			joined := strings.Join(result.Lines, "\n")
			if got := strings.Contains(joined, "engine.pin_outdated [error]"); got != c.want {
				t.Fatalf("pin_outdated reported %v, want %v:\n%s", got, c.want, joined)
			}
			if c.want && (!result.Failed || !strings.Contains(joined, "fix: bump the pin: go get github.com/labeth/engineering-model-go@v0.12.0")) {
				t.Fatalf("outdated pin must fail with its fix line: %+v", result)
			}
			if !strings.HasPrefix(result.Method, "exec") {
				t.Fatalf("method %q", result.Method)
			}
		})
	}
}
