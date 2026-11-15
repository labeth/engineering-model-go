// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package view

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// mustBuild projects the view and fails the test on a diagnostic.
func mustBuild(t *testing.T, b model.Bundle, viewID string) ProjectedView {
	t.Helper()
	v, diags := Build(b, viewID)
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	return v
}

// hasNodeID reports whether the view has a node with the ID.
func hasNodeID(v ProjectedView, id string) bool {
	for _, n := range v.Nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}

// countNodeKind is the number of nodes of the kind.
func countNodeKind(v ProjectedView, kind string) int {
	count := 0
	for _, n := range v.Nodes {
		if n.Kind == kind {
			count++
		}
	}
	return count
}

// hasEdgeType reports whether the view has an edge of the type.
func hasEdgeType(v ProjectedView, typ string) bool {
	for _, e := range v.Edges {
		if e.Type == typ {
			return true
		}
	}
	return false
}

// hasEdge reports whether the view has an edge with the type, source and target of the want.
func hasEdge(v ProjectedView, want Edge) bool {
	for _, e := range v.Edges {
		if e.Type == want.Type && e.From == want.From && e.To == want.To {
			return true
		}
	}
	return false
}

// requireOnlyEdgeType fails the test for an edge of another type.
func requireOnlyEdgeType(t *testing.T, v ProjectedView, typ string) {
	t.Helper()
	for _, e := range v.Edges {
		if e.Type != typ {
			t.Fatalf("expected only %s edges with includeMappings filter, got %+v", typ, v.Edges)
		}
	}
}

// edgeKey identifies an edge by its ends and type.
type edgeKey struct{ from, to, typ string }

// requireEdgeLabels fails the test unless the view has exactly the edges, with their labels.
func requireEdgeLabels(t *testing.T, v ProjectedView, want map[edgeKey]string) {
	t.Helper()
	got := map[edgeKey]string{}
	for _, e := range v.Edges {
		got[edgeKey{e.From, e.To, e.Type}] = e.Label
	}
	if len(v.Edges) != len(want) {
		t.Fatalf("want %d lifecycle edges, got %d: %+v", len(want), len(v.Edges), v.Edges)
	}
	for e, label := range want {
		if got[e] != label {
			t.Fatalf("edge %+v label = %q, want %q (all: %+v)", e, got[e], label, got)
		}
	}
}

// requireNodeKinds fails the test unless each node has its kind.
func requireNodeKinds(t *testing.T, v ProjectedView, want map[string]string) {
	t.Helper()
	kinds := map[string]string{}
	for _, n := range v.Nodes {
		kinds[n.ID] = n.Kind
	}
	for id, kind := range want {
		if kinds[id] != kind {
			t.Fatalf("node %s kind = %q, want %q (all: %+v)", id, kinds[id], kind, kinds)
		}
	}
}

// requireMemberIdentity fails the test unless each node has the kind of its ID
// and a label other than its ID.
func requireMemberIdentity(t *testing.T, v ProjectedView, want map[string]string) {
	t.Helper()
	for _, n := range v.Nodes {
		if n.Kind != want[n.ID] || n.Label == n.ID {
			t.Fatalf("lost member identity: %+v", n)
		}
	}
}
