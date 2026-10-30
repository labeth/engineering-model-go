// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"os"
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-010
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

	meta := scanCodeMetadata(dir, nil)
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
