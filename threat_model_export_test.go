package engmodel

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/validate"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestThreatModelExport_EndToEnd(t *testing.T) {
	examples := []string{
		filepath.Join("examples", "payments-engineering-sample", "architecture.yml"),
		filepath.Join("examples", "bedrock-pr-review-github-app-sample", "architecture.yml"),
		filepath.Join("examples", "coffee-fleet-ota-cloud-sample", "architecture.yml"),
	}
	formats := []ThreatModelFormat{ThreatModelFormatThreatDragonV2, ThreatModelFormatOpenOTM}

	for _, modelPath := range examples {
		for _, format := range formats {
			modelPath := modelPath
			format := format
			t.Run(filepath.Base(filepath.Dir(modelPath))+"-"+string(format), func(t *testing.T) {
				res, err := GenerateThreatModelExportFromFile(modelPath, ThreatModelExportOptions{Format: format})
				if err != nil {
					t.Fatalf("export failed: %v", err)
				}
				if validate.HasErrors(res.Diagnostics) {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.JSON) == 0 {
					t.Fatalf("expected non-empty json output")
				}
				var parsed any
				if err := json.Unmarshal([]byte(res.JSON), &parsed); err != nil {
					t.Fatalf("output is not valid json: %v", err)
				}
			})
		}
	}
}

func TestThreatModelExport_SchemaValidation(t *testing.T) {
	tdSchemaPath := filepath.Join("tools", "threat-dragon-schemas", "threat-dragon-v2.schema.json")
	otmSchemaPath := filepath.Join("tools", "threat-dragon-schemas", "open-threat-model.schema.json")
	if _, err := os.Stat(tdSchemaPath); err != nil {
		t.Skipf("schema missing: %s", tdSchemaPath)
	}
	if _, err := os.Stat(otmSchemaPath); err != nil {
		t.Skipf("schema missing: %s", otmSchemaPath)
	}

	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")

	t.Run("threat-dragon-v2", func(t *testing.T) {
		res, err := GenerateThreatModelExportFromFile(modelPath, ThreatModelExportOptions{Format: ThreatModelFormatThreatDragonV2})
		if err != nil {
			t.Fatalf("export failed: %v", err)
		}
		validateJSONWithSchema(t, res.JSON, tdSchemaPath)
	})

	t.Run("open-otm", func(t *testing.T) {
		res, err := GenerateThreatModelExportFromFile(modelPath, ThreatModelExportOptions{Format: ThreatModelFormatOpenOTM})
		if err != nil {
			t.Fatalf("export failed: %v", err)
		}
		validateJSONWithSchema(t, res.JSON, otmSchemaPath)
	})
}

func TestThreatModelExportCLI_EndToEnd(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	for _, format := range []string{"threat-dragon-v2", "open-otm"} {
		format := format
		t.Run(format, func(t *testing.T) {
			cmd := exec.Command("go", "run", "./cmd/engdragon", "--model", modelPath, "--format", format)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("cli failed: %v\noutput:\n%s", err, string(out))
			}
			var parsed any
			if err := json.Unmarshal(out, &parsed); err != nil {
				t.Fatalf("cli output is not valid json: %v\noutput:\n%s", err, string(out))
			}
		})
	}
}

func validateJSONWithSchema(t *testing.T, jsonText, schemaPath string) {
	t.Helper()
	c := jsonschema.NewCompiler()
	sch, err := c.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile schema failed (%s): %v", schemaPath, err)
	}
	var doc any
	if err := json.Unmarshal([]byte(jsonText), &doc); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if err := sch.Validate(doc); err != nil {
		t.Fatalf("schema validation failed: %v", err)
	}
}
