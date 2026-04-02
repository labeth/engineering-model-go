# Sample Catalog + EARS + Architecture Project

This sample shows one small domain model where:

- `catalog.yml` is the shared controlled vocabulary.
- `requirements.yml` uses EARS-style requirements written against catalog terms.
- `architecture.yml` defines the architecture model whose actor/system IDs are mapped to catalog IDs.
- `design.yml` maps catalog terms to design descriptions and architecture references.
- Architecture relationships include `catalogRefs` to anchor connections to catalog terms.
- `infra/terraform` defines an EKS target cluster + namespaces.
- `infra/flux` defines Flux GitOps resources and HelmRelease manifests.
- `infra/helm` contains one Helm chart per service (Go, Rust, TypeScript).
- `src/` contains three language-specific subsystems:
  - Go (`checkout_api.go`)
  - Rust (`payment_engine.rs`)
  - TypeScript (`risk_scorer.ts`)

The implementation files are intentionally dummy but believable:
- they include trace markers for requirement linkage
- they log/print expected actions in each subsystem
- they are meant for documentation and traceability, not production execution

System story:
- checkout starts payment sessions and customer messaging
- payment engine orchestrates authorization, review escalation, and fallback
- risk subsystem computes risk scores and persists audit traces
- support is looped in for decline/review workflows

## Files

- `catalog.yml`
- `requirements.yml`
- `architecture.yml`
- `design.yml`
- `infra/terraform/main.tf`
- `infra/flux/...`
- `infra/helm/...`

## Generate View Output

From `~/ws/engineering-model-go`:

```bash
go run ./cmd/engview --model ./examples/payments-engineering-sample/architecture.yml --view VIEW-CONTEXT --out ./examples/payments-engineering-sample/generated/VIEW-CONTEXT.mmd
go run ./cmd/engview --model ./examples/payments-engineering-sample/architecture.yml --view VIEW-CONTAINER --out ./examples/payments-engineering-sample/generated/VIEW-CONTAINER.mmd
go run ./cmd/engview --model ./examples/payments-engineering-sample/architecture.yml --view VIEW-DEPLOYMENT --out ./examples/payments-engineering-sample/generated/VIEW-DEPLOYMENT.mmd
```

Optional render to SVG:

```bash
npx -y @mermaid-js/mermaid-cli@11.4.2 -i ./examples/payments-engineering-sample/generated/VIEW-CONTEXT.mmd -o ./examples/payments-engineering-sample/generated/VIEW-CONTEXT.svg
npx -y @mermaid-js/mermaid-cli@11.4.2 -i ./examples/payments-engineering-sample/generated/VIEW-CONTAINER.mmd -o ./examples/payments-engineering-sample/generated/VIEW-CONTAINER.svg
npx -y @mermaid-js/mermaid-cli@11.4.2 -i ./examples/payments-engineering-sample/generated/VIEW-DEPLOYMENT.mmd -o ./examples/payments-engineering-sample/generated/VIEW-DEPLOYMENT.svg
```

Generate full architecture AsciiDoc:

```bash
go run ./cmd/engdoc \
  --model ./examples/payments-engineering-sample/architecture.yml \
  --requirements ./examples/payments-engineering-sample/requirements.yml \
  --design ./examples/payments-engineering-sample/design.yml \
  --out ./examples/payments-engineering-sample/generated/ARCHITECTURE.adoc
```

## Upstream References

These are known working upstream projects/docs for the same stack:

- Terraform AWS EKS module: https://github.com/terraform-aws-modules/terraform-aws-eks
- Flux bootstrap and GitOps toolkit docs: https://fluxcd.io/flux/
- Flux HelmRelease API docs: https://fluxcd.io/flux/components/helm/helmreleases/
- Helm chart docs: https://helm.sh/docs/topics/charts/

Generate AsciiDoc with code-to-architecture mapping:

```bash
go run ./cmd/engdoc \
  --model ./examples/payments-engineering-sample/architecture.yml \
  --requirements ./examples/payments-engineering-sample/requirements.yml \
  --design ./examples/payments-engineering-sample/design.yml \
  --code-root ./examples/payments-engineering-sample/src \
  --out ./examples/payments-engineering-sample/generated/ARCHITECTURE.adoc
```
