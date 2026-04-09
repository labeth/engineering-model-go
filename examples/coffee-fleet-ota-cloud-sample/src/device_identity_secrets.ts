// ENGMODEL-OWNER-UNIT: FU-DEVICE-IDENTITY-SECRETS

// TRACE-REQS: REQ-COF-004
export function verifyFirmwareSignature(signatureValid: boolean): boolean {
  return signatureValid;
}

// TRACE-REQS: REQ-COF-007
export function redactIdentityForAudit(machineId: string): string {
  return machineId.slice(0, 4) + "-redacted";
}
