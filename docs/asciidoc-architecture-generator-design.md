# AsciiDoc Architecture Generator Design

## Purpose

Generate a deterministic architecture document (AsciiDoc/PDF) from:

- manifest and module metadata (`engmod.yml`)
- authored structure and composition (`model/architecture.yml`)
- authored behavior (`model/behavior.yml`)
- assurance and compliance (`model/assurance.yml`, `model/compliance.yml`)
- requirements (`model/requirements.yml`)
- view-scoped design narratives (`model/views.yml`)
- catalog terms (`model/catalog.yml`)
- inferred runtime/code evidence from IaC and source trees

The generated AsciiDoc/PDF is the maintained publication surface. Machine-oriented development context is exposed through the MCP server, not through generated machine-view files.

## Current Model

The generator is view-centric and layered:

- Authored layer: Functional Groups + Functional Units
- Inferred layer: Runtime + Code evidence
- Traceability layer: requirements, verification (test code -> requirement), references, inferred indexes

Verification ownership semantics:
- primary: verification check to test code elements
- primary: verification check to requirements
- derived context: functional owners inferred from requirement ownership

## Inputs

### `engmod.yml`

- module metadata, explicit document paths, dependencies, and publications
- inference hints (runtime/code roots and ownership resolution order)

### `model/architecture.yml`

- functional structure, interfaces, data, deployment, hardware, contracts, and composition

### `model/behavior.yml`

- states, events, flows, and typed relationships

### `model/views.yml`

- views (kinds and roots)
  - optional view publication metadata:
    - `authoredStatus`
    - `authoredStatusExplanation`

### `model/requirements.yml`

- requirements used for alignment and coverage generation

- per-Functional Group and per-Functional Unit narratives for each view kind:
  - `architecture_intent`
  - `communication`
  - `deployment`
  - `security`
  - `traceability`
  - `state_lifecycle` (optional)

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
- View chapters (Architecture Intent/Communication/Deployment/Security/Traceability)
- Traceability Appendix
  - Requirement Details
  - Verification Inventory (test code elements + requirement mapping)
  - Verification Result Mapping
- Reference Index

## Output Contract

- deterministic ordering
- stable IDs/anchors for linkability
- reproducible artifacts for the same input set

## CLI

`cmd/engdoc`:

```bash
engdoc --model engmod.yml --requirements model/requirements.yml --design model/views.yml [--view VIEW-ID ...] [--out architecture.adoc] [--decisions-out decisions.adoc]
```

With source evidence inference:

```bash
engdoc --model engmod.yml --requirements model/requirements.yml --design model/views.yml --code-root ./src --out architecture.adoc --decisions-out decisions.adoc
```

Render PDF with `proven-docs`:

```bash
proven-docs render generated/ARCHITECTURE.adoc --output generated/ARCHITECTURE.proven.pdf
```
