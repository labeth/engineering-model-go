// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-035, REQ-EMG-036, REQ-EMG-044
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) loadContext(params any) error {
	s.applyInitOptions(params)
	loadedBundle, err := s.loadModelBundle()
	if err != nil {
		return err
	}
	if !loadedBundle {
		if err := s.loadStandaloneDocuments(); err != nil {
			return err
		}
	}
	if err := s.mergeBundleDocuments(); err != nil {
		return err
	}
	if s.repoRoot != "" {
		s.repoRoot = absPathOrSelf(s.repoRoot)
	}
	// reset file index if context changed
	s.indexOnce = sync.Once{}
	s.repoFiles = nil
	s.indexErr = nil
	s.fingerprint = s.modelFileFingerprint()
	return nil
}

// absPathOrSelf is the absolute form of the path, or the path itself when it
// cannot be made absolute.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-035, REQ-EMG-036, REQ-EMG-044
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func absPathOrSelf(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// existingPathOr is current when set, else the candidate when it exists on
// disk, else current.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-035, REQ-EMG-036, REQ-EMG-044
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func existingPathOr(current, candidate string) string {
	if current != "" {
		return current
	}
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return current
}

// applyInitOptions takes the paths of the initializationOptions of an
// initialize request over the current ones.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-035, REQ-EMG-036, REQ-EMG-044
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) applyInitOptions(params any) {
	p, _ := params.(map[string]any)
	init, _ := p["initializationOptions"].(map[string]any)
	if init == nil {
		init = map[string]any{}
	}
	s.modelPath = nonEmptyString(init["modelPath"], s.modelPath)
	s.requirementsPath = nonEmptyString(init["requirementsPath"], s.requirementsPath)
	s.designPath = nonEmptyString(init["designPath"], s.designPath)
	s.repoRoot = nonEmptyString(init["repoRoot"], s.repoRoot)
}

// loadModelBundle loads the canonical bundle of modelPath and defaults the
// repo root and the requirements and design paths from it; it reports
// whether a bundle was loaded.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-035, REQ-EMG-036, REQ-EMG-044
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) loadModelBundle() (bool, error) {
	if s.modelPath == "" {
		return false, nil
	}
	s.modelPath = absPathOrSelf(s.modelPath)
	canonical, err := model.LoadCanonicalBundle(s.modelPath)
	if err != nil {
		return false, fmt.Errorf("load canonical model bundle: %w", err)
	}
	b := canonical.Documents()
	s.canonical = &canonical
	s.bundle = &b
	s.requirements = &b.Requirements
	s.design = &b.Design
	if s.repoRoot == "" {
		s.repoRoot = filepath.Dir(filepath.Dir(b.ArchitecturePath))
	}
	s.requirementsPath = existingPathOr(s.requirementsPath, b.RequirementsPath)
	s.designPath = existingPathOr(s.designPath, b.DesignPath)
	return true, nil
}

// loadStandaloneDocuments loads the requirements and design documents named
// by their own paths when no model bundle was loaded.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-035, REQ-EMG-036, REQ-EMG-044
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) loadStandaloneDocuments() error {
	if err := s.loadStandaloneRequirements(); err != nil {
		return err
	}
	return s.loadStandaloneDesign()
}

// loadStandaloneRequirements loads the requirements document of
// requirementsPath, when set.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-035, REQ-EMG-036, REQ-EMG-044
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) loadStandaloneRequirements() error {
	if s.requirementsPath == "" {
		return nil
	}
	s.requirementsPath = absPathOrSelf(s.requirementsPath)
	r, err := model.LoadRequirements(s.requirementsPath)
	if err != nil {
		return fmt.Errorf("load requirements: %w", err)
	}
	s.requirements = &r
	return nil
}

// loadStandaloneDesign loads the design document of designPath, when set.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-035, REQ-EMG-036, REQ-EMG-044
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) loadStandaloneDesign() error {
	if s.designPath == "" {
		return nil
	}
	s.designPath = absPathOrSelf(s.designPath)
	d, err := model.LoadDesign(s.designPath)
	if err != nil {
		return fmt.Errorf("load design: %w", err)
	}
	s.design = &d
	return nil
}

// mergeBundleDocuments projects the bundle again with the loaded
// requirements and design documents.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-035, REQ-EMG-036, REQ-EMG-044
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) mergeBundleDocuments() error {
	if s.bundle == nil {
		return nil
	}
	merged := *s.bundle
	if s.requirements != nil {
		merged.Requirements = *s.requirements
	}
	if s.design != nil {
		merged.Design = *s.design
	}
	canonical, err := model.NewCanonicalBundle(merged)
	if err != nil {
		return fmt.Errorf("project canonical MCP model: %w", err)
	}
	documents := canonical.Documents()
	s.canonical = &canonical
	s.bundle = &documents
	return nil
}

// fingerprintPaths lists the files of this module's model the fingerprint
// covers.
// TRLC-LINKS: REQ-EMG-089
// ENGMODEL-LINKS: FU-MCP-SERVER
// ENGMODEL-LINKS: TR-MCP-MODEL-FILE-CHANGES, TR-MCP-FILES-LOADABILITY-CHANGES
func (s *Server) fingerprintPaths() []string {
	paths := []string{s.modelPath, s.requirementsPath, s.designPath}
	if s.bundle != nil {
		b := s.bundle
		paths = append(paths, b.ManifestPath, b.ArchitecturePath, b.BehaviorPath, b.AssurancePath, b.CompliancePath,
			b.ViewsPath, b.CatalogPath, b.DecisionsPath, b.RequirementsPath, b.DesignPath, b.AviationPath, b.ReqLinksPath)
	}
	if s.modelPath != "" {
		dir := filepath.Dir(s.modelPath)
		paths = append(paths, filepath.Join(dir, "engmod.lock.yml"), filepath.Join(dir, "engmod.lock.local.yml"), filepath.Join(dir, model.WorkspaceFileName))
	}
	return paths
}

// hashModelFile feeds one file into the fingerprint: absent for a missing or
// unreadable file, the mode for a non-regular one, size and modification
// time for one above maxFingerprintFileBytes, else size and content.
// TRLC-LINKS: REQ-EMG-089
// ENGMODEL-LINKS: FU-MCP-SERVER
// ENGMODEL-LINKS: TR-MCP-MODEL-FILE-CHANGES, TR-MCP-FILES-LOADABILITY-CHANGES
func hashModelFile(h io.Writer, p string) {
	info, err := os.Stat(p)
	switch {
	case err != nil:
		fmt.Fprintf(h, "%s\x00absent\x00", p)
		return
	case !info.Mode().IsRegular():
		fmt.Fprintf(h, "%s\x00mode %v\x00", p, info.Mode())
		return
	case info.Size() > maxFingerprintFileBytes:
		fmt.Fprintf(h, "%s\x00large %d %d\x00", p, info.Size(), info.ModTime().UnixNano())
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		fmt.Fprintf(h, "%s\x00absent\x00", p)
		return
	}
	fmt.Fprintf(h, "%s\x00%d\x00", p, len(data))
	h.Write(data)
}

// modelFileFingerprint hashes the path and content of every file of this
// module's model: the manifest, each document it names, the requirements and
// design paths, and the lock and workspace files next to the manifest. It
// does not cover the documents of dependencies resolved from sibling
// checkouts or the module cache; an edit there needs a new initialize. A
// missing file hashes as absent, so creating or deleting one also counts.
// Only regular files up to maxFingerprintFileBytes are read; a larger one
// hashes by size and modification time and a non-regular one (a FIFO or
// device) by its mode, so the fingerprint never blocks or reads unbounded
// input.
// TRLC-LINKS: REQ-EMG-089
// ENGMODEL-LINKS: FU-MCP-SERVER
// ENGMODEL-LINKS: TR-MCP-MODEL-FILE-CHANGES, TR-MCP-FILES-LOADABILITY-CHANGES
func (s *Server) modelFileFingerprint() string {
	seen := map[string]bool{}
	h := sha256.New()
	for _, p := range s.fingerprintPaths() {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		hashModelFile(h, p)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
