// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"gopkg.in/yaml.v3"
)

// Impact is a semantic version impact.
type Impact string

const (
	ImpactPatch Impact = "patch"
	ImpactMinor Impact = "minor"
	ImpactMajor Impact = "major"
)

// rank orders impacts.
// TRLC-LINKS: REQ-EMG-102
func (i Impact) rank() int {
	switch i {
	case ImpactMajor:
		return 3
	case ImpactMinor:
		return 2
	case ImpactPatch:
		return 1
	}
	return 0
}

// Change is how a contract item changed between two versions.
type Change string

const (
	ChangeAdded     Change = "added"
	ChangeRemoved   Change = "removed"
	ChangeChanged   Change = "changed"
	ChangeUnchanged Change = "unchanged"
)

// ItemChange is one contract item in a contract diff. Aspects names what
// changed (kind, text, verificationCriteria, realizers); Impact is empty for
// an unchanged item.
// TRLC-LINKS: REQ-EMG-102
type ItemChange struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Change  Change   `json:"change"`
	Aspects []string `json:"aspects,omitempty"`
	Impact  Impact   `json:"impact,omitempty"`
	Text    string   `json:"text"`
	Notes   []string `json:"notes,omitempty"`
}

// ContractDiffResult is the contract diff of a model between two versions
// with the overall semantic version impact (patch when nothing changed).
// TRLC-LINKS: REQ-EMG-102
type ContractDiffResult struct {
	From   string       `json:"from,omitempty"`
	To     string       `json:"to,omitempty"`
	Items  []ItemChange `json:"items"`
	Impact Impact       `json:"impact"`
}

// DiffContracts compares the published contract items of two versions of a
// model. Major: a capability removed or its text changed, an assumption of
// use added or its text or verificationCriteria changed, an item's kind
// changed. Minor: an assumption removed, a capability added, only the
// realizing requirements changed. Anything else is patch.
// TRLC-LINKS: REQ-EMG-102
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func DiffContracts(from, to model.Bundle) ContractDiffResult {
	before := reqlinks.PublishedItems(from, from.Manifest.Publications...)
	after := reqlinks.PublishedItems(to, to.Manifest.Publications...)
	ids := map[string]bool{}
	for id := range before {
		ids[id] = true
	}
	for id := range after {
		ids[id] = true
	}
	sorted := make([]string, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)
	result := ContractDiffResult{Items: []ItemChange{}, Impact: ImpactPatch}
	for _, id := range sorted {
		b, inBefore := before[id]
		a, inAfter := after[id]
		var c ItemChange
		switch {
		case !inBefore:
			c = ItemChange{ID: id, Kind: string(a.Kind), Change: ChangeAdded, Text: oneLine(a.Text), Impact: ImpactMinor}
			if a.Kind == reqlinks.KindAssumption {
				c.Impact = ImpactMajor
				c.Notes = append(c.Notes, "a new assumption of use binds every user")
			}
		case !inAfter:
			c = ItemChange{ID: id, Kind: string(b.Kind), Change: ChangeRemoved, Text: oneLine(b.Text), Impact: ImpactMajor}
			if b.Kind == reqlinks.KindAssumption {
				c.Impact = ImpactMinor
			}
		default:
			c = diffItem(b, a)
		}
		if c.Impact.rank() > result.Impact.rank() {
			result.Impact = c.Impact
		}
		result.Items = append(result.Items, c)
	}
	return result
}

// diffItem compares one item present in both versions.
// TRLC-LINKS: REQ-EMG-102
func diffItem(b, a reqlinks.Item) ItemChange {
	c := ItemChange{ID: a.ID, Kind: string(a.Kind), Change: ChangeUnchanged, Text: oneLine(a.Text)}
	raise := func(aspect string, impact Impact, note string) {
		c.Change = ChangeChanged
		c.Aspects = append(c.Aspects, aspect)
		if note != "" {
			c.Notes = append(c.Notes, note)
		}
		if impact.rank() > c.Impact.rank() {
			c.Impact = impact
		}
	}
	if b.Kind != a.Kind {
		raise("kind", ImpactMajor, fmt.Sprintf("kind changed from %s: treated as major", b.Kind))
	}
	if oneLine(b.Text) != oneLine(a.Text) {
		note := "text changed: treated as major"
		if a.Kind == reqlinks.KindAssumption {
			note = "assumption text changed: treated as major"
		}
		raise("text", ImpactMajor, note)
	}
	if oneLine(b.VerificationCriteria) != oneLine(a.VerificationCriteria) {
		raise("verificationCriteria", ImpactMajor, "verification criteria changed: treated as major")
	}
	if detail := diffRealizers(b.Realizers, a.Realizers); detail != "" {
		raise("realizers", ImpactMinor, detail)
	}
	return c
}

// diffRealizers describes how the realizing requirements changed, or ""
// when they did not.
// TRLC-LINKS: REQ-EMG-102
func diffRealizers(before, after []reqlinks.Realizer) string {
	canonical := func(r reqlinks.Realizer) string {
		return oneLine(r.Text) + "\x00" + oneLine(r.VerificationCriteria) + "\x00" + strings.TrimSpace(r.Status)
	}
	b, a := map[string]string{}, map[string]string{}
	for _, r := range before {
		b[r.ID] = canonical(r)
	}
	for _, r := range after {
		a[r.ID] = canonical(r)
	}
	var added, removed, changed []string
	for id, v := range a {
		if old, ok := b[id]; !ok {
			added = append(added, id)
		} else if old != v {
			changed = append(changed, id)
		}
	}
	for id := range b {
		if _, ok := a[id]; !ok {
			removed = append(removed, id)
		}
	}
	var parts []string
	for _, p := range []struct {
		label string
		ids   []string
	}{{"added", added}, {"removed", removed}, {"changed", changed}} {
		if len(p.ids) > 0 {
			sort.Strings(p.ids)
			parts = append(parts, "realizers "+p.label+": "+strings.Join(p.ids, ", "))
		}
	}
	return strings.Join(parts, "; ")
}

// gitOutput runs git in dir and returns its stdout.
// TRLC-LINKS: REQ-EMG-102, REQ-EMG-103
func gitOutput(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// LoadBundleAtRef loads the model in repo as committed at a git ref: it
// reads engmod.yml and every document the manifest names with git show into
// a temporary directory (0700, files 0600, removed afterwards) and loads it
// with the model loader. A ref starting with "-" is refused, so it is never
// read as a git option.
// TRLC-LINKS: REQ-EMG-102, REQ-EMG-103
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-MODEL-LOADER
func LoadBundleAtRef(repo, ref string) (model.Bundle, error) {
	if strings.HasPrefix(ref, "-") || strings.TrimSpace(ref) == "" {
		return model.Bundle{}, fmt.Errorf("invalid git ref %q", ref)
	}
	if _, err := gitOutput(repo, "rev-parse", "--verify", "--quiet", ref+"^{commit}"); err != nil {
		return model.Bundle{}, fmt.Errorf("unknown git ref %q in %s", ref, repo)
	}
	show := func(rel string) ([]byte, error) {
		return gitOutput(repo, "show", ref+":./"+rel)
	}
	manifest, err := show("engmod.yml")
	if err != nil {
		return model.Bundle{}, fmt.Errorf("%s has no engmod.yml at %s: %w", repo, ref, err)
	}
	var meta struct {
		Documents map[string]string `yaml:"documents"`
	}
	if err := yaml.Unmarshal(manifest, &meta); err != nil {
		return model.Bundle{}, fmt.Errorf("engmod.yml at %s: %w", ref, err)
	}
	// The committed model may be private: only the current user reads the
	// copy (MkdirTemp creates the directory 0700).
	tmp, err := os.MkdirTemp("", "engws-contract-")
	if err != nil {
		return model.Bundle{}, err
	}
	defer os.RemoveAll(tmp)
	if err := os.WriteFile(filepath.Join(tmp, "engmod.yml"), manifest, 0o600); err != nil {
		return model.Bundle{}, err
	}
	keys := make([]string, 0, len(meta.Documents))
	for key := range meta.Documents {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rel := meta.Documents[key]
		clean := path.Clean(filepath.ToSlash(strings.TrimSpace(rel)))
		if clean == "." || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
			return model.Bundle{}, fmt.Errorf("engmod.yml at %s: documents.%s %q leaves the model directory", ref, key, rel)
		}
		data, err := show(clean)
		if err != nil {
			return model.Bundle{}, fmt.Errorf("documents.%s at %s: %w", key, ref, err)
		}
		// A model committed before v0.11.0 models state behavior with the
		// retired behavior.states vocabulary, which the loader refuses. Only
		// the contract is read at a ref, and the contract has no states.
		if key == "behavior" {
			if data, err = model.StripLegacyStateVocabulary(data); err != nil {
				return model.Bundle{}, fmt.Errorf("documents.%s at %s: %w", key, ref, err)
			}
		}
		dest := filepath.Join(tmp, filepath.FromSlash(clean))
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return model.Bundle{}, err
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return model.Bundle{}, err
		}
	}
	bundle, err := model.LoadBundle(filepath.Join(tmp, "engmod.yml"))
	if err != nil {
		return model.Bundle{}, fmt.Errorf("model at %s: %w", ref, err)
	}
	return bundle, nil
}

// ContractDiff diffs the contract of the model in repo between two refs.
// TRLC-LINKS: REQ-EMG-102
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func ContractDiff(repo, fromRef, toRef string) (ContractDiffResult, error) {
	from, err := LoadBundleAtRef(repo, fromRef)
	if err != nil {
		return ContractDiffResult{}, err
	}
	to, err := LoadBundleAtRef(repo, toRef)
	if err != nil {
		return ContractDiffResult{}, err
	}
	result := DiffContracts(from, to)
	result.From, result.To = fromRef, toRef
	return result, nil
}

// printContractDiff prints one line per item and the impact.
// TRLC-LINKS: REQ-EMG-102
func printContractDiff(w io.Writer, d ContractDiffResult) {
	fmt.Fprintf(w, "contract %s..%s\n", d.From, d.To)
	for _, c := range d.Items {
		line := fmt.Sprintf("  %-9s %s %s", c.Change, c.Kind, c.ID)
		switch c.Change {
		case ChangeChanged:
			line += " (" + strings.Join(c.Aspects, ", ") + "): " + strings.Join(c.Notes, "; ")
		default:
			line += ": " + c.Text
			if len(c.Notes) > 0 {
				line += " (" + strings.Join(c.Notes, "; ") + ")"
			}
		}
		if c.Impact != "" {
			line += " [" + string(c.Impact) + "]"
		}
		fmt.Fprintln(w, line)
	}
	fmt.Fprintf(w, "impact: %s\n", d.Impact)
}

// runContractDiff prints the contract diff (text or JSON); IO and git
// errors exit 2.
// TRLC-LINKS: REQ-EMG-102
func runContractDiff(repo, fromRef, toRef string, asJSON bool, w io.Writer) (int, error) {
	d, err := ContractDiff(repo, fromRef, toRef)
	if err != nil {
		return 2, err
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return 0, enc.Encode(d)
	}
	printContractDiff(w, d)
	return 0, nil
}
