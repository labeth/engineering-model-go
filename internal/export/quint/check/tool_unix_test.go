//go:build unix

// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// processAlive reports whether pid is a running process; a zombie that init
// has not reaped yet is dead for this purpose.
func processAlive(pid int) bool {
	if syscall.Kill(pid, 0) != nil {
		return false
	}
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	fields := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))
	return len(fields) == 0 || fields[0] != "Z"
}

// A timeout kills the whole process group: the sleeping child that holds the
// output pipes open dies with the shell, so Run returns at once instead of
// waiting for the pipes (or the WaitDelay) and no orphan keeps running.
//
// TRLC-LINKS: REQ-EMG-143
func TestExecRunnerTimeoutKillsTheWholeProcessGroup(t *testing.T) {
	old := waitDelay
	waitDelay = 4 * time.Second
	t.Cleanup(func() { waitDelay = old })
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	start := time.Now()
	out, err := ExecRunner(context.Background(), Command{
		Path: "/bin/sh", Args: []string{"-c", "sleep 60 & echo $! > " + pidFile + "; wait"},
		Timeout: 300 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("a timeout is a verdict, not a start failure: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("Run blocked for %s: the child kept the pipes open", elapsed)
	}
	if !out.TimedOut || out.ExitCode != -1 || !strings.Contains(out.Stderr, "killed:") {
		t.Fatalf("output %+v", out)
	}
	data, readErr := os.ReadFile(pidFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	deadline := time.Now().Add(2 * time.Second)
	for processAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if processAlive(pid) {
		syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("child %d outlived the timeout", pid)
	}
}

// A tool that exits but leaves a child holding the pipes does not block Run
// past WaitDelay, and the leftover child is killed.
//
// TRLC-LINKS: REQ-EMG-143
func TestExecRunnerDoesNotWaitForAChildThatKeepsThePipesOpen(t *testing.T) {
	old := waitDelay
	waitDelay = 300 * time.Millisecond
	t.Cleanup(func() { waitDelay = old })
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	start := time.Now()
	out, err := ExecRunner(context.Background(), Command{
		Path: "/bin/sh", Args: []string{"-c", "echo hello; sleep 60 & echo $! > " + pidFile},
	})
	if err != nil || out.ExitCode != 0 || !strings.Contains(out.Stdout, "hello") {
		t.Fatalf("out %+v err %v", out, err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("Run blocked for %s", elapsed)
	}
	data, _ := os.ReadFile(pidFile)
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	deadline := time.Now().Add(2 * time.Second)
	for pid > 0 && processAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if pid > 0 && processAlive(pid) {
		syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("orphan %d is still running", pid)
	}
}

// The captured output of each stream is capped; the process still runs to the
// end and its exit code is kept.
//
// TRLC-LINKS: REQ-EMG-143
func TestExecRunnerCapsCapturedOutput(t *testing.T) {
	out, err := ExecRunner(context.Background(), Command{
		Path: "/bin/sh", Args: []string{"-c", "head -c 200000 /dev/zero | tr '\\000' x; echo failed >&2; exit 3"},
		MaxOutput: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Stdout) != 1000 || !out.Truncated || out.ExitCode != 3 || out.TimedOut || out.Stderr != "failed\n" {
		t.Fatalf("stdout %d bytes, truncated %v, exit %d, stderr %q", len(out.Stdout), out.Truncated, out.ExitCode, out.Stderr)
	}
}

// A nil environment is an empty one, never the caller's, and a missing program
// is a start failure.
//
// TRLC-LINKS: REQ-EMG-143
func TestExecRunnerStartsWithTheGivenEnvironmentOnly(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "do-not-leak")
	out, err := ExecRunner(context.Background(), Command{Path: "/bin/sh", Args: []string{"-c", `echo "[$GITHUB_TOKEN][$KEPT]"`}, Env: []string{"KEPT=yes"}})
	if err != nil || out.Stdout != "[][yes]\n" {
		t.Fatalf("out %+v err %v", out, err)
	}
	out, err = ExecRunner(context.Background(), Command{Path: "/bin/sh", Args: []string{"-c", `echo "[$GITHUB_TOKEN]"`}})
	if err != nil || out.Stdout != "[]\n" {
		t.Fatalf("a nil Env must not inherit: out %+v err %v", out, err)
	}
	if _, err := ExecRunner(context.Background(), Command{Path: "/nonexistent/quint"}); err == nil {
		t.Fatal("a missing program started")
	}
}

// Cancelling the caller's context kills the group too.
//
// TRLC-LINKS: REQ-EMG-143
func TestExecRunnerStopsWhenTheContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	start := time.Now()
	out, err := ExecRunner(ctx, Command{Path: "/bin/sh", Args: []string{"-c", "sleep 60 & wait"}})
	if err != nil || out.ExitCode != -1 || out.TimedOut {
		t.Fatalf("out %+v err %v", out, err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("cancel did not stop the process")
	}
}
