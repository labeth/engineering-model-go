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
	if b.Architecture.Model.ID != "sample-payments-system" {
		t.Fatalf("unexpected model id: %q", b.Architecture.Model.ID)
	}
	if len(b.Architecture.Viewpoints) != 3 {
		t.Fatalf("expected 3 viewpoints, got %d", len(b.Architecture.Viewpoints))
	}
}
