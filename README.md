# engineering-model-go

`engineering-model-go` is a Go library and CLI for generating architecture views and AsciiDoc documentation from a typed engineering model.

It combines:
- architecture model loading and validation
- viewpoint projection (context/container/deployment)
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

Example:

```go
res, err := engmodel.GenerateFromFile("examples/payments-engineering-sample/architecture.yml", "VIEW-CONTEXT")
if err != nil {
    panic(err)
}

fmt.Println(res.Mermaid)
for _, d := range res.Diagnostics {
    fmt.Printf("%s [%s] %s\n", d.Code, d.Severity, d.Message)
}
```

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
  --code-root examples/payments-engineering-sample/src \
  --out examples/payments-engineering-sample/generated/ARCHITECTURE.adoc
```

## Example Project

End-to-end sample inputs and generated outputs are under:
- `examples/payments-engineering-sample`

Core files:
- `catalog.yml`
- `architecture.yml`
- `requirements.yml`
- `design.yml`
- `infra/terraform`
- `infra/flux`
- `infra/helm`
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
