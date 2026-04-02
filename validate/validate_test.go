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
