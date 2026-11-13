// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

// The comparison of the health policy with the policy of the base revision. A
// change decides the limits it is measured by: it can raise a threshold, widen
// health.exclude or an override, or lower the tier, and the weakened
// configuration then applies to both sides of a delta. enghealth check reads
// the manifest of the base revision and reports every weakening as
// health.policy_weakened, unless the head records it with a policyDecision (an
// accepted decision that the base did not already name), which is reviewable
// because decisions.yml is code-owned.

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// The aspects of a policy weakening.
const (
	PolicyTier      = "tier"
	PolicyThreshold = "threshold"
	PolicyExclude   = "exclude"
	PolicyOverride  = "override"
)

// PolicyChange is one way in which the head policy is weaker than the base.
type PolicyChange struct {
	// Aspect is tier, threshold, exclude or override.
	Aspect  string `json:"aspect"`
	Message string `json:"message"`
}

// Policy is the part of a manifest that decides how code is judged.
type Policy struct {
	// Tier is the life-cycle tier ("" without one).
	Tier string
	// Health is the health block; nil without one.
	Health *model.ManifestHealth
}

// PolicyOf returns the policy of a loaded model, with the tier it is checked
// at (the tier override of a run included).
//
// TRLC-LINKS: REQ-EMG-155
func PolicyOf(b model.Bundle) Policy {
	tier, _, _ := validate.EffectiveTier(b)
	return Policy{Tier: tier, Health: b.Manifest.Health}
}

// PolicyFromManifest returns the policy a manifest of another revision
// declares. tierOverride, when not empty, replaces its tier as it does for the
// head, so that a run at a fixed tier compares like with like.
//
// TRLC-LINKS: REQ-EMG-155
func PolicyFromManifest(p model.HealthPolicy, tierOverride string) Policy {
	tier := ""
	if p.Lifecycle != nil {
		if rank := model.TierRank(p.Lifecycle.Tier); rank > 0 {
			tier = model.Tiers[rank-1]
		}
	}
	if tierOverride != "" {
		tier = tierOverride
	}
	return Policy{Tier: tier, Health: p.Health}
}

// effectiveLimit is the limit of a code under a policy: the threshold of the
// health block, else the default of the tier.
// TRLC-LINKS: REQ-EMG-155
func (p Policy) effectiveLimit(code string) (Threshold, bool) {
	if p.Health != nil {
		if t, ok := p.Health.Thresholds[code]; ok {
			return Threshold{Warn: t.Warn, High: t.High}, true
		}
	}
	t, ok := DefaultThresholdsForTier(p.Tier)[code]
	return t, ok
}

// raises reports whether a limit is higher than another in warn or high.
// TRLC-LINKS: REQ-EMG-155
func raises(head, base Threshold) bool {
	return head.Warn > base.Warn || head.High > base.High
}

// String is the pair in a message.
//
// TRLC-LINKS: REQ-EMG-155
func (t Threshold) String() string { return fmt.Sprintf("warn %d, high %d", t.Warn, t.High) }

// PolicyWeakenings lists how head is weaker than base, sorted: a lower tier, a
// limit that is higher (warn or high above the effective limit of the base,
// which is the default of the tier for a code the manifest does not set), a
// new exclude pattern and an override that raises a limit. Limits that differ
// only because the tier differs are the lower tier and are not listed again.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func PolicyWeakenings(base, head Policy) []PolicyChange {
	changes := []PolicyChange{}
	if model.TierRank(head.Tier) < model.TierRank(base.Tier) {
		changes = append(changes, PolicyChange{Aspect: PolicyTier,
			Message: fmt.Sprintf("the tier is lowered from %s to %s", tierLabel(base.Tier), tierLabel(head.Tier))})
	}
	changes = append(changes, thresholdWeakenings(base, head)...)
	changes = append(changes, excludeWeakenings(base, head)...)
	changes = append(changes, overrideWeakenings(base, head)...)
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].Aspect != changes[j].Aspect {
			return changes[i].Aspect < changes[j].Aspect
		}
		return changes[i].Message < changes[j].Message
	})
	return changes
}

// tierLabel is the tier for a message.
// TRLC-LINKS: REQ-EMG-155
func tierLabel(tier string) string {
	if tier == "" {
		return "none"
	}
	return tier
}

// thresholdWeakenings lists the codes whose effective limit is higher at head,
// for a code that either manifest sets.
//
// TRLC-LINKS: REQ-EMG-155
func thresholdWeakenings(base, head Policy) []PolicyChange {
	var out []PolicyChange
	for _, code := range model.HealthThresholdCodes() {
		if !setsThreshold(base, code) && !setsThreshold(head, code) {
			continue
		}
		b, bok := base.effectiveLimit(code)
		h, hok := head.effectiveLimit(code)
		if bok && hok && raises(h, b) {
			out = append(out, PolicyChange{Aspect: PolicyThreshold,
				Message: fmt.Sprintf("the limit of %s is raised from %s to %s", code, b, h)})
		}
	}
	return out
}

// setsThreshold reports whether the health block of a policy sets a code.
// TRLC-LINKS: REQ-EMG-155
func setsThreshold(p Policy, code string) bool {
	if p.Health == nil {
		return false
	}
	_, ok := p.Health.Thresholds[code]
	return ok
}

// excludeWeakenings lists the exclude patterns at head that the base did not
// have.
//
// TRLC-LINKS: REQ-EMG-155
func excludeWeakenings(base, head Policy) []PolicyChange {
	if head.Health == nil {
		return nil
	}
	have := map[string]bool{}
	if base.Health != nil {
		for _, pattern := range base.Health.Exclude {
			have[normalizePattern(pattern)] = true
		}
	}
	var out []PolicyChange
	seen := map[string]bool{}
	for _, pattern := range head.Health.Exclude {
		key := normalizePattern(pattern)
		if have[key] || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, PolicyChange{Aspect: PolicyExclude, Message: fmt.Sprintf("health.exclude has a new pattern %q", strings.TrimSpace(pattern))})
	}
	return out
}

// normalizePattern is the form in which two exclude patterns are compared.
// TRLC-LINKS: REQ-EMG-155
func normalizePattern(pattern string) string {
	return strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(pattern)), "./")
}

// overrideWeakenings lists the limits an override of head raises. A limit is
// compared with the override of the same path at base when there is one, else
// with the effective limit of the base for the whole module.
//
// TRLC-LINKS: REQ-EMG-155
func overrideWeakenings(base, head Policy) []PolicyChange {
	if head.Health == nil {
		return nil
	}
	var out []PolicyChange
	for _, o := range head.Health.Overrides {
		path := normalizePattern(o.Path)
		codes := make([]string, 0, len(o.Thresholds))
		for code := range o.Thresholds {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		for _, code := range codes {
			t := o.Thresholds[code]
			h := Threshold{Warn: t.Warn, High: t.High}
			b, ok := baseOverrideLimit(base, path, code)
			if !ok {
				b, ok = base.effectiveLimit(code)
			}
			if ok && raises(h, b) {
				out = append(out, PolicyChange{Aspect: PolicyOverride,
					Message: fmt.Sprintf("the override for %q raises the limit of %s from %s to %s", path, code, b, h)})
			}
		}
	}
	return out
}

// baseOverrideLimit is the limit the base sets for a code in an override of
// the same path.
// TRLC-LINKS: REQ-EMG-155
func baseOverrideLimit(base Policy, path, code string) (Threshold, bool) {
	if base.Health == nil {
		return Threshold{}, false
	}
	for _, o := range base.Health.Overrides {
		if normalizePattern(o.Path) != path {
			continue
		}
		if t, ok := o.Thresholds[code]; ok {
			return Threshold{Warn: t.Warn, High: t.High}, true
		}
	}
	return Threshold{}, false
}

// PolicyLevelOf returns the level of health.policy_weakened for a change from
// the base policy to the head model: an error when it is one under the head
// bundle at the head tier or under the same bundle at the tier and bundles of
// the base lifecycle. Lowering the tier therefore cannot lower the level of
// its own weakening. A base without a lifecycle adds nothing.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func PolicyLevelOf(head model.Bundle, baseLifecycle *model.ManifestLifecycle) string {
	if levelOfCode(head, model.HealthPolicyWeakenedCode) == LevelError {
		return LevelError
	}
	if baseLifecycle == nil {
		return LevelWarning
	}
	base := head
	lifecycle := *baseLifecycle
	base.Manifest.Lifecycle = &lifecycle
	return levelOfCode(base, model.HealthPolicyWeakenedCode)
}

// levelOfCode is the level of a code at the tier of a bundle, from the tier
// policy.
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func levelOfCode(b model.Bundle, code string) string {
	out := validate.ApplyTierPolicy(b, []validate.Diagnostic{{Code: code, Severity: validate.SeverityWarning}})
	if len(out) == 1 && out[0].Severity == validate.SeverityError {
		return LevelError
	}
	return LevelWarning
}
