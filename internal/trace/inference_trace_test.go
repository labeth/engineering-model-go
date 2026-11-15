// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package trace

// Code and verification inference feeding the trace matrix: which scanned
// symbols count as implementation, which as verification.

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestJavaScriptTestOnlyLinksDoNotImplement(t *testing.T) {
	for _, ext := range []string{"js", "mjs", "cjs"} {
		t.Run(ext, func(t *testing.T) { checkJavaScriptTestOnly(t, ext) })
	}
}

// checkJavaScriptTestOnly checks that a test file's links do not count as
// implementation of the requirements it links.
func checkJavaScriptTestOnly(t *testing.T, ext string) {
	f := newTraceFixture(t, map[string]string{
		"browser." + ext:      "// ENGMODEL-OWNER-UNIT: FU-WEB\n// TRLC-LINKS: REQ-WEB-001\nconst poll = () => 1;\n",
		"browser.test." + ext: "// ENGMODEL-OWNER-UNIT: FU-TEST\n// TRLC-LINKS: REQ-WEB-001, REQ-WEB-002\nfunction check() {}\n",
	})
	f.infer("REQ-WEB-001", "REQ-WEB-002")
	f.requireInferred(4, 1)
	requireEqual(t, f.checks[0].Status, "not-run")
	requireEqual(t, f.matrix().Summary.Implemented, 1)
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestPythonTestOnlyLinksDoNotImplement(t *testing.T) {
	f := newTraceFixture(t, map[string]string{
		"worker.py":      "# ENGMODEL-OWNER-UNIT: FU-PY\ntext = '''\n# ENGMODEL-OWNER-UNIT: FU-FAKE''' # real trailing comment\n# TRLC-LINKS: REQ-PY-001\ndef run(): return 1\n",
		"test_worker.py": "# ENGMODEL-OWNER-UNIT: FU-TEST\n# TRLC-LINKS: REQ-PY-001, REQ-PY-002\ndef test_run(): pass\ntext = '''\n# TRLC-LINKS: REQ-PY-003\n'''\n",
	})
	f.infer("REQ-PY-001", "REQ-PY-002", "REQ-PY-003")
	f.requireInferred(4, 1)
	requireEqual(t, f.checks[0].Status, "not-run")
	requireEqual(t, f.matrix().Summary.Implemented, 1)
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerilogExcludedSourcesAndVerification(t *testing.T) {
	f := newTraceFixture(t, map[string]string{
		"capture.v":    "// ENGMODEL-OWNER-UNIT: FU-CAPTURE\n// TRLC-LINKS: REQ-CAP-001\nmodule capture; endmodule\n",
		"tb_capture.v": "// ENGMODEL-OWNER-UNIT: FU-TEST\n// TRLC-LINKS: REQ-CAP-001, REQ-CAP-002\nmodule tb; endmodule\n",
		"unselected.v": "module missing_link; endmodule\n",
	})
	f.bundle.Architecture.InferenceHints.Exclude = []string{"unselected.v"}
	f.infer("REQ-CAP-001", "REQ-CAP-002")
	f.requireInferred(4, 1)
	requireFilesExist(t, f.code)
	requireEqual(t, []any{len(f.checks[0].Verifies), f.checks[0].Status}, []any{2, "not-run"})
	requireEqual(t, f.matrix().Summary.Implemented, 1)
	requireEqual(t, EffectiveCodeRoots(f.bundle, ""), []string{f.root})
}

// TRLC-LINKS: REQ-EMG-030
func TestVerilogFileRootRespectsNestedModel(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "engmod.yml"), []byte("schemaVersion: 3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(nested, "capture.v")
	if err := os.WriteFile(file, []byte("module capture; endmodule\n"), 0600); err != nil {
		t.Fatal(err)
	}
	items := []codescan.CodeItem{{Element: "capture", Source: "capture.v:1", AbsPath: file, Implements: []string{"REQ-CAP-001"}}}
	if got := ScopeCodeToModel(items, []string{file}, root); len(got) != 0 {
		t.Fatalf("file root bypassed nested model scope: %+v", got)
	}
	if got := ScopeCodeToModel(items, []string{file}, nested); len(got) != 1 {
		t.Fatalf("own model lost its file: %+v", got)
	}
}

// verificationFixture writes files under a temp root and runs code and
// verification inference plus the trace matrix over them.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func verificationFixture(t *testing.T, files map[string]string, reqIDs ...string) (Matrix, []codescan.VerificationCheck, []validate.Diagnostic) {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		testutil.WriteFile(t, filepath.Join(root, filepath.FromSlash(name)), content)
	}
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml")}
	reqs := model.RequirementsDocument{}
	for _, id := range reqIDs {
		reqs.Requirements = append(reqs.Requirements, model.Requirement{ID: id})
	}
	code, codeDiags := codescan.InferCodeItems(bundle, root)
	checks, verDiags := codescan.InferVerificationChecks(bundle, reqs, code, root)
	return buildTraceMatrix(bundle, Evidence{reqs, code, checks}), checks, append(codeDiags, verDiags...)
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func rowsByID(m Matrix) map[string]Row {
	out := map[string]Row{}
	for _, r := range m.Requirements {
		out[r.ID] = r
	}
	return out
}

// TRLC-LINKS: REQ-EMG-010
func helperLinkPaths(t *testing.T, diags []validate.Diagnostic) map[string]bool {
	t.Helper()
	helpers := diagsOf(diags, "verification.helper_link")
	requireEqual(t, len(diagsOfSeverity(helpers, validate.SeverityWarning)), len(helpers))
	out := map[string]bool{}
	for _, d := range helpers {
		out[d.Path] = true
	}
	return out
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerification_RustCfgTestHelpersDoNotVerifyOrImplement(t *testing.T) {
	m, _, diags := verificationFixture(t, map[string]string{"src/x.rs": rustCfgTestSource}, "REQ-SIM-001", "REQ-SIM-009")
	rows := rowsByID(m)
	// REQ-SIM-001 is implemented by now_ms only and verified by reads_clock.
	requireEqual(t, [][]int{{len(rows["REQ-SIM-001"].Code), len(rows["REQ-SIM-001"].Verifications)}, {rows["REQ-SIM-001"].Code[0].Line}}, [][]int{{1, 1}, {3}})
	// FakeClock::now neither implements nor verifies REQ-SIM-009.
	requireEqual(t, rowFacts(rows["REQ-SIM-009"]), []string{"orphan", "", "0", ""})
	requireEqual(t, len(rows["REQ-SIM-009"].Code), 0)
	requireEqual(t, helperLinkPaths(t, diags), map[string]bool{"src/x.rs:10": true})
	requireEqual(t, []int{m.Summary.Implemented, m.Summary.Tested, m.Summary.Passed}, []int{1, 1, 0})
	requireEqual(t, rows["REQ-SIM-001"].Result, "not-run") // no result artifact
}

// rustCfgTestSource is a Rust file with a production function and test-only
// helpers.
const rustCfgTestSource = "// ENGMODEL-OWNER-UNIT: FU-SIM\n" +
	"// TRLC-LINKS: REQ-SIM-001\n" +
	"pub fn now_ms() -> u64 { 0 }\n" +
	"\n" +
	"#[cfg(test)]\n" +
	"mod tests {\n" +
	"    struct FakeClock;\n" +
	"    impl FakeClock {\n" +
	"        // TRLC-LINKS: REQ-SIM-009\n" +
	"        fn now(&self) -> u64 { 1 }\n" +
	"    }\n" +
	"\n" +
	"    // TRLC-LINKS: REQ-SIM-001\n" +
	"    #[test]\n" +
	"    fn reads_clock() {}\n" +
	"}\n"

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerification_GoHelperInTestFileDoesNotVerify(t *testing.T) {
	src := "package sample\n\nimport \"testing\"\n\n" +
		"// TRLC-LINKS: REQ-GO-002\n" +
		"func failingWriter() error { return nil }\n\n" +
		"// TRLC-LINKS: REQ-GO-001\n" +
		"func TestWrites(t *testing.T) {}\n"
	m, _, diags := verificationFixture(t, map[string]string{"writer_test.go": src}, "REQ-GO-001", "REQ-GO-002")
	rows := rowsByID(m)
	if len(rows["REQ-GO-001"].Verifications) != 1 || len(rows["REQ-GO-002"].Verifications) != 0 {
		t.Fatalf("only TestWrites verifies: %+v", rows)
	}
	if len(rows["REQ-GO-002"].Code) != 0 {
		t.Fatalf("test helper must not implement: %+v", rows["REQ-GO-002"])
	}
	if got := helperLinkPaths(t, diags); !got["writer_test.go:6"] || len(got) != 1 {
		t.Fatalf("helper_link: %v", got)
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerification_PythonFixtureDoesNotVerify(t *testing.T) {
	src := "import pytest\n\n" +
		"# TRLC-LINKS: REQ-PY-002\n" +
		"@pytest.fixture\n" +
		"def fake_clock():\n" +
		"    return 1\n\n" +
		"# TRLC-LINKS: REQ-PY-001\n" +
		"def test_clock(fake_clock):\n" +
		"    assert fake_clock == 1\n"
	m, _, diags := verificationFixture(t, map[string]string{"test_clock.py": src}, "REQ-PY-001", "REQ-PY-002")
	rows := rowsByID(m)
	if len(rows["REQ-PY-001"].Verifications) != 1 || len(rows["REQ-PY-002"].Verifications) != 0 {
		t.Fatalf("only test_clock verifies: %+v", rows)
	}
	if got := helperLinkPaths(t, diags); !got["test_clock.py:5"] || len(got) != 1 {
		t.Fatalf("helper_link: %v", got)
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerification_ResultArtifactsDrivePassedAndRowResult(t *testing.T) {
	test := "package sample\n\nimport \"testing\"\n\n" +
		"// TRLC-LINKS: REQ-R-001, REQ-R-002, REQ-R-003\n" +
		"func TestAll(t *testing.T) {}\n"
	results := `{"results":[{"requirement":"REQ-R-001","status":"pass"},{"requirement":"REQ-R-002","status":"fail"}]}`
	m, _, _ := verificationFixture(t, map[string]string{"pkg/all_test.go": test, "test-results/all.json": results}, "REQ-R-001", "REQ-R-002", "REQ-R-003")
	rows := rowsByID(m)
	for id, want := range map[string]string{"REQ-R-001": "passed", "REQ-R-002": "failed", "REQ-R-003": "not-run"} {
		if rows[id].Result != want {
			t.Errorf("%s result = %q, want %q", id, rows[id].Result, want)
		}
	}
	if m.Summary.Tested != 3 || m.Summary.Passed != 1 {
		t.Fatalf("summary: %+v", m.Summary)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerification_YamlTestArtifactsKeepFileLevelLinks(t *testing.T) {
	m, _, diags := verificationFixture(t, map[string]string{"tests/e2e/flow.yaml": "# TRLC-LINKS: REQ-Y-001\nsteps: []\n"}, "REQ-Y-001")
	if r := rowsByID(m)["REQ-Y-001"]; len(r.Verifications) != 1 {
		t.Fatalf("yaml test artifacts keep file-level verification: %+v", r)
	}
	if len(helperLinkPaths(t, diags)) != 0 {
		t.Fatalf("no helper_link for undetected languages: %+v", diags)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestVerification_PythonConftestFixturesDoNotVerify(t *testing.T) {
	files := map[string]string{
		"tests/conftest.py":   "# TRLC-LINKS: REQ-PY-002\ndef clock():\n    return 1\n",
		"tests/test_clock.py": "# TRLC-LINKS: REQ-PY-001\ndef test_clock():\n    pass\n",
	}
	m, _, diags := verificationFixture(t, files, "REQ-PY-001", "REQ-PY-002")
	rows := rowsByID(m)
	if len(rows["REQ-PY-001"].Verifications) != 1 || len(rows["REQ-PY-002"].Verifications) != 0 || len(rows["REQ-PY-002"].Code) != 0 {
		t.Fatalf("conftest fixture must neither verify nor implement: %+v", rows)
	}
	if got := helperLinkPaths(t, diags); !got["tests/conftest.py:2"] || len(got) != 1 {
		t.Fatalf("helper_link: %v", got)
	}
}

// In a Go test file an unmarked helper function or method raises no
// code.missing_trlc_link, an unmarked Test function does, and a marked helper
// raises only the verification.helper_link warning.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func TestVerification_GoUnmarkedTestHelpersNeedNoLinks(t *testing.T) {
	src := "package sample\n\nimport \"testing\"\n\n" +
		"type fakeClock struct{}\n\n" +
		"func (fakeClock) Now() int { return 1 }\n\n" +
		"func newFake() fakeClock { return fakeClock{} }\n\n" +
		"// TRLC-LINKS: REQ-GO-001\n" +
		"func markedHelper() {}\n\n" +
		"func TestUnlinked(t *testing.T) {}\n\n" +
		"// TRLC-LINKS: REQ-GO-001\n" +
		"func TestLinked(t *testing.T) {}\n"
	_, _, diags := verificationFixture(t, map[string]string{"clock_test.go": src}, "REQ-GO-001")
	missing := map[string]bool{}
	for _, d := range diags {
		if d.Code == "code.missing_trlc_link" {
			missing[d.Path] = true
		}
	}
	if len(missing) != 1 || !missing["clock_test.go:14"] {
		t.Fatalf("missing_trlc_link = %v, want only the unlinked Test at clock_test.go:14", missing)
	}
	if got := helperLinkPaths(t, diags); !got["clock_test.go:12"] || len(got) != 1 {
		t.Fatalf("helper_link = %v, want only clock_test.go:12", got)
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerificationCodeUsesAbsoluteSourceContext(t *testing.T) {
	requireEqual(t, codescan.IsVerificationCodeItem(codescan.CodeItem{Source: "helper.go:3", AbsPath: "/workspace/tests/helper.go"}), true)
	requireEqual(t, codescan.IsVerificationCodeItem(codescan.CodeItem{Source: "contest.go:3", AbsPath: "/workspace/src/contest.go"}), false)
	f := newTraceFixture(t, map[string]string{"tests/helper.go": "// ENGMODEL-OWNER-UNIT: FU-A\npackage fixture\n// TRLC-LINKS: REQ-TEST-001\nfunc helper() {}\n"})
	f.infer("REQ-TEST-001")
	requireEqual(t, []int{len(f.codeDiag), len(f.code)}, []int{0, 2})
	for _, item := range f.code {
		requireEqual(t, []any{item.AbsPath, codescan.IsVerificationCodeItem(item)}, []any{filepath.Join(f.root, "tests", "helper.go"), true})
	}
	requireEqual(t, f.matrix().Summary.Implemented, 0)
}

// A requirement linked only from a Dockerfile comment or a Markdown HTML
// comment is implemented in the trace matrix, by the file's CODE identifier,
// and a Dockerfile under tests/ still implements rather than verifies; a
// Dockerfile without markers adds no missing-link error.
// Mutation: not scanning artifact files leaves REQ-IMG-001 and REQ-DOC-001 orphan.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
func TestArtifactTraceMarkersImplementRequirements(t *testing.T) {
	m, _, diags := verificationFixture(t, map[string]string{
		"Dockerfile":           "# ENGMODEL-OWNER-UNIT: FU-IMAGE\n# TRLC-LINKS: REQ-IMG-001\nFROM scratch\n",
		"tests/e2e/Dockerfile": "# TRLC-LINKS: REQ-IMG-002\nFROM scratch\n",
		"README.md":            "# Run\n\n<!-- TRLC-LINKS: REQ-DOC-001 -->\nRun it.\n",
		"deploy/Containerfile": "FROM scratch\nUSER 65532\n",
		"src/app.go":           "package app\n\n// TRLC-LINKS: REQ-IMG-001\nfunc Run() {}\n",
	}, "REQ-IMG-001", "REQ-IMG-002", "REQ-DOC-001")
	rows := rowsByID(m)
	requireEqual(t, codeRefsOf(rows["REQ-IMG-001"]), "CODE-DOCKERFILE@Dockerfile CODE-SRC-APP-GO--RUN@src/app.go")
	requireEqual(t, rows["REQ-IMG-001"].Status, "implemented")
	requireEqual(t, codeRefsOf(rows["REQ-IMG-002"]), "CODE-TESTS-E2E-DOCKERFILE@tests/e2e/Dockerfile")
	requireEqual(t, len(rows["REQ-IMG-002"].Verifications), 0)
	requireEqual(t, codeRefsOf(rows["REQ-DOC-001"]), "CODE-README-MD@README.md")
	requireEqual(t, len(diagsOfSeverity(diags, validate.SeverityError)), 0)
	requireEqual(t, len(diagsOf(diags, "code.trace_unattached")), 0)
}

// codeRefsOf lists the code of a row as symbol@path.
func codeRefsOf(r Row) string {
	var out []string
	for _, c := range r.Code {
		out = append(out, c.Symbol+"@"+c.Path)
	}
	return strings.Join(out, " ")
}

// diagsOfSeverity returns the diagnostics of a severity.
func diagsOfSeverity(diags []validate.Diagnostic, severity validate.Severity) []validate.Diagnostic {
	return slices.DeleteFunc(slices.Clone(diags), func(d validate.Diagnostic) bool { return d.Severity != severity })
}

// A Go test whose first statements skip unless an environment variable is set
// (or under testing.Short) is gated. A requirement linked only from gated
// tests is gatedOnly in the summary and its row is gated with the variable
// name; a requirement that an ungated test also links, in the same file or in
// another test file, is not. All stay test linked.
// Mutation: not marking os.Getenv-gated tests leaves gatedOnly at 0; ignoring
// ungated tests of other files marks REQ-G-004 gated.
// TRLC-LINKS: REQ-EMG-088
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-ALLOCATION-TRACE
func TestGatedOnlyRequirementsReportedSeparately(t *testing.T) {
	f := newTraceFixture(t, map[string]string{
		"sim_test.go":   gatedSimSource,
		"other_test.go": "package sim\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-G-004\nfunc TestOther(t *testing.T) {}\n",
	})
	f.infer("REQ-G-001", "REQ-G-002", "REQ-G-003", "REQ-G-004")
	matrix := f.matrix()
	requireEqual(t, matrix.Summary.TestLinked, 4) // gated tests stay test linked
	requireEqual(t, matrix.Summary.GatedOnlyRequirements, []string{"REQ-G-001", "REQ-G-003"})
	rows := rowsByID(matrix)
	requireEqual(t, []string{gateFacts(rows["REQ-G-001"]), gateFacts(rows["REQ-G-002"]), gateFacts(rows["REQ-G-003"]), gateFacts(rows["REQ-G-004"])},
		[]string{"true:SIMFAM_COMPOSE_TEST", "false:", "true:testing.Short", "false:"})
}

// gateFacts renders whether a row is gated and by which variables.
func gateFacts(r Row) string {
	return fmt.Sprintf("%v:%s", r.Gated, strings.Join(r.Gates, ","))
}

// gatedSimSource is a Go test file with gated and ungated tests.
const gatedSimSource = `package sim

import (
	"fmt"
	"os"
	"testing"
)

// TRLC-LINKS: REQ-G-001, REQ-G-002, REQ-G-004
func TestComposeUp(t *testing.T) {
	t.Helper()
	if os.Getenv("SIMFAM_COMPOSE_TEST") != "1" {
		t.Skip("set SIMFAM_COMPOSE_TEST=1")
	}
}

// TRLC-LINKS: REQ-G-002
func TestUnit(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("math")
	}
}

// TRLC-LINKS: REQ-G-003
func TestLong(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
}
`

// A requirement whose only linked tests are benchmarks (Go Benchmark*, Rust
// #[bench]) has a benchmarkOnly row; when its verification criteria name a
// number it is also verification.benchmark_only (warning): a benchmark
// asserts nothing unless something checks its result. A requirement that a
// Test* or a Fuzz* also links, in the same or another test file, is
// neither; one whose criteria name no number is benchmarkOnly without the
// warning.
// Mutation: counting Benchmark* as a test (no flag), ignoring the number
// condition or the other tests of the requirement fails this test.
// TRLC-LINKS: REQ-EMG-121
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-ALLOCATION-TRACE
func TestBenchmarkOnlyVerification(t *testing.T) {
	f := newTraceFixture(t, map[string]string{
		"sim_test.go":   benchmarkGoSource,
		"lib.rs":        "// TRLC-LINKS: REQ-B-004, REQ-B-006\n#[bench]\nfn bench_parse(b: &mut Bencher) {}\n",
		"other_test.go": "package sim\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-B-006\nfunc TestParse(t *testing.T) {}\n",
	})
	f.reqs = model.RequirementsDocument{Requirements: []model.Requirement{
		{ID: "REQ-B-001", VerificationCriteria: "One sentence is encoded in under 50 microseconds."},
		{ID: "REQ-B-002", VerificationCriteria: "One sentence is encoded in under 50 microseconds."},
		{ID: "REQ-B-003", VerificationCriteria: "The encoder output is well formed."},
		{ID: "REQ-B-004", VerificationCriteria: "Parsing takes at most 2 ms."},
		{ID: "REQ-B-005", VerificationCriteria: "Decoding takes at most 2 ms."},
		{ID: "REQ-B-006", VerificationCriteria: "Parsing takes at most 2 ms."},
	}}
	f.inferred()
	benchmarkOnly := map[string]bool{}
	for id, row := range rowsByID(f.matrix()) {
		benchmarkOnly[id] = row.BenchmarkOnly && len(row.Verifications) > 0
	}
	requireEqual(t, benchmarkOnly, map[string]bool{"REQ-B-001": true, "REQ-B-002": false, "REQ-B-003": true, "REQ-B-004": true, "REQ-B-005": false, "REQ-B-006": false})
	warnings := diagsOf(f.diagnostics(), "verification.benchmark_only")
	requireEqual(t, severitiesByCode(warnings), map[string]validate.Severity{"verification.benchmark_only": validate.SeverityWarning})
	requireEqual(t, pathsOf(warnings, "verification.benchmark_only"), []string{"requirements[0]", "requirements[3]"})
}

// benchmarkGoSource is a Go test file whose benchmarks and tests link
// requirements.
const benchmarkGoSource = `package sim

import "testing"

// TRLC-LINKS: REQ-B-001, REQ-B-002, REQ-B-003
func BenchmarkEncode(b *testing.B) {}

// TRLC-LINKS: REQ-B-002
func TestEncode(t *testing.T) {}

// TRLC-LINKS: REQ-B-005
func BenchmarkDecode(b *testing.B) {}

// TRLC-LINKS: REQ-B-005
func FuzzDecode(f *testing.F) {}
`
