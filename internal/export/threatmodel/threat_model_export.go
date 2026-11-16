// ENGMODEL-OWNER-UNIT: FU-THREAT-EXPORTER
package threatmodel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type Format string

const (
	FormatThreatDragonV2 Format = "threat-dragon-v2"
	FormatOpenOTM        Format = "open-otm"
)

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type Options struct {
	Format Format
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type Result struct {
	JSON        string
	Diagnostics []validate.Diagnostic
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateFromFile(architecturePath string, options Options) (Result, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return Result{}, err
	}
	projection, _, err := composition.ResolveCompositionProjection(bundle)
	if err != nil {
		return Result{}, err
	}
	return generateThreatModelExport(bundle, options, projection)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-004, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func Generate(bundle model.Bundle, options Options) (Result, error) {
	return generateThreatModelExport(bundle, options, composition.CompositionProjection{})
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func generateThreatModelExport(bundle model.Bundle, options Options, projection composition.CompositionProjection) (Result, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return Result{}, err
	}
	bundle = canonical.Documents()
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return Result{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}
	bundle = projection.EnrichBundle(bundle, "architecture", "behavior", "assurance")

	format := strings.TrimSpace(string(options.Format))
	if format == "" {
		format = string(FormatThreatDragonV2)
	}

	var doc any
	switch Format(format) {
	case FormatThreatDragonV2:
		doc = buildThreatDragonV2(bundle)
	case FormatOpenOTM:
		doc = buildOpenOTM(bundle)
	default:
		return Result{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("unknown threat model export format %q", format)
	}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return Result{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	return Result{JSON: string(b) + "\n", Diagnostics: validate.SortDiagnostics(diags)}, nil
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type tdv2Document struct {
	Version string      `json:"version"`
	Summary tdv2Summary `json:"summary"`
	Detail  tdv2Detail  `json:"detail"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type tdv2Summary struct {
	Title       string `json:"title"`
	Owner       string `json:"owner,omitempty"`
	Description string `json:"description,omitempty"`
	ID          int    `json:"id"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type tdv2Detail struct {
	Contributors []tdv2Contributor `json:"contributors"`
	Diagrams     []tdv2Diagram     `json:"diagrams"`
	DiagramTop   int               `json:"diagramTop"`
	Reviewer     string            `json:"reviewer"`
	ThreatTop    int               `json:"threatTop"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type tdv2Contributor struct {
	Name string `json:"name"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type tdv2Diagram struct {
	Description string     `json:"description,omitempty"`
	DiagramType string     `json:"diagramType"`
	ID          int        `json:"id"`
	Placeholder string     `json:"placeholder,omitempty"`
	Thumbnail   string     `json:"thumbnail"`
	Title       string     `json:"title"`
	Version     string     `json:"version"`
	Cells       []tdv2Cell `json:"cells,omitempty"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
type tdv2Cell struct {
	ID        string               `json:"id"`
	Shape     string               `json:"shape"`
	ZIndex    int                  `json:"zIndex"`
	Position  map[string]float64   `json:"position,omitempty"`
	Size      map[string]float64   `json:"size,omitempty"`
	Attrs     map[string]any       `json:"attrs,omitempty"`
	Data      map[string]any       `json:"data,omitempty"`
	Source    map[string]any       `json:"source,omitempty"`
	Target    map[string]any       `json:"target,omitempty"`
	Labels    []map[string]any     `json:"labels,omitempty"`
	Vertices  []map[string]float64 `json:"vertices,omitempty"`
	Connector string               `json:"connector,omitempty"`
}

// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func buildThreatDragonV2(bundle model.Bundle) tdv2Document {
	b := tdBuilder{modelID: bundle.Architecture.Model.ID, a: bundle.Architecture.AuthoredArchitecture, threatsByEntity: map[string][]map[string]any{}}
	b.indexThreats()
	nodes := b.nodes()
	cells, cellIDByNode := b.nodeCells(nodes)
	cells = append(cells, b.boundaryCells(len(cells)+1)...)
	flowZ := len(cells) + 10
	mappingFlows := b.mappingFlowCells(cellIDByNode, flowZ)
	cells = append(cells, mappingFlows...)
	cells = append(cells, b.authoredFlowCells(cellIDByNode, flowZ+len(mappingFlows))...)

	title := textutil.NonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID))
	return tdv2Document{
		Version: "2.0",
		Summary: tdv2Summary{
			Title:       title,
			Owner:       "engineering-model-go",
			Description: strings.TrimSpace(bundle.Architecture.Model.Introduction),
			ID:          0,
		},
		Detail: tdv2Detail{
			Contributors: []tdv2Contributor{{Name: "engineering-model-go"}},
			Diagrams: []tdv2Diagram{{
				Description: "Architecture-derived threat model diagram.",
				DiagramType: "STRIDE",
				ID:          0,
				Placeholder: "Generated from authored architecture.",
				Thumbnail:   "./public/content/images/thumbnail.stride.jpg",
				Title:       title + " Threat Model",
				Version:     "2.0",
				Cells:       cells,
			}},
			DiagramTop: 1,
			Reviewer:   "engineering-model-go",
			ThreatTop:  len(b.a.ThreatScenarios),
		},
	}
}

// tdNode is one architecture element drawn as a Threat Dragon cell.
//
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type tdNode struct {
	ID      string
	Name    string
	Kind    string
	Desc    string
	Threats []map[string]any
}

// tdBuilder carries the authored architecture and the threats indexed by the
// element or flow they apply to while a Threat Dragon document is built.
//
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type tdBuilder struct {
	modelID         string
	a               model.AuthoredArchitecture
	threatsByEntity map[string][]map[string]any
}

// tdEnds are the cell IDs a flow connects.
//
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type tdEnds struct {
	from string
	to   string
}

// indexThreats builds one Threat Dragon threat per scenario and files it under
// every element and flow it applies to.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) indexThreats() {
	for number, ts := range b.a.ThreatScenarios {
		th := b.threatEntry(ts, number)
		for _, id := range ts.AppliesTo {
			b.addThreat(id, th)
		}
		for _, id := range ts.FlowRefs {
			b.addThreat(id, th)
		}
	}
}

// addThreat files the threat under the trimmed, non-empty entity ID.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) addThreat(id string, threat map[string]any) {
	id = strings.TrimSpace(id)
	if id != "" {
		b.threatsByEntity[id] = append(b.threatsByEntity[id], threat)
	}
}

// threatEntry renders one threat scenario as a Threat Dragon threat.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) threatEntry(ts model.ThreatScenario, number int) map[string]any {
	return map[string]any{
		"description": textutil.NonEmpty(strings.TrimSpace(ts.Summary), strings.TrimSpace(ts.Title)),
		"mitigation":  b.mitigationText(ts),
		"modelType":   "STRIDE",
		"number":      number,
		"score":       strings.ToLower(textutil.NonEmpty(strings.TrimSpace(ts.Severity), strings.TrimSpace(ts.Impact))),
		"severity":    tdSeverity(ts.Severity, ts.Impact),
		"status":      tdStatus(ts.Status),
		"threatId":    textutil.DeterministicUUID("td-threat|" + b.modelID + "|" + ts.ID),
		"title":       textutil.NonEmpty(strings.TrimSpace(ts.Title), strings.TrimSpace(ts.ID)),
		"type":        textutil.NonEmpty(strings.TrimSpace(ts.Stride), strings.TrimSpace(ts.Category)),
	}
}

// mitigationText joins the authored mitigations of a scenario, or says that
// none is authored.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) mitigationText(ts model.ThreatScenario) string {
	mitigations := []string{}
	for _, m := range b.a.ThreatMitigations {
		if strings.TrimSpace(m.ThreatScenarioRef) == strings.TrimSpace(ts.ID) {
			mitigations = append(mitigations, tdMitigationSentence(m))
		}
	}
	if len(mitigations) == 0 {
		mitigations = append(mitigations, "No mitigation is authored for threat "+strings.TrimSpace(ts.ID)+".")
	}
	return strings.Join(mitigations, " ")
}

// tdMitigationSentence is the mitigation's notes, or a generated description.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func tdMitigationSentence(m model.ThreatMitigation) string {
	if strings.TrimSpace(m.Notes) != "" {
		return strings.TrimSpace(m.Notes)
	}
	details := []string{"Mitigation " + strings.TrimSpace(m.ID)}
	if controlRef := strings.TrimSpace(m.ControlRef); controlRef != "" {
		details = append(details, "applies control "+controlRef)
	}
	if mitigationStatus := strings.TrimSpace(m.Status); mitigationStatus != "" {
		details = append(details, "status "+mitigationStatus)
	}
	if effectiveness := strings.TrimSpace(m.Effectiveness); effectiveness != "" {
		details = append(details, "effectiveness "+effectiveness)
	}
	return strings.Join(details, "; ") + "."
}

// nodes lists the actors, functional units, data objects and interfaces as
// diagram nodes sorted by ID.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) nodes() []tdNode {
	a := b.a
	nodes := []tdNode{}
	for _, x := range a.Actors {
		nodes = append(nodes, tdNode{ID: x.ID, Name: textutil.NonEmpty(x.Name, x.ID), Kind: "actor", Desc: strings.TrimSpace(x.Description), Threats: b.threatsByEntity[x.ID]})
	}
	for _, x := range a.FunctionalUnits {
		nodes = append(nodes, tdNode{ID: x.ID, Name: textutil.NonEmpty(x.Name, x.ID), Kind: "process", Desc: strings.TrimSpace(x.Prose), Threats: b.threatsByEntity[x.ID]})
	}
	for _, x := range a.DataObjects {
		nodes = append(nodes, tdNode{ID: x.ID, Name: textutil.NonEmpty(x.Name, x.ID), Kind: "store", Desc: strings.TrimSpace(x.SchemaRef), Threats: b.threatsByEntity[x.ID]})
	}
	for _, x := range a.Interfaces {
		nodes = append(nodes, tdNode{ID: x.ID, Name: textutil.NonEmpty(x.Name, x.ID), Kind: "process", Desc: strings.TrimSpace(x.Protocol + " " + x.Endpoint), Threats: b.threatsByEntity[x.ID]})
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes
}

// nodeCells lays the nodes out on a four-column grid and returns the cells and
// the cell ID of every node.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) nodeCells(nodes []tdNode) ([]tdv2Cell, map[string]string) {
	cellIDByNode := map[string]string{}
	cells := []tdv2Cell{}
	for i, n := range nodes {
		cellUUID := textutil.DeterministicUUID("td-cell|" + b.modelID + "|" + n.ID)
		cellIDByNode[n.ID] = cellUUID
		cells = append(cells, tdNodeCell(cellUUID, i, n))
	}
	return cells, cellIDByNode
}

// tdNodeStyle returns the Threat Dragon shape, type, size and attributes of a node kind.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func tdNodeStyle(kind, name string) (shape, tmType string, size map[string]float64, attrs map[string]any) {
	line := func() map[string]any {
		return map[string]any{"stroke": "#333333", "strokeWidth": 1.0, "strokeDasharray": ""}
	}
	switch kind {
	case "actor":
		return "actor", "tm.Actor", map[string]float64{"width": 160, "height": 80}, map[string]any{"text": map[string]any{"text": name}, "body": line()}
	case "store":
		return "store", "tm.Store", map[string]float64{"width": 180, "height": 90}, map[string]any{"text": map[string]any{"text": name}, "topLine": line(), "bottomLine": line()}
	}
	return "process", "tm.Process", map[string]float64{"width": 180, "height": 90}, map[string]any{"text": map[string]any{"text": name}, "body": line()}
}

// tdNodeCell renders the node at grid index i as a cell with the given ID.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func tdNodeCell(id string, i int, n tdNode) tdv2Cell {
	shape, tmType, size, attrs := tdNodeStyle(n.Kind, n.Name)
	return tdv2Cell{
		ID:     id,
		Shape:  shape,
		ZIndex: i + 1,
		Position: map[string]float64{
			"x": float64(60 + (i%4)*260),
			"y": float64(60 + (i/4)*170),
		},
		Size:  size,
		Attrs: attrs,
		Data: map[string]any{
			"name":                   n.Name,
			"description":            n.Desc,
			"type":                   tmType,
			"isTrustBoundary":        false,
			"outOfScope":             false,
			"reasonOutOfScope":       "",
			"threats":                n.Threats,
			"hasOpenThreats":         tdHasOpenThreats(n.Threats),
			"providesAuthentication": false,
			"isALog":                 false,
			"storesCredentials":      false,
			"isEncrypted":            false,
			"isSigned":               false,
		},
	}
}

// boundaryCells renders the trust boundaries, numbering their z-index from firstZ.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) boundaryCells(firstZ int) []tdv2Cell {
	var cells []tdv2Cell
	tbZ := firstZ
	for _, tb := range b.a.TrustBoundaries {
		cells = append(cells, tdv2Cell{
			ID:       textutil.DeterministicUUID("td-boundary|" + b.modelID + "|" + tb.ID),
			Shape:    "trust-boundary-box",
			ZIndex:   tbZ,
			Size:     map[string]float64{"width": 260, "height": 180},
			Position: map[string]float64{"x": float64(30 + (tbZ%3)*300), "y": float64(30 + (tbZ/3)*220)},
			Attrs:    map[string]any{"text": map[string]any{"text": textutil.NonEmpty(tb.Name, tb.ID)}, "body": map[string]any{"stroke": "#999999", "strokeWidth": 1.0, "strokeDasharray": "3 3"}},
			Data:     map[string]any{"type": "tm.Boundary", "name": textutil.NonEmpty(tb.Name, tb.ID), "description": strings.TrimSpace(tb.Description), "isTrustBoundary": true, "hasOpenThreats": false},
		})
		tbZ++
	}
	return cells
}

// tdFlowMappingTypes are the mapping types drawn as flows.
var tdFlowMappingTypes = map[string]bool{
	"calls": true, "publishes": true, "subscribes": true, "reads": true, "writes": true, "streams": true, "interacts_with": true,
}

// tdFlowLabels is the single centered label of a flow cell.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func tdFlowLabels(name string) []map[string]any {
	return []map[string]any{{"position": 0.5, "attrs": map[string]any{"label": map[string]any{"text": name}}}}
}

// tdFlowAttrs is the line style of a flow cell with the given stroke color.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func tdFlowAttrs(stroke string) map[string]any {
	return map[string]any{"line": map[string]any{"stroke": stroke, "strokeWidth": 1.0, "targetMarker": map[string]any{"name": "block"}, "strokeDasharray": ""}}
}

// mappingFlowCells renders the flow-like mappings between known cells, numbering
// their z-index from firstZ.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) mappingFlowCells(cellIDByNode map[string]string, firstZ int) []tdv2Cell {
	var cells []tdv2Cell
	for _, m := range b.a.Mappings {
		t := strings.TrimSpace(m.Type)
		if !tdFlowMappingTypes[t] {
			continue
		}
		ends := tdEnds{from: cellIDByNode[strings.TrimSpace(m.From)], to: cellIDByNode[strings.TrimSpace(m.To)]}
		if ends.from == "" || ends.to == "" {
			continue
		}
		cells = append(cells, b.mappingFlowCell(m, ends, firstZ+len(cells)))
	}
	return cells
}

// mappingFlowCell renders one mapping as a flow cell.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) mappingFlowCell(m model.Mapping, ends tdEnds, z int) tdv2Cell {
	name := strings.TrimSpace(m.Description)
	if name == "" {
		name = strings.TrimSpace(m.Type)
	}
	return tdv2Cell{
		ID:        textutil.DeterministicUUID("td-flow|" + b.modelID + "|" + m.Type + "|" + m.From + "|" + m.To),
		Shape:     "flow",
		ZIndex:    z,
		Connector: "smooth",
		Attrs:     tdFlowAttrs("#333333"),
		Data: map[string]any{
			"type":             "tm.Flow",
			"name":             name,
			"description":      strings.TrimSpace(m.Description),
			"outOfScope":       false,
			"reasonOutOfScope": "",
			"protocol":         "",
			"isEncrypted":      false,
			"isPublicNetwork":  false,
			"hasOpenThreats":   false,
			"threats":          b.threatsByEntity[m.From],
			"isTrustBoundary":  false,
		},
		Source: map[string]any{"cell": ends.from},
		Target: map[string]any{"cell": ends.to},
		Labels: tdFlowLabels(name),
	}
}

// authoredFlowCells renders the authored flows between known cells, numbering
// their z-index from firstZ.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) authoredFlowCells(cellIDByNode map[string]string, firstZ int) []tdv2Cell {
	var cells []tdv2Cell
	for _, f := range b.a.Flows {
		ends := tdEnds{from: cellIDByNode[strings.TrimSpace(f.SourceRef)], to: cellIDByNode[strings.TrimSpace(f.DestinationRef)]}
		if ends.from == "" || ends.to == "" {
			continue
		}
		cells = append(cells, b.authoredFlowCell(f, ends, firstZ+len(cells)))
	}
	return cells
}

// authoredFlowCell renders one authored flow as a flow cell.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
func (b *tdBuilder) authoredFlowCell(f model.Flow, ends tdEnds, z int) tdv2Cell {
	name := textutil.NonEmpty(strings.TrimSpace(f.Title), strings.TrimSpace(f.ID))
	encryption := strings.ToLower(strings.TrimSpace(f.EncryptionInTransit))
	direction := strings.ToLower(strings.TrimSpace(f.Direction))
	return tdv2Cell{
		ID:        textutil.DeterministicUUID("td-flow-authored|" + b.modelID + "|" + f.ID),
		Shape:     "flow",
		ZIndex:    z,
		Connector: "smooth",
		Attrs:     tdFlowAttrs("#555555"),
		Data: map[string]any{
			"type":             "tm.Flow",
			"name":             name,
			"description":      strings.TrimSpace(f.Description),
			"outOfScope":       false,
			"reasonOutOfScope": "",
			"protocol":         strings.TrimSpace(f.Protocol),
			"isEncrypted":      strings.Contains(encryption, "tls") || strings.Contains(encryption, "encrypt"),
			"isPublicNetwork":  direction == "inbound" || direction == "outbound",
			"hasOpenThreats":   tdHasOpenThreats(b.threatsByEntity[f.ID]),
			"threats":          b.threatsByEntity[f.ID],
			"isTrustBoundary":  false,
		},
		Source: map[string]any{"cell": ends.from},
		Target: map[string]any{"cell": ends.to},
		Labels: tdFlowLabels(name),
	}
}

// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
func tdSeverity(in ...string) string {
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "high":
			return "High"
		case "medium":
			return "Medium"
		case "low":
			return "Low"
		}
	}
	return "Medium"
}

// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
func tdStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved", "closed", "verified", "completed", "mitigated":
		return "Mitigated"
	default:
		return "Open"
	}
}

// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
func tdHasOpenThreats(threats []map[string]any) bool {
	for _, t := range threats {
		if strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%v", t["status"])), "open") {
			return true
		}
	}
	return false
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type otmDocument struct {
	OTMVersion      string              `json:"otmVersion"`
	Project         otmProject          `json:"project"`
	Representations []otmRepresentation `json:"representations,omitempty"`
	Assets          []otmAsset          `json:"assets,omitempty"`
	TrustZones      []otmTrustZone      `json:"trustZones"`
	Components      []otmComponent      `json:"components,omitempty"`
	Dataflows       []otmDataflow       `json:"dataflows"`
	Threats         []otmThreat         `json:"threats,omitempty"`
	Mitigations     []otmMitigation     `json:"mitigations,omitempty"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type otmProject struct {
	Name        string         `json:"name"`
	ID          string         `json:"id"`
	Description string         `json:"description,omitempty"`
	Owner       string         `json:"owner,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Attributes  map[string]any `json:"attributes,omitempty"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type otmRepresentation struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Type string `json:"type"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type otmAsset struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Risk struct {
		Confidentiality float64 `json:"confidentiality"`
		Integrity       float64 `json:"integrity"`
		Availability    float64 `json:"availability"`
		Comment         string  `json:"comment,omitempty"`
	} `json:"risk"`
	Description string `json:"description,omitempty"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
type otmTrustZone struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Risk        struct {
		TrustRating float64 `json:"trustRating"`
	} `json:"risk"`
	Parent *struct {
		TrustZone string `json:"trustZone"`
	} `json:"parent,omitempty"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
type otmComponent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Parent      struct {
		TrustZone string `json:"trustZone"`
	} `json:"parent"`
	Threats []otmThreatRef `json:"threats,omitempty"`
	Tags    []string       `json:"tags,omitempty"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type otmDataflow struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Bidirectional bool           `json:"bidirectional,omitempty"`
	Source        string         `json:"source"`
	Destination   string         `json:"destination"`
	Assets        []string       `json:"assets,omitempty"`
	Threats       []otmThreatRef `json:"threats,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type otmThreat struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	CWEs        []string `json:"cwes,omitempty"`
	Risk        struct {
		Likelihood        float64 `json:"likelihood,omitempty"`
		LikelihoodComment string  `json:"likelihoodComment,omitempty"`
		Impact            float64 `json:"impact"`
		ImpactComment     string  `json:"impactComment"`
	} `json:"risk"`
	Tags []string `json:"tags,omitempty"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type otmMitigation struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description,omitempty"`
	RiskReduction float64 `json:"riskReduction"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type otmThreatRef struct {
	Threat      string             `json:"threat"`
	State       string             `json:"state"`
	Mitigations []otmMitigationRef `json:"mitigations,omitempty"`
}

// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
type otmMitigationRef struct {
	Mitigation string `json:"mitigation"`
	State      string `json:"state"`
}

// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func buildOpenOTM(bundle model.Bundle) otmDocument {
	b := otmBuilder{a: bundle.Architecture.AuthoredArchitecture, modelID: bundle.Architecture.Model.ID, componentIDs: map[string]bool{}}
	doc := otmDocument{
		OTMVersion: "0.2.0",
		Project: otmProject{
			Name:        textutil.NonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID)),
			ID:          strings.TrimSpace(bundle.Architecture.Model.ID),
			Description: strings.TrimSpace(bundle.Architecture.Model.Introduction),
			Owner:       "engineering-model-go",
			Tags:        []string{"generated", "engineering-model"},
		},
		Representations: []otmRepresentation{{Name: "Architecture Threat View", ID: "rep-architecture", Type: "dfd"}},
		Components:      []otmComponent{},
		Dataflows:       []otmDataflow{},
		Threats:         []otmThreat{},
		Mitigations:     []otmMitigation{},
	}
	doc.TrustZones = b.trustZones()
	doc.Components = b.components()
	doc.Assets = b.assets()
	doc.Mitigations, b.mitigationByScenario = b.mitigations()
	doc.Threats = b.threats()
	b.attachComponentThreats(doc.Components)
	doc.Dataflows = b.dataflows()
	return doc
}

// otmBuilder carries the authored architecture and the lookups shared by the
// sections of an Open Threat Model document.
//
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
type otmBuilder struct {
	a                    model.AuthoredArchitecture
	modelID              string
	componentIDs         map[string]bool
	mitigationByScenario map[string][]otmMitigationRef
}

// trustZones lists the default zone (when not authored) and the authored
// trust boundaries, sorted by ID.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) trustZones() []otmTrustZone {
	zones := []otmTrustZone{}
	if !hasDefaultTrustZone(b.a.TrustBoundaries) {
		defaultTZ := otmTrustZone{ID: "TZ-SYSTEM", Name: "System Boundary", Type: "system", Description: "Default trust zone for components without explicit boundary mapping."}
		defaultTZ.Risk.TrustRating = 50
		zones = append(zones, defaultTZ)
	}
	for _, tb := range b.a.TrustBoundaries {
		z := otmTrustZone{ID: tb.ID, Name: textutil.NonEmpty(tb.Name, tb.ID), Type: strings.TrimSpace(tb.BoundaryType), Description: strings.TrimSpace(tb.Description)}
		z.Risk.TrustRating = 50
		if strings.TrimSpace(tb.ParentRef) != "" {
			z.Parent = &struct {
				TrustZone string `json:"trustZone"`
			}{TrustZone: strings.TrimSpace(tb.ParentRef)}
		}
		zones = append(zones, z)
	}
	sort.SliceStable(zones, func(i, j int) bool { return zones[i].ID < zones[j].ID })
	return zones
}

// hasDefaultTrustZone reports whether a trust boundary already is TZ-SYSTEM.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func hasDefaultTrustZone(boundaries []model.TrustBoundary) bool {
	for _, tb := range boundaries {
		if strings.TrimSpace(tb.ID) == "TZ-SYSTEM" {
			return true
		}
	}
	return false
}

// trimmedNonEmpty returns the values trimmed, without the blank ones.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func trimmedNonEmpty(values []string) []string {
	var out []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// zoneForID maps each element to its trust zone: boundary members first, then
// bounded_by mappings.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) zoneForID() map[string]string {
	zoneForID := map[string]string{}
	for _, tb := range b.a.TrustBoundaries {
		for _, member := range trimmedNonEmpty(tb.Members) {
			zoneForID[member] = tb.ID
		}
	}
	b.applyBoundedBy(zoneForID)
	return zoneForID
}

// applyBoundedBy lets bounded_by mappings override the zone of their source.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) applyBoundedBy(zoneForID map[string]string) {
	for _, m := range b.a.Mappings {
		if strings.TrimSpace(m.Type) == "bounded_by" {
			zoneForID[strings.TrimSpace(m.From)] = strings.TrimSpace(m.To)
		}
	}
}

// otmComponentSpec is one element to add as a component.
//
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
type otmComponentSpec struct {
	ID   string
	Name string
	Kind string
	Desc string
}

// componentSpecs lists the elements that become components, in authored order.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) componentSpecs() []otmComponentSpec {
	a := b.a
	var specs []otmComponentSpec
	for _, x := range a.Actors {
		specs = append(specs, otmComponentSpec{x.ID, x.Name, "actor", x.Description})
	}
	for _, x := range a.FunctionalUnits {
		specs = append(specs, otmComponentSpec{x.ID, x.Name, "service", x.Prose})
	}
	for _, x := range a.Interfaces {
		specs = append(specs, otmComponentSpec{x.ID, x.Name, "interface", x.Protocol + " " + x.Endpoint})
	}
	for _, x := range a.DeploymentTargets {
		specs = append(specs, otmComponentSpec{x.ID, x.Name, "deployment-target", x.Environment + " " + x.Region})
	}
	for _, x := range a.ReferencedElements {
		specs = append(specs, otmComponentSpec{x.ID, x.Name, x.Kind, x.Layer})
	}
	return specs
}

// components builds the components sorted by ID and records their IDs; the
// first spec of an ID wins and an empty ID is skipped.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) components() []otmComponent {
	zoneForID := b.zoneForID()
	components := []otmComponent{}
	for _, spec := range b.componentSpecs() {
		id := strings.TrimSpace(spec.ID)
		if id == "" || b.componentIDs[id] {
			continue
		}
		b.componentIDs[id] = true
		c := otmComponent{ID: id, Name: textutil.NonEmpty(strings.TrimSpace(spec.Name), id), Type: strings.TrimSpace(spec.Kind), Description: strings.TrimSpace(spec.Desc)}
		c.Parent.TrustZone = textutil.NonEmpty(strings.TrimSpace(zoneForID[id]), "TZ-SYSTEM")
		components = append(components, c)
	}
	sort.SliceStable(components, func(i, j int) bool { return components[i].ID < components[j].ID })
	return components
}

// assets builds one asset per data object, sorted by ID.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) assets() []otmAsset {
	var assets []otmAsset
	for _, x := range b.a.DataObjects {
		asset := otmAsset{ID: x.ID, Name: textutil.NonEmpty(x.Name, x.ID), Description: strings.TrimSpace(x.SchemaRef)}
		asset.Risk.Confidentiality = scoreLevel(textutil.NonEmpty(strings.TrimSpace(x.Confidentiality), strings.TrimSpace(x.Sensitivity)))
		asset.Risk.Integrity = scoreLevel(strings.TrimSpace(x.Integrity))
		asset.Risk.Availability = scoreLevel(strings.TrimSpace(x.Availability))
		asset.Risk.Comment = strings.Join(x.RegulatoryTags, ",")
		assets = append(assets, asset)
	}
	sort.SliceStable(assets, func(i, j int) bool { return assets[i].ID < assets[j].ID })
	return assets
}

// mitigations builds the mitigations sorted by ID and the mitigation
// references indexed by threat scenario.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) mitigations() ([]otmMitigation, map[string][]otmMitigationRef) {
	mitigations := []otmMitigation{}
	mitigationByScenario := map[string][]otmMitigationRef{}
	for _, m := range b.a.ThreatMitigations {
		mid := strings.TrimSpace(m.ID)
		if mid == "" {
			continue
		}
		mitigations = append(mitigations, otmMitigation{
			ID:            mid,
			Name:          textutil.NonEmpty(strings.TrimSpace(m.ControlRef), mid),
			Description:   textutil.NonEmpty(strings.TrimSpace(m.Notes), strings.TrimSpace(m.ControlRef)),
			RiskReduction: scoreLevel(strings.TrimSpace(m.Effectiveness)),
		})
		if sid := strings.TrimSpace(m.ThreatScenarioRef); sid != "" {
			mitigationByScenario[sid] = append(mitigationByScenario[sid], otmMitigationRef{Mitigation: mid, State: otmState(m.Status)})
		}
	}
	sort.SliceStable(mitigations, func(i, j int) bool { return mitigations[i].ID < mitigations[j].ID })
	return mitigations, mitigationByScenario
}

// threats builds one threat per scenario with an ID, sorted by ID.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) threats() []otmThreat {
	threats := []otmThreat{}
	for _, ts := range b.a.ThreatScenarios {
		t := otmThreat{ID: strings.TrimSpace(ts.ID), Name: textutil.NonEmpty(strings.TrimSpace(ts.Title), strings.TrimSpace(ts.ID)), Description: strings.TrimSpace(ts.Summary)}
		if c := strings.TrimSpace(ts.Category); c != "" {
			t.Categories = append(t.Categories, c)
		}
		t.CWEs = append(t.CWEs, ts.CWE...)
		t.Risk.Likelihood = scoreLevel(strings.TrimSpace(ts.Likelihood))
		t.Risk.LikelihoodComment = strings.TrimSpace(ts.Status)
		t.Risk.Impact = scoreLevel(strings.TrimSpace(ts.Impact))
		t.Risk.ImpactComment = textutil.NonEmpty(strings.TrimSpace(ts.Severity), strings.TrimSpace(ts.Category))
		t.Tags = []string{textutil.NonEmpty(strings.TrimSpace(ts.Stride), "threat")}
		if t.ID != "" {
			threats = append(threats, t)
		}
	}
	sort.SliceStable(threats, func(i, j int) bool { return threats[i].ID < threats[j].ID })
	return threats
}

// attachComponentThreats adds a threat reference to every component a scenario applies to.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) attachComponentThreats(components []otmComponent) {
	componentIndex := map[string]int{}
	for i, c := range components {
		componentIndex[c.ID] = i
	}
	for _, ts := range b.a.ThreatScenarios {
		ref := otmThreatRef{Threat: ts.ID, State: otmState(ts.Status), Mitigations: b.mitigationByScenario[ts.ID]}
		for _, idx := range componentIndexes(componentIndex, ts.AppliesTo) {
			components[idx].Threats = append(components[idx].Threats, ref)
		}
	}
}

// componentIndexes returns the indexes of the known components among the trimmed IDs.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func componentIndexes(componentIndex map[string]int, ids []string) []int {
	var indexes []int
	for _, id := range ids {
		if idx, ok := componentIndex[strings.TrimSpace(id)]; ok {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

// otmFlowAssets maps each flow ID to the sorted unique data objects it carries.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) flowAssets() map[string][]string {
	assetByFlowID := map[string][]string{}
	for _, f := range b.a.Flows {
		if assets := trimmedNonEmpty(f.DataRefs); len(assets) > 0 {
			assetByFlowID[strings.TrimSpace(f.ID)] = textutil.UniqueSorted(assets)
		}
	}
	return assetByFlowID
}

// otmFlowMappingTypes are the mapping types exported as dataflows.
var otmFlowMappingTypes = map[string]bool{
	"calls": true, "publishes": true, "subscribes": true, "reads": true, "writes": true, "streams": true, "interacts_with": true, "depends_on": true,
}

// dataflows builds the dataflows of flow-like mappings and authored flows
// between known components, sorted by ID.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) dataflows() []otmDataflow {
	dataflows := b.mappingDataflows()
	dataflows = append(dataflows, b.authoredDataflows()...)
	sort.SliceStable(dataflows, func(i, j int) bool { return dataflows[i].ID < dataflows[j].ID })
	return dataflows
}

// mappingDataflows builds a dataflow per flow-like mapping between known components.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) mappingDataflows() []otmDataflow {
	dataflows := []otmDataflow{}
	for _, m := range b.a.Mappings {
		t := strings.TrimSpace(m.Type)
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		if !otmFlowMappingTypes[t] || !b.componentIDs[from] || !b.componentIDs[to] {
			continue
		}
		dataflows = append(dataflows, otmDataflow{
			ID:            textutil.DeterministicUUID("otm-flow|" + b.modelID + "|" + t + "|" + from + "|" + to),
			Name:          textutil.NonEmpty(strings.TrimSpace(m.Description), t),
			Description:   strings.TrimSpace(m.Description),
			Source:        from,
			Destination:   to,
			Bidirectional: t == "interacts_with" || t == "depends_on",
			Tags:          []string{t},
		})
	}
	return dataflows
}

// authoredDataflows builds a dataflow per authored flow between known components.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) authoredDataflows() []otmDataflow {
	var dataflows []otmDataflow
	assetByFlowID := b.flowAssets()
	for _, f := range b.a.Flows {
		from := strings.TrimSpace(f.SourceRef)
		to := strings.TrimSpace(f.DestinationRef)
		if !b.componentIDs[from] || !b.componentIDs[to] {
			continue
		}
		dataflows = append(dataflows, otmDataflow{
			ID:            strings.TrimSpace(f.ID),
			Name:          textutil.NonEmpty(strings.TrimSpace(f.Title), strings.TrimSpace(f.ID)),
			Description:   strings.TrimSpace(f.Description),
			Source:        from,
			Destination:   to,
			Bidirectional: strings.EqualFold(strings.TrimSpace(f.Direction), "bidirectional"),
			Assets:        assetByFlowID[strings.TrimSpace(f.ID)],
			Threats:       b.flowThreatRefs(f),
			Tags:          []string{textutil.NonEmpty(strings.TrimSpace(f.Kind), "flow")},
		})
	}
	return dataflows
}

// flowThreatRefs lists the threat references of a flow's authored scenarios.
//
// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS, DO-OPEN-OTM-JSON
func (b *otmBuilder) flowThreatRefs(f model.Flow) []otmThreatRef {
	var refs []otmThreatRef
	for _, scenarioID := range f.Threats {
		scenarioID = strings.TrimSpace(scenarioID)
		if scenarioID == "" {
			continue
		}
		ts := findThreatScenarioByID(b.a.ThreatScenarios, scenarioID)
		refs = append(refs, otmThreatRef{Threat: scenarioID, State: otmState(ts.Status), Mitigations: b.mitigationByScenario[scenarioID]})
	}
	return refs
}

// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
func otmState(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved", "closed", "completed", "verified", "mitigated", "implemented":
		return "mitigated"
	case "partial":
		return "partial"
	default:
		return "open"
	}
}

// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
func scoreLevel(level string) float64 {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "high":
		return 8
	case "medium":
		return 5
	case "low":
		return 2
	default:
		return 5
	}
}

// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
// ENGMODEL-LINKS: FU-THREAT-EXPORTER, DO-THREAT-DRAGON-JSON
func findThreatScenarioByID(all []model.ThreatScenario, id string) model.ThreatScenario {
	id = strings.TrimSpace(id)
	for _, ts := range all {
		if strings.TrimSpace(ts.ID) == id {
			return ts
		}
	}
	return model.ThreatScenario{}
}
