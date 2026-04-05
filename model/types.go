package model

type CatalogEntry struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Definition string   `yaml:"definition"`
	Aliases    []string `yaml:"aliases"`
}

type CatalogGroups struct {
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

type AuthoredArchitecture struct {
	FunctionalGroups   []FunctionalGroup   `yaml:"functionalGroups"`
	FunctionalUnits    []FunctionalUnit    `yaml:"functionalUnits"`
	Actors             []Actor             `yaml:"actors"`
	AttackVectors      []AttackVector      `yaml:"attackVectors"`
	ReferencedElements []ReferencedElement `yaml:"referencedElements"`
	Mappings           []Mapping           `yaml:"mappings"`
}

type InferenceHints struct {
	RuntimeSources           []string `yaml:"runtimeSources"`
	CodeSources              []string `yaml:"codeSources"`
	ExpectedRuntimeKinds     []string `yaml:"expectedRuntimeKinds"`
	OwnershipResolutionOrder []string `yaml:"ownershipResolutionOrder"`
}

type View struct {
	ID    string   `yaml:"id"`
	Kind  string   `yaml:"kind"`
	Roots []string `yaml:"roots"`
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
