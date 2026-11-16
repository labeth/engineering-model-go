// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"testing"
)

// The test-only links of these files are checked against the trace matrix in
// internal/trace (TestPythonTestOnlyLinksDoNotImplement).
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestPythonOwnershipAndTestOnlyInference(t *testing.T) {
	root := t.TempDir()
	writeSourceFiles(t, root, map[string]string{
		"worker.py":      "# ENGMODEL-OWNER-UNIT: FU-PY\ntext = '''\n# ENGMODEL-OWNER-UNIT: FU-FAKE''' # real trailing comment\n# TRLC-LINKS: REQ-PY-001\ndef run(): return 1\n",
		"test_worker.py": "# ENGMODEL-OWNER-UNIT: FU-TEST\n# TRLC-LINKS: REQ-PY-001, REQ-PY-002\ndef test_run(): pass\ntext = '''\n# TRLC-LINKS: REQ-PY-003\n'''\n",
	})
	if metadata := scanCodeMetadata(root, nil); metadata["worker.py"].Owner != "FU-PY" {
		t.Fatalf("false ownership: %+v", metadata)
	}
	assertCleanInference(t, root, "REQ-PY-001", "REQ-PY-002", "REQ-PY-003")
	if LanguageFromPath("worker.py") != "python" {
		t.Fatal("missing publication language")
	}
}
