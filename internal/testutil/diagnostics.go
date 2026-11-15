// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package testutil

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// DiagWant is the diagnostic AssertDiag looks for: its code and a fragment
// of its message or path.
type DiagWant struct{ Code, Fragment string }

// AssertDiag fails the test unless diags holds a diagnostic with the code
// whose message or path mentions fragment.
// TRLC-LINKS: REQ-EMG-059
func AssertDiag(t testing.TB, diags []validate.Diagnostic, want DiagWant) {
	t.Helper()
	code, fragment := want.Code, want.Fragment
	for _, d := range diags {
		if d.Code == code && (strings.Contains(d.Message, fragment) || strings.Contains(d.Path, fragment)) {
			return
		}
	}
	t.Fatalf("missing %s mentioning %q in %+v", code, fragment, diags)
}
