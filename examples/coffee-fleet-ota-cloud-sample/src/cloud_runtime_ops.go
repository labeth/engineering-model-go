// ENGMODEL-OWNER-UNIT: FU-CLOUD-RUNTIME-OPERATIONS
// ENGMODEL-CODE-DESCRIPTION: wires cloud runtime handlers and integration entrypoints
package src

// TRLC-LINKS: REQ-COF-008
func RaiseSecurityNotification(machineID string) string {
	if machineID == "" {
		return "noop"
	}
	return "security-notified"
}
