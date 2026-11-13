// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// The remedy texts of the required model switches (ADR-EMG-061): the skill
// to run or the command to execute. They are the only place that names them,
// so a change of skill or command is one edit here.
const (
	remedyTier    = "run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision}"
	remedyLinks   = "run the ef-tier-up skill (engineering-flow) to choose the tier, then set requirementLinks: strict in engmod.yml (guided is allowed only below mvp)"
	remedyLint    = "run the ef-tier-up skill (engineering-flow) to choose the tier, then set lintRun.mode: strict in requirements.yml (guided is allowed only below mvp)"
	remedyHealth  = "run the ef-tier-up skill (engineering-flow) to write the health block with its debt list, then record the baseline with enghealth baseline write"
	remedyModelID = "set module.modelId in engmod.yml to the stable identifier of the model (for example MY-SYSTEM)"
	// The remedies of the engine side of removal phase 1 (ADR-EMG-072). The
	// emitters replace <dir> and <version> with the repository and the
	// running engine version.
	remedyLockSchema    = "regenerate the lock with the current engine: run engws bump <repo> <alias> <version> (it rewrites engmod.lock.yml with the current schemaVersion), then engws sync"
	remedyCheckTemplate = "run engflow upgrade --root <dir> to replace .github/engmod-check with the current template that takes --workspace-root"
	remedyEnginePin     = "bump the pin: go get github.com/labeth/engineering-model-go@<version> in .github/engmod-check"
)

// remedyUpgrade is the schemaVersion remedy. It names only what exists: the
// manual edit to the current version. The upgrade command of engineering-flow
// is planned and not built, so the text says so instead of naming it as a
// command to run.
var remedyUpgrade = model.SchemaVersionRemedy

// RequiredSwitch is one row of the required-switch table: the switch, the
// code reported when it is absent or invalid, and the remedy. The table is
// in the order the authoring contract and the README show it.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
type RequiredSwitch struct {
	Switch string `json:"switch"`
	Code   string `json:"code"`
	Remedy string `json:"remedy"`
}

// requiredSwitches is the one table of the required model switches
// (ADR-EMG-061): an absent or invalid switch is an error, and the error says
// how to fix it. TestRequiredSwitchesHaveRemedies keeps it equal to the
// inventory of switches the validator reports, so a required code cannot be
// added without a remedy.
var requiredSwitches = []RequiredSwitch{
	{"engmod.yml lifecycle", "lifecycle.missing", remedyTier},
	{"lifecycle.tier", "lifecycle.invalid_tier", remedyTier},
	{"lifecycle.bundles", "lifecycle.bundles_missing", remedyTier},
	{"lifecycle.bundles (value)", "lifecycle.invalid_bundle", remedyTier},
	{"lifecycle.tierDecision", "lifecycle.tier_decision_missing", remedyTier},
	{"lifecycle.tierDecision (accepted decision with Tier: and Bundles: lines)", "lifecycle.tier_decision_mismatch", remedyTier},
	{"engmod.yml requirementLinks", "model.requirement_links_missing", remedyLinks},
	{"engmod.yml requirementLinks (value)", "model.invalid_requirement_links", remedyLinks},
	{"requirementLinks guided at mvp or above", "lifecycle.requirement_links_conflict", remedyLinks},
	{"requirements.yml lintRun.mode", "lint.mode_missing", remedyLint},
	{"lintRun.mode (value)", "lint.mode_invalid", remedyLint},
	{"lintRun.mode guided at mvp or above", "lint.mode_guided_at_tier", remedyLint},
	{"engmod.yml health block (tier product and operational)", "health.block_missing", remedyHealth},
	{fmt.Sprintf("schemaVersion %d in every document", model.CurrentSchemaVersion), "model.unsupported_schema_version", remedyUpgrade},
	{"module.modelId", "model.id_missing", remedyModelID},
	{"engmod.lock.yml schemaVersion (the version the engine writes)", "composition.lock_schema_outdated", remedyLockSchema},
	{"repository .github/engmod-check that takes --workspace-root", "engine.check_template_stale", remedyCheckTemplate},
	{"engineering-model-go pin of .github/engmod-check (not older than the running engine)", "engine.pin_outdated", remedyEnginePin},
}

// RequiredSwitches returns the table of required model switches with the
// code reported when one is absent and its remedy, for the authoring
// contract and the README.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-168, REQ-EMG-169
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func RequiredSwitches() []RequiredSwitch {
	return append([]RequiredSwitch(nil), requiredSwitches...)
}

// RequiredSwitchesMarkdown renders RequiredSwitches as the Markdown table of
// the README: the switch, the error code and the remedy.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-168, REQ-EMG-169
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func RequiredSwitchesMarkdown() string {
	var b strings.Builder
	b.WriteString("| Switch | Error code | Remedy |\n|---|---|---|\n")
	for _, s := range requiredSwitches {
		b.WriteString("| " + s.Switch + " | `" + s.Code + "` | " + s.Remedy + " |\n")
	}
	return b.String()
}

// remedyFor returns the remedy of a required-switch code, or "" for a code
// that is no required switch.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-168, REQ-EMG-169
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func remedyFor(code string) string {
	for _, s := range requiredSwitches {
		if s.Code == code {
			return s.Remedy
		}
	}
	return ""
}

// RequiredRemedy returns the remedy of a required-switch code, or "" for a
// code that is no required switch: the form the emitters outside this
// package (composition, engws) use to set Diagnostic.Remedy.
// TRLC-LINKS: REQ-EMG-179, REQ-EMG-180, REQ-EMG-181
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func RequiredRemedy(code string) string {
	return remedyFor(code)
}

// withRemedies sets the remedy of every diagnostic whose code is a required
// switch; it never overwrites a remedy a rule set itself.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-168, REQ-EMG-169
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func withRemedies(diags []Diagnostic) []Diagnostic {
	for i := range diags {
		if diags[i].Remedy == "" {
			diags[i].Remedy = remedyFor(diags[i].Code)
		}
	}
	return diags
}

// The remedies of the consistency checks (ADR-EMG-069): the skill that
// writes the missing link, or the edit that fixes the model.
const (
	remedyOwnerUnit     = "run the ef-code-linking skill (engineering-flow) to add the ENGMODEL-OWNER-UNIT: FU-* marker, naming a functional unit of the model, to the file header"
	remedyModelLinkKind = "run the ef-code-linking skill (engineering-flow) to move a requirement ID from ENGMODEL-LINKS to TRLC-LINKS, or to name the model element (unit, interface, machine, state, flow) the code realizes"
	remedyUnmodeled     = "run the ef-behavior skill (engineering-flow) to model the requirement as a state, transition or event of a machine, or add an invariant that lists it in requirements, or name the element in the requirement's appliesTo"
	remedyBehaviorLink  = "run the ef-behavior skill (engineering-flow) to list the requirements the state, transition or event realizes in its requirements field, or to model the behavior the requirement asks for"
	remedyMachineVerify = "run the ef-formal-behavior skill (engineering-flow) to add a machine invariant and run engquint check, so each invariant has a current formal result"
	// remedyCodeCoverage is the remedy of every element to code coverage code.
	remedyCodeCoverage = "run the ef-code-linking skill (engineering-flow) to name the element in an ENGMODEL-LINKS marker above the declaration that realizes it, or in the ENGMODEL-OWNER-UNIT marker of its file; when it is deliberately not implemented here, run the ef-tier-up skill (engineering-flow) to record an accepted decision and list the element in codeCoverage.exempt of engmod.yml"
	// remedyDependencyTier is the remedy of composition.dependency_tier_unknown
	// (ADR-EMG-073): the dependency's owner declares and releases lifecycle.
	remedyDependencyTier = "ask the owner of the dependency to declare lifecycle {tier, bundles, tierDecision} in its engmod.yml and release it, then run engws bump to pin the new release"
	// The remedies of the removed fields (ADR-EMG-075): the edit that deletes
	// the field, as for model.code_sources_removed.
	remedyPinnedFingerprint = "delete pinned.fingerprint; pin the dependency version only"
	remedyProvidesRef       = "name the realizing requirement with realizes, not ref"
	// remedyAllocations is the remedy of model.allocations_removed (ADR-EMG-078).
	remedyAllocations = "write requirement links (reqLinks) instead of composition.allocations"
	// remedySchemaVersionNotBumped is the remedy of release.schema_version_not_bumped
	// (ADR-EMG-077), reported by engws release-check.
	remedySchemaVersionNotBumped = "raise schemaVersion and model.CurrentSchemaVersion, then record the change in the CHANGELOG"
	// The remedies of the state-like code discovery (ADR-EMG-058, ADR-EMG-070).
	remedyStateCandidate  = "model the state with the ef-behavior skill (a machine whose states and transitions are in the model) and link the declaration with the ef-code-linking skill (an ENGMODEL-LINKS marker naming the SM-*, STATE-* or TR-* ID on or above the declaration); when the type is not a state, write a comment ENGMODEL-NOT-A-STATE: followed by a reason of at least 20 characters on or within three lines above the declaration"
	remedyStateUnreasoned = "write after ENGMODEL-NOT-A-STATE: a reason of at least 20 characters that says why the type is not a state; a placeholder such as TODO, TBD or n/a is not a reason"
	remedyStateUnused     = "remove the ENGMODEL-NOT-A-STATE comment: the declaration under it is not a state-like candidate, or it already links a machine"
	remedyExemption       = "run the ef-tier-up skill (engineering-flow) to record or accept the decision that justifies the exemption, then name it in codeCoverage.exempt {id, reason, decision} of engmod.yml; remove an entry that is no longer needed"
)

// consistencyRemedies maps the codes of the consistency checks to their
// remedy.
var consistencyRemedies = map[string]string{
	"code.owner_unresolved":             remedyOwnerUnit,
	"code.model_link_kind_mismatch":     remedyModelLinkKind,
	"behavior.requirement_unmodeled":    remedyUnmodeled,
	"behavior.machine_unverified":       remedyMachineVerify,
	"behavior.state_unlinked":           remedyBehaviorLink,
	"behavior.transition_unlinked":      remedyBehaviorLink,
	"behavior.event_unlinked":           remedyBehaviorLink,
	"code.unit_unlinked":                remedyCodeCoverage,
	"code.machine_unlinked":             remedyCodeCoverage,
	"code.interface_unlinked":           remedyCodeCoverage,
	"code.state_unlinked":               remedyCodeCoverage,
	"code.transition_unlinked":          remedyCodeCoverage,
	"code.event_unlinked":               remedyCodeCoverage,
	"code.flow_unlinked":                remedyCodeCoverage,
	"code.data_unlinked":                remedyCodeCoverage,
	"code.state_candidate_unlinked":     remedyStateCandidate,
	"code.state_suppression_unreasoned": remedyStateUnreasoned,
	"code.state_suppression_unused":     remedyStateUnused,
	"code.exemption_unused":             remedyExemption,
	"code.exemption_unjustified":        remedyExemption,
	"model.invalid_code_exemption":      remedyExemption,

	"composition.dependency_tier_unknown": remedyDependencyTier,

	"model.pinned_fingerprint_removed": remedyPinnedFingerprint,
	"model.provides_ref_removed":       remedyProvidesRef,
	"model.allocations_removed":        remedyAllocations,

	"release.schema_version_not_bumped": remedySchemaVersionNotBumped,
}

// ConsistencyRemedy returns the remedy of a consistency-check code
// (ADR-EMG-069) or of a removed-field code (ADR-EMG-075, ADR-EMG-078), or "" for any other
// code. The emitting packages set it on their diagnostics, because the trace
// and the requirement lint do not pass through Bundle.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-184
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func ConsistencyRemedy(code string) string {
	return consistencyRemedies[code]
}
