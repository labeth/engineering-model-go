// ENGMODEL-OWNER-UNIT: FU-AIRBORNE-ASSURANCE-EXPORTER
package airborne

import (
	"fmt"
	"os"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TestMain runs the airborne tests from the repository root, so the
// repository-relative paths they use (examples/..., engmod.yml, model/...)
// resolve.
func TestMain(m *testing.M) {
	if err := testutil.ChdirRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "airborne tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
