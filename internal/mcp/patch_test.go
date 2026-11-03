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
	modelDir := copyPaymentsModel(t)
	assurancePath := filepath.Join(modelDir, "model", "assurance.yml")
	original, err := os.ReadFile(assurancePath)
	if err != nil {
		t.Fatal(err)
	}
	// A comment that must survive every edit.
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
	s := initializedServerAt(t, modelDir)
	read := func() string {
		data, err := os.ReadFile(assurancePath)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	riskID := firstID(t, commented, "  risks:\n")

	// Dry run: a scalar set, a nested field and an append, nothing written.
	dry := callToolPayload(t, s, "model.patch", map[string]any{
		"document": "assurance", "id": riskID,
		"set":    map[string]any{"title": "Patched risk title", "rationale": "Patched: because."},
		"append": map[string]any{"appliesTo": "FU-CHECKOUT"},
	})
	patch, _ := dry["patch"].(string)
	if dry["valid"] != true || dry["written"] != false || !strings.Contains(patch, "+    title: Patched risk title") ||
		!strings.Contains(patch, "+    rationale: 'Patched: because.'") || !strings.Contains(patch, "@@ ") || !strings.HasPrefix(patch, "--- a/model/assurance.yml\n+++ b/model/assurance.yml\n") {
		t.Fatalf("dry run must be valid with a unified diff and not written: %+v", dry)
	}
	if read() != commented {
		t.Fatal("a dry run wrote the document")
	}

	// Write a valid nested edit: the file changes, the comment and mode stay.
	wrote := callToolPayload(t, s, "model.patch", map[string]any{
		"document": "model/assurance.yml", "id": riskID, "write": true,
		"set": map[string]any{"title": "Patched risk title"},
	})
	if wrote["valid"] != true || wrote["written"] != true {
		t.Fatalf("a valid edit with write must be written: %+v", wrote)
	}
	after := read()
	if !strings.Contains(after, "    title: Patched risk title\n") || !strings.Contains(after, "  # risk register, reviewed weekly\n") {
		t.Fatalf("written document lost the edit or the comment:\n%s", after)
	}
	if info, err := os.Stat(assurancePath); err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("write must keep mode 0640: %v %v", info.Mode(), err)
	}

	// An invalid result is reported with diagnostics and not written.
	invalid := callToolPayload(t, s, "model.patch", map[string]any{
		"document": "assurance", "id": riskID, "write": true,
		"set": map[string]any{"owner": "ACT-NOBODY"},
	})
	if invalid["valid"] != false || invalid["written"] != false || len(diagnosticCodes(invalid, "diagnostics")) == 0 {
		t.Fatalf("an invalid edit must report diagnostics and not be written: %+v", invalid)
	}
	if read() != after {
		t.Fatal("an invalid edit was written")
	}

	// Refusals: unknown id, unknown field segment, stable-ID rename, no
	// operation, a document outside the manifest, and wrong argument types.
	for name, args := range map[string]map[string]any{
		"unknown id":          {"document": "assurance", "id": "RISK-NONE", "set": map[string]any{"title": "x"}},
		"bad field":           {"document": "assurance", "id": riskID, "set": map[string]any{"../title": "x"}},
		"rename id":           {"document": "assurance", "id": riskID, "set": map[string]any{"id": "RISK-OTHER"}},
		"no operation":        {"document": "assurance", "id": riskID},
		"outside manifest":    {"document": "../engmod.yml", "id": riskID, "set": map[string]any{"title": "x"}},
		"manifest itself":     {"document": "engmod.yml", "id": riskID, "set": map[string]any{"title": "x"}},
		"set not object":      {"document": "assurance", "id": riskID, "set": "title=x"},
		"unset not list":      {"document": "assurance", "id": riskID, "unset": "title"},
		"write not bool":      {"document": "assurance", "id": riskID, "set": map[string]any{"title": "x"}, "write": "yes"},
		"remove missing item": {"document": "assurance", "id": riskID, "remove": map[string]any{"appliesTo": "FU-NOPE"}},
	} {
		if payload, isErr := callToolRaw(t, s, "model.patch", args); !isErr {
			t.Errorf("%s must be refused: %+v", name, payload)
		}
	}
	if read() != after {
		t.Fatal("a refused patch wrote the document")
	}

	// A manifest document that resolves outside the model root is refused.
	outside := filepath.Join(t.TempDir(), "decisions.yml")
	decisions := filepath.Join(modelDir, "model", "decisions.yml")
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
	decisionID := firstID(t, string(data), "decisions:\n")
	if payload, isErr := callToolRaw(t, s, "model.patch", map[string]any{"document": "decisions", "id": decisionID, "write": true, "set": map[string]any{"title": "x"}}); !isErr || !strings.Contains(toString(payload["error"].(map[string]any)["message"]), "outside the model root") {
		t.Fatalf("a document outside the model root must be refused: %+v", payload)
	}
	if got, _ := os.ReadFile(outside); string(got) != string(data) {
		t.Fatal("the outside document was written")
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
