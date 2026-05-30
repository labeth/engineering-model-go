// ENGMODEL-OWNER-UNIT: FU-REVIEW-PUBLICATION
// ENGMODEL-CODE-DESCRIPTION: publishes review comments and check-run outcomes back to GitHub
import { Octokit } from "github-rest-sdk/publication-client";

// ENGMODEL-LINKS: IF-BEDROCK-REVIEW-PUBLISH-API, FLOW-BEDROCK-PR-REVIEW, DO-BEDROCK-REVIEW-FINDINGS, FU-REVIEW-PUBLICATION
export class ReviewPublication {
  // ENGMODEL-LINKS: IF-BEDROCK-REVIEW-PUBLISH-API, FLOW-BEDROCK-PR-REVIEW, DO-BEDROCK-REVIEW-FINDINGS
  // TRLC-LINKS: REQ-PRR-005, REQ-PRR-007, REQ-PRR-008
  constructor(private readonly github: Octokit) {}

  // ENGMODEL-LINKS: IF-BEDROCK-REVIEW-PUBLISH-API, FLOW-BEDROCK-PR-REVIEW, DO-BEDROCK-REVIEW-FINDINGS
  // TRLC-LINKS: REQ-PRR-005
  async publishCheckRun(owner: string, repo: string, headSha: string, summary: string): Promise<void> {
    await this.github.checks.create({
      owner,
      repo,
      name: "bedrock-pr-review",
      head_sha: headSha,
      status: "completed",
      conclusion: "neutral",
      output: { title: "PR Review", summary },
    });
  }

  // ENGMODEL-LINKS: IF-BEDROCK-REVIEW-PUBLISH-API, FLOW-BEDROCK-PR-REVIEW, DO-BEDROCK-REVIEW-FINDINGS
  // TRLC-LINKS: REQ-PRR-007
  recordRedactedAuditMetadata(metadata: { promptHash: string; findingCount: number }): string {
    return `audit:${metadata.promptHash}:${metadata.findingCount}`;
  }

  // ENGMODEL-LINKS: IF-BEDROCK-REVIEW-PUBLISH-API, FLOW-BEDROCK-POLICY-TUNING
  // TRLC-LINKS: REQ-PRR-008
  notifyMaintainerDeferral(owner: string, repo: string, pullNumber: number): string {
    return `deferred:${owner}/${repo}#${pullNumber}`;
  }
}
