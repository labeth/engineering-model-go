# Generic Agent Prompt: Architecture-Aware Development

Use this for any agent framework.

## Prompt Template

You are working in a model-driven architecture repository. Use the MCP server as the primary machine context surface and treat generated AsciiDoc/PDF files as publication artifacts.

Workflow and tagging contract:

- `docs/skills/architecture-mcp-workflow.md`
- `docs/code-linking-best-practices.md`

Requirements:

1. Plan and execute changes by stable IDs: `REQ-*`, `FU-*`, `IF-*`, `FLOW-*`, `DO-*`, `CTRL-*`, `TS-*`, `VER-*`.
2. Resolve context through MCP lookup tools before editing; use `trace.matrix` for per-requirement implemented/verified/delegated/orphan status and dangling links, and `composition.resolve` for the system-of-systems and requirement delegation. A requirement is implemented in its own code or delegated to a subsystem (no requirement tiers).
3. Use repository tagging markers in code/tests:
   - `ENGMODEL-OWNER-UNIT: FU-*`
   - `TRLC-LINKS: REQ-*`
   - `ENGMODEL-LINKS: <concrete model IDs>`
4. Keep authored architecture separate from inferred runtime/code evidence.
5. Regenerate maintained artifacts when inputs change: AsciiDoc, PDFs, Structurizr, Threat Dragon/Open OTM, TRLC, LOBSTER, and OSCAL as applicable.
6. Do not generate or rely on removed machine-view artifacts.
7. Every `TRLC-LINKS`/`ENGMODEL-LINKS` id must resolve to a real model id (unresolved links fail the build). Run `bash scripts/validate-all.sh` (build, engdoc 0 errors, engtrace 0 dangling, artifact-freshness drift) — the same gauntlet CI runs — and report any remaining warnings.

Response structure to enforce:

- Target stable IDs
- MCP context used
- Files changed
- Tags added/updated
- Verification updates
- Generated artifacts refreshed
- Test results
