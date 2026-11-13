// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// The remedy texts of the required model switches (ADR-EMG-061): the skill
// to run or the command to execute. They are the only place that names them,
// so a change of skill or command is one edit here.
const (
	remedyTier = "run the ef-tier-up skill (engineering-flow) to choose and record the tier, then write engmod.yml lifecycle {tier, bundles, tierDecision}"
	// remedyUpgrade is the schemaVersion remedy. It names only what exists:
	// the manual edit. The upgrade command of engineering-flow is planned and
	// not built, so the text says so instead of naming it as a command to run.
	remedyUpgrade = model.SchemaVersionRemedy
	remedyLinks   = "run the ef-tier-up skill (engineering-flow) to choose the tier, then set requirementLinks: strict in engmod.yml (guided is allowed only below mvp)"
	remedyLint    = "run the ef-tier-up skill (engineering-flow) to choose the tier, then set lintRun.mode: strict in requirements.yml (guided is allowed only below mvp)"
	remedyHealth  = "run the ef-tier-up skill (engineering-flow) to write the health block with its debt list, then record the baseline with enghealth baseline write"
	remedyModelID = "set module.modelId in engmod.yml to the stable identifier of the model (for example MY-SYSTEM)"
)

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
	{"schemaVersion 2 in every document", "model.unsupported_schema_version", remedyUpgrade},
	{"module.modelId", "model.id_missing", remedyModelID},
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
	remedyMachineVerify = "run the ef-formal-behavior skill (engineering-flow) to add a machine invariant and run engquint check, so each invariant has a current formal result"
)

// consistencyRemedies maps the codes of the consistency checks to their
// remedy.
var consistencyRemedies = map[string]string{
	"code.owner_unresolved":          remedyOwnerUnit,
	"code.model_link_kind_mismatch":  remedyModelLinkKind,
	"behavior.requirement_unmodeled": remedyUnmodeled,
	"behavior.machine_unverified":    remedyMachineVerify,
}

// ConsistencyRemedy returns the remedy of a consistency-check code
// (ADR-EMG-069), or "" for any other code. The emitting packages set it on
// their diagnostics, because the trace and the requirement lint do not pass
// through Bundle.
// TRLC-LINKS: REQ-EMG-125
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func ConsistencyRemedy(code string) string {
	return consistencyRemedies[code]
}
