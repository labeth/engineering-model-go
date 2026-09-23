# Release validation: source tracing and hardware publications

Validation date: 2026-09-23. Base: v0.2.0 (5833375).

The release adds JavaScript and lexical Verilog source tracing, individual-file
inference roots, distinct source identities, hardware view/MCP support, and
publication fixes. Test-only links remain verification evidence rather than
production implementation.

Publications retain one complete diagram per authored view and requirement.
Large graphs scale to fit rather than being split across repeated panels.
Standalone SVG and Mermaid exports retain complete graphs.

## Checks

- Full Go tests and go vet: passed.
- Formatting and whitespace: passed.
- All 15 example models and the repository model regenerated.
- All 16 architecture PDFs rendered without Mermaid parse-error warnings.
- All 1,050 PDF pages checked for text outside page bounds and renderer error text:
  none found. Representative diagram, table, opening and closing pages were
  visually reviewed. Dense graphs intentionally use smaller labels.
- All 28 standalone SVGs rendered and visually reviewed; text stays within their
  view boxes.
- SysML metamodel, official parser, all-example syntax, KPAR reopen and native
  source round trip: passed.
- Artifact freshness: passed; regeneration produced no changes.
- Full scripts/validate-all.sh gate, including external generated-format
  conformance checks: passed.

The example models retain their authored gaps and not-run evidence states.
Successful export validation does not change those states into passed tests.

## Architecture PDFs

| Publication | Pages |
| --- | ---: |
| examples/atlas-industries/generated/aegis-sentinel/ARCHITECTURE.proven.pdf | 41 |
| examples/atlas-industries/generated/company-portfolio/ARCHITECTURE.proven.pdf | 36 |
| examples/atlas-industries/generated/orion-relay/ARCHITECTURE.proven.pdf | 40 |
| examples/atlas-industries/generated/shared-cloud-platform/ARCHITECTURE.proven.pdf | 29 |
| examples/atlas-industries/generated/shared-compliance-baseline/ARCHITECTURE.proven.pdf | 38 |
| examples/atlas-industries/generated/shared-edge-platform/ARCHITECTURE.proven.pdf | 29 |
| examples/atlas-industries/generated/shared-security-services/ARCHITECTURE.proven.pdf | 30 |
| examples/bedrock-pr-review-github-app-sample/generated/ARCHITECTURE.proven.pdf | 111 |
| examples/coffee-appliance-six-view/generated/ARCHITECTURE.proven.pdf | 36 |
| examples/coffee-fleet-ota-cloud-sample/generated/ARCHITECTURE.proven.pdf | 129 |
| examples/coffee-fleet-ota-cloud-sample/subsystems/cloud-api/generated/ARCHITECTURE.proven.pdf | 29 |
| examples/coffee-fleet-ota-cloud-sample/subsystems/ota-agent/generated/ARCHITECTURE.proven.pdf | 31 |
| examples/coffee-fleet-ota-cloud-sample/subsystems/telemetry/generated/ARCHITECTURE.proven.pdf | 31 |
| examples/dal-c-flight-control-sample/generated/ARCHITECTURE.proven.pdf | 32 |
| examples/payments-engineering-sample/generated/ARCHITECTURE.proven.pdf | 122 |
| generated/ARCHITECTURE.proven.pdf | 286 |
