// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

// pathScrubber rewrites absolute paths in tool output so nothing about the
// machine layout (home directory, user name, sibling checkouts) reaches the
// MCP client: paths under the project become project-relative, paths under
// the workspace root (multi-repository families) workspace-relative with a
// "workspace:" prefix, and the home directory "~".
type pathScrubber struct {
	prefixes []scrubPrefix
}

type scrubPrefix struct {
	from string
	to   string
}

// newPathScrubber builds the scrubber for a project root and the active
// model workspace, if any.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func newPathScrubber(projectRoot, manifestPath string) pathScrubber {
	var p pathScrubber
	add := func(from, to string) {
		if from == "" || from == string(filepath.Separator) {
			return
		}
		for _, variant := range []string{from, evalOrSame(from)} {
			p.prefixes = append(p.prefixes, scrubPrefix{from: filepath.Clean(variant), to: to})
		}
	}
	if projectRoot != "" {
		add(projectRoot, "")
	}
	if manifestPath != "" {
		if _, workspace, err := model.FindWorkspaceWith(filepath.Dir(manifestPath), ""); err == nil && workspace != "" {
			add(filepath.Dir(workspace), "workspace:")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		add(home, "~")
	}
	// Longest prefix first, so the project wins over the workspace and home.
	sort.SliceStable(p.prefixes, func(i, j int) bool { return len(p.prefixes[i].from) > len(p.prefixes[j].from) })
	return p
}

// TRLC-LINKS: REQ-EMG-069
func evalOrSame(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

// scrubString replaces every absolute-path prefix in s.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (p pathScrubber) scrubString(s string) string {
	for _, prefix := range p.prefixes {
		if !strings.Contains(s, prefix.from) {
			continue
		}
		sep := string(filepath.Separator)
		replacement := prefix.to
		withSep := prefix.from + sep
		if replacement == "" {
			s = strings.ReplaceAll(s, withSep, "")
			s = strings.ReplaceAll(s, prefix.from, ".")
			continue
		}
		if strings.HasSuffix(replacement, ":") {
			s = strings.ReplaceAll(s, withSep, replacement)
		} else {
			s = strings.ReplaceAll(s, withSep, replacement+sep)
		}
		s = strings.ReplaceAll(s, prefix.from, replacement)
	}
	return s
}

// scrub rewrites strings anywhere in a JSON-like value.
// TRLC-LINKS: REQ-EMG-069
func (p pathScrubber) scrub(value any) any {
	switch v := value.(type) {
	case string:
		return p.scrubString(v)
	case map[string]any:
		for key, item := range v {
			v[key] = p.scrub(item)
		}
		return v
	case []any:
		for i, item := range v {
			v[i] = p.scrub(item)
		}
		return v
	default:
		return value
	}
}
