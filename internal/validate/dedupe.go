// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

// DedupeDiagnostics merges diagnostic lists produced by separate checks of one
// run (for example document generation and the trace check, which both run the
// trace-integrity check) so each finding is reported once. Diagnostics are
// identified by code, path and message; the first occurrence keeps its
// position, and an error severity on any duplicate wins over a warning.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func DedupeDiagnostics(lists ...[]Diagnostic) []Diagnostic {
	merged := diagnosticMerge{index: map[diagnosticIdentity]int{}}
	for _, list := range lists {
		for _, d := range list {
			merged.add(d)
		}
	}
	return merged.out
}

// diagnosticIdentity identifies a diagnostic by code, path and message.
type diagnosticIdentity struct{ code, path, message string }

// diagnosticMerge is the merged list with the position of each identity.
type diagnosticMerge struct {
	index map[diagnosticIdentity]int
	out   []Diagnostic
}

// add appends a diagnostic with a new identity; a duplicate only raises the
// severity of the first occurrence to error.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func (m *diagnosticMerge) add(d Diagnostic) {
	k := diagnosticIdentity{d.Code, d.Path, d.Message}
	if i, ok := m.index[k]; ok {
		if d.Severity == SeverityError {
			m.out[i].Severity = SeverityError
		}
		return
	}
	m.index[k] = len(m.out)
	m.out = append(m.out, d)
}
