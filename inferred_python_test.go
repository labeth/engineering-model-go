// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engmodel

import (
	"github.com/labeth/engineering-model-go/model"
	"os"
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestPythonOwnershipAndTestOnlyInference(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"worker.py":      "# ENGMODEL-OWNER-UNIT: FU-PY\ntext = '''\n# ENGMODEL-OWNER-UNIT: FU-FAKE''' # real trailing comment\n# TRLC-LINKS: REQ-PY-001\ndef run(): return 1\n",
		"test_worker.py": "# ENGMODEL-OWNER-UNIT: FU-TEST\n# TRLC-LINKS: REQ-PY-001, REQ-PY-002\ndef test_run(): pass\ntext = '''\n# TRLC-LINKS: REQ-PY-003\n'''\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	metadata := scanCodeMetadata(root)
	if metadata["worker.py"].Owner != "FU-PY" {
		t.Fatalf("false ownership: %+v", metadata)
	}
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Architecture: model.ArchitectureDocument{InferenceHints: model.InferenceHints{CodeSources: []string{"worker.py", "test_worker.py"}}}}
	code, diags := inferCodeItems(bundle, "")
	if len(diags) != 0 || len(code) != 4 {
		t.Fatalf("code: %+v diagnostics: %+v", code, diags)
	}
	requirements := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-PY-001"}, {ID: "REQ-PY-002"}, {ID: "REQ-PY-003"}}}
	checks, diags := inferVerificationChecks(bundle, requirements, code, "")
	if len(diags) != 0 || len(checks) != 1 || checks[0].Status != "not-run" {
		t.Fatalf("checks: %+v %+v", checks, diags)
	}
	matrix := buildTraceMatrix(bundle, requirements, code, checks, nil)
	if matrix.Summary.Implemented != 1 || matrix.Summary.Verified != 2 {
		t.Fatalf("test-only code credited: %+v", matrix)
	}
	if languageFromPath("worker.py") != "python" {
		t.Fatal("missing publication language")
	}
}
