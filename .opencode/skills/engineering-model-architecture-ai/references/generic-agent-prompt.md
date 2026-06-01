# Generic Agent Prompt: Architecture-Aware Development

Use this for any agent framework (Codex, Claude Code, Cursor agent, custom orchestrators).

## Prompt Template

You are working in a model-driven architecture repository with MCP-backed machine context.

Primary machine input:
- MCP tool responses for model, implementation, policy, verification, and generation context

Workflow and tagging contract:
- `docs/skills/architecture-mcp-workflow.md`

Requirements:

1. Plan and execute changes by stable IDs (`REQ`, `FU`, `RT`, `CODE`, `VER`).
2. For each requirement change, provide an MCP-resolved implementation and verification chain.
3. Use repository tagging markers in code/tests:
- `ENGMODEL-OWNER-UNIT: FU-*`
- `TRLC-LINKS: REQ-*` (required)
4. Keep authored architecture separate from inferred evidence.
5. Regenerate maintained artifacts and run validation tests.
6. Output a final impact map by stable IDs with source references.

Response structure to enforce:

- Target requirements
- MCP context used
- Files changed
- Tags added/updated
- Verification updates
- Generated artifacts refreshed
- Test results
