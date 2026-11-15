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
	data, err := readReportBytes(r)
	if err != nil {
		return Report{}, err
	}
	// The document is decoded generically once; the header checks and the schema
	// validation read that value, and only the typed model needs a second decode.
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return Report{}, fmt.Errorf("%w: not a JSON object: %v", ErrInvalidReport, err)
	}
	if err := checkReportHeader(doc); err != nil {
		return Report{}, err
	}
	if err := validateReportDocument(doc); err != nil {
		return Report{}, fmt.Errorf("%w: %v", ErrInvalidReport, err)
	}
	rep, err := decodeReport(data)
	if err != nil {
		return Report{}, err
	}
	canonicalize(&rep)
	rep.Summary = rep.Summarize()
	return rep, nil
}

// readReportBytes reads the document, bounded by maxReportBytes.
//
// TRLC-LINKS: REQ-EMG-157
func readReportBytes(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxReportBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read: %v", ErrInvalidReport, err)
	}
	if len(data) > maxReportBytes {
		return nil, fmt.Errorf("%w: larger than %d bytes", ErrInvalidReport, maxReportBytes)
	}
	return data, nil
}

// checkReportHeader checks the schema string and the scoring version of a
// decoded document.
//
// TRLC-LINKS: REQ-EMG-157
func checkReportHeader(doc map[string]any) error {
	if schema, _ := doc["schema"].(string); schema != ReportSchema {
		return fmt.Errorf("%w: schema %q is not %q", ErrInvalidReport, fmt.Sprint(doc["schema"]), ReportSchema)
	}
	if v, unreadable := unreadableScoringVersion(doc); unreadable {
		return fmt.Errorf("%w: scoringVersion %v is not readable; this engine reads scoring versions 1 and 2 and does not rescore imported reports", ErrInvalidReport, v)
	}
	return nil
}

// unreadableScoringVersion returns the scoring version of the document when
// it is present and is not one ReadableScoringVersion accepts.
//
// TRLC-LINKS: REQ-EMG-157
func unreadableScoringVersion(doc map[string]any) (float64, bool) {
	engine, ok := doc["engine"].(map[string]any)
	if !ok {
		return 0, false
	}
	v, ok := engine["scoringVersion"].(float64)
	return v, ok && (v != float64(int(v)) || !ReadableScoringVersion(int(v)))
}

// decodeReport decodes the typed report from the validated document, refusing
// unknown fields and the rules the JSON Schema cannot express.
//
// TRLC-LINKS: REQ-EMG-157
func decodeReport(data []byte) (Report, error) {
	var rep Report
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rep); err != nil {
		return Report{}, fmt.Errorf("%w: %v", ErrInvalidReport, err)
	}
	if err := checkImported(rep); err != nil {
		return Report{}, fmt.Errorf("%w: %v", ErrInvalidReport, err)
	}
	return rep, nil
}

// checkImported applies the rules the JSON Schema cannot express.
//
// TRLC-LINKS: REQ-EMG-157
func checkImported(r Report) error {
	if err := checkFiles(r.Files); err != nil {
		return err
	}
	if err := checkWaivedPaths(r.Waived); err != nil {
		return err
	}
	if err := checkBaselinedPaths(r.Baselined); err != nil {
		return err
	}
	if err := checkAggregateBands(r.Rollups); err != nil {
		return err
	}
	return checkRequirementBands(r.Rollups.Requirements)
}

// checkFiles checks the paths, the uniqueness, the bands and the line ranges
// of the files.
//
// TRLC-LINKS: REQ-EMG-157
func checkFiles(files []FileReport) error {
	seen := map[string]bool{}
	for _, f := range files {
		if err := checkPath(f.Path); err != nil {
			return err
		}
		if seen[f.Path] {
			return fmt.Errorf("file %q is listed twice", f.Path)
		}
		seen[f.Path] = true
		if err := checkFileEntry(f); err != nil {
			return err
		}
	}
	return nil
}

// checkFileEntry checks the band of a file and the line range of each of its
// functions.
//
// TRLC-LINKS: REQ-EMG-157
func checkFileEntry(f FileReport) error {
	if BandOf(f.Score) != f.Band {
		return fmt.Errorf("file %q has score %.1f but band %q (expected %q)", f.Path, f.Score, f.Band, BandOf(f.Score))
	}
	for _, fn := range f.Functions {
		if fn.EndLine < fn.StartLine {
			return fmt.Errorf("function %q of %q ends on line %d before it starts on line %d", fn.Name, f.Path, fn.EndLine, fn.StartLine)
		}
	}
	return nil
}

// checkWaivedPaths checks the path of each waived finding.
//
// TRLC-LINKS: REQ-EMG-157
func checkWaivedPaths(waived []WaivedFinding) error {
	for _, w := range waived {
		if err := checkPath(w.Path); err != nil {
			return err
		}
	}
	return nil
}

// checkBaselinedPaths checks the path of each baselined finding.
//
// TRLC-LINKS: REQ-EMG-157
func checkBaselinedPaths(baselined []BaselinedFinding) error {
	for _, b := range baselined {
		if err := checkPath(b.Path); err != nil {
			return err
		}
	}
	return nil
}

// namedAggregates names every aggregate of the roll-ups for the band check.
//
// TRLC-LINKS: REQ-EMG-157
func namedAggregates(ro Rollups) map[string]Aggregate {
	aggregates := map[string]Aggregate{"rollups.project": ro.Project}
	if ro.Tests != nil {
		aggregates["rollups.tests"] = *ro.Tests
	}
	for i, u := range ro.Units {
		aggregates[fmt.Sprintf("rollups.units[%d]", i)] = u.Aggregate
	}
	return aggregates
}

// checkAggregateBands checks that every aggregate has the band of its score,
// in the order of the names.
//
// TRLC-LINKS: REQ-EMG-157
func checkAggregateBands(ro Rollups) error {
	aggregates := namedAggregates(ro)
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
	return nil
}

// checkRequirementBands checks that every requirement roll-up has the band of
// its score.
//
// TRLC-LINKS: REQ-EMG-157
func checkRequirementBands(rollups []RequirementRollup) error {
	for _, q := range rollups {
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
	if unusablePath(p) || path.Clean(p) != p || escapesRoot(p) {
		return fmt.Errorf("path %q is not a clean repository-relative path", p)
	}
	return nil
}

// unusablePath reports an empty path, an absolute one, or one holding a
// backslash or a drive letter.
//
// TRLC-LINKS: REQ-EMG-157
func unusablePath(p string) bool {
	return p == "" || rootedPath(p) || hasDriveLetter(p)
}

// rootedPath reports a path that starts at the root or holds a backslash.
//
// TRLC-LINKS: REQ-EMG-157
func rootedPath(p string) bool { return strings.HasPrefix(p, "/") || strings.Contains(p, "\\") }

// escapesRoot reports a path that is .. or starts with ../.
//
// TRLC-LINKS: REQ-EMG-157
func escapesRoot(p string) bool { return p == ".." || strings.HasPrefix(p, "../") }

// hasDriveLetter reports a leading Windows drive designator such as "C:".
//
// TRLC-LINKS: REQ-EMG-157
func hasDriveLetter(p string) bool {
	return len(p) >= 2 && p[1] == ':' && isASCIILetter(p[0])
}

// isASCIILetter reports a byte in a-z or A-Z.
//
// TRLC-LINKS: REQ-EMG-157
func isASCIILetter(b byte) bool {
	lower := b | 0x20
	return lower >= 'a' && lower <= 'z'
}

// findingKey orders the findings of a report: by path, line, function and the
// position of the code.
type findingKey struct {
	path, function string
	line, code     int
}

// lessFindingKey reports whether a sorts before b.
//
// TRLC-LINKS: REQ-EMG-157
func lessFindingKey(a, b findingKey) bool {
	if a.path != b.path {
		return a.path < b.path
	}
	if a.line != b.line {
		return a.line < b.line
	}
	if a.function != b.function {
		return a.function < b.function
	}
	return a.code < b.code
}

// lessDiagnostic orders diagnostics by path, code and function.
//
// TRLC-LINKS: REQ-EMG-157
func lessDiagnostic(a, b Diagnostic) bool {
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	if a.Code != b.Code {
		return a.Code < b.Code
	}
	return a.Function < b.Function
}

// canonicalize sorts every list of a report and replaces nil lists by empty
// ones, so that encoding an imported report is deterministic.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
func canonicalize(r *Report) {
	canonicalizeFiles(r)
	canonicalizeFindings(r)
	canonicalizeRollups(&r.Rollups)
}

// canonicalizeFiles sorts the files and gives each an empty function list
// instead of none.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
func canonicalizeFiles(r *Report) {
	sort.SliceStable(r.Files, func(i, j int) bool { return r.Files[i].Path < r.Files[j].Path })
	for i := range r.Files {
		if r.Files[i].Functions == nil {
			r.Files[i].Functions = []Function{}
		}
	}
}

// canonicalizeFindings sorts the waived and baselined findings and the
// diagnostics.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
func canonicalizeFindings(r *Report) {
	if r.Waived == nil {
		r.Waived = []WaivedFinding{}
	}
	waived := r.Waived
	sort.SliceStable(waived, func(i, j int) bool { return lessFindingKey(waived[i].sortKey(), waived[j].sortKey()) })
	baselined := r.Baselined
	sort.SliceStable(baselined, func(i, j int) bool { return lessFindingKey(baselined[i].sortKey(), baselined[j].sortKey()) })
	if r.Diagnostics == nil {
		r.Diagnostics = []Diagnostic{}
	}
	diagnostics := r.Diagnostics
	sort.SliceStable(diagnostics, func(i, j int) bool {
		a, b := diagnostics[i], diagnostics[j]
		return lessDiagnostic(a, b)
	})
}

// canonicalizeRollups sorts the unit and requirement roll-ups.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
func canonicalizeRollups(ro *Rollups) {
	if ro.Units == nil {
		ro.Units = []UnitRollup{}
	}
	sort.SliceStable(ro.Units, func(i, j int) bool { return ro.Units[i].Unit < ro.Units[j].Unit })
	if ro.Requirements == nil {
		ro.Requirements = []RequirementRollup{}
	}
	sort.SliceStable(ro.Requirements, func(i, j int) bool {
		return ro.Requirements[i].Requirement < ro.Requirements[j].Requirement
	})
}
