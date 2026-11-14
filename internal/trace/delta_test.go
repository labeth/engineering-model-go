// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import (
	"reflect"
	"testing"
)

func row(id, status string, code, ver int) Row {
	r := Row{ID: id, Status: status}
	for i := 0; i < code; i++ {
		r.Code = append(r.Code, CodeRef{Path: "a.go", Line: i + 1})
	}
	for i := 0; i < ver; i++ {
		r.Verifications = append(r.Verifications, "VER-X")
	}
	return r
}

// TRLC-LINKS: REQ-EMG-191
func TestDeltaOfGainedLostOrphansAndStatus(t *testing.T) {
	from := Matrix{
		Requirements: []Row{row("R1", "orphan", 0, 0), row("R2", "implemented", 1, 0), row("R3", "implemented", 1, 1), row("R4", "implemented", 1, 0), row("R5", "orphan", 0, 0)},
		Orphans:      []string{"R1", "R5"},
	}
	to := Matrix{
		Requirements: []Row{row("R1", "implemented", 1, 0), row("R2", "implemented", 1, 1), row("R3", "orphan", 0, 0), row("R4", "verified", 1, 0), row("R5", "orphan", 0, 0), row("R6", "orphan", 0, 0), row("R7", "implemented", 2, 1)},
		Orphans:      []string{"R3", "R5", "R6"},
	}
	d := DeltaOf(from, to)
	check := func(name string, got, want []string) {
		t.Helper()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	check("gained", d.Gained, []string{"R1", "R7"})
	check("testsGained", d.TestsGained, []string{"R2", "R7"})
	check("lost", d.Lost, []string{"R3"})
	check("newOrphans", d.NewOrphans, []string{"R3", "R6"})
	check("resolvedOrphans", d.ResolvedOrphans, []string{"R1"})
	if want := []StatusChange{{"R1", "orphan", "implemented"}, {"R3", "implemented", "orphan"}, {"R4", "implemented", "verified"}}; !reflect.DeepEqual(d.StatusChanged, want) {
		t.Errorf("statusChanged = %+v, want %+v", d.StatusChanged, want)
	}
}

// A requirement that was removed is the entity diff's to report; it is neither a lost trace nor an orphan.
// TRLC-LINKS: REQ-EMG-191
func TestDeltaOfIgnoresRemovedRequirementsAndEncodesEmptyListsAsArrays(t *testing.T) {
	d := DeltaOf(Matrix{Requirements: []Row{row("GONE", "implemented", 1, 1)}}, Matrix{})
	if len(d.Lost)+len(d.Gained)+len(d.NewOrphans) != 0 {
		t.Fatalf("a removed requirement shows in the trace delta: %+v", d)
	}
	if d.Gained == nil || d.Lost == nil || d.TestsGained == nil || d.NewOrphans == nil || d.ResolvedOrphans == nil || d.StatusChanged == nil {
		t.Fatal("empty lists must be non-nil so they encode as []")
	}
}
