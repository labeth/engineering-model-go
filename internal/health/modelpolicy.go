// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

// The model-level part of the policy comparison with the base revision. The
// health policy of policy.go (tier, thresholds, exclude, overrides) is not the
// only thing that decides how code and evidence are judged: the life-cycle
// bundles and tier decision, the lint and requirement link strictness, the
// health block and its baseline, the POA&M items that defer diagnostics and the
// accepted tier and health decisions do too. A change that relaxes one of them
// loosens the gate it is measured by, so enghealth check compares them with the
// manifest, requirements, assurance and decisions documents of the base
// revision and reports each relaxation as health.policy_weakened, which a
// policyDecision records exactly like the other weakenings.
//
// The baseline is compared separately (BaselineWeakenings): it can only shrink.
// A finding that is new in it, or whose recorded value grew, is reported as
// health.policy_weakened and not as health.regression. A regression is a code
// finding that is new or worse in the changed files and is never baselined;
// growing the baseline is a change of the policy that accepts debt, which is
// what a policyDecision (a reviewable, code-owned record) is for, so it takes
// the same code, level and exemption as every other change of the policy.
//
// Everything is read leniently from the raw YAML: only the keys used here, no
// schema, so a document written for an older schema still yields its policy.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/labeth/engineering-model-go/internal/model"
)

// The aspects of a model-level weakening, besides those of policy.go.
const (
	PolicyLifecycle        = "lifecycle"
	PolicyLint             = "lint"
	PolicyRequirementLinks = "requirementLinks"
	PolicyHealthBlock      = "health"
	PolicyPoam             = "poam"
	PolicyDecisionRecord   = "decision"
	PolicyBaseline         = "baseline"
)

// maxPolicyDocBytes bounds what is read of one document of a revision.
const maxPolicyDocBytes = 64 << 20

// examplesInMessage is how many findings a baseline message names.
const examplesInMessage = 3

// ModelPolicy is what a revision declares outside the health block of
// policy.go: see the comment of the file.
type ModelPolicy struct {
	// Present is true when the manifest exists at the revision; without one the
	// module is new and nothing can be weakened.
	Present bool
	// HasLifecycle, Tier, TierDecision and Bundles are the life-cycle block;
	// Bundles are the effective ones (foundation included).
	HasLifecycle bool
	Tier         string
	TierDecision string
	Bundles      []string
	// RequirementLinks and LintMode are the strictness switches, lowercased.
	RequirementLinks string
	LintMode         string
	// HasHealth is true when the health block exists, BaselineConfigured when
	// it names a baseline file, and BaselineFile that file.
	HasHealth          bool
	BaselineConfigured bool
	BaselineFile       string
	// BaselineRead is true when the baseline file was read and parsed; Baseline
	// maps each accepted finding to its recorded value.
	BaselineRead bool
	Baseline     map[model.HealthBaselineKey]int
	// Poam are the POA&M items by ID.
	Poam map[string]model.POAMItem
	// Accepted are the IDs of the accepted decisions that set the tier or the
	// health policy (DEC-TIER*, DEC-HEALTH*).
	Accepted map[string]bool
}

// PolicyReader reads a file of a revision by a path relative to the manifest
// directory. A file that is not there is an error wrapping ErrNotInRev (or
// os.ErrNotExist for the working tree).
type PolicyReader func(rel string) ([]byte, error)

// GitPolicyReader reads the files of one revision with git, never touching the
// working tree.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func GitPolicyReader(repo GitRepo, rev string) PolicyReader {
	return func(rel string) ([]byte, error) { return repo.ShowFile(rev, rel) }
}

// WorkTreePolicyReader reads regular files below dir, bounded in size. A path
// that leaves dir is refused.
//
// TRLC-LINKS: REQ-EMG-155
func WorkTreePolicyReader(dir string) PolicyReader {
	return func(rel string) ([]byte, error) {
		clean, err := normalizeRepoPath(rel)
		if err != nil {
			return nil, err
		}
		return readBounded(filepath.Join(dir, filepath.FromSlash(clean)))
	}
}

// ReadWorkTreeFile reads one regular file, bounded in size.
//
// TRLC-LINKS: REQ-EMG-155
func ReadWorkTreeFile(full string) ([]byte, error) { return readBounded(full) }

// readBounded reads a regular file of at most maxPolicyDocBytes.
//
// TRLC-LINKS: REQ-EMG-155
func readBounded(full string) ([]byte, error) {
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", filepath.Base(full))
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxPolicyDocBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPolicyDocBytes {
		return nil, fmt.Errorf("%s is larger than %d bytes", filepath.Base(full), maxPolicyDocBytes)
	}
	return data, nil
}

// policyManifest is the part of a manifest read here.
type policyManifest struct {
	Documents struct {
		Requirements string `yaml:"requirements"`
		Assurance    string `yaml:"assurance"`
		Decisions    string `yaml:"decisions"`
	} `yaml:"documents"`
	RequirementLinks string                   `yaml:"requirementLinks"`
	Lifecycle        *model.ManifestLifecycle `yaml:"lifecycle"`
	Health           *model.ManifestHealth    `yaml:"health"`
}

// policyDocs are the parts of the other documents read here.
type policyRequirements struct {
	LintRun struct {
		Mode string `yaml:"mode"`
	} `yaml:"lintRun"`
}

type policyAssurance struct {
	Assurance struct {
		PoamItems []model.POAMItem `yaml:"poamItems"`
	} `yaml:"assurance"`
}

type policyDecisions struct {
	Decisions []struct {
		ID     string `yaml:"id"`
		Status string `yaml:"status"`
	} `yaml:"decisions"`
}

// ParseModelPolicy reads the model-level policy of a manifest and of the
// documents and the baseline file it names, through read. A document or the
// baseline file that is not there contributes nothing; one that cannot be read
// or is not YAML is an error, so the comparison is never skipped silently. An
// unparsable baseline file is not an error: BaselineRead stays false.
//
// TRLC-LINKS: REQ-EMG-155
func ParseModelPolicy(manifest []byte, read PolicyReader) (ModelPolicy, error) {
	var m policyManifest
	if err := yaml.Unmarshal(manifest, &m); err != nil {
		return ModelPolicy{}, fmt.Errorf("read the manifest: %w", err)
	}
	p := ModelPolicy{Present: true, RequirementLinks: strings.ToLower(strings.TrimSpace(m.RequirementLinks))}
	p.applyBlocks(m)
	if err := p.readDocuments(m, read); err != nil {
		return ModelPolicy{}, err
	}
	p.readBaseline(read)
	return p, nil
}

// applyBlocks sets the policy from the life-cycle and health blocks of the
// manifest.
//
// TRLC-LINKS: REQ-EMG-155
func (p *ModelPolicy) applyBlocks(m policyManifest) {
	if l := m.Lifecycle; l != nil {
		p.HasLifecycle = true
		p.Tier = strings.ToLower(strings.TrimSpace(l.Tier))
		p.TierDecision = strings.TrimSpace(l.TierDecision)
		p.Bundles = l.EffectiveBundles()
	}
	if h := m.Health; h != nil {
		p.HasHealth = true
		p.BaselineFile = baselineFileOf(h)
		p.BaselineConfigured = p.BaselineFile != ""
	}
}

// baselineFileOf is the baseline file a health block names, or "".
//
// TRLC-LINKS: REQ-EMG-155
func baselineFileOf(h *model.ManifestHealth) string {
	if h.Baseline == nil {
		return ""
	}
	return strings.TrimSpace(h.Baseline.File)
}

// readDocuments reads the lint mode from the requirements document, the POA&M
// items from the assurance document and the accepted tier and health
// decisions from the decisions document.
//
// TRLC-LINKS: REQ-EMG-155
func (p *ModelPolicy) readDocuments(m policyManifest, read PolicyReader) error {
	var req policyRequirements
	if err := readPolicyDoc(read, m.Documents.Requirements, &req); err != nil {
		return err
	}
	p.LintMode = strings.ToLower(strings.TrimSpace(req.LintRun.Mode))
	var as policyAssurance
	if err := readPolicyDoc(read, m.Documents.Assurance, &as); err != nil {
		return err
	}
	p.Poam = map[string]model.POAMItem{}
	for _, item := range as.Assurance.PoamItems {
		p.Poam[strings.TrimSpace(item.ID)] = item
	}
	var dec policyDecisions
	if err := readPolicyDoc(read, m.Documents.Decisions, &dec); err != nil {
		return err
	}
	p.Accepted = acceptedPolicyDecisions(dec)
	return nil
}

// acceptedPolicyDecisions returns the IDs of the accepted DEC-TIER* and
// DEC-HEALTH* decisions.
//
// TRLC-LINKS: REQ-EMG-155
func acceptedPolicyDecisions(dec policyDecisions) map[string]bool {
	out := map[string]bool{}
	for _, d := range dec.Decisions {
		id := strings.TrimSpace(d.ID)
		if strings.EqualFold(strings.TrimSpace(d.Status), "accepted") && isPolicyDecisionID(id) {
			out[id] = true
		}
	}
	return out
}

// readBaseline reads the baseline file of the policy; a file that is missing,
// outside the manifest directory or not a valid baseline leaves it unread.
//
// TRLC-LINKS: REQ-EMG-155
func (p *ModelPolicy) readBaseline(read PolicyReader) {
	if !p.BaselineConfigured || model.HealthBaselinePathProblem(p.BaselineFile) != "" {
		return
	}
	data, err := read(p.BaselineFile)
	if err != nil {
		return
	}
	file, err := model.ParseHealthBaseline(data)
	if err != nil {
		return
	}
	p.BaselineRead = true
	p.Baseline = make(map[model.HealthBaselineKey]int, len(file.Entries))
	for _, e := range file.Entries {
		p.Baseline[e.Key()] = e.Value
	}
}

// readPolicyDoc decodes the document at rel into out; an empty path, one that
// is not clean or a file that is not there leaves out empty.
//
// TRLC-LINKS: REQ-EMG-155
func readPolicyDoc(read PolicyReader, rel string, out any) error {
	rel = strings.TrimSpace(rel)
	if rel == "" || model.HealthBaselinePathProblem(rel) != "" {
		return nil
	}
	data, err := read(rel)
	if isAbsentFile(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", rel, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("read %s: %w", rel, err)
	}
	return nil
}

// isAbsentFile reports whether a read failed because the file is not there (or
// its path is not usable).
//
// TRLC-LINKS: REQ-EMG-155
func isAbsentFile(err error) bool {
	return errors.Is(err, ErrNotInRev) || errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrInvalidPath)
}

// isPolicyDecisionID reports whether a decision ID sets the tier or the health
// policy.
//
// TRLC-LINKS: REQ-EMG-155
func isPolicyDecisionID(id string) bool {
	return strings.HasPrefix(id, "DEC-TIER") || strings.HasPrefix(id, "DEC-HEALTH")
}

// ModelPolicyChanges lists every way in which head is weaker than base at the
// model level, including the baseline, sorted. A base without a manifest has
// nothing to weaken.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func ModelPolicyChanges(base, head ModelPolicy) []PolicyChange {
	if !base.Present {
		return nil
	}
	changes := lifecycleWeakenings(base, head)
	changes = append(changes, strictnessWeakenings(base, head)...)
	changes = append(changes, healthBlockWeakenings(base, head)...)
	changes = append(changes, poamWeakenings(base, head)...)
	changes = append(changes, decisionWeakenings(base, head)...)
	changes = append(changes, BaselineWeakenings(base, head)...)
	SortPolicyChanges(changes)
	return changes
}

// SortPolicyChanges sorts changes by aspect, then message.
//
// TRLC-LINKS: REQ-EMG-155
func SortPolicyChanges(changes []PolicyChange) {
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].Aspect != changes[j].Aspect {
			return changes[i].Aspect < changes[j].Aspect
		}
		return changes[i].Message < changes[j].Message
	})
}

// lifecycleWeakenings lists the bundles the head removes and a tier decision
// it changes without raising the tier. (A lower tier, and a removed block, are
// PolicyWeakenings.)
//
// TRLC-LINKS: REQ-EMG-155
func lifecycleWeakenings(base, head ModelPolicy) []PolicyChange {
	if !base.HasLifecycle || !head.HasLifecycle {
		return nil
	}
	var out []PolicyChange
	if removed := removedBundles(base, head); len(removed) > 0 {
		out = append(out, PolicyChange{Aspect: PolicyLifecycle,
			Message: fmt.Sprintf("lifecycle.bundles no longer has %s", strings.Join(removed, ", "))})
	}
	if tierDecisionLoosened(base, head) {
		out = append(out, PolicyChange{Aspect: PolicyLifecycle,
			Message: fmt.Sprintf("lifecycle.tierDecision changes from %s to %s without a higher tier", base.TierDecision, valueLabel(head.TierDecision))})
	}
	return out
}

// removedBundles lists the bundles of base that head does not have, sorted.
//
// TRLC-LINKS: REQ-EMG-155
func removedBundles(base, head ModelPolicy) []string {
	have := map[string]bool{}
	for _, b := range head.Bundles {
		have[b] = true
	}
	var removed []string
	for _, b := range base.Bundles {
		if !have[b] {
			removed = append(removed, b)
		}
	}
	sort.Strings(removed)
	return removed
}

// tierDecisionLoosened reports whether the head changes a tier decision of the
// base without raising the tier.
//
// TRLC-LINKS: REQ-EMG-155
func tierDecisionLoosened(base, head ModelPolicy) bool {
	if base.TierDecision == "" || head.TierDecision == base.TierDecision {
		return false
	}
	return model.TierRank(head.Tier) <= model.TierRank(base.Tier)
}

// valueLabel is a value for a message.
//
// TRLC-LINKS: REQ-EMG-155
func valueLabel(v string) string {
	if v == "" {
		return "none"
	}
	return v
}

// lintRank orders the lint modes: strict above guided above none.
//
// TRLC-LINKS: REQ-EMG-155
func lintRank(mode string) int {
	switch mode {
	case "strict":
		return 2
	case "guided":
		return 1
	}
	return 0
}

// strictnessWeakenings lists a lint mode that is removed or relaxed and
// requirementLinks that leaves strict.
//
// TRLC-LINKS: REQ-EMG-155
func strictnessWeakenings(base, head ModelPolicy) []PolicyChange {
	var out []PolicyChange
	if lintRank(head.LintMode) < lintRank(base.LintMode) {
		out = append(out, PolicyChange{Aspect: PolicyLint,
			Message: fmt.Sprintf("lintRun.mode is relaxed from %s to %s", base.LintMode, valueLabel(head.LintMode))})
	}
	if base.RequirementLinks == "strict" && head.RequirementLinks != "strict" {
		out = append(out, PolicyChange{Aspect: PolicyRequirementLinks,
			Message: fmt.Sprintf("requirementLinks changes from strict to %s", valueLabel(head.RequirementLinks))})
	}
	return out
}

// healthBlockWeakenings lists a removed health block and a removed baseline.
//
// TRLC-LINKS: REQ-EMG-155
func healthBlockWeakenings(base, head ModelPolicy) []PolicyChange {
	var out []PolicyChange
	if base.HasHealth && !head.HasHealth {
		out = append(out, PolicyChange{Aspect: PolicyHealthBlock, Message: "the health block is removed"})
	} else if base.BaselineConfigured && !head.BaselineConfigured {
		out = append(out, PolicyChange{Aspect: PolicyHealthBlock, Message: "health.baseline is removed"})
	}
	return out
}

// poamWeakenings lists the POA&M items that defer more than at the base: an item
// that is new, a defers entry that is new, and an item with defers whose due
// date or milestone moved or that was reopened.
//
// TRLC-LINKS: REQ-EMG-155
func poamWeakenings(base, head ModelPolicy) []PolicyChange {
	ids := make([]string, 0, len(head.Poam))
	for id := range head.Poam {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []PolicyChange
	for _, id := range ids {
		item := head.Poam[id]
		if len(item.Defers) == 0 {
			continue
		}
		before, existed := base.Poam[id]
		if !existed {
			out = append(out, PolicyChange{Aspect: PolicyPoam,
				Message: fmt.Sprintf("poamItems %s is new and defers %s", id, deferLabels(item.Defers))})
			continue
		}
		if added := newDefers(before.Defers, item.Defers); len(added) > 0 {
			out = append(out, PolicyChange{Aspect: PolicyPoam,
				Message: fmt.Sprintf("poamItems %s defers %s more than at the base", id, deferLabels(added))})
		}
		out = append(out, poamItemMoves(id, before, item)...)
	}
	return out
}

// poamItemMoves lists how an item that defers moved later against the base: a
// later due date, a changed milestone, or a completed item that is open again.
//
// TRLC-LINKS: REQ-EMG-155
func poamItemMoves(id string, before, after model.POAMItem) []PolicyChange {
	var out []PolicyChange
	if dueLater(before.DueDate, after.DueDate) {
		out = append(out, PolicyChange{Aspect: PolicyPoam,
			Message: fmt.Sprintf("poamItems %s moves dueDate from %s to %s", id, before.DueDate, after.DueDate)})
	}
	if strings.TrimSpace(before.Milestone) != strings.TrimSpace(after.Milestone) {
		out = append(out, PolicyChange{Aspect: PolicyPoam,
			Message: fmt.Sprintf("poamItems %s changes milestone from %q to %q", id, before.Milestone, after.Milestone)})
	}
	if strings.EqualFold(strings.TrimSpace(before.Status), "completed") && !strings.EqualFold(strings.TrimSpace(after.Status), "completed") {
		out = append(out, PolicyChange{Aspect: PolicyPoam,
			Message: fmt.Sprintf("poamItems %s is reopened from completed to %s", id, valueLabel(after.Status))})
	}
	return out
}

// dueLater reports whether an item's due date moves later. A date that cannot
// be read counts as later when it differs.
//
// TRLC-LINKS: REQ-EMG-155
func dueLater(before, after string) bool {
	before, after = strings.TrimSpace(before), strings.TrimSpace(after)
	if before == after {
		return false
	}
	b, bok := parseDue(before)
	a, aok := parseDue(after)
	if bok && aok {
		return a.After(b)
	}
	return true
}

// parseDue reads the date of a due date (YYYY-MM-DD, with any time after it).
//
// TRLC-LINKS: REQ-EMG-155
func parseDue(s string) (time.Time, bool) {
	if len(s) < 10 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", s[:10])
	return t, err == nil
}

// newDefers lists the entries of after that before does not have.
//
// TRLC-LINKS: REQ-EMG-155
func newDefers(before, after []model.POAMDefer) []model.POAMDefer {
	have := map[model.POAMDefer]bool{}
	for _, d := range before {
		have[d] = true
	}
	var out []model.POAMDefer
	for _, d := range after {
		if !have[d] {
			out = append(out, d)
		}
	}
	return out
}

// deferLabels names defers entries for a message, at most three.
//
// TRLC-LINKS: REQ-EMG-155
func deferLabels(defers []model.POAMDefer) string {
	labels := make([]string, 0, len(defers))
	for _, d := range defers {
		label := d.Code
		if d.Path != "" {
			label += " at " + d.Path
		}
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return summarize(labels, len(labels), "diagnostic(s)")
}

// summarize names the first examplesInMessage labels and counts the rest.
//
// TRLC-LINKS: REQ-EMG-155
func summarize(labels []string, total int, noun string) string {
	if len(labels) > examplesInMessage {
		return fmt.Sprintf("%d %s (%s and %d more)", total, noun, strings.Join(labels[:examplesInMessage], ", "), len(labels)-examplesInMessage)
	}
	return fmt.Sprintf("%d %s (%s)", total, noun, strings.Join(labels, ", "))
}

// decisionWeakenings lists the tier and health decisions the head accepts and
// the base did not.
//
// TRLC-LINKS: REQ-EMG-155
func decisionWeakenings(base, head ModelPolicy) []PolicyChange {
	var out []PolicyChange
	for id := range head.Accepted {
		if !base.Accepted[id] {
			out = append(out, PolicyChange{Aspect: PolicyDecisionRecord,
				Message: fmt.Sprintf("decision %s is newly accepted; a tier or health decision changes the policy", id)})
		}
	}
	return out
}

// BaselineWeakenings lists how the baseline of head grows against the base: a
// finding that is new in it or whose recorded value is higher. A baseline with
// entries where the base had none is allowed only as the first introduction:
// the base has no baseline and no health block. A baseline file that cannot be
// read at the head is health.baseline_unreadable and not compared; one that
// cannot be read at the base has no entries, so every entry of the head is new.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func BaselineWeakenings(base, head ModelPolicy) []PolicyChange {
	if !base.Present || !head.BaselineRead || len(head.Baseline) == 0 {
		return nil
	}
	if !base.BaselineConfigured {
		return baselineIntroduction(base, head)
	}
	added, grown := baselineGrowth(base.Baseline, head.Baseline)
	var out []PolicyChange
	if len(added) > 0 {
		out = append(out, PolicyChange{Aspect: PolicyBaseline,
			Message: "the baseline accepts " + summarize(added, len(added), "finding(s) it did not at the base")})
	}
	if len(grown) > 0 {
		out = append(out, PolicyChange{Aspect: PolicyBaseline,
			Message: "the baseline raises the accepted value of " + summarize(grown, len(grown), "finding(s)")})
	}
	return out
}

// baselineIntroduction is the verdict on a baseline the base did not have: it
// is the first introduction, and allowed, when the base had no health block
// either.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func baselineIntroduction(base, head ModelPolicy) []PolicyChange {
	if !base.HasHealth {
		return nil
	}
	return []PolicyChange{{Aspect: PolicyBaseline,
		Message: fmt.Sprintf("a baseline of %d finding(s) is introduced, but the base had a health block without one", len(head.Baseline))}}
}

// baselineGrowth names, sorted, the findings that head has and base does not,
// and those whose recorded value is higher at head.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func baselineGrowth(base, head map[model.HealthBaselineKey]int) (added, grown []string) {
	for key, value := range head {
		was, ok := base[key]
		if !ok {
			added = append(added, fmt.Sprintf("%s %s in %s", key.Code, key.Function, key.Path))
		} else if value > was {
			grown = append(grown, fmt.Sprintf("%s %s in %s: %d to %d", key.Code, key.Function, key.Path, was, value))
		}
	}
	sort.Strings(added)
	sort.Strings(grown)
	return added, grown
}
