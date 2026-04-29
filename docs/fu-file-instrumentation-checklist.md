# FU File Instrumentation Checklist

This checklist tracks owner/trace instrumentation by functional unit without behavior changes.

## How to use

For each FU batch:

1. Add `ENGMODEL-OWNER-UNIT: FU-...` to each source file in scope.
2. Add `TRLC-LINKS: REQ-...` markers on representative declarations/tests.
3. Regenerate root artifacts (`generated/*`).
4. Verify with `go test ./...`.

## Current pass status

- Done in this pass:
  - FU-MCP-SERVER
  - FU-AI-VIEW-BUILDER
  - FU-CODEMAP-INFERENCE
  - FU-MODEL-LOADER
  - FU-VALIDATION-ENGINE
  - FU-ASCIIDOC-GENERATOR
  - FU-VIEW-PROJECTION
  - FU-THREAT-EXPORTER
  - FU-STRUCTURIZR-EXPORTER
  - FU-TRLC-EXPORTER
  - FU-LOBSTER-EXPORTER
  - FU-OSCAL-EXPORTER
  - FU-CLI-ORCHESTRATION (entrypoint coverage)

- Remaining follow-up:
  - Expand instrumentation coverage to additional helper files in each FU.
  - Add/normalize test-side `TRLC-LINKS` for each `REQ-EMG-*` to improve inferred verification coverage.

## Suggested per-FU file sets

### FU-MCP-SERVER

- `mcp/server.go`
- `cmd/engmcp/main.go`
- `cmd/engmcp/main_test.go`
- `mcp/server_test.go`

### FU-AI-VIEW-BUILDER

- `ai_view.go`
- `ai_view_schema.go`
- `ai_view_edges.go`
- `ai_view_markdown.go`
- `ai_view_test.go`

### FU-CODEMAP-INFERENCE

- `codemap/scan.go`
- `codemap/scan_test.go`
- `inferred_code.go`
- `inferred_code_test.go`
- `inferred_verification.go`
- `inferred_verification_test.go`
- `inferred_layers.go`
- `inferred_layers_test.go`

### FU-MODEL-LOADER

- `model/load.go`
- `model/load_test.go`
- `model/types.go`

### FU-VALIDATION-ENGINE

- `validate/validate.go`
- `validate/validate_test.go`
- `validate/diagnostic.go`

### FU-ASCIIDOC-GENERATOR

- `asciidoc.go`
- `asciidoc_template.go`
- `asciidoc_views.go`
- `asciidoc_backlinks.go`
- `asciidoc_design_refs.go`
- `asciidoc_diagrams_core.go`
- `asciidoc_diagrams_runtime.go`
- `asciidoc_linking_units.go`

### FU-VIEW-PROJECTION

- `view/project.go`
- `view/project_test.go`
- `view/types.go`
- `cmd/engview/main.go`

### FU-THREAT-EXPORTER

- `threat_model_export.go`
- `threat_model_export_test.go`
- `cmd/engdragon/main.go`

### FU-STRUCTURIZR-EXPORTER

- `structurizr_export.go`
- `structurizr_export_test.go`
- `cmd/engstruct/main.go`

### FU-TRLC-EXPORTER

- `trlc_export.go`
- `trlc_export_test.go`
- `cmd/engtrlc/main.go`

### FU-LOBSTER-EXPORTER

- `lobster_export.go`
- `lobster_export_test.go`
- `cmd/englobster/main.go`

### FU-OSCAL-EXPORTER

- `oscal_ssp.go`
- `oscal_ar.go`
- `oscal_poam.go`
- `oscal_ssp_test.go`
- `oscal_chain_test.go`
- `cmd/engoscal/main.go`

### FU-CLI-ORCHESTRATION

- `cmd/engdoc/main.go`
- (cross-cutting command orchestration references in other `cmd/*` entrypoints)
