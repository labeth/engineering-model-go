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
	return b == '_' || b == '$' || b >= 0x80 || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// bodyTokens splits text into tokens, dropping comments and white space. The
// language decides what a comment is (Python uses #, the others // and /* */,
// nested in Rust) and how a quote is read (a Rust quote may start a lifetime).
//
// TRLC-LINKS: REQ-EMG-156
func bodyTokens(text, language, name string) []string {
	var tokens []string
	n := len(text)
	for i := 0; i < n; {
		c := text[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f':
			i++
		case language == "python" && c == '#':
			i = skipLine(text, i)
		case language != "python" && c == '/' && i+1 < n && text[i+1] == '/':
			i = skipLine(text, i)
		case language != "python" && c == '/' && i+1 < n && text[i+1] == '*':
			i = skipBlockComment(text, i, language == "rust")
		case c == '"' || c == '\'' || c == '`':
			end := stringEnd(text, i, language)
			tokens = append(tokens, text[i:end])
			i = end
		case isWordByte(c):
			j := i
			for j < n && isWordByte(text[j]) {
				j++
			}
			word := text[i:j]
			if name != "" && word == name {
				word = "$name"
			}
			tokens = append(tokens, word)
			i = j
		default:
			tokens = append(tokens, text[i:i+1])
			i++
		}
	}
	return tokens
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
			if depth == 0 {
				return j + 1
			}
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
	quote := text[i]
	n := len(text)
	if quote == '\'' && language == "rust" {
		if end, ok := rustCharEnd(text, i); ok {
			return end
		}
		return i + 1 // a lifetime tick; the name that follows is its own token
	}
	if language == "python" && (quote == '"' || quote == '\'') && strings.HasPrefix(text[i:], strings.Repeat(string(quote), 3)) {
		if j := strings.Index(text[i+3:], strings.Repeat(string(quote), 3)); j >= 0 {
			return i + 3 + j + 3
		}
		return n
	}
	multiline := quote == '`' || (quote == '"' && language == "rust")
	escapes := !(quote == '`' && language == "go")
	for j := i + 1; j < n; j++ {
		switch {
		case escapes && text[j] == '\\':
			j++
		case text[j] == quote:
			return j + 1
		case text[j] == '\n' && !multiline:
			return j
		}
	}
	return n
}

// rustCharEnd recognises a Rust character literal at i ('x' or '\n').
//
// TRLC-LINKS: REQ-EMG-156
func rustCharEnd(text string, i int) (int, bool) {
	if i+1 >= len(text) {
		return 0, false
	}
	j := i + 1
	if text[j] == '\\' {
		for j += 2; j < len(text) && j < i+12 && text[j] != '\''; j++ {
		}
		if j < len(text) && text[j] == '\'' {
			return j + 1, true
		}
		return 0, false
	}
	_, size := utf8.DecodeRuneInString(text[j:])
	if j+size < len(text) && text[j+size] == '\'' {
		return j + size + 1, true
	}
	return 0, false
}
