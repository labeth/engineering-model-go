// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

// allowRedRepoModelEnv, when set to 1, skips TestRepositoryModelGate: the
// documented way to run the suite while the owner has not yet accepted
// DEC-TIER-OPERATIONAL and DEC-HEALTH-BASELINE or renewed the overdue POA&M
// items. Without it the test runs and fails on any gate error.
const allowRedRepoModelEnv = "ENGMOD_ALLOW_RED_REPO_MODEL"

// The repository's own model is the gate the project holds itself to: it is
// asserted here on purpose and not through a test copy (AcceptedRepoModel is
// only for exporter tests that need a valid model). The test fails, listing
// the validation errors, while the owner has not accepted the decisions or
// renewed the POA&M items; it is not weakened. Set
// ENGMOD_ALLOW_RED_REPO_MODEL=1 to skip it for a run in which that red state
// is known and accepted by the owner.
// Mutation: a proposed tier decision, an overdue POA&M item or any other gate
// error in engmod.yml or model/ fails the test.
// TRLC-LINKS: REQ-EMG-124
func TestRepositoryModelGate(t *testing.T) {
	if os.Getenv(allowRedRepoModelEnv) == "1" {
		t.Skipf("%s=1: the red repository model gate is allowed for this run", allowRedRepoModelEnv)
	}
	t.Setenv("ENGMOD_AS_OF", "") // the package pins a date; the gate runs on today's
	manifest := filepath.Join(testutil.RepoRoot(t), "engmod.yml")
	// codescan.LoadBundle attaches the CODE identifier index the validator
	// needs to resolve the control verification refs.
	bundle, err := codescan.LoadBundle(manifest)
	if err != nil {
		t.Fatalf("load %s: %v", manifest, err)
	}
	var lines []string
	for _, d := range engmodel.ValidateBundle(bundle) {
		if d.Severity == engmodel.SeverityError {
			lines = append(lines, fmt.Sprintf("  %s: %s (%s)", d.Code, d.Message, d.Path))
		}
	}
	if len(lines) > 0 {
		t.Fatalf("the repository model has gate errors; the owner accepts DEC-TIER-OPERATIONAL and DEC-HEALTH-BASELINE and renews or closes the overdue POA&M items:\n%s", strings.Join(lines, "\n"))
	}
}
