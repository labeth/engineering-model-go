// ENGMODEL-OWNER-UNIT: FU-UPDATE-CAMPAIGN-ORCHESTRATION
// ENGMODEL-CODE-DESCRIPTION: plans OTA rollout cohorts and campaign lifecycle transitions

// TRLC-LINKS: REQ-COF-003
export function planCampaign(cohort: string[]): string {
  return `planned:${cohort.length}`;
}

// TRLC-LINKS: REQ-COF-005
export function handleRollback(machineId: string): string {
  return `rollback:${machineId}`;
}
