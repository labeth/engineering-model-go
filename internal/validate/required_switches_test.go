// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// completeBundleAt is a model with every required switch written, at the
// tier: a lifecycle {tier, bundles [foundation], tierDecision} with an
// accepted DEC-TIER-<TIER> decision whose Tier: and Bundles: lines agree,
// requirementLinks strict, lintRun.mode strict, schemaVersion 3, a model
// id, a health block, and one derived requirement so that the strict link
// rules are met. It has no error.
func completeBundleAt(tier string) model.Bundle {
	b := behaviorTestBundle(nil, nil)
	id := "DEC-TIER-" + strings.ToUpper(tier)
	b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier, Bundles: []string{"foundation"}, TierDecision: id}
	b.Architecture.Decisions = []model.Decision{tierDecision(id, "accepted", "Tier: "+displayTier(tier)+"\nBundles: foundation")}
	b.Manifest.RequirementLinks = model.RequirementLinksStrict
	b.Requirements.LintRun.Mode = "strict"
	b.Manifest.Health = &model.ManifestHealth{}
	b.Requirements.Requirements = []model.Requirement{{ID: "REQ-A", Text: "The unit shall run.", AppliesTo: []string{"FU-A"}, Derived: true, DerivedRationale: "Operability.", Status: "verified"}}
	return b
}

// completeBundle is completeBundleAt mvp, the first tier with strict links.
func completeBundle(t *testing.T) model.Bundle {
	t.Helper()
	return completeBundleAt(model.TierMVP)
}

// errorCodes returns the codes of the error diagnostics.
func errorCodes(diags []Diagnostic) []string {
	var out []string
	for _, d := range diags {
		if d.Severity == SeverityError {
			out = append(out, d.Code)
		}
	}
	return out
}

// A complete model validates without an error at every tier, with the
// health block required only from product.
// Mutation: a required-switch rule that fires on a complete model fails this.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func TestCompleteBundleHasNoErrors(t *testing.T) {
	if got := errorCodes(Bundle(completeBundle(t))); len(got) != 0 {
		t.Fatalf("complete mvp bundle has errors %v", got)
	}
	for _, tier := range []string{"poc", "prototype", "mvp", "product", "operational"} {
		b := completeBundleAt(tier)
		if tier == "poc" || tier == "prototype" {
			b.Manifest.RequirementLinks = model.RequirementLinksGuided
		}
		if got := errorCodes(ApplyTierPolicy(b, Bundle(b))); len(got) != 0 {
			t.Errorf("complete %s bundle has errors %v", tier, got)
		}
	}
}

// requiredCase triggers one required-switch code with a minimal edit of a
// complete bundle.
type requiredCase struct {
	name   string
	code   string
	remedy string // substring of the remedy
	path   string
	edit   func(*model.Bundle) // nil: emitted outside the validator (the emitting package tests it)
}

var requiredCases = []requiredCase{
	{"no lifecycle", "lifecycle.missing", "ef-tier-up", "lifecycle",
		func(b *model.Bundle) { b.Manifest.Lifecycle = nil }},
	{"unknown tier", "lifecycle.invalid_tier", "ef-tier-up", "lifecycle.tier",
		func(b *model.Bundle) { b.Manifest.Lifecycle.Tier = "beta" }},
	{"no bundles", "lifecycle.bundles_missing", "ef-tier-up", "lifecycle.bundles",
		func(b *model.Bundle) { b.Manifest.Lifecycle.Bundles = nil }},
	{"unknown bundle", "lifecycle.invalid_bundle", "ef-tier-up", "lifecycle.bundles[0]",
		func(b *model.Bundle) { b.Manifest.Lifecycle.Bundles = []string{"comfort"} }},
	{"no tier decision", "lifecycle.tier_decision_missing", "write engmod.yml lifecycle {tier, bundles, tierDecision}", "lifecycle.tierDecision",
		func(b *model.Bundle) { b.Manifest.Lifecycle.TierDecision = "" }},
	{"decision without Bundles line", "lifecycle.tier_decision_mismatch", "ef-tier-up", "lifecycle.tierDecision",
		func(b *model.Bundle) { b.Architecture.Decisions[0].Decision = "Tier: MVP" }},
	{"no requirementLinks", "model.requirement_links_missing", "requirementLinks: strict", "requirementLinks",
		func(b *model.Bundle) { b.Manifest.RequirementLinks = "" }},
	{"unknown requirementLinks", "model.invalid_requirement_links", "requirementLinks: strict", "requirementLinks",
		func(b *model.Bundle) { b.Manifest.RequirementLinks = "lenient" }},
	{"guided links at mvp", "lifecycle.requirement_links_conflict", "requirementLinks: strict", "requirementLinks",
		func(b *model.Bundle) { b.Manifest.RequirementLinks = "guided" }},
	{"no lint mode", "lint.mode_missing", "lintRun.mode: strict", "lintRun.mode",
		func(b *model.Bundle) { b.Requirements.LintRun.Mode = "" }},
	{"unknown lint mode", "lint.mode_invalid", "lintRun.mode: strict", "lintRun.mode",
		func(b *model.Bundle) { b.Requirements.LintRun.Mode = "lenient" }},
	{"guided lint at mvp", "lint.mode_guided_at_tier", "lintRun.mode: strict", "lintRun.mode",
		func(b *model.Bundle) { b.Requirements.LintRun.Mode = "guided" }},
	{"no health block at product", "health.block_missing", "enghealth baseline write", "health",
		func(b *model.Bundle) {
			*b = completeBundleAt(model.TierProduct)
			b.Manifest.Health = nil
		}},
	{"schemaVersion 0", "model.unsupported_schema_version", "set schemaVersion: 3 at the top of the file", "requirements.schemaVersion",
		func(b *model.Bundle) { b.Requirements.SchemaVersion = 0 }},
	{"schemaVersion 1", "model.unsupported_schema_version", "schemaVersion: 3", "design.schemaVersion",
		func(b *model.Bundle) { b.Design.SchemaVersion = 1 }},
	{"no model id", "model.id_missing", "module.modelId", "model.id",
		func(b *model.Bundle) { b.Architecture.Model.ID = "" }},
	// Emitted by composition (the lock) and engws check (the template and the
	// pin), which set the remedy from RequiredRemedy; their tests drive them.
	{"outdated lock schema", "composition.lock_schema_outdated", "engws bump", "engmod.lock.yml", nil},
	{"stale check template", "engine.check_template_stale", "engflow upgrade --root", ".github/engmod-check", nil},
	{"outdated engine pin", "engine.pin_outdated", "go get github.com/labeth/engineering-model-go@", ".github/engmod-check/go.mod", nil},
}

// literalRemedies pins the remedy text of every required switch code. They
// are literals on purpose: comparing with remedyFor would pass for any text,
// and these are the words users act on.
var literalRemedies = map[string]string{
	"lifecycle.missing":                    "run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision}",
	"lifecycle.invalid_tier":               "run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision}",
	"lifecycle.bundles_missing":            "run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision}",
	"lifecycle.invalid_bundle":             "run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision}",
	"lifecycle.tier_decision_missing":      "run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision}",
	"lifecycle.tier_decision_mismatch":     "run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision}",
	"model.requirement_links_missing":      "run the ef-tier-up skill (engineering-flow) to choose the tier, then set requirementLinks: strict in engmod.yml (guided is allowed only below mvp)",
	"model.invalid_requirement_links":      "run the ef-tier-up skill (engineering-flow) to choose the tier, then set requirementLinks: strict in engmod.yml (guided is allowed only below mvp)",
	"lifecycle.requirement_links_conflict": "run the ef-tier-up skill (engineering-flow) to choose the tier, then set requirementLinks: strict in engmod.yml (guided is allowed only below mvp)",
	"lint.mode_missing":                    "run the ef-tier-up skill (engineering-flow) to choose the tier, then set lintRun.mode: strict in requirements.yml (guided is allowed only below mvp)",
	"lint.mode_invalid":                    "run the ef-tier-up skill (engineering-flow) to choose the tier, then set lintRun.mode: strict in requirements.yml (guided is allowed only below mvp)",
	"lint.mode_guided_at_tier":             "run the ef-tier-up skill (engineering-flow) to choose the tier, then set lintRun.mode: strict in requirements.yml (guided is allowed only below mvp)",
	"health.block_missing":                 "run the ef-tier-up skill (engineering-flow) to write the health block with its debt list, then record the baseline with enghealth baseline write",
	"model.unsupported_schema_version":     "set schemaVersion: 3 at the top of the file (edit the file; an upgrade command is planned in engineering-flow and does not exist yet)",
	"model.id_missing":                     "set module.modelId in engmod.yml to the stable identifier of the model (for example MY-SYSTEM)",
	"composition.lock_schema_outdated":     "regenerate the lock with the current engine: run engws bump <repo> <alias> <version> (it rewrites engmod.lock.yml with the current schemaVersion), then engws sync",
	"engine.check_template_stale":          "run engflow upgrade --root <dir> to replace .github/engmod-check with the current template that takes --workspace-root",
	"engine.pin_outdated":                  "bump the pin: go get github.com/labeth/engineering-model-go@<version> in .github/engmod-check",
}

// codeShape is a diagnostic code literal: family.name.
var codeShape = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9_]*)+$`)

// notRequiredSwitches are codes the two required-switch emitters report that
// are no required switch: tier-keyed warnings and the POA&M checks. Every
// other code those files emit must be in the remedy table.
var notRequiredSwitches = map[string]bool{
	"lifecycle.requirement_unapproved": true, "lifecycle.requirement_unverified": true,
	"lifecycle.aviation_profile_missing": true, "lifecycle.software_level_missing": true,
	"lifecycle.poam_defers_core": true, "lifecycle.poam_overdue": true, "lifecycle.poam_defers_unused": true,
}

// emittedBy parses a Go source file of this package and returns the
// diagnostic code literals it emits: the first argument of the add helper and
// the Code field of a Diagnostic literal.
func emittedBy(t *testing.T, file string) map[string]bool {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	ast.Inspect(parsed, func(n ast.Node) bool {
		if e := emittedCodeExpr(n); e != nil {
			takeCode(codes, e)
		}
		return true
	})
	return codes
}

// emittedCodeExpr returns the expression holding a diagnostic code when the
// node is a call of the add helper (its first argument) or the Code field of
// a literal, and nil otherwise.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func emittedCodeExpr(n ast.Node) ast.Expr {
	switch x := n.(type) {
	case *ast.CallExpr:
		if isIdentNamed(x.Fun, "add") && len(x.Args) > 0 {
			return x.Args[0]
		}
	case *ast.KeyValueExpr:
		if isIdentNamed(x.Key, "Code") {
			return x.Value
		}
	}
	return nil
}

// isIdentNamed reports whether an expression is the identifier name.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func isIdentNamed(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

// takeCode records a string literal that has the shape of a diagnostic code.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func takeCode(codes map[string]bool, e ast.Expr) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return
	}
	if v, err := strconv.Unquote(lit.Value); err == nil && codeShape.MatchString(v) {
		codes[v] = true
	}
}

// TestRequiredSwitchesHaveRemedies is the inventory of required switches
// (ADR-EMG-061): each row removes one switch from a complete model and
// expects its code, an error that no tier changes, and the remedy text,
// pinned literally. The codes are derived from the emitters' source: every
// code lifecycle.go and required_switches.go can emit is either in the remedy
// table or listed as not a required switch, and every table code has a row
// and a pinned remedy, so adding a required code without a remedy fails here.
// Mutation: dropping a remedy from requiredSwitches, a code from the rows, or
// adding an emitted code to neither list fails the comparison.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func TestRequiredSwitchesHaveRemedies(t *testing.T) {
	rowCodes := map[string]bool{}
	for _, c := range requiredCases {
		rowCodes[c.code] = true
		t.Run(c.name, func(t *testing.T) { assertRequiredCase(t, c) })
	}
	tableCodes := assertRemedyTable(t, rowCodes)
	assertInventoryCodesHaveRemedies(t, rowCodes, tableCodes)
	assertPinnedRemedies(t, tableCodes)
	emitted := requiredSourceCodes(t)
	assertEmittedCodesAreListed(t, emitted, tableCodes)
	assertSourceScanFindsKnownCodes(t, emitted)
	assertNotRequiredSwitchesAreNotInTheTable(t, tableCodes)
	if remedyFor("requirement.orphan") != "" {
		t.Error("a code outside the table must have no remedy")
	}
}

// assertRequiredCase checks that the row's edit reports its code at its path
// as an undeferred rank-0 error with the remedy.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertRequiredCase(t *testing.T, c requiredCase) {
	t.Helper()
	if c.edit == nil {
		assertExternalCase(t, c)
		return
	}
	b := completeBundle(t)
	c.edit(&b)
	got := codesOf(ApplyTierPolicy(b, Bundle(b)), c.code)
	if len(got) == 0 {
		t.Errorf("%s: %s not reported", c.name, c.code)
		return
	}
	d := got[0]
	assertUndeferredError(t, c, d)
	if d.Path != c.path {
		t.Errorf("%s: path %q, want %q", c.name, d.Path, c.path)
	}
	assertRequiredRemedy(t, c, d)
	assertRankZeroCode(t, c)
}

// assertExternalCase checks a row whose code is emitted outside the validator:
// the remedy a table lookup gives is the pinned one and contains the row's
// words, the code is never tier keyed, and the lock code is a core code.
// TRLC-LINKS: REQ-EMG-179, REQ-EMG-180, REQ-EMG-181
func assertExternalCase(t *testing.T, c requiredCase) {
	t.Helper()
	if got := RequiredRemedy(c.code); got != literalRemedies[c.code] || !strings.Contains(got, c.remedy) {
		t.Errorf("%s: RequiredRemedy %q, want the pinned remedy containing %q", c.name, got, c.remedy)
	}
	if TierKeyed(c.code) || IsCoreCode(c.code) != strings.HasPrefix(c.code, "composition.") {
		t.Errorf("%s: %s must be untiered, and core only for the lock code", c.name, c.code)
	}
}

// assertUndeferredError checks that a diagnostic is an undeferred error.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertUndeferredError(t *testing.T, c requiredCase, d Diagnostic) {
	t.Helper()
	if d.Severity != SeverityError || d.Deferred || d.PoamRef != "" {
		t.Errorf("%s: %s must be an undeferred error, got %+v", c.name, c.code, d)
	}
}

// assertRequiredRemedy checks the remedy of a row: the substring of the row
// and the pinned literal.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertRequiredRemedy(t *testing.T, c requiredCase, d Diagnostic) {
	t.Helper()
	if !strings.Contains(d.Remedy, c.remedy) {
		t.Errorf("%s: remedy %q lacks %q", c.name, d.Remedy, c.remedy)
	}
	if want := literalRemedies[c.code]; want == "" || d.Remedy != want {
		t.Errorf("%s: remedy %q, want the pinned %q", c.name, d.Remedy, want)
	}
}

// assertRankZeroCode checks that the code of a row is core and not tier keyed.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertRankZeroCode(t *testing.T, c requiredCase) {
	t.Helper()
	if !IsCoreCode(c.code) || TierKeyed(c.code) {
		t.Errorf("%s: %s must be a rank-0 core code and not tier keyed", c.name, c.code)
	}
}

// assertRemedyTable checks every required switch of the remedy table against
// the inventory rows and the pinned remedies, and returns the table's codes.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertRemedyTable(t *testing.T, rowCodes map[string]bool) map[string]bool {
	t.Helper()
	tableCodes := map[string]bool{}
	for _, s := range RequiredSwitches() {
		tableCodes[s.Code] = true
		if strings.TrimSpace(s.Switch) == "" || strings.TrimSpace(s.Remedy) == "" {
			t.Errorf("required switch %+v has no name or remedy", s)
		}
		if !rowCodes[s.Code] {
			t.Errorf("remedy table code %s has no inventory row", s.Code)
		}
		if s.Remedy != literalRemedies[s.Code] {
			t.Errorf("table remedy of %s is %q, pinned %q", s.Code, s.Remedy, literalRemedies[s.Code])
		}
	}
	return tableCodes
}

// assertInventoryCodesHaveRemedies checks that every inventory code is in the
// remedy table.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertInventoryCodesHaveRemedies(t *testing.T, rowCodes, tableCodes map[string]bool) {
	t.Helper()
	for code := range rowCodes {
		if !tableCodes[code] {
			t.Errorf("inventory code %s has no remedy in the table", code)
		}
	}
}

// assertPinnedRemedies checks that there is one pinned remedy per table code
// and that none names a command that does not exist.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertPinnedRemedies(t *testing.T, tableCodes map[string]bool) {
	t.Helper()
	if len(literalRemedies) != len(tableCodes) {
		t.Errorf("%d pinned remedies for %d table codes", len(literalRemedies), len(tableCodes))
	}
	for code, text := range literalRemedies {
		// engflow upgrade exists since ADR-EMG-072; only the remedy of the
		// stale check template may name it.
		if (strings.Contains(text, "engflow upgrade") || strings.Contains(text, "run engflow")) && code != "engine.check_template_stale" {
			t.Errorf("remedy of %s names a command that does not exist: %q", code, text)
		}
	}
}

// requiredSourceCodes returns the codes the emitters can report, read from their
// source.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func requiredSourceCodes(t *testing.T) map[string]bool {
	t.Helper()
	emitted := map[string]bool{}
	for _, file := range []string{"required_switches.go", "lifecycle.go"} {
		for code := range emittedBy(t, file) {
			emitted[code] = true
		}
	}
	return emitted
}

// assertEmittedCodesAreListed checks that every emitted code has a remedy or
// a not-a-switch entry.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertEmittedCodesAreListed(t *testing.T, emitted, tableCodes map[string]bool) {
	t.Helper()
	var unlisted []string
	for code := range emitted {
		if !tableCodes[code] && !notRequiredSwitches[code] {
			unlisted = append(unlisted, code)
		}
	}
	sort.Strings(unlisted)
	if len(unlisted) > 0 {
		t.Errorf("emitted codes with neither a remedy nor a not-a-switch entry: %v", unlisted)
	}
}

// assertSourceScanFindsKnownCodes checks that the source scan finds codes
// known to be emitted: a broken scan would pass the comparison vacuously.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertSourceScanFindsKnownCodes(t *testing.T, emitted map[string]bool) {
	t.Helper()
	for _, code := range []string{"lifecycle.missing", "lifecycle.tier_decision_mismatch", "health.block_missing", "model.id_missing", "lint.mode_guided_at_tier", "lifecycle.poam_defers_core", "lifecycle.poam_overdue"} {
		if !emitted[code] {
			t.Errorf("source scan did not find %s: the scan is broken", code)
		}
	}
}

// assertNotRequiredSwitchesAreNotInTheTable checks that no code is both a
// required switch and listed as not one.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func assertNotRequiredSwitchesAreNotInTheTable(t *testing.T, tableCodes map[string]bool) {
	t.Helper()
	for code := range notRequiredSwitches {
		if tableCodes[code] {
			t.Errorf("%s is listed as both a required switch and not one", code)
		}
	}
}

// Each required code is rank 0 end to end: an open POA&M item at the
// project tier cannot defer it, lifecycle.raise changes nothing, and the
// POA&M that names it is itself an error. A control entry for a deferrable
// code in the same item proves the item is honoured, so the refusal is the
// code's and not the item's. A model without lifecycle (lifecycle.missing)
// still reports the defer: the POA&M check does not depend on the block.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-129
func TestRequiredSwitchesCannotBeDeferred(t *testing.T) {
	for _, c := range requiredCases {
		if c.edit == nil && !IsCoreCode(c.code) {
			continue // the engine.* codes are untiered errors outside the tier policy
		}
		t.Run(c.name, func(t *testing.T) { assertRequiredCodeCannotBeDeferred(t, c) })
	}
}

// deferringBundle is the row's model with a POA&M item that defers the row's
// code and a control code, at the model's own tier (even the unknown one: the
// most adversarial item for that row), and lifecycle.raise naming the code.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-129
func deferringBundle(t *testing.T, c requiredCase) model.Bundle {
	t.Helper()
	b := completeBundle(t)
	if c.edit != nil {
		c.edit(&b)
	}
	tier := "mvp"
	if lc := b.Manifest.Lifecycle; lc != nil {
		tier = strings.ToLower(lc.Tier)
		lc.Raise = []string{c.code}
	}
	b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: "POAM-X", Milestone: "tier:" + tier, Status: "in-progress",
		Defers: []model.POAMDefer{{Code: c.code}, {Code: "requirement.orphan"}}}}
	return b
}

// assertRequiredCodeCannotBeDeferred checks that the tier policy leaves the
// row's code alone, honours the control defer and reports the defers entry.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-129
func assertRequiredCodeCannotBeDeferred(t *testing.T, c requiredCase) {
	t.Helper()
	b := deferringBundle(t, c)
	in := []Diagnostic{
		{Code: c.code, Severity: SeverityError, Path: c.path},
		{Code: "requirement.orphan", Severity: SeverityWarning, Path: "requirements[0]"},
	}
	out := ApplyTierPolicy(b, in)
	if out[0].Severity != SeverityError || out[0].Deferred || out[0].PoamRef != "" {
		t.Errorf("%s: %s was changed by the tier policy: %+v", c.name, c.code, out[0])
	}
	_, _, hasTier := EffectiveTier(b)
	assertControlDefer(t, hasTier, out[1])
	if got := codesOf(Bundle(b), "lifecycle.poam_defers_core"); len(got) != 1 {
		t.Errorf("%s: a defer naming %s must be lifecycle.poam_defers_core, got %+v", c.name, c.code, got)
	}
}

// assertControlDefer checks the control diagnostic: deferred by the item when
// the model has a valid tier, untouched when it has none.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-129
func assertControlDefer(t *testing.T, hasTier bool, control Diagnostic) {
	t.Helper()
	honoured := control.Deferred && control.PoamRef == "POAM-X"
	applied := control.Deferred || control.PoamRef != ""
	if hasTier && !honoured {
		t.Errorf("the control defer was not honoured, so the test proves nothing: %+v", control)
	}
	if !hasTier && applied {
		t.Errorf("without a valid tier no defer may apply: %+v", control)
	}
}

// A defer of lifecycle.missing is reported even though the model has no
// lifecycle block (the POA&M loop runs before the nil lifecycle return).
// Mutation: returning early on a nil lifecycle before the POA&M loop leaves
// only lifecycle.missing.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-129
func TestPoamDefersCoreIsReportedWithoutLifecycle(t *testing.T) {
	b := completeBundle(t)
	b.Manifest.Lifecycle = nil
	b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: "POAM-X", Milestone: "tier:mvp", Status: "open", Defers: []model.POAMDefer{{Code: "lifecycle.missing"}}}}
	diags := Bundle(b)
	if got := codesOf(diags, "lifecycle.poam_defers_core"); len(got) != 1 || got[0].Severity != SeverityError {
		t.Fatalf("poam_defers_core = %+v", got)
	}
	if got := codesOf(diags, "lifecycle.missing"); len(got) != 1 {
		t.Fatalf("lifecycle.missing = %+v", got)
	}
}

// A model without lifecycle is lifecycle.missing, an error at the
// lifecycle path; the library keeps loading it (no tier rule applies) and
// every other switch is still checked.
// TRLC-LINKS: REQ-EMG-124
func TestLifecycleMissingIsAnErrorAndTheLibraryStaysLenient(t *testing.T) {
	b := completeBundle(t)
	b.Manifest.Lifecycle = nil
	diags := ApplyTierPolicy(b, Bundle(b))
	if got := errorCodes(diags); len(got) != 1 || got[0] != "lifecycle.missing" {
		t.Fatalf("errors = %v, want only lifecycle.missing", got)
	}
	if _, _, ok := EffectiveTier(b); ok {
		t.Error("EffectiveTier must stay ok=false without lifecycle (library behaviour)")
	}
	in := []Diagnostic{{Code: "requirement.orphan", Severity: SeverityWarning}}
	if out := ApplyTierPolicy(b, in); out[0].Severity != SeverityWarning || out[0].RequiredFrom != "" {
		t.Errorf("ApplyTierPolicy must return diagnostics unchanged without lifecycle: %+v", out[0])
	}
}

// The tier decision must carry a Bundles: line; a decision without one no
// longer means foundation.
// TRLC-LINKS: REQ-EMG-124
func TestTierDecisionNeedsABundlesLine(t *testing.T) {
	b := completeBundle(t)
	if got := codesOf(Bundle(b), "lifecycle.tier_decision_mismatch"); len(got) != 0 {
		t.Fatalf("complete bundle: %+v", got)
	}
	b.Architecture.Decisions[0].Decision = "Tier: MVP"
	got := codesOf(Bundle(b), "lifecycle.tier_decision_mismatch")
	if len(got) != 1 || !strings.Contains(got[0].Message, "Bundles:") {
		t.Fatalf("got %+v, want one mismatch naming the Bundles: line", got)
	}
}

// The health block is required at product and operational and nowhere
// below; an empty block satisfies it. The expectation is a literal table.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func TestHealthBlockRequiredFromProduct(t *testing.T) {
	want := map[string]bool{"poc": false, "prototype": false, "mvp": false, "product": true, "operational": true}
	for tier, required := range want {
		b := completeBundleAt(tier)
		b.Manifest.Health = nil
		if got := len(codesOf(Bundle(b), "health.block_missing")) == 1; got != required {
			t.Errorf("%s without health block: reported=%v, want %v", tier, got, required)
		}
		b.Manifest.Health = &model.ManifestHealth{}
		if len(codesOf(Bundle(b), "health.block_missing")) != 0 {
			t.Errorf("%s with an empty health block still reports health.block_missing", tier)
		}
	}
	if len(want) != len(model.Tiers) {
		t.Errorf("the table covers %d tiers, the model has %d", len(want), len(model.Tiers))
	}
}

// lintRun.mode guided is allowed at poc and prototype and an error at mvp,
// product and operational; strict is accepted everywhere (any case, padded)
// and an unknown mode is lint.mode_invalid. The expectation is a literal.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func TestLintModeByTier(t *testing.T) {
	rejected := map[string]bool{"poc": false, "prototype": false, "mvp": true, "product": true, "operational": true}
	for tier, want := range rejected {
		b := completeBundleAt(tier)
		b.Requirements.LintRun.Mode = "Guided"
		if got := len(codesOf(Bundle(b), "lint.mode_guided_at_tier")) == 1; got != want {
			t.Errorf("guided at %s: reported=%v, want %v", tier, got, want)
		}
		b.Requirements.LintRun.Mode = " STRICT "
		if got := Bundle(b); len(codesOf(got, "lint.mode_guided_at_tier"))+len(codesOf(got, "lint.mode_invalid"))+len(codesOf(got, "lint.mode_missing")) != 0 {
			t.Errorf("strict at %s reported a lint mode error", tier)
		}
		b.Requirements.LintRun.Mode = "lenient"
		if len(codesOf(Bundle(b), "lint.mode_invalid")) != 1 {
			t.Errorf("unknown mode at %s is not lint.mode_invalid", tier)
		}
	}
	if len(rejected) != len(model.Tiers) {
		t.Errorf("the table covers %d tiers, the model has %d", len(rejected), len(model.Tiers))
	}
}

// requirementLinks guided is rejected (lifecycle.requirement_links_conflict)
// at mvp, product and operational and allowed at poc and prototype, in any
// case and padding; an empty value is model.requirement_links_missing at
// every tier. The schema-version error carries the manual fix and names no
// command that does not exist. The expectations are literals.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-127
func TestRequirementLinksRequiredAtEveryTier(t *testing.T) {
	conflict := map[string]bool{"poc": false, "prototype": false, "mvp": true, "product": true, "operational": true}
	for tier, want := range conflict {
		t.Run(tier, func(t *testing.T) { assertRequirementLinksAtTier(t, tier, want) })
	}
	if len(conflict) != len(model.Tiers) {
		t.Errorf("the table covers %d tiers, the model has %d", len(conflict), len(model.Tiers))
	}
	assertSchemaVersionErrorCarriesTheFix(t)
}

// assertRequirementLinksAtTier checks the requirementLinks rules at a tier:
// empty is missing, guided conflicts when want, strict never conflicts.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-127
func assertRequirementLinksAtTier(t *testing.T, tier string, want bool) {
	t.Helper()
	b := completeBundleAt(tier)
	b.Manifest.RequirementLinks = ""
	if len(codesOf(Bundle(b), "model.requirement_links_missing")) != 1 {
		t.Errorf("%s: empty requirementLinks not reported", tier)
	}
	for _, guided := range []string{"guided", " Guided "} {
		b.Manifest.RequirementLinks = guided
		if got := len(codesOf(Bundle(b), "lifecycle.requirement_links_conflict")) == 1; got != want {
			t.Errorf("%q at %s: conflict=%v, want %v", guided, tier, got, want)
		}
	}
	b.Manifest.RequirementLinks = "strict"
	if len(codesOf(Bundle(b), "lifecycle.requirement_links_conflict")) != 0 {
		t.Errorf("strict at %s reported a conflict", tier)
	}
}

// assertSchemaVersionErrorCarriesTheFix checks that the schema-version error
// carries the manual fix and names no command that does not exist.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-127
func assertSchemaVersionErrorCarriesTheFix(t *testing.T) {
	t.Helper()
	_, err := model.EffectiveSchemaVersion(1)
	var versionErr *model.SchemaVersionError
	if !errors.As(err, &versionErr) || !strings.Contains(err.Error(), "fix: set schemaVersion: 3") || strings.Contains(err.Error(), "run engflow") {
		t.Fatalf("schema version error = %v", err)
	}
}

// The remedy is a json field and prints as a fix line after the diagnostic;
// a diagnostic without a remedy prints no fix line.
// TRLC-LINKS: REQ-EMG-125
func TestDiagnosticRemedyIsRenderedAndSerialized(t *testing.T) {
	b := completeBundle(t)
	b.Manifest.Lifecycle = nil
	diag := codesOf(Bundle(b), "lifecycle.missing")[0]
	var out bytes.Buffer
	WriteDiagnostic(&out, diag)
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "lifecycle.missing [error] ") || !strings.HasSuffix(lines[0], "(lifecycle)") || lines[1] != "  fix: "+remedyFor("lifecycle.missing") {
		t.Fatalf("rendered %q", out.String())
	}
	data, err := json.Marshal(diag)
	if err != nil || !strings.Contains(string(data), `"remedy":"run the ef-tier-up skill`) {
		t.Fatalf("json = %s, %v", data, err)
	}
	out.Reset()
	WriteDiagnostic(&out, Diagnostic{Code: "x.y", Severity: SeverityWarning, Message: "m"})
	if out.String() != "x.y [warning] m\n" {
		t.Fatalf("rendered %q", out.String())
	}
	if data, _ := json.Marshal(Diagnostic{Code: "x.y"}); strings.Contains(string(data), "remedy") {
		t.Fatalf("an empty remedy must be omitted: %s", data)
	}
}

// The authoring contract rules list the required switches, and the README
// "Required model switches" table is the current one.
// TRLC-LINKS: REQ-EMG-045, REQ-EMG-124, REQ-EMG-125
func TestRequiredSwitchesAreInTheContractAndTheReadme(t *testing.T) {
	rules := AuthoringRules()
	if !reflect.DeepEqual(rules.Required, RequiredSwitches()) || len(rules.Required) == 0 {
		t.Fatalf("rules.required = %+v", rules.Required)
	}
	for _, r := range rules.Required {
		if r.Switch == "" || r.Code == "" || r.Remedy == "" {
			t.Errorf("incomplete required row %+v", r)
		}
	}
	data, err := json.Marshal(rules)
	if err != nil || !strings.Contains(string(data), `"required":[{"switch":"engmod.yml lifecycle","code":"lifecycle.missing","remedy":"run the ef-tier-up skill`) {
		t.Fatalf("rules json lacks the required section: %v", err)
	}
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if table := RequiredSwitchesMarkdown(); !strings.Contains(string(readme), table) {
		t.Fatalf("README.md does not contain the current RequiredSwitchesMarkdown() table:\n%s", table)
	}
}
