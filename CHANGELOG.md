# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project aims to follow
semantic versioning.

## [Unreleased]

### Changed (breaking)

- Code is scanned from the module root (git working tree root, or the manifest directory
  when nested under another module) minus git-ignored files, nested modules and the new
  `inferenceHints.exclude` globs. `inferenceHints.codeSources` is removed and rejected
  with `model.code_sources_removed`.
- Inferred declaration IDs are path-qualified: `CODE-<PATH>--<NAME>` instead of
  `CODE-<NAME>`, so same-named declarations in different files no longer collide.
- `referencedElements` must describe artifacts outside the repository: a local or
  `file:` uri is rejected (`model.referenced_element_local_uri`), and a mutable uri needs
  a `version` unless it is pinned to a commit or doi.

### Added

- Every in-scope file gets a referenceable `CODE-<PATH>` ID. Authored references to
  `CODE-*` IDs are resolved during validation (unknown → broken reference, collisions →
  `model.ambiguous_code_reference`); the code index is only built when one is used.
Fixes from field use of the trace, MCP and Gemara outputs.

### Changed

- **Trace:** the summary adds `testLinked` (equal to `tested`). `verified` is **deprecated**: it stays as an
  alias of `testLinked` for existing callers, because a linked test is not a verification. `trace.matrix`
  over MCP now reports `tested`, `testLinked` and `passed`, and accepts `summaryOnly` (`true` or `"true"`) to
  return only the summary and diagnostics.
- **Gemara enforcement log:** an action is `Enforced` only for a `completed` POA&M item and `Tolerated` only
  for an accepted one; `planned`, `in-progress`, `deferred` and unset items are `Undetermined`. The aggregate
  is `Undetermined` while any item is unresolved (it was `Enforced` for planned work).
- **Gemara policy:** `risks.mitigated` lists only `closed` risks; `risks.accepted` lists risks with status
  `accepted` or response `accept`. Open and mitigating risks are in neither list (Gemara has no other bucket
  or notes field) and the export diagnostic `gemara.policy_risks_untreated` names them.
- **Gemara metadata:** the default `metadata.author` (the generator) is typed `Software`, not `Human`;
  `GemaraExportOptions.AuthorType` overrides it.
- `catalog.term_unreferenced` counts a term as referenced when its name or alias appears in any authored
  model text other than its own entry (requirement notes, verification criteria and rationale, other
  catalog entries, architecture/behavior/assurance/compliance/views/decisions text). The message now says
  "is not referenced anywhere in the model".
- `golang.org/x/text` v0.39.0 (GO-2026-5970); `golang.org/x/tools` v0.47.0 follows.
- **Trace:** only test entry points verify a requirement (Go `Test/Benchmark/Fuzz/Example`, Rust `#[test]`-like
  functions including inside `#[cfg(test)]` modules, Python `test_*`). `TRLC-LINKS` on helpers, fakes and
  fixtures in test code raise `verification.helper_link` instead of counting. Rust `#[cfg(test)]` code is no
  longer counted as implementation. The summary adds `tested` and `passed` and each row a `result`
  (`passed`/`failed`/`not-run`); `verified` equals `tested` for compatibility. The CSV gains a `result` column.
- **Gemara contacts and adherence:** policy contacts come from model owners ("not set" otherwise); no automated
  gate or continuous evaluation is claimed unless the model supports it; the default risk appetite is disclosed and
  `max-severity` left unset; the enforcement log cites only evaluation-log entries that exist; mapping-reference
  ids match artifact ids.

### Added

- `CheckGemaraModelConsistency`: compares rendered dispositions and risk treatment with the model;
  `gemara.validate` returns `modelConsistencyErrors` and is `valid: false` on any. Adds REQ-EMG-062.
- `model.authoringContract` returns `rules`: allowed enums, every allowed relationship type/from/to pair and
  field rules, built from the validator's own tables (`validate.AuthoringRules`).
- `gemara.validate` checks cross references across the rendered set (`crossReferenceErrors`, `valid`).
- Requirement `lintWaivers: [{code: lint.*, reason}]`; waived warnings are suppressed, errors never are, and a
  waiver without a reason is rejected.
- `validate.DedupeDiagnostics` for callers that merge diagnostic lists.

### Fixed

- `files.owner` and `interfaces.matchFromCode` read only the `ENGMODEL-OWNER-UNIT` marker, not the first unit
  id mentioned anywhere in the file; `files.owner` adds `ownerKnown` and `markerLine`.
- `catalog.term_unreferenced` reports only terms used nowhere in the model.
- Stacked `TRLC-LINKS` lines above one declaration accumulate.

- Keep Manhattan and requirement alignment matrices in single scalable vector figures; keep system boundary and decomposition figures together during pagination.

### Added

- Python declaration tracing with decorator-aware attachment, qualified nested names,
  parser-based comment isolation, syntax diagnostics and Python test-file discovery.

- JavaScript trace extraction for named functions, methods and directly bound
  function expressions in JavaScript, ES modules and CommonJS sources.
- Lexical Verilog module trace extraction, with diagnostics for unresolved macros
  and malformed module boundaries; this does not perform HDL elaboration.
- Individual-file inference roots and JavaScript/Verilog test-source discovery.
- Hardware and hardware-interface view nodes, validated endpoint references,
  allocation relationships and trust-boundary membership.

### Fixed

- Reduce publication edge-label crowding with wider authored/requirement flowchart
  spacing and numbered long deployment relationships with complete description tables.

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
