// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package scan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

const architecture = `  functionalGroups:
    - id: FG-A
      name: Group
  functionalUnits:
    - id: FU-A
      name: Unit A
      group: FG-A
    - id: FU-B
      name: Unit B
      group: FG-A
`

// bigGo is a Go file with a function of cyclomatic complexity 17 (1 + 16 ifs):
// a warning at the default limits (10 / 20) and a high finding at the strict
// limits of product (8 / 15).
func bigGo(owner string) string {
	var b strings.Builder
	b.WriteString("// ENGMODEL-OWNER-UNIT: " + owner + "\npackage pkg\n\n// TRLC-LINKS: REQ-X\nfunc Big(a int) int {\n")
	for i := 0; i < 16; i++ {
		fmt.Fprintf(&b, "\tif a == %d { return %d }\n", i, i)
	}
	b.WriteString("\treturn 0\n}\n")
	return b.String()
}

// fixture writes a model module with a lifecycle and health block (extra is
// inserted before the documents) and some code, and returns the manifest path.
func fixture(t *testing.T, extra string) (dir, manifest string) {
	t.Helper()
	dir = t.TempDir()
	testutil.WriteV2ModuleFiles(t, dir, "example.com/hs@v0", "v0.1.0", "hs", extra, architecture)
	testutil.WriteFile(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 3\nlintRun: {}\nrequirements:\n  - id: REQ-X\n    text: The unit shall decide.\n")
	// Accepted decisions that the waivers, the baseline and the policy decision
	// of the tests refer to: a waiver (ADR-EMG-051, ADR-EMG-052), a baseline or a
	// policy decision whose decision is not accepted does nothing; ADR-DRAFT is
	// proposed.
	testutil.WriteFile(t, filepath.Join(dir, "model", "decisions.yml"), "schemaVersion: 3\ndecisions:\n"+
		"  - {id: ADR-W, title: Waive, status: accepted, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n"+
		"  - {id: ADR-SVC, title: Waive svc, status: accepted, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n"+
		"  - {id: ADR-DEBT, title: Carry debt, status: accepted, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n"+
		"  - {id: ADR-POLICY, title: Weaken policy, status: accepted, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n"+
		"  - {id: ADR-DRAFT, title: Not yet, status: proposed, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n")
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "big.go"), bigGo("FU-A"))
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "clean.go"), "// ENGMODEL-OWNER-UNIT: FU-B\npackage pkg\n\n// TRLC-LINKS: REQ-X\nfunc Clean() int { return 1 }\n")
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "big_test.go"), "// ENGMODEL-OWNER-UNIT: FU-A\npackage pkg\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-X\nfunc TestBig(t *testing.T) { Big(1) }\n")
	testutil.WriteFile(t, filepath.Join(dir, "generated", "gen.go"), bigGo("FU-A"))
	testutil.WriteFile(t, filepath.Join(dir, "skipme", "x.go"), bigGo("FU-A"))
	testutil.WriteFile(t, filepath.Join(dir, "vendor", "v", "v.go"), bigGo("FU-A"))
	testutil.WriteFile(t, filepath.Join(dir, "docs", "readme.md"), "# not source\n")
	return dir, filepath.Join(dir, "engmod.yml")
}

// findingsOf returns the codes found in a function of a file report.
func findingsOf(r health.Report, path, function string) []health.Finding {
	for _, f := range r.Files {
		if f.Path == path {
			return functionFindingsIn(f, function)
		}
	}
	return nil
}

// functionFindingsIn returns the findings of a function of a file report.
func functionFindingsIn(f health.FileReport, function string) []health.Finding {
	for _, fn := range f.Functions {
		if fn.Name == function {
			return fn.Findings
		}
	}
	return nil
}

// A scan of a model at tier product uses the strict limits, excludes the
// default excludes and health.exclude, lists the files sorted, attributes
// files to the functional unit of their owner marker, reaches the requirement
// through TRLC-LINKS and reports the findings as errors (row WWWEE).
//
// Mutation: using the default limits at product (the finding would be warn),
// scanning generated/ or vendor/, or reading no markers fails this test.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-152, REQ-EMG-153, REQ-EMG-155, REQ-EMG-158
func TestScanProductTierEndToEnd(t *testing.T) {
	// The floor of 8.0 is the score of big.go: the floor is tested in floor_test.go.
	dir, manifest := fixture(t, "lifecycle:\n  tier: product\nhealth:\n  exclude: [skipme]\n  floor: 8.0\n")
	r := mustScan(t, Options{ManifestPath: manifest})
	var paths []string
	for _, f := range r.Files {
		paths = append(paths, f.Path)
	}
	mustEqual(t, strings.Join(paths, ","), "pkg/big.go,pkg/big_test.go,pkg/clean.go") // only the three source files of pkg
	wantEqual(t, [3]any{r.Tier, r.Levels[health.CodeComplexMethod], r.Engine.Name}, [3]any{"product", health.LevelError, "treesitter"})
	requireBigFinding(t, r)
	requireProductRollups(t, r)
	data := mustEncode(t, r)
	wantEqual(t, health.ValidateReportJSON(data), nil)
	wantNone(t, string(data), dir)
}

// requireBigFinding checks the finding, the score and the counts of big.go at
// product.
func requireBigFinding(t *testing.T, r health.Report) {
	t.Helper()
	fs := findingsOf(r, "pkg/big.go", "Big")
	mustEqual(t, len(fs), 1)
	wantEqual(t, [3]any{fs[0].Code, fs[0].Severity, fs[0].Value}, [3]any{health.CodeComplexMethod, health.SeverityHigh, 17})
	// 17 >= 15 (high at product): the deduction is 1.0 x 2 = 2.0, score 8.0.
	wantEqual(t, [2]any{r.Files[0].Score, r.Files[0].Band}, [2]any{8.0, health.BandGreen})
	wantEqual(t, [2]int{r.Summary.Errors, r.Summary.Warnings}, [2]int{1, 0})
}

// requireProductRollups checks the roll-ups of the product fixture: the files,
// the tests, the units and the requirement.
func requireProductRollups(t *testing.T, r health.Report) {
	t.Helper()
	wantEqual(t, r.Rollups.Project.Files, 2)
	mustEqual(t, r.Rollups.Tests != nil, true)
	wantEqual(t, r.Rollups.Tests.Files, 1)
	mustEqual(t, len(r.Rollups.Units), 2)
	wantEqual(t, [3]any{r.Rollups.Units[0].Unit, r.Rollups.Units[0].Files, r.Rollups.Units[1].Unit}, [3]any{"FU-A", 1, "FU-B"})
	// REQ-X is linked on Big and Clean (production); the test's link does not count.
	mustEqual(t, len(r.Rollups.Requirements), 1)
	q := r.Rollups.Requirements[0]
	wantEqual(t, [3]any{q.Requirement, q.Functions, q.Findings}, [3]any{"REQ-X", 2, 1})
}

// The tier flag replaces the manifest tier: at mvp the default limits apply, so
// the same function is only a warn finding and a warning; a manifest threshold
// still replaces the limit of its code at every tier.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
func TestScanTierOverrideAndManifestThresholds(t *testing.T) {
	_, manifest := fixture(t, "lifecycle:\n  tier: product\n")
	r := mustScan(t, Options{ManifestPath: manifest, Tier: "MVP"})
	fs := findingsOf(r, "pkg/big.go", "Big")
	mustEqual(t, len(fs), 1)
	wantEqual(t, [4]any{r.Tier, fs[0].Severity, r.Summary.Errors, r.Summary.Warnings}, [4]any{"mvp", health.SeverityWarn, 0, 4})

	_, manifest = fixture(t, "lifecycle:\n  tier: product\nhealth:\n  thresholds:\n    health.complex_method: {warn: 14, high: 15}\n  overrides:\n    - path: pkg/big.go\n      thresholds:\n        health.complex_method: {warn: 20, high: 30}\n")
	r = mustScan(t, Options{ManifestPath: manifest})
	wantEqual(t, len(findingsOf(r, "pkg/big.go", "Big")), 0) // the override lifts the limit to 20/30
}

// A model without a lifecycle tier is scanned with the default limits and every
// finding is a warning; the health block is optional. This is kept library
// behaviour for tools that embed the scanner (ADR-EMG-066 item 3): the
// validator reports such a model as lifecycle.missing, so the test strips the
// complete switches from the fixture's manifest.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-160
func TestScanWithoutLifecycleIsWarnings(t *testing.T) {
	_, manifest := fixture(t, "")
	testutil.WithoutSwitches(t, manifest)
	r := mustScan(t, Options{ManifestPath: manifest})
	wantEqual(t, [4]any{r.Tier, r.Bundles == nil, r.Summary.Errors, r.Summary.Warnings}, [4]any{"", true, 0, 4})
	for _, f := range r.Files {
		wantEqual(t, strings.HasPrefix(f.Path, "generated/") || strings.HasPrefix(f.Path, "vendor/"), false) // default excludes
	}
	wantEqual(t, len(r.Files), 4) // pkg/* (3) and skipme/x.go
}

// A waiver with an accepted decision moves its finding to the waived list and
// lifts the score; a waiver that matches nothing becomes a diagnostic.
//
// TRLC-LINKS: REQ-EMG-154
func TestScanAppliesWaivers(t *testing.T) {
	extra := "lifecycle:\n  tier: product\nhealth:\n  floor: 8.0\n  waivers:\n" +
		"    - {code: health.complex_method, path: pkg/big.go, function: Big, decision: ADR-W}\n" +
		"    - {code: health.large_method, path: pkg/big.go, function: Big, decision: ADR-W}\n"
	_, manifest := fixture(t, extra)
	r := mustScan(t, Options{ManifestPath: manifest})
	mustEqual(t, len(r.Waived), 1)
	wantEqual(t, [2]string{r.Waived[0].Decision, r.Waived[0].Function}, [2]string{"ADR-W", "Big"})
	wantEqual(t, [2]int{int(r.Files[0].Score), len(findingsOf(r, "pkg/big.go", "Big"))}, [2]int{10, 0})
	// The unused large_method waiver is an error at product.
	mustEqual(t, len(r.Diagnostics), 1)
	wantEqual(t, [2]string{r.Diagnostics[0].Code, r.Diagnostics[0].Level}, [2]string{"health.waiver_unused", health.LevelError})

	// A scan restricted to another file does not call the waivers unused.
	r = mustScan(t, Options{ManifestPath: manifest, Paths: []string{filepath.Join(filepath.Dir(manifest), "pkg", "clean.go")}})
	mustEqual(t, len(r.Files), 1)
	wantEqual(t, [2]any{r.Files[0].Path, len(r.Diagnostics)}, [2]any{"pkg/clean.go", 0})
}

// Paths limit the scan to files and directories inside the project root; a
// path outside, a missing path, an unsupported file, a bad tier and a model
// that does not load are setup errors.
//
// TRLC-LINKS: REQ-EMG-158
func TestScanSetupErrors(t *testing.T) {
	dir, manifest := fixture(t, "")
	r, err := Scan(Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}})
	if err != nil || len(r.Files) != 3 {
		t.Fatalf("directory scope: %d files, err %v", len(r.Files), err)
	}
	bad := []struct {
		name string
		opts Options
	}{
		{"outside", Options{ManifestPath: manifest, Paths: []string{os.TempDir()}}},
		{"missing path", Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "nope")}}},
		{"unsupported file", Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "docs", "readme.md")}}},
		{"tier", Options{ManifestPath: manifest, Tier: "gold"}},
		{"no model", Options{ManifestPath: filepath.Join(dir, "missing.yml")}},
	}
	for _, c := range bad {
		if _, err := Scan(c.opts); !errors.Is(err, ErrSetup) {
			t.Errorf("%s: err = %v, want ErrSetup", c.name, err)
		}
	}
}

// Two scans of an unchanged tree give byte-identical reports.
//
// TRLC-LINKS: REQ-EMG-159
func TestScanIsDeterministic(t *testing.T) {
	_, manifest := fixture(t, "lifecycle:\n  tier: operational\n")
	var first []byte
	for i := 0; i < 3; i++ {
		r, err := Scan(Options{ManifestPath: manifest})
		if err != nil {
			t.Fatal(err)
		}
		data, err := health.Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = data
		} else if string(data) != string(first) {
			t.Fatalf("run %d differs", i)
		}
	}
}

// failingEngine fails on files whose path contains "big".
type failingEngine struct{ health.Engine }

// Analyze fails for one file and delegates for the others.
func (e failingEngine) Analyze(inputs []health.FileInput, cfg health.Config) ([]health.FileResult, error) {
	if strings.Contains(inputs[0].Path, "big.go") {
		return nil, errors.New("boom")
	}
	return e.Engine.Analyze(inputs, cfg)
}

// A file the engine cannot analyse is a diagnostic and never a clean file; the
// other files are still scanned. A file above the size limit is skipped the
// same way.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-158
func TestScanReportsFilesItCouldNotAnalyse(t *testing.T) {
	dir, manifest := fixture(t, "")
	huge := "package pkg\n\n// " + strings.Repeat("x", MaxFileBytes) + "\nfunc Huge() {}\n"
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "huge.go"), huge)
	r := mustScan(t, Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, Engine: failingEngine{Engine: realEngine()}})
	codes := map[string]string{}
	for _, d := range r.Diagnostics {
		codes[d.Path] = d.Code
		wantEqual(t, [2]any{d.Level, strings.Contains(d.Message, dir)}, [2]any{health.LevelWarning, false})
	}
	wantEqual(t, [3]any{codes["pkg/big.go"], codes["pkg/huge.go"], len(r.Files)}, [3]any{CodeFileFailed, CodeFileSkipped, 2})
}

// A waiver whose decision is proposed, not accepted, waives nothing: the
// finding stays in the report and the waiver is not counted as used.
// Mutation: dropping AcceptedHealthWaivers from the scan lets a proposed
// decision hide a finding.
//
// TRLC-LINKS: REQ-EMG-154
func TestScanIgnoresWaiverWithoutAcceptedDecision(t *testing.T) {
	extra := "lifecycle:\n  tier: product\nhealth:\n  waivers:\n" +
		"    - {code: health.complex_method, path: pkg/big.go, function: Big, decision: ADR-P}\n"
	dir, manifest := fixture(t, extra)
	testutil.WriteFile(t, filepath.Join(dir, "model", "decisions.yml"), "schemaVersion: 3\ndecisions:\n"+
		"  - {id: ADR-P, title: Not yet, status: proposed, date: 2026-11-12, context: c, decision: d, consequences: [c]}\n")
	r, err := Scan(Options{ManifestPath: manifest})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Waived) != 0 {
		t.Fatalf("a proposed decision waived %+v", r.Waived)
	}
	if len(findingsOf(r, "pkg/big.go", "Big")) == 0 {
		t.Error("the finding of Big disappeared although its waiver has no accepted decision")
	}
}
