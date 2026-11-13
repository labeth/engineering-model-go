// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package mermaid

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/view"
)

// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-050
func TestRenderHardwareUsesKnownStyles(t *testing.T) {
	for _, kind := range []string{"architecture-intent", "deployment", "physical"} {
		v := view.ProjectedView{ID: "VIEW-HW", Kind: kind, Nodes: []view.Node{{ID: "HW-A", Label: "ADC", Kind: "hardware_item"}, {ID: "HWIF-A", Label: "Samples", Kind: "hardware_interface"}}}
		out := Render(v)
		if strings.Contains(out, ":::unknown") || !strings.Contains(out, ":::deployment_element") || !strings.Contains(out, ":::interface") {
			t.Fatalf("hardware style lost in %s: %s", kind, out)
		}
	}
}

// The state lifecycle diagram of a machine keeps the format of the legacy
// state view: state, event and control nodes with their classes and one plain
// arrow per transition, event and guard edge.
// Mutation: a view that drops the machine edges renders no arrow.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-139
func TestRenderStateLifecycleOfAMachine(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalUnits: []model.FunctionalUnit{{ID: "FU-A", Name: "Unit"}},
			Machines: []model.Machine{{
				ID: "SM-A", Owner: "FU-A", Initial: "STATE-A",
				States:      []model.MachineState{{ID: "STATE-A", Name: "Ready"}, {ID: "STATE-B", Name: "Applied"}},
				Transitions: []model.MachineTransition{{ID: "TR-A", From: "STATE-A", To: "STATE-B", On: "EVT-A"}},
			}},
			Events:   []model.Event{{ID: "EVT-A", Name: "Requested"}},
			Controls: []model.Control{{ID: "CTRL-A", Name: "Gate"}},
			Mappings: []model.Mapping{{Type: "guarded_by", From: "STATE-B", To: "CTRL-A"}},
		},
		Views: []model.View{{ID: "V-STATE", Kind: "state-lifecycle", Roots: []string{"SM-A"}}},
	}}
	projected, diags := view.Build(b, "V-STATE")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	out := Render(projected)
	for _, want := range []string{
		`N_STATE_A(["Ready"]):::state`, `N_STATE_B(["Applied"]):::state`, `N_EVT_A(("Requested")):::event`, `N_CTRL_A[["Gate"]]:::control`,
		"N_STATE_A --> N_STATE_B", "N_STATE_A --> N_EVT_A", "N_STATE_B --> N_CTRL_A",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("state lifecycle diagram lacks %q:\n%s", want, out)
		}
	}
}
