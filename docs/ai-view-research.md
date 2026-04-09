# Executive Summary

Branch verification: current working branch is `master` (no explicit user-specified target branch was provided in this request, so implementation proceeded on `master`).

The repository already had a strong deterministic generation pipeline for human-readable AsciiDoc/PDF + Mermaid outputs. The implemented PoC keeps that behavior intact and adds an additive AI-first export path with three outputs:

- canonical normalized machine artifact: `architecture.ai.json`
- optional derived audit document: `architecture.ai.md`
- optional dense graph stream: `edges.ndjson`

The key design decision implemented here is to avoid a cards-only/prose-heavy dump. Instead, the JSON is normalized and provenance-first, with explicit authored vs inferred semantics, confidence on inferred fields, stable IDs, cross-links, and precomputed requirement support paths.

# Current-State Findings

## 1) Existing generation flow (human-first) and data path

Verified flow in current codebase:

1. Load: `GenerateAsciiDocFromFiles` loads architecture, requirements, design and normalizes relative `CodeRoot` in [`asciidoc.go`](/home/labeth/ws/engineering-model-go/asciidoc.go:23).
2. Validate + lint: `validate.Bundle` + EARS lint run at [`asciidoc.go`](/home/labeth/ws/engineering-model-go/asciidoc.go:44).
3. Project views: each selected view is projected via `Generate(...)` in loop at [`asciidoc.go`](/home/labeth/ws/engineering-model-go/asciidoc.go:57), where `Generate` validates then calls `view.Build` and Mermaid render in [`generate.go`](/home/labeth/ws/engineering-model-go/generate.go:27).
4. Infer layers: runtime/code/verification inference is merged into AsciiDoc assembly at [`asciidoc.go`](/home/labeth/ws/engineering-model-go/asciidoc.go:81).
5. Assemble large template DTO: many sections carried in one template data struct at [`asciidoc_template.go`](/home/labeth/ws/engineering-model-go/asciidoc_template.go:15).
6. Render large narrative template with repeated per-view scoped FG/FU sections in [`templates/asciidoc.tmpl`](/home/labeth/ws/engineering-model-go/templates/asciidoc.tmpl:258).

## 2) AI-consumption pain points in the current human output

### Repetition / duplicated facts

- The template repeats view-scoped functional groups and units with narrative overlays across views ([`templates/asciidoc.tmpl:258`](/home/labeth/ws/engineering-model-go/templates/asciidoc.tmpl:258), [`templates/asciidoc.tmpl:280`](/home/labeth/ws/engineering-model-go/templates/asciidoc.tmpl:280), [`templates/asciidoc.tmpl:344`](/home/labeth/ws/engineering-model-go/templates/asciidoc.tmpl:344)).
- This is strong for reading one report, weak for targeted retrieval due repeated entity facts in many sections.

### Mixed authored vs inferred semantics

- Authored unit assembly and inferred evidence are merged in the same FU section at [`asciidoc.go:109-146`](/home/labeth/ws/engineering-model-go/asciidoc.go:109).
- Inferred runtime/code evidence is flattened to string fields via `buildOwnerEvidence` at [`asciidoc_linking_units.go:333-382`](/home/labeth/ws/engineering-model-go/asciidoc_linking_units.go:333).

### Weak machine navigation surface

- Stable anchors exist in reference index builder ([`asciidoc_design_refs.go:112-172`](/home/labeth/ws/engineering-model-go/asciidoc_design_refs.go:112)), but this is an appendix-style navigation layer, not a compact machine index optimized for direct ID lookup.

### Oversized narrative sections

- The template includes broad narrative + repeated sections + appendix inventories in one artifact ([`templates/asciidoc.tmpl:105-496`](/home/labeth/ws/engineering-model-go/templates/asciidoc.tmpl:105)).

### Flattened evidence strings hide structure

- `runtime: ... | code modules: ...` flattening in [`asciidoc_linking_units.go:371-379`](/home/labeth/ws/engineering-model-go/asciidoc_linking_units.go:371) loses typed relation structure.

### Implicit inference confidence

- Inference has heuristic ownership and kind normalization in runtime inference ([`inferred_layers.go:154-213`](/home/labeth/ws/engineering-model-go/inferred_layers.go:154)), but confidence is not a first-class exported field in AsciiDoc sections.

### Determinism strengths already present and reusable

- View ordering: [`asciidoc_views.go:121-154`](/home/labeth/ws/engineering-model-go/asciidoc_views.go:121)
- Projected node/edge sorting: [`view/project.go:146-168`](/home/labeth/ws/engineering-model-go/view/project.go:146)
- Runtime item sorting: [`inferred_layers.go:87-95`](/home/labeth/ws/engineering-model-go/inferred_layers.go:87)
- Code item sorting: [`inferred_code.go:99-107`](/home/labeth/ws/engineering-model-go/inferred_code.go:99)
- Verification sorting: [`inferred_verification.go:247-252`](/home/labeth/ws/engineering-model-go/inferred_verification.go:247)
- Codemap symbol sorting: [`codemap/scan.go:91-99`](/home/labeth/ws/engineering-model-go/codemap/scan.go:91)

### Authored/inferred boundary exists in validation but not strongly in output

- Validation blocks authored mappings to inferred IDs (`RT-*`, `CODE-*`) at [`validate/validate.go:89-91`](/home/labeth/ws/engineering-model-go/validate/validate.go:89).
- Yet output still merges authored + inferred prose in one chapter flow.

### Provenance limitation to note

- YAML loader decodes directly and does not persist YAML AST node positions ([`model/load.go:64-77`](/home/labeth/ws/engineering-model-go/model/load.go:64)); exact line-level authored provenance needs an extra lookup pass.

# AI View Spec v1

Implemented canonical artifact: `AIViewDocument` in [`ai_view_schema.go:18-26`](/home/labeth/ws/engineering-model-go/ai_view_schema.go:18).

Top-level keys (implemented):

- `schema_version`
- `model`
- `entry_points`
- `entity_index`
- `support_paths`
- `entities`
- `source_blocks`

Implemented at build return in [`ai_view.go:437-445`](/home/labeth/ws/engineering-model-go/ai_view.go:437).

## Entity kinds (minimum required set)

Supported in implementation:

- `functional_group`
- `functional_unit`
- `requirement`
- `runtime_element`
- `code_element`
- `verification`

Kind ranking/order is explicit in [`ai_view.go:618-635`](/home/labeth/ws/engineering-model-go/ai_view.go:618).

## Authored vs inferred separation and confidence

- Entity origin is explicit (`authored`, `inferred`, `verification`) in `AIEntity.Origin` ([`ai_view_schema.go:83`](/home/labeth/ws/engineering-model-go/ai_view_schema.go:83)).
- Inferred field provenance includes `origin`, `confidence`, `source_refs` via `AIFieldProvenance` ([`ai_view_schema.go:103-108`](/home/labeth/ws/engineering-model-go/ai_view_schema.go:103)).
- Runtime/code/verification confidence heuristics implemented in:
  - [`ai_view.go:558-570`](/home/labeth/ws/engineering-model-go/ai_view.go:558)
  - [`ai_view.go:579-591`](/home/labeth/ws/engineering-model-go/ai_view.go:579)
  - [`ai_view.go:593-606`](/home/labeth/ws/engineering-model-go/ai_view.go:593)

## Provenance-first source blocks

- Source blocks modeled by `AISourceBlock` ([`ai_view_schema.go:110-118`](/home/labeth/ws/engineering-model-go/ai_view_schema.go:110)).
- Block creation, dedupe, stable IDs, and deterministic ordering implemented in:
  - [`ai_view.go:825-856`](/home/labeth/ws/engineering-model-go/ai_view.go:825)
  - [`ai_view.go:880-896`](/home/labeth/ws/engineering-model-go/ai_view.go:880)
- Line discovery uses best-effort lookup for IDs/tokens:
  - [`ai_view.go:919-950`](/home/labeth/ws/engineering-model-go/ai_view.go:919)

## Retrieval-oriented support paths

- Requirement support paths precomputed in [`ai_view.go:637-713`](/home/labeth/ws/engineering-model-go/ai_view.go:637).
- Curated entry points emitted in [`ai_view.go:715-791`](/home/labeth/ws/engineering-model-go/ai_view.go:715).

## Dense graph separation

- NDJSON edge stream generated from canonical JSON model in [`ai_view_edges.go:10-111`](/home/labeth/ws/engineering-model-go/ai_view_edges.go:10).
- Keeps dense relationship stream separate from compact machine index.

## Direct lookup examples

Requirement lookup by ID (`REQ-PRR-001`) in `entity_index.lookup`:

```json
{ "id": "REQ-PRR-001", "kind": "requirement", "title": "REQ-PRR-001" }
```

Requirement support path to verification:

```json
{
  "id": "PATH-REQ-PRR-001",
  "from_id": "REQ-PRR-001",
  "path": ["REQ-PRR-001", "FU-...", "RT-...", "CODE-...", "VER-..."],
  "confidence": "high"
}
```

Functional unit evidence pivot:

```json
{
  "id": "FU-GITHUB-WEBHOOK-INGRESS",
  "runtime_ids": ["RT-..."],
  "code_ids": ["CODE-..."],
  "verification_ids": ["VER-..."],
  "field_provenance": [
    { "field": "runtime_ids", "origin": "inferred", "confidence": "medium" }
  ]
}
```

# Implementation Patch Plan

## Phase 1: Additive AI schema and generator path

Implemented:

- AI schema types: [`ai_view_schema.go`](/home/labeth/ws/engineering-model-go/ai_view_schema.go)
- Additive generator entry points:
  - `GenerateAIViewFromFiles` [`ai_view.go:17`](/home/labeth/ws/engineering-model-go/ai_view.go:17)
  - `GenerateAIView` [`ai_view.go:39`](/home/labeth/ws/engineering-model-go/ai_view.go:39)
- Reused existing validate/infer pipeline (same runtime/code/verification inference).

## Phase 2: Retrieval and provenance structures

Implemented:

- deterministic entity index and lookup ([`ai_view.go:401-412`](/home/labeth/ws/engineering-model-go/ai_view.go:401))
- support paths ([`ai_view.go:637-713`](/home/labeth/ws/engineering-model-go/ai_view.go:637))
- curated entry points ([`ai_view.go:715-791`](/home/labeth/ws/engineering-model-go/ai_view.go:715))
- source block dedupe/sort ([`ai_view.go:825-896`](/home/labeth/ws/engineering-model-go/ai_view.go:825))

## Phase 3: Optional AI markdown and NDJSON edges

Implemented:

- Markdown derived view: [`ai_view_markdown.go`](/home/labeth/ws/engineering-model-go/ai_view_markdown.go)
- Dense edges NDJSON: [`ai_view_edges.go`](/home/labeth/ws/engineering-model-go/ai_view_edges.go)

## Phase 4: CLI integration with backward compatibility

Implemented additive flags in [`cmd/engdoc/main.go:34-36`](/home/labeth/ws/engineering-model-go/cmd/engdoc/main.go:34) with existing AsciiDoc behavior preserved ([`cmd/engdoc/main.go:46-70`](/home/labeth/ws/engineering-model-go/cmd/engdoc/main.go:46)).

## Phase 5: Determinism and shape tests

Implemented tests in [`ai_view_test.go`](/home/labeth/ws/engineering-model-go/ai_view_test.go):

- byte-repeatability across two runs ([`ai_view_test.go:11-37`](/home/labeth/ws/engineering-model-go/ai_view_test.go:11))
- schema/shape + ordering + NDJSON parse checks ([`ai_view_test.go:39-132`](/home/labeth/ws/engineering-model-go/ai_view_test.go:39))

Also fixed a determinism edge case in verification mapping tie-break logic ([`inferred_verification.go:172-175`](/home/labeth/ws/engineering-model-go/inferred_verification.go:172)).

# Validation Results

## 1) Build/tests

Executed:

```bash
go test ./...
```

Result: pass.

## 2) Artifact generation for bedrock sample

Executed:

```bash
go run ./cmd/engdoc \
  --model examples/bedrock-pr-review-github-app-sample/architecture.yml \
  --requirements examples/bedrock-pr-review-github-app-sample/requirements.yml \
  --design examples/bedrock-pr-review-github-app-sample/design.yml \
  --code-root /home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/src \
  --ai-json-out examples/bedrock-pr-review-github-app-sample/generated/ARCHITECTURE.ai.json \
  --ai-md-out examples/bedrock-pr-review-github-app-sample/generated/ARCHITECTURE.ai.md \
  --ai-edges-out examples/bedrock-pr-review-github-app-sample/generated/ARCHITECTURE.edges.ndjson
```

Result: generated successfully (lint warnings preserved, no fatal errors).

## 3) 10 representative AI questions (before vs after)

Legend: `Yes` = directly answerable with stable path, `Partial` = answerable but requires broad prose scan or ambiguous inference.

1. Which FU owns requirement `REQ-PRR-001`?
- Before (AsciiDoc): Partial
- After (`architecture.ai.json`): Yes (`REQ` -> `related_ids` + support path)

2. What runtime evidence supports `FU-GITHUB-WEBHOOK-INGRESS`?
- Before: Partial (`Evidence` prose flattening)
- After: Yes (`runtime_ids` + `field_provenance` + `source_refs`)

3. Which verification checks verify `REQ-PRR-004`?
- Before: Partial (appendix scan)
- After: Yes (`requirement.verification_ids` and `support_paths`)

4. Distinguish authored vs inferred facts for a FU.
- Before: Partial (mixed prose)
- After: Yes (`origin`, typed relation IDs, provenance)

5. Show shortest requirement->verification support path.
- Before: Partial/manual reconstruction
- After: Yes (`support_paths.path`)

6. Which inferred links are low confidence?
- Before: No explicit confidence field
- After: Yes (`entry_points` `EP-LOW-CONFIDENCE-INFERRED`)

7. Which code elements impact `REQ-PRR-006`?
- Before: Partial (traceability narrative/appendix)
- After: Yes (`REQ` -> FU -> `code_ids`, plus NDJSON edges)

8. Which runtime component impacts a specific requirement?
- Before: Partial
- After: Yes (`support_paths`, `runtime_ids`)

9. Which verification checks are failing or partial?
- Before: Partial appendix traversal
- After: Yes (`EP-VERIFICATION-FAILURES`, verification `status`)

10. What cited source lines support this inferred relation?
- Before: Mostly coarse source path strings
- After: Yes (`source_blocks` with path and line + `source_refs`)

## 4) Token footprint estimate method and results

Method used: rough token estimate `tokens ~= characters / 4` (explicit approximation for model-agnostic comparison).

Bedrock sample generated sizes:

- `ARCHITECTURE.adoc`: 129,832 chars (~32,458 tokens)
- `ARCHITECTURE.ai.json`: 119,002 chars (~29,750 tokens)
- `ARCHITECTURE.ai.md`: 46,323 chars (~11,581 tokens)
- `ARCHITECTURE.edges.ndjson`: 109,875 chars (~27,469 tokens)

Interpretation:

- For direct retrieval and targeted QA, `architecture.ai.json` alone is smaller and more query-efficient than the current monolithic human AsciiDoc.
- `edges.ndjson` is intentionally dense and should be loaded selectively for graph indexing/RAG, not always in-context with full JSON.

# Open Risks / Next Iteration

## Risk register

1. Authored source-line provenance is best-effort, not AST-precise.
- Evidence: loader does not retain YAML node positions ([`model/load.go:64-77`](/home/labeth/ws/engineering-model-go/model/load.go:64)).
- Next: optional YAML-node location index pass.

2. Support path selection currently picks representative first-hop chains.
- Evidence: path assembly picks first FU/runtime/code/verification in sorted sets ([`ai_view.go:663-675`](/home/labeth/ws/engineering-model-go/ai_view.go:663)).
- Next: add configurable multi-path expansion + path scoring.

3. Runtime inference coarsening remains heuristic for some terraform resources.
- Evidence: fallback `terraform_resource` in normalization ([`inferred_layers.go:172`](/home/labeth/ws/engineering-model-go/inferred_layers.go:172)).
- Next: richer resource taxonomy and provider-specific resolvers.

4. `architecture.ai.json` is compact relative to AsciiDoc but still sizable on larger repos.
- Next: optional profile mode with shallow cards + lazy source block/edge loading.

5. Verification linkage is inferred by requirement token overlap and artifact matching.
- Evidence: overlap-based attach in [`inferred_verification.go:156-176`](/home/labeth/ws/engineering-model-go/inferred_verification.go:156).
- Next: explicit check IDs in test metadata to reduce ambiguity.

## Next iteration priorities

1. Add strict JSON schema validation test against emitted AI JSON.
2. Add optional `--ai-profile compact|full` to control token footprint.
3. Extend `support_paths` to include multiple ranked paths per requirement.
4. Add stable per-entity deep-link anchors in AsciiDoc that mirror AI IDs.

