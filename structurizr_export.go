// ENGMODEL-OWNER-UNIT: FU-STRUCTURIZR-EXPORTER
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

var structurizrTemplate = template.Must(template.New("structurizr").Funcs(template.FuncMap{
	"joinCSV": func(items []string) string {
		vals := []string{}
		for _, x := range items {
			x = safeDSLText(strings.TrimSpace(x))
			if x != "" {
				vals = append(vals, x)
			}
		}
		return strings.Join(vals, ",")
	},
}).Parse(structurizrTemplateText))

type StructurizrExportResult struct {
	DSL         string
	Diagnostics []validate.Diagnostic
}

type structurizrTemplateData struct {
	Title                  string
	Description            string
	RootIdentifier         string
	ContainerGroups        []structurizrContainerGroup
	Containers             []structurizrElement
	Elements               []structurizrElement
	Relationships          []structurizrRelationship
	DeploymentEnvironments []structurizrDeploymentEnvironment
	DynamicViews           []structurizrDynamicView
	DeploymentViews        []structurizrDeploymentView
}

type structurizrElement struct {
	Identifier  string
	Keyword     string
	Name        string
	Description string
	Technology  string
	Tags        []string
	Properties  []structurizrProperty
}

type structurizrContainerGroup struct {
	Name       string
	Containers []structurizrElement
}

type structurizrRelationship struct {
	From       string
	To         string
	Label      string
	Tags       []string
	Properties []structurizrProperty
}

type structurizrProperty struct {
	Name  string
	Value string
}

type structurizrDeploymentEnvironment struct {
	Name  string
	Nodes []structurizrDeploymentNode
}

type structurizrDeploymentNode struct {
	Identifier              string
	Name                    string
	Description             string
	Technology              string
	Tags                    []string
	Properties              []structurizrProperty
	ContainerInstances      []structurizrInstance
	SoftwareSystemInstances []structurizrInstance
}

type structurizrInstance struct {
	Identifier string
	Tags       []string
	Properties []structurizrProperty
}

type structurizrDynamicView struct {
	Scope       string
	Key         string
	Description string
	Steps       []structurizrRelationship
}

type structurizrDeploymentView struct {
	Scope       string
	Environment string
	Key         string
	Description string
}

// TRLC-LINKS: REQ-EMG-005
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
	elementKindByModelID := map[string]string{}

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
		elementKindByModelID[strings.TrimSpace(x.ID)] = "person"
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "person", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"Actor"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}}})
	}

	for _, x := range sortedFunctionalGroups(a.FunctionalGroups) {
		id := registerIdentifier(x.ID, "group", usedIdentifiers, elementIDByModelID)
		elementKindByModelID[strings.TrimSpace(x.ID)] = "softwareSystem"
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"FunctionalGroup"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}}})
	}

	for _, x := range sortedFunctionalUnits(a.FunctionalUnits) {
		id := registerIdentifier(x.ID, "fu", usedIdentifiers, elementIDByModelID)
		elementKindByModelID[strings.TrimSpace(x.ID)] = "container"
		containers = append(containers, structurizrElement{Identifier: id, Keyword: "container", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Prose)), Technology: safeDSLText("Functional Unit"), Tags: []string{"FunctionalUnit"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "functionalGroup", Value: strings.TrimSpace(x.Group)}}})
	}

	for _, x := range sortedReferencedElements(a.ReferencedElements) {
		id := registerIdentifier(x.ID, "ref", usedIdentifiers, elementIDByModelID)
		d := strings.TrimSpace(x.Layer)
		if d == "" {
			d = strings.TrimSpace(x.Kind)
		}
		elementKindByModelID[strings.TrimSpace(x.ID)] = "softwareSystem"
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(d), Tags: []string{"ReferencedElement", sanitizeIdentifier(strings.TrimSpace(x.Kind))}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "layer", Value: strings.TrimSpace(x.Layer)}, {Name: "kind", Value: strings.TrimSpace(x.Kind)}}})
	}

	for _, x := range sortedInterfaces(a.Interfaces) {
		id := registerIdentifier(x.ID, "if", usedIdentifiers, elementIDByModelID)
		elementKindByModelID[strings.TrimSpace(x.ID)] = "softwareSystem"
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Protocol + " " + x.Endpoint)), Tags: []string{"Interface"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "protocol", Value: strings.TrimSpace(x.Protocol)}, {Name: "endpoint", Value: strings.TrimSpace(x.Endpoint)}, {Name: "owner", Value: strings.TrimSpace(x.Owner)}}})
	}

	for _, x := range sortedDataObjects(a.DataObjects) {
		id := registerIdentifier(x.ID, "data", usedIdentifiers, elementIDByModelID)
		elementKindByModelID[strings.TrimSpace(x.ID)] = "softwareSystem"
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.SchemaRef)), Tags: []string{"DataObject", strings.TrimSpace(x.Sensitivity)}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "classification", Value: strings.TrimSpace(x.Classification)}, {Name: "retention", Value: strings.TrimSpace(x.Retention)}}})
	}

	for _, x := range sortedDeploymentTargets(a.DeploymentTargets) {
		id := registerIdentifier(x.ID, "dep", usedIdentifiers, elementIDByModelID)
		elementKindByModelID[strings.TrimSpace(x.ID)] = "deployment_target"
		_ = id
	}

	for _, x := range sortedControls(a.Controls) {
		id := registerIdentifier(x.ID, "ctrl", usedIdentifiers, elementIDByModelID)
		elementKindByModelID[strings.TrimSpace(x.ID)] = "softwareSystem"
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"Control", strings.TrimSpace(x.Category)}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}}})
	}

	for _, x := range sortedAttackVectors(a.AttackVectors) {
		id := registerIdentifier(x.ID, "av", usedIdentifiers, elementIDByModelID)
		elementKindByModelID[strings.TrimSpace(x.ID)] = "softwareSystem"
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"AttackVector"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}}})
	}

	for _, x := range sortedTrustBoundaries(a.TrustBoundaries) {
		id := registerIdentifier(x.ID, "tb", usedIdentifiers, elementIDByModelID)
		elementKindByModelID[strings.TrimSpace(x.ID)] = "softwareSystem"
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"TrustBoundary", strings.TrimSpace(x.BoundaryType)}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "parentRef", Value: strings.TrimSpace(x.ParentRef)}}})
	}

	for _, x := range sortedThreatScenarios(a.ThreatScenarios) {
		id := registerIdentifier(x.ID, "ts", usedIdentifiers, elementIDByModelID)
		elementKindByModelID[strings.TrimSpace(x.ID)] = "softwareSystem"
		elements = append(elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(nonEmpty(strings.TrimSpace(x.Title), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Summary)), Tags: []string{"ThreatScenario", strings.TrimSpace(x.Stride), strings.TrimSpace(x.Status)}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "severity", Value: strings.TrimSpace(x.Severity)}, {Name: "likelihood", Value: strings.TrimSpace(x.Likelihood)}, {Name: "impact", Value: strings.TrimSpace(x.Impact)}}})
	}

	containerGroups := buildContainerGroups(containers, a.FunctionalUnits, a.FunctionalGroups)

	relationships := []structurizrRelationship{}
	for _, m := range sortedMappings(a.Mappings) {
		if strings.TrimSpace(m.Type) == "deployed_to" {
			continue
		}
		from := elementIDByModelID[strings.TrimSpace(m.From)]
		to := elementIDByModelID[strings.TrimSpace(m.To)]
		if from == "" || to == "" {
			continue
		}
		label := nonEmpty(strings.TrimSpace(m.Description), strings.TrimSpace(m.Type))
		relationships = append(relationships, structurizrRelationship{From: from, To: to, Label: safeDSLText(label), Tags: []string{"Mapping", strings.TrimSpace(m.Type)}, Properties: []structurizrProperty{{Name: "mappingType", Value: strings.TrimSpace(m.Type)}, {Name: "fromId", Value: strings.TrimSpace(m.From)}, {Name: "toId", Value: strings.TrimSpace(m.To)}}})
	}
	relationships = appendFlowRelationships(a, relationships, elementIDByModelID)

	deploymentEnvs, deploymentViews := buildDeploymentModel(a, elementIDByModelID, elementKindByModelID, rootIdentifier, usedIdentifiers)
	dynamicViews := buildDynamicViews(a, elementIDByModelID, rootIdentifier)

	tplData := structurizrTemplateData{
		Title:                  title,
		Description:            description,
		RootIdentifier:         rootIdentifier,
		ContainerGroups:        containerGroups,
		Containers:             containers,
		Elements:               elements,
		Relationships:          relationships,
		DeploymentEnvironments: deploymentEnvs,
		DynamicViews:           dynamicViews,
		DeploymentViews:        deploymentViews,
	}
	normalizeStructurizrData(&tplData)

	doc, err := executeTextTemplate(structurizrTemplate, tplData)
	if err != nil {
		return StructurizrExportResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	return StructurizrExportResult{DSL: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

func buildContainerGroups(containers []structurizrElement, units []model.FunctionalUnit, groups []model.FunctionalGroup) []structurizrContainerGroup {
	unitByID := map[string]model.FunctionalUnit{}
	for _, u := range units {
		unitByID[strings.TrimSpace(u.ID)] = u
	}
	groupNameByID := map[string]string{}
	for _, g := range groups {
		groupNameByID[strings.TrimSpace(g.ID)] = nonEmpty(strings.TrimSpace(g.Name), strings.TrimSpace(g.ID))
	}
	groupItems := map[string][]structurizrElement{}
	ungrouped := []structurizrElement{}
	for _, c := range containers {
		u := unitByID[strings.TrimSpace(propValue(c.Properties, "sourceId"))]
		groupID := strings.TrimSpace(u.Group)
		if groupID == "" {
			ungrouped = append(ungrouped, c)
			continue
		}
		name := nonEmpty(groupNameByID[groupID], groupID)
		groupItems[name] = append(groupItems[name], c)
	}
	names := make([]string, 0, len(groupItems))
	for name := range groupItems {
		names = append(names, name)
	}
	sort.Strings(names)
	out := []structurizrContainerGroup{}
	for _, name := range names {
		items := groupItems[name]
		sort.SliceStable(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
		out = append(out, structurizrContainerGroup{Name: name, Containers: items})
	}
	if len(ungrouped) > 0 {
		sort.SliceStable(ungrouped, func(i, j int) bool { return ungrouped[i].Identifier < ungrouped[j].Identifier })
		out = append(out, structurizrContainerGroup{Name: "Ungrouped", Containers: ungrouped})
	}
	return out
}

func buildDeploymentModel(a model.AuthoredArchitecture, elementIDByModelID map[string]string, elementKindByModelID map[string]string, scope string, usedIdentifiers map[string]bool) ([]structurizrDeploymentEnvironment, []structurizrDeploymentView) {
	deployMappings := []model.Mapping{}
	for _, m := range a.Mappings {
		if strings.TrimSpace(m.Type) == "deployed_to" {
			deployMappings = append(deployMappings, m)
		}
	}
	if len(a.DeploymentTargets) == 0 {
		return nil, nil
	}

	instancesByTarget := map[string][]structurizrInstance{}
	ssInstancesByTarget := map[string][]structurizrInstance{}
	for _, m := range deployMappings {
		fromID := strings.TrimSpace(m.From)
		toID := strings.TrimSpace(m.To)
		fromElem := elementIDByModelID[fromID]
		if fromElem == "" || toID == "" {
			continue
		}
		kind := elementKindByModelID[fromID]
		inst := structurizrInstance{Identifier: fromElem, Tags: []string{"Deployed"}, Properties: []structurizrProperty{{Name: "sourceId", Value: fromID}}}
		if kind == "container" {
			instancesByTarget[toID] = append(instancesByTarget[toID], inst)
		} else {
			ssInstancesByTarget[toID] = append(ssInstancesByTarget[toID], inst)
		}
	}

	envNodes := map[string][]structurizrDeploymentNode{}
	for _, t := range sortedDeploymentTargets(a.DeploymentTargets) {
		targetID := strings.TrimSpace(t.ID)
		env := nonEmpty(strings.TrimSpace(t.Environment), "default")
		nodeID := uniqueIdentifier("dn_"+sanitizeIdentifier(targetID), usedIdentifiers)
		node := structurizrDeploymentNode{
			Identifier:  nodeID,
			Name:        safeDSLText(nonEmpty(strings.TrimSpace(t.Name), targetID)),
			Description: safeDSLText(strings.TrimSpace(strings.Join([]string{t.Account, t.Region, t.Namespace}, " "))),
			Technology:  safeDSLText(nonEmpty(strings.TrimSpace(t.Cluster), "Deployment Target")),
			Tags:        []string{"DeploymentTarget", env},
			Properties: []structurizrProperty{
				{Name: "sourceId", Value: targetID},
				{Name: "environment", Value: strings.TrimSpace(t.Environment)},
				{Name: "region", Value: strings.TrimSpace(t.Region)},
				{Name: "account", Value: strings.TrimSpace(t.Account)},
				{Name: "cluster", Value: strings.TrimSpace(t.Cluster)},
				{Name: "namespace", Value: strings.TrimSpace(t.Namespace)},
				{Name: "trustZone", Value: strings.TrimSpace(t.TrustZone)},
			},
			ContainerInstances:      dedupeInstances(instancesByTarget[targetID]),
			SoftwareSystemInstances: dedupeInstances(ssInstancesByTarget[targetID]),
		}
		envNodes[env] = append(envNodes[env], node)
	}

	envNames := make([]string, 0, len(envNodes))
	for env := range envNodes {
		envNames = append(envNames, env)
	}
	sort.Strings(envNames)
	envs := []structurizrDeploymentEnvironment{}
	views := []structurizrDeploymentView{}
	for _, env := range envNames {
		nodes := envNodes[env]
		sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].Identifier < nodes[j].Identifier })
		envs = append(envs, structurizrDeploymentEnvironment{Name: env, Nodes: nodes})
		views = append(views, structurizrDeploymentView{Scope: scope, Environment: env, Key: "deployment_" + sanitizeIdentifier(env), Description: "Deployment view for environment: " + env})
	}
	return envs, views
}

func buildDynamicViews(a model.AuthoredArchitecture, elementIDByModelID map[string]string, scope string) []structurizrDynamicView {
	views := []structurizrDynamicView{}
	for _, f := range a.Flows {
		from := elementIDByModelID[strings.TrimSpace(f.SourceRef)]
		to := elementIDByModelID[strings.TrimSpace(f.DestinationRef)]
		if from == "" || to == "" {
			continue
		}
		label := nonEmpty(strings.TrimSpace(f.Description), nonEmpty(strings.TrimSpace(f.Title), strings.TrimSpace(f.ID)))
		step := structurizrRelationship{From: from, To: to, Label: safeDSLText(label)}
		views = append(views, structurizrDynamicView{
			Scope:       scope,
			Key:         "dynamic_" + sanitizeIdentifier(strings.TrimSpace(f.ID)),
			Description: safeDSLText(nonEmpty(strings.TrimSpace(f.Description), nonEmpty(strings.TrimSpace(f.Title), strings.TrimSpace(f.ID)))),
			Steps:       []structurizrRelationship{step},
		})
	}
	sort.SliceStable(views, func(i, j int) bool { return views[i].Key < views[j].Key })
	return views
}

func dedupeInstances(in []structurizrInstance) []structurizrInstance {
	seen := map[string]bool{}
	out := []structurizrInstance{}
	for _, x := range in {
		if seen[x.Identifier] {
			continue
		}
		seen[x.Identifier] = true
		out = append(out, x)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Identifier < out[j].Identifier })
	return out
}

func propValue(props []structurizrProperty, name string) string {
	for _, p := range props {
		if strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(name)) {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

func appendFlowRelationships(a model.AuthoredArchitecture, relationships []structurizrRelationship, elementIDByModelID map[string]string) []structurizrRelationship {
	seen := map[string]bool{}
	for _, r := range relationships {
		seen[r.From+"->"+r.To] = true
	}
	for _, f := range a.Flows {
		from := elementIDByModelID[strings.TrimSpace(f.SourceRef)]
		to := elementIDByModelID[strings.TrimSpace(f.DestinationRef)]
		if from == "" || to == "" {
			continue
		}
		key := from + "->" + to
		if seen[key] {
			continue
		}
		seen[key] = true
		label := "flow"
		if title := strings.TrimSpace(f.Title); title != "" {
			label = title
		}
		relationships = append(relationships, structurizrRelationship{From: from, To: to, Label: safeDSLText(label), Tags: []string{"Flow"}, Properties: []structurizrProperty{{Name: "flowId", Value: strings.TrimSpace(f.ID)}}})
	}
	return relationships
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

func normalizeStructurizrData(d *structurizrTemplateData) {
	for i := range d.Containers {
		d.Containers[i].Tags = compactTags(d.Containers[i].Tags)
		d.Containers[i].Properties = compactProperties(d.Containers[i].Properties)
	}
	for i := range d.ContainerGroups {
		for j := range d.ContainerGroups[i].Containers {
			d.ContainerGroups[i].Containers[j].Tags = compactTags(d.ContainerGroups[i].Containers[j].Tags)
			d.ContainerGroups[i].Containers[j].Properties = compactProperties(d.ContainerGroups[i].Containers[j].Properties)
		}
	}
	for i := range d.Elements {
		d.Elements[i].Tags = compactTags(d.Elements[i].Tags)
		d.Elements[i].Properties = compactProperties(d.Elements[i].Properties)
	}
	for i := range d.Relationships {
		d.Relationships[i].Tags = compactTags(d.Relationships[i].Tags)
		d.Relationships[i].Properties = compactProperties(d.Relationships[i].Properties)
	}
	for i := range d.DeploymentEnvironments {
		for j := range d.DeploymentEnvironments[i].Nodes {
			node := &d.DeploymentEnvironments[i].Nodes[j]
			node.Tags = compactTags(node.Tags)
			node.Properties = compactProperties(node.Properties)
			for k := range node.ContainerInstances {
				node.ContainerInstances[k].Tags = compactTags(node.ContainerInstances[k].Tags)
				node.ContainerInstances[k].Properties = compactProperties(node.ContainerInstances[k].Properties)
			}
			for k := range node.SoftwareSystemInstances {
				node.SoftwareSystemInstances[k].Tags = compactTags(node.SoftwareSystemInstances[k].Tags)
				node.SoftwareSystemInstances[k].Properties = compactProperties(node.SoftwareSystemInstances[k].Properties)
			}
		}
	}
}

func compactTags(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range in {
		x = safeDSLText(strings.TrimSpace(x))
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	return out
}

func compactProperties(in []structurizrProperty) []structurizrProperty {
	seen := map[string]bool{}
	out := []structurizrProperty{}
	for _, p := range in {
		name := safeDSLText(strings.TrimSpace(p.Name))
		value := safeDSLText(strings.TrimSpace(p.Value))
		if name == "" || value == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, structurizrProperty{Name: name, Value: value})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
