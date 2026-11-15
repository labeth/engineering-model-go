// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// The tier gap (ADR-EMG-031 §tier.gap): what a target life-cycle tier would
// require of a model that it does not meet now.

import (
	"fmt"
	"sort"
	"strings"

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
func TierGapFromFiles(req FileRequest, target string, options Options) (TierGapReport, error) {
	rank := model.TierRank(target)
	if rank == 0 {
		return TierGapReport{}, fmt.Errorf("unknown target tier %q: want one of %s", target, strings.Join(model.Tiers, ", "))
	}
	bundle, err := loadRequestBundle(req)
	if err != nil {
		return TierGapReport{}, err
	}
	report := TierGapReport{TargetTier: model.Tiers[rank-1], ByCode: map[string]int{}, Blocking: []TierGapFinding{},
		AviationBelowFloor: []validate.AviationFloorGap{}, Dependencies: []TierGapDependency{}}
	report.CurrentTier, _, _ = validate.EffectiveTier(bundle)

	var composed composition.CompositionResult
	options.Composed = &composed
	now, err := diagnosticsAt(bundle, req, options)
	if err != nil {
		return report, err
	}
	targeted, lifecycle := withTier(bundle, report.TargetTier)
	options.Composition, options.Composed = nil, nil
	then, err := diagnosticsAt(targeted, req, options)
	if err != nil {
		return report, err
	}
	report.addBlocking(now, then)
	if bundle.Aviation != nil && lifecycle.HasBundle(model.BundleSafety) {
		report.AviationBelowFloor = append(report.AviationBelowFloor, validate.AviationFloorGaps(bundle.Aviation.Aviation, rank)...)
	}
	report.addDependenciesBelow(composed, rank)
	return report, nil
}

// diagnosticsAt is every diagnostic of the bundle at its own tier: the trace
// diagnostics and the validation diagnostics under the tier policy.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func diagnosticsAt(bundle model.Bundle, req FileRequest, options Options) ([]validate.Diagnostic, error) {
	_, diags, err := buildFromBundle(bundle, req, options)
	if err != nil {
		return nil, err
	}
	return append(diags, validate.ApplyTierPolicy(bundle, validate.Bundle(bundle))...), nil
}

// withTier is a copy of the bundle whose lifecycle tier is set in memory, and
// that lifecycle.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func withTier(bundle model.Bundle, tier string) (model.Bundle, model.ManifestLifecycle) {
	targeted := bundle
	lifecycle := model.ManifestLifecycle{}
	if bundle.Manifest.Lifecycle != nil {
		lifecycle = *bundle.Manifest.Lifecycle
	}
	lifecycle.Tier = tier
	targeted.Manifest.Lifecycle = &lifecycle
	return targeted, lifecycle
}

// diagnosticKey identifies a diagnostic across the two tiers.
type diagnosticKey struct{ code, path, message string }

// addBlocking adds the findings that are errors at the target tier (then) but
// not at the current one (now), with the counts by code.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func (r *TierGapReport) addBlocking(now, then []validate.Diagnostic) {
	before := map[diagnosticKey]validate.Diagnostic{}
	for _, d := range now {
		before[diagnosticKey{d.Code, d.Path, d.Message}] = d
	}
	seen := map[diagnosticKey]bool{}
	for _, d := range then {
		k := diagnosticKey{d.Code, d.Path, d.Message}
		prior, existed := before[k]
		if seen[k] || !blocksAtTarget(d, prior, existed) {
			continue
		}
		seen[k] = true
		severity := "absent"
		if existed {
			severity = string(prior.Severity)
		}
		r.Blocking = append(r.Blocking, TierGapFinding{Code: d.Code, RequiredFrom: d.RequiredFrom, SeverityNow: severity,
			Deferred: prior.Deferred, PoamRef: prior.PoamRef, Path: d.Path, Message: d.Message})
		r.ByCode[d.Code]++
	}
	sort.SliceStable(r.Blocking, func(i, j int) bool {
		if r.Blocking[i].Code != r.Blocking[j].Code {
			return r.Blocking[i].Code < r.Blocking[j].Code
		}
		return r.Blocking[i].Path < r.Blocking[j].Path
	})
}

// blocksAtTarget reports whether a diagnostic at the target tier is a new
// error: an error that was not one already (prior, when existed). A
// tierDecision mismatch the in-memory tier causes is not one.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func blocksAtTarget(d, prior validate.Diagnostic, existed bool) bool {
	if d.Severity != validate.SeverityError || d.Code == "lifecycle.tier_decision_mismatch" {
		return false
	}
	return !existed || prior.Severity != validate.SeverityError
}

// addDependenciesBelow lists the composed dependencies whose tier is below
// the target rank or unset.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func (r *TierGapReport) addDependenciesBelow(composed composition.CompositionResult, rank int) {
	if composed.Root == nil {
		return
	}
	for _, child := range composed.Root.Children {
		tier, childRank, ok := validate.EffectiveTier(child.Bundle)
		if ok && childRank >= rank {
			continue
		}
		r.Dependencies = append(r.Dependencies, TierGapDependency{Alias: child.Dependency, Subsystem: child.SubsystemID,
			Module: child.ModulePath, Version: child.Version, Tier: tier, Required: r.TargetTier})
	}
}
