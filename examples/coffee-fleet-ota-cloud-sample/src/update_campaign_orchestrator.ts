// ENGMODEL-OWNER-UNIT: FU-UPDATE-CAMPAIGN-ORCHESTRATION

// TRACE-REQS: REQ-COF-003
export function planCampaign(cohort: string[]): string {
  return `planned:${cohort.length}`;
}

// TRACE-REQS: REQ-COF-005
export function handleRollback(machineId: string): string {
  return `rollback:${machineId}`;
}
