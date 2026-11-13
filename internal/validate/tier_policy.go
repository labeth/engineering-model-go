// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

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
	"code.owner_unresolved":                       {base: rowStrictCoverage, note: "a source file whose ENGMODEL-OWNER-UNIT marker is missing or names no functional unit (test and generated files excepted)"},
	"code.model_link_kind_mismatch":               {base: rowStrictCoverage, note: "an ENGMODEL-LINKS target that exists but is a requirement or a catalog term, not a model element"},
	"code.trace_unattached":                       {base: rowStrictCoverage, note: "a trace marker block not attached to any declaration"},
	"behavior.requirement_unmodeled":              {base: rowHygiene, note: "an event-driven or state-driven requirement that no machine invariant lists and whose appliesTo names no machine, state, transition, event or flow (only in a model with a machine; retired, rejected and derived requirements excepted)"},
	"behavior.machine_unverified":                 {base: rowHygiene, note: "a machine with no invariant, or an invariant with no current formal result when a formal evidence file exists"},
	"behavior.state_unlinked":                     {base: rowHygiene, note: "a state of a machine that lists no requirement in its own requirements (only in a model with a machine; imported alias::ID machines excepted)"},
	"behavior.transition_unlinked":                {base: rowHygiene, note: "a transition of a machine that lists no requirement in its own requirements (only in a model with a machine; imported alias::ID machines excepted)"},
	"behavior.event_unlinked":                     {base: rowHygiene, note: "an event that a machine transition triggers on and that lists no requirement in its own requirements (only in a model with a machine)"},
	"code.unit_unlinked":                          {base: rowDebtToProduct, note: "a functional unit that no scanned non-test source names in ENGMODEL-OWNER-UNIT or ENGMODEL-LINKS (only with scanned source; an element in codeCoverage.exempt excepted)"},
	"code.machine_unlinked":                       {base: rowDebtToProduct, note: "a machine whose owner unit, own ID, state IDs and transition IDs no scanned non-test source names"},
	"code.interface_unlinked":                     {base: rowDebtToProduct, note: "a software interface owned by a local functional unit that no scanned non-test source names in ENGMODEL-LINKS"},
	"code.state_unlinked":                         {base: rowHygiene, note: "a state of a machine owned by a local functional unit that no scanned non-test source names in ENGMODEL-LINKS"},
	"code.transition_unlinked":                    {base: rowHygiene, note: "a transition of a machine owned by a local functional unit that no scanned non-test source names in ENGMODEL-LINKS"},
	"code.event_unlinked":                         {base: rowHygiene, note: "an event that a transition of a machine owned by a local functional unit triggers on and that no scanned non-test source names in ENGMODEL-LINKS"},
	"code.flow_unlinked":                          {base: rowDebtToOperational, note: "a flow that touches a software interface owned by a local functional unit and that no scanned non-test source names in ENGMODEL-LINKS"},
	"code.data_unlinked":                          {base: rowDebtToOperational, note: "a data object that a flow touching a local software interface carries and that no scanned non-test source names in ENGMODEL-LINKS"},
	"code.exemption_unused":                       {base: "WWWWE", note: "a codeCoverage.exempt entry that matches no unlinked element, so it can be removed (only with scanned source)"},
	"gemara.control_needs_review":                 {base: rowDebtToOperational, note: "the Gemara evaluation log marks the control Needs Review"},
	"lifecycle.aviation_profile_missing":          {base: rowSafetyFromMVP, bundle: model.BundleSafety},
	"lifecycle.software_level_missing":            {base: rowSafetyFromMVP, bundle: model.BundleSafety},
	"aviation.state_below_tier":                   {base: rowSafetyFromMVP, note: "evidence state below the tier floor (none below mvp)"},
	"aviation.lifecycle.missing_category":         {base: rowSafetyFromMVP, note: "load-fatal without lifecycle"},
	"aviation.tool.incomplete_credit_assessment":  {base: rowSafetyFromMVP, note: "load-fatal without lifecycle"},
	"aviation.derived.incomplete_safety_feedback": {base: rowSafetyFromMVP, note: "load-fatal without lifecycle"},
	"catalog.term_unreferenced":                   {base: rowStrictCoverage, note: "a catalog term referenced nowhere in the model (only with lintRun.mode: strict)"},
	"composition.dependency_below_tier":           {base: rowDebtToProduct, note: "unless a risk with recorded acceptance names the dependency alias in dependencies (or the composing subsystem in appliesTo)"},
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
	"composition.dependency_tier_unknown",
}

// CoreCodes returns the rank-0 code patterns (a trailing * matches a code
// prefix): integrity rules that are errors at every tier, are never keyed by
// the tier policy and that no POA&M item can defer.
// TRLC-LINKS: REQ-EMG-125
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func CoreCodes() []string {
	return append([]string(nil), coreCodes...)
}

// nonDeferrable are codes that stay tier keyed (or tiered by their own rule)
// but that no POA&M item may defer: they report that the gate itself is being
// weakened or that evidence failed, so a defer would hide the thing the item
// is meant to be measured by. They are not core: core codes are errors at
// every tier and never tier keyed, whereas these keep their severity by tier
// (health.regression is WWWEE, verification.failed is DDDWE) and only the
// POA&M defer is refused. ApplyTierPolicy ignores a defer for them, and a
// defers entry that names one, exactly or through a family pattern, is
// lifecycle.poam_defers_core. The list is closed (TestNonDeferrableCodes...).
var nonDeferrable = []string{
	"health.policy_weakened", "health.regression", "health.baseline_stale", "health.waiver_unused",
	"verification.failed", "lifecycle.poam_overdue", "lifecycle.poam_defers_unused",
}

// NonDeferrableCodes returns the codes no POA&M item can defer although the
// tier policy keys their severity.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func NonDeferrableCodes() []string {
	return append([]string(nil), nonDeferrable...)
}

// IsNonDeferrable reports whether a code is one no POA&M item can defer.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func IsNonDeferrable(code string) bool {
	for _, c := range nonDeferrable {
		if c == code {
			return true
		}
	}
	return false
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

// asOfOverride is the date SetAsOf pins; asOfMu guards it.
var (
	asOfMu       sync.RWMutex
	asOfOverride time.Time
)

// asOfEnv names the environment variable that pins the date POA&M expiry is
// measured at (YYYY-MM-DD), for reproducible runs and tests.
const asOfEnv = "ENGMOD_AS_OF"

// SetAsOf pins the date AsOf returns; the zero time clears the pin. The CLIs
// need not call it: AsOf falls back to ENGMOD_AS_OF and then to today.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func SetAsOf(t time.Time) {
	asOfMu.Lock()
	defer asOfMu.Unlock()
	asOfOverride = t
}

// AsOf returns the date POA&M expiry is measured at, the edge of the
// otherwise clock-free engine: the date SetAsOf pinned, else ENGMOD_AS_OF
// (YYYY-MM-DD; an unparsable value is ignored, never read as "no expiry"),
// else today's UTC date. ApplyTierPolicy and Bundle use it; the pure
// ApplyTierPolicyAt takes the date as a parameter, and the zero time means
// that no item expires.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func AsOf() time.Time {
	asOfMu.RLock()
	pinned := asOfOverride
	asOfMu.RUnlock()
	if !pinned.IsZero() {
		return pinned
	}
	if v := strings.TrimSpace(os.Getenv(asOfEnv)); v != "" {
		if t, err := time.Parse(time.DateOnly, v); err == nil {
			return t
		}
	}
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// ApplyTierPolicy is ApplyTierPolicyAt at AsOf(): the form every entry point
// calls on the merged list, with POA&M expiry measured at the CLI date.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-126, REQ-EMG-127, REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func ApplyTierPolicy(b model.Bundle, diags []Diagnostic) []Diagnostic {
	return ApplyTierPolicyAt(b, diags, AsOf())
}

// ApplyTierPolicyAt returns the diagnostics with the severities of the
// project tier: each tier-keyed code gets RequiredFrom and the severity of
// its row (deferred is a warning with Deferred set); an open POA&M item with
// milestone tier:<project tier> that names a diagnostic's code (and path or
// requirement ref) defers it with PoamRef (a tier-keyed code or a warning; an
// untiered error is never deferred, nor is a non-deferrable code); an item
// whose dueDate is before asOf is overdue and defers nothing (the zero asOf
// disables expiry, so the function is deterministic); lifecycle.raise makes a
// code an error. A requirement link rule stays an error with
// requirementLinks: strict declared; rank-0 core codes and codes the table
// does not key are never changed. Without lifecycle it returns the
// diagnostics unchanged. It is idempotent and does not modify its input.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-126, REQ-EMG-127, REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func ApplyTierPolicyAt(b model.Bundle, diags []Diagnostic, asOf time.Time) []Diagnostic {
	out := append([]Diagnostic(nil), diags...)
	tier, rank, ok := EffectiveTier(b)
	if !ok {
		return out
	}
	run := tierPolicyRun{
		b:              b,
		rank:           rank,
		safety:         b.Manifest.Lifecycle.HasBundle(model.BundleSafety),
		declaredStrict: model.RequirementLinkMode(b.Manifest) == model.RequirementLinksStrict,
		defers:         poamDefers(b, tier, asOf),
	}
	for i := range out {
		run.apply(&out[i])
	}
	return out
}

// tierPolicyRun is the context ApplyTierPolicyAt applies to each diagnostic.
type tierPolicyRun struct {
	b              model.Bundle
	rank           int
	safety         bool
	declaredStrict bool
	defers         func(code, path string) string
}

// apply sets the tier severity, the POA&M defer and the raise of one
// diagnostic; a rank-0 core code is left as it is.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-126, REQ-EMG-127, REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (r tierPolicyRun) apply(d *Diagnostic) {
	if IsCoreCode(d.Code) {
		return
	}
	r.applyRow(d)
	r.applyDefer(d)
	if raised(r.b, d.Code) {
		d.Severity, d.Deferred, d.PoamRef = SeverityError, false, ""
	}
}

// applyRow gives a tier-keyed code the severity of its row at the tier.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-127
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (r tierPolicyRun) applyRow(d *Diagnostic) {
	row, keyed := tierTable[d.Code]
	if !keyed {
		return
	}
	severities := rowSeverities(row.line(r.safety))
	severity := severities[r.rank-1]
	d.RequiredFrom = requiredFrom(severities)
	if r.declaredStrict && requirementLinkCodes[d.Code] {
		severity, d.RequiredFrom = tierError, model.TierPoC
	}
	d.Severity, d.Deferred, d.PoamRef = SeverityWarning, severity == tierDeferred, ""
	if severity == tierError {
		d.Severity = SeverityError
	}
}

// line returns the row of the tier table for a model with or without the
// safety bundle.
// TRLC-LINKS: REQ-EMG-125
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (row tierRow) line(safety bool) string {
	if safety && row.safety != "" {
		return row.safety
	}
	return row.base
}

// applyDefer defers a diagnostic named by an honoured POA&M item. A defer
// covers tier-keyed codes and warnings; an untiered error keeps its severity
// (the engine never lowers it outside the table), and a non-deferrable code
// keeps its tier severity.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (r tierPolicyRun) applyDefer(d *Diagnostic) {
	if !deferrable(d) {
		return
	}
	if ref := r.defers(d.Code, d.Path); ref != "" {
		d.Severity, d.Deferred, d.PoamRef = SeverityWarning, true, ref
	}
}

// deferrable reports whether a POA&M item may defer a diagnostic.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func deferrable(d *Diagnostic) bool {
	return !IsNonDeferrable(d.Code) && (TierKeyed(d.Code) || d.Severity != SeverityError)
}

// activePoamItem is an open POA&M item of the project tier that carries
// defers and is not overdue, with its index in poamItems.
type activePoamItem struct {
	index int
	item  model.POAMItem
}

// activePoamItems returns the POA&M items whose defers are honoured at the
// tier and date: milestone tier:<tier>, a status other than completed, at
// least one defers entry, and no dueDate before asOf.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func activePoamItems(b model.Bundle, tier string, asOf time.Time) []activePoamItem {
	milestone := "tier:" + tier
	var items []activePoamItem
	for i, p := range b.Architecture.AuthoredArchitecture.POAMItems {
		if honoursDefers(p, milestone, asOf) {
			items = append(items, activePoamItem{index: i, item: p})
		}
	}
	return items
}

// honoursDefers reports whether an item is open at the milestone, carries
// defers and is not overdue at asOf.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func honoursDefers(p model.POAMItem, milestone string, asOf time.Time) bool {
	if !strings.EqualFold(strings.TrimSpace(p.Milestone), milestone) || poamCompleted(p) {
		return false
	}
	return len(p.Defers) > 0 && !poamOverdue(p, asOf)
}

// poamCompleted reports whether an item's status is completed.
// TRLC-LINKS: REQ-EMG-129
func poamCompleted(p model.POAMItem) bool {
	return strings.EqualFold(strings.TrimSpace(p.Status), "completed")
}

// poamOverdue reports whether an open item's dueDate (YYYY-MM-DD; a longer
// timestamp is read by its date) is before asOf. An unparsable dueDate or the
// zero asOf is never overdue.
// TRLC-LINKS: REQ-EMG-129
func poamOverdue(p model.POAMItem, asOf time.Time) bool {
	if asOf.IsZero() || poamCompleted(p) {
		return false
	}
	due := strings.TrimSpace(p.DueDate)
	if len(due) > len(time.DateOnly) {
		due = due[:len(time.DateOnly)]
	}
	t, err := time.Parse(time.DateOnly, due)
	if err != nil {
		return false
	}
	y, m, d := asOf.UTC().Date()
	return t.Before(time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
}

// deferMatches reports whether a defers entry names a diagnostic: the code
// (exactly or as family.*) and, when the entry sets them, the exact path and
// the requirement its ref resolves to.
// TRLC-LINKS: REQ-EMG-129
func deferMatches(b model.Bundle, d model.POAMDefer, x Diagnostic) bool {
	if !matchCodePattern(d.Code, x.Code) {
		return false
	}
	if p := strings.TrimSpace(d.Path); p != "" && p != x.Path {
		return false
	}
	if ref := strings.TrimSpace(d.Ref); ref != "" && !pathUnderRequirement(b, ref, x.Path) {
		return false
	}
	return true
}

// pathUnderRequirement reports whether a diagnostic path is the path of the
// requirement with the given ID, or below it (requirements[N].status):
// positional requirements[N] or by ID requirements[ID]. An ID that names no
// requirement matches nothing.
// TRLC-LINKS: REQ-EMG-129
func pathUnderRequirement(b model.Bundle, id, path string) bool {
	under := func(key string) bool {
		base := "requirements[" + key + "]"
		return path == base || strings.HasPrefix(path, base+".")
	}
	for i, r := range b.Requirements.Requirements {
		if strings.TrimSpace(r.ID) == id {
			return under(fmt.Sprint(i)) || under(id)
		}
	}
	return false
}

// poamDefers returns the lookup of the open POA&M items of the project
// tier: for a code and path, the ID of the first honoured item (see
// activePoamItems) with a defers entry naming the code (exactly or as
// family.*) with no path or that path, or with a ref to the requirement.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func poamDefers(b model.Bundle, tier string, asOf time.Time) func(code, path string) string {
	items := activePoamItems(b, tier, asOf)
	return func(code, path string) string {
		for _, p := range items {
			for _, d := range p.item.Defers {
				if deferMatches(b, d, Diagnostic{Code: code, Path: path}) {
					return strings.TrimSpace(p.item.ID)
				}
			}
		}
		return ""
	}
}
