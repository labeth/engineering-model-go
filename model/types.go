package model

type CatalogEntry struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Definition string   `yaml:"definition"`
	Aliases    []string `yaml:"aliases"`
}

type CatalogGroups struct {
	Systems            []CatalogEntry `yaml:"systems"`
	FunctionalGroups   []CatalogEntry `yaml:"functionalGroups"`
	FunctionalUnits    []CatalogEntry `yaml:"functionalUnits"`
	ReferencedElements []CatalogEntry `yaml:"referencedElements"`
	Actors             []CatalogEntry `yaml:"actors"`
	AttackVectors      []CatalogEntry `yaml:"attackVectors"`
	Events             []CatalogEntry `yaml:"events"`
	States             []CatalogEntry `yaml:"states"`
	Features           []CatalogEntry `yaml:"features"`
	Modes              []CatalogEntry `yaml:"modes"`
	Conditions         []CatalogEntry `yaml:"conditions"`
	DataTerms          []CatalogEntry `yaml:"dataTerms"`
}

type CatalogDocument struct {
	Catalog CatalogGroups `yaml:"catalog"`
}

type LintRun struct {
	ID         string `yaml:"id"`
	Mode       string `yaml:"mode"`
	CommaAsAnd bool   `yaml:"commaAsAnd"`
	CatalogRef string `yaml:"catalogRef"`
}

type Requirement struct {
	ID        string   `yaml:"id"`
	Text      string   `yaml:"text"`
	Notes     string   `yaml:"notes"`
	AppliesTo []string `yaml:"appliesTo"`
}

type RequirementsDocument struct {
	LintRun      LintRun       `yaml:"lintRun"`
	Requirements []Requirement `yaml:"requirements"`
	Expected     []Expected    `yaml:"expected"`
}

type Expected struct {
	ID      string `yaml:"id"`
	Pattern string `yaml:"pattern"`
}

type DesignView struct {
	Title     string `yaml:"title"`
	Narrative string `yaml:"narrative"`
}

type DesignFunctionalGroup struct {
	ID    string                `yaml:"id"`
	Views map[string]DesignView `yaml:"views"`
}

type DesignFunctionalUnit struct {
	ID    string                `yaml:"id"`
	Group string                `yaml:"group"`
	Views map[string]DesignView `yaml:"views"`
}

type DesignModel struct {
	ID               string                  `yaml:"id"`
	Title            string                  `yaml:"title"`
	FunctionalGroups []DesignFunctionalGroup `yaml:"functionalGroups"`
	FunctionalUnits  []DesignFunctionalUnit  `yaml:"functionalUnits"`
}

type DesignDocument struct {
	Design DesignModel `yaml:"design"`
}

type ModelMeta struct {
	ID             string `yaml:"id"`
	Title          string `yaml:"title"`
	Introduction   string `yaml:"introduction"`
	BaseCatalogRef string `yaml:"baseCatalogRef"`
}

type FunctionalGroup struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Prose       string   `yaml:"prose"`
	Tags        []string `yaml:"tags"`
}

type FunctionalUnit struct {
	ID    string   `yaml:"id"`
	Name  string   `yaml:"name"`
	Group string   `yaml:"group"`
	Tags  []string `yaml:"tags"`
	Prose string   `yaml:"prose"`
}

type Actor struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type AttackVector struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type ReferencedElement struct {
	ID    string `yaml:"id"`
	Kind  string `yaml:"kind"`
	Layer string `yaml:"layer"`
	Name  string `yaml:"name"`
}

type Mapping struct {
	Type        string `yaml:"type"`
	From        string `yaml:"from"`
	To          string `yaml:"to"`
	Description string `yaml:"description"`
}

type Interface struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	Protocol  string `yaml:"protocol"`
	Endpoint  string `yaml:"endpoint"`
	SchemaRef string `yaml:"schemaRef"`
	Owner     string `yaml:"owner"`
}

type DataObject struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	TermRef     string `yaml:"termRef"`
	SchemaRef   string `yaml:"schemaRef"`
	Sensitivity string `yaml:"sensitivity"`
}

type DeploymentTarget struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Region      string `yaml:"region"`
	Account     string `yaml:"account"`
	Cluster     string `yaml:"cluster"`
	Namespace   string `yaml:"namespace"`
	TrustZone   string `yaml:"trustZone"`
}

type Control struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Category    string `yaml:"category"`
	Description string `yaml:"description"`
}

type ControlEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

type ControlAllocation struct {
	ID                 string            `yaml:"id"`
	ControlRef         string            `yaml:"controlRef"`
	OSCALControlIDs    []string          `yaml:"oscalControlIds"`
	AppliesTo          []string          `yaml:"appliesTo"`
	ImplementationType string            `yaml:"implementationType"`
	Status             string            `yaml:"status"`
	Narrative          string            `yaml:"narrative"`
	Evidence           []ControlEvidence `yaml:"evidence"`
	InheritedFrom      []string          `yaml:"inheritedFrom"`
	ResponsibleRoles   []string          `yaml:"responsibleRoles"`
}

type RiskEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

type Risk struct {
	ID              string         `yaml:"id"`
	Title           string         `yaml:"title"`
	Statement       string         `yaml:"statement"`
	Status          string         `yaml:"status"`
	Likelihood      string         `yaml:"likelihood"`
	Impact          string         `yaml:"impact"`
	Response        string         `yaml:"response"`
	Owner           string         `yaml:"owner"`
	AppliesTo       []string       `yaml:"appliesTo"`
	RelatedControls []string       `yaml:"relatedControls"`
	AttackVectors   []string       `yaml:"attackVectors"`
	Evidence        []RiskEvidence `yaml:"evidence"`
	ResidualRisk    string         `yaml:"residualRisk"`
	Rationale       string         `yaml:"rationale"`
}

type POAMArtifact struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

type POAMItem struct {
	ID              string         `yaml:"id"`
	RiskRef         string         `yaml:"riskRef"`
	Milestone       string         `yaml:"milestone"`
	DueDate         string         `yaml:"dueDate"`
	Status          string         `yaml:"status"`
	ResponsibleRole string         `yaml:"responsibleRole"`
	Artifacts       []POAMArtifact `yaml:"artifacts"`
}

type TrustBoundary struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type State struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Event struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type FlowStep struct {
	ID       string   `yaml:"id"`
	Ref      string   `yaml:"ref"`
	Kind     string   `yaml:"kind"`
	Action   string   `yaml:"action"`
	DataIn   []string `yaml:"dataIn"`
	DataOut  []string `yaml:"dataOut"`
	Next     []string `yaml:"next"`
	OnError  []string `yaml:"onError"`
	Async    bool     `yaml:"async"`
	Optional bool     `yaml:"optional"`
}

type Flow struct {
	ID    string     `yaml:"id"`
	Title string     `yaml:"title"`
	Entry []string   `yaml:"entry"`
	Exits []string   `yaml:"exits"`
	Steps []FlowStep `yaml:"steps"`
}

type AuthoredArchitecture struct {
	FunctionalGroups   []FunctionalGroup   `yaml:"functionalGroups"`
	FunctionalUnits    []FunctionalUnit    `yaml:"functionalUnits"`
	Actors             []Actor             `yaml:"actors"`
	AttackVectors      []AttackVector      `yaml:"attackVectors"`
	ReferencedElements []ReferencedElement `yaml:"referencedElements"`
	Interfaces         []Interface         `yaml:"interfaces"`
	DataObjects        []DataObject        `yaml:"dataObjects"`
	DeploymentTargets  []DeploymentTarget  `yaml:"deploymentTargets"`
	Controls           []Control           `yaml:"controls"`
	ControlAllocations []ControlAllocation `yaml:"controlAllocations"`
	Risks              []Risk              `yaml:"risks"`
	POAMItems          []POAMItem          `yaml:"poamItems"`
	TrustBoundaries    []TrustBoundary     `yaml:"trustBoundaries"`
	States             []State             `yaml:"states"`
	Events             []Event             `yaml:"events"`
	Flows              []Flow              `yaml:"flows"`
	Mappings           []Mapping           `yaml:"mappings"`
}

type InferenceHints struct {
	RuntimeSources           []string `yaml:"runtimeSources"`
	CodeSources              []string `yaml:"codeSources"`
	ExpectedRuntimeKinds     []string `yaml:"expectedRuntimeKinds"`
	OwnershipResolutionOrder []string `yaml:"ownershipResolutionOrder"`
}

type View struct {
	ID                        string   `yaml:"id"`
	Kind                      string   `yaml:"kind"`
	Roots                     []string `yaml:"roots"`
	AuthoredStatus            string   `yaml:"authoredStatus"`
	AuthoredStatusExplanation string   `yaml:"authoredStatusExplanation"`
	IncludeKinds              []string `yaml:"includeKinds"`
	ExcludeKinds              []string `yaml:"excludeKinds"`
	IncludeMappings           []string `yaml:"includeMappings"`
	ExcludeMappings           []string `yaml:"excludeMappings"`
	MaxDepth                  int      `yaml:"maxDepth"`
	Audience                  string   `yaml:"audience"`
	Abstraction               string   `yaml:"abstraction"`
}

type ArchitectureDocument struct {
	Model                ModelMeta            `yaml:"model"`
	AuthoredArchitecture AuthoredArchitecture `yaml:"authoredArchitecture"`
	InferenceHints       InferenceHints       `yaml:"inferenceHints"`
	Views                []View               `yaml:"views"`
}

type Bundle struct {
	ArchitecturePath string
	CatalogPath      string

	Architecture ArchitectureDocument
	Catalog      CatalogDocument
}
