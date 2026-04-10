// ENGMODEL-OWNER-UNIT: FU-RISK-SCORING
// ENGMODEL-CODE-DESCRIPTION: formats support-review audit envelopes for traceability workflows
export type AuditEnvelope = {
  paymentId: string;
  riskScore: number;
  recordType: "fraud-audit";
};

export function createAuditEnvelope(paymentId: string, riskScore: number): AuditEnvelope {
  return {
    paymentId,
    riskScore,
    recordType: "fraud-audit",
  };
}
