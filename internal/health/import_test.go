// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// importDoc imports a JSON document.
func importDoc(t *testing.T, doc any) (Report, error) {
	t.Helper()
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return Import(bytes.NewReader(data))
}

// A conforming report imports and re-encodes to the same bytes, so an imported
// report gives the same roll-ups, summary and output as the built-in one.
//
// Mutation: a lossy import (dropping rollups, waived or levels), or a summary
// that is not recomputed, fails.
//
// TRLC-LINKS: REQ-EMG-157
func TestImportRoundTripsAGeneratedReport(t *testing.T) {
	built := BuildReport(sampleResults(), sampleOptions())
	data, err := Encode(built)
	if err != nil {
		t.Fatal(err)
	}
	imported, err := Import(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	again, err := Encode(imported)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again) {
		t.Fatalf("re-encoded import differs:\n%s\n---\n%s", data, again)
	}
	if !reflect.DeepEqual(imported.Rollups, built.Rollups) || imported.Summary != built.Summary {
		t.Error("roll-ups or summary differ after import")
	}
}

// Lists in any order import and are sorted; a summary in the document is
// replaced by the recomputed one.
//
// TRLC-LINKS: REQ-EMG-157
func TestImportSortsAndRecomputesTheSummary(t *testing.T) {
	doc := sampleDocument(t)
	files := doc["files"].([]any)
	files[0], files[2] = files[2], files[0]
	doc["summary"] = map[string]any{"files": 99, "findings": 99, "waived": 99, "errors": 99, "warnings": 99}
	r, err := importDoc(t, doc)
	if err != nil {
		t.Fatal(err)
	}
	if r.Files[0].Path != "pkg/a.go" || r.Files[2].Path != "pkg/z_test.go" {
		t.Errorf("files not sorted: %v %v", r.Files[0].Path, r.Files[2].Path)
	}
	if r.Summary.Files != 3 || r.Summary.Errors != 3 {
		t.Errorf("summary = %+v, want recomputed", r.Summary)
	}
}

// A document that does not conform is rejected with ErrInvalidReport: another
// schema string, a missing required field, a score outside 1.0..10.0, a band
// that does not match the score, an absolute or escaping path, a duplicate file,
// an unknown field, a function that ends before it starts, a roll-up band that
// does not match, or something that is not JSON at all.
//
// Mutation: skipping the schema check, the band check or the path check fails.
//
// TRLC-LINKS: REQ-EMG-157
func TestImportRejectsNonConformingReports(t *testing.T) {
	file := func(doc map[string]any, i int) map[string]any { return doc["files"].([]any)[i].(map[string]any) }
	cases := []struct {
		name    string
		mutate  func(doc map[string]any)
		message string
	}{
		{"another schema", func(d map[string]any) { d["schema"] = "engmod-health/2" }, "schema"},
		{"missing schema", func(d map[string]any) { delete(d, "schema") }, "schema"},
		{"missing engine", func(d map[string]any) { delete(d, "engine") }, "engine"},
		{"missing files", func(d map[string]any) { delete(d, "files") }, "files"},
		{"score out of range", func(d map[string]any) { file(d, 0)["score"] = 11.0 }, "score"},
		{"band does not match", func(d map[string]any) { file(d, 0)["band"] = "red" }, "band"},
		{"absolute path", func(d map[string]any) { file(d, 0)["path"] = "/abs/a.go" }, "path"},
		{"escaping path", func(d map[string]any) { file(d, 0)["path"] = "../a.go" }, "path"},
		{"unclean path", func(d map[string]any) { file(d, 0)["path"] = "pkg/./a.go" }, "path"},
		{"drive letter path", func(d map[string]any) { file(d, 0)["path"] = "C:/x/a.go" }, "path"},
		{"lower-case drive path", func(d map[string]any) { file(d, 0)["path"] = "d:a.go" }, "path"},
		{"UNC path", func(d map[string]any) { file(d, 0)["path"] = `\\server\share\a.go` }, "path"},
		{"waived drive path", func(d map[string]any) { d["waived"].([]any)[0].(map[string]any)["path"] = "C:/x/b.go" }, "path"},
		{"duplicate file", func(d map[string]any) { file(d, 1)["path"] = file(d, 0)["path"] }, "twice"},
		{"unknown field", func(d map[string]any) { d["extra"] = 1 }, "extra"},
		{"function ends before start", func(d map[string]any) {
			fn := file(d, 1)["functions"].([]any)[0].(map[string]any)
			fn["startLine"], fn["endLine"] = 50.0, 40.0
		}, "before it starts"},
		{"project band", func(d map[string]any) { d["rollups"].(map[string]any)["project"].(map[string]any)["band"] = "red" }, "rollups.project"},
	}
	for _, c := range cases {
		doc := sampleDocument(t)
		c.mutate(doc)
		t.Run(c.name, func(t *testing.T) {
			_, err := importDoc(t, doc)
			requireInvalidReport(t, err, c.message)
		})
	}
	requireJunkRejected(t)
}

// requireInvalidReport checks that an import failed with ErrInvalidReport and a
// message that mentions the field.
func requireInvalidReport(t *testing.T, err error, field string) {
	t.Helper()
	mustEqual(t, errors.Is(err, ErrInvalidReport), true)
	wantContains(t, err.Error(), field)
}

// requireJunkRejected checks that text that is no report is refused.
func requireJunkRejected(t *testing.T) {
	t.Helper()
	for _, bad := range []string{"", "not json", "[]", `{"schema": 5}`} {
		_, err := Import(strings.NewReader(bad))
		wantEqual(t, [2]any{bad, errors.Is(err, ErrInvalidReport)}, [2]any{bad, true})
	}
}

// A report of another engine need not carry optional blocks: no tier, levels,
// waived, diagnostics, units or summary.
//
// TRLC-LINKS: REQ-EMG-157
func TestImportAcceptsAMinimalForeignReport(t *testing.T) {
	const minimal = `{
	  "schema": "engmod-health/1",
	  "engine": {"name": "other", "version": "9", "scoringVersion": 1},
	  "files": [{"path": "a.go", "language": "go", "loc": 10, "score": 9.0, "band": "green", "functions": []}],
	  "rollups": {"project": {"files": 1, "loc": 10, "score": 9.0, "band": "green", "bands": {"green": 1, "yellow": 0, "red": 0}}}
	}`
	r, err := Import(strings.NewReader(minimal))
	mustEqual(t, err, nil)
	wantEqual(t, [2]any{r.Engine.Name, r.Summary.Files}, [2]any{"other", 1})
	wantEqual(t, [3]bool{r.Waived != nil, r.Diagnostics != nil, r.Rollups.Units != nil}, [3]bool{true, true, true})
	wantEqual(t, r.LevelOf(CodeComplexMethod), LevelWarning) // no levels read as warnings
}

// Import reads scoring versions 1 and 2 and keeps the one the report was
// written with; the scores are not recomputed. Another version, or none, is
// rejected with a message that names the versions this engine reads.
//
// Mutation: accepting any version, rewriting the version to the current one
// or rescoring the files fails.
//
// TRLC-LINKS: REQ-EMG-151, REQ-EMG-157
func TestImportReadsScoringVersions1And2(t *testing.T) {
	for _, v := range []int{1, 2} {
		requireVersionKept(t, v)
	}
	for _, v := range []int{0, 3, 99} {
		doc := sampleDocument(t)
		doc["engine"].(map[string]any)["scoringVersion"] = v
		_, err := importDoc(t, doc)
		requireInvalidReport(t, err, "scoring versions 1 and 2")
	}
	wantEqual(t, [4]bool{ReadableScoringVersion(1), ReadableScoringVersion(2), ReadableScoringVersion(3), ReadableScoringVersion(0)}, [4]bool{true, true, false, false})
}

// requireVersionKept checks that a report of a scoring version imports with
// that version and the scores it carries.
func requireVersionKept(t *testing.T, v int) {
	t.Helper()
	doc := sampleDocument(t)
	doc["engine"].(map[string]any)["scoringVersion"] = v
	first := doc["files"].([]any)[0].(map[string]any)
	first["score"], first["band"] = 9.5, "green"
	first["deductions"] = []any{}
	r, err := importDoc(t, doc)
	mustEqual(t, err, nil)
	wantEqual(t, r.Engine.ScoringVersion, v)
	for _, f := range r.Files {
		wantEqual(t, f.Path == first["path"] && f.Score != 9.5, false) // the score is not recomputed
	}
}

// The size cap is 64 MiB: a document above it is refused before any parsing, one
// right at it is read.
// Mutation: the old 256 MiB cap lets the oversized document through to the parser.
//
// TRLC-LINKS: REQ-EMG-157
func TestImportRefusesDocumentsAboveTheCap(t *testing.T) {
	if maxReportBytes != 64<<20 {
		t.Fatalf("cap = %d, want 64 MiB", maxReportBytes)
	}
	big := strings.NewReader(`{"schema":"engmod-health/1","pad":"` + strings.Repeat("x", maxReportBytes) + `"}`)
	_, err := Import(big)
	if !errors.Is(err, ErrInvalidReport) || !strings.Contains(err.Error(), "larger than") {
		t.Errorf("err = %v, want the size error", err)
	}
}
