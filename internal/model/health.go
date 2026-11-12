// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

// Code health diagnostic codes (ADR-EMG-051). The health report itself lives
// in internal/health; the manifest block that configures it is validated in
// internal/validate, which must not import the report package, so the code
// vocabulary the configuration refers to is declared here as string literals.
const (
	// HealthWaiverUnjustifiedCode is the error for a waiver whose decision
	// does not exist or is not accepted.
	HealthWaiverUnjustifiedCode = "health.waiver_unjustified"
	// HealthWaiverUnusedCode is the warning for a waiver that matches no
	// finding; the health report emits it.
	HealthWaiverUnusedCode = "health.waiver_unused"
	// HealthRegressionCode is the delta code: a new or worsened high finding.
	HealthRegressionCode = "health.regression"
)

// HealthThresholdCodes returns the smell codes with numeric limits, the only
// valid keys of health.thresholds and of health.overrides[].thresholds.
// TRLC-LINKS: REQ-EMG-153
func HealthThresholdCodes() []string {
	return []string{
		"health.complex_method", "health.large_method", "health.deep_nesting", "health.bumpy_road",
		"health.complex_conditional", "health.excess_arguments", "health.large_assertion_block",
	}
}

// HealthSmellCodes returns every code health smell (debt) code: the ones with
// numeric limits and the structural ones. A waiver names one of these.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154
func HealthSmellCodes() []string {
	return []string{
		"health.complex_method", "health.large_method", "health.deep_nesting", "health.bumpy_road",
		"health.complex_conditional", "health.excess_arguments", "health.primitive_obsession",
		"health.brain_method", "health.large_assertion_block", "health.duplicated_assertion_block",
	}
}

// IsHealthThresholdCode reports whether a code takes numeric limits.
// TRLC-LINKS: REQ-EMG-153
func IsHealthThresholdCode(code string) bool {
	return containsString(HealthThresholdCodes(), code)
}

// IsHealthSmellCode reports whether a code is a health smell code.
// TRLC-LINKS: REQ-EMG-154
func IsHealthSmellCode(code string) bool {
	return containsString(HealthSmellCodes(), code)
}
