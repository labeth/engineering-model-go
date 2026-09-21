// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001
type CatalogEntry struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Definition string   `yaml:"definition"`
	Aliases    []string `yaml:"aliases"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type CatalogDocument struct {
	SchemaVersion int           `yaml:"schemaVersion"`
	Catalog       CatalogGroups `yaml:"catalog"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type LintRun struct {
	ID         string `yaml:"id"`
	Mode       string `yaml:"mode"`
	CommaAsAnd bool   `yaml:"commaAsAnd"`
	CatalogRef string `yaml:"catalogRef"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Requirement struct {
	ID                   string   `yaml:"id"`
	Title                string   `yaml:"title,omitempty"`
	Text                 string   `yaml:"text"`
	Notes                string   `yaml:"notes"`
	Category             string   `yaml:"category,omitempty"`
	Rationale            string   `yaml:"rationale,omitempty"`
	SourceRefs           []string `yaml:"sourceRefs,omitempty"`
	VerificationMethods  []string `yaml:"verificationMethods,omitempty"`
	VerificationCriteria string   `yaml:"verificationCriteria,omitempty"`
	Priority             string   `yaml:"priority,omitempty"`
	Criticality          string   `yaml:"criticality,omitempty"`
	Status               string   `yaml:"status,omitempty"`
	Derived              bool     `yaml:"derived,omitempty"`
	DerivedRationale     string   `yaml:"derivedRationale,omitempty"`
	Tags                 []string `yaml:"tags,omitempty"`
	AppliesTo            []string `yaml:"appliesTo"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type RequirementsDocument struct {
	SchemaVersion int           `yaml:"schemaVersion"`
	LintRun       LintRun       `yaml:"lintRun"`
	Requirements  []Requirement `yaml:"requirements"`
	Expected      []Expected    `yaml:"expected"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type DecisionsDocument struct {
	SchemaVersion int        `yaml:"schemaVersion"`
	Decisions     []Decision `yaml:"decisions"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Expected struct {
	ID      string `yaml:"id"`
	Pattern string `yaml:"pattern"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-VIEW-PROJECTION
type DesignView struct {
	Title     string `yaml:"title"`
	Narrative string `yaml:"narrative"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type DesignFunctionalGroup struct {
	ID    string                `yaml:"id"`
	Views map[string]DesignView `yaml:"views"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type DesignFunctionalUnit struct {
	ID    string                `yaml:"id"`
	Group string                `yaml:"group"`
	Views map[string]DesignView `yaml:"views"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type DesignModel struct {
	ID               string                  `yaml:"id"`
	Title            string                  `yaml:"title"`
	FunctionalGroups []DesignFunctionalGroup `yaml:"functionalGroups"`
	FunctionalUnits  []DesignFunctionalUnit  `yaml:"functionalUnits"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type DesignDocument struct {
	SchemaVersion int         `yaml:"schemaVersion"`
	Design        DesignModel `yaml:"design"`
}

// DocumentReferences makes the complete schema-v2 domain document set explicit.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
type DocumentReferences struct {
	Catalog      string `yaml:"catalog"`
	Requirements string `yaml:"requirements"`
	Architecture string `yaml:"architecture"`
	Behavior     string `yaml:"behavior"`
	Assurance    string `yaml:"assurance"`
	Compliance   string `yaml:"compliance"`
	Views        string `yaml:"views"`
	Decisions    string `yaml:"decisions"`
	Aviation     string `yaml:"aviation,omitempty"`
}

// ModuleIdentity is the published identity and human-facing metadata of a model.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-047
type ModuleIdentity struct {
	Path         string `yaml:"path"`
	Version      string `yaml:"version"`
	ModelID      string `yaml:"modelId"`
	Title        string `yaml:"title"`
	Introduction string `yaml:"introduction"`
	Kind         string `yaml:"kind"`
}

// ManifestDependency selects publications from an exact external module.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
type ManifestDependency struct {
	Alias        string   `yaml:"alias"`
	Path         string   `yaml:"path"`
	Version      string   `yaml:"version"`
	Publications []string `yaml:"publications"`
}

// ManifestPublication exports typed stable identifiers from this module.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
type ManifestPublication struct {
	ID           string   `yaml:"id"`
	Architecture []string `yaml:"architecture"`
	Behavior     []string `yaml:"behavior"`
	Assurance    []string `yaml:"assurance"`
	Compliance   []string `yaml:"compliance"`
	Requirements []string `yaml:"requirements"`
	Views        []string `yaml:"views"`
}

// ManifestDocument is the only schema-v2 model entry point.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
type ManifestDocument struct {
	SchemaVersion  int                   `yaml:"schemaVersion"`
	Module         ModuleIdentity        `yaml:"module"`
	Documents      DocumentReferences    `yaml:"documents"`
	Dependencies   []ManifestDependency  `yaml:"dependencies"`
	Publications   []ManifestPublication `yaml:"publications"`
	InferenceHints InferenceHints        `yaml:"inferenceHints"`
}

// ModelMeta is the aggregate model identity retained for downstream consumers.
type ModelMeta struct {
	ID             string             `yaml:"id"`
	Title          string             `yaml:"title"`
	Introduction   string             `yaml:"introduction"`
	Documents      DocumentReferences `yaml:"-"`
	BaseCatalogRef string             `yaml:"-"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Decision struct {
	ID           string   `yaml:"id"`
	Title        string   `yaml:"title"`
	Status       string   `yaml:"status"`
	Date         string   `yaml:"date"`
	Context      string   `yaml:"context"`
	Decision     string   `yaml:"decision"`
	Consequences []string `yaml:"consequences"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type FunctionalGroup struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Prose       string   `yaml:"prose"`
	Tags        []string `yaml:"tags"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type FunctionalUnit struct {
	ID    string   `yaml:"id"`
	Name  string   `yaml:"name"`
	Group string   `yaml:"group"`
	Tags  []string `yaml:"tags"`
	Prose string   `yaml:"prose"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Actor struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
type AttackVector struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type ReferencedElement struct {
	ID          string `yaml:"id"`
	Kind        string `yaml:"kind"`
	Layer       string `yaml:"layer"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	Date        string `yaml:"date"`
	URI         string `yaml:"uri"`
	Publisher   string `yaml:"publisher"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Mapping struct {
	Type        string `yaml:"type"`
	From        string `yaml:"from"`
	To          string `yaml:"to"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Interface struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	Protocol  string `yaml:"protocol"`
	Endpoint  string `yaml:"endpoint"`
	SchemaRef string `yaml:"schemaRef"`
	Owner     string `yaml:"owner"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, CTRL-TRACEABILITY-COVERAGE
type Control struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Category    string `yaml:"category"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, CTRL-TRACEABILITY-COVERAGE
type ControlEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, CTRL-TRACEABILITY-COVERAGE, FU-OSCAL-EXPORTER
type ComplianceProfile struct {
	ID          string `yaml:"id"`
	Href        string `yaml:"href"`
	CatalogHref string `yaml:"catalogHref"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, CTRL-TRACEABILITY-COVERAGE, FU-OSCAL-EXPORTER
type ComplianceMapping struct {
	ID                   string            `yaml:"id"`
	ProfileRef           string            `yaml:"profileRef"`
	ControlIDs           []string          `yaml:"controlIds"`
	ModelControlRef      string            `yaml:"modelControlRef"`
	AppliesTo            []string          `yaml:"appliesTo"`
	ImplementationType   string            `yaml:"implementationType"`
	ImplementationStatus string            `yaml:"implementationStatus"`
	Status               string            `yaml:"status"`
	Narrative            string            `yaml:"narrative"`
	Rationale            string            `yaml:"rationale"`
	Evidence             []ControlEvidence `yaml:"evidence"`
	InheritedFrom        []string          `yaml:"inheritedFrom"`
	ResponsibleRoles     []string          `yaml:"responsibleRoles"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, CTRL-TRACEABILITY-COVERAGE, FU-OSCAL-EXPORTER
type ComplianceModel struct {
	Profiles []ComplianceProfile `yaml:"profiles"`
	Mappings []ComplianceMapping `yaml:"mappings"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
type RiskEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type POAMArtifact struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
type POAMItem struct {
	ID              string         `yaml:"id"`
	RiskRef         string         `yaml:"riskRef"`
	Milestone       string         `yaml:"milestone"`
	DueDate         string         `yaml:"dueDate"`
	Status          string         `yaml:"status"`
	ResponsibleRole string         `yaml:"responsibleRole"`
	Artifacts       []POAMArtifact `yaml:"artifacts"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
type TrustBoundary struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	BoundaryType string   `yaml:"boundaryType"`
	ParentRef    string   `yaml:"parentRef"`
	Members      []string `yaml:"members"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, STATE-MODEL-VALID, STATE-MODEL-INVALID, STATE-ARTIFACTS-FRESH
type State struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, EVT-MODEL-UPDATED, EVT-GENERATION-RUN-REQUESTED, EVT-VALIDATION-FAILED, EVT-ARTIFACT-GENERATED
type Event struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
type ThreatScenarioEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
type ThreatAssumptionEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
type ThreatOutOfScopeEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER
type ThreatMitigationEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, CTRL-TRACEABILITY-COVERAGE
type ControlVerificationEvidence struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, CTRL-TRACEABILITY-COVERAGE
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

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type AuthoredArchitecture struct {
	FunctionalGroups     []FunctionalGroup     `yaml:"functionalGroups"`
	FunctionalUnits      []FunctionalUnit      `yaml:"functionalUnits"`
	Actors               []Actor               `yaml:"actors"`
	AttackVectors        []AttackVector        `yaml:"attackVectors"`
	ReferencedElements   []ReferencedElement   `yaml:"referencedElements"`
	Interfaces           []Interface           `yaml:"interfaces"`
	DataObjects          []DataObject          `yaml:"dataObjects"`
	DeploymentTargets    []DeploymentTarget    `yaml:"deploymentTargets"`
	HardwareItems        []HardwareItem        `yaml:"hardwareItems"`
	HardwareInterfaces   []HardwareInterface   `yaml:"hardwareInterfaces"`
	Controls             []Control             `yaml:"controls"`
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

// ArchitectureInput owns schema-v2 structural architecture declarations.
type ArchitectureInput struct {
	FunctionalGroups   []FunctionalGroup   `yaml:"functionalGroups"`
	FunctionalUnits    []FunctionalUnit    `yaml:"functionalUnits"`
	Actors             []Actor             `yaml:"actors"`
	ReferencedElements []ReferencedElement `yaml:"referencedElements"`
	Interfaces         []Interface         `yaml:"interfaces"`
	DataObjects        []DataObject        `yaml:"dataObjects"`
	DeploymentTargets  []DeploymentTarget  `yaml:"deploymentTargets"`
	HardwareItems      []HardwareItem      `yaml:"hardwareItems"`
	HardwareInterfaces []HardwareInterface `yaml:"hardwareInterfaces"`
	Contract           ContractModel       `yaml:"contract"`
	Composition        InputComposition    `yaml:"composition"`
	Semantics          SemanticContent     `yaml:"semantics"`
}

// InputComposition references manifest dependencies and their selected publications.
type InputComposition struct {
	Subsystems    []InputSubsystem `yaml:"subsystems"`
	Allocations   []Allocation     `yaml:"allocations"`
	Satisfactions []Satisfaction   `yaml:"satisfactions"`
}

// InputSubsystem identifies a child solely through a manifest dependency publication.
type InputSubsystem struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Dependency  string `yaml:"dependency"`
	Publication string `yaml:"publication"`
	Description string `yaml:"description"`
}

// ArchitectureInputDocument is the schema-v2 structural architecture document.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
type ArchitectureInputDocument struct {
	SchemaVersion int               `yaml:"schemaVersion"`
	Architecture  ArchitectureInput `yaml:"architecture"`
}

// BehaviorModel owns authored behavior and typed relationships.
type BehaviorModel struct {
	States        []State   `yaml:"states"`
	Events        []Event   `yaml:"events"`
	Flows         []Flow    `yaml:"flows"`
	Relationships []Mapping `yaml:"relationships"`
}

// BehaviorDocument is the schema-v2 behavioral document.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
type BehaviorDocument struct {
	SchemaVersion int           `yaml:"schemaVersion"`
	Behavior      BehaviorModel `yaml:"behavior"`
}

// AssuranceModel owns schema-v2 security and assurance declarations.
type AssuranceModel struct {
	AttackVectors        []AttackVector        `yaml:"attackVectors"`
	Controls             []Control             `yaml:"controls"`
	Risks                []Risk                `yaml:"risks"`
	POAMItems            []POAMItem            `yaml:"poamItems"`
	TrustBoundaries      []TrustBoundary       `yaml:"trustBoundaries"`
	ThreatScenarios      []ThreatScenario      `yaml:"threatScenarios"`
	ThreatAssumptions    []ThreatAssumption    `yaml:"threatAssumptions"`
	ThreatOutOfScope     []ThreatOutOfScope    `yaml:"threatOutOfScope"`
	ThreatMitigations    []ThreatMitigation    `yaml:"threatMitigations"`
	ControlVerifications []ControlVerification `yaml:"controlVerifications"`
}

// AssuranceDocument is the schema-v2 assurance document.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
type AssuranceDocument struct {
	SchemaVersion int            `yaml:"schemaVersion"`
	Assurance     AssuranceModel `yaml:"assurance"`
}

// ComplianceDocument is the schema-v2 compliance document.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
type ComplianceDocument struct {
	SchemaVersion int             `yaml:"schemaVersion"`
	Compliance    ComplianceModel `yaml:"compliance"`
}

// ViewsDocument owns projections, NAF metadata, and authored narratives.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
type ViewsDocument struct {
	SchemaVersion int                  `yaml:"schemaVersion"`
	Views         []View               `yaml:"views"`
	NAF           NAFProfile           `yaml:"naf"`
	Design        DesignModel          `yaml:"design"`
	Documents     []DocumentDefinition `yaml:"documents"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type InferenceHints struct {
	RuntimeSources           []string `yaml:"runtimeSources"`
	CodeSources              []string `yaml:"codeSources"`
	ExpectedRuntimeKinds     []string `yaml:"expectedRuntimeKinds"`
	OwnershipResolutionOrder []string `yaml:"ownershipResolutionOrder"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, FU-VIEW-PROJECTION
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

// DocumentControl captures format-neutral identity, lifecycle, and handling metadata.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
type DocumentControl struct {
	Identifier             string `yaml:"identifier"`
	Revision               string `yaml:"revision"`
	Status                 string `yaml:"status"`
	IssuedBy               string `yaml:"issuedBy"`
	IssueDate              string `yaml:"issueDate"`
	Language               string `yaml:"language"`
	DocumentType           string `yaml:"documentType"`
	Confidentiality        string `yaml:"confidentiality"`
	SecurityClassification string `yaml:"securityClassification"`
	ExportControlled       bool   `yaml:"exportControlled"`
	CountryOfOrigin        string `yaml:"countryOfOrigin"`
	ConfidentialityStamp   bool   `yaml:"confidentialityStamp"`
}

// DocumentSection adds authored prose and optional model selections to a document.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
type DocumentSection struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Narrative   string   `yaml:"narrative"`
	IncludeRefs []string `yaml:"includeRefs"`
}

// DocumentDefinition selects canonical model content for a formal document.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
type DocumentDefinition struct {
	ID              string            `yaml:"id"`
	Title           string            `yaml:"title"`
	Kind            string            `yaml:"kind"`
	Purpose         string            `yaml:"purpose"`
	Audience        []string          `yaml:"audience"`
	StakeholderRefs []string          `yaml:"stakeholderRefs"`
	ReferenceRefs   []string          `yaml:"referenceRefs"`
	ContentRefs     []string          `yaml:"contentRefs"`
	Sections        []DocumentSection `yaml:"sections"`
	Control         DocumentControl   `yaml:"control"`
}

// NAFProfile adds NATO Architecture Framework metadata to canonical model views.
// It does not define another architecture element graph.
// TRLC-LINKS: REQ-EMG-041
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
type NAFProfile struct {
	Framework               string           `yaml:"framework"`
	Version                 string           `yaml:"version"`
	ArchitectureDescription string           `yaml:"architectureDescription"`
	Stakeholders            []NAFStakeholder `yaml:"stakeholders"`
	Concerns                []NAFConcern     `yaml:"concerns"`
	Products                []NAFProduct     `yaml:"products"`
}

// Enabled reports whether a NAF profile was authored.
// TRLC-LINKS: REQ-EMG-041
func (p NAFProfile) Enabled() bool {
	return p.Framework != "" || p.Version != "" || p.ArchitectureDescription != "" ||
		len(p.Stakeholders) > 0 || len(p.Concerns) > 0 || len(p.Products) > 0
}

// NAFStakeholder binds a NAF stakeholder role to an existing canonical actor.
// TRLC-LINKS: REQ-EMG-041
type NAFStakeholder struct {
	ActorRef string `yaml:"actorRef"`
	Role     string `yaml:"role"`
}

// NAFConcern identifies an architecture concern and its existing stakeholders.
// TRLC-LINKS: REQ-EMG-041
type NAFConcern struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	StakeholderRefs []string `yaml:"stakeholderRefs"`
}

// NAFProduct applies an official NAF viewpoint to an existing Engmod view.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-042
type NAFProduct struct {
	ID          string   `yaml:"id"`
	Viewpoint   string   `yaml:"viewpoint"`
	Title       string   `yaml:"title"`
	ViewRef     string   `yaml:"viewRef"`
	ConcernRefs []string `yaml:"concernRefs"`
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type ArchitectureDocument struct {
	SchemaVersion        int                  `yaml:"schemaVersion"`
	Model                ModelMeta            `yaml:"model"`
	Semantics            SemanticContent      `yaml:"semantics"`
	Decisions            []Decision           `yaml:"-"`
	AuthoredArchitecture AuthoredArchitecture `yaml:"authoredArchitecture"`
	Compliance           ComplianceModel      `yaml:"compliance"`
	Contract             ContractModel        `yaml:"contract"`
	Composition          CompositionModel     `yaml:"composition"`
	InferenceHints       InferenceHints       `yaml:"inferenceHints"`
	NAF                  NAFProfile           `yaml:"naf"`
	Views                []View               `yaml:"views"`
}

// HardwareItem represents a physical platform element (the DO-254 side of the HW/SW boundary).
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type HardwareItem struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Kind        string   `yaml:"kind"` // processor|board|lru|sensor|actuator|bus|fpga|gateway|cloud
	PartNumber  string   `yaml:"partNumber"`
	Supplier    string   `yaml:"supplier"`
	SafetyLevel string   `yaml:"safetyLevel"` // DO-254 design assurance level (optional)
	Description string   `yaml:"description"`
	Hosts       []string `yaml:"hosts"` // functional unit or subsystem ids hosted on this item
}

// HardwareInterface represents a physical connection between hardware items (ICD/IRS data).
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type HardwareInterface struct {
	ID                   string `yaml:"id"`
	Name                 string `yaml:"name"`
	BusType              string `yaml:"busType"` // ARINC429|CAN|SPI|I2C|ethernet|cellular|discrete|analog
	From                 string `yaml:"from"`    // hardware item id
	To                   string `yaml:"to"`      // hardware item id
	Direction            string `yaml:"direction"`
	SoftwareInterfaceRef string `yaml:"softwareInterfaceRef"`
	Description          string `yaml:"description"`
}

// ContractModel is a system's published, parent-agnostic boundary: what it provides and requires.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type ContractModel struct {
	Provides []ContractEntry `yaml:"provides"` // the public surface a parent may allocate onto
	Requires []ContractEntry `yaml:"requires"` // what the system needs from its environment
}

// ContractEntry is a single provided or required interface/capability/assumption.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type ContractEntry struct {
	ID   string `yaml:"id"`
	Kind string `yaml:"kind"` // interface|capability|assumption
	Ref  string `yaml:"ref"`  // optional local id this contract entry exposes
	Note string `yaml:"note"`
}

// CompositionModel holds parent-only composition data: referenced subsystems and the allocation of
// this system's requirements onto them. Absent in a leaf system.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type CompositionModel struct {
	Subsystems    []Subsystem    `yaml:"subsystems"`
	Allocations   []Allocation   `yaml:"allocations"`
	Satisfactions []Satisfaction `yaml:"satisfactions"`
}

// Subsystem is a downward reference to one publication selected from a manifest
// dependency.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Subsystem struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Dependency  string `yaml:"dependency"`
	Publication string `yaml:"publication"`
	Description string `yaml:"description"`
}

// WorkspaceDocument provides local development replacements for published CUE
// module identities. It is never included in a published module.
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
type WorkspaceDocument struct {
	SchemaVersion int                    `yaml:"schemaVersion"`
	Replacements  []WorkspaceReplacement `yaml:"replacements"`
}

// WorkspaceReplacement maps one CUE module identity to a local repository.
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
type WorkspaceReplacement struct {
	Module string `yaml:"module"`
	Path   string `yaml:"path"`
}

// Allocation binds a parent requirement onto an alias-qualified identifier
// visible through the subsystem's selected publication.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Allocation struct {
	Requirement string `yaml:"requirement"` // this system's requirement id
	To          string `yaml:"to"`          // subsystem id or hardware item id
	Target      string `yaml:"target"`      // alias::ID selected by the subsystem publication
	Rationale   string `yaml:"rationale"`
}

// Satisfaction records how a published subsystem need is satisfied by a
// publication-qualified provider or a local hardware item.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Satisfaction struct {
	Need string `yaml:"need"` // alias::ID
	By   string `yaml:"by"`   // alias::ID or local hardware item id
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type Bundle struct {
	ManifestPath     string
	ArchitecturePath string
	BehaviorPath     string
	AssurancePath    string
	CompliancePath   string
	ViewsPath        string
	CatalogPath      string
	DecisionsPath    string
	RequirementsPath string
	DesignPath       string
	AviationPath     string

	Manifest     ManifestDocument
	Architecture ArchitectureDocument
	Behavior     BehaviorDocument
	Assurance    AssuranceDocument
	Compliance   ComplianceDocument
	Views        ViewsDocument
	Catalog      CatalogDocument
	Decisions    DecisionsDocument
	Requirements RequirementsDocument
	Design       DesignDocument
	Aviation     *AviationDocument
}
