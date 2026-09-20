// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package svg

import (
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/view"
)

const (
	nodeWidth  = 220
	nodeHeight = 64
	columnGap  = 90
	rowGap     = 34
	margin     = 36
)

// Render produces standalone, deterministic SVG without a browser or external
// diagram runtime.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func Render(projected view.ProjectedView, source string) string {
	nodes := append([]view.Node(nil), projected.Nodes...)
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	positions := make(map[string][2]int, len(nodes))
	rows := (len(nodes) + 1) / 2
	if rows < 1 {
		rows = 1
	}
	width := margin*2 + nodeWidth*2 + columnGap
	height := margin*2 + 54 + rows*(nodeHeight+rowGap)
	for i, node := range nodes {
		positions[node.ID] = [2]int{margin + (i%2)*(nodeWidth+columnGap), margin + 54 + (i/2)*(nodeHeight+rowGap)}
	}

	var b strings.Builder
	titleID := svgID(projected.ID) + "-title"
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-labelledby="%s" viewBox="0 0 %d %d" data-source="%s" data-view="%s">`+"\n",
		titleID, width, height, html.EscapeString(source), html.EscapeString(projected.ID))
	fmt.Fprintf(&b, "  <title id=\"%s\">%s — %s view</title>\n", titleID, html.EscapeString(projected.Title), html.EscapeString(projected.Kind))
	fmt.Fprintf(&b, "  <metadata>source=%s; view=%s; kind=%s</metadata>\n", html.EscapeString(source), html.EscapeString(projected.ID), html.EscapeString(projected.Kind))
	fmt.Fprintln(&b, `  <defs><marker id="arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="#455a64"/></marker></defs>`)
	fmt.Fprintln(&b, `  <rect width="100%" height="100%" fill="#ffffff"/>`)
	fmt.Fprintf(&b, "  <text x=\"%d\" y=\"30\" font-family=\"sans-serif\" font-size=\"18\" font-weight=\"bold\">%s</text>\n", margin, html.EscapeString(projected.Title))
	edges := append([]view.Edge(nil), projected.Edges...)
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Sequence != edges[j].Sequence {
			return edges[i].Sequence < edges[j].Sequence
		}
		return edges[i].ID < edges[j].ID
	})
	for _, edge := range edges {
		from, fromOK := positions[edge.From]
		to, toOK := positions[edge.To]
		if !fromOK || !toOK {
			continue
		}
		x1, y1 := from[0]+nodeWidth/2, from[1]+nodeHeight/2
		x2, y2 := to[0]+nodeWidth/2, to[1]+nodeHeight/2
		if edge.From == edge.To {
			fmt.Fprintf(&b, "  <path d=\"M%d %d C%d %d %d %d %d %d\" fill=\"none\" stroke=\"#455a64\" marker-end=\"url(#arrow)\"/>\n",
				x1+nodeWidth/2-8, y1, x1+nodeWidth, y1-36, x1+nodeWidth, y1+36, x1+nodeWidth/2-8, y1+4)
		} else {
			fmt.Fprintf(&b, "  <line x1=\"%d\" y1=\"%d\" x2=\"%d\" y2=\"%d\" stroke=\"#455a64\" marker-end=\"url(#arrow)\"/>\n", x1, y1, x2, y2)
		}
		label := strings.TrimSpace(edge.Label)
		if edge.ItemRef != "" {
			label += ": " + edge.ItemRef
		}
		fmt.Fprintf(&b, "  <text x=\"%d\" y=\"%d\" text-anchor=\"middle\" font-family=\"sans-serif\" font-size=\"11\">%s</text>\n",
			(x1+x2)/2, (y1+y2)/2-5, html.EscapeString(label))
	}
	for _, node := range nodes {
		p := positions[node.ID]
		fill, stroke := colors(node.Kind)
		fmt.Fprintf(&b, "  <g id=\"%s\"><rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" rx=\"10\" fill=\"%s\" stroke=\"%s\" stroke-width=\"2\"/>\n",
			svgID(node.ID), p[0], p[1], nodeWidth, nodeHeight, fill, stroke)
		fmt.Fprintf(&b, "    <text x=\"%d\" y=\"%d\" text-anchor=\"middle\" font-family=\"sans-serif\" font-size=\"13\" font-weight=\"bold\">%s</text>\n",
			p[0]+nodeWidth/2, p[1]+28, html.EscapeString(node.Label))
		fmt.Fprintf(&b, "    <text x=\"%d\" y=\"%d\" text-anchor=\"middle\" font-family=\"monospace\" font-size=\"10\">%s</text></g>\n",
			p[0]+nodeWidth/2, p[1]+47, html.EscapeString(node.ID))
	}
	fmt.Fprintln(&b, "</svg>")
	return b.String()
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func svgID(value string) string {
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "view"
	}
	return b.String()
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func colors(kind string) (string, string) {
	switch kind {
	case "actor", "use_case":
		return "#fff8e1", "#ef6c00"
	case "capability", "logical_component":
		return "#e3f2fd", "#1565c0"
	case "hardware", "deployment_target":
		return "#ede7f6", "#5e35b1"
	case "software_component":
		return "#eceff1", "#455a64"
	case "port", "interface":
		return "#e8f5e9", "#2e7d32"
	default:
		return "#f5f5f5", "#616161"
	}
}
