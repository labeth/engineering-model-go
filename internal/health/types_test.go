// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import "testing"

// The two highest tiers share a strictly tighter limit set than the defaults,
// every other tier gets the defaults, and a caller cannot change the shared
// tables through the returned map.
// Mutation: returning DefaultThresholds for product, or a strict limit that is
// not below its default, fails this test.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestDefaultThresholdsForTier(t *testing.T) {
	defaults := DefaultThresholds()
	for _, tier := range []string{"", "poc", "prototype", "mvp"} {
		got := DefaultThresholdsForTier(tier)
		if len(got) != len(defaults) {
			t.Fatalf("tier %q: %d limits, want the %d defaults", tier, len(got), len(defaults))
		}
		for code, want := range defaults {
			if got[code] != want {
				t.Errorf("tier %q %s = %+v, want the default %+v", tier, code, got[code], want)
			}
		}
	}
	for _, tier := range []string{"product", "operational"} {
		got := DefaultThresholdsForTier(tier)
		if len(got) != len(defaults) {
			t.Fatalf("tier %q: %d limits, want %d", tier, len(got), len(defaults))
		}
		for code, base := range defaults {
			strict := got[code]
			// bumpy_road and complex_conditional keep the default limits
			// (ADR-EMG-059): at 1 and 2 they produced a quarter of all findings.
			same := code == CodeBumpyRoad || code == CodeComplexConditional
			switch {
			case same && strict != base:
				t.Errorf("tier %q %s = %+v, want the default %+v", tier, code, strict, base)
			case !same && (strict.Warn >= base.Warn || strict.High >= base.High):
				t.Errorf("tier %q %s = %+v is not stricter than %+v", tier, code, strict, base)
			}
			if strict.Warn < 1 || strict.High < strict.Warn {
				t.Errorf("tier %q %s = %+v is not a valid limit", tier, code, strict)
			}
		}
	}
	a := DefaultThresholdsForTier("operational")
	a[CodeComplexMethod] = Threshold{Warn: 999, High: 999}
	if b := DefaultThresholdsForTier("operational"); b[CodeComplexMethod].Warn == 999 {
		t.Error("the returned map is shared between calls")
	}
	if len(Codes()) != 10 {
		t.Errorf("Codes() lists %d codes, want 10", len(Codes()))
	}
}

// The strict limits of product and operational are pinned with literals, one
// per value, from the table of ADR-EMG-059, and the two tiers are equal. The
// test above compares the strict set with the defaults; this one fails when
// either number of any code drifts, so a limit cannot be moved by accident.
//
// Mutation: changing any one number of DefaultThresholdsForTier fails.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestStrictLimitsArePinnedToADR059(t *testing.T) {
	want := map[string]Threshold{
		"health.complex_method":        {Warn: 8, High: 15},
		"health.large_method":          {Warn: 50, High: 100},
		"health.deep_nesting":          {Warn: 3, High: 4},
		"health.bumpy_road":            {Warn: 2, High: 3},
		"health.complex_conditional":   {Warn: 3, High: 5},
		"health.excess_arguments":      {Warn: 4, High: 6},
		"health.large_assertion_block": {Warn: 5, High: 8},
	}
	product, operational := DefaultThresholdsForTier("product"), DefaultThresholdsForTier("operational")
	for _, got := range []map[string]Threshold{product, operational} {
		if len(got) != len(want) {
			t.Fatalf("%d limits, want %d: %+v", len(got), len(want), got)
		}
		for code, limit := range want {
			if got[code] != limit {
				t.Errorf("%s = %+v, want %+v", code, got[code], limit)
			}
		}
	}
	for code, limit := range product {
		if operational[code] != limit {
			t.Errorf("%s: product %+v differs from operational %+v", code, limit, operational[code])
		}
	}
	// The defaults of the lower tiers are pinned the same way (ADR-EMG-047).
	defaults := map[string]Threshold{
		"health.complex_method":        {Warn: 10, High: 20},
		"health.large_method":          {Warn: 70, High: 120},
		"health.deep_nesting":          {Warn: 4, High: 5},
		"health.bumpy_road":            {Warn: 2, High: 3},
		"health.complex_conditional":   {Warn: 3, High: 5},
		"health.excess_arguments":      {Warn: 5, High: 7},
		"health.large_assertion_block": {Warn: 6, High: 10},
	}
	for _, tier := range []string{"", "poc", "prototype", "mvp"} {
		got := DefaultThresholdsForTier(tier)
		for code, limit := range defaults {
			if got[code] != limit {
				t.Errorf("tier %q %s = %+v, want %+v", tier, code, got[code], limit)
			}
		}
	}
}
