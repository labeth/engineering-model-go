// ENGMODEL-OWNER-UNIT: FU-TRLC-EXPORTER
package trlc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// expectation names a generated document and the fragments it must contain.
type expectation struct {
	name  string
	text  string
	wants []string
}

func assertContainsAll(t *testing.T, e expectation) {
	t.Helper()
	for _, want := range e.wants {
		if !strings.Contains(e.text, want) {
			t.Fatalf("%s missing %q", e.name, want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-006
func TestGenerateTRLCRequirementsFromFile_Examples(t *testing.T) {
	paths := []string{
		filepath.Join("examples", "payments-engineering-sample", "model", "requirements.yml"),
		filepath.Join("examples", "bedrock-pr-review-github-app-sample", "model", "requirements.yml"),
		filepath.Join("examples", "coffee-fleet-ota-cloud-sample", "model", "requirements.yml"),
	}
	for _, p := range paths {
		p := p
		t.Run(filepath.Base(filepath.Dir(p)), func(t *testing.T) {
			res, err := GenerateRequirementsFromFile(p, Options{})
			if err != nil {
				t.Fatalf("generate trlc from file failed: %v", err)
			}
			if strings.TrimSpace(res.PackageName) == "" {
				t.Fatalf("expected non-empty package name")
			}
			assertContainsAll(t, expectation{"model.rsl", res.ModelRSL, []string{"package ", "type Requirement", "applies_to String [0 .. *]"}})
			assertContainsAll(t, expectation{"requirements.trlc", res.RequirementsTRLC, []string{"package ", "Requirement ", "id         = \"REQ-"}})
		})
	}
}

// TRLC-LINKS: REQ-EMG-006
func TestEngTRLC_CLIAndValidation(t *testing.T) {
	manifest := filepath.Join("examples", "payments-engineering-sample", "engmod.yml")
	reqPath := filepath.Join("examples", "payments-engineering-sample", "model", "requirements.yml")
	outDir := filepath.Join(t.TempDir(), "trlc")

	cmd := exec.Command("go", "run", "./cmd/engtrlc", "--model", manifest, "--requirements", reqPath, "--out-dir", outDir, "--package", "PaymentsReqs")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("engtrlc failed: %v\noutput:\n%s", err, string(out))
	}

	modelPath := filepath.Join(outDir, "model.rsl")
	reqlibPath := filepath.Join(outDir, "requirements.trlc")
	if _, err := os.Stat(modelPath); err != nil {
		t.Fatalf("missing model.rsl: %v", err)
	}
	if _, err := os.Stat(reqlibPath); err != nil {
		t.Fatalf("missing requirements.trlc: %v", err)
	}

	if _, err := exec.LookPath("trlc"); err != nil {
		t.Skip("trlc binary not found on PATH; skipping syntax validation")
	}
	validate := exec.Command("trlc", outDir)
	vout, err := validate.CombinedOutput()
	if err != nil {
		t.Fatalf("trlc validation failed: %v\noutput:\n%s", err, string(vout))
	}
}
