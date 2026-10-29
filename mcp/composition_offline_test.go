// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"cuelang.org/go/mod/modregistrytest"
	engmodel "github.com/labeth/engineering-model-go"
)

// countingRegistry publishes example.com/child@v1 v1.0.0 behind a proxy that
// counts registry requests, and points CUE_REGISTRY at the proxy.
// TRLC-LINKS: REQ-EMG-069
func countingRegistry(t *testing.T) *atomic.Int64 {
	t.Helper()
	registryRoot := filepath.Join(t.TempDir(), "registry")
	writeMCPModule(t, filepath.Join(registryRoot, "example.com_child_v1.0.0"), "example.com/child@v1", "v1.0.0", "CHILD", `publications:
  - id: public
    architecture: [CAP-CHILD]
`, `  contract:
    provides:
      - id: CAP-CHILD
        kind: capability
        ref: REQ-CHILD
`)
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(registry.Close)
	target, err := url.Parse("http://" + registry.Host())
	if err != nil {
		t.Fatal(err)
	}
	var count atomic.Int64
	proxy := httputil.NewSingleHostReverseProxy(target)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CUE_REGISTRY", strings.TrimPrefix(srv.URL, "http://")+"+insecure")
	return &count
}

// The composing MCP read tools (model.complete, model.diagnose,
// composition.resolve, trace.matrix) never write engmod.lock.yml and work
// offline from a warm module cache: no registry request, also with
// ENGMOD_WORK=off. resolvedDirectory of a cached dependency is
// module@version, never an absolute path.
// Mutation: resolving with the default options (lock write, cache refresh
// with the workspace off) in the MCP server, or returning c.Dir as
// resolvedDirectory, fails this test.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-064
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY
func TestMCPCompositionToolsNeverWriteLockAndWorkOffline(t *testing.T) {
	t.Setenv("ENGMOD_WORK", "")
	t.Setenv("ENGMOD_LOCK", "")
	t.Setenv("ENGMOD_MODULE_CACHE", "")
	requests := countingRegistry(t)
	parent := filepath.Join(t.TempDir(), "parent")
	writeMCPModule(t, parent, "example.com/parent@v1", "v1.0.0", "PARENT", `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.0.0
    publications: [public]
`, `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
    allocations:
      - requirement: REQ-PARENT
        to: SUB-CHILD
        target: child::CAP-CHILD
`)
	manifest := filepath.Join(parent, "engmod.yml")
	lockPath := filepath.Join(parent, engmodel.CompositionLockFileName)
	// Warm the module cache without writing a lock.
	if _, err := engmodel.GenerateCompositionFromFileWith(manifest, engmodel.ResolutionOptions{Lock: engmodel.LockModeCheck}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(lockPath); err == nil {
		t.Fatal("warm-up wrote the lock")
	}
	requirements, err := os.ReadFile(filepath.Join(parent, "model", "requirements.yml"))
	if err != nil {
		t.Fatal(err)
	}
	s := initializedServerAt(t, parent)
	callAll := func(label string) {
		t.Helper()
		requests.Store(0)
		complete := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "child::"})
		if !strings.Contains(mustJSON(t, complete), "child::CAP-CHILD") {
			t.Fatalf("%s: model.complete lost the imported IDs: %+v", label, complete)
		}
		callToolPayload(t, s, "model.diagnose", map[string]any{"document": "requirements", "content": string(requirements)})
		comp := callToolData(t, s, "composition.resolve")
		text := mustJSON(t, comp)
		if strings.Contains(text, "module_fetch_failed") {
			t.Fatalf("%s: composition.resolve fetched: %s", label, text)
		}
		subs, _ := comp["subsystems"].([]any)
		if len(subs) != 1 || subs[0].(map[string]any)["resolvedDirectory"] != "example.com/child@v1.0.0" {
			t.Fatalf("%s: resolvedDirectory must be module@version: %s", label, text)
		}
		for _, p := range comp["provenance"].([]any) {
			if dir := p.(map[string]any)["ResolvedDir"]; dir != nil && dir != "example.com/child@v1.0.0" {
				t.Fatalf("%s: provenance directory %v", label, dir)
			}
		}
		if strings.Contains(text, filepath.Join(parent, ".engmod")) {
			t.Fatalf("%s: composition.resolve returns the cache path: %s", label, text)
		}
		callToolData(t, s, "trace.matrix")
		if n := requests.Load(); n != 0 {
			t.Fatalf("%s: %d registry requests with a warm cache", label, n)
		}
	}
	callAll("no lock")
	if _, err := os.Stat(lockPath); err == nil {
		t.Fatal("an MCP read tool created engmod.lock.yml")
	}
	committed := "schemaVersion: 1\ndependencies: []\n# committed\n"
	if err := os.WriteFile(lockPath, []byte(committed), 0o644); err != nil {
		t.Fatal(err)
	}
	callAll("committed lock")
	t.Setenv("ENGMOD_WORK", "off")
	callAll("workspace off")
	if data, _ := os.ReadFile(lockPath); string(data) != committed {
		t.Fatalf("an MCP read tool changed engmod.lock.yml:\n%s", data)
	}
}

// A workspace replacement's resolvedDirectory is relative to the workspace
// root.
// Mutation: returning the absolute c.Dir fails this test.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func TestCompositionResolveReturnsWorkspaceRelativeDirectory(t *testing.T) {
	t.Setenv("ENGMOD_WORK", "")
	t.Setenv("ENGMOD_LOCK", "")
	t.Setenv("CUE_REGISTRY", "none")
	parent := writeMCPCompositionFixture(t)
	s := initializedServerAt(t, parent)
	comp := callToolData(t, s, "composition.resolve")
	subs, _ := comp["subsystems"].([]any)
	if len(subs) != 1 || subs[0].(map[string]any)["resolvedDirectory"] != "child" {
		t.Fatalf("resolvedDirectory must be relative to the workspace root: %s", mustJSON(t, comp))
	}
	for _, p := range comp["provenance"].([]any) {
		if dir := p.(map[string]any)["ResolvedDir"]; dir != "child" {
			t.Fatalf("provenance directory %v", dir)
		}
	}
}

// TRLC-LINKS: REQ-EMG-069
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// No tool output names an absolute path: the workspace root (here the parent
// of the project and its sibling) and the project root are rewritten, also in
// diagnostics such as composition.lock_check_skipped that carry file paths.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func TestToolOutputCarriesNoAbsolutePaths(t *testing.T) {
	t.Setenv("ENGMOD_WORK", "")
	t.Setenv("ENGMOD_LOCK", "")
	t.Setenv("CUE_REGISTRY", "none")
	parent := writeMCPCompositionFixture(t)
	workspace := filepath.Dir(parent)
	s := initializedServerAt(t, parent)
	for _, tool := range []string{"composition.resolve", "trace.matrix"} {
		text := mustJSON(t, callToolData(t, s, tool))
		for _, abs := range []string{workspace, evalOrSame(workspace)} {
			if strings.Contains(text, abs) {
				t.Fatalf("%s leaks absolute path %s:\n%s", tool, abs, text)
			}
		}
	}
	// A sibling checkout path (as a diagnostic from a dependency would carry)
	// is named relative to the workspace root found from the model.
	sibling := filepath.Join(workspace, "child", "model", "architecture.yml")
	if got := newPathScrubber(s.repoRoot, s.modelPath).scrubString("in " + sibling); got != "in workspace:child/model/architecture.yml" {
		t.Fatalf("sibling path scrubbed to %q", got)
	}
	comp := callToolData(t, s, "composition.resolve")
	found := false
	for _, d := range comp["diagnostics"].([]any) {
		if d.(map[string]any)["code"] == "composition.lock_check_skipped" {
			found = true
			if path := d.(map[string]any)["path"]; path != "engmod.lock.yml" {
				t.Fatalf("diagnostic path must be project-relative, got %v", path)
			}
		}
	}
	if !found {
		t.Fatalf("fixture must raise composition.lock_check_skipped: %s", mustJSON(t, comp))
	}
}

// TRLC-LINKS: REQ-EMG-069
func TestPathScrubberPrefixes(t *testing.T) {
	home, _ := os.UserHomeDir()
	p := pathScrubber{prefixes: []scrubPrefix{{from: "/w/fam/proj", to: ""}, {from: "/w/fam", to: "workspace:"}, {from: home, to: "~"}}}
	for in, want := range map[string]string{
		"/w/fam/proj/engmod.lock.yml":     "engmod.lock.yml",
		"/w/fam/proj":                     ".",
		"/w/fam/sibling/model/a.yml":      "workspace:sibling/model/a.yml",
		home + "/go/pkg/mod/x":            "~/go/pkg/mod/x",
		"see (/w/fam/proj/model/r.yml:3)": "see (model/r.yml:3)",
	} {
		if got := p.scrubString(in); got != want {
			t.Fatalf("scrub %q = %q, want %q", in, got, want)
		}
	}
}
