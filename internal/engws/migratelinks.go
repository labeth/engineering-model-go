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
)

// DefaultReqLinksPath is the requirement link file migrate-links creates.
const DefaultReqLinksPath = "model/req-links.yml"

// MigratedLink is one allocation converted to a requirement link.
// TRLC-LINKS: REQ-EMG-104
type MigratedLink struct {
	Link model.ReqLink
	Note string // why the fingerprint is empty, when it is
}

// MigratedRef is one contract ref converted to realizes (or kept, with Note).
// TRLC-LINKS: REQ-EMG-104
type MigratedRef struct {
	Capability, Requirement string
	Note                    string
}

// MigrateResult is what migrate-links changed (or, in a dry run, would
// change) and the requirements that report requirement.no_parent after it,
// sorted by ID, with the severity the model's strictness gives them.
// TRLC-LINKS: REQ-EMG-104
type MigrateResult struct {
	Links            []MigratedLink
	Refs             []MigratedRef
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
// pinned to the dependency version with the current fingerprint (empty when
// the target cannot be resolved), documents.reqLinks is added when missing,
// the allocations are removed, and each capability contract.provides[].ref
// becomes realizes on the referenced requirement. Every edit keeps the other
// lines of the files. dryRun writes nothing. A second run changes nothing.
// TRLC-LINKS: REQ-EMG-104
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
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

	// Contract refs -> realizes.
	if err := migrateRefs(bundle, read, writes, &result); err != nil {
		return result, err
	}

	// Allocations -> reqLinks.
	if allocations := bundle.Architecture.Composition.Allocations; len(allocations) > 0 {
		if err := migrateAllocations(bundle, manifestPath, read, writes, &result); err != nil {
			return result, err
		}
	}

	for path := range writes {
		result.Files = append(result.Files, path)
	}
	sort.Strings(result.Files)
	if !dryRun {
		for _, path := range result.Files {
			if err := writeFileAtomic(path, writes[path]); err != nil {
				return result, err
			}
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

// applyInMemory applies the ref conversion of a dry run to a bundle copy,
// so the no_parent list matches what a real run would leave.
// TRLC-LINKS: REQ-EMG-104
func applyInMemory(b model.Bundle, result MigrateResult) model.Bundle {
	reqs := append([]model.Requirement(nil), b.Requirements.Requirements...)
	for _, ref := range result.Refs {
		if ref.Note != "" {
			continue
		}
		for i := range reqs {
			if strings.TrimSpace(reqs[i].ID) == ref.Requirement {
				reqs[i].Realizes = append(append([]string(nil), reqs[i].Realizes...), ref.Capability)
			}
		}
	}
	b.Requirements.Requirements = reqs
	links := append([]model.ReqLink(nil), b.ReqLinks.ReqLinks...)
	for _, m := range result.Links {
		links = append(links, m.Link)
	}
	b.ReqLinks.ReqLinks = links
	return b
}

// migrateRefs moves each capability's contract ref to realizes on the
// referenced requirement (appended, deduplicated) and drops the ref. A ref
// on a non-capability entry or naming no local requirement is kept and
// reported.
// TRLC-LINKS: REQ-EMG-104
func migrateRefs(b model.Bundle, read func(string) ([]byte, error), writes map[string][]byte, result *MigrateResult) error {
	requirements := map[string]model.Requirement{}
	for _, r := range b.Requirements.Requirements {
		requirements[strings.TrimSpace(r.ID)] = r
	}
	realizes := map[string][]string{} // requirement -> capabilities to add
	var drop []string
	for _, entry := range b.Architecture.Contract.Provides {
		id, ref := strings.TrimSpace(entry.ID), strings.TrimSpace(entry.Ref)
		if ref == "" || strings.Contains(id, "::") {
			continue
		}
		r, known := requirements[ref]
		switch {
		case !strings.EqualFold(strings.TrimSpace(entry.Kind), string(reqlinks.KindCapability)):
			result.Refs = append(result.Refs, MigratedRef{Capability: id, Requirement: ref, Note: fmt.Sprintf("kept: %s is a %s, not a capability; realizes names capabilities and assumptions of use only", id, entry.Kind)})
			continue
		case !known:
			result.Refs = append(result.Refs, MigratedRef{Capability: id, Requirement: ref, Note: fmt.Sprintf("kept: %s is not a requirement of this model", ref)})
			continue
		}
		result.Refs = append(result.Refs, MigratedRef{Capability: id, Requirement: ref})
		if !containsTrimmed(r.Realizes, id) && !containsTrimmed(realizes[ref], id) {
			realizes[ref] = append(realizes[ref], id)
		}
		drop = append(drop, id)
	}
	if len(drop) == 0 {
		return nil
	}
	reqData, err := read(b.RequirementsPath)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(realizes))
	for id := range realizes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		values := append(append([]string(nil), requirements[id].Realizes...), realizes[id]...)
		if reqData, err = setKey(reqData, findListItem([]string{"requirements"}, "id", id), "realizes", flowList(values)); err != nil {
			return fmt.Errorf("%s: requirement %s: %w", b.RequirementsPath, id, err)
		}
	}
	if len(ids) > 0 {
		writes[b.RequirementsPath] = reqData
	}
	archData, err := read(b.ArchitecturePath)
	if err != nil {
		return err
	}
	for _, id := range drop {
		if archData, _, err = removeKey(archData, findListItem([]string{"architecture", "contract", "provides"}, "id", id), "ref"); err != nil {
			return fmt.Errorf("%s: contract entry %s: %w", b.ArchitecturePath, id, err)
		}
	}
	writes[b.ArchitecturePath] = archData
	return nil
}

// flowList encodes IDs as a YAML flow sequence.
// TRLC-LINKS: REQ-EMG-104
func flowList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, yamlScalar(strings.TrimSpace(v)))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// containsTrimmed reports whether values holds want after trimming.
// TRLC-LINKS: REQ-EMG-104
func containsTrimmed(values []string, want string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == want {
			return true
		}
	}
	return false
}

// migrateAllocations writes one claimed link per allocation, adds
// documents.reqLinks when missing and removes the allocations.
// TRLC-LINKS: REQ-EMG-104
func migrateAllocations(b model.Bundle, manifestPath string, read func(string) ([]byte, error), writes map[string][]byte, result *MigrateResult) error {
	pins := map[string]string{}
	for _, d := range b.Manifest.Dependencies {
		pins[strings.TrimSpace(d.Alias)] = strings.TrimSpace(d.Version)
	}
	items, resolveNote := resolveDependencyItems(b)
	used := map[string]bool{}
	for _, l := range b.ReqLinks.ReqLinks {
		used[strings.TrimSpace(l.ID)] = true
	}
	for _, a := range b.Architecture.Composition.Allocations {
		target := strings.TrimSpace(a.Target)
		alias, itemID, err := model.ParseQualifiedReference(target)
		if err != nil {
			return fmt.Errorf("allocation of %s: target %q is not alias::ID: %w", a.Requirement, a.Target, err)
		}
		link := model.ReqLink{
			ID: uniqueLinkID(linkID(a.Requirement, alias), used), Parent: strings.TrimSpace(a.Requirement), Child: target,
			Pinned: model.ReqLinkPin{Version: pins[alias]}, Status: model.ReqLinkClaimed, Rationale: oneLine(a.Rationale),
		}
		migrated := MigratedLink{Link: link}
		switch item, ok := items[alias][itemID]; {
		case ok:
			migrated.Link.Pinned.Fingerprint = reqlinks.Fingerprint(item)
		case items[alias] == nil:
			migrated.Note = fmt.Sprintf("fingerprint left empty: dependency %s could not be resolved (%s); the check reports the link until engws review-links --accept", alias, resolveNote[alias])
		default:
			migrated.Note = fmt.Sprintf("fingerprint left empty: %s is not a published capability or assumption of use of %s", itemID, alias)
		}
		result.Links = append(result.Links, migrated)
	}

	// The link file: extend the existing one, or create it.
	linksRel := strings.TrimSpace(b.Manifest.Documents.ReqLinks)
	if linksRel == "" {
		linksRel = DefaultReqLinksPath
		manifest, err := read(manifestPath)
		if err != nil {
			return err
		}
		if manifest, err = setKey(manifest, findPath("documents"), "reqLinks", yamlScalar(linksRel)); err != nil {
			return fmt.Errorf("%s: %w", manifestPath, err)
		}
		writes[manifestPath] = manifest
		result.AddedDocument = true
	}
	linksPath := filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(linksRel))
	existing, err := read(linksPath)
	var blocks []string
	for _, m := range result.Links {
		blocks = append(blocks, linkBlock(m.Link))
	}
	switch {
	case os.IsNotExist(err):
		writes[linksPath] = []byte("schemaVersion: 2\nreqLinks:\n" + strings.Join(blocks, ""))
	case err != nil:
		return err
	default:
		if existing, err = appendSequenceItems(existing, findPath(), "reqLinks", blocks); err != nil {
			return fmt.Errorf("%s: %w", linksPath, err)
		}
		writes[linksPath] = existing
	}

	archData, err := read(b.ArchitecturePath)
	if err != nil {
		return err
	}
	if archData, _, err = removeKey(archData, findPath("architecture", "composition"), "allocations"); err != nil {
		return fmt.Errorf("%s: %w", b.ArchitecturePath, err)
	}
	writes[b.ArchitecturePath] = archData
	return nil
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
	fmt.Fprintf(&b, "- id: %s\n", yamlScalar(l.ID))
	fmt.Fprintf(&b, "  parent: %s\n", yamlScalar(l.Parent))
	fmt.Fprintf(&b, "  child: %s\n", yamlScalar(l.Child))
	b.WriteString("  pinned:\n")
	fmt.Fprintf(&b, "    version: %s\n", yamlScalar(l.Pinned.Version))
	fmt.Fprintf(&b, "    fingerprint: %s\n", yamlScalar(l.Pinned.Fingerprint))
	fmt.Fprintf(&b, "  status: %s\n", l.Status)
	if l.Rationale != "" {
		fmt.Fprintf(&b, "  rationale: %s\n", yamlScalar(l.Rationale))
	}
	return b.String()
}

// resolveDependencyItems composes the model (lock check, module cache
// reuse, so no lock is written) and returns the published contract items of
// each resolved dependency by alias, and a reason per alias that did not
// resolve.
// TRLC-LINKS: REQ-EMG-104
func resolveDependencyItems(b model.Bundle) (map[string]map[string]reqlinks.Item, map[string]string) {
	items := map[string]map[string]reqlinks.Item{}
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
		return items, notes
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
		items[dep.Alias] = reqlinks.PublishedItems(dep.Bundle, dep.Publications...)
	}
	return items, notes
}

// runMigrateLinks prints the migration and the requirements without an
// upward trace.
// TRLC-LINKS: REQ-EMG-104
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
	for _, r := range result.Refs {
		if r.Note != "" {
			fmt.Fprintf(w, "  %sref %s -> %s %s\n", prefix, r.Capability, r.Requirement, r.Note)
		} else {
			fmt.Fprintf(w, "  %sref %s -> %s realizes [%s]\n", prefix, r.Capability, r.Requirement, r.Capability)
		}
	}
	for _, m := range result.Links {
		fmt.Fprintf(w, "  %slink %s %s -> %s (%s, %s)\n", prefix, m.Link.ID, m.Link.Parent, m.Link.Child, m.Link.Pinned.Version, m.Link.Status)
		if m.Note != "" {
			fmt.Fprintf(w, "      %s\n", m.Note)
		}
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
