// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFileAtomic replaces a file's content and keeps its permission bits
// (a 0600 link file stays 0600, an executable stays executable); a new file
// gets 0644.
// Mutation: writing every file 0644 fails this test.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestWriteFileAtomicKeepsMode(t *testing.T) {
	dir := t.TempDir()
	for _, mode := range []os.FileMode{0o600, 0o640, 0o755} {
		path := filepath.Join(dir, "f"+mode.String())
		if err := os.WriteFile(path, []byte("old"), mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		if err := writeFileAtomic(path, []byte("new")); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != mode || read(t, path) != "new" {
			t.Fatalf("%s: mode %v err %v content %q, want %v", path, info.Mode().Perm(), err, read(t, path), mode)
		}
	}
	fresh := filepath.Join(dir, "fresh")
	if err := writeFileAtomic(fresh, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(fresh); info.Mode().Perm() != 0o644 {
		t.Fatalf("new file mode %v, want 0644", info.Mode().Perm())
	}
}
