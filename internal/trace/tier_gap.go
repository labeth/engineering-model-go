// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// The tier gap (ADR-EMG-031 §tier.gap): what a target life-cycle tier would
// require of a model that it does not meet now.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TierGapReport lists what a target tier requires that the model does not
// meet: each finding that is an error at the target but not now (code,
// requiredFrom, current severity, path, message), the counts by code, the
// aviation records below the target's evidence floor and the composed
// dependencies below the target tier or without one. It names rule codes
// only: engineering-flow joins them to life-cycle objectives.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
type TierGapReport struct {
	CurrentTier        string                      `json:"currentTier"`
	TargetTier         string                      `json:"targetTier"`
	Blocking           []TierGapFinding            `json:"blocking"`
	ByCode             map[string]int              `json:"byCode"`
	AviationBelowFloor []validate.AviationFloorGap `json:"aviationBelowFloor"`
	Dependencies       []TierGapDependency         `json:"dependencies"`
}

// TierGapFinding is one diagnostic the target tier makes an error.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type TierGapFinding struct {
	Code         string `json:"code"`
	RequiredFrom string `json:"requiredFrom,omitempty"`
	SeverityNow  string `json:"severityNow"`
	Deferred     bool   `json:"deferredNow,omitempty"`
	PoamRef      string `json:"poamRef,omitempty"`
	Path         string `json:"path,omitempty"`
	Message      string `json:"message"`
}

// TierGapDependency is a composed dependency below the target tier; Tier is
// empty when the dependency declares none.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
type TierGapDependency struct {
	Alias     string `json:"alias"`
	Subsystem string `json:"subsystem"`
	Module    string `json:"module"`
	Version   string `json:"version"`
	Tier      string `json:"tier"`
	Required  string `json:"required"`
}

// TierGapFromFiles computes the tier gap of a model towards target: it
// builds the model's diagnostics (validation, composition, trace) at its own
// tier and again with lifecycle.tier set to target in memory (the bundles,
// tierDecision and raise kept; a model without lifecycle is compared from
// no tier), and reports the findings that are errors only at the target. A
// tierDecision mismatch the in-memory tier causes is not reported.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func TierGapFromFiles(modelPath, requirementsPath, codeRoot, target string, options Options) (TierGapReport, error) {
	rank := model.TierRank(target)
	if rank == 0 {
		return TierGapReport{}, fmt.Errorf("unknown target tier %q: want one of %s", target, strings.Join(model.Tiers, ", "))
	}
	bundle, err := codescan.LoadBundle(modelPath)
	if err != nil {
		return TierGapReport{}, err
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		return TierGapReport{}, err
	}
	bundle.Requirements = requirements
	report := TierGapReport{TargetTier: model.Tiers[rank-1], ByCode: map[string]int{}, Blocking: []TierGapFinding{},
		AviationBelowFloor: []validate.AviationFloorGap{}, Dependencies: []TierGapDependency{}}
	current, _, _ := validate.EffectiveTier(bundle)
	report.CurrentTier = current

	var composed composition.CompositionResult
	options.Composed = &composed
	_, now, err := buildFromBundle(bundle, modelPath, codeRoot, options)
	if err != nil {
		return report, err
	}
	now = append(now, validate.ApplyTierPolicy(bundle, validate.Bundle(bundle))...)

	targeted := bundle
	lifecycle := model.ManifestLifecycle{}
	if bundle.Manifest.Lifecycle != nil {
		lifecycle = *bundle.Manifest.Lifecycle
	}
	lifecycle.Tier = report.TargetTier
	targeted.Manifest.Lifecycle = &lifecycle
	at := options
	at.Composition, at.Composed = nil, nil
	_, then, err := buildFromBundle(targeted, modelPath, codeRoot, at)
	if err != nil {
		return report, err
	}
	then = append(then, validate.ApplyTierPolicy(targeted, validate.Bundle(targeted))...)

	type key struct{ code, path, message string }
	before := map[key]validate.Diagnostic{}
	for _, d := range now {
		before[key{d.Code, d.Path, d.Message}] = d
	}
	seen := map[key]bool{}
	for _, d := range then {
		k := key{d.Code, d.Path, d.Message}
		if d.Severity != validate.SeverityError || seen[k] || d.Code == "lifecycle.tier_decision_mismatch" {
			continue
		}
		seen[k] = true
		prior, existed := before[k]
		if existed && prior.Severity == validate.SeverityError {
			continue
		}
		severity := "absent"
		if existed {
			severity = string(prior.Severity)
		}
		report.Blocking = append(report.Blocking, TierGapFinding{Code: d.Code, RequiredFrom: d.RequiredFrom, SeverityNow: severity,
			Deferred: prior.Deferred, PoamRef: prior.PoamRef, Path: d.Path, Message: d.Message})
		report.ByCode[d.Code]++
	}
	sort.SliceStable(report.Blocking, func(i, j int) bool {
		if report.Blocking[i].Code != report.Blocking[j].Code {
			return report.Blocking[i].Code < report.Blocking[j].Code
		}
		return report.Blocking[i].Path < report.Blocking[j].Path
	})
	if bundle.Aviation != nil && lifecycle.HasBundle(model.BundleSafety) {
		report.AviationBelowFloor = append(report.AviationBelowFloor, validate.AviationFloorGaps(bundle.Aviation.Aviation, rank)...)
	}
	if composed.Root != nil {
		for _, child := range composed.Root.Children {
			tier, childRank, ok := validate.EffectiveTier(child.Bundle)
			if ok && childRank >= rank {
				continue
			}
			report.Dependencies = append(report.Dependencies, TierGapDependency{Alias: child.Dependency, Subsystem: child.SubsystemID,
				Module: child.ModulePath, Version: child.Version, Tier: tier, Required: report.TargetTier})
		}
	}
	return report, nil
}
