# AsciiDoc Architecture Generator Design

## Purpose

Generate a deterministic architecture document (AsciiDoc/PDF) from:

- authored architecture (`architecture.yml`)
- requirements (`requirements.yml`)
- view-scoped design narratives (`design.yml`)
- catalog terms (`catalog.yml`, referenced by `architecture.yml`)
- inferred runtime/code evidence from IaC and source trees

## Current Model

The generator is view-centric and layered:

- Authored layer: Functional Groups + Functional Units
- Inferred layer: Runtime + Code evidence
- Traceability layer: requirements, references, inferred indexes

## Inputs

### `architecture.yml`

- model metadata and introduction
- authored architecture entities and mappings
- inference hints (runtime/code roots and ownership resolution order)
- views (kinds and roots)

### `requirements.yml`

- requirements used for alignment and coverage generation

### `design.yml`

- per-Functional Group and per-Functional Unit narratives for each view kind:
  - `functional`
  - `runtime`
  - `deployment`
  - `code_ownership`
  - `security`

## Generation Pipeline

1. Load/validate authored architecture, requirements, design, and catalog.
2. Infer runtime/code evidence from configured roots.
3. Build selected views and Mermaid blocks.
4. Build view-scoped FG/FU sections from design narratives.
5. Compute view guidance and quality signals:
   - What This View Answers
   - Coverage Summary
   - Coverage Gaps
   - Recommended Next Evidence Additions
6. Build `Document Health Snapshot` across all views.
7. Build requirement alignment + cross-layer coverage.
8. Build reference index (authored, catalog, inferred runtime, inferred code).
9. Render AsciiDoc template deterministically.

## Output Structure

The generated document includes:

- Introduction
- Scope and Assumptions
- How To Read This Document
- Document Health Snapshot
- Terms and Definitions
- View chapters (Functional/Runtime/Deployment/Realization/Security)
- Generated Evidence Appendix
  - Requirement Alignment
  - Cross-Layer Coverage
  - Reference Index

## Output Contract

- deterministic ordering
- stable IDs/anchors for linkability
- reproducible artifacts for the same input set

## CLI

`cmd/engdoc`:

```bash
engdoc --model architecture.yml --requirements requirements.yml --design design.yml [--view VIEW-ID ...] [--out architecture.adoc]
```

With source evidence inference:

```bash
engdoc --model architecture.yml --requirements requirements.yml --design design.yml --code-root ./src --out architecture.adoc
```
