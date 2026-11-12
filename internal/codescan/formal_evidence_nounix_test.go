//go:build !unix

// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import "testing"

// mkfifo skips where FIFOs do not exist.
//
// TRLC-LINKS: REQ-EMG-145
func mkfifo(t *testing.T, path string) {
	t.Helper()
	t.Skip("no FIFOs on this platform")
}
