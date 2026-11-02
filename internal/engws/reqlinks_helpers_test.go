// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"gopkg.in/yaml.v3"
)

// linksExample is the three-repository requirement links fixture.
const linksExample = "../../examples/requirement-links"

// copyTree copies src into dst without generated output, module caches and
// locks.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			if d.Name() == "generated" || d.Name() == ".engmod" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if strings.HasPrefix(d.Name(), "engmod.lock") {
			return nil
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

// replaceIn replaces old (which must occur once or more) with new in a file.
func replaceIn(t *testing.T, path, old, new string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	write(t, path, strings.Replace(string(data), old, new, 1))
}

// read returns a file's content.
func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// linksRegistry publishes fixture children into a CUE test registry: each
// release copies examples/requirement-links/repos/<child>, sets
// module.version and applies edit. It routes models.example to the registry
// and clears the engine's workspace, lock and module cache modes.
type linksRegistry struct {
	t    *testing.T
	root string
}

func newLinksRegistry(t *testing.T) *linksRegistry {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, "")
	t.Setenv(composition.ModuleCacheEnv, "")
	return &linksRegistry{t: t, root: filepath.Join(t.TempDir(), "registry")}
}

func (r *linksRegistry) publish(child, version string, edit func(dir string)) {
	r.t.Helper()
	dir := filepath.Join(r.root, "models.example_reqlinks_"+child+"_"+version)
	copyTree(r.t, filepath.Join(linksExample, "repos", child), dir)
	replaceIn(r.t, filepath.Join(dir, "engmod.yml"), "  version: v0.1.0\n", "  version: "+version+"\n")
	if edit != nil {
		edit(dir)
	}
}

// serve starts the registry with everything published so far.
func (r *linksRegistry) serve() {
	r.t.Helper()
	registry, err := modregistrytest.New(os.DirFS(r.root), "")
	if err != nil {
		r.t.Fatal(err)
	}
	r.t.Cleanup(registry.Close)
	r.t.Setenv("CUE_REGISTRY", "models.example="+registry.Host()+"+insecure")
}

// systemWorkspace copies the fixture system alone into a fresh workspace
// root (no engmod.work.yml), so it resolves its children from the registry.
func systemWorkspace(t *testing.T) (root, system string) {
	t.Helper()
	root = t.TempDir()
	system = filepath.Join(root, "system")
	copyTree(t, filepath.Join(linksExample, "repos", "system"), system)
	return root, system
}

// checkErrors runs the in-process model check of dir and returns its error
// lines.
func checkErrors(t *testing.T, dir string) []string {
	t.Helper()
	res := libraryCheck(CheckTarget{Entry: Entry{Checkout: Checkout{Name: filepath.Base(dir), Dir: dir}}})
	return res.Lines
}

// hasLine reports whether a line of lines contains every part.
func hasLine(lines []string, parts ...string) bool {
	for _, line := range lines {
		all := true
		for _, p := range parts {
			all = all && strings.Contains(line, p)
		}
		if all {
			return true
		}
	}
	return false
}

// decodeStrict decodes YAML with unknown fields rejected.
func decodeStrict(data []byte, out any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(out)
}
