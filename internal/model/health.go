// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

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
	// HealthBaselineStaleCode is the warning for a baseline entry that matches
	// no finding (ADR-EMG-059); the health report emits it.
	HealthBaselineStaleCode = "health.baseline_stale"
	// HealthBaselineUnreadableCode is the untiered error for a baseline file
	// that is missing, not valid JSON or not engmod-health-baseline/1.
	HealthBaselineUnreadableCode = "health.baseline_unreadable"
	// HealthRegressionCode is the delta code: a new or worsened high finding.
	HealthRegressionCode = "health.regression"
	// HealthPolicyWeakenedCode is the delta code for a change that weakens the
	// health policy (a higher limit, a wider exclude or override, a lower tier)
	// against the base revision without a policyDecision that records it.
	HealthPolicyWeakenedCode = "health.policy_weakened"
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

// HealthPolicy is the part of a manifest that decides how code is judged: the
// life-cycle block (tier, bundles, raise) and the health block. The delta
// check reads it from the base revision to see whether a change weakens it.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
type HealthPolicy struct {
	Lifecycle *ManifestLifecycle `yaml:"lifecycle"`
	Health    *ManifestHealth    `yaml:"health"`
}

// ParseHealthPolicy reads the lifecycle and health blocks of the manifest
// source of another revision. It is deliberately lenient: every other key is
// ignored and the CUE contract of the current schema is not applied, so a
// manifest written for an older schema still yields its policy. Source that is
// not a YAML mapping is an error.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
func ParseHealthPolicy(source []byte) (HealthPolicy, error) {
	var policy HealthPolicy
	if err := yaml.NewDecoder(bytes.NewReader(source)).Decode(&policy); err != nil && !errors.Is(err, io.EOF) {
		return HealthPolicy{}, fmt.Errorf("read the health policy of the manifest: %w", err)
	}
	return policy, nil
}
