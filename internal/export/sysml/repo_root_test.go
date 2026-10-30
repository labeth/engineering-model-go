// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package sysml

import (
	"fmt"
	"os"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TestMain runs the sysml tests from the repository root, so the
// repository-relative paths they use (examples/..., engmod.yml, model/...)
// resolve.
func TestMain(m *testing.M) {
	if err := testutil.ChdirRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "sysml tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
