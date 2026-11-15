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
		t.Run(ext, func(t *testing.T) { checkJavaScriptInference(t, ext) })
	}
}

// checkJavaScriptInference infers ownership, code items and checks of an
// implementation file and a test file of one extension.
func checkJavaScriptInference(t *testing.T, ext string) {
	root := t.TempDir()
	impl := "browser." + ext
	writeSourceFiles(t, root, map[string]string{
		impl:                  "// ENGMODEL-OWNER-UNIT: FU-WEB\n// TRLC-LINKS: REQ-WEB-001\nconst poll = () => 1;\n",
		"browser.test." + ext: "// ENGMODEL-OWNER-UNIT: FU-TEST\n// TRLC-LINKS: REQ-WEB-001, REQ-WEB-002\nfunction check() {}\n",
	})
	if metadata := scanCodeMetadata(root, nil); metadata[impl].Owner != "FU-WEB" {
		t.Fatalf("ownership lost: %+v", metadata)
	}
	assertCleanInference(t, root, "REQ-WEB-001", "REQ-WEB-002")
	if LanguageFromPath(impl) != "javascript" {
		t.Fatal("publication language missing")
	}
}

// assertCleanInference infers the code of root (four items) and the checks of
// the requirements (one check, not run), each without diagnostics.
func assertCleanInference(t *testing.T, root string, reqIDs ...string) {
	t.Helper()
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml")}
	code, diags := InferCodeItems(bundle, "")
	if len(diags) != 0 || len(code) != 4 {
		t.Fatalf("code: %+v diagnostics: %+v", code, diags)
	}
	var reqs model.RequirementsDocument
	for _, id := range reqIDs {
		reqs.Requirements = append(reqs.Requirements, model.Requirement{ID: id})
	}
	checks, diags := InferVerificationChecks(bundle, reqs, code, "")
	if len(diags) != 0 || len(checks) != 1 || checks[0].Status != "not-run" {
		t.Fatalf("checks: %+v %+v", checks, diags)
	}
}

// writeSourceFiles writes each named file under root.
func writeSourceFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
