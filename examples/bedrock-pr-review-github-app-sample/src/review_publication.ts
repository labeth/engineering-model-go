// ENGMODEL-OWNER-UNIT: FU-REVIEW-PUBLICATION
// ENGMODEL-CODE-DESCRIPTION: publishes review comments and check-run outcomes back to GitHub
import { Octokit } from "github-rest-sdk/publication-client";

export class ReviewPublication {
  constructor(private readonly github: Octokit) {}

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

  // TRLC-LINKS: REQ-PRR-007
  recordRedactedAuditMetadata(metadata: { promptHash: string; findingCount: number }): string {
    return `audit:${metadata.promptHash}:${metadata.findingCount}`;
  }

  // TRLC-LINKS: REQ-PRR-008
  notifyMaintainerDeferral(owner: string, repo: string, pullNumber: number): string {
    return `deferred:${owner}/${repo}#${pullNumber}`;
  }
}
