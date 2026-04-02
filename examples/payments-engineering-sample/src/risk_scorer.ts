export class RiskScoringService {
  // TRACE-REQS: REQ-PAY-002
  calculateRiskScore(paymentId: string, amountCents: number): number {
    console.log(
      `risk-scoring-service: calculate risk for ${paymentId} (${amountCents} cents)`
    );
    return 72;
  }

  // TRACE-REQS: REQ-PAY-002
  classifyRisk(riskScore: number): "low" | "medium" | "high" {
    console.log(`risk-scoring-service: classify score ${riskScore}`);
    if (riskScore >= 70) return "high";
    if (riskScore >= 40) return "medium";
    return "low";
  }

  // TRACE-REQS: REQ-PAY-004
  isHighRisk(riskScore: number): boolean {
    console.log(`risk-scoring-service: high-risk gate check for ${riskScore}`);
    return riskScore >= 70;
  }
}

export class FraudAuditService {
  // TRACE-REQS: REQ-PAY-005
  createAuditRecord(paymentId: string, riskScore: number): void {
    console.log(
      `fraud-audit-service: persist audit record for ${paymentId} with risk ${riskScore}`
    );
  }

  // TRACE-REQS: REQ-PAY-005
  enrichAuditMetadata(paymentId: string): void {
    console.log(`fraud-audit-service: enrich metadata for ${paymentId}`);
  }

  // TRACE-REQS: REQ-PAY-004
  buildReviewReason(riskScore: number): string {
    console.log(`fraud-audit-service: build review reason for score ${riskScore}`);
    return `risk-score-${riskScore}-above-threshold`;
  }
}
