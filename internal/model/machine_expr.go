// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MachineExprKind selects the entry rule of ParseMachineExpr.
type MachineExprKind string

const (
	// MachineExprBoolean parses one boolean-position expression (a guard or
	// an invariant). ":=" is rejected.
	MachineExprBoolean MachineExprKind = "boolean"
	// MachineExprEffect parses ";"-separated "variable := expression"
	// assignments (an optional trailing ";") into one "all" node.
	MachineExprEffect MachineExprKind = "effect"
)

// MachineExprDiagnosticCode is the code of every MachineExprDiagnostic. It
// reuses the existing semantic.invalid_expression namespace entry.
const MachineExprDiagnosticCode = "semantic.invalid_expression"

// Kind markers ParseMachineExpr sets on SemanticExpression nodes so that
// consumers can tell leaves from operators without inspecting operands.
const (
	// MachineKindOperator marks a node with Operator and Operands.
	MachineKindOperator = "operator"
	// MachineKindLiteral marks an integer, boolean or string literal; the
	// value lives in TypedValue (kind integer, boolean or string).
	MachineKindLiteral = "literal"
	// MachineKindVariable marks a machine variable; Value is its name. It is
	// deliberately not a typedValue reference, which must resolve to an
	// element ID.
	MachineKindVariable = "variable"
	// MachineKindReference marks the state operand of in_state; TypedValue
	// is a reference whose Reference is the state ID.
	MachineKindReference = "reference"
)

// Limits on hostile or accidental input. The parser is recursive descent, so
// unbounded nesting would exhaust the Go stack, which is a fatal error and not
// a recoverable panic. Both limits produce a MachineExprDiagnostic.
const (
	// MaxMachineExprSourceBytes is the longest source ParseMachineExpr accepts
	// (8 KiB). It also bounds the operands of a left-nested chain such as
	// "a + a + a ..." to 4096, and with it the height of any parsed tree.
	MaxMachineExprSourceBytes = 8 << 10
	// MaxMachineExprDepth is the deepest nesting ParseMachineExpr accepts,
	// counted across parentheses, prefix operators (not, unary minus) and the
	// right-associative implies.
	MaxMachineExprDepth = 64
)

// MachineExprDiagnostic is one syntax or static problem found while parsing a
// machine expression. Line and Column are 1-based; Column counts runes from
// the start of the line, so multi-line YAML block strings are located exactly.
//
// TRLC-LINKS: REQ-EMG-136
type MachineExprDiagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
}

// String renders the diagnostic as "line:column: message".
//
// TRLC-LINKS: REQ-EMG-136
func (d MachineExprDiagnostic) String() string {
	return fmt.Sprintf("%d:%d: %s", d.Line, d.Column, d.Message)
}

var machineKeywords = map[string]bool{
	"and": true, "or": true, "not": true, "implies": true,
	"true": true, "false": true, "in_state": true,
}

// IsMachineKeyword reports whether name is reserved and so cannot be a
// machine variable name.
//
// TRLC-LINKS: REQ-EMG-136
func IsMachineKeyword(name string) bool { return machineKeywords[name] }

type machineTokenKind int

const (
	mtEOF machineTokenKind = iota
	mtInt
	mtString
	mtIdent
	mtPunct
	mtStateID // raw state ID following "in_state("; text may be empty
	mtError   // lexical error; err describes it
)

type machineToken struct {
	kind machineTokenKind
	text string // identifier, punctuation or state ID text; decoded string value
	num  int64
	// minMagnitude marks the literal 9223372036854775808, which only fits an
	// int64 as the operand of a unary minus (num is then math.MinInt64).
	minMagnitude bool
	start, end   int // rune offsets into the source, end exclusive
	err          string
}

// is reports whether t has the given kind and text.
//
// TRLC-LINKS: REQ-EMG-136
func (t machineToken) is(kind machineTokenKind, text string) bool {
	return t.kind == kind && t.text == text
}

type machineSyntaxError struct {
	pos int
	msg string
}

// ParseMachineExpr parses src, a guard, invariant or effect of a behavior
// machine, into the structured SemanticExpression tree.
//
// The grammar, from lowest to highest precedence:
//
//	implies            (right associative)
//	or                 (left)
//	and                (left)
//	not                (prefix)
//	== != < <= > >=    (non-associative)
//	+ -                (left)
//	*                  (left)
//	- (unary, "neg")
//	primary: integer, true, false, "string", variable, in_state(ID), (expr)
//
// Whitespace is insignificant. Types are not checked here. A boolean parse
// returns the expression tree. An effect parse always returns an "all" node
// whose operands are ":=" nodes (variable leaf, value expression), even for a
// single assignment, so generators handle one shape. Each variable may be
// assigned once per effect.
//
// On any diagnostic the returned expression is the zero SemanticExpression.
// Syntax errors stop at the first one; duplicate assignments are all reported.
// An empty string literal is rejected because SemanticValue cannot represent
// it (validateSemanticValue requires a non-empty string value).
//
// A source longer than MaxMachineExprSourceBytes, or nested deeper than
// MaxMachineExprDepth, is rejected with a diagnostic and not parsed further.
// The literal -9223372036854775808 (a unary minus applied to 9223372036854775808)
// is the integer minimum; 9223372036854775808 alone is out of range.
//
// TRLC-LINKS: REQ-EMG-136
func ParseMachineExpr(src string, kind MachineExprKind) (SemanticExpression, []MachineExprDiagnostic) {
	if len(src) > MaxMachineExprSourceBytes {
		return SemanticExpression{}, []MachineExprDiagnostic{machineSourceTooLong(src)}
	}
	runes := []rune(src)
	p := &machineParser{src: runes, toks: lexMachineExpr(runes)}
	var expr SemanticExpression
	var err *machineSyntaxError
	switch kind {
	case MachineExprBoolean:
		expr, err = p.parseBoolean()
	case MachineExprEffect:
		expr, err = p.parseEffect()
	default:
		return SemanticExpression{}, []MachineExprDiagnostic{{
			Code: MachineExprDiagnosticCode, Message: fmt.Sprintf("unknown machine expression kind %q", string(kind)), Line: 1, Column: 1,
		}}
	}
	if err != nil {
		p.diags = append(p.diags, p.diagnostic(err.pos, err.msg))
	}
	if len(p.diags) > 0 {
		return SemanticExpression{}, p.diags
	}
	return expr, nil
}

// machineSourceTooLong reports the first byte beyond MaxMachineExprSourceBytes.
// It walks only the accepted prefix, so a huge source costs no more than a
// limit-sized one.
//
// TRLC-LINKS: REQ-EMG-136
func machineSourceTooLong(src string) MachineExprDiagnostic {
	line, column := 1, 1
	for off, r := range src {
		if _, size := utf8.DecodeRuneInString(src[off:]); off+size > MaxMachineExprSourceBytes {
			break
		}
		if r == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return MachineExprDiagnostic{
		Code:    MachineExprDiagnosticCode,
		Message: fmt.Sprintf("expression is %d bytes long; the limit is %d bytes (8 KiB)", len(src), MaxMachineExprSourceBytes),
		Line:    line, Column: column,
	}
}

// machineSpan is a half-open rune range of the source.
type machineSpan struct{ start, end int }

// machineLexer scans a machine expression one token at a time.
type machineLexer struct {
	src  []rune
	toks []machineToken
	i    int
}

// noMachineRune is returned by machineRuneAt past the end of the source.
const noMachineRune rune = -1

// Character classes of the machine expression lexer, as ASCII range tables.
var (
	machineDigitRunes = &unicode.RangeTable{R16: []unicode.Range16{{Lo: '0', Hi: '9', Stride: 1}}}
	machineIdentRunes = &unicode.RangeTable{R16: []unicode.Range16{
		{Lo: '0', Hi: '9', Stride: 1}, {Lo: 'A', Hi: 'Z', Stride: 1}, {Lo: '_', Hi: '_', Stride: 1}, {Lo: 'a', Hi: 'z', Stride: 1},
	}}
	machineStartRunes = &unicode.RangeTable{R16: []unicode.Range16{
		{Lo: 'A', Hi: 'Z', Stride: 1}, {Lo: '_', Hi: '_', Stride: 1}, {Lo: 'a', Hi: 'z', Stride: 1},
	}}
	machineStateIDRunes = &unicode.RangeTable{R16: []unicode.Range16{
		{Lo: '-', Hi: '-', Stride: 1}, {Lo: '0', Hi: '9', Stride: 1}, {Lo: 'A', Hi: 'Z', Stride: 1},
		{Lo: '_', Hi: '_', Stride: 1}, {Lo: 'a', Hi: 'z', Stride: 1},
	}}
)

const (
	machineOnePunct = "();+-*<>"
	machineTwoFirst = "=!<>:"
)

// lexMachineExpr tokenizes src. It stops after the first lexical error,
// appending an mtError token, so the parser reports the earliest problem.
//
// TRLC-LINKS: REQ-EMG-136
func lexMachineExpr(src []rune) []machineToken {
	lx := &machineLexer{src: src}
	for lx.step() {
	}
	return lx.toks
}

// step lexes one token and reports whether more input may follow.
//
// TRLC-LINKS: REQ-EMG-136
func (lx *machineLexer) step() bool {
	for lx.i < len(lx.src) && unicode.IsSpace(lx.src[lx.i]) {
		lx.i++
	}
	if lx.afterInState() {
		sp := lx.take(machineStateIDRunes)
		lx.toks = append(lx.toks, machineToken{kind: mtStateID, text: lx.text(sp), start: sp.start, end: sp.end})
		return true
	}
	if lx.i >= len(lx.src) {
		lx.toks = append(lx.toks, machineToken{kind: mtEOF, start: lx.i, end: lx.i})
		return false
	}
	return lx.lexToken(lx.src[lx.i])
}

// lexToken lexes the token that starts with c at the current position.
//
// TRLC-LINKS: REQ-EMG-136
func (lx *machineLexer) lexToken(c rune) bool {
	switch {
	case unicode.Is(machineDigitRunes, c):
		return lx.lexNumber()
	case isMachineIdentStart(c):
		sp := lx.take(machineIdentRunes)
		lx.toks = append(lx.toks, machineToken{kind: mtIdent, text: lx.text(sp), start: sp.start, end: sp.end})
		return true
	case c == '"':
		tok, next := lexMachineString(lx.src, lx.i)
		lx.toks = append(lx.toks, tok)
		lx.i = next
		return tok.kind != mtError
	default:
		return lx.lexPunct(c)
	}
}

// afterInState reports whether the previous tokens are "in_state" and "(".
//
// TRLC-LINKS: REQ-EMG-136
func (lx *machineLexer) afterInState() bool {
	n := len(lx.toks)
	return n >= 2 && lx.toks[n-1].is(mtPunct, "(") && lx.toks[n-2].is(mtIdent, "in_state")
}

// take consumes the run of runes in class and returns its span.
//
// TRLC-LINKS: REQ-EMG-136
func (lx *machineLexer) take(class *unicode.RangeTable) machineSpan {
	start := lx.i
	for lx.i < len(lx.src) && unicode.Is(class, lx.src[lx.i]) {
		lx.i++
	}
	return machineSpan{start, lx.i}
}

// text returns the source text of sp.
//
// TRLC-LINKS: REQ-EMG-136
func (lx *machineLexer) text(sp machineSpan) string { return string(lx.src[sp.start:sp.end]) }

// lexNumber lexes an integer literal at the current position.
//
// TRLC-LINKS: REQ-EMG-136
func (lx *machineLexer) lexNumber() bool {
	sp := lx.take(machineDigitRunes)
	tok := machineErrorToken(sp, "invalid integer literal: digits followed by a letter")
	if lx.i >= len(lx.src) || !isMachineIdentStart(lx.src[lx.i]) {
		tok = machineIntToken(lx.text(sp), sp)
	}
	lx.toks = append(lx.toks, tok)
	return tok.kind != mtError
}

// machineIntToken converts digits to an integer token, or to an error token
// when the value does not fit. 9223372036854775808 is kept as the minimum
// magnitude, which is valid only as the operand of a unary minus.
//
// TRLC-LINKS: REQ-EMG-136
func machineIntToken(digits string, sp machineSpan) machineToken {
	v, err := strconv.ParseInt(digits, 10, 64)
	if err == nil {
		return machineToken{kind: mtInt, num: v, text: digits, start: sp.start, end: sp.end}
	}
	if u, uerr := strconv.ParseUint(digits, 10, 64); uerr == nil && u == 1<<63 {
		return machineToken{kind: mtInt, num: math.MinInt64, minMagnitude: true, text: digits, start: sp.start, end: sp.end}
	}
	return machineErrorToken(sp, "integer literal out of range")
}

// TRLC-LINKS: REQ-EMG-136
func machineErrorToken(sp machineSpan, msg string) machineToken {
	return machineToken{kind: mtError, start: sp.start, end: sp.end, err: msg}
}

// lexPunct lexes an operator or punctuation token starting with c.
//
// TRLC-LINKS: REQ-EMG-136
func (lx *machineLexer) lexPunct(c rune) bool {
	start := lx.i
	text := ""
	if isMachineTwoRune(c, machineRuneAt(lx.src, start+1)) {
		text = string(lx.src[start : start+2])
	} else if strings.ContainsRune(machineOnePunct, c) {
		text = string(c)
	}
	if text == "" {
		msg := fmt.Sprintf("unexpected character %q", c)
		if c == '=' || c == '!' {
			msg += "; did you mean '==' or '!='"
		}
		lx.toks = append(lx.toks, machineErrorToken(machineSpan{start, start + 1}, msg))
		return false
	}
	lx.i += len([]rune(text))
	lx.toks = append(lx.toks, machineToken{kind: mtPunct, text: text, start: start, end: lx.i})
	return true
}

// TRLC-LINKS: REQ-EMG-136
func machineRuneAt(src []rune, i int) rune {
	if i < 0 || i >= len(src) {
		return noMachineRune
	}
	return src[i]
}

// TRLC-LINKS: REQ-EMG-136
func isMachineTwoRune(a, b rune) bool {
	return b == '=' && strings.ContainsRune(machineTwoFirst, a)
}

// TRLC-LINKS: REQ-EMG-136
func isMachineIdentStart(c rune) bool { return unicode.Is(machineStartRunes, c) }

// lexMachineString scans the string literal starting at the quote at src[start].
//
// TRLC-LINKS: REQ-EMG-136
func lexMachineString(src []rune, start int) (machineToken, int) {
	var value []rune
	i := start + 1
	for !isMachineStringEnd(machineRuneAt(src, i)) {
		if src[i] != '\\' {
			value = append(value, src[i])
			i++
			continue
		}
		esc := machineRuneAt(src, i+1)
		if esc != '"' && esc != '\\' {
			return machineErrorToken(machineSpan{i, i + 1}, `invalid escape sequence; only \" and \\ are supported`), i
		}
		value = append(value, esc)
		i += 2
	}
	if machineRuneAt(src, i) != '"' {
		return machineErrorToken(machineSpan{start, start + 1}, "unterminated string literal"), i
	}
	return machineToken{kind: mtString, text: string(value), start: start, end: i + 1}, i + 1
}

// TRLC-LINKS: REQ-EMG-136
func isMachineStringEnd(c rune) bool { return c == noMachineRune || c == '"' || c == '\n' }

type machineParser struct {
	src   []rune
	toks  []machineToken
	i     int
	depth int // current nesting, see MaxMachineExprDepth
	diags []MachineExprDiagnostic
}

// enter counts one level of nesting opened by tok and fails beyond
// MaxMachineExprDepth. Every successful enter is paired with leave.
//
// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) enter(tok machineToken) *machineSyntaxError {
	if p.depth >= MaxMachineExprDepth {
		return &machineSyntaxError{pos: tok.start, msg: fmt.Sprintf("expression is nested too deeply; the limit is %d levels of parentheses, 'not', unary '-' and 'implies'", MaxMachineExprDepth)}
	}
	p.depth++
	return nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) leave() { p.depth-- }

// diagnostic converts a rune offset into a 1-based line/column diagnostic.
//
// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) diagnostic(pos int, msg string) MachineExprDiagnostic {
	line, lineStart := 1, 0
	for i := 0; i < pos && i < len(p.src); i++ {
		if p.src[i] == '\n' {
			line++
			lineStart = i + 1
		}
	}
	return MachineExprDiagnostic{Code: MachineExprDiagnosticCode, Message: msg, Line: line, Column: pos - lineStart + 1}
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) cur() machineToken { return p.toks[p.i] }

// advance consumes the current token; the final token is always EOF or an
// error, so it never moves past the end.
//
// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) advance() {
	if p.i < len(p.toks)-1 {
		p.i++
	}
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) isPunct(text string) bool {
	t := p.cur()
	return t.kind == mtPunct && t.text == text
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) isWord(text string) bool {
	t := p.cur()
	return t.kind == mtIdent && t.text == text
}

// errAt reports msg at tok, or the lexical error when tok is one.
//
// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) errAt(tok machineToken, msg string) *machineSyntaxError {
	if tok.kind == mtError {
		return &machineSyntaxError{pos: tok.start, msg: tok.err}
	}
	return &machineSyntaxError{pos: tok.start, msg: msg}
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) describe(tok machineToken) string {
	switch tok.kind {
	case mtEOF:
		return "end of input"
	case mtString:
		return "string literal"
	case mtInt:
		return "integer " + tok.text
	case mtStateID:
		return "state ID"
	default:
		return fmt.Sprintf("%q", string(p.src[tok.start:tok.end]))
	}
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseBoolean() (SemanticExpression, *machineSyntaxError) {
	expr, err := p.parseImplies()
	if err != nil {
		return SemanticExpression{}, err
	}
	if t := p.cur(); t.kind != mtEOF {
		switch {
		case t.kind == mtPunct && t.text == ":=":
			return SemanticExpression{}, p.errAt(t, "':=' is only allowed in effects, not in guards or invariants")
		case t.kind == mtPunct && t.text == ")":
			return SemanticExpression{}, p.errAt(t, "unbalanced parenthesis: unexpected ')'")
		default:
			return SemanticExpression{}, p.errAt(t, "unexpected "+p.describe(t)+" after the end of the expression")
		}
	}
	return expr, nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseEffect() (SemanticExpression, *machineSyntaxError) {
	var assigns []SemanticExpression
	seen := map[string]bool{}
	for {
		assign, err := p.parseAssignment(seen, len(assigns) == 0)
		if err != nil {
			return SemanticExpression{}, err
		}
		assigns = append(assigns, assign)
		if err := p.endAssignment(); err != nil {
			return SemanticExpression{}, err
		}
		if p.cur().kind == mtEOF {
			return machineOperator("all", assigns...), nil
		}
	}
}

// parseAssignment parses one "variable := expression" of an effect. first
// marks the first assignment, where an empty effect is reported.
//
// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseAssignment(seen map[string]bool, first bool) (SemanticExpression, *machineSyntaxError) {
	t, err := p.effectTarget(first)
	if err != nil {
		return SemanticExpression{}, err
	}
	if seen[t.text] {
		p.diags = append(p.diags, p.diagnostic(t.start, fmt.Sprintf("variable %q is assigned more than once in this effect", t.text)))
	}
	seen[t.text] = true
	p.advance()
	if !p.isPunct(":=") {
		return SemanticExpression{}, p.errAt(p.cur(), "unexpected "+p.describe(p.cur())+", expected ':=' after the variable name")
	}
	p.advance()
	value, err := p.parseImplies()
	if err != nil {
		return SemanticExpression{}, err
	}
	return machineOperator(":=", machineVariable(t.text), value), nil
}

// effectTarget returns the current token when it can be assigned to.
//
// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) effectTarget(first bool) (machineToken, *machineSyntaxError) {
	t := p.cur()
	switch {
	case t.kind == mtEOF && first:
		return t, p.errAt(t, "empty effect: expected 'variable := expression'")
	case t.kind != mtIdent:
		return t, p.errAt(t, "unexpected "+p.describe(t)+", expected a variable name")
	case IsMachineKeyword(t.text):
		return t, p.errAt(t, fmt.Sprintf("keyword %q cannot be used as a variable name", t.text))
	}
	return t, nil
}

// endAssignment consumes the ";" after an assignment, which is optional at
// the end of the effect.
//
// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) endAssignment() *machineSyntaxError {
	if p.isPunct(";") {
		p.advance()
		return nil
	}
	if p.cur().kind != mtEOF {
		return p.errAt(p.cur(), "unexpected "+p.describe(p.cur())+", expected ';' or the end of the effect")
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseImplies() (SemanticExpression, *machineSyntaxError) {
	left, err := p.parseOr()
	if err != nil || !p.isWord("implies") {
		return left, err
	}
	if err := p.enter(p.cur()); err != nil {
		return SemanticExpression{}, err
	}
	defer p.leave()
	p.advance()
	right, err := p.parseImplies()
	if err != nil {
		return SemanticExpression{}, err
	}
	return machineOperator("implies", left, right), nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseOr() (SemanticExpression, *machineSyntaxError) {
	left, err := p.parseAnd()
	for err == nil && p.isWord("or") {
		p.advance()
		var right SemanticExpression
		if right, err = p.parseAnd(); err == nil {
			left = machineOperator("or", left, right)
		}
	}
	return left, err
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseAnd() (SemanticExpression, *machineSyntaxError) {
	left, err := p.parseNot()
	for err == nil && p.isWord("and") {
		p.advance()
		var right SemanticExpression
		if right, err = p.parseNot(); err == nil {
			left = machineOperator("and", left, right)
		}
	}
	return left, err
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseNot() (SemanticExpression, *machineSyntaxError) {
	if !p.isWord("not") {
		return p.parseComparison()
	}
	if err := p.enter(p.cur()); err != nil {
		return SemanticExpression{}, err
	}
	defer p.leave()
	p.advance()
	operand, err := p.parseNot()
	if err != nil {
		return SemanticExpression{}, err
	}
	return machineOperator("not", operand), nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseComparison() (SemanticExpression, *machineSyntaxError) {
	left, err := p.parseAdditive()
	if err != nil || !isMachineComparison(p.cur()) {
		return left, err
	}
	op := p.cur().text
	p.advance()
	right, err := p.parseAdditive()
	if err != nil {
		return SemanticExpression{}, err
	}
	if t := p.cur(); isMachineComparison(t) {
		return SemanticExpression{}, p.errAt(t, "comparison operators are not associative; use parentheses")
	}
	return machineOperator(op, left, right), nil
}

// TRLC-LINKS: REQ-EMG-136
func isMachineComparison(t machineToken) bool {
	if t.kind != mtPunct {
		return false
	}
	switch t.text {
	case "==", "!=", "<", "<=", ">", ">=":
		return true
	}
	return false
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseAdditive() (SemanticExpression, *machineSyntaxError) {
	left, err := p.parseMultiplicative()
	for err == nil && (p.isPunct("+") || p.isPunct("-")) {
		op := p.cur().text
		p.advance()
		var right SemanticExpression
		if right, err = p.parseMultiplicative(); err == nil {
			left = machineOperator(op, left, right)
		}
	}
	return left, err
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseMultiplicative() (SemanticExpression, *machineSyntaxError) {
	left, err := p.parseUnary()
	for err == nil && p.isPunct("*") {
		p.advance()
		var right SemanticExpression
		if right, err = p.parseUnary(); err == nil {
			left = machineOperator("*", left, right)
		}
	}
	return left, err
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseUnary() (SemanticExpression, *machineSyntaxError) {
	if !p.isPunct("-") {
		return p.parsePrimary()
	}
	if p.i+1 < len(p.toks) && p.toks[p.i+1].kind == mtInt && p.toks[p.i+1].minMagnitude {
		v := p.toks[p.i+1].num
		p.advance()
		p.advance()
		return SemanticExpression{Kind: MachineKindLiteral, TypedValue: &SemanticValue{Kind: "integer", Integer: &v}}, nil
	}
	if err := p.enter(p.cur()); err != nil {
		return SemanticExpression{}, err
	}
	defer p.leave()
	p.advance()
	operand, err := p.parseUnary()
	if err != nil {
		return SemanticExpression{}, err
	}
	return machineOperator("neg", operand), nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parsePrimary() (SemanticExpression, *machineSyntaxError) {
	t := p.cur()
	switch t.kind {
	case mtInt:
		return p.parseIntLiteral(t)
	case mtString:
		return p.parseStringLiteral(t)
	case mtIdent:
		return p.parseIdentifier(t)
	case mtPunct:
		if t.text == "(" {
			return p.parseParenthesised()
		}
	}
	return SemanticExpression{}, p.errAt(t, "unexpected "+p.describe(t)+", expected an expression")
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseIntLiteral(t machineToken) (SemanticExpression, *machineSyntaxError) {
	if t.minMagnitude {
		return SemanticExpression{}, p.errAt(t, "integer literal out of range")
	}
	p.advance()
	v := t.num
	return SemanticExpression{Kind: MachineKindLiteral, TypedValue: &SemanticValue{Kind: "integer", Integer: &v}}, nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseStringLiteral(t machineToken) (SemanticExpression, *machineSyntaxError) {
	if t.text == "" {
		return SemanticExpression{}, p.errAt(t, "empty string literals are not supported")
	}
	p.advance()
	return SemanticExpression{Kind: MachineKindLiteral, TypedValue: &SemanticValue{Kind: "string", String: t.text}}, nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseIdentifier(t machineToken) (SemanticExpression, *machineSyntaxError) {
	switch t.text {
	case "true", "false":
		p.advance()
		v := t.text == "true"
		return SemanticExpression{Kind: MachineKindLiteral, TypedValue: &SemanticValue{Kind: "boolean", Boolean: &v}}, nil
	case "in_state":
		return p.parseInState()
	}
	if t.text == "not" {
		return SemanticExpression{}, p.errAt(t, "'not' is not allowed here; wrap it in parentheses")
	}
	if IsMachineKeyword(t.text) {
		return SemanticExpression{}, p.errAt(t, fmt.Sprintf("keyword %q cannot be used as a variable name", t.text))
	}
	p.advance()
	return machineVariable(t.text), nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseParenthesised() (SemanticExpression, *machineSyntaxError) {
	if err := p.enter(p.cur()); err != nil {
		return SemanticExpression{}, err
	}
	defer p.leave()
	p.advance()
	inner, err := p.parseImplies()
	if err != nil {
		return SemanticExpression{}, err
	}
	if !p.isPunct(")") {
		return SemanticExpression{}, p.errAt(p.cur(), "unbalanced parenthesis: expected ')' but found "+p.describe(p.cur()))
	}
	p.advance()
	return inner, nil
}

// TRLC-LINKS: REQ-EMG-136
func (p *machineParser) parseInState() (SemanticExpression, *machineSyntaxError) {
	p.advance()
	if !p.isPunct("(") {
		return SemanticExpression{}, p.errAt(p.cur(), "malformed in_state: expected '(' after in_state")
	}
	p.advance()
	id := p.cur()
	if id.kind != mtStateID || id.text == "" {
		return SemanticExpression{}, p.errAt(id, "malformed in_state: expected a state ID of letters, digits, '_' and '-'")
	}
	p.advance()
	if !p.isPunct(")") {
		return SemanticExpression{}, p.errAt(p.cur(), "malformed in_state: expected ')' after the state ID")
	}
	p.advance()
	ref := SemanticExpression{Kind: MachineKindReference, TypedValue: &SemanticValue{Kind: "reference", Reference: id.text}}
	return machineOperator("in_state", ref), nil
}

// TRLC-LINKS: REQ-EMG-136
func machineOperator(op string, operands ...SemanticExpression) SemanticExpression {
	return SemanticExpression{Kind: MachineKindOperator, Operator: op, Operands: operands}
}

// TRLC-LINKS: REQ-EMG-136
func machineVariable(name string) SemanticExpression {
	return SemanticExpression{Kind: MachineKindVariable, Value: name}
}
