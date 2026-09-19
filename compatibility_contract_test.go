// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engmodel_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/mcp"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TestCurrentProjectionCompatibilityContract protects the observable shape of
// every legacy output class without duplicating their detailed exporter tests.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, CTRL-TRACEABILITY-COVERAGE
func TestCurrentProjectionCompatibilityContract(t *testing.T) {
	payments := filepath.Join("examples", "payments-engineering-sample")
	coffee := filepath.Join("examples", "coffee-fleet-ota-cloud-sample")
	architecture := filepath.Join(payments, "architecture.yml")
	requirements := filepath.Join(payments, "requirements.yml")
	design := filepath.Join(payments, "design.yml")

	contracts := []struct {
		name     string
		generate func() (string, error)
		contains []string
		json     bool
	}{
		{
			name: "asciidoc",
			generate: func() (string, error) {
				result, err := engmodel.GenerateAsciiDocFromFiles(architecture, requirements, design, engmodel.AsciiDocOptions{})
				return result.Document, err
			},
			contains: []string{"= Sample Payments Layered Design", "[source,mermaid]", "=== Verification Inventory"},
		},
		{
			name: "mermaid",
			generate: func() (string, error) {
				result, err := engmodel.GenerateFromFile(architecture, "VIEW-ARCHITECTURE-INTENT")
				return result.Mermaid, err
			},
			contains: []string{"flowchart LR", "view: VIEW-ARCHITECTURE-INTENT", "N_FU_PAYMENT_AUTHORIZATION"},
		},
		{
			name: "structurizr",
			generate: func() (string, error) {
				result, err := engmodel.GenerateStructurizrDSLFromFile(architecture)
				return result.DSL, err
			},
			contains: []string{"workspace ", "model {", "views {", "deploymentEnvironment"},
		},
		{
			name: "naf",
			generate: func() (string, error) {
				result, err := engmodel.GenerateNAFV41FromFile("architecture.yml")
				return result.Document, err
			},
			contains: []string{":naf-version: 4.1", "=== A2 - Architecture Products", "`VIEW-ARCHITECTURE-INTENT`"},
		},
		{
			name: "threat-dragon",
			generate: func() (string, error) {
				result, err := engmodel.GenerateThreatModelExportFromFile(architecture, engmodel.ThreatModelExportOptions{Format: engmodel.ThreatModelFormatThreatDragonV2})
				return result.JSON, err
			},
			contains: []string{`"version"`, `"2.0"`, `"diagramType"`, `"STRIDE"`},
			json:     true,
		},
		{
			name: "open-otm",
			generate: func() (string, error) {
				result, err := engmodel.GenerateThreatModelExportFromFile(architecture, engmodel.ThreatModelExportOptions{Format: engmodel.ThreatModelFormatOpenOTM})
				return result.JSON, err
			},
			contains: []string{`"otmVersion"`, `"threats"`},
			json:     true,
		},
		{
			name: "trlc",
			generate: func() (string, error) {
				result, err := engmodel.GenerateTRLCRequirementsFromFile(requirements, engmodel.TRLCExportOptions{})
				return result.ModelRSL + "\n" + result.RequirementsTRLC, err
			},
			contains: []string{"type Requirement", "applies_to String [0 .. *]", "REQ-PAY-001"},
		},
		{
			name: "lobster",
			generate: func() (string, error) {
				result, err := engmodel.GenerateLobsterActivityTraceFromDir(filepath.Join(payments, "tests"), engmodel.LobsterActivityExportOptions{
					RequirementsPackage: "PaymentsRequirements",
					ActivityNamespace:   "payments.tests",
				})
				return result.JSON, err
			},
			contains: []string{`"schema"`, `"lobster-act-trace"`, "PaymentsRequirements.REQ_PAY_"},
			json:     true,
		},
		{
			name: "oscal",
			generate: func() (string, error) {
				result, err := engmodel.GenerateOSCALSSPFromFile(architecture, engmodel.OSCALSSPOptions{})
				return result.JSON, err
			},
			contains: []string{`"system-security-plan"`, `"control-implementation"`, `"implemented-requirements"`},
			json:     true,
		},
		{
			name: "gemara",
			generate: func() (string, error) {
				result, err := engmodel.GenerateGemaraFromFile(architecture, engmodel.GemaraExportOptions{
					Version: "1.0.0",
					Date:    "2026-06-26T00:00:00Z",
				})
				if err != nil {
					return "", err
				}
				names := make([]string, 0, len(result.YAML))
				for name := range result.YAML {
					names = append(names, name)
				}
				sort.Strings(names)
				return strings.Join(names, "\n") + "\n" + result.YAML["control-catalog"], nil
			},
			contains: []string{"capability-catalog", "control-catalog", "threat-catalog", "ControlCatalog"},
		},
		{
			name: "trace-matrix",
			generate: func() (string, error) {
				matrix, diagnostics, err := engmodel.BuildTraceMatrixFromFiles(architecture, requirements, ".")
				if err != nil {
					return "", err
				}
				if validate.HasErrors(diagnostics) {
					return "", fmt.Errorf("trace matrix diagnostics: %+v", diagnostics)
				}
				data, err := json.Marshal(matrix)
				return string(data), err
			},
			contains: []string{`"model"`, `"sample-payments-layered-model"`, `"requirements"`, `"implemented"`, `"verified"`},
			json:     true,
		},
		{
			name: "composition",
			generate: func() (string, error) {
				result, err := engmodel.GenerateCompositionFromFile(filepath.Join(coffee, "architecture.yml"))
				if err != nil {
					return "", err
				}
				if result.Root == nil {
					return "", fmt.Errorf("composition root is nil")
				}
				return fmt.Sprintf("root=%s children=%d allocations=%d", result.Root.Bundle.Architecture.Model.ID, len(result.Root.Children), len(result.Allocations)), nil
			},
			contains: []string{"root=sample-coffee-fleet-ota-cloud-model", "children=3", "allocations=3"},
		},
		{
			name: "mcp-model",
			generate: func() (string, error) {
				return mcpToolPayload(architecture, requirements, design, "model.list", map[string]any{})
			},
			contains: []string{`"ok":true`, `"schemaVersion"`, `"tool":"model.list"`, `"entities"`},
			json:     true,
		},
		{
			name: "mcp-ownership",
			generate: func() (string, error) {
				return mcpToolPayload(architecture, requirements, design, "ownership.resolve", map[string]any{"entityId": "FU-PAYMENT-AUTHORIZATION"})
			},
			contains: []string{`"ok":true`, `"tool":"ownership.resolve"`, `"id":"FU-PAYMENT-AUTHORIZATION"`, `"owner"`},
			json:     true,
		},
		{
			name: "mcp-impact",
			generate: func() (string, error) {
				return mcpToolPayload(architecture, requirements, design, "requirements.impact", map[string]any{"requirementId": "REQ-PAY-001"})
			},
			contains: []string{`"ok":true`, `"tool":"requirements.impact"`, `"requirement"`, `"impactedFlows"`, `"relevantFiles"`},
			json:     true,
		},
		{
			name: "mcp-coverage",
			generate: func() (string, error) {
				return mcpToolPayload(architecture, requirements, design, "coverage.strictStatus", map[string]any{})
			},
			contains: []string{`"ok":true`, `"tool":"coverage.strictStatus"`, `"status"`, `"summary"`},
			json:     true,
		},
		{
			name: "strict-validation",
			generate: func() (string, error) {
				bundle, err := model.LoadBundle(architecture)
				if err != nil {
					return "", err
				}
				diagnostics := validate.Bundle(bundle)
				if validate.HasErrors(diagnostics) {
					return "", fmt.Errorf("strict validation diagnostics: %+v", diagnostics)
				}
				return "strict-valid canonical-and-legacy", nil
			},
			contains: []string{"strict-valid", "canonical-and-legacy"},
		},
	}

	for _, contract := range contracts {
		t.Run(contract.name, func(t *testing.T) {
			output, err := contract.generate()
			if err != nil {
				t.Fatalf("generate compatibility fixture: %v", err)
			}
			if strings.TrimSpace(output) == "" {
				t.Fatal("generated compatibility fixture is empty")
			}
			if contract.json {
				var document any
				if err := json.Unmarshal([]byte(output), &document); err != nil {
					t.Fatalf("compatibility fixture is not valid JSON: %v", err)
				}
			}
			for _, anchor := range contract.contains {
				if !strings.Contains(output, anchor) {
					t.Fatalf("compatibility fixture missing %q", anchor)
				}
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
func mcpToolPayload(architecture, requirements, design, tool string, arguments map[string]any) (string, error) {
	server := mcp.NewServer()
	initialize := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath":        architecture,
			"requirementsPath": requirements,
			"designPath":       design,
			"repoRoot":         ".",
		}},
	}
	if _, err := callMCP(server, initialize); err != nil {
		return "", err
	}
	response, err := callMCP(server, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      tool,
			"arguments": arguments,
		},
	})
	if err != nil {
		return "", err
	}
	result, _ := response["result"].(map[string]any)
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		return "", fmt.Errorf("%s returned no content", tool)
	}
	chunk, _ := content[0].(map[string]any)
	text, _ := chunk["text"].(string)
	if text == "" {
		return "", fmt.Errorf("%s returned empty content", tool)
	}
	return text, nil
}

// TRLC-LINKS: REQ-EMG-035
func callMCP(server *mcp.Server, request map[string]any) (map[string]any, error) {
	input, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	output, err := server.Handle(input)
	if err != nil {
		return nil, err
	}
	var response map[string]any
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}
	if response["error"] != nil {
		return nil, fmt.Errorf("MCP error: %+v", response["error"])
	}
	return response, nil
}
