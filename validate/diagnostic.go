// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import "sort"

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Path     string   `json:"path,omitempty"`
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func SortDiagnostics(in []Diagnostic) []Diagnostic {
	out := append([]Diagnostic(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		a := out[i]
		b := out[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Message != b.Message {
			return a.Message < b.Message
		}
		return string(a.Severity) < string(b.Severity)
	})
	if len(out) < 2 {
		return out
	}
	unique := out[:0]
	var prev Diagnostic
	for i, d := range out {
		if i > 0 && d.Code == prev.Code && d.Severity == prev.Severity && d.Message == prev.Message && d.Path == prev.Path {
			continue
		}
		unique = append(unique, d)
		prev = d
	}
	return unique
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func HasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}
