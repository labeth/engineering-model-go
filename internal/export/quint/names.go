// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package quint

import (
	"strings"
)

// quintReserved lists the Quint words a module name must not take. Every other
// generated identifier carries a prefix (v_, tr_, inv_, wit_) or is a
// PascalCase constructor, so it cannot clash with a lower-case keyword.
var quintReserved = map[string]bool{
	"module": true, "import": true, "export": true, "const": true, "var": true,
	"val": true, "def": true, "pure": true, "action": true, "run": true,
	"temporal": true, "assume": true, "type": true, "if": true, "else": true,
	"and": true, "or": true, "iff": true, "implies": true, "all": true,
	"any": true, "match": true, "nondet": true, "true": true, "false": true,
	"init": true, "step": true, "from": true, "as": true,
}

// windowsDeviceNames lists the file names Windows reserves for devices, with
// or without an extension ("con.qnt" is the console there). A module name is
// also a file name, so it must not be one of them. The names are compared in
// the lower case the module names have.
var windowsDeviceNames = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true,
	"com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true,
	"lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// reservedModule explains why a module name is not usable, or returns "".
//
// TRLC-LINKS: REQ-EMG-141
func reservedModule(module string) string {
	switch {
	case quintReserved[module]:
		return "the reserved Quint word"
	case windowsDeviceNames[module]:
		return "a device name Windows reserves for files (it cannot be the module file name)"
	}
	return ""
}

// snakeCase turns a model ID into a Quint identifier part: ASCII letters and
// digits are lower-cased and every other run of characters becomes one
// underscore ("SM-PAYMENTS-AUTH" -> "sm_payments_auth"). The result is empty
// when the ID holds no ASCII letter or digit.
//
// TRLC-LINKS: REQ-EMG-141
func snakeCase(id string) string {
	var out strings.Builder
	pendingSeparator := false
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r >= 'A' && r <= 'Z':
			r += 'a' - 'A'
		default:
			pendingSeparator = out.Len() > 0
			continue
		}
		if pendingSeparator {
			out.WriteByte('_')
			pendingSeparator = false
		}
		out.WriteRune(r)
	}
	return out.String()
}

// pascalCase turns a model ID into a Quint constructor name: the ID is split
// at every character that is not an ASCII letter or digit and each part gets
// an upper-case first letter and lower-case rest ("STATE-PAYMENTS-IDLE" ->
// "StatePaymentsIdle"). The result is empty when the ID has no such character.
//
// TRLC-LINKS: REQ-EMG-141
func pascalCase(id string) string {
	var out strings.Builder
	startOfPart := true
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
			if startOfPart {
				r -= 'a' - 'A'
			}
		case r >= 'A' && r <= 'Z':
			if !startOfPart {
				r += 'a' - 'A'
			}
		case r >= '0' && r <= '9':
		default:
			startOfPart = true
			continue
		}
		startOfPart = false
		out.WriteRune(r)
	}
	return out.String()
}

// oneLine collapses all white space so a name or description is safe inside a
// single-line Quint comment.
//
// TRLC-LINKS: REQ-EMG-141
func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// quintString renders a Quint string literal. Quint has no escape sequences:
// a literal ends at the next double quote and cannot span lines, so a value
// holding either cannot be represented.
//
// TRLC-LINKS: REQ-EMG-141
func quintString(value string) (string, bool) {
	if strings.ContainsAny(value, "\"\r\n") {
		return "", false
	}
	return `"` + value + `"`, true
}

// prefixed joins a kind prefix and a snake-case ID without repeating the
// prefix: "inv_" + "inv_payments_001" stays "inv_payments_001".
//
// TRLC-LINKS: REQ-EMG-141
func prefixed(prefix, snake string) string {
	if strings.HasPrefix(snake, prefix) {
		return snake
	}
	return prefix + snake
}
