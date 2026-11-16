// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// requiredSwitchDiagnostics reports the required model switches outside the
// lifecycle block (ADR-EMG-061, no silent defaults): requirementLinks
// (model.requirement_links_missing; a value other than strict or guided stays
// model.invalid_requirement_links), requirements lintRun.mode
// (lint.mode_missing, lint.mode_invalid, and lint.mode_guided_at_tier when a
// guided mode meets tier mvp or above), the health block from tier product
// (health.block_missing) and the model id (model.id_missing). Each is an
// error that is never tier keyed and that no POA&M item can defer; the
// lifecycle block itself is lifecycleDiagnostics.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-162, REQ-EMG-163, REQ-EMG-164, REQ-EMG-165, REQ-EMG-167
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func requiredSwitchDiagnostics(b model.Bundle) []Diagnostic {
	var s diagSink
	_, rank, hasTier := EffectiveTier(b)
	atMVP := hasTier && rank >= model.TierRank(model.TierMVP)
	atProduct := hasTier && rank >= model.TierRank(model.TierProduct)

	if strings.TrimSpace(b.Manifest.RequirementLinks) == "" {
		s.fail("model.requirement_links_missing", "requirementLinks", "engmod.yml has no requirementLinks: write strict or guided (guided only below mvp)")
	}
	s.lintModeSwitch(b, atMVP)
	if atProduct && b.Manifest.Health == nil {
		s.fail("health.block_missing", "health", fmt.Sprintf("tier %s needs the engmod.yml health block (thresholds, waivers and the baseline of accepted debt); a model with no debt writes an empty block", strings.ToLower(b.Manifest.Lifecycle.Tier)))
	}
	if strings.TrimSpace(b.Architecture.Model.ID) == "" {
		s.fail("model.id_missing", "model.id", "the model has no id: module.modelId in engmod.yml is empty")
	}
	return s.diags
}

// lintModeSwitch requires requirements lintRun.mode to be written, strict or
// guided, and strict from tier mvp.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-162, REQ-EMG-163, REQ-EMG-164, REQ-EMG-165, REQ-EMG-167
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s *diagSink) lintModeSwitch(b model.Bundle, atMVP bool) {
	switch mode := strings.ToLower(strings.TrimSpace(b.Requirements.LintRun.Mode)); mode {
	case "":
		s.fail("lint.mode_missing", "lintRun.mode", "requirements.yml has no lintRun.mode: write strict or guided (guided only below mvp); an empty mode no longer means strict")
	case "strict":
	case "guided":
		if atMVP {
			s.fail("lint.mode_guided_at_tier", "lintRun.mode", fmt.Sprintf("lintRun.mode guided at tier %s: the EARS lint must be strict from mvp", strings.ToLower(b.Manifest.Lifecycle.Tier)))
		}
	default:
		s.fail("lint.mode_invalid", "lintRun.mode", fmt.Sprintf("lintRun.mode %q must be strict or guided", b.Requirements.LintRun.Mode))
	}
}
