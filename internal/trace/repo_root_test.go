// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import (
	"fmt"
	"os"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TestMain runs the trace tests from the repository root, so the
// repository-relative paths they use (model/..., examples/...) resolve.
func TestMain(m *testing.M) {
	if err := testutil.ChdirRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "trace tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
