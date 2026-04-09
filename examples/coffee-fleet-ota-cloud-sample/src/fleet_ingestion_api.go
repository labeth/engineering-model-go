// ENGMODEL-OWNER-UNIT: FU-FLEET-INGESTION-API
package src

type IngestPayload struct {
	MachineID string
	Nonce     string
}

// TRACE-REQS: REQ-COF-001, REQ-COF-002
func IngestTelemetry(p IngestPayload) bool {
	return p.MachineID != ""
}

// TRACE-REQS: REQ-COF-008
func RejectReplay(nonceSeen bool) bool {
	return nonceSeen
}
