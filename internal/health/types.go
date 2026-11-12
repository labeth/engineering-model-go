// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

// The neutral data model of the code health analysis (ADR-EMG-046, ADR-EMG-050).
// The tree-sitter engine fills FileResult; scoring, roll-ups, waivers, deltas
// and every export consume only Report and never an engine's internals, so
// another engine can be plugged in behind Engine later.

// ReportSchema is the schema string of the JSON report.
const ReportSchema = "engmod-health/1"

// ScoringVersion is the version of the scoring in scoring.go: the weights
// table and the combination rule (version 2 caps the tail, ADR-EMG-059).
const ScoringVersion = 2

// Severity of one finding.
type Severity string

const (
	// SeverityWarn is the lower severity of a smell.
	SeverityWarn Severity = "warn"
	// SeverityHigh is the higher severity of a smell.
	SeverityHigh Severity = "high"
)

// Diagnostic codes of the findings (the smells of ADR-EMG-047).
const (
	CodeComplexMethod       = "health.complex_method"
	CodeLargeMethod         = "health.large_method"
	CodeDeepNesting         = "health.deep_nesting"
	CodeBumpyRoad           = "health.bumpy_road"
	CodeComplexConditional  = "health.complex_conditional"
	CodeExcessArguments     = "health.excess_arguments"
	CodePrimitiveObsession  = "health.primitive_obsession"
	CodeBrainMethod         = "health.brain_method"
	CodeLargeAssertionBlock = "health.large_assertion_block"
	CodeDuplicatedAssertion = "health.duplicated_assertion_block"
)

// Diagnostic codes about the analysis itself (a source file that could not be
// analysed). A file that is not analysed is no clean file: both have the level
// of health.regression at the tier, so they block where a regression blocks.
const (
	CodeFileSkipped = "health.file_skipped"
	CodeFileFailed  = "health.file_failed"
	// CodeSyntaxError marks a file the parser could only partly read: the
	// functions it lost are not measured, so the file must not read as clean.
	CodeSyntaxError = "health.syntax_error"
	// CodeRollupUnavailable marks a report whose unit and requirement roll-ups
	// could not be built from the code scan of the model.
	CodeRollupUnavailable = "health.rollup_unavailable"
)

// SyntaxErrorCap bounds the number of ERROR and MISSING syntax nodes an engine
// counts in one file (FileResult.SyntaxErrors); a count at the cap means "at
// least".
const SyntaxErrorCap = 1000

// Codes lists every finding code in report order.
//
// TRLC-LINKS: REQ-EMG-150
func Codes() []string {
	return []string{
		CodeComplexMethod, CodeLargeMethod, CodeDeepNesting, CodeBumpyRoad,
		CodeComplexConditional, CodeExcessArguments, CodePrimitiveObsession,
		CodeBrainMethod, CodeLargeAssertionBlock, CodeDuplicatedAssertion,
	}
}

// Threshold is the warn and high limit of one smell. A metric at or above Warn
// is a warn finding, at or above High a high finding.
type Threshold struct {
	Warn int `json:"warn"`
	High int `json:"high"`
}

// DefaultThresholds returns the default limits of ADR-EMG-047 (OWNER items).
// A smell without a numeric limit (primitive_obsession, brain_method,
// duplicated_assertion_block) is not in the map.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-153
func DefaultThresholds() map[string]Threshold {
	return map[string]Threshold{
		CodeComplexMethod:       {Warn: 10, High: 20},
		CodeLargeMethod:         {Warn: 70, High: 120},
		CodeDeepNesting:         {Warn: 4, High: 5},
		CodeBumpyRoad:           {Warn: 2, High: 3},
		CodeComplexConditional:  {Warn: 3, High: 5},
		CodeExcessArguments:     {Warn: 5, High: 7},
		CodeLargeAssertionBlock: {Warn: 6, High: 10},
	}
}

// DefaultThresholdsForTier returns the limits the scan applies for a project
// life-cycle tier (ADR-EMG-057, calibrated by ADR-EMG-059, OWNER items). The tiers product and operational
// share the stricter set because findings are errors there and AI-assisted
// work needs cleaner code; every other tier, and an empty tier, gets
// DefaultThresholds.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func DefaultThresholdsForTier(tier string) map[string]Threshold {
	if tier != "product" && tier != "operational" {
		return DefaultThresholds()
	}
	return map[string]Threshold{
		CodeComplexMethod:       {Warn: 8, High: 15},
		CodeLargeMethod:         {Warn: 50, High: 100},
		CodeDeepNesting:         {Warn: 3, High: 4},
		CodeBumpyRoad:           {Warn: 2, High: 3},
		CodeComplexConditional:  {Warn: 3, High: 5},
		CodeExcessArguments:     {Warn: 4, High: 6},
		CodeLargeAssertionBlock: {Warn: 5, High: 8},
	}
}

// Config is what an engine needs to produce findings.
type Config struct {
	// Thresholds override DefaultThresholds per code (never partially: a code
	// present here replaces both limits).
	Thresholds map[string]Threshold
}

// Finding is one smell found in one function.
type Finding struct {
	Code      string   `json:"code"`
	Severity  Severity `json:"severity"`
	StartLine int      `json:"startLine"`
	EndLine   int      `json:"endLine"`
	// Metric names what was measured (cyclomatic, loc, nesting, bumps,
	// boolean_operators, parameters, assertions) and Value is its value.
	Metric  string `json:"metric"`
	Value   int    `json:"value"`
	Message string `json:"message"`
}

// Function is one function, method or named closure with its metrics.
// Anonymous closures count toward their enclosing function; a nested named
// function is a Function of its own and its rows are not in the LOC of the
// enclosing one (ADR-EMG-047).
type Function struct {
	// Name is the qualified name: Type.Method, Outer.inner, or the plain name.
	Name       string    `json:"name"`
	Kind       string    `json:"kind"` // function, method
	StartLine  int       `json:"startLine"`
	EndLine    int       `json:"endLine"`
	LOC        int       `json:"loc"` // logical lines: not blank, not comment, without the rows of nested named units
	Parameters int       `json:"parameters"`
	Cyclomatic int       `json:"cyclomatic"`
	MaxNesting int       `json:"maxNesting"`
	Bumps      int       `json:"bumps"`
	MaxBoolOps int       `json:"maxBooleanOperators"`
	Assertions int       `json:"assertions,omitempty"` // longest consecutive run, tests only
	IsTest     bool      `json:"isTest,omitempty"`
	Findings   []Finding `json:"findings,omitempty"`
}

// FileInput is one source file handed to an engine.
type FileInput struct {
	// Path is repo-relative with forward slashes.
	Path     string
	Language string // go, rust, typescript, tsx, javascript, python
	Source   []byte
}

// FileResult is the engine output for one file.
type FileResult struct {
	Path      string     `json:"path"`
	Language  string     `json:"language"`
	LOC       int        `json:"loc"`
	IsTest    bool       `json:"isTest,omitempty"`
	Functions []Function `json:"functions"`
	// Unsupported lists smells that cannot be computed reliably for this
	// language; they are reported as unsupported, never as a silent zero.
	Unsupported []string `json:"unsupported,omitempty"`
	// SyntaxErrors counts the ERROR and MISSING nodes of the syntax tree, up to
	// SyntaxErrorCap. A file with any is analysed as far as the parser recovered
	// it and carries a health.syntax_error diagnostic.
	SyntaxErrors int `json:"syntaxErrors,omitempty"`
}

// Engine analyses source files. Implementations must be deterministic: the
// same inputs and Config give the same results in the same order (files in
// input order, functions in source order, findings in the order of Codes()).
type Engine interface {
	Name() string
	Version() string
	Analyze(inputs []FileInput, cfg Config) ([]FileResult, error)
}

// Bands of a file score (ADR-EMG-048, OWNER items).
const (
	BandGreen  = "green"  // score >= 8.0
	BandYellow = "yellow" // score >= 4.0
	BandRed    = "red"    // score < 4.0
)

// Deduction is one finding's contribution to a file score; a score always
// carries the deductions that produced it.
type Deduction struct {
	Code     string   `json:"code"`
	Function string   `json:"function"`
	Severity Severity `json:"severity"`
	// Points is weight(code) x severity factor (warn 1, high 2), before the
	// diminishing-returns combination across functions.
	Points float64 `json:"points"`
	// Counted is what the combination left of it (worst function x1, next
	// x0.5, the rest x0.25).
	Counted float64 `json:"counted"`
}

// FileScore is the explainable score of one file.
type FileScore struct {
	Score      float64     `json:"score"` // 1.0 .. 10.0, one decimal
	Band       string      `json:"band"`
	Deductions []Deduction `json:"deductions,omitempty"`
}

// Scorer turns an engine result into a score. ScoreFile (scoring.go) is the
// implementation; the delta check takes a Scorer so that it does not depend
// on the weights table.
type Scorer func(FileResult) FileScore
