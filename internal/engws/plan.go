// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Checkout states, in the order changed detection applies them.
const (
	StateNoGit       = "nogit"
	StateDirty       = "dirty"
	StateUntagged    = "untagged"
	StateUnpublished = "unpublished"
	StatePinned      = "pinned"
	StatePublished   = "published"
	StateUnknown     = "unknown"
)

// Entry is one checkout with its changed-detection result.
type Entry struct {
	Checkout
	Pinned  bool
	Reasons []string // why the checkout is changed, in detection order
	State   string
	Local   bool // listed as a replacement
	Note    string
}

// Plan is the workspace decision for every checkout under Root.
type Plan struct {
	Root    string
	Entries []Entry
	Notes   []string
	// Warnings name the checkouts Scan skipped as unsafe.
	Warnings []string
}

// BuildPlan classifies every checkout under root. A checkout is local when it
// is pinned, changed, or all is set; registry problems are notes, never
// changes, so an offline developer is not forced into local mode.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func BuildPlan(ctx context.Context, root string, registry Registry, all bool) (Plan, error) {
	checkouts, warnings, err := ScanWithWarnings(root)
	if err != nil {
		return Plan{}, err
	}
	pins, err := LoadPins(root)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Root: root, Entries: make([]Entry, len(checkouts)), Warnings: warnings}
	var wg sync.WaitGroup
	for i, checkout := range checkouts {
		wg.Add(1)
		go func(i int, checkout Checkout) {
			defer wg.Done()
			plan.Entries[i] = classify(ctx, checkout, registry, pins[checkout.Name], all)
		}(i, checkout)
	}
	wg.Wait()
	known := map[string]bool{}
	notes := map[string]bool{}
	for _, entry := range plan.Entries {
		known[entry.Name] = true
		if entry.Note != "" && !notes[entry.Note] {
			notes[entry.Note] = true
			plan.Notes = append(plan.Notes, entry.Note)
		}
	}
	for pin := range pins {
		if !known[pin] {
			plan.Notes = append(plan.Notes, fmt.Sprintf("pin %q in %s names no checkout under the root", pin, StateFileName))
		}
	}
	sort.Strings(plan.Notes)
	return plan, nil
}

// classify applies changed detection in order: no own git repository, dirty
// tree, HEAD not at tag v<version>, version not published.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func classify(ctx context.Context, checkout Checkout, registry Registry, pinned, all bool) Entry {
	entry := Entry{Checkout: checkout, Pinned: pinned}
	version := checkout.Manifest.Module.Version
	tag := tagFor(version)
	if own, why := ownGitRepository(checkout.Dir); !own {
		entry.Reasons = append(entry.Reasons, StateNoGit+": "+why+"; changes cannot be detected, treated as local")
	} else {
		if out, err := git(checkout.Dir, "status", "--porcelain"); err != nil {
			entry.Reasons = append(entry.Reasons, StateDirty+": git status failed: "+err.Error())
		} else if out != "" {
			entry.Reasons = append(entry.Reasons, StateDirty+": working tree has changes")
		}
		out, err := git(checkout.Dir, "tag", "--points-at", "HEAD")
		if err != nil || !containsLine(out, tag) {
			entry.Reasons = append(entry.Reasons, StateUntagged+": HEAD is not at tag "+tag)
		}
	}
	publication, why := registry.Published(ctx, checkout.Manifest.Module.Path, version)
	switch publication {
	case PublicationMissing:
		entry.Reasons = append(entry.Reasons, StateUnpublished+": "+tag+" is not in the registry")
	case PublicationUnknown:
		entry.Note = why
	}
	switch {
	case len(entry.Reasons) > 0:
		entry.State, _, _ = strings.Cut(entry.Reasons[0], ":")
	case pinned:
		entry.State = StatePinned
	case publication == PublicationPublished:
		entry.State = StatePublished
	default:
		entry.State = StateUnknown
	}
	entry.Local = all || pinned || len(entry.Reasons) > 0
	return entry
}

// ownGitRepository reports whether dir is the top of its own git work tree,
// so that a checkout inside a parent repository is not judged by the parent.
// TRLC-LINKS: REQ-EMG-068
func ownGitRepository(dir string) (bool, string) {
	top, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return false, "not a git repository"
	}
	want, err1 := filepath.EvalSymlinks(dir)
	got, err2 := filepath.EvalSymlinks(top)
	if err1 != nil || err2 != nil || want != got {
		return false, "not the top of its own git repository (" + top + ")"
	}
	return true, ""
}

// git runs a read-only git command in dir without taking optional locks.
// TRLC-LINKS: REQ-EMG-068
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// containsLine reports whether text has a line equal to want.
// TRLC-LINKS: REQ-EMG-068
func containsLine(text, want string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

// Local returns the local entries.
// TRLC-LINKS: REQ-EMG-068
func (p Plan) Local() []Entry {
	var out []Entry
	for _, entry := range p.Entries {
		if entry.Local {
			out = append(out, entry)
		}
	}
	return out
}

// Skew is a consumer pin that a local replacement overrides with another version.
type Skew struct {
	Consumer, Alias, Module, Pinned, LocalCheckout, LocalVersion string
}

// String is the SKEW line: consumer, alias, pinned, local, then context. A Go
// require skew has alias go:<module path>.
// TRLC-LINKS: REQ-EMG-068
func (s Skew) String() string {
	if strings.HasPrefix(s.Alias, "go:") {
		return fmt.Sprintf("SKEW %s %s %s %s (go.mod requires %s; go.work uses ./%s at %s)",
			s.Consumer, s.Alias, s.Pinned, s.LocalVersion, s.Module, s.LocalCheckout, s.LocalVersion)
	}
	return fmt.Sprintf("SKEW %s %s %s %s (%s is replaced by local ./%s at %s)",
		s.Consumer, s.Alias, s.Pinned, s.LocalVersion, s.Module, s.LocalCheckout, s.LocalVersion)
}

// Skews lists every dependency pin of a checkout whose module is local at a
// different version (the replacement overrides every pin of its module), and
// every go.mod require of a workspace checkout's Go module at another version
// than that checkout's version tag (go.work uses every Go checkout).
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func (p Plan) Skews() []Skew {
	local := map[string]Entry{}
	for _, entry := range p.Local() {
		local[entry.Manifest.Module.Path] = entry
	}
	var out []Skew
	for _, entry := range p.Entries {
		for _, dep := range entry.Manifest.Dependencies {
			l, ok := local[strings.TrimSpace(dep.Path)]
			if !ok || tagFor(dep.Version) == tagFor(l.Manifest.Module.Version) {
				continue
			}
			out = append(out, Skew{
				Consumer: entry.Name, Alias: dep.Alias, Module: dep.Path, Pinned: dep.Version,
				LocalCheckout: l.Name, LocalVersion: l.Manifest.Module.Version,
			})
		}
	}
	byGoModule := map[string]Entry{}
	for _, entry := range p.Entries {
		if entry.GoModule != "" {
			byGoModule[entry.GoModule] = entry
		}
	}
	for _, entry := range p.Entries {
		paths := make([]string, 0, len(entry.GoRequires))
		for path := range entry.GoRequires {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			c, ok := byGoModule[path]
			if !ok || c.Name == entry.Name || entry.GoRequires[path] == tagFor(c.Manifest.Module.Version) {
				continue
			}
			out = append(out, Skew{
				Consumer: entry.Name, Alias: "go:" + path, Module: path, Pinned: entry.GoRequires[path],
				LocalCheckout: c.Name, LocalVersion: tagFor(c.Manifest.Module.Version),
			})
		}
	}
	return out
}
