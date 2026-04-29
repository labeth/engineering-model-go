// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engmodel

import (
	"fmt"

	"github.com/labeth/engineering-model-go/model"
	mermaidrenderer "github.com/labeth/engineering-model-go/render/mermaid"
	"github.com/labeth/engineering-model-go/validate"
	"github.com/labeth/engineering-model-go/view"
)

type Result struct {
	Bundle      model.Bundle
	View        view.ProjectedView
	Mermaid     string
	Diagnostics []validate.Diagnostic
}

// TRLC-LINKS: REQ-EMG-001
func GenerateFromFile(architecturePath, viewID string) (Result, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return Result{}, err
	}
	return Generate(bundle, viewID)
}

func Generate(bundle model.Bundle, viewID string) (Result, error) {
	diags := validate.Bundle(bundle)
	pv, viewDiags := view.Build(bundle, viewID)
	diags = append(diags, viewDiags...)
	diags = validate.SortDiagnostics(diags)

	if validate.HasErrors(viewDiags) {
		return Result{Bundle: bundle, View: pv, Diagnostics: diags}, fmt.Errorf("view projection failed")
	}

	mmd := mermaidrenderer.Render(pv)
	return Result{Bundle: bundle, View: pv, Mermaid: mmd, Diagnostics: diags}, nil
}
