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

// Every exported top-level name of internal/engine is re-exported at the root
// under the same name, so the implementation move kept the public API whole.
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044
func TestFacadeReexportsEngine(t *testing.T) {
	engine := exportedNames(t, filepath.Join("internal", "engine"))
	root := exportedNames(t, ".")
	if len(engine) == 0 {
		t.Fatal("found no exported names in internal/engine")
	}
	for name := range engine {
		if !root[name] {
			t.Errorf("internal/engine exports %s but the root facade does not", name)
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
