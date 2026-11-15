// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	"fmt"
	"sort"

	"github.com/labeth/engineering-model-go/internal/health"
)

const (
	// primitiveMinParameters is the smallest parameter list primitive obsession
	// is reported for, primitiveShare the percentage of primitive parameters.
	primitiveMinParameters = 4
	primitiveShare         = 75
)

// numericSmell binds a finding code to the metric it thresholds.
type numericSmell struct {
	code, metric string
	value        func(f *health.Function) int
	testOnly     bool
}

// numericSmells lists the smells decided by one number, in report order.
var numericSmells = []numericSmell{
	{code: health.CodeComplexMethod, metric: "cyclomatic", value: func(f *health.Function) int { return f.Cyclomatic }},
	{code: health.CodeLargeMethod, metric: "loc", value: func(f *health.Function) int { return f.LOC }},
	{code: health.CodeDeepNesting, metric: "nesting", value: func(f *health.Function) int { return f.MaxNesting }},
	{code: health.CodeBumpyRoad, metric: "bumps", value: func(f *health.Function) int { return f.Bumps }},
	{code: health.CodeComplexConditional, metric: "boolean_operators", value: func(f *health.Function) int { return f.MaxBoolOps }},
	{code: health.CodeExcessArguments, metric: "parameters", value: func(f *health.Function) int { return f.Parameters }},
	{code: health.CodeLargeAssertionBlock, metric: "assertions", value: func(f *health.Function) int { return f.Assertions }, testOnly: true},
}

// severityOf maps a value to a severity: at or above High is high, at or above
// Warn is warn, below is none.
// TRLC-LINKS: REQ-EMG-150
func severityOf(value int, t health.Threshold) (health.Severity, bool) {
	switch {
	case value >= t.High:
		return health.SeverityHigh, true
	case value >= t.Warn:
		return health.SeverityWarn, true
	}
	return "", false
}

// findingContext is what the findings of a function depend on besides the
// function: whether primitive obsession is computable for the file, the
// duplicated assertion blocks found at file level and the thresholds.
type findingContext struct {
	primitiveOK bool
	duplicates  []health.Finding
	thresholds  map[string]health.Threshold
}

// functionFindings returns the findings of one function in the order of
// health.Codes().
// TRLC-LINKS: REQ-EMG-150
func functionFindings(u *unitRecord, fc findingContext) []health.Finding {
	f := &u.fn
	out, over := numericFindings(f, fc.thresholds)
	if fc.primitiveOK && primitiveObsessed(u) {
		out = append(out, primitiveFinding(u))
	}
	if over[health.CodeLargeMethod] && over[health.CodeComplexMethod] && over[health.CodeDeepNesting] {
		out = append(out, brainFinding(f))
	}
	out = append(out, fc.duplicates...)
	sort.SliceStable(out, func(i, j int) bool { return codeRank[out[i].Code] < codeRank[out[j].Code] })
	return out
}

// numericFindings returns the findings decided by one number, and the set of
// codes that were found.
// TRLC-LINKS: REQ-EMG-150
func numericFindings(f *health.Function, thresholds map[string]health.Threshold) ([]health.Finding, map[string]bool) {
	var out []health.Finding
	over := map[string]bool{}
	for _, s := range numericSmells {
		if finding, ok := smellFinding(s, f, thresholds); ok {
			over[s.code] = true
			out = append(out, finding)
		}
	}
	return out, over
}

// smellFinding is the finding of one numeric smell of a function, if its value
// reaches the threshold.
// TRLC-LINKS: REQ-EMG-150
func smellFinding(s numericSmell, f *health.Function, thresholds map[string]health.Threshold) (health.Finding, bool) {
	t, ok := thresholds[s.code]
	if !ok || (s.testOnly && !f.IsTest) {
		return health.Finding{}, false
	}
	value := s.value(f)
	severity, found := severityOf(value, t)
	if !found {
		return health.Finding{}, false
	}
	return health.Finding{
		Code: s.code, Severity: severity, StartLine: f.StartLine, EndLine: f.EndLine,
		Metric: s.metric, Value: value,
		Message: fmt.Sprintf("%s %d in %s (warn at %d, high at %d)", s.metric, value, f.Name, t.Warn, t.High),
	}, true
}

// primitiveObsessed reports whether the parameters of a function are mostly
// primitive types.
// TRLC-LINKS: REQ-EMG-150
func primitiveObsessed(u *unitRecord) bool {
	return u.fn.Parameters >= primitiveMinParameters && u.params.primitive*100 >= primitiveShare*u.fn.Parameters
}

// primitiveFinding is the primitive obsession finding of a function.
// TRLC-LINKS: REQ-EMG-150
func primitiveFinding(u *unitRecord) health.Finding {
	f := &u.fn
	return health.Finding{
		Code: health.CodePrimitiveObsession, Severity: health.SeverityWarn, StartLine: f.StartLine, EndLine: f.EndLine,
		Metric: "primitive_parameters", Value: u.params.primitive,
		Message: fmt.Sprintf("%d of %d parameters of %s are primitive types", u.params.primitive, f.Parameters, f.Name),
	}
}

// brainFinding is the brain method finding of a function that is large,
// complex and deeply nested.
// TRLC-LINKS: REQ-EMG-150
func brainFinding(f *health.Function) health.Finding {
	return health.Finding{
		Code: health.CodeBrainMethod, Severity: health.SeverityHigh, StartLine: f.StartLine, EndLine: f.EndLine,
		Metric: "cyclomatic", Value: f.Cyclomatic,
		Message: fmt.Sprintf("%s is large (%d lines), complex (cyclomatic %d) and deeply nested (%d)", f.Name, f.LOC, f.Cyclomatic, f.MaxNesting),
	}
}

// codeRank maps a finding code to its position in health.Codes(); it is only
// looked up, never iterated, so output order stays deterministic.
var codeRank = func() map[string]int {
	ranks := map[string]int{}
	for i, code := range health.Codes() {
		ranks[code] = i
	}
	return ranks
}()
