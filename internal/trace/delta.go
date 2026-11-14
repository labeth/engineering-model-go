// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import "sort"

// StatusChange is a requirement whose trace status differs between two matrices.
type StatusChange struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

// Delta is how the traceability of the requirements that exist in a later matrix moved since an earlier one.
// A requirement that is traced when it has code or a verification linked to it.
type Delta struct {
	// Gained lists requirements that are traced now and were not (new requirements that arrive traced included).
	Gained []string `json:"gained"`
	// TestsGained lists requirements that now have a verification linked and had none.
	TestsGained []string `json:"testsGained"`
	// Lost lists requirements that existed before, were traced, and are not any more.
	Lost []string `json:"lost"`
	// NewOrphans lists requirements that are orphans now and were not before: new, or traced before.
	NewOrphans []string `json:"newOrphans"`
	// ResolvedOrphans lists requirements that were orphans and are traced now.
	ResolvedOrphans []string `json:"resolvedOrphans"`
	// StatusChanged lists requirements whose Status field differs.
	StatusChanged []StatusChange `json:"statusChanged"`
}

// TRLC-LINKS: REQ-EMG-191
func traced(r Row) bool { return len(r.Code) > 0 || len(r.Verifications) > 0 }

// DeltaOf compares two matrices of one model by requirement ID. Every list is sorted.
//
// TRLC-LINKS: REQ-EMG-191
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func DeltaOf(from, to Matrix) Delta {
	d := Delta{Gained: []string{}, TestsGained: []string{}, Lost: []string{}, NewOrphans: []string{}, ResolvedOrphans: []string{}, StatusChanged: []StatusChange{}}
	before := make(map[string]Row, len(from.Requirements))
	for _, r := range from.Requirements {
		before[r.ID] = r
	}
	orphanBefore := make(map[string]bool, len(from.Orphans))
	for _, id := range from.Orphans {
		orphanBefore[id] = true
	}
	orphanAfter := make(map[string]bool, len(to.Orphans))
	for _, id := range to.Orphans {
		orphanAfter[id] = true
	}
	for _, r := range to.Requirements {
		old, existed := before[r.ID]
		switch {
		case traced(r) && !(existed && traced(old)):
			d.Gained = append(d.Gained, r.ID)
		case existed && traced(old) && !traced(r):
			d.Lost = append(d.Lost, r.ID)
		}
		if len(r.Verifications) > 0 && !(existed && len(old.Verifications) > 0) {
			d.TestsGained = append(d.TestsGained, r.ID)
		}
		if existed && old.Status != r.Status {
			d.StatusChanged = append(d.StatusChanged, StatusChange{ID: r.ID, From: old.Status, To: r.Status})
		}
		if orphanAfter[r.ID] && !(existed && orphanBefore[r.ID]) {
			d.NewOrphans = append(d.NewOrphans, r.ID)
		}
		if existed && orphanBefore[r.ID] && !orphanAfter[r.ID] {
			d.ResolvedOrphans = append(d.ResolvedOrphans, r.ID)
		}
	}
	for _, list := range [][]string{d.Gained, d.TestsGained, d.Lost, d.NewOrphans, d.ResolvedOrphans} {
		sort.Strings(list)
	}
	sort.Slice(d.StatusChanged, func(i, j int) bool { return d.StatusChanged[i].ID < d.StatusChanged[j].ID })
	return d
}
