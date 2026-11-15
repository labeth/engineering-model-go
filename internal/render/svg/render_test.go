// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package svg

import (
	"encoding/xml"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/view"
)

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func TestDenseViewRetainsEdgesWithoutTextOverlap(t *testing.T) {
	nodes := []view.Node{{ID: "HW-A", Label: "ADC <bank>", Kind: "hardware_item"}, {ID: "HW-B", Label: "Capture & storage", Kind: "hardware_item"}}
	edges := []view.Edge{
		{ID: "forward", From: "HW-A", To: "HW-B", Label: strings.Repeat("long relationship description ", 12)},
		{ID: "return", From: "HW-B", To: "HW-A", Label: "return"},
		{ID: "self", From: "HW-B", To: "HW-B", Label: "feedback"},
	}
	projected := view.ProjectedView{ID: "VIEW-TEST", Title: "Dense view", Nodes: nodes, Edges: edges}
	got := Render(projected, "model&source")
	if strings.Count(got, "data-edge=") != len(edges) || !strings.Contains(got, "ADC &lt;bank&gt;") || !strings.Contains(got, "Capture &amp; storage") {
		t.Fatal("SVG lost a relationship or failed to escape labels")
	}
	assertDistinctTextAnchors(t, got)
	projected.Nodes = []view.Node{nodes[1], nodes[0]}
	projected.Edges = []view.Edge{edges[2], edges[1], edges[0]}
	if again := Render(projected, "model&source"); again != got {
		t.Fatal("layout depends on input order")
	}
}

// assertDistinctTextAnchors fails when two text elements share an x/y anchor.
// Text uses fixed-pitch glyphs, so rows must not collide at a shared anchor.
func assertDistinctTextAnchors(t *testing.T, svg string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(svg))
	positions := map[string]bool{}
	for {
		token, err := d.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "text" {
			continue
		}
		key := textAnchor(start)
		if positions[key] {
			t.Fatalf("overlapping text anchors: %s", key)
		}
		positions[key] = true
	}
}

// textAnchor is the x:y anchor of a text element.
func textAnchor(start xml.StartElement) string {
	attrs := map[string]string{}
	for _, a := range start.Attr {
		attrs[a.Name.Local] = a.Value
	}
	return attrs["x"] + ":" + attrs["y"]
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func TestWrapPreservesUnicodeAndLongIdentifiers(t *testing.T) {
	got := wrap("αβγδεζηθ hello world", 5)
	want := []string{"αβγδε", "ζηθ", "hello", "world"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func TestParallelUnidentifiedEdgesHaveDeterministicOrder(t *testing.T) {
	p := view.ProjectedView{Nodes: []view.Node{{ID: "A"}, {ID: "B"}}, Edges: []view.Edge{
		{From: "A", To: "B", Label: "second"}, {From: "A", To: "B", Label: "first"},
	}}
	before := Render(p, "source")
	p.Edges[0], p.Edges[1] = p.Edges[1], p.Edges[0]
	if Render(p, "source") != before {
		t.Fatal("parallel edges without authored IDs depend on input order")
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func TestEmptyAndDanglingViewRemainValid(t *testing.T) {
	got := Render(view.ProjectedView{ID: "empty", Edges: []view.Edge{{From: "missing", To: "absent"}}}, "source")
	if strings.Contains(got, "data-edge=") {
		t.Fatal("rendered dangling edge")
	}
	d := xml.NewDecoder(strings.NewReader(got))
	for {
		_, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}
