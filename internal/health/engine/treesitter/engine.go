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
		delta, ok := advance(cursor, true)
		if !ok {
			return deepest
		}
		depth += delta
		if deepest = max(deepest, depth); deepest > limit {
			return deepest
		}
	}
}

// advance moves the cursor to the next node of the walk: the first child when
// descend is set and there is one, else the next sibling of the node or of the
// nearest ancestor that has one. It returns the change of the depth; false when
// the walk is over.
// TRLC-LINKS: REQ-EMG-150
func advance(cursor *sitter.TreeCursor, descend bool) (int, bool) {
	if descend && cursor.GotoFirstChild() {
		return 1, true
	}
	up := 0
	for !cursor.GotoNextSibling() {
		if !cursor.GotoParent() {
			return 0, false
		}
		up++
	}
	return -up, true
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
		n := cursor.Node()
		if isErrorNode(n) {
			count++
		}
		if count >= limit {
			return limit
		}
		if _, ok := advance(cursor, n.HasError()); !ok {
			return count
		}
	}
}

// isErrorNode reports an ERROR or MISSING node.
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func isErrorNode(n *sitter.Node) bool {
	return n.HasError() && (n.IsError() || n.IsMissing())
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
		res, ctx = fileAnalysis{l: l, in: in, thresholds: thresholds}.tree(root)
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

// fileAnalysis is the input of the analysis of one parsed file.
type fileAnalysis struct {
	l          *language
	in         health.FileInput
	thresholds map[string]health.Threshold
}

// tree measures the functions of a parsed file and finds their smells.
// TRLC-LINKS: REQ-EMG-150
func (a fileAnalysis) tree(root *sitter.Node) (health.FileResult, *fileCtx) {
	c := &fileCtx{src: a.in.Source, path: a.in.Path, deadline: time.Now().Add(AnalysisBudget)}
	c.isTest = a.l.fileIsTest(c, root)
	col := &collector{l: a.l, c: c}
	col.collect(root, "")
	lines := newLineIndex(a.in.Source, col.comments)

	supported := a.l.primitivesComputable(col.units)
	duplicates := duplicateFindings(unitRuns(col.units))
	res := health.FileResult{
		Path: a.in.Path, Language: a.l.name, LOC: lines.total(), IsTest: c.isTest,
		Functions: make([]health.Function, 0, len(col.units)),
	}
	if !supported {
		res.Unsupported = []string{health.CodePrimitiveObsession}
	}
	for i, u := range col.units {
		u.fn.LOC = ownLOC(lines, col.units, i)
		u.fn.Findings = functionFindings(u, findingContext{primitiveOK: supported, duplicates: duplicates[i], thresholds: a.thresholds})
		res.Functions = append(res.Functions, u.fn)
	}
	return res, c
}

// unitRuns lists the assertion runs of each unit.
// TRLC-LINKS: REQ-EMG-150
func unitRuns(units []*unitRecord) [][]assertRun {
	runs := make([][]assertRun, len(units))
	for i, u := range units {
		runs[i] = u.runs
	}
	return runs
}

// primitivesComputable reports whether primitive obsession can be computed for
// a file: the language writes its parameter types, or it does and the file
// annotates them.
// TRLC-LINKS: REQ-EMG-150
func (l *language) primitivesComputable(units []*unitRecord) bool {
	annotated := false
	for _, u := range units {
		annotated = annotated || u.params.annotated
	}
	return l.primitives == primitiveSupported || (l.primitives == primitiveIfAnnotated && annotated)
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
	if isCommentKind(kind) || col.isDocstring(n, kind) {
		col.comments = append(col.comments, byteRange{n.StartByte(), n.EndByte()})
		return
	}
	inner := col.unitScope(n, kind, scope)
	if inner == scope && col.l.scopeKinds[kind] {
		inner = col.namedScope(n, scope)
	}
	col.collect(n, inner)
}

// isDocstring reports a Python docstring, which counts as a comment.
// TRLC-LINKS: REQ-EMG-150
func (col *collector) isDocstring(n *sitter.Node, kind string) bool {
	return kind == "expression_statement" && col.l.name == "python" && pythonDocstring(n)
}

// unitScope measures the node when it is a function unit and returns the name
// of the scope below it; the scope itself for any other node.
// TRLC-LINKS: REQ-EMG-150
func (col *collector) unitScope(n *sitter.Node, kind, scope string) string {
	if !col.l.unitKinds[kind] {
		return scope
	}
	info, ok := col.l.unit(col.c, n)
	if !ok {
		return scope
	}
	return col.measureUnit(n, info, scope).fn.Name
}

// namedScope is the scope below a node that names one (a class, a module), or
// the scope itself when it has no name.
// TRLC-LINKS: REQ-EMG-150
func (col *collector) namedScope(n *sitter.Node, scope string) string {
	if name := col.l.scope(col.c, n); name != "" {
		return qualify(scope, name)
	}
	return scope
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
