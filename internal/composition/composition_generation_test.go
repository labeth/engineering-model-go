// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// A generator composes once: without subsystems there is nothing to resolve,
// and a caller's composition result is reused instead of resolving again
// (the bundle below has no manifest, so a fresh resolution would fail).
// TRLC-LINKS: REQ-EMG-066
func TestComposeForGenerationReusesGivenResult(t *testing.T) {
	plain := model.Bundle{}
	if got, err := ComposeForGeneration(plain, nil, ResolutionOptions{}); err != nil || got.Root != nil || got.Workspace != "" {
		t.Fatalf("model without composition: got %+v, %v", got, err)
	}

	composed := model.Bundle{}
	composed.Architecture.Composition.Subsystems = []model.Subsystem{{ID: "SUB-A"}}
	given := CompositionResult{Workspace: "/work"}
	got, err := ComposeForGeneration(composed, &given, ResolutionOptions{})
	if err != nil || got.Workspace != "/work" {
		t.Fatalf("given result not reused: got %+v, %v", got, err)
	}
}
