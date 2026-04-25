---
name: engineering-model-architecture-ai
description: Architecture-aware workflow for engineering-model repositories using ARCHITECTURE.ai.json support paths, stable ID impact mapping, and required owner/trace tagging.
---

# Architecture AI Development Skill (Framework-Neutral)

This skill defines the expected workflow for AI agents developing in this repository using the AI-first architecture artifacts.

## Purpose

Use the AI artifacts as the machine contract for implementation planning, impact analysis, and verification:

- canonical: `generated/ARCHITECTURE.ai.json`
- optional dense graph: `generated/ARCHITECTURE.edges.ndjson`
- optional audit view: `generated/ARCHITECTURE.ai.md`

Human AsciiDoc/PDF outputs remain publication outputs, not the primary machine navigation surface.

## Required Inputs

1. Architecture model files (`architecture.yml`, `requirements.yml`, `design.yml`)
2. Current generated AI artifacts for the example/system under change
3. Source tree and tests

## Required Workflow

1. Select target from AI entry points.
- Start from `entry_points` in `architecture.ai.json`.
- Pick requirement-driven (`EP-REQ-*`) or gap-driven (`EP-LOW-CONFIDENCE-INFERRED`, `EP-VERIFICATION-FAILURES`) work.

2. Resolve the support chain before editing.
- For a requirement `REQ-*`, read `support_paths` to identify:
  - owning FU (`FU-*`)
  - runtime evidence (`RT-*`)
  - code evidence (`CODE-*`)
  - verification checks (`VER-*`)

3. Make minimal, traceable edits.
- Prefer edits in modules already linked by `code_ids`/`runtime_ids`.
- If new code is added, tag ownership and requirement trace markers (see Tagging Rules).

4. Update/extend verification with requirement traces.
- Add or update tests that include `REQ-*` tokens.
- Ensure test-result artifacts can be parsed for status where applicable.

5. Regenerate AI artifacts.
- Re-run `engdoc` AI export for the target example/system.
- Re-check `support_paths`, `entry_points`, and confidence changes.

6. Run validation gates.
- Run `go test ./...`.
- Confirm deterministic outputs are unchanged across repeated generation when no new edits occur.

7. Summarize by stable IDs.
- Report changes as affected `REQ/FU/RT/CODE/VER` IDs and confidence deltas.

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

- Do not treat `architecture.ai.md` as source of truth.
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
  --ai-json-out <example>/generated/ARCHITECTURE.ai.json \
  --ai-md-out <example>/generated/ARCHITECTURE.ai.md \
  --ai-edges-out <example>/generated/ARCHITECTURE.edges.ndjson
```

## Done Criteria (Agent Must Check)

1. Every changed requirement has an explicit `REQ-*` support path.
2. New/changed code has owner and requirement trace tags where applicable.
3. Verification links exist for changed requirement scope.
4. `go test ./...` passes.
5. AI artifacts regenerate without schema/order regressions.
6. Final report references stable IDs and source refs.
