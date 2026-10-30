// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package trace

// Code and verification inference feeding the trace matrix: which scanned
// symbols count as implementation, which as verification.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestJavaScriptTestOnlyLinksDoNotImplement(t *testing.T) {
	for _, ext := range []string{"js", "mjs", "cjs"} {
		t.Run(ext, func(t *testing.T) {
			root := t.TempDir()
			impl := "browser." + ext
			test := "browser.test." + ext
			for name, text := range map[string]string{
				impl: "// ENGMODEL-OWNER-UNIT: FU-WEB\n// TRLC-LINKS: REQ-WEB-001\nconst poll = () => 1;\n",
				test: "// ENGMODEL-OWNER-UNIT: FU-TEST\n// TRLC-LINKS: REQ-WEB-001, REQ-WEB-002\nfunction check() {}\n",
			} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Architecture: model.ArchitectureDocument{InferenceHints: model.InferenceHints{CodeSources: []string{impl, test}}}}
			code, diags := codescan.InferCodeItems(bundle, "")
			if len(diags) != 0 || len(code) != 4 {
				t.Fatalf("code inference: %+v %+v", code, diags)
			}
			reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-WEB-001"}, {ID: "REQ-WEB-002"}}}
			checks, diags := codescan.InferVerificationChecks(bundle, reqs, code, "")
			if len(diags) != 0 || len(checks) != 1 || checks[0].Status != "not-run" {
				t.Fatalf("checks: %+v %+v", checks, diags)
			}
			matrix := buildTraceMatrix(bundle, reqs, code, checks, nil)
			if matrix.Summary.Implemented != 1 || matrix.Summary.Verified != 2 {
				t.Fatalf("test-only links counted as implementation: %+v", matrix)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestPythonTestOnlyLinksDoNotImplement(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"worker.py":      "# ENGMODEL-OWNER-UNIT: FU-PY\ntext = '''\n# ENGMODEL-OWNER-UNIT: FU-FAKE''' # real trailing comment\n# TRLC-LINKS: REQ-PY-001\ndef run(): return 1\n",
		"test_worker.py": "# ENGMODEL-OWNER-UNIT: FU-TEST\n# TRLC-LINKS: REQ-PY-001, REQ-PY-002\ndef test_run(): pass\ntext = '''\n# TRLC-LINKS: REQ-PY-003\n'''\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Architecture: model.ArchitectureDocument{InferenceHints: model.InferenceHints{CodeSources: []string{"worker.py", "test_worker.py"}}}}
	code, diags := codescan.InferCodeItems(bundle, "")
	if len(diags) != 0 || len(code) != 4 {
		t.Fatalf("code: %+v diagnostics: %+v", code, diags)
	}
	requirements := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-PY-001"}, {ID: "REQ-PY-002"}, {ID: "REQ-PY-003"}}}
	checks, diags := codescan.InferVerificationChecks(bundle, requirements, code, "")
	if len(diags) != 0 || len(checks) != 1 || checks[0].Status != "not-run" {
		t.Fatalf("checks: %+v %+v", checks, diags)
	}
	matrix := buildTraceMatrix(bundle, requirements, code, checks, nil)
	if matrix.Summary.Implemented != 1 || matrix.Summary.Verified != 2 {
		t.Fatalf("test-only code credited: %+v", matrix)
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerilogExcludedSourcesAndVerification(t *testing.T) {
	root := t.TempDir()
	fixtures := map[string]string{
		"capture.v":    "// ENGMODEL-OWNER-UNIT: FU-CAPTURE\n// TRLC-LINKS: REQ-CAP-001\nmodule capture; endmodule\n",
		"tb_capture.v": "// ENGMODEL-OWNER-UNIT: FU-TEST\n// TRLC-LINKS: REQ-CAP-001, REQ-CAP-002\nmodule tb; endmodule\n",
		"unselected.v": "module missing_link; endmodule\n",
	}
	for name, text := range fixtures {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Architecture: model.ArchitectureDocument{InferenceHints: model.InferenceHints{Exclude: []string{"unselected.v"}}}}
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-CAP-001"}, {ID: "REQ-CAP-002"}}}
	code, diags := codescan.InferCodeItems(bundle, "")
	if len(diags) != 0 {
		t.Fatalf("unexpected source diagnostics: %+v", diags)
	}
	if len(code) != 4 {
		t.Fatalf("expected two files and modules: %+v", code)
	}
	for _, item := range code {
		if item.Kind == "symbol" {
			if _, err := os.Stat(item.AbsPath); err != nil {
				t.Fatalf("bad absolute source path: %+v", item)
			}
		}
	}
	checks, diags := codescan.InferVerificationChecks(bundle, reqs, code, "")
	if len(diags) != 0 || len(checks) != 1 || len(checks[0].Verifies) != 2 || checks[0].Status != "not-run" {
		t.Fatalf("bad verification inference: %+v %+v", checks, diags)
	}
	matrix := buildTraceMatrix(bundle, reqs, code, checks, nil)
	if matrix.Summary.Implemented != 1 || matrix.Summary.Verified != 2 {
		t.Fatalf("test-only links must not imply implementation: %+v", matrix)
	}
	roots := EffectiveCodeRoots(bundle, "")
	if len(roots) != 1 || roots[0] != root {
		t.Fatalf("module root not used for scoping: %+v", roots)
	}
}

// TRLC-LINKS: REQ-EMG-030
func TestVerilogFileRootRespectsNestedModel(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "engmod.yml"), []byte("schemaVersion: 2\n"), 0600); err != nil {
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
	return buildTraceMatrix(bundle, reqs, code, checks, nil), checks, append(codeDiags, verDiags...)
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
	out := map[string]bool{}
	for _, d := range diags {
		if d.Code == "verification.helper_link" {
			if d.Severity != validate.SeverityWarning {
				t.Fatalf("helper_link must be a warning: %+v", d)
			}
			out[d.Path] = true
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerification_RustCfgTestHelpersDoNotVerifyOrImplement(t *testing.T) {
	src := "// ENGMODEL-OWNER-UNIT: FU-SIM\n" +
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
	m, checks, diags := verificationFixture(t, map[string]string{"src/x.rs": src}, "REQ-SIM-001", "REQ-SIM-009")
	rows := rowsByID(m)
	if r := rows["REQ-SIM-001"]; len(r.Code) != 1 || r.Code[0].Line != 3 || len(r.Verifications) != 1 {
		t.Fatalf("REQ-SIM-001 must be implemented by now_ms only and verified by reads_clock: %+v (checks %+v)", r, checks)
	}
	if r := rows["REQ-SIM-009"]; len(r.Code) != 0 || len(r.Verifications) != 0 || r.Status != "orphan" {
		t.Fatalf("FakeClock::now must neither implement nor verify REQ-SIM-009: %+v", r)
	}
	if got := helperLinkPaths(t, diags); !got["src/x.rs:10"] || len(got) != 1 {
		t.Fatalf("expected one helper_link at src/x.rs:10, got %v (all %+v)", got, diags)
	}
	if m.Summary.Implemented != 1 || m.Summary.Tested != 1 || m.Summary.Verified != 1 || m.Summary.Passed != 0 {
		t.Fatalf("summary: %+v", m.Summary)
	}
	if r := rows["REQ-SIM-001"]; r.Result != "not-run" {
		t.Fatalf("no result artifact: want not-run, got %q", r.Result)
	}
}

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
	if m.Summary.Tested != 3 || m.Summary.Verified != 3 || m.Summary.Passed != 1 {
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
	if !codescan.IsVerificationCodeItem(codescan.CodeItem{Source: "helper.go:3", AbsPath: "/workspace/tests/helper.go"}) {
		t.Fatal("test source root context lost")
	}
	if codescan.IsVerificationCodeItem(codescan.CodeItem{Source: "contest.go:3", AbsPath: "/workspace/src/contest.go"}) {
		t.Fatal("production path classified as test")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "tests")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "helper.go")
	if err := os.WriteFile(path, []byte("// ENGMODEL-OWNER-UNIT: FU-A\npackage fixture\n// TRLC-LINKS: REQ-TEST-001\nfunc helper() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Architecture: model.ArchitectureDocument{InferenceHints: model.InferenceHints{CodeSources: []string{path}}}}
	code, diags := codescan.InferCodeItems(bundle, "")
	if len(diags) != 0 || len(code) != 2 {
		t.Fatalf("inference failed: %+v %+v", code, diags)
	}
	for _, item := range code {
		if item.AbsPath != path || !codescan.IsVerificationCodeItem(item) {
			t.Fatalf("test file root context lost: %+v", item)
		}
	}
	matrix := buildTraceMatrix(bundle, model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-TEST-001"}}}, code, nil, nil)
	if matrix.Summary.Implemented != 0 {
		t.Fatalf("test-root helper credited as production: %+v", matrix)
	}
}
