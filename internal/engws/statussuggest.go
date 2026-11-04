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
func SuggestStatus(repo, tree string) (StatusSuggestion, error) {
	manifest := filepath.Join(repo, "engmod.yml")
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		return StatusSuggestion{}, err
	}
	top, err := gitText(repo, "rev-parse", "--show-toplevel")
	if err != nil {
		return StatusSuggestion{}, fmt.Errorf("status-suggest needs the git work tree of %s: %w", repo, err)
	}
	if tree == "" {
		if tree, err = workingTreeHash(top); err != nil {
			return StatusSuggestion{}, err
		}
	}
	logPath := filepath.Join(top, filepath.FromSlash(RedChecksLog))
	red, err := redRequirements(logPath, tree)
	if err != nil {
		return StatusSuggestion{}, err
	}
	matrix, _, err := engmodel.BuildTraceMatrixFromFilesWith(manifest, bundle.RequirementsPath, "", engmodel.TraceOptions{Resolution: engmodel.ResolutionOptions{Lock: engmodel.LockModeCheck, ModuleCache: engmodel.ModuleCacheReuse}})
	if err != nil {
		return StatusSuggestion{}, err
	}
	current := map[string]string{}
	for _, r := range bundle.Requirements.Requirements {
		current[strings.TrimSpace(r.ID)] = strings.TrimSpace(r.Status)
	}
	out := StatusSuggestion{Tree: tree, Log: RedChecksLog, Requirements: []RequirementSuggestion{}, ControlVerifications: []ControlVerificationSuggestion{}}
	earned := map[string]string{}
	for _, row := range matrix.Requirements {
		if strings.Contains(row.ID, "::") {
			continue
		}
		s := RequirementSuggestion{ID: row.ID, Current: current[row.ID], Reasons: []string{}}
		implemented := len(row.Code) > 0 || (row.SatisfiedBy == "links" && row.LinksImplemented)
		testLinked := len(row.Verifications) > 0 && !row.Gated && !row.BenchmarkOnly
		switch {
		case len(row.Code) > 0:
			s.Reasons = append(s.Reasons, "code linked")
		case row.SatisfiedBy == "links" && row.LinksImplemented:
			s.Reasons = append(s.Reasons, "implemented through reviewed links")
		}
		switch {
		case len(row.Verifications) == 0 && row.SatisfiedBy != "links":
			s.Reasons = append(s.Reasons, "no test link")
		case row.Gated:
			s.Reasons = append(s.Reasons, "only environment-gated tests")
		case row.BenchmarkOnly:
			s.Reasons = append(s.Reasons, "only benchmarks")
		case !red[row.ID] && len(row.Verifications) > 0:
			s.Reasons = append(s.Reasons, "no RED red-check for this tree")
		}
		switch {
		case implemented && testLinked && red[row.ID]:
			s.Suggested = "verified"
			s.Reasons = append(s.Reasons, "test linked", "RED red-check for this tree")
		case implemented && row.SatisfiedBy == "links" && row.LinksTestLinked:
			s.Suggested = "verified"
			s.Reasons = append(s.Reasons, "test linked through reviewed links")
		case implemented:
			s.Suggested = "implemented"
		}
		earned[row.ID] = s.Suggested
		out.Requirements = append(out.Requirements, s)
	}
	for _, cv := range bundle.Architecture.AuthoredArchitecture.ControlVerifications {
		id := strings.TrimSpace(cv.ID)
		if id == "" || strings.Contains(id, "::") {
			continue
		}
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
		out.ControlVerifications = append(out.ControlVerifications, s)
	}
	sort.SliceStable(out.ControlVerifications, func(i, j int) bool { return out.ControlVerifications[i].ID < out.ControlVerifications[j].ID })
	return out, nil
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
		var line struct {
			Tree    string   `json:"tree"`
			Verdict string   `json:"verdict"`
			Failure string   `json:"failure"`
			Reqs    []string `json:"reqs"`
		}
		if json.Unmarshal(scanner.Bytes(), &line) != nil || line.Tree != tree || line.Verdict != "RED" || line.Failure == "build" {
			continue
		}
		for _, r := range line.Reqs {
			red[strings.TrimSpace(r)] = true
		}
	}
	return red, scanner.Err()
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
func runStatusSuggest(repo, tree string, asJSON bool, w io.Writer) (int, error) {
	s, err := SuggestStatus(repo, tree)
	if err != nil {
		return 1, err
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return 0, enc.Encode(s)
	}
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
	return 0, nil
}

// dash returns s, or "-" when it is empty.
// TRLC-LINKS: REQ-EMG-122
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
