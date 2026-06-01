---
name: engineering-model-architecture-ai
description: Architecture-aware workflow for engineering-model repositories using MCP context, stable ID impact mapping, and required owner/trace tagging.
---

# Architecture MCP Development Skill

This skill defines the expected workflow for AI agents developing in this repository using MCP context and maintained generated publication/export artifacts.

## Purpose

Use MCP tool responses as the machine contract for implementation planning, impact analysis, and verification. AsciiDoc/PDF outputs remain publication outputs.

## Required Inputs

1. Architecture model files (`architecture.yml`, `requirements.yml`, `design.yml`)
2. MCP tool responses for implementation, verification, policy, and generation context
3. Source tree and tests

## Required Workflow

1. Select target stable IDs.
- Start from affected `REQ-*`, `FU-*`, `IF-*`, `FLOW-*`, `DO-*`, `CTRL-*`, or `TS-*` IDs.
- Use MCP lookup tools to resolve implementation, verification, policy, and generation context.

2. Resolve the support chain before editing.
- For a requirement `REQ-*`, use MCP context to identify owning FUs, runtime/code evidence, and verification checks.

3. Make minimal, traceable edits.
- Prefer edits in modules already linked by `code_ids`/`runtime_ids`.
- If new code is added, tag ownership and requirement trace markers (see Tagging Rules).

4. Update/extend verification with requirement traces.
- Add or update tests that include `REQ-*` tokens.
- Ensure test-result artifacts can be parsed for status where applicable.

5. Regenerate maintained artifacts.
- Re-run `engdoc` for AsciiDoc/decisions and `proven-docs` for PDFs when publication inputs change.
- Re-run export generators such as `engstruct`, `engdragon`, `engtrlc`, `englobster`, and `engoscal` when those inputs change.

6. Run validation gates.
- Run `go test ./...`.
- Confirm deterministic outputs are unchanged across repeated generation when no new edits occur.

7. Summarize by stable IDs.
- Report changes as affected `REQ/FU/RT/CODE/VER` IDs and verification or coverage changes.

## Tagging Rules (Code + Tests)

### Owner tag (file-level)

Add exactly one owner marker in each owned source file where possible:

- `ENGMODEL-OWNER-UNIT: FU-...`

Examples:

```go
// ENGMODEL-OWNER-UNIT: FU-GITHUB-WEBHOOK-INGRESS
```

```ts
// ENGMODEL-OWNER-UNIT: FU-PR-CONTEXT-ASSEMBLY
```

```rust
// ENGMODEL-OWNER-UNIT: FU-REVIEW-ORCHESTRATION
```

### Requirement trace tags (near behavior)

Use this required marker near declarations/logic:

- `TRLC-LINKS: REQ-...`

Examples:

```go
// TRLC-LINKS: REQ-PRR-001, REQ-PRR-008
```

```ts
// TRLC-LINKS: REQ-PRR-003
```

```rust
// TRLC-LINKS: REQ-PRR-006
```

### Optional symbolic trace tags

When useful for stable symbol-level mapping:

- `TRACE-ID: ...`
- `TRACE-PART-OF: FU-...`

### Test trace requirements

- Test code must include `TRLC-LINKS: REQ-*` markers to be linked as verification checks.
- If result artifacts are produced, ensure statuses map cleanly (`pass`, `fail`, `partial`, `blocked`, `not-run`, `flaky`).

## What Not To Do

- Do not generate or rely on removed machine-view artifacts.
- Do not encode inferred IDs (`RT-*`, `CODE-*`) inside authored architecture mappings.
- Do not leave new files without owner tags when ownership is clear.
- Do not add broad prose-only updates without stable ID links.

## Regeneration Command Template

```bash
go run ./cmd/engdoc \
  --model <example>/architecture.yml \
  --requirements <example>/requirements.yml \
  --design <example>/design.yml \
  --code-root <absolute path to example>/src \
  --out <example>/generated/ARCHITECTURE.adoc \
  --decisions-out <example>/generated/DECISIONS.adoc
```

## Done Criteria (Agent Must Check)

1. Every changed requirement has explicit MCP-resolvable implementation and verification context.
2. New/changed code has owner and requirement trace tags where applicable.
3. Verification links exist for changed requirement scope.
4. `go test ./...` passes.
5. Maintained generated artifacts are refreshed when their inputs changed.
6. Final report references stable IDs and source refs.
