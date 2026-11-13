// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package exportgate

import (
	"io"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ValidationGate loads the model named by its engmod.yml manifest with its CODE
// identifier index (control verification refs name inferred test declarations,
// so a bundle without the index would reject them), validates
// it with the severities of its life-cycle tier, prints every diagnostic to w
// (with the "fix:" line of a required-switch error) and reports whether the
// model has no error. An exporter CLI that does not validate through its
// library runs this gate first and refuses to export when ok is false. A
// model that does not load is returned as an error.
//
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-VALIDATION-ENGINE
func ValidationGate(manifestPath string, w io.Writer) (bundle model.Bundle, ok bool, err error) {
	bundle, err = codescan.LoadBundle(manifestPath)
	if err != nil {
		return model.Bundle{}, false, err
	}
	diagnostics := validate.SortDiagnostics(validate.ApplyTierPolicy(bundle, validate.Bundle(bundle)))
	for _, d := range diagnostics {
		validate.WriteDiagnostic(w, d)
	}
	return bundle, !validate.HasErrors(diagnostics), nil
}
