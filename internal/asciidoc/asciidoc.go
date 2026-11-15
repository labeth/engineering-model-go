// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type Options struct {
	ViewIDs          []string
	CodeRoot         string
	DecisionsDocPath string
	// Resolution overrides ENGMOD_WORK / ENGMOD_LOCK for the composition run.
	Resolution composition.ResolutionOptions
	// Composition, when set, is a composition already resolved for this
	// bundle's manifest; Generate then does not compose again.
	Composition *composition.CompositionResult
	// UsedBy adds the "Used By" section to the Contract Items chapter: the
	// requirement links of the other checkouts of the active workspace onto
	// this model's items (D16). Off by default, also when a workspace is
	// found, so a committed render of a child never names its users; a
	// document rendered with it must not be committed into the child.
	UsedBy bool
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type Result struct {
	Document          string
	DecisionsDocument string
	Diagnostics       []validate.Diagnostic
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014
func GenerateFromFiles(architecturePath, requirementsPath, designPath string, options Options) (Result, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return Result{}, err
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		return Result{}, err
	}
	design, err := model.LoadDesign(designPath)
	if err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(options.CodeRoot) != "" && !filepath.IsAbs(options.CodeRoot) {
		baseDir := filepath.Dir(architecturePath)
		options.CodeRoot = filepath.Join(baseDir, options.CodeRoot)
	}
	return Generate(bundle, requirements, design, options)
}

// traceFindingCodes are the requirement link trace findings that strict
// mode makes errors; like link.*, they do not stop the render (D15).
var traceFindingCodes = map[string]bool{
	"requirement.no_parent": true, "requirement.unsatisfied": true, "contract.realizer_unpublished": true,
}

// hasBlockingErrors reports an error that stops the render. Requirement link
// findings (link.* and the strict-mode trace findings requirement.no_parent,
// requirement.unsatisfied and contract.realizer_unpublished) are errors of
// the model check but do not stop the render: the document shows the
// marked links and the trace a reviewer needs (D15). With lifecycle, an
// error the tier policy sets (validate.RendersDespite) does not stop it
// either: only rank-0 and untiered errors do (D15 extended, ADR-EMG-032).
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-130
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE
func hasBlockingErrors(bundle model.Bundle, diags []validate.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == validate.SeverityError && blocksRender(bundle, d.Code) {
			return true
		}
	}
	return false
}

// blocksRender reports whether an error with the code stops the render.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-130
func blocksRender(bundle model.Bundle, code string) bool {
	return !strings.HasPrefix(code, "link.") && !traceFindingCodes[code] && !validate.RendersDespite(bundle, code)
}

// validationBundle enriches the bundle with the dependencies' published
// architecture, behavior, assurance and compliance entities, so
// alias::ID references validate. Requirements are not imported. It returns the
// composition diagnostics to report: those of the resolution, or, without
// composition, an unpublished_reference for every alias::ID reference.
// TRLC-LINKS: REQ-EMG-065, REQ-EMG-066
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func validationBundle(bundle model.Bundle, composed composition.CompositionResult, compositionErr error) (model.Bundle, []validate.Diagnostic) {
	if !composition.HasComposition(bundle) {
		return bundle, composition.QualifiedReferenceDiagnostics(bundle, composition.CompositionResult{})
	}
	if compositionErr != nil {
		return bundle, []validate.Diagnostic{{
			Code: "composition.resolve_failed", Severity: validate.SeverityError,
			Message: fmt.Sprintf("composition could not be resolved: %v", compositionErr),
			Path:    bundle.ManifestPath,
		}}
	}
	projection := composition.BuildCompositionProjection(composed)
	enriched := projection.EnrichBundle(bundle, "architecture", "behavior", "assurance", "compliance")
	return enriched, composed.Diagnostics
}

// excludeUnpublishedReferenceEchoes drops the generic validation errors that
// repeat a composition.unpublished_reference (the same alias::ID reported as a
// broken reference, unknown flow target, unknown owner, ...), so each
// unpublished reference is reported once with its field path.
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func excludeUnpublishedReferenceEchoes(diags, compositionDiags []validate.Diagnostic) []validate.Diagnostic {
	unpublished := unpublishedReferences(compositionDiags)
	if len(unpublished) == 0 {
		return diags
	}
	out := diags[:0:0]
	for _, d := range diags {
		if !echoesUnpublishedReference(d, unpublished) {
			out = append(out, d)
		}
	}
	return out
}

// unpublishedReferences is the set of alias::ID references that the
// composition diagnostics report as unpublished.
// TRLC-LINKS: REQ-EMG-065
func unpublishedReferences(compositionDiags []validate.Diagnostic) map[string]bool {
	unpublished := map[string]bool{}
	for _, d := range compositionDiags {
		if d.Code != "composition.unpublished_reference" {
			continue
		}
		if ref, ok := firstQuoted(d.Message); ok {
			unpublished[ref] = true
		}
	}
	return unpublished
}

// echoesUnpublishedReference reports whether a generic validation error
// repeats one of the unpublished references.
// TRLC-LINKS: REQ-EMG-065
func echoesUnpublishedReference(d validate.Diagnostic, unpublished map[string]bool) bool {
	if d.Severity != validate.SeverityError || strings.HasPrefix(d.Code, "composition.") {
		return false
	}
	for ref := range unpublished {
		if strings.Contains(d.Message, strconv.Quote(ref)) {
			return true
		}
	}
	return false
}

// firstQuoted returns the first Go-quoted string in a diagnostic message.
// TRLC-LINKS: REQ-EMG-065
func firstQuoted(message string) (string, bool) {
	start := strings.Index(message, `"`)
	if start < 0 {
		return "", false
	}
	for end := start + 1; end < len(message); end++ {
		if message[end] == '"' && message[end-1] != '\\' {
			value, err := strconv.Unquote(message[start : end+1])
			return value, err == nil
		}
	}
	return "", false
}
