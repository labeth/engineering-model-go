> Paths before the v0.5.0 restructure; see README for the current layout.

# FU File Instrumentation Checklist

This checklist tracks owner/trace instrumentation by functional unit without behavior changes.

## How to use

For each FU batch:

1. Add `ENGMODEL-OWNER-UNIT: FU-...` to each source file in scope.
2. Add `TRLC-LINKS: REQ-...` markers on representative declarations/tests.
3. Regenerate maintained root artifacts (`generated/ARCHITECTURE.adoc`, `generated/DECISIONS.adoc`, `generated/ARCHITECTURE.proven.pdf`, and affected exchange artifacts).
4. Verify with `go test ./...`.

## Current pass status

- Done in this pass:
  - FU-MCP-SERVER
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
  - FU-GEMARA-EXPORTER
  - FU-SYSTEM-COMPOSITION
  - FU-ALLOCATION-TRACE
  - FU-CLI-ORCHESTRATION (entrypoint coverage)

- Remaining follow-up:
  - Expand instrumentation coverage to additional helper files in each FU.
  - Add/normalize test-side `TRLC-LINKS` for each `REQ-EMG-*` to improve inferred verification coverage.

## Suggested per-FU file sets

> Note: This list is a point-in-time snapshot. The authoritative owner/trace
> assignments are the `ENGMODEL-OWNER-UNIT` and `TRLC-LINKS` markers in source,
> enforced by the `engdoc` 0-error gate and the `engtrace` dangling-link gate
> (exit 1 on unresolved code trace links). When this list and the in-source
> markers disagree, the markers and gates win.

### FU-MCP-SERVER

- `internal/mcp/server.go`
- `cmd/engmcp/main.go`
- `cmd/engmcp/main_test.go`
- `internal/mcp/server_test.go`

### FU-CODEMAP-INFERENCE

- `internal/codemap/scan.go`
- `internal/codemap/scan_test.go`
- `internal/codescan/code_scope.go`, `inferred_code.go`, `inferred_layers.go`, `inferred_verification.go`, `code_labels.go`
- `internal/codescan/*_test.go`
- `internal/trace/inference_trace_test.go` (inference checked through the trace matrix)

### FU-MODEL-LOADER

- `internal/model/load.go`
- `internal/model/load_test.go`
- `internal/model/types.go`

### FU-VALIDATION-ENGINE

- `internal/validate/validate.go`
- `internal/validate/validate_test.go`
- `internal/validate/diagnostic.go`

### FU-ASCIIDOC-GENERATOR

- `internal/engine/asciidoc.go`
- `internal/engine/asciidoc_template.go`
- `internal/engine/asciidoc_views.go`
- `internal/engine/asciidoc_backlinks.go`
- `internal/engine/asciidoc_design_refs.go`
- `internal/engine/asciidoc_diagrams_core.go`
- `internal/engine/asciidoc_diagrams_runtime.go`
- `internal/engine/asciidoc_linking_units.go`

### FU-VIEW-PROJECTION

- `internal/view/project.go`
- `internal/view/project_test.go`
- `internal/view/types.go`
- `cmd/engview/main.go`

### FU-THREAT-EXPORTER

- `internal/export/threatmodel/threat_model_export.go`
- `internal/export/threatmodel/threat_model_export_test.go`
- `cmd/engdragon/main.go`

### FU-STRUCTURIZR-EXPORTER

- `internal/export/structurizr/structurizr_export.go`
- `internal/export/structurizr/structurizr_export_test.go`
- `cmd/engstruct/main.go`

### FU-TRLC-EXPORTER

- `internal/export/trlc/trlc_export.go`
- `internal/export/trlc/trlc_export_test.go`
- `cmd/engtrlc/main.go`

### FU-LOBSTER-EXPORTER

- `internal/export/lobster/lobster_export.go`
- `internal/export/lobster/lobster_export_test.go`
- `cmd/englobster/main.go`

### FU-OSCAL-EXPORTER

- `internal/export/oscal/oscal_compliance.go`
- `internal/export/oscal/oscal_ssp.go`
- `internal/export/oscal/oscal_ar.go`
- `internal/export/oscal/oscal_poam.go`
- `internal/export/oscal/oscal_ssp_test.go`
- `internal/export/oscal/oscal_chain_test.go`
- `cmd/engoscal/main.go`

### FU-GEMARA-EXPORTER

- `internal/export/gemara/gemara_export.go`
- `internal/export/gemara/gemara_evaluation.go`
- `internal/export/gemara/gemara_extended.go`
- `internal/export/gemara/gemara_l1l3.go`
- `internal/export/gemara/gemara_oscal.go`
- `internal/export/gemara/gemara_export_test.go`
- `internal/export/gemara/gemara_oscal_test.go`
- (the `cmd/enggemara/main.go` entrypoint is owned by FU-CLI-ORCHESTRATION)

### FU-SYSTEM-COMPOSITION

- `internal/composition/composition.go`, `composition_confinement.go`, `composition_projection.go`
- `internal/composition/*_test.go`
- `internal/engine/composition_multirepo_test.go`, `internal/engine/composition_integration_test.go` (composition with generators and trace)
- `internal/testutil/fixtures.go` (shared model-module test fixtures)
- (composition rendering in `internal/engine/asciidoc_composition.go` is owned by FU-ASCIIDOC-GENERATOR)

### FU-ALLOCATION-TRACE

- `internal/trace/trace_matrix.go`
- `internal/trace/trace_matrix_test.go`
- `cmd/engtrace/main.go`

### FU-CLI-ORCHESTRATION

- `cmd/engdoc/main.go`
- `cmd/enggemara/main.go`
- `internal/textutil/textutil.go` (shared string and collection helpers)
- `internal/testutil/files.go` (shared test file helpers)
- (cross-cutting command orchestration references in other `cmd/*` entrypoints)
