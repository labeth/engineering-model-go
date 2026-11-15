// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
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
	if result.Root == nil || !engmodel.HasComposition(result.Root.Bundle) {
		t.Fatalf("no composed system: %+v", result.Root)
	}
	want := []composedChild{
		{"SUB-ENCODER", "encoder", "public", "models.example/reqlinks/child-a@v0", "v0.1.0", true},
		{"SUB-BRIDGE", "bridge", "public", "models.example/reqlinks/child-b@v0", "v0.1.0", true},
	}
	if got := composedChildren(result); !reflect.DeepEqual(got, want) {
		t.Errorf("dependencies %+v, want %+v", got, want)
	}
	if len(result.Lock.Dependencies) != 2 || result.Lock.Dependencies[0].Alias != "encoder" || result.Lock.Dependencies[1].Alias != "bridge" || result.Lock.Dependencies[0].Digest == "" {
		t.Errorf("lock %+v lists each dependency with its digest", result.Lock)
	}
	if engmodel.HasErrors(result.Diagnostics) || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "composition.lock_check_skipped" {
		t.Errorf("diagnostics %+v, want the one note that the committed lock is not checked under replacements", result.Diagnostics)
	}
	if after := snapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Errorf("the check wrote a file: %d files before, %d after", len(before), len(after))
	}
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

// A dependency that cannot be resolved is an error of the composition and is not listed as a resolved child.
// Mutation: dropping the error diagnostic or listing the unresolved dependency fails this test.
// TRLC-LINKS: REQ-EMG-202
func TestGenerateCompositionFromFileWithAnUnresolvableDependencyIsAnError(t *testing.T) {
	isolatedModel(t)
	t.Setenv("CUE_REGISTRY", "models.example=127.0.0.1:1+insecure")
	dir := filepath.Join(t.TempDir(), "system")
	copyModel(t, filepath.Join("examples", "requirement-links", "repos", "system"), dir)
	result, err := engmodel.GenerateCompositionFromFileWith(filepath.Join(dir, "engmod.yml"), engmodel.ResolutionOptions{Workspace: engmodel.WorkspaceOff, Lock: engmodel.LockModeCheck, ModuleCache: engmodel.ModuleCacheReuse})
	if err != nil {
		t.Fatalf("the failure is reported as diagnostics of the result: %v", err)
	}
	if !engmodel.HasErrors(result.Diagnostics) {
		t.Fatal("the composition of unresolvable dependencies has no error diagnostic")
	}
	for _, alias := range []string{"encoder", "bridge"} {
		found := false
		for _, d := range result.Diagnostics {
			found = found || (d.Severity == engmodel.SeverityError && d.Code == "composition.module_fetch_failed" && strings.HasPrefix(d.Message, `dependency "`+alias+`" `))
		}
		if !found {
			t.Errorf("no error for the dependency %s: %+v", alias, result.Diagnostics)
		}
	}
	if result.Root != nil && len(result.Root.Children) != 0 {
		t.Errorf("unresolved dependencies are listed as children: %d", len(result.Root.Children))
	}
}

// A model in a workspace whose sibling links one of its capabilities gets that link: the model that holds it, its ID,
// its parent requirement, the item and its status. Without an active workspace nothing is returned.
// Mutation: dropping a field of a use, skipping a sibling, the wrong order or returning uses without a workspace fails this test.
// TRLC-LINKS: REQ-EMG-205
func TestWorkspaceLinkUsersReturnsTheLinksOfTheSiblings(t *testing.T) {
	root := linksWorkspace(t)
	workspace := filepath.Join(root, "engmod.work.yml")
	// A link onto an item that the encoder does not publish is no use of one of its contract items.
	links := filepath.Join(root, "repos", "system", "model", "req-links.yml")
	replaceInFile(t, links, "reqLinks:\n", "reqLinks:\n- id: LNK-SYS-009-ENCODER\n  parent: REQ-SYS-001\n  child: encoder::CAP-A-9\n  pinned:\n    version: v0.1.0\n  status: claimed\n")
	cases := []struct {
		child string
		want  []engmodel.LinkUse
	}{
		{"child-a", []engmodel.LinkUse{
			{Model: "REQLINKS-SYSTEM", Link: "LNK-SYS-001-ENCODER", Parent: "REQ-SYS-001", Item: "CAP-A-1", Status: "accepted"},
			{Model: "REQLINKS-SYSTEM", Link: "LNK-SYS-002-ENCODER", Parent: "REQ-SYS-002", Item: "CAP-A-2", Status: "accepted"},
		}},
		{"child-b", []engmodel.LinkUse{
			{Model: "REQLINKS-SYSTEM", Link: "LNK-AOU-B-1", Parent: "", Item: "AOU-B-1", Status: "accepted"},
			{Model: "REQLINKS-SYSTEM", Link: "LNK-SYS-001-BRIDGE", Parent: "REQ-SYS-001", Item: "CAP-B-1", Status: "accepted"},
		}},
	}
	for _, c := range cases {
		t.Run(c.child, func(t *testing.T) {
			bundle, err := engmodel.LoadBundle(filepath.Join(root, "repos", c.child, "engmod.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if got := engmodel.WorkspaceLinkUsers(bundle, workspace); !reflect.DeepEqual(got, c.want) {
				t.Errorf("uses %+v, want %+v", got, c.want)
			}
			if got := engmodel.WorkspaceLinkUsers(bundle, engmodel.WorkspaceOff); len(got) != 0 {
				t.Errorf("a workspace that is off returned %+v", got)
			}
		})
	}
	alone := filepath.Join(t.TempDir(), "child-a")
	copyModel(t, filepath.Join(root, "repos", "child-a"), alone)
	bundle, err := engmodel.LoadBundle(filepath.Join(alone, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := engmodel.WorkspaceLinkUsers(bundle, ""); len(got) != 0 {
		t.Errorf("a model outside every workspace returned %+v", got)
	}
}
