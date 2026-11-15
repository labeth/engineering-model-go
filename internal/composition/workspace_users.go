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
	modulePath, items := publishedContract(bundle)
	if len(items) == 0 {
		return nil
	}
	doc, path, err := model.FindWorkspaceWith(bundle.ManifestPath, mode)
	if err != nil || path == "" {
		return nil
	}
	self, _ := filepath.Abs(filepath.Dir(bundle.ManifestPath))
	var uses []LinkUse
	for _, dir := range uniqueExcept(workspaceCheckouts(doc, filepath.Dir(path), modulePath), self) {
		uses = append(uses, linkUsesIn(dir, modulePath, items)...)
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

// publishedContract is the module path of a model and the contract items its
// publications publish; none when it has no module path or manifest.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-108
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func publishedContract(bundle model.Bundle) (string, map[string]reqlinks.Item) {
	modulePath := strings.TrimSpace(bundle.Manifest.Module.Path)
	if modulePath == "" || bundle.ManifestPath == "" {
		return modulePath, nil
	}
	return modulePath, reqlinks.PublishedItems(bundle, bundle.Manifest.Publications...)
}

// workspaceCheckouts lists the checkouts of the workspace: the model at the
// workspace root itself (a system whose engmod.work.yml replaces its children
// below it) and every replacement of another module that, like a composition
// replacement, lies inside the workspace directory.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-108
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func workspaceCheckouts(doc model.WorkspaceDocument, workspace, modulePath string) []string {
	dirs := []string{workspace}
	for _, r := range doc.Replacements {
		dir := filepath.Clean(filepath.Join(workspace, r.Path))
		if r.Module != modulePath && PathWithin(workspace, dir) {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// uniqueExcept is the directories in order, each once, without skip.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-108
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func uniqueExcept(dirs []string, skip string) []string {
	seen := map[string]bool{skip: true}
	var out []string
	for _, dir := range dirs {
		if !seen[dir] {
			seen[dir] = true
			out = append(out, dir)
		}
	}
	return out
}

// linkUsesIn is the requirement links of the checkout in dir that name a
// published contract item of the module. A checkout without a manifest, whose
// manifest names a path outside its own directory or that does not load is
// skipped.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-108
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func linkUsesIn(dir, modulePath string, items map[string]reqlinks.Item) []LinkUse {
	sibling, ok := loadSibling(dir)
	if !ok {
		return nil
	}
	aliases := dependencyAliases(sibling, modulePath)
	var uses []LinkUse
	for _, l := range sibling.ReqLinks.ReqLinks {
		alias, id, err := model.ParseQualifiedReference(strings.TrimSpace(l.Child))
		if _, ok := items[id]; err != nil || !aliases[alias] || !ok {
			continue
		}
		uses = append(uses, LinkUse{
			Model: strings.TrimSpace(sibling.Manifest.Module.ModelID), Link: strings.TrimSpace(l.ID),
			Parent: strings.TrimSpace(l.Parent), Item: id, Status: strings.TrimSpace(l.Status),
		})
	}
	return uses
}

// dependencyAliases is the aliases under which a model depends on the module.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-108
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func dependencyAliases(sibling model.Bundle, modulePath string) map[string]bool {
	aliases := map[string]bool{}
	for _, d := range sibling.Manifest.Dependencies {
		if strings.TrimSpace(d.Path) == modulePath {
			aliases[strings.TrimSpace(d.Alias)] = true
		}
	}
	return aliases
}

// loadSibling loads the model of a checkout, which must have a manifest and
// keep every path it names inside its own directory.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-108
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func loadSibling(dir string) (model.Bundle, bool) {
	manifest := filepath.Join(dir, "engmod.yml")
	if _, err := os.Stat(manifest); err != nil {
		return model.Bundle{}, false
	}
	// A sibling is another repository's content: confine it to its
	// directory like a dependency before reading any file it names.
	if len(dependencyPathEscapes(filepath.Base(dir), dir, manifest)) > 0 {
		return model.Bundle{}, false
	}
	sibling, err := model.LoadBundle(manifest)
	return sibling, err == nil
}
