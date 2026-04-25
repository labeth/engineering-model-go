# TRLC Requirements Export

This repo can generate TRLC requirement files from `requirements.yml`.

## Generate

```bash
go run ./cmd/engtrlc \
  --requirements examples/payments-engineering-sample/requirements.yml \
  --out-dir examples/payments-engineering-sample/generated/trlc \
  --package PaymentsRequirements
```

Generated files:

- `model.rsl`
- `requirements.trlc`

## Validate with TRLC

Install TRLC once:

```bash
python3 -m pip install --user trlc
```

Validate a generated package:

```bash
scripts/validate-trlc.sh examples/payments-engineering-sample/generated/trlc
```

Validate all examples:

```bash
scripts/validate-trlc.sh examples/payments-engineering-sample/generated/trlc
scripts/validate-trlc.sh examples/bedrock-pr-review-github-app-sample/generated/trlc
scripts/validate-trlc.sh examples/coffee-fleet-ota-cloud-sample/generated/trlc
```
