// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package integration

import (
	"fmt"
	"os"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TestMain runs the integration tests from the repository root, so the
// repository-relative paths they use (examples/..., engmod.yml, model/...,
// go run ./cmd/...) resolve.
func TestMain(m *testing.M) {
	if err := testutil.ChdirRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "integration tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
