// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel_test

import (
	"bufio"
	"bytes"
	"os/exec"
	"path"
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
	"internal/composition":         {"internal/model", "internal/validate", "internal/textutil"},
	"internal/codescan":            {"internal/composition", "internal/codemap", "internal/model", "internal/validate", "internal/textutil"},
	"internal/trace":               {"internal/codescan", "internal/composition", "internal/codemap", "internal/model", "internal/validate", "internal/textutil"},
	"internal/reqlint":             {"internal/model", "internal/validate", "internal/textutil"},
	"internal/reqdelta":            {"internal/reqlint", "internal/trace", "internal/codescan", "internal/composition", "internal/codemap", "internal/model", "internal/validate", "internal/textutil"},
	"internal/generate":            {"internal/codescan", "internal/composition", "internal/codemap", "internal/model", "internal/validate", "internal/textutil", "internal/view", "internal/render/*"},
	"internal/export/*": {"internal/export/*", "internal/codescan", "internal/composition", "internal/codemap", "internal/model",
		"internal/validate", "internal/textutil", "internal/view", "internal/sysmlmetamodel"},
	"internal/asciidoc": {"internal/export/*", "internal/generate", "internal/reqlint", "internal/trace", "internal/codescan",
		"internal/composition", "internal/codemap", "internal/model", "internal/validate", "internal/textutil", "internal/view",
		"internal/render/*"},
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
	out, err := exec.Command("go", "list", "-f", "{{.ImportPath}}|{{join .Imports \" \"}}|{{join .TestImports \" \"}} {{join .XTestImports \" \"}}", "./...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	checked := 0
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 3)
		pkg, ok := relPackage(parts[0])
		if !ok || strings.HasPrefix(pkg, "examples/") {
			continue
		}
		imports := inModule(parts[1])
		testImports := inModule(parts[2])
		if pkg == "" || strings.HasPrefix(pkg, "cmd/") {
			continue
		}
		checked++
		for _, imp := range imports {
			if _, top := topPackages[pkg]; imp == "" && !top {
				t.Errorf("%s imports the root facade; only cmd/*, internal/mcp and internal/engws may", pkg)
			}
		}
		// Tests may drive the public facade only in the end-to-end package.
		for _, imp := range testImports {
			if _, top := topPackages[pkg]; imp == "" && !top && pkg != "internal/integration" {
				t.Errorf("%s tests import the root facade; put end-to-end tests in internal/integration", pkg)
			}
		}
		if other, top := topPackages[pkg]; top {
			for _, imp := range imports {
				if imp == other || strings.HasPrefix(imp, "cmd/") {
					t.Errorf("%s must not import %s", pkg, imp)
				}
			}
			continue
		}
		allowed, ok := rulesFor(pkg)
		if !ok {
			t.Errorf("%s has no layering rule; add it to layerRules", pkg)
			continue
		}
		for _, imp := range imports {
			if imp == pkg || matchesAny(imp, allowed) {
				continue
			}
			t.Errorf("%s imports %s, which its layer does not allow (allowed: %s)", pkg, imp, strings.Join(allowed, ", "))
		}
	}
	if checked < len(layerRules) {
		t.Fatalf("checked %d packages, want at least %d", checked, len(layerRules))
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
