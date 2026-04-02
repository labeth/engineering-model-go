# AsciiDoc Architecture Generator Design

## Purpose

Generate an architecture document in AsciiDoc from:

- architecture model (`architecture.yml`)
- requirements (`requirements.yml`)
- design mapping (`design.yml`)
- shared catalog (`catalog.yml`, referenced by `architecture.yml`)
- optional code tree with trace markers (`TRACE-ID`, `TRACE-PART-OF`, `TRACE-IMPLEMENTS`)
- optional code tree with trace markers (minimal: `TRACE-REQS`)

The generator must keep catalog IDs as the semantic anchor and automatically derive:

- requirement links
- chapter-local direct-dependency scope in the architecture model

## Scope

In scope:

- generate deterministic AsciiDoc output
- embed Mermaid view blocks for selected viewpoints
- connect design chapters to catalog IDs
- derive chapter-local architecture connections from relationship `catalogRefs`
- auto-link requirements by matching requirement text against catalog term name and aliases
- map traced code symbols to architecture elements via `TRACE-PART-OF`
- if `TRACE-PART-OF` is omitted, infer container mapping from traced requirements

Out of scope:

- free-text NLP or semantic inference beyond deterministic phrase matching
- repo/workflow management
- runtime-state/operations modeling

## Inputs

### `architecture.yml`

- architecture model (`model`, `c4`, `relationships`, `viewpoints`)
- relationships may contain `catalogRefs`

### `requirements.yml`

- requirement list (`id`, `text`, optional notes)

### `design.yml`

```yaml
design:
  id: string
  title: string
  views: [VIEW-ID, ...]   # optional default view list
  chapters:
    - id: string
      title: string
      narrative: string
      catalogRefs: [CATALOG-ID, ...]   # required
```

## Validation Rules

### Existing model validation

- c4 IDs are unique
- relationship endpoints resolve
- view kinds and roots are valid
- c4 people and systems map to catalog actors/systems
- relationship `catalogRefs` resolve to catalog IDs

### Design-document validation

- each chapter has unique `id`
- each chapter has one or more `catalogRefs`
- each `catalogRef` resolves to catalog

## Generation Pipeline

1. Load `architecture.yml`, `requirements.yml`, `design.yml`.
2. Validate model/catalog mapping and design mapping.
3. Resolve target views:
   - CLI `--view` list if provided
   - else `design.views`
   - else all views from `architecture.yml`
4. Generate Mermaid for each selected view.
5. For each design chapter:
   - resolve chapter catalog terms from `catalogRefs`
   - derive direct architecture scope:
     - select relationships where `relationship.catalogRefs` intersects chapter `catalogRefs`
     - derive involved architecture IDs from relationship endpoints (`from`, `to`)
   - auto-link requirements:
     - normalize text (lowercase, collapse punctuation to spaces)
     - match catalog canonical names + aliases as exact normalized phrase containment
   - generate chapter scope diagram:
     - Mermaid graph with only direct derived relationships (no transitive expansion)
6. Render final AsciiDoc sections:
   - introduction
   - full views (`[source,mermaid]` blocks)
   - design chapters with:
     - narrative
     - chapter scope diagram
     - derived reference map
   - requirements appendix with anchors for cross-reference links
7. Optional code mapping:
   - scan source tree for trace markers
   - parse declarations with Tree-sitter (Go, TypeScript/TSX, Rust)
   - generate symbol IDs from declaration names when `TRACE-ID` is omitted
   - map traced symbols to architecture elements:
     - explicit: `TRACE-PART-OF`
     - inferred: traced requirements -> chapter derived container scope
   - render `Code Mapping` chapter grouped by architecture element

## Output Contract

- deterministic section ordering
- stable requirement and relationship ordering (sorted by IDs)
- machine-generated and reproducible from same inputs

## CLI

`cmd/engdoc`:

```bash
engdoc --model architecture.yml --requirements requirements.yml --design design.yml [--view VIEW-ID ...] [--out architecture.adoc]
```

With code mapping:

```bash
engdoc --model architecture.yml --requirements requirements.yml --design design.yml --code-root ./src --out architecture.adoc
```
