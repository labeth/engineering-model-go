# Structurizr DSL Export Testing

This repo includes a Structurizr DSL exporter and validation workflow.

## Generate DSL

```bash
go run ./cmd/engstruct --model examples/payments-engineering-sample/architecture.yml --out examples/payments-engineering-sample/generated/STRUCTURIZR.dsl
```

## Validate DSL

```bash
scripts/validate-structurizr.sh examples/payments-engineering-sample/generated/STRUCTURIZR.dsl
```

The validator uses the official `docker.io/structurizr/structurizr` container and accepts either Podman or Docker.

## Validate all example DSL outputs

```bash
scripts/validate-structurizr.sh examples/payments-engineering-sample/generated/STRUCTURIZR.dsl
scripts/validate-structurizr.sh examples/bedrock-pr-review-github-app-sample/generated/STRUCTURIZR.dsl
scripts/validate-structurizr.sh examples/coffee-fleet-ota-cloud-sample/generated/STRUCTURIZR.dsl
```
