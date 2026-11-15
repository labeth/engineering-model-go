// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// model.patch edits one entity by stable ID with line-preserving edits,
// validates the whole model with the edited document like model.diagnose and
// returns valid, diagnostics and a unified diff. Without write nothing is
// written; with write it writes only a valid result, atomically and keeping
// the file mode; an invalid result, an unknown id, a document outside the
// manifest and a document that resolves outside the model are refused with
// nothing written, and comments of untouched lines survive.
// Mutations: writing without write, writing an invalid result, validating the
// on-disk document instead of the edited one, dropping the mode, or resolving
// a symlinked document outside the model root fails this test.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func TestModelPatchEditsValidatesAndWritesOnlyValidResults(t *testing.T) {
	f := newPatchFixture(t)
	f.dryRun(t)
	f.validWrite(t)
	f.invalidWrite(t)
	f.refusals(t)
	f.outsideModelRoot(t)
}

// patchFixture is the payments model with a commented risk register and a
// server initialized on it; current is the document text expected on disk.
type patchFixture struct {
	s         *Server
	modelDir  string
	path      string
	riskID    string
	commented string
	current   string
}

// newPatchFixture copies the payments model, adds a comment that must survive
// every edit, makes the assurance document mode 0640 and starts a server.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func newPatchFixture(t *testing.T) *patchFixture {
	t.Helper()
	modelDir := copyPaymentsModel(t)
	assurancePath := filepath.Join(modelDir, "model", "assurance.yml")
	original, err := os.ReadFile(assurancePath)
	if err != nil {
		t.Fatal(err)
	}
	commented := strings.Replace(string(original), "  risks:\n", "  # risk register, reviewed weekly\n  risks:\n", 1)
	if commented == string(original) {
		t.Fatal("fixture: no risks section")
	}
	if err := os.WriteFile(assurancePath, []byte(commented), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(assurancePath, 0o640); err != nil {
		t.Fatal(err)
	}
	return &patchFixture{
		s: initializedServerAt(t, modelDir), modelDir: modelDir, path: assurancePath,
		riskID: firstID(t, commented, "  risks:\n"), commented: commented, current: commented,
	}
}

// read returns the assurance document on disk.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-REPO-LOCAL-WRITES
func (f *patchFixture) read(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(f.path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// dryRun applies a scalar set, a nested field and an append without write.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func (f *patchFixture) dryRun(t *testing.T) {
	t.Helper()
	dry := f.s.callToolPayload(t, "model.patch", map[string]any{
		"document": "assurance", "id": f.riskID,
		"set":    map[string]any{"title": "Patched risk title", "rationale": "Patched: because."},
		"append": map[string]any{"appliesTo": "FU-CHECKOUT"},
	})
	patch, _ := dry["patch"].(string)
	if dry["valid"] != true || dry["written"] != false {
		t.Fatalf("dry run must be valid with a unified diff and not written: %+v", dry)
	}
	check(t, "dry run patch").textHas(patch, "+    title: Patched risk title", "+    rationale: 'Patched: because.'", "@@ ")
	if !strings.HasPrefix(patch, "--- a/model/assurance.yml\n+++ b/model/assurance.yml\n") {
		t.Fatalf("dry run patch lacks the unified diff header: %s", patch)
	}
	if f.read(t) != f.commented {
		t.Fatal("a dry run wrote the document")
	}
}

// validWrite writes a valid nested edit: the file changes, the comment and
// the mode stay.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func (f *patchFixture) validWrite(t *testing.T) {
	t.Helper()
	wrote := f.s.callToolPayload(t, "model.patch", map[string]any{
		"document": "model/assurance.yml", "id": f.riskID, "write": true,
		"set": map[string]any{"title": "Patched risk title"},
	})
	if wrote["valid"] != true || wrote["written"] != true {
		t.Fatalf("a valid edit with write must be written: %+v", wrote)
	}
	f.current = f.read(t)
	check(t, "written document lost the edit or the comment").textHas(f.current, "    title: Patched risk title\n", "  # risk register, reviewed weekly\n")
	if info, err := os.Stat(f.path); err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("write must keep mode 0640: %v %v", info.Mode(), err)
	}
}

// invalidWrite reports an invalid result with diagnostics and writes nothing.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func (f *patchFixture) invalidWrite(t *testing.T) {
	t.Helper()
	invalid := f.s.callToolPayload(t, "model.patch", map[string]any{
		"document": "assurance", "id": f.riskID, "write": true,
		"set": map[string]any{"owner": "ACT-NOBODY"},
	})
	if invalid["valid"] != false || invalid["written"] != false || len(diagnosticCodes(invalid, "diagnostics")) == 0 {
		t.Fatalf("an invalid edit must report diagnostics and not be written: %+v", invalid)
	}
	if f.read(t) != f.current {
		t.Fatal("an invalid edit was written")
	}
}

// refusals covers an unknown id, an unknown field segment, a stable-ID
// rename, no operation, a document outside the manifest and wrong argument
// types: each is a tool error and nothing is written.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func (f *patchFixture) refusals(t *testing.T) {
	t.Helper()
	id := f.riskID
	title := map[string]any{"title": "x"}
	for name, args := range map[string]map[string]any{
		"unknown id":          {"document": "assurance", "id": "RISK-NONE", "set": title},
		"bad field":           {"document": "assurance", "id": id, "set": map[string]any{"../title": "x"}},
		"rename id":           {"document": "assurance", "id": id, "set": map[string]any{"id": "RISK-OTHER"}},
		"no operation":        {"document": "assurance", "id": id},
		"outside manifest":    {"document": "../engmod.yml", "id": id, "set": title},
		"manifest itself":     {"document": "engmod.yml", "id": id, "set": title},
		"set not object":      {"document": "assurance", "id": id, "set": "title=x"},
		"unset not list":      {"document": "assurance", "id": id, "unset": "title"},
		"write not bool":      {"document": "assurance", "id": id, "set": title, "write": "yes"},
		"remove missing item": {"document": "assurance", "id": id, "remove": map[string]any{"appliesTo": "FU-NOPE"}},
	} {
		if payload, isErr := f.s.callToolRaw(t, "model.patch", args); !isErr {
			t.Errorf("%s must be refused: %+v", name, payload)
		}
	}
	if f.read(t) != f.current {
		t.Fatal("a refused patch wrote the document")
	}
}

// outsideModelRoot refuses a manifest document that resolves outside the
// model root through a symlink and leaves the target untouched.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func (f *patchFixture) outsideModelRoot(t *testing.T) {
	t.Helper()
	outside := filepath.Join(t.TempDir(), "decisions.yml")
	decisions := filepath.Join(f.modelDir, "model", "decisions.yml")
	data, err := os.ReadFile(decisions)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(decisions); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, decisions); err != nil {
		t.Fatal(err)
	}
	f.requireDecisionsRefused(t, firstID(t, string(data), "decisions:\n"))
	check(t, "the outside document was written").fileIs(outside, data)
}

// requireDecisionsRefused requires model.patch on the decisions document to
// be refused as outside the model root.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func (f *patchFixture) requireDecisionsRefused(t *testing.T, decisionID string) {
	t.Helper()
	payload, isErr := f.s.callToolRaw(t, "model.patch", map[string]any{"document": "decisions", "id": decisionID, "write": true, "set": map[string]any{"title": "x"}})
	if !isErr || !strings.Contains(toString(payload["error"].(map[string]any)["message"]), "outside the model root") {
		t.Fatalf("a document outside the model root must be refused: %+v", payload)
	}
}

// firstID returns the id of the first list item after marker.
func firstID(t *testing.T, doc, marker string) string {
	t.Helper()
	i := strings.Index(doc, marker)
	if i < 0 {
		t.Fatalf("fixture: no %q", marker)
	}
	rest := doc[i+len(marker):]
	j := strings.Index(rest, "id: ")
	if j < 0 {
		t.Fatalf("fixture: no id after %q", marker)
	}
	id, _, _ := strings.Cut(rest[j+4:], "\n")
	return strings.TrimSpace(id)
}
