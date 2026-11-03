// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"cuelang.org/go/mod/modregistrytest"
	engmodel "github.com/labeth/engineering-model-go"
)

// copyLinksExample copies examples/requirement-links (with engmod.work.yml,
// without generated output and locks) into a temporary directory.
func copyLinksExample(t *testing.T) string {
	t.Helper()
	t.Setenv(engmodel.WorkspaceEnv, "")
	t.Setenv(engmodel.LockEnv, "")
	t.Setenv(engmodel.ModuleCacheEnv, "")
	src := filepath.Join("..", "..", "examples", "requirement-links")
	root := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			if d.Name() == "generated" || d.Name() == ".engmod" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(root, rel), 0o755)
		}
		if strings.HasPrefix(d.Name(), "engmod.lock") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// publishLinksChildren publishes the fixture children of root (as edited
// so far) as v0.1.0 into a CUE test registry and routes models.example to
// it, so a workspace-off composition (links.review) resolves them.
func publishLinksChildren(t *testing.T, root string) {
	t.Helper()
	registryRoot := filepath.Join(t.TempDir(), "registry")
	for _, child := range []string{"child-a", "child-b"} {
		src := filepath.Join(root, "repos", child)
		dst := filepath.Join(registryRoot, "models.example_reqlinks_"+child+"_v0.1.0")
		err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(src, path)
			if d.IsDir() {
				return os.MkdirAll(filepath.Join(dst, rel), 0o755)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(registry.Close)
	t.Setenv("CUE_REGISTRY", "models.example="+registry.Host()+"+insecure")
}

// linksServer initializes a server on repos/<repo> of a copied fixture.
func linksServer(t *testing.T, root, repo string) *Server {
	t.Helper()
	dir := filepath.Join(root, "repos", repo)
	s := NewServer()
	if resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath": filepath.Join(dir, "engmod.yml"), "repoRoot": dir,
		}},
	}); resp["error"] != nil {
		t.Fatalf("initialize %s: %+v", repo, resp["error"])
	}
	return s
}

// payloadRow returns the requirement row with id of a trace.matrix payload.
func payloadRow(t *testing.T, payload map[string]any, id string) map[string]any {
	t.Helper()
	for _, item := range payload["requirements"].([]any) {
		if row := item.(map[string]any); row["id"] == id {
			return row
		}
	}
	t.Fatalf("no row %s", id)
	return nil
}

// strs converts a JSON array to strings.
func strs(v any) []string {
	var out []string
	list, _ := v.([]any)
	for _, x := range list {
		s, _ := x.(string)
		out = append(out, s)
	}
	return out
}

// trace.matrix returns parents, children and links per row and the link
// counts in the summary, also with summaryOnly.
// Mutation: dropping parents or children from the row map, or a link
// counter from the summary map, fails this test.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestTraceMatrixToolReportsRequirementLinks(t *testing.T) {
	root := copyLinksExample(t)
	s := linksServer(t, root, "system")
	for _, args := range []map[string]any{{}, {"summaryOnly": true}} {
		summary := callToolPayload(t, s, "trace.matrix", args)["summary"].(map[string]any)
		for _, key := range []string{"linksSuspect", "linksBroken", "linksClaimed", "unparented"} {
			if summary[key] != float64(0) {
				t.Fatalf("summary %v: %s = %v, want 0: %+v", args, key, summary[key], summary)
			}
		}
	}
	full := callToolPayload(t, s, "trace.matrix", map[string]any{})
	sys := payloadRow(t, full, "REQ-SYS-002")
	if got := strings.Join(strs(sys["children"]), ","); got != "encoder::CAP-A-2,encoder::REQ-A-003" {
		t.Fatalf("REQ-SYS-002 children %q: %+v", got, sys)
	}
	if links, _ := sys["links"].([]any); len(links) != 1 {
		t.Fatalf("REQ-SYS-002 links: %+v", sys)
	}
	if got := strs(payloadRow(t, full, "encoder::REQ-A-003")["parents"]); len(got) != 1 || got[0] != "encoder::CAP-A-2" {
		t.Fatalf("encoder::REQ-A-003 parents %v", got)
	}
}

// toolError calls a tool and returns its error message ("" when it succeeds).
func toolError(t *testing.T, s *Server, name string, args map[string]any) string {
	t.Helper()
	resp := rpcCall(t, s, map[string]any{"jsonrpc": "2.0", "id": name, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
	if resp["error"] != nil {
		return fmt.Sprint(resp["error"])
	}
	result, _ := resp["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); !isErr {
		return ""
	}
	content, _ := result["content"].([]any)
	return fmt.Sprint(content[0])
}

// linkByID returns the entry with id of a JSON array of objects.
func linkByID(t *testing.T, list any, id string) map[string]any {
	t.Helper()
	items, _ := list.([]any)
	for _, item := range items {
		if m := item.(map[string]any); m["id"] == id {
			return m
		}
	}
	t.Fatalf("no entry %s in %+v", id, list)
	return nil
}

// links.list lists every link with its state, the pinned and the
// dependency version and the recorded and current fingerprint;
// links.suspect lists only the suspect one with the item after the change
// (the before version is not resolvable offline here, so a note); a model
// without links returns empty lists.
// Mutation: dropping the state or the current fingerprint from links.list,
// listing an unchanged link in links.suspect or dropping the after item
// fails this test.
// TRLC-LINKS: REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestLinksListAndSuspectTools(t *testing.T) {
	root := copyLinksExample(t)
	editTestFile(t, filepath.Join(root, "repos", "child-a", "model", "requirements.yml"), "shall publish one position sentence.", "shall publish two position sentences.")
	s := linksServer(t, root, "system")
	list := callToolPayload(t, s, "links.list", map[string]any{})
	if links, _ := list["links"].([]any); len(links) != 4 {
		t.Fatalf("links.list: %+v", list)
	}
	suspect := linkByID(t, list["links"], "LNK-SYS-001-ENCODER")
	unchanged := linkByID(t, list["links"], "LNK-SYS-002-ENCODER")
	if suspect["state"] != "suspect" || suspect["pinnedVersion"] != "v0.1.0" || suspect["dependencyVersion"] != "v0.1.0" ||
		suspect["status"] != "accepted" || suspect["parent"] != "REQ-SYS-001" || suspect["child"] != "encoder::CAP-A-1" {
		t.Fatalf("suspect link: %+v", suspect)
	}
	if rec, cur := suspect["recordedFingerprint"], suspect["currentFingerprint"]; rec == cur || !strings.HasPrefix(fmt.Sprint(cur), "sha256:") {
		t.Fatalf("fingerprints recorded %v current %v", rec, cur)
	}
	if unchanged["state"] != "unchanged" || unchanged["recordedFingerprint"] != unchanged["currentFingerprint"] {
		t.Fatalf("unchanged link: %+v", unchanged)
	}

	review := callToolPayload(t, s, "links.suspect", map[string]any{})
	reviews, _ := review["links"].([]any)
	if len(reviews) != 1 {
		t.Fatalf("links.suspect: want one link, got %+v", review)
	}
	r := reviews[0].(map[string]any)
	after, _ := r["after"].(map[string]any)
	if r["id"] != "LNK-SYS-001-ENCODER" || r["state"] != "suspect" || after == nil || !strings.Contains(fmt.Sprint(after["realizers"]), "shall publish two position sentences.") ||
		r["parentText"] == "" || (r["before"] == nil && r["beforeNote"] == nil) {
		t.Fatalf("links.suspect entry: %+v", r)
	}

	child := linksServer(t, root, "child-a")
	if links, _ := callToolPayload(t, child, "links.list", map[string]any{})["links"].([]any); len(links) != 0 {
		t.Fatalf("child-a has no links: %+v", links)
	}
	if links, _ := callToolPayload(t, child, "links.suspect", map[string]any{})["links"].([]any); len(links) != 0 {
		t.Fatalf("child-a has no suspect links: %+v", links)
	}
}

// links.review writes nothing: it returns the engws review-links --accept
// command and the patch of the requirement link file the command would
// write (the new fingerprint, status, reviewer, the full reviewedOn line
// from the server clock, and rationale), computed like the command from the
// published dependency at its pin (workspace off); it refuses an actor the
// model does not define, an unknown link and a broken link.
// model.patch refuses the link file.
// Mutation: writing the link file, dropping the fingerprint line from the
// patch, accepting an unknown actor, using the wall clock instead of the
// server clock, reading the workspace checkout or letting model.patch edit
// the link file fails this test.
// TRLC-LINKS: REQ-EMG-106, REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestLinksReviewToolReturnsCommandAndPatch(t *testing.T) {
	root := copyLinksExample(t)
	childReqs := filepath.Join(root, "repos", "child-a", "model", "requirements.yml")
	// The published v0.1.0 changed the realizer (three sentences); the
	// workspace checkout differs again (two).
	editTestFile(t, childReqs, "shall publish one position sentence.", "shall publish three position sentences.")
	publishLinksChildren(t, root)
	editTestFile(t, childReqs, "shall publish three position sentences.", "shall publish two position sentences.")
	linksFile := filepath.Join(root, "repos", "system", "model", "req-links.yml")
	before, err := os.ReadFile(linksFile)
	if err != nil {
		t.Fatal(err)
	}
	s := linksServer(t, root, "system")
	s.clock = func() time.Time { return time.Date(2031, 4, 5, 6, 7, 8, 0, time.UTC) }
	workspace := linkByID(t, callToolPayload(t, s, "links.list", map[string]any{})["links"], "LNK-SYS-001-ENCODER")["currentFingerprint"].(string)
	got := callToolPayload(t, s, "links.review", map[string]any{"linkId": "LNK-SYS-001-ENCODER", "by": "ACT-SYSTEM-OWNER", "rationale": "Three sentences still give one position per epoch."})
	command, _ := got["command"].(string)
	patch, _ := got["patch"].(string)
	if !strings.Contains(command, "engws review-links") || !strings.Contains(command, "--accept LNK-SYS-001-ENCODER") || !strings.Contains(command, "--by ACT-SYSTEM-OWNER") ||
		!strings.Contains(command, "Three sentences still give one position per epoch.") {
		t.Fatalf("command %q", command)
	}
	if !strings.Contains(patch, "\n+    fingerprint: sha256:v1:") || strings.Contains(patch, "+    fingerprint: "+workspace) || !strings.Contains(patch, "-    fingerprint: sha256:v1:f1e64f2b") ||
		!strings.Contains(patch, "\n-  reviewedOn: \"2026-11-02\"\n+  reviewedOn: \"2031-04-05\"\n") ||
		!strings.Contains(patch, "+  rationale: Three sentences still give one position per epoch.") || got["written"] != false || got["path"] != "model/req-links.yml" || got["reviewedOn"] != "2031-04-05" {
		t.Fatalf("patch:\n%s\n%+v (workspace fingerprint %s)", patch, got, workspace)
	}
	if after, _ := os.ReadFile(linksFile); string(after) != string(before) {
		t.Fatal("links.review wrote the link file")
	}
	if msg := toolError(t, s, "links.review", map[string]any{"linkId": "LNK-SYS-001-ENCODER", "by": "ACT-NOBODY"}); !strings.Contains(msg, "ACT-NOBODY is not an actor") {
		t.Fatalf("unknown actor: %q", msg)
	}
	if msg := toolError(t, s, "links.review", map[string]any{"linkId": "LNK-NONE", "by": "ACT-SYSTEM-OWNER"}); !strings.Contains(msg, "no requirement link LNK-NONE") {
		t.Fatalf("unknown link: %q", msg)
	}
	// model.patch does not edit the link file either (D14, ADR-EMG-029).
	if msg := toolError(t, s, "model.patch", map[string]any{"document": "reqLinks", "id": "LNK-SYS-001-ENCODER", "write": true, "set": map[string]any{"status": "accepted"}}); !strings.Contains(msg, "engws review-links --accept") {
		t.Fatalf("model.patch on the link file: %q", msg)
	}
	if after, _ := os.ReadFile(linksFile); string(after) != string(before) {
		t.Fatal("model.patch wrote the link file")
	}
}

// requirements.parents and requirements.get show the upward trace of a
// requirement (its realizes targets) and, for a system requirement, the
// links it is the parent of with the linked child items and their
// realizing requirements.
// Mutation: dropping the realizes targets, the links or the child items
// from requirements.parents, or the parents or children from
// requirements.get, fails this test.
// TRLC-LINKS: REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestRequirementsParentsTool(t *testing.T) {
	root := copyLinksExample(t)
	sys := linksServer(t, root, "system")
	got := callToolPayload(t, sys, "requirements.parents", map[string]any{"requirementId": "REQ-SYS-001"})
	if len(strs(got["realizes"])) != 0 || got["exists"] != true {
		t.Fatalf("REQ-SYS-001 realizes nothing: %+v", got)
	}
	links, _ := got["links"].([]any)
	if len(links) != 2 {
		t.Fatalf("REQ-SYS-001 links: %+v", got)
	}
	enc := linkByID(t, got["links"], "LNK-SYS-001-ENCODER")
	item, _ := enc["item"].(map[string]any)
	if enc["state"] != "unchanged" || item == nil || item["id"] != "encoder::CAP-A-1" || !strings.Contains(fmt.Sprint(item["realizers"]), "encoder::REQ-A-002") {
		t.Fatalf("LNK-SYS-001-ENCODER: %+v", enc)
	}
	card := callToolPayload(t, sys, "requirements.get", map[string]any{"requirementId": "REQ-SYS-001"})
	if want := "bridge::CAP-B-1,bridge::REQ-B-001,encoder::CAP-A-1,encoder::REQ-A-001,encoder::REQ-A-002"; strings.Join(sortedStrs(card["children"]), ",") != want {
		t.Fatalf("requirements.get children %v, want %s", card["children"], want)
	}

	child := linksServer(t, root, "child-a")
	got = callToolPayload(t, child, "requirements.parents", map[string]any{"requirementId": "REQ-A-002"})
	if r := strs(got["realizes"]); len(r) != 1 || r[0] != "CAP-A-1" {
		t.Fatalf("REQ-A-002 realizes %v", got)
	}
	if links, _ := got["links"].([]any); len(links) != 0 {
		t.Fatalf("a child requirement is the parent of no link: %+v", got)
	}
	card = callToolPayload(t, child, "requirements.get", map[string]any{"requirementId": "REQ-A-002"})
	if p := strs(card["parents"]); len(p) != 1 || p[0] != "CAP-A-1" || len(strs(card["children"])) != 0 {
		t.Fatalf("requirements.get REQ-A-002: %+v", card)
	}
	if got := callToolPayload(t, child, "requirements.parents", map[string]any{"requirementId": "REQ-A-004"}); got["derived"] != true {
		t.Fatalf("REQ-A-004 is derived: %+v", got)
	}
}

// editTestFile replaces old (which must occur) with new in a file.
func editTestFile(t *testing.T, path, old, new string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// sortedStrs is strs, sorted.
func sortedStrs(v any) []string {
	out := strs(v)
	sort.Strings(out)
	return out
}
