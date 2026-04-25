// ENGMODEL-OWNER-UNIT: FU-FLEET-INGESTION-API
// ENGMODEL-CODE-DESCRIPTION: ingests device telemetry and emits normalized fleet ingestion events
package src

type IngestPayload struct {
	MachineID string
	Nonce     string
}

// TRLC-LINKS: REQ-COF-001, REQ-COF-002
func IngestTelemetry(p IngestPayload) bool {
	return p.MachineID != ""
}

// TRLC-LINKS: REQ-COF-008
func RejectReplay(nonceSeen bool) bool {
	return nonceSeen
}
