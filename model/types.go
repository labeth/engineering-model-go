package model

type CatalogEntry struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Definition string   `yaml:"definition"`
	Aliases    []string `yaml:"aliases"`
}

type CatalogGroups struct {
	Systems    []CatalogEntry `yaml:"systems"`
	Actors     []CatalogEntry `yaml:"actors"`
	Events     []CatalogEntry `yaml:"events"`
	States     []CatalogEntry `yaml:"states"`
	Features   []CatalogEntry `yaml:"features"`
	Modes      []CatalogEntry `yaml:"modes"`
	Conditions []CatalogEntry `yaml:"conditions"`
	DataTerms  []CatalogEntry `yaml:"dataTerms"`
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
	ID    string `yaml:"id"`
	Text  string `yaml:"text"`
	Notes string `yaml:"notes"`
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

type DesignChapter struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Narrative   string   `yaml:"narrative"`
	CatalogRefs []string `yaml:"catalogRefs"`
}

type DesignModel struct {
	ID       string          `yaml:"id"`
	Title    string          `yaml:"title"`
	Views    []string        `yaml:"views"`
	Chapters []DesignChapter `yaml:"chapters"`
}

type DesignDocument struct {
	Design DesignModel `yaml:"design"`
}

type ModelMeta struct {
	ID             string `yaml:"id"`
	Title          string `yaml:"title"`
	BaseCatalogRef string `yaml:"baseCatalogRef"`
}

type Person struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type SoftwareSystem struct {
	ID       string `yaml:"id"`
	Kind     string `yaml:"kind"`
	Name     string `yaml:"name"`
	Boundary string `yaml:"boundary"`
}

type Container struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	Technology string `yaml:"technology"`
	PartOf     string `yaml:"partOf"`
}

type Component struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	PartOf string `yaml:"partOf"`
}

type C4 struct {
	People          []Person         `yaml:"people"`
	SoftwareSystems []SoftwareSystem `yaml:"softwareSystems"`
	Containers      []Container      `yaml:"containers"`
	Components      []Component      `yaml:"components"`
}

type Relationship struct {
	Type        string   `yaml:"type"`
	From        string   `yaml:"from"`
	To          string   `yaml:"to"`
	Description string   `yaml:"description"`
	CatalogRefs []string `yaml:"catalogRefs"`
}

type Viewpoint struct {
	ID               string   `yaml:"id"`
	Kind             string   `yaml:"kind"`
	Roots            []string `yaml:"roots"`
	IncludeRelations []string `yaml:"includeRelations"`
}

type ArchitectureDocument struct {
	Model         ModelMeta      `yaml:"model"`
	C4            C4             `yaml:"c4"`
	Relationships []Relationship `yaml:"relationships"`
	Viewpoints    []Viewpoint    `yaml:"viewpoints"`
}

type Bundle struct {
	ArchitecturePath string
	CatalogPath      string

	Architecture ArchitectureDocument
	Catalog      CatalogDocument
}
