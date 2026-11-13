# Removing backwards compatibility: audit and plan

Proposed decisions: ADR-EMG-061 (no backwards compatibility, every required field is an error that names its remedy),
ADR-EMG-060 (no legacy state), ADR-EMG-062 (both repositories run at operational), ADR-EMG-063 (this plan), ADR-EMG-066
(the engine part of phase 2) and ADR-EMG-067 (the tier-up of this repository). This plan comes from two read-only
audits of `engineering-model-go` (engine) and `engineering-flow` (flow) on 2026-11-12 and orders the removal.

Status 2026-11-13: the engine part of phase 2 and the tier-up of `engineering-model-go` (ADR-EMG-062, 067) are
implemented and independently reviewed. `engmod.yml` declares lifecycle tier `operational` (`DEC-TIER-OPERATIONAL`, from
the owner's instruction "same for model, i want it at the highest level"), its code health debt is in
`model/health-baseline.json` (`DEC-HEALTH-BASELINE`), and the requirements that have no recorded RED check are deferred by
the open POA&M items `POAM-TIER-OPERATIONAL-01` to `-16` (due 2027-06-30). **Both decisions are `proposed`, not accepted:
the owner has not reviewed them, and an agent must not accept the decision that opens its own gate.** The gate of this
repository therefore stays red until the owner accepts them. Flow phases 1 and 2, the tier-up of `engineering-flow` and
phases 3 to 7 are open.

## Rule

1. A model, a schema version, an engine pin, a tier decision, an installed file and a lock must be at the current version.
   An older or missing one is an error that says what to upgrade, never a mode that still works.
2. Everything the system needs in order to function must be filled in. If it is absent it is an **error**, and the
   message carries the **remedy**: the skill to run (`ef-tier-up`, `ef-behavior`, `ef-formal-behavior`, `ef-threat-model`,
   `ct-ai-guidance`, `engineering-flow`) or the command to execute (`engflow upgrade`, `engws bump`, `engws link add`).
3. The CI gate prints every such error with its remedy in the job summary and fails, so an agent that skipped the skill is
   told what to run. The same codes run in the pre-push hook. A test keeps an inventory of required fields and remedies.
4. The engine stays usable as a library by tools that embed it: `LoadBundle` stays lenient so tools can report; the
   validate step and the CLIs error. This is the only place where an "absent is accepted" path survives, and it ends
   with phases 6 and 7 (ADR-EMG-066 item 3).

## What the audits found

**Engine (about 40 items):** `lifecycle` is optional and absent in all 20 manifests (18 examples, the Quint test model,
the repository's own); `requirementLinks`, `bundles`, `tierDecision`, `lintRun.mode` and the `health` block silently
default; the golden `no-lifecycle-golden.json` and about 20 tests pinned the no-lifecycle behaviour (phase 2 replaced the golden by `examples-golden.json` at each example's declared tier); deprecated
`composition.allocations`, `contract.provides[].ref` and `pinned.fingerprint` are still read; `migrate-links`,
`--allow-suspect`, `engoscal --out`, MCP `links.suspect`, `entities.list` and many MCP argument aliases exist only for
old callers; `Summary.Verified`, `LinksSuspect`, `BaseCatalogRef`, `Compatibility` and `InferenceHints.CodeSources` are
dead; the `model.State` alias, `legacy_state.go` and `StripLegacyStateVocabulary` are tombstones of v0.11.0; a lock file
version is never checked; `engws check` falls back to the in-process library for a stale check template.

**Flow (30 items):** `legacy` tier state (warning `tier.decision_missing`, `Tier: legacy` accepted in proposals,
`tierRulesEnabled` toggle, mvp fallback) spread over Go, Node scripts, the PR template, the OpenSpec config and about ten
skills; the `engflowVersion` / `available` gating of 71 policy rows; warnings that guard the floor
(`ci.ai_path_unrecorded`, `ci.registry_unapproved`, `ci.codeowners_missing`, `spec.scenario_drift`,
`openspec.substage_missing`); silent skips when `.github/engflow.json`, the installed tiers.json, the skills or the
OpenSpec config are absent; older-engine branches in the installer; migration guides in the README. **There is no
upgrade command, and nothing compares the project's installed version with the check tool's.**

## Order of work

Each phase is one reviewable change set with its own tests; the tier-up effort (ADR-EMG-062) runs after phase 2.

| Phase | Engine | Flow |
|---|---|---|
| **1. Upgrade foundation** | `lock` schema check (`composition.lock_schema_outdated`), `engine.pin_outdated`, `engine.check_template_stale` replace the silent fallbacks | `engflow upgrade` (rewrites the managed set, checks the seed files, stamps `.github/engflow.json`); `engflow.version_mismatch`, `engine.pin_too_old`, `tier.policy_missing`, `ci.engflow_config_missing`, `skills.missing`, `openspec.config_missing`, `repo.gitignore_incomplete`; every remedy then points at a real command |
| **2. Required switches** | `lifecycle.missing` (rank 0, not deferrable) with `tierDecision`, explicit `bundles`, `requirementLinks`, `lintRun.mode`, `schemaVersion` required; `health.block_missing` at product and operational (the baseline stays optional, ADR-EMG-066); add lifecycle and a tier decision to the 18 examples, the Quint test model and the shared fixtures; replace the no-lifecycle golden by a golden at a stated tier | `tier.decision_missing` and `tier.lifecycle_missing` become errors; delete `legacy` (Go, scripts, PR template, OpenSpec config, skills, docs); delete `available`, `engflowVersion`, `Not yet enforced:` and `TIERS.md`'s column |
| **3. Alias and dead code** | MCP argument aliases and duplicate tools, `Summary.Verified`, `LinksSuspect`, `BaseCatalogRef`, `Compatibility`, `CodeSources`, `--allow-suspect`, `engoscal --out` | old skill-directory fallbacks, `previousHookScripts`, `.engflow/approved-usage.md`, the sync-spec fallback |
| **4. Deprecated features** | delete `composition.allocations` and delegation code, `contract.provides[].ref` (migrate the two Atlas examples), `pinned.fingerprint`, `engws migrate-links` | PR-policy legacy rules and `--require-agree` |
| **5. Warnings to errors** | `dependency_tier_unknown`, aviation profile, `catalog.term_unreferenced` where strict | the floor rules (`ci.*`, `spec.scenario_drift`, `openspec.substage_missing`), `ci.language_job_missing`, a local `ci.ai_path_record_missing` |
| **6. Tombstones** | `model.State`, `legacy_state.go`, `StripLegacyStateVocabulary` | migration text in the README |
| **7. Docs and contract** | rewrite README sections, REQ and ADR notes, bump `AuthoringContractVersion` | one "upgrade" section replaces the migration guides |

Phase 6 waits for the first tag after v0.11.0, because `release-check` still reads the v0.10.0 tag with the stripper.

## Proposed decisions and the OWNER items

- `engflow upgrade` is the single upgrade command; `stig project upgrade` stays the route for STIG projects.
- Phase 1 comes before phase 2: a remedy that names a command that does not exist is a lie. The engine part of phase 2
  ran first, so its remedies name only the manual fix (`set schemaVersion: 2`) and the skills; `engflow upgrade` is named
  only after phase 1 of the flow part ships.
- A fresh `engflow init` seeds a tier decision as **proposed**; the first run fails until the owner accepts it (OWNER).
- The remedy for `health`, `lintRun` and `requirementLinks` problems is `ef-tier-up` unless the owner names another skill (OWNER).
- The lock mode default `write` may become `check` (OWNER); `ImportSysMLV2Project` for legacy payloads is a separate decision (OWNER).
- CI keeps one advisory path: `ci.policy_bootstrap`, the chicken-and-egg of the very first PR.

## Blast radius

About 109 test files build `engmod.yml` fixtures and 25 mention lifecycle, so the shared fixtures in
`internal/testutil/fixtures.go` get a lifecycle first. About 40 flow tests use `tierProject(t, root, "", "")` or
`tierRulesEnabled=false` and are rewritten to build an accepted tier. Generated artifacts of all examples change with the
golden. Edit skills only in `prompts/` and run `node init/refresh.mjs`; the STIG golden is regenerated.
