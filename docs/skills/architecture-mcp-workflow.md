# Architecture MCP Development Workflow

This is the AI-first workflow for agents that model, engineer, and implement against an
engineering-model repository. The `engmcp` server is the machine context surface; the
generated AsciiDoc/PDF and exchange artifacts (Gemara, OSCAL, Structurizr, Threat Dragon,
TRLC, LOBSTER, `TRACE-MATRIX.json`) are publication outputs. Trace links are
**existence-checked and enforced** — a broken link or a stale artifact fails the build —
so the agent's job is to keep the model, the code, and the trace links consistent.

## Required inputs

1. Model files: `architecture.yml`, `requirements.yml`, `design.yml`, `catalog.yml`, `decisions.yml`.
2. MCP tool responses for model context, implementation lookup, trace status, composition, and verification.
3. The source tree, tests, and the generated artifacts under each model's `generated/`.

## MCP tools to lean on

Discover and inspect the model:
- `model.list` / `entities.list`, `model.entity`, `model.implementations` — enumerate and inspect any entity (functional units, interfaces, data objects, flows, controls, threats, risks, hardware items, hardware interfaces).
- `requirements.get`, `requirements.impact`, `requirements.supportPath`, `requirements.suggestEditPlan` — requirement context and edit planning.
- `files.forRequirement` / `files.forControl` / `files.forThreat` / `files.owner` — find the code linked to a model id.
- `code.contextForTask` — assemble compact code + model context for a task.

Understand the system-of-systems and what is (un)implemented:
- `composition.resolve` — the federated system-of-systems: subsystems and their provides/requires contracts, the parent→subsystem requirement allocations (with the specific delegated subsystem requirement), and composition diagnostics.
- `trace.matrix` — per requirement, the status (`implemented` / `verified` / `delegated` / `orphan`), the code references, the delegated subsystem requirement, the orphan list, and any **dangling code trace links**. Call this before editing to find work, and after editing to confirm nothing is orphaned or dangling.

Verify, plan, and self-check:
- `verification.status`, `verification.gaps`, `verification.recommend`, `tests.forRequirement`.
- `tasks.entryPoints`, `tasks.nextBestActions`, `generation.status`, `governance.checkPatch`.

## Workflow

1. **Frame the task in model ids.** Start from the user task and the affected `REQ-*`, `FU-*`, `IF-*`, `FLOW-*`, `DO-*`, `CTRL-*`, `TS-*` ids. Use `requirements.impact` and `model.entity` to scope the blast radius.

2. **Establish the trace baseline.** Run `trace.matrix` to see which requirements are implemented, verified, delegated, or orphan, and whether any code trace links are dangling. For a composed system, run `composition.resolve` to see how requirements are delegated to subsystems.

3. **Model (author the model when behavior is new).** Add or change requirements (EARS-linted), functional units, interfaces, data objects, controls, hardware items/interfaces, or composition (subsystems, allocations, satisfactions) directly in the YAML. A requirement is either **implemented here** (its code) or **delegated** to a subsystem via `composition.allocations` to a published contract entry — there are no requirement tiers.

4. **Engineer (compose and delegate).** For multi-repo or multi-team work, model subsystems under `composition.subsystems` (local `ref:` or external `git:`), publish each subsystem's `contract.provides`/`requires`, and allocate parent requirements to a subsystem's contract entry whose `ref` names the realizing subsystem requirement. Verify with `composition.resolve` that every allocation resolves and every required interface is satisfied.

5. **Implement (minimal, traceable edits).** Prefer files already linked to the target entities. On each function/method that realizes behavior, add the markers (below). Every trace-required function MUST carry a `TRLC-LINKS` marker, and every linked id MUST exist — unresolved links are build errors.

6. **Verify.** Add or update tests carrying `TRLC-LINKS: REQ-*`; keep test-result artifacts parseable. Re-run `trace.matrix` and confirm the touched requirements moved to `implemented`/`verified` and there are zero dangling links.

7. **Regenerate and validate.** Regenerate the maintained artifacts (`engdoc` for AsciiDoc + decisions, `engtrace` for `TRACE-MATRIX.json`, `proven-docs render` for PDF, and the relevant exporters), then run the gauntlet `bash scripts/validate-all.sh`. Generation is deterministic and CI fails on drift, so regenerate and commit the artifacts you changed.

## Tagging rules

File ownership marker (one per source file that a functional unit owns):

```go
// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
```

Requirement trace marker (required on every trace-required function/method; ids must exist):

```go
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
```

Concrete model-element links (must resolve to real model ids):

```go
// ENGMODEL-LINKS: IF-CLI-ENGMCP, DO-MCP-TOOL-RESULT, EVT-MCP-TOOL-CALL-RECEIVED
```

Rules:
- Link to specific model ids, not generic catalog concepts.
- A `TRLC-LINKS` to a non-existent requirement is `code.dangling_requirement_link` (error); an `ENGMODEL-LINKS` to a non-existent element is `code.dangling_model_link` (error); a trace-required function with no `TRLC-LINKS` is `code.missing_trlc_link` (error). All fail engdoc and engtrace.
- Code is attributed to the model whose `architecture.yml` is its nearest enclosing root, so a parent model never claims a subsystem's code.
- Do not add inferred `RT-*` or `CODE-*` ids to authored mappings. Generated, vendor, and dependency-cache (dot-directory) files do not take ownership markers.

## Done criteria

1. The changed model or requirement ids are named in the final summary.
2. `trace.matrix` shows the touched requirements as implemented/verified/delegated with **0 orphan among them and 0 dangling links**.
3. Code and tests carry owner/trace/model links, and MCP lookup resolves the implementation/verification context for the changed entities.
4. For composed systems, `composition.resolve` reports all allocations resolved and all required interfaces satisfied.
5. Maintained generated artifacts are regenerated and committed; `bash scripts/validate-all.sh` passes (build, engdoc 0 errors, engtrace 0 dangling, artifact-freshness drift) — the same gauntlet CI runs.
