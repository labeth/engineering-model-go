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
	switch {
	case p == "" || p == "." || p == "./":
		return "is empty or names the whole module"
	case strings.HasPrefix(p, "/") || filepath.IsAbs(pattern):
		return "is absolute"
	}
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimSuffix(strings.TrimSuffix(p, "/**"), "/")
	for _, segment := range strings.Split(p, "/") {
		if segment == ".." {
			return "leaves the module root"
		}
	}
	if strings.Contains(p, "**") {
		return "uses ** other than as a trailing /**"
	}
	if _, err := path.Match(p, ""); err != nil {
		return "is not a valid glob"
	}
	return ""
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
