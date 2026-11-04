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

// registryRoots keeps one registry directory per test, so each
// publishLinksChildren call adds versions to what the test published before.
var registryRoots = map[string]string{}

// publishLinksChildren publishes the fixture children of root (as edited
// so far) at each version into the test's CUE test registry (versions
// published earlier stay) and routes models.example to it, so a
// workspace-off composition resolves them.
func publishLinksChildren(t *testing.T, root string, versions ...string) {
	t.Helper()
	registryRoot, ok := registryRoots[t.Name()]
	if !ok {
		registryRoot = filepath.Join(t.TempDir(), "registry")
		registryRoots[t.Name()] = registryRoot
		t.Cleanup(func() { delete(registryRoots, t.Name()) })
	}
	for _, version := range versions {
		for _, child := range []string{"child-a", "child-b"} {
			src := filepath.Join(root, "repos", child)
			dst := filepath.Join(registryRoot, "models.example_reqlinks_"+child+"_"+version)
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
				if rel == "engmod.yml" {
					data = []byte(strings.Replace(string(data), "  version: v0.1.0\n", "  version: "+version+"\n", 1))
				}
				return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
			})
			if err != nil {
				t.Fatal(err)
			}
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
		for _, key := range []string{"linksNeedReview", "linksVersionMismatch", "linksSuspect", "linksBroken", "linksClaimed", "unparented"} {
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

// links.list lists every link with its state, review status, needsReview
// and the pinned and dependency version; in a workspace a changed child
// checkout is not compared (D23), so every link stays unchanged.
// links.suspect lists the links that need review against the published
// dependencies (workspace off): a stale pin with the item before and after
// and what changed, and a claimed link; a model without links returns
// empty lists.
// Mutation: comparing workspace content, dropping needsReview, listing an
// accepted link at its pin in links.suspect or dropping the change list
// fails this test.
// TRLC-LINKS: REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestLinksListAndSuspectTools(t *testing.T) {
	root := copyLinksExample(t)
	childReqs := filepath.Join(root, "repos", "child-a", "model", "requirements.yml")
	publishLinksChildren(t, root, "v0.1.0")
	editTestFile(t, childReqs, "shall publish one position sentence.", "shall publish two position sentences.")
	publishLinksChildren(t, root, "v0.2.0")
	s := linksServer(t, root, "system")
	list := callToolPayload(t, s, "links.list", map[string]any{})
	if links, _ := list["links"].([]any); len(links) != 4 {
		t.Fatalf("links.list: %+v", list)
	}
	for _, id := range []string{"LNK-SYS-001-ENCODER", "LNK-SYS-002-ENCODER"} {
		if l := linkByID(t, list["links"], id); l["state"] != "unchanged" || l["needsReview"] != false || l["local"] != true {
			t.Fatalf("a workspace edit changed link %s: %+v", id, l)
		}
	}
	// In the workspace links.suspect still reviews against the published
	// v0.1.0 (one sentence), not the edited checkout (two).
	linksFile := filepath.Join(root, "repos", "system", "model", "req-links.yml")
	claimedEncoder := "  status: claimed\n  rationale: The encoder publishes one position sentence per epoch.\n"
	acceptedEncoder := "  status: accepted\n  rationale: The encoder publishes one position sentence per epoch.\n  reviewedBy: ACT-SYSTEM-OWNER\n  reviewedOn: \"2026-11-02\"\n"
	editTestFile(t, linksFile, acceptedEncoder, claimedEncoder)
	inWorkspace, _ := callToolPayload(t, s, "links.suspect", map[string]any{})["links"].([]any)
	if len(inWorkspace) != 1 || !strings.Contains(fmt.Sprint(inWorkspace[0].(map[string]any)["after"]), "shall publish one position sentence.") {
		t.Fatalf("links.suspect in the workspace must show the published v0.1.0: %+v", inWorkspace)
	}
	editTestFile(t, linksFile, claimedEncoder, acceptedEncoder)
	// Outside the workspace the system moves to v0.2.0 (published with two
	// sentences); one link stays reviewed at v0.1.0 and one becomes claimed.
	if err := os.Remove(filepath.Join(root, "engmod.work.yml")); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "repos", "system", "engmod.yml")
	editTestFile(t, manifest, "  path: models.example/reqlinks/child-a@v0\n  version: v0.1.0\n", "  path: models.example/reqlinks/child-a@v0\n  version: v0.2.0\n")
	editTestFile(t, linksFile, "- id: LNK-SYS-002-ENCODER\n  parent: REQ-SYS-002\n  child: encoder::CAP-A-2\n  pinned:\n    version: v0.1.0\n", "- id: LNK-SYS-002-ENCODER\n  parent: REQ-SYS-002\n  child: encoder::CAP-A-2\n  pinned:\n    version: v0.2.0\n")
	editTestFile(t, linksFile, "  status: accepted\n  rationale: The bridge forwards each sentence onto the lab bus.\n  reviewedBy: ACT-SYSTEM-OWNER\n  reviewedOn: \"2026-11-02\"\n", "  status: claimed\n")
	list = callToolPayload(t, s, "links.list", map[string]any{})
	stale := linkByID(t, list["links"], "LNK-SYS-001-ENCODER")
	if stale["state"] != "version_mismatch" || stale["pinnedVersion"] != "v0.1.0" || stale["dependencyVersion"] != "v0.2.0" || stale["needsReview"] != true ||
		stale["status"] != "accepted" || stale["parent"] != "REQ-SYS-001" || stale["child"] != "encoder::CAP-A-1" {
		t.Fatalf("stale link: %+v", stale)
	}

	review := callToolPayload(t, s, "links.suspect", map[string]any{})
	reviews, _ := review["links"].([]any)
	if len(reviews) != 2 {
		t.Fatalf("links.suspect: want two links, got %+v", review)
	}
	r := linkByID(t, review["links"], "LNK-SYS-001-ENCODER")
	before, _ := r["before"].(map[string]any)
	after, _ := r["after"].(map[string]any)
	if r["state"] != "version_mismatch" || before == nil || after == nil || !strings.Contains(fmt.Sprint(before["realizers"]), "shall publish one position sentence.") ||
		!strings.Contains(fmt.Sprint(after["realizers"]), "shall publish two position sentences.") || fmt.Sprint(r["changes"]) != "[REQ-A-001 text]" || r["parentText"] == "" {
		t.Fatalf("links.suspect entry: %+v", r)
	}
	if c := linkByID(t, review["links"], "LNK-SYS-001-BRIDGE"); c["status"] != "claimed" || c["changes"] != nil {
		t.Fatalf("claimed entry: %+v", c)
	}

	child := linksServer(t, root, "child-a")
	if links, _ := callToolPayload(t, child, "links.list", map[string]any{})["links"].([]any); len(links) != 0 {
		t.Fatalf("child-a has no links: %+v", links)
	}
	if links, _ := callToolPayload(t, child, "links.suspect", map[string]any{})["links"].([]any); len(links) != 0 {
		t.Fatalf("child-a has no links to review: %+v", links)
	}
}

// links.review writes nothing: it returns the engws review-links --accept
// command and the patch of the requirement link file the command would
// write (the dependency pin, status, reviewer, the full reviewedOn line
// from the server clock, and rationale), computed like the command from the
// published dependency at its pin (workspace off); it refuses an actor the
// model does not define and an unknown link. model.patch refuses the link
// file.
// Mutation: writing the link file, dropping the version line from the
// patch, accepting an unknown actor, using the wall clock instead of the
// server clock or letting model.patch edit the link file fails this test.
// TRLC-LINKS: REQ-EMG-106, REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestLinksReviewToolReturnsCommandAndPatch(t *testing.T) {
	root := copyLinksExample(t)
	publishLinksChildren(t, root, "v0.1.0", "v0.2.0")
	if err := os.Remove(filepath.Join(root, "engmod.work.yml")); err != nil {
		t.Fatal(err)
	}
	editTestFile(t, filepath.Join(root, "repos", "system", "engmod.yml"), "  path: models.example/reqlinks/child-a@v0\n  version: v0.1.0\n", "  path: models.example/reqlinks/child-a@v0\n  version: v0.2.0\n")
	linksFile := filepath.Join(root, "repos", "system", "model", "req-links.yml")
	before, err := os.ReadFile(linksFile)
	if err != nil {
		t.Fatal(err)
	}
	s := linksServer(t, root, "system")
	s.clock = func() time.Time { return time.Date(2031, 4, 5, 6, 7, 8, 0, time.UTC) }
	got := callToolPayload(t, s, "links.review", map[string]any{"linkId": "LNK-SYS-001-ENCODER", "by": "ACT-SYSTEM-OWNER", "rationale": "Same meaning at v0.2.0."})
	command, _ := got["command"].(string)
	patch, _ := got["patch"].(string)
	if !strings.Contains(command, "engws review-links") || !strings.Contains(command, "--accept LNK-SYS-001-ENCODER") || !strings.Contains(command, "--by ACT-SYSTEM-OWNER") ||
		!strings.Contains(command, "Same meaning at v0.2.0.") {
		t.Fatalf("command %q", command)
	}
	if !strings.Contains(patch, "\n-    version: v0.1.0\n+    version: v0.2.0\n") || strings.Contains(patch, "fingerprint") ||
		!strings.Contains(patch, "\n-  reviewedOn: \"2026-11-02\"\n+  reviewedOn: \"2031-04-05\"\n") ||
		!strings.Contains(patch, "+  rationale: Same meaning at v0.2.0.") || got["written"] != false || got["path"] != "model/req-links.yml" || got["reviewedOn"] != "2031-04-05" {
		t.Fatalf("patch:\n%s\n%+v", patch, got)
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
