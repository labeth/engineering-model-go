// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// The facade is the only public API: an external caller loads and validates a
// model with engmodel.* alone, without importing any internal package.
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044
func TestFacadeLoadsAndValidatesWithoutInternalImports(t *testing.T) {
	bundle, err := engmodel.LoadBundle(filepath.Join("examples", "payments-engineering-sample", "engmod.yml"))
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	if len(bundle.Requirements.Requirements) == 0 {
		t.Fatal("LoadBundle returned no requirements")
	}
	var diags []engmodel.Diagnostic = engmodel.ValidateBundle(bundle)
	if engmodel.HasErrors(diags) {
		t.Fatalf("sample model has validation errors: %+v", diags)
	}

	broken := bundle
	broken.Requirements.Requirements = append([]engmodel.Requirement(nil), bundle.Requirements.Requirements...)
	broken.Requirements.Requirements[0].AppliesTo = []string{"FU-DOES-NOT-EXIST"}
	diags = engmodel.SortDiagnostics(engmodel.ValidateBundle(broken))
	if !engmodel.HasErrors(diags) {
		t.Fatal("ValidateBundle accepted a requirement that applies to an unknown ID")
	}
	found := false
	for _, d := range diags {
		if d.Code == "requirement.invalid_applies_to" && d.Severity == engmodel.SeverityError {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected requirement.invalid_applies_to error, got %+v", diags)
	}
}

// Every exported top-level name of the implementation packages is
// re-exported at the root, so splitting the implementation kept the public API
// whole. A package's rootName maps a de-stuttered package name to its public
// root name (trace.Matrix is engmodel.TraceMatrix); other names keep their
// name. internalOnly lists the names a package exports only for its sibling
// internal packages; they are not public API.
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044
func TestFacadeReexportsEngine(t *testing.T) {
	type implPackage struct {
		dir          string
		rootName     map[string]string
		internalOnly []string
	}
	packages := []implPackage{
		{dir: filepath.Join("internal", "engine")},
		{dir: filepath.Join("internal", "composition"), internalOnly: []string{
			// used by internal/engine and internal/trace.
			"ComposeForGeneration", "DelegationsByRequirement", "EnrichBundleFromComposition",
			"MaterializeAllocations", "PathWithin", "ResolutionCount",
		}},
		{dir: filepath.Join("internal", "codescan"), internalOnly: []string{
			// code, runtime and verification inference, used by internal/trace
			// and internal/engine; none of it is public API.
			"CodeItem", "EvidenceFileKey", "EvidenceLabel", "EvidenceLabelLookup",
			"GroupedEvidenceLabels", "GroupedFileLabels", "InferCodeItems",
			"InferRuntimeItems", "InferVerificationChecks", "IsVerificationCodeItem",
			"ItemDisplayName", "ItemEvidenceElement", "ItemPath", "JoinLineNumbers",
			"LanguageFromPath", "ModelRootDir", "NormalizeResultStatus",
			"ParseLineNumberList", "RuntimeItem", "SanitizeNode", "SanitizeSourcePath",
			"SkipScanDir", "SplitEvidencePathLines", "SplitSourceLine",
			"UniqueExistingSources", "VerificationCheck", "VerificationResult",
			"WithCodeIndex", "WithDefaultCodeRoot",
			// codescan.LoadBundle attaches the code index; the root LoadBundle is
			// model.LoadBundle, a different function.
			"LoadBundle",
		}},
		{dir: filepath.Join("internal", "trace"), rootName: map[string]string{
			"Matrix": "TraceMatrix", "Row": "TraceRow", "Provenance": "TraceProvenance",
			"DelegatedStatus": "TraceDelegatedStatus", "CodeRef": "TraceCodeRef",
			"Delegation": "TraceDelegation", "Summary": "TraceSummary", "Options": "TraceOptions",
			"BuildFromFiles": "BuildTraceMatrixFromFiles", "BuildFromFilesWith": "BuildTraceMatrixFromFilesWith",
		}, internalOnly: []string{
			// used by internal/engine (documents, requirements delta).
			"EffectiveCodeRoots", "KnownModelIDs", "ScopeCodeToModel", "ValidateIntegrity",
		}},
	}
	root := exportedNames(t, ".")
	for _, pkg := range packages {
		impl := exportedNames(t, pkg.dir)
		if len(impl) == 0 {
			t.Fatalf("found no exported names in %s", pkg.dir)
		}
		internal := map[string]bool{}
		for _, name := range pkg.internalOnly {
			internal[name] = true
			if !impl[name] {
				t.Errorf("%s lists %s as internal-only but does not export it", pkg.dir, name)
			}
			if root[name] && name != "LoadBundle" {
				t.Errorf("%s is listed as internal-only but the root facade exports it", name)
			}
		}
		for name := range impl {
			if internal[name] {
				continue
			}
			public := name
			if renamed, ok := pkg.rootName[name]; ok {
				public = renamed
			}
			if !root[public] {
				t.Errorf("%s exports %s but the root facade does not export %s", pkg.dir, name, public)
			}
		}
	}
}

func exportedNames(t *testing.T, dir string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", dir, err)
	}
	names := map[string]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Recv == nil && d.Name.IsExported() {
						names[d.Name.Name] = true
					}
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						switch s := spec.(type) {
						case *ast.TypeSpec:
							if s.Name.IsExported() {
								names[s.Name.Name] = true
							}
						case *ast.ValueSpec:
							for _, n := range s.Names {
								if n.IsExported() {
									names[n.Name] = true
								}
							}
						}
					}
				}
			}
		}
	}
	return names
}
