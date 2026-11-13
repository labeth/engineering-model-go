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

// A delegation resolves to the one requirement of the subsystem that lists
// the target in realizes (ADR-EMG-075); a contract ref names nothing, and none
// or several realizers leave the delegation unresolved.
// Mutation: reading provides.ref again, taking the first of several
// realizers or not trimming the IDs fails a row.
// TRLC-LINKS: REQ-EMG-184
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestDelegationTargetIsTheRealizer(t *testing.T) {
	cases := []struct {
		name string
		reqs []model.Requirement
		want string
	}{
		{"one realizer", []model.Requirement{{ID: " REQ-A ", Realizes: []string{" CAP-X "}}, {ID: "REQ-B"}}, "REQ-A"},
		{"no realizer", []model.Requirement{{ID: "REQ-A"}, {ID: "REQ-B", Realizes: []string{"CAP-Y"}}}, ""},
		{"two realizers", []model.Requirement{{ID: "REQ-A", Realizes: []string{"CAP-X"}}, {ID: "REQ-B", Realizes: []string{"CAP-X"}}}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := delegationTarget(model.RequirementsDocument{Requirements: c.reqs}, "CAP-X")
			if got != c.want {
				t.Errorf("delegationTarget = %q, want %q", got, c.want)
			}
		})
	}
}
