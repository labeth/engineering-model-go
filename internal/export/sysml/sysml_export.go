// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package sysml

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ExportResult is a deterministic textual projection plus explicit
// projection and conformance diagnostics.
type ExportResult struct {
	Text        string                     `json:"text"`
	Diagnostics []model.SemanticDiagnostic `json:"diagnostics"`
	// Validation holds the validation diagnostics of the run with their
	// remedies; Diagnostics holds the same findings without the remedy.
	Validation []validate.Diagnostic `json:"-"`
}

// PrintableDiagnostics returns Diagnostics as validate diagnostics, each with
// the remedy of the validation diagnostic of the same code, path and message,
// so that a CLI prints the "fix:" line of a required-switch error.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-168
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML
func (r ExportResult) PrintableDiagnostics() []validate.Diagnostic {
	remedies := map[[3]string]string{}
	for _, v := range r.Validation {
		if v.Remedy != "" {
			remedies[[3]string{v.Code, v.Path, v.Message}] = v.Remedy
		}
	}
	out := make([]validate.Diagnostic, 0, len(r.Diagnostics))
	for _, d := range r.Diagnostics {
		out = append(out, validate.Diagnostic{
			Code: d.Code, Severity: validate.Severity(d.Severity), Message: d.Message, Path: d.Path,
			Remedy: remedies[[3]string{d.Code, d.Path, d.Message}],
		})
	}
	return out
}

type RendererCoverage struct {
	OfficialMetaclasses int                `json:"officialMetaclasses"`
	NativeRules         int                `json:"nativeRules"`
	NonInstantiable     int                `json:"nonInstantiable"`
	MissingRules        int                `json:"missingRules"`
	Relationships       int                `json:"relationships"`
	Entries             []RendererEvidence `json:"entries"`
}

type RendererEvidence struct {
	Metaclass   string `json:"metaclass"`
	Status      string `json:"status"`
	Syntax      string `json:"syntax,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}

// GenerateV2FromFile loads the canonical YAML input and generates a SysML
// v2 textual projection.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-047, REQ-EMG-050
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL, REF-SYSML-V2-TOOLCHAIN
func GenerateV2FromFile(architecturePath string) (ExportResult, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return ExportResult{}, err
	}
	var compositionDiagnostics []model.SemanticDiagnostic
	var compositionValidation []validate.Diagnostic
	var projection composition.CompositionProjection
	if composition.HasComposition(bundle) {
		composed, err := composition.GenerateCompositionFromFile(bundle.ManifestPath)
		if err != nil {
			return ExportResult{}, err
		}
		compositionValidation = composed.Diagnostics
		for _, diagnostic := range composed.Diagnostics {
			compositionDiagnostics = append(compositionDiagnostics, model.SemanticDiagnostic{
				Code: diagnostic.Code, Severity: model.SemanticSeverity(diagnostic.Severity),
				Message: diagnostic.Message, Path: diagnostic.Path,
			})
		}
		if semanticDiagnosticsHaveErrors(compositionDiagnostics) {
			return ExportResult{Diagnostics: sortSemanticDiagnostics(compositionDiagnostics), Validation: compositionValidation}, fmt.Errorf("composition validation failed")
		}
		projection = composition.BuildCompositionProjection(composed)
	}
	result, err := GenerateV2(bundle)
	if err == nil {
		result.Text = enrichSysMLWithPublications(result.Text, projection)
	}
	result.Diagnostics = sortSemanticDiagnostics(append(result.Diagnostics, compositionDiagnostics...))
	result.Validation = append(result.Validation, compositionValidation...)
	return result, err
}

// enrichSysMLWithPublications emits only concepts with a direct native SysML
// representation. The dependency alias is a package, making each declaration's
// semantic identity alias::ID without embedding payloads or property bags.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func enrichSysMLWithPublications(text string, projection composition.CompositionProjection) string {
	if len(projection.Publications) == 0 {
		return text
	}
	var out strings.Builder
	for _, publication := range projection.Publications {
		declarations := publicationDeclarations(text, publication)
		if len(declarations) == 0 {
			continue
		}
		writePublicationPackage(&out, publication, declarations)
	}
	index := strings.LastIndex(text, "}")
	if out.Len() == 0 || index < 0 {
		return text
	}
	return text[:index] + out.String() + text[index:]
}

// publicationDeclarations lists the native declarations of the publication
// entities that the text does not declare yet.
//
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func publicationDeclarations(text string, publication composition.PublishedProjection) []string {
	var declarations []string
	for _, entity := range publication.Entities {
		keyword := publishedSysMLKeyword(entity)
		if keyword == "" || strings.Contains(text, sysmlName(entity.QualifiedID)) {
			continue
		}
		declarations = append(declarations, fmt.Sprintf("        %s %s;", keyword, sysmlName(entity.SourceID)))
	}
	return declarations
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func writePublicationPackage(out *strings.Builder, publication composition.PublishedProjection, declarations []string) {
	fmt.Fprintf(out, "\n    // Publication %s from %s@%s (%s).\n",
		publication.Publication, publication.ModulePath, publication.Version, publication.SourceModel)
	fmt.Fprintf(out, "    package %s {\n", sysmlName(publication.Alias))
	for _, declaration := range declarations {
		out.WriteString(declaration)
		out.WriteByte('\n')
	}
	fmt.Fprintln(out, "    }")
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func publishedSysMLKeyword(entity composition.PublishedProjectionEntity) string {
	switch entity.Domain {
	case "requirements":
		return "requirement def"
	case "behavior":
		return behaviorSysMLKeyword(entity.Value)
	case "architecture":
		return architectureSysMLKeyword(entity.Value)
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func behaviorSysMLKeyword(value any) string {
	switch value.(type) {
	case model.State:
		return "state def"
	case model.Event, model.Flow:
		return "action def"
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func architectureSysMLKeyword(value any) string {
	switch value.(type) {
	case model.FunctionalGroup, model.FunctionalUnit, model.Actor, model.ReferencedElement,
		model.DataObject, model.DeploymentTarget, model.HardwareItem, model.ContractEntry:
		return "part def"
	case model.Interface, model.HardwareInterface:
		return "port def"
	}
	return ""
}

// GenerateV2 validates and projects a Bundle before rendering.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func GenerateV2(bundle model.Bundle) (ExportResult, error) {
	var diagnostics []model.SemanticDiagnostic
	validation := validate.Bundle(bundle)
	for _, diagnostic := range validation {
		diagnostics = append(diagnostics, model.SemanticDiagnostic{
			Code: diagnostic.Code, Severity: model.SemanticSeverity(diagnostic.Severity),
			Message: diagnostic.Message, Path: diagnostic.Path,
		})
	}
	if semanticDiagnosticsHaveErrors(diagnostics) {
		return ExportResult{Diagnostics: sortSemanticDiagnostics(diagnostics), Validation: validation}, fmt.Errorf("validation failed")
	}

	semantic, projectionDiagnostics := model.ProjectSemanticModel(bundle)
	diagnostics = append(diagnostics, projectionDiagnostics...)
	if semanticDiagnosticsHaveErrors(diagnostics) {
		return ExportResult{Diagnostics: sortSemanticDiagnostics(diagnostics), Validation: validation}, fmt.Errorf("semantic projection failed")
	}

	text, renderingDiagnostics := GenerateV2Projection(semantic)
	diagnostics = append(diagnostics, renderingDiagnostics...)
	if semanticDiagnosticsHaveErrors(diagnostics) {
		return ExportResult{Text: text, Diagnostics: sortSemanticDiagnostics(diagnostics), Validation: validation}, fmt.Errorf("SysML rendering failed")
	}
	return ExportResult{Text: text, Diagnostics: sortSemanticDiagnostics(diagnostics), Validation: validation}, nil
}

// GenerateV2Projection renders a pre-built semantic graph. Missing
// rendering rules are errors; official concepts are never replaced by a
// generic item declaration. Behavior machines are rendered as one state
// definition each; see writeMachine.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-137
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func GenerateV2Projection(semantic model.SemanticModel) (string, []model.SemanticDiagnostic) {
	var out bytes.Buffer
	var diagnostics []model.SemanticDiagnostic
	elementIDs := map[string]bool{}
	for _, element := range semantic.Elements {
		elementIDs[element.ID] = true
	}

	writeProjectionHeader(&out, semantic)
	machines := indexMachines(semantic)
	externalEndpoints := compositionExternalEndpoints(semantic.Relationships, elementIDs)
	w := &sysmlWriter{
		out: &out, children: groupNamespaceChildren(semantic, machines), imports: semantic.Imports,
		elementIDs: elementIDs, machines: machines, diagnostics: &diagnostics,
	}
	w.namespaceElements(semantic.ID, "    ")
	writeCompositionExternalEndpoints(&out, externalEndpoints, elementIDs)
	for _, relationship := range semantic.Relationships {
		w.relationship(relationship)
	}
	fmt.Fprintln(&out, "}")

	return out.String(), sortSemanticDiagnostics(diagnostics)
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-137
func writeProjectionHeader(out *bytes.Buffer, semantic model.SemanticModel) {
	fmt.Fprintln(out, "// Generated deterministic SysML v2 project source.")
	fmt.Fprintln(out, "// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.")
	fmt.Fprintf(out, "package %s {\n", sysmlName(textutil.NonEmpty(strings.TrimSpace(semantic.ID), "EngineeringModel")))
	fmt.Fprintln(out, "    private import ScalarValues::*;")
	for _, imported := range semantic.Imports {
		if strings.TrimSpace(imported.Namespace) != "" && imported.Namespace != semantic.ID {
			continue
		}
		writeImport(out, imported, "    ")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "    private metadata def EngineeringElement {")
	fmt.Fprintln(out, "        attribute sourceId : String;")
	fmt.Fprintln(out, "        attribute conceptKind : String;")
	fmt.Fprintln(out, "    }")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "    private metadata def EngineeringRelationship :> EngineeringElement {")
	fmt.Fprintln(out, "        attribute relationshipKind : String;")
	fmt.Fprintln(out, "    }")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "    private metadata def EngineeringProject :> EngineeringElement;")
	fmt.Fprintln(out)
	writeNativeSupportTypes(out)
	fmt.Fprintln(out, "    @EngineeringProject {")
	fmt.Fprintf(out, "        sourceId = %s;\n", sysmlString(semantic.ID))
	fmt.Fprintln(out, "        conceptKind = \"engineering_model\";")
	fmt.Fprintln(out, "    }")
	fmt.Fprintln(out, "    attribute 'engineeringModelInterchange';")
}

// groupNamespaceChildren groups the elements the generic writer renders by
// their namespace; the model package itself and the parts a machine owns are
// left out.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-137
func groupNamespaceChildren(semantic model.SemanticModel, machines *machineIndex) map[string][]model.SemanticElement {
	children := map[string][]model.SemanticElement{}
	for _, element := range semantic.Elements {
		if element.Kind == model.ElementPackageDefinition && element.ID == semantic.ID {
			continue
		}
		if machines.ownsElement(element.ID) {
			continue
		}
		namespace := strings.TrimSpace(element.Namespace)
		if namespace == "" {
			namespace = semantic.ID
		}
		children[namespace] = append(children[namespace], element)
	}
	return children
}

// sysmlWriter renders the namespace tree of one semantic model into out and
// collects the diagnostics of the constructs it cannot render.
type sysmlWriter struct {
	out         *bytes.Buffer
	children    map[string][]model.SemanticElement
	imports     []model.SemanticImport
	elementIDs  map[string]bool
	machines    *machineIndex
	diagnostics *[]model.SemanticDiagnostic
}

func (w *sysmlWriter) diagnoseError(code, message, path string) {
	*w.diagnostics = append(*w.diagnostics, model.SemanticDiagnostic{
		Code: code, Severity: model.SemanticSeverityError, Message: message, Path: path,
	})
}

// relationship writes one top-level relationship unless it is a containment
// or a machine owns it.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-039, REQ-EMG-137
func (w *sysmlWriter) relationship(relationship model.SemanticRelationship) {
	if relationship.Kind == model.RelationshipContainment || w.machines.ownsRelationship(relationship.ID) {
		return
	}
	fmt.Fprintln(w.out)
	path := "relationships." + relationship.ID
	if !w.elementIDs[relationship.Source] || !w.elementIDs[relationship.Target] {
		w.diagnoseError("sysml.external_relationship_endpoint",
			fmt.Sprintf("relationship %q references an endpoint outside the loaded bundle", relationship.ID), path)
		return
	}
	writeRelationshipAnnotation(w.out, relationship)
	if !writeNativeRelationship(w.out, relationship, "    ") {
		w.diagnoseError("sysml.missing_relationship_rule",
			fmt.Sprintf("relationship %q has no native rendering rule for %q (%s)", relationship.ID, relationship.Kind, relationship.Metaclass), path)
	}
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
		addExternalEndpoints(byID, relationship, elementIDs)
	}
	out := make([]compositionExternalEndpoint, 0, len(byID))
	for _, endpoint := range byID {
		out = append(out, endpoint)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// addExternalEndpoints records the endpoints of an allocation or satisfaction
// that the loaded bundle does not define; a satisfaction target is a requirement.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-047, REQ-EMG-050
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, FU-SYSTEM-COMPOSITION
func addExternalEndpoints(byID map[string]compositionExternalEndpoint, relationship model.SemanticRelationship, elementIDs map[string]bool) {
	targetIsRequirement := false
	switch relationship.Kind {
	case model.RelationshipAllocation:
	case model.RelationshipSatisfaction:
		targetIsRequirement = true
	default:
		return
	}
	if !elementIDs[relationship.Source] {
		byID[relationship.Source] = compositionExternalEndpoint{ID: relationship.Source}
	}
	if !elementIDs[relationship.Target] {
		byID[relationship.Target] = compositionExternalEndpoint{ID: relationship.Target, Requirement: targetIsRequirement}
	}
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

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-137
func (w *sysmlWriter) namespaceElements(namespace, indent string) {
	for _, element := range w.children[namespace] {
		fmt.Fprintln(w.out)
		w.element(element, indent)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-137
func (w *sysmlWriter) element(element model.SemanticElement, indent string) {
	writeElementAnnotationAt(w.out, element, indent)
	keyword, ok := nativeElementKeyword(element)
	if !ok {
		w.diagnoseError("sysml.missing_element_rule",
			fmt.Sprintf("element %q has no native rendering rule for %q (%s)", element.ID, element.Kind, element.Metaclass),
			"elements."+element.ID)
		return
	}
	switch {
	case element.Kind == model.ElementPackageDefinition:
		w.packageElement(element, indent)
	case w.machines.isMachine(element.ID):
		w.machine(element, indent)
	default:
		w.declaration(element, keyword, indent)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (w *sysmlWriter) packageElement(element model.SemanticElement, indent string) {
	fmt.Fprintf(w.out, "%spackage %s {\n", indent, sysmlName(element.ID))
	for _, imported := range w.imports {
		if imported.Namespace == element.ID {
			writeImport(w.out, imported, indent+"    ")
		}
	}
	w.namespaceElements(element.ID, indent+"    ")
	fmt.Fprintf(w.out, "%s}\n", indent)
}

// declaration writes an element as a declaration with its typing, implied
// relationship ends, features and nested elements.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-039
func (w *sysmlWriter) declaration(element model.SemanticElement, keyword, indent string) {
	fmt.Fprintf(w.out, "%s%s ", indent, keyword)
	fmt.Fprint(w.out, sysmlName(element.ID))
	writeElementTyping(w.out, element)
	impliedEnds := needsImpliedRelationshipEnds(element.Metaclass)
	if len(element.Features) == 0 && len(w.children[element.ID]) == 0 && !impliedEnds {
		fmt.Fprintln(w.out, ";")
		return
	}
	fmt.Fprintln(w.out, " {")
	if impliedEnds {
		writeImpliedRelationshipEnds(w.out, element.Metaclass, indent)
	}
	for _, feature := range element.Features {
		writeFeature(w.out, feature, indent+"    ")
	}
	w.namespaceElements(element.ID, indent+"    ")
	fmt.Fprintf(w.out, "%s}\n", indent)
}

// TRLC-LINKS: REQ-EMG-039
func writeImpliedRelationshipEnds(out *bytes.Buffer, metaclass, indent string) {
	endKeyword := "item"
	if metaclass == "SysML::InterfaceDefinition" {
		endKeyword = "port"
	}
	typeName := "Engineering" + strings.ToUpper(endKeyword[:1]) + endKeyword[1:]
	fmt.Fprintf(out, "%s    end %s 'source' : %s;\n", indent, endKeyword, typeName)
	fmt.Fprintf(out, "%s    end %s 'target' : %s;\n", indent, endKeyword, typeName)
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

// featureKeywords maps a feature kind to its declaration keyword; every other
// kind is an attribute.
var featureKeywords = map[model.SemanticFeatureKind]string{
	model.FeatureParameter: "item ",
	model.FeaturePort:      "port ",
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func writeFeature(out *bytes.Buffer, feature model.SemanticFeature, indent string) {
	fmt.Fprint(out, indent)
	if direction := strings.TrimSpace(feature.Direction); direction != "" {
		fmt.Fprintf(out, "%s ", direction)
	}
	keyword, ok := featureKeywords[feature.Kind]
	if !ok {
		keyword = "attribute "
	}
	fmt.Fprint(out, keyword)
	fmt.Fprint(out, sysmlName(feature.Name))
	if typeName, ok := featureTypeName(feature); ok {
		writeTypingClause(out, typeName, feature.Conjugated && feature.Kind == model.FeaturePort)
	}
	writeMultiplicity(out, multiplicitySpec{feature.Multiplicity, feature.Ordered, feature.Unique})
	writeNameList(out, " :> ", feature.Specializes)
	writeNameList(out, " :> ", feature.Subsets)
	writeNameList(out, " :>> ", feature.Redefines)
	if feature.Value != nil {
		fmt.Fprintf(out, " = %s", sysmlExpression(*feature.Value))
	}
	fmt.Fprintln(out, ";")
}

// featureTypeName returns the rendered type of a feature: its own type when
// that is a native type reference, else the default type of a parameter or port.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func featureTypeName(feature model.SemanticFeature) (string, bool) {
	if strings.TrimSpace(feature.Type) != "" && isNativeTypeReference(feature.Type) {
		return sysmlName(feature.Type), true
	}
	if feature.Kind == model.FeatureParameter || feature.Kind == model.FeaturePort {
		return defaultFeatureType(feature.Kind), true
	}
	return "", false
}

// writeTypingClause writes " : Type", conjugated as " : ~Type".
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func writeTypingClause(out *bytes.Buffer, typeName string, conjugated bool) {
	fmt.Fprint(out, " : ")
	if conjugated {
		fmt.Fprint(out, "~")
	}
	fmt.Fprint(out, typeName)
}

// writeNameList writes each name after the operator, for example " :> 'A'".
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func writeNameList(out *bytes.Buffer, operator string, names []string) {
	for _, name := range names {
		fmt.Fprintf(out, "%s%s", operator, sysmlName(name))
	}
}

// multiplicitySpec groups the multiplicity, ordering and uniqueness of a
// feature or element.
type multiplicitySpec struct {
	Multiplicity *model.Multiplicity
	Ordered      bool
	Unique       *bool
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func writeMultiplicity(out *bytes.Buffer, spec multiplicitySpec) {
	if spec.Multiplicity != nil {
		fmt.Fprint(out, multiplicityRange(*spec.Multiplicity))
	}
	if spec.Ordered {
		fmt.Fprint(out, " ordered")
	}
	if spec.Unique != nil && !*spec.Unique {
		fmt.Fprint(out, " nonunique")
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func multiplicityRange(multiplicity model.Multiplicity) string {
	lower := "0"
	upper := "*"
	if multiplicity.Lower != nil {
		lower = fmt.Sprintf("%d", *multiplicity.Lower)
	}
	if multiplicity.Upper != nil && *multiplicity.Upper >= 0 {
		upper = fmt.Sprintf("%d", *multiplicity.Upper)
	}
	if lower == upper {
		return fmt.Sprintf(" [%s]", lower)
	}
	return fmt.Sprintf(" [%s..%s]", lower, upper)
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
	model.RelationshipDependency: writeDependency,
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
		transitionClauses{}.write(out, relationship, indent)
	},
	model.RelationshipEventTrigger: writeAcceptTransition,
	model.RelationshipGuard: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		transitionClauses{middle: "if " + relationshipExpression(relationship.Guard) + " "}.write(out, relationship, indent)
	},
	model.RelationshipEffect: func(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
		transitionClauses{effect: " do " + relationshipExpression(relationship.Effect)}.write(out, relationship, indent)
	},
	model.RelationshipExtension: writeDependency,
}

// dependencyRelationshipKinds are the relationship kinds SysML renders as a
// dependency between their source and target.
var dependencyRelationshipKinds = map[model.SemanticRelationshipKind]bool{
	model.RelationshipTyping:         true,
	model.RelationshipSpecialization: true,
	model.RelationshipSubsetting:     true,
	model.RelationshipRedefinition:   true,
	model.RelationshipVariant:        true,
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func writeNativeRelationship(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) bool {
	if relationship.Kind == model.RelationshipReference {
		fmt.Fprintf(out, "%sref item %s subsets %s = %s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Target), sysmlName(relationship.Source))
		return true
	}
	if dependencyRelationshipKinds[relationship.Kind] {
		writeDependency(out, relationship, indent)
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
func writeDependency(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
	fmt.Fprintf(out, "%sdependency %s from %s to %s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Source), sysmlName(relationship.Target))
}

// transitionClauses are the parts of a transition declaration between its
// source and its target (middle) and after its target (effect).
type transitionClauses struct {
	middle string
	effect string
}

// write renders a transition between two actions.
//
// TRLC-LINKS: REQ-EMG-039
func (c transitionClauses) write(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
	fmt.Fprintf(out, "%sstate %s : EngineeringState {\n", indent, sysmlName(relationship.ID+"-context"))
	fmt.Fprintf(out, "%s    action %s : EngineeringAction;\n", indent, sysmlName(relationship.Source))
	fmt.Fprintf(out, "%s    action %s : EngineeringAction;\n", indent, sysmlName(relationship.Target))
	fmt.Fprintf(out, "%s    transition %s first %s %sthen %s%s;\n", indent, sysmlName(relationship.ID), sysmlName(relationship.Source), c.middle, sysmlName(relationship.Target), c.effect)
	fmt.Fprintf(out, "%s}\n", indent)
}

// writeAcceptTransition renders a transition between two states that accepts a trigger.
//
// TRLC-LINKS: REQ-EMG-039
func writeAcceptTransition(out *bytes.Buffer, relationship model.SemanticRelationship, indent string) {
	fmt.Fprintf(out, "%sstate %s : EngineeringState {\n", indent, sysmlName(relationship.ID+"-context"))
	fmt.Fprintf(out, "%s    state %s : EngineeringState;\n", indent, sysmlName(relationship.Source))
	fmt.Fprintf(out, "%s    action transitionEffect : EngineeringAction;\n", indent)
	fmt.Fprintf(out, "%s    transition %s first %s accept %s : EngineeringItem then transitionEffect;\n",
		indent, sysmlName(relationship.ID), sysmlName(relationship.Source), sysmlName(relationship.Target))
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
func sysMLRendererCoverage() RendererCoverage {
	coverage := RendererCoverage{OfficialMetaclasses: len(model.OfficialSysMLMetaclasses)}
	for id, metaclass := range model.OfficialSysMLMetaclasses {
		entry := rendererEvidenceFor(id, metaclass)
		switch entry.Status {
		case "native":
			coverage.NativeRules++
		case "non-instantiable":
			coverage.NonInstantiable++
		default:
			coverage.MissingRules++
		}
		coverage.Entries = append(coverage.Entries, entry)
		if metaclass.Relationship {
			coverage.Relationships++
		}
	}
	sort.Slice(coverage.Entries, func(i, j int) bool {
		return coverage.Entries[i].Metaclass < coverage.Entries[j].Metaclass
	})
	return coverage
}

// rendererEvidenceFor classifies one official metaclass as native,
// non-instantiable or missing a rendering rule.
//
// TRLC-LINKS: REQ-EMG-039
func rendererEvidenceFor(id string, metaclass model.OfficialSysMLMetaclass) RendererEvidence {
	if syntax, ok := officialNativeMetaclassKeywords[id]; ok {
		return RendererEvidence{
			Metaclass: id, Status: "native", Syntax: syntax,
			Explanation: "Rendered by the table-driven canonical SysML textual dispatcher.",
		}
	}
	if metaclass.Abstract {
		return RendererEvidence{
			Metaclass: id, Status: "non-instantiable",
			Explanation: "Abstract metaclass; represented only through a concrete native subtype.",
		}
	}
	if isSemanticOnlyMetaclass(id, metaclass) {
		return RendererEvidence{
			Metaclass: id, Status: "non-instantiable",
			Explanation: "No standalone concrete textual production; produced implicitly by native ownership, typing, membership, expression, or relationship syntax.",
		}
	}
	return RendererEvidence{
		Metaclass: id, Status: "missing",
		Explanation: "Concrete canonical instance has no native textual rendering rule.",
	}
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
	if typeName, ok := elementTypeName(element); ok {
		writeTypingClause(out, typeName, element.Conjugated && element.Kind == model.ElementPortUsage)
	}
	writeMultiplicity(out, multiplicitySpec{element.Multiplicity, element.Ordered, element.Unique})
	writeNameList(out, " :> ", element.Specializes)
	writeNameList(out, " :> ", element.Subsets)
	writeNameList(out, " :>> ", element.Redefines)
}

// elementTypeName returns the rendered type of an element: its type reference,
// except for actions, else the default type of its kind.
//
// TRLC-LINKS: REQ-EMG-039
func elementTypeName(element model.SemanticElement) (string, bool) {
	if element.TypeRef != "" && element.Kind != model.ElementActionUsage && element.Kind != model.ElementControlNode {
		return sysmlName(element.TypeRef), true
	}
	if defaultType := defaultElementType(element.Kind); defaultType != "" {
		return defaultType, true
	}
	return "", false
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

// nativeScalarTypes are the SysML scalar types a feature type may name.
var nativeScalarTypes = map[string]bool{"String": true, "Boolean": true, "Integer": true, "Real": true}

// TRLC-LINKS: REQ-EMG-039
func isNativeTypeReference(value string) bool {
	value = strings.TrimSpace(value)
	if nativeScalarTypes[value] {
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
