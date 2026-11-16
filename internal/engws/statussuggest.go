// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
)

// RedChecksLog is the red-check evidence log of engineering-flow's
// red-check.mjs, relative to the git top level of the project.
const RedChecksLog = ".engflow/red-checks.jsonl"

// RequirementSuggestion is the status a requirement earns from the tree's
// evidence: verified, implemented, or "" (nothing), with why.
// TRLC-LINKS: REQ-EMG-122
type RequirementSuggestion struct {
	ID        string   `json:"id"`
	Current   string   `json:"current"`
	Suggested string   `json:"suggested"`
	Reasons   []string `json:"reasons"`
}

// ControlVerificationSuggestion is the status a control verification earns:
// pass when every requirement its verificationRefs name earns verified,
// else its current status.
// TRLC-LINKS: REQ-EMG-122
type ControlVerificationSuggestion struct {
	ID        string   `json:"id"`
	Current   string   `json:"current"`
	Suggested string   `json:"suggested"`
	Missing   []string `json:"missing,omitempty"`
}

// StatusSuggestion is the output of engws status-suggest.
// TRLC-LINKS: REQ-EMG-122
type StatusSuggestion struct {
	Tree                 string                          `json:"tree"`
	Log                  string                          `json:"log"`
	Requirements         []RequirementSuggestion         `json:"requirements"`
	ControlVerifications []ControlVerificationSuggestion `json:"controlVerifications"`
}

// SuggestStatus computes the status each requirement and control
// verification of the model in repo earns from this tree's evidence,
// writing nothing. A requirement earns implemented when code links it, and
// verified when also a test links it that is not environment gated and not
// only a benchmark, and a RED line of .engflow/red-checks.jsonl (at the git
// top level) for tree names it in reqs. tree is the working-tree hash as
// engineering-flow's tree-hash.mjs computes it (git add -A into a temporary
// index, git write-tree) when empty. A requirement satisfied through
// reviewed requirement links (trace satisfiedBy links) earns what the trace
// rolls up. A control verification earns pass when every requirement in
// its verificationRefs earns verified; CODE-* references are not judged and
// keep it at its status.
// TRLC-LINKS: REQ-EMG-122
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-MODEL-STATUS-SUGGEST
func SuggestStatus(repo, tree string) (StatusSuggestion, error) {
	evidence, err := loadStatusEvidence(repo, tree)
	if err != nil {
		return StatusSuggestion{}, err
	}
	out := StatusSuggestion{Tree: evidence.tree, Log: RedChecksLog, Requirements: []RequirementSuggestion{}}
	earned := map[string]string{}
	current := currentRequirementStatuses(evidence.bundle)
	for _, row := range evidence.matrix.Requirements {
		if strings.Contains(row.ID, "::") {
			continue
		}
		s := suggestRequirement(row, current[row.ID], evidence.red[row.ID])
		earned[row.ID] = s.Suggested
		out.Requirements = append(out.Requirements, s)
	}
	out.ControlVerifications = suggestControlVerifications(evidence.bundle, earned)
	return out, nil
}

// statusEvidence is what status-suggest judges: the model, the tree hash, the requirements with a RED red-check on
// that tree and the trace matrix.
// TRLC-LINKS: REQ-EMG-122
type statusEvidence struct {
	bundle model.Bundle
	tree   string
	red    map[string]bool
	matrix engmodel.TraceMatrix
}

// loadStatusEvidence loads the model of repo, the tree hash (the working tree's when tree is empty), the red checks
// of that tree and the trace matrix.
// TRLC-LINKS: REQ-EMG-122
func loadStatusEvidence(repo, tree string) (statusEvidence, error) {
	manifest := filepath.Join(repo, "engmod.yml")
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		return statusEvidence{}, err
	}
	top, err := gitText(repo, "rev-parse", "--show-toplevel")
	if err != nil {
		return statusEvidence{}, fmt.Errorf("status-suggest needs the git work tree of %s: %w", repo, err)
	}
	if tree == "" {
		if tree, err = workingTreeHash(top); err != nil {
			return statusEvidence{}, err
		}
	}
	red, err := redRequirements(filepath.Join(top, filepath.FromSlash(RedChecksLog)), tree)
	if err != nil {
		return statusEvidence{}, err
	}
	matrix, _, err := engmodel.BuildTraceMatrixFromFilesWith(engmodel.TraceFiles{ModelPath: manifest, RequirementsPath: bundle.RequirementsPath}, engmodel.TraceOptions{Resolution: engmodel.ResolutionOptions{Lock: engmodel.LockModeCheck, ModuleCache: engmodel.ModuleCacheReuse}})
	if err != nil {
		return statusEvidence{}, err
	}
	return statusEvidence{bundle: bundle, tree: tree, red: red, matrix: matrix}, nil
}

// currentRequirementStatuses maps each requirement ID of the bundle to its trimmed status.
// TRLC-LINKS: REQ-EMG-122
func currentRequirementStatuses(bundle model.Bundle) map[string]string {
	current := map[string]string{}
	for _, r := range bundle.Requirements.Requirements {
		current[strings.TrimSpace(r.ID)] = strings.TrimSpace(r.Status)
	}
	return current
}

// suggestRequirement is the status one trace row earns, with why; red tells whether a RED red-check names it.
// TRLC-LINKS: REQ-EMG-122
func suggestRequirement(row engmodel.TraceRow, current string, red bool) RequirementSuggestion {
	s := RequirementSuggestion{ID: row.ID, Current: current}
	s.Reasons = append(implementationReasons(row), verificationReasons(row, red)...)
	suggested, reasons := earnedRequirementStatus(row, red)
	s.Suggested = suggested
	s.Reasons = append(s.Reasons, reasons...)
	return s
}

// earnedRequirementStatus is the status a row earns (verified, implemented or "") with the reasons it adds.
// TRLC-LINKS: REQ-EMG-122
func earnedRequirementStatus(row engmodel.TraceRow, red bool) (string, []string) {
	implemented := len(row.Code) > 0 || rowLinksImplemented(row)
	switch {
	case implemented && rowVerifiedByRed(row, red):
		return "verified", []string{"test linked", "RED red-check for this tree"}
	case implemented && rowLinksTested(row):
		return "verified", []string{"test linked through reviewed links"}
	case implemented:
		return "implemented", nil
	}
	return "", nil
}

// rowLinksImplemented reports whether the row is implemented through reviewed requirement links.
// TRLC-LINKS: REQ-EMG-122
func rowLinksImplemented(row engmodel.TraceRow) bool {
	return row.SatisfiedBy == "links" && row.LinksImplemented
}

// rowLinksTested reports whether the row has a test linked through reviewed requirement links.
// TRLC-LINKS: REQ-EMG-122
func rowLinksTested(row engmodel.TraceRow) bool {
	return row.SatisfiedBy == "links" && row.LinksTestLinked
}

// rowVerifiedByRed reports whether the row has an ungated, non-benchmark test link and a RED red-check.
// TRLC-LINKS: REQ-EMG-122
func rowVerifiedByRed(row engmodel.TraceRow, red bool) bool {
	return rowTestLinked(row) && red
}

// rowTestLinked reports whether the row has a test link that is not environment gated and not only a benchmark.
// TRLC-LINKS: REQ-EMG-122
func rowTestLinked(row engmodel.TraceRow) bool {
	return len(row.Verifications) > 0 && !row.Gated && !row.BenchmarkOnly
}

// implementationReasons says how the row is implemented: code linked or reviewed links.
// TRLC-LINKS: REQ-EMG-122
func implementationReasons(row engmodel.TraceRow) []string {
	reasons := []string{}
	switch {
	case len(row.Code) > 0:
		reasons = append(reasons, "code linked")
	case rowLinksImplemented(row):
		reasons = append(reasons, "implemented through reviewed links")
	}
	return reasons
}

// verificationReasons says what keeps the row from verified: a missing, gated or benchmark-only test or a missing RED line.
// TRLC-LINKS: REQ-EMG-122
func verificationReasons(row engmodel.TraceRow, red bool) []string {
	switch {
	case len(row.Verifications) == 0 && row.SatisfiedBy != "links":
		return []string{"no test link"}
	case row.Gated:
		return []string{"only environment-gated tests"}
	case row.BenchmarkOnly:
		return []string{"only benchmarks"}
	case !red && len(row.Verifications) > 0:
		return []string{"no RED red-check for this tree"}
	}
	return nil
}

// suggestControlVerifications is the pass status each control verification earns from the earned requirement
// statuses, sorted by ID.
// TRLC-LINKS: REQ-EMG-122
func suggestControlVerifications(bundle model.Bundle, earned map[string]string) []ControlVerificationSuggestion {
	out := []ControlVerificationSuggestion{}
	for _, cv := range bundle.Architecture.AuthoredArchitecture.ControlVerifications {
		id := strings.TrimSpace(cv.ID)
		if id == "" || strings.Contains(id, "::") {
			continue
		}
		out = append(out, suggestControlVerification(id, cv, earned))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// suggestControlVerification earns pass when every requirement in the verificationRefs earns verified.
// TRLC-LINKS: REQ-EMG-122
func suggestControlVerification(id string, cv model.ControlVerification, earned map[string]string) ControlVerificationSuggestion {
	s := ControlVerificationSuggestion{ID: id, Current: strings.TrimSpace(cv.Status), Suggested: strings.TrimSpace(cv.Status)}
	reqs := 0
	for _, ref := range cv.VerificationRefs {
		ref = strings.TrimSpace(ref)
		if !strings.HasPrefix(ref, "REQ-") {
			s.Missing = append(s.Missing, ref+" (not a requirement)")
			continue
		}
		reqs++
		if earned[ref] != "verified" {
			s.Missing = append(s.Missing, ref)
		}
	}
	if reqs > 0 && len(s.Missing) == 0 {
		s.Suggested = "pass"
	}
	return s
}

// redRequirements returns the requirements named in reqs of a RED line of
// the log for tree; a missing log is no evidence. Unparseable lines and
// lines without reqs are skipped (red-check.mjs --req or a mutation's req
// records them).
// TRLC-LINKS: REQ-EMG-122
func redRequirements(path, tree string) (map[string]bool, error) {
	red := map[string]bool{}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return red, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		for _, r := range redLineRequirements(scanner.Bytes(), tree) {
			red[strings.TrimSpace(r)] = true
		}
	}
	return red, scanner.Err()
}

// redLineRequirements returns the reqs of one log line when it is a RED line for tree, else nil.
// TRLC-LINKS: REQ-EMG-122
func redLineRequirements(raw []byte, tree string) []string {
	var line struct {
		Tree    string   `json:"tree"`
		Verdict string   `json:"verdict"`
		Failure string   `json:"failure"`
		Reqs    []string `json:"reqs"`
	}
	if json.Unmarshal(raw, &line) != nil || line.Tree != tree {
		return nil
	}
	if line.Verdict != "RED" || line.Failure == "build" {
		return nil
	}
	return line.Reqs
}

// workingTreeHash is the tree hash of the working tree at top with
// uncommitted and untracked (not ignored) files, computed on a temporary
// copy of the index like tree-hash.mjs, so the real index is unchanged.
// TRLC-LINKS: REQ-EMG-122
func workingTreeHash(top string) (string, error) {
	indexRel, err := gitText(top, "rev-parse", "--git-path", "index")
	if err != nil {
		return "", err
	}
	index := indexRel
	if !filepath.IsAbs(index) {
		index = filepath.Join(top, indexRel)
	}
	dir, err := os.MkdirTemp("", "engws-tree-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	temp := filepath.Join(dir, "index")
	if data, err := os.ReadFile(index); err == nil {
		if err := os.WriteFile(temp, data, 0o600); err != nil {
			return "", err
		}
	}
	env := append(os.Environ(), "GIT_INDEX_FILE="+temp)
	if _, err := gitTextEnv(top, env, "add", "-A"); err != nil {
		return "", err
	}
	return gitTextEnv(top, env, "write-tree")
}

// gitText runs git in dir and returns its trimmed output.
// TRLC-LINKS: REQ-EMG-122
func gitText(dir string, args ...string) (string, error) {
	return gitTextEnv(dir, nil, args...)
}

// gitTextEnv is gitText with an environment (nil: the process's).
// TRLC-LINKS: REQ-EMG-122
func gitTextEnv(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// runStatusSuggest prints the suggestion as tables or JSON.
// TRLC-LINKS: REQ-EMG-122
// ENGMODEL-LINKS: EVT-STATUS-SUGGEST-REQUESTED
func runStatusSuggest(options StatusSuggestOptions, w io.Writer) (int, error) {
	s, err := SuggestStatus(options.Repo, options.Tree)
	if err != nil {
		return 1, err
	}
	if options.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return 0, enc.Encode(s)
	}
	printStatusSuggestion(w, s)
	return 0, nil
}

// StatusSuggestOptions are the flags of engws status-suggest: the model directory, the tree hash ("": the working
// tree) and whether to print JSON.
// TRLC-LINKS: REQ-EMG-122
type StatusSuggestOptions struct {
	Repo string
	Tree string
	JSON bool
}

// printStatusSuggestion prints the suggestion as two tables.
// TRLC-LINKS: REQ-EMG-122
// ENGMODEL-LINKS: EVT-STATUS-SUGGEST-REQUESTED
func printStatusSuggestion(w io.Writer, s StatusSuggestion) {
	fmt.Fprintf(w, "engws: status earned on tree %s (red checks: %s); nothing is written\n", s.Tree, s.Log)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "REQUIREMENT\tCURRENT\tSUGGESTED\tWHY")
	for _, r := range s.Requirements {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.ID, dash(r.Current), dash(r.Suggested), strings.Join(r.Reasons, "; "))
	}
	fmt.Fprintln(tw, "CONTROL VERIFICATION\tCURRENT\tSUGGESTED\tMISSING")
	for _, v := range s.ControlVerifications {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", v.ID, dash(v.Current), dash(v.Suggested), strings.Join(v.Missing, ", "))
	}
	tw.Flush()
	fmt.Fprintln(w, "engws: the owner approves each change; set it in model/requirements.yml and model/assurance.yml")
}

// dash returns s, or "-" when it is empty.
// TRLC-LINKS: REQ-EMG-122
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
