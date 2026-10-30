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
	parent := writeAuthoringCompositionFixture(t, true)
	childManifest := filepath.Join(filepath.Dir(parent), "child", "engmod.yml")
	data, err := os.ReadFile(childManifest)
	if err != nil {
		t.Fatal(err)
	}
	writeMCPFile(t, childManifest, strings.Replace(string(data), "[FU-CHILD-UNIT, CAP-CHILD]", "[FG-CHILD, FU-CHILD-UNIT, CAP-CHILD]", 1))
	writeMCPFile(t, filepath.Join(parent, "model", "catalog.yml"), "schemaVersion: 2\ncatalog:\n  systems:\n    - id: SYS-PARENT\n      name: parent system\n      definition: Test system.\n")
	requirementsPath := filepath.Join(parent, "model", "requirements.yml")
	good := "schemaVersion: 2\nlintRun: {mode: strict, catalogRef: ./catalog.yml}\nrequirements:\n  - id: REQ-PARENT\n    text: 'The parent system shall publish one message.'\n    appliesTo: [child::FU-CHILD-UNIT]\n"
	bad := strings.Replace(good, "child::FU-CHILD-UNIT", "child::FU-NOPE", 1)
	writeMCPFile(t, requirementsPath, good)
	s := initializedServerAt(t, parent)
	before := snapshotDir(t, filepath.Dir(parent))

	baseline := callToolPayload(t, s, "model.diagnose", map[string]any{"document": "requirements", "content": good})
	if baseline["valid"] != true {
		t.Fatalf("good content must be valid: %+v", baseline)
	}
	result := callToolPayload(t, s, "model.diagnose", map[string]any{"document": "requirements", "content": bad})
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

	writeMCPFile(t, requirementsPath, bad)
	before = snapshotDir(t, filepath.Dir(parent))
	fixed := callToolPayload(t, s, "model.diagnose", map[string]any{"document": "requirements", "content": good})
	if fixed["valid"] != true {
		t.Fatalf("on-disk bad reference fixed by the submitted content must be valid: %+v", fixed)
	}
	if after := snapshotDir(t, filepath.Dir(parent)); fmt.Sprint(after) != fmt.Sprint(before) {
		for k, v := range after {
			if before[k] != v {
				t.Errorf("changed: %s", k)
			}
		}
		t.Fatalf("model.diagnose wrote into the workspace")
	}
}
