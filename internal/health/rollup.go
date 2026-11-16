// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"path/filepath"
	"sort"
	"strings"
)

// The roll-ups of ADR-EMG-049: file scores reach the elements the model
// knows. Production files and test files are rolled up apart; the unit,
// requirement and project levels see production code only, and the tests block
// reports the test files.

// BandCounts is the number of files in each band.
type BandCounts struct {
	Green  int `json:"green"`
	Yellow int `json:"yellow"`
	Red    int `json:"red"`
}

// WorstFile names the file with the lowest score of an aggregate.
type WorstFile struct {
	Path  string  `json:"path"`
	Score float64 `json:"score"`
}

// Aggregate is the roll-up of a set of files: the LOC-weighted mean score
// (a file counts with its LOC, at least 1), the worst file and the number of
// files per band. A set without files has score 10.0, band green and no worst
// file; Files = 0 tells it apart.
type Aggregate struct {
	Files int        `json:"files"`
	LOC   int        `json:"loc"`
	Score float64    `json:"score"`
	Band  string     `json:"band"`
	Worst *WorstFile `json:"worst,omitempty"`
	Bands BandCounts `json:"bands"`
}

// UnitRollup is the roll-up of the production files attributed to one
// functional unit by the file-level ENGMODEL-OWNER-UNIT marker.
type UnitRollup struct {
	Unit string `json:"unit"`
	Aggregate
}

// RequirementRollup is the roll-up of one requirement: the functions that
// marker-linked symbols with TRLC-LINKS to the requirement lie in. Score is
// the worst score of the files of those functions.
type RequirementRollup struct {
	Requirement string  `json:"requirement"`
	Functions   int     `json:"functions"`
	Findings    int     `json:"findings"`
	Score       float64 `json:"score"`
	Band        string  `json:"band"`
	WorstFile   string  `json:"worstFile"`
}

// Rollups holds every roll-up of a report.
type Rollups struct {
	Project      Aggregate           `json:"project"`
	Units        []UnitRollup        `json:"units"`
	Requirements []RequirementRollup `json:"requirements"`
	// Tests rolls up the test files; it is absent when there are none.
	Tests *Aggregate `json:"tests,omitempty"`
}

// SymbolLink is a marker-linked symbol of the code map: the declaration line
// of a symbol and the requirements its TRLC-LINKS name.
type SymbolLink struct {
	Path         string
	Line         int
	Requirements []string
}

// aggregate rolls a set of files up.
//
// TRLC-LINKS: REQ-EMG-152
func aggregate(files []FileReport) Aggregate {
	if len(files) == 0 {
		return Aggregate{Score: MaxScore, Band: BandGreen}
	}
	var a Aggregate
	var weighted float64
	var weight int
	for i, f := range files {
		w := max(f.LOC, 1)
		weighted += f.Score * float64(w)
		weight += w
		a.Files++
		a.LOC += f.LOC
		a.Bands.count(f.Band)
		// files are sorted by path, so the first of equal scores wins.
		if i == 0 || f.Score < a.Worst.Score {
			a.Worst = &WorstFile{Path: f.Path, Score: f.Score}
		}
	}
	a.Score = RoundTo(weighted/float64(weight), 1)
	a.Band = BandOf(a.Score)
	return a
}

// count counts one file of a band.
//
// TRLC-LINKS: REQ-EMG-152
func (c *BandCounts) count(band string) {
	switch band {
	case BandGreen:
		c.Green++
	case BandYellow:
		c.Yellow++
	default:
		c.Red++
	}
}

// BuildRollups computes the roll-ups of a set of file reports sorted by path.
// owners maps a file path to its functional unit (a file without an entry, or
// with the empty value or "unresolved", counts for the project only) and
// links are the marker-linked symbols used to reach the requirements.
//
// TRLC-LINKS: REQ-EMG-152
// ENGMODEL-LINKS: FU-CODE-HEALTH
func BuildRollups(files []FileReport, owners map[string]string, links []SymbolLink) Rollups {
	g := groupFiles(files, owners)
	r := Rollups{Project: aggregate(g.production), Units: unitRollups(g.byUnit), Requirements: requirementRollups(g.byPath, links)}
	if len(g.tests) > 0 {
		t := aggregate(g.tests)
		r.Tests = &t
	}
	return r
}

// fileGroups are the files of a report grouped for the roll-ups.
type fileGroups struct {
	production, tests []FileReport
	byUnit            map[string][]FileReport
	byPath            map[string]*FileReport
}

// groupFiles splits the files into production and test files, and the
// production files by the unit that owns them.
//
// TRLC-LINKS: REQ-EMG-152
func groupFiles(files []FileReport, owners map[string]string) fileGroups {
	g := fileGroups{byUnit: map[string][]FileReport{}, byPath: map[string]*FileReport{}}
	for i := range files {
		f := files[i]
		g.byPath[f.Path] = &files[i]
		if f.IsTest {
			g.tests = append(g.tests, f)
			continue
		}
		g.production = append(g.production, f)
		if unit := strings.TrimSpace(owners[f.Path]); unit != "" && unit != "unresolved" {
			g.byUnit[unit] = append(g.byUnit[unit], f)
		}
	}
	return g
}

// unitRollups rolls up the files of each unit, sorted by unit.
//
// TRLC-LINKS: REQ-EMG-152
func unitRollups(byUnit map[string][]FileReport) []UnitRollup {
	units := make([]string, 0, len(byUnit))
	for unit := range byUnit {
		units = append(units, unit)
	}
	sort.Strings(units)
	out := make([]UnitRollup, 0, len(units))
	for _, unit := range units {
		out = append(out, UnitRollup{Unit: unit, Aggregate: aggregate(byUnit[unit])})
	}
	return out
}

// reqFnKey names one function of a file by its index.
type reqFnKey struct {
	path string
	fn   int
}

// requirementRollups joins the functions of production files to the symbols
// by path and line range (the innermost function whose range holds the symbol
// line) and rolls them up per requirement, sorted by requirement ID.
//
// TRLC-LINKS: REQ-EMG-152
func requirementRollups(byPath map[string]*FileReport, links []SymbolLink) []RequirementRollup {
	perReq := requirementFunctions(byPath, links)
	reqs := make([]string, 0, len(perReq))
	for req := range perReq {
		reqs = append(reqs, req)
	}
	sort.Strings(reqs)
	out := make([]RequirementRollup, 0, len(reqs))
	for _, req := range reqs {
		out = append(out, requirementRow(req, sortedFnKeys(perReq[req]), byPath))
	}
	return out
}

// requirementFunctions maps each requirement to the functions its linked
// symbols lie in.
//
// TRLC-LINKS: REQ-EMG-152
func requirementFunctions(byPath map[string]*FileReport, links []SymbolLink) reqFunctionSets {
	perReq := reqFunctionSets{}
	for _, link := range links {
		f, fn, ok := linkedFunction(byPath, link)
		if !ok {
			continue
		}
		for _, req := range link.Requirements {
			perReq.add(req, reqFnKey{f.Path, fn})
		}
	}
	return perReq
}

// reqFunctionSets maps a requirement to a set of functions.
type reqFunctionSets map[string]map[reqFnKey]bool

// add puts a function in the set of a requirement.
//
// TRLC-LINKS: REQ-EMG-152
func (s reqFunctionSets) add(req string, key reqFnKey) {
	if s[req] == nil {
		s[req] = map[reqFnKey]bool{}
	}
	s[req][key] = true
}

// linkedFunction finds the production function a symbol link lies in.
//
// TRLC-LINKS: REQ-EMG-152
func linkedFunction(byPath map[string]*FileReport, link SymbolLink) (*FileReport, int, bool) {
	f := byPath[filepath.ToSlash(link.Path)]
	if f == nil || f.IsTest {
		return nil, 0, false
	}
	fn := innermostFunction(f.Functions, link.Line)
	return f, fn, fn >= 0
}

// sortedFnKeys lists a set of functions by path and index.
//
// TRLC-LINKS: REQ-EMG-152
func sortedFnKeys(set map[reqFnKey]bool) []reqFnKey {
	keys := make([]reqFnKey, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		if keys[a].path != keys[b].path {
			return keys[a].path < keys[b].path
		}
		return keys[a].fn < keys[b].fn
	})
	return keys
}

// requirementRow rolls up the functions of one requirement.
//
// TRLC-LINKS: REQ-EMG-152
func requirementRow(req string, keys []reqFnKey, byPath map[string]*FileReport) RequirementRollup {
	row := RequirementRollup{Requirement: req, Functions: len(keys)}
	for i, k := range keys {
		f := byPath[k.path]
		row.Findings += len(f.Functions[k.fn].Findings)
		if i == 0 || f.Score < row.Score {
			row.Score, row.WorstFile = f.Score, f.Path
		}
	}
	row.Band = BandOf(row.Score)
	return row
}

// innermostFunction returns the index of the function with the smallest line
// range that holds line, or -1.
//
// TRLC-LINKS: REQ-EMG-152
func innermostFunction(fns []Function, line int) int {
	best := -1
	for i, f := range fns {
		if line < f.StartLine || line > f.EndLine {
			continue
		}
		if best < 0 || f.EndLine-f.StartLine < fns[best].EndLine-fns[best].StartLine {
			best = i
		}
	}
	return best
}

// RankedFunction is a function with findings and the points they deduct.
type RankedFunction struct {
	Path     string
	Function Function
	Points   float64
}

// WorstFunctions returns the n functions of production files with the most
// deducted points (the sum of weight x severity factor of their findings),
// the worst first; ties go to the path and the line. It ranks functions, not
// files, and is the list the text report shows.
//
// TRLC-LINKS: REQ-EMG-152
// ENGMODEL-LINKS: FU-CODE-HEALTH
func WorstFunctions(r Report, n int) []RankedFunction {
	var ranked []RankedFunction
	for _, f := range r.Files {
		if !f.IsTest {
			ranked = append(ranked, rankedFunctions(f)...)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool { return lessRanked(ranked[i], ranked[j]) })
	if n >= 0 && len(ranked) > n {
		ranked = ranked[:n]
	}
	return ranked
}

// rankedFunctions lists the functions of a file that have deducted points.
//
// TRLC-LINKS: REQ-EMG-152
func rankedFunctions(f FileReport) []RankedFunction {
	var ranked []RankedFunction
	for _, fn := range f.Functions {
		points := 0.0
		for _, finding := range fn.Findings {
			points += weights[finding.Code] * SeverityFactor(finding.Severity)
		}
		if points > 0 {
			ranked = append(ranked, RankedFunction{Path: f.Path, Function: fn, Points: RoundTo(points, 4)})
		}
	}
	return ranked
}

// lessRanked orders by points (most first), then path and line.
//
// TRLC-LINKS: REQ-EMG-152
func lessRanked(a, b RankedFunction) bool {
	if a.Points != b.Points {
		return a.Points > b.Points
	}
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	return a.Function.StartLine < b.Function.StartLine
}
