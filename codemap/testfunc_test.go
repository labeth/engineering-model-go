// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"os"
	"path/filepath"
	"testing"
)

// scanFixture writes one file into a fresh root and returns its symbols by name.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func scanFixture(t *testing.T, name, content string) map[string]Symbol {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	symbols, diags, err := Scan(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	for _, d := range diags {
		if d.Severity == "error" {
			t.Fatalf("unexpected error diagnostic: %+v", d)
		}
	}
	out := map[string]Symbol{}
	for _, s := range symbols {
		out[s.TraceID] = s
	}
	return out
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func TestScan_MarksRustTestFunctionsAndCfgTestScope(t *testing.T) {
	src := `// TRLC-LINKS: REQ-SIM-001
pub fn now_ms() -> u64 { 0 }

#[cfg(test)]
mod tests {
    // TRLC-LINKS: REQ-SIM-009
    struct FakeClock;
    impl FakeClock {
        // TRLC-LINKS: REQ-SIM-009
        fn now(&self) -> u64 { 1 }
    }

    // TRLC-LINKS: REQ-SIM-001
    #[test]
    fn reads_clock() {}

    #[tokio::test(flavor = "multi_thread")]
    // TRLC-LINKS: REQ-SIM-002
    async fn async_case() {}

    // TRLC-LINKS: REQ-SIM-003
    #[rstest]
    #[ignore]
    fn param_case() {}

    // TRLC-LINKS: REQ-SIM-004
    #[test_case(1)]
    fn case_one(_x: u8) {}
}

#[cfg(not(test))]
mod prod {
    // TRLC-LINKS: REQ-SIM-005
    fn real() {}
}
`
	got := scanFixture(t, "src/clock.rs", src)
	want := map[string][2]bool{ // name -> {IsTest, TestOnly}
		"CODE-NOWMS":      {false, false},
		"CODE-FAKECLOCK":  {false, true},
		"CODE-NOW":        {false, true},
		"CODE-READSCLOCK": {true, true},
		"CODE-ASYNCCASE":  {true, true},
		"CODE-PARAMCASE":  {true, true},
		"CODE-CASEONE":    {true, true},
		"CODE-REAL":       {false, false},
	}
	for name, w := range want {
		s, ok := got[name]
		if !ok {
			t.Fatalf("missing symbol %s in %+v", name, got)
		}
		if s.IsTest != w[0] || s.TestOnly != w[1] {
			t.Errorf("%s: IsTest=%v TestOnly=%v, want IsTest=%v TestOnly=%v", name, s.IsTest, s.TestOnly, w[0], w[1])
		}
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func TestScan_MarksRustInnerCfgTestFileAsTestOnly(t *testing.T) {
	got := scanFixture(t, "src/support.rs", "#![cfg(test)]\n// TRLC-LINKS: REQ-SIM-001\nfn helper() {}\n")
	if s := got["CODE-HELPER"]; !s.TestOnly || s.IsTest {
		t.Fatalf("inner cfg(test) helper: %+v", s)
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func TestScan_MarksGoTestEntryPointsOnlyInTestFiles(t *testing.T) {
	src := `package sample

import "testing"

// TRLC-LINKS: REQ-GO-001
func TestReads(t *testing.T) {}

// TRLC-LINKS: REQ-GO-001
func BenchmarkReads(b *testing.B) {}

// TRLC-LINKS: REQ-GO-001
func FuzzReads(f *testing.F) {}

// TRLC-LINKS: REQ-GO-001
func ExampleReads() {}

// TRLC-LINKS: REQ-GO-002
func fakeClock() int { return 1 }

// TRLC-LINKS: REQ-GO-002
func Testdata() string { return "" }

// TRLC-LINKS: REQ-GO-002
func TestMain(m *testing.M) {}

type writer struct{}

// TRLC-LINKS: REQ-GO-002
func (writer) TestWrite(t *testing.T) {}
`
	got := scanFixture(t, "sample_test.go", src)
	for name, isTest := range map[string]bool{
		"CODE-TESTREADS": true, "CODE-BENCHMARKREADS": true, "CODE-FUZZREADS": true, "CODE-EXAMPLEREADS": true,
		"CODE-FAKECLOCK": false, "CODE-TESTDATA": false, "CODE-TESTMAIN": false, "CODE-TESTWRITE": false,
	} {
		s, ok := got[name]
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if s.IsTest != isTest || !s.TestOnly {
			t.Errorf("%s: IsTest=%v TestOnly=%v, want IsTest=%v TestOnly=true", name, s.IsTest, s.TestOnly, isTest)
		}
	}
	prod := scanFixture(t, "sample.go", "package sample\n\n// TRLC-LINKS: REQ-GO-001\nfunc TestLooking() {}\n")
	if s := prod["CODE-TESTLOOKING"]; s.IsTest || s.TestOnly {
		t.Fatalf("production Go file must not yield tests: %+v", s)
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func TestScan_MarksPythonTestFunctionsAndMethods(t *testing.T) {
	src := `import pytest

# TRLC-LINKS: REQ-PY-001
def test_reads():
    # TRLC-LINKS: REQ-PY-002
    def test_nested():
        pass

# TRLC-LINKS: REQ-PY-002
@pytest.fixture
def fake_clock():
    return 1

class TestClock:
    # TRLC-LINKS: REQ-PY-001
    def test_method(self):
        pass

    # TRLC-LINKS: REQ-PY-002
    def helper(self):
        pass

    # TRLC-LINKS: REQ-PY-001
    def testCamel(self):
        pass
`
	got := scanFixture(t, "test_clock.py", src)
	for name, isTest := range map[string]bool{
		"CODE-TESTREADS": true, "CODE-TESTREADSTESTNESTED": false, "CODE-FAKECLOCK": false,
		"CODE-TESTCLOCKTESTMETHOD": true, "CODE-TESTCLOCKHELPER": false, "CODE-TESTCLOCKTESTCAMEL": true,
	} {
		s, ok := got[name]
		if !ok {
			t.Fatalf("missing %s in %+v", name, got)
		}
		if s.IsTest != isTest {
			t.Errorf("%s: IsTest=%v want %v", name, s.IsTest, isTest)
		}
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func TestScan_TestHelpersDoNotRequireTRLCLinks(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"helper_test.go":         "package sample\n\nimport \"testing\"\n\nfunc fakeClock() int { return 1 }\n\nfunc TestUnlinked(t *testing.T) {}\n",
		"src/clock.rs":           "#[cfg(test)]\nmod tests {\n    fn fake() {}\n    #[test]\n    fn unlinked() {}\n}\n\nfn prod() {}\n",
		"tests/integration/a.rs": "fn support() {}\n\n// TRLC-LINKS: REQ-SIM-001\n#[test]\nfn case() {}\n",
		"test_mod.py":            "def helper():\n    pass\n\ndef test_unlinked():\n    pass\n",
	}
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	symbols, diags, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	missing := map[string]bool{}
	for _, d := range diags {
		if d.Code == "code.missing_trlc_link" {
			missing[d.Path] = true
		}
	}
	want := map[string]bool{"helper_test.go:7": true, "src/clock.rs:5,8": true, "test_mod.py:4": true}
	if len(missing) != len(want) {
		t.Fatalf("missing_trlc_link paths = %v, want %v", missing, want)
	}
	for p := range want {
		if !missing[p] {
			t.Fatalf("missing_trlc_link paths = %v, want %v", missing, want)
		}
	}
	for _, s := range symbols {
		if s.Path == "tests/integration/a.rs" && (!s.TestOnly || !s.IsTest) {
			t.Fatalf("cargo integration test not marked: %+v", s)
		}
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func TestScan_AccumulatesStackedTRLCLinksLines(t *testing.T) {
	got := scanFixture(t, "multi_test.go", "package sample\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-COF-003\n// TRLC-LINKS: REQ-COF-004\nfunc TestSigned(t *testing.T) {}\n")
	s := got["CODE-TESTSIGNED"]
	if len(s.Implements) != 2 || s.Implements[0] != "REQ-COF-003" || s.Implements[1] != "REQ-COF-004" {
		t.Fatalf("stacked TRLC-LINKS lines must accumulate: %+v", s)
	}
}
