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
	// The examples carry POA&M items with due dates and lifecycle.poam_overdue is
	// a core error, so the as-of date is pinned for the whole package to keep the
	// recorded goldens and renders independent of the day the tests run.
	// TestRepositoryModelGate clears the pin: it asserts the real date.
	if err := os.Setenv("ENGMOD_AS_OF", "2026-01-01"); err != nil {
		fmt.Fprintln(os.Stderr, "integration tests:", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}
