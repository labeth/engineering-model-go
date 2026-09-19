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
		return r == '.' || r == '-' || r == '_' || r == ':'
	})
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return "Engineering::" + strings.Join(parts, "")
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func validateMetamodelInstance(id, metaclass string, properties map[string]MetamodelPropertyValue, path string, extension bool) []SemanticDiagnostic {
	metaclass = strings.TrimSpace(metaclass)
	if metaclass == "" {
		return nil
	}
	class, ok := OfficialSysMLMetaclasses[metaclass]
	if !ok {
		if strings.HasPrefix(metaclass, "Engineering::") && (extension || metaclass != "Engineering::Extension") {
			return validateExtensionProperties(properties, path)
		}
		return []SemanticDiagnostic{{
			Code: "semantic.unknown_metaclass", Severity: SemanticSeverityError,
			Message: fmt.Sprintf("semantic instance %q has unknown official metaclass %q", id, metaclass),
			Path:    path + ".metaclass",
		}}
	}
	var diagnostics []SemanticDiagnostic
	for name, value := range properties {
		definition, exists := class.Properties[name]
		propertyPath := path + ".properties." + name
		if !exists {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.unknown_property", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("metaclass %q has no property %q", metaclass, name), Path: propertyPath,
			})
			continue
		}
		if definition.Derived && !value.Derived {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.read_only_property", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("derived property %q must be marked derived", definition.ID), Path: propertyPath,
			})
		}
		count := len(value.Values)
		if count < definition.Lower || definition.Upper >= 0 && count > definition.Upper {
			diagnostics = append(diagnostics, SemanticDiagnostic{
				Code: "semantic.invalid_property_multiplicity", Severity: SemanticSeverityError,
				Message: fmt.Sprintf("property %q has %d values, expected %d..%s", definition.ID, count, definition.Lower, multiplicityUpper(definition.Upper)),
				Path:    propertyPath + ".values",
			})
		}
		for index, item := range value.Values {
			if !metamodelValueMatches(definition.Kind, definition.Type, item) {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_property_value", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("property %q requires %s values, got %q", definition.ID, metamodelValueKind(definition.Kind, definition.Type), item.Kind),
					Path:    indexPath(propertyPath+".values", index),
				})
			}
		}
	}
	return diagnostics
}

// TRLC-LINKS: REQ-EMG-040
func validateExtensionProperties(properties map[string]MetamodelPropertyValue, path string) []SemanticDiagnostic {
	var diagnostics []SemanticDiagnostic
	for name, property := range properties {
		for index, value := range property.Values {
			if !validMetamodelValue(value) {
				diagnostics = append(diagnostics, SemanticDiagnostic{
					Code: "semantic.invalid_extension_value", Severity: SemanticSeverityError,
					Message: fmt.Sprintf("extension property %q has an invalid typed value", name),
					Path:    indexPath(path+".properties."+name+".values", index),
				})
			}
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

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func validMetamodelValue(value MetamodelValue) bool {
	count := 0
	switch value.Kind {
	case "null":
		return value.String == "" && value.Integer == nil && value.Real == nil && value.Boolean == nil &&
			value.Reference == "" && len(value.Object) == 0 && len(value.List) == 0
	case "string":
		count++
	case "integer":
		if value.Integer != nil {
			count++
		}
	case "real":
		if value.Real != nil {
			count++
		}
	case "boolean":
		if value.Boolean != nil {
			count++
		}
	case "reference":
		if strings.TrimSpace(value.Reference) != "" {
			count++
		}
	case "object":
		count++
		for _, child := range value.Object {
			if !validMetamodelValue(child) {
				return false
			}
		}
	case "list":
		count++
		for _, child := range value.List {
			if !validMetamodelValue(child) {
				return false
			}
		}
	default:
		return false
	}
	return count == 1
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func multiplicityUpper(upper int) string {
	if upper < 0 {
		return "*"
	}
	return fmt.Sprint(upper)
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func normalizeElementMetamodelProperties(element *SemanticElement) {
	if element.Metaclass == "" || strings.HasPrefix(element.Metaclass, "Engineering::") {
		return
	}
	addOfficialProperty(element.Metaclass, &element.Properties, "elementId", false, stringMetamodelValue(element.ID))
	if element.Name != "" {
		addOfficialProperty(element.Metaclass, &element.Properties, "declaredName", false, stringMetamodelValue(element.Name))
	}
	if element.Owner != "" {
		addOfficialProperty(element.Metaclass, &element.Properties, "owner", true, referenceMetamodelValue(element.Owner))
	}
	if element.Namespace != "" {
		addOfficialProperty(element.Metaclass, &element.Properties, "owningNamespace", true, referenceMetamodelValue(element.Namespace))
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func normalizeRelationshipMetamodelProperties(relationship *SemanticRelationship) {
	if relationship.Metaclass == "" || strings.HasPrefix(relationship.Metaclass, "Engineering::") {
		return
	}
	addOfficialProperty(relationship.Metaclass, &relationship.Properties, "elementId", false, stringMetamodelValue(relationship.ID))
	addOfficialProperty(relationship.Metaclass, &relationship.Properties, "source", false, referenceMetamodelValue(relationship.Source))
	addOfficialProperty(relationship.Metaclass, &relationship.Properties, "target", false, referenceMetamodelValue(relationship.Target))
	if relationship.Owner != "" {
		addOfficialProperty(relationship.Metaclass, &relationship.Properties, "owningRelatedElement", false, referenceMetamodelValue(relationship.Owner))
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func addOfficialProperty(metaclass string, properties *map[string]MetamodelPropertyValue, name string, derived bool, value MetamodelValue) {
	class, ok := OfficialSysMLMetaclasses[metaclass]
	if !ok {
		return
	}
	definition, ok := class.Properties[name]
	if !ok {
		return
	}
	if *properties == nil {
		*properties = map[string]MetamodelPropertyValue{}
	}
	if _, exists := (*properties)[name]; exists {
		return
	}
	(*properties)[name] = MetamodelPropertyValue{
		Values:  []MetamodelValue{value},
		Derived: derived || definition.Derived,
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
