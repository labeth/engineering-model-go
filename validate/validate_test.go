package validate

import (
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

func TestBundleValidationNoErrors(t *testing.T) {
	p := filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml")
	b, err := model.LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	diags := Bundle(b)
	if HasErrors(diags) {
		t.Fatalf("expected no validation errors, got: %+v", diags)
	}
}

func TestViewIDIsFreeButKindIsStrict(t *testing.T) {
	p := filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml")
	b, err := model.LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	if len(b.Architecture.Views) == 0 {
		t.Fatalf("expected sample views")
	}

	// Free-form IDs should be accepted as long as kind is supported.
	b.Architecture.Views[0].ID = "run"
	b.Architecture.Views[0].Kind = "communication"
	diags := Bundle(b)
	if HasErrors(diags) {
		t.Fatalf("expected no errors for free-form view id with valid kind, got: %+v", diags)
	}

	// Unsupported kind should fail validation regardless of ID.
	b.Architecture.Views[0].Kind = "run"
	diags = Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected errors for unsupported view kind")
	}
	found := false
	for _, d := range diags {
		if d.Code == "model.unknown_view_kind" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected model.unknown_view_kind diagnostic, got: %+v", diags)
	}
}
