// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// model.diagnose checks alias::ID references in the submitted content against
// the dependencies' publications, and reports the submitted content, not the
// document on disk: a submitted unpublished reference is an error attributed
// to the document with its line, and a bad reference on disk that the
// submitted content fixes is valid. Nothing is written into the model root.
// Mutation: composing from the on-disk manifest (the root's qualified
// references come from the files on disk) fails both halves of this test.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-065
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func TestModelDiagnoseChecksSubmittedQualifiedReferences(t *testing.T) {
	t.Setenv("CUE_REGISTRY", "none")
	t.Setenv("ENGMOD_LOCK", "")
	t.Setenv("ENGMOD_WORK", "")
	parent, good := diagnoseQualifiedFixture(t)
	bad := strings.Replace(good, "child::FU-CHILD-UNIT", "child::FU-NOPE", 1)
	requirementsPath := filepath.Join(parent, "model", "requirements.yml")
	writeMCPFile(t, requirementsPath, good)
	s := initializedServerAt(t, parent)

	baseline := s.callToolPayload(t, "model.diagnose", map[string]any{"document": "requirements", "content": good})
	if baseline["valid"] != true {
		t.Fatalf("good content must be valid: %+v", baseline)
	}
	requireUnpublishedReference(t, s.callToolPayload(t, "model.diagnose", map[string]any{"document": "requirements", "content": bad}))

	writeMCPFile(t, requirementsPath, bad)
	before := snapshotDir(t, filepath.Dir(parent))
	fixed := s.callToolPayload(t, "model.diagnose", map[string]any{"document": "requirements", "content": good})
	if fixed["valid"] != true {
		t.Fatalf("on-disk bad reference fixed by the submitted content must be valid: %+v", fixed)
	}
	requireSnapshotUnchanged(t, before, snapshotDir(t, filepath.Dir(parent)))
}

// diagnoseQualifiedFixture writes the composition fixture with the child
// publishing a functional group and returns the parent directory and a
// requirements document with a valid qualified reference.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-065
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func diagnoseQualifiedFixture(t *testing.T) (string, string) {
	t.Helper()
	parent := writeAuthoringCompositionFixture(t, true)
	childManifest := filepath.Join(filepath.Dir(parent), "child", "engmod.yml")
	data, err := os.ReadFile(childManifest)
	if err != nil {
		t.Fatal(err)
	}
	writeMCPFile(t, childManifest, strings.Replace(string(data), "[FU-CHILD-UNIT, CAP-CHILD]", "[FG-CHILD, FU-CHILD-UNIT, CAP-CHILD]", 1))
	writeMCPFile(t, filepath.Join(parent, "model", "catalog.yml"), "schemaVersion: 3\ncatalog:\n  systems:\n    - id: SYS-PARENT\n      name: parent system\n      definition: Test system.\n")
	good := "schemaVersion: 3\nlintRun: {mode: strict, catalogRef: ./catalog.yml}\nrequirements:\n  - id: REQ-PARENT\n    text: 'The parent system shall publish one message.'\n    appliesTo: [child::FU-CHILD-UNIT]\n"
	return parent, good
}

// requireUnpublishedReference requires an invalid result carrying an
// unpublished_reference diagnostic with line 6 and no temp path.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-065
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func requireUnpublishedReference(t *testing.T, result map[string]any) {
	t.Helper()
	var row map[string]any
	for _, item := range result["diagnostics"].([]any) {
		if d := item.(map[string]any); d["code"] == "composition.unpublished_reference" {
			row = d
		}
	}
	if row == nil || result["valid"] != false {
		t.Fatalf("submitted child::FU-NOPE must be an unpublished_reference of the document: %+v", result)
	}
	if row["line"] != float64(6) || strings.Contains(fmt.Sprint(row["path"]), os.TempDir()) {
		t.Fatalf("unpublished_reference must carry line 6 and no temp path: %+v", row)
	}
}

// requireSnapshotUnchanged fails, naming the changed files, when the workspace
// differs from the snapshot taken before the call.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-065
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func requireSnapshotUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	if fmt.Sprint(after) == fmt.Sprint(before) {
		return
	}
	for k, v := range after {
		if before[k] != v {
			t.Errorf("changed: %s", k)
		}
	}
	t.Fatalf("model.diagnose wrote into the workspace")
}
