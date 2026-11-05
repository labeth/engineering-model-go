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
	// FieldType is the YAML type of one field of an entity kind
	// (ValidationRules.FieldTypes).
	FieldType = validate.FieldType
)

// Diagnostic severities.
const (
	SeverityError   = validate.SeverityError
	SeverityWarning = validate.SeverityWarning
)

// ValidateBundle runs every model validation rule over a loaded bundle and
// returns the diagnostics with the severities of the model's life-cycle tier
// (ApplyTierPolicy; unchanged without lifecycle). It is validate.Bundle,
// renamed because Bundle is the model type at the package root.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-125
func ValidateBundle(b Bundle) []Diagnostic {
	return validate.ApplyTierPolicy(b, validate.Bundle(b))
}

// TierRule is one tier-keyed code of the engine tier policy: its severity
// (deferred, warning, error) at each tier poc..operational, with the safety
// bundle, and the tier it is required from.
type TierRule = validate.TierRule

// TierPolicy returns the engine tier policy table, sorted by code.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
// TRLC-LINKS: REQ-EMG-125
func TierPolicy() []TierRule {
	return validate.TierPolicy()
}

// TierPolicyMarkdown renders TierPolicy as the "Severity by tier" Markdown
// table (D deferred, W warning, E error).
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
// TRLC-LINKS: REQ-EMG-125
func TierPolicyMarkdown() string {
	return validate.TierPolicyMarkdown()
}

// ApplyTierPolicy returns diags with the severities of the model's
// life-cycle tier: tier-keyed codes per TierPolicy, POA&M defers, and
// lifecycle.raise; rank-0 core codes and models without lifecycle are
// unchanged. It is idempotent; a wrapper with its own diagnostics applies it
// to its merged list.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-129
func ApplyTierPolicy(b Bundle, diags []Diagnostic) []Diagnostic {
	return validate.ApplyTierPolicy(b, diags)
}

// EffectiveTier returns the model's life-cycle tier and its rank 1..5; ok
// is false without lifecycle.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-124
func EffectiveTier(b Bundle) (string, int, bool) {
	return validate.EffectiveTier(b)
}

// EffectiveRequirementLinkMode returns the requirement link strictness in
// force: requirementLinks, raised to strict from tier mvp.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-127
func EffectiveRequirementLinkMode(b Bundle) string {
	return validate.EffectiveRequirementLinkMode(b)
}

// CoreCodes returns the rank-0 code patterns: never keyed by tier, never
// deferred.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-125
func CoreCodes() []string {
	return validate.CoreCodes()
}

// IsCoreCode reports whether a code is a rank-0 core code.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-125
func IsCoreCode(code string) bool {
	return validate.IsCoreCode(code)
}

// TierKeyed reports whether the tier policy sets a code's severity.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-125
func TierKeyed(code string) bool {
	return validate.TierKeyed(code)
}

// RendersDespite reports whether an error with the code lets the AsciiDoc
// document render (D15 extended).
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-130
func RendersDespite(b Bundle, code string) bool {
	return validate.RendersDespite(b, code)
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
// TRLC-LINKS: REQ-EMG-045, REQ-EMG-009, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-087
func AuthoringRules() ValidationRules {
	return validate.AuthoringRules()
}

// ExcludeAcceptedForms names the inferenceHints.exclude forms the code scan
// applies.
const ExcludeAcceptedForms = validate.ExcludeAcceptedForms

// ExcludePatternProblem returns why an inferenceHints.exclude pattern cannot
// be applied, or "" when it can.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-CODEMAP-INFERENCE
// TRLC-LINKS: REQ-EMG-114
func ExcludePatternProblem(pattern string) string {
	return validate.ExcludePatternProblem(pattern)
}
