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

// side is how one requirement stood in one matrix; a requirement that is not in the matrix is not present.
type side struct {
	row     Row
	present bool
	orphan  bool
}

// TRLC-LINKS: REQ-EMG-191
func (s side) traced() bool { return s.present && traced(s.row) }

// TRLC-LINKS: REQ-EMG-191
func (s side) tested() bool { return s.present && len(s.row.Verifications) > 0 }

// TRLC-LINKS: REQ-EMG-191
func (s side) orphaned() bool { return s.present && s.orphan }

// TRLC-LINKS: REQ-EMG-191
func sidesOf(m Matrix) map[string]side {
	out := make(map[string]side, len(m.Requirements))
	for _, r := range m.Requirements {
		out[r.ID] = side{row: r, present: true}
	}
	for _, id := range m.Orphans {
		if s, ok := out[id]; ok {
			s.orphan = true
			out[id] = s
		}
	}
	return out
}

// DeltaOf compares two matrices of one model by requirement ID. Every list is sorted.
//
// TRLC-LINKS: REQ-EMG-191
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func DeltaOf(from, to Matrix) Delta {
	d := Delta{Gained: []string{}, TestsGained: []string{}, Lost: []string{}, NewOrphans: []string{}, ResolvedOrphans: []string{}, StatusChanged: []StatusChange{}}
	before, after := sidesOf(from), sidesOf(to)
	for _, r := range to.Requirements {
		d.recordTrace(before[r.ID], after[r.ID])
		d.recordOrphans(before[r.ID], after[r.ID])
	}
	for _, list := range [][]string{d.Gained, d.TestsGained, d.Lost, d.NewOrphans, d.ResolvedOrphans} {
		sort.Strings(list)
	}
	sort.Slice(d.StatusChanged, func(i, j int) bool { return d.StatusChanged[i].ID < d.StatusChanged[j].ID })
	return d
}

// recordTrace notes a requirement that gained or lost code or a verification, gained a test, or changed status.
// TRLC-LINKS: REQ-EMG-191
func (d *Delta) recordTrace(old, now side) {
	id := now.row.ID
	switch {
	case now.traced() && !old.traced():
		d.Gained = append(d.Gained, id)
	case old.traced() && !now.traced():
		d.Lost = append(d.Lost, id)
	}
	if now.tested() && !old.tested() {
		d.TestsGained = append(d.TestsGained, id)
	}
	d.recordStatus(old, now)
}

// TRLC-LINKS: REQ-EMG-191
func (d *Delta) recordStatus(old, now side) {
	if old.present && old.row.Status != now.row.Status {
		d.StatusChanged = append(d.StatusChanged, StatusChange{ID: now.row.ID, From: old.row.Status, To: now.row.Status})
	}
}

// recordOrphans notes a requirement that became an orphan or stopped being one.
// TRLC-LINKS: REQ-EMG-191
func (d *Delta) recordOrphans(old, now side) {
	if now.orphaned() && !old.orphaned() {
		d.NewOrphans = append(d.NewOrphans, now.row.ID)
	}
	if old.orphaned() && !now.orphaned() {
		d.ResolvedOrphans = append(d.ResolvedOrphans, now.row.ID)
	}
}
