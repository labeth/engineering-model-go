// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"reflect"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// With no health block the limits are the tier defaults: the standard set
// below product and the strict set at product and operational.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
func TestConfigForBaseIsTheTierDefault(t *testing.T) {
	for _, tier := range []string{"", "poc", "mvp", "product", "operational"} {
		got := ConfigFor(tier, nil, "a.go")
		if !reflect.DeepEqual(got.Thresholds, DefaultThresholdsForTier(tier)) {
			t.Errorf("tier %q: %+v, want the tier defaults", tier, got.Thresholds)
		}
	}
	if ConfigFor("product", nil, "a.go").Thresholds[CodeComplexMethod] != (Threshold{Warn: 8, High: 15}) {
		t.Error("product does not use the strict complex_method limit")
	}
}

// Manifest thresholds replace both limits of a code over the tier base, and a
// code they do not name keeps the base.
//
// TRLC-LINKS: REQ-EMG-153
func TestConfigForManifestThresholdsReplacePerCode(t *testing.T) {
	h := &model.ManifestHealth{Thresholds: map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 15, High: 25}}}
	got := ConfigFor("operational", h, "a.go").Thresholds
	if got[CodeComplexMethod] != (Threshold{Warn: 15, High: 25}) {
		t.Errorf("complex_method = %+v, want the manifest 15/25", got[CodeComplexMethod])
	}
	if got[CodeLargeMethod] != (Threshold{Warn: 50, High: 100}) {
		t.Errorf("large_method = %+v, want the operational base 50/100", got[CodeLargeMethod])
	}
}

// The first override whose path matches replaces per code over the manifest
// thresholds; a later matching override is ignored and a file that matches
// none keeps the manifest thresholds.
//
// Mutation: applying every matching override, or the last one, fails.
//
// TRLC-LINKS: REQ-EMG-153
func TestConfigForFirstMatchingOverrideWins(t *testing.T) {
	h := &model.ManifestHealth{
		Thresholds: map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 15, High: 25}},
		Overrides: []model.HealthOverride{
			{Path: "internal/legacy/**", Thresholds: map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 30, High: 60}}},
			{Path: "internal/**", Thresholds: map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 1, High: 2}, CodeLargeMethod: {Warn: 500, High: 900}}},
			{Path: "*.gen.go", Thresholds: map[string]model.HealthThreshold{CodeBumpyRoad: {Warn: 9, High: 9}}},
		},
	}
	legacy := ConfigFor("", h, "internal/legacy/old.go").Thresholds
	if legacy[CodeComplexMethod] != (Threshold{Warn: 30, High: 60}) || legacy[CodeLargeMethod] != DefaultThresholds()[CodeLargeMethod] {
		t.Errorf("legacy file: %+v, want the first override only", legacy)
	}
	other := ConfigFor("", h, "internal/x/new.go").Thresholds
	if other[CodeComplexMethod] != (Threshold{Warn: 1, High: 2}) || other[CodeLargeMethod] != (Threshold{Warn: 500, High: 900}) {
		t.Errorf("internal file: %+v, want the second override", other)
	}
	if none := ConfigFor("", h, "cmd/x/main.go").Thresholds; none[CodeComplexMethod] != (Threshold{Warn: 15, High: 25}) {
		t.Errorf("unmatched file: %+v, want the manifest threshold", none)
	}
	if gen := ConfigFor("", h, "a.gen.go").Thresholds; gen[CodeBumpyRoad] != (Threshold{Warn: 9, High: 9}) {
		t.Errorf("glob override: %+v", gen)
	}
}

// ConfigFor returns a map the caller owns: changing it does not change the
// defaults.
//
// TRLC-LINKS: REQ-EMG-153
func TestConfigForReturnsPrivateMap(t *testing.T) {
	ConfigFor("", nil, "a.go").Thresholds[CodeComplexMethod] = Threshold{Warn: 1, High: 1}
	if DefaultThresholds()[CodeComplexMethod] != (Threshold{Warn: 10, High: 20}) {
		t.Fatal("the shared defaults were changed")
	}
}

// The exclusion dialect is the one of inferenceHints.exclude: patterns are
// anchored at the module root, * stays within one segment, dir and dir/** leave
// a whole directory out, ** is only valid as a trailing /**, and a pattern the
// dialect refuses matches nothing. The default excludes apply without a block,
// and testdata/ is not excluded.
//
// TRLC-LINKS: REQ-EMG-153
func TestExcludedUsesTheExcludeDialect(t *testing.T) {
	h := &model.ManifestHealth{Exclude: []string{"*.gen.go", "third_party", "docs/*", "**/bad", "./legacy/**"}}
	cases := map[string]bool{
		"generated/a.go":          true,
		"vendor/x/y.go":           true,
		"node_modules/p/i.js":     true,
		".engmod/cache/a.go":      true,
		"internal/generated/a.go": false, // anchored: only the top-level generated/
		"testdata/fixture.go":     false,
		"main.go":                 false,
		"a.gen.go":                true,
		"pkg/a.gen.go":            false, // * does not cross a slash and the pattern is anchored
		"third_party/lib/a.go":    true,
		"docs/api/x.go":           true,
		"x/bad/a.go":              false, // **/bad is refused and matches nothing
		"legacy/old/a.go":         true,
		"legacy.go":               false,
	}
	for p, want := range cases {
		if got := Excluded(h, p); got != want {
			t.Errorf("Excluded(%q) = %v, want %v", p, got, want)
		}
	}
	if Excluded(nil, "generated/a.go") != true || Excluded(nil, "a.gen.go") != false {
		t.Error("a nil health block must still apply the default excludes only")
	}
	if PathMatches("../up", "a.go") || PathMatches("/abs", "abs") || PathMatches("", "a.go") {
		t.Error("a refused pattern must match nothing")
	}
}

// bundleAt builds a bundle with a lifecycle tier for the level tests.
func bundleAt(tier string, bundles []string, raise []string) model.Bundle {
	var b model.Bundle
	b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier, Bundles: bundles, Raise: raise}
	return b
}

// Levels come from the tier policy of internal/validate: warnings up to mvp,
// errors at product and operational (row WWWEE), errors from mvp with the
// safety bundle (WWEEE), errors at any tier for a code that lifecycle.raise
// names, and warnings for a model without a lifecycle tier. The unused waiver
// follows the debt row.
//
// Mutation: copying the rows instead of asking the policy, or ignoring the
// bundle or raise, fails.
//
// TRLC-LINKS: REQ-EMG-155
func TestLevelsFollowTheTierPolicy(t *testing.T) {
	cases := []struct {
		name  string
		b     model.Bundle
		error bool
	}{
		{"no lifecycle", model.Bundle{}, false},
		{"poc", bundleAt("poc", nil, nil), false},
		{"prototype", bundleAt("prototype", nil, nil), false},
		{"mvp", bundleAt("mvp", nil, nil), false},
		{"product", bundleAt("product", nil, nil), true},
		{"operational", bundleAt("operational", nil, nil), true},
		{"mvp safety", bundleAt("mvp", []string{"safety"}, nil), true},
		{"prototype safety", bundleAt("prototype", []string{"safety"}, nil), false},
		{"poc raised", bundleAt("poc", nil, []string{"health.*"}), true},
	}
	for _, c := range cases {
		levels := Levels(c.b)
		want := LevelWarning
		if c.error {
			want = LevelError
		}
		for _, code := range append(Codes(), model.HealthWaiverUnusedCode) {
			if levels[code] != want {
				t.Errorf("%s: level of %s = %q, want %q", c.name, code, levels[code], want)
			}
		}
	}
	raised := Levels(bundleAt("poc", nil, []string{"health.complex_method"}))
	if raised[CodeComplexMethod] != LevelError || raised[CodeLargeMethod] != LevelWarning {
		t.Errorf("raise of one code: %v", raised)
	}
}

// WithTier replaces the tier and keeps the other lifecycle fields without
// changing the bundle it was given.
//
// TRLC-LINKS: REQ-EMG-155
func TestWithTierKeepsTheRestAndTheOriginal(t *testing.T) {
	orig := bundleAt("mvp", []string{"safety"}, []string{"health.*"})
	got := WithTier(orig, "operational")
	if got.Manifest.Lifecycle.Tier != "operational" || !got.Manifest.Lifecycle.HasBundle("safety") || len(got.Manifest.Lifecycle.Raise) != 1 {
		t.Errorf("got %+v", got.Manifest.Lifecycle)
	}
	if orig.Manifest.Lifecycle.Tier != "mvp" {
		t.Error("WithTier changed the original bundle")
	}
	if fresh := WithTier(model.Bundle{}, "product"); fresh.Manifest.Lifecycle == nil || Levels(fresh)[CodeBumpyRoad] != LevelError {
		t.Error("a bundle without lifecycle did not get the tier")
	}
}

// Relevel gives the diagnostics of a file that was not analysed the level of
// health.regression at the new tier, and the summary counts them: an imported
// or older report read at product fails where it read at poc only warned.
//
// Mutation: leaving the diagnostics at the level they were written with fails.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-157
func TestRelevelMovesFileDiagnosticsWithTheTier(t *testing.T) {
	r := Report{Levels: map[string]string{}, Diagnostics: []Diagnostic{
		{Code: CodeFileSkipped, Level: LevelWarning, Path: "a.go"},
		{Code: CodeFileFailed, Level: LevelWarning, Path: "b.go"},
		{Code: model.HealthWaiverUnjustifiedCode, Level: LevelError, Path: "model/b.json"},
	}}
	product := Relevel(r, bundleAt("product", nil, nil))
	if product.Diagnostics[0].Level != LevelError || product.Diagnostics[1].Level != LevelError || product.Summary.Errors != 3 {
		t.Errorf("product: %+v %+v", product.Diagnostics, product.Summary)
	}
	poc := Relevel(product, bundleAt("poc", nil, nil))
	if poc.Diagnostics[0].Level != LevelWarning || poc.Diagnostics[1].Level != LevelWarning || poc.Diagnostics[2].Level != LevelError || poc.Summary.Errors != 1 || poc.Summary.Warnings != 2 {
		t.Errorf("poc: %+v %+v", poc.Diagnostics, poc.Summary)
	}
}
