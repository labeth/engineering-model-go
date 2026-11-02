# Requirement links across composed models (v0.8.0)

Plan and decisions for child-to-parent requirement links. The task description
is the source (`req-impl-task.md`, from the simulator-family retro); this file
records what the engine implements and where it deviates. Requirement IDs:
REQ-EMG-090 onward in `model/requirements.yml`.

## Decisions

| # | Decision | Reason |
|---|---|---|
| D1 | Strictness is a manifest setting, `requirementLinks: strict \| guided` (default `guided`). In `strict`, `requirement.no_parent` and the `link.*` rules are errors in every model, child or system. In `guided`, they are warnings, and only in a model that publishes contract items or has `reqLinks`; a standalone model without either gets none. `engflow init` writes `strict`. | The owner wants links required and enforced as errors; the default must not break every existing model on upgrade. |
| D2 | A system computes a child item's fingerprint itself, from the child's published content at the pinned version. Every requirement that realizes a published contract item must be in the same publication (`contract.realizer_unpublished`, error). No generated fingerprint file. | Nothing to keep fresh; the fingerprint is always that of the content the system actually resolved. |
| D3 | Pins may name a tag or a commit. A dependency `version` is a release tag (`v1.2.3`) or a Go-style pseudo-version that carries the commit (`v1.2.4-0.20261102103400-abcdef123456`). A link's `pinned.version` must equal the dependency pin. A release always uses tags: `engws release-check` (run by the pre-push gate for a tag push and by CI on a tag) fails with `dependency.unreleased_pin` when any dependency is a pseudo-version or another pre-release. | The owner's rule: point at a tag or hash while working, release only on tags. |
| D4 | New commands are `engws` subcommands: `engws bump` (extended), `engws review-links`, `engws contract-diff`, `engws migrate-links` and `engws release-check`. | There is no `engmod` CLI; engws already owns dependency pins. |
| D5 | `composition.allocations` is deprecated: still read for one release, reported as `composition.allocation_deprecated` (warning), converted by `engws migrate-links`. `composition.satisfactions` stays: it wires a published need to a provider (architecture), not a requirement trace. | Keep architecture wiring separate from requirement links. |
| D6 | `contract.provides[].ref` is deprecated the same way (`contract.ref_deprecated`, warning); `migrate-links` moves each `ref` to `realizes` on the referenced requirement. | Direction stays child-to-parent inside the repository too. |
| D7 | Contract items are `contract.provides` entries of `kind: capability` and the new `contract.assumptions` (`AOU-*`). `kind: interface` entries are not link targets. | Links are about behaviour and conditions, not wiring. |

## Model

- `Requirement.realizes: [CAP-… | AOU-…]` (local contract items only).
- `contract.assumptions: [{id, text, verificationCriteria?}]`.
- Document key `reqLinks` (`model/req-links.yml`): `reqLinks: [{id, parent?, child, evidence?, pinned: {version, fingerprint}, status: claimed|accepted, rationale?, reviewedBy?, reviewedOn?}]`. A link to an `AOU-*` has `evidence` (local requirement, test or control IDs) instead of, or in addition to, `parent`.
- Fingerprint: `sha256:` over the canonical JSON of `{id, kind, text|note, verificationCriteria}` of the item, followed by the realizing requirements sorted by ID, each `{id, text, verificationCriteria, status}`. Strings have runs of whitespace collapsed to one space and are trimmed; case is kept. A golden test pins the algorithm.

## Rules

| Code | Severity (strict / guided) | Rule |
|---|---|---|
| `requirement.no_parent` | error / warning | No `realizes`, not `derived` with `derivedRationale`, no stakeholder `sourceRefs`, and not exempt as top level (`module.kind: system` with `inferenceHints.topLevel: true`). In a system, a requirement that is the `parent` of a link also counts as traced downward but still needs its own upward trace. |
| `requirement.realizes_unknown` | error | A `realizes` target is not a local capability or assumption of use. |
| `contract.unrealized` | warning | A published capability no requirement realizes. |
| `contract.realizer_unpublished` | error | A requirement realizes a published contract item but is not in that publication. |
| `link.parent_unknown`, `link.child_unpublished` | error | |
| `link.version_mismatch` | error | |
| `link.suspect` | error | The message shows the before and after text of the item and its realizing requirements. |
| `link.broken` | error | |
| `link.claimed_only` | warning (error in `ef-ready`) | |
| `link.aou_unverified` | error / warning | A dependency's assumption of use without a link carrying `evidence`. |
| `requirement.unsatisfied` | error / warning | A system requirement with no code, test, link or `derived`. |
| `dependency.unreleased_pin` | error, in `engws release-check` only | |

## Tools

- `engws bump`: after pinning, prints each link as unchanged, suspect or broken; exits non-zero when any is suspect or broken; never re-stamps.
- `engws review-links [--accept LNK-… --rationale … --by ACT-…]`: shows suspect links with diffs; `--accept` writes the new fingerprint, `reviewedBy`, `reviewedOn`, the rationale and `status: accepted`.
- `engws contract-diff <from> <to>`: added, removed or changed contract items with the semver impact. `engws release-check` fails when the tag's bump is smaller than the diff requires.
- `engws migrate-links`: allocations → `model/req-links.yml` (`status: claimed`, current fingerprints); `contract.provides[].ref` → `realizes`.
- MCP: `links.list`, `links.suspect`, `links.review`, `requirements.parents`; `requirements.get` shows parents and children.
- Trace: rows get `parents` and `children`; summary counts `suspect`, `broken`, `claimed`, `unparented`.
- Renderers: the system view shows the requirement tree through links; a child rendered alone shows no users; a child rendered in a workspace with a using system shows "used by", computed at render time.

## Out of scope

The "Later" items of the task: anonymous criticality signals, link files in their own module.
