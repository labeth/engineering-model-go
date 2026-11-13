// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package integration

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/export/gemara"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
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
// links.list output for its requirements. Rendered with the workspace
// active it still names none by default; only with the Used By opt-in does
// it show Used By with the system's links, computed at render time.
// Mutation: rendering Used By without the opt-in, rendering it without an
// active workspace, or not rendering it when opted in fails this test.
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
	editFile(t, filepath.Join(root, "repos", "system", "model", "req-links.yml"), "- id: LNK-AOU-B-1\n", "- id: LNK-SYS-002-GHOST\n  parent: REQ-SYS-002\n  child: encoder::CAP-GHOST\n  pinned:\n    version: v0.1.0\n  status: claimed\n- id: LNK-AOU-B-1\n")

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

	// In the workspace with the system, by default: still no user. Used By
	// is opt-in (AsciiDocOptions.UsedBy, engdoc --used-by), so a committed
	// render of a child never lists its users.
	t.Setenv(model.WorkspaceEnv, "")
	assertNoDisclosure(t, "default AsciiDoc of child-a in the workspace", renderRepo(t, root, "child-a", "").Document, markers)

	// Opted in, in the workspace: Used By, from the system's link file.
	used := section(renderRepoUsedBy(t, root, "child-a", "").Document, "=== Used By")
	for _, want := range []string{"|CAP-A-1\n|REQLINKS-SYSTEM\n|LNK-SYS-001-ENCODER\n|REQ-SYS-001\n|accepted", "|CAP-A-2\n|REQLINKS-SYSTEM\n|LNK-SYS-002-ENCODER\n|REQ-SYS-002\n|accepted"} {
		if !strings.Contains(used, want) {
			t.Fatalf("Used By lacks %q:\n%s", want, used)
		}
	}
	if strings.Contains(used, "GHOST") {
		t.Fatalf("Used By lists a link onto no contract item of child-a:\n%s", used)
	}
	// Opted in with the workspace off: nothing to see, nothing shown.
	assertNoDisclosure(t, "opted-in AsciiDoc of child-a without a workspace", renderRepoUsedBy(t, root, "child-a", model.WorkspaceOff).Document, markers)
}

// A workspace file next to the using system's own manifest (the system is
// the workspace root, its children replacements below it) makes the system
// a visible user of each child: Used By lists its links.
// Mutation: scanning only the replacements fails this test.
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ASCIIDOC-GENERATOR
func TestUsedByIncludesTheWorkspaceRootModel(t *testing.T) {
	root := copyRequirementLinksExample(t)
	system := filepath.Join(root, "repos", "system")
	work := filepath.Join(system, model.WorkspaceFileName)
	testutil.WriteFile(t, work, "schemaVersion: 3\nreplacements:\n- module: models.example/reqlinks/child-a@v0\n  path: ../child-a\n")
	bundle, err := model.LoadBundle(filepath.Join(root, "repos", "child-a", "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	uses := composition.WorkspaceLinkUsers(bundle, work)
	if len(uses) != 2 || uses[0].Model != "REQLINKS-SYSTEM" || uses[0].Link != "LNK-SYS-001-ENCODER" || uses[1].Item != "CAP-A-2" {
		t.Fatalf("uses = %+v", uses)
	}
}

// The Used By scan reads only checkouts inside the workspace: a replacement
// whose path leaves the workspace directory, and a sibling whose manifest
// names a document outside its own module (the dependency confinement of
// composition), are skipped, so a crafted engmod.work.yml or sibling
// manifest cannot make a render read and show files elsewhere.
// Mutation: dropping the PathWithin bound or the dependencyPathEscapes
// check of the sibling scan fails this test.
// TRLC-LINKS: REQ-EMG-108
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func TestUsedByScanStaysInTheWorkspace(t *testing.T) {
	root := copyRequirementLinksExample(t)
	system := filepath.Join(root, "repos", "system")
	// A copy of the system outside the workspace, named by a replacement.
	outside := filepath.Join(t.TempDir(), "outside")
	copyDir(t, system, outside)
	editFile(t, filepath.Join(outside, "engmod.yml"), "modelId: REQLINKS-SYSTEM", "modelId: OUTSIDE-SYSTEM")
	rel, err := filepath.Rel(root, outside)
	if err != nil {
		t.Fatal(err)
	}
	// A sibling inside the workspace whose link file lies outside its module.
	escaping := filepath.Join(root, "repos", "escaping")
	copyDir(t, system, escaping)
	editFile(t, filepath.Join(escaping, "engmod.yml"), "modelId: REQLINKS-SYSTEM", "modelId: ESCAPING-SYSTEM")
	editFile(t, filepath.Join(escaping, "engmod.yml"), "reqLinks: model/req-links.yml", "reqLinks: ../../escaped-links.yml")
	data, err := os.ReadFile(filepath.Join(system, "model", "req-links.yml"))
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(root, "escaped-links.yml"), string(data))
	editFile(t, filepath.Join(root, model.WorkspaceFileName), "replacements:\n",
		"replacements:\n- module: models.example/reqlinks/outside@v0\n  path: "+filepath.ToSlash(rel)+"\n- module: models.example/reqlinks/escaping@v0\n  path: ./repos/escaping\n")

	bundle, err := model.LoadBundle(filepath.Join(root, "repos", "child-a", "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	uses := composition.WorkspaceLinkUsers(bundle, "")
	models := map[string]bool{}
	for _, u := range uses {
		models[u.Model] = true
	}
	if !models["REQLINKS-SYSTEM"] || models["OUTSIDE-SYSTEM"] || models["ESCAPING-SYSTEM"] {
		t.Fatalf("uses = %+v", uses)
	}
}

// copyDir copies a fixture directory tree.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
