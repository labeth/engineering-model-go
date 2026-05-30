// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

// ENGMODEL-LINKS: EM-CATALOG-ENTRY
// TRLC-LINKS: REQ-EMG-001
type CatalogEntry struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Definition string   `yaml:"definition"`
	Aliases    []string `yaml:"aliases"`
}

// ENGMODEL-LINKS: EM-CATALOG
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

// ENGMODEL-LINKS: EM-CATALOG
type CatalogDocument struct {
	Catalog CatalogGroups `yaml:"catalog"`
}

// ENGMODEL-LINKS: EM-LINT-RUN
type LintRun struct {
	ID         string `yaml:"id"`
	Mode       string `yaml:"mode"`
	CommaAsAnd bool   `yaml:"commaAsAnd"`
	CatalogRef string `yaml:"catalogRef"`
}

// ENGMODEL-LINKS: EM-REQUIREMENT
type Requirement struct {
	ID        string   `yaml:"id"`
	Text      string   `yaml:"text"`
	Notes     string   `yaml:"notes"`
	AppliesTo []string `yaml:"appliesTo"`
}

// ENGMODEL-LINKS: EM-REQUIREMENT
type RequirementsDocument struct {
	LintRun      LintRun       `yaml:"lintRun"`
	Requirements []Requirement `yaml:"requirements"`
	Expected     []Expected    `yaml:"expected"`
}

// ENGMODEL-LINKS: EM-DECISION
type DecisionsDocument struct {
	Decisions []Decision `yaml:"decisions"`
}

// ENGMODEL-LINKS: EM-REQUIREMENT
type Expected struct {
	ID      string `yaml:"id"`
	Pattern string `yaml:"pattern"`
}

// ENGMODEL-LINKS: EM-DESIGN-VIEW
type DesignView struct {
	Title     string `yaml:"title"`
	Narrative string `yaml:"narrative"`
}

// ENGMODEL-LINKS: EM-DESIGN, EM-FUNCTIONAL-GROUP
type DesignFunctionalGroup struct {
	ID    string                `yaml:"id"`
	Views map[string]DesignView `yaml:"views"`
}

// ENGMODEL-LINKS: EM-DESIGN, EM-FUNCTIONAL-UNIT
type DesignFunctionalUnit struct {
	ID    string                `yaml:"id"`
	Group string                `yaml:"group"`
	Views map[string]DesignView `yaml:"views"`
}

// ENGMODEL-LINKS: EM-DESIGN
type DesignModel struct {
	ID               string                  `yaml:"id"`
	Title            string                  `yaml:"title"`
	FunctionalGroups []DesignFunctionalGroup `yaml:"functionalGroups"`
	FunctionalUnits  []DesignFunctionalUnit  `yaml:"functionalUnits"`
}

// ENGMODEL-LINKS: EM-DESIGN
type DesignDocument struct {
	Design DesignModel `yaml:"design"`
}

// ENGMODEL-LINKS: EM-MODEL
type ModelMeta struct {
	ID             string `yaml:"id"`
	Title          string `yaml:"title"`
	Introduction   string `yaml:"introduction"`
	BaseCatalogRef string `yaml:"baseCatalogRef"`
}

// ENGMODEL-LINKS: EM-DECISION
type Decision struct {
	ID           string   `yaml:"id"`
	Title        string   `yaml:"title"`
	Status       string   `yaml:"status"`
	Date         string   `yaml:"date"`
	Context      string   `yaml:"context"`
	Decision     string   `yaml:"decision"`
	Consequences []string `yaml:"consequences"`
}

// ENGMODEL-LINKS: EM-FUNCTIONAL-GROUP
type FunctionalGroup struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Prose       string   `yaml:"prose"`
	Tags        []string `yaml:"tags"`
}

// ENGMODEL-LINKS: EM-FUNCTIONAL-UNIT
type FunctionalUnit struct {
	ID    string   `yaml:"id"`
	Name  string   `yaml:"name"`
	Group string   `yaml:"group"`
	Tags  []string `yaml:"tags"`
	Prose string   `yaml:"prose"`
}

// ENGMODEL-LINKS: EM-ACTOR
type Actor struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-ATTACK-VECTOR
type AttackVector struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-REFERENCED-ELEMENT
type ReferencedElement struct {
	ID    string `yaml:"id"`
	Kind  string `yaml:"kind"`
	Layer string `yaml:"layer"`
	Name  string `yaml:"name"`
}

// ENGMODEL-LINKS: EM-AUTHORED-MAPPING
type Mapping struct {
	Type        string `yaml:"type"`
	From        string `yaml:"from"`
	To          string `yaml:"to"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-INTERFACE
type Interface struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	Protocol  string `yaml:"protocol"`
	Endpoint  string `yaml:"endpoint"`
	SchemaRef string `yaml:"schemaRef"`
	Owner     string `yaml:"owner"`
}

// ENGMODEL-LINKS: EM-DATA-OBJECT
type DataObject struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	TermRef         string   `yaml:"termRef"`
	SchemaRef       string   `yaml:"schemaRef"`
	Sensitivity     string   `yaml:"sensitivity"`
	Classification  string   `yaml:"classification"`
	RegulatoryTags  []string `yaml:"regulatoryTags"`
	Retention       string   `yaml:"retention"`
	Confidentiality string   `yaml:"confidentiality"`
	Integrity       string   `yaml:"integrity"`
	Availability    string   `yaml:"availability"`
}

// ENGMODEL-LINKS: EM-DEPLOYMENT-TARGET
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

// ENGMODEL-LINKS: EM-CONTROL
type Control struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Category    string `yaml:"category"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-EVIDENCE, EM-CONTROL
type ControlEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-CONTROL-ALLOCATION, EM-CONTROL
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

// ENGMODEL-LINKS: EM-EVIDENCE, EM-RISK
type RiskEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-RISK
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
	ThreatScenarios []string       `yaml:"threatScenarios"`
	Evidence        []RiskEvidence `yaml:"evidence"`
	ResidualRisk    string         `yaml:"residualRisk"`
	Rationale       string         `yaml:"rationale"`
}

// ENGMODEL-LINKS: EM-EVIDENCE, EM-POAM-ITEM
type POAMArtifact struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-POAM-ITEM, EM-RISK
type POAMItem struct {
	ID              string         `yaml:"id"`
	RiskRef         string         `yaml:"riskRef"`
	Milestone       string         `yaml:"milestone"`
	DueDate         string         `yaml:"dueDate"`
	Status          string         `yaml:"status"`
	ResponsibleRole string         `yaml:"responsibleRole"`
	Artifacts       []POAMArtifact `yaml:"artifacts"`
}

// ENGMODEL-LINKS: EM-TRUST-BOUNDARY
type TrustBoundary struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	BoundaryType string   `yaml:"boundaryType"`
	ParentRef    string   `yaml:"parentRef"`
	Members      []string `yaml:"members"`
}

// ENGMODEL-LINKS: EM-STATE
type State struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-EVENT
type Event struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-FLOW-STEP
type FlowStep struct {
	ID                  string   `yaml:"id"`
	Ref                 string   `yaml:"ref"`
	Kind                string   `yaml:"kind"`
	FlowType            string   `yaml:"flowType"`
	Direction           string   `yaml:"direction"`
	Frequency           string   `yaml:"frequency"`
	SourceRef           string   `yaml:"sourceRef"`
	DestinationRef      string   `yaml:"destinationRef"`
	Action              string   `yaml:"action"`
	Channel             string   `yaml:"channel"`
	Protocol            string   `yaml:"protocol"`
	DataIn              []string `yaml:"dataIn"`
	DataOut             []string `yaml:"dataOut"`
	DataRefs            []string `yaml:"dataRefs"`
	InterfaceRef        string   `yaml:"interfaceRef"`
	TrustBoundaryRef    string   `yaml:"trustBoundaryRef"`
	BoundaryCrossing    bool     `yaml:"boundaryCrossing"`
	Authentication      string   `yaml:"authentication"`
	EncryptionInTransit string   `yaml:"encryptionInTransit"`
	IntegrityProtection string   `yaml:"integrityProtection"`
	Next                []string `yaml:"next"`
	OnError             []string `yaml:"onError"`
	Async               bool     `yaml:"async"`
	Optional            bool     `yaml:"optional"`
}

// ENGMODEL-LINKS: EM-FLOW
type Flow struct {
	ID                  string            `yaml:"id"`
	Title               string            `yaml:"title"`
	Kind                string            `yaml:"kind"`
	Methodology         string            `yaml:"methodology"`
	Direction           string            `yaml:"direction"`
	Frequency           string            `yaml:"frequency"`
	SourceRef           string            `yaml:"sourceRef"`
	DestinationRef      string            `yaml:"destinationRef"`
	Protocol            string            `yaml:"protocol"`
	Channel             string            `yaml:"channel"`
	Authentication      string            `yaml:"authentication"`
	EncryptionInTransit string            `yaml:"encryptionInTransit"`
	IntegrityProtection string            `yaml:"integrityProtection"`
	DataRefs            []string          `yaml:"dataRefs"`
	Description         string            `yaml:"description"`
	Criticality         string            `yaml:"criticality"`
	Threats             []string          `yaml:"threats"`
	Entry               []string          `yaml:"entry"`
	Exits               []string          `yaml:"exits"`
	Steps               []FlowStep        `yaml:"steps"`
	Properties          map[string]string `yaml:"properties"`
}

// ENGMODEL-LINKS: EM-EVIDENCE, EM-THREAT-SCENARIO
type ThreatScenarioEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-THREAT-SCENARIO, EM-ATTACK-VECTOR
type ThreatScenario struct {
	ID               string                   `yaml:"id"`
	Title            string                   `yaml:"title"`
	Summary          string                   `yaml:"summary"`
	Category         string                   `yaml:"category"`
	Stride           string                   `yaml:"stride"`
	CWE              []string                 `yaml:"cwe"`
	AttackVectorRef  string                   `yaml:"attackVectorRef"`
	AppliesTo        []string                 `yaml:"appliesTo"`
	FlowRefs         []string                 `yaml:"flowRefs"`
	EntryPoint       string                   `yaml:"entryPoint"`
	Preconditions    []string                 `yaml:"preconditions"`
	ExploitPath      []string                 `yaml:"exploitPath"`
	Likelihood       string                   `yaml:"likelihood"`
	Impact           string                   `yaml:"impact"`
	Severity         string                   `yaml:"severity"`
	Status           string                   `yaml:"status"`
	Owner            string                   `yaml:"owner"`
	RiskRef          string                   `yaml:"riskRef"`
	RelatedControls  []string                 `yaml:"relatedControls"`
	AssumptionRefs   []string                 `yaml:"assumptionRefs"`
	OutOfScopeRefs   []string                 `yaml:"outOfScopeRefs"`
	MitigationRefs   []string                 `yaml:"mitigationRefs"`
	VerificationRefs []string                 `yaml:"verificationRefs"`
	Detection        []string                 `yaml:"detection"`
	Evidence         []ThreatScenarioEvidence `yaml:"evidence"`
}

// ENGMODEL-LINKS: EM-EVIDENCE, EM-THREAT-ASSUMPTION
type ThreatAssumptionEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-THREAT-ASSUMPTION
type ThreatAssumption struct {
	ID        string                     `yaml:"id"`
	Title     string                     `yaml:"title"`
	Statement string                     `yaml:"statement"`
	Status    string                     `yaml:"status"`
	Owner     string                     `yaml:"owner"`
	AppliesTo []string                   `yaml:"appliesTo"`
	Rationale string                     `yaml:"rationale"`
	Evidence  []ThreatAssumptionEvidence `yaml:"evidence"`
}

// ENGMODEL-LINKS: EM-EVIDENCE, EM-THREAT-OUT-OF-SCOPE
type ThreatOutOfScopeEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-THREAT-OUT-OF-SCOPE
type ThreatOutOfScope struct {
	ID        string                     `yaml:"id"`
	Title     string                     `yaml:"title"`
	Reason    string                     `yaml:"reason"`
	Status    string                     `yaml:"status"`
	Owner     string                     `yaml:"owner"`
	AppliesTo []string                   `yaml:"appliesTo"`
	ExpiresOn string                     `yaml:"expiresOn"`
	Evidence  []ThreatOutOfScopeEvidence `yaml:"evidence"`
}

// ENGMODEL-LINKS: EM-EVIDENCE, EM-THREAT-MITIGATION
type ThreatMitigationEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-THREAT-MITIGATION, EM-CONTROL
type ThreatMitigation struct {
	ID                string                     `yaml:"id"`
	ThreatScenarioRef string                     `yaml:"threatScenarioRef"`
	ControlRef        string                     `yaml:"controlRef"`
	Status            string                     `yaml:"status"`
	Effectiveness     string                     `yaml:"effectiveness"`
	Owner             string                     `yaml:"owner"`
	Notes             string                     `yaml:"notes"`
	VerificationRefs  []string                   `yaml:"verificationRefs"`
	Evidence          []ThreatMitigationEvidence `yaml:"evidence"`
}

// ENGMODEL-LINKS: EM-EVIDENCE, EM-CONTROL-VERIFICATION
type ControlVerificationEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: EM-CONTROL-VERIFICATION, EM-CONTROL
type ControlVerification struct {
	ID                 string                        `yaml:"id"`
	ControlRef         string                        `yaml:"controlRef"`
	ThreatScenarioRefs []string                      `yaml:"threatScenarioRefs"`
	RiskRefs           []string                      `yaml:"riskRefs"`
	Method             string                        `yaml:"method"`
	Status             string                        `yaml:"status"`
	Owner              string                        `yaml:"owner"`
	LastTested         string                        `yaml:"lastTested"`
	Findings           []string                      `yaml:"findings"`
	Evidence           []ControlVerificationEvidence `yaml:"evidence"`
}

// ENGMODEL-LINKS: EM-MODEL
type AuthoredArchitecture struct {
	FunctionalGroups     []FunctionalGroup     `yaml:"functionalGroups"`
	FunctionalUnits      []FunctionalUnit      `yaml:"functionalUnits"`
	Actors               []Actor               `yaml:"actors"`
	AttackVectors        []AttackVector        `yaml:"attackVectors"`
	ReferencedElements   []ReferencedElement   `yaml:"referencedElements"`
	Interfaces           []Interface           `yaml:"interfaces"`
	DataObjects          []DataObject          `yaml:"dataObjects"`
	DeploymentTargets    []DeploymentTarget    `yaml:"deploymentTargets"`
	Controls             []Control             `yaml:"controls"`
	ControlAllocations   []ControlAllocation   `yaml:"controlAllocations"`
	Risks                []Risk                `yaml:"risks"`
	POAMItems            []POAMItem            `yaml:"poamItems"`
	TrustBoundaries      []TrustBoundary       `yaml:"trustBoundaries"`
	States               []State               `yaml:"states"`
	Events               []Event               `yaml:"events"`
	Flows                []Flow                `yaml:"flows"`
	ThreatScenarios      []ThreatScenario      `yaml:"threatScenarios"`
	ThreatAssumptions    []ThreatAssumption    `yaml:"threatAssumptions"`
	ThreatOutOfScope     []ThreatOutOfScope    `yaml:"threatOutOfScope"`
	ThreatMitigations    []ThreatMitigation    `yaml:"threatMitigations"`
	ControlVerifications []ControlVerification `yaml:"controlVerifications"`
	Mappings             []Mapping             `yaml:"mappings"`
}

// ENGMODEL-LINKS: EM-INFERENCE-HINT
type InferenceHints struct {
	RuntimeSources           []string `yaml:"runtimeSources"`
	CodeSources              []string `yaml:"codeSources"`
	ExpectedRuntimeKinds     []string `yaml:"expectedRuntimeKinds"`
	OwnershipResolutionOrder []string `yaml:"ownershipResolutionOrder"`
}

// ENGMODEL-LINKS: EM-VIEW
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

// ENGMODEL-LINKS: EM-MODEL
type ArchitectureDocument struct {
	Model                ModelMeta            `yaml:"model"`
	Decisions            []Decision           `yaml:"-"`
	AuthoredArchitecture AuthoredArchitecture `yaml:"authoredArchitecture"`
	InferenceHints       InferenceHints       `yaml:"inferenceHints"`
	Views                []View               `yaml:"views"`
}

// ENGMODEL-LINKS: EM-BUNDLE
type Bundle struct {
	ArchitecturePath string
	CatalogPath      string
	DecisionsPath    string

	Architecture ArchitectureDocument
	Catalog      CatalogDocument
	Decisions    DecisionsDocument
}
