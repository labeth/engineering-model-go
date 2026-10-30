// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func TestCompositionProjectionQualifiesCollidingPublishedIDsWithoutMutatingSources(t *testing.T) {
	left := model.Bundle{Manifest: model.ManifestDocument{
		Module:       model.ModuleIdentity{Path: "example.com/left@v1", Version: "v1.0.0", ModelID: "LEFT"},
		Publications: []model.ManifestPublication{{ID: "public", Architecture: []string{"FU-SHARED"}}},
	}}
	left.Architecture.AuthoredArchitecture.FunctionalUnits = []model.FunctionalUnit{
		{ID: "FU-SHARED", Name: "Published"},
		{ID: "FU-INTERNAL", Name: "Internal"},
	}
	right := left
	right.Manifest.Module = model.ModuleIdentity{Path: "example.com/right@v1", Version: "v1.0.0", ModelID: "RIGHT"}

	result := CompositionResult{Root: &ComposedSystem{Children: []*ComposedSystem{
		{Dependency: "left", Publication: "public", ModulePath: left.Manifest.Module.Path, Version: left.Manifest.Module.Version, Bundle: left, PublishedFacet: map[string]string{"FU-SHARED": "architecture"}},
		{Dependency: "right", Publication: "public", ModulePath: right.Manifest.Module.Path, Version: right.Manifest.Module.Version, Bundle: right, PublishedFacet: map[string]string{"FU-SHARED": "architecture"}},
	}}}
	result.Provenance = append(
		publicationProvenance("left", model.ManifestDependency{Alias: "left", Path: left.Manifest.Module.Path, Version: left.Manifest.Module.Version}, result.Root.Children[0], left.Manifest.Publications[0], ""),
		publicationProvenance("right", model.ManifestDependency{Alias: "right", Path: right.Manifest.Module.Path, Version: right.Manifest.Module.Version}, result.Root.Children[1], right.Manifest.Publications[0], "")...,
	)

	projection := BuildCompositionProjection(result)
	if len(projection.Entities) != 2 ||
		projection.Entities[0].QualifiedID != "left::FU-SHARED" ||
		projection.Entities[1].QualifiedID != "right::FU-SHARED" {
		t.Fatalf("unexpected qualified projection: %+v", projection.Entities)
	}
	for _, entity := range projection.Entities {
		if entity.SourceID == "FU-INTERNAL" {
			t.Fatal("unpublished entity leaked into projection")
		}
	}
	if left.Architecture.AuthoredArchitecture.FunctionalUnits[0].ID != "FU-SHARED" {
		t.Fatal("projection mutated the authored source bundle")
	}
}
