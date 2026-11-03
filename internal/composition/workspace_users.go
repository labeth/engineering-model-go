// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

// "Used by" for a child: the requirement links of the sibling checkouts of
// the active workspace that point at this module's published contract
// items. It is computed at render time from what the reader can already see
// and never written into the child.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
)

// LinkUse is one requirement link of a using model in the workspace onto a
// contract item of this model.
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
type LinkUse struct {
	Model  string // model id of the using model
	Link   string // its link id
	Parent string // its parent requirement ("" for an evidence link)
	Item   string // this model's contract item (local id)
	Status string // claimed or accepted
}

// WorkspaceLinkUsers finds the workspace of bundle's manifest under mode
// (empty: ENGMOD_WORK, else search upwards; "off": none) and returns, sorted,
// every reqLinks entry of a sibling checkout (a replacement other than this
// module, or the model at the workspace root) whose child names one of this
// model's published contract items through a dependency on this module path.
// Without an active workspace it returns nothing; a replacement outside the
// workspace directory, a sibling whose manifest names a path outside its own
// directory and a sibling that does not load are skipped.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-108
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func WorkspaceLinkUsers(bundle model.Bundle, mode string) []LinkUse {
	modulePath := strings.TrimSpace(bundle.Manifest.Module.Path)
	if modulePath == "" || bundle.ManifestPath == "" {
		return nil
	}
	items := reqlinks.PublishedItems(bundle, bundle.Manifest.Publications...)
	if len(items) == 0 {
		return nil
	}
	doc, path, err := model.FindWorkspaceWith(bundle.ManifestPath, mode)
	if err != nil || path == "" {
		return nil
	}
	self, _ := filepath.Abs(filepath.Dir(bundle.ManifestPath))
	// The checkouts of the workspace: every replacement, and the model at
	// the workspace root itself (a system whose engmod.work.yml replaces
	// its children below it).
	workspace := filepath.Dir(path)
	dirs := []string{workspace}
	for _, r := range doc.Replacements {
		dir := filepath.Clean(filepath.Join(workspace, r.Path))
		// Like a composition replacement, a sibling must lie inside the
		// workspace directory.
		if r.Module != modulePath && PathWithin(workspace, dir) {
			dirs = append(dirs, dir)
		}
	}
	var uses []LinkUse
	seen := map[string]bool{}
	for _, dir := range dirs {
		if dir == self || seen[dir] {
			continue
		}
		seen[dir] = true
		manifest := filepath.Join(dir, "engmod.yml")
		if _, err := os.Stat(manifest); err != nil {
			continue
		}
		// A sibling is another repository's content: confine it to its
		// directory like a dependency before reading any file it names.
		if len(dependencyPathEscapes(filepath.Base(dir), dir, manifest)) > 0 {
			continue
		}
		sibling, err := model.LoadBundle(manifest)
		if err != nil {
			continue
		}
		aliases := map[string]bool{}
		for _, d := range sibling.Manifest.Dependencies {
			if strings.TrimSpace(d.Path) == modulePath {
				aliases[strings.TrimSpace(d.Alias)] = true
			}
		}
		for _, l := range sibling.ReqLinks.ReqLinks {
			alias, id, err := model.ParseQualifiedReference(strings.TrimSpace(l.Child))
			if err != nil || !aliases[alias] {
				continue
			}
			if _, ok := items[id]; !ok {
				continue
			}
			uses = append(uses, LinkUse{
				Model: strings.TrimSpace(sibling.Manifest.Module.ModelID), Link: strings.TrimSpace(l.ID),
				Parent: strings.TrimSpace(l.Parent), Item: id, Status: strings.TrimSpace(l.Status),
			})
		}
	}
	sort.SliceStable(uses, func(i, j int) bool {
		a, b := uses[i], uses[j]
		if a.Item != b.Item {
			return a.Item < b.Item
		}
		if a.Model != b.Model {
			return a.Model < b.Model
		}
		return a.Link < b.Link
	})
	return uses
}
