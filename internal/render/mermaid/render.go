// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package mermaid

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/labeth/engineering-model-go/internal/render/diagramstyle"
	"github.com/labeth/engineering-model-go/internal/view"
)

//go:embed templates/diagram.tmpl
var diagramTemplateText string

var diagramTemplate = template.Must(template.New("mermaid-diagram").Parse(diagramTemplateText))
var nonID = regexp.MustCompile(`[^a-zA-Z0-9_]`)

type edgeLine struct {
	From  string
	Label string
	To    string
}

type diagramTemplateData struct {
	ViewID    string
	ViewKind  string
	Nodes     []string
	Edges     []edgeLine
	ClassDefs []string
}

// kindRenderers are the renderers of the views that have their own diagram
// form, by view kind; every other kind is rendered by renderFlowchart.
var kindRenderers = map[string]func(view.ProjectedView) string{
	"use-case": renderUseCase,
	"logical": func(v view.ProjectedView) string {
		return renderConcernFlowchart(v, "TB", renderLogicalNode)
	},
	"process": renderProcess,
	"physical": func(v view.ProjectedView) string {
		return renderConcernFlowchart(v, "LR", renderPhysicalNode)
	},
	"implementation": renderImplementation,
	"deployment": func(v view.ProjectedView) string {
		return renderConcernFlowchart(v, "LR", renderDeploymentNode)
	},
}

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR
func Render(v view.ProjectedView) string {
	if render, ok := kindRenderers[strings.TrimSpace(v.Kind)]; ok {
		return render(v)
	}
	return renderFlowchart(v)
}

// renderFlowchart renders the view through the shared flowchart template.
//
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR
func renderFlowchart(v view.ProjectedView) string {
	nodes := sortedNodes(v.Nodes)
	nodeLines := make([]string, 0, len(nodes))
	for _, n := range nodes {
		nodeLines = append(nodeLines, renderNode(n))
	}
	edges := sortedEdges(v.Edges)
	edgeLines := make([]edgeLine, 0, len(edges))
	for _, e := range edges {
		edgeLines = append(edgeLines, edgeLine{
			From: mermaidID(e.From), Label: escapeLabel(compactEdgeLabel(e.Type, e.Label)), To: mermaidID(e.To),
		})
	}
	data := diagramTemplateData{
		ViewID: escapeComment(v.ID), ViewKind: escapeComment(v.Kind),
		Nodes: nodeLines, Edges: edgeLines, ClassDefs: diagramstyle.MermaidClassDefs(),
	}
	var b bytes.Buffer
	if err := diagramTemplate.Execute(&b, data); err != nil {
		return "flowchart LR\n%% render_error: " + escapeComment(err.Error()) + "\n"
	}
	return b.String()
}

// TRLC-LINKS: REQ-EMG-052
func renderUseCase(v view.ProjectedView) string {
	var b strings.Builder
	fmt.Fprintln(&b, "flowchart LR")
	fmt.Fprintf(&b, "%%%% view: %s (%s)\n", escapeComment(v.ID), escapeComment(v.Kind))
	for _, node := range sortedNodes(v.Nodes) {
		fmt.Fprintln(&b, useCaseNodeLine(node))
	}
	for _, edge := range sortedEdges(v.Edges) {
		fmt.Fprintln(&b, useCaseEdgeLine(edge))
	}
	fmt.Fprintln(&b, "classDef actor fill:#fff8e1,stroke:#ef6c00,color:#7f3600;")
	fmt.Fprintln(&b, "classDef use_case fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20;")
	return b.String()
}

// TRLC-LINKS: REQ-EMG-052
func useCaseNodeLine(node view.Node) string {
	if node.Kind == "actor" {
		return fmt.Sprintf("%s((\"%s\")):::actor", mermaidID(node.ID), escapeLabel(node.Label))
	}
	return fmt.Sprintf("%s([\"%s\"]):::use_case", mermaidID(node.ID), escapeLabel(node.Label))
}

// TRLC-LINKS: REQ-EMG-052
func useCaseEdgeLine(edge view.Edge) string {
	if edge.Label == "include" {
		return fmt.Sprintf("%s -.->|include| %s", mermaidID(edge.From), mermaidID(edge.To))
	}
	return fmt.Sprintf("%s -->|%s| %s", mermaidID(edge.From), escapeLabel(edge.Label), mermaidID(edge.To))
}

// TRLC-LINKS: REQ-EMG-052
func renderProcess(v view.ProjectedView) string {
	var b strings.Builder
	fmt.Fprintln(&b, "sequenceDiagram")
	fmt.Fprintf(&b, "%%%% view: %s (%s)\n", escapeComment(v.ID), escapeComment(v.Kind))
	fmt.Fprintln(&b, "autonumber")
	participants := map[string]view.Node{}
	for _, node := range v.Nodes {
		participants[node.ID] = node
	}
	orderedIDs := make([]string, 0, len(participants))
	for id := range participants {
		orderedIDs = append(orderedIDs, id)
	}
	sort.Strings(orderedIDs)
	for _, id := range orderedIDs {
		fmt.Fprintf(&b, "participant %s as %s\n", mermaidID(id), escapeLabel(participants[id].Label))
	}
	edges := append([]view.Edge(nil), v.Edges...)
	sort.SliceStable(edges, func(i, j int) bool { return edges[i].Sequence < edges[j].Sequence })
	for _, edge := range edges {
		label := edge.Label
		if edge.ItemRef != "" {
			label += ": " + edge.ItemRef
		}
		fmt.Fprintf(&b, "%s->>%s: %s\n", mermaidID(edge.From), mermaidID(edge.To), escapeSequenceLabel(label))
	}
	return b.String()
}

// TRLC-LINKS: REQ-EMG-052
func renderConcernFlowchart(v view.ProjectedView, direction string, nodeRenderer func(view.Node) string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "flowchart %s\n", direction)
	fmt.Fprintf(&b, "%%%% view: %s (%s)\n", escapeComment(v.ID), escapeComment(v.Kind))
	for _, node := range sortedNodes(v.Nodes) {
		fmt.Fprintln(&b, nodeRenderer(node))
	}
	for _, edge := range sortedEdges(v.Edges) {
		fmt.Fprintf(&b, "%s -->|%s| %s\n", mermaidID(edge.From), escapeLabel(edge.Label), mermaidID(edge.To))
	}
	for _, classDef := range diagramstyle.MermaidClassDefs() {
		fmt.Fprintln(&b, classDef)
	}
	return b.String()
}

// TRLC-LINKS: REQ-EMG-052
func renderLogicalNode(node view.Node) string {
	shape := "[\"" + escapeLabel(node.Label) + "\"]"
	if node.Kind == "capability" {
		shape = "([\"" + escapeLabel(node.Label) + "\"])"
	}
	return mermaidID(node.ID) + shape + ":::" + concernClass(node.Kind)
}

// TRLC-LINKS: REQ-EMG-052
func renderPhysicalNode(node view.Node) string {
	shape := "[[\"" + escapeLabel(node.Label) + "\"]]"
	if node.Kind == "port" {
		shape = "((\"" + escapeLabel(node.Label) + "\"))"
	} else if node.Kind == "quantity" {
		shape = "[/\"" + escapeLabel(node.Label) + "\"/]"
	}
	return mermaidID(node.ID) + shape + ":::" + concernClass(node.Kind)
}

// TRLC-LINKS: REQ-EMG-052
func renderDeploymentNode(node view.Node) string {
	shape := "[\"" + escapeLabel(node.Label) + "\"]"
	if node.Kind == "deployment_target" || node.Kind == "hardware" || node.Kind == "hardware_item" {
		shape = "[[\"" + escapeLabel(node.Label) + "\"]]"
	}
	return mermaidID(node.ID) + shape + ":::" + concernClass(node.Kind)
}

// TRLC-LINKS: REQ-EMG-052
func renderImplementation(v view.ProjectedView) string {
	var b strings.Builder
	fmt.Fprintln(&b, "classDiagram")
	fmt.Fprintf(&b, "%%%% view: %s (%s)\n", escapeComment(v.ID), escapeComment(v.Kind))
	for _, node := range sortedNodes(v.Nodes) {
		fmt.Fprintf(&b, "class %s[\"%s\"]\n", mermaidID(node.ID), escapeLabel(node.Label))
		for _, feature := range node.Features {
			fmt.Fprintf(&b, "%s : %s\n", mermaidID(node.ID), escapeLabel(feature))
		}
	}
	for _, edge := range sortedEdges(v.Edges) {
		fmt.Fprintf(&b, "%s --> %s : %s\n", mermaidID(edge.From), mermaidID(edge.To), escapeLabel(edge.Label))
	}
	return b.String()
}

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-052
func sortedNodes(nodes []view.Node) []view.Node {
	out := append([]view.Node(nil), nodes...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-052
func sortedEdges(edges []view.Edge) []view.Edge {
	out := append([]view.Edge(nil), edges...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return out[i].Label < out[j].Label
	})
	return out
}

// TRLC-LINKS: REQ-EMG-052
func concernClass(kind string) string {
	switch kind {
	case "capability", "logical_component":
		return "functional_unit"
	case "hardware", "hardware_item":
		return "deployment_element"
	case "software_component":
		return "code_element"
	case "deployment_target":
		return "deployment_target"
	case "interface", "port", "hardware_interface":
		return "interface"
	case "data", "quantity":
		return "data_object"
	default:
		return "unknown"
	}
}

// nodeShape is the opening and closing bracket of a node kind and its class.
type nodeShape struct{ open, close, class string }

var nodeShapes = map[string]nodeShape{
	"hardware_item":      {`[[`, `]]`, "deployment_element"},
	"hardware_interface": {`[/`, `/]`, "interface"},
	"functional_group":   {`[`, `]`, "functional_group"},
	"functional_unit":    {`[`, `]`, "functional_unit"},
	"actor":              {`((`, `))`, "actor"},
	"attack_vector":      {`((`, `))`, "attack_vector"},
	"referenced_element": {`[`, `]`, "referenced_element"},
	"interface":          {`[/`, `/]`, "interface"},
	"data_object":        {`[(`, `)]`, "data_object"},
	"deployment_target":  {`[`, `]`, "deployment_target"},
	"control":            {`[[`, `]]`, "control"},
	"trust_boundary":     {`[/`, `\]`, "trust_boundary"},
	"state":              {`([`, `])`, "state"},
	"event":              {`((`, `))`, "event"},
	"flow":               {`{`, `}`, "flow"},
	"flow_step":          {`>`, `]`, "flow_step"},
}

// TRLC-LINKS: REQ-EMG-003
func renderNode(n view.Node) string {
	shape, ok := nodeShapes[n.Kind]
	if !ok {
		shape = nodeShape{`[`, `]`, "unknown"}
	}
	return mermaidID(n.ID) + shape.open + `"` + escapeLabel(n.Label) + `"` + shape.close + ":::" + shape.class
}

// TRLC-LINKS: REQ-EMG-003
func mermaidID(raw string) string {
	r := nonID.ReplaceAllString(raw, "_")
	if r == "" {
		r = "node"
	}
	if r[0] >= '0' && r[0] <= '9' {
		r = "n_" + r
	}
	return "N_" + r
}

// TRLC-LINKS: REQ-EMG-003
func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.NewReplacer("|", "/", "[", " ", "]", " ", "(", " ", ")", " ", "{", " ", "}", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// TRLC-LINKS: REQ-EMG-052
func escapeSequenceLabel(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, ":", " -")
	return strings.Join(strings.Fields(value), " ")
}

// TRLC-LINKS: REQ-EMG-003
func escapeComment(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}

// TRLC-LINKS: REQ-EMG-003
func compactEdgeLabel(edgeType, fallback string) string {
	switch strings.TrimSpace(edgeType) {
	case "calls", "reads", "writes", "publishes", "subscribes", "streams", "contains":
		return strings.TrimSpace(edgeType)
	case "flow_async":
		return "async"
	case "flow_error":
		return "error"
	case "flow_ref":
		return "ref"
	default:
		_ = fallback
		return ""
	}
}
