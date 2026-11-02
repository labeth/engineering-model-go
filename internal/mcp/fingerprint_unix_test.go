//go:build unix

// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// The reload fingerprint reads only regular files: a FIFO at the local lock
// path neither blocks it nor fails the tool call.
// Mutation: dropping the regular-file check reads the FIFO, which blocks
// until the timeout fails the test.
// TRLC-LINKS: REQ-EMG-089
// ENGMODEL-LINKS: FU-MCP-SERVER, TB-REPO-WORKSPACE
func TestModelFingerprintSkipsNonRegularFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "payments")
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("..", "..", "examples", "payments-engineering-sample"))); err != nil {
		t.Fatal(err)
	}
	s := initializedServerAt(t, dir)
	if err := syscall.Mkfifo(filepath.Join(dir, "engmod.lock.local.yml"), 0o600); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	done := make(chan string, 1)
	go func() { done <- s.modelFileFingerprint() }()
	select {
	case fp := <-done:
		if fp == "" || fp == s.fingerprint {
			t.Fatalf("a new FIFO must change the fingerprint without reading it: %q", fp)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("modelFileFingerprint blocked on a FIFO")
	}
}
