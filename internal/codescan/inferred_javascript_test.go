// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// The test-only links of these files are checked against the trace matrix in
// internal/trace (TestJavaScriptTestOnlyLinksDoNotImplement).
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestJavaScriptOwnershipAndTestOnlyTraceInference(t *testing.T) {
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
			metadata := scanCodeMetadata(root, nil)
			if metadata[impl].Owner != "FU-WEB" {
				t.Fatalf("ownership lost: %+v", metadata)
			}
			bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Architecture: model.ArchitectureDocument{InferenceHints: model.InferenceHints{CodeSources: []string{impl, test}}}}
			code, diags := InferCodeItems(bundle, "")
			if len(diags) != 0 || len(code) != 4 {
				t.Fatalf("code inference: %+v %+v", code, diags)
			}
			reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-WEB-001"}, {ID: "REQ-WEB-002"}}}
			checks, diags := InferVerificationChecks(bundle, reqs, code, "")
			if len(diags) != 0 || len(checks) != 1 || checks[0].Status != "not-run" {
				t.Fatalf("checks: %+v %+v", checks, diags)
			}
			if LanguageFromPath(impl) != "javascript" {
				t.Fatal("publication language missing")
			}
		})
	}
}
