// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"os"
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-010
func TestVerilogFileOwnership(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "capture.v"), []byte("// ENGMODEL-OWNER-UNIT: FU-CAPTURE\n// TRLC-LINKS: REQ-CAP-001\nmodule capture;\nendmodule\n"), 0600); err != nil {
		t.Fatal(err)
	}
	metadata := scanCodeMetadata(root, nil)
	if len(metadata) != 1 || metadata["capture.v"].Owner != "FU-CAPTURE" {
		t.Fatalf("lost RTL ownership: %+v", metadata)
	}
}
