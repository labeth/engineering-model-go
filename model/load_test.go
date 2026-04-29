// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-001
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

func TestLoadBundle_Decisions(t *testing.T) {
	p := filepath.Join("..", "architecture.yml")
	b, err := LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	if len(b.Architecture.Decisions) == 0 {
		t.Fatalf("expected root model decisions")
	}
	if filepath.Base(b.DecisionsPath) != "decisions.yml" {
		t.Fatalf("unexpected decisions path: %q", b.DecisionsPath)
	}
	if len(b.Decisions.Decisions) != len(b.Architecture.Decisions) {
		t.Fatalf("expected decisions document and architecture decisions to match")
	}
	d := b.Architecture.Decisions[0]
	if d.ID != "ADR-EMG-001" {
		t.Fatalf("unexpected decision id: %q", d.ID)
	}
	if d.Status != "accepted" {
		t.Fatalf("unexpected decision status: %q", d.Status)
	}
	if len(d.Consequences) == 0 {
		t.Fatalf("expected decision consequences")
	}
}
