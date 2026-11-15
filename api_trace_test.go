// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package engmodel_test

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// traceRow returns the row of a requirement, or fails the test.
func traceRow(t *testing.T, matrix engmodel.TraceMatrix, id string) engmodel.TraceRow {
	t.Helper()
	i := slices.IndexFunc(matrix.Requirements, func(r engmodel.TraceRow) bool { return r.ID == id })
	if i < 0 {
		t.Fatalf("%s has no row in the matrix", id)
	}
	return matrix.Requirements[i]
}

// rowWant is what a row of the trace matrix must say: its status, the code that links it and whether a test does.
type rowWant struct {
	status   string
	codeAt   []string
	verified bool
}

// codeAt lists where the code of a row is, as path:line.
func codeAt(row engmodel.TraceRow) []string {
	var at []string
	for _, ref := range row.Code {
		at = append(at, fmt.Sprintf("%s:%d", ref.Path, ref.Line))
	}
	return at
}

// codeFindings are the severity of each finding of a trace, by its code and message.
func codeFindings(diagnostics []engmodel.Diagnostic) map[string]string {
	out := map[string]string{}
	for _, d := range diagnostics {
		out[d.Code+" "+d.Message] = string(d.Severity)
	}
	return out
}

// BuildTraceMatrixFromFiles gives each requirement the code and the tests that link it, lists the requirements that
// nothing links as orphans and returns the findings of the scan.
// Mutation: dropping the code, the verifications, the orphan list or the findings fails this test.
// TRLC-LINKS: REQ-EMG-198
func TestBuildTraceMatrixFromFilesRowsOrphansAndFindings(t *testing.T) {
	dir := childAReleased(t)
	writeFile(t, filepath.Join(dir, "src", "more.go"), "// ENGMODEL-OWNER-UNIT: FU-ENCODER\npackage src\n\n// TRLC-LINKS: REQ-A-002, REQ-A-404\nfunc More() {}\n")
	matrix, diagnostics, err := engmodel.BuildTraceMatrixFromFiles(filepath.Join(dir, "engmod.yml"), filepath.Join(dir, "model", "requirements.yml"), dir)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]rowWant{
		"REQ-A-001": {"implemented", []string{"src/encoder.go:5"}, true},
		"REQ-A-002": {"implemented", []string{"src/more.go:5"}, false},
		"REQ-A-003": {"orphan", nil, false},
		"REQ-A-005": {"orphan", nil, false},
	}
	for id, want := range rows {
		row := traceRow(t, matrix, id)
		check{t}.equal(id, rowWant{row.Status, codeAt(row), len(row.Verifications) > 0}, want)
	}
	check{t}.equal("orphans", matrix.Orphans, []string{"REQ-A-003", "REQ-A-005"})
	check{t}.equal("dangling links", matrix.DanglingLinks, []engmodel.DanglingLink{{Kind: "requirement", Target: "REQ-A-404", From: "src/more.go:5"}})
	findings := codeFindings(diagnostics)
	check{t}.equal("severity of the dangling link", findings[`code.dangling_requirement_link TRLC-LINKS "REQ-A-404" resolves to no requirement in the model`], "error")
	check{t}.equal("severity of an orphan", findings["requirement.orphan requirement REQ-A-003 is traced by no code or verification"], "warning")
}

// traceRowOf is a row of a matrix with the given number of code references and verifications.
type traceRowOf struct {
	id, status string
	code, ver  int
}

// matrixOf is a trace matrix of the rows and the orphans among them.
func matrixOf(orphans []string, rows ...traceRowOf) engmodel.TraceMatrix {
	m := engmodel.TraceMatrix{Orphans: orphans}
	for _, r := range rows {
		row := engmodel.TraceRow{ID: r.id, Status: r.status}
		for i := 0; i < r.code; i++ {
			row.Code = append(row.Code, engmodel.TraceCodeRef{Path: "a.go", Line: i + 1})
		}
		for i := 0; i < r.ver; i++ {
			row.Verifications = append(row.Verifications, "VER-X")
		}
		m.Requirements = append(m.Requirements, row)
	}
	return m
}

// DiffTraceMatrices reports the requirements that gained or lost code or tests and became or stopped being orphans, and
// lists nothing for a removed requirement.
// Mutation: swapping the two matrices or dropping a list fails this test.
// TRLC-LINKS: REQ-EMG-191
func TestDiffTraceMatricesThroughTheFacade(t *testing.T) {
	from := matrixOf([]string{"R1", "R5"},
		traceRowOf{"R1", "orphan", 0, 0}, traceRowOf{"R2", "implemented", 1, 0}, traceRowOf{"R3", "implemented", 1, 1},
		traceRowOf{"GONE", "implemented", 1, 1}, traceRowOf{"R5", "orphan", 0, 0}, traceRowOf{"R8", "orphan", 0, 0})
	to := matrixOf([]string{"R3", "R4", "R5"},
		traceRowOf{"R1", "implemented", 1, 0}, traceRowOf{"R2", "implemented", 1, 1}, traceRowOf{"R3", "orphan", 0, 0},
		traceRowOf{"R4", "orphan", 0, 0}, traceRowOf{"R5", "orphan", 0, 0}, traceRowOf{"R8", "verified", 0, 1})
	delta := engmodel.DiffTraceMatrices(from, to)
	check{t}.equal("gained", delta.Gained, []string{"R1", "R8"})
	check{t}.equal("tests gained", delta.TestsGained, []string{"R2", "R8"})
	check{t}.equal("lost", delta.Lost, []string{"R3"})
	check{t}.equal("new orphans", delta.NewOrphans, []string{"R3", "R4"})
	check{t}.equal("resolved orphans", delta.ResolvedOrphans, []string{"R1"})
	check{t}.equal("status changes", delta.StatusChanged, []engmodel.TraceStatusChange{{ID: "R1", From: "orphan", To: "implemented"}, {ID: "R3", From: "implemented", To: "orphan"}, {ID: "R8", From: "orphan", To: "verified"}})
}
