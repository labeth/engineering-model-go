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

// functionFindings returns the findings of one function in the order of
// health.Codes(). duplicates are the duplicated assertion blocks found at file
// level; primitiveOK says whether primitive obsession is computable for the file.
// TRLC-LINKS: REQ-EMG-150
func functionFindings(u *unitRecord, primitiveOK bool, duplicates []health.Finding, thresholds map[string]health.Threshold) []health.Finding {
	f := &u.fn
	var out []health.Finding
	over := map[string]bool{}
	for _, s := range numericSmells {
		t, ok := thresholds[s.code]
		if !ok || (s.testOnly && !f.IsTest) {
			continue
		}
		value := s.value(f)
		severity, found := severityOf(value, t)
		if !found {
			continue
		}
		over[s.code] = true
		out = append(out, health.Finding{
			Code: s.code, Severity: severity, StartLine: f.StartLine, EndLine: f.EndLine,
			Metric: s.metric, Value: value,
			Message: fmt.Sprintf("%s %d in %s (warn at %d, high at %d)", s.metric, value, f.Name, t.Warn, t.High),
		})
	}
	if primitiveOK && f.Parameters >= primitiveMinParameters && u.params.primitive*100 >= primitiveShare*f.Parameters {
		out = append(out, health.Finding{
			Code: health.CodePrimitiveObsession, Severity: health.SeverityWarn, StartLine: f.StartLine, EndLine: f.EndLine,
			Metric: "primitive_parameters", Value: u.params.primitive,
			Message: fmt.Sprintf("%d of %d parameters of %s are primitive types", u.params.primitive, f.Parameters, f.Name),
		})
	}
	if over[health.CodeLargeMethod] && over[health.CodeComplexMethod] && over[health.CodeDeepNesting] {
		out = append(out, health.Finding{
			Code: health.CodeBrainMethod, Severity: health.SeverityHigh, StartLine: f.StartLine, EndLine: f.EndLine,
			Metric: "cyclomatic", Value: f.Cyclomatic,
			Message: fmt.Sprintf("%s is large (%d lines), complex (cyclomatic %d) and deeply nested (%d)", f.Name, f.LOC, f.Cyclomatic, f.MaxNesting),
		})
	}
	out = append(out, duplicates...)
	sort.SliceStable(out, func(i, j int) bool { return codeRank[out[i].Code] < codeRank[out[j].Code] })
	return out
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
