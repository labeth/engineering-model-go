# engineering-model-go

`engineering-model-go` is a Go library and CLI for generating architecture views and AsciiDoc documentation from a typed engineering model.

It combines:
- architecture model loading and validation
- viewpoint projection (architecture-intent/communication/deployment/security/traceability)
- Mermaid rendering
- design + requirement narrative generation to AsciiDoc
- EARS requirement preflight linting
- code trace mapping (Go, TypeScript, Rust)

## Scope

This project models development-state architecture inputs from YAML and produces deterministic documentation/view artifacts.

It is not a runtime observability or incident/compliance runtime system.

## Features

- strict YAML loading for architecture/catalog/requirements/design documents
- model validation for IDs, references, relations, and viewpoint configuration
- catalog-linked architecture relation labeling
- deployment view extraction from:
  - Terraform (`.tf`) using HashiCorp HCL parsing
  - Flux resources (GitRepository/Kustomization/HelmRelease)
  - Helm chart metadata via Helm SDK chart loader
- deterministic Mermaid view rendering
- AsciiDoc architecture generation with chapter scope diagrams
- EARS preflight linting via `github.com/labeth/ears-lint-go`
  - `lintRun.mode: strict` is the default and recommended project mode
  - `guided` is optional for drafting workflows and non-blocking author guidance
- Tree-sitter based code symbol extraction and trace mapping for:
  - Go
  - TypeScript/TSX
  - Rust

## Installation

```bash
go get github.com/labeth/engineering-model-go
```

## Library API

Primary entry points:
- `GenerateFromFile(architecturePath, viewID)`
- `Generate(bundle, viewID)`
- `GenerateAsciiDocFromFiles(architecturePath, requirementsPath, designPath, options)`
- `GenerateAIViewFromFiles(architecturePath, requirementsPath, designPath, options)`

Example:

```go
res, err := engmodel.GenerateFromFile("examples/payments-engineering-sample/architecture.yml", "VIEW-ARCHITECTURE-INTENT")
if err != nil {
    panic(err)
}

fmt.Println(res.Mermaid)
for _, d := range res.Diagnostics {
    fmt.Printf("%s [%s] %s\n", d.Code, d.Severity, d.Message)
}
```

View IDs are free-form, but view `kind` must be one of:
- `architecture-intent`
- `communication`
- `deployment`
- `security`
- `traceability`
- `state-lifecycle` (optional)

Optional per-view publication metadata (in `architecture.yml`):
- `authoredStatus` (for example `draft`, `in-review`, `stable`)
- `authoredStatusExplanation` (short rationale shown in Document Health Snapshot)

Verification metadata is inferred from test artifacts (not authored in `architecture.yml`):
- test sources under `tests/` (for inferred verification checks and test code element links)
- result artifacts under `test-results/` (for inferred requirement-level outcomes)
- requirement IDs are inferred by matching `REQ-*` tokens in test and result artifacts
- published verification chain is `Verification Check -> Test Code Element -> Requirement`
- functional ownership in verification tables is shown as derived context from requirement ownership
- strict EARS lint also warns when catalog terms (systems, actors, events, states, features, modes, conditions, data terms) are not referenced by any requirement text (`catalog.term_unreferenced`)

## CLI Usage

Generate a single Mermaid view:

```bash
go run ./cmd/engview \
  --model examples/payments-engineering-sample/architecture.yml \
  --view VIEW-DEPLOYMENT \
  --out out.mmd
```

Generate architecture AsciiDoc:

```bash
go run ./cmd/engdoc \
  --model examples/payments-engineering-sample/architecture.yml \
  --requirements examples/payments-engineering-sample/requirements.yml \
  --design examples/payments-engineering-sample/design.yml \
  --code-root ./src \
  --out examples/payments-engineering-sample/generated/ARCHITECTURE.adoc
```

Generate AI-first artifacts (normalized JSON, derived markdown, dense edge stream):

```bash
go run ./cmd/engdoc \
  --model examples/bedrock-pr-review-github-app-sample/architecture.yml \
  --requirements examples/bedrock-pr-review-github-app-sample/requirements.yml \
  --design examples/bedrock-pr-review-github-app-sample/design.yml \
  --code-root /abs/path/to/examples/bedrock-pr-review-github-app-sample/src \
  --ai-json-out examples/bedrock-pr-review-github-app-sample/generated/ARCHITECTURE.ai.json \
  --ai-md-out examples/bedrock-pr-review-github-app-sample/generated/ARCHITECTURE.ai.md \
  --ai-edges-out examples/bedrock-pr-review-github-app-sample/generated/ARCHITECTURE.edges.ndjson
```

Render PDF with proven-docs:

```bash
proven-docs render \
  examples/payments-engineering-sample/generated/ARCHITECTURE.adoc \
  --output examples/payments-engineering-sample/generated/ARCHITECTURE.proven.pdf
```

The generated document starts with:
- `Introduction` (from `model.introduction`)
- `Scope and Assumptions`
- `How To Read This Document`
- `Document Health Snapshot` (per-view coverage and confidence)
- `Terms and Definitions` as a compact `Term | Definition` table
  - terms are sorted A-Z by term text
  - term IDs are shown as `Term (ID)` in the term column

Each generated view includes:
- `What This View Answers`
- `Coverage Gaps` and `Recommended Next Evidence Additions`
- view-scoped narratives from `design.yml`
  - for `security`, content is organized as attack-vector chapters with per-attack diagrams and related FU sections

The Traceability View includes:
- `Requirement-to-Unit Mapping (Compact)`
- `Verification Result Mapping` (inferred per-requirement outcomes)

The traceability appendix includes:
- `Requirement Details`
- `Verification Inventory` (inferred test/check mappings to test code elements and requirements)
- `Reference Index` with consistent entry layout:
  - chapter per ID
  - short prose description
  - compact key/value table with the most relevant fields
  - `Mentioned In` backlinks to jump from registry entries to sections where each entry is referenced
  - sections: `Authored References`, `Catalog References`, `Inferred Runtime References`, `Inferred Code References`, `Verification References`

AI-first export includes:
- `architecture.ai.json` style normalized machine artifact (`--ai-json-out`)
- optional derived audit markdown (`--ai-md-out`)
- optional dense edge stream for graph indexing (`--ai-edges-out`)

## Agent Skills

Use these docs to drive AI-agent development workflows and tagging conventions:

- OpenCode-ready skill (auto-discoverable by `opencode debug skill`): `.opencode/skills/engineering-model-architecture-ai/SKILL.md`
- framework-neutral workflow and tagging contract: `docs/skills/architecture-ai-workflow.md`
- OpenCode adapter prompt: `docs/skills/adapters/opencode-skill-prompt.md`
- generic adapter prompt for other frameworks: `docs/skills/adapters/generic-agent-prompt.md`

## Example Project

End-to-end sample inputs and generated outputs are under:
- `examples/payments-engineering-sample`
- `examples/bedrock-pr-review-github-app-sample`
- `examples/coffee-fleet-ota-cloud-sample`

Core files:
- `catalog.yml`
- `architecture.yml`
- `requirements.yml`
- `design.yml`
- `infra/terraform`
- `src` (Go/Rust/TypeScript traced code)

## Development

Run all checks locally:

```bash
go test ./...
go vet ./...
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

This project is licensed under the MIT License.
See [LICENSE](./LICENSE).
