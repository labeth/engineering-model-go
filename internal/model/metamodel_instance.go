// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"fmt"
	"strings"
)

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func officialMetaclassForElement(kind SemanticElementKind, extension string) string {
	if kind == ElementExtension {
		return engineeringMetaclass(extension)
	}
	if official := OfficialSysMLSemanticAliases[string(kind)]; official != "" {
		return official
	}
	if _, exception := SysMLSemanticAliasExceptions[string(kind)]; exception {
		return engineeringMetaclass("sysml." + string(kind))
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func officialMetaclassForRelationship(kind SemanticRelationshipKind, sourceType string) string {
	if kind == RelationshipExtension {
		return engineeringMetaclass(sourceType)
	}
	if official := OfficialSysMLSemanticAliases[string(kind)]; official != "" {
		return official
	}
	if _, exception := SysMLSemanticAliasExceptions[string(kind)]; exception {
		return engineeringMetaclass("sysml." + string(kind))
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-040
func engineeringMetaclass(extension string) string {
	extension = strings.TrimSpace(extension)
	if extension == "" {
		return "Engineering::Extension"
	}
	parts := strings.FieldsFunc(extension, func(r rune) bool {
		return strings.ContainsRune(".-_:", r)
	})
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return "Engineering::" + strings.Join(parts, "")
}

// metamodelInstance is one element or relationship whose metaclass and
// property values are checked against the official metamodel.
type metamodelInstance struct {
	id         string
	metaclass  string
	properties map[string]MetamodelPropertyValue
	path       string
	extension  bool
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func validateMetamodelInstance(instance metamodelInstance) []SemanticDiagnostic {
	metaclass := strings.TrimSpace(instance.metaclass)
	if metaclass == "" {
		return nil
	}
	class, ok := OfficialSysMLMetaclasses[metaclass]
	if !ok {
		return validateUnofficialMetaclass(instance, metaclass)
	}
	official := officialInstance{class: class, metaclass: metaclass, path: instance.path}
	var diagnostics []SemanticDiagnostic
	for name, value := range instance.properties {
		diagnostics = append(diagnostics, official.validateProperty(name, value)...)
	}
	return diagnostics
}

// validateUnofficialMetaclass accepts the Engineering:: extension metaclasses
// and rejects every other metaclass that is not in the official inventory.
//
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func validateUnofficialMetaclass(instance metamodelInstance, metaclass string) []SemanticDiagnostic {
	if isEngineeringMetaclass(metaclass, instance.extension) {
		return validateExtensionProperties(instance.properties, instance.path)
	}
	return []SemanticDiagnostic{semanticError("semantic.unknown_metaclass", instance.path+".metaclass",
		fmt.Sprintf("semantic instance %q has unknown official metaclass %q", instance.id, metaclass))}
}

// isEngineeringMetaclass reports whether metaclass is an Engineering::
// extension metaclass; the generic Engineering::Extension is one only for an
// extension instance.
//
// TRLC-LINKS: REQ-EMG-040
func isEngineeringMetaclass(metaclass string, extension bool) bool {
	if !strings.HasPrefix(metaclass, "Engineering::") {
		return false
	}
	return extension || metaclass != "Engineering::Extension"
}

// officialInstance checks the properties of an instance of one official metaclass.
type officialInstance struct {
	class     OfficialSysMLMetaclass
	metaclass string
	path      string
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (o officialInstance) validateProperty(name string, value MetamodelPropertyValue) []SemanticDiagnostic {
	definition, exists := o.class.Properties[name]
	propertyPath := o.path + ".properties." + name
	if !exists {
		return []SemanticDiagnostic{semanticError("semantic.unknown_property", propertyPath,
			fmt.Sprintf("metaclass %q has no property %q", o.metaclass, name))}
	}
	var diagnostics []SemanticDiagnostic
	if definition.Derived && !value.Derived {
		diagnostics = append(diagnostics, semanticError("semantic.read_only_property", propertyPath,
			fmt.Sprintf("derived property %q must be marked derived", definition.ID)))
	}
	if count := len(value.Values); propertyCountInvalid(definition, count) {
		diagnostics = append(diagnostics, semanticError("semantic.invalid_property_multiplicity", propertyPath+".values",
			fmt.Sprintf("property %q has %d values, expected %d..%s", definition.ID, count, definition.Lower, multiplicityUpper(definition.Upper))))
	}
	return append(diagnostics, validatePropertyValues(definition, value, propertyPath)...)
}

// propertyCountInvalid reports a value count outside the property multiplicity.
//
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func propertyCountInvalid(definition OfficialSysMLProperty, count int) bool {
	if count < definition.Lower {
		return true
	}
	return definition.Upper >= 0 && count > definition.Upper
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func validatePropertyValues(definition OfficialSysMLProperty, value MetamodelPropertyValue, propertyPath string) []SemanticDiagnostic {
	var diagnostics []SemanticDiagnostic
	for index, item := range value.Values {
		if !metamodelValueMatches(definition.Kind, definition.Type, item) {
			diagnostics = append(diagnostics, semanticError("semantic.invalid_property_value", indexPath(propertyPath+".values", index),
				fmt.Sprintf("property %q requires %s values, got %q", definition.ID, metamodelValueKind(definition.Kind, definition.Type), item.Kind)))
		}
	}
	return diagnostics
}

// TRLC-LINKS: REQ-EMG-040
func validateExtensionProperties(properties map[string]MetamodelPropertyValue, path string) []SemanticDiagnostic {
	var diagnostics []SemanticDiagnostic
	for name, property := range properties {
		diagnostics = append(diagnostics, invalidExtensionValues(name, property, path)...)
	}
	return diagnostics
}

// TRLC-LINKS: REQ-EMG-040
func invalidExtensionValues(name string, property MetamodelPropertyValue, path string) []SemanticDiagnostic {
	var diagnostics []SemanticDiagnostic
	for index, value := range property.Values {
		if !validMetamodelValue(value) {
			diagnostics = append(diagnostics, semanticError("semantic.invalid_extension_value",
				indexPath(path+".properties."+name+".values", index),
				fmt.Sprintf("extension property %q has an invalid typed value", name)))
		}
	}
	return diagnostics
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func metamodelValueMatches(propertyKind, propertyType string, value MetamodelValue) bool {
	return value.Kind == metamodelValueKind(propertyKind, propertyType) && validMetamodelValue(value)
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func metamodelValueKind(propertyKind, propertyType string) string {
	if propertyKind == "reference" {
		return "reference"
	}
	switch propertyType {
	case "Ecore::Boolean", "SysML::EBoolean":
		return "boolean"
	case "Ecore::Integer":
		return "integer"
	case "Ecore::Real":
		return "real"
	default:
		return "string"
	}
}

// metamodelScalarChecks tells, per scalar value kind, whether a value of that
// kind carries exactly its one representation.
var metamodelScalarChecks = map[string]func(MetamodelValue) bool{
	"string":    func(MetamodelValue) bool { return true },
	"integer":   func(value MetamodelValue) bool { return value.Integer != nil },
	"real":      func(value MetamodelValue) bool { return value.Real != nil },
	"boolean":   func(value MetamodelValue) bool { return value.Boolean != nil },
	"reference": func(value MetamodelValue) bool { return strings.TrimSpace(value.Reference) != "" },
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func validMetamodelValue(value MetamodelValue) bool {
	switch value.Kind {
	case "null":
		return metamodelNullIsEmpty(value)
	case "object":
		return validMetamodelObject(value.Object)
	case "list":
		return validMetamodelList(value.List)
	default:
		check, ok := metamodelScalarChecks[value.Kind]
		return ok && check(value)
	}
}

// metamodelNullIsEmpty reports a null value that carries no representation.
//
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func metamodelNullIsEmpty(value MetamodelValue) bool {
	absent := []bool{
		value.String == "", value.Integer == nil, value.Real == nil, value.Boolean == nil,
		value.Reference == "", len(value.Object) == 0, len(value.List) == 0,
	}
	for _, isAbsent := range absent {
		if !isAbsent {
			return false
		}
	}
	return true
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func validMetamodelObject(object map[string]MetamodelValue) bool {
	for _, child := range object {
		if !validMetamodelValue(child) {
			return false
		}
	}
	return true
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func validMetamodelList(list []MetamodelValue) bool {
	for _, child := range list {
		if !validMetamodelValue(child) {
			return false
		}
	}
	return true
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func multiplicityUpper(upper int) string {
	if upper < 0 {
		return "*"
	}
	return fmt.Sprint(upper)
}

// officialProperty is one official property value to add to an instance.
type officialProperty struct {
	name    string
	derived bool
	value   MetamodelValue
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func normalizeElementMetamodelProperties(element *SemanticElement) {
	if element.Metaclass == "" || strings.HasPrefix(element.Metaclass, "Engineering::") {
		return
	}
	addOfficialProperty(element.Metaclass, &element.Properties, officialProperty{"elementId", false, stringMetamodelValue(element.ID)})
	if element.Name != "" {
		addOfficialProperty(element.Metaclass, &element.Properties, officialProperty{"declaredName", false, stringMetamodelValue(element.Name)})
	}
	if element.Owner != "" {
		addOfficialProperty(element.Metaclass, &element.Properties, officialProperty{"owner", true, referenceMetamodelValue(element.Owner)})
	}
	if element.Namespace != "" {
		addOfficialProperty(element.Metaclass, &element.Properties, officialProperty{"owningNamespace", true, referenceMetamodelValue(element.Namespace)})
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func normalizeRelationshipMetamodelProperties(relationship *SemanticRelationship) {
	if relationship.Metaclass == "" || strings.HasPrefix(relationship.Metaclass, "Engineering::") {
		return
	}
	properties := &relationship.Properties
	addOfficialProperty(relationship.Metaclass, properties, officialProperty{"elementId", false, stringMetamodelValue(relationship.ID)})
	addOfficialProperty(relationship.Metaclass, properties, officialProperty{"source", false, referenceMetamodelValue(relationship.Source)})
	addOfficialProperty(relationship.Metaclass, properties, officialProperty{"target", false, referenceMetamodelValue(relationship.Target)})
	if relationship.Owner != "" {
		addOfficialProperty(relationship.Metaclass, properties, officialProperty{"owningRelatedElement", false, referenceMetamodelValue(relationship.Owner)})
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func addOfficialProperty(metaclass string, properties *map[string]MetamodelPropertyValue, property officialProperty) {
	class, ok := OfficialSysMLMetaclasses[metaclass]
	if !ok {
		return
	}
	definition, ok := class.Properties[property.name]
	if !ok {
		return
	}
	if *properties == nil {
		*properties = map[string]MetamodelPropertyValue{}
	}
	if _, exists := (*properties)[property.name]; exists {
		return
	}
	(*properties)[property.name] = MetamodelPropertyValue{
		Values:  []MetamodelValue{property.value},
		Derived: property.derived || definition.Derived,
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func stringMetamodelValue(value string) MetamodelValue {
	return MetamodelValue{Kind: "string", String: value}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func referenceMetamodelValue(value string) MetamodelValue {
	return MetamodelValue{Kind: "reference", Reference: value}
}
