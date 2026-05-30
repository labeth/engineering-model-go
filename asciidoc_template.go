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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocTerm struct {
	Anchor      string
	IndexAnchor string
	ID          string
	Name        string
	Definition  string
	Aliases     []string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocReferenceIndex struct {
	Authored     []asciidocReferenceEntry
	Catalog      []asciidocReferenceEntry
	Runtime      []asciidocReferenceEntry
	Code         []asciidocReferenceEntry
	Verification []asciidocReferenceEntry
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocSummary struct {
	FunctionalGroups   string
	FunctionalUnits    string
	Actors             string
	AttackVectors      string
	ReferencedElements string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocModelMeta struct {
	ID             string
	Title          string
	BaseCatalogRef string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type asciidocLintRun struct {
	ID         string
	Mode       string
	CommaAsAnd bool
	CatalogRef string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocViewConfig struct {
	ID    string
	Kind  string
	Roots string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type asciidocInferenceHints struct {
	RuntimeSources           string
	CodeSources              string
	ExpectedRuntimeKinds     string
	OwnershipResolutionOrder string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocActorSection struct {
	ID          string
	Name        string
	Description string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
type asciidocAttackVectorSection struct {
	ID          string
	Name        string
	Description string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocReferencedSection struct {
	ID    string
	Name  string
	Kind  string
	Layer string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocMappingSection struct {
	Type        string
	From        string
	To          string
	Description string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocProjectedNode struct {
	ID    string
	Kind  string
	Label string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type asciidocInferredRow struct {
	Name   string
	Kind   string
	Owner  string
	Source string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type asciidocRuntimeAPIRow struct {
	Consumer string
	Provider string
	Ports    string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
type asciidocDeploymentRow struct {
	From string
	To   string
	How  string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
type asciidocPlatformOpRow struct {
	Actor  string
	Unit   string
	Target string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
type asciidocSecurityPathRow struct {
	AttackVectorID string
	AttackVector   string
	TargetID       string
	Target         string
	DependsOn      string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type asciidocSecurityObsRow struct {
	Signal   string
	Layer    string
	Owner    string
	Evidence string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocSecurityContextDiagram struct {
	GroupID   string
	GroupName string
	Mermaid   string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
type asciidocThreatAssumptionRow struct {
	ID        string
	Title     string
	Status    string
	Owner     string
	AppliesTo string
	Rationale string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
type asciidocThreatOutOfScopeRow struct {
	ID        string
	Title     string
	Status    string
	Owner     string
	AppliesTo string
	ExpiresOn string
	Reason    string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, CTRL-TRACEABILITY-COVERAGE
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
type asciidocSecurityAttackChapter struct {
	ID              string
	Name            string
	Description     string
	MitigatedBy     string
	TrustBoundaries string
	Diagram         string
	Units           []asciidocUnitSection
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocDesignDetail struct {
	ViewID    string
	Title     string
	Narrative string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, CTRL-TRACEABILITY-COVERAGE
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, CTRL-TRACEABILITY-COVERAGE
type asciidocVerificationResultRow struct {
	CheckID     string
	CheckName   string
	Requirement string
	Status      string
	Evidence    string
	Notes       string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type asciidocDecisionSection struct {
	ID           string
	Title        string
	Status       string
	Date         string
	Context      string
	Decision     string
	Consequences []string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func renderAsciiDocTemplate(data asciidocTemplateData) (string, error) {
	var b bytes.Buffer
	if err := asciidocTemplate.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute asciidoc template: %w", err)
	}
	return b.String(), nil
}
