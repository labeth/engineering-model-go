// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"bufio"
	"os"
	"strings"
)

// generatedHeaderLines is how many leading lines IsGeneratedSource reads: the
// Go convention puts the "Code generated ... DO NOT EDIT." line before the
// package clause, and other generators write theirs first.
const generatedHeaderLines = 10

// IsGeneratedSource reports whether a source file announces that a tool wrote
// it: a leading comment line with "Code generated" and "DO NOT EDIT" (the Go
// convention) or "@generated". A generated file carries no
// ENGMODEL-OWNER-UNIT marker, so the owner check leaves it out. An unreadable
// file is not generated.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func IsGeneratedSource(absPath string) bool {
	f, err := os.Open(absPath)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for n := 0; n < generatedHeaderLines && scanner.Scan(); n++ {
		if isGeneratedHeaderLine(scanner.Text()) {
			return true
		}
	}
	return false
}

// isGeneratedHeaderLine reports whether one line is a generated-code comment.
// TRLC-LINKS: REQ-EMG-010
func isGeneratedHeaderLine(line string) bool {
	text := strings.TrimSpace(line)
	if !strings.HasPrefix(text, "//") && !strings.HasPrefix(text, "#") && !strings.HasPrefix(text, "/*") {
		return false
	}
	return strings.Contains(text, "@generated") || (strings.Contains(text, "Code generated") && strings.Contains(text, "DO NOT EDIT"))
}
