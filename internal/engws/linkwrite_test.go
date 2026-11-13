// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeAllOrNothing restores an existing file and removes a file it
// created when a later write fails, and leaves an untouched file alone.
// Mutation: not removing a created file, or not restoring an existing one,
// fails this test.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestWriteAllOrNothingRemovesCreatedFiles(t *testing.T) {
	dir := t.TempDir()
	created, existing, failing := filepath.Join(dir, "a-new.yml"), filepath.Join(dir, "b-old.yml"), filepath.Join(dir, "c-fail.yml")
	write(t, existing, "old\n")
	write(t, failing, "keep\n")
	old := writeFile
	t.Cleanup(func() { writeFile = old })
	writeFile = func(path string, data []byte) error {
		if path == failing {
			return fmt.Errorf("injected")
		}
		return old(path, data)
	}
	paths := []string{created, existing, failing}
	err := writeAllOrNothing(paths, map[string][]byte{created: []byte("new\n"), existing: []byte("changed\n"), failing: []byte("x\n")})
	if err == nil || !strings.Contains(err.Error(), "every file restored") {
		t.Fatalf("writeAllOrNothing: %v", err)
	}
	if _, err := os.Stat(created); !os.IsNotExist(err) {
		t.Fatalf("created file kept: %v", err)
	}
	if read(t, existing) != "old\n" || read(t, failing) != "keep\n" {
		t.Fatalf("existing %q, failing %q", read(t, existing), read(t, failing))
	}
}

// engws has no migrate-links command (ADR-EMG-078): the existing dispatch
// rejects it as an unknown command with exit 2, with or without its former
// flags, and the usage text no longer names it.
// Mutation: putting "migrate-links": 0 back into the command table of Main
// fails this test.
// TRLC-LINKS: REQ-EMG-187
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestMigrateLinksIsRejectedAsUnknownCommand(t *testing.T) {
	for _, args := range [][]string{{"migrate-links"}, {"migrate-links", "--repo", t.TempDir()}} {
		code, out, errOut := runMain(t, fakeRegistry{}, args...)
		if code != 2 || out != "" || !strings.Contains(errOut, `engws: unknown command "migrate-links"`) {
			t.Fatalf("%v: exit %d\n%s\n%s", args, code, out, errOut)
		}
		assertUsageLacksMigrateLinks(t, errOut)
	}
	if code, _, _ := runMain(t, fakeRegistry{}, "migrate-links", "--dry-run"); code != 2 {
		t.Fatalf("migrate-links --dry-run: exit %d, want 2", code)
	}
	_, help, _ := runMain(t, fakeRegistry{}, "--help")
	assertUsageLacksMigrateLinks(t, help)
}

// assertUsageLacksMigrateLinks fails when the usage part of text names
// migrate-links.
// TRLC-LINKS: REQ-EMG-187
func assertUsageLacksMigrateLinks(t *testing.T, text string) {
	t.Helper()
	usage := text[strings.Index(text, "usage: engws"):]
	if strings.Contains(usage, "migrate-links") {
		t.Fatalf("the usage still names migrate-links:\n%s", text)
	}
}
