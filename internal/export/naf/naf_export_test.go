// ENGMODEL-OWNER-UNIT: FU-NAF-EXPORTER
package naf

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-041, REQ-EMG-042
// ENGMODEL-LINKS: FU-NAF-EXPORTER, DO-NAF-V4-ARCHITECTURE, REF-NAF-V4-1-SPECIFICATION
func TestGenerateNAFV41UsesCanonicalViews(t *testing.T) {
	manifest := testutil.AcceptedRepoModel(t)
	first, err := GenerateV41FromFile(manifest)
	if err != nil {
		t.Fatalf("generate NAF document: %v\n%+v", err, first.Diagnostics)
	}
	second, err := GenerateV41FromFile(manifest)
	if err != nil {
		t.Fatalf("regenerate NAF document: %v", err)
	}
	if first.Document != second.Document {
		t.Fatal("NAF generation is not deterministic")
	}
	for _, expected := range []string{
		":naf-version: 4.1",
		"== Stakeholders",
		"== Concerns",
		"=== A2 - Architecture Products",
		"`VIEW-ARCHITECTURE-INTENT`",
		"`FU-NAF-EXPORTER`",
		"canonical SysML/KerML-aligned semantic model",
	} {
		if !strings.Contains(first.Document, expected) {
			t.Fatalf("generated NAF document missing %q", expected)
		}
	}
}

// hasMetadata reports whether the element with the given ID carries metadata of the type.
func hasMetadata(elements []model.SemanticElement, id, metadataType string) bool {
	for _, element := range elements {
		if element.ID == id && carriesMetadata(element, metadataType) {
			return true
		}
	}
	return false
}

func carriesMetadata(element model.SemanticElement, metadataType string) bool {
	for _, metadata := range element.Metadata {
		if metadata.Type == metadataType {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-041
func TestNAFProfileIsCanonicalMetadata(t *testing.T) {
	bundle, err := model.LoadBundle("engmod.yml")
	if err != nil {
		t.Fatal(err)
	}
	semantic, diagnostics := model.ProjectSemanticModel(bundle)
	if model.HasSemanticErrors(diagnostics) {
		t.Fatalf("project semantic model: %+v", diagnostics)
	}
	foundProfile := hasMetadata(semantic.Elements, semantic.ID, "naf.profile.v4_1")
	foundViewpoint := hasMetadata(semantic.Elements, "VIEW-ARCHITECTURE-INTENT", "naf.viewpoint.A2")
	if !foundProfile || !foundViewpoint {
		t.Fatalf("NAF profile was not projected as metadata: profile=%v viewpoint=%v", foundProfile, foundViewpoint)
	}
}

// TRLC-LINKS: REQ-EMG-043
func TestNAFValidationRejectsInvalidProfile(t *testing.T) {
	tests := []struct {
		name string
		edit func(*model.Bundle)
		code string
	}{
		{
			name: "unsupported version",
			edit: func(bundle *model.Bundle) { bundle.Architecture.NAF.Version = "4.2" },
			code: "naf.unsupported_version",
		},
		{
			name: "unknown viewpoint",
			edit: func(bundle *model.Bundle) { bundle.Architecture.NAF.Products[0].Viewpoint = "X99" },
			code: "naf.unknown_viewpoint",
		},
		{
			name: "dangling actor",
			edit: func(bundle *model.Bundle) { bundle.Architecture.NAF.Stakeholders[0].ActorRef = "ACT-MISSING" },
			code: "naf.invalid_stakeholder_actor_ref",
		},
		{
			name: "dangling concern",
			edit: func(bundle *model.Bundle) { bundle.Architecture.NAF.Products[0].ConcernRefs[0] = "NAF-CONCERN-MISSING" },
			code: "naf.invalid_product_concern_ref",
		},
		{
			name: "dangling view",
			edit: func(bundle *model.Bundle) { bundle.Architecture.NAF.Products[0].ViewRef = "VIEW-MISSING" },
			code: "naf.invalid_view_ref",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bundle, err := model.LoadBundle("engmod.yml")
			if err != nil {
				t.Fatal(err)
			}
			test.edit(&bundle)
			diagnostics := validate.Bundle(bundle)
			if !hasDiagnosticCode(diagnostics, test.code) {
				t.Fatalf("expected diagnostic %q, got %+v", test.code, diagnostics)
			}
		})
	}
}

func hasDiagnosticCode(diagnostics []validate.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
