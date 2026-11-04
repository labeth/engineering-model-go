# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project aims to follow
semantic versioning.

## [Unreleased]

### Changed

- **Requirement links are reviewed by version, not by fingerprint** (D23, ADR-EMG-030;
  REQ-EMG-094/095/097/100/101/104/105/106/107, REQ-EMG-109 retired). A link is
  `{id, parent | evidence, child, pinned: {version}, status, rationale?, reviewedBy?, reviewedOn?}`:
  published versions are immutable, so the pinned version is the review record.
  - The check compares no content: `link.version_mismatch` when the pin differs from
    `engmod.yml`, `link.broken` when the item is gone at the published pin. A dependency
    resolved from a workspace replacement raises no link finding (state `local`), so editing a
    child checkout never churns its users. `link.suspect` and `link.fingerprint_outdated` are
    removed; diagnostics name the link ID only.
  - `engws bump` compares each link between the pinned and the new published version by
    meaning (item kind, text and criteria; each realizer's text and criteria; not status,
    notes, criticality, methods, appliesTo, title or tags): same meaning moves the pin,
    otherwise the old pin stays for review. One line per link (`--show` adds the text). The
    bump is now always written; exit 1 when a link needs review unless `--allow-review`
    (`--allow-suspect` stays as an alias). An unresolvable pinned version is left for review.
  - `engws review-links` lists version-mismatched, broken and claimed links one line each with
    the change by ID; `--show` adds the parent text and the item before and after. It composes
    with the workspace off and verifies cached modules against the registry digest, so a
    re-staged version is refetched. `--accept` writes the dependency pin (no fingerprint) and
    re-evaluates the link, exiting 1 when it still does not pass.
  - `engws migrate-links` writes allocations to an assumption of use as `evidence` links and
    removes every recorded fingerprint (count printed).
  - Trace summary: new `linksNeedReview` and `linksVersionMismatch`; `linksSuspect` is a
    deprecated alias of `linksVersionMismatch`. MCP `links.list` adds `needsReview` and `local`
    and drops the fingerprints; `links.suspect` (name kept) lists the links needing review with
    `before`, `after` and `changes`. AsciiDoc Contract Items no longer print a fingerprint.
  - Facade: `ContractItemFingerprint`, `CanonicalContractItem`, `ContractItemFingerprintVersion`,
    `ContractItemCanonicalVersion`, `SuspectLinkMessage`, `OutdatedLinkMessage`,
    `LinkStateSuspect` and `LinkStateFingerprintOutdated` are removed; `ContractItemMeaning`,
    `SameContractItemMeaning`, `ContractItemMeaningChanges` and `LinkStateLocal` are added.
  - Examples (`requirement-links`, `coffee-fleet-ota-cloud-sample`, `atlas-industries`) record no
    fingerprints.

### Deprecated

- `pinned.fingerprint` in `model/req-links.yml` (REQ-EMG-118): accepted for one release,
  ignored, reported as `link.fingerprint_deprecated` (warning); `engws migrate-links` removes it.

### Added

- `engws status-suggest [--repo DIR] [--tree HASH] [--json]` (REQ-EMG-122): prints the status
  each requirement earns from this tree's evidence (implemented: code linked; verified: also an
  ungated, non-benchmark test link and a RED line in `.engflow/red-checks.jsonl` for the
  working-tree hash naming it) and each control verification (pass when every requirement it
  names is verified). Writes nothing; the owner approves the changes.
- `verification.benchmark_only` (warning, REQ-EMG-121): a requirement whose
  `verificationCriteria` name a number and whose only linked tests are benchmarks (Go
  `Benchmark*`, Rust `#[bench]`, now detected as a test entry point); the trace row gets
  `benchmarkOnly`. Criterion and JS bench files are not classified.
- MCP `repo` argument (REQ-EMG-120, D24): the model-reading tools (`model.diagnose`,
  `model.patch`, `trace.matrix`, `links.*`, `requirements.*`, `gemara.*` and others) take an
  optional `repo`, a family member checkout inside the workspace root, so one server answers
  for every member. The workspace is resolved on every call (with the upward search's trust
  check), so a server started before `engws sync` wrote `engmod.work.yml` sees it; `repo` is
  confined to the root after symlinks and must hold `engmod.yml`.
- `engws link add [--repo DIR] (--parent REQ | --evidence REQ[,REQ]) <alias::CAP-…|alias::AOU-…>
  [--rationale TEXT]` (REQ-EMG-119): writes a claimed link pinned to the dependency version after
  checking the parent and that the child is published at that version; refuses duplicates;
  creates `model/req-links.yml` and `documents.reqLinks` when missing.

- MCP `model.patch` (REQ-EMG-113, ADR-EMG-029): edit one model entity by stable ID
  (`{document, id, set, unset, append, remove, write}`) with line-preserving YAML edits,
  validate the whole model like `model.diagnose` and return `valid`, `diagnostics`, a unified
  diff `patch` and `written`. It is the one MCP tool that writes: only with `write: true`, only
  a valid result, only a manifest document inside the model root (not `engmod.yml` or the
  requirement link file), atomically keeping the file mode. `internal/yamledit` gains
  `FindByID`, `SetPath`, `UnsetPath`, `AppendPath`, `RemovePath`, `UnifiedDiff` and
  `WriteFileAtomic`.
- Authoring contract field types (REQ-EMG-115): `rules.fieldTypes` gives each field of every
  `fieldSets` kind as `{name, type, of?}` (`string`, `bool`, `int`, `list<string>`,
  `list<object>`, `object`, `map<string,string>`; `of` names the kind of an object field),
  derived from the strict-decoding types. New field rules state the flow step `ref` and
  unique-step-ID rules, the exact `catalog.term_unreferenced` rule and the exclude forms.
- `risk.acceptance_by_decision_only` (warning, REQ-EMG-116): an `acceptedBy` of a risk or threat
  scenario with a `decisionRef` but no `statement`; record the owner's own words on the risk.
- Assurance consistency warnings (REQ-EMG-117): `risk.severity_inconsistent` (one
  likelihood × impact pair rated with two severities across threat scenarios),
  `controlVerification.risk_control_unlinked` (a verification's risk does not list its control)
  and `threat.attack_vector_missing` (a threat scenario without `attackVectorRef` while others
  have one). The Atlas `shared-compliance-baseline` example now lists
  `CTRL-COMP-PERSONNEL-SCREENING` on `RISK-COMP-RETAINED-ACCESS`.
- `expr.unknown_term` and `catalog.<role>_unresolved` name the term and the catalog section it
  belongs in, from its clause position (REQ-EMG-123): `unknown term "pump starts" in
  expression; add it to catalog.events (or an alias of an existing entry)`.

### Fixed

- MCP code tools scan the check's code scope (REQ-EMG-114): `coverage.strictStatus`,
  `files.for*`, `*.implementations`, `tests.*` and `code.contextForTask` now leave out
  git-ignored files, `inferenceHints.exclude` paths and nested modules
  (`codescan.ScanCodeScope`), so git-ignored build output no longer fails
  `coverage.strictStatus` while the check passes.
- `inferenceHints.exclude` accepts `dir`, `dir/`, `dir/*`, `dir/**` and `./dir` alike (each
  leaves the whole directory out). A pattern the scan cannot apply (absolute, `..`, `**` other
  than a trailing `/**`, malformed) is the new error `model.invalid_exclude_pattern`, which names
  the accepted forms.

## [0.8.0] — 2026-11-03: Child-to-parent requirement links across composed models

### Added

- Requirement links across composed models (REQ-EMG-090 to REQ-EMG-099, REQ-EMG-108 to
  REQ-EMG-112; ADR-EMG-022 to ADR-EMG-028; design in
  `docs/requirement-links.md`):
  - `requirements[].realizes` names the local contract items (`CAP-*` capabilities,
    `AOU-*` assumptions of use) a requirement exists to deliver; `contract.assumptions`
    (`id`, `text`, `verificationCriteria`) declares assumptions of use, published and
    imported (`alias::AOU-*`) like capabilities.
  - `engmod.yml` `requirementLinks: strict | guided` (default `guided`) and the optional
    document `documents.reqLinks` (`model/req-links.yml`, CUE schema `reqlinks.cue`) with
    `reqLinks: [{id, parent, child, evidence, pinned: {version, fingerprint}, status,
    rationale, reviewedBy, reviewedOn}]`.
  - Upward trace (D22, ADR-EMG-028): no model can know it is top level, so every
    requirement traces through a capability. `requirement.no_parent` is reported unless the
    requirement (a) `realizes` a local capability or assumption of use, (b) is the `parent`
    of a link to a dependency's capability, or (c) is `derived: true` with
    `derivedRationale`; the message names these three options. Stakeholder `sourceRefs` do
    not count and there is no top-level exemption. The trace `unparented` count follows the
    rule, and `parents` lists only the realized items. A system publishes its own
    capabilities so a system above can link to them.
  - Validation: `requirement.no_parent`, `requirement.realizes_unknown`,
    `contract.unrealized` (warning), `contract.realizer_unpublished`, `contract.ref_deprecated`
    (warning), `composition.allocation_deprecated` (warning), `link.parent_unknown`,
    `link.child_unpublished`, `link.version_mismatch`, `link.claimed_only` (warning),
    `model.invalid_requirement_links`. The composed check adds `link.suspect`,
    `link.broken`, `link.child_unpublished` and `link.aou_unverified`; trace integrity adds
    `requirement.unsatisfied`. `requirement.no_parent`, `link.aou_unverified` and
    `requirement.unsatisfied` are errors in strict mode and warnings in guided mode, where
    guided applies them only to a model that publishes contract items or has links.
  - Root facade: types `ContractItem`, `ContractItemRealizer`, `ContractItemKind`,
    `LinkState`, `LinkStatus`, `LinkReason`, `LinkDependency`, `LinkUse`, `ReqLink`,
    `ReqLinkPin`, `ReqLinksDocument`, `ContractAssumption`, `TraceLinkRef`; functions
    `ContractItemFingerprint`, `ContractItemFingerprintVersion`, `CanonicalContractItem`,
    `ContractItemsFromBundle`, `PublishedContractItems`, `RealizedContractItems`,
    `ContractCapabilityRefs`, `ContractNonItemReason`, `SortedContractItemRealizers`,
    `EvaluateRequirementLinks`, `RequirementLinkDiagnostics`, `SuspectLinkMessage`,
    `OutdatedLinkMessage`, `LinkDependencies`, `WorkspaceLinkUsers`, `RequirementLinkMode`;
    constants `ContractItemCanonicalVersion`, `LinkState*`, `LinkReason*`,
    `RequirementLinksStrict`/`Guided`, `ReqLinkClaimed`/`Accepted`. The fingerprint is
    `sha256:v1:<hex>` of the canonical form of a contract item with its realizing requirements
    (item id, kind, text, criteria; realizer id, text, notes, criteria, sorted verification
    methods, criticality, status), pinned by a golden test; another form is reported as
    `link.fingerprint_outdated`. Link evaluation: unchanged, suspect, broken,
    version_mismatch, child_unpublished, unresolved, fingerprint_outdated.
  - Validation also reports `link.review_unrecorded` (an accepted link without a model actor
    in `reviewedBy` or a calendar `reviewedOn`) and `link.parent_missing` (a capability link
    without `parent`).
  - Example `examples/requirement-links/` with three strict repositories (child-a, child-b,
    system) resolved offline through `engmod.work.yml`.
- Requirement link tooling in `engws` (REQ-EMG-100 to REQ-EMG-104; decisions D8 and on in
  `docs/requirement-links.md`):
  - `engws bump <repo> <alias> <version> [--allow-suspect]` evaluates the consumer's links on
    the alias at the new version and prints each as unchanged, suspect or broken. Unchanged
    links move `pinned.version` to the new version; suspect and broken links keep their pin
    and fingerprint (never re-stamped). A suspect or broken link aborts the bump with every
    file restored, the requirement link file and the lock included, unless `--allow-suspect`.
  - `engws review-links [--repo DIR] [--json]` lists suspect, broken and version-mismatched
    links with the parent requirement text and the child item and its realizing requirements
    before (at the link's pinned version, from the module cache or registry; "not available
    offline" otherwise) and after (exit 1 when any). `--accept LNK-ID --by ACT-ID --rationale
    TEXT` writes the current fingerprint, the dependency pin, `reviewedBy`, `reviewedOn`, the
    rationale and `status: accepted`, editing only those lines; an unknown actor or a broken
    link is refused. The fingerprint is that of the published dependency at its pin (composed
    with the workspace off), never of a workspace checkout.
  - `engws contract-diff <from-ref> <to-ref> [--repo DIR] [--json]` compares the published
    contract items of the model at two git refs (added, removed, changed text or realizers,
    unchanged) and states the semver impact: a removed capability, a changed capability text
    ("treated as major"), an added or changed assumption of use are major; a removed
    assumption, an added capability or realizer-only changes are minor; else patch.
  - `engws release-check [--repo DIR] [--tag vX.Y.Z]` reports `dependency.unreleased_pin` for a
    pre-release or pseudo-version in `engmod.yml` or `engmod.lock.yml`, and
    `contract.semver_too_small` when the tag's bump over the previous release (the nearest
    release tag in the commit's history, `git tag --merged`) is smaller than the contract diff
    requires (in v0.x a breaking change needs a minor bump). Exit 0 ok, 1 findings, 2 usage
    or IO error. CI runs it on every `v*` tag push (`release-check` job).
  - `engws migrate-links [--repo DIR] [--dry-run]` converts each `composition.allocations` entry
    into a claimed link in `model/req-links.yml` (`LNK-<REQ>-<ALIAS>`, the dependency pin, the
    current fingerprint, or `""` with a note when the dependency cannot be resolved), adds
    `documents.reqLinks`, removes the converted allocations, keeps (and names) an allocation
    whose target is an interface or other non-item contract entry (D13), turns each capability
    `contract.provides[].ref` into `realizes` and prints the requirements left with
    `requirement.no_parent`. Other lines are kept; a second run changes nothing; a failed
    write restores every file.
  - Every `engws` command accepts its flags after the positional arguments.
- Requirement links in the trace (REQ-EMG-105): trace matrix rows carry `parents` (realized
  contract items, qualified on published rows, or stakeholder `sourceRefs`), `children` (a
  system requirement's linked child items and their realizing requirements), `links` with their
  state and `satisfiedBy: links`: a system requirement without code or tests of its own is
  implemented (and test linked) when every realizing requirement of every linked item is, in
  its own repository, and is no longer an orphan, through accepted, unchanged links only (a
  claimed or changed link gives `satisfiedBy: links-unreviewed` and does not count). The summary counts `linksSuspect`,
  `linksBroken`, `linksClaimed` and `unparented`; MCP `trace.matrix` returns them, also with
  `summaryOnly`. Root facade: `TraceLinkRef`.
- MCP requirement link tools (REQ-EMG-106): `links.list`, `links.suspect`, `links.review`
  (returns the `engws review-links --accept` command and the link file patch; writes nothing)
  and `requirements.parents`; `requirements.get` adds `parents`, `links` and `children`.
- AsciiDoc requirement link chapters (REQ-EMG-107): "Requirement Links" in a model with links
  (requirement → link with state and review status → child item → realizing requirements
  with their implementation and test status in the child; assumptions of use with their
  evidence; suspect, broken and claimed links marked) and "Contract Items" in a model that
  publishes items (text, realizing requirements, fingerprint). "Used By" is opt-in
  (`AsciiDocOptions.UsedBy`, `engdoc --used-by`) and needs an active workspace whose sibling
  checkouts link the items (`WorkspaceLinkUsers`, computed at render time, replacements
  outside the workspace and siblings naming paths outside their directory skipped); do not
  commit such a render into the child. `link.*` findings and the strict-mode
  `requirement.no_parent`, `requirement.unsatisfied` and `contract.realizer_unpublished` no
  longer stop the render; `engdoc` still exits 1 on them. Dependency text is rendered without
  AsciiDoc passthrough.
- `pinned.fingerprint` may be `""` (a link migrated before its item could be fingerprinted).
- Compatibility (ADR-EMG-027): upgrade the engine of every using system to v0.8.0 before a
  dependency publishes a release that uses `realizes` or `contract.assumptions`.
- A dependency `version` may be a Go-style pseudo-version (`v1.2.4-0.20261102103400-abcdef123456`).

### Changed

- The upward search for `engmod.work.yml` refuses a world-writable file or one owned by
  another user (name it with `ENGMOD_WORK=<path>` to use it anyway).
- `requirement.orphan` diagnostics carry the path `requirements[<ID>]`.
- engws keeps the permission bits of the files it rewrites; `contract-diff` writes its
  temporary model copy 0700/0600.

### Fixed

- A requirement imported from a dependency publication (OSCAL assessment results enrich the
  bundle with requirements) keeps its `realizes` qualified with the dependency alias and is not
  checked against the importing model's contract (`requirement.realizes_unknown`).
- Examples: `coffee-fleet-ota-cloud-sample` (with its subsystems) and every
  `atlas-industries` repository are migrated to requirement links (`engws migrate-links`):
  no `composition.allocations` or contract `ref` remain, every child requirement realizes a
  capability or is derived; the coffee fleet system publishes its own capabilities
  (`CAP-COFFEE-FLEET-TELEMETRY`, `-OTA`, `-AUDIT`) that its unlinked requirements realize. The
  Atlas links are accepted; the coffee fleet links stay `claimed` as the migration writes them.

### Deprecated

- `composition.allocations` and `contract.provides[].ref`: still read for this release; use
  `reqLinks` and `realizes`.

## [0.7.0] — 2026-11-02: Assurance and acceptance diagnostics; engws go.work replaces; gated tests; fresh MCP renders

### Added

- Assurance link diagnostics (REQ-EMG-082): `threat.verification_missing` (warning at a threat
  scenario's `verificationRefs` that omit a control verification its mitigations reference or
  that names it in `threatScenarioRefs`), `mitigation.verification_control_mismatch` (error at a
  mitigation `verificationRefs` entry whose control verification verifies another control) and
  `verification.unlinked` (warning at a `method: test` control verification with status `pass`
  or `partial` and no `verificationRefs`).
- `gemara.control_needs_review` (REQ-EMG-083): a warning at each control without a control
  verification, which the Gemara evaluation log marks Needs Review, naming another control's
  verification when that is its only evidence. Validation reports it, so `engdoc`, `engtrace`
  and `model.diagnose` show it without rendering Gemara.
- Trace markers in non-code artifacts (REQ-EMG-084): `# TRLC-LINKS:`, `# ENGMODEL-LINKS:` and
  `# ENGMODEL-OWNER-UNIT:` in `Dockerfile`, `Containerfile`, `*.Dockerfile` and `*.Containerfile`,
  and the same markers in Markdown HTML comments (`<!-- TRLC-LINKS: REQ-… -->`) outside code
  fences. The file implements the linked requirements under its `CODE-<PATH>` identifier; the
  markers are optional and never raise `code.missing_trlc_link`.
- README: building needs cgo (tree-sitter); static-link recipe
  `-tags netgo,osusergo -ldflags '-linkmode external -extldflags -static'`.
- Owner acceptance records (REQ-EMG-085): optional `acceptedBy: {actor, date, statement,
  decisionRef}` on risks and threat scenarios (Go type `AcceptanceRecord`, CUE schema).
  `risk.acceptance_unrecorded` (error at `acceptedBy`) for a risk with `status: accepted` or
  `response: accept`, or a threat scenario with `status: accepted`, that lacks an actor, a
  date, and a statement or decisionRef; `model.invalid_acceptance` (error) for an actor that is
  no actor, a date that is no YYYY-MM-DD calendar date, or a decisionRef that is no decision or
  a decision whose status is not `accepted`; `risk.acceptance_not_owner` (warning) when the
  accepting actor is not the risk's `owner`; `risk.accept_while_open` (warning) for
  `response: accept` with `status: open`. The record is an authored claim: the checks make it
  complete and consistent, and review of the diff that adds it is what shows the owner gave it.
- Threat scenarios, mitigations and control verifications in the engine's own model for
  `go.work` generation from untrusted checkouts, artifact parsing of untrusted repository files
  and false acceptance claims.
- `engws sync` writes `replace <module> <version> => ./<dir>` into the family `go.work` for each
  version of a workspace Go module that any checkout requires, so unpublished family modules
  build (REQ-EMG-086). A `go.mod` module path that fails `module.CheckPath` (for example a quoted
  path with a line break) skips that checkout's Go module, and an invalid require is ignored,
  each with a warning, so neither reaches `go.work`.
- The authoring contract rules (`validate.AuthoringRules`, MCP `model.authoringContract`) add
  `fieldSets` (the YAML fields each entity kind accepts, from the strict-decoding types) and
  `publicationSections` (decisions cannot be published), and field rules for
  `boundaryCrossing`/`trustBoundaryRef`, what `behavior.event_unused` counts, the
  `lint.multiple_responses` field-list exemption, the strict EARS system term and acceptance
  records (REQ-EMG-087).
- Environment-gated tests (REQ-EMG-088): a Go test that skips depending on an environment
  variable or `testing.Short()` is gated: one of its first three statements is an `if` that
  reads `os.Getenv`/`os.LookupEnv` or calls `testing.Short()` and whose body calls `t.Skip`,
  `t.Skipf` or `t.SkipNow`. A gate in a later statement or in a helper is not detected.
  Trace rows whose every linked test is gated carry `gated` and `gates`; the summary adds
  `gatedOnly` and `gatedOnlyRequirements` (still counted in `testLinked`), also in MCP
  `trace.matrix`.

### Fixed

- The MCP server renders every tool call from the model files on disk (REQ-EMG-089). It
  previously kept the model loaded at `initialize`, so a call right after an edit (for example
  `gemara.policy` after changing a risk) returned the old result until the client initialized
  again. It now fingerprints this module's model files (the manifest, its documents, and the
  lock and workspace files next to it) and reloads when they change; a reload that fails is the
  call's error (-32001). Edits in a sibling checkout that the workspace resolves a dependency
  from are not detected: re-send `initialize` after them.

### Changed

- **Breaking:** `risk.acceptance_unrecorded` is a new error. A model with a risk at
  `status: accepted` or `response: accept`, or a threat scenario at `status: accepted`, fails
  validation until it has `acceptedBy` (actor, date, statement or decisionRef) from its owner.
- **Breaking:** the Gemara policy lists a risk under `risks.accepted` only when it has a
  complete, resolving `acceptedBy` record and status `accepted`, or response `accept` with a
  status other than `open`; `CheckGemaraModelConsistency` checks the same rule (REQ-EMG-085).
- **Breaking:** authored risk and threat scenario IDs must not contain `::` (schema error);
  `alias::ID` is reserved for entities imported by composition.
- **Breaking:** existing `TRLC-LINKS` markers in Markdown HTML comments and in Dockerfile `#`
  comments now count as implementation of the linked requirements, and an invalid or dangling
  one is now an error (`code.invalid_trlc_link`, `code.dangling_requirement_link`). The engine's
  own `docs/sysml-v2-superset-plan.md` marker for REQ-EMG-038/039 is removed: a plan document is
  not their implementation.
- Markdown marker scanning closes a fence only on a fence of the same character that is at
  least as long (CommonMark) and ignores lines indented four or more columns; artifact files
  are split on bytes (a line longer than 1 MiB no longer stops the scan) and only regular files
  up to 512 KiB are read.
- `gemara.control_needs_review` also warns at a control whose verifications need review (for
  example `partial`) and none failed, so it warns at exactly the controls the evaluation log
  marks Needs Review; both use one status mapping (REQ-EMG-083).
- `publicationSections` in the authoring contract is derived from the manifest publication
  type's fields.
- The authoring contract defines `triggered_by` (state → event) as an event handled within the
  state; the event that enters a state belongs to `transitions_to` (its description, or a
  `semantics.relationships` transition with `triggers`). `behavior.event_unused` says so.
- The engine's own model lists its threat scenarios' control verifications and links
  `VER-MCP-PATH-BOUNDARY` to REQ-EMG-008.

## [0.6.0] — 2026-10-30: Behavior, catalog and verification diagnostics; engws stage; engspec

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
### Added (multi-repository staging and spec rendering)

- `engws stage [--version-from manifest|tag] [--serve] [--addr]` publishes every checkout
  under the workspace root, in dependency order, into `<root>/.engmod/staging` without
  writing inside any checkout: `registry/<module path>/` holds each CUE module as an OCI
  image layout with the content `cue mod publish` would push (source kind `git`: committed
  files only, with VCS metadata), and `goproxy/` is a GOPROXY file tree
  (`<module>/@v/<version>.{info,mod,zip}`, `list`) for each checkout with `go.mod`. A dirty
  checkout (or, with `--version-from tag`, one whose HEAD lacks tag `v<module.version>`)
  stops staging; version skew between a consumer pin and the staged version is warned
  about. CUE has no file-based registry, so `--serve` loads the layouts into an in-memory
  read-only OCI registry on a loopback address and prints `CUE_REGISTRY` routes for the
  staged modules (merged with the current value), `GOPROXY=file://…/goproxy,off`,
  `GONOPROXY=none`, `GONOSUMDB`, `GOWORK=off` and `ENGMOD_WORK=off`, so the pre-push gate can
  run against the staged family before anything is pushed (REQ-EMG-080).
- `engmodel.RenderSpecDelta`, `RenderSpecDeltaFromModel` and the `engspec` CLI render
  OpenSpec spec delta markdown from model requirements (`--ids` or `--prefix`,
  `--capability`, `--modified`): one `### Requirement:` per requirement with the line
  `<ID>: <model text>` (shall/must as SHALL/MUST, so it passes `openspec validate --strict`
  and starts with the model text) and one scenario from `verificationCriteria`, WHEN/THEN
  shaped when the criteria are (REQ-EMG-081).
- `cuelabs.dev/go/oci/ociregistry` (already an indirect dependency through `cuelang.org/go`)
  is now a direct requirement.

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
