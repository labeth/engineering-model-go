// ENGMODEL-OWNER-UNIT: FU-STRUCTURIZR-EXPORTER
package structurizr

import (
	"fmt"
	"os"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TestMain runs the structurizr tests from the repository root, so the
// repository-relative paths they use (examples/..., engmod.yml, model/...)
// resolve.
func TestMain(m *testing.M) {
	if err := testutil.ChdirRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "structurizr tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
