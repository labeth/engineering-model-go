// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-003
func TestBuildDeploymentMermaid_UsesNamespaceAndClusterContainers(t *testing.T) {
	rows := []asciidocDeploymentRow{
		{From: "payments-repo", To: "payments-kustomization", How: "reconciles"},
		{From: "payments-kustomization", To: "payments/payments-api", How: "applies"},
		{From: "payments/payments-api", To: "payments/api-deployment", How: "deploys"},
		{From: "payments/payments-api", To: "payments", How: "targets"},
		{From: "payments", To: "payments-cluster", How: "part_of"},
	}

	out := buildDeploymentMermaid(rows)

	if !strings.Contains(out, `subgraph CLUSTER_PAYMENTS_CLUSTER["Cluster: payments-cluster"]`) {
		t.Fatalf("missing cluster container")
	}
	if !strings.Contains(out, `subgraph NS_PAYMENTS["Namespace: payments"]`) {
		t.Fatalf("missing namespace container")
	}
	if !strings.Contains(out, `DP_PAYMENTS_API_DEPLOYMENT["Workload: payments/api-deployment"]:::runtime_element`) {
		t.Fatalf("missing runtime workload node inside deployment graph")
	}
	if !strings.Contains(out, `subgraph CONTROL_PLANE["Control Plane"]`) {
		t.Fatalf("missing control plane container")
	}
	if !strings.Contains(out, `DP_PAYMENTS_PAYMENTS_API["Release: payments/payments-api"]:::deployment_element`) {
		t.Fatalf("missing release node with role-prefixed label")
	}
	if !strings.Contains(out, `DP_PAYMENTS_PAYMENTS_API -->|targets| DP_PAYMENTS`) {
		t.Fatalf("missing release-to-namespace edge")
	}
	if !strings.Contains(out, `DP_PAYMENTS_PAYMENTS_API -->|deploys| DP_PAYMENTS_API_DEPLOYMENT`) {
		t.Fatalf("missing release-to-workload edge")
	}
	if strings.Contains(out, "|part_of|") {
		t.Fatalf("did not expect explicit part_of edges when using cluster containers")
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestBuildRuntimeAPIRows_UsesLambdaInferenceOwnership(t *testing.T) {
	runtime := []inferredRuntimeItem{
		{Name: "webhook_ingress", Kind: "lambda_function", Owner: "FU-GITHUB-WEBHOOK-INGRESS"},
		{Name: "review_orchestrator", Kind: "lambda_function", Owner: "FU-REVIEW-ORCHESTRATION"},
	}
	mappings := []model.Mapping{
		{Type: "depends_on", From: "FU-GITHUB-WEBHOOK-INGRESS", To: "FU-REVIEW-ORCHESTRATION"},
	}

	rows := buildRuntimeAPIRows(runtime, mappings)
	if len(rows) != 1 {
		t.Fatalf("expected one runtime api row, got %d", len(rows))
	}
	if rows[0].Consumer != "webhook_ingress" {
		t.Fatalf("expected consumer webhook_ingress, got %q", rows[0].Consumer)
	}
	if rows[0].Provider != "review_orchestrator" {
		t.Fatalf("expected provider review_orchestrator, got %q", rows[0].Provider)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestBuildSecurityPathMermaid_GroupsCodeNodesByFile(t *testing.T) {
	rows := []asciidocSecurityPathRow{{
		AttackVectorID: "AV-A",
		AttackVector:   "Input spoofing",
		TargetID:       "FU-A",
		Target:         "Unit A",
	}}
	code := []inferredCodeItem{
		{Kind: "symbol", Owner: "FU-A", Source: "src/payment_engine.rs:25"},
		{Kind: "symbol", Owner: "FU-A", Source: "src/payment_engine.rs:11"},
	}

	out := buildSecurityPathMermaid(rows, nil, code)

	if got := strings.Count(out, `["payment_engine.rs"]:::code_element`); got != 1 {
		t.Fatalf("expected one security code box at file granularity, got %d:\n%s", got, out)
	}
	if strings.Contains(out, `payment_engine.rs:11`) || strings.Contains(out, `payment_engine.rs:25`) || strings.Contains(out, `payment_engine.rs:11,25`) {
		t.Fatalf("did not expect line numbers in security attack-vector code boxes:\n%s", out)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestBuildSecurityContextDFDMermaidByGroup_GroupsOwnedNodesInsideFG(t *testing.T) {
	a := model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{
			{ID: "FG-A", Name: "Group A"},
			{ID: "FG-B", Name: "Group B"},
		},
		FunctionalUnits: []model.FunctionalUnit{
			{ID: "FU-A", Name: "Unit A", Group: "FG-A"},
			{ID: "FU-B", Name: "Unit B", Group: "FG-B"},
		},
		Actors:             []model.Actor{{ID: "ACT-USER", Name: "User"}},
		ReferencedElements: []model.ReferencedElement{{ID: "REF-EXT", Name: "External Service"}},
		Interfaces:         []model.Interface{{ID: "IF-A", Name: "API A", Owner: "FU-A"}},
		DataObjects: []model.DataObject{
			{ID: "DO-A", Name: "Owned Data"},
			{ID: "DO-EXT", Name: "External Data"},
			{ID: "DO-B", Name: "Other Group Data"},
		},
		TrustBoundaries: []model.TrustBoundary{
			{ID: "TB-A", Name: "A Boundary", Members: []string{"FU-A"}},
			{ID: "TB-EXT", Name: "External Boundary", Members: []string{"REF-EXT"}},
		},
		Mappings: []model.Mapping{
			{Type: "interacts_with", From: "ACT-USER", To: "FU-A"},
			{Type: "depends_on", From: "FU-A", To: "FU-B"},
			{Type: "depends_on", From: "FU-A", To: "REF-EXT"},
			{Type: "contains", From: "FU-A", To: "IF-A"},
			{Type: "writes", From: "FU-A", To: "DO-A"},
			{Type: "reads", From: "FU-A", To: "DO-EXT"},
			{Type: "bounded_by", From: "FU-A", To: "TB-A"},
			{Type: "bounded_by", From: "FU-A", To: "TB-EXT"},
			{Type: "writes", From: "FU-B", To: "DO-B"},
		},
	}

	out := buildSecurityContextDFDMermaidByGroup(a, nil)
	if len(out) != 2 {
		t.Fatalf("expected one context diagram per functional group, got %d", len(out))
	}
	graph := out[0].Mermaid
	localStart := strings.Index(graph, `subgraph CTXGROUP_FG_A["Group A"]`)
	if localStart < 0 {
		t.Fatalf("missing FG-A container:\n%s", graph)
	}
	localEnd := strings.Index(graph[localStart:], "\n  end")
	if localEnd < 0 {
		t.Fatalf("missing FG-A container close:\n%s", graph)
	}
	localBlock := graph[localStart : localStart+localEnd]
	for _, want := range []string{
		`CTX_FU_A["FU-A"]:::functional_unit`,
		`CTX_IF_A[/"IF-A"/]:::interface`,
		`CTX_DO_A[("DO-A")]:::data_object`,
		`CTX_TB_A[/"TB-A"\]:::trust_boundary`,
	} {
		if !strings.Contains(localBlock, want) {
			t.Fatalf("expected owned node inside FG-A container %q:\n%s", want, graph)
		}
	}
	for _, external := range []string{
		`CTX_ACT_USER(("ACT-USER")):::actor`,
		`CTX_FU_B["FU-B"]:::functional_unit`,
		`CTX_REF_EXT["REF-EXT"]:::referenced_element`,
		`CTX_DO_EXT[("DO-EXT")]:::data_object`,
		`CTX_TB_EXT[/"TB-EXT"\]:::trust_boundary`,
	} {
		if !strings.Contains(graph, external) {
			t.Fatalf("expected external node %q in diagram:\n%s", external, graph)
		}
		if strings.Contains(localBlock, external) {
			t.Fatalf("external node should not be inside FG-A container %q:\n%s", external, graph)
		}
	}
	if strings.Contains(graph, `CTX_DO_B[("DO-B")]:::data_object`) {
		t.Fatalf("did not expect unrelated other-group data object in FG-A diagram:\n%s", graph)
	}
	for _, want := range []string{
		`CTX_ACT_USER -->|interacts_with| CTX_FU_A`,
		`CTX_FU_A -->|depends_on| CTX_FU_B`,
		`CTX_FU_A -->|reads| CTX_DO_EXT`,
	} {
		if !strings.Contains(graph, want) {
			t.Fatalf("missing context edge %q:\n%s", want, graph)
		}
	}
}
