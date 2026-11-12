// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"sort"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// sortedKeys returns the keys of a map, sorted.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sameSet reports a difference between two code lists, ignoring order.
func sameSet(t *testing.T, what string, got, want []string) {
	t.Helper()
	g, w := append([]string(nil), got...), append([]string(nil), want...)
	sort.Strings(g)
	sort.Strings(w)
	if len(g) != len(w) {
		t.Errorf("%s: %v, want %v", what, g, w)
		return
	}
	for i := range g {
		if g[i] != w[i] {
			t.Errorf("%s: %v, want %v", what, g, w)
			return
		}
	}
}

// The health code vocabulary lives in several places: Codes() of this package,
// HealthSmellCodes() and HealthThresholdCodes() of internal/model (the manifest
// and waiver validation), the weights table of the scoring, the rule descriptions
// of the SARIF export, the default thresholds and the levels. One test keeps them
// together, so adding a smell to one place and not to the others fails here
// (Q9 of the second review pass).
// Mutation: adding a code to Codes() only fails the model, weights and SARIF rows.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-151, REQ-EMG-153, REQ-EMG-157
func TestHealthCodeVocabularyIsOneList(t *testing.T) {
	smells := Codes()
	sameSet(t, "model.HealthSmellCodes", model.HealthSmellCodes(), smells)
	sameSet(t, "the weights table", sortedKeys(Weights()), smells)

	analysis := []string{CodeFileSkipped, CodeFileFailed, CodeSyntaxError, CodeRollupUnavailable}
	rules := append(append(append([]string(nil), smells...), analysis...), model.HealthWaiverUnusedCode, model.HealthBaselineStaleCode)
	sameSet(t, "the SARIF rule descriptions", sortedKeys(ruleDescriptions), rules)

	thresholds := model.HealthThresholdCodes()
	sameSet(t, "model.HealthThresholdCodes vs DefaultThresholds", thresholds, sortedKeys(DefaultThresholds()))
	sameSet(t, "model.HealthThresholdCodes vs the product thresholds", thresholds, sortedKeys(DefaultThresholdsForTier("product")))
	for _, code := range thresholds {
		if !model.IsHealthSmellCode(code) {
			t.Errorf("threshold code %s is not a smell code", code)
		}
	}

	// Every code has a level at a tier: the smells and the waiver and baseline
	// codes through the tier policy, the analysis codes through health.regression.
	levels := Levels(model.Bundle{})
	sameSet(t, "the levels", sortedKeys(levels), rules)
}
