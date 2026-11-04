// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// The repo argument of the model-reading tools: one server answers for
// every checkout of a model family. The workspace is resolved on every call
// (a server started before engws sync wrote engmod.work.yml sees it as soon
// as it exists), with the trust check of the upward search, and a repo is
// confined to the workspace root.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// familyRoot returns the root a repo argument may name: the directory of
// the engmod.work.yml that model.FindWorkspaceWith selects for the server's
// model now (the upward search refuses a world-writable file or one owned by
// another user; ENGMOD_WORK=<path> names one explicitly), else the server's
// own model directory. With the workspace off it is the model directory.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func (s *Server) familyRoot() (string, error) {
	if s.bundle == nil || s.bundle.ManifestPath == "" {
		return "", fmt.Errorf("repo needs a server initialized with an engmod.yml modelPath")
	}
	own := filepath.Dir(s.bundle.ManifestPath)
	_, workspace, err := model.FindWorkspaceWith(own, "")
	if err != nil {
		return "", fmt.Errorf("repo: %w", err)
	}
	if workspace == "" {
		return own, nil
	}
	return filepath.Dir(workspace), nil
}

// resolveRepo returns the model directory a repo argument names: relative
// to the family root (or absolute), holding engmod.yml, inside the root
// after symlinks.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (s *Server) resolveRepo(repo string) (string, error) {
	repo = strings.TrimSpace(repo)
	root, err := s.familyRoot()
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("repo: resolve workspace root: %w", err)
	}
	dir := repo
	switch {
	case repo == "" || repo == ".":
		dir = filepath.Dir(s.bundle.ManifestPath)
	case !filepath.IsAbs(repo):
		dir = filepath.Join(root, filepath.FromSlash(repo))
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", fmt.Errorf("repo %q: %w", repo, err)
	}
	if rel, err := filepath.Rel(realRoot, real); err != nil || (rel != "." && !filepath.IsLocal(rel)) {
		return "", fmt.Errorf("repo %q lies outside the workspace root (the directory of engmod.work.yml above this model, or the model itself without one)", repo)
	}
	if info, err := os.Stat(filepath.Join(real, "engmod.yml")); err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("repo %q has no engmod.yml", repo)
	}
	return real, nil
}

// memberServer returns the server that answers for repo: this one for its
// own checkout, else a server initialized on the member's engmod.yml
// (cached per directory; it reloads its model per call like this one).
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (s *Server) memberServer(repo string) (*Server, error) {
	dir, err := s.resolveRepo(repo)
	if err != nil {
		return nil, err
	}
	if own, err := filepath.EvalSymlinks(filepath.Dir(s.bundle.ManifestPath)); err == nil && own == dir {
		return s, nil
	}
	if member, ok := s.members[dir]; ok {
		return member, nil
	}
	member := NewServer()
	member.clock = s.clock
	member.modelPath = filepath.Join(dir, "engmod.yml")
	member.repoRoot = dir
	if err := member.loadContext(nil); err != nil {
		return nil, fmt.Errorf("repo %q: %w", repo, err)
	}
	if s.members == nil {
		s.members = map[string]*Server{}
	}
	s.members[dir] = member
	return member, nil
}
