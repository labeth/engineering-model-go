// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A Dockerfile, a *.Dockerfile, a Containerfile and a Markdown file with
// TRLC-LINKS markers in their comments each yield one file-level symbol that
// implements the linked requirements; a marker in a Markdown code fence or
// in prose is ignored, and a Dockerfile without markers yields no symbol and
// no diagnostic.
// Mutation: dropping the code-fence check makes README.md also link REQ-DOC-099.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func TestScanReadsTraceMarkersInDockerfilesAndMarkdown(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"Dockerfile":           "# syntax=docker/dockerfile:1\n# TRLC-LINKS: REQ-IMG-001, REQ-IMG-002\n# ENGMODEL-LINKS: DEP-IMAGE\nFROM scratch\n",
		"build/app.Dockerfile": "FROM scratch\n# TRLC-LINKS: REQ-IMG-003\nRUN true\n",
		"Containerfile":        "# TRLC-LINKS: REQ-IMG-004\nFROM scratch\n",
		"plain/Dockerfile":     "FROM scratch\nRUN echo '# TRLC-LINKS: REQ-IMG-098'\n",
		"README.md":            "# Run\n\n<!-- TRLC-LINKS: REQ-DOC-001 -->\n\nSee `<!-- TRLC-LINKS: REQ-DOC-097 -->`.\n\n```md\n<!-- TRLC-LINKS: REQ-DOC-099 -->\n```\n<!--\n  TRLC-LINKS: REQ-DOC-002\n-->\n",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	symbols, diags, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Fatalf("artifacts must raise no diagnostics, got %+v", diags)
	}
	got := map[string]string{}
	for _, s := range symbols {
		if s.IsTest || s.TestOnly {
			t.Fatalf("an artifact is not a test: %+v", s)
		}
		got[s.Path] = s.TraceID + " " + strings.Join(s.Implements, ",") + " " + strings.Join(s.ModelLinks, ",")
	}
	want := map[string]string{
		"Dockerfile":           "CODE-DOCKERFILE REQ-IMG-001,REQ-IMG-002 DEP-IMAGE",
		"build/app.Dockerfile": "CODE-BUILD-APP-DOCKERFILE REQ-IMG-003 ",
		"Containerfile":        "CODE-CONTAINERFILE REQ-IMG-004 ",
		"README.md":            "CODE-README-MD REQ-DOC-001,REQ-DOC-002 ",
	}
	if len(got) != len(want) {
		t.Fatalf("want artifact symbols %v, got %v", want, got)
	}
	for path, w := range want {
		if got[path] != w {
			t.Fatalf("%s: want %q, got %q (all %v)", path, w, got[path], got)
		}
	}
}

// An artifact's ENGMODEL-OWNER-UNIT marker is read from a Dockerfile comment
// and a Markdown HTML comment, not from prose or a code fence.
// Mutation: taking the whole Dockerfile line as comment text (not stripping #) loses FU-IMAGE.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func TestArtifactOwnerUnitIsReadFromComments(t *testing.T) {
	cases := map[string]string{
		"Dockerfile": "# ENGMODEL-OWNER-UNIT: FU-IMAGE\nFROM scratch\n",
		"README.md":  "<!-- ENGMODEL-OWNER-UNIT: FU-DOCS -->\n# Title\n",
		"notes.md":   "ENGMODEL-OWNER-UNIT: FU-PROSE\n```\n<!-- ENGMODEL-OWNER-UNIT: FU-FENCE -->\n```\n",
	}
	want := map[string]string{"Dockerfile": "FU-IMAGE", "README.md": "FU-DOCS", "notes.md": ""}
	for name, content := range cases {
		if !IsArtifactFile(name) {
			t.Fatalf("%s must be an artifact file", name)
		}
		if got, _ := ArtifactMarker(name, []byte(content), "ENGMODEL-OWNER-UNIT:"); got != want[name] {
			t.Fatalf("%s: want owner %q, got %q", name, want[name], got)
		}
	}
	for _, name := range []string{"main.go", "Dockerfile.go", "docs.mdx", "dockerfile.txt"} {
		if IsArtifactFile(name) {
			t.Fatalf("%s must not be an artifact file", name)
		}
	}
}
