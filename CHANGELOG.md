# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project aims to follow
semantic versioning.

## [0.3.0] — Source tracing and hardware publications

### Added

- JavaScript trace extraction for named functions, methods and directly bound
  function expressions in JavaScript, ES modules and CommonJS sources.
- Lexical Verilog module trace extraction, with diagnostics for unresolved macros
  and malformed module boundaries; this does not perform HDL elaboration.
- Individual-file inference roots and JavaScript/Verilog test-source discovery.
- Hardware and hardware-interface view nodes, validated endpoint references,
  allocation relationships and trust-boundary membership.

### Fixed

- Keep test-only requirement links separate from production implementation.
- Preserve distinct source identities when separately scanned files share a name
  and declaration location, while deduplicating overlapping scan roots.
- Keep authored view scope and limitations in publications, distinguish repeated
  view kinds, and preserve complete diagrams without splitting them into panels.
- Improve hardware labels, SVG routing, evidence-table wrapping and requirement
  coverage graphs for readable publications.

## [0.2.0] — Schema-v2 domain documents

This release introduces a breaking model input contract. `engmod.yml` is the
only entry point and declares module identity, exact dependencies,
publications, inference hints, and all eight schema-v2 domain-document paths.
Architecture, behavior, assurance, compliance, views, catalog, requirements,
and decisions are authored separately under `model/`.

### Added

- **Gemara is now a first-class rendering** of the model (OpenSSF Gemara,
  https://gemara.openssf.org), built with the official `go-gemara` SDK types and
  validated against the published Gemara CUE schemas. All 13 Gemara artifact types
  are produced: L1 Vector/Principle/Guidance catalogs, L2 Capability/Threat/Control
  catalogs, L3 Risk catalog + Policy, L5 Evaluation Log, L6 Enforcement Log,
  L7 Audit Log, Mapping Document, and Lexicon.
- New `cmd/enggemara` CLI (writes every produced artifact; optional `--oscal-catalog-out`
  / `--oscal-ar-out` Gemara→OSCAL bridge).
- New library entry points (`GenerateGemara*`, `GenerateGemaraEvaluationLog*`,
  `GenerateGemaraOSCAL*`).
- 13 new MCP tools: `gemara.controlCatalog`, `gemara.threatCatalog`, `gemara.riskCatalog`,
  `gemara.vectorCatalog`, `gemara.capabilityCatalog`, `gemara.principleCatalog`,
  `gemara.guidanceCatalog`, `gemara.policy`, `gemara.lexicon`, `gemara.mappingDocument`,
  `gemara.auditLog`, `gemara.enforcementLog`, `gemara.evaluationLog`, plus `gemara.validate`.
- A `Gemara GRC Model` chapter appended to the generated `ARCHITECTURE.adoc`.
- `scripts/validate-gemara.sh` (`cue vet` against the Gemara schemas) and
  `docs/gemara-rendering.md`.
- Self-model additions: `FU-GEMARA-EXPORTER`, `REQ-EMG-015`, `FEAT-GEMARA-EXPORT`,
  authored `risks`/`poamItems`/`threatMitigations`, and ADRs `ADR-EMG-003`…`ADR-EMG-007`.
- **New `cmd/engtrace` CLI and `TRACE-MATRIX.json` traceability matrix** — emits a
  machine-readable traceability matrix (JSON and CSV) with a per-requirement status
  rollup (implemented / verified / delegated / orphan), resolved code references, and
  delegations. The CLI exits non-zero when any code trace link is dangling.
- **System-of-systems composition** — a model may reference downward subsystems either
  from local subdirectories or from external git repositories (cloned into a
  `.engmod/subsystems` cache). Composition enforces a workspace boundary, acyclicity, and
  `provides`/`requires` contract bindings, with diagnostics `composition.cycle`,
  `composition.out_of_workspace`, `composition.missing_ref`, `composition.clone_failed`,
  `composition.unsatisfied_require`, and `composition.untraceable_delegation`.
- **Hardware items and HW/SW interfaces** — model hardware items plus ICD/IRS interfaces
  carrying DO-254 DAL safety levels, part numbers, suppliers, and buses
  (ARINC429 / CAN / SPI / I2C / ethernet / cellular); rendered into `ARCHITECTURE.adoc`.
- **Requirement delegation** — a parent requirement delegates to a subsystem contract
  entry (no tiers). A delegation without a specific target raises
  `composition.untraceable_delegation`, and a delegation counts against the
  `requirement.orphan` check.
- **CI validation gauntlet** — `scripts/validate-all.sh`, wired into
  `.github/workflows/ci.yml`. Strict gates: `go build`, `engdoc` with zero errors,
  `engtrace` with zero dangling links, and artifact-freshness drift detected via `git diff`
  on `ARCHITECTURE.adoc` / `DECISIONS.adoc` / `TRACE-MATRIX.json`. Best-effort gates:
  Gemara `cue vet`, Structurizr DSL (behind `ENGMOD_VALIDATE_STRUCTURIZR=1`), and TRLC.

### Changed (behavioral)

- **Code-linking scanner now attaches markers to package-level `var`/`const` declarations**
  (`codemap/scan.go`; see `ADR-EMG-007`). Consequences:
  - `TRLC-LINKS`/`ENGMODEL-LINKS` markers placed before a package `var`/`const` now
    **link** instead of producing a `code.trace_unattached` warning — so some existing
    warnings disappear.
  - Scan output (and the inferred-code sections of generated docs) **gains** the linked
    `var`/`const` symbols.
  - **No previously-passing model fails**: `var`/`const` are not trace-*required*, so no
    new `code.missing_trlc_link` errors are introduced. Only functions/methods remain
    trace-required.
  - Consumers relying on `var`-marker warnings as a lint signal should note that
    markers before `var`/`const` are now considered valid placements.
- The generated `ARCHITECTURE.adoc` gains a trailing `Gemara GRC Model` chapter. Existing
  chapters are unchanged; consumers that assert an exact document structure should expect
  the additional chapter.
- **Trace-link integrity is now enforced as hard errors.** `TRLC-LINKS` markers must
  resolve to a requirement and `ENGMODEL-LINKS` markers to a model element (code links are
  scoped to the nearest enclosing model root); unresolved markers raise
  `code.dangling_requirement_link` / `code.dangling_model_link`, which fail the `engdoc`
  zero-error gate and make `engtrace` exit `1`. Functions/methods remain trace-required
  (`code.missing_trlc_link`), requirements with internal-only links raise
  `requirement.internal_link`, and untraced requirements raise the `requirement.orphan`
  warning.
- **Composition and delegation are validated.** Subsystem references are checked for
  workspace containment, acyclicity, and satisfied `provides`/`requires` bindings, emitting
  the `composition.*` diagnostics listed above; delegations without a resolvable target are
  rejected (`composition.untraceable_delegation`).

### Dependencies

- Added: `github.com/gemaraproj/go-gemara`, `github.com/defenseunicorns/go-oscal`,
  `github.com/goccy/go-yaml`, `github.com/santhosh-tekuri/jsonschema` (and transitive deps).
  Library consumers will pull these into their dependency graph. No dependency was removed.

## [0.0.1] — Baseline

Initial tagged baseline prior to the Gemara GRC rendering work.
