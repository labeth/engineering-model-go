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

type asciidocTerm struct {
	Anchor      string
	IndexAnchor string
	ID          string
	Name        string
	Definition  string
	Aliases     []string
}

type asciidocReferenceIndex struct {
	Authored     []asciidocReferenceEntry
	Catalog      []asciidocReferenceEntry
	Runtime      []asciidocReferenceEntry
	Code         []asciidocReferenceEntry
	Verification []asciidocReferenceEntry
}

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

type asciidocSummary struct {
	FunctionalGroups   string
	FunctionalUnits    string
	Actors             string
	AttackVectors      string
	ReferencedElements string
}

type asciidocModelMeta struct {
	ID             string
	Title          string
	BaseCatalogRef string
}

type asciidocLintRun struct {
	ID         string
	Mode       string
	CommaAsAnd bool
	CatalogRef string
}

type asciidocViewConfig struct {
	ID    string
	Kind  string
	Roots string
}

type asciidocInferenceHints struct {
	RuntimeSources           string
	CodeSources              string
	ExpectedRuntimeKinds     string
	OwnershipResolutionOrder string
}

type asciidocActorSection struct {
	ID          string
	Name        string
	Description string
}

type asciidocAttackVectorSection struct {
	ID          string
	Name        string
	Description string
}

type asciidocReferencedSection struct {
	ID    string
	Name  string
	Kind  string
	Layer string
}

type asciidocMappingSection struct {
	Type        string
	From        string
	To          string
	Description string
}

type asciidocProjectedNode struct {
	ID    string
	Kind  string
	Label string
}

type asciidocInferredRow struct {
	Name   string
	Kind   string
	Owner  string
	Source string
}

type asciidocRuntimeAPIRow struct {
	Consumer string
	Provider string
	Ports    string
}

type asciidocDeploymentRow struct {
	From string
	To   string
	How  string
}

type asciidocPlatformOpRow struct {
	Actor  string
	Unit   string
	Target string
}

type asciidocSecurityPathRow struct {
	AttackVectorID string
	AttackVector   string
	TargetID       string
	Target         string
	DependsOn      string
}

type asciidocSecurityObsRow struct {
	Signal   string
	Layer    string
	Owner    string
	Evidence string
}

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
	FuncCollabGraph           string
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
	SecurityDataFlowDFD       string
	SecurityThreatOverlayDFD  string
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

type asciidocThreatScenarioRow struct {
	ID            string
	Title         string
	AttackVector  string
	Scope         string
	Flows         string
	Likelihood    string
	Impact        string
	Severity      string
	Status        string
	Owner         string
	Risk          string
	Controls      string
	Mitigations   string
	Verifications string
}

type asciidocThreatAssumptionRow struct {
	ID        string
	Title     string
	Status    string
	Owner     string
	AppliesTo string
	Rationale string
}

type asciidocThreatOutOfScopeRow struct {
	ID        string
	Title     string
	Status    string
	Owner     string
	AppliesTo string
	ExpiresOn string
	Reason    string
}

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

type asciidocSecurityAttackChapter struct {
	ID          string
	Name        string
	Description string
	Diagram     string
	Units       []asciidocUnitSection
}

type asciidocDesignDetail struct {
	ViewID    string
	Title     string
	Narrative string
}

type asciidocEntitySection struct {
	Anchor      string
	ID          string
	Name        string
	Description string
	Tags        string
	Intro       string
	Details     []asciidocDesignDetail
}

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

type asciidocVerificationResultRow struct {
	CheckID     string
	CheckName   string
	Requirement string
	Status      string
	Evidence    string
	Notes       string
}

type asciidocDecisionSection struct {
	ID           string
	Title        string
	Status       string
	Date         string
	Context      string
	Decision     string
	Consequences []string
}

func renderAsciiDocTemplate(data asciidocTemplateData) (string, error) {
	var b bytes.Buffer
	if err := asciidocTemplate.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute asciidoc template: %w", err)
	}
	return b.String(), nil
}
