// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func find(decls []Declaration, name string) (Declaration, bool) {
	for _, d := range decls {
		if d.Name == name {
			return d, true
		}
	}
	return Declaration{}, false
}

const goSample = `package p

type (
	Alpha struct{}
	Beta  int
)

const (
	One = 1
	Two = 2
)

var x, y = 1, 2

func (a Alpha) Method() {}

// TRLC-LINKS: REQ-X-001
func Marked() {
	return
}

func plain() {}
`

func goSampleDeclarations(t *testing.T) []Declaration {
	t.Helper()
	decls, err := DeclarationsOf("p/p.go", []byte(goSample))
	if err != nil {
		t.Fatal(err)
	}
	return decls
}

// Every named declaration is listed, with or without a trace marker, including each name of a grouped Go block.
// Mutation: listing only declarations that carry a marker, or only the first name of a block, fails this test.
// TRLC-LINKS: REQ-EMG-192
func TestDeclarationsOfGoListsUnmarkedAndGroupedNames(t *testing.T) {
	decls := goSampleDeclarations(t)
	want := map[string]string{"Alpha": "type", "Beta": "type", "One": "const", "Two": "const", "x": "var", "y": "var", "Method": "method", "Marked": "function", "plain": "function"}
	for name, kind := range want {
		d, _ := find(decls, name)
		if d.Kind != kind || d.Language != "go" || d.Path != "p/p.go" {
			t.Errorf("%s: %+v, want kind %s", name, d, kind)
		}
	}
}

// TRLC-LINKS: REQ-EMG-192
func TestDeclarationsOfGoReportsLines(t *testing.T) {
	decls := goSampleDeclarations(t)
	marked, _ := find(decls, "Marked")
	beta, _ := find(decls, "Beta")
	if marked.Line != 18 || marked.EndLine != 20 || beta.Line != 5 {
		t.Errorf("Marked spans %d..%d (want 18..20), Beta is on line %d (want 5)", marked.Line, marked.EndLine, beta.Line)
	}
}

// A type declares one name: the type it is defined as is no declaration, and neither is the type of a var.
// Mutation: reading every leading identifier of a spec as a name fails this test.
// TRLC-LINKS: REQ-EMG-192
func TestDeclarationsOfGoTypeSpecDeclaresOneName(t *testing.T) {
	decls, err := DeclarationsOf("p/q.go", []byte("package p\n\ntype Beta int\n\ntype Gamma = Beta\n\nvar a, b Beta\n\nconst C Beta = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, d := range decls {
		got = append(got, d.Name)
	}
	if !reflect.DeepEqual(got, []string{"Beta", "Gamma", "a", "b", "C"}) {
		t.Errorf("declared names %v, want Beta, Gamma, a, b, C once each", got)
	}
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, text := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func sampleTree(t *testing.T) []Declaration {
	t.Helper()
	root := writeTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		"a_test.go":             "package a\nimport \"testing\"\nfunc TestA(t *testing.T) {}\n",
		"web/app.ts":            "export function render(): void {}\nclass View {}\n",
		"tool.py":               "def run():\n    pass\n",
		"node_modules/dep/x.go": "package x\nfunc Hidden() {}\n",
		".cache/y.go":           "package y\nfunc Cached() {}\n",
		"notes.txt":             "func NotCode() {}\n",
	})
	decls, err := ListDeclarations(root, ScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return decls
}

// TRLC-LINKS: REQ-EMG-192
func TestListDeclarationsReadsSeveralLanguages(t *testing.T) {
	decls := sampleTree(t)
	render, _ := find(decls, "render")
	view, _ := find(decls, "View")
	run, _ := find(decls, "run")
	got := []string{render.Language + " " + render.Kind, view.Kind, run.Language + " " + run.Kind}
	if !reflect.DeepEqual(got, []string{"typescript function", "class", "python function"}) {
		t.Errorf("render, View, run: %v", got)
	}
}

// A walk skips dependency caches, dot-directories and files of no known language like the trace scan does.
// TRLC-LINKS: REQ-EMG-192
func TestListDeclarationsSkipsCachesAndUnknownFiles(t *testing.T) {
	decls := sampleTree(t)
	for _, name := range []string{"Hidden", "Cached", "NotCode"} {
		if _, found := find(decls, name); found {
			t.Errorf("%s must not be listed", name)
		}
	}
}

// A test file's declarations are marked test-only; production code is not.
// TRLC-LINKS: REQ-EMG-192
func TestListDeclarationsMarksTestOnlyCode(t *testing.T) {
	decls := sampleTree(t)
	testA, _ := find(decls, "TestA")
	prodA, _ := find(decls, "A")
	if !testA.TestOnly || testA.Path != "a_test.go" || prodA.TestOnly {
		t.Errorf("TestA %+v, A %+v", testA, prodA)
	}
}

// TRLC-LINKS: REQ-EMG-192
func TestListDeclarationsIsInPathOrder(t *testing.T) {
	decls := sampleTree(t)
	for i := 1; i < len(decls); i++ {
		if decls[i-1].Path > decls[i].Path {
			t.Errorf("not in path order: %v before %v", decls[i-1].Path, decls[i].Path)
		}
	}
}
