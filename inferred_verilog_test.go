// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engmodel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-010
func TestVerilogFileOwnership(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "capture.v"), []byte("// ENGMODEL-OWNER-UNIT: FU-CAPTURE\n// TRLC-LINKS: REQ-CAP-001\nmodule capture;\nendmodule\n"), 0600); err != nil {
		t.Fatal(err)
	}
	metadata := scanCodeMetadata(root)
	if len(metadata) != 1 || metadata["capture.v"].Owner != "FU-CAPTURE" {
		t.Fatalf("lost RTL ownership: %+v", metadata)
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerilogIndividualSourcesAndVerification(t *testing.T) {
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
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Architecture: model.ArchitectureDocument{InferenceHints: model.InferenceHints{CodeSources: []string{"capture.v", "tb_capture.v"}}}}
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-CAP-001"}, {ID: "REQ-CAP-002"}}}
	code, diags := inferCodeItems(bundle, "")
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
	checks, diags := inferVerificationChecks(bundle, reqs, code, "")
	if len(diags) != 0 || len(checks) != 1 || len(checks[0].Verifies) != 2 || checks[0].Status != "not-run" {
		t.Fatalf("bad verification inference: %+v %+v", checks, diags)
	}
	matrix := buildTraceMatrix(bundle, reqs, code, checks, nil)
	if matrix.Summary.Implemented != 1 || matrix.Summary.Verified != 2 {
		t.Fatalf("test-only links must not imply implementation: %+v", matrix)
	}
	roots := effectiveCodeRoots(bundle, "")
	if len(roots) != 2 {
		t.Fatalf("file roots lost from scoping: %+v", roots)
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
	items := []inferredCodeItem{{Element: "capture", Source: "capture.v:1", AbsPath: file, Implements: []string{"REQ-CAP-001"}}}
	if got := scopeCodeToModel(items, []string{file}, root); len(got) != 0 {
		t.Fatalf("file root bypassed nested model scope: %+v", got)
	}
	if got := scopeCodeToModel(items, []string{file}, nested); len(got) != 1 {
		t.Fatalf("own model lost its file: %+v", got)
	}
}
