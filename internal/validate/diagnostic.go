// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
)

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type Diagnostic struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Path     string   `json:"path,omitempty"`
	// Deferred marks a warning the life-cycle tier policy defers: evidence
	// debt of a tier below the one it is required from, or covered by an
	// open POA&M item (PoamRef). It never fails a check.
	Deferred bool `json:"deferred,omitempty"`
	// RequiredFrom is the tier from which the policy makes the code an
	// error (empty when it never does or the model has no lifecycle).
	RequiredFrom string `json:"requiredFrom,omitempty"`
	// PoamRef is the POA&M item that defers the diagnostic.
	PoamRef string `json:"poamRef,omitempty"`
	// Remedy says how to fix the diagnostic: the skill to run or the
	// command to execute. It is set for the codes of the required-switch
	// table (remedies.go) and printed as "fix: <remedy>" by every CLI.
	Remedy string `json:"remedy,omitempty"`
}

// Line renders a diagnostic as "code [severity] message (path)", the form
// the CLIs print; the path is left out when empty. Message, path and code
// carry model strings, so control characters in them (newline, carriage
// return, ESC, DEL, other C0 and C1 controls, line separators and invisible
// format characters such as bidi overrides) print as visible escapes: a model
// string cannot start a forged "  fix:" line or drive the terminal. JSON
// output keeps the raw strings, which JSON escapes itself.
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-125
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (d Diagnostic) Line() string {
	line := fmt.Sprintf("%s [%s] %s", escapeControl(d.Code), d.Severity, escapeControl(d.Message))
	if d.Path != "" {
		line += " (" + escapeControl(d.Path) + ")"
	}
	return line
}

// escapeControl replaces the characters that could forge a line or drive a
// terminal with a visible escape: \n, \r, \t as such and the others as \xNN
// or \uNNNN.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func escapeControl(s string) string {
	if !strings.ContainsFunc(s, unsafeRune) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		writeEscapedRune(&b, r)
	}
	return b.String()
}

// shortEscapes maps the control characters that keep a short escape.
var shortEscapes = map[rune]string{'\n': `\n`, '\r': `\r`, '\t': `\t`}

// writeEscapedRune writes r to b, escaped when escapeControl would escape it.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func writeEscapedRune(b *strings.Builder, r rune) {
	if short, ok := shortEscapes[r]; ok {
		b.WriteString(short)
		return
	}
	if !unsafeRune(r) {
		b.WriteRune(r)
		return
	}
	if r < 0x100 {
		fmt.Fprintf(b, `\x%02x`, r)
		return
	}
	fmt.Fprintf(b, `\u%04x`, r)
}

// unsafeRune reports whether a rune is escaped by escapeControl.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func unsafeRune(r rune) bool {
	return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || isLineSeparator(r)
}

// isLineSeparator reports whether r is the Unicode line or paragraph separator.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func isLineSeparator(r rune) bool {
	return r == '\u2028' || r == '\u2029'
}

// FixLine returns "  fix: <remedy>" for a diagnostic that has a remedy and
// "" otherwise.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (d Diagnostic) FixLine() string {
	if d.Remedy == "" {
		return ""
	}
	return "  fix: " + escapeControl(d.Remedy)
}

// WriteDiagnostic writes Line and, when the diagnostic has a remedy, a
// following FixLine to w: the common printing form of the CLIs.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func WriteDiagnostic(w io.Writer, d Diagnostic) {
	fmt.Fprintln(w, d.Line())
	if fix := d.FixLine(); fix != "" {
		fmt.Fprintln(w, fix)
	}
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-012
func SortDiagnostics(in []Diagnostic) []Diagnostic {
	out := append([]Diagnostic(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		a := out[i]
		b := out[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Message != b.Message {
			return a.Message < b.Message
		}
		return string(a.Severity) < string(b.Severity)
	})
	if len(out) < 2 {
		return out
	}
	unique := out[:0]
	var prev Diagnostic
	for i, d := range out {
		if i > 0 && d.Code == prev.Code && d.Severity == prev.Severity && d.Message == prev.Message && d.Path == prev.Path {
			continue
		}
		unique = append(unique, d)
		prev = d
	}
	return unique
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// ENGMODEL-LINKS: TR-MODEL-VALIDATION-FAILS, TR-MODEL-VALIDATION-RECOVERS
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func HasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}
