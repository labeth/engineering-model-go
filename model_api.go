// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package engmodel

import "github.com/labeth/engineering-model-go/internal/model"

// Model types. Each is an alias of the internal model type, so values move
// freely between the facade and every generator in this package. The set is
// every exported type reachable from Bundle, CanonicalBundle,
// WorkspaceDocument, ManifestDocument, AuthoringContract and SemanticModel.
type (
	Actor                       = model.Actor
	AirborneApproval            = model.AirborneApproval
	AirborneProfile             = model.AirborneProfile
	AirborneReview              = model.AirborneReview
	AirborneTool                = model.AirborneTool
	AirborneVerificationCase    = model.AirborneVerificationCase
	Allocation                  = model.Allocation
	ArchitectureDocument        = model.ArchitectureDocument
	AssuranceDocument           = model.AssuranceDocument
	AssuranceModel              = model.AssuranceModel
	AttackVector                = model.AttackVector
	AuthoredArchitecture        = model.AuthoredArchitecture
	AuthoringContract           = model.AuthoringContract
	AuthoringDocument           = model.AuthoringDocument
	AviationDocument            = model.AviationDocument
	AviationProfile             = model.AviationProfile
	AviationValidationError     = model.AviationValidationError
	BehaviorDocument            = model.BehaviorDocument
	BehaviorModel               = model.BehaviorModel
	Bundle                      = model.Bundle
	CanonicalBundle             = model.CanonicalBundle
	CatalogDocument             = model.CatalogDocument
	CatalogEntry                = model.CatalogEntry
	CatalogGroups               = model.CatalogGroups
	CertificationLiaison        = model.CertificationLiaison
	ComplianceDocument          = model.ComplianceDocument
	ComplianceMapping           = model.ComplianceMapping
	ComplianceModel             = model.ComplianceModel
	ComplianceProfile           = model.ComplianceProfile
	CompositionModel            = model.CompositionModel
	ConfigurationArtifact       = model.ConfigurationArtifact
	ConfigurationBaseline       = model.ConfigurationBaseline
	ConfigurationBuild          = model.ConfigurationBuild
	ContractEntry               = model.ContractEntry
	ContractModel               = model.ContractModel
	Control                     = model.Control
	ControlEvidence             = model.ControlEvidence
	ControlVerification         = model.ControlVerification
	ControlVerificationEvidence = model.ControlVerificationEvidence
	DataObject                  = model.DataObject
	Decision                    = model.Decision
	DecisionsDocument           = model.DecisionsDocument
	DependencyGuide             = model.DependencyGuide
	DeploymentTarget            = model.DeploymentTarget
	DesignDocument              = model.DesignDocument
	DesignFunctionalGroup       = model.DesignFunctionalGroup
	DesignFunctionalUnit        = model.DesignFunctionalUnit
	DesignModel                 = model.DesignModel
	DesignView                  = model.DesignView
	DocumentControl             = model.DocumentControl
	DocumentDefinition          = model.DocumentDefinition
	DocumentReferences          = model.DocumentReferences
	DocumentSection             = model.DocumentSection
	Event                       = model.Event
	EvidenceReference           = model.EvidenceReference
	Expected                    = model.Expected
	Flow                        = model.Flow
	FlowStep                    = model.FlowStep
	FunctionalGroup             = model.FunctionalGroup
	FunctionalUnit              = model.FunctionalUnit
	HardwareInterface           = model.HardwareInterface
	HardwareItem                = model.HardwareItem
	IndependenceRecord          = model.IndependenceRecord
	InferenceHints              = model.InferenceHints
	Interface                   = model.Interface
	LifecycleDataRecord         = model.LifecycleDataRecord
	LintRun                     = model.LintRun
	LintWaiver                  = model.LintWaiver
	ManifestDependency          = model.ManifestDependency
	ManifestDocument            = model.ManifestDocument
	ManifestPublication         = model.ManifestPublication
	Mapping                     = model.Mapping
	MetamodelPropertyValue      = model.MetamodelPropertyValue
	MetamodelValue              = model.MetamodelValue
	ModelMeta                   = model.ModelMeta
	ModuleIdentity              = model.ModuleIdentity
	Multiplicity                = model.Multiplicity
	NAFConcern                  = model.NAFConcern
	NAFProduct                  = model.NAFProduct
	NAFProfile                  = model.NAFProfile
	NAFStakeholder              = model.NAFStakeholder
	ObjectiveReference          = model.ObjectiveReference
	POAMArtifact                = model.POAMArtifact
	POAMItem                    = model.POAMItem
	ProblemReport               = model.ProblemReport
	ReferencedElement           = model.ReferencedElement
	Requirement                 = model.Requirement
	RequirementAssurance        = model.RequirementAssurance
	RequirementsDocument        = model.RequirementsDocument
	Risk                        = model.Risk
	RiskEvidence                = model.RiskEvidence
	Satisfaction                = model.Satisfaction
	SemanticContent             = model.SemanticContent
	SemanticDiagnostic          = model.SemanticDiagnostic
	SemanticElement             = model.SemanticElement
	SemanticElementKind         = model.SemanticElementKind
	SemanticExpression          = model.SemanticExpression
	SemanticFeature             = model.SemanticFeature
	SemanticFeatureKind         = model.SemanticFeatureKind
	SemanticImport              = model.SemanticImport
	SemanticMetadata            = model.SemanticMetadata
	SemanticModel               = model.SemanticModel
	SemanticQuantity            = model.SemanticQuantity
	SemanticRelationship        = model.SemanticRelationship
	SemanticRelationshipKind    = model.SemanticRelationshipKind
	SemanticSeverity            = model.SemanticSeverity
	SemanticValue               = model.SemanticValue
	SQAAudit                    = model.SQAAudit
	StableIdentifierGuide       = model.StableIdentifierGuide
	State                       = model.State
	StructuralCoverage          = model.StructuralCoverage
	Subsystem                   = model.Subsystem
	ThreatAssumption            = model.ThreatAssumption
	ThreatAssumptionEvidence    = model.ThreatAssumptionEvidence
	ThreatMitigation            = model.ThreatMitigation
	ThreatMitigationEvidence    = model.ThreatMitigationEvidence
	ThreatOutOfScope            = model.ThreatOutOfScope
	ThreatOutOfScopeEvidence    = model.ThreatOutOfScopeEvidence
	ThreatScenario              = model.ThreatScenario
	ThreatScenarioEvidence      = model.ThreatScenarioEvidence
	ToolAssessment              = model.ToolAssessment
	TrustBoundary               = model.TrustBoundary
	UncoveredDisposition        = model.UncoveredDisposition
	VerificationExecution       = model.VerificationExecution
	VerificationProcedure       = model.VerificationProcedure
	View                        = model.View
	ViewsDocument               = model.ViewsDocument
	WorkspaceDocument           = model.WorkspaceDocument
	WorkspaceReplacement        = model.WorkspaceReplacement
)

// Model constants.
const (
	// CurrentSchemaVersion is the document schemaVersion this release reads.
	CurrentSchemaVersion = model.CurrentSchemaVersion
	// AuthoringContractVersion identifies the authoring contract format.
	AuthoringContractVersion = model.AuthoringContractVersion
	// WorkspaceFileName is the workspace file looked up from a model directory upwards.
	WorkspaceFileName = model.WorkspaceFileName
	// WorkspaceEnv names the environment variable that selects the workspace.
	WorkspaceEnv = model.WorkspaceEnv
	// WorkspaceOff is the workspace mode that disables workspace lookup.
	WorkspaceOff = model.WorkspaceOff
)

// LoadBundle reads the engmod.yml manifest at manifestPath and every domain
// document it names, validated against the canonical CUE schema.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046, REQ-EMG-049, REQ-EMG-051, REQ-EMG-053, REQ-EMG-054
func LoadBundle(manifestPath string) (Bundle, error) {
	return model.LoadBundle(manifestPath)
}

// LoadRequirements reads a single requirements document.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046
func LoadRequirements(path string) (RequirementsDocument, error) {
	return model.LoadRequirements(path)
}

// LoadDesign reads a single design document.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046
func LoadDesign(path string) (DesignDocument, error) {
	return model.LoadDesign(path)
}

// LoadCanonicalBundle loads the manifest at manifestPath and projects it into
// the canonical semantic model; projection errors are returned as an error.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-044, REQ-EMG-046
func LoadCanonicalBundle(manifestPath string) (CanonicalBundle, error) {
	return model.LoadCanonicalBundle(manifestPath)
}

// NewCanonicalBundle projects an already loaded bundle into the canonical
// semantic model.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func NewCanonicalBundle(bundle Bundle) (CanonicalBundle, error) {
	return model.NewCanonicalBundle(bundle)
}

// ProjectSemanticModel projects a bundle into the canonical semantic model
// and returns the projection diagnostics.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-035
func ProjectSemanticModel(bundle Bundle) (SemanticModel, []SemanticDiagnostic) {
	return model.ProjectSemanticModel(bundle)
}

// ValidateSemanticModel checks identity, relationship and ownership
// invariants of a semantic model.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
func ValidateSemanticModel(semantic SemanticModel) []SemanticDiagnostic {
	return model.ValidateSemanticModel(semantic)
}

// HasSemanticErrors reports whether any semantic diagnostic is an error.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func HasSemanticErrors(diagnostics []SemanticDiagnostic) bool {
	return model.HasSemanticErrors(diagnostics)
}

// BuildAuthoringContract describes the documents, schemas, identifiers and
// dependency rules an author or agent needs to edit the model.
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-MODEL-AUTHORING-CONTRACT
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-045, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-053
func BuildAuthoringContract(bundle Bundle) AuthoringContract {
	return model.BuildAuthoringContract(bundle)
}

// LoadWorkspace reads the workspace file at path.
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
// TRLC-LINKS: REQ-EMG-048, REQ-EMG-049
func LoadWorkspace(path string) (WorkspaceDocument, error) {
	return model.LoadWorkspace(path)
}

// FindWorkspace looks for engmod.work.yml from start upwards, honouring
// ENGMOD_WORK, and returns the document and its path (empty when none).
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
// TRLC-LINKS: REQ-EMG-048
func FindWorkspace(start string) (WorkspaceDocument, string, error) {
	return model.FindWorkspace(start)
}

// FindWorkspaceWith is FindWorkspace with an explicit mode: "" (look up,
// honouring ENGMOD_WORK), WorkspaceOff, or a path to one workspace file.
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
// TRLC-LINKS: REQ-EMG-048, REQ-EMG-063
func FindWorkspaceWith(start, mode string) (WorkspaceDocument, string, error) {
	return model.FindWorkspaceWith(start, mode)
}

// QualifyReference joins a dependency alias and a local ID into alias::ID.
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func QualifyReference(alias, id string) (string, error) {
	return model.QualifyReference(alias, id)
}

// ParseQualifiedReference splits an alias::ID reference.
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func ParseQualifiedReference(ref string) (alias, id string, err error) {
	return model.ParseQualifiedReference(ref)
}
