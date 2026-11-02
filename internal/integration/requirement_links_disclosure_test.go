// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/export/gemara"
	"github.com/labeth/engineering-model-go/internal/model"
)

// systemMarkers are strings that only the using system knows: its model id,
// title, introduction, requirement IDs and texts, link IDs and rationale.
func systemMarkers(t *testing.T, root string) []string {
	t.Helper()
	system, err := model.LoadBundle(filepath.Join(root, "repos", "system", "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	markers := []string{"REQ-SYS", "LNK-", system.Manifest.Module.ModelID, system.Manifest.Module.Title, system.Manifest.Module.Introduction, system.Manifest.Module.Path,
		"ACT-SYSTEM-OWNER", "Used By"}
	for _, r := range system.Requirements.Requirements {
		markers = append(markers, r.Text)
	}
	for _, l := range system.ReqLinks.ReqLinks {
		markers = append(markers, l.Rationale)
	}
	return markers
}

// assertNoDisclosure fails when output names any system marker.
func assertNoDisclosure(t *testing.T, what, output string, markers []string) {
	t.Helper()
	for _, m := range markers {
		if m != "" && strings.Contains(output, m) {
			t.Fatalf("%s discloses %q of the using system", what, m)
		}
	}
}

// Acceptance 8: child-a rendered on its own (workspace off) names no using
// system, no system requirement and no parent text in its AsciiDoc, in every
// Gemara artifact or in the MCP requirements.get, requirements.parents and
// links.list output for its requirements; rendered with the workspace
// active it shows Used By with the system's links, computed at render time.
// Mutation: rendering Used By without an active workspace, or not rendering
// it in the workspace, fails this test.
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-GEMARA-EXPORTER, FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func TestRequirementLinksChildDisclosesNoUser(t *testing.T) {
	root := copyRequirementLinksExample(t)
	markers := systemMarkers(t, root)
	childDir := filepath.Join(root, "repos", "child-a")
	manifest := filepath.Join(childDir, "engmod.yml")
	requirements := filepath.Join(childDir, "model", "requirements.yml")
	// A system link onto an ID that is no contract item of child-a is not
	// a use of child-a's items.
	editFile(t, filepath.Join(root, "repos", "system", "model", "req-links.yml"), "- id: LNK-AOU-B-1\n", "- id: LNK-SYS-002-GHOST\n  parent: REQ-SYS-002\n  child: encoder::CAP-GHOST\n  pinned:\n    version: v0.1.0\n    fingerprint: \"\"\n  status: claimed\n- id: LNK-AOU-B-1\n")

	// The render's own workspace mode decides: off, although
	// engmod.work.yml is above the checkout and ENGMOD_WORK is unset.
	assertNoDisclosure(t, "AsciiDoc of child-a", renderRepo(t, root, "child-a", model.WorkspaceOff).Document, markers)
	t.Setenv(model.WorkspaceEnv, model.WorkspaceOff)
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	res, err := gemara.Generate(bundle, gemara.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.YAML) == 0 {
		t.Fatal("no Gemara artifacts")
	}
	for kind, doc := range res.YAML {
		assertNoDisclosure(t, "Gemara "+kind+" of child-a", doc, markers)
	}
	eval, err := gemara.GenerateEvaluationLog(bundle, bundle.Requirements, childDir, gemara.Options{})
	if err != nil {
		t.Fatal(err)
	}
	assertNoDisclosure(t, "Gemara evaluation log of child-a", eval.YAML, markers)
	for _, call := range []struct {
		tool string
		args map[string]any
	}{
		{"links.list", map[string]any{}}, {"links.suspect", map[string]any{}}, {"trace.matrix", map[string]any{}},
		{"requirements.get", map[string]any{"requirementId": "REQ-A-001"}}, {"requirements.parents", map[string]any{"requirementId": "REQ-A-001"}},
		{"requirements.get", map[string]any{"requirementId": "REQ-A-003"}}, {"requirements.parents", map[string]any{"requirementId": "REQ-A-003"}},
	} {
		out, err := mcpToolPayload(manifest, requirements, filepath.Join(childDir, "model", "views.yml"), call.tool, call.args)
		if err != nil {
			t.Fatal(err)
		}
		assertNoDisclosure(t, "MCP "+call.tool+" of child-a", out, markers)
	}

	// In the workspace with the system: Used By, from the system's link file.
	t.Setenv(model.WorkspaceEnv, "")
	used := section(renderRepo(t, root, "child-a", "").Document, "=== Used By")
	for _, want := range []string{"|CAP-A-1\n|REQLINKS-SYSTEM\n|LNK-SYS-001-ENCODER\n|REQ-SYS-001\n|accepted", "|CAP-A-2\n|REQLINKS-SYSTEM\n|LNK-SYS-002-ENCODER\n|REQ-SYS-002\n|accepted"} {
		if !strings.Contains(used, want) {
			t.Fatalf("Used By lacks %q:\n%s", want, used)
		}
	}
	if strings.Contains(used, "GHOST") {
		t.Fatalf("Used By lists a link onto no contract item of child-a:\n%s", used)
	}
}
