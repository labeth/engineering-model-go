package engmodel

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

func TestGenerateAsciiDocFromFiles_EndToEnd(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")

	res, err := GenerateAsciiDocFromFiles(modelPath, requirementsPath, designPath, AsciiDocOptions{})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if validate.HasErrors(res.Diagnostics) {
		t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
	}
	if !strings.Contains(res.Document, "= Sample Payments Layered Design") {
		t.Fatalf("missing document title")
	}
	if !strings.Contains(res.Document, "[source,mermaid]") {
		t.Fatalf("missing mermaid blocks")
	}
	if strings.Contains(res.Document, "MERMAID:") || strings.Contains(res.Document, "INF:") {
		t.Fatalf("did not expect helper markers in final document")
	}
	if !strings.Contains(res.Document, "=== Functional Groups (View Scoped)") {
		t.Fatalf("missing view-scoped functional groups section")
	}
	if !strings.Contains(res.Document, "=== Functional Units (View Scoped)") {
		t.Fatalf("missing view-scoped functional units section")
	}
}

func TestEngdocCLI_EndToEnd(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")

	cmd := exec.Command("go", "run", "./cmd/engdoc",
		"--model", modelPath,
		"--requirements", requirementsPath,
		"--design", designPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("engdoc cli failed: %v\noutput:\n%s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "== Functional View") {
		t.Fatalf("cli output missing functional view chapter")
	}
	if !strings.Contains(text, "== Deployment View") {
		t.Fatalf("cli output missing deployment view chapter")
	}
}

func TestGenerateAsciiDoc_EARSLintStrictFailure(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")

	bundle, err := model.LoadBundle(modelPath)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		t.Fatalf("load requirements failed: %v", err)
	}
	design, err := model.LoadDesign(designPath)
	if err != nil {
		t.Fatalf("load design failed: %v", err)
	}

	requirements.LintRun.Mode = "strict"
	requirements.Requirements[0].Text = "The door control system lock the door."

	res, err := GenerateAsciiDoc(bundle, requirements, design, AsciiDocOptions{})
	if err == nil {
		t.Fatalf("expected strict EARS lint failure")
	}
	if !validate.HasErrors(res.Diagnostics) {
		t.Fatalf("expected error diagnostics, got: %+v", res.Diagnostics)
	}
}
