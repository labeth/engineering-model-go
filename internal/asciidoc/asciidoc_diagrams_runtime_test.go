// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
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

	requireContains(t, out,
		docCheck{`subgraph CLUSTER_PAYMENTS_CLUSTER["Cluster: payments-cluster"]`, "missing cluster container"},
		docCheck{`subgraph NS_PAYMENTS["Namespace: payments"]`, "missing namespace container"},
		docCheck{`DP_PAYMENTS_API_DEPLOYMENT["Workload: payments/api-deployment"]:::runtime_element`, "missing runtime workload node inside deployment graph"},
		docCheck{`subgraph CONTROL_PLANE["Control Plane"]`, "missing control plane container"},
		docCheck{`DP_PAYMENTS_PAYMENTS_API["Release: payments/payments-api"]:::deployment_element`, "missing release node with role-prefixed label"},
		docCheck{`DP_PAYMENTS_PAYMENTS_API -->|targets| DP_PAYMENTS`, "missing release-to-namespace edge"},
		docCheck{`DP_PAYMENTS_PAYMENTS_API -->|deploys| DP_PAYMENTS_API_DEPLOYMENT`, "missing release-to-workload edge"},
	)
	requireNotContains(t, out, docCheck{"|part_of|", "did not expect explicit part_of edges when using cluster containers"})
}

// TRLC-LINKS: REQ-EMG-003
func TestBuildRuntimeAPIRows_UsesLambdaInferenceOwnership(t *testing.T) {
	runtime := []codescan.RuntimeItem{
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
	code := []codescan.CodeItem{
		{Kind: "symbol", Owner: "FU-A", Source: "src/payment_engine.rs:25"},
		{Kind: "symbol", Owner: "FU-A", Source: "src/payment_engine.rs:11"},
	}

	out := buildSecurityPathMermaid(rows, nil, code)

	requireCount(t, out, `["payment_engine.rs"]:::code_element`, 1, "expected one security code box at file granularity, got %d:\n%s")
	requireDiagramLacks(t, out, "did not expect line numbers in security attack-vector code boxes:\n%[2]s", `payment_engine.rs:11`, `payment_engine.rs:25`, `payment_engine.rs:11,25`)
}

// groupContextArchitecture is two groups with an actor, a referenced element,
// an interface, data objects and trust boundaries, some owned by the first group.
func groupContextArchitecture() model.AuthoredArchitecture {
	return model.AuthoredArchitecture{
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
}

// TRLC-LINKS: REQ-EMG-003
func TestBuildSecurityContextDFDMermaidByGroup_GroupsOwnedNodesInsideFG(t *testing.T) {
	out := buildSecurityContextDFDMermaidByGroup(groupContextArchitecture(), nil, nil)
	if len(out) != 2 {
		t.Fatalf("expected one context diagram per functional group, got %d", len(out))
	}
	graph := out[0].Mermaid
	localBlock := blockUntil(t, graph, `subgraph CTXGROUP_FG_A["Group A"]`, "\n  end", graph,
		"missing FG-A container:\n%s", "missing FG-A container close:\n%s")
	requireIn(t, localBlock, graph, "expected owned node inside FG-A container %q:\n%s",
		`CTX_FU_A["FU-A"]:::functional_unit`,
		`CTX_IF_A[/"IF-A"/]:::interface`,
		`CTX_DO_A[("DO-A")]:::data_object`,
		`CTX_TB_A[/"TB-A"\]:::trust_boundary`,
	)
	external := []string{
		`CTX_ACT_USER(("ACT-USER")):::actor`,
		`CTX_FU_B["FU-B"]:::functional_unit`,
		`CTX_REF_EXT["REF-EXT"]:::referenced_element`,
		`CTX_DO_EXT[("DO-EXT")]:::data_object`,
		`CTX_TB_EXT[/"TB-EXT"\]:::trust_boundary`,
	}
	requireDiagramHas(t, graph, "expected external node %q in diagram:\n%s", external...)
	requireNotIn(t, localBlock, graph, "external node should not be inside FG-A container %q:\n%s", external...)
	requireDiagramLacks(t, graph, "did not expect unrelated other-group data object in FG-A diagram:\n%[2]s", `CTX_DO_B[("DO-B")]:::data_object`)
	requireDiagramHas(t, graph, "missing context edge %q:\n%s",
		`CTX_ACT_USER -->|interacts_with| CTX_FU_A`,
		`CTX_FU_A -->|depends_on| CTX_FU_B`,
		`CTX_FU_A -->|reads| CTX_DO_EXT`,
	)
}
