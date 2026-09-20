// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engmodel

import (
	"fmt"

	"github.com/labeth/engineering-model-go/model"
	mermaidrenderer "github.com/labeth/engineering-model-go/render/mermaid"
	svgrenderer "github.com/labeth/engineering-model-go/render/svg"
	"github.com/labeth/engineering-model-go/validate"
	"github.com/labeth/engineering-model-go/view"
)

// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type Result struct {
	Bundle      model.Bundle
	View        view.ProjectedView
	Mermaid     string
	SVG         string
	Diagnostics []validate.Diagnostic
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION
func GenerateFromFile(architecturePath, viewID string) (Result, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return Result{}, err
	}
	result, err := Generate(bundle, viewID)
	if err == nil {
		result.SVG = svgrenderer.Render(result.View, architecturePath)
	}
	return result, err
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func Generate(bundle model.Bundle, viewID string) (Result, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return Result{Bundle: bundle}, err
	}
	bundle = canonical.Documents()
	diags := validate.Bundle(bundle)
	pv, viewDiags := view.Build(bundle, viewID)
	diags = append(diags, viewDiags...)
	diags = validate.SortDiagnostics(diags)

	if validate.HasErrors(viewDiags) {
		return Result{Bundle: bundle, View: pv, Diagnostics: diags}, fmt.Errorf("view projection failed")
	}

	mmd := mermaidrenderer.Render(pv)
	svg := svgrenderer.Render(pv, bundle.Architecture.Model.ID)
	return Result{Bundle: bundle, View: pv, Mermaid: mmd, SVG: svg, Diagnostics: diags}, nil
}
