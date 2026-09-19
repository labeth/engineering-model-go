// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-SYSML-V2-MODEL
func TestRunCoverageJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--coverage"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run returned %d: %s", code, stderr.String())
	}

	var manifest engmodel.SysMLCoverageManifest
	if err := json.Unmarshal(stdout.Bytes(), &manifest); err != nil {
		t.Fatalf("decode coverage JSON: %v\n%s", err, stdout.String())
	}
	if err := engmodel.ValidateSysMLV2Coverage(manifest); err != nil {
		t.Fatalf("invalid coverage manifest: %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func TestRunValidateCoverageRejectsStaleArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	data, err := json.MarshalIndent(engmodel.SysMLV2Coverage(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n', '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--validate-coverage", path}, &stdout, &stderr); code != 1 {
		t.Fatalf("expected validation failure, got %d", code)
	}
	if !strings.Contains(stderr.String(), "stale") {
		t.Fatalf("expected stale artifact diagnostic, got %q", stderr.String())
	}
}

// TRLC-LINKS: REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-SYSML-V2-MODEL
func TestRunGeneratesProjectionAndDiagnostic(t *testing.T) {
	var stdout, stderr bytes.Buffer
	modelPath := filepath.Join("..", "..", "architecture.yml")
	if code := run([]string{"--model", modelPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("run returned %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "package 'engineering-model-go'") {
		t.Fatalf("missing package declaration:\n%s", stdout.String())
	}
	if strings.Contains(stderr.String(), "lossy") || strings.Contains(stderr.String(), "unsupported") {
		t.Fatalf("generated project has blocking diagnostics: %s", stderr.String())
	}
}

// TRLC-LINKS: REQ-EMG-036
func TestRunRequiresModelOrCoverage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("expected usage exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: engsysml") {
		t.Fatalf("missing usage: %s", stderr.String())
	}
}
