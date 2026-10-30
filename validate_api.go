// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel

import "github.com/labeth/engineering-model-go/internal/validate"

// Validation types.
type (
	// Diagnostic is one validation finding with a stable code, severity,
	// message and model path.
	Diagnostic = validate.Diagnostic
	// Severity ranks a Diagnostic.
	Severity = validate.Severity
	// ValidationRules is the machine-readable subset of the validation rules
	// (controlled vocabularies, relationship kind pairs, field rules).
	ValidationRules = validate.Rules
	// RelationshipPair is one allowed mapping type with its end kinds.
	RelationshipPair = validate.RelationshipPair
)

// Diagnostic severities.
const (
	SeverityError   = validate.SeverityError
	SeverityWarning = validate.SeverityWarning
)

// ValidateBundle runs every model validation rule over a loaded bundle and
// returns the diagnostics. It is validate.Bundle, renamed because Bundle is
// the model type at the package root.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065
func ValidateBundle(b Bundle) []Diagnostic {
	return validate.Bundle(b)
}

// HasErrors reports whether any diagnostic has SeverityError.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func HasErrors(diags []Diagnostic) bool {
	return validate.HasErrors(diags)
}

// SortDiagnostics returns a copy of in in stable path, severity, code and
// message order.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-012
func SortDiagnostics(in []Diagnostic) []Diagnostic {
	return validate.SortDiagnostics(in)
}

// DedupeDiagnostics merges diagnostic lists from separate checks of one run
// so each finding (code, path, message) is reported once; an error severity
// on a duplicate wins over a warning.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func DedupeDiagnostics(lists ...[]Diagnostic) []Diagnostic {
	return validate.DedupeDiagnostics(lists...)
}

// AuthoringRules returns the validation rules an author needs before editing
// YAML, derived from the same tables ValidateBundle uses.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
// TRLC-LINKS: REQ-EMG-045, REQ-EMG-009
func AuthoringRules() ValidationRules {
	return validate.AuthoringRules()
}
