// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-058
func TestFileIDFoldsPathSeparators(t *testing.T) {
	cases := map[string]string{
		"app/internal/sramcapture/capture.go": "CODE-APP-INTERNAL-SRAMCAPTURE-CAPTURE-GO",
		"fpga/images/stack_tile.v":            "CODE-FPGA-IMAGES-STACK-TILE-V",
		".gitignore":                          "CODE-GITIGNORE",
	}
	for in, want := range cases {
		if got := codemap.FileID(in); got != want {
			t.Fatalf("FileID(%q) = %q, want %q", in, got, want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-057
func TestModuleCodeRootStopsAtEnclosingModule(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	mustWrite(t, filepath.Join(repo, "model", "engmod.yml"), "")
	mustWrite(t, filepath.Join(repo, "engmod.yml"), "")
	mustWrite(t, filepath.Join(repo, "examples", "a", "engmod.yml"), "")

	if got := moduleCodeRoot(model.Bundle{ManifestPath: filepath.Join(repo, "engmod.yml")}); got != repo {
		t.Fatalf("root model should own the repository, got %s", got)
	}
	nested := filepath.Join(repo, "examples", "a")
	if got := moduleCodeRoot(model.Bundle{ManifestPath: filepath.Join(nested, "engmod.yml")}); got != nested {
		t.Fatalf("nested model should own its directory, got %s", got)
	}

	solo := t.TempDir()
	mustWrite(t, filepath.Join(solo, ".git", "HEAD"), "ref: refs/heads/main\n")
	mustWrite(t, filepath.Join(solo, "model", "engmod.yml"), "")
	if got := moduleCodeRoot(model.Bundle{ManifestPath: filepath.Join(solo, "model", "engmod.yml")}); got != solo {
		t.Fatalf("model in a subdirectory should own the repository, got %s", got)
	}
}

// TRLC-LINKS: REQ-EMG-057
func TestExcludedPathPatterns(t *testing.T) {
	patterns := []string{"build/**", "*.gen.go", "docs/notes.md"}
	for _, rel := range []string{"build", "build/out/x.go", "a.gen.go", "docs/notes.md"} {
		if !excludedPath(rel, patterns) {
			t.Fatalf("%q should be excluded", rel)
		}
	}
	for _, rel := range []string{"src/a.go", "builder/x.go", "docs/other.md"} {
		if excludedPath(rel, patterns) {
			t.Fatalf("%q should not be excluded", rel)
		}
	}
}

// TRLC-LINKS: REQ-EMG-057, REQ-EMG-058, REQ-EMG-059
func TestCodeReferencesResolveAgainstModuleFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "app", "capture.go"), "package app\n\n// TRLC-LINKS: REQ-X-001\nfunc Capture() {}\n")
	mustWrite(t, filepath.Join(root, "app", "README.md"), "notes\n")
	mustWrite(t, filepath.Join(root, "app", "a-b.go"), "package app\n")
	mustWrite(t, filepath.Join(root, "app", "a_b.go"), "package app\n")
	mustWrite(t, filepath.Join(root, "build", "out.go"), "package build\n")
	mustWrite(t, filepath.Join(root, "sub", "engmod.yml"), "")
	mustWrite(t, filepath.Join(root, "sub", "own.go"), "package sub\n")

	bundle := model.Bundle{ManifestPath: filepath.Join(root, "engmod.yml")}
	bundle.Architecture.InferenceHints.Exclude = []string{"build/**"}
	index := buildCodeIndex(bundle)
	for _, id := range []string{"CODE-APP-CAPTURE-GO", "CODE-APP-README-MD", "CODE-APP-CAPTURE-GO--CAPTURE"} {
		if index[id] != 1 {
			t.Fatalf("expected %s once, index=%v", id, index)
		}
	}
	if index["CODE-APP-A-B-GO"] != 2 {
		t.Fatalf("colliding paths must be counted twice, index=%v", index)
	}
	for _, id := range []string{"CODE-BUILD-OUT-GO", "CODE-SUB-OWN-GO"} {
		if index[id] != 0 {
			t.Fatalf("%s is excluded or owned by a nested module, index=%v", id, index)
		}
	}

	bundle.CodeIndex = func() map[string]int { return index }
	bundle.Requirements.Requirements = []model.Requirement{{ID: "REQ-X-001", Text: "x", SourceRefs: []string{"CODE-APP-CAPTURE-GO", "CODE-APP-GONE-GO", "CODE-APP-A-B-GO"}}}
	bundle.Architecture.AuthoredArchitecture.FunctionalUnits = []model.FunctionalUnit{{ID: "FU-X", Name: "x"}}
	bundle.Architecture.AuthoredArchitecture.Mappings = []model.Mapping{
		{Type: "depends_on", From: "FU-X", To: "CODE-APP-CAPTURE-GO--CAPTURE"},
		{Type: "depends_on", From: "FU-X", To: "AUTO-app/x.go:3"},
	}
	diags := validate.Bundle(bundle)
	assertDiag(t, diags, "model.invalid_requirement_source_ref", "CODE-APP-GONE-GO")
	assertDiag(t, diags, "model.ambiguous_code_reference", "CODE-APP-A-B-GO")
	assertDiag(t, diags, "model.inferred_id_not_allowed", "AUTO-")
	for _, d := range diags {
		if strings.Contains(d.Message, `"CODE-APP-CAPTURE-GO"`) || strings.Contains(d.Message, "CODE-APP-CAPTURE-GO--CAPTURE") {
			t.Fatalf("resolvable CODE reference rejected: %+v", d)
		}
	}
}

// TRLC-LINKS: REQ-EMG-057
func TestCodeSourcesIsRejected(t *testing.T) {
	bundle := model.Bundle{}
	bundle.Architecture.InferenceHints.CodeSources = []string{"./src"}
	assertDiag(t, validate.Bundle(bundle), "model.code_sources_removed", "exclude")
}

// TRLC-LINKS: REQ-EMG-060
func TestReferencedElementURIRules(t *testing.T) {
	refs := []model.ReferencedElement{
		{ID: "REF-LOCAL", Name: "a", URI: "../app/capture.go"},
		{ID: "REF-FILE", Name: "b", URI: "file:///repo/app/capture.go"},
		{ID: "REF-WEB", Name: "c", URI: "https://example.com/datasheet.pdf"},
		{ID: "REF-WEB-V", Name: "d", URI: "https://example.com/manual.pdf", Version: "rev C"},
		{ID: "REF-PINNED", Name: "e", URI: "https://github.com/o/r/blob/0123456789abcdef0123456789abcdef01234567/x.go"},
		{ID: "REF-DOI", Name: "f", URI: "doi:10.1000/182"},
		{ID: "REF-OFFLINE", Name: "g", Version: "Issue 2"},
	}
	bundle := model.Bundle{}
	bundle.Architecture.AuthoredArchitecture.ReferencedElements = refs
	diags := validate.Bundle(bundle)
	assertDiag(t, diags, "model.referenced_element_local_uri", "REF-LOCAL")
	assertDiag(t, diags, "model.referenced_element_local_uri", "REF-FILE")
	assertDiag(t, diags, "model.referenced_element_missing_version", "REF-WEB")
	for _, d := range diags {
		for _, ok := range []string{"REF-WEB-V", "REF-PINNED", "REF-DOI", "REF-OFFLINE"} {
			if strings.Contains(d.Message, `"`+ok+`"`) {
				t.Fatalf("valid referenced element rejected: %+v", d)
			}
		}
	}
}

// TRLC-LINKS: REQ-EMG-059
func assertDiag(t *testing.T, diags []validate.Diagnostic, code, fragment string) {
	t.Helper()
	for _, d := range diags {
		if d.Code == code && (strings.Contains(d.Message, fragment) || strings.Contains(d.Path, fragment)) {
			return
		}
	}
	t.Fatalf("missing %s mentioning %q in %+v", code, fragment, diags)
}

// TRLC-LINKS: REQ-EMG-057
func mustWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// GenerateAsciiDoc resolves CODE references for a bundle a caller loaded with
// model.LoadBundle (engws check, MCP model.diagnose), not only through the
// package's own file loaders.
// Mutation: not attaching the code index in GenerateAsciiDoc fails this test.
// TRLC-LINKS: REQ-EMG-059, REQ-EMG-068, REQ-EMG-069
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-CODEMAP-INFERENCE
func TestGenerateAsciiDocResolvesCodeReferencesForLoadedBundle(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	dir := t.TempDir()
	writeInterfaceModule(t, dir)
	mustWrite(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 2\nlintRun: {}\nrequirements:\n  - id: REQ-IF-001\n    text: Interface requirement.\n    sourceRefs: [CODE-SRC-IFACE-GO, CODE-SRC-GONE-GO]\n")
	mustWrite(t, filepath.Join(dir, "src", "iface.go"), "package iface\n\n// TRLC-LINKS: REQ-IF-001\nfunc Encode() {}\n")
	bundle, err := model.LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	result, _ := GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, AsciiDocOptions{})
	assertDiag(t, result.Diagnostics, "model.invalid_requirement_source_ref", "CODE-SRC-GONE-GO")
	for _, d := range result.Diagnostics {
		if strings.Contains(d.Message, "CODE-SRC-IFACE-GO") {
			t.Fatalf("resolvable CODE reference rejected: %+v", d)
		}
	}
}
