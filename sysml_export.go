// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package engmodel

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// SysMLExportResult is a deterministic textual projection plus explicit
// projection and conformance diagnostics.
type SysMLExportResult struct {
	Text        string                     `json:"text"`
	Diagnostics []model.SemanticDiagnostic `json:"diagnostics"`
}

type SysMLRendererCoverage struct {
	OfficialMetaclasses int                     `json:"officialMetaclasses"`
	NativeRules         int                     `json:"nativeRules"`
	NonInstantiable     int                     `json:"nonInstantiable"`
	MissingRules        int                     `json:"missingRules"`
	Relationships       int                     `json:"relationships"`
	Entries             []SysMLRendererEvidence `json:"entries"`
}

type SysMLRendererEvidence struct {
	Metaclass   string `json:"metaclass"`
	Status      string `json:"status"`
	Syntax      string `json:"syntax,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}

// GenerateSysMLV2FromFile loads the canonical YAML input and generates a SysML
// v2 textual projection.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-047, REQ-EMG-050
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL, REF-SYSML-V2-TOOLCHAIN
func GenerateSysMLV2FromFile(architecturePath string) (SysMLExportResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return SysMLExportResult{}, err
	}
	var compositionDiagnostics []model.SemanticDiagnostic
	var projection CompositionProjection
	if HasComposition(bundle) {
		composition, err := GenerateCompositionFromFile(bundle.ManifestPath)
		if err != nil {
			return SysMLExportResult{}, err
		}
		for _, diagnostic := range composition.Diagnostics {
			compositionDiagnostics = append(compositionDiagnostics, model.SemanticDiagnostic{
				Code: diagnostic.Code, Severity: model.SemanticSeverity(diagnostic.Severity),
				Message: diagnostic.Message, Path: diagnostic.Path,
			})
		}
		if semanticDiagnosticsHaveErrors(compositionDiagnostics) {
			return SysMLExportResult{Diagnostics: sortSemanticDiagnostics(compositionDiagnostics)}, fmt.Errorf("composition validation failed")
		}
		projection = BuildCompositionProjection(composition)
	}
	result, err := GenerateSysMLV2(bundle)
	if err == nil {
		result.Text = enrichSysMLWithPublications(result.Text, projection)
	}
	result.Diagnostics = sortSemanticDiagnostics(append(result.Diagnostics, compositionDiagnostics...))
	return result, err
}

// enrichSysMLWithPublications emits only concepts with a direct native SysML
// representation. The dependency alias is a package, making each declaration's
// semantic identity alias::ID without embedding payloads or property bags.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func enrichSysMLWithPublications(text string, projection CompositionProjection) string {
	if len(projection.Publications) == 0 {
		return text
	}
	var out strings.Builder
	for _, publication := range projection.Publications {
		var declarations []string
		for _, entity := range publication.Entities {
			keyword := publishedSysMLKeyword(entity)
			if keyword == "" || strings.Contains(text, sysmlName(entity.QualifiedID)) {
				continue
			}
			declarations = append(declarations, fmt.Sprintf("        %s %s;", keyword, sysmlName(entity.SourceID)))
		}
		if len(declarations) == 0 {
			continue
		}
		fmt.Fprintf(&out, "\n    // Publication %s from %s@%s (%s).\n",
			publication.Publication, publication.ModulePath, publication.Version, publication.SourceModel)
		fmt.Fprintf(&out, "    package %s {\n", sysmlName(publication.Alias))
		for _, declaration := range declarations {
			out.WriteString(declaration)
			out.WriteByte('\n')
		}
		fmt.Fprintln(&out, "    }")
	}
	if out.Len() == 0 {
		return text
	}
	index := strings.LastIndex(text, "}")
	if index < 0 {
		return text
	}
	return text[:index] + out.String() + text[index:]
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func publishedSysMLKeyword(entity PublishedProjectionEntity) string {
	switch entity.Domain {
	case "requirements":
		return "requirement def"
	case "behavior":
		switch entity.Value.(type) {
		case model.State:
			return "state def"
		case model.Event, model.Flow:
			return "action def"
		}
	case "architecture":
		switch entity.Value.(type) {
		case model.FunctionalGroup, model.FunctionalUnit, model.Actor, model.ReferencedElement,
			model.DataObject, model.DeploymentTarget, model.HardwareItem, model.ContractEntry:
			return "part def"
		case model.Interface, model.HardwareInterface:
			return "port def"
		}
	}
	return ""
}

// GenerateSysMLV2 validates and projects a Bundle before rendering.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func GenerateSysMLV2(bundle model.Bundle) (SysMLExportResult, error) {
	var diagnostics []model.SemanticDiagnostic
	for _, diagnostic := range validate.Bundle(bundle) {
		diagnostics = append(diagnostics, model.SemanticDiagnostic{
			Code: diagnostic.Code, Severity: model.SemanticSeverity(diagnostic.Severity),
			Message: diagnostic.Message, Path: diagnostic.Path,
		})
	}
	if semanticDiagnosticsHaveErrors(diagnostics) {
		return SysMLExportResult{Diagnostics: sortSemanticDiagnostics(diagnostics)}, fmt.Errorf("validation failed")
	}

	semantic, projectionDiagnostics := model.ProjectSemanticModel(bundle)
	diagnostics = append(diagnostics, projectionDiagnostics...)
	if semanticDiagnosticsHaveErrors(diagnostics) {
		return SysMLExportResult{Diagnostics: sortSemanticDiagnostics(diagnostics)}, fmt.Errorf("semantic projection failed")
	}

	text, renderingDiagnostics := GenerateSysMLV2Projection(semantic)
	diagnostics = append(diagnostics, renderingDiagnostics...)
	if semanticDiagnosticsHaveErrors(diagnostics) {
		return SysMLExportResult{Text: text, Diagnostics: sortSemanticDiagnostics(diagnostics)}, fmt.Errorf("SysML rendering failed")
	}
	return SysMLExportResult{Text: text, Diagnostics: sortSemanticDiagnostics(diagnostics)}, nil
}

// GenerateSysMLV2Projection renders a pre-built semantic graph. Missing
// rendering rules are errors; official concepts are never replaced by a
// generic item declaration.
//
// TRLC-LINKS: REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func GenerateSysMLV2Projection(semantic model.SemanticModel) (string, []model.SemanticDiagnostic) {
	var out bytes.Buffer
	var diagnostics []model.SemanticDiagnostic
	elementIDs := map[string]bool{}
	for _, element := range semantic.Elements {
		elementIDs[element.ID] = true
	}

	fmt.Fprintln(&out, "// Generated deterministic SysML v2 project source.")
	fmt.Fprintln(&out, "// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.")
	fmt.Fprintf(&out, "package %s {\n", sysmlName(nonEmpty(strings.TrimSpace(semantic.ID), "EngineeringModel")))
	fmt.Fprintln(&out, "    private import ScalarValues::*;")
	for _, imported := range semantic.Imports {
		if strings.TrimSpace(imported.Namespace) != "" && imported.Namespace != semantic.ID {
			continue
		}
		writeImport(&out, imported, "    ")
	}
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "    private metadata def EngineeringElement {")
	fmt.Fprintln(&out, "        attribute sourceId : String;")
	fmt.Fprintln(&out, "        attribute conceptKind : String;")
	fmt.Fprintln(&out, "    }")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "    private metadata def EngineeringRelationship :> EngineeringElement {")
	fmt.Fprintln(&out, "        attribute relationshipKind : String;")
	fmt.Fprintln(&out, "    }")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "    private metadata def EngineeringProject :> EngineeringElement;")
	fmt.Fprintln(&out)
	writeNativeSupportTypes(&out)
	fmt.Fprintln(&out, "    @EngineeringProject {")
	fmt.Fprintf(&out, "        sourceId = %s;\n", sysmlString(semantic.ID))
	fmt.Fprintln(&out, "        conceptKind = \"engineering_model\";")
	fmt.Fprintln(&out, "    }")
	fmt.Fprintln(&out, "    attribute 'engineeringModelInterchange';")

	elements := append([]model.SemanticElement(nil), semantic.Elements...)
	children := map[string][]model.SemanticElement{}
	for _, element := range elements {
		if element.Kind == model.ElementPackageDefinition && element.ID == semantic.ID {
			continue
		}
		namespace := strings.TrimSpace(element.Namespace)
		if namespace == "" {
			namespace = semantic.ID
		}
		children[namespace] = append(children[namespace], element)
	}
	externalEndpoints := compositionExternalEndpoints(semantic.Relationships, elementIDs)
	writeNamespaceElements(&out, semantic.ID, "    ", children, semantic.Imports, elementIDs, &diagnostics)
	writeCompositionExternalEndpoints(&out, externalEndpoints, elementIDs)

	relationships := append([]model.SemanticRelationship(nil), semantic.Relationships...)
	for _, relationship := range relationships {
		if relationship.Kind == model.RelationshipContainment {
			continue
		}
		fmt.Fprintln(&out)
		if !elementIDs[relationship.Source] || !elementIDs[relationship.Target] {
			diagnostics = append(diagnostics, model.SemanticDiagnostic{
				Code: "sysml.external_relationship_endpoint", Severity: model.SemanticSeverityError,
				Message: fmt.Sprintf("relationship %q references an endpoint outside the loaded bundle", relationship.ID),
				Path:    "relationships." + relationship.ID,
			})
			continue
		}
		writeRelationshipAnnotation(&out, relationship)
		if !writeNativeRelationship(&out, relationship, "    ") {
			diagnostics = append(diagnostics, model.SemanticDiagnostic{
				Code: "sysml.missing_relationship_rule", Severity: model.SemanticSeverityError,
				Message: fmt.Sprintf("relationship %q has no native rendering rule for %q (%s)", relationship.ID, relationship.Kind, relationship.Metaclass),
				Path:    "relationships." + relationship.ID,
			})
		}
	}
	fmt.Fprintln(&out, "}")

	return out.String(), sortSemanticDiagnostics(diagnostics)
}

type compositionExternalEndpoint struct {
	ID          string
	Requirement bool
}

// compositionExternalEndpoints materializes dependency contract endpoints that
// intentionally live in separately versioned subsystem modules.
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-047, REQ-EMG-050
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, FU-SYSTEM-COMPOSITION
func compositionExternalEndpoints(relationships []model.SemanticRelationship, elementIDs map[string]bool) []compositionExternalEndpoint {
	byID := map[string]compositionExternalEndpoint{}
	for _, relationship := range relationships {
		switch relationship.Kind {
		case model.RelationshipAllocation:
			if !elementIDs[relationship.Source] {
				byID[relationship.Source] = compositionExternalEndpoint{ID: relationship.Source}
			}
			if !elementIDs[relationship.Target] {
				byID[relationship.Target] = compositionExternalEndpoint{ID: relationship.Target}
			}
		case model.RelationshipSatisfaction:
			if !elementIDs[relationship.Source] {
				byID[relationship.Source] = compositionExternalEndpoint{ID: relationship.Source}
			}
			if !elementIDs[relationship.Target] {
				byID[relationship.Target] = compositionExternalEndpoint{ID: relationship.Target, Requirement: true}
			}
		}
	}
	out := make([]compositionExternalEndpoint, 0, len(byID))
	for _, endpoint := range byID {
		out = append(out, endpoint)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// writeCompositionExternalEndpoints emits typed placeholders for contract
// endpoints owned by separately packaged subsystem models.
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-047, REQ-EMG-050
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, FU-SYSTEM-COMPOSITION
func writeCompositionExternalEndpoints(out *bytes.Buffer, endpoints []compositionExternalEndpoint, elementIDs map[string]bool) {
	for _, endpoint := range endpoints {
		fmt.Fprintln(out)
		if endpoint.Requirement {
			fmt.Fprintf(out, "    requirement %s : EngineeringRequirement;\n", sysmlName(endpoint.ID))
		} else {
			fmt.Fprintf(out, "    part %s : EngineeringPart;\n", sysmlName(endpoint.ID))
		}
		elementIDs[endpoint.ID] = true
	}
}

// TRLC-LINKS: REQ-EMG-036
func writeElementAnnotation(out *bytes.Buffer, element model.SemanticElement) {
	fmt.Fprintln(out, "    @EngineeringElement {")
	fmt.Fprintf(out, "        sourceId = %s;\n", sysmlString(element.ID))
	fmt.Fprintf(out, "        conceptKind = %s;\n", sysmlString(string(element.Kind)))
	fmt.Fprintln(out, "    }")
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func writeNamespaceElements(out *bytes.Buffer, namespace, indent string, children map[string][]model.SemanticElement, imports []model.SemanticImport, elementIDs map[string]bool, diagnostics *[]model.SemanticDiagnostic) {
	for _, element := range children[namespace] {
		fmt.Fprintln(out)
		writeElement(out, element, indent, children, imports, elementIDs, diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func writeElement(out *bytes.Buffer, element model.SemanticElement, indent string, children map[string][]model.SemanticElement, imports []model.SemanticImport, elementIDs map[string]bool, diagnostics *[]model.SemanticDiagnostic) {
	writeElementAnnotationAt(out, element, indent)
	keyword, ok := nativeElementKeyword(element)
	if !ok {
		*diagnostics = append(*diagnostics, model.SemanticDiagnostic{
			Code: "sysml.missing_element_rule", Severity: model.SemanticSeverityError,
			Message: fmt.Sprintf("element %q has no native rendering rule for %q (%s)", element.ID, element.Kind, element.Metaclass),
			Path:    "elements." + element.ID,
		})
		return
	}
	if element.Kind == model.ElementPackageDefinition {
		fmt.Fprintf(out, "%spackage %s {\n", indent, sysmlName(element.ID))
		for _, imported := range imports {
			if imported.Namespace == element.ID {
				writeImport(out, imported, indent+"    ")
			}
		}
		writeNamespaceElements(out, element.ID, indent+"    ", children, imports, elementIDs, diagnostics)
		fmt.Fprintf(out, "%s}\n", indent)
		return
	}
	fmt.Fprintf(out, "%s%s ", indent, keyword)
	fmt.Fprint(out, sysmlName(element.ID))
	writeElementTyping(out, element)
	hasChildren := len(children[element.ID]) > 0
	hasBody := len(element.Features) > 0 || hasChildren || needsImpliedRelationshipEnds(element.Metaclass)
	if !hasBody {
		fmt.Fprintln(out, ";")
		return
	}
	fmt.Fprintln(out, " {")
	if needsImpliedRelationshipEnds(element.Metaclass) {
		endKeyword := "item"
		if element.Metaclass == "SysML::InterfaceDefinition" {
			endKeyword = "port"
		}
		fmt.Fprintf(out, "%s    end %s 'source' : Engineering%s;\n", indent, endKeyword, strings.ToUpper(endKeyword[:1])+endKeyword[1:])
		fmt.Fprintf(out, "%s    end %s 'target' : Engineering%s;\n", indent, endKeyword, strings.ToUpper(endKeyword[:1])+endKeyword[1:])
	}
	for _, feature := range element.Features {
		writeFeature(out, feature, indent+"    ")
	}
	writeNamespaceElements(out, element.ID, indent+"    ", children, imports, elementIDs, diagnostics)
	fmt.Fprintf(out, "%s}\n", indent)
}

// TRLC-LINKS: REQ-EMG-036
func writeElementAnnotationAt(out *bytes.Buffer, element model.SemanticElement, indent string) {
	fmt.Fprintf(out, "%s@EngineeringElement {\n", indent)
	fmt.Fprintf(out, "%s    sourceId = %s;\n", indent, sysmlString(element.ID))
	fmt.Fprintf(out, "%s    conceptKind = %s;\n", indent, sysmlString(string(element.Kind)))
	fmt.Fprintf(out, "%s}\n", indent)
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func writeImport(out *bytes.Buffer, imported model.SemanticImport, indent string) {
	visibility := strings.TrimSpace(imported.Visibility)
	if visibility == "" {
		visibility = "private"
	}
	suffix := ""
	if imported.Recursive {
		suffix = "::*"
	}
	fmt.Fprintf(out, "%s%s import %s%s;\n", indent, visibility, sysmlName(imported.Imported), suffix)
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func writeFeature(out *bytes.Buffer, feature model.SemanticFeature, indent string) {
	direction := strings.TrimSpace(feature.Direction)
	if direction != "" {
		fmt.Fprintf(out, "%s%s ", indent, direction)
	} else {
		fmt.Fprint(out, indent)
	}
	switch feature.Kind {
	case model.FeatureParameter:
		fmt.Fprint(out, "item ")
	case model.FeaturePort:
		fmt.Fprint(out, "port ")
	default:
		fmt.Fprint(out, "attribute ")
	}
	fmt.Fprint(out, sysmlName(feature.Name))
	if strings.TrimSpace(feature.Type) != "" && isNativeTypeReference(feature.Type) {
		fmt.Fprint(out, " : ")
		if feature.Conjugated && feature.Kind == model.FeaturePort {
			fmt.Fprint(out, "~")
		}
		fmt.Fprint(out, sysmlName(feature.Type))
	} else if feature.Kind == model.FeatureParameter || feature.Kind == model.FeaturePort {
		fmt.Fprint(out, " : ")
		if feature.Conjugated && feature.Kind == model.FeaturePort {
			fmt.Fprint(out, "~")
		}
		fmt.Fprint(out, defaultFeatureType(feature.Kind))
	}
	writeMultiplicity(out, feature.Multiplicity, feature.Ordered, feature.Unique)
	for _, specialized := range feature.Specializes {
		fmt.Fprintf(out, " :> %s", sysmlName(specialized))
	}
	for _, subsetted := range feature.Subsets {
		fmt.Fprintf(out, " :> %s", sysmlName(subsetted))
	}
	for _, redefined := range feature.Redefines {
		fmt.Fprintf(out, " :>> %s", sysmlName(redefined))
	}
	if feature.Value != nil {
		fmt.Fprintf(out, " = %s", sysmlExpression(*feature.Value))
	}
	fmt.Fprintln(out, ";")
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func writeMultiplicity(out *bytes.Buffer, multiplicity *model.Multiplicity, ordered bool, unique *bool) {
	if multiplicity != nil {
		lower := "0"
		upper := "*"
		if multiplicity.Lower != nil {
			lower = fmt.Sprintf("%d", *multiplicity.Lower)
		}
		if multiplicity.Upper != nil && *multiplicity.Upper >= 0 {
			upper = fmt.Sprintf("%d", *multiplicity.Upper)
		}
		if lower == upper {
			fmt.Fprintf(out, " [%s]", lower)
		} else {
			fmt.Fprintf(out, " [%s..%s]", lower, upper)
		}
	}
	if ordered {
		fmt.Fprint(out, " ordered")
	}
	if unique != nil && !*unique {
		fmt.Fprint(out, " nonunique")
	}
}

// TRLC-LINKS: REQ-EMG-036
func sysmlExpression(expression model.SemanticExpression) string {
	if expression.Operator == "" || len(expression.Operands) == 0 {
		if expression.TypedValue != nil {
			return sysmlValue(*expression.TypedValue)
		}
		if expression.Language == "literal" || expression.Language == "" {
			return sysmlString(expression.Value)
		}
		return sysmlString(expression.Language + ":" + expression.Value)
	}
	operands := make([]string, 0, len(expression.Operands))
	for _, operand := range expression.Operands {
		operands = append(operands, sysmlExpression(operand))
	}
	return "(" + strings.Join(operands, " "+expression.Operator+" ") + ")"
}

// TRLC-LINKS: REQ-EMG-036
func sysmlValue(value model.SemanticValue) string {
	switch {
	case value.String != "":
		return sysmlString(value.String)
	case value.Integer != nil:
		return fmt.Sprintf("%d", *value.Integer)
	case value.Real != nil:
		return fmt.Sprintf("%g", *value.Real)
	case value.Boolean != nil:
		return fmt.Sprintf("%t", *value.Boolean)
	case value.Reference != "":
		return sysmlName(value.Reference)
	case value.Quantity != nil:
		return fmt.Sprintf("%g [%s]", value.Quantity.Value, sysmlName(value.Quantity.Unit))
	default:
		return "null"
	}
}

// TRLC-LINKS: REQ-EMG-036
func writeRelationshipAnnotation(out *bytes.Buffer, relationship model.SemanticRelationship) {
	fmt.Fprintln(out, "    @EngineeringRelationship {")
	fmt.Fprintf(out, "        sourceId = %s;\n", sysmlString(relationship.ID))
	fmt.Fprintf(out, "        conceptKind = %s;\n", sysmlString(string(relationship.Kind)))
	fmt.Fprintf(out, "        relationshipKind = %s;\n", sysmlString(relationship.SourceType))
	fmt.Fprintln(out, "    }")
}

var nativeRelationshipWriters = map[model.SemanticRelationshipKind]func(*bytes.Buffer, model.SemanticRelationship, string){
	model.RelationshipDependency: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		fmt.Fprintf(out, "%sdependency %s from %s to %s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Source), sysmlName(relationship.Target))
	},
	model.RelationshipConnection: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		fmt.Fprintf(out, "%spart def %s {\n", indent, sysmlName(relationship.ID+"-context"))
		fmt.Fprintf(out, "%s    part source : %s;\n", indent, sysmlName(relationship.Source))
		fmt.Fprintf(out, "%s    part target : %s;\n", indent, sysmlName(relationship.Target))
		fmt.Fprintf(out, "%s    connection %s connect source to target;\n", indent, sysmlName(relationship.ID))
		fmt.Fprintf(out, "%s}\n", indent)
	},
	model.RelationshipBinding: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		fmt.Fprintf(out, "%sbinding %s bind %s = %s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Source), sysmlName(relationship.Target))
	},
	model.RelationshipAllocation: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		fmt.Fprintf(out, "%spart def %s {\n", indent, sysmlName(relationship.ID+"-context"))
		fmt.Fprintf(out, "%s    part %s : EngineeringPart;\n", indent, sysmlName(relationship.Source))
		fmt.Fprintf(out, "%s    part %s : EngineeringPart;\n", indent, sysmlName(relationship.Target))
		fmt.Fprintf(out, "%s    allocation %s allocate %s to %s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Source), sysmlName(relationship.Target))
		fmt.Fprintf(out, "%s}\n", indent)
	},
	model.RelationshipSuccession: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		fmt.Fprintf(out, "%ssuccession %s first %s then %s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Source), sysmlName(relationship.Target))
	},
	model.RelationshipTransfer: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		fmt.Fprintf(out, "%spart def %s {\n", indent, sysmlName(relationship.ID+"-context"))
		if relationship.Source == relationship.Target {
			fmt.Fprintf(out, "%s    part participant : EngineeringPart { port pOut : EngineeringPort; port pIn : EngineeringPort; }\n", indent)
			fmt.Fprintf(out, "%s    flow %s", indent, sysmlName(relationship.ID))
			if relationship.ItemRef != "" {
				fmt.Fprintf(out, " of %s", sysmlName(relationship.ItemRef))
			}
			fmt.Fprintln(out, " from participant.pOut to participant.pIn;")
			fmt.Fprintf(out, "%s}\n", indent)
			return
		}
		fmt.Fprintf(out, "%s    part %s : EngineeringPart { port pOut : EngineeringPort; }\n", indent, sysmlName(relationship.Source))
		fmt.Fprintf(out, "%s    part %s : EngineeringPart { port pIn : EngineeringPort; }\n", indent, sysmlName(relationship.Target))
		fmt.Fprintf(out, "%s    flow %s", indent, sysmlName(relationship.ID))
		if relationship.ItemRef != "" {
			fmt.Fprintf(out, " of %s", sysmlName(relationship.ItemRef))
		}
		fmt.Fprintf(out, " from %s.pOut to %s.pIn;\n", sysmlName(relationship.Source), sysmlName(relationship.Target))
		fmt.Fprintf(out, "%s}\n", indent)
	},
	model.RelationshipSatisfaction: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		fmt.Fprintf(out, "%srequirement def %s {\n", indent, sysmlName(relationship.ID+"-context"))
		fmt.Fprintf(out, "%s    part satisfyingPart : EngineeringPart;\n", indent)
		fmt.Fprintf(out, "%s    satisfy requirement targetRequirement : EngineeringRequirement by satisfyingPart;\n", indent)
		fmt.Fprintf(out, "%s}\n", indent)
	},
	model.RelationshipVerification: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		fmt.Fprintf(out, "%sverification %s : EngineeringVerification {\n", indent, sysmlName(relationship.ID))
		fmt.Fprintf(out, "%s    subject %s : EngineeringItem;\n", indent, sysmlName(relationship.Source))
		fmt.Fprintf(out, "%s    objective %s : EngineeringRequirement;\n", indent, sysmlName(relationship.Target))
		fmt.Fprintf(out, "%s}\n", indent)
	},
	model.RelationshipTransition: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		writeTransitionRelationship(out, relationship, indent, "", "")
	},
	model.RelationshipEventTrigger: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		writeTransitionRelationship(out, relationship, indent, "accept trigger ", "")
	},
	model.RelationshipGuard: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		writeTransitionRelationship(out, relationship, indent, "if "+relationshipExpression(relationship.Guard)+" ", "")
	},
	model.RelationshipEffect: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		writeTransitionRelationship(out, relationship, indent, "", " do "+relationshipExpression(relationship.Effect))
	},
	model.RelationshipExtension: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		fmt.Fprintf(out, "%sdependency %s from %s to %s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Source), sysmlName(relationship.Target))
	},
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func writeNativeRelationship(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) bool {
	if relationship.Kind == model.RelationshipReference {
		fmt.Fprintf(out, "%sref item %s subsets %s = %s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Target), sysmlName(relationship.Source))
		return true
	}
	if relationship.Kind == model.RelationshipTyping ||
		relationship.Kind == model.RelationshipSpecialization ||
		relationship.Kind == model.RelationshipSubsetting ||
		relationship.Kind == model.RelationshipRedefinition ||
		relationship.Kind == model.RelationshipVariant {
		fmt.Fprintf(out, "%sdependency %s from %s to %s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Source), sysmlName(relationship.Target))
		return true
	}
	writer, ok := nativeRelationshipWriters[relationship.Kind]
	if !ok {
		return false
	}
	writer(out, relationship, indent)
	return true
}

// TRLC-LINKS: REQ-EMG-039
func writeTransitionRelationship(out *bytes.Buffer, relationship model.SemanticRelationship, indent, middle, effect string) {
	fmt.Fprintf(out, "%sstate %s : EngineeringState {\n", indent, sysmlName(relationship.ID+"-context"))
	if strings.HasPrefix(middle, "accept") {
		fmt.Fprintf(out, "%s    state %s : EngineeringState;\n", indent, sysmlName(relationship.Source))
		fmt.Fprintf(out, "%s    action transitionEffect : EngineeringAction;\n", indent)
		fmt.Fprintf(out, "%s    transition %s first %s accept %s : EngineeringItem then transitionEffect;\n",
			indent, sysmlName(relationship.ID), sysmlName(relationship.Source), sysmlName(relationship.Target))
		fmt.Fprintf(out, "%s}\n", indent)
		return
	}
	fmt.Fprintf(out, "%s    action %s : EngineeringAction;\n", indent, sysmlName(relationship.Source))
	fmt.Fprintf(out, "%s    action %s : EngineeringAction;\n", indent, sysmlName(relationship.Target))
	fmt.Fprintf(out, "%s    transition %s first %s %sthen %s%s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Source), middle, sysmlName(relationship.Target), effect)
	fmt.Fprintf(out, "%s}\n", indent)
}

// TRLC-LINKS: REQ-EMG-039
func relationshipExpression(expression *model.SemanticExpression) string {
	if expression == nil {
		return "true"
	}
	return sysmlExpression(*expression)
}

// TRLC-LINKS: REQ-EMG-039
func sysMLRendererCoverage() SysMLRendererCoverage {
	coverage := SysMLRendererCoverage{OfficialMetaclasses: len(model.OfficialSysMLMetaclasses)}
	for id, metaclass := range model.OfficialSysMLMetaclasses {
		if syntax, ok := officialNativeMetaclassKeywords[id]; ok {
			coverage.NativeRules++
			coverage.Entries = append(coverage.Entries, SysMLRendererEvidence{
				Metaclass: id, Status: "native", Syntax: syntax,
				Explanation: "Rendered by the table-driven canonical SysML textual dispatcher.",
			})
		} else if metaclass.Abstract || isSemanticOnlyMetaclass(id, metaclass) {
			coverage.NonInstantiable++
			reason := "No standalone concrete textual production; produced implicitly by native ownership, typing, membership, expression, or relationship syntax."
			if metaclass.Abstract {
				reason = "Abstract metaclass; represented only through a concrete native subtype."
			}
			coverage.Entries = append(coverage.Entries, SysMLRendererEvidence{
				Metaclass: id, Status: "non-instantiable", Explanation: reason,
			})
		} else {
			coverage.MissingRules++
			coverage.Entries = append(coverage.Entries, SysMLRendererEvidence{
				Metaclass: id, Status: "missing",
				Explanation: "Concrete canonical instance has no native textual rendering rule.",
			})
		}
		if metaclass.Relationship {
			coverage.Relationships++
		}
	}
	sort.Slice(coverage.Entries, func(i, j int) bool {
		return coverage.Entries[i].Metaclass < coverage.Entries[j].Metaclass
	})
	return coverage
}

var officialNativeMetaclassKeywords = func() map[string]string {
	out := make(map[string]string, len(nativeElementKeywords)+len(nativeRelationshipWriters)+6)
	for kind, keyword := range nativeElementKeywords {
		if metaclass := model.OfficialSysMLSemanticAliases[string(kind)]; metaclass != "" {
			if current, exists := out[metaclass]; !exists || keyword < current {
				out[metaclass] = keyword
			}
		}
	}
	for kind := range nativeRelationshipWriters {
		if metaclass := model.OfficialSysMLSemanticAliases[string(kind)]; metaclass != "" {
			syntax := string(kind)
			if current, exists := out[metaclass]; !exists || syntax < current {
				out[metaclass] = syntax
			}
		}
	}
	for _, kind := range []model.SemanticRelationshipKind{
		model.RelationshipContainment, model.RelationshipReference, model.RelationshipTyping,
		model.RelationshipSpecialization, model.RelationshipSubsetting, model.RelationshipRedefinition,
		model.RelationshipVariant,
	} {
		if metaclass := model.OfficialSysMLSemanticAliases[string(kind)]; metaclass != "" {
			syntax := string(kind)
			if current, exists := out[metaclass]; !exists || syntax < current {
				out[metaclass] = syntax
			}
		}
	}
	return out
}()

// TRLC-LINKS: REQ-EMG-039
func isSemanticOnlyMetaclass(id string, metaclass model.OfficialSysMLMetaclass) bool {
	if metaclass.Abstract {
		return true
	}
	for alias, official := range model.OfficialSysMLSemanticAliases {
		if official == id && !strings.HasPrefix(alias, "official:") {
			return false
		}
	}
	return true
}

// TRLC-LINKS: REQ-EMG-036
var nativeElementKeywords = map[model.SemanticElementKind]string{
	model.ElementPackageDefinition:          "package",
	model.ElementPartDefinition:             "part def",
	model.ElementPartUsage:                  "part",
	model.ElementItemDefinition:             "item def",
	model.ElementItemUsage:                  "item",
	model.ElementAttributeDefinition:        "attribute def",
	model.ElementAttributeUsage:             "attribute",
	model.ElementPortDefinition:             "port def",
	model.ElementPortUsage:                  "port",
	model.ElementInterfaceDefinition:        "interface def",
	model.ElementInterfaceUsage:             "interface",
	model.ElementConnectionDefinition:       "connection def",
	model.ElementConnectionUsage:            "connection",
	model.ElementActionDefinition:           "action def",
	model.ElementActionUsage:                "action",
	model.ElementControlNode:                "action",
	model.ElementStateDefinition:            "state def",
	model.ElementStateUsage:                 "state",
	model.ElementEventDefinition:            "event occurrence",
	model.ElementEventUsage:                 "event occurrence",
	model.ElementViewDefinition:             "view def",
	model.ElementViewUsage:                  "view",
	model.ElementViewpointDefinition:        "viewpoint def",
	model.ElementViewpointUsage:             "viewpoint",
	model.ElementRequirementDefinition:      "requirement def",
	model.ElementRequirementUsage:           "requirement",
	model.ElementConcernDefinition:          "concern def",
	model.ElementConcernUsage:               "concern",
	model.ElementStakeholderDefinition:      "item def",
	model.ElementStakeholderUsage:           "item",
	model.ElementConstraintDefinition:       "constraint def",
	model.ElementConstraintUsage:            "constraint",
	model.ElementCalculationDefinition:      "calc def",
	model.ElementCalculationUsage:           "calc",
	model.ElementCaseDefinition:             "case def",
	model.ElementCaseUsage:                  "case",
	model.ElementAnalysisCaseDefinition:     "analysis def",
	model.ElementAnalysisCaseUsage:          "analysis",
	model.ElementVerificationCaseDefinition: "verification def",
	model.ElementVerificationCaseUsage:      "verification",
	model.ElementUseCaseDefinition:          "use case def",
	model.ElementUseCaseUsage:               "use case",
	model.ElementQuantityDefinition:         "attribute def",
	model.ElementUnitDefinition:             "attribute def",
	model.ElementOccurrenceDefinition:       "occurrence def",
	model.ElementOccurrenceUsage:            "occurrence",
	model.ElementIndividualDefinition:       "individual def",
	model.ElementIndividualUsage:            "individual",
	model.ElementSnapshot:                   "occurrence",
	model.ElementTimeSlice:                  "occurrence",
	model.ElementExtension:                  "attribute",
}

// TRLC-LINKS: REQ-EMG-039
func nativeElementKeyword(element model.SemanticElement) (string, bool) {
	keyword, ok := nativeElementKeywords[element.Kind]
	if !ok {
		return "", false
	}
	if element.Metaclass == "" || strings.HasPrefix(element.Metaclass, "Engineering::") {
		return keyword, true
	}
	official := model.OfficialSysMLSemanticAliases[string(element.Kind)]
	return keyword, official != "" && official == element.Metaclass
}

// TRLC-LINKS: REQ-EMG-039
func writeElementTyping(out *bytes.Buffer, element model.SemanticElement) {
	if element.TypeRef != "" && element.Kind != model.ElementActionUsage && element.Kind != model.ElementControlNode {
		fmt.Fprint(out, " : ")
		if element.Conjugated && element.Kind == model.ElementPortUsage {
			fmt.Fprint(out, "~")
		}
		fmt.Fprint(out, sysmlName(element.TypeRef))
	} else if defaultType := defaultElementType(element.Kind); defaultType != "" {
		fmt.Fprint(out, " : ")
		if element.Conjugated && element.Kind == model.ElementPortUsage {
			fmt.Fprint(out, "~")
		}
		fmt.Fprint(out, defaultType)
	}
	writeMultiplicity(out, element.Multiplicity, element.Ordered, element.Unique)
	for _, specialized := range element.Specializes {
		fmt.Fprintf(out, " :> %s", sysmlName(specialized))
	}
	for _, subsetted := range element.Subsets {
		fmt.Fprintf(out, " :> %s", sysmlName(subsetted))
	}
	for _, redefined := range element.Redefines {
		fmt.Fprintf(out, " :>> %s", sysmlName(redefined))
	}
}

// TRLC-LINKS: REQ-EMG-039
func needsImpliedRelationshipEnds(metaclass string) bool {
	switch metaclass {
	case "SysML::InterfaceDefinition", "SysML::ConnectionDefinition":
		return true
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-039
func writeNativeSupportTypes(out *bytes.Buffer) {
	fmt.Fprintln(out, "    private item def EngineeringItem;")
	fmt.Fprintln(out, "    private part def EngineeringPart;")
	fmt.Fprintln(out, "    private port def EngineeringPort;")
	fmt.Fprintln(out, "    private action def EngineeringAction;")
	fmt.Fprintln(out, "    private action def EngineeringEvent;")
	fmt.Fprintln(out, "    private state def EngineeringState;")
	fmt.Fprintln(out, "    private requirement def EngineeringRequirement;")
	fmt.Fprintln(out, "    private concern def EngineeringConcern;")
	fmt.Fprintln(out, "    private constraint def EngineeringConstraint;")
	fmt.Fprintln(out, "    private calc def EngineeringCalculation;")
	fmt.Fprintln(out, "    private analysis def EngineeringAnalysis;")
	fmt.Fprintln(out, "    private verification def EngineeringVerification;")
	fmt.Fprintln(out, "    private use case def EngineeringUseCase;")
	fmt.Fprintln(out, "    private view def EngineeringView;")
	fmt.Fprintln(out, "    private viewpoint def EngineeringViewpoint;")
	fmt.Fprintln(out, "    private occurrence def EngineeringOccurrence;")
	fmt.Fprintln(out)
}

// TRLC-LINKS: REQ-EMG-039
func defaultElementType(kind model.SemanticElementKind) string {
	defaults := map[model.SemanticElementKind]string{
		model.ElementPartUsage:             "EngineeringPart",
		model.ElementItemUsage:             "EngineeringItem",
		model.ElementPortUsage:             "EngineeringPort",
		model.ElementActionUsage:           "EngineeringAction",
		model.ElementControlNode:           "EngineeringAction",
		model.ElementStateUsage:            "EngineeringState",
		model.ElementEventDefinition:       "EngineeringEvent",
		model.ElementEventUsage:            "EngineeringEvent",
		model.ElementRequirementUsage:      "EngineeringRequirement",
		model.ElementConcernUsage:          "EngineeringConcern",
		model.ElementConstraintUsage:       "EngineeringConstraint",
		model.ElementCalculationUsage:      "EngineeringCalculation",
		model.ElementAnalysisCaseUsage:     "EngineeringAnalysis",
		model.ElementVerificationCaseUsage: "EngineeringVerification",
		model.ElementUseCaseUsage:          "EngineeringUseCase",
		model.ElementViewUsage:             "EngineeringView",
		model.ElementViewpointUsage:        "EngineeringViewpoint",
		model.ElementOccurrenceUsage:       "EngineeringOccurrence",
		model.ElementIndividualUsage:       "EngineeringOccurrence",
		model.ElementSnapshot:              "EngineeringOccurrence",
		model.ElementTimeSlice:             "EngineeringOccurrence",
	}
	return defaults[kind]
}

// TRLC-LINKS: REQ-EMG-039
func defaultFeatureType(kind model.SemanticFeatureKind) string {
	if kind == model.FeaturePort {
		return "EngineeringPort"
	}
	return "EngineeringItem"
}

// TRLC-LINKS: REQ-EMG-039
func isNativeTypeReference(value string) bool {
	value = strings.TrimSpace(value)
	if value == "String" || value == "Boolean" || value == "Integer" || value == "Real" {
		return true
	}
	return value != "" && value[0] >= 'A' && value[0] <= 'Z'
}

// TRLC-LINKS: REQ-EMG-036
func knownRelationshipKind(kind model.SemanticRelationshipKind) bool {
	switch kind {
	case model.RelationshipContainment, model.RelationshipDependency, model.RelationshipTyping,
		model.RelationshipSpecialization, model.RelationshipSubsetting, model.RelationshipRedefinition,
		model.RelationshipConnection, model.RelationshipBinding, model.RelationshipTransfer,
		model.RelationshipSuccession, model.RelationshipTransition, model.RelationshipEventTrigger,
		model.RelationshipGuard, model.RelationshipEffect, model.RelationshipAllocation,
		model.RelationshipSatisfaction, model.RelationshipVerification, model.RelationshipVariant,
		model.RelationshipReference, model.RelationshipExtension:
		return true
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-036
func sysmlName(value string) string {
	return "'" + strings.ReplaceAll(strings.TrimSpace(value), "'", "''") + "'"
}

// TRLC-LINKS: REQ-EMG-036
func sysmlString(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n", "\r", "\\r", "\t", "\\t")
	return "\"" + replacer.Replace(value) + "\""
}

// TRLC-LINKS: REQ-EMG-036
func semanticDiagnosticsHaveErrors(diagnostics []model.SemanticDiagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == model.SemanticSeverityError {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-036
func sortSemanticDiagnostics(in []model.SemanticDiagnostic) []model.SemanticDiagnostic {
	out := append([]model.SemanticDiagnostic(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Message < out[j].Message
	})
	return out
}
