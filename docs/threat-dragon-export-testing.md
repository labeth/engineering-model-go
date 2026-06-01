# Threat Dragon Export Testing

This repo includes local JSON schemas and scripts to validate Threat Dragon-compatible exports.

## Files

- `tools/threat-dragon-schemas/threat-dragon-v2.schema.json`
- `tools/threat-dragon-schemas/open-threat-model.schema.json`
- `scripts/fetch-threat-dragon-schemas.sh`
- `scripts/validate-threat-dragon.sh`

## Refresh Schemas

```bash
scripts/fetch-threat-dragon-schemas.sh
```

## Validate Export JSON

Threat Dragon v2 model:

```bash
go run ./cmd/engdragon --model examples/payments-engineering-sample/architecture.yml --format threat-dragon-v2 --out examples/payments-engineering-sample/generated/threat-dragon-v2.json
scripts/validate-threat-dragon.sh td-v2 examples/payments-engineering-sample/generated/threat-dragon-v2.json
```

Open Threat Model format:

```bash
go run ./cmd/engdragon --model examples/payments-engineering-sample/architecture.yml --format open-otm --out examples/payments-engineering-sample/generated/open-threat-model.json
scripts/validate-threat-dragon.sh open-otm examples/payments-engineering-sample/generated/open-threat-model.json
```

## Quick Output Inspection

Use `jq` to sanity check generated files:

```bash
jq '.version, .summary.title, (.detail.diagrams | length)' examples/payments-engineering-sample/generated/threat-dragon-v2.json
jq '.otmVersion, .project.name, (.components | length), (.dataflows | length)' examples/payments-engineering-sample/generated/open-threat-model.json
```
