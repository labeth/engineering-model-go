// ENGMODEL-OWNER-UNIT: FU-POLICY-CHECKS
// ENGMODEL-CODE-DESCRIPTION: evaluates policy and style rules before review publication
pub struct PolicyChecks;

impl PolicyChecks {
    // TRLC-LINKS: REQ-PRR-004
    pub fn run_deterministic_checks(&self, changed_files: &[String]) -> Vec<String> {
        let mut findings = Vec::new();
        for file in changed_files {
            if file.contains("terraform") || file.ends_with(".tf") {
                findings.push("infra-risk-change".to_string());
            }
            if file.to_lowercase().contains("secret") {
                findings.push("secret-exposure-risk".to_string());
            }
        }
        findings
    }

    // TRLC-LINKS: REQ-PRR-006
    pub fn policy_only_summary(&self, findings: &[String]) -> String {
        format!("policy-findings:{}", findings.len())
    }
}
