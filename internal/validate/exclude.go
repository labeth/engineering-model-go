// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// ExcludeAcceptedForms names the inferenceHints.exclude forms the code scan
// applies; every diagnostic about a pattern quotes it.
const ExcludeAcceptedForms = "dir, dir/, dir/* or dir/** (each leaves the whole directory out), a file path, or a glob with * ? [...] within one path segment such as *.gen.go, relative to the module root"

// ExcludePatternProblem returns why an inferenceHints.exclude pattern cannot
// be applied, or "" when it can: it is empty, absolute, leaves the module
// root (..), uses ** other than as a trailing /**, or is a malformed glob.
// TRLC-LINKS: REQ-EMG-114
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-CODEMAP-INFERENCE
func ExcludePatternProblem(pattern string) string {
	p := filepath.ToSlash(strings.TrimSpace(pattern))
	if problem := excludeFormProblem(p, pattern); problem != "" {
		return problem
	}
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimSuffix(strings.TrimSuffix(p, "/**"), "/")
	switch {
	case hasParentSegment(p):
		return "leaves the module root"
	case strings.Contains(p, "**"):
		return "uses ** other than as a trailing /**"
	}
	if _, err := path.Match(p, ""); err != nil {
		return "is not a valid glob"
	}
	return ""
}

// excludeFormProblem says why a slashed pattern is empty, names the whole
// module or is absolute, or returns "".
// TRLC-LINKS: REQ-EMG-114
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-CODEMAP-INFERENCE
func excludeFormProblem(slashed, pattern string) string {
	if slashed == "" || slashed == "." || slashed == "./" {
		return "is empty or names the whole module"
	}
	if strings.HasPrefix(slashed, "/") || filepath.IsAbs(pattern) {
		return "is absolute"
	}
	return ""
}

// hasParentSegment reports whether a slashed path has a .. segment.
// TRLC-LINKS: REQ-EMG-114
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-CODEMAP-INFERENCE
func hasParentSegment(p string) bool {
	for _, segment := range strings.Split(p, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

// excludePatternDiagnostics reports model.invalid_exclude_pattern (error) at
// each exclude pattern the code scan cannot apply, naming the accepted forms.
// TRLC-LINKS: REQ-EMG-114
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func excludePatternDiagnostics(patterns []string) []Diagnostic {
	var diags []Diagnostic
	for i, pattern := range patterns {
		if problem := ExcludePatternProblem(pattern); problem != "" {
			diags = append(diags, Diagnostic{
				Code: "model.invalid_exclude_pattern", Severity: SeverityError,
				Message: fmt.Sprintf("inferenceHints.exclude pattern %q %s; use %s", pattern, problem, ExcludeAcceptedForms),
				Path:    fmt.Sprintf("inferenceHints.exclude[%d]", i),
			})
		}
	}
	return diags
}
