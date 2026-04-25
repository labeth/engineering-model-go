package engmodel

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

//go:embed templates/structurizr.dsl.tmpl
var structurizrTemplateText string

var structurizrTemplate = template.Must(template.New("structurizr").Parse(structurizrTemplateText))

type StructurizrExportResult struct {
	DSL         string
	Diagnostics []validate.Diagnostic
}

type structurizrTemplateData struct {
	Title          string
	Description    string
	RootIdentifier string
	Containers     []structurizrElement
	Elements       []structurizrElement
	Relationships  []structurizrRelationship
}

type structurizrElement struct {
	Identifier  string
	Keyword     string
	Parent      string
	Name        string
	Description string
	Technology  string
}

type structurizrRelationship struct {
	From  string
	To    string
	Label string
}

func GenerateStructurizrDSLFromFile(architecturePath string) (StructurizrExportResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return StructurizrExportResult{}, err
	}
	return GenerateStructurizrDSL(bundle)
}

func GenerateStructurizrDSL(bundle model.Bundle) (StructurizrExportResult, error) {
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return StructurizrExportResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}

	a := bundle.Architecture.AuthoredArchitecture
	usedIdentifiers := map[string]bool{}
	elementIDByModelID := map[string]string{}

	rootIdentifier := uniqueIdentifier("sys_"+sanitizeIdentifier(bundle.Architecture.Model.ID), usedIdentifiers)
	if rootIdentifier == "" {
		rootIdentifier = uniqueIdentifier("sys_main", usedIdentifiers)
	}

	title := safeDSLText(nonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), nonEmpty(strings.TrimSpace(bundle.Architecture.Model.ID), "Architecture")))
	description := safeDSLText(strings.TrimSpace(bundle.Architecture.Model.Introduction))

	containers := []structurizrElement{}
	elements := []structurizrElement{}

	for _, x := range sortedActors(a.Actors) {
		id := registerIdentifier(x.ID, "person", usedIdentifiers, elementIDByModelID)
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "person", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description))})
	}

	for _, x := range sortedFunctionalGroups(a.FunctionalGroups) {
		id := registerIdentifier(x.ID, "group", usedIdentifiers, elementIDByModelID)
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText("Group: " + nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description))})
	}

	for _, x := range sortedFunctionalUnits(a.FunctionalUnits) {
		id := registerIdentifier(x.ID, "fu", usedIdentifiers, elementIDByModelID)
		containers = append(containers, structurizrElement{Identifier: id, Keyword: "container", Parent: rootIdentifier, Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Prose)), Technology: safeDSLText("Functional Unit")})
	}

	for _, x := range sortedReferencedElements(a.ReferencedElements) {
		id := registerIdentifier(x.ID, "ref", usedIdentifiers, elementIDByModelID)
		d := strings.TrimSpace(x.Layer)
		if d == "" {
			d = strings.TrimSpace(x.Kind)
		}
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText("Ref: " + nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(d)})
	}

	for _, x := range sortedInterfaces(a.Interfaces) {
		id := registerIdentifier(x.ID, "if", usedIdentifiers, elementIDByModelID)
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText("Interface: " + nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Protocol + " " + x.Endpoint))})
	}

	for _, x := range sortedDataObjects(a.DataObjects) {
		id := registerIdentifier(x.ID, "data", usedIdentifiers, elementIDByModelID)
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText("Data: " + nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.SchemaRef))})
	}

	for _, x := range sortedDeploymentTargets(a.DeploymentTargets) {
		id := registerIdentifier(x.ID, "dep", usedIdentifiers, elementIDByModelID)
		d := strings.TrimSpace(strings.Join([]string{x.Environment, x.Cluster, x.Namespace, x.Region}, " "))
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText("Deployment: " + nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(d)})
	}

	for _, x := range sortedControls(a.Controls) {
		id := registerIdentifier(x.ID, "ctrl", usedIdentifiers, elementIDByModelID)
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText("Control: " + nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description))})
	}

	for _, x := range sortedAttackVectors(a.AttackVectors) {
		id := registerIdentifier(x.ID, "av", usedIdentifiers, elementIDByModelID)
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText("Attack: " + nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description))})
	}

	for _, x := range sortedTrustBoundaries(a.TrustBoundaries) {
		id := registerIdentifier(x.ID, "tb", usedIdentifiers, elementIDByModelID)
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText("Boundary: " + nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description))})
	}

	for _, x := range sortedThreatScenarios(a.ThreatScenarios) {
		id := registerIdentifier(x.ID, "ts", usedIdentifiers, elementIDByModelID)
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText("Threat: " + nonEmpty(strings.TrimSpace(x.Title), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Summary))})
	}

	relationships := []structurizrRelationship{}
	for _, m := range sortedMappings(a.Mappings) {
		from := elementIDByModelID[strings.TrimSpace(m.From)]
		to := elementIDByModelID[strings.TrimSpace(m.To)]
		if from == "" || to == "" {
			continue
		}
		label := strings.TrimSpace(m.Type)
		if strings.TrimSpace(m.Description) != "" {
			label = label + ": " + strings.TrimSpace(m.Description)
		}
		relationships = append(relationships, structurizrRelationship{From: from, To: to, Label: safeDSLText(label)})
	}

	tplData := structurizrTemplateData{
		Title:          title,
		Description:    description,
		RootIdentifier: rootIdentifier,
		Containers:     containers,
		Elements:       elements,
		Relationships:  relationships,
	}

	doc, err := executeTextTemplate(structurizrTemplate, tplData)
	if err != nil {
		return StructurizrExportResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	return StructurizrExportResult{DSL: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

func registerIdentifier(modelID, prefix string, used map[string]bool, byID map[string]string) string {
	key := strings.TrimSpace(modelID)
	if key == "" {
		return ""
	}
	if existing := byID[key]; existing != "" {
		return existing
	}
	id := uniqueIdentifier(prefix+"_"+sanitizeIdentifier(key), used)
	byID[key] = id
	return id
}

func uniqueIdentifier(base string, used map[string]bool) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "e"
	}
	if !used[base] {
		used[base] = true
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s_%d", base, i)
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

func sanitizeIdentifier(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	if in == "" {
		return ""
	}
	b := strings.Builder{}
	for i, r := range in {
		isAlpha := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isNum := r >= '0' && r <= '9'
		if isAlpha || isNum {
			if i == 0 && isNum {
				b.WriteByte('e')
				b.WriteByte('_')
			}
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	out = strings.Trim(out, "_")
	if out == "" {
		return "e"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "e_" + out
	}
	return out
}

func sortedMappings(in []model.Mapping) []model.Mapping {
	out := append([]model.Mapping(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

func sortedActors(in []model.Actor) []model.Actor {
	out := append([]model.Actor(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedFunctionalGroups(in []model.FunctionalGroup) []model.FunctionalGroup {
	out := append([]model.FunctionalGroup(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedFunctionalUnits(in []model.FunctionalUnit) []model.FunctionalUnit {
	out := append([]model.FunctionalUnit(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedReferencedElements(in []model.ReferencedElement) []model.ReferencedElement {
	out := append([]model.ReferencedElement(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedInterfaces(in []model.Interface) []model.Interface {
	out := append([]model.Interface(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedDataObjects(in []model.DataObject) []model.DataObject {
	out := append([]model.DataObject(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedDeploymentTargets(in []model.DeploymentTarget) []model.DeploymentTarget {
	out := append([]model.DeploymentTarget(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedControls(in []model.Control) []model.Control {
	out := append([]model.Control(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedAttackVectors(in []model.AttackVector) []model.AttackVector {
	out := append([]model.AttackVector(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedTrustBoundaries(in []model.TrustBoundary) []model.TrustBoundary {
	out := append([]model.TrustBoundary(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedThreatScenarios(in []model.ThreatScenario) []model.ThreatScenario {
	out := append([]model.ThreatScenario(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func executeTextTemplate(t *template.Template, data any) (string, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute structurizr template: %w", err)
	}
	return b.String(), nil
}

func safeDSLText(in string) string {
	in = strings.ReplaceAll(in, "\r", " ")
	in = strings.ReplaceAll(in, "\n", " ")
	in = strings.Join(strings.Fields(in), " ")
	return strings.TrimSpace(in)
}
