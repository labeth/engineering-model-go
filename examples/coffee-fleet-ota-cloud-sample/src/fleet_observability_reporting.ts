// ENGMODEL-OWNER-UNIT: FU-FLEET-OBSERVABILITY-REPORTING
// ENGMODEL-CODE-DESCRIPTION: aggregates telemetry and OTA outcome signals for fleet observability reporting

// TRLC-LINKS: REQ-COF-002
export function persistTelemetryMetric(machineId: string): string {
  return `metric:${machineId}`;
}

// TRLC-LINKS: REQ-COF-007
export function persistAuditLog(recordId: string): string {
  return `audit:${recordId}`;
}

// TRLC-LINKS: REQ-COF-008
export function notifyReplayAbuse(machineId: string): string {
  return `abuse:${machineId}`;
}
