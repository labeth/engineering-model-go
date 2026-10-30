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
- `internal/engine/inferred_code.go`
- `internal/engine/inferred_code_test.go`
- `internal/engine/inferred_verification.go`
- `internal/engine/inferred_verification_test.go`
- `internal/engine/inferred_layers.go`
- `internal/engine/inferred_layers_test.go`

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

- `internal/engine/threat_model_export.go`
- `internal/engine/threat_model_export_test.go`
- `cmd/engdragon/main.go`

### FU-STRUCTURIZR-EXPORTER

- `internal/engine/structurizr_export.go`
- `internal/engine/structurizr_export_test.go`
- `cmd/engstruct/main.go`

### FU-TRLC-EXPORTER

- `internal/engine/trlc_export.go`
- `internal/engine/trlc_export_test.go`
- `cmd/engtrlc/main.go`

### FU-LOBSTER-EXPORTER

- `internal/engine/lobster_export.go`
- `internal/engine/lobster_export_test.go`
- `cmd/englobster/main.go`

### FU-OSCAL-EXPORTER

- `internal/engine/oscal_compliance.go`
- `internal/engine/oscal_ssp.go`
- `internal/engine/oscal_ar.go`
- `internal/engine/oscal_poam.go`
- `internal/engine/oscal_ssp_test.go`
- `internal/engine/oscal_chain_test.go`
- `cmd/engoscal/main.go`

### FU-GEMARA-EXPORTER

- `internal/engine/gemara_export.go`
- `internal/engine/gemara_evaluation.go`
- `internal/engine/gemara_extended.go`
- `internal/engine/gemara_l1l3.go`
- `internal/engine/gemara_oscal.go`
- `internal/engine/gemara_export_test.go`
- `internal/engine/gemara_oscal_test.go`
- (the `cmd/enggemara/main.go` entrypoint is owned by FU-CLI-ORCHESTRATION)

### FU-SYSTEM-COMPOSITION

- `internal/engine/composition.go`
- `internal/engine/composition_test.go`
- (composition rendering in `internal/engine/asciidoc_composition.go` is owned by FU-ASCIIDOC-GENERATOR)

### FU-ALLOCATION-TRACE

- `internal/engine/trace_matrix.go`
- `internal/engine/trace_matrix_test.go`
- `cmd/engtrace/main.go`

### FU-CLI-ORCHESTRATION

- `cmd/engdoc/main.go`
- `cmd/enggemara/main.go`
- (cross-cutting command orchestration references in other `cmd/*` entrypoints)
