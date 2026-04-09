package engmodel

import (
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
