// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package svg

import (
	"encoding/xml"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/view"
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
	// Text uses fixed-pitch glyphs. Rows must not collide at a shared x/y anchor.
	d := xml.NewDecoder(strings.NewReader(got))
	positions := map[string]bool{}
	for {
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "text" {
			attrs := map[string]string{}
			for _, a := range start.Attr {
				attrs[a.Name.Local] = a.Value
			}
			key := attrs["x"] + ":" + attrs["y"]
			if positions[key] {
				t.Fatalf("overlapping text anchors: %s", key)
			}
			positions[key] = true
		}
	}
	projected.Nodes = []view.Node{nodes[1], nodes[0]}
	projected.Edges = []view.Edge{edges[2], edges[1], edges[0]}
	if again := Render(projected, "model&source"); again != got {
		t.Fatal("layout depends on input order")
	}
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
