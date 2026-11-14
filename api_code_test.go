// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engmodel

import "testing"

// Candidates are ranked by closeness to the use and test-only code ranks after production code.
// Mutation: sorting by path only, or ignoring the test-only flag, fails this test.
// TRLC-LINKS: REQ-EMG-192, REQ-EMG-200
func TestFindCodeDefinitionsRanksByCloseness(t *testing.T) {
	decls := []CodeDeclaration{
		{Name: "Load", Kind: "function", Language: "go", Path: "a/other.go", Line: 3},
		{Name: "Load", Kind: "function", Language: "go", Path: "z/far.go", Line: 9},
		{Name: "Load", Kind: "function", Language: "go", Path: "a/b/load_test.go", Line: 4, TestOnly: true},
		{Name: "Load", Kind: "function", Language: "go", Path: "a/b/same_dir.go", Line: 7},
		{Name: "Load", Kind: "function", Language: "go", Path: "a/b/use.go", Line: 1},
		{Name: "Load", Kind: "function", Language: "typescript", Path: "a/b/load.ts", Line: 2},
		{Name: "Other", Kind: "function", Language: "go", Path: "a/b/use.go", Line: 30},
	}
	got := FindCodeDefinitions(decls, "a/b/use.go", "Load")
	want := []string{"a/b/use.go", "a/b/same_dir.go", "a/other.go", "z/far.go", "a/b/load_test.go"}
	if len(got) != len(want) {
		t.Fatalf("%d candidates, want %d: %+v", len(got), len(want), got)
	}
	for i, p := range want {
		if got[i].Path != p {
			t.Errorf("candidate %d is %s, want %s", i, got[i].Path, p)
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
