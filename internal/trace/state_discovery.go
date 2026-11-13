// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// The state-like code discovery of ADR-EMG-058 and ADR-EMG-070 (Go only): a
// named type that looks like the state of an object (three constants, a field
// of the type, an assignment to a field of that name) and that no machine of
// the model is linked to is code.state_candidate_unlinked. It is cleared by an
// ENGMODEL-LINKS marker naming a machine, state or transition of the model, or
// by an ENGMODEL-NOT-A-STATE comment with a reason. The facts come from
// codemap.ExtractGoStateFacts, one tree walk of each scanned Go source file.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// The codes of the state-like code discovery. The unreasoned suppression is an
// error at every tier, like code.exemption_unjustified.
const (
	codeStateCandidate  = "code.state_candidate_unlinked"
	codeStateUnreasoned = "code.state_suppression_unreasoned"
	codeStateUnused     = "code.state_suppression_unused"
)

const (
	// stateSuppressionMinReason is the least number of characters of the
	// reason of an ENGMODEL-NOT-A-STATE comment.
	stateSuppressionMinReason = 20
	// stateSuppressionWindow is how many lines above a declaration (or on it)
	// a suppression comment may sit.
	stateSuppressionWindow = 3
)

// placeholderPattern matches the reasons that say nothing, alone or repeated. The
// text is lower-cased and every character that is not a letter or a slash is
// a space before the match.
var placeholderPattern = regexp.MustCompile(`^(?:(?:todo|tbd|n/a|none|fixme|xxx|not a state|ignore|later|wip)(?:\s+|$))+$`)

var nonWordPattern = regexp.MustCompile(`[^a-z/]+`)

// stateFile is a scanned Go source file with its facts.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type stateFile struct {
	rel   string
	facts codemap.GoStateFacts
}

// stateDiscovery is what the diagnostics of the discovery read: the
// candidates by file and line, the model links by file and declaration line,
// and the IDs of the machines, states and transitions of the model.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type stateDiscovery struct {
	cands   map[string]codemap.GoStateCandidate
	links   map[string][]string
	modeled map[string]bool
}

// spotKey is the key of a file and line.
// TRLC-LINKS: REQ-EMG-188
func spotKey(file string, line int) string {
	return fmt.Sprintf("%s:%d", file, line)
}

// stateDiscoveryDiagnostics runs the discovery over the scanned Go source of a
// model: test files and generated files are left out, as in the owner check.
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
func stateDiscoveryDiagnostics(bundle model.Bundle, scopedCode []codescan.CodeItem) []validate.Diagnostic {
	files := goStateFiles(scopedCode)
	d := newStateDiscovery(bundle, scopedCode, files)
	var diags []validate.Diagnostic
	for _, f := range files {
		cleared, found := d.suppressionDiagnostics(f)
		diags = append(diags, found...)
		diags = append(diags, d.candidateDiagnostics(f, cleared)...)
	}
	return diags
}

// goStateFiles reads the facts of each scanned non-test, non-generated Go
// source file, in path order. A file that cannot be read or parsed is left out
// (the scan reports it).
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
func goStateFiles(scopedCode []codescan.CodeItem) []stateFile {
	seen := map[string]bool{}
	var files []stateFile
	for _, it := range scopedCode {
		if !goStateSource(it) || seen[it.AbsPath] {
			continue
		}
		seen[it.AbsPath] = true
		if f, ok := readStateFile(it); ok {
			files = append(files, f)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].facts.File < files[j].facts.File })
	return files
}

// goStateSource reports whether a code item is a Go source file the discovery
// reads: not a test and not generated.
// TRLC-LINKS: REQ-EMG-188
func goStateSource(it codescan.CodeItem) bool {
	if it.Kind != "source_file" || it.AbsPath == "" || !strings.EqualFold(filepath.Ext(it.AbsPath), ".go") {
		return false
	}
	return !codescan.IsVerificationCodeItem(it) && !codescan.IsGeneratedSource(it.AbsPath)
}

// readStateFile reads one file and its state facts.
// TRLC-LINKS: REQ-EMG-188
func readStateFile(it codescan.CodeItem) (stateFile, bool) {
	src, err := os.ReadFile(it.AbsPath)
	if err != nil {
		return stateFile{}, false
	}
	facts, err := codemap.ExtractGoStateFacts(it.AbsPath, src)
	if err != nil {
		return stateFile{}, false
	}
	return stateFile{rel: it.Source, facts: facts}, true
}

// newStateDiscovery gathers the candidates, the links of the declarations and
// the machine IDs.
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func newStateDiscovery(bundle model.Bundle, scopedCode []codescan.CodeItem, files []stateFile) stateDiscovery {
	d := stateDiscovery{cands: candidatesByDeclaration(files), links: declarationLinks(scopedCode), modeled: map[string]bool{}}
	for _, m := range bundle.Architecture.AuthoredArchitecture.Machines {
		addTrimmed(d.modeled, append(append([]string{m.ID}, stateIDs(m)...), transitionIDs(m)...))
	}
	return d
}

// candidatesByDeclaration indexes the candidates of every file by the spot of
// their declaration.
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func candidatesByDeclaration(files []stateFile) map[string]codemap.GoStateCandidate {
	facts := make([]codemap.GoStateFacts, 0, len(files))
	for _, f := range files {
		facts = append(facts, f.facts)
	}
	out := map[string]codemap.GoStateCandidate{}
	for _, c := range codemap.GoStateCandidates(facts) {
		out[spotKey(c.File, c.Line)] = c
	}
	return out
}

// declarationLinks collects the model link markers of every symbol by the
// spot of its declaration.
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func declarationLinks(scopedCode []codescan.CodeItem) map[string][]string {
	out := map[string][]string{}
	for _, it := range scopedCode {
		path, line := codescan.SplitSourceLine(it.Source)
		if it.Kind == "symbol" && line > 0 && path != "" {
			key := spotKey(it.AbsPath, line)
			out[key] = append(out[key], it.ModelLinks...)
		}
	}
	return out
}

// linked reports whether a candidate's declaration carries an ENGMODEL-LINKS
// marker naming a machine, state or transition of the model.
// TRLC-LINKS: REQ-EMG-188
func (d stateDiscovery) linked(c codemap.GoStateCandidate) bool {
	for _, id := range d.links[spotKey(c.File, c.DeclLine)] {
		if d.modeled[strings.TrimSpace(id)] {
			return true
		}
	}
	return false
}

// diagnostic builds one diagnostic of the discovery at a line of a file.
// TRLC-LINKS: REQ-EMG-188
func diagnostic(code string, severity validate.Severity, at stateSpot) validate.Diagnostic {
	return validate.Diagnostic{
		Code: code, Severity: severity, Path: fmt.Sprintf("%s:%d", at.rel, at.line), Remedy: validate.ConsistencyRemedy(code),
	}
}

// stateSpot is a file and a line of a state discovery diagnostic.
type stateSpot struct {
	rel  string
	line int
}

// reasonProblem describes why the reason of a suppression is not one, or "".
// A reason is at least 20 characters after trimming and is not a placeholder
// (todo, tbd, n/a, none, fixme, xxx, not a state, ignore, later, wip, alone or
// repeated).
// TRLC-LINKS: REQ-EMG-188
func reasonProblem(reason string) string {
	trimmed := strings.TrimSpace(reason)
	switch {
	case trimmed == "":
		return "has no reason"
	case placeholderPattern.MatchString(strings.TrimSpace(nonWordPattern.ReplaceAllString(strings.ToLower(trimmed), " "))):
		return fmt.Sprintf("has the placeholder reason %q", trimmed)
	case utf8.RuneCountInString(trimmed) < stateSuppressionMinReason:
		return fmt.Sprintf("has a reason of fewer than %d characters (%q)", stateSuppressionMinReason, trimmed)
	}
	return ""
}

// attachedType returns the type declaration a suppression comment on the given
// line belongs to: the first one on that line or within three lines below it.
// TRLC-LINKS: REQ-EMG-188
func attachedType(types []codemap.GoTypeDecl, line int) (codemap.GoTypeDecl, bool) {
	var best codemap.GoTypeDecl
	found := false
	for _, t := range types {
		gap := t.Line - line
		if gap < 0 || gap > stateSuppressionWindow {
			continue
		}
		if !found || t.Line < best.Line {
			best, found = t, true
		}
	}
	return best, found
}

// suppressionDiagnostics checks each ENGMODEL-NOT-A-STATE comment of a file: a
// bad reason is code.state_suppression_unreasoned and clears nothing; a good
// one on a candidate that is not linked clears it (its line is in the result);
// a good one anywhere else is code.state_suppression_unused.
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (d stateDiscovery) suppressionDiagnostics(f stateFile) (map[int]bool, []validate.Diagnostic) {
	cleared := map[int]bool{}
	var diags []validate.Diagnostic
	for _, s := range f.facts.Suppressions {
		if problem := reasonProblem(s.Reason); problem != "" {
			diag := diagnostic(codeStateUnreasoned, validate.SeverityError, stateSpot{f.rel, s.Line})
			diag.Message = fmt.Sprintf("ENGMODEL-NOT-A-STATE %s; a suppression needs a reason of at least %d characters that is not a placeholder", problem, stateSuppressionMinReason)
			diags = append(diags, diag)
			continue
		}
		if line, ok := d.clearedLine(f, s.Line); ok {
			cleared[line] = true
			continue
		}
		diag := diagnostic(codeStateUnused, validate.SeverityWarning, stateSpot{f.rel, s.Line})
		diag.Message = "ENGMODEL-NOT-A-STATE sits on no state-like candidate (a type with three constants, a field of that type and an assignment to it, not linked to a machine); remove it"
		diags = append(diags, diag)
	}
	return cleared, diags
}

// clearedLine returns the line of the candidate that a suppression at the
// given line clears, when it sits on a candidate that is not linked.
// TRLC-LINKS: REQ-EMG-188
func (d stateDiscovery) clearedLine(f stateFile, line int) (int, bool) {
	t, ok := attachedType(f.facts.Types, line)
	if !ok {
		return 0, false
	}
	c, isCandidate := d.cands[spotKey(f.facts.File, t.Line)]
	if !isCandidate || d.linked(c) {
		return 0, false
	}
	return c.Line, true
}

// candidateDiagnostics reports each candidate of a file that is neither
// linked nor cleared by a suppression.
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (d stateDiscovery) candidateDiagnostics(f stateFile, cleared map[int]bool) []validate.Diagnostic {
	var diags []validate.Diagnostic
	for _, t := range f.facts.Types {
		c, ok := d.cands[spotKey(f.facts.File, t.Line)]
		if !ok || cleared[c.Line] || d.linked(c) {
			continue
		}
		diag := diagnostic(codeStateCandidate, validate.SeverityWarning, stateSpot{f.rel, c.Line})
		diag.Message = fmt.Sprintf("Go type %s looks like a state: %d constants of the type, field %s of the type and an assignment to it, and no machine of the model is linked to it; model it and link it with ENGMODEL-LINKS, or write ENGMODEL-NOT-A-STATE: with a reason of at least %d characters", c.Type, c.Constants, c.Field, stateSuppressionMinReason)
		diags = append(diags, diag)
	}
	return diags
}
