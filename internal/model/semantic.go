// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SemanticElementKind identifies the SysML/KerML concept used by a canonical element.
type SemanticElementKind string

const (
	ElementPackageDefinition          SemanticElementKind = "package_definition"
	ElementPartDefinition             SemanticElementKind = "part_definition"
	ElementPartUsage                  SemanticElementKind = "part_usage"
	ElementItemDefinition             SemanticElementKind = "item_definition"
	ElementItemUsage                  SemanticElementKind = "item_usage"
	ElementAttributeDefinition        SemanticElementKind = "attribute_definition"
	ElementAttributeUsage             SemanticElementKind = "attribute_usage"
	ElementPortDefinition             SemanticElementKind = "port_definition"
	ElementPortUsage                  SemanticElementKind = "port_usage"
	ElementInterfaceDefinition        SemanticElementKind = "interface_definition"
	ElementInterfaceUsage             SemanticElementKind = "interface_usage"
	ElementConnectionDefinition       SemanticElementKind = "connection_definition"
	ElementConnectionUsage            SemanticElementKind = "connection_usage"
	ElementActionDefinition           SemanticElementKind = "action_definition"
	ElementActionUsage                SemanticElementKind = "action_usage"
	ElementControlNode                SemanticElementKind = "control_node"
	ElementStateDefinition            SemanticElementKind = "state_definition"
	ElementStateUsage                 SemanticElementKind = "state_usage"
	ElementEventDefinition            SemanticElementKind = "event_definition"
	ElementEventUsage                 SemanticElementKind = "event_usage"
	ElementViewDefinition             SemanticElementKind = "view_definition"
	ElementViewUsage                  SemanticElementKind = "view_usage"
	ElementViewpointDefinition        SemanticElementKind = "viewpoint_definition"
	ElementViewpointUsage             SemanticElementKind = "viewpoint_usage"
	ElementRequirementDefinition      SemanticElementKind = "requirement_definition"
	ElementRequirementUsage           SemanticElementKind = "requirement_usage"
	ElementConcernDefinition          SemanticElementKind = "concern_definition"
	ElementConcernUsage               SemanticElementKind = "concern_usage"
	ElementStakeholderDefinition      SemanticElementKind = "stakeholder_definition"
	ElementStakeholderUsage           SemanticElementKind = "stakeholder_usage"
	ElementConstraintDefinition       SemanticElementKind = "constraint_definition"
	ElementConstraintUsage            SemanticElementKind = "constraint_usage"
	ElementCalculationDefinition      SemanticElementKind = "calculation_definition"
	ElementCalculationUsage           SemanticElementKind = "calculation_usage"
	ElementCaseDefinition             SemanticElementKind = "case_definition"
	ElementCaseUsage                  SemanticElementKind = "case_usage"
	ElementAnalysisCaseDefinition     SemanticElementKind = "analysis_case_definition"
	ElementAnalysisCaseUsage          SemanticElementKind = "analysis_case_usage"
	ElementVerificationCaseDefinition SemanticElementKind = "verification_case_definition"
	ElementVerificationCaseUsage      SemanticElementKind = "verification_case_usage"
	ElementUseCaseDefinition          SemanticElementKind = "use_case_definition"
	ElementUseCaseUsage               SemanticElementKind = "use_case_usage"
	ElementQuantityDefinition         SemanticElementKind = "quantity_definition"
	ElementUnitDefinition             SemanticElementKind = "unit_definition"
	ElementOccurrenceDefinition       SemanticElementKind = "occurrence_definition"
	ElementOccurrenceUsage            SemanticElementKind = "occurrence_usage"
	ElementIndividualDefinition       SemanticElementKind = "individual_definition"
	ElementIndividualUsage            SemanticElementKind = "individual_usage"
	ElementSnapshot                   SemanticElementKind = "snapshot"
	ElementTimeSlice                  SemanticElementKind = "time_slice"
	ElementExtension                  SemanticElementKind = "engineering_extension"
	ElementRequirement                SemanticElementKind = ElementRequirementDefinition
)

// SemanticRelationshipKind identifies a typed relationship without reducing it to a generic edge.
type SemanticRelationshipKind string

const (
	RelationshipContainment    SemanticRelationshipKind = "containment"
	RelationshipDependency     SemanticRelationshipKind = "dependency"
	RelationshipTyping         SemanticRelationshipKind = "typing"
	RelationshipSpecialization SemanticRelationshipKind = "specialization"
	RelationshipSubsetting     SemanticRelationshipKind = "subsetting"
	RelationshipRedefinition   SemanticRelationshipKind = "redefinition"
	RelationshipConnection     SemanticRelationshipKind = "connection"
	RelationshipBinding        SemanticRelationshipKind = "binding"
	RelationshipTransfer       SemanticRelationshipKind = "transfer"
	RelationshipSuccession     SemanticRelationshipKind = "succession"
	RelationshipTransition     SemanticRelationshipKind = "transition"
	RelationshipEventTrigger   SemanticRelationshipKind = "event_trigger"
	RelationshipGuard          SemanticRelationshipKind = "guard"
	RelationshipEffect         SemanticRelationshipKind = "effect"
	RelationshipAllocation     SemanticRelationshipKind = "allocation"
	RelationshipSatisfaction   SemanticRelationshipKind = "satisfaction"
	RelationshipVerification   SemanticRelationshipKind = "verification"
	RelationshipVariant        SemanticRelationshipKind = "variant_membership"
	RelationshipReference      SemanticRelationshipKind = "reference"
	RelationshipExtension      SemanticRelationshipKind = "engineering_extension"
)

type SemanticSeverity string

const (
	SemanticSeverityError   SemanticSeverity = "error"
	SemanticSeverityWarning SemanticSeverity = "warning"
)

// SemanticDiagnostic records a projection limitation instead of silently dropping information.
type SemanticDiagnostic struct {
	Code     string           `json:"code"`
	Severity SemanticSeverity `json:"severity"`
	Message  string           `json:"message"`
	Path     string           `json:"path,omitempty"`
}

// Multiplicity is available to canonical features even though the current YAML schema
// only supplies multiplicity for a limited set of concepts.
type Multiplicity struct {
	Lower *int `json:"lower,omitempty" yaml:"lower,omitempty"`
	Upper *int `json:"upper,omitempty" yaml:"upper,omitempty"`
}

// SemanticExpression preserves an expression as a typed tree or source expression.
type SemanticExpression struct {
	Kind       string               `json:"kind,omitempty" yaml:"kind,omitempty"`
	Language   string               `json:"language,omitempty" yaml:"language,omitempty"`
	Operator   string               `json:"operator,omitempty" yaml:"operator,omitempty"`
	Value      string               `json:"value,omitempty" yaml:"value,omitempty"`
	TypedValue *SemanticValue       `json:"typedValue,omitempty" yaml:"typedValue,omitempty"`
	Operands   []SemanticExpression `json:"operands,omitempty" yaml:"operands,omitempty"`
}

// SemanticValue is the compact reusable representation for literals,
// references, quantities, and unit-bearing values.
type SemanticValue struct {
	Kind      string            `json:"kind" yaml:"kind"`
	String    string            `json:"string,omitempty" yaml:"string,omitempty"`
	Integer   *int64            `json:"integer,omitempty" yaml:"integer,omitempty"`
	Real      *float64          `json:"real,omitempty" yaml:"real,omitempty"`
	Boolean   *bool             `json:"boolean,omitempty" yaml:"boolean,omitempty"`
	Reference string            `json:"reference,omitempty" yaml:"reference,omitempty"`
	Quantity  *SemanticQuantity `json:"quantity,omitempty" yaml:"quantity,omitempty"`
}

type SemanticQuantity struct {
	Value float64 `json:"value" yaml:"value"`
	Unit  string  `json:"unit" yaml:"unit"`
}

// MetamodelValue is a lossless typed value used by official metamodel
// properties and Engineering Model extension properties.
//
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
type MetamodelValue struct {
	Kind      string                    `json:"kind" yaml:"kind"`
	String    string                    `json:"string,omitempty" yaml:"string,omitempty"`
	Integer   *int64                    `json:"integer,omitempty" yaml:"integer,omitempty"`
	Real      *float64                  `json:"real,omitempty" yaml:"real,omitempty"`
	Boolean   *bool                     `json:"boolean,omitempty" yaml:"boolean,omitempty"`
	Reference string                    `json:"reference,omitempty" yaml:"reference,omitempty"`
	Object    map[string]MetamodelValue `json:"object,omitempty" yaml:"object,omitempty"`
	List      []MetamodelValue          `json:"list,omitempty" yaml:"list,omitempty"`
}

// UnmarshalYAML accepts the typed canonical form and legacy scalar/list/object
// extension values at the input boundary.
//
// TRLC-LINKS: REQ-EMG-040
func (v *MetamodelValue) UnmarshalYAML(node *yaml.Node) error {
	type plain MetamodelValue
	if yamlMappingHasKind(node) {
		var decoded plain
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		*v = MetamodelValue(decoded)
		return nil
	}
	var raw any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	*v = typedMetamodelValue(raw)
	return nil
}

// yamlMappingHasKind reports whether node is a mapping with a "kind" key,
// which marks the typed canonical form of a metamodel value.
//
// TRLC-LINKS: REQ-EMG-040
func yamlMappingHasKind(node *yaml.Node) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == "kind" {
			return true
		}
	}
	return false
}

// MetamodelPropertyValue preserves ordered multiplicity and records values
// supplied by derivation or implication rather than authored assignment.
//
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
type MetamodelPropertyValue struct {
	Values  []MetamodelValue `json:"values" yaml:"values"`
	Derived bool             `json:"derived,omitempty" yaml:"derived,omitempty"`
	Implied bool             `json:"implied,omitempty" yaml:"implied,omitempty"`
}

type SemanticFeatureKind string

const (
	FeatureAttribute SemanticFeatureKind = "attribute"
	FeaturePort      SemanticFeatureKind = "port"
	FeatureParameter SemanticFeatureKind = "parameter"
)

// SemanticFeature is the common typed-feature representation used by exporters.
type SemanticFeature struct {
	Name         string              `json:"name" yaml:"name"`
	Kind         SemanticFeatureKind `json:"kind,omitempty" yaml:"kind,omitempty"`
	Type         string              `json:"type,omitempty" yaml:"type,omitempty"`
	Direction    string              `json:"direction,omitempty" yaml:"direction,omitempty"`
	Multiplicity *Multiplicity       `json:"multiplicity,omitempty" yaml:"multiplicity,omitempty"`
	Ordered      bool                `json:"ordered,omitempty" yaml:"ordered,omitempty"`
	Unique       *bool               `json:"unique,omitempty" yaml:"unique,omitempty"`
	Conjugated   bool                `json:"conjugated,omitempty" yaml:"conjugated,omitempty"`
	Specializes  []string            `json:"specializes,omitempty" yaml:"specializes,omitempty"`
	Subsets      []string            `json:"subsets,omitempty" yaml:"subsets,omitempty"`
	Redefines    []string            `json:"redefines,omitempty" yaml:"redefines,omitempty"`
	Value        *SemanticExpression `json:"value,omitempty" yaml:"value,omitempty"`
	References   []string            `json:"references,omitempty" yaml:"references,omitempty"`
}

// SemanticMetadata is a typed extension payload. Values are copied from the
// YAML-backed source object, so exporters can preserve information they cannot
// express as standard SysML concepts.
type SemanticMetadata struct {
	Namespace  string                    `json:"namespace" yaml:"namespace"`
	Type       string                    `json:"type" yaml:"type"`
	Target     string                    `json:"target" yaml:"target"`
	Properties map[string]MetamodelValue `json:"properties,omitempty" yaml:"properties,omitempty"`
}

// SemanticElement is the single canonical representation of an authored concept.
type SemanticElement struct {
	ID                 string                            `json:"id" yaml:"id"`
	Name               string                            `json:"name,omitempty" yaml:"name,omitempty"`
	Kind               SemanticElementKind               `json:"kind" yaml:"kind"`
	Metaclass          string                            `json:"metaclass,omitempty" yaml:"metaclass,omitempty"`
	Properties         map[string]MetamodelPropertyValue `json:"properties,omitempty" yaml:"properties,omitempty"`
	Namespace          string                            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Owner              string                            `json:"owner,omitempty" yaml:"owner,omitempty"`
	TypeRef            string                            `json:"typeRef,omitempty" yaml:"typeRef,omitempty"`
	Multiplicity       *Multiplicity                     `json:"multiplicity,omitempty" yaml:"multiplicity,omitempty"`
	Ordered            bool                              `json:"ordered,omitempty" yaml:"ordered,omitempty"`
	Unique             *bool                             `json:"unique,omitempty" yaml:"unique,omitempty"`
	Conjugated         bool                              `json:"conjugated,omitempty" yaml:"conjugated,omitempty"`
	Specializes        []string                          `json:"specializes,omitempty" yaml:"specializes,omitempty"`
	Subsets            []string                          `json:"subsets,omitempty" yaml:"subsets,omitempty"`
	Redefines          []string                          `json:"redefines,omitempty" yaml:"redefines,omitempty"`
	ControlKind        string                            `json:"controlKind,omitempty" yaml:"controlKind,omitempty"`
	OccurrenceID       string                            `json:"occurrenceId,omitempty" yaml:"occurrenceId,omitempty"`
	PortionOf          string                            `json:"portionOf,omitempty" yaml:"portionOf,omitempty"`
	Variation          bool                              `json:"variation,omitempty" yaml:"variation,omitempty"`
	Variants           []string                          `json:"variants,omitempty" yaml:"variants,omitempty"`
	References         []string                          `json:"references,omitempty" yaml:"references,omitempty"`
	Features           []SemanticFeature                 `json:"features,omitempty" yaml:"features,omitempty"`
	Metadata           []SemanticMetadata                `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	ExtensionNamespace string                            `json:"extensionNamespace,omitempty" yaml:"extensionNamespace,omitempty"`
	Extension          string                            `json:"extension,omitempty" yaml:"extension,omitempty"`
	Targets            []string                          `json:"targets,omitempty" yaml:"targets,omitempty"`
	Authored           bool                              `json:"-" yaml:"-"`
}

// SemanticRelationship preserves relationship identity, endpoints, and the
// original engineering relationship type.
type SemanticRelationship struct {
	ID         string                            `json:"id" yaml:"id"`
	Kind       SemanticRelationshipKind          `json:"kind" yaml:"kind"`
	Metaclass  string                            `json:"metaclass,omitempty" yaml:"metaclass,omitempty"`
	Properties map[string]MetamodelPropertyValue `json:"properties,omitempty" yaml:"properties,omitempty"`
	Source     string                            `json:"source" yaml:"source"`
	Target     string                            `json:"target" yaml:"target"`
	Owner      string                            `json:"owner,omitempty" yaml:"owner,omitempty"`
	SourceType string                            `json:"sourceType,omitempty" yaml:"sourceType,omitempty"`
	ItemRef    string                            `json:"itemRef,omitempty" yaml:"itemRef,omitempty"`
	Triggers   []string                          `json:"triggers,omitempty" yaml:"triggers,omitempty"`
	Guard      *SemanticExpression               `json:"guard,omitempty" yaml:"guard,omitempty"`
	Effect     *SemanticExpression               `json:"effect,omitempty" yaml:"effect,omitempty"`
	Metadata   []SemanticMetadata                `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type SemanticImport struct {
	Namespace  string `json:"namespace" yaml:"namespace"`
	Imported   string `json:"imported" yaml:"imported"`
	Visibility string `json:"visibility,omitempty" yaml:"visibility,omitempty"`
	Recursive  bool   `json:"recursive,omitempty" yaml:"recursive,omitempty"`
}

// SemanticContent is the canonical YAML surface for concepts that cannot be
// expressed by legacy architecture fields. Legacy fields remain migration
// aliases and are adapted into these same primitives.
type SemanticContent struct {
	Imports       []SemanticImport       `json:"imports,omitempty" yaml:"imports,omitempty"`
	Elements      []SemanticElement      `json:"elements,omitempty" yaml:"elements,omitempty"`
	Relationships []SemanticRelationship `json:"relationships,omitempty" yaml:"relationships,omitempty"`
}

// SemanticModel is the canonical exporter-facing projection. It is derived
// from Bundle and is not a second authored or persistent model.
type SemanticModel struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title,omitempty"`
	Imports       []SemanticImport       `json:"imports,omitempty"`
	Elements      []SemanticElement      `json:"elements"`
	Relationships []SemanticRelationship `json:"relationships"`
}

var semanticIDPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[A-Za-z0-9]`)

// semanticError builds the error diagnostic that projection and validation
// rules report.
//
// TRLC-LINKS: REQ-EMG-035
func semanticError(code, path, message string) SemanticDiagnostic {
	return SemanticDiagnostic{Code: code, Severity: SemanticSeverityError, Message: message, Path: path}
}

// elementSpec describes one semantic element to project.
type elementSpec struct {
	path      string
	kind      SemanticElementKind
	extension string
	id        string
	name      string
	owner     string
	typeRef   string
	source    any
	features  []SemanticFeature
}

// relationshipSpec describes one semantic relationship to project.
type relationshipSpec struct {
	path       string
	kind       SemanticRelationshipKind
	sourceType string
	source     string
	target     string
	owner      string
	value      any
}

// evidenceSpec describes one evidence or artifact item projected as an extension element.
type evidenceSpec struct {
	path         string
	extension    string
	target       string
	evidencePath string
	description  string
}

// semanticProjector accumulates the projection of one Bundle.
type semanticProjector struct {
	bundle              Bundle
	a                   AuthoredArchitecture
	out                 SemanticModel
	diagnostics         []SemanticDiagnostic
	elementIndex        map[string]int
	relationshipIDs     map[string]string
	relationshipAliases map[string]int
	catalogIDs          map[string]string
}

// ProjectSemanticModel derives one deterministic semantic graph from the
// YAML-backed Bundle. It never mutates or replaces the authored model.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func ProjectSemanticModel(bundle Bundle) (SemanticModel, []SemanticDiagnostic) {
	p := newSemanticProjector(bundle)
	p.projectRoot()
	p.projectAuthoredElements()
	p.projectAuthoredRelationships()
	p.projectOrganization()
	p.projectTechnicalItems()
	p.projectAssurance()
	p.projectFlows()
	p.projectMachines()
	p.projectThreats()
	p.projectMappings()
	p.projectCompliance()
	p.projectContractAndComposition()
	p.projectDocuments()
	p.projectRequirementsAndDesign()
	p.projectCatalog()
	p.finish()
	return p.out, p.diagnostics
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func newSemanticProjector(bundle Bundle) *semanticProjector {
	return &semanticProjector{
		bundle: bundle,
		a:      bundle.Architecture.AuthoredArchitecture,
		out: SemanticModel{
			ID:      strings.TrimSpace(bundle.Architecture.Model.ID),
			Title:   strings.TrimSpace(bundle.Architecture.Model.Title),
			Imports: append([]SemanticImport(nil), bundle.Architecture.Semantics.Imports...),
		},
		elementIndex:        map[string]int{},
		relationshipIDs:     map[string]string{},
		relationshipAliases: map[string]int{},
		catalogIDs:          map[string]string{},
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) report(diagnostic SemanticDiagnostic) {
	p.diagnostics = append(p.diagnostics, diagnostic)
}

// elementAt returns the projected element with the given source ID.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) elementAt(id string) (*SemanticElement, bool) {
	index, ok := p.elementIndex[strings.TrimSpace(id)]
	if !ok {
		return nil, false
	}
	return &p.out.Elements[index], true
}

// addElement projects one element and reports whether it was added (not
// rejected as invalid or duplicate).
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) addElement(spec elementSpec) bool {
	id := strings.TrimSpace(spec.id)
	if id == "" {
		p.report(semanticError("semantic.missing_id", spec.path, "semantic elements require a stable source ID"))
		return false
	}
	if previous, exists := p.elementIndex[id]; exists {
		p.report(semanticError("semantic.duplicate_identity", spec.path,
			fmt.Sprintf("duplicate semantic element ID %q (already projected at elements[%d])", id, previous)))
		return false
	}
	p.elementIndex[id] = len(p.out.Elements)
	p.out.Elements = append(p.out.Elements, p.newElement(id, spec))
	return true
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) newElement(id string, spec elementSpec) SemanticElement {
	var targets []string
	if spec.kind == ElementExtension && p.out.ID != "" && id != p.out.ID {
		targets = []string{p.out.ID}
	}
	return SemanticElement{
		ID:                 id,
		Name:               strings.TrimSpace(spec.name),
		Kind:               spec.kind,
		Metaclass:          officialMetaclassForElement(spec.kind, spec.extension),
		Namespace:          p.out.ID,
		Owner:              strings.TrimSpace(spec.owner),
		TypeRef:            strings.TrimSpace(spec.typeRef),
		References:         nil,
		Features:           compactFeatures(spec.features),
		Metadata:           []SemanticMetadata{semanticMetadata(spec.extension, id, spec.source)},
		ExtensionNamespace: semanticTypeNamespace(spec.extension),
		Extension:          spec.extension,
		Targets:            targets,
	}
}

// rejectDuplicateRelationship reports a duplicate relationship identity and
// says whether id was already projected.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) rejectDuplicateRelationship(id, path string) bool {
	previous, exists := p.relationshipIDs[id]
	if exists {
		p.report(semanticError("semantic.duplicate_identity", path,
			fmt.Sprintf("duplicate semantic relationship %q (already projected from %s)", id, previous)))
	}
	return exists
}

// addRelationship projects one deterministically identified relationship.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) addRelationship(spec relationshipSpec) {
	spec.source = strings.TrimSpace(spec.source)
	spec.target = strings.TrimSpace(spec.target)
	if spec.source == "" || spec.target == "" {
		p.report(semanticError("semantic.lossy_relationship", spec.path,
			fmt.Sprintf("relationship %q requires non-empty source and target", spec.sourceType)))
		return
	}
	aliasKey := semanticRelationshipAliasKey(spec.kind, spec.source, spec.target)
	if spec.sourceType == "contains" && p.mergeRelationshipAlias(aliasKey, spec) {
		return
	}
	id := semanticRelationshipID(spec)
	if p.rejectDuplicateRelationship(id, spec.path) {
		return
	}
	p.relationshipIDs[id] = spec.path
	p.relationshipAliases[aliasKey] = len(p.out.Relationships)
	p.out.Relationships = append(p.out.Relationships, SemanticRelationship{
		ID: id, Kind: spec.kind, Source: spec.source, Target: spec.target, Owner: strings.TrimSpace(spec.owner),
		Metaclass:  officialMetaclassForRelationship(spec.kind, spec.sourceType),
		SourceType: spec.sourceType,
		Metadata:   []SemanticMetadata{semanticMetadata("engineering.relationship."+spec.sourceType, id, spec.value)},
	})
}

// mergeRelationshipAlias records spec as an alias of an already projected
// relationship with the same kind and ends.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) mergeRelationshipAlias(aliasKey string, spec relationshipSpec) bool {
	existing, ok := p.relationshipAliases[aliasKey]
	if !ok {
		return false
	}
	relationship := &p.out.Relationships[existing]
	relationship.Metadata = append(relationship.Metadata,
		semanticMetadata("engineering.relationship_alias."+spec.sourceType, relationship.ID, spec.value))
	return true
}

// addTransition projects a transition relationship that keeps its authored ID.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) addTransition(path string, relationship SemanticRelationship, value any) {
	id := strings.TrimSpace(relationship.ID)
	if p.rejectDuplicateRelationship(id, path) {
		return
	}
	if previous, exists := p.elementIndex[id]; exists {
		p.report(semanticError("semantic.duplicate_identity", path,
			fmt.Sprintf("duplicate semantic identity %q (already projected at elements[%d])", id, previous)))
		return
	}
	p.relationshipIDs[id] = path
	relationship.Metaclass = officialMetaclassForRelationship(relationship.Kind, relationship.SourceType)
	relationship.Metadata = []SemanticMetadata{semanticMetadata("engineering.relationship."+relationship.SourceType, id, value)}
	p.out.Relationships = append(p.out.Relationships, relationship)
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) setExtensionTargets(id string, targets []string) {
	if element, ok := p.elementAt(id); ok {
		element.Targets = compactStrings(targets)
		element.References = compactStrings(append(element.References, targets...))
	}
}

// setReferences records the elements an already projected element refers to.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) setReferences(id string, references []string) {
	if element, ok := p.elementAt(id); ok {
		element.References = compactStrings(append(element.References, references...))
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) appendMetadata(id, metadataType string, source any) {
	if element, ok := p.elementAt(id); ok {
		element.Metadata = append(element.Metadata, semanticMetadata(metadataType, id, source))
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) addEvidence(spec evidenceSpec) {
	if strings.TrimSpace(spec.evidencePath) == "" && strings.TrimSpace(spec.description) == "" {
		return
	}
	id := derivedSemanticID("EVIDENCE", spec.target, spec.path, spec.evidencePath)
	source := map[string]string{"path": spec.evidencePath, "description": spec.description}
	p.addElement(elementSpec{
		path: spec.path, kind: ElementExtension, extension: spec.extension, id: id, name: spec.evidencePath, source: source,
	})
	p.setExtensionTargets(id, []string{spec.target})
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectRoot() {
	model := p.bundle.Architecture.Model
	p.addElement(elementSpec{path: "model", kind: ElementPackageDefinition, extension: "engineering.model", id: model.ID, name: model.Title, source: model})
	if root, ok := p.elementAt(p.out.ID); ok {
		root.Namespace = ""
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectAuthoredElements() {
	for i, element := range p.bundle.Architecture.Semantics.Elements {
		p.projectAuthoredElement(indexPath("semantics.elements", i), element)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectAuthoredElement(path string, element SemanticElement) {
	if strings.TrimSpace(element.Namespace) == "" {
		element.Namespace = p.out.ID
	}
	if strings.TrimSpace(element.Extension) == "" {
		element.Extension = "sysml." + string(element.Kind)
	}
	element.Metadata = append(element.Metadata, semanticMetadata("engineering.semantic_source", element.ID, element))
	p.addElement(elementSpec{
		path: path, kind: element.Kind, extension: element.Extension, id: element.ID, name: element.Name,
		owner: element.Owner, typeRef: element.TypeRef, source: element, features: element.Features,
	})
	if projected, ok := p.elementAt(element.ID); ok {
		p.applyAuthoredElement(projected, element)
	}
}

// applyAuthoredElement copies the canonical fields an authored semantic
// element declares onto its projection.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) applyAuthoredElement(projected *SemanticElement, element SemanticElement) {
	projected.Authored = true
	if strings.TrimSpace(element.Metaclass) != "" {
		projected.Metaclass = strings.TrimSpace(element.Metaclass)
	}
	projected.Properties = cloneMetamodelProperties(element.Properties)
	projected.Namespace = strings.TrimSpace(element.Namespace)
	projected.Multiplicity = element.Multiplicity
	projected.Ordered = element.Ordered
	projected.Unique = element.Unique
	projected.Conjugated = element.Conjugated
	projected.Specializes = compactStrings(element.Specializes)
	projected.Subsets = compactStrings(element.Subsets)
	projected.Redefines = compactStrings(element.Redefines)
	projected.ControlKind = strings.TrimSpace(element.ControlKind)
	projected.OccurrenceID = strings.TrimSpace(element.OccurrenceID)
	projected.PortionOf = strings.TrimSpace(element.PortionOf)
	projected.Variation = element.Variation
	projected.Variants = compactStrings(element.Variants)
	projected.References = compactStrings(append(projected.References, element.References...))
	projected.ExtensionNamespace = strings.TrimSpace(element.ExtensionNamespace)
	projected.Targets = compactStrings(element.Targets)
	if element.Kind == ElementExtension {
		p.completeExtensionTargets(projected, element)
	}
	projected.Metadata = normalizeMetadata(element.Metadata, element.ID)
}

// completeExtensionTargets defaults the namespace and the targets of an
// authored extension element.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) completeExtensionTargets(projected *SemanticElement, element SemanticElement) {
	if projected.ExtensionNamespace == "" {
		projected.ExtensionNamespace = semanticTypeNamespace(element.Extension)
	}
	if len(projected.Targets) > 0 {
		return
	}
	projected.Targets = compactStrings(projected.References)
	if len(projected.Targets) == 0 && element.ID != p.out.ID {
		projected.Targets = []string{p.out.ID}
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectAuthoredRelationships() {
	for i, relationship := range p.bundle.Architecture.Semantics.Relationships {
		p.projectAuthoredRelationship(indexPath("semantics.relationships", i), relationship)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectAuthoredRelationship(path string, relationship SemanticRelationship) {
	relationship.Source = strings.TrimSpace(relationship.Source)
	relationship.Target = strings.TrimSpace(relationship.Target)
	if relationship.Source == "" || relationship.Target == "" {
		p.report(semanticError("semantic.lossy_relationship", path,
			fmt.Sprintf("relationship %q requires non-empty source and target", relationship.Kind)))
		return
	}
	if strings.TrimSpace(relationship.ID) == "" {
		relationship.ID = semanticRelationshipID(relationshipSpec{
			sourceType: string(relationship.Kind), source: relationship.Source, target: relationship.Target, owner: relationship.Owner,
		})
	}
	if p.rejectDuplicateRelationship(relationship.ID, path) {
		return
	}
	p.relationshipIDs[relationship.ID] = path
	p.relationshipAliases[semanticRelationshipAliasKey(relationship.Kind, relationship.Source, relationship.Target)] = len(p.out.Relationships)
	p.out.Relationships = append(p.out.Relationships, completeAuthoredRelationship(relationship))
}

// completeAuthoredRelationship defaults the source type and metaclass of an
// authored relationship and normalizes its properties and metadata.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func completeAuthoredRelationship(relationship SemanticRelationship) SemanticRelationship {
	if strings.TrimSpace(relationship.SourceType) == "" {
		relationship.SourceType = string(relationship.Kind)
	}
	if strings.TrimSpace(relationship.Metaclass) == "" {
		relationship.Metaclass = officialMetaclassForRelationship(relationship.Kind, relationship.SourceType)
	}
	relationship.Properties = cloneMetamodelProperties(relationship.Properties)
	relationship.Metadata = normalizeMetadata(relationship.Metadata, relationship.ID)
	return relationship
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectOrganization() {
	for i, x := range p.a.FunctionalGroups {
		p.addElement(elementSpec{path: indexPath("authoredArchitecture.functionalGroups", i), kind: ElementPartDefinition, extension: "engineering.functional_group", id: x.ID, name: x.Name, source: x})
	}
	for i, x := range p.a.FunctionalUnits {
		p.projectFunctionalUnit(indexPath("authoredArchitecture.functionalUnits", i), x)
	}
	for i, x := range p.a.Actors {
		p.addElement(elementSpec{path: indexPath("authoredArchitecture.actors", i), kind: ElementStakeholderDefinition, extension: "engineering.actor", id: x.ID, name: x.Name, source: x})
	}
	for i, x := range p.a.AttackVectors {
		p.addElement(elementSpec{path: indexPath("authoredArchitecture.attackVectors", i), kind: ElementExtension, extension: "engineering.attack_vector", id: x.ID, name: x.Name, source: x})
	}
	for i, x := range p.a.ReferencedElements {
		p.addElement(elementSpec{path: indexPath("authoredArchitecture.referencedElements", i), kind: ElementExtension, extension: "engineering.referenced_element", id: x.ID, name: x.Name, source: x})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectFunctionalUnit(path string, x FunctionalUnit) {
	p.addElement(elementSpec{path: path, kind: ElementPartUsage, extension: "engineering.functional_unit", id: x.ID, name: x.Name, owner: x.Group, source: x})
	if strings.TrimSpace(x.Group) != "" {
		p.addRelationship(relationshipSpec{path: path + ".group", kind: RelationshipContainment, sourceType: "contains", source: x.Group, target: x.ID, owner: x.Group, value: x})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectTechnicalItems() {
	for i, x := range p.a.Interfaces {
		p.addElement(elementSpec{
			path: indexPath("authoredArchitecture.interfaces", i), kind: ElementInterfaceDefinition, extension: "engineering.interface",
			id: x.ID, name: x.Name, owner: x.Owner, source: x,
			features: []SemanticFeature{attributeFeature("protocol", x.Protocol), attributeFeature("endpoint", x.Endpoint), attributeFeature("schemaRef", x.SchemaRef)},
		})
	}
	for i, x := range p.a.DataObjects {
		p.addElement(elementSpec{
			path: indexPath("authoredArchitecture.dataObjects", i), kind: ElementItemDefinition, extension: "engineering.data_object",
			id: x.ID, name: x.Name, source: x,
			features: []SemanticFeature{attributeFeature("schemaRef", x.SchemaRef), attributeFeature("classification", x.Classification)},
		})
	}
	for i, x := range p.a.DeploymentTargets {
		p.addElement(elementSpec{
			path: indexPath("authoredArchitecture.deploymentTargets", i), kind: ElementPartDefinition, extension: "engineering.deployment_target",
			id: x.ID, name: x.Name, source: x,
			features: []SemanticFeature{
				attributeFeature("environment", x.Environment), attributeFeature("region", x.Region),
				attributeFeature("namespace", x.Namespace), attributeFeature("trustZone", x.TrustZone),
			},
		})
	}
	for i, x := range p.a.HardwareItems {
		p.projectHardwareItem(indexPath("authoredArchitecture.hardwareItems", i), x)
	}
	for i, x := range p.a.HardwareInterfaces {
		p.projectHardwareInterface(indexPath("authoredArchitecture.hardwareInterfaces", i), x)
	}
	for i, x := range p.a.Controls {
		p.addElement(elementSpec{path: indexPath("authoredArchitecture.controls", i), kind: ElementExtension, extension: "engineering.control", id: x.ID, name: x.Name, source: x})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectHardwareItem(path string, x HardwareItem) {
	p.addElement(elementSpec{
		path: path, kind: ElementPartDefinition, extension: "engineering.hardware_item", id: x.ID, name: x.Name, source: x,
		features: []SemanticFeature{
			attributeFeature("kind", x.Kind), attributeFeature("partNumber", x.PartNumber),
			attributeFeature("supplier", x.Supplier), attributeFeature("safetyLevel", x.SafetyLevel),
		},
	})
	for _, hosted := range x.Hosts {
		p.addRelationship(relationshipSpec{path: path + ".hosts", kind: RelationshipAllocation, sourceType: "hosts", source: hosted, target: x.ID, owner: x.ID, value: x})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectHardwareInterface(path string, x HardwareInterface) {
	p.addElement(elementSpec{
		path: path, kind: ElementConnectionDefinition, extension: "engineering.hardware_interface", id: x.ID, name: x.Name, source: x,
		features: []SemanticFeature{attributeFeature("busType", x.BusType), attributeFeature("direction", x.Direction)},
	})
	p.addRelationship(relationshipSpec{path: path, kind: RelationshipConnection, sourceType: "hardware_interface", source: x.From, target: x.To, owner: x.ID, value: x})
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectAssurance() {
	for i, x := range p.a.Risks {
		p.projectRisk(indexPath("authoredArchitecture.risks", i), x)
	}
	for i, x := range p.a.POAMItems {
		p.projectPOAMItem(indexPath("authoredArchitecture.poamItems", i), x)
	}
	for i, x := range p.a.TrustBoundaries {
		p.projectTrustBoundary(indexPath("authoredArchitecture.trustBoundaries", i), x)
	}
	for i, x := range p.a.Events {
		p.addElement(elementSpec{path: indexPath("authoredArchitecture.events", i), kind: ElementEventDefinition, extension: "engineering.event", id: x.ID, name: x.Name, source: x})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectRisk(path string, x Risk) {
	p.addElement(elementSpec{path: path, kind: ElementExtension, extension: "engineering.risk", id: x.ID, name: x.Title, owner: x.Owner, source: x})
	for j, evidence := range x.Evidence {
		p.addEvidence(evidenceSpec{path: indexPath(path+".evidence", j), extension: "engineering.verification_evidence", target: x.ID, evidencePath: evidence.Path, description: evidence.Description})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectPOAMItem(path string, x POAMItem) {
	p.addElement(elementSpec{path: path, kind: ElementExtension, extension: "engineering.poam_item", id: x.ID, name: x.Milestone, owner: x.ResponsibleRole, source: x})
	for j, artifact := range x.Artifacts {
		p.addEvidence(evidenceSpec{path: indexPath(path+".artifacts", j), extension: "engineering.poam_artifact", target: x.ID, evidencePath: artifact.Path, description: artifact.Description})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectTrustBoundary(path string, x TrustBoundary) {
	p.addElement(elementSpec{path: path, kind: ElementExtension, extension: "engineering.trust_boundary", id: x.ID, name: x.Name, owner: x.ParentRef, source: x})
	for _, member := range x.Members {
		p.addRelationship(relationshipSpec{path: path + ".members", kind: RelationshipContainment, sourceType: "boundary_contains", source: x.ID, target: member, owner: x.ID, value: x})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectFlows() {
	for i, x := range p.a.Flows {
		p.projectFlow(indexPath("authoredArchitecture.flows", i), x)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectFlow(path string, x Flow) {
	features := make([]SemanticFeature, 0, len(x.DataRefs))
	for _, ref := range x.DataRefs {
		features = append(features, referenceFeature(flowItemFeature, ref))
	}
	p.addElement(elementSpec{path: path, kind: ElementActionDefinition, extension: "engineering.flow", id: x.ID, name: x.Title, source: x, features: features})
	if x.SourceRef != "" && x.DestinationRef != "" {
		p.addRelationship(relationshipSpec{path: path, kind: RelationshipTransfer, sourceType: "flow_transfer", source: x.SourceRef, target: x.DestinationRef, owner: x.ID, value: x})
	}
	for j, step := range x.Steps {
		p.projectFlowStep(indexPath(path+".steps", j), x.ID, step)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectFlowStep(stepPath, flowID string, step FlowStep) {
	p.addElement(elementSpec{
		path: stepPath, kind: ElementActionUsage, extension: "engineering.flow_step", id: step.ID, name: step.Action,
		owner: flowID, typeRef: step.Ref, source: step, features: flowStepFeatures(step),
	})
	if step.Ref != "" {
		p.addRelationship(relationshipSpec{path: stepPath + ".ref", kind: RelationshipDependency, sourceType: "flow_ref", source: step.ID, target: step.Ref, owner: flowID, value: step})
	}
	if step.SourceRef != "" && step.DestinationRef != "" {
		p.addRelationship(relationshipSpec{path: stepPath + ".transfer", kind: RelationshipTransfer, sourceType: "step_transfer", source: step.SourceRef, target: step.DestinationRef, owner: step.ID, value: step})
	}
	for _, next := range step.Next {
		p.addRelationship(relationshipSpec{path: stepPath + ".next", kind: RelationshipSuccession, sourceType: "flow_next", source: step.ID, target: next, owner: flowID, value: step})
	}
	for _, next := range step.OnError {
		p.addRelationship(relationshipSpec{path: stepPath + ".onError", kind: RelationshipSuccession, sourceType: "flow_error", source: step.ID, target: next, owner: flowID, value: step})
	}
}

// flowStepFeatures returns the input, output and item parameters of a flow step.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func flowStepFeatures(step FlowStep) []SemanticFeature {
	features := make([]SemanticFeature, 0, len(step.DataIn)+len(step.DataOut)+len(step.DataRefs))
	for _, ref := range step.DataIn {
		features = append(features, referenceFeature(flowInputFeature, ref))
	}
	for _, ref := range step.DataOut {
		features = append(features, referenceFeature(flowOutputFeature, ref))
	}
	for _, ref := range step.DataRefs {
		features = append(features, referenceFeature(flowItemFeature, ref))
	}
	return features
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectMachines() {
	for i, x := range p.a.Machines {
		projectMachine(p, indexPath("authoredArchitecture.machines", i), x)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectThreats() {
	for i, x := range p.a.ThreatScenarios {
		p.projectThreatScenario(indexPath("authoredArchitecture.threatScenarios", i), x)
	}
	for i, x := range p.a.ThreatAssumptions {
		p.projectThreatAssumption(indexPath("authoredArchitecture.threatAssumptions", i), x)
	}
	for i, x := range p.a.ThreatOutOfScope {
		p.projectThreatOutOfScope(indexPath("authoredArchitecture.threatOutOfScope", i), x)
	}
	for i, x := range p.a.ThreatMitigations {
		p.projectThreatMitigation(indexPath("authoredArchitecture.threatMitigations", i), x)
	}
	for i, x := range p.a.ControlVerifications {
		p.projectControlVerification(indexPath("authoredArchitecture.controlVerifications", i), x)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectThreatScenario(path string, x ThreatScenario) {
	p.addElement(elementSpec{path: path, kind: ElementExtension, extension: "engineering.threat_scenario", id: x.ID, name: x.Title, owner: x.Owner, source: x})
	for j, evidence := range x.Evidence {
		p.addEvidence(evidenceSpec{path: indexPath(path+".evidence", j), extension: "engineering.verification_evidence", target: x.ID, evidencePath: evidence.Path, description: evidence.Description})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectThreatAssumption(path string, x ThreatAssumption) {
	p.addElement(elementSpec{path: path, kind: ElementExtension, extension: "engineering.threat_assumption", id: x.ID, name: x.Title, owner: x.Owner, source: x})
	for j, evidence := range x.Evidence {
		p.addEvidence(evidenceSpec{path: indexPath(path+".evidence", j), extension: "engineering.verification_evidence", target: x.ID, evidencePath: evidence.Path, description: evidence.Description})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectThreatOutOfScope(path string, x ThreatOutOfScope) {
	p.addElement(elementSpec{path: path, kind: ElementExtension, extension: "engineering.threat_out_of_scope", id: x.ID, name: x.Title, owner: x.Owner, source: x})
	for j, evidence := range x.Evidence {
		p.addEvidence(evidenceSpec{path: indexPath(path+".evidence", j), extension: "engineering.verification_evidence", target: x.ID, evidencePath: evidence.Path, description: evidence.Description})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectThreatMitigation(path string, x ThreatMitigation) {
	p.addElement(elementSpec{path: path, kind: ElementExtension, extension: "engineering.threat_mitigation", id: x.ID, name: x.ID, owner: x.Owner, source: x})
	for j, evidence := range x.Evidence {
		p.addEvidence(evidenceSpec{path: indexPath(path+".evidence", j), extension: "engineering.verification_evidence", target: x.ID, evidencePath: evidence.Path, description: evidence.Description})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectControlVerification(path string, x ControlVerification) {
	p.addElement(elementSpec{path: path, kind: ElementVerificationCaseUsage, extension: "engineering.control_verification", id: x.ID, name: x.ID, owner: x.Owner, source: x})
	if x.ControlRef != "" {
		p.addRelationship(relationshipSpec{path: path + ".controlRef", kind: RelationshipVerification, sourceType: "verifies", source: x.ID, target: x.ControlRef, owner: x.ID, value: x})
	}
	for j, evidence := range x.Evidence {
		p.addEvidence(evidenceSpec{path: indexPath(path+".evidence", j), extension: "engineering.verification_evidence", target: x.ID, evidencePath: evidence.Path, description: evidence.Description})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectMappings() {
	for i, x := range p.a.Mappings {
		// Mappings to inferred CODE identifiers are code trace links, not architecture
		// relationships; validation resolves them against the code index instead.
		if IsCodeID(x.From) || IsCodeID(x.To) {
			continue
		}
		p.projectMapping(indexPath("authoredArchitecture.mappings", i), x)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectMapping(path string, x Mapping) {
	kind, ok := semanticMappingKind(x.Type)
	if !ok {
		kind = RelationshipExtension
		p.report(semanticError("semantic.unsupported_relationship", path,
			fmt.Sprintf("mapping type %q has no canonical relationship mapping", x.Type)))
	}
	p.addRelationship(relationshipSpec{path: path, kind: kind, sourceType: strings.TrimSpace(x.Type), source: x.From, target: x.To, value: x})
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectCompliance() {
	for i, x := range p.bundle.Architecture.Compliance.Profiles {
		p.addElement(elementSpec{path: indexPath("compliance.profiles", i), kind: ElementExtension, extension: "engineering.compliance_profile", id: x.ID, name: x.ID, source: x})
	}
	for i, x := range p.bundle.Architecture.Compliance.Mappings {
		p.projectComplianceMapping(indexPath("compliance.mappings", i), x)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectComplianceMapping(path string, x ComplianceMapping) {
	p.addElement(elementSpec{path: path, kind: ElementExtension, extension: "engineering.compliance_mapping", id: x.ID, name: x.ID, source: x})
	for j, evidence := range x.Evidence {
		p.addEvidence(evidenceSpec{path: indexPath(path+".evidence", j), extension: "engineering.compliance_evidence", target: x.ID, evidencePath: evidence.Path, description: evidence.Description})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectContractAndComposition() {
	architecture := p.bundle.Architecture
	for i, x := range architecture.Contract.Provides {
		p.addElement(elementSpec{
			path: indexPath("contract.provides", i), kind: ElementPortUsage, extension: "engineering.contract_provides",
			id: x.ID, name: x.ID, owner: p.out.ID, typeRef: "EngineeringPort", source: x,
			features: []SemanticFeature{referenceFeature(providedPortFeature, "EngineeringPort")},
		})
	}
	for i, x := range architecture.Contract.Requires {
		p.projectRequiredPort(indexPath("contract.requires", i), x)
	}
	for i, x := range architecture.Composition.Subsystems {
		p.addElement(elementSpec{path: indexPath("composition.subsystems", i), kind: ElementPartUsage, extension: "engineering.subsystem", id: x.ID, name: x.Name, owner: p.out.ID, source: x})
	}
	for i, x := range architecture.Composition.Satisfactions {
		p.addRelationship(relationshipSpec{path: indexPath("composition.satisfactions", i), kind: RelationshipSatisfaction, sourceType: "satisfaction", source: x.By, target: x.Need, owner: p.out.ID, value: x})
	}
	for i, x := range architecture.Views {
		p.addElement(elementSpec{path: indexPath("views", i), kind: ElementViewUsage, extension: "engineering.view", id: x.ID, name: x.ID, owner: p.out.ID, source: x})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectRequiredPort(path string, x ContractEntry) {
	p.addElement(elementSpec{
		path: path, kind: ElementPortUsage, extension: "engineering.contract_requires",
		id: x.ID, name: x.ID, owner: p.out.ID, typeRef: "EngineeringPort", source: x,
		features: []SemanticFeature{referenceFeature(requiredPortFeature, "EngineeringPort")},
	})
	if element, ok := p.elementAt(x.ID); ok {
		element.Conjugated = true
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectDocuments() {
	for i, document := range p.bundle.Views.Documents {
		p.projectDocument(indexPath("documents", i), document)
	}
	if p.bundle.Architecture.NAF.Enabled() {
		p.projectNAF()
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectDocument(path string, document DocumentDefinition) {
	p.addElement(elementSpec{path: path, kind: ElementViewUsage, extension: "engineering.document_definition", id: document.ID, name: document.Title, owner: p.out.ID, source: document})
	refGroups := []struct {
		field      string
		sourceType string
		refs       []string
	}{
		{".contentRefs", "document_content", document.ContentRefs},
		{".stakeholderRefs", "document_stakeholder", document.StakeholderRefs},
		{".referenceRefs", "document_reference", document.ReferenceRefs},
	}
	for _, group := range refGroups {
		for _, ref := range group.refs {
			p.addRelationship(relationshipSpec{path: path + group.field, kind: RelationshipDependency, sourceType: group.sourceType, source: document.ID, target: ref, owner: document.ID, value: document})
		}
	}
	p.projectDocumentSections(path, document)
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectDocumentSections(path string, document DocumentDefinition) {
	for j, section := range document.Sections {
		p.appendMetadata(document.ID, "engineering.document_section."+section.ID, section)
		for _, ref := range section.IncludeRefs {
			p.addRelationship(relationshipSpec{
				path: indexPath(path+".sections", j) + ".includeRefs", kind: RelationshipDependency, sourceType: "document_section_content",
				source: document.ID, target: ref, owner: document.ID, value: section,
			})
		}
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectNAF() {
	naf := p.bundle.Architecture.NAF
	p.appendMetadata(p.out.ID, "naf.profile.v4_1", naf)
	for _, product := range naf.Products {
		p.appendMetadata(product.ViewRef, "naf.viewpoint."+product.Viewpoint, product)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectRequirementsAndDesign() {
	for i, x := range p.bundle.Architecture.Decisions {
		p.addElement(elementSpec{path: indexPath("decisions", i), kind: ElementExtension, extension: "engineering.architecture_decision", id: x.ID, name: x.Title, source: x})
	}
	for i, x := range p.bundle.Requirements.Requirements {
		p.addElement(elementSpec{
			path: indexPath("requirements", i), kind: ElementRequirementDefinition, extension: "engineering.requirement_source",
			id: x.ID, name: x.ID, owner: p.out.ID, source: x,
			features: []SemanticFeature{attributeFeature("text", x.Text), attributeFeature("notes", x.Notes)},
		})
	}
	for i, x := range p.bundle.Requirements.Expected {
		p.addElement(elementSpec{path: indexPath("expected", i), kind: ElementExtension, extension: "engineering.requirement_pattern", id: x.ID, name: x.ID, owner: p.out.ID, source: x})
	}
	if !reflect.DeepEqual(p.bundle.Requirements.LintRun, LintRun{}) {
		p.appendMetadata(p.out.ID, "engineering.requirements_lint_policy", p.bundle.Requirements.LintRun)
	}
	p.projectDesign()
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectDesign() {
	design := p.bundle.Design.Design
	if !reflect.DeepEqual(design, DesignModel{}) {
		p.appendMetadata(p.out.ID, "engineering.design_model", design)
	}
	for _, group := range design.FunctionalGroups {
		p.appendMetadata(group.ID, "engineering.design_functional_group", group)
		p.addNarratives("design.functionalGroups."+group.ID+".views.", group.ID, group.Views)
	}
	for _, unit := range design.FunctionalUnits {
		p.appendMetadata(unit.ID, "engineering.design_functional_unit", unit)
		p.addNarratives("design.functionalUnits."+unit.ID+".views.", unit.ID, unit.Views)
	}
	if !reflect.DeepEqual(p.bundle.Architecture.InferenceHints, InferenceHints{}) {
		id := derivedSemanticID("EXTENSION", p.out.ID, "code-ownership-policy")
		p.addElement(elementSpec{path: "inferenceHints", kind: ElementExtension, extension: "engineering.code_ownership_policy", id: id, name: "Code ownership policy", source: p.bundle.Architecture.InferenceHints})
		p.setExtensionTargets(id, []string{p.out.ID})
	}
}

// addNarratives projects the design narratives of one functional group or unit.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) addNarratives(pathPrefix, ownerID string, views map[string]DesignView) {
	for key, narrative := range views {
		p.addElement(elementSpec{
			path: pathPrefix + key, kind: ElementConcernUsage, extension: "engineering.design_narrative",
			id: derivedSemanticID("CONCERN", ownerID, key), name: narrative.Title, owner: ownerID, source: narrative,
			features: []SemanticFeature{attributeFeature("narrative", narrative.Narrative)},
		})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) projectCatalog() {
	c := p.bundle.Catalog.Catalog
	groups := []struct {
		name    string
		entries []CatalogEntry
	}{
		{"systems", c.Systems}, {"functional_groups", c.FunctionalGroups}, {"functional_units", c.FunctionalUnits},
		{"referenced_elements", c.ReferencedElements}, {"actors", c.Actors}, {"attack_vectors", c.AttackVectors},
		{"events", c.Events}, {"states", c.States}, {"features", c.Features}, {"modes", c.Modes},
		{"conditions", c.Conditions}, {"data_terms", c.DataTerms},
	}
	for _, group := range groups {
		for i, entry := range group.entries {
			p.mergeCatalogEntry(indexPath("catalog."+group.name, i), group.name, entry)
		}
	}
}

// mergeCatalogEntry adds a catalog entry as an extension element, or attaches
// it as metadata to the element that already has its ID.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) mergeCatalogEntry(path, group string, entry CatalogEntry) {
	id := strings.TrimSpace(entry.ID)
	if previous, exists := p.catalogIDs[id]; exists {
		p.report(semanticError("semantic.duplicate_identity", path,
			fmt.Sprintf("duplicate catalog ID %q (already declared in %s)", id, previous)))
		return
	}
	p.catalogIDs[id] = group
	if index, exists := p.elementIndex[id]; exists {
		element := &p.out.Elements[index]
		element.Metadata = append(element.Metadata, semanticMetadata("engineering.catalog."+group, id, entry))
		return
	}
	p.addElement(elementSpec{path: path, kind: ElementExtension, extension: "engineering.catalog." + group, id: entry.ID, name: entry.Name, source: entry})
}

// finish normalizes the metamodel properties and sorts the projection.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func (p *semanticProjector) finish() {
	for index := range p.out.Elements {
		normalizeElementMetamodelProperties(&p.out.Elements[index])
	}
	for index := range p.out.Relationships {
		normalizeRelationshipMetamodelProperties(&p.out.Relationships[index])
	}
	elements, relationships, imports := p.out.Elements, p.out.Relationships, p.out.Imports
	sort.Slice(elements, func(i, j int) bool { return elements[i].ID < elements[j].ID })
	sort.Slice(relationships, func(i, j int) bool { return relationships[i].ID < relationships[j].ID })
	sort.Slice(imports, func(i, j int) bool { return semanticImportLess(imports[i], imports[j]) })
	sortSemanticDiagnostics(p.diagnostics)
}

// TRLC-LINKS: REQ-EMG-035
func semanticImportLess(a, b SemanticImport) bool {
	if a.Namespace != b.Namespace {
		return a.Namespace < b.Namespace
	}
	return a.Imported < b.Imported
}

// semanticValidator accumulates the diagnostics of one validation run.
type semanticValidator struct {
	semantic    SemanticModel
	diagnostics []SemanticDiagnostic
	identities  map[string]string
	elements    map[string]SemanticElement
	occurrences map[string]string
}

// ValidateSemanticModel checks invariants of the canonical projection independently
// of the YAML structures from which it was derived.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func ValidateSemanticModel(semantic SemanticModel) []SemanticDiagnostic {
	v := &semanticValidator{
		semantic:    semantic,
		identities:  map[string]string{},
		elements:    map[string]SemanticElement{},
		occurrences: map[string]string{},
	}
	v.checkDeclarations()
	v.checkImports()
	v.checkElements()
	v.checkRelationships()
	v.checkMetadata()
	sortSemanticDiagnostics(v.diagnostics)
	return v.diagnostics
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) fail(code, path, message string) {
	v.diagnostics = append(v.diagnostics, semanticError(code, path, message))
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) addIdentity(id, path string) {
	id = strings.TrimSpace(id)
	if id == "" {
		v.fail("semantic.missing_identity", path, "canonical semantic identities must not be empty")
		return
	}
	if previous, exists := v.identities[id]; exists {
		v.fail("semantic.duplicate_identity", path,
			fmt.Sprintf("duplicate canonical semantic identity %q (already declared at %s)", id, previous))
		return
	}
	v.identities[id] = path
}

// unresolved reports whether ref names neither a projected element nor a
// qualified reference into a dependency.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) unresolved(ref string) bool {
	if strings.Contains(ref, "::") {
		return false
	}
	_, exists := v.elements[ref]
	return !exists
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkDeclarations() {
	for i, element := range v.semantic.Elements {
		v.checkElementDeclaration(indexPath("semantic.elements", i), element)
	}
	for i, relationship := range v.semantic.Relationships {
		path := indexPath("semantic.relationships", i)
		v.addIdentity(relationship.ID, path)
		v.diagnostics = append(v.diagnostics, validateMetamodelInstance(metamodelInstance{
			id: relationship.ID, metaclass: relationship.Metaclass, properties: relationship.Properties,
			path: path, extension: relationship.Kind == RelationshipExtension,
		})...)
		if !validSemanticRelationshipKind(relationship.Kind) {
			v.fail("semantic.unsupported_relationship", path+".kind",
				fmt.Sprintf("semantic relationship %q has unsupported kind %q", relationship.ID, relationship.Kind))
		}
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkElementDeclaration(path string, element SemanticElement) {
	v.addIdentity(element.ID, path)
	v.diagnostics = append(v.diagnostics, validateMetamodelInstance(metamodelInstance{
		id: element.ID, metaclass: element.Metaclass, properties: element.Properties,
		path: path, extension: element.Kind == ElementExtension,
	})...)
	if strings.TrimSpace(element.ID) == "" {
		return
	}
	if _, exists := v.elements[element.ID]; !exists {
		v.elements[element.ID] = element
	}
	v.checkElementKind(path, element)
	v.checkElementRoles(path, element)
	v.checkOccurrence(path, element)
	v.checkExtensionIdentity(path, element)
	v.checkMultiplicity(element.Multiplicity, path+".multiplicity")
	for j, feature := range element.Features {
		v.checkFeature(indexPath(path+".features", j), feature)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkElementKind(path string, element SemanticElement) {
	if !validSemanticElementKind(element.Kind) {
		v.fail("semantic.unsupported_element_kind", path+".kind",
			fmt.Sprintf("semantic element %q has unsupported kind %q", element.ID, element.Kind))
	}
	if element.Kind == ElementControlNode && !validControlKind(element.ControlKind) {
		v.fail("semantic.invalid_control_node", path+".controlKind",
			fmt.Sprintf("control node %q has unsupported kind %q", element.ID, element.ControlKind))
	}
}

// checkElementRoles checks the conjugation and time portion an element declares.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkElementRoles(path string, element SemanticElement) {
	if element.Conjugated && !isConjugableKind(element.Kind) {
		v.fail("semantic.invalid_conjugation", path+".conjugated",
			fmt.Sprintf("semantic element %q can only be conjugated when it is a port or interface usage", element.ID))
	}
	if strings.TrimSpace(element.PortionOf) != "" && !isTimePortionKind(element.Kind) {
		v.fail("semantic.invalid_time_portion", path+".portionOf",
			fmt.Sprintf("semantic element %q can only use portionOf as a snapshot or time slice", element.ID))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkOccurrence(path string, element SemanticElement) {
	occurrenceID := strings.TrimSpace(element.OccurrenceID)
	if occurrenceID == "" {
		return
	}
	if previous, exists := v.occurrences[occurrenceID]; exists {
		v.fail("semantic.duplicate_occurrence_identity", path+".occurrenceId",
			fmt.Sprintf("occurrence identity %q is already used at %s", occurrenceID, previous))
	} else {
		v.occurrences[occurrenceID] = path + ".occurrenceId"
	}
	if !isOccurrenceKind(element.Kind) {
		v.fail("semantic.invalid_occurrence_identity", path+".occurrenceId",
			fmt.Sprintf("semantic element %q cannot declare occurrence identity", element.ID))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkExtensionIdentity(path string, element SemanticElement) {
	if element.Kind != ElementExtension {
		return
	}
	if strings.TrimSpace(element.ExtensionNamespace) == "" || strings.TrimSpace(element.Extension) == "" {
		v.fail("semantic.invalid_extension_type", path,
			fmt.Sprintf("engineering extension %q requires namespace and type identity", element.ID))
	}
	if len(element.Targets) == 0 && element.Authored {
		v.fail("semantic.invalid_extension_target", path+".targets",
			fmt.Sprintf("engineering extension %q requires at least one explicit target", element.ID))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkMultiplicity(multiplicity *Multiplicity, path string) {
	if multiplicity == nil {
		return
	}
	if multiplicity.Lower != nil && *multiplicity.Lower < 0 {
		v.fail("semantic.invalid_multiplicity", path+".lower", "multiplicity lower bound must not be negative")
	}
	if multiplicity.Upper != nil && *multiplicity.Upper < -1 {
		v.fail("semantic.invalid_multiplicity", path+".upper", "multiplicity upper bound must be -1 (unbounded) or non-negative")
	}
	if multiplicityInverted(multiplicity) {
		v.fail("semantic.invalid_multiplicity", path, "multiplicity lower bound must not exceed upper bound")
	}
}

// multiplicityInverted reports a bounded multiplicity whose lower bound exceeds its upper bound.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
func multiplicityInverted(multiplicity *Multiplicity) bool {
	if multiplicity.Lower == nil || multiplicity.Upper == nil {
		return false
	}
	return *multiplicity.Upper >= 0 && *multiplicity.Lower > *multiplicity.Upper
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkFeature(path string, feature SemanticFeature) {
	if strings.TrimSpace(feature.Name) == "" {
		v.fail("semantic.invalid_feature", path+".name", "semantic feature name must not be empty")
	}
	if !validSemanticFeatureKind(feature.Kind) {
		v.fail("semantic.invalid_feature", path+".kind",
			fmt.Sprintf("semantic feature %q has unsupported kind %q", feature.Name, feature.Kind))
	}
	if feature.Kind == FeatureParameter && !validParameterDirection(feature.Direction) {
		v.fail("semantic.invalid_parameter_direction", path+".direction",
			fmt.Sprintf("parameter %q has unsupported direction %q", feature.Name, feature.Direction))
	}
	v.checkFeatureShape(path, feature)
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkFeatureShape(path string, feature SemanticFeature) {
	if feature.Conjugated && feature.Kind != FeaturePort {
		v.fail("semantic.invalid_conjugation", path+".conjugated",
			fmt.Sprintf("semantic feature %q can only be conjugated when it is a port", feature.Name))
	}
	v.checkMultiplicity(feature.Multiplicity, path+".multiplicity")
	if featureValueContradictsMultiplicity(feature) {
		v.fail("semantic.value_multiplicity_mismatch", path+".value",
			fmt.Sprintf("semantic feature %q cannot have a value when multiplicity upper bound is zero", feature.Name))
	}
}

// featureValueContradictsMultiplicity reports a feature that has a value
// although its multiplicity upper bound is zero.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
func featureValueContradictsMultiplicity(feature SemanticFeature) bool {
	if feature.Value == nil || feature.Multiplicity == nil {
		return false
	}
	return feature.Multiplicity.Upper != nil && *feature.Multiplicity.Upper == 0
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkImports() {
	for i, imported := range v.semantic.Imports {
		v.checkImport(indexPath("semantic.imports", i), imported)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkImport(path string, imported SemanticImport) {
	if strings.TrimSpace(imported.Imported) == "" {
		v.fail("semantic.invalid_import", path+".imported", "semantic import requires an imported namespace")
	}
	if visibility := strings.TrimSpace(imported.Visibility); invalidImportVisibility(visibility) {
		v.fail("semantic.invalid_import", path+".visibility",
			fmt.Sprintf("semantic import visibility %q must be private or public", visibility))
	}
	if namespace := strings.TrimSpace(imported.Namespace); namespace != "" {
		v.checkImportNamespace(path+".namespace", namespace)
	}
}

// invalidImportVisibility reports a non-empty visibility other than private or public.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
func invalidImportVisibility(visibility string) bool {
	return visibility != "" && visibility != "private" && visibility != "public"
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkImportNamespace(path, namespace string) {
	if element, exists := v.elements[namespace]; !exists || element.Kind != ElementPackageDefinition {
		v.fail("semantic.invalid_import", path,
			fmt.Sprintf("semantic import namespace %q must resolve to a package definition", namespace))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkElements() {
	for i, element := range v.semantic.Elements {
		v.checkElementReferences(indexPath("semantic.elements", i), element)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkElementReferences(path string, element SemanticElement) {
	if !v.checkNamespace(path, element) {
		return
	}
	v.checkOwner(path, element)
	for j, reference := range element.References {
		if _, exists := v.elements[strings.TrimSpace(reference)]; !exists {
			v.fail("semantic.dangling_reference", indexPath(path+".references", j),
				fmt.Sprintf("semantic element %q reference %q does not exist", element.ID, reference))
		}
	}
	for j := range element.Features {
		v.validateExpression(element.Features[j].Value, indexPath(path+".features", j)+".value")
	}
	v.checkPortion(path, element)
	v.checkVariants(path, element)
}

// checkNamespace validates the namespace of an element and reports whether
// the remaining reference checks still apply to it.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkNamespace(path string, element SemanticElement) bool {
	namespace := strings.TrimSpace(element.Namespace)
	if namespace == "" {
		return true
	}
	if namespace == strings.TrimSpace(element.ID) {
		v.fail("semantic.invalid_ownership", path+".namespace",
			fmt.Sprintf("semantic element %q cannot be its own namespace", element.ID))
		return false
	}
	if namespaceElement, exists := v.elements[namespace]; !exists || namespaceElement.Kind != ElementPackageDefinition {
		v.fail("semantic.invalid_ownership", path+".namespace",
			fmt.Sprintf("semantic element %q namespace %q must resolve to a package definition", element.ID, namespace))
	}
	return true
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkOwner(path string, element SemanticElement) {
	owner := strings.TrimSpace(element.Owner)
	if owner == "" {
		return
	}
	if owner == strings.TrimSpace(element.ID) {
		v.fail("semantic.invalid_ownership", path+".owner",
			fmt.Sprintf("semantic element %q cannot own itself", element.ID))
		return
	}
	if _, exists := v.elements[owner]; !exists && !isQualifiedSemanticReference(owner) {
		v.fail("semantic.invalid_ownership", path+".owner",
			fmt.Sprintf("semantic element %q owner %q does not exist", element.ID, owner))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkPortion(path string, element SemanticElement) {
	portionOf := strings.TrimSpace(element.PortionOf)
	if portionOf == "" {
		return
	}
	if _, exists := v.elements[portionOf]; !exists {
		v.fail("semantic.dangling_reference", path+".portionOf",
			fmt.Sprintf("time portion %q source occurrence %q does not exist", element.ID, portionOf))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkVariants(path string, element SemanticElement) {
	for j, variant := range element.Variants {
		target, exists := v.elements[strings.TrimSpace(variant)]
		if !exists {
			v.fail("semantic.invalid_variant", indexPath(path+".variants", j),
				fmt.Sprintf("variation %q variant %q does not exist", element.ID, variant))
		} else if target.ID == element.ID {
			v.fail("semantic.invalid_variant", indexPath(path+".variants", j),
				fmt.Sprintf("variation %q cannot contain itself as a variant", element.ID))
		}
	}
	if len(element.Variants) > 0 && !element.Variation {
		v.fail("semantic.invalid_variant", path+".variants",
			fmt.Sprintf("semantic element %q declares variants without being a variation", element.ID))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkRelationships() {
	for i, relationship := range v.semantic.Relationships {
		v.checkRelationship(indexPath("semantic.relationships", i), relationship)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkRelationship(path string, relationship SemanticRelationship) {
	v.checkRelationshipOwner(path, relationship)
	v.validateExpression(relationship.Guard, path+".guard")
	v.validateExpression(relationship.Effect, path+".effect")
	if semanticRelationshipAllowsExternalEndpoints(relationship) {
		return
	}
	v.checkEndpoints(path, relationship)
	v.checkTriggers(path, relationship)
	v.checkItemRef(path, relationship)
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkRelationshipOwner(path string, relationship SemanticRelationship) {
	owner := strings.TrimSpace(relationship.Owner)
	if owner == "" {
		return
	}
	if _, exists := v.elements[owner]; !exists && !isQualifiedSemanticReference(owner) {
		v.fail("semantic.invalid_ownership", path+".owner",
			fmt.Sprintf("semantic relationship %q owner %q does not exist", relationship.ID, owner))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkTriggers(path string, relationship SemanticRelationship) {
	for j, trigger := range relationship.Triggers {
		if trigger = strings.TrimSpace(trigger); v.unresolved(trigger) {
			v.fail("semantic.dangling_relationship", indexPath(path+".triggers", j),
				fmt.Sprintf("semantic transition %q trigger %q does not exist", relationship.ID, trigger))
		}
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkItemRef(path string, relationship SemanticRelationship) {
	itemRef := strings.TrimSpace(relationship.ItemRef)
	if itemRef != "" && v.unresolved(itemRef) {
		v.fail("semantic.dangling_relationship", path+".itemRef",
			fmt.Sprintf("semantic transfer %q item %q does not exist", relationship.ID, itemRef))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkEndpoints(path string, relationship SemanticRelationship) {
	ends := []struct{ endpoint, id string }{{"source", relationship.Source}, {"target", relationship.Target}}
	for _, end := range ends {
		if id := strings.TrimSpace(end.id); v.unresolved(id) {
			v.fail("semantic.dangling_relationship", path+"."+end.endpoint,
				fmt.Sprintf("semantic relationship %q %s %q does not exist", relationship.ID, end.endpoint, id))
		}
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkMetadata() {
	for i, element := range v.semantic.Elements {
		v.checkElementMetadata(indexPath("semantic.elements", i), element)
	}
	for i, relationship := range v.semantic.Relationships {
		v.checkRelationshipMetadata(indexPath("semantic.relationships", i), relationship)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkElementMetadata(path string, element SemanticElement) {
	for j, metadata := range element.Metadata {
		v.validateMetadata(metadata, element.ID, indexPath(path+".metadata", j))
	}
	for j, target := range element.Targets {
		target = strings.TrimSpace(target)
		if _, exists := v.identities[target]; !exists && !strings.Contains(target, "::") {
			v.fail("semantic.invalid_extension_target", indexPath(path+".targets", j),
				fmt.Sprintf("engineering extension %q target %q does not exist", element.ID, target))
		}
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func (v *semanticValidator) checkRelationshipMetadata(path string, relationship SemanticRelationship) {
	for j, metadata := range relationship.Metadata {
		v.validateMetadata(metadata, relationship.ID, indexPath(path+".metadata", j))
	}
	if relationship.Kind != RelationshipVariant {
		return
	}
	if source, exists := v.elements[strings.TrimSpace(relationship.Source)]; !exists || !source.Variation {
		v.fail("semantic.invalid_variant", path+".source",
			fmt.Sprintf("variant relationship %q source must be a variation", relationship.ID))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (v *semanticValidator) validateMetadata(metadata SemanticMetadata, defaultTarget, path string) {
	if strings.TrimSpace(metadata.Namespace) == "" || strings.TrimSpace(metadata.Type) == "" {
		v.fail("semantic.invalid_metadata_type", path, "typed metadata requires namespace and type identity")
	}
	target := strings.TrimSpace(metadata.Target)
	if target == "" {
		target = strings.TrimSpace(defaultTarget)
	}
	if _, exists := v.identities[target]; !exists {
		v.fail("semantic.invalid_metadata_target", path+".target", fmt.Sprintf("metadata target %q does not exist", target))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (v *semanticValidator) validateExpression(expression *SemanticExpression, path string) {
	if expression == nil {
		return
	}
	v.checkExpressionShape(expression, path)
	if expression.TypedValue != nil {
		v.validateSemanticValue(*expression.TypedValue, path+".typedValue")
	}
	for i := range expression.Operands {
		v.validateExpression(&expression.Operands[i], indexPath(path+".operands", i))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (v *semanticValidator) checkExpressionShape(expression *SemanticExpression, path string) {
	hasOperator := strings.TrimSpace(expression.Operator) != ""
	if hasOperator && len(expression.Operands) == 0 {
		v.fail("semantic.invalid_expression", path, "operator expressions require operands")
	}
	if len(expression.Operands) > 0 && !hasOperator {
		v.fail("semantic.invalid_expression", path, "expression operands require an operator")
	}
	if hasOperator && expressionHasScalar(expression) {
		v.fail("semantic.invalid_expression", path, "operator expressions cannot also declare a scalar value")
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func expressionHasScalar(expression *SemanticExpression) bool {
	return strings.TrimSpace(expression.Value) != "" || expression.TypedValue != nil
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (v *semanticValidator) validateSemanticValue(value SemanticValue, path string) {
	if value.Reference != "" {
		v.checkValueReference(value.Reference, path+".reference")
	}
	if value.Quantity != nil {
		v.checkQuantityUnit(value.Quantity.Unit, path+".quantity.unit")
	}
	if strings.TrimSpace(value.Kind) == "" || semanticValueRepresentations(value) != 1 {
		v.fail("semantic.invalid_value", path, "typed values require a kind and exactly one value representation")
	}
}

// semanticValueRepresentations counts the value representations a typed value declares.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func semanticValueRepresentations(value SemanticValue) int {
	declared := []bool{
		value.String != "", value.Integer != nil, value.Real != nil,
		value.Boolean != nil, value.Reference != "", value.Quantity != nil,
	}
	count := 0
	for _, present := range declared {
		if present {
			count++
		}
	}
	return count
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (v *semanticValidator) checkValueReference(reference, path string) {
	if _, exists := v.identities[strings.TrimSpace(reference)]; !exists && !strings.Contains(reference, "::") {
		v.fail("semantic.dangling_reference", path, fmt.Sprintf("typed value reference %q does not exist", reference))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (v *semanticValidator) checkQuantityUnit(unit, path string) {
	if strings.TrimSpace(unit) == "" {
		v.fail("semantic.invalid_quantity", path, "quantity values require a unit reference")
		return
	}
	if _, exists := v.identities[strings.TrimSpace(unit)]; !exists && !strings.Contains(unit, "::") {
		v.fail("semantic.dangling_reference", path, fmt.Sprintf("quantity unit %q does not exist", unit))
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func isConjugableKind(kind SemanticElementKind) bool {
	return kind == ElementPortUsage || kind == ElementInterfaceUsage
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func isTimePortionKind(kind SemanticElementKind) bool {
	return kind == ElementSnapshot || kind == ElementTimeSlice
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func isOccurrenceKind(kind SemanticElementKind) bool {
	switch kind {
	case ElementEventDefinition, ElementEventUsage, ElementOccurrenceDefinition, ElementOccurrenceUsage,
		ElementIndividualDefinition, ElementIndividualUsage, ElementSnapshot, ElementTimeSlice:
		return true
	default:
		return false
	}
}

// isQualifiedSemanticReference reports whether ref is an alias::ID reference
// into a dependency. Its existence and visibility are not known to a single
// model's projection; composition validates it against the dependency's
// selected publication (composition.unpublished_reference).
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
func isQualifiedSemanticReference(ref string) bool {
	_, _, err := ParseQualifiedReference(ref)
	return err == nil
}

// TRLC-LINKS: REQ-EMG-035
func semanticRelationshipAllowsExternalEndpoints(relationship SemanticRelationship) bool {
	switch strings.TrimSpace(relationship.SourceType) {
	case "satisfaction":
		return true
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-035
func semanticMetadata(kind, target string, source any) SemanticMetadata {
	properties := typedObject(source)
	return SemanticMetadata{
		Namespace:  semanticTypeNamespace(kind),
		Type:       kind,
		Target:     strings.TrimSpace(target),
		Properties: properties,
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func typedObject(source any) map[string]MetamodelValue {
	data, err := json.Marshal(source)
	if err != nil {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	out := make(map[string]MetamodelValue, len(raw))
	for key, value := range raw {
		out[key] = typedMetamodelValue(value)
	}
	return out
}

// TRLC-LINKS: REQ-EMG-040
func typedMetamodelValue(value any) MetamodelValue {
	switch value := value.(type) {
	case nil:
		return MetamodelValue{Kind: "null"}
	case string:
		return MetamodelValue{Kind: "string", String: value}
	case bool:
		return MetamodelValue{Kind: "boolean", Boolean: &value}
	case float64:
		return typedNumber(value)
	case []any:
		return typedList(value)
	case map[string]any:
		return typedMap(value)
	default:
		return MetamodelValue{Kind: "string", String: fmt.Sprint(value)}
	}
}

// typedNumber types a decoded JSON number as an integer when it has no fraction.
//
// TRLC-LINKS: REQ-EMG-040
func typedNumber(value float64) MetamodelValue {
	if value == float64(int64(value)) {
		integer := int64(value)
		return MetamodelValue{Kind: "integer", Integer: &integer}
	}
	return MetamodelValue{Kind: "real", Real: &value}
}

// TRLC-LINKS: REQ-EMG-040
func typedList(value []any) MetamodelValue {
	items := make([]MetamodelValue, len(value))
	for i := range value {
		items[i] = typedMetamodelValue(value[i])
	}
	return MetamodelValue{Kind: "list", List: items}
}

// TRLC-LINKS: REQ-EMG-040
func typedMap(value map[string]any) MetamodelValue {
	object := make(map[string]MetamodelValue, len(value))
	for key, item := range value {
		object[key] = typedMetamodelValue(item)
	}
	return MetamodelValue{Kind: "object", Object: object}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func cloneMetamodelProperties(in map[string]MetamodelPropertyValue) map[string]MetamodelPropertyValue {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]MetamodelPropertyValue, len(in))
	for name, property := range in {
		property.Values = append([]MetamodelValue(nil), property.Values...)
		out[name] = property
	}
	return out
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func normalizeMetadata(metadata []SemanticMetadata, target string) []SemanticMetadata {
	out := append([]SemanticMetadata(nil), metadata...)
	for i := range out {
		if strings.TrimSpace(out[i].Namespace) == "" {
			out[i].Namespace = semanticTypeNamespace(out[i].Type)
		}
		if strings.TrimSpace(out[i].Target) == "" {
			out[i].Target = strings.TrimSpace(target)
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func semanticTypeNamespace(typeID string) string {
	typeID = strings.TrimSpace(typeID)
	if index := strings.Index(typeID, "."); index > 0 {
		return typeID[:index]
	}
	return typeID
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func derivedSemanticID(prefix string, parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return strings.ToUpper(prefix) + "-" + strings.ToUpper(hex.EncodeToString(sum[:8]))
}

// TRLC-LINKS: REQ-EMG-035
func semanticReferences(source any, ownID string) []string {
	collector := referenceCollector{seen: map[string]bool{}, ownID: ownID}
	collector.visit(reflect.ValueOf(source))
	refs := make([]string, 0, len(collector.seen))
	for ref := range collector.seen {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

// referenceCollector gathers the string values reachable from a value that
// look like model IDs other than its own.
type referenceCollector struct {
	seen  map[string]bool
	ownID string
}

// TRLC-LINKS: REQ-EMG-035
func (c *referenceCollector) visit(value reflect.Value) {
	if !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.Pointer:
		c.visitPointer(value)
	case reflect.Struct:
		c.visitStruct(value)
	case reflect.Slice, reflect.Array:
		c.visitList(value)
	case reflect.Map:
		c.visitMap(value)
	case reflect.String:
		c.addString(value.String())
	}
}

// TRLC-LINKS: REQ-EMG-035
func (c *referenceCollector) visitPointer(value reflect.Value) {
	if !value.IsNil() {
		c.visit(value.Elem())
	}
}

// TRLC-LINKS: REQ-EMG-035
func (c *referenceCollector) visitStruct(value reflect.Value) {
	for i := 0; i < value.NumField(); i++ {
		c.visit(value.Field(i))
	}
}

// TRLC-LINKS: REQ-EMG-035
func (c *referenceCollector) visitList(value reflect.Value) {
	for i := 0; i < value.Len(); i++ {
		c.visit(value.Index(i))
	}
}

// TRLC-LINKS: REQ-EMG-035
func (c *referenceCollector) visitMap(value reflect.Value) {
	iter := value.MapRange()
	for iter.Next() {
		c.visit(iter.Value())
	}
}

// TRLC-LINKS: REQ-EMG-035
func (c *referenceCollector) addString(text string) {
	ref := strings.TrimSpace(text)
	if ref != c.ownID && semanticIDPattern.MatchString(ref) {
		c.seen[ref] = true
	}
}

// TRLC-LINKS: REQ-EMG-035
func semanticRelationshipID(spec relationshipSpec) string {
	sum := sha256.Sum256([]byte(spec.sourceType + "\x00" + spec.source + "\x00" + spec.target + "\x00" + spec.owner))
	return "REL-" + strings.ToUpper(hex.EncodeToString(sum[:8]))
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func semanticRelationshipAliasKey(kind SemanticRelationshipKind, source, target string) string {
	return string(kind) + "\x00" + strings.TrimSpace(source) + "\x00" + strings.TrimSpace(target)
}

// semanticMappingKinds maps the engineering mapping types to the canonical
// relationship kind they project to.
var semanticMappingKinds = map[string]SemanticRelationshipKind{
	"contains":       RelationshipContainment,
	"calls":          RelationshipTransfer,
	"publishes":      RelationshipTransfer,
	"subscribes":     RelationshipTransfer,
	"reads":          RelationshipTransfer,
	"writes":         RelationshipTransfer,
	"streams":        RelationshipTransfer,
	"allocated_to":   RelationshipAllocation,
	"deployed_to":    RelationshipAllocation,
	"satisfies":      RelationshipSatisfaction,
	"verified_by":    RelationshipVerification,
	"depends_on":     RelationshipDependency,
	"interacts_with": RelationshipDependency,
	"targets":        RelationshipDependency,
	"implements":     RelationshipDependency,
	"guarded_by":     RelationshipGuard,
	"mitigated_by":   RelationshipExtension,
	"bounded_by":     RelationshipExtension,
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-139
func semanticMappingKind(kind string) (SemanticRelationshipKind, bool) {
	relationship, ok := semanticMappingKinds[strings.TrimSpace(kind)]
	return relationship, ok
}

// TRLC-LINKS: REQ-EMG-035
func attributeFeature(name, value string) SemanticFeature {
	value = strings.TrimSpace(value)
	if value == "" {
		return SemanticFeature{}
	}
	return SemanticFeature{
		Name:  name,
		Kind:  FeatureAttribute,
		Type:  "String",
		Value: &SemanticExpression{Language: "literal", Value: value},
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func referenceFeature(shape featureShape, ref string) SemanticFeature {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return SemanticFeature{}
	}
	return SemanticFeature{
		Name: shape.name + "_" + strings.NewReplacer("-", "_", "/", "_").Replace(ref),
		Kind: shape.kind, Type: ref, Direction: shape.direction, References: []string{ref},
	}
}

// featureShape names the kind and direction of a reference feature.
type featureShape struct {
	name      string
	kind      SemanticFeatureKind
	direction string
}

var (
	flowItemFeature     = featureShape{name: "item", kind: FeatureParameter, direction: "inout"}
	flowInputFeature    = featureShape{name: "input", kind: FeatureParameter, direction: "in"}
	flowOutputFeature   = featureShape{name: "output", kind: FeatureParameter, direction: "out"}
	providedPortFeature = featureShape{name: "provided", kind: FeaturePort, direction: "out"}
	requiredPortFeature = featureShape{name: "required", kind: FeaturePort, direction: "in"}
)

// TRLC-LINKS: REQ-EMG-035
func compactFeatures(features []SemanticFeature) []SemanticFeature {
	out := features[:0]
	for _, feature := range features {
		if strings.TrimSpace(feature.Name) != "" {
			out = append(out, feature)
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func compactStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func validSemanticElementKind(kind SemanticElementKind) bool {
	if !knownSysMLSemanticAlias(string(kind)) {
		return false
	}
	switch kind {
	case ElementPackageDefinition, ElementPartDefinition, ElementPartUsage,
		ElementItemDefinition, ElementItemUsage, ElementAttributeDefinition, ElementAttributeUsage,
		ElementPortDefinition, ElementPortUsage, ElementInterfaceDefinition,
		ElementInterfaceUsage, ElementConnectionDefinition, ElementConnectionUsage, ElementActionDefinition,
		ElementActionUsage, ElementControlNode, ElementStateDefinition,
		ElementStateUsage, ElementEventDefinition, ElementEventUsage, ElementViewDefinition,
		ElementViewUsage, ElementViewpointDefinition, ElementViewpointUsage,
		ElementRequirementDefinition, ElementRequirementUsage,
		ElementConcernDefinition, ElementConcernUsage,
		ElementStakeholderDefinition, ElementStakeholderUsage,
		ElementConstraintDefinition, ElementConstraintUsage,
		ElementCalculationDefinition, ElementCalculationUsage,
		ElementCaseDefinition, ElementCaseUsage,
		ElementAnalysisCaseDefinition, ElementAnalysisCaseUsage,
		ElementVerificationCaseDefinition, ElementVerificationCaseUsage,
		ElementUseCaseDefinition, ElementUseCaseUsage,
		ElementQuantityDefinition, ElementUnitDefinition,
		ElementOccurrenceDefinition, ElementOccurrenceUsage,
		ElementIndividualDefinition, ElementIndividualUsage,
		ElementSnapshot, ElementTimeSlice, ElementExtension:
		return true
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func validSemanticRelationshipKind(kind SemanticRelationshipKind) bool {
	if !knownSysMLSemanticAlias(string(kind)) {
		return false
	}
	switch kind {
	case RelationshipContainment, RelationshipDependency, RelationshipTyping,
		RelationshipSpecialization, RelationshipSubsetting, RelationshipRedefinition,
		RelationshipConnection, RelationshipBinding, RelationshipTransfer,
		RelationshipSuccession, RelationshipTransition, RelationshipEventTrigger,
		RelationshipGuard, RelationshipEffect, RelationshipAllocation,
		RelationshipSatisfaction, RelationshipVerification, RelationshipVariant,
		RelationshipReference, RelationshipExtension:
		return true
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func knownSysMLSemanticAlias(kind string) bool {
	if _, ok := OfficialSysMLSemanticAliases[kind]; ok {
		return true
	}
	_, ok := SysMLSemanticAliasExceptions[kind]
	return ok
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func validSemanticFeatureKind(kind SemanticFeatureKind) bool {
	switch kind {
	case "", FeatureAttribute, FeaturePort, FeatureParameter:
		return true
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func validParameterDirection(direction string) bool {
	switch strings.TrimSpace(direction) {
	case "", "in", "out", "inout":
		return true
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func validControlKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "initial", "final", "fork", "join", "merge", "decision":
		return true
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-035
func indexPath(prefix string, index int) string {
	return fmt.Sprintf("%s[%d]", prefix, index)
}

// IsCodeID reports whether id is an inferred, path-derived CODE identifier.
// TRLC-LINKS: REQ-EMG-059
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-CODEMAP-INFERENCE
func IsCodeID(id string) bool {
	return strings.HasPrefix(strings.TrimSpace(id), "CODE-")
}
