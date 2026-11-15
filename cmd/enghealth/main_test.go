// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

// project writes a small model with one complex function and returns the
// manifest path; extra is inserted into the manifest (lifecycle, health).
func project(t *testing.T, extra string) string {
	t.Helper()
	dir := t.TempDir()
	testutil.WriteV2ModuleFiles(t, dir, "example.com/hc@v0", "v0.1.0", "hc", extra, `  functionalGroups:
    - id: FG-A
      name: Group
  functionalUnits:
    - id: FU-A
      name: Unit A
      group: FG-A
`)
	testutil.WriteFile(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 3\nlintRun: {}\nrequirements:\n  - id: REQ-X\n    text: The unit shall decide.\n")
	// Decisions that the waivers, the baseline and the policy decision of the
	// tests refer to: a waiver (ADR-EMG-051, ADR-EMG-052), a baseline or a policy
	// decision whose decision is not accepted does nothing; ADR-DRAFT is proposed.
	testutil.WriteFile(t, filepath.Join(dir, "model", "decisions.yml"), "schemaVersion: 3\ndecisions:\n"+
		"  - {id: ADR-W, title: Waive, status: accepted, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n"+
		"  - {id: ADR-SVC, title: Waive svc, status: accepted, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n"+
		"  - {id: ADR-DEBT, title: Carry debt, status: accepted, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n"+
		"  - {id: ADR-POLICY, title: Weaken policy, status: accepted, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n"+
		"  - {id: ADR-DRAFT, title: Not yet, status: proposed, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n")
	var b strings.Builder
	b.WriteString("// ENGMODEL-OWNER-UNIT: FU-A\npackage pkg\n\n// TRLC-LINKS: REQ-X\nfunc Big(a int) int {\n")
	for i := 0; i < 16; i++ {
		fmt.Fprintf(&b, "\tif a == %d { return %d }\n", i, i)
	}
	b.WriteString("\treturn 0\n}\n")
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "big.go"), b.String())
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "ok.go"), "// ENGMODEL-OWNER-UNIT: FU-A\npackage pkg\n\n// TRLC-LINKS: REQ-X\nfunc Ok() int { return 1 }\n")
	return filepath.Join(dir, "engmod.yml")
}

// runCLI runs enghealth with args and returns the exit code and the outputs.
func runCLI(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// The dispatcher: no arguments and an unknown command are usage errors (2),
// help is 0, and scan and check are registered.
//
// TRLC-LINKS: REQ-EMG-158
func TestRunDispatch(t *testing.T) {
	requireCLI(t, cliCase{code: exitUsage, stderr: "usage: enghealth"})
	requireCLI(t, cliCase{args: []string{"help"}, code: exitOK, stdout: "scan"})
	requireCLI(t, cliCase{args: []string{"frobnicate"}, code: exitUsage, stderr: "unknown command"})
	for _, name := range []string{"scan", "check", "baseline"} {
		wantEqual(t, [2]any{name, commands[name].run != nil}, [2]any{name, true})
	}
}

// Flag errors and setup errors exit 2 and never 0 or 1: no model, an unknown
// format or tier, a model that does not load, a path outside the project, a
// bad flag.
//
// TRLC-LINKS: REQ-EMG-158
func TestScanSetupErrorsExitTwo(t *testing.T) {
	manifest := project(t, "")
	cases := [][]string{
		{"scan"},
		{"scan", "--model", manifest, "--format", "xml"},
		{"scan", "--model", manifest, "--tier", "gold"},
		{"scan", "--model", filepath.Join(filepath.Dir(manifest), "nope.yml")},
		{"scan", "--model", manifest, os.TempDir()},
		{"scan", "--no-such-flag"},
		{"scan", "--import", filepath.Join(filepath.Dir(manifest), "nope.json")},
	}
	for _, args := range cases {
		if code, _, _ := runCLI(args...); code != exitUsage {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if code, _, _ := runCLI("scan", "-h"); code != exitOK {
		t.Errorf("scan -h: exit %d, want 0", code)
	}
}

// The exit code follows the tier: findings are errors at product, so
// --fail-on-error exits 1 there, 0 at mvp where they are warnings, and 0 at
// product without the flag. The report is still written.
//
// Mutation: ignoring the level (failing at mvp) or failing without the flag
// fails this test.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func TestScanExitCodeFollowsTheTier(t *testing.T) {
	manifest := project(t, "lifecycle:\n  tier: product\n")
	code, stdout, stderr := runCLI("scan", "--model", manifest, "--fail-on-error")
	if code != exitFail || !strings.Contains(stdout, "project:") || !strings.Contains(stderr, "are errors at tier product") {
		t.Errorf("product with flag: %d %q %q", code, stdout, stderr)
	}
	if code, _, _ := runCLI("scan", "--model", manifest); code != exitOK {
		t.Errorf("product without flag: exit %d, want 0", code)
	}
	if code, _, _ := runCLI("scan", "--model", manifest, "--tier", "mvp", "--fail-on-error"); code != exitOK {
		t.Errorf("mvp with flag: exit %d, want 0", code)
	}
	if code, _, _ := runCLI("scan", "--model", manifest, "--tier", "operational", "--fail-on-error"); code != exitFail {
		t.Errorf("operational with flag: exit %d, want 1", code)
	}
}

// The text format is a short summary: the project score and band, the
// functional units, the worst functions with their findings and the counts.
// The project score is hand-computed: (8.0 x 20 + 10.0 x 2) / 22 = 8.18, 8.2.
//
// TRLC-LINKS: REQ-EMG-158
func TestScanTextSummary(t *testing.T) {
	manifest := project(t, "lifecycle:\n  tier: product\n")
	code, stdout, _ := runCLI("scan", "--model", manifest)
	if code != exitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{
		"tier product", "project: 8.2 green (2 production files", "functional units:", "FU-A",
		"worst 1 functions:", "pkg/big.go:5 Big", "health.complex_method", "[error]", "findings: 1 (2 errors, 0 warnings at tier product), 0 waived",
		"files below the floor of 10.0: 1, 0 covered by the baseline, 1 not covered", "health.file_below_floor",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("text output lacks %q:\n%s", want, stdout)
		}
	}
}

// The json format validates against the published schema and holds no absolute
// path; the sarif format is SARIF 2.1.0; --out writes the file and leaves stdout
// empty.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func TestScanJSONSARIFAndOut(t *testing.T) {
	// The floor of 8.0 is the score of big.go, so the output holds only its finding.
	manifest := project(t, "lifecycle:\n  tier: product\nhealth:\n  floor: 8.0\n")
	stdout, _ := requireCLI(t, cliCase{args: []string{"scan", "--model", manifest, "--format", "json"}, code: exitOK})
	_, err := engmodel.ImportHealthReport(strings.NewReader(stdout))
	wantEqual(t, err, nil) // the json output is a conforming report
	wantNone(t, stdout, filepath.Dir(manifest))
	requireSARIFOut(t, manifest)
	requireCLI(t, cliCase{args: []string{"scan", "--model", manifest, "--out", filepath.Join(t.TempDir(), "no", "dir", "x.txt")}, code: exitUsage})
}

// requireSARIFOut checks that scan --format sarif --out writes SARIF 2.1.0 with
// the one finding to the file and nothing to stdout.
func requireSARIFOut(t *testing.T, manifest string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "report.sarif")
	stdout, _ := requireCLI(t, cliCase{args: []string{"scan", "--model", manifest, "--format", "sarif", "--out", out}, code: exitOK})
	wantEqual(t, stdout, "")
	var doc struct {
		Version string `json:"version"`
		Runs    []struct {
			Results []struct {
				RuleID string `json:"ruleId"`
				Level  string `json:"level"`
			} `json:"results"`
		} `json:"runs"`
	}
	mustEqual(t, json.Unmarshal(mustRead(t, out), &doc), nil)
	mustEqual(t, [2]any{doc.Version, len(doc.Runs)}, [2]any{"2.1.0", 1})
	mustEqual(t, len(doc.Runs[0].Results), 1)
	wantEqual(t, [2]string{doc.Runs[0].Results[0].RuleID, doc.Runs[0].Results[0].Level}, [2]string{"health.complex_method", "error"})
}

// --import re-renders a conforming report of another engine in any format, can
// read it at the tier of a flag or a model, and rejects what does not conform
// with exit 2.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func TestScanImportRerendersAReport(t *testing.T) {
	manifest := project(t, "lifecycle:\n  tier: mvp\n")
	report := filepath.Join(t.TempDir(), "report.json")
	requireCLI(t, cliCase{args: []string{"scan", "--model", manifest, "--format", "json", "--out", report}, code: exitOK})
	original := string(mustRead(t, report))

	stdout, _ := requireCLI(t, cliCase{args: []string{"scan", "--import", report, "--format", "json"}, code: exitOK})
	wantEqual(t, stdout, original)
	requireCLI(t, cliCase{args: []string{"scan", "--import", report, "--format", "sarif"}, code: exitOK, stdout: `"version": "2.1.0"`})
	stdout, _ = requireCLI(t, cliCase{args: []string{"scan", "--import", report}, code: exitOK, stdout: "tier mvp"})
	wantContains(t, stdout, "0 errors, 2 warnings")
	// Read at product: the same finding is an error.
	requireCLI(t, cliCase{args: []string{"scan", "--import", report, "--tier", "product", "--fail-on-error"}, code: exitFail, stdout: "2 errors, 0 warnings at tier product"})
	requireCLI(t, cliCase{args: []string{"scan", "--import", report, "--model", manifest, "--tier", "operational", "--fail-on-error"}, code: exitFail})
	requireCLI(t, cliCase{args: []string{"scan", "--import", report, "--tier", "gold"}, code: exitUsage})
	requireCLI(t, cliCase{args: []string{"scan", "--import", report, "pkg"}, code: exitUsage})
	requireBrokenReportsRejected(t, original)
}

// requireBrokenReportsRejected checks that a report with another schema, a
// missing body or no JSON at all is rejected with exit 2 and the reason.
func requireBrokenReportsRejected(t *testing.T, original string) {
	t.Helper()
	broken := filepath.Join(t.TempDir(), "broken.json")
	for name, content := range map[string]string{
		"schema":  strings.Replace(original, "engmod-health/1", "engmod-health/9", 1),
		"missing": `{"schema": "engmod-health/1"}`,
		"junk":    "not json",
	} {
		writeFileOrFail(t, broken, content)
		_, stderr := requireCLI(t, cliCase{args: []string{"scan", "--import", broken}, code: exitUsage})
		wantEqual(t, [2]any{name, strings.Contains(stderr, "invalid health report")}, [2]any{name, true})
	}
}
