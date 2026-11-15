// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package engmodel_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// traceRow returns the row of a requirement, or fails the test.
func traceRow(t *testing.T, matrix engmodel.TraceMatrix, id string) engmodel.TraceRow {
	t.Helper()
	for _, row := range matrix.Requirements {
		if row.ID == id {
			return row
		}
	}
	t.Fatalf("%s has no row in the matrix", id)
	return engmodel.TraceRow{}
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
	cases := []struct {
		id         string
		status     string
		codeAt     []string
		hasVerific bool
	}{
		{"REQ-A-001", "implemented", []string{"src/encoder.go:5"}, true},
		{"REQ-A-002", "implemented", []string{"src/more.go:5"}, false},
		{"REQ-A-003", "orphan", nil, false},
		{"REQ-A-005", "orphan", nil, false},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			row := traceRow(t, matrix, c.id)
			var at []string
			for _, ref := range row.Code {
				at = append(at, fmt.Sprintf("%s:%d", ref.Path, ref.Line))
			}
			if row.Status != c.status || !reflect.DeepEqual(at, c.codeAt) || (len(row.Verifications) > 0) != c.hasVerific {
				t.Errorf("status %q, code %v, verifications %v; want %q, %v, verifications %v", row.Status, at, row.Verifications, c.status, c.codeAt, c.hasVerific)
			}
		})
	}
	if !reflect.DeepEqual(matrix.Orphans, []string{"REQ-A-003", "REQ-A-005"}) {
		t.Errorf("orphans %v, want REQ-A-003 and REQ-A-005", matrix.Orphans)
	}
	if len(matrix.DanglingLinks) != 1 || matrix.DanglingLinks[0].Target != "REQ-A-404" {
		t.Errorf("dangling links %+v, want REQ-A-404", matrix.DanglingLinks)
	}
	findings := map[string]string{}
	for _, d := range diagnostics {
		findings[d.Code+" "+d.Message] = string(d.Severity)
	}
	if findings[`code.dangling_requirement_link TRLC-LINKS "REQ-A-404" resolves to no requirement in the model`] != "error" {
		t.Errorf("the dangling link is not an error finding: %v", findings)
	}
	if !slices.ContainsFunc(diagnostics, func(d engmodel.Diagnostic) bool {
		return d.Code == "requirement.orphan" && d.Severity == engmodel.SeverityWarning
	}) {
		t.Errorf("no orphan warning among the findings: %v", findings)
	}
}

// DiffTraceMatrices reports the requirements that gained or lost code or tests and became or stopped being orphans, and
// lists nothing for a removed requirement.
// Mutation: swapping the two matrices or dropping a list fails this test.
// TRLC-LINKS: REQ-EMG-191
func TestDiffTraceMatricesThroughTheFacade(t *testing.T) {
	row := func(id, status string, code, verifications int) engmodel.TraceRow {
		r := engmodel.TraceRow{ID: id, Status: status}
		for i := 0; i < code; i++ {
			r.Code = append(r.Code, engmodel.TraceCodeRef{Path: "a.go", Line: i + 1})
		}
		for i := 0; i < verifications; i++ {
			r.Verifications = append(r.Verifications, "VER-X")
		}
		return r
	}
	from := engmodel.TraceMatrix{
		Requirements: []engmodel.TraceRow{row("R1", "orphan", 0, 0), row("R2", "implemented", 1, 0), row("R3", "implemented", 1, 1), row("GONE", "implemented", 1, 1), row("R5", "orphan", 0, 0), row("R8", "orphan", 0, 0)},
		Orphans:      []string{"R1", "R5"},
	}
	to := engmodel.TraceMatrix{
		Requirements: []engmodel.TraceRow{row("R1", "implemented", 1, 0), row("R2", "implemented", 1, 1), row("R3", "orphan", 0, 0), row("R4", "orphan", 0, 0), row("R5", "orphan", 0, 0), row("R8", "verified", 0, 1)},
		Orphans:      []string{"R3", "R4", "R5"},
	}
	delta := engmodel.DiffTraceMatrices(from, to)
	for name, c := range map[string]struct{ got, want []string }{
		"gained":          {delta.Gained, []string{"R1", "R8"}},
		"tests gained":    {delta.TestsGained, []string{"R2", "R8"}},
		"lost":            {delta.Lost, []string{"R3"}},
		"new orphans":     {delta.NewOrphans, []string{"R3", "R4"}},
		"resolved orphan": {delta.ResolvedOrphans, []string{"R1"}},
	} {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s: %v, want %v", name, c.got, c.want)
		}
	}
	if len(delta.StatusChanged) != 3 || delta.StatusChanged[0] != (engmodel.TraceStatusChange{ID: "R1", From: "orphan", To: "implemented"}) {
		t.Errorf("status changes %+v", delta.StatusChanged)
	}
}
