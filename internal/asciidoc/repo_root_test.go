// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"fmt"
	"os"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TestMain runs the asciidoc tests from the repository root, so the
// repository-relative paths they use (examples/..., engmod.yml, model/...,
// go run ./cmd/...) resolve.
func TestMain(m *testing.M) {
	if err := testutil.ChdirRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "asciidoc tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
