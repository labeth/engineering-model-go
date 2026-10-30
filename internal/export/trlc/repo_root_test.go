// ENGMODEL-OWNER-UNIT: FU-TRLC-EXPORTER
package trlc

import (
	"fmt"
	"os"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TestMain runs the trlc tests from the repository root, so the
// repository-relative paths they use (examples/..., engmod.yml, model/...)
// resolve.
func TestMain(m *testing.M) {
	if err := testutil.ChdirRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "trlc tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
