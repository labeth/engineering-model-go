// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// reportSchemaJSON is the JSON Schema (draft 2020-12) of the report. The
// published copy is docs/health-report.schema.json; go:embed cannot reach
// docs/, so this package keeps the copy it embeds, and a test keeps both
// byte-identical.
//
//go:embed health-report.schema.json
var reportSchemaJSON []byte

// ReportSchemaJSON returns the JSON Schema document of engmod-health/1.
//
// TRLC-LINKS: REQ-EMG-157
// ENGMODEL-LINKS: FU-CODE-HEALTH
func ReportSchemaJSON() []byte {
	return append([]byte(nil), reportSchemaJSON...)
}

var (
	compiledOnce   sync.Once
	compiledSchema *jsonschema.Schema
	compiledErr    error
)

// compiledReportSchema compiles the embedded schema once.
//
// TRLC-LINKS: REQ-EMG-157
func compiledReportSchema() (*jsonschema.Schema, error) {
	compiledOnce.Do(func() {
		c := jsonschema.NewCompiler()
		c.Draft = jsonschema.Draft2020
		const name = "health-report.schema.json"
		if compiledErr = c.AddResource(name, bytes.NewReader(reportSchemaJSON)); compiledErr != nil {
			return
		}
		compiledSchema, compiledErr = c.Compile(name)
	})
	return compiledSchema, compiledErr
}

// ValidateReportJSON checks a JSON document against the schema of
// engmod-health/1 and returns the first violations in one error.
//
// TRLC-LINKS: REQ-EMG-157
// ENGMODEL-LINKS: FU-CODE-HEALTH
func ValidateReportJSON(data []byte) error {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("health report is not JSON: %w", err)
	}
	return validateReportDocument(doc)
}

// validateReportDocument checks an already decoded document against the schema.
//
// TRLC-LINKS: REQ-EMG-157
func validateReportDocument(doc any) error {
	schema, err := compiledReportSchema()
	if err != nil {
		return fmt.Errorf("health report schema: %w", err)
	}
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("health report does not conform to engmod-health/1: %w", err)
	}
	return nil
}
