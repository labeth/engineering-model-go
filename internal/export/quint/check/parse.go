// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	okRe        = regexp.MustCompile(`\[ok\] No violation found`)
	violationRe = regexp.MustCompile(`\[violation\]`)
	statsRe     = regexp.MustCompile(`Trace length statistics: max=(\d+), min=(\d+)`)
	witnessRe   = regexp.MustCompile(`(?m)^(\S+) was witnessed in (\d+) trace\(s\) out of (\d+) explored`)
	timingRe    = regexp.MustCompile(`\s*\(\d+ms[^)]*\)`)
)

// runParse is what the output of one quint run or quint verify says.
type runParse struct {
	// Status is pass, violated or error.
	Status   string
	ExitCode int
	// TraceMin and TraceMax are the trace length statistics of a pass; nil when
	// the output has none.
	TraceMin, TraceMax *int
	// Message explains an error.
	Message string
}

// parseQuintOutput classifies the output of a quint run or quint verify: exit
// 0 with the ok line is a pass, exit 1 with the violation line is a violation
// and anything else (a runtime error, a crash, an unknown output) is an error.
// Timing text is never kept, so the result is reproducible.
//
// TRLC-LINKS: REQ-EMG-143, REQ-EMG-144
func parseQuintOutput(out Output) runParse {
	p := runParse{ExitCode: out.ExitCode}
	all := out.Stdout + "\n" + out.Stderr
	switch {
	case out.TimedOut || out.Truncated:
		p.Status = StatusError
		p.Message = errorMessage(out)
	case out.ExitCode == 0 && okRe.MatchString(all):
		p.Status = StatusPass
		p.TraceMin, p.TraceMax = traceStats(all)
	case out.ExitCode == 1 && violationRe.MatchString(all):
		p.Status = StatusViolated
	default:
		p.Status = StatusError
		p.Message = errorMessage(out)
	}
	return p
}

// traceStats reads the trace length statistics of a quint run; both are nil
// when the output has none.
//
// TRLC-LINKS: REQ-EMG-143
func traceStats(output string) (min, max *int) {
	m := statsRe.FindStringSubmatch(output)
	if m == nil {
		return nil, nil
	}
	hi, _ := strconv.Atoi(m[1])
	lo, _ := strconv.Atoi(m[2])
	return &lo, &hi
}

// errorMessage picks the most telling line of a failed quint call: an
// [error] line, an "error:" line, else the first line of stderr or stdout.
//
// TRLC-LINKS: REQ-EMG-143
func errorMessage(out Output) string {
	switch {
	case out.TimedOut:
		return "quint was killed: it exceeded its time limit"
	case out.Truncated:
		return "quint output exceeded the capture limit and was cut off, so its verdict cannot be read"
	}
	pick := strings.TrimSpace(timingRe.ReplaceAllString(telltaleLine(out), ""))
	pick = truncateUTF8(pick, maxMessageBytes)
	if pick == "" {
		return fmt.Sprintf("quint exited with code %d", out.ExitCode)
	}
	return fmt.Sprintf("quint exited with code %d: %s", out.ExitCode, pick)
}

// telltaleLine picks the first [error], "error:" or "Error " line of the
// output, else the first line of stderr, else the first line of stdout.
//
// TRLC-LINKS: REQ-EMG-143
func telltaleLine(out Output) string {
	for _, line := range strings.Split(out.Stdout+"\n"+out.Stderr, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[error]") || strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "Error ") {
			return line
		}
	}
	if line := firstLine(out.Stderr); line != "" {
		return line
	}
	return firstLine(out.Stdout)
}

// maxMessageBytes bounds the quint message kept in a result.
const maxMessageBytes = 200

// truncateUTF8 cuts s to at most max bytes without splitting a rune.
//
// TRLC-LINKS: REQ-EMG-143
func truncateUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// witnessCount is one line of the Witnesses block of quint run.
type witnessCount struct{ Witnessed, Total int }

// parseWitnesses reads "wit_x was witnessed in N trace(s) out of M explored".
//
// TRLC-LINKS: REQ-EMG-143
func parseWitnesses(out Output) map[string]witnessCount {
	counts := map[string]witnessCount{}
	for _, m := range witnessRe.FindAllStringSubmatch(out.Stdout+"\n"+out.Stderr, -1) {
		n, _ := strconv.Atoi(m[2])
		total, _ := strconv.Atoi(m[3])
		counts[m[1]] = witnessCount{Witnessed: n, Total: total}
	}
	return counts
}

// counterexampleFromITF reads an ITF trace file and returns the TR-* IDs of
// the steps in order. The first state is the initial state, which no
// transition led to, so it is dropped.
//
// TRLC-LINKS: REQ-EMG-143, REQ-EMG-144
func counterexampleFromITF(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var trace struct {
		States []map[string]json.RawMessage `json:"states"`
	}
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, fmt.Errorf("parse ITF trace: %w", err)
	}
	if len(trace.States) == 0 {
		return nil, errors.New("the ITF trace has no states")
	}
	ids, err := transitionIDs(trace.States)
	if err != nil {
		return nil, err
	}
	if len(trace.States) > 1 && len(ids) == 0 {
		return nil, errors.New("the ITF trace has steps but none names its lastTransition")
	}
	return ids, nil
}

// transitionIDs reads the lastTransition of every state after the first.
//
// TRLC-LINKS: REQ-EMG-143, REQ-EMG-144
func transitionIDs(states []map[string]json.RawMessage) ([]string, error) {
	var ids []string
	for i := 1; i < len(states); i++ {
		raw, ok := states[i]["lastTransition"]
		if !ok {
			continue
		}
		var id string
		if err := json.Unmarshal(raw, &id); err != nil {
			return nil, fmt.Errorf("state %d: lastTransition is not a string", i)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// typeErrors reads the errors of `quint typecheck --out`, one "file:line:col:
// message" per error. Quint locations are zero based.
//
// TRLC-LINKS: REQ-EMG-142
func typeErrors(jsonPath string, out Output) []string {
	if msgs := readTypeErrors(jsonPath); len(msgs) > 0 {
		return msgs
	}
	return []string{errorMessage(out)}
}

// quintTypeError is one error of the `quint typecheck --out` file.
type quintTypeError struct {
	Explanation string `json:"explanation"`
	Locs        []struct {
		Source string `json:"source"`
		Start  struct {
			Line int `json:"line"`
			Col  int `json:"col"`
		} `json:"start"`
	} `json:"locs"`
}

// describe renders the error as "file:line:col: message".
//
// TRLC-LINKS: REQ-EMG-142
func (e quintTypeError) describe() string {
	text := firstLine(e.Explanation)
	if len(e.Locs) == 0 {
		return text
	}
	loc := e.Locs[0]
	base := loc.Source
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	return fmt.Sprintf("%s:%d:%d: %s", base, loc.Start.Line+1, loc.Start.Col+1, text)
}

// readTypeErrors reads the type errors of the typecheck output file; nil when
// the file is missing, unreadable or lists none.
//
// TRLC-LINKS: REQ-EMG-142
func readTypeErrors(jsonPath string) []string {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil
	}
	var parsed struct {
		Errors []quintTypeError `json:"errors"`
	}
	if json.Unmarshal(data, &parsed) != nil {
		return nil
	}
	var msgs []string
	for _, e := range parsed.Errors {
		msgs = append(msgs, e.describe())
	}
	return msgs
}
