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
	var diags []Diagnostic
	add := func(code, path, format string, args ...any) {
		diags = append(diags, Diagnostic{Code: code, Severity: SeverityError, Message: fmt.Sprintf(format, args...), Path: path})
	}
	_, rank, hasTier := EffectiveTier(b)
	atMVP := hasTier && rank >= model.TierRank(model.TierMVP)

	if strings.TrimSpace(b.Manifest.RequirementLinks) == "" {
		add("model.requirement_links_missing", "requirementLinks", "engmod.yml has no requirementLinks: write strict or guided (guided only below mvp)")
	}

	switch mode := strings.ToLower(strings.TrimSpace(b.Requirements.LintRun.Mode)); mode {
	case "":
		add("lint.mode_missing", "lintRun.mode", "requirements.yml has no lintRun.mode: write strict or guided (guided only below mvp); an empty mode no longer means strict")
	case "strict":
	case "guided":
		if atMVP {
			add("lint.mode_guided_at_tier", "lintRun.mode", "lintRun.mode guided at tier %s: the EARS lint must be strict from mvp", strings.ToLower(b.Manifest.Lifecycle.Tier))
		}
	default:
		add("lint.mode_invalid", "lintRun.mode", "lintRun.mode %q must be strict or guided", b.Requirements.LintRun.Mode)
	}

	if hasTier && rank >= model.TierRank(model.TierProduct) && b.Manifest.Health == nil {
		add("health.block_missing", "health", "tier %s needs the engmod.yml health block (thresholds, waivers and the baseline of accepted debt); a model with no debt writes an empty block", strings.ToLower(b.Manifest.Lifecycle.Tier))
	}

	if strings.TrimSpace(b.Architecture.Model.ID) == "" {
		add("model.id_missing", "model.id", "the model has no id: module.modelId in engmod.yml is empty")
	}
	return diags
}
