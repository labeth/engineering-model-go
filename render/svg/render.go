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
	margin     = 36
	nodeWidth  = 340
	lineHeight = 18
)

// Render produces a lossless arc diagram and relationship key. Separate ports
// and lanes keep edges outside the nodes; prose belongs in the wrapped key.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func Render(projected view.ProjectedView, source string) string {
	nodes := append([]view.Node(nil), projected.Nodes...)
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	byID := make(map[string]view.Node, len(nodes))
	for _, n := range nodes {
		byID[n.ID] = n
	}
	edges := append([]view.Edge(nil), projected.Edges...)
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Sequence != edges[j].Sequence {
			return edges[i].Sequence < edges[j].Sequence
		}
		if edges[i].ID != edges[j].ID {
			return edges[i].ID < edges[j].ID
		}
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		if edges[i].Type != edges[j].Type {
			return edges[i].Type < edges[j].Type
		}
		if edges[i].Label != edges[j].Label {
			return edges[i].Label < edges[j].Label
		}
		return edges[i].ItemRef < edges[j].ItemRef
	})
	valid := edges[:0]
	degree := map[string]int{}
	for _, e := range edges {
		if _, ok := byID[e.From]; !ok {
			continue
		}
		if _, ok := byID[e.To]; !ok {
			continue
		}
		valid = append(valid, e)
		degree[e.From]++
		degree[e.To]++
	}
	edges = valid
	positions, heights := map[string]int{}, map[string]int{}
	titleLines := wrap(projected.Title, 70)
	top := 60 + len(titleLines)*lineHeight
	if top < 100 {
		top = 100
	}
	y := top
	for _, n := range nodes {
		h := 30 + lineHeight*(len(wrap(n.Label, 36))+len(wrap(n.ID, 43)))
		if ports := 28 + degree[n.ID]*lineHeight; ports > h {
			h = ports
		}
		positions[n.ID], heights[n.ID] = y, h
		y += h + 24
	}
	keyX := margin + nodeWidth + 90 + len(edges)*18
	width := keyX + 630 + margin
	keyLines := make([][]string, len(edges))
	keyHeight := top
	for i, e := range edges {
		label := strings.TrimSpace(e.Label)
		if e.ItemRef != "" {
			label += ": " + e.ItemRef
		}
		lines := wrap(e.From+" → "+e.To, 67)
		lines = append(lines, wrap(label, 67)...)
		keyLines[i] = append(wrap(fmt.Sprintf("%d. %s", i+1, e.ID), 67), lines...)
		keyHeight += 12 + lineHeight*len(keyLines[i])
	}
	height := y + margin
	if keyHeight+margin > height {
		height = keyHeight + margin
	}
	var b strings.Builder
	titleID := svgID(projected.ID) + "-title"
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-labelledby="%s" viewBox="0 0 %d %d" data-source="%s" data-view="%s">`+"\n",
		titleID, width, height, html.EscapeString(source), html.EscapeString(projected.ID))
	fmt.Fprintf(&b, "<title id=\"%s\">%s — %s view</title>\n", titleID, html.EscapeString(projected.Title), html.EscapeString(projected.Kind))
	fmt.Fprintf(&b, "<metadata>source=%s; view=%s; kind=%s</metadata>\n", html.EscapeString(source), html.EscapeString(projected.ID), html.EscapeString(projected.Kind))
	fmt.Fprintln(&b, `<defs><marker id="arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="#455a64"/></marker></defs>`)
	fmt.Fprintln(&b, `<rect width="100%" height="100%" fill="#ffffff"/>`)
	textLines(&b, margin, 30, 18, "bold", titleLines)
	textLines(&b, keyX, top-22, 14, "bold", []string{"Relationships · numbered arrows point to the destination"})
	used := map[string]int{}
	keyY := top
	for i, e := range edges {
		y1 := positions[e.From] + 18 + used[e.From]*lineHeight
		used[e.From]++
		y2 := positions[e.To] + 18 + used[e.To]*lineHeight
		used[e.To]++
		x, lane := margin+nodeWidth, margin+nodeWidth+60+i*18
		fmt.Fprintf(&b, "<path data-edge=\"%s\" d=\"M%d %d H%d V%d H%d\" fill=\"none\" stroke=\"#78909c\" marker-end=\"url(#arrow)\"/>\n", html.EscapeString(e.ID), x, y1, lane, y2, x)
		fmt.Fprintf(&b, "<rect x=\"%d\" y=\"%d\" width=\"34\" height=\"16\" rx=\"3\" fill=\"white\"/><text x=\"%d\" y=\"%d\" font-family=\"monospace\" font-size=\"11\">%d</text>\n", x+8, y1-12, x+11, y1, i+1)
		textLines(&b, keyX, keyY, 12, "normal", keyLines[i])
		keyY += 12 + lineHeight*len(keyLines[i])
	}
	for _, n := range nodes {
		fill, stroke := colors(n.Kind)
		fmt.Fprintf(&b, "<g id=\"%s\"><rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" rx=\"10\" fill=\"%s\" stroke=\"%s\" stroke-width=\"2\"/>\n", svgID(n.ID), margin, positions[n.ID], nodeWidth, heights[n.ID], fill, stroke)
		labels := wrap(n.Label, 36)
		textLines(&b, margin+14, positions[n.ID]+24, 13, "bold", labels)
		textLines(&b, margin+14, positions[n.ID]+24+len(labels)*lineHeight, 11, "normal", wrap(n.ID, 43))
		fmt.Fprintln(&b, "</g>")
	}
	fmt.Fprintln(&b, "</svg>")
	return b.String()
}

// wrap limits line lengths even for long unbroken identifiers.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func wrap(value string, limit int) []string {
	lines := []string{}
	line := ""
	for _, word := range strings.Fields(value) {
		runes := []rune(word)
		for len(runes) > limit {
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			lines = append(lines, string(runes[:limit]))
			runes = runes[limit:]
		}
		word = string(runes)
		if line == "" {
			line = word
		} else if len([]rune(line))+1+len(runes) <= limit {
			line += " " + word
		} else {
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func textLines(b *strings.Builder, x, y, size int, weight string, lines []string) {
	for i, line := range lines {
		fmt.Fprintf(b, "<text x=\"%d\" y=\"%d\" font-family=\"monospace\" font-size=\"%d\" font-weight=\"%s\">%s</text>\n", x, y+i*lineHeight, size, weight, html.EscapeString(line))
	}
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
	case "hardware", "hardware_item", "deployment_target":
		return "#ede7f6", "#5e35b1"
	case "software_component":
		return "#eceff1", "#455a64"
	case "port", "interface", "hardware_interface":
		return "#e8f5e9", "#2e7d32"
	default:
		return "#f5f5f5", "#616161"
	}
}
