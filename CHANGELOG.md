# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project aims to follow
semantic versioning.

## [Unreleased]

### Added

- Behavior warnings: `behavior.event_unused` (an event in no `triggered_by` relationship,
  transition trigger or flow), `behavior.state_unreachable` (with transitions in the model, a
  state other than the first declared, initial one that no transition enters) and
  `behavior.flow_step_no_onerror` (a `boundaryCrossing: true` step without `onError`), each at
  the entity's field path (REQ-EMG-070).
- `catalog.name_has_operator`: one warning at the catalog name or alias that contains a
  standalone `and`, `or`, `not`, a comma or a parenthesis, instead of the unresolved-term
  diagnostics on every requirement that uses it (REQ-EMG-071).
- `controlVerifications[].verificationRefs` names the tests that verify a control (inferred
  `CODE-<PATH>--<NAME>` IDs of test declarations or requirement IDs);
  `model.invalid_control_verification_ref` rejects other entries. `verification.status_stale`
  warns when those tests exist but the control verification status is unset or `not-run`, or a
  threat mitigation that references it is unset or `planned`; `engdoc`/`GenerateAsciiDoc` and
  the trace matrix report it (REQ-EMG-073).

### Changed

- `lint.multiple_responses` is no longer reported when the only conjunctions follow a single
  response verb inside a `with`, `including` or `of` field list ("…shall publish one world
  state message with UTC time, latitude, longitude and altitude"); genuine double responses
  are still reported (REQ-EMG-072).

## [0.5.0] — 2026-10-30: Root package is the public API; focused internal packages

### Changed (breaking)

- The Go packages `model`, `validate`, `codemap`, `view`, `render/{diagramstyle,mermaid,svg}`
  and `mcp` moved under `internal/`; the root package `engmodel` is the only public Go API.
  It re-exports the model types (`engmodel.Bundle`, `engmodel.Requirement`, ...) and loader
  functions (`LoadBundle`, `LoadCanonicalBundle`, `FindWorkspace`, ...) in `model_api.go`, and
  the validation types and functions in `validate_api.go`. Renamed on the way:
  `validate.Bundle` → `engmodel.ValidateBundle`, `validate.Rules` → `engmodel.ValidationRules`.
  Migrate `model.X` and `validate.X` to `engmodel.X`.
- `model/` holds only this repository's own engineering model YAML. The embedded CUE schema is
  now `internal/model/schema/`; the authoring contract reports schema paths as
  `internal/model/schema/*.cue`.
- The implementation of the root package moved into focused `internal/` packages; the
  repository root holds only the public facade (`doc.go`, `model_api.go`, `validate_api.go`
  and `api_*.go`), which re-exports every previously exported name (types as aliases,
  constants, error variables and functions with unchanged signatures). Importers of
  `engmodel.X` need no change; importers of the former `model`, `validate`, `codemap`,
  `view`, `render/*` and `mcp` package paths must move to `engmodel`. Final layout:
  `internal/{model,validate,codemap,view,render/*,composition,codescan,trace,reqlint,
  reqdelta,generate,asciidoc,export/*,mcp,engws,cliflags,sysmlmetamodel,textutil,testutil}`
  plus the test-only `internal/integration`; `layering_test.go` enforces that imports point
  down the layers and that only `cmd/*`, `internal/mcp` and `internal/engws` import the root.
- The exporters moved to one package per format under `internal/export/` (`gemara`, `oscal`,
  `sysml`, `structurizr`, `threatmodel`, `naf`, `trlc`, `lobster`, `airborne`); the root
  facade keeps their public names.
- Assets moved next to their package: `templates/asciidoc.tmpl` → `internal/asciidoc/templates/`,
  the Structurizr and TRLC templates → `internal/export/{structurizr,trlc}/templates/`,
  `tools/sysml/metamodel-*.json` → `internal/export/sysml/metamodel/` (the `sysmlmetamodel`
  defaults follow), `testdata/threat-dragon-schemas/` →
  `internal/export/threatmodel/testdata/threat-dragon-schemas/`.
  The SysML toolchain pins moved to `scripts/sysml-toolchain.env`. No top-level `templates/`,
  `tools/` or `testdata/` remain.
- Inferred code IDs of the moved files change with their path (`CODE-TRACE-MATRIX-GO--...` →
  `CODE-INTERNAL-TRACE-TRACE-MATRIX-GO--...`); generated trace outputs change once. The SysML
  coverage evidence names the new paths. Gemara evaluation-log steps, which go-gemara names
  after the Go step function, now read
  `github.com/labeth/engineering-model-go/internal/export/gemara.gemaraInferredStep`.

## [0.4.0] — 2026-10-30: Path-derived code scope, honest trace and Gemara output, local multi-repository models

Code references derived from repository paths, fixes from field use of the trace, MCP and Gemara outputs,
and multi-repository support from a multi-repository spike: everything runs locally (CLIs, `engws`, a
per-checkout stdio MCP server) and dependencies come from the OCI registry into the local cache.

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
- A requirement `appliesTo` naming an unknown local ID is
  now the error `requirement.invalid_applies_to` in model validation (it was only checked for
  requirements deltas); `alias::ID` entries stay with composition.
- An `alias::ID` reference in a model without composition (no dependency publishes it) is
  now the error `composition.unpublished_reference`; it used to pass until a parent composed the model.

### Added

- Every in-scope file gets a referenceable `CODE-<PATH>` ID. Authored references to
  `CODE-*` IDs are resolved during validation (unknown → broken reference, collisions →
  `model.ambiguous_code_reference`); the code index is only built when one is used.
- **Workspace selection** (REQ-EMG-063): `ENGMOD_WORK=off` ignores every `engmod.work.yml`;
  `ENGMOD_WORK=<path>` uses that file only. `model.FindWorkspaceWith`, `engmodel.ResolutionOptions`, and
  `--workspace=off|<path>` on `engtrace`, `engdoc`, `enggemara`, `engoscal`, `engmcp`.
- **Lock policy** (REQ-EMG-064): with workspace replacements active the committed `engmod.lock.yml` is never
  written; the resolved lock goes to `engmod.lock.local.yml`. `ENGMOD_LOCK=check` / `--lock=check` compares
  with the committed lock and reports `composition.lock_missing`, `composition.lock_invalid` or
  `composition.lock_mismatch` (missing, extra or changed dependency; changed digest after a moved tag).
- **Qualified references** (REQ-EMG-065): validation and the trace resolve `alias::ID` against the published
  entities of selected dependencies, including ownership (FU group, interface owner, trust boundary parent,
  risk/threat/verification owners) and view roots. An unpublished reference is one
  `composition.unpublished_reference` error with its field path.
- **Composition diagnostics** (REQ-EMG-066): `GenerateAsciiDoc` returns them and composes once per call
  (`AsciiDocOptions.Composition` reuses a result). `composition.unsatisfied_require` is an error at the root of
  a `kind: system` composition; `composition.allocation_target_not_contract` warns on non-contract targets.
- **Delegated status** (REQ-EMG-067): delegated trace rows carry `delegatedStatus` with the target
  requirement's status in its repository; the summary adds `delegatedImplemented` and `delegatedTestLinked`.
- **`cmd/engws`** (REQ-EMG-068): `status`, `sync`, `pin`, `unpin`, `skew`, `check`, `bump` for a workspace of
  checkouts; local only. `sync` writes `engmod.work.yml` and `go.work` in the workspace root, `pin`/`unpin`
  write `.engws.yml` there, and `bump` writes inside the consumer checkout (`engmod.yml`, `go.mod`, `go.sum`,
  `engmod.lock.yml`).
- **MCP output paths** (REQ-EMG-069): every tool response and tool error is scrubbed of absolute paths (whole
  path matches only), JSON-RPC errors included: project paths become relative, workspace paths `workspace:<dir>/…`, the home directory `~`.
- **MCP read-only defaults** (REQ-EMG-069): `engmcp` sets `ENGMOD_LOCK=check` and `ENGMOD_MODULE_CACHE=reuse` unless
  set, so no composing tool (exporters included) writes the lock or contacts the registry with a warm cache.
- **MCP authoring help** (REQ-EMG-069): `model.complete` (local and imported IDs, catalog terms, enum values,
  relationship types) and `model.diagnose` (validate one document's edited content in memory).
- `CheckGemaraModelConsistency`: compares rendered dispositions and risk treatment with the model;
  `gemara.validate` returns `modelConsistencyErrors` and is `valid: false` on any. Adds REQ-EMG-062.
- `model.authoringContract` returns `rules`: allowed enums, every allowed relationship type/from/to pair and
  field rules, built from the validator's own tables (`validate.AuthoringRules`).
- `gemara.validate` checks cross references across the rendered set (`crossReferenceErrors`, `valid`).
- Requirement `lintWaivers: [{code: lint.*, reason}]`; waived warnings are suppressed, errors never are, and a
  waiver without a reason is rejected.
- `validate.DedupeDiagnostics` for callers that merge diagnostic lists.

### Changed

- **Trace:** the summary adds `testLinked` (equal to `tested`). `verified` is **deprecated**: it stays as an
  alias of `testLinked` for existing callers, because a linked test is not a verification. `trace.matrix`
  over MCP now reports `tested`, `testLinked` and `passed`, and accepts `summaryOnly` (`true` or `"true"`) to
  return only the summary and diagnostics.
- **Gemara enforcement log:** an action is `Enforced` only for a `completed` POA&M item and `Tolerated` only
  for an accepted one; `planned`, `in-progress`, `deferred` and unset items are `Undetermined`. The aggregate
  is `Undetermined` while any item is unresolved (it was `Enforced` for planned work).
  An action is rendered only for a POA&M item whose risk has a control verification in the evaluation log
  (Gemara requires every action to cite an assessment); other items are reported as
  `gemara.enforcement_unjustified`, and no enforcement log is written when none qualifies.
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
- A resolved dependency's code scope (delegated status, REQ-EMG-067) is its own module directory — the
  workspace checkout or the module cache entry — minus its `inferenceHints.exclude` globs, never the
  consumer repository around it; a dependency with no code there reports `not-available`.

### Fixed

- `files.owner` and `interfaces.matchFromCode` read only the `ENGMODEL-OWNER-UNIT` marker, not the first unit
  id mentioned anywhere in the file; `files.owner` adds `ownerKnown` and `markerLine`.
- `catalog.term_unreferenced` reports only terms used nowhere in the model.
- **Multi-repository acceptance run fixes:**
  - `engws check` runs a repository's own `.github/engmod-check` with `GOWORK` set to its
    `.github/engmod-check/go.work` when that file exists (a deliberately local engine) and `off` otherwise;
    the workspace root `go.work` is never inherited (`engws.ToolGoWork`).
  - `engws sync` writes a root `go.work` that uses every checkout with `go.mod`, not only the local ones, so
    `go test` and `go run` work in every checkout. `status`/`skew` report a consumer `go.mod` require of a
    checkout's Go module at another version than the checkout's tag as `SKEW <consumer> go:<module> ...`.
  - `engws bump` updates `go.sum` with the `go.mod` require (`go get module@version`, `GOWORK=off`, the user's
    `GOPROXY`/`GOFLAGS`), so `GOWORK=off go test` passes; any failure restores `engmod.yml`, `go.mod` and
    `go.sum`.
  - The module cache no longer masks a moved tag (REQ-EMG-064): `ENGMOD_MODULE_CACHE=refresh`
    (`ResolutionOptions.ModuleCache`, `engws check --refresh`) re-fetches every dependency; lock
    regeneration (write mode with `ENGMOD_WORK=off`, as in `engws bump`) always re-fetches; lock check mode
    compares each cached module's manifest digest with the registry (one lookup per dependency; offline it
    reuses the cache) and re-fetches a moved tag, which then fails as `composition.lock_mismatch`.
  - `model.diagnose` checks the submitted content's `alias::ID` references and composition: dependencies
    resolve from the real project, the root system is the in-memory bundle
    (`engmodel.GenerateCompositionForBundle`, also used by `GenerateAsciiDoc` and the trace matrix), so an
    unpublished reference in the content is reported for the document and a broken reference on disk that
    the content fixes is not.
  - MCP `trace.matrix` returns each delegated row's `delegatedStatus` and the summary's
    `delegatedImplemented` and `delegatedTestLinked` (also with `summaryOnly`).
  - Composition diagnostics located in a fetched dependency name it as `module@version:path` instead of the
    `.engmod/modules/<hash>` cache path; `composition.lock_mismatch` names `module@version` without the
    major suffix twice (`nats-interface@v0.1.0`, not `nats-interface@v0@v0.1.0`).
  - `engws check --all` no longer warns that `engmod.work.yml` lacks the unchanged checkouts it checks on
    purpose.
- Stacked `TRLC-LINKS` lines above one declaration accumulate.
- **Security review fixes:**
  - Dependency confinement (REQ-EMG-049): a resolved dependency (registry module or workspace replacement)
    whose manifest names a document, `inferenceHints.runtimeSources` entry or `inferenceHints.exclude` glob that is absolute
    or resolves (after symlinks) outside the dependency module is rejected as
    `composition.dependency_path_escape`, naming the alias and field, before any of its documents is read; no
    content of the target reaches a diagnostic. The root model's own paths are unchanged.
  - Module zips (REQ-EMG-047): a zip entry that is not a regular file or directory (a symlink) fails the
    fetch; `..` and absolute entries (rejected by `modzip.Unzip`) and a blob whose bytes do not match its
    digest (rejected by the OCI client) are covered by tests.
  - `engws check` runs each checkout's own `.github/engmod-check` with an allowlisted environment (`PATH`,
    `HOME`, Go, CUE, proxy, certificate, XDG, `DOCKER_CONFIG`, `CI`, `ENGMOD_*`); API keys and other
    credentials are no longer passed. The help and the check output state that every checkout under the
    root is trusted code and name `--in-process` as the choice for untrusted checkouts.
  - `engws` root search stops below `$HOME` (never `$HOME` or an ancestor) and refuses a world-writable
    root unless `--root` names it; every command prints the root first (`skew` on stderr).
  - `engws sync` skips with a warning a checkout whose `module.path` is not a valid major-qualified CUE
    module path or whose directory name is outside `[A-Za-z0-9._-]`, so neither reaches the work files.
  - `engws bump` validates the Go module path (`golang.org/x/mod/module.CheckPath`) before `go get`.
  - MCP `composition.resolve` returns `resolvedDirectory` (also in provenance) relative to the workspace
    root, or `module@version` for module cache entries, never an absolute path
    (`CompositionResult.DisplayDir`).
  - MCP read tools never write the lock and work offline (REQ-EMG-069): `model.complete`, `model.diagnose`,
    `composition.resolve` and `trace.matrix` resolve with `ENGMOD_LOCK=check` unless `ENGMOD_LOCK` is set and
    with the new explicit `ENGMOD_MODULE_CACHE=reuse` / `ModuleCacheReuse` (cache reused in every lock mode)
    unless `ENGMOD_MODULE_CACHE` is set, so a call makes no registry request with a warm cache. Verifying the
    cache against the registry stays with the CLI/CI check path (`ENGMOD_LOCK=check`). `model.complete` lists
    imported IDs also while the lock is missing or stale.
  - `ENGMOD_LOCK=check` with workspace replacements active warns `composition.lock_check_skipped` instead of
    skipping the lock comparison silently (REQ-EMG-064).
  - Qualified-reference validation ignores `schemaRef`, `catalogRef` and `href` (paths and URLs, e.g. an IPv6
    URL with `::`); prose fields were already ignored and are now covered by a test (REQ-EMG-065).
  - `engws check` in-process: the pre-REQ-EMG-065 enrichment fallback (`library, enriched`) is removed.

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
