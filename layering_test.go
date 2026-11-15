// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel_test

import (
	"bufio"
	"bytes"
	"os/exec"
	"path"
	"slices"
	"sort"
	"strings"
	"testing"
)

const modulePath = "github.com/labeth/engineering-model-go"

// layerRules lists, for every package of the module except the root and
// cmd/*, the in-module packages it may import directly (production code).
// A pattern ending in /* matches the packages one level below it. Imports
// point down the layers only: textutil and model import nothing in-module,
// validate builds on model, the scanners, views and renderers on those, the
// composition, code scan and trace on them, then requirement lint, the
// requirements delta, view generation, the exporters and the AsciiDoc
// publication; mcp and engws sit on top and may use the root facade.
var layerRules = map[string][]string{
	"internal/textutil":            {},
	"internal/model":               {},
	"internal/sysmlmetamodel":      {},
	"internal/render/diagramstyle": {},
	"internal/validate":            {"internal/model"},
	"internal/cliflags":            {"internal/model"},
	"internal/testutil":            {"internal/model", "internal/validate"},
	"internal/codemap":             {"internal/model", "internal/validate"},
	"internal/view":                {"internal/model", "internal/validate"},
	"internal/render/*":            {"internal/view", "internal/render/diagramstyle"},
	"internal/reqlinks":            {"internal/model", "internal/textutil"},
	"internal/yamledit":            {},
	"internal/gitmodel":            {"internal/model"},
	"internal/composition":         {"internal/model", "internal/validate", "internal/textutil", "internal/reqlinks"},
	"internal/codescan":            {"internal/composition", "internal/codemap", "internal/model", "internal/validate", "internal/textutil"},
	"internal/trace":               {"internal/codescan", "internal/composition", "internal/reqlinks", "internal/codemap", "internal/model", "internal/validate", "internal/textutil"},
	"internal/linkreview":          {"internal/composition", "internal/reqlinks", "internal/yamledit", "internal/model", "internal/validate"},
	"internal/reqlint":             {"internal/model", "internal/validate", "internal/textutil"},
	"internal/reqdelta":            {"internal/reqlint", "internal/trace", "internal/codescan", "internal/composition", "internal/codemap", "internal/model", "internal/validate", "internal/textutil"},
	"internal/generate":            {"internal/codescan", "internal/composition", "internal/codemap", "internal/model", "internal/validate", "internal/textutil", "internal/view", "internal/render/*"},
	"internal/export/*": {"internal/export/*", "internal/codescan", "internal/composition", "internal/codemap", "internal/model",
		"internal/validate", "internal/textutil", "internal/view", "internal/sysmlmetamodel"},
	// The Quint check runner builds on the Quint generator and the shared tool
	// lookup only; it reads no trace or scan package (the evidence file is the
	// contract with internal/codescan, which must not import internal/export).
	"internal/export/quint/check": {"internal/export/quint", "internal/export/exportutil", "internal/model"},
	"internal/asciidoc": {"internal/export/*", "internal/generate", "internal/reqlint", "internal/trace", "internal/reqlinks", "internal/codescan",
		"internal/composition", "internal/codemap", "internal/model", "internal/validate", "internal/textutil", "internal/view",
		"internal/render/*"},
	// The code health data model and the engines behind it: the engines read
	// source through codemap and fill the types of internal/health; internal/health
	// itself never imports an engine.
	"internal/health":                   {"internal/codescan", "internal/codemap", "internal/model", "internal/validate", "internal/textutil"},
	"internal/health/engine/treesitter": {"internal/health", "internal/codescan", "internal/codemap", "internal/model", "internal/validate", "internal/textutil"},
	// The scan pipeline picks and constructs the engine, so it sits above both:
	// it is the one package that imports the tree-sitter engine and the code
	// scope of the other tools.
	"internal/health/scan": {"internal/health", "internal/health/engine/treesitter", "internal/codescan", "internal/codemap", "internal/model", "internal/validate", "internal/textutil"},
	// Test-only package for cross-package end-to-end tests; its tests may
	// also drive the public facade and the MCP server.
	"internal/integration": {},
}

// The top packages may import the root facade and any internal package
// except each other.
var topPackages = map[string]string{"internal/mcp": "internal/engws", "internal/engws": "internal/mcp"}

// Every internal package imports only packages below it, and only cmd/*,
// internal/mcp and internal/engws import the root facade; among tests only
// internal/integration (end-to-end) may use it.
// TRLC-LINKS: REQ-EMG-001
func TestPackageLayering(t *testing.T) {
	pkgs := listLayerPackages(t)
	for _, p := range pkgs {
		checkPackageLayering(t, p)
	}
	if len(pkgs) < len(layerRules) {
		t.Fatalf("checked %d packages, want at least %d", len(pkgs), len(layerRules))
	}
}

// layerPackage is a package of the module with the in-module packages its
// production code and its tests import, relative to the module root.
type layerPackage struct {
	name        string
	imports     []string
	testImports []string
}

// listLayerPackages lists the packages of the module the layering rules cover:
// every package except the root, cmd/* and examples/*.
func listLayerPackages(t *testing.T) []layerPackage {
	t.Helper()
	out, err := exec.Command("go", "list", "-f", "{{.ImportPath}}|{{join .Imports \" \"}}|{{join .TestImports \" \"}} {{join .XTestImports \" \"}}", "./...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	var pkgs []layerPackage
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 3)
		pkg, ok := relPackage(parts[0])
		if !ok || !coveredByLayering(pkg) {
			continue
		}
		pkgs = append(pkgs, layerPackage{name: pkg, imports: inModule(parts[1]), testImports: inModule(parts[2])})
	}
	return pkgs
}

// coveredByLayering reports whether the layering rules cover a package of the
// module: every package except the root, cmd/* and examples/*.
func coveredByLayering(pkg string) bool {
	return pkg != "" && !strings.HasPrefix(pkg, "cmd/") && !strings.HasPrefix(pkg, "examples/")
}

// checkPackageLayering reports every import of p that its layer does not allow.
func checkPackageLayering(t *testing.T, p layerPackage) {
	t.Helper()
	checkRootFacadeImports(t, p)
	if other, top := topPackages[p.name]; top {
		checkTopPackageImports(t, p, other)
		return
	}
	checkLayerImports(t, p)
}

// checkRootFacadeImports reports a package that imports the root facade
// ("" relative to the module) although it is not a top package, and tests
// that drive the facade outside the end-to-end package.
func checkRootFacadeImports(t *testing.T, p layerPackage) {
	t.Helper()
	if _, top := topPackages[p.name]; top {
		return
	}
	if slices.Contains(p.imports, "") {
		t.Errorf("%s imports the root facade; only cmd/*, internal/mcp and internal/engws may", p.name)
	}
	// Tests may drive the public facade only in the end-to-end package.
	if slices.Contains(p.testImports, "") && p.name != "internal/integration" {
		t.Errorf("%s tests import the root facade; put end-to-end tests in internal/integration", p.name)
	}
}

// checkTopPackageImports reports a top package that imports its sibling top
// package or a command.
func checkTopPackageImports(t *testing.T, p layerPackage, other string) {
	t.Helper()
	for _, imp := range p.imports {
		if imp == other || strings.HasPrefix(imp, "cmd/") {
			t.Errorf("%s must not import %s", p.name, imp)
		}
	}
}

// checkLayerImports reports the imports of p that layerRules does not allow.
func checkLayerImports(t *testing.T, p layerPackage) {
	t.Helper()
	allowed, ok := rulesFor(p.name)
	if !ok {
		t.Errorf("%s has no layering rule; add it to layerRules", p.name)
		return
	}
	for _, imp := range p.imports {
		if imp == p.name || matchesAny(imp, allowed) {
			continue
		}
		t.Errorf("%s imports %s, which its layer does not allow (allowed: %s)", p.name, imp, strings.Join(allowed, ", "))
	}
}

// relPackage returns an import path relative to the module ("" for the root).
func relPackage(importPath string) (string, bool) {
	if importPath == modulePath {
		return "", true
	}
	rel, ok := strings.CutPrefix(importPath, modulePath+"/")
	return rel, ok
}

func inModule(list string) []string {
	var rels []string
	seen := map[string]bool{}
	for _, imp := range strings.Fields(list) {
		if rel, ok := relPackage(imp); ok && !seen[rel] {
			seen[rel] = true
			rels = append(rels, rel)
		}
	}
	sort.Strings(rels)
	return rels
}

func rulesFor(pkg string) ([]string, bool) {
	if allowed, ok := layerRules[pkg]; ok {
		return allowed, true
	}
	allowed, ok := layerRules[path.Dir(pkg)+"/*"]
	return allowed, ok
}

func matchesAny(pkg string, patterns []string) bool {
	for _, p := range patterns {
		if p == pkg || (strings.HasSuffix(p, "/*") && path.Dir(pkg)+"/*" == p) {
			return true
		}
	}
	return false
}
