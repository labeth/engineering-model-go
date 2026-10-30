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
	if got, err := ComposeForGeneration(plain, nil, ResolutionOptions{}); err != nil || len(got.Allocations) != 0 {
		t.Fatalf("model without composition: got %+v, %v", got, err)
	}

	composed := model.Bundle{}
	composed.Architecture.Composition.Subsystems = []model.Subsystem{{ID: "SUB-A"}}
	given := CompositionResult{Allocations: []MaterializedAllocation{{Requirement: "REQ-A"}}}
	got, err := ComposeForGeneration(composed, &given, ResolutionOptions{})
	if err != nil || len(got.Allocations) != 1 || got.Allocations[0].Requirement != "REQ-A" {
		t.Fatalf("given result not reused: got %+v, %v", got, err)
	}
}

// TRLC-LINKS: REQ-EMG-020, REQ-EMG-066
func TestDelegationsByRequirementIndexesTrimmedIDs(t *testing.T) {
	got := DelegationsByRequirement(CompositionResult{Allocations: []MaterializedAllocation{
		{Requirement: " REQ-A ", TargetRef: "x"}, {Requirement: "REQ-A", TargetRef: "y"}, {Requirement: "REQ-B"},
	}})
	if len(got) != 2 || len(got["REQ-A"]) != 2 || len(got["REQ-B"]) != 1 {
		t.Fatalf("delegations by requirement: %+v", got)
	}
}
