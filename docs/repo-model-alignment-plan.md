# Engineering-Model-Go Repository Model Alignment Plan

> Status: historical plan, not a live backlog. The ongoing-alignment quality
> gates described in Phase 5 are now in place — see `scripts/validate-all.sh`
> (wired into `.github/workflows/ci.yml`) and `cmd/engtrace`. The drift and
> traceability gates below have shipped; remaining items are noted inline.

## Scope

Align root model artifacts with the current repository without changing runtime behavior.

- Inputs: `catalog.yml`, `architecture.yml`, `requirements.yml`, `design.yml`
- Generated artifacts: maintained files under `generated/`, excluding removed machine-view artifacts.
- Code scope for alignment: `cmd/`, `mcp/`, root `*.go`, `codemap/`, `validate/`, `view/`, exporters (`*_export.go`)

## Constraints

- No functional code changes while performing the initial alignment pass.
- Prefer metadata-only updates first (model + trace/owner markers in code comments later).
- Keep stable IDs as the primary contract for all updates.

## Phase 1: Baseline and Inventory

1. Regenerate artifacts from root model and confirm success.
2. Build an inventory of repository files by ownership candidate path:
   - `cmd/engdoc` -> `FU-ASCIIDOC-GENERATOR`
   - `cmd/engmcp`, `mcp/` -> `FU-MCP-SERVER`
   - `cmd/engdragon`, `threat_model_export.go` -> `FU-THREAT-EXPORTER`
   - `cmd/engstruct`, `structurizr_export.go` -> `FU-STRUCTURIZR-EXPORTER`
   - `cmd/engtrlc`, `trlc_export.go` -> `FU-TRLC-EXPORTER`
   - `cmd/englobster`, `lobster_export.go` -> `FU-LOBSTER-EXPORTER`
   - `cmd/engoscal`, `oscal_*.go` -> `FU-OSCAL-EXPORTER`
   - `model/` -> `FU-MODEL-LOADER`
   - `validate/` -> `FU-VALIDATION-ENGINE`
   - `codemap/`, `inferred_*.go` -> `FU-CODEMAP-INFERENCE`
   - `view/`, `asciidoc_views.go` -> `FU-VIEW-PROJECTION`

Deliverable: checked-in mapping table (path -> FU ID).

Machine-oriented context for implementation agents comes from MCP tool responses. AsciiDoc/PDF and exchange files are generated publication/interchange artifacts.

## Phase 2: Ownership Marking Pass (No Behavior Changes)

Goal: add one file-level owner marker to each owned source file.

- Marker format: `ENGMODEL-OWNER-UNIT: FU-...`
- Priority order:
  1. `mcp/` and `cmd/engmcp`
  2. exporter files (`*_export.go`, export commands)
  3. loader/validate/inference/view files
  4. remaining command wiring files

Deliverable: owner coverage report and remaining unowned file list.

## Phase 3: Requirement Trace Marking Pass (No Behavior Changes)

Goal: add requirement tags where behavior is implemented or asserted.

- Marker format: `TRLC-LINKS: REQ-EMG-...`
- Apply to:
  - command entrypoints and core functions implementing each requirement
  - tests that verify each requirement
- Start with high-priority requirements:
  - `REQ-EMG-007`, `REQ-EMG-008` (MCP contract/safety)
  - `REQ-EMG-010` (traceability inference)
  - `REQ-EMG-004`, `REQ-EMG-005`, `REQ-EMG-006` (export pipelines)

Deliverable: requirement-to-file matrix with at least one source and one test link per requirement.

## Phase 4: Model-to-Code Reconciliation

1. Compare inferred ownership and verification outputs against expected FUs.
2. Resolve mismatches by updating model mappings first, then comments/tags if needed.
3. Confirm each FU has:
   - owned files
   - linked requirements
   - linked verification evidence

Deliverable: zero unresolved core FU ownership for in-scope files.

## Phase 5: Quality Gates for Ongoing Alignment (implemented)

Repeatable checks now run in `scripts/validate-all.sh`, which is invoked by
`.github/workflows/ci.yml`:

1. Artifact generation gate — IMPLEMENTED:
   - `scripts/validate-all.sh` regenerates `ARCHITECTURE.adoc`, `DECISIONS.adoc`,
     and `TRACE-MATRIX.json` from each model, then fails on drift via
     `git diff` against the committed copies (no stale generated docs).
2. Traceability gate — IMPLEMENTED (as trace-integrity, not test coverage):
   - `cmd/engtrace` exits non-zero on any dangling code trace link
     (`code.dangling_requirement_link` / `code.dangling_model_link`), and the
     `cmd/engdoc` 0-error gate enforces the same unresolved-link errors. The
     pipeline checks link integrity rather than the originally proposed
     "fail if any `REQ-EMG-*` has no linked tests"; an unlinked requirement is
     surfaced as a `requirement.orphan` warning.
3. Ownership gate — NOT IMPLEMENTED:
   - no gate yet fails in-scope source files missing an `ENGMODEL-OWNER-UNIT`
     marker.
4. MCP gate — NOT IMPLEMENTED as a dedicated CI step:
   - the gauntlet runs `go build`; a contract-stability `go test ./mcp ./cmd/engmcp`
     step is still outstanding.

Best-effort gates also wired in: Gemara CUE validation, Structurizr DSL
(behind `ENGMOD_VALIDATE_STRUCTURIZR=1`), and TRLC.

Deliverable: CI checks that detect model/code divergence early — met for the
drift and trace-integrity gates above.

## Suggested Working Rhythm

- Batch size: 15-25 files per PR.
- PR type sequence:
  1. ownership-only,
  2. requirement links,
  3. model mapping corrections,
  4. CI gate tightening.
- Keep each PR behavior-neutral unless explicitly planned otherwise.
