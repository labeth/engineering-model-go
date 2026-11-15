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
	if err := filepath.WalkDir(src, linksExampleCopier(src, root)); err != nil {
		t.Fatal(err)
	}
	return root
}

// linksExampleCopier returns the WalkDir callback of copyLinksExample.
func linksExampleCopier(src, root string) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
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
	}
}

// registryRoots keeps one registry directory per test, so each
// publishLinksChildren call adds versions to what the test published before.
var registryRoots = map[string]string{}

// linksRegistryRoot returns the test's registry directory, creating it once.
func linksRegistryRoot(t *testing.T) string {
	t.Helper()
	registryRoot, ok := registryRoots[t.Name()]
	if !ok {
		registryRoot = filepath.Join(t.TempDir(), "registry")
		registryRoots[t.Name()] = registryRoot
		t.Cleanup(func() { delete(registryRoots, t.Name()) })
	}
	return registryRoot
}

// childPublication copies one child checkout into a registry directory,
// stamping the published version into its manifest.
type childPublication struct{ src, dst, version string }

// copyEntry is the WalkDir callback of publish.
func (c childPublication) copyEntry(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(c.src, path)
	if d.IsDir() {
		return os.MkdirAll(filepath.Join(c.dst, rel), 0o755)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if rel == "engmod.yml" {
		data = []byte(strings.Replace(string(data), "  version: v0.1.0\n", "  version: "+c.version+"\n", 1))
	}
	return os.WriteFile(filepath.Join(c.dst, rel), data, 0o644)
}

// publish copies the child checkout.
func (c childPublication) publish() error {
	return filepath.WalkDir(c.src, c.copyEntry)
}

// linksRegistry is a registry directory fed from the children of a fixture.
type linksRegistry struct{ root, dir string }

// publishVersion publishes both children of the fixture at the version.
func (r linksRegistry) publishVersion(t *testing.T, version string) {
	t.Helper()
	for _, child := range []string{"child-a", "child-b"} {
		c := childPublication{
			src:     filepath.Join(r.root, "repos", child),
			dst:     filepath.Join(r.dir, "models.example_reqlinks_"+child+"_"+version),
			version: version,
		}
		if err := c.publish(); err != nil {
			t.Fatal(err)
		}
	}
}

// publishLinksChildren publishes the fixture children of root (as edited
// so far) at each version into the test's CUE test registry (versions
// published earlier stay) and routes models.example to it, so a
// workspace-off composition resolves them.
func publishLinksChildren(t *testing.T, root string, versions ...string) {
	t.Helper()
	registryRoot := linksRegistryRoot(t)
	for _, version := range versions {
		linksRegistry{root: root, dir: registryRoot}.publishVersion(t, version)
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
		requireLinkCountersZero(t, args, s.callToolPayload(t, "trace.matrix", args)["summary"].(map[string]any))
	}
	full := s.callToolPayload(t, "trace.matrix", map[string]any{})
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

// requireLinkCountersZero requires every link counter of the summary to be 0.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func requireLinkCountersZero(t *testing.T, args, summary map[string]any) {
	t.Helper()
	for _, key := range []string{"linksNeedReview", "linksVersionMismatch", "linksBroken", "linksClaimed", "unparented"} {
		if summary[key] != float64(0) {
			t.Fatalf("summary %v: %s = %v, want 0: %+v", args, key, summary[key], summary)
		}
	}
}

// toolError calls a tool and returns its error message ("" when it succeeds).
func (s *Server) toolError(t *testing.T, name string, args map[string]any) string {
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
	f := newLinksSuspectFixture(t)
	f.requireWorkspaceLinksUnchanged(t)
	f.requireWorkspaceSuspectShowsPublished(t)
	f.moveSystemToV2(t)
	f.requireStaleLink(t)
	f.requireSuspectEntries(t)
	f.requireChildHasNoLinks(t)
}

// linksSuspectFixture is the links example with a published v0.1.0 and a
// v0.2.0 whose child-a text changed, and a server on the system checkout.
type linksSuspectFixture struct {
	s         *Server
	root      string
	linksFile string
}

const (
	claimedEncoder  = "  status: claimed\n  rationale: The encoder publishes one position sentence per epoch.\n"
	acceptedEncoder = "  status: accepted\n  rationale: The encoder publishes one position sentence per epoch.\n  reviewedBy: ACT-SYSTEM-OWNER\n  reviewedOn: \"2026-11-02\"\n"
	childAPin       = "  path: models.example/reqlinks/child-a@v0\n  version: "
)

// newLinksSuspectFixture publishes v0.1.0, edits the child text, publishes
// v0.2.0 and starts a server on the system checkout.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func newLinksSuspectFixture(t *testing.T) *linksSuspectFixture {
	t.Helper()
	root := copyLinksExample(t)
	childReqs := filepath.Join(root, "repos", "child-a", "model", "requirements.yml")
	publishLinksChildren(t, root, "v0.1.0")
	editTestFile(t, childReqs, textEdit{"shall publish one position sentence.", "shall publish two position sentences."})
	publishLinksChildren(t, root, "v0.2.0")
	return &linksSuspectFixture{s: linksServer(t, root, "system"), root: root, linksFile: filepath.Join(root, "repos", "system", "model", "req-links.yml")}
}

// requireWorkspaceLinksUnchanged requires links.list to keep every link
// unchanged although the child checkout was edited.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (f *linksSuspectFixture) requireWorkspaceLinksUnchanged(t *testing.T) {
	t.Helper()
	list := f.s.callToolPayload(t, "links.list", map[string]any{})
	if links, _ := list["links"].([]any); len(links) != 4 {
		t.Fatalf("links.list: %+v", list)
	}
	for _, id := range []string{"LNK-SYS-001-ENCODER", "LNK-SYS-002-ENCODER"} {
		check(t, "a workspace edit changed link "+id).fieldsEqual(linkByID(t, list["links"], id), map[string]any{"state": "unchanged", "needsReview": false, "local": true})
	}
}

// requireWorkspaceSuspectShowsPublished requires links.suspect in the
// workspace to review against the published v0.1.0 (one sentence), not the
// edited checkout (two).
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (f *linksSuspectFixture) requireWorkspaceSuspectShowsPublished(t *testing.T) {
	t.Helper()
	editTestFile(t, f.linksFile, textEdit{acceptedEncoder, claimedEncoder})
	inWorkspace, _ := f.s.callToolPayload(t, "links.suspect", map[string]any{})["links"].([]any)
	if len(inWorkspace) != 1 || !strings.Contains(fmt.Sprint(inWorkspace[0].(map[string]any)["after"]), "shall publish one position sentence.") {
		t.Fatalf("links.suspect in the workspace must show the published v0.1.0: %+v", inWorkspace)
	}
	editTestFile(t, f.linksFile, textEdit{claimedEncoder, acceptedEncoder})
}

// moveSystemToV2 leaves the workspace: the system moves to v0.2.0 (published
// with two sentences); one link stays reviewed at v0.1.0 and one becomes
// claimed.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (f *linksSuspectFixture) moveSystemToV2(t *testing.T) {
	t.Helper()
	if err := os.Remove(filepath.Join(f.root, "engmod.work.yml")); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(f.root, "repos", "system", "engmod.yml")
	editTestFile(t, manifest, textEdit{childAPin + "v0.1.0\n", childAPin + "v0.2.0\n"})
	link2 := "- id: LNK-SYS-002-ENCODER\n  parent: REQ-SYS-002\n  child: encoder::CAP-A-2\n  pinned:\n    version: "
	editTestFile(t, f.linksFile, textEdit{link2 + "v0.1.0\n", link2 + "v0.2.0\n"})
	editTestFile(t, f.linksFile, textEdit{"  status: accepted\n  rationale: The bridge forwards each sentence onto the lab bus.\n  reviewedBy: ACT-SYSTEM-OWNER\n  reviewedOn: \"2026-11-02\"\n", "  status: claimed\n"})
}

// requireStaleLink requires links.list to report the stale pin.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (f *linksSuspectFixture) requireStaleLink(t *testing.T) {
	t.Helper()
	list := f.s.callToolPayload(t, "links.list", map[string]any{})
	check(t, "stale link").fieldsEqual(linkByID(t, list["links"], "LNK-SYS-001-ENCODER"), map[string]any{
		"state": "version_mismatch", "pinnedVersion": "v0.1.0", "dependencyVersion": "v0.2.0", "needsReview": true,
		"status": "accepted", "parent": "REQ-SYS-001", "child": "encoder::CAP-A-1",
	})
}

// requireSuspectEntries requires links.suspect to list the stale link with
// the item before and after and what changed, and the claimed link.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (f *linksSuspectFixture) requireSuspectEntries(t *testing.T) {
	t.Helper()
	review := f.s.callToolPayload(t, "links.suspect", map[string]any{})
	if reviews, _ := review["links"].([]any); len(reviews) != 2 {
		t.Fatalf("links.suspect: want two links, got %+v", review)
	}
	requireSuspectEntry(t, linkByID(t, review["links"], "LNK-SYS-001-ENCODER"))
	if c := linkByID(t, review["links"], "LNK-SYS-001-BRIDGE"); c["status"] != "claimed" || c["changes"] != nil {
		t.Fatalf("claimed entry: %+v", c)
	}
}

// requireSuspectEntry requires the stale links.suspect entry to carry the item
// before and after, the change list and the parent text.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func requireSuspectEntry(t *testing.T, r map[string]any) {
	t.Helper()
	before, _ := r["before"].(map[string]any)
	after, _ := r["after"].(map[string]any)
	if before == nil || after == nil {
		t.Fatalf("links.suspect entry lacks before or after: %+v", r)
	}
	check(t, "links.suspect before").textHas(fmt.Sprint(before["realizers"]), "shall publish one position sentence.")
	check(t, "links.suspect after").textHas(fmt.Sprint(after["realizers"]), "shall publish two position sentences.")
	check(t, "links.suspect entry").fieldsEqual(r, map[string]any{"state": "version_mismatch"})
	if fmt.Sprint(r["changes"]) != "[REQ-A-001 text]" || r["parentText"] == "" {
		t.Fatalf("links.suspect entry: %+v", r)
	}
}

// requireChildHasNoLinks requires a model without links to return empty lists.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (f *linksSuspectFixture) requireChildHasNoLinks(t *testing.T) {
	t.Helper()
	child := linksServer(t, f.root, "child-a")
	if links, _ := child.callToolPayload(t, "links.list", map[string]any{})["links"].([]any); len(links) != 0 {
		t.Fatalf("child-a has no links: %+v", links)
	}
	if links, _ := child.callToolPayload(t, "links.suspect", map[string]any{})["links"].([]any); len(links) != 0 {
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
	editTestFile(t, filepath.Join(root, "repos", "system", "engmod.yml"), textEdit{childAPin + "v0.1.0\n", childAPin + "v0.2.0\n"})
	linksFile := filepath.Join(root, "repos", "system", "model", "req-links.yml")
	before, err := os.ReadFile(linksFile)
	if err != nil {
		t.Fatal(err)
	}
	s := linksServer(t, root, "system")
	s.clock = func() time.Time { return time.Date(2031, 4, 5, 6, 7, 8, 0, time.UTC) }
	got := s.callToolPayload(t, "links.review", map[string]any{"linkId": "LNK-SYS-001-ENCODER", "by": "ACT-SYSTEM-OWNER", "rationale": "Same meaning at v0.2.0."})
	check(t, "links.review command").textHas(got["command"].(string), "engws review-links", "--accept LNK-SYS-001-ENCODER", "--by ACT-SYSTEM-OWNER", "Same meaning at v0.2.0.")
	requireReviewPatch(t, got)
	check(t, "links.review wrote the link file").fileIs(linksFile, before)
	for _, c := range []toolErrorCase{
		{"links.review", map[string]any{"linkId": "LNK-SYS-001-ENCODER", "by": "ACT-NOBODY"}, "ACT-NOBODY is not an actor", "unknown actor"},
		{"links.review", map[string]any{"linkId": "LNK-NONE", "by": "ACT-SYSTEM-OWNER"}, "no requirement link LNK-NONE", "unknown link"},
		// model.patch does not edit the link file either (D14, ADR-EMG-029).
		{"model.patch", map[string]any{"document": "reqLinks", "id": "LNK-SYS-001-ENCODER", "write": true, "set": map[string]any{"status": "accepted"}}, "engws review-links --accept", "model.patch on the link file"},
	} {
		requireToolError(t, s, c)
	}
	check(t, "model.patch wrote the link file").fileIs(linksFile, before)
}

// requireReviewPatch requires the links.review patch to move the pin and the
// reviewedOn line, add the rationale and not be written.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func requireReviewPatch(t *testing.T, got map[string]any) {
	t.Helper()
	patch, _ := got["patch"].(string)
	check(t, "links.review patch").textHas(patch,
		"\n-    version: v0.1.0\n+    version: v0.2.0\n",
		"\n-  reviewedOn: \"2026-11-02\"\n+  reviewedOn: \"2031-04-05\"\n",
		"+  rationale: Same meaning at v0.2.0.")
	check(t, "links.review patch").textLacks(patch, "fingerprint")
	check(t, "links.review result").fieldsEqual(got, map[string]any{"written": false, "path": "model/req-links.yml", "reviewedOn": "2031-04-05"})
}

// toolErrorCase is a tool call that must fail with an error containing want.
type toolErrorCase struct {
	tool  string
	args  map[string]any
	want  string
	label string
}

// requireToolError requires the call to fail with the wanted message.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func requireToolError(t *testing.T, s *Server, c toolErrorCase) {
	t.Helper()
	if msg := s.toolError(t, c.tool, c.args); !strings.Contains(msg, c.want) {
		t.Fatalf("%s: %q", c.label, msg)
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
	requireSystemRequirementParents(t, linksServer(t, root, "system"))
	requireChildRequirementParents(t, linksServer(t, root, "child-a"))
}

// requireSystemRequirementParents covers REQ-SYS-001: no realizes, two links
// with child items, and the children on requirements.get.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func requireSystemRequirementParents(t *testing.T, sys *Server) {
	t.Helper()
	got := sys.callToolPayload(t, "requirements.parents", map[string]any{"requirementId": "REQ-SYS-001"})
	if len(strs(got["realizes"])) != 0 || got["exists"] != true {
		t.Fatalf("REQ-SYS-001 realizes nothing: %+v", got)
	}
	if links, _ := got["links"].([]any); len(links) != 2 {
		t.Fatalf("REQ-SYS-001 links: %+v", got)
	}
	enc := linkByID(t, got["links"], "LNK-SYS-001-ENCODER")
	item, _ := enc["item"].(map[string]any)
	if item == nil {
		t.Fatalf("LNK-SYS-001-ENCODER has no item: %+v", enc)
	}
	check(t, "LNK-SYS-001-ENCODER").fieldsEqual(enc, map[string]any{"state": "unchanged"})
	check(t, "LNK-SYS-001-ENCODER item").fieldsEqual(item, map[string]any{"id": "encoder::CAP-A-1"})
	check(t, "LNK-SYS-001-ENCODER item realizers").textHas(fmt.Sprint(item["realizers"]), "encoder::REQ-A-002")
	card := sys.callToolPayload(t, "requirements.get", map[string]any{"requirementId": "REQ-SYS-001"})
	if want := "bridge::CAP-B-1,bridge::REQ-B-001,encoder::CAP-A-1,encoder::REQ-A-001,encoder::REQ-A-002"; strings.Join(sortedStrs(card["children"]), ",") != want {
		t.Fatalf("requirements.get children %v, want %s", card["children"], want)
	}
}

// requireChildRequirementParents covers REQ-A-002 (realizes CAP-A-1, parent of
// no link) and the derived REQ-A-004 of the child checkout.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func requireChildRequirementParents(t *testing.T, child *Server) {
	t.Helper()
	got := child.callToolPayload(t, "requirements.parents", map[string]any{"requirementId": "REQ-A-002"})
	if r := strs(got["realizes"]); len(r) != 1 || r[0] != "CAP-A-1" {
		t.Fatalf("REQ-A-002 realizes %v", got)
	}
	if links, _ := got["links"].([]any); len(links) != 0 {
		t.Fatalf("a child requirement is the parent of no link: %+v", got)
	}
	requireChildCard(t, child)
	if got := child.callToolPayload(t, "requirements.parents", map[string]any{"requirementId": "REQ-A-004"}); got["derived"] != true {
		t.Fatalf("REQ-A-004 is derived: %+v", got)
	}
}

// requireChildCard covers requirements.get of REQ-A-002: parent CAP-A-1, no
// children.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func requireChildCard(t *testing.T, child *Server) {
	t.Helper()
	card := child.callToolPayload(t, "requirements.get", map[string]any{"requirementId": "REQ-A-002"})
	if p := strs(card["parents"]); len(p) != 1 || p[0] != "CAP-A-1" || len(strs(card["children"])) != 0 {
		t.Fatalf("requirements.get REQ-A-002: %+v", card)
	}
}

// textEdit replaces old with new once.
type textEdit struct{ old, new string }

// editTestFile replaces the edit's old text (which must occur) with its new
// text in a file.
func editTestFile(t *testing.T, path string, edit textEdit) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), edit.old) {
		t.Fatalf("%s does not contain %q", path, edit.old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), edit.old, edit.new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// sortedStrs is strs, sorted.
func sortedStrs(v any) []string {
	out := strs(v)
	sort.Strings(out)
	return out
}
