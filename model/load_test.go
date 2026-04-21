package model

import (
	"path/filepath"
	"testing"
)

func TestLoadBundle(t *testing.T) {
	p := filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml")
	b, err := LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	if b.Architecture.Model.ID != "sample-payments-layered-model" {
		t.Fatalf("unexpected model id: %q", b.Architecture.Model.ID)
	}
	if len(b.Architecture.Views) != 7 {
		t.Fatalf("expected 7 views, got %d", len(b.Architecture.Views))
	}
}
