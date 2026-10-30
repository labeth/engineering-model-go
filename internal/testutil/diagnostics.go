// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package testutil

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// AssertDiag fails the test unless diags holds a diagnostic with the code
// whose message or path mentions fragment.
// TRLC-LINKS: REQ-EMG-059
func AssertDiag(t testing.TB, diags []validate.Diagnostic, code, fragment string) {
	t.Helper()
	for _, d := range diags {
		if d.Code == code && (strings.Contains(d.Message, fragment) || strings.Contains(d.Path, fragment)) {
			return
		}
	}
	t.Fatalf("missing %s mentioning %q in %+v", code, fragment, diags)
}
