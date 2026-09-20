// ENGMODEL-OWNER-UNIT: FU-NAF-EXPORTER
package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-042, REQ-EMG-043
// ENGMODEL-LINKS: FU-NAF-EXPORTER, IF-CLI-ENGNAF, DO-NAF-V4-ARCHITECTURE
func TestRunGeneratesNAFDocument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	modelPath := filepath.Join("..", "..", "engmod.yml")
	if code := run([]string{"--model", modelPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("run returned %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "=== A2 - Architecture Products") {
		t.Fatalf("missing NAF product:\n%s", stdout.String())
	}
}

// TRLC-LINKS: REQ-EMG-042
func TestRunRequiresModel(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("expected usage exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: engnaf") {
		t.Fatalf("missing usage: %s", stderr.String())
	}
}
