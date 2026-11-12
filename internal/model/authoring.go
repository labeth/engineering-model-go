// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"reflect"
	"strings"
)

// AuthoringContractVersion identifies the contract shape. v3 added the
// machines guide (expression grammar, operator table, Quint workflow and
// formal evidence semantics) and the SM, TR and INV identifier prefixes.
const AuthoringContractVersion = "engineering-model.authoring.v3"

// AuthoringContract is compact machine context for editing the canonical YAML set.
type AuthoringContract struct {
	ContractVersion   string                `json:"contractVersion"`
	CanonicalFormat   string                `json:"canonicalFormat"`
	SchemaAuthority   string                `json:"schemaAuthority"`
	Documents         []AuthoringDocument   `json:"documents"`
	StableIdentifiers StableIdentifierGuide `json:"stableIdentifiers"`
	Dependencies      DependencyGuide       `json:"dependencies"`
	Machines          MachineGuide          `json:"machines"`
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
