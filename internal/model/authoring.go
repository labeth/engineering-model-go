// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"reflect"
	"strings"
)

// AuthoringContractVersion identifies the contract shape. v3 added the
// machines guide (expression grammar, operator table, Quint workflow and
// formal evidence semantics) and the SM, TR and INV identifier prefixes. v4
// adds the health guide, the manifest health block and the health field sets.
// v5 adds health.baseline (the committed baseline of accepted debt), the
// healthBaseline field set and the codes health.baseline_stale and
// health.baseline_unreadable. v6 adds health.policyDecision, the code
// health.policy_weakened and the metric value of a baseline entry.
const AuthoringContractVersion = "engineering-model.authoring.v6"

// AuthoringContract is compact machine context for editing the canonical YAML set.
type AuthoringContract struct {
	ContractVersion   string                `json:"contractVersion"`
	CanonicalFormat   string                `json:"canonicalFormat"`
	SchemaAuthority   string                `json:"schemaAuthority"`
	Documents         []AuthoringDocument   `json:"documents"`
	StableIdentifiers StableIdentifierGuide `json:"stableIdentifiers"`
	Dependencies      DependencyGuide       `json:"dependencies"`
	Machines          MachineGuide          `json:"machines"`
	Health            HealthGuide           `json:"health"`
	AuthoringOrder    []string              `json:"authoringOrder"`
	Invariants        []string              `json:"invariants"`
	Compatibility     []string              `json:"compatibility"`
}

type AuthoringDocument struct {
	Kind           string   `json:"kind"`
	Path           string   `json:"path"`
	SchemaVersion  int      `json:"schemaVersion"`
	SchemaPath     string   `json:"schemaPath"`
	Required       bool     `json:"required"`
	TopLevelFields []string `json:"topLevelFields"`
}

type StableIdentifierGuide struct {
	Pattern  string            `json:"pattern"`
	Prefixes map[string]string `json:"prefixes"`
}

// MachineGuide is the authoring contract of behavior.machines beyond the
// field sets and diagnostics that the validation rules list: the expression
// grammar and operator table (derived from the table the parser, validator and
// Quint generator share), the Quint workflow and the formal evidence
// semantics.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-MODEL-AUTHORING-CONTRACT
type MachineGuide struct {
	Grammar   []string             `json:"grammar"`
	Types     []string             `json:"types"`
	Operators []MachineOperatorDoc `json:"operators"`
	Workflow  []string             `json:"workflow"`
	Evidence  []string             `json:"evidence"`
}

// MachineOperatorDoc is one row of the machine expression operator table.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-MODEL-AUTHORING-CONTRACT
type MachineOperatorDoc struct {
	Operator         string   `json:"operator"`
	Syntax           string   `json:"syntax,omitempty"`
	Form             string   `json:"form"`
	Operands         []string `json:"operands"`
	Variadic         bool     `json:"variadic,omitempty"`
	SameType         bool     `json:"sameType,omitempty"`
	Result           string   `json:"result"`
	Precedence       int      `json:"precedence,omitempty"`
	RightAssociative bool     `json:"rightAssociative,omitempty"`
}

// HealthGuide is the authoring contract of the optional engmod.yml health
// block (ADR-EMG-051): the shape of the block, the health codes and the
// rules that validate it. The field sets healthThreshold, healthOverride and
// healthWaiver in the validation rules list the accepted keys.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-MODEL-AUTHORING-CONTRACT
type HealthGuide struct {
	Config []string        `json:"config"`
	Codes  []HealthCodeDoc `json:"codes"`
	Rules  []string        `json:"rules"`
}

// HealthCodeDoc is one health diagnostic code. Threshold is true for the
// smell codes that take numeric warn and high limits.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-MODEL-AUTHORING-CONTRACT
type HealthCodeDoc struct {
	Code      string `json:"code"`
	Meaning   string `json:"meaning"`
	Threshold bool   `json:"threshold,omitempty"`
}

type DependencyGuide struct {
	PreferredSourceField string   `json:"preferredSourceField"`
	SourceKinds          []string `json:"sourceKinds"`
	WorkspaceFile        string   `json:"workspaceFile"`
	WorkspaceSchemaPath  string   `json:"workspaceSchemaPath"`
	RegistryEnvironment  string   `json:"registryEnvironment"`
	CacheDirectory       string   `json:"cacheDirectory"`
	Rules                []string `json:"rules"`
}

// BuildAuthoringContract describes the loaded input model without copying its domain graph.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-045, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-053, REQ-EMG-094, REQ-EMG-148
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func BuildAuthoringContract(bundle Bundle) AuthoringContract {
	baseDir := filepath.Dir(bundle.ManifestPath)
	relative := func(path string) string {
		if rel, err := filepath.Rel(baseDir, path); err == nil {
			return filepath.ToSlash(rel)
		}
		return filepath.ToSlash(path)
	}
	documents := []AuthoringDocument{
		{Kind: "manifest", Path: relative(bundle.ManifestPath), SchemaVersion: contractSchemaVersion(bundle.Manifest.SchemaVersion), SchemaPath: "internal/model/schema/manifest.cue", Required: true, TopLevelFields: yamlTopLevelFields(ManifestDocument{})},
		{Kind: "catalog", Path: relative(bundle.CatalogPath), SchemaVersion: contractSchemaVersion(bundle.Catalog.SchemaVersion), SchemaPath: "internal/model/schema/catalog.cue", Required: true, TopLevelFields: yamlTopLevelFields(CatalogDocument{})},
		{Kind: "requirements", Path: relative(bundle.RequirementsPath), SchemaVersion: contractSchemaVersion(bundle.Requirements.SchemaVersion), SchemaPath: "internal/model/schema/requirements.cue", Required: true, TopLevelFields: yamlTopLevelFields(RequirementsDocument{})},
		{Kind: "architecture", Path: relative(bundle.ArchitecturePath), SchemaVersion: contractSchemaVersion(bundle.Architecture.SchemaVersion), SchemaPath: "internal/model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(ArchitectureInputDocument{})},
		{Kind: "behavior", Path: relative(bundle.BehaviorPath), SchemaVersion: contractSchemaVersion(bundle.Behavior.SchemaVersion), SchemaPath: "internal/model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(BehaviorDocument{})},
		{Kind: "assurance", Path: relative(bundle.AssurancePath), SchemaVersion: contractSchemaVersion(bundle.Assurance.SchemaVersion), SchemaPath: "internal/model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(AssuranceDocument{})},
		{Kind: "compliance", Path: relative(bundle.CompliancePath), SchemaVersion: contractSchemaVersion(bundle.Compliance.SchemaVersion), SchemaPath: "internal/model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(ComplianceDocument{})},
		{Kind: "views", Path: relative(bundle.ViewsPath), SchemaVersion: contractSchemaVersion(bundle.Views.SchemaVersion), SchemaPath: "internal/model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(ViewsDocument{})},
		{Kind: "decisions", Path: relative(bundle.DecisionsPath), SchemaVersion: contractSchemaVersion(bundle.Decisions.SchemaVersion), SchemaPath: "internal/model/schema/decisions.cue", Required: true, TopLevelFields: yamlTopLevelFields(DecisionsDocument{})},
	}
	if bundle.Aviation != nil {
		documents = append(documents, AuthoringDocument{Kind: "aviation", Path: relative(bundle.AviationPath), SchemaVersion: contractSchemaVersion(bundle.Aviation.SchemaVersion), SchemaPath: "internal/model/schema/aviation.cue", Required: false, TopLevelFields: yamlTopLevelFields(AviationDocument{})})
	}
	if bundle.ReqLinksPath != "" {
		documents = append(documents, AuthoringDocument{Kind: "reqLinks", Path: relative(bundle.ReqLinksPath), SchemaVersion: contractSchemaVersion(bundle.ReqLinks.SchemaVersion), SchemaPath: "internal/model/schema/reqlinks.cue", Required: false, TopLevelFields: yamlTopLevelFields(ReqLinksDocument{})})
	}
	return AuthoringContract{
		ContractVersion: AuthoringContractVersion,
		CanonicalFormat: "YAML",
		SchemaAuthority: "CUE",
		Documents:       documents,
		StableIdentifiers: StableIdentifierGuide{
			Pattern: "^[A-Z][A-Z0-9-]*$",
			Prefixes: map[string]string{
				"REQ": "requirement", "FU": "functional unit", "FG": "functional group",
				"IF": "interface", "DO": "data object", "FLOW": "flow", "ACT": "actor",
				"CTRL": "control", "STATE": "state", "EVT": "event", "ADR": "decision",
				"CAP": "contract capability", "AOU": "assumption of use", "LNK": "requirement link",
				"SM": "state machine", "TR": "machine transition", "INV": "machine invariant",
			},
		},
		Dependencies: DependencyGuide{
			PreferredSourceField: "dependencies[]",
			SourceKinds:          []string{"cue-module"},
			WorkspaceFile:        WorkspaceFileName,
			WorkspaceSchemaPath:  "internal/model/schema/workspace.cue",
			RegistryEnvironment:  "CUE_REGISTRY",
			CacheDirectory:       ".engmod/modules",
			Rules: []string{
				"Declare dependencies only in engmod.yml with a unique alias.",
				"Use a major-qualified CUE module path, exact semantic version, and at least one selected publication.",
				"Use engmod.work.yml only for local module-to-repository replacements.",
				"Reference imported stable identifiers as alias::ID.",
				"Only identifiers selected through dependency publications are visible.",
				"A dependency version is a release tag (v1.2.3) or a Go-style pseudo-version carrying the commit (v1.2.4-0.20261102103400-abcdef123456); a reqLinks pinned.version must equal it.",
			},
		},
		Machines:       machineGuide(),
		Health:         healthGuide(),
		AuthoringOrder: []string{"manifest", "catalog", "requirements", "architecture", "behavior", "assurance", "compliance", "aviation", "views", "decisions", "reqLinks"},
		Invariants: []string{
			"Define controlled vocabulary and stable IDs before referencing them.",
			"Represent each semantic concept once; outputs are projections of the canonical graph.",
			"Use requirements deltas through engflow instead of rewriting requirements.yml directly.",
			"Run CUE validation before strict Go decoding and cross-document validation.",
		},
		Compatibility: []string{},
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-045
func contractSchemaVersion(version int) int {
	return version
}

// TRLC-LINKS: REQ-EMG-045
func yamlTopLevelFields(value any) []string {
	t := reflect.TypeOf(value)
	fields := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		name := strings.Split(t.Field(i).Tag.Get("yaml"), ",")[0]
		if name != "" && name != "-" {
			fields = append(fields, name)
		}
	}
	return fields
}

// machineGuide builds the machines section of the authoring contract. The
// operator rows come from MachineOperators, so they cannot drift from the
// parser, the validator and the Quint generator.
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func machineGuide() MachineGuide {
	operators := []MachineOperatorDoc{}
	for _, op := range MachineOperators() {
		operands := make([]string, len(op.Operands))
		for i, t := range op.Operands {
			operands[i] = string(t)
		}
		operators = append(operators, MachineOperatorDoc{Operator: op.Name, Syntax: op.Syntax, Form: string(op.Form), Operands: operands,
			Variadic: op.Variadic, SameType: op.SameType, Result: string(op.Result), Precedence: op.Precedence, RightAssociative: op.RightAssociative})
	}
	return MachineGuide{
		Grammar: []string{
			"A guard and an invariant (expr) are Boolean expressions; an effect is a list of assignments 'variable := expression' separated by semicolons (an optional trailing semicolon; each variable at most once per effect).",
			"Precedence from loosest to tightest: implies (right associative), or, and, not (prefix), comparison (== != < <= > >=, non-associative), + and - (left), * (left), unary minus, primary.",
			"A primary is an integer literal, true, false, a double-quoted string, a variable name, in_state(STATE-ID) or a parenthesized expression. Whitespace is insignificant.",
			"The words and, or, not, implies, true, false and in_state are reserved: a variable cannot use them. An empty string literal is rejected; a string literal has no escapes in the generated Quint, so it must not contain a double quote or a line break.",
			"Types are Integer, Boolean and String. Arithmetic and ordering need Integer operands; == and != need two operands of one type; := needs the variable's type; in_state names a state of the same machine.",
		},
		Types:     []string{string(MachineTypeInteger), string(MachineTypeBoolean), string(MachineTypeString)},
		Operators: operators,
		Workflow: []string{
			"Author the machine in behavior.machines (see rules.fieldSets: machine, machineVariable, machineState, machineTransition, machineInvariant) and link each invariant to the requirements it protects.",
			"engquint generate --model engmod.yml [--out dir] writes one deterministic Quint module per machine to generated/quint/ (sm_<id>.qnt); regenerate after every machine change, the artifact freshness gate fails on drift.",
			"engquint check --model engmod.yml [--mode run|verify|both] regenerates and compares the committed modules (drift is reported as stale and runs nothing), runs quint run on each invariant (mode run; verify or both adds quint verify with Apalache) and writes generated/quint/results.json and test-results/quint-formal.json.",
			"Quint is called with --backend=typescript; engquint never downloads a tool. Set ENGMOD_QUINT (and QUINT_HOME for verify) or put quint on PATH.",
			"A violated invariant fails the check and prints a counterexample as the ordered transition IDs; fix the model or the invariant, regenerate and check again.",
		},
		Evidence: []string{
			"test-results/quint-formal.json holds one entry per requirement an invariant is linked to: requirement, status pass or fail, machine, invariant, mode (run, verify or run+verify), qntFile and qntSha256 of the module the result was produced from.",
			"A passing result whose qntSha256 still matches the committed module is analysis evidence for the linked requirement: the trace matrix row records the method analysis and the result passed.",
			"A failing result makes the requirement failed (verification.failed with a lifecycle tier), never unverified.",
			"A result whose qntSha256 differs from the module (or whose module is missing) is stale: it is no evidence, reported as verification.formal_stale (warning), and it does not downgrade a requirement that other passing evidence verifies. Rerun engquint check.",
			"An invariant linked to no requirement produces no evidence. The Gemara evaluation log holds one requirement-keyed entry per formal result, and verification.status and verification.gaps report pass, failed or stale per requirement.",
		},
	}
}

// healthGuide builds the health section of the authoring contract. The code
// rows come from the code vocabulary of this package, so a smell code cannot
// be documented without being valid in the block.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-155, REQ-EMG-160
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func healthGuide() HealthGuide {
	meaning := map[string]string{
		"health.complex_method":             "a function whose cyclomatic complexity reaches the limit",
		"health.large_method":               "a function with too many lines",
		"health.deep_nesting":               "a function nested too deeply",
		"health.bumpy_road":                 "a function with several separate nested-conditional bumps",
		"health.complex_conditional":        "a condition combining too many logical operators",
		"health.excess_arguments":           "a function with too many parameters",
		"health.primitive_obsession":        "a function whose parameters are mostly primitive types",
		"health.brain_method":               "a function that is large, complex, deeply nested and takes many arguments at once",
		"health.large_assertion_block":      "a test with too many consecutive assertions",
		"health.duplicated_assertion_block": "a test whose assertion block repeats in another test",
	}
	codes := []HealthCodeDoc{}
	for _, code := range HealthSmellCodes() {
		codes = append(codes, HealthCodeDoc{Code: code, Meaning: meaning[code], Threshold: IsHealthThresholdCode(code)})
	}
	codes = append(codes,
		HealthCodeDoc{Code: HealthRegressionCode, Meaning: "a new or worsened high finding against the compared baseline"},
		HealthCodeDoc{Code: HealthWaiverUnusedCode, Meaning: "a waiver that matches no finding (emitted by the health report)"},
		HealthCodeDoc{Code: HealthWaiverUnjustifiedCode, Meaning: "a waiver or a baseline whose decision does not exist or is not accepted (error, untiered)"},
		HealthCodeDoc{Code: HealthBaselineStaleCode, Meaning: "a baseline entry that matches no finding, so it can be removed (emitted by the health report)"},
		HealthCodeDoc{Code: HealthBaselineUnreadableCode, Meaning: "a baseline file that is missing, not valid JSON or not engmod-health-baseline/1 (error, untiered)"},
		HealthCodeDoc{Code: HealthPolicyWeakenedCode, Meaning: "a change that raises a health limit, widens health.exclude or an override or lowers the tier against the base revision, without an accepted policyDecision (reported by enghealth check)"},
	)
	return HealthGuide{
		Config: []string{
			"health is an optional block of engmod.yml: health: {exclude: [pattern], thresholds: {<code>: {warn: int, high: int}}, overrides: [{path: pattern, thresholds: {<code>: {warn, high}}}], waivers: [{code, path, function, decision}], baseline: {file, decision}, policyDecision: <decision id>}. Every key is optional; without the block nothing about code health is validated, computed or reported.",
			"exclude entries and overrides[].path use the pattern dialect of inferenceHints.exclude (relative to the module root; * ? [...] within one path segment; ** only as a trailing /**; no negation).",
			"thresholds and overrides[].thresholds are keyed by a smell code that takes numeric limits (the codes with threshold: true); warn is at least 1 and high is at least warn. A threshold replaces the built-in default of the code for the whole module, an override for the paths it matches.",
			"A waiver waives the finding of code in function of the file path (a file path or a pattern of the exclude dialect, relative to the module root) and names the decision that justifies it. There is no history key.",
			"baseline {file, decision} accepts existing debt: file is a path relative to the manifest directory (no absolute path, no ..) of a JSON file {schema: engmod-health-baseline/1, entries: [{path, function, code, value}]} sorted by path, function and code, written by enghealth baseline write and never edited by hand (value is the metric value of the finding when it was written, at least 1); decision is an accepted decision. A finding listed in the baseline is reported as baselined while its current value is at most the recorded value: it still counts toward the score but is neither an error nor counted by the gate. A finding whose value grew past the recorded one is no longer baselined and counts at the level of its code. A regression (a new or worsened high finding) is never baselined. A baseline whose decision is missing or not accepted suppresses nothing.",
			"policyDecision is an accepted decision that records a weakening of this block against the base revision (a higher limit, a new exclude, an override that raises a limit, a lower tier). enghealth check reads the manifest of the base revision and reports such a change as health.policy_weakened unless the head names a policyDecision that the base did not already name; the decision is the reviewable record (decisions.yml is code-owned).",
		},
		Codes: codes,
		Rules: []string{
			"A threshold key that is no smell code with numeric limits, a threshold whose warn is below 1 or whose high is below warn, an exclude or override path the pattern dialect rejects, and a waiver without code, path, function or decision, with a code that is no smell code or with a path that is no module-relative path are model.invalid_health_threshold, model.invalid_health_exclude, model.invalid_health_override and model.invalid_health_waiver (errors at every tier, never deferred).",
			"health.waiver_unjustified (error, untiered: never changed or deferred by the tier policy) is reported at a waiver whose decision does not exist or whose status is not accepted; the message names the waiver and the decision. A waiver without an accepted decision waives nothing. The same code is reported for health.baseline.decision (the message names the baseline) and for health.policyDecision (the message names the policy decision).",
			"health.baseline_unreadable (error, untiered) is reported when the baseline file is missing, is not valid JSON, is not engmod-health-baseline/1 or holds an invalid entry. An empty or unclean baseline file path, or an empty decision, is model.invalid_health_baseline (error at every tier, never deferred).",
			"health.baseline_stale (a baseline entry that matches no finding) is a warning at poc to product and an error at operational, so the baseline can only shrink; it is emitted by the health report and not reported for a file outside a path-restricted scan.",
			"Tier policy: every smell code and health.waiver_unused is a warning up to mvp and an error at product and operational (from mvp with the safety bundle); health.regression and health.policy_weakened are warnings at poc, prototype and mvp and errors from product, and with the safety bundle errors from mvp (a source file that enghealth skipped or failed to analyse has the level of health.regression). lifecycle.raise: [health.*] makes the debt codes errors.",
			"Scores are this project's own heuristics over the scanned source; they are not CodeScene's Code Health and do not claim to match it.",
		},
	}
}
