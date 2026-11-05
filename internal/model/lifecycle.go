// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import "strings"

// Life-cycle tiers of engmod.yml lifecycle.tier, in ascending rank (1..5).
const (
	TierPoC         = "poc"
	TierPrototype   = "prototype"
	TierMVP         = "mvp"
	TierProduct     = "product"
	TierOperational = "operational"
)

// Assurance bundles of engmod.yml lifecycle.bundles.
const (
	BundleFoundation = "foundation"
	BundleSafety     = "safety"
	BundleSecurity   = "security"
)

// Tiers lists the life-cycle tiers in ascending rank.
var Tiers = []string{TierPoC, TierPrototype, TierMVP, TierProduct, TierOperational}

// Bundles lists the assurance bundles.
var Bundles = []string{BundleFoundation, BundleSafety, BundleSecurity}

// ManifestLifecycle is engmod.yml lifecycle: the project tier (the bar for
// the default branch and every release), the assurance bundles (default
// foundation), the accepted DEC-TIER-* decision that set the tier, and rule
// codes raised to error regardless of tier.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type ManifestLifecycle struct {
	Tier         string   `yaml:"tier"`
	Bundles      []string `yaml:"bundles,omitempty"`
	TierDecision string   `yaml:"tierDecision,omitempty"`
	Raise        []string `yaml:"raise,omitempty"`
}

// TierRank returns the rank 1..5 of a tier, compared case-insensitively, or
// 0 for an unknown tier.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-MODEL-LOADER
func TierRank(tier string) int {
	tier = strings.ToLower(strings.TrimSpace(tier))
	for i, t := range Tiers {
		if t == tier {
			return i + 1
		}
	}
	return 0
}

// EffectiveBundles returns the bundles of a lifecycle: the declared ones,
// lowercased, with foundation always included; [foundation] when none.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-MODEL-LOADER
func (l ManifestLifecycle) EffectiveBundles() []string {
	out := []string{BundleFoundation}
	for _, b := range l.Bundles {
		b = strings.ToLower(strings.TrimSpace(b))
		if b == "" || containsString(out, b) {
			continue
		}
		out = append(out, b)
	}
	return out
}

// HasBundle reports whether the lifecycle includes a bundle.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-MODEL-LOADER
func (l ManifestLifecycle) HasBundle(bundle string) bool {
	return containsString(l.EffectiveBundles(), strings.ToLower(strings.TrimSpace(bundle)))
}
