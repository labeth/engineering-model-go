// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed templates/asciidoc.tmpl
var asciidocTemplateText string

var asciidocTemplate = template.Must(template.New("asciidoc").Parse(asciidocTemplateText))

// ENGMODEL-LINKS: EM-ASCIIDOC-DOCUMENT, EM-ASCIIDOC-SECTION, EM-REFERENCE-INDEX, EM-ASCIIDOC-DIAGRAM
type asciidocTemplateData struct {
	Title                      string
	Introduction               string
	HealthRows                 []asciidocHealthRow
	Terms                      []asciidocTerm
	Purpose                    string
	ReaderTracks               []string
	Legend                     []string
	ModelMeta                  asciidocModelMeta
	LintRun                    asciidocLintRun
	ViewConfig                 []asciidocViewConfig
	InferenceHints             asciidocInferenceHints
	Actors                     []asciidocActorSection
	AttackVectors              []asciidocAttackVectorSection
	ReferencedElements         []asciidocReferencedSection
	Mappings                   []asciidocMappingSection
	InferredRuntime            []asciidocInferredRow
	InferredCode               []asciidocInferredRow
	Summary                    asciidocSummary
	MermaidClassDefs           string
	Views                      []asciidocViewSection
	RequirementMermaid         string
	RequirementCoverageMermaid string
	RequirementInf             string
	Requirements               []asciidocRequirementSection
	Verifications              []asciidocVerificationSection
	VerificationResults        []asciidocVerificationResultRow
	ReferenceIndex             asciidocReferenceIndex
	Decisions                  []asciidocDecisionSection
	DecisionsDocPath           string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-VIEW
type asciidocHealthRow struct {
	ViewID                    string
	ViewHeading               string
	AuthoredStatus            string
	AuthoredStatusExplanation string
	UnitsInScope              int
	WithEvidence              int
	GapCount                  int
	Confidence                string
	HeuristicBasisExplanation string
}

// ENGMODEL-LINKS: EM-CATALOG-ENTRY, EM-ASCIIDOC-SECTION
type asciidocTerm struct {
	Anchor      string
	IndexAnchor string
	ID          string
	Name        string
	Definition  string
	Aliases     []string
}

// ENGMODEL-LINKS: EM-REFERENCE-INDEX, EM-ASCIIDOC-SECTION
type asciidocReferenceIndex struct {
	Authored     []asciidocReferenceEntry
	Catalog      []asciidocReferenceEntry
	Runtime      []asciidocReferenceEntry
	Code         []asciidocReferenceEntry
	Verification []asciidocReferenceEntry
}

// ENGMODEL-LINKS: EM-REFERENCE-INDEX
type asciidocReferenceEntry struct {
	Anchor       string
	TargetAnchor string
	ID           string
	Name         string
	Kind         string
	Status       string
	Owner        string
	Aliases      []string
	Description  string
	Source       string
	Backlinks    []string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-MODEL
type asciidocSummary struct {
	FunctionalGroups   string
	FunctionalUnits    string
	Actors             string
	AttackVectors      string
	ReferencedElements string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-MODEL
type asciidocModelMeta struct {
	ID             string
	Title          string
	BaseCatalogRef string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-LINT-RUN
type asciidocLintRun struct {
	ID         string
	Mode       string
	CommaAsAnd bool
	CatalogRef string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-VIEW
type asciidocViewConfig struct {
	ID    string
	Kind  string
	Roots string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-INFERENCE-HINT
type asciidocInferenceHints struct {
	RuntimeSources           string
	CodeSources              string
	ExpectedRuntimeKinds     string
	OwnershipResolutionOrder string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-ACTOR
type asciidocActorSection struct {
	ID          string
	Name        string
	Description string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-ATTACK-VECTOR
type asciidocAttackVectorSection struct {
	ID          string
	Name        string
	Description string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-REFERENCED-ELEMENT
type asciidocReferencedSection struct {
	ID    string
	Name  string
	Kind  string
	Layer string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-AUTHORED-MAPPING
type asciidocMappingSection struct {
	Type        string
	From        string
	To          string
	Description string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-VIEW
type asciidocProjectedNode struct {
	ID    string
	Kind  string
	Label string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-RUNTIME-ELEMENT, EM-CODE-ELEMENT
type asciidocInferredRow struct {
	Name   string
	Kind   string
	Owner  string
	Source string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-RUNTIME-ELEMENT, EM-INTERFACE
type asciidocRuntimeAPIRow struct {
	Consumer string
	Provider string
	Ports    string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-RUNTIME-ELEMENT, EM-DEPLOYMENT-TARGET
type asciidocDeploymentRow struct {
	From string
	To   string
	How  string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-ACTOR, EM-DEPLOYMENT-TARGET
type asciidocPlatformOpRow struct {
	Actor  string
	Unit   string
	Target string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-ATTACK-VECTOR, EM-CONTROL, EM-TRUST-BOUNDARY
type asciidocSecurityPathRow struct {
	AttackVectorID string
	AttackVector   string
	TargetID       string
	Target         string
	DependsOn      string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-RUNTIME-ELEMENT, EM-CODE-ELEMENT
type asciidocSecurityObsRow struct {
	Signal   string
	Layer    string
	Owner    string
	Evidence string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-VIEW, EM-ASCIIDOC-DIAGRAM
type asciidocViewSection struct {
	ID                        string
	Kind                      string
	Heading                   string
	AuthoredStatus            string
	AuthoredStatusExplanation string
	Inf                       string
	ViewQuestions             []string
	CoverageSummary           string
	CoverageGaps              []string
	NextActions               []string
	Mermaid                   string
	FuncContextGraph          string
	FuncDecompGraph           string
	FuncMatrixTable           string
	Groups                    []asciidocEntitySection
	Units                     []asciidocUnitSection
	InferredGraph             string
	InferredRows              []asciidocInferredRow
	RuntimeAPIGraph           string
	RuntimeAPIRows            []asciidocRuntimeAPIRow
	DeploymentGraph           string
	DeploymentRows            []asciidocDeploymentRow
	PlatformOpsGraph          string
	PlatformOpsRows           []asciidocPlatformOpRow
	SecurityGraph             string
	SecurityContextDFD        string
	SecurityContextDiagrams   []asciidocSecurityContextDiagram
	SecurityRows              []asciidocSecurityPathRow
	SecurityThreatScenarios   []asciidocThreatScenarioRow
	SecurityThreatAssumptions []asciidocThreatAssumptionRow
	SecurityThreatOutOfScope  []asciidocThreatOutOfScopeRow
	SecurityThreatMitigations []asciidocThreatMitigationRow
	SecurityControlChecks     []asciidocControlVerificationRow
	SecurityFlowRows          []asciidocSecurityFlowRow
	SecurityObsRows           []asciidocSecurityObsRow
	SecurityAttackChapters    []asciidocSecurityAttackChapter
	ProjectedNodes            []asciidocProjectedNode
	ProjectedMappings         []asciidocMappingSection
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-SECURITY-CONTEXT
type asciidocSecurityContextDiagram struct {
	GroupID   string
	GroupName string
	Mermaid   string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-THREAT-SCENARIO
type asciidocThreatScenarioRow struct {
	ID                   string
	Title                string
	Summary              string
	AttackVector         string
	Scope                string
	Flows                string
	Likelihood           string
	Impact               string
	Severity             string
	Status               string
	Owner                string
	Risk                 string
	Controls             string
	Mitigations          string
	Verifications        string
	ControlVerifications []asciidocControlVerificationRow
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-THREAT-ASSUMPTION
type asciidocThreatAssumptionRow struct {
	ID        string
	Title     string
	Status    string
	Owner     string
	AppliesTo string
	Rationale string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-THREAT-OUT-OF-SCOPE
type asciidocThreatOutOfScopeRow struct {
	ID        string
	Title     string
	Status    string
	Owner     string
	AppliesTo string
	ExpiresOn string
	Reason    string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-THREAT-MITIGATION
type asciidocThreatMitigationRow struct {
	ID             string
	ThreatScenario string
	Control        string
	Status         string
	Effectiveness  string
	Owner          string
	Verifications  string
	Notes          string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-CONTROL-VERIFICATION
type asciidocControlVerificationRow struct {
	ID              string
	Control         string
	ThreatScenarios string
	Risks           string
	Method          string
	Status          string
	Owner           string
	LastTested      string
	Findings        string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-FLOW
type asciidocSecurityFlowRow struct {
	ID               string
	Title            string
	Kind             string
	Direction        string
	Frequency        string
	Source           string
	Destination      string
	Protocol         string
	Authentication   string
	Encryption       string
	Integrity        string
	TrustBoundary    string
	BoundaryCrossing string
	Threats          string
	Data             string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-ASCIIDOC-DIAGRAM, EM-ATTACK-VECTOR
type asciidocSecurityAttackChapter struct {
	ID              string
	Name            string
	Description     string
	MitigatedBy     string
	TrustBoundaries string
	Diagram         string
	Units           []asciidocUnitSection
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-DESIGN, EM-VIEW
type asciidocDesignDetail struct {
	ViewID    string
	Title     string
	Narrative string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-FUNCTIONAL-GROUP
type asciidocEntitySection struct {
	Anchor          string
	ID              string
	Name            string
	Description     string
	Tags            string
	Intro           string
	Details         []asciidocDesignDetail
	DependencyGraph string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-FUNCTIONAL-UNIT
type asciidocUnitSection struct {
	Anchor      string
	GroupAnchor string
	ID          string
	Name        string
	Group       string
	Tags        string
	Intro       string
	Details     []asciidocDesignDetail
	WhatOwns    string
	Triggers    string
	Consumes    string
	Produces    string
	DependsOn   string
	Threats     string
	Evidence    string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-REQUIREMENT, EM-ASCIIDOC-DIAGRAM
type asciidocRequirementSection struct {
	Anchor               string
	ID                   string
	Text                 string
	Notes                string
	AlignmentMermaid     string
	CoverageMermaid      string
	AlignmentExplanation string
	CoverageExplanation  string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-VERIFICATION-CHECK
type asciidocVerificationSection struct {
	Anchor        string
	IndexAnchor   string
	ID            string
	Name          string
	Kind          string
	Status        string
	Verifies      string
	TestCode      string
	DerivedOwners string
	Evidence      string
	ResultSummary string
	Description   string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-VERIFICATION-CHECK
type asciidocVerificationResultRow struct {
	CheckID     string
	CheckName   string
	Requirement string
	Status      string
	Evidence    string
	Notes       string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-DECISION
type asciidocDecisionSection struct {
	ID           string
	Title        string
	Status       string
	Date         string
	Context      string
	Decision     string
	Consequences []string
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DOCUMENT, EM-ASCIIDOC-SECTION
// TRLC-LINKS: REQ-EMG-003
func renderAsciiDocTemplate(data asciidocTemplateData) (string, error) {
	var b bytes.Buffer
	if err := asciidocTemplate.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute asciidoc template: %w", err)
	}
	return b.String(), nil
}
