//go:build !unix

// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import "os/exec"

// isolateProcess has no process groups to use here: only the tool itself is
// killed on timeout, and WaitDelay bounds the wait for children that keep the
// output pipes open.
//
// TRLC-LINKS: REQ-EMG-143
func isolateProcess(*exec.Cmd) {}

// killGroup does nothing where there is no process group.
//
// TRLC-LINKS: REQ-EMG-143
func killGroup(*exec.Cmd) error { return nil }
