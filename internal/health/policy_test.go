// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"reflect"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// thresholdsOf is a health block with the given thresholds.
func thresholdsOf(t map[string]model.HealthThreshold) *model.ManifestHealth {
	return &model.ManifestHealth{Thresholds: t}
}

// aspects lists the aspects of the changes in order.
func aspects(changes []PolicyChange) []string {
	out := []string{}
	for _, c := range changes {
		out = append(out, c.Aspect)
	}
	return out
}

// PolicyWeakenings finds each way a change loosens the policy: a lower tier, a
// limit above the base's effective limit (the tier default for a code the
// manifest does not set), a new exclude pattern and an override that raises a
// limit; it finds nothing for a stricter or equal policy, for an override that
// is no looser than the base's for the same path, or for a threshold that only
// differs because the tier does.
//
// Mutation: comparing only warn (not high), comparing against the head's tier
// default instead of the base's, or ignoring the override or the exclude case
// fails the matching row.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func TestPolicyWeakenings(t *testing.T) {
	cm := model.HealthThreshold{Warn: 8, High: 15} // the product default of complex_method
	cases := []struct {
		name       string
		base, head Policy
		want       []string
	}{
		{"identical", Policy{Tier: "product"}, Policy{Tier: "product"}, []string{}},
		{"tier lowered", Policy{Tier: "product"}, Policy{Tier: "mvp"}, []string{PolicyTier}},
		{"tier removed", Policy{Tier: "product"}, Policy{}, []string{PolicyTier}},
		{"tier raised", Policy{Tier: "mvp"}, Policy{Tier: "product"}, []string{}},
		{"warn raised over the tier default", Policy{Tier: "product"},
			Policy{Tier: "product", Health: thresholdsOf(map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 9, High: 15}})}, []string{PolicyThreshold}},
		{"high raised only", Policy{Tier: "product"},
			Policy{Tier: "product", Health: thresholdsOf(map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 8, High: 16}})}, []string{PolicyThreshold}},
		{"equal to the default", Policy{Tier: "product"},
			Policy{Tier: "product", Health: thresholdsOf(map[string]model.HealthThreshold{CodeComplexMethod: cm})}, []string{}},
		{"stricter than the default", Policy{Tier: "product"},
			Policy{Tier: "product", Health: thresholdsOf(map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 5, High: 10}})}, []string{}},
		{"base raised, head the same", Policy{Tier: "product", Health: thresholdsOf(map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 12, High: 20}})},
			Policy{Tier: "product", Health: thresholdsOf(map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 12, High: 20}})}, []string{}},
		{"base raised, head removed it", Policy{Tier: "product", Health: thresholdsOf(map[string]model.HealthThreshold{CodeComplexMethod: {Warn: 5, High: 10}})},
			Policy{Tier: "product"}, []string{PolicyThreshold}},
		{"default of a lower tier is the tier alone", Policy{Tier: "product"}, Policy{Tier: "mvp"}, []string{PolicyTier}},
		{"new exclude", Policy{Tier: "product", Health: &model.ManifestHealth{Exclude: []string{"gen/**"}}},
			Policy{Tier: "product", Health: &model.ManifestHealth{Exclude: []string{"./gen/**", "internal/big/**"}}}, []string{PolicyExclude}},
		{"exclude removed", Policy{Tier: "product", Health: &model.ManifestHealth{Exclude: []string{"gen/**"}}}, Policy{Tier: "product"}, []string{}},
		{"override raises over the module limit", Policy{Tier: "product"},
			Policy{Tier: "product", Health: &model.ManifestHealth{Overrides: []model.HealthOverride{{Path: "legacy/**", Thresholds: map[string]model.HealthThreshold{CodeLargeMethod: {Warn: 200, High: 400}}}}}}, []string{PolicyOverride}},
		{"override stricter", Policy{Tier: "product"},
			Policy{Tier: "product", Health: &model.ManifestHealth{Overrides: []model.HealthOverride{{Path: "legacy/**", Thresholds: map[string]model.HealthThreshold{CodeLargeMethod: {Warn: 30, High: 60}}}}}}, []string{}},
		{"override unchanged from the base", Policy{Tier: "product", Health: &model.ManifestHealth{Overrides: []model.HealthOverride{{Path: "legacy/**", Thresholds: map[string]model.HealthThreshold{CodeLargeMethod: {Warn: 200, High: 400}}}}}},
			Policy{Tier: "product", Health: &model.ManifestHealth{Overrides: []model.HealthOverride{{Path: "./legacy/**", Thresholds: map[string]model.HealthThreshold{CodeLargeMethod: {Warn: 200, High: 400}}}}}}, []string{}},
		{"override of the same path raised further", Policy{Tier: "product", Health: &model.ManifestHealth{Overrides: []model.HealthOverride{{Path: "legacy/**", Thresholds: map[string]model.HealthThreshold{CodeLargeMethod: {Warn: 200, High: 400}}}}}},
			Policy{Tier: "product", Health: &model.ManifestHealth{Overrides: []model.HealthOverride{{Path: "legacy/**", Thresholds: map[string]model.HealthThreshold{CodeLargeMethod: {Warn: 200, High: 500}}}}}}, []string{PolicyOverride}},
		{"everything at once", Policy{Tier: "operational"},
			Policy{Tier: "mvp", Health: &model.ManifestHealth{Exclude: []string{"x/**"}, Thresholds: map[string]model.HealthThreshold{CodeBumpyRoad: {Warn: 5, High: 9}}}},
			[]string{PolicyExclude, PolicyThreshold, PolicyTier}},
	}
	for _, c := range cases {
		got := PolicyWeakenings(c.base, c.head)
		if !reflect.DeepEqual(aspects(got), c.want) {
			t.Errorf("%s: aspects = %v (%+v), want %v", c.name, aspects(got), got, c.want)
		}
		for _, change := range got {
			if strings.TrimSpace(change.Message) == "" {
				t.Errorf("%s: a change without a message: %+v", c.name, change)
			}
		}
	}
}

// PolicyFromManifest reads the tier of a manifest of another revision (an
// unknown tier is none) and a tier override replaces it, as it does for the head.
//
// TRLC-LINKS: REQ-EMG-155
func TestPolicyFromManifest(t *testing.T) {
	if got := PolicyFromManifest(model.HealthPolicy{Lifecycle: &model.ManifestLifecycle{Tier: "Product"}}, "").Tier; got != "product" {
		t.Errorf("tier = %q", got)
	}
	if got := PolicyFromManifest(model.HealthPolicy{Lifecycle: &model.ManifestLifecycle{Tier: "bogus"}}, "").Tier; got != "" {
		t.Errorf("an unknown tier = %q, want none", got)
	}
	if got := PolicyFromManifest(model.HealthPolicy{Lifecycle: &model.ManifestLifecycle{Tier: "mvp"}}, "product").Tier; got != "product" {
		t.Errorf("override = %q", got)
	}
	if got := PolicyFromManifest(model.HealthPolicy{}, "").Tier; got != "" {
		t.Errorf("no lifecycle = %q", got)
	}
}

// The level of health.policy_weakened is the stricter of the head's and the
// base's: lowering the tier from product to mvp, or dropping the lifecycle, must
// not make its own weakening a warning. Row WWWEE, with safety WWEEE.
//
// Mutation: evaluating only the head bundle fails the lowered-tier rows.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func TestPolicyLevelIsTheStricterOfBaseAndHead(t *testing.T) {
	life := func(tier string, bundles ...string) *model.ManifestLifecycle {
		return &model.ManifestLifecycle{Tier: tier, Bundles: bundles}
	}
	cases := []struct {
		name string
		head model.Bundle
		base *model.ManifestLifecycle
		want string
	}{
		{"both product", bundleAt("product", nil, nil), life("product"), LevelError},
		{"both mvp", bundleAt("mvp", nil, nil), life("mvp"), LevelWarning},
		{"lowered from product to mvp", bundleAt("mvp", nil, nil), life("product"), LevelError},
		{"lowered from operational to poc", bundleAt("poc", nil, nil), life("operational"), LevelError},
		{"lifecycle removed", model.Bundle{}, life("product"), LevelError},
		{"no base lifecycle, head mvp", bundleAt("mvp", nil, nil), nil, LevelWarning},
		{"no base lifecycle, head product", bundleAt("product", nil, nil), nil, LevelError},
		{"mvp with safety at the head", bundleAt("mvp", []string{"safety"}, nil), life("mvp"), LevelError},
		{"safety dropped at the head", bundleAt("mvp", nil, nil), life("mvp", "safety"), LevelError},
		{"raised by lifecycle.raise", bundleAt("poc", nil, []string{"health.*"}), life("poc"), LevelError},
	}
	for _, c := range cases {
		if got := PolicyLevelOf(c.head, c.base); got != c.want {
			t.Errorf("%s: level = %s, want %s", c.name, got, c.want)
		}
	}
}
