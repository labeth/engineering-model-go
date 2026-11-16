// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engmodel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Candidates are ranked by closeness to the use and test-only code ranks after production code.
// Mutation: sorting by path only, or ignoring the test-only flag, fails this test.
// TRLC-LINKS: REQ-EMG-192, REQ-EMG-200
func TestFindCodeDefinitionsRanksByCloseness(t *testing.T) {
	decls := []CodeDeclaration{
		{Name: "Load", Kind: "function", Language: "go", Path: "a/a_other.go", Line: 3},
		{Name: "Load", Kind: "function", Language: "go", Path: "0/far.go", Line: 9},
		{Name: "Load", Kind: "function", Language: "go", Path: "a/b/use.go", Line: 12},
		{Name: "Loader", Kind: "function", Language: "go", Path: "a/b/use.go", Line: 40},
		{Name: "Load", Kind: "function", Language: "go", Path: "a/b/load_test.go", Line: 4, TestOnly: true},
		{Name: "Load", Kind: "function", Language: "go", Path: "a/b/same_dir.go", Line: 7},
		{Name: "Load", Kind: "function", Language: "go", Path: "a/b/use.go", Line: 1},
		{Name: "Load", Kind: "function", Language: "typescript", Path: "a/b/load.ts", Line: 2},
		{Name: "Other", Kind: "function", Language: "go", Path: "a/b/use.go", Line: 30},
	}
	got := FindCodeDefinitions(decls, "a/b/use.go", "Load")
	want := []string{"a/b/use.go:1", "a/b/use.go:12", "a/b/same_dir.go:7", "a/a_other.go:3", "0/far.go:9", "a/b/load_test.go:4"}
	if len(got) != len(want) {
		t.Fatalf("%d candidates, want %d: %+v", len(got), len(want), got)
	}
	for i, p := range want {
		if at := fmt.Sprintf("%s:%d", got[i].Path, got[i].Line); at != p {
			t.Errorf("candidate %d is %s, want %s", i, at, p)
		}
	}
	fromTest := FindCodeDefinitions(decls, "a/b/load_test.go", "Load")
	if len(fromTest) == 0 || fromTest[0].Path != "a/b/load_test.go" {
		t.Errorf("a use inside test code finds its own file first: %+v", fromTest)
	}
	if got := FindCodeDefinitions(decls, "a/b/use.go", "Nope"); len(got) != 0 {
		t.Errorf("unknown name: %+v", got)
	}
}

// The engine lists the declarations of its own code scope, marked or not.
// TRLC-LINKS: REQ-EMG-192, REQ-EMG-199
func TestListCodeDeclarationsOfThisRepository(t *testing.T) {
	bundle, err := LoadBundle("engmod.yml")
	if err != nil {
		t.Fatal(err)
	}
	decls, err := ListCodeDeclarations(bundle, ".")
	if err != nil {
		t.Fatal(err)
	}
	found := FindCodeDefinitions(decls, "api_code.go", "ListCodeDeclarations")
	if len(found) != 1 || found[0].Path != "api_code.go" || found[0].Kind != "function" {
		t.Fatalf("ListCodeDeclarations: %+v", found)
	}
	if len(decls) < 1000 {
		t.Errorf("only %d declarations listed", len(decls))
	}
}

// declarationWant is what a declaration of a source text is expected to be: its kind and first and last line (0: not checked).
type declarationWant struct {
	kind       string
	line, last int
}

// declarationCase is one source text and the declarations it must list, each once.
type declarationCase struct {
	name, path, src, language string
	testOnly                  bool
	want                      map[string]declarationWant
}

// problem says what is wrong with a declaration of the case, or "".
func (c declarationCase) problem(d CodeDeclaration) string {
	w, ok := c.want[d.Name]
	if !ok {
		return fmt.Sprintf("unexpected declaration %+v", d)
	}
	if got := (declarationWant{d.Kind, d.Line, d.EndLine}); got.kind != w.kind || !w.matchesLines(got) {
		return fmt.Sprintf("%s: %+v, want %+v", d.Name, got, w)
	}
	if got := [3]any{d.Language, d.Path, d.TestOnly}; got != [3]any{c.language, c.path, c.testOnly} {
		return fmt.Sprintf("%s: language, path and test-only mark %v, want %v", d.Name, got, [3]any{c.language, c.path, c.testOnly})
	}
	return ""
}

// matchesLines reports whether the lines of got are the wanted ones; a first line of 0 is not checked.
func (w declarationWant) matchesLines(got declarationWant) bool {
	return lineMatches(w.line, got.line) && lineMatches(w.last, got.last)
}

// lineMatches reports whether a line is the wanted one, or the want is 0 or the first line of a one-line declaration.
func lineMatches(want, got int) bool {
	return want == 0 || got == want
}

// problems lists what is wrong with the declarations of a case: a wrong declaration, and a name listed other than once.
func (c declarationCase) problems(decls []CodeDeclaration) []string {
	var out []string
	seen := map[string]int{}
	for _, d := range decls {
		seen[d.Name]++
		out = append(out, c.problem(d))
	}
	for name := range c.want {
		out = append(out, onceProblem(name, seen[name]))
	}
	return slices.DeleteFunc(out, func(p string) bool { return p == "" })
}

// onceProblem says what is wrong when a name is listed n times instead of once, or "".
func onceProblem(name string, n int) string {
	if n == 1 {
		return ""
	}
	return fmt.Sprintf("%s is listed %d times, want once", name, n)
}

// CodeDeclarationsOf lists every named declaration of one source text once, with kind, language, path and lines, and
// marks the declarations of a test file test-only.
// Mutation: dropping a name of a grouped declaration, a wrong kind, a wrong first or last line or the language fails this test.
// TRLC-LINKS: REQ-EMG-199
func TestCodeDeclarationsOfOneSourceText(t *testing.T) {
	cases := []declarationCase{
		{"go", "p/p.go", "package p\n\ntype (\n\tAlpha struct{}\n\tBeta  int\n)\n\nconst (\n\tOne = 1\n\tTwo = 2\n)\n\nvar x, y = 1, 2\n\nvar p, q Alpha\n\nfunc (a Alpha) Method() {}\n\n// TRLC-LINKS: REQ-X-001\nfunc Marked() {\n\treturn\n}\n\nfunc plain() {}\n", "go", false, map[string]declarationWant{
			"Alpha": {"type", 0, 0}, "Beta": {"type", 5, 5}, "One": {"const", 0, 0}, "Two": {"const", 0, 0}, "x": {"var", 13, 13}, "y": {"var", 13, 13}, "p": {"var", 15, 15}, "q": {"var", 15, 15},
			"Method": {"method", 17, 17}, "Marked": {"function", 20, 22}, "plain": {"function", 24, 24},
		}},
		{"go test file", "p/p_test.go", "package p\n\nimport \"testing\"\n\nfunc TestIt(t *testing.T) {}\n", "go", true, map[string]declarationWant{"TestIt": {"function", 5, 5}}},
		{"typescript", "web/app.ts", "export function render(): void {}\n\nclass View {}\n", "typescript", false, map[string]declarationWant{"render": {"function", 1, 1}, "View": {"class", 3, 3}}},
		{"python", "tool.py", "def run():\n    pass\n\n\nclass Tool:\n    pass\n", "python", false, map[string]declarationWant{"run": {"function", 1, 2}, "Tool": {"class", 5, 6}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			decls, err := CodeDeclarationsOf(c.path, []byte(c.src))
			if err != nil {
				t.Fatal(err)
			}
			for _, problem := range c.problems(decls) {
				t.Error(problem)
			}
		})
	}
}

// A file of no known language has no declarations.
// TRLC-LINKS: REQ-EMG-199
func TestCodeDeclarationsOfAFileOfNoKnownLanguageIsEmpty(t *testing.T) {
	if decls, err := CodeDeclarationsOf("notes.txt", []byte("func NotCode() {}\n")); err != nil || len(decls) != 0 {
		t.Errorf("declarations %+v, error %v", decls, err)
	}
}

// codeScopeModel is child-a as a git repository whose manifest excludes gen/** and that holds Go files in and outside
// the code scope: one git-ignored, one under an excluded path and one in a nested module.
func codeScopeModel(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "model")
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("examples", "requirement-links", "repos", "child-a"))); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "engmod.yml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"engmod.yml":        strings.Replace(string(data), "inferenceHints: {}", "inferenceHints:\n  exclude:\n  - gen/**", 1),
		"src/b.go":          "package src\n\nfunc Second() {}\n\nfunc First() {}\n",
		"src/a.go":          "package src\n\ntype Thing struct{}\n",
		"src/ignored.go":    "package src\n\nfunc Ignored() {}\n",
		"gen/generated.go":  "package gen\n\nfunc Generated() {}\n",
		"nested/engmod.yml": "schemaVersion: 3\n",
		"nested/n.go":       "package nested\n\nfunc Nested() {}\n",
		".gitignore":        "src/ignored.go\n",
	}
	for name, text := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return dir
}

// declaredAt is where a declaration is: its path, name, kind and line.
type declaredAt struct {
	path, name, kind string
	line             int
}

// ListCodeDeclarations lists the declarations of Go files with kind and line in path and line order and leaves out what
// the code scope leaves out: git-ignored files, paths the model excludes and nested modules.
// Mutation: ignoring the git ignore rules, the exclude list or the nested module, or an unsorted result, fails this test.
// TRLC-LINKS: REQ-EMG-199
func TestListCodeDeclarationsFollowsTheCodeScope(t *testing.T) {
	dir := codeScopeModel(t)
	t.Setenv(WorkspaceEnv, WorkspaceOff)
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	decls, err := ListCodeDeclarations(bundle, dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []declaredAt
	for _, d := range decls {
		got = append(got, declaredAt{d.Path, d.Name, d.Kind, d.Line})
	}
	want := []declaredAt{{"src/a.go", "Thing", "type", 3}, {"src/b.go", "Second", "function", 3}, {"src/b.go", "First", "function", 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("declarations %+v, want %+v", got, want)
	}
}
