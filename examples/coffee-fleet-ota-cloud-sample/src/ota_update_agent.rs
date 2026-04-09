// ENGMODEL-OWNER-UNIT: FU-OTA-UPDATE-AGENT

// TRACE-REQS: REQ-COF-003, REQ-COF-004
pub fn validate_and_apply(signature_valid: bool) -> &'static str {
    if signature_valid {
        "applied"
    } else {
        "rejected"
    }
}

// TRACE-REQS: REQ-COF-005
pub fn rollback_and_report() -> &'static str {
    "rollback_reported"
}
