// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

// The body hash of a function (ADR-EMG-053): the delta check uses it to tell
// that a function was renamed or moved with its body unchanged, so its
// findings are existing debt and not new ones. The hash covers the normalized
// token text of the function: comments and white space are gone, line endings
// do not matter, and the function's own name is replaced by a placeholder so
// that a rename does not change the hash.

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

// BodyHash returns the hex sha256 of the normalized tokens of fn in source
// (a function with lines StartLine..EndLine). It is empty when the line range
// holds no tokens, so an invalid range never matches another.
//
// TRLC-LINKS: REQ-EMG-156
func BodyHash(source []byte, fn Function, language string) string {
	text := strings.ReplaceAll(string(source), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	start, end := fn.StartLine, fn.EndLine
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return ""
	}
	simple := fn.Name
	if i := strings.LastIndex(simple, "."); i >= 0 {
		simple = simple[i+1:]
	}
	tokens := bodyTokens(strings.Join(lines[start-1:end], "\n"), language, simple)
	if len(tokens) == 0 {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join(tokens, " ")))
	return hex.EncodeToString(sum[:])
}

// isWordByte reports whether b can be part of an identifier or number.
//
// TRLC-LINKS: REQ-EMG-156
func isWordByte(b byte) bool {
	return strings.IndexByte("_$", b) >= 0 || b >= 0x80 || isASCIIAlnum(b)
}

// isASCIIAlnum reports a byte in 0-9, a-z or A-Z.
//
// TRLC-LINKS: REQ-EMG-156
func isASCIIAlnum(b byte) bool { return isASCIILetter(b) || (b >= '0' && b <= '9') }

// isSpaceByte reports a white space byte.
//
// TRLC-LINKS: REQ-EMG-156
func isSpaceByte(b byte) bool { return strings.IndexByte(" \t\n\v\f", b) >= 0 }

// isQuoteByte reports a byte that opens a string or character literal.
//
// TRLC-LINKS: REQ-EMG-156
func isQuoteByte(b byte) bool { return b == '"' || b == '\'' || b == '`' }

// tokenizer splits the text of a function into tokens.
type tokenizer struct {
	text, language, name string
	tokens               []string
}

// bodyTokens splits text into tokens, dropping comments and white space. The
// language decides what a comment is (Python uses #, the others // and /* */,
// nested in Rust) and how a quote is read (a Rust quote may start a lifetime).
//
// TRLC-LINKS: REQ-EMG-156
func bodyTokens(text, language, name string) []string {
	t := tokenizer{text: text, language: language, name: name}
	for i := 0; i < len(text); {
		i = t.step(i)
	}
	return t.tokens
}

// step reads what starts at i and returns the index after it.
//
// TRLC-LINKS: REQ-EMG-156
func (t *tokenizer) step(i int) int {
	c := t.text[i]
	if isSpaceByte(c) {
		return i + 1
	}
	if end, ok := t.commentEnd(i); ok {
		return end
	}
	switch {
	case isQuoteByte(c):
		return t.emit(i, stringEnd(t.text, i, t.language))
	case isWordByte(c):
		return t.word(i)
	}
	return t.emit(i, i+1)
}

// commentEnd returns the index after the comment that starts at i, if one does.
//
// TRLC-LINKS: REQ-EMG-156
func (t *tokenizer) commentEnd(i int) (int, bool) {
	c := t.text[i]
	if t.language == "python" {
		if c != '#' {
			return 0, false
		}
		return skipLine(t.text, i), true
	}
	if c != '/' || i+1 >= len(t.text) {
		return 0, false
	}
	switch t.text[i+1] {
	case '/':
		return skipLine(t.text, i), true
	case '*':
		return skipBlockComment(t.text, i, t.language == "rust"), true
	}
	return 0, false
}

// emit adds the token text[i:end] and returns end.
//
// TRLC-LINKS: REQ-EMG-156
func (t *tokenizer) emit(i, end int) int {
	t.tokens = append(t.tokens, t.text[i:end])
	return end
}

// word reads the identifier or number at i; the function's own name becomes a
// placeholder.
//
// TRLC-LINKS: REQ-EMG-156
func (t *tokenizer) word(i int) int {
	j := i
	for j < len(t.text) && isWordByte(t.text[j]) {
		j++
	}
	word := t.text[i:j]
	if t.name != "" && word == t.name {
		word = "$name"
	}
	t.tokens = append(t.tokens, word)
	return j
}

// skipLine returns the index of the newline that ends the line of i.
//
// TRLC-LINKS: REQ-EMG-156
func skipLine(text string, i int) int {
	if j := strings.IndexByte(text[i:], '\n'); j >= 0 {
		return i + j
	}
	return len(text)
}

// skipBlockComment returns the index after the block comment that starts at i.
//
// TRLC-LINKS: REQ-EMG-156
func skipBlockComment(text string, i int, nested bool) int {
	depth := 1
	for j := i + 2; j < len(text); j++ {
		switch {
		case nested && strings.HasPrefix(text[j:], "/*"):
			depth++
			j++
		case strings.HasPrefix(text[j:], "*/"):
			depth--
			j++
		}
		if depth == 0 {
			return j + 1
		}
	}
	return len(text)
}

// stringEnd returns the index after the string or character literal that
// starts at i. An unterminated literal ends at the end of its line (the
// backtick forms at the end of the text), so a stray quote cannot swallow the
// rest of the function.
//
// TRLC-LINKS: REQ-EMG-156
func stringEnd(text string, i int, language string) int {
	if text[i] == '\'' && language == "rust" {
		return rustQuoteEnd(text, i)
	}
	if end, ok := pythonTripleEnd(text, i, language); ok {
		return end
	}
	return quotedEnd(text, i, language)
}

// rustQuoteEnd reads a Rust quote: a character literal, or a lifetime tick
// whose name is its own token.
//
// TRLC-LINKS: REQ-EMG-156
func rustQuoteEnd(text string, i int) int {
	if end, ok := rustCharEnd(text, i); ok {
		return end
	}
	return i + 1
}

// pythonTripleEnd returns the index after a Python triple-quoted string that
// starts at i; an unterminated one ends at the end of the text.
//
// TRLC-LINKS: REQ-EMG-156
func pythonTripleEnd(text string, i int, language string) (int, bool) {
	quote := text[i]
	if language != "python" || (quote != '"' && quote != '\'') {
		return 0, false
	}
	triple := strings.Repeat(string(quote), 3)
	if !strings.HasPrefix(text[i:], triple) {
		return 0, false
	}
	if j := strings.Index(text[i+3:], triple); j >= 0 {
		return i + 3 + j + 3, true
	}
	return len(text), true
}

// quoteRules says whether a literal opened by quote may span lines and
// whether a backslash escapes in it.
//
// TRLC-LINKS: REQ-EMG-156
func quoteRules(quote byte, language string) (multiline, escapes bool) {
	multiline = quote == '`' || (quote == '"' && language == "rust")
	escapes = !(quote == '`' && language == "go")
	return multiline, escapes
}

// quotedEnd returns the index after the literal that starts at i.
//
// TRLC-LINKS: REQ-EMG-156
func quotedEnd(text string, i int, language string) int {
	quote := text[i]
	multiline, escapes := quoteRules(quote, language)
	for j := i + 1; j < len(text); j++ {
		switch {
		case escapes && text[j] == '\\':
			j++
		case text[j] == quote:
			return j + 1
		case text[j] == '\n' && !multiline:
			return j
		}
	}
	return len(text)
}

// rustCharEnd recognises a Rust character literal at i ('x' or '\n').
//
// TRLC-LINKS: REQ-EMG-156
func rustCharEnd(text string, i int) (int, bool) {
	if i+1 >= len(text) {
		return 0, false
	}
	if text[i+1] == '\\' {
		return rustEscapedCharEnd(text, i)
	}
	j := i + 1
	_, size := utf8.DecodeRuneInString(text[j:])
	if j+size < len(text) && text[j+size] == '\'' {
		return j + size + 1, true
	}
	return 0, false
}

// rustEscapedCharEnd recognises a Rust character literal with an escape
// ('\n') at i; the closing tick must be within a few bytes.
//
// TRLC-LINKS: REQ-EMG-156
func rustEscapedCharEnd(text string, i int) (int, bool) {
	j := i + 3
	for j < len(text) && j < i+12 && text[j] != '\'' {
		j++
	}
	if j < len(text) && text[j] == '\'' {
		return j + 1, true
	}
	return 0, false
}
