package engmodel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanCodeMetadata_DescriptionMarker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "handler.go")
	content := `// ENGMODEL-OWNER-UNIT: FU-GITHUB-WEBHOOK-INGRESS
// engmodel:code-description: validates webhook signatures and normalizes pull request events
package sample
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write code fixture: %v", err)
	}

	meta := scanCodeMetadata(dir)
	got, ok := meta["handler.go"]
	if !ok {
		t.Fatalf("expected metadata for handler.go")
	}
	if got.Owner != "FU-GITHUB-WEBHOOK-INGRESS" {
		t.Fatalf("unexpected owner: %q", got.Owner)
	}
	wantDesc := "validates webhook signatures and normalizes pull request events"
	if got.Description != wantDesc {
		t.Fatalf("unexpected description: got %q want %q", got.Description, wantDesc)
	}
}

func TestBuildCodeReferences_UsesOwnerAndDescriptionFields(t *testing.T) {
	refs := buildCodeReferences([]inferredCodeItem{{
		Element:     "src/webhook_ingress.go",
		Kind:        "source_file",
		Owner:       "FU-GITHUB-WEBHOOK-INGRESS",
		Description: "validates webhook signatures and routes pull request events",
		Source:      "src/webhook_ingress.go",
	}})

	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(refs))
	}
	if refs[0].Owner != "FU-GITHUB-WEBHOOK-INGRESS" {
		t.Fatalf("unexpected owner: %q", refs[0].Owner)
	}
	wantDesc := "validates webhook signatures and routes pull request events"
	if refs[0].Description != wantDesc {
		t.Fatalf("unexpected description: got %q want %q", refs[0].Description, wantDesc)
	}
}
