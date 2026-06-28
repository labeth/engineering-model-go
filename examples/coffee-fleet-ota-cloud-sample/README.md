# Coffee Fleet OTA + Cloud Operations Sample

This sample models a fleet of connected coffee machines that report telemetry to a central cloud site,
receive OTA firmware updates, and emit operational/audit logging evidence.

The authored architecture captures stable capability boundaries. Runtime, code, and verification layers are
inferred from Terraform resources, tagged source modules, and test/result artifacts.

Highlights:

- Device telemetry is collected at the machine edge and ingested by cloud APIs.
- OTA update campaigns are orchestrated centrally and executed by device update agents.
- Logging and audit pipelines track brew telemetry, update rollout, and rollback outcomes.
- Platform units own runtime operations and identity/secrets boundaries.

Inference fixtures included:

- runtime: `infra/terraform`
- code ownership and requirement traces: `src/`
- verification checks and outcomes: `tests/` + `test-results/`

## System-of-systems composition

Although the highlights above describe the fleet as a single system, this sample is actually authored as a
parent system that composes three downward subsystems. The parent `architecture.yml` declares them under
`composition.subsystems`, each referencing a local subdirectory:

- `subsystems/telemetry` (`SUB-TELEMETRY`) — collects and reports machine telemetry.
- `subsystems/ota-agent` (`SUB-OTA-AGENT`) — verifies and applies signed OTA updates on the machine.
- `subsystems/cloud-api` (`SUB-CLOUD-API`) — ingests telemetry and drives OTA campaigns.

Each subsystem directory is a complete engmod model in its own right (its own `architecture.yml`,
`requirements.yml`, `catalog.yml`, `design.yml`, and `src/`), and each regenerates its own
`generated/ARCHITECTURE.adoc` and `generated/TRACE-MATRIX.json` independently of the parent.

Requirement delegation runs downward across the boundary. Top-level requirements are delegated to a specific
subsystem contract entry via `composition.allocations`, for example:

- `REQ-COF-001` delegated to `SUB-TELEMETRY` / `CAP-TELEM-REPORT`.
- `REQ-COF-002` delegated to `SUB-CLOUD-API` / `CAP-CLOUD-INGEST`.
- `REQ-COF-003` delegated to `SUB-OTA-AGENT` / `CAP-OTA-APPLY`.

A requirement delegated to a subsystem is rolled up with a `delegated` status in the parent
`TRACE-MATRIX.json` rather than being flagged as orphan. A delegation that does not name a specific target
contract entry raises a `composition.untraceable_delegation` diagnostic. Subsystem needs are bound to
providers via `composition.satisfactions` (for example `SUB-OTA-AGENT/NEED-TELEMETRY-FEED` satisfied by
`SUB-TELEMETRY/CAP-TELEM-REPORT`, and hardware needs satisfied by hosting hardware items).

All four models — the parent plus the three subsystems — are validated together by
`scripts/validate-all.sh`, so each one is built, documented, and trace-checked on its own.

This sample is synthetic and intended for architecture documentation and traceability workflows.
