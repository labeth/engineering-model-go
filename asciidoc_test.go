package engmodel

import (
	"os/exec"
	"path/filepath"
	"regexp"
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
	if !strings.Contains(res.Document, "= Sample Payments Architecture") {
		t.Fatalf("missing document title")
	}
	if !strings.Contains(res.Document, "=== VIEW-CONTEXT") {
		t.Fatalf("missing context view section")
	}
	if !strings.Contains(res.Document, "== Design Chapters") {
		t.Fatalf("missing design chapters section")
	}
	if !strings.Contains(res.Document, "== Requirements Appendix") {
		t.Fatalf("missing requirements appendix chapter")
	}
	if !strings.Contains(res.Document, "== Reference Index") {
		t.Fatalf("missing reference index chapter")
	}
	if !strings.Contains(res.Document, "[[EVT-PAYMENT-AUTH-REQUESTED]]") {
		t.Fatalf("missing expected glossary anchor for EVT-PAYMENT-AUTH-REQUESTED")
	}
	if !strings.Contains(res.Document, "==== Chapter Scope Diagram") {
		t.Fatalf("missing chapter scope diagram section")
	}
	if !strings.Contains(res.Document, "* Derived Architecture Refs:") {
		t.Fatalf("missing derived c4 refs in reference map")
	}
	if !strings.Contains(res.Document, "<<REQ-PAY-004,REQ-PAY-004>>") {
		t.Fatalf("expected requirement cross reference to REQ-PAY-004")
	}
}

func TestC4AdocCLI_EndToEnd(t *testing.T) {
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
	if !strings.Contains(text, "= Sample Payments Architecture") {
		t.Fatalf("cli output missing asciidoc title")
	}
	if !strings.Contains(text, "==== Chapter Scope Diagram") {
		t.Fatalf("cli output missing chapter scope diagram section")
	}
	if !strings.Contains(text, "==== Reference Map") {
		t.Fatalf("cli output missing reference map section")
	}
	if !strings.Contains(text, "* Derived Requirement Refs:") {
		t.Fatalf("cli output missing derived requirement refs")
	}
}

func TestGenerateAsciiDocWithCodeMapping(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")
	codeRoot := filepath.Join("examples", "payments-engineering-sample", "src")

	res, err := GenerateAsciiDocFromFiles(modelPath, requirementsPath, designPath, AsciiDocOptions{
		CodeRoot: codeRoot,
	})
	if err != nil {
		t.Fatalf("generate asciidoc with code mapping failed: %v", err)
	}
	if validate.HasErrors(res.Diagnostics) {
		t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
	}
	if !strings.Contains(res.Document, "== Code Mapping") {
		t.Fatalf("missing code mapping chapter")
	}
	if !strings.Contains(res.Document, "CODE-STARTSESSION") {
		t.Fatalf("missing expected auto-generated symbol CODE-STARTSESSION")
	}
	if !strings.Contains(res.Document, "CONT-CHECKOUT-API") {
		t.Fatalf("missing expected mapping CONT-CHECKOUT-API")
	}
}

func TestGenerateAsciiDoc_NoDeadInternalReferences(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")
	codeRoot := filepath.Join("examples", "payments-engineering-sample", "src")

	res, err := GenerateAsciiDocFromFiles(modelPath, requirementsPath, designPath, AsciiDocOptions{
		CodeRoot: codeRoot,
	})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if validate.HasErrors(res.Diagnostics) {
		t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
	}

	anchorRe := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	xrefRe := regexp.MustCompile(`<<([^>,]+)(?:,[^>]+)?>`)
	anchors := map[string]bool{}
	for _, m := range anchorRe.FindAllStringSubmatch(res.Document, -1) {
		anchors[m[1]] = true
	}
	for _, m := range xrefRe.FindAllStringSubmatch(res.Document, -1) {
		if !anchors[m[1]] {
			t.Fatalf("missing anchor for xref target %q", m[1])
		}
	}
}

func TestGenerateAsciiDoc_NoListToAnchorAdjacency(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")
	codeRoot := filepath.Join("examples", "payments-engineering-sample", "src")

	res, err := GenerateAsciiDocFromFiles(modelPath, requirementsPath, designPath, AsciiDocOptions{
		CodeRoot: codeRoot,
	})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if validate.HasErrors(res.Diagnostics) {
		t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
	}

	lines := strings.Split(res.Document, "\n")
	for i := 0; i < len(lines)-1; i++ {
		if strings.HasPrefix(lines[i], "* ") && strings.HasPrefix(lines[i+1], "[[") {
			t.Fatalf("list item directly followed by anchor at line %d", i+1)
		}
	}
}

func TestC4AdocCLI_WithCodeRoot(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	requirementsPath := filepath.Join("examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("examples", "payments-engineering-sample", "design.yml")
	codeRoot := filepath.Join("examples", "payments-engineering-sample", "src")

	cmd := exec.Command("go", "run", "./cmd/engdoc",
		"--model", modelPath,
		"--requirements", requirementsPath,
		"--design", designPath,
		"--code-root", codeRoot,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("engdoc cli with code-root failed: %v\noutput:\n%s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "== Code Mapping") {
		t.Fatalf("cli output missing code mapping chapter")
	}
	if !strings.Contains(text, "CODE-PERSISTAUDITRECORD") {
		t.Fatalf("cli output missing expected traced symbol")
	}
}

func TestGenerateAsciiDoc_IDInputUsesNameLabelWithIDTarget(t *testing.T) {
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
	if len(design.Design.Chapters) == 0 {
		t.Fatalf("expected at least one chapter in design fixture")
	}
	design.Design.Chapters[0].Narrative = "When EVT-PAYMENT-AUTH-REQUESTED is raised by ACT-CUSTOMER, the system responds."

	res, err := GenerateAsciiDoc(bundle, requirements, design, AsciiDocOptions{})
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	if validate.HasErrors(res.Diagnostics) {
		t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
	}
	if !strings.Contains(res.Document, "<<EVT-PAYMENT-AUTH-REQUESTED,payment authorization is requested>>") {
		t.Fatalf("expected ID-target + name-label xref for EVT-PAYMENT-AUTH-REQUESTED")
	}
	if !strings.Contains(res.Document, "<<ACT-CUSTOMER,customer>>") {
		t.Fatalf("expected ID-target + name-label xref for ACT-CUSTOMER")
	}
	if strings.Contains(res.Document, "<<EVT-PAYMENT-AUTH-REQUESTED,EVT-PAYMENT-AUTH-REQUESTED>>") {
		t.Fatalf("did not expect ID label for EVT-PAYMENT-AUTH-REQUESTED")
	}
	if strings.Contains(res.Document, "<<ACT-CUSTOMER,ACT-CUSTOMER>>") {
		t.Fatalf("did not expect ID label for ACT-CUSTOMER")
	}
}

func TestGenerateAsciiDoc_EARSStrictFailureBlocksGeneration(t *testing.T) {
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

	requirements.Requirements[0].Text = "The door control system lock the door."
	requirements.LintRun.Mode = "strict"

	res, err := GenerateAsciiDoc(bundle, requirements, design, AsciiDocOptions{})
	if err == nil {
		t.Fatalf("expected generation error for strict EARS failure")
	}
	if !validate.HasErrors(res.Diagnostics) {
		t.Fatalf("expected error diagnostics, got: %+v", res.Diagnostics)
	}
	if !hasDiagCodeAndPathPrefix(res.Diagnostics, "ears.missing_shall", "requirements[REQ-PAY-001]") {
		t.Fatalf("expected ears.missing_shall diagnostic for REQ-PAY-001, got: %+v", res.Diagnostics)
	}
}

func TestGenerateAsciiDoc_EARSGuidedDoesNotBlock(t *testing.T) {
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

	requirements.Requirements[0].Text = "The door control system lock the door."
	requirements.LintRun.Mode = "guided"

	res, err := GenerateAsciiDoc(bundle, requirements, design, AsciiDocOptions{})
	if err != nil {
		t.Fatalf("did not expect generation error in guided mode, got: %v", err)
	}
	if validate.HasErrors(res.Diagnostics) {
		t.Fatalf("did not expect error diagnostics in guided mode, got: %+v", res.Diagnostics)
	}
	if !hasDiagCodeAndPathPrefix(res.Diagnostics, "ears.missing_shall", "requirements[REQ-PAY-001]") {
		t.Fatalf("expected guided ears.missing_shall diagnostic for REQ-PAY-001, got: %+v", res.Diagnostics)
	}
}

func hasDiagCodeAndPathPrefix(diags []validate.Diagnostic, code, pathPrefix string) bool {
	for _, d := range diags {
		if d.Code == code && strings.HasPrefix(d.Path, pathPrefix) {
			return true
		}
	}
	return false
}
