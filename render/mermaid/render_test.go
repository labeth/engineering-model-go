package mermaid

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/view"
)

func TestRender_IncludesFlowClassDefsAndNodeKinds(t *testing.T) {
	v := view.ProjectedView{
		ID:   "V-FLOW",
		Kind: "interaction-flow",
		Nodes: []view.Node{
			{ID: "FLOW-A", Label: "Flow A", Kind: "flow"},
			{ID: "FLOW-A::submit", Label: "Submit", Kind: "flow_step"},
		},
		Edges: []view.Edge{{From: "FLOW-A", To: "FLOW-A::submit", Type: "flow_next", Label: "flow_next"}},
	}

	out := Render(v)
	for _, want := range []string{"classDef flow ", "classDef flow_step ", ":::flow", ":::flow_step"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected rendered diagram to include %q, got:\n%s", want, out)
		}
	}
}
