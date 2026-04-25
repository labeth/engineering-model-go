package engmodel

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/validate"
)

func TestGenerateStructurizrDSLFromFile_Examples(t *testing.T) {
	examples := []string{
		filepath.Join("examples", "payments-engineering-sample", "architecture.yml"),
		filepath.Join("examples", "bedrock-pr-review-github-app-sample", "architecture.yml"),
		filepath.Join("examples", "coffee-fleet-ota-cloud-sample", "architecture.yml"),
	}
	for _, modelPath := range examples {
		modelPath := modelPath
		t.Run(filepath.Base(filepath.Dir(modelPath)), func(t *testing.T) {
			res, err := GenerateStructurizrDSLFromFile(modelPath)
			if err != nil {
				t.Fatalf("generate structurizr dsl failed: %v", err)
			}
			if validate.HasErrors(res.Diagnostics) {
				t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
			}
			for _, want := range []string{"workspace ", "model {", "views {", "systemLandscape", "systemContext", "container ", " -> "} {
				if !strings.Contains(res.DSL, want) {
					t.Fatalf("expected generated dsl to contain %q", want)
				}
			}
		})
	}
}

func TestGenerateStructurizrDSL_CLI(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	cmd := exec.Command("go", "run", "./cmd/engstruct", "--model", modelPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("engstruct CLI failed: %v\noutput:\n%s", err, string(out))
	}
	s := string(out)
	for _, want := range []string{"workspace ", "model {", "views {"} {
		if !strings.Contains(s, want) {
			t.Fatalf("expected CLI output to contain %q", want)
		}
	}
}
