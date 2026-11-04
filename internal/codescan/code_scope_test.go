// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
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
	testutil.AssertDiag(t, diags, "model.invalid_requirement_source_ref", "CODE-APP-GONE-GO")
	testutil.AssertDiag(t, diags, "model.ambiguous_code_reference", "CODE-APP-A-B-GO")
	testutil.AssertDiag(t, diags, "model.inferred_id_not_allowed", "AUTO-")
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
	testutil.AssertDiag(t, validate.Bundle(bundle), "model.code_sources_removed", "exclude")
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
	testutil.AssertDiag(t, diags, "model.referenced_element_local_uri", "REF-LOCAL")
	testutil.AssertDiag(t, diags, "model.referenced_element_local_uri", "REF-FILE")
	testutil.AssertDiag(t, diags, "model.referenced_element_missing_version", "REF-WEB")
	for _, d := range diags {
		for _, ok := range []string{"REF-WEB-V", "REF-PINNED", "REF-DOI", "REF-OFFLINE"} {
			if strings.Contains(d.Message, `"`+ok+`"`) {
				t.Fatalf("valid referenced element rejected: %+v", d)
			}
		}
	}
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

// A Dockerfile and a README.md with trace markers become code items owned by
// the unit their ENGMODEL-OWNER-UNIT comment names, with their file CODE
// identifier counted once; inferenceHints.exclude removes an excluded
// Dockerfile from both.
// Mutation: dropping the artifact owner lookup leaves the Dockerfile unresolved.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func TestArtifactFilesAreOwnedIndexedAndExcluded(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "Dockerfile"), "# ENGMODEL-OWNER-UNIT: FU-IMAGE\n# TRLC-LINKS: REQ-IMG-001\nFROM scratch\n")
	mustWrite(t, filepath.Join(root, "docs", "RUN.md"), "<!-- ENGMODEL-OWNER-UNIT: FU-DOCS -->\n<!-- TRLC-LINKS: REQ-DOC-001 -->\n# Run\n")
	mustWrite(t, filepath.Join(root, "build", "Dockerfile"), "# ENGMODEL-OWNER-UNIT: FU-IMAGE\n# TRLC-LINKS: REQ-IMG-009\nFROM scratch\n")
	bundle := model.Bundle{ManifestPath: filepath.Join(root, "engmod.yml")}
	bundle.Architecture.InferenceHints.Exclude = []string{"build/**"}

	items, diags := InferCodeItems(bundle, "")
	for _, d := range diags {
		if d.Severity == validate.SeverityError {
			t.Fatalf("unexpected error %+v", d)
		}
	}
	owners := map[string]string{}
	for _, it := range items {
		owners[it.Kind+" "+it.Source] = it.Owner + " " + strings.Join(it.Implements, ",")
	}
	want := map[string]string{
		"source_file Dockerfile":  "FU-IMAGE ",
		"symbol Dockerfile:2":     "FU-IMAGE REQ-IMG-001",
		"source_file docs/RUN.md": "FU-DOCS ",
		"symbol docs/RUN.md:2":    "FU-DOCS REQ-DOC-001",
	}
	if len(owners) != len(want) {
		t.Fatalf("want items %v, got %v", want, owners)
	}
	for k, v := range want {
		if owners[k] != v {
			t.Fatalf("%s: want %q, got %q (all %v)", k, v, owners[k], owners)
		}
	}
	index := buildCodeIndex(bundle)
	if index["CODE-DOCKERFILE"] != 1 || index["CODE-DOCS-RUN-MD"] != 1 || index["CODE-BUILD-DOCKERFILE"] != 0 {
		t.Fatalf("artifact CODE ids must be indexed once and excluded ones not at all: %v", index)
	}
}

// Every directory form of an exclude pattern (dir, dir/, dir/*, dir/**)
// leaves the whole directory out of the code scope, also for a file below a
// subdirectory, and a pattern that only shares a prefix keeps its files.
// Mutation: matching patterns against the full path only (no ancestor
// match) keeps dist/assets/app.js for dir/*, and keeping the trailing slash
// makes dir/ match nothing.
// TRLC-LINKS: REQ-EMG-114
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func TestExcludeAcceptsDirectoryForms(t *testing.T) {
	for _, pattern := range []string{"viewer/dist", "viewer/dist/", "viewer/dist/*", "viewer/dist/**", "./viewer/dist"} {
		if !excludedPath("viewer/dist/assets/app.js", []string{pattern}) || !excludedPath("viewer/dist/index.js", []string{pattern}) {
			t.Errorf("%q must exclude everything below viewer/dist", pattern)
		}
		if excludedPath("viewer/distribution/app.js", []string{pattern}) || excludedPath("viewer/src/main.js", []string{pattern}) {
			t.Errorf("%q must not exclude files outside viewer/dist", pattern)
		}
	}
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "viewer", "dist", "assets", "app.js"), "function bundled() {}\n")
	mustWrite(t, filepath.Join(root, "viewer", "src", "main.js"), "// TRLC-LINKS: REQ-X-001\nfunction main() {}\n")
	bundle := model.Bundle{ManifestPath: filepath.Join(root, "engmod.yml")}
	bundle.Architecture.InferenceHints.Exclude = []string{"viewer/dist/*"}
	symbols, diags, err := ScanCodeScope(bundle, root)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, s := range symbols {
		paths[s.Path] = true
	}
	for _, d := range diags {
		paths[d.Path] = true
	}
	if !paths["viewer/src/main.js"] || len(diags) != 0 {
		t.Fatalf("ScanCodeScope must scan viewer/src and leave viewer/dist out: %v %+v", paths, diags)
	}
}

// An exclude pattern the matcher cannot apply is a validation error naming
// the accepted forms; every accepted form validates.
// Mutation: dropping the ** check or the accepted-forms text fails this test.
// TRLC-LINKS: REQ-EMG-114
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-VALIDATION-ENGINE
func TestInvalidExcludePatternNamesAcceptedForms(t *testing.T) {
	bundle := model.Bundle{}
	bundle.Architecture.InferenceHints.Exclude = []string{"viewer/dist", "viewer/dist/", "viewer/dist/*", "viewer/dist/**", "*.gen.go", "/abs/dir", "../up/**", "**/dist", "a/**/b", "bad[glob"}
	reported := map[string]string{}
	for _, d := range validate.Bundle(bundle) {
		if d.Code == "model.invalid_exclude_pattern" {
			reported[d.Path] = d.Message
		}
	}
	for i := 5; i < 10; i++ {
		msg, ok := reported[fmt.Sprintf("inferenceHints.exclude[%d]", i)]
		if !ok || !strings.Contains(msg, "dir, dir/, dir/* or dir/**") {
			t.Errorf("exclude[%d] must be model.invalid_exclude_pattern naming the accepted forms, got %q", i, msg)
		}
	}
	if len(reported) != 5 {
		t.Fatalf("only the five unusable patterns may be reported: %v", reported)
	}
}
