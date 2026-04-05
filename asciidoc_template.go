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
	Views                      []asciidocViewSection
	RequirementMermaid         string
	RequirementCoverageMermaid string
	RequirementInf             string
	Requirements               []asciidocRequirementSection
	ReferenceIndex             asciidocReferenceIndex
}

type asciidocHealthRow struct {
	ViewID       string
	ViewHeading  string
	UnitsInScope int
	WithEvidence int
	GapCount     int
}

type asciidocTerm struct {
	Anchor     string
	ID         string
	Name       string
	Definition string
	Aliases    []string
}

type asciidocReferenceIndex struct {
	Authored []asciidocReferenceEntry
	Catalog  []asciidocReferenceEntry
	Runtime  []asciidocReferenceEntry
	Code     []asciidocReferenceEntry
}

type asciidocReferenceEntry struct {
	Anchor       string
	TargetAnchor string
	ID           string
	Name         string
	Kind         string
	Description  string
	Source       string
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
	AttackVector string
	Target       string
	DependsOn    string
}

type asciidocSecurityObsRow struct {
	Signal   string
	Layer    string
	Owner    string
	Evidence string
}

type asciidocViewSection struct {
	ID               string
	Kind             string
	Heading          string
	Inf              string
	ViewQuestions    []string
	CoverageSummary  string
	CoverageGaps     []string
	NextActions      []string
	Mermaid          string
	FuncContextGraph string
	FuncDecompGraph  string
	FuncCollabGraph  string
	Groups           []asciidocEntitySection
	Units            []asciidocUnitSection
	InferredGraph    string
	InferredRows     []asciidocInferredRow
	RuntimeAPIGraph  string
	RuntimeAPIRows   []asciidocRuntimeAPIRow
	DeploymentGraph  string
	DeploymentRows   []asciidocDeploymentRow
	PlatformOpsGraph string
	PlatformOpsRows  []asciidocPlatformOpRow
	SecurityGraph    string
	SecurityRows     []asciidocSecurityPathRow
	SecurityObsRows  []asciidocSecurityObsRow
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
	Anchor string
	ID     string
	Text   string
	Notes  string
}

func renderAsciiDocTemplate(data asciidocTemplateData) (string, error) {
	var b bytes.Buffer
	if err := asciidocTemplate.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute asciidoc template: %w", err)
	}
	return b.String(), nil
}
