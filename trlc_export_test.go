// ENGMODEL-OWNER-UNIT: FU-TRLC-EXPORTER
package engmodel

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-006
func TestGenerateTRLCRequirementsFromFile_Examples(t *testing.T) {
	paths := []string{
		filepath.Join("examples", "payments-engineering-sample", "requirements.yml"),
		filepath.Join("examples", "bedrock-pr-review-github-app-sample", "requirements.yml"),
		filepath.Join("examples", "coffee-fleet-ota-cloud-sample", "requirements.yml"),
	}
	for _, p := range paths {
		p := p
		t.Run(filepath.Base(filepath.Dir(p)), func(t *testing.T) {
			res, err := GenerateTRLCRequirementsFromFile(p, TRLCExportOptions{})
			if err != nil {
				t.Fatalf("generate trlc from file failed: %v", err)
			}
			if strings.TrimSpace(res.PackageName) == "" {
				t.Fatalf("expected non-empty package name")
			}
			for _, want := range []string{"package ", "type Requirement", "applies_to String [0 .. *]"} {
				if !strings.Contains(res.ModelRSL, want) {
					t.Fatalf("model.rsl missing %q", want)
				}
			}
			for _, want := range []string{"package ", "Requirement ", "id         = \"REQ-"} {
				if !strings.Contains(res.RequirementsTRLC, want) {
					t.Fatalf("requirements.trlc missing %q", want)
				}
			}
		})
	}
}

func TestEngTRLC_CLIAndValidation(t *testing.T) {
	reqPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	outDir := filepath.Join(t.TempDir(), "trlc")

	cmd := exec.Command("go", "run", "./cmd/engtrlc", "--requirements", reqPath, "--out-dir", outDir, "--package", "PaymentsReqs")
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
