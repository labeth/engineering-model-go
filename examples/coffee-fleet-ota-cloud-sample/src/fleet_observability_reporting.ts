// ENGMODEL-OWNER-UNIT: FU-FLEET-OBSERVABILITY-REPORTING
// ENGMODEL-CODE-DESCRIPTION: aggregates telemetry and OTA outcome signals for fleet observability reporting

// ENGMODEL-LINKS: IF-COFFEE-TELEMETRY-INGEST, FLOW-COFFEE-TELEMETRY-INGEST, DO-COFFEE-TELEMETRY-EVENT, FU-FLEET-OBSERVABILITY-REPORTING
// TRLC-LINKS: REQ-COF-002
export function persistTelemetryMetric(machineId: string): string {
  return `metric:${machineId}`;
}

// ENGMODEL-LINKS: FLOW-COFFEE-TELEMETRY-INGEST, DO-COFFEE-TELEMETRY-EVENT
// TRLC-LINKS: REQ-COF-007
export function persistAuditLog(recordId: string): string {
  return `audit:${recordId}`;
}

// ENGMODEL-LINKS: IF-COFFEE-TELEMETRY-INGEST, FLOW-COFFEE-TELEMETRY-INGEST, CTRL-COFFEE-DEVICE-IDENTITY
// TRLC-LINKS: REQ-COF-008
export function notifyReplayAbuse(machineId: string): string {
  return `abuse:${machineId}`;
}
