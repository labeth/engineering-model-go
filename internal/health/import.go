// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

// maxReportBytes bounds what Import reads: an imported report is data from
// another program and is never trusted to be small. A report of the largest
// project this tool is run on is a few MiB.
const maxReportBytes = 64 << 20

// ReadableScoringVersion reports whether Import accepts a report of this
// scoring version: 1 (no tail cap) and 2 (ADR-EMG-059). The scores of an
// imported report are taken as they are and are never recomputed.
//
// TRLC-LINKS: REQ-EMG-151, REQ-EMG-157
func ReadableScoringVersion(v int) bool { return v == 1 || v == 2 }

// ErrInvalidReport is wrapped by every error of Import: the document is not a
// conforming engmod-health/1 report. The CLI treats it as a setup error.
var ErrInvalidReport = errors.New("invalid health report")

// Import loads a report from another engine (ADR-EMG-050). The document must
// carry the schema string engmod-health/1, a scoringVersion of 1 or 2 (the
// report keeps the one it was written with and is not rescored), conform to the JSON Schema, hold no
// unknown field, name only repository-relative paths without .. segments, have
// a band that matches every score (bands: green from 8.0, yellow from 4.0)
// and list each file once. The order of the lists is not required; Import
// sorts them canonically, so a re-rendered report is deterministic. The summary
// is recomputed from the files, the diagnostics and the levels. The report is
// data; it is validated and never executed.
//
// TRLC-LINKS: REQ-EMG-157
// ENGMODEL-LINKS: FU-CODE-HEALTH
func Import(r io.Reader) (Report, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxReportBytes+1))
	if err != nil {
		return Report{}, fmt.Errorf("%w: read: %v", ErrInvalidReport, err)
	}
	if len(data) > maxReportBytes {
		return Report{}, fmt.Errorf("%w: larger than %d bytes", ErrInvalidReport, maxReportBytes)
	}
	// The document is decoded generically once; the header checks and the schema
	// validation read that value, and only the typed model needs a second decode.
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return Report{}, fmt.Errorf("%w: not a JSON object: %v", ErrInvalidReport, err)
	}
	if schema, _ := doc["schema"].(string); schema != ReportSchema {
		return Report{}, fmt.Errorf("%w: schema %q is not %q", ErrInvalidReport, fmt.Sprint(doc["schema"]), ReportSchema)
	}
	if engine, ok := doc["engine"].(map[string]any); ok {
		if v, ok := engine["scoringVersion"].(float64); ok && (v != float64(int(v)) || !ReadableScoringVersion(int(v))) {
			return Report{}, fmt.Errorf("%w: scoringVersion %v is not readable; this engine reads scoring versions 1 and 2 and does not rescore imported reports", ErrInvalidReport, v)
		}
	}
	if err := validateReportDocument(doc); err != nil {
		return Report{}, fmt.Errorf("%w: %v", ErrInvalidReport, err)
	}
	var rep Report
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rep); err != nil {
		return Report{}, fmt.Errorf("%w: %v", ErrInvalidReport, err)
	}
	if err := checkImported(rep); err != nil {
		return Report{}, fmt.Errorf("%w: %v", ErrInvalidReport, err)
	}
	canonicalize(&rep)
	rep.Summary = rep.Summarize()
	return rep, nil
}

// checkImported applies the rules the JSON Schema cannot express.
//
// TRLC-LINKS: REQ-EMG-157
func checkImported(r Report) error {
	seen := map[string]bool{}
	for _, f := range r.Files {
		if err := checkPath(f.Path); err != nil {
			return err
		}
		if seen[f.Path] {
			return fmt.Errorf("file %q is listed twice", f.Path)
		}
		seen[f.Path] = true
		if BandOf(f.Score) != f.Band {
			return fmt.Errorf("file %q has score %.1f but band %q (expected %q)", f.Path, f.Score, f.Band, BandOf(f.Score))
		}
		for _, fn := range f.Functions {
			if fn.EndLine < fn.StartLine {
				return fmt.Errorf("function %q of %q ends on line %d before it starts on line %d", fn.Name, f.Path, fn.EndLine, fn.StartLine)
			}
		}
	}
	for _, w := range r.Waived {
		if err := checkPath(w.Path); err != nil {
			return err
		}
	}
	for _, b := range r.Baselined {
		if err := checkPath(b.Path); err != nil {
			return err
		}
	}
	aggregates := map[string]Aggregate{"rollups.project": r.Rollups.Project}
	if r.Rollups.Tests != nil {
		aggregates["rollups.tests"] = *r.Rollups.Tests
	}
	for i, u := range r.Rollups.Units {
		aggregates[fmt.Sprintf("rollups.units[%d]", i)] = u.Aggregate
	}
	names := make([]string, 0, len(aggregates))
	for name := range aggregates {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if a := aggregates[name]; BandOf(a.Score) != a.Band {
			return fmt.Errorf("%s has score %.1f but band %q (expected %q)", name, a.Score, a.Band, BandOf(a.Score))
		}
	}
	for _, q := range r.Rollups.Requirements {
		if BandOf(q.Score) != q.Band {
			return fmt.Errorf("requirement roll-up %s has score %.1f but band %q (expected %q)", q.Requirement, q.Score, q.Band, BandOf(q.Score))
		}
	}
	return nil
}

// checkPath rejects a path that is absolute, names a volume (a drive letter such
// as C:/x, or a UNC share, which holds a backslash) or leaves the repository.
//
// TRLC-LINKS: REQ-EMG-157
func checkPath(p string) error {
	if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "\\") || hasDriveLetter(p) || path.Clean(p) != p || p == ".." || strings.HasPrefix(p, "../") {
		return fmt.Errorf("path %q is not a clean repository-relative path", p)
	}
	return nil
}

// hasDriveLetter reports a leading Windows drive designator such as "C:".
//
// TRLC-LINKS: REQ-EMG-157
func hasDriveLetter(p string) bool {
	return len(p) >= 2 && p[1] == ':' && (p[0] >= 'a' && p[0] <= 'z' || p[0] >= 'A' && p[0] <= 'Z')
}

// canonicalize sorts every list of a report and replaces nil lists by empty
// ones, so that encoding an imported report is deterministic.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
func canonicalize(r *Report) {
	sort.SliceStable(r.Files, func(i, j int) bool { return r.Files[i].Path < r.Files[j].Path })
	for i := range r.Files {
		if r.Files[i].Functions == nil {
			r.Files[i].Functions = []Function{}
		}
	}
	if r.Waived == nil {
		r.Waived = []WaivedFinding{}
	}
	sort.SliceStable(r.Waived, func(i, j int) bool {
		a, b := r.Waived[i], r.Waived[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.StartLine != b.StartLine {
			return a.StartLine < b.StartLine
		}
		if a.Function != b.Function {
			return a.Function < b.Function
		}
		return codeOrder(a.Code) < codeOrder(b.Code)
	})
	sort.SliceStable(r.Baselined, func(i, j int) bool {
		a, b := r.Baselined[i], r.Baselined[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.StartLine != b.StartLine {
			return a.StartLine < b.StartLine
		}
		if a.Function != b.Function {
			return a.Function < b.Function
		}
		return codeOrder(a.Code) < codeOrder(b.Code)
	})
	if r.Diagnostics == nil {
		r.Diagnostics = []Diagnostic{}
	}
	sort.SliceStable(r.Diagnostics, func(i, j int) bool {
		a, b := r.Diagnostics[i], r.Diagnostics[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Function < b.Function
	})
	if r.Rollups.Units == nil {
		r.Rollups.Units = []UnitRollup{}
	}
	sort.SliceStable(r.Rollups.Units, func(i, j int) bool { return r.Rollups.Units[i].Unit < r.Rollups.Units[j].Unit })
	if r.Rollups.Requirements == nil {
		r.Rollups.Requirements = []RequirementRollup{}
	}
	sort.SliceStable(r.Rollups.Requirements, func(i, j int) bool {
		return r.Rollups.Requirements[i].Requirement < r.Rollups.Requirements[j].Requirement
	})
}
