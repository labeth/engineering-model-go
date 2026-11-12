// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

// The tree-sitter code health engine (ADR-EMG-047). It parses each file with
// the grammar the codemap package already ships, finds every function unit and
// measures it: logical lines, cyclomatic complexity, nesting depth, bumpy
// road, boolean operators in one condition, parameters, and in tests the
// longest run of assertions. The output is deterministic: files in input
// order, functions in source order, findings in the order of health.Codes().

import (
	"fmt"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/health"
)

// EngineVersion is the version string of the engine; it changes whenever an
// analysis change could alter a result.
const EngineVersion = "2"

// Engine is the tree-sitter implementation of health.Engine.
type Engine struct{}

var _ health.Engine = Engine{}

// New returns the engine.
// TRLC-LINKS: REQ-EMG-150
func New() Engine { return Engine{} }

// Name is "treesitter".
// TRLC-LINKS: REQ-EMG-150
func (Engine) Name() string { return "treesitter" }

// Version is the engine version, part of the report cache key.
// TRLC-LINKS: REQ-EMG-159
func (Engine) Version() string { return EngineVersion }

// Analyze measures every input file; a file whose language is not supported is
// an error. A file with syntax errors is analysed as far as the parser
// recovered it and carries the count of its ERROR and MISSING nodes
// (FileResult.SyntaxErrors).
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-159
func (Engine) Analyze(inputs []health.FileInput, cfg health.Config) ([]health.FileResult, error) {
	thresholds := health.DefaultThresholds()
	for code, t := range cfg.Thresholds {
		thresholds[code] = t
	}
	results := make([]health.FileResult, 0, len(inputs))
	for _, in := range inputs {
		res, err := analyzeFile(in, thresholds)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

// MaxSyntaxDepth is the deepest syntax tree the engine walks. The walk is
// recursive, and a file of a few MiB of opening brackets would otherwise
// exhaust the stack of the process, which no recover can catch. The check is
// iterative, so the file is refused (an error, reported by the caller as a file
// that was not analysed) before any recursion starts.
var MaxSyntaxDepth = 10000

// syntaxDepth returns the depth of the tree under root, or limit+1 as soon as
// it is deeper than limit. It walks with a tree cursor and uses no recursion.
// TRLC-LINKS: REQ-EMG-150
func syntaxDepth(root *sitter.Node, limit int) int {
	cursor := root.Walk()
	defer cursor.Close()
	depth, deepest := 1, 1
	for {
		if cursor.GotoFirstChild() {
			if depth++; depth > deepest {
				if deepest = depth; deepest > limit {
					return deepest
				}
			}
			continue
		}
		for !cursor.GotoNextSibling() {
			if !cursor.GotoParent() {
				return deepest
			}
			depth--
		}
	}
}

// syntaxErrors counts the ERROR and MISSING nodes under root, up to limit. It
// walks with a tree cursor and descends only into nodes that hold an error, so a
// clean file costs one HasError call.
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func syntaxErrors(root *sitter.Node, limit int) int {
	if !root.HasError() {
		return 0
	}
	cursor := root.Walk()
	defer cursor.Close()
	count := 0
	for {
		if n := cursor.Node(); n.HasError() {
			if n.IsError() || n.IsMissing() {
				if count++; count >= limit {
					return limit
				}
			}
			if cursor.GotoFirstChild() {
				continue
			}
		}
		for !cursor.GotoNextSibling() {
			if !cursor.GotoParent() {
				return count
			}
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func analyzeFile(in health.FileInput, thresholds map[string]health.Threshold) (health.FileResult, error) {
	name := in.Language
	if name == "" {
		name, _ = codemap.LanguageForPath(in.Path)
	}
	l, ok := languageFor(name)
	if !ok {
		return health.FileResult{}, fmt.Errorf("health: %s: unsupported language %q", in.Path, in.Language)
	}
	var res health.FileResult
	err := codemap.ParseSource(name, in.Source, func(root *sitter.Node, _ string) error {
		if depth := syntaxDepth(root, MaxSyntaxDepth); depth > MaxSyntaxDepth {
			return fmt.Errorf("the syntax is nested more than %d levels deep", MaxSyntaxDepth)
		}
		var ctx *fileCtx
		res, ctx = analyzeTree(l, in, root, thresholds)
		if ctx.over {
			return fmt.Errorf("the analysis exceeded its budget of %s (deeply nested source?)", AnalysisBudget)
		}
		res.SyntaxErrors = syntaxErrors(root, health.SyntaxErrorCap)
		return nil
	})
	if err != nil {
		return health.FileResult{}, fmt.Errorf("health: %s: %w", in.Path, err)
	}
	return res, nil
}

// unitRecord is a function unit with what is needed after the walk.
type unitRecord struct {
	fn     health.Function
	params paramStats
	runs   []assertRun
}

// collector finds the units of one file in source order.
type collector struct {
	l     *language
	c     *fileCtx
	units []*unitRecord
	// comments are the byte ranges of comments (and Python docstrings) in
	// source order; rows holds the zero-based first and last row of each unit.
	comments []byteRange
}

// TRLC-LINKS: REQ-EMG-150
func analyzeTree(l *language, in health.FileInput, root *sitter.Node, thresholds map[string]health.Threshold) (health.FileResult, *fileCtx) {
	c := &fileCtx{src: in.Source, path: in.Path, deadline: time.Now().Add(AnalysisBudget)}
	c.isTest = l.fileIsTest(c, root)
	col := &collector{l: l, c: c}
	col.collect(root, "")
	lines := newLineIndex(in.Source, col.comments)

	annotated := false
	runs := make([][]assertRun, len(col.units))
	for i, u := range col.units {
		annotated = annotated || u.params.annotated
		runs[i] = u.runs
	}
	supported := l.primitives == primitiveSupported || (l.primitives == primitiveIfAnnotated && annotated)
	duplicates := duplicateFindings(runs)

	res := health.FileResult{
		Path: in.Path, Language: l.name, LOC: lines.total(), IsTest: c.isTest,
		Functions: make([]health.Function, 0, len(col.units)),
	}
	if !supported {
		res.Unsupported = []string{health.CodePrimitiveObsession}
	}
	for i, u := range col.units {
		u.fn.LOC = ownLOC(lines, col.units, i)
		u.fn.Findings = functionFindings(u, supported, duplicates[i], thresholds)
		res.Functions = append(res.Functions, u.fn)
	}
	return res, c
}

// ownLOC is the logical lines of unit i without the rows of the named units
// nested in it. They are measured and reported on their own, and the other
// metrics of the unit skip them as well, so no row counts twice: every row of a
// nested unit's span, its header row included, belongs to that unit. units is
// in source order, so the units nested in i follow it; the rows of a unit are
// subtracted once even when several nested units share a row or one lies inside
// another (the latter takes its own nested units out of its own count).
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-159
func ownLOC(lines lineIndex, units []*unitRecord, i int) int {
	u := units[i]
	first, last := u.fn.StartLine-1, u.fn.EndLine-1
	loc := lines.count(first, last)
	done := first - 1 // the last row already subtracted
	for _, v := range units[i+1:] {
		if v.fn.StartLine-1 > last {
			break
		}
		from, to := max(v.fn.StartLine-1, done+1), min(v.fn.EndLine-1, last)
		if to >= from {
			loc -= lines.count(from, to)
			done = to
		}
	}
	return max(loc, 0)
}

// collect visits the named children of n; scope qualifies the names of the
// functions below it.
// TRLC-LINKS: REQ-EMG-150
func (col *collector) collect(n *sitter.Node, scope string) {
	for i := uint(0); i < n.NamedChildCount(); i++ {
		if child := n.NamedChild(i); child != nil {
			col.visit(child, scope)
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func (col *collector) visit(n *sitter.Node, scope string) {
	if col.c.expired() {
		return
	}
	kind := n.Kind()
	if isCommentKind(kind) || (kind == "expression_statement" && col.l.name == "python" && pythonDocstring(n)) {
		col.comments = append(col.comments, byteRange{n.StartByte(), n.EndByte()})
		return
	}
	inner := scope
	if col.l.unitKinds[kind] {
		if info, ok := col.l.unit(col.c, n); ok {
			inner = col.measureUnit(n, info, scope).fn.Name
		}
	}
	if inner == scope && col.l.scopeKinds[kind] {
		if name := col.l.scope(col.c, n); name != "" {
			inner = qualify(scope, name)
		}
	}
	col.collect(n, inner)
}

// TRLC-LINKS: REQ-EMG-150
func qualify(scope, name string) string {
	if scope == "" {
		return name
	}
	return scope + "." + name
}

// measureUnit measures one function unit and appends it to the file's units.
// TRLC-LINKS: REQ-EMG-150
func (col *collector) measureUnit(n *sitter.Node, info unitInfo, scope string) *unitRecord {
	m := measure(col.l, col.c, info.body)
	ps := col.l.params(col.c, n, info.kind == "method")
	first, last := int(n.StartPosition().Row), int(n.EndPosition().Row)
	if last > first && n.EndPosition().Column == 0 {
		last--
	}
	rec := &unitRecord{params: ps}
	rec.fn = health.Function{
		Name: qualify(scope, info.name), Kind: info.kind,
		StartLine: first + 1, EndLine: last + 1,
		Parameters: ps.count,
		Cyclomatic: 1 + m.cyc,
		MaxNesting: m.maxDepth,
		Bumps:      m.bumps,
		MaxBoolOps: m.maxBool,
		IsTest:     info.test,
	}
	if info.test {
		rec.runs = assertionRuns(col.l, col.c, info.body)
		rec.fn.Assertions = longestRun(rec.runs)
	}
	col.units = append(col.units, rec)
	return rec
}
