// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engine

// Composition tests that also exercise the generators and the trace; the
// composition-only tests live in internal/composition.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

const confinementSecret = "TOP-SECRET-CREDENTIAL-7f3a"

// TRLC-LINKS: REQ-EMG-049
func assertNoSecret(t *testing.T, diags []validate.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if strings.Contains(d.Message, confinementSecret) || strings.Contains(d.Path, confinementSecret) {
			t.Fatalf("diagnostic leaks the content of a file outside the module: %+v", d)
		}
	}
}

// TRLC-LINKS: REQ-EMG-049
func escapeDiagnostics(diags []validate.Diagnostic) []validate.Diagnostic {
	var out []validate.Diagnostic
	for _, d := range diags {
		if d.Code == "composition.dependency_path_escape" {
			out = append(out, d)
		}
	}
	return out
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

// A published module whose manifest points a document outside the module
// (documents.catalog: ../../../../secret.yml) or names an absolute code
// source is rejected before any of its documents is read: one
// composition.dependency_path_escape per field naming the alias and field,
// and the content of the target file appears in no diagnostic.
// Mutation: dropping the dependency manifest confinement check in
// resolveDependency fails this test.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, REF-CUE-OCI-MODULE-REGISTRY
func TestRegistryDependencyPathsAreConfinedToTheModule(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, "")
	t.Setenv(composition.ModuleCacheEnv, "")
	root := t.TempDir()
	// workspace/.engmod/modules/<entry>/../../../../secret.yml is root/secret.yml.
	testutil.WriteFile(t, filepath.Join(root, "secret.yml"), "schemaVersion: 2\ncatalog:\n  "+confinementSecret+": leaked\n")
	registryRoot := filepath.Join(root, "registry")
	moduleDir := filepath.Join(registryRoot, "models.atlas.example_shared-edge_v0.1.0")
	testutil.WriteOCIModuleFixture(t, moduleDir, "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	manifest := filepath.Join(moduleDir, "engmod.yml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	escaped := strings.Replace(string(data), "catalog: model/catalog.yml", "catalog: ../../../../secret.yml", 1)
	escaped += "inferenceHints:\n  exclude:\n    - ../../../../**\n"
	testutil.WriteFile(t, manifest, escaped)
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatalf("start registry: %v", err)
	}
	defer registry.Close()
	t.Setenv("CUE_REGISTRY", registry.Host()+"+insecure")

	workspace := filepath.Join(root, "workspace")
	testutil.WriteParentWithOCIModule(t, workspace, "models.atlas.example/shared-edge@v0", "v0.1.0")
	res, err := composition.GenerateCompositionFromFile(filepath.Join(workspace, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecret(t, res.Diagnostics)
	got := escapeDiagnostics(res.Diagnostics)
	var fields []string
	for _, d := range got {
		if !strings.Contains(d.Message, `"shared"`) || d.Severity != validate.SeverityError {
			t.Fatalf("escape diagnostic must be an error naming the alias: %+v", d)
		}
		fields = append(fields, d.Message)
	}
	joined := strings.Join(fields, "\n")
	if len(got) != 2 || !strings.Contains(joined, "documents.catalog") || !strings.Contains(joined, "inferenceHints.exclude[0]") {
		t.Fatalf("want documents.catalog and inferenceHints.exclude[0] escapes, got %+v", res.Diagnostics)
	}
	if res.Root != nil && len(res.Root.Children) != 0 {
		t.Fatal("a dependency with escaping paths must not be loaded")
	}
	// The trace never reads the rejected dependency either.
	_, traceDiags, err := BuildTraceMatrixFromFiles(filepath.Join(workspace, "engmod.yml"), filepath.Join(workspace, "model", "requirements.yml"), "")
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecret(t, traceDiags)
}
