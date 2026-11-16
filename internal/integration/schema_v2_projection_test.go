// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package integration

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/mcp"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// projectionContract is one output class of the schema-v2 fixture: how to
// generate it, what it must contain and whether it must be JSON.
type projectionContract struct {
	name     string
	generate func() (string, error)
	contains []string
	json     bool
}

// projectionFixture holds the example models and documents the contracts use.
type projectionFixture struct {
	payments, coffee                   string
	architecture, requirements, design string
}

func newProjectionFixture() projectionFixture {
	payments := filepath.Join("examples", "payments-engineering-sample")
	return projectionFixture{
		payments:     payments,
		coffee:       filepath.Join("examples", "coffee-fleet-ota-cloud-sample"),
		architecture: filepath.Join(payments, "engmod.yml"),
		requirements: filepath.Join(payments, "model", "requirements.yml"),
		design:       filepath.Join(payments, "model", "views.yml"),
	}
}

// mcp is the MCP server configuration of the payments example.
func (f projectionFixture) mcp() mcpModel {
	return mcpModel{architecture: f.architecture, requirements: f.requirements, design: f.design}
}

// TestSchemaV2ProjectionContract protects the observable shape of every output
// class when loaded from the schema-v2 manifest and domain documents.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, CTRL-TRACEABILITY-COVERAGE
func TestSchemaV2ProjectionContract(t *testing.T) {
	fixture := newProjectionFixture()
	var contracts []projectionContract
	contracts = append(contracts, fixture.documentContracts(t)...)
	contracts = append(contracts, fixture.threatAndRequirementContracts()...)
	contracts = append(contracts, fixture.assuranceContracts()...)
	contracts = append(contracts, fixture.modelContracts()...)
	contracts = append(contracts, fixture.mcpContracts()...)
	for _, contract := range contracts {
		t.Run(contract.name, func(t *testing.T) {
			assertProjectionContract(t, contract)
		})
	}
}

// assertProjectionContract checks that a contract generates non-empty output
// of the right form that holds every anchor.
func assertProjectionContract(t *testing.T, contract projectionContract) {
	t.Helper()
	output, err := contract.generate()
	if err != nil {
		t.Fatalf("generate schema-v2 fixture: %v", err)
	}
	if strings.TrimSpace(output) == "" {
		t.Fatal("generated schema-v2 fixture is empty")
	}
	if contract.json {
		assertValidJSON(t, output)
	}
	for _, anchor := range contract.contains {
		if !strings.Contains(output, anchor) {
			t.Fatalf("schema-v2 fixture missing %q", anchor)
		}
	}
}

// assertValidJSON fails when output is not a JSON document.
func assertValidJSON(t *testing.T, output string) {
	t.Helper()
	var document any
	if err := json.Unmarshal([]byte(output), &document); err != nil {
		t.Fatalf("schema-v2 fixture is not valid JSON: %v", err)
	}
}

// documentContracts are the document and diagram outputs.
func (f projectionFixture) documentContracts(t *testing.T) []projectionContract {
	return []projectionContract{
		{
			name: "asciidoc",
			generate: func() (string, error) {
				result, err := engmodel.GenerateAsciiDocFromFiles(engmodel.AsciiDocFiles{Architecture: f.architecture, Requirements: f.requirements, Design: f.design}, engmodel.AsciiDocOptions{})
				return result.Document, err
			},
			contains: []string{"= Sample Payments Layered Design", "[source,mermaid]", "=== Verification Inventory"},
		},
		{
			name: "mermaid",
			generate: func() (string, error) {
				result, err := engmodel.GenerateFromFile(f.architecture, "VIEW-ARCHITECTURE-INTENT")
				return result.Mermaid, err
			},
			contains: []string{"flowchart LR", "view: VIEW-ARCHITECTURE-INTENT", "N_FU_PAYMENT_AUTHORIZATION"},
		},
		{
			name: "structurizr",
			generate: func() (string, error) {
				result, err := engmodel.GenerateStructurizrDSLFromFile(f.architecture)
				return result.DSL, err
			},
			contains: []string{"workspace ", "model {", "views {", "deploymentEnvironment"},
		},
		{
			name: "naf",
			generate: func() (string, error) {
				result, err := engmodel.GenerateNAFV41FromFile(testutil.AcceptedRepoModel(t))
				return result.Document, err
			},
			contains: []string{":naf-version: 4.1", "=== A2 - Architecture Products", "`VIEW-ARCHITECTURE-INTENT`"},
		},
	}
}

// threatAndRequirementContracts are the threat model and requirement exports.
func (f projectionFixture) threatAndRequirementContracts() []projectionContract {
	return []projectionContract{
		{
			name: "threat-dragon",
			generate: func() (string, error) {
				result, err := engmodel.GenerateThreatModelExportFromFile(f.architecture, engmodel.ThreatModelExportOptions{Format: engmodel.ThreatModelFormatThreatDragonV2})
				return result.JSON, err
			},
			contains: []string{`"version"`, `"2.0"`, `"diagramType"`, `"STRIDE"`},
			json:     true,
		},
		{
			name: "open-otm",
			generate: func() (string, error) {
				result, err := engmodel.GenerateThreatModelExportFromFile(f.architecture, engmodel.ThreatModelExportOptions{Format: engmodel.ThreatModelFormatOpenOTM})
				return result.JSON, err
			},
			contains: []string{`"otmVersion"`, `"threats"`},
			json:     true,
		},
		{
			name: "trlc",
			generate: func() (string, error) {
				result, err := engmodel.GenerateTRLCRequirementsFromFile(f.requirements, engmodel.TRLCExportOptions{})
				return result.ModelRSL + "\n" + result.RequirementsTRLC, err
			},
			contains: []string{"type Requirement", "applies_to String [0 .. *]", "REQ-PAY-001"},
		},
	}
}

// assuranceContracts are the activity trace, OSCAL and Gemara exports.
func (f projectionFixture) assuranceContracts() []projectionContract {
	return []projectionContract{
		{
			name: "lobster",
			generate: func() (string, error) {
				result, err := engmodel.GenerateLobsterActivityTraceFromDir(filepath.Join(f.payments, "tests"), engmodel.LobsterActivityExportOptions{
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
				result, err := engmodel.GenerateOSCALSSPFromFile(f.architecture, engmodel.OSCALSSPOptions{})
				return result.JSON, err
			},
			contains: []string{`"system-security-plan"`, `"control-implementation"`, `"implemented-requirements"`},
			json:     true,
		},
		{
			name:     "gemara",
			generate: func() (string, error) { return gemaraCatalogs(f.architecture) },
			contains: []string{"capability-catalog", "control-catalog", "threat-catalog", "ControlCatalog"},
		},
	}
}

// gemaraCatalogs lists the Gemara documents and returns the control catalog.
func gemaraCatalogs(architecture string) (string, error) {
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
}

// modelContracts are the trace matrix, composition and validation outputs.
func (f projectionFixture) modelContracts() []projectionContract {
	return []projectionContract{
		{
			name:     "trace-matrix",
			generate: func() (string, error) { return traceMatrixJSON(f.architecture, f.requirements) },
			contains: []string{`"model"`, `"sample-payments-layered-model"`, `"requirements"`, `"implemented"`, `"verified"`},
			json:     true,
		},
		{
			name:     "composition",
			generate: func() (string, error) { return compositionSummary(filepath.Join(f.coffee, "engmod.yml")) },
			contains: []string{"root=sample-coffee-fleet-ota-cloud-model", "children=3", "links=3"},
		},
		{
			name:     "strict-validation",
			generate: func() (string, error) { return strictValidation(f.architecture) },
			contains: []string{"strict-valid", "schema-v2"},
		},
	}
}

// traceMatrixJSON builds the trace matrix of the model and returns it as JSON.
func traceMatrixJSON(architecture, requirements string) (string, error) {
	matrix, diagnostics, err := engmodel.BuildTraceMatrixFromFiles(architecture, requirements, ".")
	if err != nil {
		return "", err
	}
	if validate.HasErrors(diagnostics) {
		return "", fmt.Errorf("trace matrix diagnostics: %+v", diagnostics)
	}
	data, err := json.Marshal(matrix)
	return string(data), err
}

// compositionSummary composes the model and summarizes its root.
func compositionSummary(manifest string) (string, error) {
	result, err := engmodel.GenerateCompositionFromFile(manifest)
	if err != nil {
		return "", err
	}
	if result.Root == nil {
		return "", fmt.Errorf("composition root is nil")
	}
	return fmt.Sprintf("root=%s children=%d links=%d", result.Root.Bundle.Architecture.Model.ID, len(result.Root.Children), len(result.Root.Bundle.ReqLinks.ReqLinks)), nil
}

// strictValidation validates the model and fails on any error.
func strictValidation(architecture string) (string, error) {
	bundle, err := model.LoadBundle(architecture)
	if err != nil {
		return "", err
	}
	diagnostics := validate.Bundle(bundle)
	if validate.HasErrors(diagnostics) {
		return "", fmt.Errorf("strict validation diagnostics: %+v", diagnostics)
	}
	return "strict-valid schema-v2", nil
}

// mcpContracts are the MCP tool outputs.
func (f projectionFixture) mcpContracts() []projectionContract {
	server := f.mcp()
	return []projectionContract{
		{
			name:     "mcp-model",
			generate: func() (string, error) { return server.toolPayload("model.list", map[string]any{}) },
			contains: []string{`"ok":true`, `"schemaVersion"`, `"tool":"model.list"`, `"entities"`},
			json:     true,
		},
		{
			name: "mcp-ownership",
			generate: func() (string, error) {
				return server.toolPayload("ownership.resolve", map[string]any{"entityId": "FU-PAYMENT-AUTHORIZATION"})
			},
			contains: []string{`"ok":true`, `"tool":"ownership.resolve"`, `"id":"FU-PAYMENT-AUTHORIZATION"`, `"owner"`},
			json:     true,
		},
		{
			name: "mcp-impact",
			generate: func() (string, error) {
				return server.toolPayload("requirements.impact", map[string]any{"requirementId": "REQ-PAY-001"})
			},
			contains: []string{`"ok":true`, `"tool":"requirements.impact"`, `"requirement"`, `"impactedFlows"`, `"relevantFiles"`},
			json:     true,
		},
		{
			name:     "mcp-coverage",
			generate: func() (string, error) { return server.toolPayload("coverage.strictStatus", map[string]any{}) },
			contains: []string{`"ok":true`, `"tool":"coverage.strictStatus"`, `"status"`, `"summary"`},
			json:     true,
		},
	}
}

// mcpModel names the documents an MCP server is initialized with.
type mcpModel struct{ architecture, requirements, design string }

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
func (m mcpModel) toolPayload(tool string, arguments map[string]any) (string, error) {
	server := mcp.NewServer()
	initialize := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath":        m.architecture,
			"requirementsPath": m.requirements,
			"designPath":       m.design,
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
