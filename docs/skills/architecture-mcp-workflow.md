# Architecture MCP Development Workflow

This workflow is for implementation agents working in engineering-model repositories after the generated machine-view artifacts were removed. The MCP server is the machine context surface; AsciiDoc/PDF outputs are publication artifacts.

## Required Inputs

1. Model files: `architecture.yml`, `requirements.yml`, `design.yml`, and `catalog.yml`.
2. MCP tool responses for model context, implementation lookup, policy context, and verification status.
3. Source tree, tests, and generated publication/export artifacts relevant to the task.

## Workflow

1. Identify the target stable IDs.
   - Start from the user task, affected `REQ-*`, `FU-*`, `IF-*`, `FLOW-*`, `DO-*`, `CTRL-*`, or `TS-*` IDs.
   - Use MCP lookup tools to resolve implementations, related model entities, verification evidence, and next actions.

2. Resolve context before editing.
   - For requirements, identify owning functional units and verification evidence.
   - For APIs/interfaces/data/flows, identify concrete code declarations and tests through model links.
   - For compliance work, inspect `compliance.profiles[]` and `compliance.mappings[]`.

3. Make minimal traceable edits.
   - Prefer files already linked to the target model entities.
   - Add or update `ENGMODEL-OWNER-UNIT` and `TRLC-LINKS` markers where ownership or requirement behavior changes.
   - Use `ENGMODEL-LINKS` for concrete model entities implemented by declarations.

4. Update verification.
   - Add or update tests with `TRLC-LINKS: REQ-*` markers.
   - Keep test result artifacts parseable when status evidence is authored.

5. Regenerate maintained artifacts.
   - AsciiDoc and decisions: `engdoc`.
   - PDF: `proven-docs render`.
   - Exchange artifacts as applicable: `engstruct`, `engdragon`, `engtrlc`, `englobster`, `engoscal`.
   - Do not regenerate or reference removed machine-view artifacts.

6. Run validation.
   - Run `go test ./...`.
   - Run focused export validation scripts when touching those exporters.
   - Search for stale generated references after removing or renaming model IDs.

## Tagging Rules

File ownership marker:

```go
// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
```

Requirement trace marker:

```go
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
```

Concrete model links:

```go
// ENGMODEL-LINKS: IF-CLI-ENGMCP, DO-MCP-TOOL-RESULT, EVT-MCP-TOOL-CALL-RECEIVED
```

Rules:

- Link to specific model IDs, not generic catalog concepts.
- Do not add inferred `RT-*` or `CODE-*` IDs to authored architecture mappings.
- Generated, vendor, and dependency-cache files should not require new ownership markers.

## Done Criteria

1. The changed model or requirement IDs are named in the final summary.
2. Code and tests have owner/trace/model links where applicable.
3. MCP lookup would resolve the implementation or verification context for changed APIs, flows, data objects, controls, and requirements.
4. Maintained generated artifacts are refreshed when their inputs changed.
5. `go test ./...` passes.
