# Plan 001: C4 Mermaid Go Library

## Goal
Build a reusable Go library that converts a C4-style architecture model into Mermaid diagram text for embedding in AsciiDoc/Markdown.

The library will be importable by a larger Go program and deterministic in output.

## Scope
In scope:
- Parse and validate C4 model input (YAML -> typed Go model).
- Build view projections from model data.
- Render Mermaid text for core views.
- Keep output deterministic (stable node/edge ordering).
- Provide machine-readable diagnostics for invalid model input.

Out of scope (initially):
- Full documentation website generation.
- Runtime system discovery.
- Editing tools/UI.
- Non-Mermaid renderers.

## Reference Inputs
Reference documents moved into this repo:
- `examples/payments-engineering-sample/catalog.yml`
- `examples/payments-engineering-sample/requirements.yml`
- `examples/payments-engineering-sample/architecture.yml`

These define the example domain vocabulary, EARS-base requirements, and C4 extension structure.

## Proposed Package Layout
- `model`:
  - core types (systems, people, containers, components, relations, viewpoints)
  - YAML decode helpers
- `validate`:
  - structural checks
  - ID existence checks
  - allowed relation checks
  - diagnostics
- `view`:
  - context projection
  - container projection
  - verification projection
- `render/mermaid`:
  - Mermaid node/edge generation
  - style/class helpers
  - stable deterministic ordering
- `examples`:
  - reference model docs
  - expected Mermaid outputs for snapshots
- `cmd/engview` (optional after library core is stable):
  - simple CLI wrapper around the library

## Input/Output Contracts
Input:
- C4 model YAML (plus optional external requirement/catalog refs by path)
- selected viewpoint ID

Output:
- Mermaid text
- diagnostics list

No side effects in core library.

## Milestones
1. Repository bootstrap
- initialize module
- define public interfaces
- add minimal README and development notes

2. Model types + YAML decoding
- implement typed model structs
- decode reference `03-architecture-model.yml`
- add strict unknown-field handling in decoder

3. Validator
- ensure IDs are unique
- ensure relation endpoints exist
- ensure viewpoint roots exist
- emit deterministic diagnostics

4. View projection engine
- `system-context`
- `container`
- `verification`

5. Mermaid renderer
- map projected view nodes/edges to Mermaid flowchart syntax
- deterministic ordering
- stable labels and IDs

6. Snapshot tests
- golden output files for each view
- verify stable output against references

7. Optional CLI
- `engview render --model <file> --view <id>`
- output Mermaid to stdout or file

## Acceptance Criteria
- Library can load example model docs and render all defined viewpoints.
- Generated Mermaid is deterministic across runs.
- Validation catches broken IDs and bad relations with clear diagnostics.
- Public API is stable enough for integration into the larger Go program.

## Immediate Next Steps
1. Add `go.mod` for `c4-mermaid-go`.
2. Define `model` package structs based on `03-architecture-model.yml`.
3. Implement first renderer path for `VIEW-CONTEXT`.
4. Add golden-file test for `VIEW-CONTEXT` output.
