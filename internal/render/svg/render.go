// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package svg

import (
	"cmp"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/view"
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
	d := newDiagram(projected, source)
	var b strings.Builder
	d.writeHeader(&b)
	d.writeEdges(&b)
	d.writeNodes(&b)
	fmt.Fprintln(&b, "</svg>")
	return b.String()
}

// diagram is the computed layout of one view: the sorted nodes, the edges that
// connect two known nodes, their key text and the canvas size.
type diagram struct {
	projected  view.ProjectedView
	source     string
	nodes      []view.Node
	edges      []view.Edge
	degree     map[string]int
	positions  map[string]int
	heights    map[string]int
	titleLines []string
	keyLines   [][]string
	top        int
	keyX       int
	width      int
	height     int
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func newDiagram(projected view.ProjectedView, source string) *diagram {
	d := &diagram{projected: projected, source: source, positions: map[string]int{}, heights: map[string]int{}}
	d.nodes = sortedNodes(projected.Nodes)
	d.edges, d.degree = routableEdges(sortedEdges(projected.Edges), d.nodes)
	d.titleLines = wrap(projected.Title, 70)
	d.top = max(100, 60+len(d.titleLines)*lineHeight)
	nodesBottom := d.layoutNodes()
	d.keyX = margin + nodeWidth + 90 + len(d.edges)*18
	d.width = d.keyX + 630 + margin
	keyBottom := d.layoutKey()
	d.height = max(nodesBottom+margin, keyBottom+margin)
	return d
}

// sortedNodes returns the nodes ordered by ID.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func sortedNodes(nodes []view.Node) []view.Node {
	sorted := append([]view.Node(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	return sorted
}

// sortedEdges returns the edges in a deterministic order independent of the
// input order.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func sortedEdges(edges []view.Edge) []view.Edge {
	sorted := append([]view.Edge(nil), edges...)
	sort.SliceStable(sorted, func(i, j int) bool { return compareEdges(sorted[i], sorted[j]) < 0 })
	return sorted
}

// compareEdges orders edges by sequence, ID, endpoints, type, label and item.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func compareEdges(a, b view.Edge) int {
	return cmp.Or(
		cmp.Compare(a.Sequence, b.Sequence),
		cmp.Compare(a.ID, b.ID),
		cmp.Compare(a.From, b.From),
		cmp.Compare(a.To, b.To),
		cmp.Compare(a.Type, b.Type),
		cmp.Compare(a.Label, b.Label),
		cmp.Compare(a.ItemRef, b.ItemRef),
	)
}

// routableEdges keeps the edges whose two ends are nodes and counts the edges
// at each node.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func routableEdges(edges []view.Edge, nodes []view.Node) ([]view.Edge, map[string]int) {
	known := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		known[n.ID] = true
	}
	valid := edges[:0]
	degree := map[string]int{}
	for _, e := range edges {
		if !known[e.From] || !known[e.To] {
			continue
		}
		valid = append(valid, e)
		degree[e.From]++
		degree[e.To]++
	}
	return valid, degree
}

// layoutNodes stacks the nodes top to bottom and returns the bottom edge.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (d *diagram) layoutNodes() int {
	y := d.top
	for _, n := range d.nodes {
		h := 30 + lineHeight*(len(wrap(n.Label, 36))+len(wrap(n.ID, 43)))
		h = max(h, 28+d.degree[n.ID]*lineHeight)
		d.positions[n.ID], d.heights[n.ID] = y, h
		y += h + 24
	}
	return y
}

// layoutKey wraps the key entry of every edge and returns the key's bottom edge.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (d *diagram) layoutKey() int {
	d.keyLines = make([][]string, len(d.edges))
	keyHeight := d.top
	for i, e := range d.edges {
		label := strings.TrimSpace(e.Label)
		if e.ItemRef != "" {
			label += ": " + e.ItemRef
		}
		lines := wrap(e.From+" → "+e.To, 67)
		lines = append(lines, wrap(label, 67)...)
		d.keyLines[i] = append(wrap(fmt.Sprintf("%d. %s", i+1, e.ID), 67), lines...)
		keyHeight += 12 + lineHeight*len(d.keyLines[i])
	}
	return keyHeight
}

// writeHeader writes the svg element, title, metadata, defs and background.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (d *diagram) writeHeader(b *strings.Builder) {
	p := d.projected
	titleID := svgID(p.ID) + "-title"
	fmt.Fprintf(b, `<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-labelledby="%s" viewBox="0 0 %d %d" data-source="%s" data-view="%s">`+"\n",
		titleID, d.width, d.height, html.EscapeString(d.source), html.EscapeString(p.ID))
	fmt.Fprintf(b, "<title id=\"%s\">%s — %s view</title>\n", titleID, html.EscapeString(p.Title), html.EscapeString(p.Kind))
	fmt.Fprintf(b, "<metadata>source=%s; view=%s; kind=%s</metadata>\n", html.EscapeString(d.source), html.EscapeString(p.ID), html.EscapeString(p.Kind))
	fmt.Fprintln(b, `<defs><marker id="arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="#455a64"/></marker></defs>`)
	fmt.Fprintln(b, `<rect width="100%" height="100%" fill="#ffffff"/>`)
	textBlock{x: margin, y: 30, size: 18, weight: "bold", lines: d.titleLines}.write(b)
	textBlock{x: d.keyX, y: d.top - 22, size: 14, weight: "bold", lines: []string{"Relationships · numbered arrows point to the destination"}}.write(b)
}

// writeEdges writes each edge's route, its number and its key entry.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (d *diagram) writeEdges(b *strings.Builder) {
	used := map[string]int{}
	keyY := d.top
	for i, e := range d.edges {
		y1 := d.positions[e.From] + 18 + used[e.From]*lineHeight
		used[e.From]++
		y2 := d.positions[e.To] + 18 + used[e.To]*lineHeight
		used[e.To]++
		x, lane := margin+nodeWidth, margin+nodeWidth+60+i*18
		fmt.Fprintf(b, "<path data-edge=\"%s\" d=\"M%d %d H%d V%d H%d\" fill=\"none\" stroke=\"#78909c\" marker-end=\"url(#arrow)\"/>\n", html.EscapeString(e.ID), x, y1, lane, y2, x)
		fmt.Fprintf(b, "<rect x=\"%d\" y=\"%d\" width=\"34\" height=\"16\" rx=\"3\" fill=\"white\"/><text x=\"%d\" y=\"%d\" font-family=\"monospace\" font-size=\"11\">%d</text>\n", x+8, y1-12, x+11, y1, i+1)
		textBlock{x: d.keyX, y: keyY, size: 12, weight: "normal", lines: d.keyLines[i]}.write(b)
		keyY += 12 + lineHeight*len(d.keyLines[i])
	}
}

// writeNodes writes each node box with its label and ID.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (d *diagram) writeNodes(b *strings.Builder) {
	for _, n := range d.nodes {
		fill, stroke := colors(n.Kind)
		fmt.Fprintf(b, "<g id=\"%s\"><rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" rx=\"10\" fill=\"%s\" stroke=\"%s\" stroke-width=\"2\"/>\n", svgID(n.ID), margin, d.positions[n.ID], nodeWidth, d.heights[n.ID], fill, stroke)
		labels := wrap(n.Label, 36)
		textBlock{x: margin + 14, y: d.positions[n.ID] + 24, size: 13, weight: "bold", lines: labels}.write(b)
		textBlock{x: margin + 14, y: d.positions[n.ID] + 24 + len(labels)*lineHeight, size: 11, weight: "normal", lines: wrap(n.ID, 43)}.write(b)
		fmt.Fprintln(b, "</g>")
	}
}

// wrapper accumulates the lines of wrap.
type wrapper struct {
	limit int
	lines []string
	line  string
}

// wrap limits line lengths even for long unbroken identifiers.
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func wrap(value string, limit int) []string {
	w := &wrapper{limit: limit, lines: []string{}}
	for _, word := range strings.Fields(value) {
		w.add(word)
	}
	w.flush()
	if len(w.lines) == 0 {
		return []string{""}
	}
	return w.lines
}

// add places one word, first splitting a word longer than the limit.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func (w *wrapper) add(word string) {
	runes := []rune(word)
	for len(runes) > w.limit {
		w.flush()
		w.lines = append(w.lines, string(runes[:w.limit]))
		runes = runes[w.limit:]
	}
	w.place(string(runes), len(runes))
}

// flush ends the line being built.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func (w *wrapper) flush() {
	if w.line != "" {
		w.lines = append(w.lines, w.line)
		w.line = ""
	}
}

// place appends word of n runes to the line being built or starts a new line.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func (w *wrapper) place(word string, n int) {
	switch {
	case w.line == "":
		w.line = word
	case len([]rune(w.line))+1+n <= w.limit:
		w.line += " " + word
	default:
		w.lines = append(w.lines, w.line)
		w.line = word
	}
}

// textBlock is a run of text lines at one position and style.
type textBlock struct {
	x, y, size int
	weight     string
	lines      []string
}

// write emits one text element per line.
//
// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func (t textBlock) write(b *strings.Builder) {
	for i, line := range t.lines {
		fmt.Fprintf(b, "<text x=\"%d\" y=\"%d\" font-family=\"monospace\" font-size=\"%d\" font-weight=\"%s\">%s</text>\n", t.x, t.y+i*lineHeight, t.size, t.weight, html.EscapeString(line))
	}
}

// svgIDChars are the characters that may appear in an SVG id unchanged.
const svgIDChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func svgID(value string) string {
	id := strings.Map(func(r rune) rune {
		if strings.ContainsRune(svgIDChars, r) {
			return r
		}
		return '_'
	}, value)
	if id == "" {
		return "view"
	}
	return id
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
