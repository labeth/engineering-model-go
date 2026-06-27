// ENGMODEL-OWNER-UNIT: FU-TELEM-REPORT
// ENGMODEL-CODE-DESCRIPTION: samples machine sensors and reports signed telemetry to the fleet

// ENGMODEL-LINKS: DO-TELEM-RECORD, IF-TELEM-REPORT, FU-TELEM-REPORT
pub struct TelemetryRecord {
    pub machine_id: String,
    pub flow_ml: i32,
}

// TRLC-LINKS: REQ-TELEM-001
// ENGMODEL-LINKS: FU-TELEM-REPORT, IF-TELEM-REPORT, DO-TELEM-RECORD
pub fn report_telemetry_record(record: &TelemetryRecord) -> bool {
    !record.machine_id.is_empty()
}

// TRLC-LINKS: REQ-TELEM-002
// ENGMODEL-LINKS: FU-TELEM-SAMPLE
pub fn sample_machine_sensors() -> TelemetryRecord {
    TelemetryRecord {
        machine_id: String::new(),
        flow_ml: 0,
    }
}

// TRLC-LINKS: REQ-TELEM-003
// ENGMODEL-LINKS: FU-TELEM-REPORT, CTRL-TELEM-INTEGRITY
pub fn sign_telemetry_record(record: &TelemetryRecord) -> String {
    format!("sig:{}", record.machine_id)
}
