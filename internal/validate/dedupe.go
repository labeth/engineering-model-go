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
	type key struct{ code, path, message string }
	index := map[key]int{}
	var out []Diagnostic
	for _, list := range lists {
		for _, d := range list {
			k := key{d.Code, d.Path, d.Message}
			if i, ok := index[k]; ok {
				if d.Severity == SeverityError {
					out[i].Severity = SeverityError
				}
				continue
			}
			index[k] = len(out)
			out = append(out, d)
		}
	}
	return out
}
