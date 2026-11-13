//go:build unix

// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// isolateProcess puts the tool in its own process group and makes the
// cancellation of the context kill the whole group, so the Apalache JVM or
// the Node child that Quint starts dies with it instead of keeping the
// output pipes open.
//
// TRLC-LINKS: REQ-EMG-143
func isolateProcess(proc *exec.Cmd) {
	proc.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	proc.Cancel = func() error { return killGroup(proc) }
}

// killGroup kills every process of the tool's process group; a group that is
// already gone is not an error.
//
// TRLC-LINKS: REQ-EMG-143
func killGroup(proc *exec.Cmd) error {
	if proc.Process == nil {
		return nil
	}
	err := syscall.Kill(-proc.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
