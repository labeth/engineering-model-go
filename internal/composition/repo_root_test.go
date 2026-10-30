// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

import (
	"fmt"
	"os"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TestMain runs the composition tests from the repository root, so the
// repository-relative paths they use (model/..., examples/...) resolve.
func TestMain(m *testing.M) {
	if err := testutil.ChdirRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "composition tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
