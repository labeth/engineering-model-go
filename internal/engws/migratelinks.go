// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
	"github.com/labeth/engineering-model-go/internal/yamledit"
)

// DefaultReqLinksPath is the requirement link file migrate-links creates.
const DefaultReqLinksPath = "model/req-links.yml"

// MigratedLink is one allocation converted to a requirement link.
// TRLC-LINKS: REQ-EMG-104
type MigratedLink struct {
	Link model.ReqLink
	Note string // why the target's kind could not be checked, when it could not
}

// KeptAllocation is an allocation migrate-links leaves in place because its
// target is not a link target (D13), with the reason printed for it.
// TRLC-LINKS: REQ-EMG-104
type KeptAllocation struct {
	Requirement, Target string
	Reason              string
}

// MigrateResult is what migrate-links changed (or, in a dry run, would
// change) and the requirements that report requirement.no_parent after it,
// sorted by ID, with the severity the model's strictness gives them.
// TRLC-LINKS: REQ-EMG-104
type MigrateResult struct {
	Links []MigratedLink
	Kept  []KeptAllocation
	// Removed are the removed fields the model still carries
	// (model.pinned_fingerprint_removed, model.provides_ref_removed, ADR-EMG-075):
	// reported with their remedy, never edited.
	Removed          []validate.Diagnostic
	AddedDocument    bool
	Files            []string // files written (or to write)
	NoParent         []string
	NoParentSeverity validate.Severity
}

// Changed reports whether the migration changes anything.
// TRLC-LINKS: REQ-EMG-104
func (r MigrateResult) Changed() bool { return len(r.Files) > 0 }

// MigrateLinks converts the deprecated requirement trace of the model in
// repo: each composition.allocations entry becomes a claimed reqLinks entry
// pinned to the dependency version (a parent link to a capability, an
// evidence link to an assumption of use), documents.reqLinks is added when
// missing and the allocations are removed. It writes no pinned.fingerprint and
// edits no contract.provides[].ref or pinned.fingerprint the model still
// carries: both are errors of their own (ADR-EMG-075) and are only reported in
// Removed. Every edit keeps the other lines of the files. dryRun writes
// nothing. A second run changes nothing.
// TRLC-LINKS: REQ-EMG-104, REQ-EMG-118, REQ-EMG-184
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-MIGRATE
func MigrateLinks(repo string, dryRun bool) (MigrateResult, error) {
	var result MigrateResult
	manifestPath := filepath.Join(repo, "engmod.yml")
	bundle, err := model.LoadBundle(manifestPath)
	if err != nil {
		return result, err
	}
	writes := map[string][]byte{}
	read := func(path string) ([]byte, error) {
		if data, ok := writes[path]; ok {
			return data, nil
		}
		return os.ReadFile(path)
	}

	// Allocations -> reqLinks.
	if allocations := bundle.Architecture.Composition.Allocations; len(allocations) > 0 {
		if err := migrateAllocations(bundle, manifestPath, read, writes, &result); err != nil {
			return result, err
		}
	}

	result.Removed = validate.RemovedFieldDiagnostics(bundle)

	for path := range writes {
		result.Files = append(result.Files, path)
	}
	sort.Strings(result.Files)
	if !dryRun {
		if err := writeAllOrNothing(result.Files, writes); err != nil {
			return result, err
		}
	}

	// requirement.no_parent after the migration (as the files now are, or
	// would be).
	after := bundle
	if !dryRun && result.Changed() {
		if after, err = model.LoadBundle(manifestPath); err != nil {
			return result, fmt.Errorf("the migrated model does not load: %w", err)
		}
	} else if dryRun {
		after = applyInMemory(bundle, result)
	}
	for _, d := range validate.Bundle(after) {
		if d.Code != "requirement.no_parent" {
			continue
		}
		var i int
		if _, err := fmt.Sscanf(d.Path, "requirements[%d]", &i); err == nil && i >= 0 && i < len(after.Requirements.Requirements) {
			result.NoParent = append(result.NoParent, strings.TrimSpace(after.Requirements.Requirements[i].ID))
			result.NoParentSeverity = d.Severity
		}
	}
	sort.Strings(result.NoParent)
	return result, nil
}

// writeAllOrNothing writes every file of paths with its content in writes.
// It snapshots each target first; when a write fails, every file already
// written is restored to its snapshot and a file that did not exist is
// removed, so the model is never left half migrated.
// TRLC-LINKS: REQ-EMG-104
func writeAllOrNothing(paths []string, writes map[string][]byte) error {
	type snapshot struct {
		data    []byte
		existed bool
	}
	snapshots := map[string]snapshot{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			snapshots[path] = snapshot{data: data, existed: true}
		case os.IsNotExist(err):
			snapshots[path] = snapshot{}
		default:
			return err
		}
	}
	for i, path := range paths {
		if err := writeFile(path, writes[path]); err != nil {
			var failed []string
			for _, done := range paths[:i] {
				snap := snapshots[done]
				var rerr error
				if snap.existed {
					rerr = writeFileAtomic(done, snap.data)
				} else if rmErr := os.Remove(done); rmErr != nil && !os.IsNotExist(rmErr) {
					rerr = rmErr
				}
				if rerr != nil {
					failed = append(failed, fmt.Sprintf("%s (%v)", done, rerr))
				}
			}
			if len(failed) > 0 {
				return fmt.Errorf("write %s: %w; restoring also failed for %s", path, err, strings.Join(failed, ", "))
			}
			return fmt.Errorf("write %s: %w; every file restored, nothing migrated", path, err)
		}
	}
	return nil
}

// applyInMemory applies the links of a dry run to a bundle copy, so the
// no_parent list matches what a real run would leave.
// TRLC-LINKS: REQ-EMG-104
func applyInMemory(b model.Bundle, result MigrateResult) model.Bundle {
	links := append([]model.ReqLink(nil), b.ReqLinks.ReqLinks...)
	for _, m := range result.Links {
		links = append(links, m.Link)
	}
	b.ReqLinks.ReqLinks = links
	return b
}

// migrateAllocations writes one claimed link per allocation, adds
// documents.reqLinks when missing and removes the converted allocations. An
// allocation whose target resolves to a contract entry that is not a
// published capability or assumption of use (an interface entry, D13) is
// kept in place and reported; one whose dependency cannot be resolved is
// converted with a note. A target that is an assumption of use (AOU-*)
// becomes an evidence link (evidence: [requirement], no parent).
// TRLC-LINKS: REQ-EMG-104
func migrateAllocations(b model.Bundle, manifestPath string, read func(string) ([]byte, error), writes map[string][]byte, result *MigrateResult) error {
	pins := map[string]string{}
	for _, d := range b.Manifest.Dependencies {
		pins[strings.TrimSpace(d.Alias)] = strings.TrimSpace(d.Version)
	}
	deps, resolveNote := resolveDependencyItems(b)
	used := map[string]bool{}
	for _, l := range b.ReqLinks.ReqLinks {
		used[strings.TrimSpace(l.ID)] = true
	}
	var converted []int
	for i, a := range b.Architecture.Composition.Allocations {
		target := strings.TrimSpace(a.Target)
		alias, itemID, err := model.ParseQualifiedReference(target)
		if err != nil {
			return fmt.Errorf("allocation of %s: target %q is not alias::ID: %w", a.Requirement, a.Target, err)
		}
		dep, resolved := deps[alias]
		item, isItem := dep.items[itemID]
		if resolved && !isItem {
			result.Kept = append(result.Kept, KeptAllocation{Requirement: strings.TrimSpace(a.Requirement), Target: target, Reason: notLinkTarget(dep.bundle, alias, itemID)})
			continue
		}
		link := model.ReqLink{
			ID: uniqueLinkID(linkID(a.Requirement, alias), used), Child: target,
			Pinned: model.ReqLinkPin{Version: pins[alias]}, Status: model.ReqLinkClaimed, Rationale: oneLine(a.Rationale),
		}
		if (isItem && item.Kind == reqlinks.KindAssumption) || (!isItem && strings.HasPrefix(itemID, "AOU-")) {
			link.Evidence = []string{strings.TrimSpace(a.Requirement)}
		} else {
			link.Parent = strings.TrimSpace(a.Requirement)
		}
		migrated := MigratedLink{Link: link}
		if !isItem {
			migrated.Note = fmt.Sprintf("dependency %s could not be resolved (%s), so the target's kind was not checked; the check reports the link until it resolves", alias, resolveNote[alias])
		}
		result.Links = append(result.Links, migrated)
		converted = append(converted, i)
	}
	if len(converted) == 0 {
		return nil
	}

	links := make([]model.ReqLink, 0, len(result.Links))
	for _, m := range result.Links {
		links = append(links, m.Link)
	}
	added, err := appendLinks(b, manifestPath, links, read, writes)
	if err != nil {
		return err
	}
	result.AddedDocument = added

	archData, err := read(b.ArchitecturePath)
	if err != nil {
		return err
	}
	if len(converted) == len(b.Architecture.Composition.Allocations) {
		archData, _, err = yamledit.RemoveKey(archData, yamledit.FindPath("architecture", "composition"), "allocations")
	} else {
		archData, err = yamledit.RemoveSequenceItems(archData, yamledit.FindPath("architecture", "composition"), "allocations", converted)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", b.ArchitecturePath, err)
	}
	writes[b.ArchitecturePath] = archData
	return nil
}

// appendLinks adds links to the requirement link file of b in writes:
// appended to the existing file, line-preserving, or a new
// model/req-links.yml with documents.reqLinks added to engmod.yml; it
// reports whether it added the document.
// TRLC-LINKS: REQ-EMG-104, REQ-EMG-119
func appendLinks(b model.Bundle, manifestPath string, links []model.ReqLink, read func(string) ([]byte, error), writes map[string][]byte) (bool, error) {
	added := false
	linksRel := strings.TrimSpace(b.Manifest.Documents.ReqLinks)
	if linksRel == "" {
		linksRel = DefaultReqLinksPath
		manifest, err := read(manifestPath)
		if err != nil {
			return false, err
		}
		if manifest, err = yamledit.SetKey(manifest, yamledit.FindPath("documents"), "reqLinks", yamledit.Scalar(linksRel)); err != nil {
			return false, fmt.Errorf("%s: %w", manifestPath, err)
		}
		writes[manifestPath] = manifest
		added = true
	}
	linksPath := filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(linksRel))
	existing, err := read(linksPath)
	var blocks []string
	for _, l := range links {
		blocks = append(blocks, linkBlock(l))
	}
	switch {
	case os.IsNotExist(err):
		writes[linksPath] = []byte(model.SchemaVersionLine + "\nreqLinks:\n" + strings.Join(blocks, ""))
	case err != nil:
		return added, err
	default:
		if existing, err = yamledit.AppendSequenceItems(existing, yamledit.FindPath(), "reqLinks", blocks); err != nil {
			return added, fmt.Errorf("%s: %w", linksPath, err)
		}
		writes[linksPath] = existing
	}
	return added, nil
}

// notLinkTarget says why alias::id of a resolved dependency is no link
// target (reqlinks.NonItemReason): an unpublished item, an interface (or
// other non-item) contract entry, or no contract item at all.
// TRLC-LINKS: REQ-EMG-104
func notLinkTarget(dep model.Bundle, alias, id string) string {
	switch reqlinks.NonItemReason(dep, id) {
	case reqlinks.ReasonNotInPublication:
		return fmt.Sprintf("%s is not in a selected publication of %s; publish it and link that", id, alias)
	case reqlinks.ReasonNotContractItem:
		kind := "contract"
		for _, entries := range [][]model.ContractEntry{dep.Architecture.Contract.Provides, dep.Architecture.Contract.Requires} {
			for _, e := range entries {
				if strings.TrimSpace(e.ID) == id && strings.TrimSpace(e.Kind) != "" {
					kind = strings.ToLower(strings.TrimSpace(e.Kind))
				}
			}
		}
		return kind + " entries are not link targets; publish a capability that realizes it and link that"
	}
	return fmt.Sprintf("%s is not a capability or assumption of use of %s; link a published contract item", id, alias)
}

// linkID is LNK-<requirement without REQ->-<ALIAS>, upper case.
// TRLC-LINKS: REQ-EMG-104
func linkID(requirement, alias string) string {
	req := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(requirement)), "REQ-")
	clean := func(s string) string {
		var b strings.Builder
		for _, r := range strings.ToUpper(s) {
			if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			} else {
				b.WriteRune('-')
			}
		}
		return strings.Trim(b.String(), "-")
	}
	return "LNK-" + clean(req) + "-" + clean(alias)
}

// uniqueLinkID returns id, or id-2, id-3 ... when taken, and marks it used.
// TRLC-LINKS: REQ-EMG-104
func uniqueLinkID(id string, used map[string]bool) string {
	candidate := id
	for n := 2; used[candidate]; n++ {
		candidate = fmt.Sprintf("%s-%d", id, n)
	}
	used[candidate] = true
	return candidate
}

// linkBlock renders a link as a block sequence item at column 0.
// TRLC-LINKS: REQ-EMG-104
func linkBlock(l model.ReqLink) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- id: %s\n", yamledit.Scalar(l.ID))
	if l.Parent != "" {
		fmt.Fprintf(&b, "  parent: %s\n", yamledit.Scalar(l.Parent))
	}
	fmt.Fprintf(&b, "  child: %s\n", yamledit.Scalar(l.Child))
	if len(l.Evidence) > 0 {
		b.WriteString("  evidence:\n")
		for _, e := range l.Evidence {
			fmt.Fprintf(&b, "  - %s\n", yamledit.Scalar(e))
		}
	}
	b.WriteString("  pinned:\n")
	fmt.Fprintf(&b, "    version: %s\n", yamledit.Scalar(l.Pinned.Version))
	fmt.Fprintf(&b, "  status: %s\n", l.Status)
	if l.Rationale != "" {
		fmt.Fprintf(&b, "  rationale: %s\n", yamledit.Scalar(l.Rationale))
	}
	return b.String()
}

// resolvedDependency is a resolved dependency's model and published
// contract items.
type resolvedDependency struct {
	bundle model.Bundle
	items  map[string]reqlinks.Item
}

// resolveDependencyItems composes the model (lock check, module cache
// reuse, so no lock is written) and returns each resolved dependency by
// alias with its published contract items, and a reason per alias that did
// not resolve.
// TRLC-LINKS: REQ-EMG-104
func resolveDependencyItems(b model.Bundle) (map[string]resolvedDependency, map[string]string) {
	deps := map[string]resolvedDependency{}
	notes := map[string]string{}
	for _, d := range b.Manifest.Dependencies {
		notes[strings.TrimSpace(d.Alias)] = "not composed"
	}
	comp, err := engmodel.GenerateCompositionForBundle(b, engmodel.ResolutionOptions{Lock: "check", ModuleCache: "reuse"})
	if err != nil || comp.Root == nil {
		reason := "composition failed"
		if err != nil {
			reason = err.Error()
		}
		for alias := range notes {
			notes[alias] = reason
		}
		return deps, notes
	}
	for _, d := range comp.Diagnostics {
		if d.Severity == validate.SeverityError && strings.HasPrefix(d.Code, "composition.") {
			for alias := range notes {
				if strings.Contains(d.Message, fmt.Sprintf("%q", alias)) {
					notes[alias] = d.Code + ": " + d.Message
				}
			}
		}
	}
	for _, dep := range engmodel.LinkDependencies(comp.Root) {
		deps[dep.Alias] = resolvedDependency{bundle: dep.Bundle, items: reqlinks.PublishedItems(dep.Bundle, dep.Publications...)}
	}
	return deps, notes
}

// runMigrateLinks prints the migration and the requirements without an
// upward trace.
// TRLC-LINKS: REQ-EMG-104
// ENGMODEL-LINKS: EVT-LINK-MIGRATION-REQUESTED
func runMigrateLinks(repo string, dryRun bool, w io.Writer) (int, error) {
	if abs, err := filepath.Abs(repo); err == nil {
		repo = abs
	}
	result, err := MigrateLinks(repo, dryRun)
	if err != nil {
		return 1, err
	}
	prefix := ""
	if dryRun {
		prefix = "dry run: "
	}
	for _, m := range result.Links {
		from := m.Link.Parent
		if from == "" {
			from = "evidence " + strings.Join(m.Link.Evidence, ",")
		}
		fmt.Fprintf(w, "  %slink %s %s -> %s (%s, %s)\n", prefix, m.Link.ID, from, m.Link.Child, m.Link.Pinned.Version, m.Link.Status)
		if m.Note != "" {
			fmt.Fprintf(w, "      %s\n", m.Note)
		}
	}
	for _, k := range result.Kept {
		fmt.Fprintf(w, "  %skept allocation %s → %s: %s\n", prefix, k.Requirement, k.Target, k.Reason)
	}
	for _, d := range result.Removed {
		fmt.Fprintf(w, "  %snot migrated, edit the file: %s %s: %s; %s\n", prefix, d.Path, d.Code, d.Message, d.Remedy)
	}
	if result.AddedDocument {
		fmt.Fprintf(w, "  %sadded documents.reqLinks: %s\n", prefix, DefaultReqLinksPath)
	}
	if !result.Changed() {
		fmt.Fprintln(w, "engws: nothing to migrate")
	} else {
		verb := "wrote"
		if dryRun {
			verb = "would write"
		}
		rel := make([]string, 0, len(result.Files))
		for _, f := range result.Files {
			if r, err := filepath.Rel(repo, f); err == nil {
				f = r
			}
			rel = append(rel, f)
		}
		fmt.Fprintf(w, "engws: %s %s\n", verb, strings.Join(rel, ", "))
	}
	fmt.Fprintf(w, "engws: %d requirement(s) without an upward trace (requirement.no_parent):\n", len(result.NoParent))
	for _, id := range result.NoParent {
		fmt.Fprintf(w, "  requirement.no_parent: %s [%s]\n", id, result.NoParentSeverity)
	}
	if len(result.NoParent) > 0 {
		fmt.Fprintln(w, "engws: give each a realizes naming the contract item it serves, or mark it derived with derivedRationale")
	}
	return 0, nil
}
