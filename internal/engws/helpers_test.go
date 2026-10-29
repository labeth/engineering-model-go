// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testFamily = "example.com/fam"

// fakeRegistry publishes the module@version pairs in published.
type fakeRegistry struct {
	published map[string]bool
	unknown   bool
}

func (f fakeRegistry) Published(_ context.Context, modulePath, version string) (Publication, string) {
	if f.unknown {
		return PublicationUnknown, "registry offline"
	}
	if f.published[modulePath+" "+tagFor(version)] {
		return PublicationPublished, ""
	}
	return PublicationMissing, ""
}

// allPublished is a registry that has every module at v0.1.0.
func allPublished(names ...string) fakeRegistry {
	r := fakeRegistry{published: map[string]bool{}}
	for _, name := range names {
		r.published[modulePathOf(name)+" v0.1.0"] = true
	}
	return r
}

func modulePathOf(name string) string { return testFamily + "/" + name + "@v0" }

// dep is a test dependency: alias, checkout name, pinned version.
type dep struct{ alias, name, version string }

// writeModel writes a minimal valid schema-v2 model into dir.
func writeModel(t *testing.T, dir, name, version string, deps ...dep) {
	t.Helper()
	var d strings.Builder
	if len(deps) == 0 {
		d.WriteString("dependencies: []\n")
	} else {
		d.WriteString("dependencies:\n")
		for _, x := range deps {
			d.WriteString("  - alias: " + x.alias + "\n    path: " + modulePathOf(x.name) + "\n    version: " + x.version + "\n    publications: [public]\n")
		}
	}
	tag := strings.ToUpper(strings.ReplaceAll(name, "-", ""))
	write(t, filepath.Join(dir, "engmod.yml"), `schemaVersion: 2
module:
  path: `+modulePathOf(name)+`
  version: `+version+`
  modelId: `+name+`
  title: '`+name+`'
  introduction: ''
  kind: system
documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
`+d.String()+`publications:
  - id: public
    architecture: [CAP-`+tag+`]
    requirements: [REQ-`+tag+`-001]
`)
	write(t, filepath.Join(dir, "cue.mod", "module.cue"), "module: \""+modulePathOf(name)+"\"\nlanguage: version: \"v0.17.0\"\nsource: kind: \"self\"\n")
	system := strings.ToLower(tag) + " system"
	write(t, filepath.Join(dir, "model", "catalog.yml"), "schemaVersion: 2\ncatalog:\n  systems:\n    - id: SYS-"+tag+"\n      name: "+system+"\n      definition: Test system.\n")
	write(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 2\nlintRun:\n  id: lint\n  mode: strict\n  catalogRef: ./catalog.yml\nrequirements:\n  - id: REQ-"+tag+"-001\n    text: 'The "+system+" shall publish one message.'\n")
	var composition strings.Builder
	if len(deps) > 0 {
		composition.WriteString("  composition:\n    subsystems:\n")
		for _, x := range deps {
			composition.WriteString("      - id: SUB-" + strings.ToUpper(x.alias) + "\n        dependency: " + x.alias + "\n        publication: public\n")
		}
	}
	write(t, filepath.Join(dir, "model", "architecture.yml"), "schemaVersion: 2\narchitecture:\n  contract:\n    provides:\n      - id: CAP-"+tag+"\n        kind: capability\n        ref: REQ-"+tag+"-001\n"+composition.String())
	write(t, filepath.Join(dir, "model", "behavior.yml"), "schemaVersion: 2\nbehavior: {}\n")
	write(t, filepath.Join(dir, "model", "assurance.yml"), "schemaVersion: 2\nassurance: {}\n")
	write(t, filepath.Join(dir, "model", "compliance.yml"), "schemaVersion: 2\ncompliance: {}\n")
	write(t, filepath.Join(dir, "model", "views.yml"), "schemaVersion: 2\nviews: []\n")
	write(t, filepath.Join(dir, "model", "decisions.yml"), "schemaVersion: 2\ndecisions: []\n")
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runGit runs git in dir with a fixed identity and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// checkout writes a model as its own git repository, committed and tagged v0.1.0.
func checkout(t *testing.T, root, name string, deps ...dep) string {
	t.Helper()
	dir := filepath.Join(root, name)
	writeModel(t, dir, name, "v0.1.0", deps...)
	write(t, filepath.Join(dir, ".gitignore"), ".engmod/\nengmod.lock.local.yml\n")
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")
	runGit(t, dir, "tag", "v0.1.0")
	return dir
}

// entry returns the plan entry for name.
func entry(t *testing.T, plan Plan, name string) Entry {
	t.Helper()
	for _, e := range plan.Entries {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("no entry %s in %+v", name, plan.Entries)
	return Entry{}
}

// runMain runs the CLI and returns exit code, stdout and stderr.
func runMain(t *testing.T, registry Registry, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Main(context.Background(), args, &out, &errOut, registry)
	return code, out.String(), errOut.String()
}

// assertClean fails when git status shows anything in dir.
func assertClean(t *testing.T, dir string) {
	t.Helper()
	if out := runGit(t, dir, "status", "--porcelain", "--ignored"); out != "" {
		t.Fatalf("checkout %s was modified:\n%s", dir, out)
	}
}
