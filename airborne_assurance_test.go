// ENGMODEL-OWNER-UNIT: FU-AIRBORNE-ASSURANCE-EXPORTER
package engmodel

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-053, REQ-EMG-055, REQ-EMG-056
func TestGenerateAirborneAssuranceExample(t *testing.T) {
	path := filepath.Join("examples", "dal-c-flight-control-sample", "engmod.yml")
	first, err := GenerateAirborneAssuranceFromFile(path, AirborneAssuranceOptions{Date: "2026-09-19", Version: "v0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateAirborneAssuranceFromFile(path, AirborneAssuranceOptions{Date: "2026-09-19", Version: "v0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Files) != 15 {
		t.Fatalf("expected 15 outputs, got %d", len(first.Files))
	}
	for name, content := range first.Files {
		if !bytes.Equal(content, second.Files[name]) {
			t.Fatalf("%s is not deterministic", name)
		}
		if !bytes.Contains(content, []byte(AirborneDisclaimer)) {
			t.Fatalf("%s lacks disclaimer", name)
		}
	}
	if !bytes.Contains(first.Files["OPEN-GAPS.json"], []byte("GAP-COVERAGE-001")) {
		t.Fatal("open coverage gap is absent")
	}
	if bytes.Contains(first.Files["OBJECTIVE-EVIDENCE-MATRIX.csv"], []byte("objectiveText")) {
		t.Fatal("objective matrix must not contain normative objective text")
	}
}

// TRLC-LINKS: REQ-EMG-053, REQ-EMG-055
func TestGenerateAirborneAssuranceAbsentProfile(t *testing.T) {
	_, err := GenerateAirborneAssuranceFromFile("engmod.yml", AirborneAssuranceOptions{})
	if !errors.Is(err, ErrNoAviationProfile) {
		t.Fatalf("expected sentinel, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-055, REQ-EMG-056
func TestEngairCLI(t *testing.T) {
	out := t.TempDir()
	command := exec.Command("go", "run", "./cmd/engair", "--model", "examples/dal-c-flight-control-sample/engmod.yml", "--out-dir", out, "--date", "2026-09-19", "--version", "v0.1.0")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("engair failed: %v\n%s", err, output)
	}
	content, err := os.ReadFile(filepath.Join(out, "READINESS-SUMMARY.md"))
	if err != nil || !strings.Contains(string(content), AirborneDisclaimer) {
		t.Fatalf("invalid CLI output: %v", err)
	}
}
