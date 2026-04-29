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

func Render(v view.ProjectedView) string {
	nodes := append([]view.Node(nil), v.Nodes...)
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		return nodes[i].ID < nodes[j].ID
	})
	nodeLines := make([]string, 0, len(nodes))
	for _, n := range nodes {
		nodeLines = append(nodeLines, renderNode(n))
	}

	edges := append([]view.Edge(nil), v.Edges...)
	sort.SliceStable(edges, func(i, j int) bool {
		a := edges[i]
		c := edges[j]
		if a.From != c.From {
			return a.From < c.From
		}
		if a.To != c.To {
			return a.To < c.To
		}
		if a.Type != c.Type {
			return a.Type < c.Type
		}
		return a.Label < c.Label
	})
	edgeLines := make([]edgeLine, 0, len(edges))
	for _, e := range edges {
		label := compactEdgeLabel(e.Type, e.Label)
		edgeLines = append(edgeLines, edgeLine{
			From:  mermaidID(e.From),
			Label: escapeLabel(label),
			To:    mermaidID(e.To),
		})
	}

	data := diagramTemplateData{
		ViewID:    escapeComment(v.ID),
		ViewKind:  escapeComment(v.Kind),
		Nodes:     nodeLines,
		Edges:     edgeLines,
		ClassDefs: diagramstyle.MermaidClassDefs(),
	}

	var b bytes.Buffer
	if err := diagramTemplate.Execute(&b, data); err != nil {
		// Preserve current API shape (string-only return); embed fallback content on template failure.
		return "flowchart LR\n%% render_error: " + escapeComment(err.Error()) + "\n"
	}
	return b.String()
}

func renderNode(n view.Node) string {
	id := mermaidID(n.ID)
	label := escapeLabel(n.Label)
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

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.NewReplacer(
		"|", "/",
		"[", " ",
		"]", " ",
		"(", " ",
		")", " ",
		"{", " ",
		"}", " ",
	).Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func escapeComment(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}

func compactEdgeLabel(edgeType, fallback string) string {
	// Keep labels selective: show interaction/activity semantics so flow diagrams
	// communicate behavior directly, while avoiding noisy labels everywhere.
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
