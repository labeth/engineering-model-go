// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import "testing"

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func TestDedupeDiagnosticsReportsEachFindingOnce(t *testing.T) {
	orphan := Diagnostic{Code: "requirement.orphan", Severity: SeverityWarning, Message: "requirement REQ-X-001 is traced by no code, verification, or delegation", Path: "requirements"}
	other := Diagnostic{Code: "lint.multiple_responses", Severity: SeverityWarning, Message: "m", Path: "requirements[REQ-X-002]"}
	docDiags := []Diagnostic{orphan, other}
	traceDiags := []Diagnostic{orphan, {Code: "code.dangling_requirement_link", Severity: SeverityError, Message: "d", Path: "a.go:1"}}

	got := DedupeDiagnostics(docDiags, traceDiags)
	if len(got) != 3 {
		t.Fatalf("got %d diagnostics, want 3: %+v", len(got), got)
	}
	if got[0] != orphan || got[1] != other || got[2].Code != "code.dangling_requirement_link" {
		t.Fatalf("order not preserved: %+v", got)
	}
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func TestDedupeDiagnosticsKeepsErrorSeverity(t *testing.T) {
	warn := Diagnostic{Code: "c", Severity: SeverityWarning, Message: "m", Path: "p"}
	err := warn
	err.Severity = SeverityError
	got := DedupeDiagnostics([]Diagnostic{warn}, []Diagnostic{err})
	if len(got) != 1 || got[0].Severity != SeverityError {
		t.Fatalf("got %+v, want one error", got)
	}
	if !HasErrors(got) {
		t.Fatalf("deduplication must not hide an error")
	}
}
