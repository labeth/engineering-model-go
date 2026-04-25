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
scripts/validate-threat-dragon.sh td-v2 out/threat-dragon.json
```

Open Threat Model format:

```bash
scripts/validate-threat-dragon.sh open-otm out/open-threat-model.json
```

## Quick Output Inspection

Use `jq` to sanity check generated files:

```bash
jq '.version, .summary.title, (.detail.diagrams | length)' out/threat-dragon.json
jq '.otmVersion, .project.name, (.components | length), (.dataflows | length)' out/open-threat-model.json
```
