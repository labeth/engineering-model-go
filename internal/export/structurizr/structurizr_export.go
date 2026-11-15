// ENGMODEL-OWNER-UNIT: FU-STRUCTURIZR-EXPORTER
package structurizr

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

//go:embed templates/structurizr.dsl.tmpl
var structurizrTemplateText string

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
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

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type Result struct {
	DSL         string
	Diagnostics []validate.Diagnostic
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
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

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
type structurizrElement struct {
	Identifier  string
	Keyword     string
	Name        string
	Description string
	Technology  string
	Tags        []string
	Properties  []structurizrProperty
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
type structurizrContainerGroup struct {
	Name       string
	Containers []structurizrElement
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
type structurizrRelationship struct {
	From       string
	To         string
	Label      string
	Technology string
	Tags       []string
	Properties []structurizrProperty
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
type structurizrProperty struct {
	Name  string
	Value string
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
type structurizrDeploymentEnvironment struct {
	Name  string
	Nodes []structurizrDeploymentNode
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
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

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
type structurizrInstance struct {
	Identifier string
	Tags       []string
	Properties []structurizrProperty
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FU-VIEW-PROJECTION
type structurizrDynamicView struct {
	Scope       string
	Key         string
	Description string
	Steps       []structurizrRelationship
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION
type structurizrDeploymentView struct {
	Scope       string
	Environment string
	Key         string
	Description string
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateDSLFromFile(architecturePath string) (Result, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return Result{}, err
	}
	projection, _, err := composition.ResolveCompositionProjection(bundle)
	if err != nil {
		return Result{}, err
	}
	return generateStructurizrDSL(bundle, projection)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-005, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func GenerateDSL(bundle model.Bundle) (Result, error) {
	return generateStructurizrDSL(bundle, composition.CompositionProjection{})
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func generateStructurizrDSL(bundle model.Bundle, projection composition.CompositionProjection) (Result, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return Result{}, err
	}
	bundle = canonical.Documents()
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return Result{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}
	bundle = projection.EnrichBundle(bundle, "architecture", "behavior")

	a := bundle.Architecture.AuthoredArchitecture
	b := newStructurizrBuilder()
	rootIdentifier := b.rootIdentifier(bundle.Architecture.Model.ID)
	b.addElements(a)
	deploymentEnvs, deploymentViews := b.buildDeploymentModel(a, rootIdentifier)

	tplData := structurizrTemplateData{
		Title:                  safeDSLText(textutil.NonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), textutil.NonEmpty(strings.TrimSpace(bundle.Architecture.Model.ID), "Architecture"))),
		Description:            safeDSLText(strings.TrimSpace(bundle.Architecture.Model.Introduction)),
		RootIdentifier:         rootIdentifier,
		ContainerGroups:        buildContainerGroups(b.containers, a.FunctionalUnits, a.FunctionalGroups),
		Containers:             b.containers,
		Elements:               b.elements,
		Relationships:          b.buildRelationships(a),
		DeploymentEnvironments: deploymentEnvs,
		DynamicViews:           buildDynamicViews(a, b.idByModelID, rootIdentifier),
		DeploymentViews:        deploymentViews,
	}
	normalizeStructurizrData(&tplData)

	doc, err := executeTextTemplate(structurizrTemplate, tplData)
	if err != nil {
		return Result{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	return Result{DSL: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
type structurizrClass struct {
	prefix string
	kind   string
}

var (
	classPerson           = structurizrClass{prefix: "person", kind: "person"}
	classGroup            = structurizrClass{prefix: "group", kind: "softwareSystem"}
	classUnit             = structurizrClass{prefix: "fu", kind: "container"}
	classReferenced       = structurizrClass{prefix: "ref", kind: "softwareSystem"}
	classInterface        = structurizrClass{prefix: "if", kind: "softwareSystem"}
	classData             = structurizrClass{prefix: "data", kind: "softwareSystem"}
	classDeploymentTarget = structurizrClass{prefix: "dep", kind: "deployment_target"}
	classControl          = structurizrClass{prefix: "ctrl", kind: "softwareSystem"}
	classAttackVector     = structurizrClass{prefix: "av", kind: "softwareSystem"}
	classTrustBoundary    = structurizrClass{prefix: "tb", kind: "softwareSystem"}
	classThreatScenario   = structurizrClass{prefix: "ts", kind: "softwareSystem"}
)

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
type structurizrBuilder struct {
	used          map[string]bool
	idByModelID   map[string]string
	kindByModelID map[string]string
	elements      []structurizrElement
	containers    []structurizrElement
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func newStructurizrBuilder() *structurizrBuilder {
	return &structurizrBuilder{
		used:          map[string]bool{},
		idByModelID:   map[string]string{},
		kindByModelID: map[string]string{},
		elements:      []structurizrElement{},
		containers:    []structurizrElement{},
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) rootIdentifier(modelID string) string {
	root := uniqueIdentifier("sys_"+sanitizeIdentifier(modelID), b.used)
	if root == "" {
		root = uniqueIdentifier("sys_main", b.used)
	}
	return root
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) register(modelID string, class structurizrClass) string {
	id := b.registerIdentifier(modelID, class.prefix)
	b.kindByModelID[strings.TrimSpace(modelID)] = class.kind
	return id
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) addElements(a model.AuthoredArchitecture) {
	b.addActors(a.Actors)
	b.addFunctionalGroups(a.FunctionalGroups)
	b.addFunctionalUnits(a.FunctionalUnits)
	b.addReferencedElements(a.ReferencedElements)
	b.addInterfaces(a.Interfaces)
	b.addDataObjects(a.DataObjects)
	for _, x := range sortedDeploymentTargets(a.DeploymentTargets) {
		b.register(x.ID, classDeploymentTarget)
	}
	b.addControls(a.Controls)
	b.addAttackVectors(a.AttackVectors)
	b.addTrustBoundaries(a.TrustBoundaries)
	b.addThreatScenarios(a.ThreatScenarios)
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) addActors(in []model.Actor) {
	for _, x := range sortedActors(in) {
		id := b.register(x.ID, classPerson)
		b.elements = append(b.elements, structurizrElement{Identifier: id, Keyword: "person", Name: safeDSLText(projectionDisplayName(x.ID, x.Name)), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"Actor"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}}})
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) addFunctionalGroups(in []model.FunctionalGroup) {
	for _, x := range sortedFunctionalGroups(in) {
		id := b.register(x.ID, classGroup)
		b.elements = append(b.elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(projectionDisplayName(x.ID, x.Name)), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"FunctionalGroup"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}}})
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) addFunctionalUnits(in []model.FunctionalUnit) {
	for _, x := range sortedFunctionalUnits(in) {
		id := b.register(x.ID, classUnit)
		b.containers = append(b.containers, structurizrElement{Identifier: id, Keyword: "container", Name: safeDSLText(projectionDisplayName(x.ID, x.Name)), Description: safeDSLText(strings.TrimSpace(x.Prose)), Technology: safeDSLText("Functional Unit"), Tags: []string{"FunctionalUnit"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "functionalGroup", Value: strings.TrimSpace(x.Group)}}})
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) addReferencedElements(in []model.ReferencedElement) {
	for _, x := range sortedReferencedElements(in) {
		id := b.register(x.ID, classReferenced)
		d := textutil.NonEmpty(strings.TrimSpace(x.Layer), strings.TrimSpace(x.Kind))
		b.elements = append(b.elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(projectionDisplayName(x.ID, x.Name)), Description: safeDSLText(d), Tags: []string{"ReferencedElement", sanitizeIdentifier(strings.TrimSpace(x.Kind))}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "layer", Value: strings.TrimSpace(x.Layer)}, {Name: "kind", Value: strings.TrimSpace(x.Kind)}}})
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) addInterfaces(in []model.Interface) {
	for _, x := range sortedInterfaces(in) {
		id := b.register(x.ID, classInterface)
		b.elements = append(b.elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(projectionDisplayName(x.ID, x.Name)), Description: safeDSLText(strings.TrimSpace(x.Protocol + " " + x.Endpoint)), Tags: []string{"Interface"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "protocol", Value: strings.TrimSpace(x.Protocol)}, {Name: "endpoint", Value: strings.TrimSpace(x.Endpoint)}, {Name: "owner", Value: strings.TrimSpace(x.Owner)}}})
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) addDataObjects(in []model.DataObject) {
	for _, x := range sortedDataObjects(in) {
		id := b.register(x.ID, classData)
		b.elements = append(b.elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(projectionDisplayName(x.ID, x.Name)), Description: safeDSLText(strings.TrimSpace(x.SchemaRef)), Tags: []string{"DataObject", strings.TrimSpace(x.Sensitivity)}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "classification", Value: strings.TrimSpace(x.Classification)}, {Name: "retention", Value: strings.TrimSpace(x.Retention)}}})
	}
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, CTRL-TRACEABILITY-COVERAGE
func (b *structurizrBuilder) addControls(in []model.Control) {
	for _, x := range sortedControls(in) {
		id := b.register(x.ID, classControl)
		b.elements = append(b.elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(textutil.NonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"Control", strings.TrimSpace(x.Category)}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}}})
	}
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FU-THREAT-EXPORTER
func (b *structurizrBuilder) addAttackVectors(in []model.AttackVector) {
	for _, x := range sortedAttackVectors(in) {
		id := b.register(x.ID, classAttackVector)
		b.elements = append(b.elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(textutil.NonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"AttackVector"}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}}})
	}
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *structurizrBuilder) addTrustBoundaries(in []model.TrustBoundary) {
	for _, x := range sortedTrustBoundaries(in) {
		id := b.register(x.ID, classTrustBoundary)
		b.elements = append(b.elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(textutil.NonEmpty(strings.TrimSpace(x.Name), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Description)), Tags: []string{"TrustBoundary", strings.TrimSpace(x.BoundaryType)}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "parentRef", Value: strings.TrimSpace(x.ParentRef)}}})
	}
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FU-THREAT-EXPORTER
func (b *structurizrBuilder) addThreatScenarios(in []model.ThreatScenario) {
	for _, x := range sortedThreatScenarios(in) {
		id := b.register(x.ID, classThreatScenario)
		b.elements = append(b.elements, structurizrElement{Identifier: id, Keyword: "softwareSystem", Name: safeDSLText(textutil.NonEmpty(strings.TrimSpace(x.Title), strings.TrimSpace(x.ID))), Description: safeDSLText(strings.TrimSpace(x.Summary)), Tags: []string{"ThreatScenario", strings.TrimSpace(x.Stride), strings.TrimSpace(x.Status)}, Properties: []structurizrProperty{{Name: "sourceId", Value: strings.TrimSpace(x.ID)}, {Name: "severity", Value: strings.TrimSpace(x.Severity)}, {Name: "likelihood", Value: strings.TrimSpace(x.Likelihood)}, {Name: "impact", Value: strings.TrimSpace(x.Impact)}}})
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) buildRelationships(a model.AuthoredArchitecture) []structurizrRelationship {
	relationships := []structurizrRelationship{}
	for _, m := range sortedMappings(a.Mappings) {
		if strings.TrimSpace(m.Type) == "deployed_to" {
			continue
		}
		from := b.idByModelID[strings.TrimSpace(m.From)]
		to := b.idByModelID[strings.TrimSpace(m.To)]
		if from == "" || to == "" {
			continue
		}
		relationships = append(relationships, mappingRelationship(m, from, to))
	}
	return appendFlowRelationships(a, relationships, b.idByModelID)
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func mappingRelationship(m model.Mapping, from, to string) structurizrRelationship {
	label := textutil.NonEmpty(strings.TrimSpace(m.Description), strings.TrimSpace(m.Type))
	technology := "Model relationship"
	if mappingType := strings.TrimSpace(m.Type); mappingType != "" {
		technology = "Model relationship: " + mappingType
	}
	return structurizrRelationship{From: from, To: to, Label: safeDSLText(label), Technology: safeDSLText(technology), Tags: []string{"Mapping", strings.TrimSpace(m.Type)}, Properties: []structurizrProperty{{Name: "mappingType", Value: strings.TrimSpace(m.Type)}, {Name: "fromId", Value: strings.TrimSpace(m.From)}, {Name: "toId", Value: strings.TrimSpace(m.To)}}}
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func projectionDisplayName(id, name string) string {
	id = strings.TrimSpace(id)
	name = textutil.NonEmpty(strings.TrimSpace(name), id)
	if strings.Contains(id, "::") {
		return id + " — " + name
	}
	return name
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func buildContainerGroups(containers []structurizrElement, units []model.FunctionalUnit, groups []model.FunctionalGroup) []structurizrContainerGroup {
	unitByID := map[string]model.FunctionalUnit{}
	for _, u := range units {
		unitByID[strings.TrimSpace(u.ID)] = u
	}
	groupNameByID := map[string]string{}
	for _, g := range groups {
		groupNameByID[strings.TrimSpace(g.ID)] = textutil.NonEmpty(strings.TrimSpace(g.Name), strings.TrimSpace(g.ID))
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
		name := textutil.NonEmpty(groupNameByID[groupID], groupID)
		groupItems[name] = append(groupItems[name], c)
	}
	out := sortedContainerGroups(groupItems)
	if len(ungrouped) > 0 {
		sort.SliceStable(ungrouped, func(i, j int) bool { return ungrouped[i].Identifier < ungrouped[j].Identifier })
		out = append(out, structurizrContainerGroup{Name: "Ungrouped", Containers: ungrouped})
	}
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func sortedContainerGroups(groupItems map[string][]structurizrElement) []structurizrContainerGroup {
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
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION
func (b *structurizrBuilder) buildDeploymentModel(a model.AuthoredArchitecture, scope string) ([]structurizrDeploymentEnvironment, []structurizrDeploymentView) {
	if len(a.DeploymentTargets) == 0 {
		return nil, nil
	}
	instances := b.collectDeploymentInstances(a.Mappings)

	envNodes := map[string][]structurizrDeploymentNode{}
	for _, t := range sortedDeploymentTargets(a.DeploymentTargets) {
		env := textutil.NonEmpty(strings.TrimSpace(t.Environment), "default")
		envNodes[env] = append(envNodes[env], b.deploymentNode(t, instances))
	}
	return deploymentEnvironmentsAndViews(envNodes, scope)
}

// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION
type deploymentInstances struct {
	containers      map[string][]structurizrInstance
	softwareSystems map[string][]structurizrInstance
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION
func (b *structurizrBuilder) collectDeploymentInstances(mappings []model.Mapping) deploymentInstances {
	out := deploymentInstances{containers: map[string][]structurizrInstance{}, softwareSystems: map[string][]structurizrInstance{}}
	for _, m := range mappings {
		if strings.TrimSpace(m.Type) != "deployed_to" {
			continue
		}
		fromID := strings.TrimSpace(m.From)
		toID := strings.TrimSpace(m.To)
		fromElem := b.idByModelID[fromID]
		if fromElem == "" || toID == "" {
			continue
		}
		inst := structurizrInstance{Identifier: fromElem, Tags: []string{"Deployed"}, Properties: []structurizrProperty{{Name: "sourceId", Value: fromID}}}
		if b.kindByModelID[fromID] == "container" {
			out.containers[toID] = append(out.containers[toID], inst)
		} else {
			out.softwareSystems[toID] = append(out.softwareSystems[toID], inst)
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION
func (b *structurizrBuilder) deploymentNode(t model.DeploymentTarget, instances deploymentInstances) structurizrDeploymentNode {
	targetID := strings.TrimSpace(t.ID)
	env := textutil.NonEmpty(strings.TrimSpace(t.Environment), "default")
	return structurizrDeploymentNode{
		Identifier:  uniqueIdentifier("dn_"+sanitizeIdentifier(targetID), b.used),
		Name:        safeDSLText(textutil.NonEmpty(strings.TrimSpace(t.Name), targetID)),
		Description: safeDSLText(strings.TrimSpace(strings.Join([]string{t.Account, t.Region, t.Namespace}, " "))),
		Technology:  safeDSLText(textutil.NonEmpty(strings.TrimSpace(t.Cluster), "Deployment Target")),
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
		ContainerInstances:      dedupeInstances(instances.containers[targetID]),
		SoftwareSystemInstances: dedupeInstances(instances.softwareSystems[targetID]),
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, FU-VIEW-PROJECTION
func deploymentEnvironmentsAndViews(envNodes map[string][]structurizrDeploymentNode, scope string) ([]structurizrDeploymentEnvironment, []structurizrDeploymentView) {
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

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FU-VIEW-PROJECTION
func buildDynamicViews(a model.AuthoredArchitecture, elementIDByModelID map[string]string, scope string) []structurizrDynamicView {
	views := []structurizrDynamicView{}
	for _, f := range a.Flows {
		from := elementIDByModelID[strings.TrimSpace(f.SourceRef)]
		to := elementIDByModelID[strings.TrimSpace(f.DestinationRef)]
		if from == "" || to == "" {
			continue
		}
		step := structurizrRelationship{From: from, To: to, Label: safeDSLText(flowRelationshipLabel(f)), Technology: safeDSLText(flowTechnology(f))}
		views = append(views, structurizrDynamicView{
			Scope:       scope,
			Key:         "dynamic_" + sanitizeIdentifier(strings.TrimSpace(f.ID)),
			Description: safeDSLText(textutil.NonEmpty(strings.TrimSpace(f.Description), textutil.NonEmpty(strings.TrimSpace(f.Title), strings.TrimSpace(f.ID)))),
			Steps:       []structurizrRelationship{step},
		})
	}
	sort.SliceStable(views, func(i, j int) bool { return views[i].Key < views[j].Key })
	return views
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
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

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func propValue(props []structurizrProperty, name string) string {
	for _, p := range props {
		if strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(name)) {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func appendFlowRelationships(a model.AuthoredArchitecture, relationships []structurizrRelationship, elementIDByModelID map[string]string) []structurizrRelationship {
	seen := map[string]bool{}
	for _, r := range relationships {
		seen[structurizrRelationshipKey(r)] = true
	}
	for _, f := range a.Flows {
		from := elementIDByModelID[strings.TrimSpace(f.SourceRef)]
		to := elementIDByModelID[strings.TrimSpace(f.DestinationRef)]
		if from == "" || to == "" {
			continue
		}
		relationship := structurizrRelationship{From: from, To: to, Label: safeDSLText(flowRelationshipLabel(f)), Technology: safeDSLText(flowTechnology(f)), Tags: []string{"Flow"}, Properties: []structurizrProperty{{Name: "flowId", Value: strings.TrimSpace(f.ID)}}}
		key := structurizrRelationshipKey(relationship)
		if seen[key] {
			continue
		}
		seen[key] = true
		relationships = append(relationships, relationship)
	}
	return relationships
}

// TRLC-LINKS: REQ-EMG-005, REQ-EMG-012
func structurizrRelationshipKey(relationship structurizrRelationship) string {
	return strings.Join([]string{relationship.From, relationship.To, relationship.Label, relationship.Technology}, "\x00")
}

// TRLC-LINKS: REQ-EMG-005, REQ-EMG-012
func flowRelationshipLabel(flow model.Flow) string {
	if title := strings.TrimSpace(flow.Title); title != "" {
		return title
	}
	return "flow"
}

// TRLC-LINKS: REQ-EMG-005, REQ-EMG-012
func flowTechnology(flow model.Flow) string {
	parts := []string{}
	for _, value := range []string{flow.Protocol, flow.Channel} {
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " / ")
	}
	if kind := strings.TrimSpace(flow.Kind); kind != "" {
		return "Model flow: " + kind
	}
	return "Model flow"
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func (b *structurizrBuilder) registerIdentifier(modelID, prefix string) string {
	key := strings.TrimSpace(modelID)
	if key == "" {
		return ""
	}
	if existing := b.idByModelID[key]; existing != "" {
		return existing
	}
	id := uniqueIdentifier(prefix+"_"+sanitizeIdentifier(key), b.used)
	b.idByModelID[key] = id
	return id
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
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

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func sanitizeIdentifier(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	if in == "" {
		return ""
	}
	b := strings.Builder{}
	for i, r := range in {
		b.WriteString(identifierRune(r, i == 0))
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	out = strings.Trim(out, "_")
	if out == "" {
		return "e"
	}
	if isASCIIDigit(rune(out[0])) {
		return "e_" + out
	}
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func identifierRune(r rune, first bool) string {
	switch {
	case isASCIIDigit(r) && first:
		return "e_" + string(r)
	case isASCIIDigit(r), isASCIILetter(r):
		return string(r)
	}
	return "_"
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func isASCIILetter(r rune) bool {
	lower := r | 0x20
	return lower >= 'a' && lower <= 'z'
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
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

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func sortedActors(in []model.Actor) []model.Actor {
	out := append([]model.Actor(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func sortedFunctionalGroups(in []model.FunctionalGroup) []model.FunctionalGroup {
	out := append([]model.FunctionalGroup(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func sortedFunctionalUnits(in []model.FunctionalUnit) []model.FunctionalUnit {
	out := append([]model.FunctionalUnit(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func sortedReferencedElements(in []model.ReferencedElement) []model.ReferencedElement {
	out := append([]model.ReferencedElement(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func sortedInterfaces(in []model.Interface) []model.Interface {
	out := append([]model.Interface(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func sortedDataObjects(in []model.DataObject) []model.DataObject {
	out := append([]model.DataObject(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
func sortedDeploymentTargets(in []model.DeploymentTarget) []model.DeploymentTarget {
	out := append([]model.DeploymentTarget(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, CTRL-TRACEABILITY-COVERAGE
func sortedControls(in []model.Control) []model.Control {
	out := append([]model.Control(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FU-THREAT-EXPORTER
func sortedAttackVectors(in []model.AttackVector) []model.AttackVector {
	out := append([]model.AttackVector(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func sortedTrustBoundaries(in []model.TrustBoundary) []model.TrustBoundary {
	out := append([]model.TrustBoundary(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL, FU-THREAT-EXPORTER
func sortedThreatScenarios(in []model.ThreatScenario) []model.ThreatScenario {
	out := append([]model.ThreatScenario(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func executeTextTemplate(t *template.Template, data any) (string, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute structurizr template: %w", err)
	}
	return b.String(), nil
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func safeDSLText(in string) string {
	in = strings.ReplaceAll(in, "\r", " ")
	in = strings.ReplaceAll(in, "\n", " ")
	in = strings.Join(strings.Fields(in), " ")
	return strings.TrimSpace(in)
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func normalizeStructurizrData(d *structurizrTemplateData) {
	compactElements(d.Containers)
	for i := range d.ContainerGroups {
		compactElements(d.ContainerGroups[i].Containers)
	}
	compactElements(d.Elements)
	for i := range d.Relationships {
		d.Relationships[i].Tags = compactTags(d.Relationships[i].Tags)
		d.Relationships[i].Properties = compactProperties(d.Relationships[i].Properties)
	}
	for i := range d.DeploymentEnvironments {
		compactNodes(d.DeploymentEnvironments[i].Nodes)
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func compactElements(elements []structurizrElement) {
	for i := range elements {
		elements[i].Tags = compactTags(elements[i].Tags)
		elements[i].Properties = compactProperties(elements[i].Properties)
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func compactInstances(instances []structurizrInstance) {
	for i := range instances {
		instances[i].Tags = compactTags(instances[i].Tags)
		instances[i].Properties = compactProperties(instances[i].Properties)
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
func compactNodes(nodes []structurizrDeploymentNode) {
	for i := range nodes {
		nodes[i].Tags = compactTags(nodes[i].Tags)
		nodes[i].Properties = compactProperties(nodes[i].Properties)
		compactInstances(nodes[i].ContainerInstances)
		compactInstances(nodes[i].SoftwareSystemInstances)
	}
}

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
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

// TRLC-LINKS: REQ-EMG-005
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, DO-STRUCTURIZR-DSL
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
