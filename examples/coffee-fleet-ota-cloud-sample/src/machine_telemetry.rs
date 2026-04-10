// ENGMODEL-OWNER-UNIT: FU-MACHINE-TELEMETRY-COLLECTION
// ENGMODEL-CODE-DESCRIPTION: builds machine telemetry and status payloads at the edge

pub struct TelemetryRecord {
    pub machine_id: String,
    pub brew_temp_c: i32,
}

// TRACE-REQS: REQ-COF-001
pub fn publish_brew_telemetry(record: &TelemetryRecord) -> bool {
    !record.machine_id.is_empty() && record.brew_temp_c > 0
}

// TRACE-REQS: REQ-COF-006
pub fn queue_for_retry(record: &TelemetryRecord) -> String {
    format!("retry:{}:{}", record.machine_id, record.brew_temp_c)
}
