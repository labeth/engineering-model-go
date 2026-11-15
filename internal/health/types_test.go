// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import "testing"

// The tiers below product get the default limits.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestDefaultThresholdsBelowProduct(t *testing.T) {
	defaults := DefaultThresholds()
	for _, tier := range []string{"", "poc", "prototype", "mvp"} {
		wantDeepEqual(t, DefaultThresholdsForTier(tier), defaults)
	}
}

// The two highest tiers share a strictly tighter limit set than the defaults,
// except for bumpy_road and complex_conditional, and every limit is valid.
// Mutation: returning DefaultThresholds for product, or a strict limit that is
// not below its default, fails this test.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestStrictTiersAreStricterThanTheDefaults(t *testing.T) {
	defaults := DefaultThresholds()
	for _, tier := range []string{"product", "operational"} {
		got := DefaultThresholdsForTier(tier)
		mustEqual(t, len(got), len(defaults))
		for code, base := range defaults {
			wantEqual(t, [3]string{tier, code, limitProblem(code, got[code], base)}, [3]string{tier, code, ""})
		}
	}
}

// limitProblem says what is wrong with the strict limit of a code, or "".
// bumpy_road and complex_conditional keep the default limits (ADR-EMG-059): at
// 1 and 2 they produced a quarter of all findings.
func limitProblem(code string, strict, base Threshold) string {
	if strict.Warn < 1 || strict.High < strict.Warn {
		return "not a valid limit"
	}
	if code == CodeBumpyRoad || code == CodeComplexConditional {
		return problemIf(strict != base, "not the default")
	}
	return problemIf(!isStricter(strict, base), "not stricter than the default")
}

// problemIf is the message when bad, else "".
func problemIf(bad bool, message string) string {
	if bad {
		return message
	}
	return ""
}

// isStricter reports a limit that is below the base in both numbers.
func isStricter(strict, base Threshold) bool {
	return strict.Warn < base.Warn && strict.High < base.High
}

// A caller cannot change the shared tables through the returned map, and there
// are ten codes.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestDefaultThresholdsAreNotShared(t *testing.T) {
	a := DefaultThresholdsForTier("operational")
	a[CodeComplexMethod] = Threshold{Warn: 999, High: 999}
	wantEqual(t, DefaultThresholdsForTier("operational")[CodeComplexMethod].Warn == 999, false)
	wantEqual(t, len(Codes()), 10)
}

// strictLimits are the limits of product and operational, from the table of
// ADR-EMG-059.
var strictLimits = map[string]Threshold{
	"health.complex_method":        {Warn: 8, High: 15},
	"health.large_method":          {Warn: 50, High: 100},
	"health.deep_nesting":          {Warn: 3, High: 4},
	"health.bumpy_road":            {Warn: 2, High: 3},
	"health.complex_conditional":   {Warn: 3, High: 5},
	"health.excess_arguments":      {Warn: 4, High: 6},
	"health.large_assertion_block": {Warn: 5, High: 8},
}

// lowerTierLimits are the defaults of the lower tiers (ADR-EMG-047).
var lowerTierLimits = map[string]Threshold{
	"health.complex_method":        {Warn: 10, High: 20},
	"health.large_method":          {Warn: 70, High: 120},
	"health.deep_nesting":          {Warn: 4, High: 5},
	"health.bumpy_road":            {Warn: 2, High: 3},
	"health.complex_conditional":   {Warn: 3, High: 5},
	"health.excess_arguments":      {Warn: 5, High: 7},
	"health.large_assertion_block": {Warn: 6, High: 10},
}

// The strict limits of product and operational are pinned with literals, one
// per value, from the table of ADR-EMG-059, and the two tiers are equal. The
// tests above compare the strict set with the defaults; this one fails when
// either number of any code drifts, so a limit cannot be moved by accident.
//
// Mutation: changing any one number of DefaultThresholdsForTier fails.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestStrictLimitsArePinnedToADR059(t *testing.T) {
	wantDeepEqual(t, DefaultThresholdsForTier("product"), strictLimits)
	wantDeepEqual(t, DefaultThresholdsForTier("operational"), strictLimits)
}

// The defaults of the lower tiers are pinned the same way.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestLowerTierLimitsArePinned(t *testing.T) {
	for _, tier := range []string{"", "poc", "prototype", "mvp"} {
		wantDeepEqual(t, DefaultThresholdsForTier(tier), lowerTierLimits)
	}
}
