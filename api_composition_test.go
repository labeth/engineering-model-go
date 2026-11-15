// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// linksWorkspace copies the requirement links example, its workspace file and its three models, into a temporary directory.
func linksWorkspace(t *testing.T) string {
	t.Helper()
	isolatedModel(t)
	root := filepath.Join(t.TempDir(), "workspace")
	copyModel(t, filepath.Join("examples", "requirement-links"), root)
	return root
}

// snapshot is every file of a directory tree with its content.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		files[path] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// lockedAliases lists the aliases of the dependencies in a lock, and whether each has a digest.
func lockedAliases(lock engmodel.CompositionLockDocument) map[string]bool {
	out := map[string]bool{}
	for _, d := range lock.Dependencies {
		out[d.Alias] = d.Digest != ""
	}
	return out
}

// A manifest whose dependencies a workspace file replaces is composed in lock check mode: the result lists each
// dependency and the lock, and no file is written.
// Mutation: dropping a dependency, writing a lock in check mode or losing the diagnostics fails this test.
// TRLC-LINKS: REQ-EMG-202
func TestGenerateCompositionFromFileWithReplacedDependenciesWritesNothing(t *testing.T) {
	root := linksWorkspace(t)
	before := snapshot(t, root)
	result, err := engmodel.GenerateCompositionFromFileWith(filepath.Join(root, "repos", "system", "engmod.yml"), engmodel.ResolutionOptions{Workspace: filepath.Join(root, "engmod.work.yml"), Lock: engmodel.LockModeCheck})
	if err != nil {
		t.Fatal(err)
	}
	if result.Root == nil {
		t.Fatal("no composed system")
	}
	expect(t, engmodel.HasComposition(result.Root.Bundle), "the root declares no composition")
	check{t}.equal("dependencies", composedChildren(result), []composedChild{
		{"SUB-ENCODER", "encoder", "public", "models.example/reqlinks/child-a@v0", "v0.1.0", true},
		{"SUB-BRIDGE", "bridge", "public", "models.example/reqlinks/child-b@v0", "v0.1.0", true},
	})
	check{t}.equal("locked dependencies with a digest", lockedAliases(result.Lock), map[string]bool{"encoder": true, "bridge": true})
	check{t}.equal("the one note that the committed lock is not checked under replacements", diagnosticCodes(result.Diagnostics), []string{"composition.lock_check_skipped"})
	expect(t, !engmodel.HasErrors(result.Diagnostics), "diagnostics %+v", result.Diagnostics)
	check{t}.equal("the files of the workspace after the check", snapshot(t, root), before)
}

// composedChild is one resolved dependency of a composed system.
type composedChild struct {
	subsystem, dependency, publication, path, version string
	local                                             bool
}

// composedChildren lists the dependencies of the root of a composition.
func composedChildren(result engmodel.CompositionResult) []composedChild {
	var out []composedChild
	for _, c := range result.Root.Children {
		out = append(out, composedChild{c.SubsystemID, c.Dependency, c.Publication, c.ModulePath, c.Version, c.Local})
	}
	return out
}

// diagnosticCodes lists the codes of diagnostics.
func diagnosticCodes(diagnostics []engmodel.Diagnostic) []string {
	var out []string
	for _, d := range diagnostics {
		out = append(out, d.Code)
	}
	return out
}

// fetchErrors are the aliases of the dependencies a composition could not fetch.
func fetchErrors(diagnostics []engmodel.Diagnostic) map[string]bool {
	out := map[string]bool{}
	for _, d := range diagnostics {
		if d.Severity == engmodel.SeverityError && d.Code == "composition.module_fetch_failed" {
			alias, _, _ := strings.Cut(strings.TrimPrefix(d.Message, `dependency "`), `"`)
			out[alias] = true
		}
	}
	return out
}

// A dependency that cannot be resolved is an error of the composition and is not listed as a resolved child.
// Mutation: dropping the error diagnostic or listing the unresolved dependency fails this test.
// TRLC-LINKS: REQ-EMG-202
func TestGenerateCompositionFromFileWithAnUnresolvableDependencyIsAnError(t *testing.T) {
	isolatedModel(t)
	t.Setenv("CUE_REGISTRY", "models.example=127.0.0.1:1+insecure")
	dir := filepath.Join(t.TempDir(), "system")
	copyModel(t, filepath.Join("examples", "requirement-links", "repos", "system"), dir)
	result, err := engmodel.GenerateCompositionFromFileWith(filepath.Join(dir, "engmod.yml"), engmodel.ResolutionOptions{Workspace: engmodel.WorkspaceOff, Lock: engmodel.LockModeCheck, ModuleCache: engmodel.ModuleCacheReuse})
	expect(t, err == nil, "the failure is reported as diagnostics of the result: %v", err)
	expect(t, engmodel.HasErrors(result.Diagnostics), "the composition of unresolvable dependencies has no error diagnostic")
	check{t}.equal("dependencies reported as not fetched", fetchErrors(result.Diagnostics), map[string]bool{"encoder": true, "bridge": true})
	if result.Root != nil {
		check{t}.equal("resolved children", len(result.Root.Children), 0)
	}
}

// siblingUse is the use a sibling checkout of the system makes of one of the encoder's or the bridge's items.
func siblingUse(link, parent, item string) engmodel.LinkUse {
	return engmodel.LinkUse{Model: "REQLINKS-SYSTEM", Link: link, Parent: parent, Item: item, Status: "accepted"}
}

// A model in a workspace whose sibling links one of its capabilities gets that link: the model that holds it, its ID,
// its parent requirement, the item and its status. Without an active workspace nothing is returned.
// Mutation: dropping a field of a use, skipping a sibling, the wrong order or returning uses without a workspace fails this test.
// TRLC-LINKS: REQ-EMG-205
func TestWorkspaceLinkUsersReturnsTheLinksOfTheSiblings(t *testing.T) {
	root := linksWorkspace(t)
	// A link onto an item that the encoder does not publish is no use of one of its contract items.
	links := filepath.Join(root, "repos", "system", "model", "req-links.yml")
	inFile(links).replace(t, "reqLinks:\n", "reqLinks:\n- id: LNK-SYS-009-ENCODER\n  parent: REQ-SYS-001\n  child: encoder::CAP-A-9\n  pinned:\n    version: v0.1.0\n  status: claimed\n")
	cases := map[string][]engmodel.LinkUse{
		"child-a": {siblingUse("LNK-SYS-001-ENCODER", "REQ-SYS-001", "CAP-A-1"), siblingUse("LNK-SYS-002-ENCODER", "REQ-SYS-002", "CAP-A-2")},
		"child-b": {siblingUse("LNK-AOU-B-1", "", "AOU-B-1"), siblingUse("LNK-SYS-001-BRIDGE", "REQ-SYS-001", "CAP-B-1")},
	}
	for child, want := range cases {
		t.Run(child, func(t *testing.T) {
			bundle, err := engmodel.LoadBundle(filepath.Join(root, "repos", child, "engmod.yml"))
			if err != nil {
				t.Fatal(err)
			}
			check{t}.equal("uses", engmodel.WorkspaceLinkUsers(bundle, filepath.Join(root, "engmod.work.yml")), want)
			check{t}.equal("uses with the workspace off", len(engmodel.WorkspaceLinkUsers(bundle, engmodel.WorkspaceOff)), 0)
		})
	}
	alone := filepath.Join(t.TempDir(), "child-a")
	copyModel(t, filepath.Join(root, "repos", "child-a"), alone)
	bundle, err := engmodel.LoadBundle(filepath.Join(alone, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	check{t}.equal("uses of a model outside every workspace", len(engmodel.WorkspaceLinkUsers(bundle, "")), 0)
}
