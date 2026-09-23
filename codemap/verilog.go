// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/labeth/engineering-model-go/validate"
)

// verilogToken retains positions while comments and string literals are omitted.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type verilogToken struct {
	text string
	line int
}

// extractVerilogDeclarations is a lexical module-boundary extractor, not an
// elaborator or HDL syntax validator. Conditional branches are all inspected;
// preprocessing and module-internal behavior require independent build evidence.
// TRLC-LINKS: REQ-EMG-010
func extractVerilogDeclarations(path string, src []byte) ([]declaration, map[int]bool, []validate.Diagnostic, error) {
	comments := map[int]bool{}
	tokens := []verilogToken{}
	diags := []validate.Diagnostic{}
	line := 1
	moduleOpen := false
	problem := func(at int, message string) {
		diags = append(diags, validate.Diagnostic{Code: "code.verilog_parse_error", Severity: validate.SeverityError, Message: message, Path: fmt.Sprintf("%s:%d", path, at)})
	}
	for i := 0; i < len(src); {
		c := src[i]
		if c == '\n' {
			line++
			i++
			continue
		}
		if unicode.IsSpace(rune(c)) {
			i++
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			comments[line] = true
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			start := line
			i += 2
			closed := false
			for i < len(src) {
				comments[line] = true
				if src[i] == '*' && i+1 < len(src) && src[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if !closed {
				problem(start, "unterminated Verilog block comment")
			}
			continue
		}
		if c == '"' {
			start := line
			i++
			closed := false
			for i < len(src) {
				if src[i] == '"' {
					i++
					closed = true
					break
				}
				if src[i] == '\\' && i+1 < len(src) {
					i++
					if src[i] == '\n' {
						line++
					}
					i++
					continue
				}
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if !closed {
				problem(start, "unterminated Verilog string")
			}
			continue
		}
		// Definitions may contain apparent module text; do not count their bodies.
		// A macro invocation that creates declarations cannot be resolved lexically.
		if c == '`' {
			start := line
			i++
			begin := i
			for i < len(src) && (src[i] == '_' || src[i] == '$' || src[i] >= '0' && src[i] <= '9' || src[i] >= 'a' && src[i] <= 'z' || src[i] >= 'A' && src[i] <= 'Z') {
				i++
			}
			directive := string(src[begin:i])
			switch directive {
			case "define", "include", "timescale", "default_nettype", "undef", "ifdef", "ifndef", "elsif", "else", "endif", "resetall", "celldefine", "endcelldefine", "unconnected_drive", "nounconnected_drive", "line":
			default:
				// These positions require an expression in a valid module. Keep
				// scanning the rest of the line so a literal endmodule is not lost.
				// We do not expand the macro or claim that its value is validated.
				expressionPosition := verilogExpressionMacroPosition(tokens, src[i:])
				if directive != "" && moduleOpen && expressionPosition {
					diags = append(diags, validate.Diagnostic{Code: "code.verilog_expression_macro", Severity: validate.SeverityWarning,
						Message: "Unexpanded Verilog expression macro requires build evidence: " + directive, Path: fmt.Sprintf("%s:%d", path, start)})
					tokens = append(tokens, verilogToken{"`" + directive, start})
					continue
				}
				problem(start, "unresolved Verilog macro invocation: "+directive)
			}
			for i < len(src) {
				if src[i] == '\n' {
					if i > 0 && src[i-1] == '\\' {
						line++
						i++
						continue
					}
					break
				}
				i++
			}
			continue
		}
		start := i
		if c == '\\' {
			i++
			for i < len(src) && !unicode.IsSpace(rune(src[i])) {
				i++
			}
		} else if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			i++
			for i < len(src) && (src[i] == '_' || src[i] == '$' || src[i] >= '0' && src[i] <= '9' || src[i] >= 'a' && src[i] <= 'z' || src[i] >= 'A' && src[i] <= 'Z') {
				i++
			}
		} else {
			i++
		}
		text := string(src[start:i])
		tokens = append(tokens, verilogToken{text, line})
		if text == "module" || text == "macromodule" {
			moduleOpen = true
		} else if text == "endmodule" {
			moduleOpen = false
		}
	}
	decls := []declaration{}
	active := false
	declarationLines := map[int]bool{}
	for i, t := range tokens {
		if t.text == "endmodule" {
			if !active {
				problem(t.line, "endmodule without module")
			}
			active = false
			continue
		}
		if t.text != "module" && t.text != "macromodule" {
			continue
		}
		if active {
			problem(t.line, "nested module or missing endmodule")
		}
		active = true
		j := i + 1
		if j < len(tokens) && (tokens[j].text == "automatic" || tokens[j].text == "static") {
			j++
		}
		if j >= len(tokens) || !verilogIdentifier(tokens[j].text) {
			problem(t.line, "module has no literal identifier")
			continue
		}
		if declarationLines[t.line] {
			problem(t.line, "multiple module declarations on one line cannot receive distinct trace links")
		}
		declarationLines[t.line] = true
		name := tokens[j].text
		decls = append(decls, declaration{Name: name, Kind: "module_declaration", Line: t.line, Signature: t.text + " " + name, RequiresTRLC: true})
	}
	if active {
		problem(line, "module missing endmodule")
	}
	return decls, comments, diags, nil
}

// TRLC-LINKS: REQ-EMG-010
func verilogIdentifier(s string) bool {
	if s == "" || s == "module" || s == "macromodule" || s == "endmodule" {
		return false
	}
	if strings.HasPrefix(s, "\\") {
		return len(s) > 1
	}
	c := s[0]
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// verilogExpressionMacroPosition recognizes value positions only. It does not
// expand macros or validate their values; declarations still need literal names.
// TRLC-LINKS: REQ-EMG-010
func verilogExpressionMacroPosition(tokens []verilogToken, following []byte) bool {
	if len(tokens) == 0 {
		return false
	}
	previous := tokens[len(tokens)-1].text
	switch previous {
	case "=", "[", "{", ":", "+", "-", "*", "/", "%", "&", "|", "^", "~", "!", "?", "<", ">":
		return true
	}
	// Parenthesized values and argument lists are accepted only after the module
	// header has ended. A macro-generated port declaration remains ambiguous.
	body, depth, cases := false, 0, 0
	for _, token := range tokens {
		switch token.text {
		case "module", "macromodule", "endmodule":
			body = false
			depth = 0
			cases = 0
		case ";":
			body = true
		case "(":
			depth++
		case ")":
			if depth > 0 {
				depth--
			}
		case "case", "casez", "casex":
			cases++
		case "endcase":
			if cases > 0 {
				cases--
			}
		}
	}
	if body && depth > 0 && (previous == "(" || previous == ",") {
		return true
	}
	// A bare macro followed by a colon inside a case is a case-item value,
	// not a macro-generated declaration. Keep its remaining statement tokens.
	return body && cases > 0 && strings.HasPrefix(strings.TrimSpace(string(following)), ":")
}
