// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engine

import (
	"fmt"
	"os"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TestMain runs the engine tests from the repository root, so the paths they
// use (examples/..., engmod.yml, model/..., go run ./cmd/...) mean what they
// meant before the implementation moved out of the root package.
func TestMain(m *testing.M) {
	if err := testutil.ChdirRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "engine tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
