package mermaid

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/template"

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
		label := strings.TrimSpace(e.Label)
		if label == "" {
			label = e.Type
		}
		edgeLines = append(edgeLines, edgeLine{
			From:  mermaidID(e.From),
			Label: escapeLabel(label),
			To:    mermaidID(e.To),
		})
	}

	data := diagramTemplateData{
		ViewID:   escapeComment(v.ID),
		ViewKind: escapeComment(v.Kind),
		Nodes:    nodeLines,
		Edges:    edgeLines,
		ClassDefs: []string{
			"classDef person fill:#d8f3dc,stroke:#1b4332,stroke-width:1px;",
			"classDef system fill:#e7f5ff,stroke:#1c7ed6,stroke-width:1px;",
			"classDef external fill:#fff3bf,stroke:#f08c00,stroke-width:1px;",
			"classDef container fill:#f1f3f5,stroke:#495057,stroke-width:1px;",
			"classDef component fill:#f8f0fc,stroke:#862e9c,stroke-width:1px;",
			"classDef environment fill:#e6fcf5,stroke:#099268,stroke-width:1px;",
			"classDef cluster fill:#e7f5ff,stroke:#1971c2,stroke-width:1px;",
			"classDef namespace fill:#fff9db,stroke:#f08c00,stroke-width:1px;",
			"classDef flux fill:#f3f0ff,stroke:#7048e8,stroke-width:1px;",
			"classDef helm fill:#fff0f6,stroke:#c2255c,stroke-width:1px;",
			"classDef source fill:#f8f9fa,stroke:#495057,stroke-width:1px;",
			"classDef unknown fill:#ffffff,stroke:#adb5bd,stroke-width:1px;",
		},
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
	case "person":
		return fmt.Sprintf("%s((\"%s\")):::person", id, label)
	case "external_system":
		return fmt.Sprintf("%s[[\"%s\"]]:::external", id, label)
	case "system":
		return fmt.Sprintf("%s[\"%s\"]:::system", id, label)
	case "container":
		return fmt.Sprintf("%s[\"%s\"]:::container", id, label)
	case "component":
		return fmt.Sprintf("%s[\"%s\"]:::component", id, label)
	case "environment":
		return fmt.Sprintf("%s[\"%s\"]:::environment", id, label)
	case "cluster":
		return fmt.Sprintf("%s[\"%s\"]:::cluster", id, label)
	case "namespace":
		return fmt.Sprintf("%s[\"%s\"]:::namespace", id, label)
	case "flux_kustomization":
		return fmt.Sprintf("%s[\"%s\"]:::flux", id, label)
	case "helm_release":
		return fmt.Sprintf("%s[\"%s\"]:::helm", id, label)
	case "helm_chart":
		return fmt.Sprintf("%s[\"%s\"]:::helm", id, label)
	case "git_source":
		return fmt.Sprintf("%s[\"%s\"]:::source", id, label)
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
