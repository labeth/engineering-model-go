package engmodel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

func TestNormalizeTerraformKind_AWSLambdaFunction(t *testing.T) {
	got := normalizeTerraformKind("aws_lambda_function")
	if got != "lambda_function" {
		t.Fatalf("expected lambda_function, got %q", got)
	}
}

func TestResolveRuntimeOwner_ByFunctionalUnitSimilarity(t *testing.T) {
	units := []model.FunctionalUnit{
		{ID: "FU-GITHUB-WEBHOOK-INGRESS", Name: "GitHub Webhook Ingress"},
		{ID: "FU-REVIEW-ORCHESTRATION", Name: "Review Orchestration"},
		{ID: "FU-REVIEW-PUBLICATION", Name: "Review Publication"},
	}

	item := inferredRuntimeItem{
		Name:  "review_orchestrator",
		Kind:  "lambda_function",
		Owner: "unresolved",
	}
	got := resolveRuntimeOwner(item, units)
	if got != "FU-REVIEW-ORCHESTRATION" {
		t.Fatalf("expected FU-REVIEW-ORCHESTRATION, got %q", got)
	}
}

func TestResolveRuntimeOwner_PublisherMapsToPublication(t *testing.T) {
	units := []model.FunctionalUnit{
		{ID: "FU-REVIEW-ORCHESTRATION", Name: "Review Orchestration"},
		{ID: "FU-REVIEW-PUBLICATION", Name: "Review Publication"},
	}

	item := inferredRuntimeItem{
		Name:  "review_publisher",
		Kind:  "lambda_function",
		Owner: "unresolved",
	}
	got := resolveRuntimeOwner(item, units)
	if got != "FU-REVIEW-PUBLICATION" {
		t.Fatalf("expected FU-REVIEW-PUBLICATION, got %q", got)
	}
}

func TestParseTerraformRuntime_DescriptionHintComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.tf")
	content := `
	# engmodel:runtime-description: handles inbound webhook validation and request normalization
	resource "aws_lambda_function" "webhook_ingress" {
	  function_name = "pr-review-webhook-ingress"
	}
	`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write tf fixture: %v", err)
	}

	items, diags := parseTerraformRuntime(path)
	if len(diags) > 0 {
		t.Fatalf("unexpected terraform parse diagnostics: %+v", diags)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 runtime item, got %d", len(items))
	}
	if items[0].Description != "handles inbound webhook validation and request normalization" {
		t.Fatalf("unexpected runtime description: %q", items[0].Description)
	}
}

func TestParseManifestRuntime_AnnotationDescription(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runtime.yaml")
	content := `
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: checkout-api
  namespace: payments
  annotations:
    engmodel.dev/owner-unit: FU-CHECKOUT
    engmodel.dev/runtime-description: serves checkout ingress traffic and forwards normalized payment requests
spec:
  values:
    service:
      port: 8080
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest fixture: %v", err)
	}

	items, diags := parseManifestRuntime(path)
	if len(diags) > 0 {
		t.Fatalf("unexpected manifest parse diagnostics: %+v", diags)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 runtime item, got %d", len(items))
	}
	if items[0].Description != "serves checkout ingress traffic and forwards normalized payment requests" {
		t.Fatalf("unexpected runtime description: %q", items[0].Description)
	}
}
