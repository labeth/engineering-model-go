// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import (
	"reflect"
	"testing"
)

// rowSpec is a requirement row: its status and how many code references and verifications it has.
type rowSpec struct {
	id, status string
	code, ver  int
}

func rows(specs ...rowSpec) []Row {
	out := []Row{}
	for _, s := range specs {
		r := Row{ID: s.id, Status: s.status}
		for i := 0; i < s.code; i++ {
			r.Code = append(r.Code, CodeRef{Path: "a.go", Line: i + 1})
		}
		for i := 0; i < s.ver; i++ {
			r.Verifications = append(r.Verifications, "VER-X")
		}
		out = append(out, r)
	}
	return out
}

// listCase is a list of requirement IDs the delta reported and the one it should have.
type listCase struct {
	name      string
	got, want []string
}

func sameList(t *testing.T, c listCase) {
	t.Helper()
	if !reflect.DeepEqual(c.got, c.want) {
		t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
	}
}

func sampleDelta() Delta {
	from := Matrix{
		Requirements: rows(rowSpec{"R1", "orphan", 0, 0}, rowSpec{"R2", "implemented", 1, 0}, rowSpec{"R3", "implemented", 1, 1}, rowSpec{"R4", "implemented", 1, 0}, rowSpec{"R5", "orphan", 0, 0}),
		Orphans:      []string{"R1", "R5"},
	}
	to := Matrix{
		Requirements: rows(rowSpec{"R1", "implemented", 1, 0}, rowSpec{"R2", "implemented", 1, 1}, rowSpec{"R3", "orphan", 0, 0}, rowSpec{"R4", "verified", 1, 0}, rowSpec{"R5", "orphan", 0, 0}, rowSpec{"R6", "orphan", 0, 0}, rowSpec{"R7", "implemented", 2, 1}),
		Orphans:      []string{"R3", "R5", "R6"},
	}
	return DeltaOf(from, to)
}

// TRLC-LINKS: REQ-EMG-191
func TestDeltaOfGainedAndLostTrace(t *testing.T) {
	d := sampleDelta()
	sameList(t, listCase{"gained", d.Gained, []string{"R1", "R7"}})
	sameList(t, listCase{"testsGained", d.TestsGained, []string{"R2", "R7"}})
	sameList(t, listCase{"lost", d.Lost, []string{"R3"}})
}

// TRLC-LINKS: REQ-EMG-191
func TestDeltaOfOrphans(t *testing.T) {
	d := sampleDelta()
	sameList(t, listCase{"newOrphans", d.NewOrphans, []string{"R3", "R6"}})
	sameList(t, listCase{"resolvedOrphans", d.ResolvedOrphans, []string{"R1"}})
}

// TRLC-LINKS: REQ-EMG-191
func TestDeltaOfStatusChanges(t *testing.T) {
	want := []StatusChange{{"R1", "orphan", "implemented"}, {"R3", "implemented", "orphan"}, {"R4", "implemented", "verified"}}
	if got := sampleDelta().StatusChanged; !reflect.DeepEqual(got, want) {
		t.Errorf("statusChanged = %+v, want %+v", got, want)
	}
}

// A requirement that was removed is the entity diff's to report; it is neither a lost trace nor an orphan.
// TRLC-LINKS: REQ-EMG-191
func TestDeltaOfIgnoresRemovedRequirements(t *testing.T) {
	d := DeltaOf(Matrix{Requirements: rows(rowSpec{"GONE", "implemented", 1, 1})}, Matrix{})
	total := len(d.Lost) + len(d.Gained) + len(d.NewOrphans) + len(d.StatusChanged)
	if total != 0 {
		t.Fatalf("a removed requirement shows in the trace delta: %+v", d)
	}
}

// Empty lists must be non-nil so they encode as [].
// TRLC-LINKS: REQ-EMG-191
func TestDeltaOfEncodesEmptyListsAsArrays(t *testing.T) {
	d := DeltaOf(Matrix{}, Matrix{})
	for name, nonNil := range map[string]bool{"gained": d.Gained != nil, "lost": d.Lost != nil, "testsGained": d.TestsGained != nil, "newOrphans": d.NewOrphans != nil, "resolvedOrphans": d.ResolvedOrphans != nil, "statusChanged": d.StatusChanged != nil} {
		if !nonNil {
			t.Errorf("%s is nil", name)
		}
	}
}
