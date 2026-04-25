# TRLC + LOBSTER Traceability

This repository can generate TRLC requirements and LOBSTER traceability reports.

## Prerequisites

Install tooling:

```bash
python3 -m pip install --user trlc bmw-lobster-core bmw-lobster-tool-trlc
```

Ensure `~/.local/bin` is on your `PATH`.

## One-command generation

Generate a full LOBSTER report (requirements + test activities + HTML report):

```bash
scripts/generate-lobster-report.sh examples/payments-engineering-sample PaymentsRequirements
```

Repeat for other examples:

```bash
scripts/generate-lobster-report.sh examples/bedrock-pr-review-github-app-sample BedrockPRReviewRequirements
scripts/generate-lobster-report.sh examples/coffee-fleet-ota-cloud-sample CoffeeFleetRequirements
```

Outputs are placed under:

- `<example>/generated/lobster/requirements.lobster`
- `<example>/generated/lobster/activities.lobster`
- `<example>/generated/lobster/lobster.conf`
- `<example>/generated/lobster/report.lobster`
- `<example>/generated/lobster/report.html`

## Notes

- Test-to-requirement links are extracted from `TRLC-LINKS: REQ-*` markers.
- If some requirements have no linked tests, `lobster-ci-report` will flag coverage gaps; this is a traceability result, not a tooling failure.
