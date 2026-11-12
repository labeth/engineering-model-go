// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// Tier severities of the policy table: deferred is a warning with the
// Deferred flag (evidence debt, still reported), then warning, then error.
const (
	tierDeferred = "deferred"
	tierWarning  = "warning"
	tierError    = "error"
)

// TierRule is one tier-keyed code of the policy table: its severity at each
// tier poc..operational (deferred, warning or error), the row that applies
// with the safety bundle, the tier from which it is an error (empty when it
// never is) without and with safety, and the bundle that emits it (empty for
// every model with lifecycle).
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
type TierRule struct {
	Code               string    `json:"code"`
	Severities         [5]string `json:"severities"`
	SafetySeverities   [5]string `json:"safetySeverities"`
	RequiredFrom       string    `json:"requiredFrom,omitempty"`
	RequiredFromSafety string    `json:"requiredFromSafety,omitempty"`
	Bundle             string    `json:"bundle,omitempty"`
	Note               string    `json:"note,omitempty"`
}

// tierRow is one table entry before export.
type tierRow struct {
	base   string // D, W or E per tier, poc..operational
	safety string // the row with the safety bundle; empty: same as base
	bundle string
	note   string
}

const (
	rowDebtToProduct     = "DDWEE" // deferred to prototype, warning at mvp, error from product
	rowDebtToOperational = "DDDWE"
	rowHygiene           = "DDWWE"
	rowStrictCoverage    = "WWWEE"
	rowSafetyFromMVP     = "DDEEE"
	// rowHealthDebt is a warning up to mvp and an error at product and
	// operational, the two highest tiers, where the bar for code quality is the
	// same and highest (ADR-EMG-057); with the safety bundle it is an error from
	// mvp. lifecycle.raise: [health.*] remains the escape hatch that makes debt
	// blocking at a lower tier.
	rowHealthDebt       = "WWWEE"
	rowHealthDebtSafety = "WWEEE"
	rowHealthDebtNote   = "code health debt (only with a health block); a warning up to mvp, an error at product and operational (from mvp with the safety bundle), an error at any tier with lifecycle.raise: [health.*]"
	// The requirement link rules follow the effective requirementLinks mode
	// (ADR-EMG-022 amended): guided below mvp unless declared strict,
	// strict from mvp.
	rowLinksStrictFromMVP = "DDEEE"
)

// requirementLinkCodes are the strictness-dependent rules of ADR-EMG-022:
// with requirementLinks: strict declared they are errors at every tier (the
// effective mode is the maximum of the declared one and the tier default).
var requirementLinkCodes = map[string]bool{
	"requirement.no_parent": true, "requirement.unsatisfied": true, "link.aou_unverified": true,
}

// tierTable is the engine tier policy (ADR-EMG-031, first pass of the
// tiered-development guidance §6.2 adapted to v0.10.0, ADR-EMG-036).
var tierTable = map[string]tierRow{
	"requirement.orphan":                          {base: rowDebtToProduct},
	"link.claimed_only":                           {base: rowDebtToProduct},
	"verification.unlinked":                       {base: rowDebtToProduct},
	"threat.verification_missing":                 {base: rowDebtToProduct},
	"requirement.no_parent":                       {base: rowLinksStrictFromMVP, note: "an error at every tier with requirementLinks: strict; strict from mvp"},
	"requirement.unsatisfied":                     {base: rowLinksStrictFromMVP, note: "an error at every tier with requirementLinks: strict; strict from mvp"},
	"link.aou_unverified":                         {base: rowLinksStrictFromMVP, note: "an error at every tier with requirementLinks: strict; strict from mvp"},
	"lifecycle.requirement_unapproved":            {base: rowDebtToProduct, note: "requirement status draft or proposed"},
	"lifecycle.requirement_unverified":            {base: rowDebtToOperational, note: "requirement status not verified (rejected and retired excepted)"},
	"verification.failed":                         {base: rowDebtToOperational, note: "a linked test result failed"},
	"verification.gated_only":                     {base: rowDebtToOperational, note: "every linked test is environment gated"},
	"behavior.state_unreachable":                  {base: rowHygiene},
	"behavior.machine_deadlock":                   {base: rowHygiene, note: "a non-final machine state without outgoing transition"},
	"behavior.transition_nondeterministic":        {base: rowHygiene, note: "two transitions from one state on one trigger without distinguishing guards"},
	"behavior.transition_no_trigger":              {base: rowHygiene, note: "a transition without on and without guard"},
	"behavior.invariant_unlinked":                 {base: rowHygiene, note: "a machine invariant linked to no existing requirement"},
	"behavior.event_unused":                       {base: rowHygiene},
	"behavior.flow_step_no_onerror":               {base: rowHygiene},
	"contract.unrealized":                         {base: rowHygiene},
	"verification.status_stale":                   {base: rowHygiene},
	"code.missing_trlc_link":                      {base: rowStrictCoverage, note: "strict code linking (coverage.strictStatus); an error without lifecycle"},
	"code.invalid_trlc_link":                      {base: rowStrictCoverage, note: "strict code linking (coverage.strictStatus); an error without lifecycle"},
	"code.scan_failed":                            {base: rowStrictCoverage, note: "strict code linking (coverage.strictStatus)"},
	"lifecycle.aviation_profile_missing":          {base: rowSafetyFromMVP, bundle: model.BundleSafety},
	"lifecycle.software_level_missing":            {base: rowSafetyFromMVP, bundle: model.BundleSafety},
	"aviation.state_below_tier":                   {base: rowSafetyFromMVP, note: "evidence state below the tier floor (none below mvp)"},
	"aviation.lifecycle.missing_category":         {base: rowSafetyFromMVP, note: "load-fatal without lifecycle"},
	"aviation.tool.incomplete_credit_assessment":  {base: rowDebtToProduct, note: "load-fatal without lifecycle"},
	"aviation.derived.incomplete_safety_feedback": {base: rowDebtToProduct, note: "load-fatal without lifecycle"},
	"composition.dependency_below_tier":           {base: rowDebtToProduct, note: "unless a risk with recorded acceptance names the dependency alias in dependencies (or the composing subsystem in appliesTo)"},
	"composition.dependency_tier_unknown":         {base: "WWWWW", safety: "WWWEE"},
	// Code health (ADR-EMG-052, ADR-EMG-057): debt is a warning up to mvp and an
	// error at product and operational (from mvp with the safety bundle);
	// lifecycle.raise: [health.*] makes it an error at any tier. A regression, a
	// new or worsened high finding against the base revision, has the same row.
	"health.complex_method":             {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.large_method":               {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.deep_nesting":               {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.bumpy_road":                 {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.complex_conditional":        {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.excess_arguments":           {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.primitive_obsession":        {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.brain_method":               {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.large_assertion_block":      {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.duplicated_assertion_block": {base: rowHealthDebt, safety: rowHealthDebtSafety, note: rowHealthDebtNote},
	"health.waiver_unused":              {base: rowHealthDebt, safety: rowHealthDebtSafety, note: "a health waiver that matches no finding (only with a health block); a stale waiver hides debt, so it is an error where the debt is an error"},
	// A stale baseline entry is a debt that was paid but not removed from the
	// file: a warning at every tier below operational (the baseline can only
	// shrink, ADR-EMG-059) and an error at operational, where the debt is zero
	// or accepted.
	"health.baseline_stale": {base: "WWWWE", note: "a health baseline entry that matches no finding, so it can be removed (only with a health baseline)"},
	"health.regression":     {base: "WWWEE", safety: "WWEEE", note: "a new or worsened high health finding against the baseline (only with a health block)"},
	// A change that weakens the policy it is measured by (a higher limit, a
	// wider exclude or override, a lower tier) is reported against the base
	// revision by enghealth check; a policyDecision records it.
	"health.policy_weakened": {base: "WWWEE", safety: "WWEEE", note: "a change that raises a health limit, widens health.exclude or an override or lowers the tier against the base revision, without an accepted health.policyDecision (only with a health block)"},
}

// coreCodes are the rank-0 code patterns: never tier keyed and never
// deferred by a POA&M item. The list is closed (TestCoreCodesAreClosed...).
var coreCodes = []string{
	"model.duplicate_id", "model.broken_reference", "model.invalid_*", "model.missing_*",
	"code.dangling_*", "composition.cycle", "composition.lock_*",
	"link.version_mismatch", "link.broken", "link.parent_unknown", "link.parent_missing", "link.child_unpublished",
	"link.review_unrecorded", "link.invalid_status", "contract.realizer_unpublished", "requirement.realizes_unknown",
	"risk.acceptance_unrecorded",
	"lifecycle.invalid_tier", "lifecycle.invalid_bundle", "lifecycle.tier_decision_mismatch",
	"lifecycle.requirement_links_conflict", "lifecycle.poam_defers_core",
	"lifecycle.missing", "lifecycle.bundles_missing", "lifecycle.tier_decision_missing",
	"model.requirement_links_missing", "model.unsupported_schema_version", "model.id_missing",
	"lint.mode_missing", "lint.mode_invalid", "lint.mode_guided_at_tier", "health.block_missing",
	"aviation.ai_actor_as_credit", "aviation.authority_state_unevidenced",
}

// CoreCodes returns the rank-0 code patterns (a trailing * matches a code
// prefix): integrity rules that are errors at every tier, are never keyed by
// the tier policy and that no POA&M item can defer.
// TRLC-LINKS: REQ-EMG-125
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func CoreCodes() []string {
	return append([]string(nil), coreCodes...)
}

// IsCoreCode reports whether a code is a rank-0 core code.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func IsCoreCode(code string) bool {
	for _, p := range coreCodes {
		if matchCodePattern(p, code) {
			return true
		}
	}
	return false
}

// matchCodePattern matches a code against an exact code or a pattern ending
// in * (a prefix).
// TRLC-LINKS: REQ-EMG-125
func matchCodePattern(pattern, code string) bool {
	pattern = strings.TrimSpace(pattern)
	if prefix, ok := strings.CutSuffix(pattern, "*"); ok {
		return strings.HasPrefix(code, prefix)
	}
	return pattern == code
}

// letterSeverity maps a table letter to its severity name.
// TRLC-LINKS: REQ-EMG-125
func letterSeverity(letter byte) string {
	switch letter {
	case 'E':
		return tierError
	case 'W':
		return tierWarning
	}
	return tierDeferred
}

// rowSeverities expands a row string to the five severity names.
// TRLC-LINKS: REQ-EMG-125
func rowSeverities(row string) [5]string {
	var out [5]string
	for i := range out {
		out[i] = letterSeverity(row[i])
	}
	return out
}

// requiredFrom returns the first tier at which a row is an error.
// TRLC-LINKS: REQ-EMG-125
func requiredFrom(row [5]string) string {
	for i, s := range row {
		if s == tierError {
			return model.Tiers[i]
		}
	}
	return ""
}

// TierPolicy returns the tier-keyed codes with their severity per tier,
// sorted by code. It is the table ApplyTierPolicy applies; engineering-flow
// renders it as "Severity by tier" in MODEL-RULES.md.
// TRLC-LINKS: REQ-EMG-125
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func TierPolicy() []TierRule {
	out := make([]TierRule, 0, len(tierTable))
	for code, row := range tierTable {
		safety := row.safety
		if safety == "" {
			safety = row.base
		}
		r := TierRule{Code: code, Severities: rowSeverities(row.base), SafetySeverities: rowSeverities(safety), Bundle: row.bundle, Note: row.note}
		r.RequiredFrom, r.RequiredFromSafety = requiredFrom(r.Severities), requiredFrom(r.SafetySeverities)
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// TierPolicyMarkdown renders TierPolicy as a Markdown table: one row per
// code, D (deferred), W (warning) or E (error) per tier, with the safety
// row where it differs.
// TRLC-LINKS: REQ-EMG-125
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func TierPolicyMarkdown() string {
	letter := map[string]string{tierDeferred: "D", tierWarning: "W", tierError: "E"}
	var b strings.Builder
	b.WriteString("| Code | poc | prototype | mvp | product | operational |\n|---|---|---|---|---|---|\n")
	row := func(label string, severities [5]string) {
		b.WriteString("| " + label + " |")
		for _, s := range severities {
			b.WriteString(" " + letter[s] + " |")
		}
		b.WriteString("\n")
	}
	for _, r := range TierPolicy() {
		label := "`" + r.Code + "`"
		if r.Bundle != "" {
			label += " (" + r.Bundle + " bundle)"
		}
		row(label, r.Severities)
		if r.SafetySeverities != r.Severities {
			row("`"+r.Code+"` with safety", r.SafetySeverities)
		}
	}
	return b.String()
}

// TierKeyed reports whether the tier policy sets a code's severity.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-130
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TierKeyed(code string) bool {
	_, ok := tierTable[code]
	return ok
}

// EffectiveTier returns the project tier of a model (lowercased) and its
// rank 1..5; ok is false without lifecycle or with an unknown tier, and then
// no tier rule applies.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func EffectiveTier(b model.Bundle) (string, int, bool) {
	if b.Manifest.Lifecycle == nil {
		return "", 0, false
	}
	rank := model.TierRank(b.Manifest.Lifecycle.Tier)
	if rank == 0 {
		return "", 0, false
	}
	return model.Tiers[rank-1], rank, true
}

// EffectiveRequirementLinkMode returns the requirement link strictness that
// applies: the declared requirementLinks, raised to strict from tier mvp
// (the effective maximum of ADR-EMG-031).
// TRLC-LINKS: REQ-EMG-127
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func EffectiveRequirementLinkMode(b model.Bundle) string {
	if _, rank, ok := EffectiveTier(b); ok && rank >= model.TierRank(model.TierMVP) {
		return model.RequirementLinksStrict
	}
	return model.RequirementLinkMode(b.Manifest)
}

// raised reports whether lifecycle.raise names a code (exactly or by a
// family.* pattern).
// TRLC-LINKS: REQ-EMG-125
func raised(b model.Bundle, code string) bool {
	if b.Manifest.Lifecycle == nil {
		return false
	}
	for _, p := range b.Manifest.Lifecycle.Raise {
		if matchCodePattern(p, code) {
			return true
		}
	}
	return false
}

// RendersDespite reports whether an error diagnostic lets the AsciiDoc
// document render in a model with lifecycle (D15 extended, ADR-EMG-032): a
// code whose severity the tier policy sets (a tier-keyed code or one
// lifecycle.raise names) fails the check but not the render; every other
// error, the rank-0 core codes included, stops it as before.
// TRLC-LINKS: REQ-EMG-130
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ASCIIDOC-GENERATOR
func RendersDespite(b model.Bundle, code string) bool {
	if _, _, ok := EffectiveTier(b); !ok || IsCoreCode(code) {
		return false
	}
	return TierKeyed(code) || raised(b, code)
}

// ApplyTierPolicy returns the diagnostics with the severities of the
// project tier: each tier-keyed code gets RequiredFrom and the severity of
// its row (deferred is a warning with Deferred set); an open POA&M item with
// milestone tier:<project tier> that names a diagnostic's code (and path)
// defers it with PoamRef (a tier-keyed code or a warning; an untiered error
// is never deferred); lifecycle.raise makes a code an error. A
// requirement link rule stays an error with requirementLinks: strict
// declared; rank-0 core codes and codes the table does not key are never
// changed. Without lifecycle it returns the diagnostics unchanged. It is idempotent and does
// not modify its input; it runs at each public entry on the merged list
// (ValidateBundle, GenerateAsciiDoc, the trace builders, engws check, MCP).
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-126, REQ-EMG-127, REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func ApplyTierPolicy(b model.Bundle, diags []Diagnostic) []Diagnostic {
	out := append([]Diagnostic(nil), diags...)
	tier, rank, ok := EffectiveTier(b)
	if !ok {
		return out
	}
	safety := b.Manifest.Lifecycle.HasBundle(model.BundleSafety)
	declaredStrict := model.RequirementLinkMode(b.Manifest) == model.RequirementLinksStrict
	defers := poamDefers(b, tier)
	for i := range out {
		d := &out[i]
		if IsCoreCode(d.Code) {
			continue
		}
		if row, keyed := tierTable[d.Code]; keyed {
			line := row.base
			if safety && row.safety != "" {
				line = row.safety
			}
			severities := rowSeverities(line)
			severity := severities[rank-1]
			d.RequiredFrom = requiredFrom(severities)
			if declaredStrict && requirementLinkCodes[d.Code] {
				severity, d.RequiredFrom = tierError, model.TierPoC
			}
			d.Severity, d.Deferred, d.PoamRef = SeverityWarning, severity == tierDeferred, ""
			if severity == tierError {
				d.Severity = SeverityError
			}
		}
		// A defer covers tier-keyed codes and warnings; an untiered error
		// keeps its severity (the engine never lowers it outside the table).
		if ref := defers(d.Code, d.Path); ref != "" && (TierKeyed(d.Code) || d.Severity != SeverityError) {
			d.Severity, d.Deferred, d.PoamRef = SeverityWarning, true, ref
		}
		if raised(b, d.Code) {
			d.Severity, d.Deferred, d.PoamRef = SeverityError, false, ""
		}
	}
	return out
}

// poamDefers returns the lookup of the open POA&M items of the project
// tier: for a code and path, the ID of the first item with milestone
// tier:<tier>, a status other than completed and a defers entry naming the
// code (exactly or as family.*) with no path or that path.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func poamDefers(b model.Bundle, tier string) func(code, path string) string {
	milestone := "tier:" + tier
	var items []model.POAMItem
	for _, p := range b.Architecture.AuthoredArchitecture.POAMItems {
		if strings.EqualFold(strings.TrimSpace(p.Milestone), milestone) && !strings.EqualFold(strings.TrimSpace(p.Status), "completed") && len(p.Defers) > 0 {
			items = append(items, p)
		}
	}
	return func(code, path string) string {
		for _, p := range items {
			for _, d := range p.Defers {
				if matchCodePattern(d.Code, code) && (strings.TrimSpace(d.Path) == "" || strings.TrimSpace(d.Path) == path) {
					return strings.TrimSpace(p.ID)
				}
			}
		}
		return ""
	}
}
