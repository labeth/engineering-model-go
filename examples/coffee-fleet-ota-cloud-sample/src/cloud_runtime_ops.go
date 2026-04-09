// ENGMODEL-OWNER-UNIT: FU-CLOUD-RUNTIME-OPERATIONS
package src

// TRACE-REQS: REQ-COF-008
func RaiseSecurityNotification(machineID string) string {
	if machineID == "" {
		return "noop"
	}
	return "security-notified"
}
