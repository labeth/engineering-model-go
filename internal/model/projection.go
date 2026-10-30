// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"fmt"
	"sort"
	"strings"
)

// CanonicalBundle is the single exporter-facing compatibility facade.
// Documents are retained as serialization DTOs for output fidelity; Semantic
// is the only normalized concept graph.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-CANONICAL-SEMANTIC-MODEL
type CanonicalBundle struct {
	documents   Bundle
	semantic    SemanticModel
	diagnostics []SemanticDiagnostic
}

// LoadCanonicalBundle validates YAML, decodes it, and constructs the canonical
// semantic projection before returning exporter input.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-044, REQ-EMG-046
func LoadCanonicalBundle(manifestPath string) (CanonicalBundle, error) {
	bundle, err := LoadBundle(manifestPath)
	if err != nil {
		return CanonicalBundle{}, err
	}
	return NewCanonicalBundle(bundle)
}

// NewCanonicalBundle constructs the one normalized semantic graph used by
// compatibility exporters. Projection errors are never hidden by falling back
// to the decoded YAML DTOs.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func NewCanonicalBundle(bundle Bundle) (CanonicalBundle, error) {
	projectionInput := bundle
	if strings.TrimSpace(projectionInput.Architecture.Model.ID) == "" {
		projectionInput.Architecture.Model.ID = "COMPATIBILITY-MODEL"
		projectionInput.Architecture.Model.Title = "Compatibility model"
	}
	semantic, diagnostics := ProjectSemanticModel(projectionInput)
	diagnostics = append(diagnostics, ValidateSemanticModel(semantic)...)
	diagnostics = sortSemanticDiagnostics(diagnostics)
	canonical := CanonicalBundle{
		documents:   bundle,
		semantic:    semantic,
		diagnostics: diagnostics,
	}
	if HasSemanticErrors(diagnostics) {
		return canonical, &SemanticProjectionError{Diagnostics: diagnostics}
	}
	return canonical, nil
}

// NewCanonicalRequirements constructs canonical requirement concepts for
// requirement-only projections such as TRLC. The synthetic package is an
// adapter namespace, not an independently maintained domain graph.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func NewCanonicalRequirements(requirements RequirementsDocument) (CanonicalBundle, error) {
	namespace := strings.TrimSpace(requirements.LintRun.ID)
	if namespace == "" {
		namespace = "REQUIREMENTS-MODEL"
	}
	return NewCanonicalBundle(Bundle{
		Architecture: ArchitectureDocument{Model: ModelMeta{ID: namespace, Title: namespace}},
		Requirements: requirements,
	})
}

// Documents returns the validated serialization DTOs associated with the
// canonical graph. Callers must not treat them as another normalized graph.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (c CanonicalBundle) Documents() Bundle {
	return c.documents
}

// Semantic returns the canonical semantic model.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (c CanonicalBundle) Semantic() SemanticModel {
	return c.semantic
}

// Diagnostics returns projection warnings and errors.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (c CanonicalBundle) Diagnostics() []SemanticDiagnostic {
	return append([]SemanticDiagnostic(nil), c.diagnostics...)
}

// HasSemanticErrors reports whether canonical projection failed.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func HasSemanticErrors(diagnostics []SemanticDiagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SemanticSeverityError {
			return true
		}
	}
	return false
}

// SemanticProjectionError preserves all canonical projection diagnostics.
type SemanticProjectionError struct {
	Diagnostics []SemanticDiagnostic
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func (e *SemanticProjectionError) Error() string {
	if len(e.Diagnostics) == 0 {
		return "canonical semantic projection failed"
	}
	first := e.Diagnostics[0]
	if first.Path == "" {
		return fmt.Sprintf("canonical semantic projection failed: %s: %s", first.Code, first.Message)
	}
	return fmt.Sprintf("canonical semantic projection failed at %s: %s: %s", first.Path, first.Code, first.Message)
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func sortSemanticDiagnostics(diagnostics []SemanticDiagnostic) []SemanticDiagnostic {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path != diagnostics[j].Path {
			return diagnostics[i].Path < diagnostics[j].Path
		}
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
	return diagnostics
}
