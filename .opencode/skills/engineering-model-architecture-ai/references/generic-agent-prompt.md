# Generic Agent Prompt: Architecture-Aware Development

Use this for any agent framework (Codex, Claude Code, Cursor agent, custom orchestrators).

## Prompt Template

You are working in a model-driven architecture repository with AI-first artifacts.

Primary machine input:
- `generated/ARCHITECTURE.ai.json`
Optional:
- `generated/ARCHITECTURE.edges.ndjson`
- `generated/ARCHITECTURE.ai.md`

Workflow and tagging contract:
- `docs/skills/architecture-ai-workflow.md`

Requirements:

1. Plan and execute changes by stable IDs (`REQ`, `FU`, `RT`, `CODE`, `VER`).
2. For each requirement change, provide a support chain using `support_paths`.
3. Use repository tagging markers in code/tests:
- `ENGMODEL-OWNER-UNIT: FU-*`
- `TRLC-LINKS: REQ-*` (required)
4. Keep authored architecture separate from inferred evidence.
5. Regenerate AI artifacts and run validation tests.
6. Output a final impact map by stable IDs with source references.

Response structure to enforce:

- Target requirements
- Support paths used
- Files changed
- Tags added/updated
- Verification updates
- AI artifact deltas (coverage/confidence)
- Test results
