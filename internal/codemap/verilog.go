// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/labeth/engineering-model-go/internal/validate"
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
	lexer := &verilogLexer{
		path:             path,
		src:              src,
		line:             1,
		comments:         map[int]bool{},
		tokens:           []verilogToken{},
		diags:            []validate.Diagnostic{},
		decls:            []declaration{},
		declarationLines: map[int]bool{},
	}
	for lexer.i < len(src) {
		lexer.step()
	}
	for i, t := range lexer.tokens {
		lexer.declare(i, t)
	}
	if lexer.active {
		lexer.problem(lexer.line, "module missing endmodule")
	}
	return lexer.decls, lexer.comments, lexer.diags, nil
}

// verilogLexer reads Verilog source into position-keeping tokens and then
// the module declarations among them.
type verilogLexer struct {
	path             string
	src              []byte
	i                int
	line             int
	moduleOpen       bool
	active           bool
	comments         map[int]bool
	tokens           []verilogToken
	diags            []validate.Diagnostic
	decls            []declaration
	declarationLines map[int]bool
}

// problem reports a Verilog parse error at a line.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) problem(at int, message string) {
	l.diags = append(l.diags, validate.Diagnostic{Code: "code.verilog_parse_error", Severity: validate.SeverityError, Message: message, Path: fmt.Sprintf("%s:%d", l.path, at)})
}

// at reports whether the source continues with prefix at the cursor.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) at(prefix string) bool {
	return bytes.HasPrefix(l.src[l.i:], []byte(prefix))
}

// advance moves past one byte, counting a newline.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) advance() {
	if l.src[l.i] == '\n' {
		l.line++
	}
	l.i++
}

// skipWhile moves past the bytes that satisfy keep.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) skipWhile(keep func(byte) bool) {
	for l.i < len(l.src) && keep(l.src[l.i]) {
		l.i++
	}
}

// step reads the next white space, comment, string, directive or token.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) step() {
	c := l.src[l.i]
	switch {
	case unicode.IsSpace(rune(c)):
		l.advance()
	case l.at("//"):
		l.comments[l.line] = true
		l.skipWhile(func(b byte) bool { return b != '\n' })
	case l.at("/*"):
		l.blockComment()
	case c == '"':
		l.stringLiteral()
	case c == '`':
		l.directive()
	default:
		l.token(c)
	}
}

// blockComment skips a block comment, marking each of its lines.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) blockComment() {
	start := l.line
	l.i += 2
	for l.i < len(l.src) {
		l.comments[l.line] = true
		if l.at("*/") {
			l.i += 2
			return
		}
		l.advance()
	}
	l.problem(start, "unterminated Verilog block comment")
}

// stringLiteral skips a string literal and its escapes.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) stringLiteral() {
	start := l.line
	l.i++
	for l.i < len(l.src) {
		c := l.src[l.i]
		if c == '"' {
			l.i++
			return
		}
		if c == '\\' && l.i+1 < len(l.src) {
			l.i++
		}
		l.advance()
	}
	l.problem(start, "unterminated Verilog string")
}

// verilogCompilerDirectives are the directives that carry no module text.
var verilogCompilerDirectives = []string{"define", "include", "timescale", "default_nettype", "undef", "ifdef", "ifndef", "elsif", "else", "endif", "resetall", "celldefine", "endcelldefine", "unconnected_drive", "nounconnected_drive", "line"}

// directive handles a backtick directive or macro invocation. Definitions may
// contain apparent module text; do not count their bodies. A macro invocation
// that creates declarations cannot be resolved lexically.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) directive() {
	start := l.line
	l.i++
	begin := l.i
	l.skipWhile(isVerilogIdentifierPart)
	name := string(l.src[begin:l.i])
	if !slices.Contains(verilogCompilerDirectives, name) {
		// These positions require an expression in a valid module. Keep
		// scanning the rest of the line so a literal endmodule is not lost.
		// We do not expand the macro or claim that its value is validated.
		if l.expressionMacro(name, start) {
			return
		}
		l.problem(start, "unresolved Verilog macro invocation: "+name)
	}
	for l.i < len(l.src) && !l.endsDirective() {
		l.advance()
	}
}

// endsDirective reports whether the cursor is at a newline that is not
// escaped.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) endsDirective() bool {
	return l.src[l.i] == '\n' && !(l.i > 0 && l.src[l.i-1] == '\\')
}

// expressionMacro records a macro that stands where a module expression is
// required, as a token and a warning, and reports whether it did.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) expressionMacro(name string, start int) bool {
	if name == "" || !l.moduleOpen || !verilogExpressionMacroPosition(l.tokens, l.src[l.i:]) {
		return false
	}
	l.diags = append(l.diags, validate.Diagnostic{Code: "code.verilog_expression_macro", Severity: validate.SeverityWarning,
		Message: "Unexpanded Verilog expression macro requires build evidence: " + name, Path: fmt.Sprintf("%s:%d", l.path, start)})
	l.tokens = append(l.tokens, verilogToken{"`" + name, start})
	return true
}

// token reads one escaped identifier, identifier or punctuation byte.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) token(c byte) {
	start := l.i
	l.i++
	switch {
	case c == '\\':
		l.skipWhile(func(b byte) bool { return !unicode.IsSpace(rune(b)) })
	case isVerilogIdentifierStart(c):
		l.skipWhile(isVerilogIdentifierPart)
	}
	text := string(l.src[start:l.i])
	l.tokens = append(l.tokens, verilogToken{text, l.line})
	switch text {
	case "module", "macromodule":
		l.moduleOpen = true
	case "endmodule":
		l.moduleOpen = false
	}
}

// isVerilogIdentifierStart reports whether c may start an identifier.
// TRLC-LINKS: REQ-EMG-010
func isVerilogIdentifierStart(c byte) bool {
	return c == '_' || c|0x20 >= 'a' && c|0x20 <= 'z'
}

// isVerilogIdentifierPart reports whether c may continue an identifier.
// TRLC-LINKS: REQ-EMG-010
func isVerilogIdentifierPart(c byte) bool {
	return isVerilogIdentifierStart(c) || c == '$' || isASCIIDigit(c)
}

// isASCIIDigit reports whether c is a decimal digit.
// TRLC-LINKS: REQ-EMG-010
func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// declare handles one token of the second pass: a module keyword opens a
// declaration and endmodule closes it.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) declare(i int, t verilogToken) {
	switch t.text {
	case "endmodule":
		if !l.active {
			l.problem(t.line, "endmodule without module")
		}
		l.active = false
	case "module", "macromodule":
		l.openModule(i, t)
	}
}

// openModule declares the module whose keyword is the token at i.
// TRLC-LINKS: REQ-EMG-010
func (l *verilogLexer) openModule(i int, t verilogToken) {
	if l.active {
		l.problem(t.line, "nested module or missing endmodule")
	}
	l.active = true
	j := i + 1
	if j < len(l.tokens) && slices.Contains([]string{"automatic", "static"}, l.tokens[j].text) {
		j++
	}
	if j >= len(l.tokens) || !verilogIdentifier(l.tokens[j].text) {
		l.problem(t.line, "module has no literal identifier")
		return
	}
	if l.declarationLines[t.line] {
		l.problem(t.line, "multiple module declarations on one line cannot receive distinct trace links")
	}
	l.declarationLines[t.line] = true
	name := l.tokens[j].text
	l.decls = append(l.decls, declaration{Name: name, Kind: "module_declaration", Line: t.line, Signature: t.text + " " + name, RequiresTRLC: true})
}

// TRLC-LINKS: REQ-EMG-010
func verilogIdentifier(s string) bool {
	if slices.Contains([]string{"", "module", "macromodule", "endmodule"}, s) {
		return false
	}
	if strings.HasPrefix(s, "\\") {
		return len(s) > 1
	}
	return isVerilogIdentifierStart(s[0])
}

// verilogValuePrecedents are the tokens after which a value is expected.
var verilogValuePrecedents = []string{"=", "[", "{", ":", "+", "-", "*", "/", "%", "&", "|", "^", "~", "!", "?", "<", ">"}

// verilogBodyState tracks, over the tokens before a macro, whether the module
// header has ended, how deep the parentheses are and how many case statements
// are open.
type verilogBodyState struct {
	body  bool
	depth int
	cases int
}

// note advances the state over one token.
// TRLC-LINKS: REQ-EMG-010
func (s *verilogBodyState) note(text string) {
	switch text {
	case "module", "macromodule", "endmodule":
		*s = verilogBodyState{}
	case ";":
		s.body = true
	case "(":
		s.depth++
	case ")":
		s.depth = max(s.depth-1, 0)
	case "case", "casez", "casex":
		s.cases++
	case "endcase":
		s.cases = max(s.cases-1, 0)
	}
}

// verilogExpressionMacroPosition recognizes value positions only. It does not
// expand macros or validate their values; declarations still need literal names.
// TRLC-LINKS: REQ-EMG-010
func verilogExpressionMacroPosition(tokens []verilogToken, following []byte) bool {
	if len(tokens) == 0 {
		return false
	}
	previous := tokens[len(tokens)-1].text
	if slices.Contains(verilogValuePrecedents, previous) {
		return true
	}
	var state verilogBodyState
	for _, token := range tokens {
		state.note(token.text)
	}
	return state.macroInBody(previous, following)
}

// macroInBody reports whether a macro after the previous token and before
// the following text stands in a value position of the module body.
// TRLC-LINKS: REQ-EMG-010
func (s verilogBodyState) macroInBody(previous string, following []byte) bool {
	// Parenthesized values and argument lists are accepted only after the module
	// header has ended. A macro-generated port declaration remains ambiguous.
	if s.body && s.depth > 0 && slices.Contains([]string{"(", ","}, previous) {
		return true
	}
	// A bare macro followed by a colon inside a case is a case-item value,
	// not a macro-generated declaration. Keep its remaining statement tokens.
	return s.body && s.cases > 0 && strings.HasPrefix(strings.TrimSpace(string(following)), ":")
}
