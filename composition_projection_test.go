// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
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

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func TestAtlasPublicationEnrichesNativeOutputs(t *testing.T) {
	manifest := filepath.Join("examples", "atlas-industries", "repos", "aegis-sentinel", "engmod.yml")
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}

	asciidoc, err := GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, AsciiDocOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Selected Dependency Publications", "security::CTRL-SEC-SHORT-LIVED-IDENTITY", "models.atlas.example/shared/security-services@v0"} {
		if !strings.Contains(asciidoc.Document, expected) {
			t.Fatalf("AsciiDoc missing %q", expected)
		}
	}
	if strings.Contains(asciidoc.Document, "security::INTERNAL") {
		t.Fatal("AsciiDoc exposed unpublished dependency data")
	}

	trace, _, err := BuildTraceMatrixFromFiles(manifest, bundle.RequirementsPath, "")
	if err != nil {
		t.Fatal(err)
	}
	foundRequirement := false
	for _, row := range trace.Requirements {
		if row.ID == "security::REQ-SEC-001" {
			foundRequirement = row.Provenance != nil && row.Provenance.Publication == "public"
		}
	}
	if !foundRequirement {
		t.Fatal("trace matrix omitted qualified published requirement provenance")
	}

	sysml, err := GenerateSysMLV2FromFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sysml.Text, "package 'security'") ||
		!strings.Contains(sysml.Text, "requirement def 'REQ-SEC-001'") {
		t.Fatalf("SysML omitted native published declarations:\n%s", sysml.Text)
	}
	for _, forbidden := range []string{"payload =", "json", "properties {"} {
		if strings.Contains(strings.ToLower(sysml.Text), forbidden) {
			t.Fatalf("SysML contains forbidden opaque representation %q", forbidden)
		}
	}

	structurizr, err := GenerateStructurizrDSLFromFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(structurizr.DSL, "security::FU-SEC-IDENTITY") {
		t.Fatal("Structurizr omitted published structural entity")
	}
	if strings.Contains(structurizr.DSL, "properties {") {
		t.Fatal("Structurizr output contains a generic property bag")
	}

	poam, err := GenerateOSCALPOAMFromFile(manifest, OSCALPOAMOptions{LastModified: "2026-09-19T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(poam.JSON, "compliance::POAM-COMP-001") {
		t.Fatal("OSCAL omitted published dependency POA&M item")
	}

	gemara, err := GenerateGemaraFromFile(manifest, GemaraExportOptions{Date: "2026-09-19T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gemara.YAML["control-catalog"], "security::CTRL-SEC-SHORT-LIVED-IDENTITY") {
		t.Fatal("Gemara omitted published dependency control")
	}

	threats, err := GenerateThreatModelExportFromFile(manifest, ThreatModelExportOptions{Format: ThreatModelFormatOpenOTM})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(threats.JSON, "security::TS-SEC-CREDENTIAL-REPLAY") {
		t.Fatal("threat model omitted natively representable published threat")
	}
}
