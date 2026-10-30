// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMain runs the engine tests from the repository root, so the paths they
// use (examples/..., engmod.yml, model/..., go run ./cmd/...) mean what they
// meant before the implementation moved out of the root package.
func TestMain(m *testing.M) {
	root, err := repoRoot()
	if err == nil {
		err = os.Chdir(root)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "engine tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// repoRoot walks up from the package directory to the directory holding go.mod.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above the engine package")
		}
		dir = parent
	}
}
