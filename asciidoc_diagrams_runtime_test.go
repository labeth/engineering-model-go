package engmodel

import (
	"strings"
	"testing"
)

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
