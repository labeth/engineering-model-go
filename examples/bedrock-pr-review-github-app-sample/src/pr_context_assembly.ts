// ENGMODEL-OWNER-UNIT: FU-PR-CONTEXT-ASSEMBLY
import { Octokit } from "github-rest-sdk";

export class PRContextAssembly {
  constructor(private readonly github: Octokit) {}

  // TRACE-REQS: REQ-PRR-002
  async fetchChangedFiles(owner: string, repo: string, pullNumber: number): Promise<string[]> {
    const files = await this.github.pulls.listFiles({ owner, repo, pull_number: pullNumber });
    return files.data.map((f) => f.filename);
  }

  // TRACE-REQS: REQ-PRR-003
  buildDiffContext(files: string[], patchByFile: Record<string, string>): string {
    return files
      .map((file) => `FILE:${file}\n${patchByFile[file] ?? ""}`)
      .join("\n---\n")
      .slice(0, 60000);
  }
}
