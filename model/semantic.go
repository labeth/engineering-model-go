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
	if node.Kind == yaml.MappingNode {
		hasKind := false
		for index := 0; index+1 < len(node.Content); index += 2 {
			if node.Content[index].Value == "kind" {
				hasKind = true
				break
			}
		}
		if hasKind {
			var decoded plain
			if err := node.Decode(&decoded); err != nil {
				return err
			}
			*v = MetamodelValue(decoded)
			return nil
		}
	}
	var raw any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	*v = typedMetamodelValue(raw)
	return nil
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

// ProjectSemanticModel derives one deterministic semantic graph from the
// YAML-backed Bundle. It never mutates or replaces the authored model.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL, DO-ARCHITECTURE-MODEL
func ProjectSemanticModel(bundle Bundle) (SemanticModel, []SemanticDiagnostic) {
	a := bundle.Architecture.AuthoredArchitecture
	out := SemanticModel{
		ID:      strings.TrimSpace(bundle.Architecture.Model.ID),
		Title:   strings.TrimSpace(bundle.Architecture.Model.Title),
		Imports: append([]SemanticImport(nil), bundle.Architecture.Semantics.Imports...),
	}
	var diagnostics []SemanticDiagnostic
	elementIndex := map[string]int{}
	relationshipIDs := map[string]string{}
	relationshipAliases := map[string]int{}

	addElement := func(path string, kind SemanticElementKind, extension, id, name, owner, typeRef string, source any, features ...SemanticFeature) {
		id = strings.TrimSpace(id)
		if id == "" {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.missing_id", Severity: SemanticSeverityError,
				Message: "semantic elements require a stable source ID", Path: path,
			})
			return
		}
		if previous, exists := elementIndex[id]; exists {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.duplicate_identity", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("duplicate semantic element ID %q (already projected at elements[%d])", id, previous),
				Path:    path,
			})
			return
		}
		elementIndex[id] = len(out.Elements)
		var targets []string
		if kind == ElementExtension && out.ID != "" && id != out.ID {
			targets = []string{out.ID}
		}
		out.Elements = append(out.Elements, SemanticElement{
			ID:                 id,
			Name:               strings.TrimSpace(name),
			Kind:               kind,
			Metaclass:          officialMetaclassForElement(kind, extension),
			Namespace:          out.ID,
			Owner:              strings.TrimSpace(owner),
			TypeRef:            strings.TrimSpace(typeRef),
			References:         nil,
			Features:           compactFeatures(features),
			Metadata:           []SemanticMetadata{semanticMetadata(extension, id, source)},
			ExtensionNamespace: semanticTypeNamespace(extension),
			Extension:          extension,
			Targets:            targets,
		})
	}

	addRelationship := func(path string, kind SemanticRelationshipKind, sourceType, source, target, owner string, value any) {
		source = strings.TrimSpace(source)
		target = strings.TrimSpace(target)
		if source == "" || target == "" {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.lossy_relationship", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("relationship %q requires non-empty source and target", sourceType), Path: path,
			})
			return
		}
		aliasKey := semanticRelationshipAliasKey(kind, source, target)
		if sourceType == "contains" {
			if existing, ok := relationshipAliases[aliasKey]; ok {
				out.Relationships[existing].Metadata = append(out.Relationships[existing].Metadata,
					semanticMetadata("engineering.relationship_alias."+sourceType, out.Relationships[existing].ID, value))
				return
			}
		}
		id := semanticRelationshipID(sourceType, source, target, owner)
		if previous, exists := relationshipIDs[id]; exists {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.duplicate_identity", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("duplicate semantic relationship %q (already projected from %s)", id, previous),
				Path:    path,
			})
			return
		}
		relationshipIDs[id] = path
		relationshipAliases[aliasKey] = len(out.Relationships)
		out.Relationships = append(out.Relationships, SemanticRelationship{
			ID: id, Kind: kind, Source: source, Target: target, Owner: strings.TrimSpace(owner),
			Metaclass:  officialMetaclassForRelationship(kind, sourceType),
			SourceType: sourceType,
			Metadata:   []SemanticMetadata{semanticMetadata("engineering.relationship."+sourceType, id, value)},
		})
	}
	setExtensionTargets := func(id string, targets []string) {
		if index, ok := elementIndex[strings.TrimSpace(id)]; ok {
			out.Elements[index].Targets = compactStrings(targets)
			out.Elements[index].References = compactStrings(append(out.Elements[index].References, targets...))
		}
	}
	appendMetadata := func(id, metadataType string, source any) {
		if index, ok := elementIndex[strings.TrimSpace(id)]; ok {
			out.Elements[index].Metadata = append(out.Elements[index].Metadata, semanticMetadata(metadataType, id, source))
		}
	}
	addEvidence := func(path, extension, target, evidencePath, description string) {
		if strings.TrimSpace(evidencePath) == "" && strings.TrimSpace(description) == "" {
			return
		}
		id := derivedSemanticID("EVIDENCE", target, path, evidencePath)
		source := map[string]string{"path": evidencePath, "description": description}
		addElement(path, ElementExtension, extension, id, evidencePath, "", "", source)
		setExtensionTargets(id, []string{target})
	}

	addElement("model", ElementPackageDefinition, "engineering.model", bundle.Architecture.Model.ID, bundle.Architecture.Model.Title, "", "", bundle.Architecture.Model)
	if root, ok := elementIndex[out.ID]; ok {
		out.Elements[root].Namespace = ""
	}
	for i, element := range bundle.Architecture.Semantics.Elements {
		path := indexPath("semantics.elements", i)
		if strings.TrimSpace(element.Namespace) == "" {
			element.Namespace = out.ID
		}
		if strings.TrimSpace(element.Extension) == "" {
			element.Extension = "sysml." + string(element.Kind)
		}
		element.Metadata = append(element.Metadata, semanticMetadata("engineering.semantic_source", element.ID, element))
		addElement(path, element.Kind, element.Extension, element.ID, element.Name, element.Owner, element.TypeRef, element, element.Features...)
		if index, ok := elementIndex[strings.TrimSpace(element.ID)]; ok {
			out.Elements[index].Authored = true
			if strings.TrimSpace(element.Metaclass) != "" {
				out.Elements[index].Metaclass = strings.TrimSpace(element.Metaclass)
			}
			out.Elements[index].Properties = cloneMetamodelProperties(element.Properties)
			out.Elements[index].Namespace = strings.TrimSpace(element.Namespace)
			out.Elements[index].Multiplicity = element.Multiplicity
			out.Elements[index].Ordered = element.Ordered
			out.Elements[index].Unique = element.Unique
			out.Elements[index].Conjugated = element.Conjugated
			out.Elements[index].Specializes = compactStrings(element.Specializes)
			out.Elements[index].Subsets = compactStrings(element.Subsets)
			out.Elements[index].Redefines = compactStrings(element.Redefines)
			out.Elements[index].ControlKind = strings.TrimSpace(element.ControlKind)
			out.Elements[index].OccurrenceID = strings.TrimSpace(element.OccurrenceID)
			out.Elements[index].PortionOf = strings.TrimSpace(element.PortionOf)
			out.Elements[index].Variation = element.Variation
			out.Elements[index].Variants = compactStrings(element.Variants)
			out.Elements[index].References = compactStrings(append(out.Elements[index].References, element.References...))
			out.Elements[index].ExtensionNamespace = strings.TrimSpace(element.ExtensionNamespace)
			out.Elements[index].Targets = compactStrings(element.Targets)
			if element.Kind == ElementExtension {
				if out.Elements[index].ExtensionNamespace == "" {
					out.Elements[index].ExtensionNamespace = semanticTypeNamespace(element.Extension)
				}
				if len(out.Elements[index].Targets) == 0 {
					out.Elements[index].Targets = compactStrings(out.Elements[index].References)
					if len(out.Elements[index].Targets) == 0 && element.ID != out.ID {
						out.Elements[index].Targets = []string{out.ID}
					}
				}
			}
			out.Elements[index].Metadata = normalizeMetadata(element.Metadata, element.ID)
		}
	}
	for i, relationship := range bundle.Architecture.Semantics.Relationships {
		path := indexPath("semantics.relationships", i)
		relationship.Source = strings.TrimSpace(relationship.Source)
		relationship.Target = strings.TrimSpace(relationship.Target)
		if relationship.Source == "" || relationship.Target == "" {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.lossy_relationship", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("relationship %q requires non-empty source and target", relationship.Kind), Path: path,
			})
			continue
		}
		if strings.TrimSpace(relationship.ID) == "" {
			relationship.ID = semanticRelationshipID(string(relationship.Kind), relationship.Source, relationship.Target, relationship.Owner)
		}
		if previous, exists := relationshipIDs[relationship.ID]; exists {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.duplicate_identity", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("duplicate semantic relationship %q (already projected from %s)", relationship.ID, previous),
				Path:    path,
			})
			continue
		}
		relationshipIDs[relationship.ID] = path
		relationshipAliases[semanticRelationshipAliasKey(relationship.Kind, relationship.Source, relationship.Target)] = len(out.Relationships)
		if strings.TrimSpace(relationship.SourceType) == "" {
			relationship.SourceType = string(relationship.Kind)
		}
		if strings.TrimSpace(relationship.Metaclass) == "" {
			relationship.Metaclass = officialMetaclassForRelationship(relationship.Kind, relationship.SourceType)
		}
		relationship.Properties = cloneMetamodelProperties(relationship.Properties)
		relationship.Metadata = normalizeMetadata(relationship.Metadata, relationship.ID)
		out.Relationships = append(out.Relationships, relationship)
	}
	for i, x := range a.FunctionalGroups {
		addElement(indexPath("authoredArchitecture.functionalGroups", i), ElementPartDefinition, "engineering.functional_group", x.ID, x.Name, "", "", x)
	}
	for i, x := range a.FunctionalUnits {
		path := indexPath("authoredArchitecture.functionalUnits", i)
		addElement(path, ElementPartUsage, "engineering.functional_unit", x.ID, x.Name, x.Group, "", x)
		if strings.TrimSpace(x.Group) != "" {
			addRelationship(path+".group", RelationshipContainment, "contains", x.Group, x.ID, x.Group, x)
		}
	}
	for i, x := range a.Actors {
		addElement(indexPath("authoredArchitecture.actors", i), ElementStakeholderDefinition, "engineering.actor", x.ID, x.Name, "", "", x)
	}
	for i, x := range a.AttackVectors {
		addElement(indexPath("authoredArchitecture.attackVectors", i), ElementExtension, "engineering.attack_vector", x.ID, x.Name, "", "", x)
	}
	for i, x := range a.ReferencedElements {
		addElement(indexPath("authoredArchitecture.referencedElements", i), ElementExtension, "engineering.referenced_element", x.ID, x.Name, "", "", x)
	}
	for i, x := range a.Interfaces {
		addElement(indexPath("authoredArchitecture.interfaces", i), ElementInterfaceDefinition, "engineering.interface", x.ID, x.Name, x.Owner, "", x,
			attributeFeature("protocol", x.Protocol), attributeFeature("endpoint", x.Endpoint),
			attributeFeature("schemaRef", x.SchemaRef))
	}
	for i, x := range a.DataObjects {
		addElement(indexPath("authoredArchitecture.dataObjects", i), ElementItemDefinition, "engineering.data_object", x.ID, x.Name, "", "", x,
			attributeFeature("schemaRef", x.SchemaRef), attributeFeature("classification", x.Classification))
	}
	for i, x := range a.DeploymentTargets {
		addElement(indexPath("authoredArchitecture.deploymentTargets", i), ElementPartDefinition, "engineering.deployment_target", x.ID, x.Name, "", "", x,
			attributeFeature("environment", x.Environment), attributeFeature("region", x.Region),
			attributeFeature("namespace", x.Namespace), attributeFeature("trustZone", x.TrustZone))
	}
	for i, x := range a.HardwareItems {
		addElement(indexPath("authoredArchitecture.hardwareItems", i), ElementPartDefinition, "engineering.hardware_item", x.ID, x.Name, "", "", x,
			attributeFeature("kind", x.Kind), attributeFeature("partNumber", x.PartNumber),
			attributeFeature("supplier", x.Supplier), attributeFeature("safetyLevel", x.SafetyLevel))
		for _, hosted := range x.Hosts {
			addRelationship(indexPath("authoredArchitecture.hardwareItems", i)+".hosts", RelationshipAllocation, "hosts", hosted, x.ID, x.ID, x)
		}
	}
	for i, x := range a.HardwareInterfaces {
		path := indexPath("authoredArchitecture.hardwareInterfaces", i)
		addElement(path, ElementConnectionUsage, "engineering.hardware_interface", x.ID, x.Name, "", "", x,
			attributeFeature("busType", x.BusType), attributeFeature("direction", x.Direction))
		addRelationship(path, RelationshipConnection, "hardware_interface", x.From, x.To, x.ID, x)
	}
	for i, x := range a.Controls {
		addElement(indexPath("authoredArchitecture.controls", i), ElementExtension, "engineering.control", x.ID, x.Name, "", "", x)
	}
	for i, x := range a.Risks {
		path := indexPath("authoredArchitecture.risks", i)
		addElement(path, ElementExtension, "engineering.risk", x.ID, x.Title, x.Owner, "", x)
		for j, evidence := range x.Evidence {
			addEvidence(indexPath(path+".evidence", j), "engineering.verification_evidence", x.ID, evidence.Path, evidence.Description)
		}
	}
	for i, x := range a.POAMItems {
		path := indexPath("authoredArchitecture.poamItems", i)
		addElement(path, ElementExtension, "engineering.poam_item", x.ID, x.Milestone, x.ResponsibleRole, "", x)
		for j, artifact := range x.Artifacts {
			addEvidence(indexPath(path+".artifacts", j), "engineering.poam_artifact", x.ID, artifact.Path, artifact.Description)
		}
	}
	for i, x := range a.TrustBoundaries {
		path := indexPath("authoredArchitecture.trustBoundaries", i)
		addElement(path, ElementExtension, "engineering.trust_boundary", x.ID, x.Name, x.ParentRef, "", x)
		for _, member := range x.Members {
			addRelationship(path+".members", RelationshipContainment, "boundary_contains", x.ID, member, x.ID, x)
		}
	}
	for i, x := range a.States {
		addElement(indexPath("authoredArchitecture.states", i), ElementStateDefinition, "engineering.state", x.ID, x.Name, "", "", x)
	}
	for i, x := range a.Events {
		addElement(indexPath("authoredArchitecture.events", i), ElementEventDefinition, "engineering.event", x.ID, x.Name, "", "", x)
	}
	for i, x := range a.Flows {
		path := indexPath("authoredArchitecture.flows", i)
		flowFeatures := make([]SemanticFeature, 0, len(x.DataRefs))
		for _, ref := range x.DataRefs {
			flowFeatures = append(flowFeatures, referenceFeature("item", FeatureParameter, "inout", ref))
		}
		addElement(path, ElementActionDefinition, "engineering.flow", x.ID, x.Title, "", "", x, flowFeatures...)
		if x.SourceRef != "" && x.DestinationRef != "" {
			addRelationship(path, RelationshipTransfer, "flow_transfer", x.SourceRef, x.DestinationRef, x.ID, x)
		}
		for j, step := range x.Steps {
			stepPath := indexPath(path+".steps", j)
			stepFeatures := make([]SemanticFeature, 0, len(step.DataIn)+len(step.DataOut)+len(step.DataRefs))
			for _, ref := range step.DataIn {
				stepFeatures = append(stepFeatures, referenceFeature("input", FeatureParameter, "in", ref))
			}
			for _, ref := range step.DataOut {
				stepFeatures = append(stepFeatures, referenceFeature("output", FeatureParameter, "out", ref))
			}
			for _, ref := range step.DataRefs {
				stepFeatures = append(stepFeatures, referenceFeature("item", FeatureParameter, "inout", ref))
			}
			addElement(stepPath, ElementActionUsage, "engineering.flow_step", step.ID, step.Action, x.ID, step.Ref, step, stepFeatures...)
			if step.Ref != "" {
				addRelationship(stepPath+".ref", RelationshipDependency, "flow_ref", step.ID, step.Ref, x.ID, step)
			}
			if step.SourceRef != "" && step.DestinationRef != "" {
				addRelationship(stepPath+".transfer", RelationshipTransfer, "step_transfer", step.SourceRef, step.DestinationRef, step.ID, step)
			}
			for _, next := range step.Next {
				addRelationship(stepPath+".next", RelationshipSuccession, "flow_next", step.ID, next, x.ID, step)
			}
			for _, next := range step.OnError {
				addRelationship(stepPath+".onError", RelationshipSuccession, "flow_error", step.ID, next, x.ID, step)
			}
		}
	}
	for i, x := range a.ThreatScenarios {
		path := indexPath("authoredArchitecture.threatScenarios", i)
		addElement(path, ElementExtension, "engineering.threat_scenario", x.ID, x.Title, x.Owner, "", x)
		for j, evidence := range x.Evidence {
			addEvidence(indexPath(path+".evidence", j), "engineering.verification_evidence", x.ID, evidence.Path, evidence.Description)
		}
	}
	for i, x := range a.ThreatAssumptions {
		path := indexPath("authoredArchitecture.threatAssumptions", i)
		addElement(path, ElementExtension, "engineering.threat_assumption", x.ID, x.Title, x.Owner, "", x)
		for j, evidence := range x.Evidence {
			addEvidence(indexPath(path+".evidence", j), "engineering.verification_evidence", x.ID, evidence.Path, evidence.Description)
		}
	}
	for i, x := range a.ThreatOutOfScope {
		path := indexPath("authoredArchitecture.threatOutOfScope", i)
		addElement(path, ElementExtension, "engineering.threat_out_of_scope", x.ID, x.Title, x.Owner, "", x)
		for j, evidence := range x.Evidence {
			addEvidence(indexPath(path+".evidence", j), "engineering.verification_evidence", x.ID, evidence.Path, evidence.Description)
		}
	}
	for i, x := range a.ThreatMitigations {
		path := indexPath("authoredArchitecture.threatMitigations", i)
		addElement(path, ElementExtension, "engineering.threat_mitigation", x.ID, x.ID, x.Owner, "", x)
		for j, evidence := range x.Evidence {
			addEvidence(indexPath(path+".evidence", j), "engineering.verification_evidence", x.ID, evidence.Path, evidence.Description)
		}
	}
	for i, x := range a.ControlVerifications {
		path := indexPath("authoredArchitecture.controlVerifications", i)
		addElement(path, ElementVerificationCaseUsage, "engineering.control_verification", x.ID, x.ID, x.Owner, "", x)
		if x.ControlRef != "" {
			addRelationship(path+".controlRef", RelationshipVerification, "verifies", x.ID, x.ControlRef, x.ID, x)
		}
		for j, evidence := range x.Evidence {
			addEvidence(indexPath(path+".evidence", j), "engineering.verification_evidence", x.ID, evidence.Path, evidence.Description)
		}
	}
	for i, x := range a.Mappings {
		kind, ok := semanticMappingKind(x.Type)
		if !ok {
			kind = RelationshipExtension
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.unsupported_relationship", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("mapping type %q has no canonical relationship mapping", x.Type),
				Path:    indexPath("authoredArchitecture.mappings", i),
			})
		}
		if kind == RelationshipEventTrigger || kind == RelationshipGuard {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.legacy_lifecycle_attachment", Severity: SemanticSeverityWarning,
				Message: fmt.Sprintf("legacy %q mapping is attached to state %q and preserved with its original mapping type", x.Type, x.From),
				Path:    indexPath("authoredArchitecture.mappings", i),
			})
		}
		addRelationship(indexPath("authoredArchitecture.mappings", i), kind, strings.TrimSpace(x.Type), x.From, x.To, "", x)
	}

	for i, x := range bundle.Architecture.Compliance.Profiles {
		addElement(indexPath("compliance.profiles", i), ElementExtension, "engineering.compliance_profile", x.ID, x.ID, "", "", x)
	}
	for i, x := range bundle.Architecture.Compliance.Mappings {
		path := indexPath("compliance.mappings", i)
		addElement(path, ElementExtension, "engineering.compliance_mapping", x.ID, x.ID, "", "", x)
		for j, evidence := range x.Evidence {
			addEvidence(indexPath(path+".evidence", j), "engineering.compliance_evidence", x.ID, evidence.Path, evidence.Description)
		}
	}
	for i, x := range bundle.Architecture.Contract.Provides {
		addElement(indexPath("contract.provides", i), ElementPortUsage, "engineering.contract_provides", x.ID, x.ID, out.ID, x.Ref, x,
			referenceFeature("provided", FeaturePort, "out", x.Ref))
	}
	for i, x := range bundle.Architecture.Contract.Requires {
		addElement(indexPath("contract.requires", i), ElementPortUsage, "engineering.contract_requires", x.ID, x.ID, out.ID, x.Ref, x,
			referenceFeature("required", FeaturePort, "in", x.Ref))
		if index, ok := elementIndex[strings.TrimSpace(x.ID)]; ok {
			out.Elements[index].Conjugated = true
		}
	}
	for i, x := range bundle.Architecture.Composition.Subsystems {
		addElement(indexPath("composition.subsystems", i), ElementPartUsage, "engineering.subsystem", x.ID, x.Name, out.ID, "", x)
	}
	for i, x := range bundle.Architecture.Composition.Allocations {
		target := x.To
		if strings.TrimSpace(x.Target) != "" {
			target += "/" + strings.TrimSpace(x.Target)
		}
		addRelationship(indexPath("composition.allocations", i), RelationshipAllocation, "allocation", x.Requirement, target, out.ID, x)
	}
	for i, x := range bundle.Architecture.Composition.Satisfactions {
		addRelationship(indexPath("composition.satisfactions", i), RelationshipSatisfaction, "satisfaction", x.Need, x.By, out.ID, x)
	}
	for i, x := range bundle.Architecture.Views {
		addElement(indexPath("views", i), ElementViewUsage, "engineering.view", x.ID, x.ID, out.ID, "", x)
	}
	if bundle.Architecture.NAF.Enabled() {
		appendMetadata(out.ID, "naf.profile.v4_1", bundle.Architecture.NAF)
		for _, product := range bundle.Architecture.NAF.Products {
			appendMetadata(product.ViewRef, "naf.viewpoint."+product.Viewpoint, product)
		}
	}
	for i, x := range bundle.Architecture.Decisions {
		addElement(indexPath("decisions", i), ElementExtension, "engineering.architecture_decision", x.ID, x.Title, "", "", x)
	}
	for i, x := range bundle.Requirements.Requirements {
		path := indexPath("requirements", i)
		addElement(path, ElementRequirementDefinition, "engineering.requirement_source", x.ID, x.ID, out.ID, "", x,
			attributeFeature("text", x.Text), attributeFeature("notes", x.Notes))
	}
	for i, x := range bundle.Requirements.Expected {
		addElement(indexPath("expected", i), ElementExtension, "engineering.requirement_pattern", x.ID, x.ID, out.ID, "", x)
	}
	if !reflect.DeepEqual(bundle.Requirements.LintRun, LintRun{}) {
		appendMetadata(out.ID, "engineering.requirements_lint_policy", bundle.Requirements.LintRun)
	}
	if !reflect.DeepEqual(bundle.Design.Design, DesignModel{}) {
		appendMetadata(out.ID, "engineering.design_model", bundle.Design.Design)
	}
	for _, group := range bundle.Design.Design.FunctionalGroups {
		appendMetadata(group.ID, "engineering.design_functional_group", group)
		for key, narrative := range group.Views {
			path := "design.functionalGroups." + group.ID + ".views." + key
			id := derivedSemanticID("CONCERN", group.ID, key)
			addElement(path, ElementConcernUsage, "engineering.design_narrative", id, narrative.Title, group.ID, "", narrative,
				attributeFeature("narrative", narrative.Narrative))
		}
	}
	for _, unit := range bundle.Design.Design.FunctionalUnits {
		appendMetadata(unit.ID, "engineering.design_functional_unit", unit)
		for key, narrative := range unit.Views {
			path := "design.functionalUnits." + unit.ID + ".views." + key
			id := derivedSemanticID("CONCERN", unit.ID, key)
			addElement(path, ElementConcernUsage, "engineering.design_narrative", id, narrative.Title, unit.ID, "", narrative,
				attributeFeature("narrative", narrative.Narrative))
		}
	}
	if !reflect.DeepEqual(bundle.Architecture.InferenceHints, InferenceHints{}) {
		id := derivedSemanticID("EXTENSION", out.ID, "code-ownership-policy")
		addElement("inferenceHints", ElementExtension, "engineering.code_ownership_policy", id, "Code ownership policy", "", "", bundle.Architecture.InferenceHints)
		setExtensionTargets(id, []string{out.ID})
	}

	catalogIDs := map[string]string{}
	mergeCatalog := func(group string, entries []CatalogEntry) {
		for i, entry := range entries {
			path := indexPath("catalog."+group, i)
			id := strings.TrimSpace(entry.ID)
			if previous, exists := catalogIDs[id]; exists {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.duplicate_identity", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("duplicate catalog ID %q (already declared in %s)", id, previous), Path: path,
				})
				continue
			}
			catalogIDs[id] = group
			if idx, exists := elementIndex[id]; exists {
				out.Elements[idx].Metadata = append(out.Elements[idx].Metadata, semanticMetadata("engineering.catalog."+group, id, entry))
				continue
			}
			addElement(path, ElementExtension, "engineering.catalog."+group, entry.ID, entry.Name, "", "", entry)
		}
	}
	c := bundle.Catalog.Catalog
	mergeCatalog("systems", c.Systems)
	mergeCatalog("functional_groups", c.FunctionalGroups)
	mergeCatalog("functional_units", c.FunctionalUnits)
	mergeCatalog("referenced_elements", c.ReferencedElements)
	mergeCatalog("actors", c.Actors)
	mergeCatalog("attack_vectors", c.AttackVectors)
	mergeCatalog("events", c.Events)
	mergeCatalog("states", c.States)
	mergeCatalog("features", c.Features)
	mergeCatalog("modes", c.Modes)
	mergeCatalog("conditions", c.Conditions)
	mergeCatalog("data_terms", c.DataTerms)

	for index := range out.Elements {
		normalizeElementMetamodelProperties(&out.Elements[index])
	}
	for index := range out.Relationships {
		normalizeRelationshipMetamodelProperties(&out.Relationships[index])
	}
	sort.Slice(out.Elements, func(i, j int) bool { return out.Elements[i].ID < out.Elements[j].ID })
	sort.Slice(out.Relationships, func(i, j int) bool { return out.Relationships[i].ID < out.Relationships[j].ID })
	sort.Slice(out.Imports, func(i, j int) bool {
		if out.Imports[i].Namespace != out.Imports[j].Namespace {
			return out.Imports[i].Namespace < out.Imports[j].Namespace
		}
		return out.Imports[i].Imported < out.Imports[j].Imported
	})
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path != diagnostics[j].Path {
			return diagnostics[i].Path < diagnostics[j].Path
		}
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
	return out, diagnostics
}

// ValidateSemanticModel checks invariants of the canonical projection independently
// of the YAML structures from which it was derived.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func ValidateSemanticModel(semantic SemanticModel) []SemanticDiagnostic {
	var diagnostics []SemanticDiagnostic
	identities := map[string]string{}
	elements := map[string]SemanticElement{}
	occurrences := map[string]string{}

	addIdentity := func(id, path string) {
		id = strings.TrimSpace(id)
		if id == "" {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.missing_identity", Severity: SemanticSeverityError,
				Message: "canonical semantic identities must not be empty", Path: path,
			})
			return
		}
		if previous, exists := identities[id]; exists {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.duplicate_identity", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("duplicate canonical semantic identity %q (already declared at %s)", id, previous),
				Path:    path,
			})
			return
		}
		identities[id] = path
	}

	for i, element := range semantic.Elements {
		path := indexPath("semantic.elements", i)
		addIdentity(element.ID, path)
		diagnostics = append(diagnostics, validateMetamodelInstance(
			element.ID, element.Metaclass, element.Properties, path, element.Kind == ElementExtension,
		)...)
		if strings.TrimSpace(element.ID) != "" {
			if _, exists := elements[element.ID]; !exists {
				elements[element.ID] = element
			}
			if !validSemanticElementKind(element.Kind) {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.unsupported_element_kind", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic element %q has unsupported kind %q", element.ID, element.Kind), Path: path + ".kind",
				})
			}
			if element.Conjugated && element.Kind != ElementPortUsage && element.Kind != ElementInterfaceUsage {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_conjugation", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic element %q can only be conjugated when it is a port or interface usage", element.ID),
					Path:    path + ".conjugated",
				})
			}
			if element.Kind == ElementControlNode && !validControlKind(element.ControlKind) {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_control_node", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("control node %q has unsupported kind %q", element.ID, element.ControlKind),
					Path:    path + ".controlKind",
				})
			}
			if occurrenceID := strings.TrimSpace(element.OccurrenceID); occurrenceID != "" {
				if previous, exists := occurrences[occurrenceID]; exists {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.duplicate_occurrence_identity", Severity: SemanticSeverityError,
						Message: fmt.Sprintf("occurrence identity %q is already used at %s", occurrenceID, previous),
						Path:    path + ".occurrenceId",
					})
				} else {
					occurrences[occurrenceID] = path + ".occurrenceId"
				}
				if !isOccurrenceKind(element.Kind) {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_occurrence_identity", Severity: SemanticSeverityError,
						Message: fmt.Sprintf("semantic element %q cannot declare occurrence identity", element.ID),
						Path:    path + ".occurrenceId",
					})
				}
			}
			if portionOf := strings.TrimSpace(element.PortionOf); portionOf != "" && element.Kind != ElementSnapshot && element.Kind != ElementTimeSlice {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_time_portion", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic element %q can only use portionOf as a snapshot or time slice", element.ID),
					Path:    path + ".portionOf",
				})
			}
			if element.Kind == ElementExtension {
				if strings.TrimSpace(element.ExtensionNamespace) == "" || strings.TrimSpace(element.Extension) == "" {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_extension_type", Severity: SemanticSeverityError,
						Message: fmt.Sprintf("engineering extension %q requires namespace and type identity", element.ID),
						Path:    path,
					})
				}
				if len(element.Targets) == 0 && element.Authored {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_extension_target", Severity: SemanticSeverityError,
						Message: fmt.Sprintf("engineering extension %q requires at least one explicit target", element.ID),
						Path:    path + ".targets",
					})
				}
			}
			validateMultiplicity := func(multiplicity *Multiplicity, multiplicityPath string) {
				if multiplicity == nil {
					return
				}
				if multiplicity.Lower != nil && *multiplicity.Lower < 0 {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_multiplicity", Severity: SemanticSeverityError,
						Message: "multiplicity lower bound must not be negative", Path: multiplicityPath + ".lower",
					})
				}
				if multiplicity.Upper != nil && *multiplicity.Upper < -1 {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_multiplicity", Severity: SemanticSeverityError,
						Message: "multiplicity upper bound must be -1 (unbounded) or non-negative", Path: multiplicityPath + ".upper",
					})
				}
				if multiplicity.Lower != nil && multiplicity.Upper != nil && *multiplicity.Upper >= 0 && *multiplicity.Lower > *multiplicity.Upper {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_multiplicity", Severity: SemanticSeverityError,
						Message: "multiplicity lower bound must not exceed upper bound", Path: multiplicityPath,
					})
				}
			}
			validateMultiplicity(element.Multiplicity, path+".multiplicity")
			for j, feature := range element.Features {
				featurePath := indexPath(path+".features", j)
				if strings.TrimSpace(feature.Name) == "" {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_feature", Severity: SemanticSeverityError,
						Message: "semantic feature name must not be empty", Path: featurePath + ".name",
					})
				}
				if !validSemanticFeatureKind(feature.Kind) {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_feature", Severity: SemanticSeverityError,
						Message: fmt.Sprintf("semantic feature %q has unsupported kind %q", feature.Name, feature.Kind), Path: featurePath + ".kind",
					})
				}
				if feature.Kind == FeatureParameter && !validParameterDirection(feature.Direction) {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_parameter_direction", Severity: SemanticSeverityError,
						Message: fmt.Sprintf("parameter %q has unsupported direction %q", feature.Name, feature.Direction), Path: featurePath + ".direction",
					})
				}
				if feature.Conjugated && feature.Kind != FeaturePort {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.invalid_conjugation", Severity: SemanticSeverityError,
						Message: fmt.Sprintf("semantic feature %q can only be conjugated when it is a port", feature.Name),
						Path:    featurePath + ".conjugated",
					})
				}
				validateMultiplicity(feature.Multiplicity, featurePath+".multiplicity")
				if feature.Value != nil && feature.Multiplicity != nil && feature.Multiplicity.Upper != nil && *feature.Multiplicity.Upper == 0 {
					diagnostics = append(diagnostics, SemanticDiagnostic{
						Code: "semantic.value_multiplicity_mismatch", Severity: SemanticSeverityError,
						Message: fmt.Sprintf("semantic feature %q cannot have a value when multiplicity upper bound is zero", feature.Name),
						Path:    featurePath + ".value",
					})
				}
			}
		}
	}
	for i, relationship := range semantic.Relationships {
		path := indexPath("semantic.relationships", i)
		addIdentity(relationship.ID, path)
		diagnostics = append(diagnostics, validateMetamodelInstance(
			relationship.ID, relationship.Metaclass, relationship.Properties, path, relationship.Kind == RelationshipExtension,
		)...)
		if !validSemanticRelationshipKind(relationship.Kind) {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.unsupported_relationship", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("semantic relationship %q has unsupported kind %q", relationship.ID, relationship.Kind),
				Path:    indexPath("semantic.relationships", i) + ".kind",
			})
		}
	}

	for i, imported := range semantic.Imports {
		path := indexPath("semantic.imports", i)
		if strings.TrimSpace(imported.Imported) == "" {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.invalid_import", Severity: SemanticSeverityError,
				Message: "semantic import requires an imported namespace", Path: path + ".imported",
			})
		}
		if visibility := strings.TrimSpace(imported.Visibility); visibility != "" && visibility != "private" && visibility != "public" {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.invalid_import", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("semantic import visibility %q must be private or public", visibility), Path: path + ".visibility",
			})
		}
		if namespace := strings.TrimSpace(imported.Namespace); namespace != "" {
			element, exists := elements[namespace]
			if !exists || element.Kind != ElementPackageDefinition {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_import", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic import namespace %q must resolve to a package definition", namespace), Path: path + ".namespace",
				})
			}
		}
	}

	for i, element := range semantic.Elements {
		path := indexPath("semantic.elements", i)
		if namespace := strings.TrimSpace(element.Namespace); namespace != "" {
			if namespace == strings.TrimSpace(element.ID) {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_ownership", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic element %q cannot be its own namespace", element.ID),
					Path:    path + ".namespace",
				})
				continue
			}
			namespaceElement, exists := elements[namespace]
			if !exists || namespaceElement.Kind != ElementPackageDefinition {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_ownership", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic element %q namespace %q must resolve to a package definition", element.ID, namespace),
					Path:    path + ".namespace",
				})
			}
		}
		if owner := strings.TrimSpace(element.Owner); owner != "" {
			if owner == strings.TrimSpace(element.ID) {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_ownership", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic element %q cannot own itself", element.ID),
					Path:    path + ".owner",
				})
			} else if _, exists := elements[owner]; !exists {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_ownership", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic element %q owner %q does not exist", element.ID, owner),
					Path:    path + ".owner",
				})
			}
		}
		for j, reference := range element.References {
			if _, exists := elements[strings.TrimSpace(reference)]; !exists {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.dangling_reference", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic element %q reference %q does not exist", element.ID, reference),
					Path:    indexPath(path+".references", j),
				})
			}
		}
		for j := range element.Features {
			validateExpression(element.Features[j].Value, indexPath(path+".features", j)+".value", identities, &diagnostics)
		}
		if portionOf := strings.TrimSpace(element.PortionOf); portionOf != "" {
			if _, exists := elements[portionOf]; !exists {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.dangling_reference", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("time portion %q source occurrence %q does not exist", element.ID, portionOf),
					Path:    path + ".portionOf",
				})
			}
		}
		for j, variant := range element.Variants {
			target, exists := elements[strings.TrimSpace(variant)]
			if !exists {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_variant", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("variation %q variant %q does not exist", element.ID, variant),
					Path:    indexPath(path+".variants", j),
				})
			} else if target.ID == element.ID {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_variant", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("variation %q cannot contain itself as a variant", element.ID),
					Path:    indexPath(path+".variants", j),
				})
			}
		}
		if len(element.Variants) > 0 && !element.Variation {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.invalid_variant", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("semantic element %q declares variants without being a variation", element.ID),
				Path:    path + ".variants",
			})
		}
	}

	for i, relationship := range semantic.Relationships {
		path := indexPath("semantic.relationships", i)
		if owner := strings.TrimSpace(relationship.Owner); owner != "" {
			if _, exists := elements[owner]; !exists {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_ownership", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic relationship %q owner %q does not exist", relationship.ID, owner),
					Path:    path + ".owner",
				})
			}
		}
		validateExpression(relationship.Guard, path+".guard", identities, &diagnostics)
		validateExpression(relationship.Effect, path+".effect", identities, &diagnostics)
		if semanticRelationshipAllowsExternalEndpoints(relationship) {
			continue
		}
		for endpoint, id := range map[string]string{"source": relationship.Source, "target": relationship.Target} {
			id = strings.TrimSpace(id)
			if _, exists := elements[id]; exists {
				continue
			}
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.dangling_relationship", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("semantic relationship %q %s %q does not exist", relationship.ID, endpoint, id),
				Path:    path + "." + endpoint,
			})
		}
		for j, trigger := range relationship.Triggers {
			if _, exists := elements[strings.TrimSpace(trigger)]; !exists {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.dangling_relationship", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic transition %q trigger %q does not exist", relationship.ID, trigger),
					Path:    indexPath(path+".triggers", j),
				})
			}
		}
		if itemRef := strings.TrimSpace(relationship.ItemRef); itemRef != "" {
			if _, exists := elements[itemRef]; !exists {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.dangling_relationship", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("semantic transfer %q item %q does not exist", relationship.ID, itemRef),
					Path:    path + ".itemRef",
				})
			}
		}
	}

	for i, element := range semantic.Elements {
		path := indexPath("semantic.elements", i)
		for j, metadata := range element.Metadata {
			validateMetadata(metadata, element.ID, indexPath(path+".metadata", j), identities, &diagnostics)
		}
		for j, target := range element.Targets {
			if _, exists := identities[strings.TrimSpace(target)]; !exists {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_extension_target", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("engineering extension %q target %q does not exist", element.ID, target),
					Path:    indexPath(path+".targets", j),
				})
			}
		}
	}
	for i, relationship := range semantic.Relationships {
		path := indexPath("semantic.relationships", i)
		for j, metadata := range relationship.Metadata {
			validateMetadata(metadata, relationship.ID, indexPath(path+".metadata", j), identities, &diagnostics)
		}
		if relationship.Kind == RelationshipVariant {
			source, sourceExists := elements[strings.TrimSpace(relationship.Source)]
			if !sourceExists || !source.Variation {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_variant", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("variant relationship %q source must be a variation", relationship.ID),
					Path:    path + ".source",
				})
			}
		}
	}

	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path != diagnostics[j].Path {
			return diagnostics[i].Path < diagnostics[j].Path
		}
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
	return diagnostics
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func validateMetadata(metadata SemanticMetadata, defaultTarget, path string, identities map[string]string, diagnostics *[]SemanticDiagnostic) {
	if strings.TrimSpace(metadata.Namespace) == "" || strings.TrimSpace(metadata.Type) == "" {
		*diagnostics = append(*diagnostics, SemanticDiagnostic{
			Code: "semantic.invalid_metadata_type", Severity: SemanticSeverityError,
			Message: "typed metadata requires namespace and type identity", Path: path,
		})
	}
	target := strings.TrimSpace(metadata.Target)
	if target == "" {
		target = strings.TrimSpace(defaultTarget)
	}
	if _, exists := identities[target]; !exists {
		*diagnostics = append(*diagnostics, SemanticDiagnostic{
			Code: "semantic.invalid_metadata_target", Severity: SemanticSeverityError,
			Message: fmt.Sprintf("metadata target %q does not exist", target), Path: path + ".target",
		})
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func validateExpression(expression *SemanticExpression, path string, identities map[string]string, diagnostics *[]SemanticDiagnostic) {
	if expression == nil {
		return
	}
	hasOperator := strings.TrimSpace(expression.Operator) != ""
	hasScalar := strings.TrimSpace(expression.Value) != "" || expression.TypedValue != nil
	if hasOperator && len(expression.Operands) == 0 {
		*diagnostics = append(*diagnostics, SemanticDiagnostic{
			Code: "semantic.invalid_expression", Severity: SemanticSeverityError,
			Message: "operator expressions require operands", Path: path,
		})
	}
	if len(expression.Operands) > 0 && !hasOperator {
		*diagnostics = append(*diagnostics, SemanticDiagnostic{
			Code: "semantic.invalid_expression", Severity: SemanticSeverityError,
			Message: "expression operands require an operator", Path: path,
		})
	}
	if hasOperator && hasScalar {
		*diagnostics = append(*diagnostics, SemanticDiagnostic{
			Code: "semantic.invalid_expression", Severity: SemanticSeverityError,
			Message: "operator expressions cannot also declare a scalar value", Path: path,
		})
	}
	if expression.TypedValue != nil {
		validateSemanticValue(*expression.TypedValue, path+".typedValue", identities, diagnostics)
	}
	for i := range expression.Operands {
		validateExpression(&expression.Operands[i], indexPath(path+".operands", i), identities, diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func validateSemanticValue(value SemanticValue, path string, identities map[string]string, diagnostics *[]SemanticDiagnostic) {
	count := 0
	if value.String != "" {
		count++
	}
	if value.Integer != nil {
		count++
	}
	if value.Real != nil {
		count++
	}
	if value.Boolean != nil {
		count++
	}
	if value.Reference != "" {
		count++
		if _, exists := identities[strings.TrimSpace(value.Reference)]; !exists && !strings.Contains(value.Reference, "::") {
			*diagnostics = append(*diagnostics, SemanticDiagnostic{
				Code: "semantic.dangling_reference", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("typed value reference %q does not exist", value.Reference), Path: path + ".reference",
			})
		}
	}
	if value.Quantity != nil {
		count++
		if strings.TrimSpace(value.Quantity.Unit) == "" {
			*diagnostics = append(*diagnostics, SemanticDiagnostic{
				Code: "semantic.invalid_quantity", Severity: SemanticSeverityError,
				Message: "quantity values require a unit reference", Path: path + ".quantity.unit",
			})
		} else if _, exists := identities[strings.TrimSpace(value.Quantity.Unit)]; !exists && !strings.Contains(value.Quantity.Unit, "::") {
			*diagnostics = append(*diagnostics, SemanticDiagnostic{
				Code: "semantic.dangling_reference", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("quantity unit %q does not exist", value.Quantity.Unit), Path: path + ".quantity.unit",
			})
		}
	}
	if strings.TrimSpace(value.Kind) == "" || count != 1 {
		*diagnostics = append(*diagnostics, SemanticDiagnostic{
			Code: "semantic.invalid_value", Severity: SemanticSeverityError,
			Message: "typed values require a kind and exactly one value representation", Path: path,
		})
	}
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

// TRLC-LINKS: REQ-EMG-035
func semanticRelationshipAllowsExternalEndpoints(relationship SemanticRelationship) bool {
	switch strings.TrimSpace(relationship.SourceType) {
	case "allocation", "satisfaction":
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
		if value == float64(int64(value)) {
			integer := int64(value)
			return MetamodelValue{Kind: "integer", Integer: &integer}
		}
		return MetamodelValue{Kind: "real", Real: &value}
	case []any:
		items := make([]MetamodelValue, len(value))
		for i := range value {
			items[i] = typedMetamodelValue(value[i])
		}
		return MetamodelValue{Kind: "list", List: items}
	case map[string]any:
		object := make(map[string]MetamodelValue, len(value))
		for key, item := range value {
			object[key] = typedMetamodelValue(item)
		}
		return MetamodelValue{Kind: "object", Object: object}
	default:
		return MetamodelValue{Kind: "string", String: fmt.Sprint(value)}
	}
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
	seen := map[string]bool{}
	var visit func(reflect.Value)
	visit = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		if value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			visit(value.Elem())
			return
		}
		switch value.Kind() {
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				visit(value.Field(i))
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < value.Len(); i++ {
				visit(value.Index(i))
			}
		case reflect.Map:
			iter := value.MapRange()
			for iter.Next() {
				visit(iter.Value())
			}
		case reflect.String:
			ref := strings.TrimSpace(value.String())
			if ref != ownID && semanticIDPattern.MatchString(ref) {
				seen[ref] = true
			}
		}
	}
	visit(reflect.ValueOf(source))
	refs := make([]string, 0, len(seen))
	for ref := range seen {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

// TRLC-LINKS: REQ-EMG-035
func semanticRelationshipID(kind, source, target, owner string) string {
	sum := sha256.Sum256([]byte(kind + "\x00" + source + "\x00" + target + "\x00" + owner))
	return "REL-" + strings.ToUpper(hex.EncodeToString(sum[:8]))
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func semanticRelationshipAliasKey(kind SemanticRelationshipKind, source, target string) string {
	return string(kind) + "\x00" + strings.TrimSpace(source) + "\x00" + strings.TrimSpace(target)
}

// TRLC-LINKS: REQ-EMG-035
func semanticMappingKind(kind string) (SemanticRelationshipKind, bool) {
	switch strings.TrimSpace(kind) {
	case "contains":
		return RelationshipContainment, true
	case "calls", "publishes", "subscribes", "reads", "writes", "streams":
		return RelationshipTransfer, true
	case "transitions_to":
		return RelationshipTransition, true
	case "allocated_to", "deployed_to":
		return RelationshipAllocation, true
	case "satisfies":
		return RelationshipSatisfaction, true
	case "verified_by":
		return RelationshipVerification, true
	case "depends_on", "interacts_with", "targets", "implements":
		return RelationshipDependency, true
	case "triggered_by":
		return RelationshipEventTrigger, true
	case "guarded_by":
		return RelationshipGuard, true
	case "mitigated_by", "bounded_by":
		return RelationshipExtension, true
	default:
		return "", false
	}
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
func referenceFeature(name string, kind SemanticFeatureKind, direction, ref string) SemanticFeature {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return SemanticFeature{}
	}
	return SemanticFeature{
		Name: name + "_" + strings.NewReplacer("-", "_", "/", "_").Replace(ref),
		Kind: kind, Type: ref, Direction: direction, References: []string{ref},
	}
}

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
