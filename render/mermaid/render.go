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

	"github.com/labeth/engineering-model-go/render/diagramstyle"
	"github.com/labeth/engineering-model-go/view"
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

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR
func Render(v view.ProjectedView) string {
	switch strings.TrimSpace(v.Kind) {
	case "use-case":
		return renderUseCase(v)
	case "logical":
		return renderConcernFlowchart(v, "TB", renderLogicalNode)
	case "process":
		return renderProcess(v)
	case "physical":
		return renderConcernFlowchart(v, "LR", renderPhysicalNode)
	case "implementation":
		return renderImplementation(v)
	case "deployment":
		return renderConcernFlowchart(v, "LR", renderDeploymentNode)
	}
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
		if node.Kind == "actor" {
			fmt.Fprintf(&b, "%s((\"%s\")):::actor\n", mermaidID(node.ID), escapeLabel(node.Label))
		} else {
			fmt.Fprintf(&b, "%s([\"%s\"]):::use_case\n", mermaidID(node.ID), escapeLabel(node.Label))
		}
	}
	for _, edge := range sortedEdges(v.Edges) {
		if edge.Label == "include" {
			fmt.Fprintf(&b, "%s -.->|include| %s\n", mermaidID(edge.From), mermaidID(edge.To))
		} else {
			fmt.Fprintf(&b, "%s -->|%s| %s\n", mermaidID(edge.From), escapeLabel(edge.Label), mermaidID(edge.To))
		}
	}
	fmt.Fprintln(&b, "classDef actor fill:#fff8e1,stroke:#ef6c00,color:#7f3600;")
	fmt.Fprintln(&b, "classDef use_case fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20;")
	return b.String()
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
	if node.Kind == "deployment_target" || node.Kind == "hardware" {
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
	case "hardware":
		return "deployment_element"
	case "software_component":
		return "code_element"
	case "deployment_target":
		return "deployment_target"
	case "interface", "port":
		return "interface"
	case "data", "quantity":
		return "data_object"
	default:
		return "unknown"
	}
}

// TRLC-LINKS: REQ-EMG-003
func renderNode(n view.Node) string {
	id, label := mermaidID(n.ID), escapeLabel(n.Label)
	switch n.Kind {
	case "functional_group":
		return fmt.Sprintf("%s[\"%s\"]:::functional_group", id, label)
	case "functional_unit":
		return fmt.Sprintf("%s[\"%s\"]:::functional_unit", id, label)
	case "actor":
		return fmt.Sprintf("%s((\"%s\")):::actor", id, label)
	case "attack_vector":
		return fmt.Sprintf("%s((\"%s\")):::attack_vector", id, label)
	case "referenced_element":
		return fmt.Sprintf("%s[\"%s\"]:::referenced_element", id, label)
	case "interface":
		return fmt.Sprintf("%s[/\"%s\"/]:::interface", id, label)
	case "data_object":
		return fmt.Sprintf("%s[(\"%s\")]:::data_object", id, label)
	case "deployment_target":
		return fmt.Sprintf("%s[\"%s\"]:::deployment_target", id, label)
	case "control":
		return fmt.Sprintf("%s[[\"%s\"]]:::control", id, label)
	case "trust_boundary":
		return fmt.Sprintf("%s[/\"%s\"\\]:::trust_boundary", id, label)
	case "state":
		return fmt.Sprintf("%s([\"%s\"]):::state", id, label)
	case "event":
		return fmt.Sprintf("%s((\"%s\")):::event", id, label)
	case "flow":
		return fmt.Sprintf("%s{\"%s\"}:::flow", id, label)
	case "flow_step":
		return fmt.Sprintf("%s>\"%s\"]:::flow_step", id, label)
	default:
		return fmt.Sprintf("%s[\"%s\"]:::unknown", id, label)
	}
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
