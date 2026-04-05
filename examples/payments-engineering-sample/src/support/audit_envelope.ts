// ENGMODEL-OWNER-UNIT: FU-RISK-SCORING
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
