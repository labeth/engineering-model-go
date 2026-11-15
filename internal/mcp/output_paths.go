// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
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

// scrubString replaces every absolute-path prefix in s. A prefix matches only
// as a whole path: at the start of s or after a character that cannot be part
// of a path, and followed by a separator or a character that cannot continue
// a path element, so /w/proj never rewrites /w/proj-b and a URL path such as
// https://h/w/proj/x is left alone.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (p pathScrubber) scrubString(s string) string {
	for _, prefix := range p.prefixes {
		s = replacePathPrefix(s, prefix)
	}
	return s
}

// TRLC-LINKS: REQ-EMG-069
func replacePathPrefix(s string, prefix scrubPrefix) string {
	from := prefix.from
	if !strings.Contains(s, from) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], from)
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		at := i + j
		end := at + len(from)
		if !wholePath(s, at, end) {
			b.WriteString(s[i : at+1])
			i = at + 1
			continue
		}
		b.WriteString(s[i:at])
		repl, dropSep := pathReplacement(prefix.to, end < len(s) && s[end] == filepath.Separator)
		b.WriteString(repl)
		if dropSep {
			end++
		}
		i = end
	}
	return b.String()
}

// wholePath reports whether the prefix found at s[at:end] is a whole path.
// TRLC-LINKS: REQ-EMG-069
func wholePath(s string, at, end int) bool {
	return pathStartBoundary(s, at) && pathEndBoundary(s, end)
}

// pathStartBoundary reports whether a prefix found at index at begins a
// path: at the start of s or after a character that cannot be part of a
// path. "file://" + path is a path, not a URL host or path segment.
// TRLC-LINKS: REQ-EMG-069
func pathStartBoundary(s string, at int) bool {
	return at == 0 || !pathChar(s[at-1]) || strings.HasSuffix(s[:at], "file://")
}

// pathEndBoundary reports whether a prefix ending before index end is a
// whole path: at the end of s, before a separator or a character that
// cannot continue a path element, or before a '.' that ends a sentence
// (followed by the end or a non-path character).
// TRLC-LINKS: REQ-EMG-069
func pathEndBoundary(s string, end int) bool {
	if end == len(s) {
		return true
	}
	next := s[end]
	if next == filepath.Separator || !pathChar(next) {
		return true
	}
	return next == '.' && (end+1 == len(s) || !pathChar(s[end+1]))
}

// pathReplacement returns the replacement of a matched prefix and whether
// the separator that follows it is consumed too. An empty replacement drops a
// following separator (project-relative) or becomes "."; a replacement ending
// in ":" also takes the separator.
// TRLC-LINKS: REQ-EMG-069
func pathReplacement(to string, atSep bool) (string, bool) {
	switch {
	case to == "" && atSep:
		return "", true // project-relative: drop the separator too
	case to == "":
		return ".", false
	case strings.HasSuffix(to, ":") && atSep:
		return to, true
	}
	return to, false
}

// pathChar reports whether c can appear inside a path element, so a prefix
// next to it is part of a longer name (or of a URL host or path).
// TRLC-LINKS: REQ-EMG-069
func pathChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return strings.IndexByte("._-+@~%/", c) >= 0
}

// scrub rewrites strings anywhere in a JSON-like value.
// TRLC-LINKS: REQ-EMG-069
func (p pathScrubber) scrub(value any) any {
	switch v := value.(type) {
	case string:
		return p.scrubString(v)
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[p.scrubString(key)] = p.scrub(item)
		}
		return out
	case []any:
		for i, item := range v {
			v[i] = p.scrub(item)
		}
		return v
	default:
		return value
	}
}
