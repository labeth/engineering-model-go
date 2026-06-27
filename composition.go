// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

// System-of-systems composition. A system model may reference subsystem models in
// local subdirectories; this resolves the parent->child DAG, enforces the
// workspace boundary and acyclicity, materializes parent->subsystem allocation
// traceability (without modifying any subsystem), and validates the bindings.
// References are downward-only: a subsystem never knows its parents.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// ComposedSystem is a resolved system together with its resolved subsystems.
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, DO-ARCHITECTURE-MODEL
type ComposedSystem struct {
	SubsystemID  string // id of this system within its parent (empty for the root)
	Dir          string // resolved directory of this system's model
	Bundle       model.Bundle
	Requirements model.RequirementsDocument // this system's own requirements (for delegation target resolution)
	Children     []*ComposedSystem
}

// MaterializedAllocation is a parent->subsystem allocation with its composed-view resolution status.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type MaterializedAllocation struct {
	System            string // owning (parent) system id
	Requirement       string
	Subsystem         string
	Target            string // public id within the subsystem
	TargetRef         string // the subsystem requirement that realizes the target (its contract ref)
	Resolved          bool   // target is published in the subsystem contract
	TargetRefResolved bool   // TargetRef names a requirement that exists in the subsystem
	Note              string
}

// CompositionResult is the resolved tree plus materialized allocation trace and diagnostics.
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
type CompositionResult struct {
	Root        *ComposedSystem
	Allocations []MaterializedAllocation
	Diagnostics []validate.Diagnostic
}

// HasComposition reports whether the model declares any subsystems.
// TRLC-LINKS: REQ-EMG-016
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func HasComposition(bundle model.Bundle) bool {
	return len(bundle.Architecture.Composition.Subsystems) > 0
}

// GenerateCompositionFromFile resolves the system-of-systems rooted at architecturePath.
// TRLC-LINKS: REQ-EMG-016
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateCompositionFromFile(architecturePath string) (CompositionResult, error) {
	absTop, err := filepath.Abs(architecturePath)
	if err != nil {
		return CompositionResult{}, err
	}
	workspace := filepath.Dir(absTop)
	ancestry := map[string]bool{}
	root, diags := resolveSystem("", absTop, workspace, ancestry)
	res := CompositionResult{Root: root, Diagnostics: diags}
	if root != nil {
		res.Allocations = materializeAllocations(root)
		res.Diagnostics = append(res.Diagnostics, validateComposition(root)...)
	}
	return res, nil
}

// resolveSystem loads the model and recursively resolves its subsystems, detecting
// cycles via the ancestry stack and rejecting references outside the workspace.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-017, REQ-EMG-018
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func resolveSystem(subsystemID, architectureAbsPath, workspace string, ancestry map[string]bool) (*ComposedSystem, []validate.Diagnostic) {
	var diags []validate.Diagnostic
	if ancestry[architectureAbsPath] {
		return nil, []validate.Diagnostic{{
			Code: "composition.cycle", Severity: validate.SeverityError,
			Message: fmt.Sprintf("subsystem reference cycle detected at %s", architectureAbsPath),
			Path:    architectureAbsPath,
		}}
	}
	ancestry[architectureAbsPath] = true
	defer delete(ancestry, architectureAbsPath)

	bundle, err := model.LoadBundle(architectureAbsPath)
	if err != nil {
		return nil, []validate.Diagnostic{{
			Code: "composition.load_failed", Severity: validate.SeverityError,
			Message: err.Error(), Path: architectureAbsPath,
		}}
	}
	sys := &ComposedSystem{SubsystemID: subsystemID, Dir: filepath.Dir(architectureAbsPath), Bundle: bundle}
	if reqs, rerr := model.LoadRequirements(filepath.Join(sys.Dir, "requirements.yml")); rerr == nil {
		sys.Requirements = reqs
	}

	for _, sub := range bundle.Architecture.Composition.Subsystems {
		childArch, rdiags := resolveRef(sys.Dir, sub, workspace)
		diags = append(diags, rdiags...)
		if childArch == "" {
			continue
		}
		child, cdiags := resolveSystem(sub.ID, childArch, workspace, ancestry)
		diags = append(diags, cdiags...)
		if child != nil {
			sys.Children = append(sys.Children, child)
		}
	}
	return sys, diags
}

// resolveRef resolves a subsystem reference to a child architecture file path.
// Phase 1 supports local subdirectory paths only; the resolved path must stay
// within the workspace boundary.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func resolveRef(baseDir string, sub model.Subsystem, workspace string) (string, []validate.Diagnostic) {
	at := func(code, msg string) []validate.Diagnostic {
		return []validate.Diagnostic{{Code: code, Severity: validate.SeverityError, Message: msg, Path: baseDir}}
	}
	if strings.TrimSpace(sub.Git) != "" {
		return resolveGitRef(sub, workspace, at)
	}
	ref := strings.TrimSpace(sub.Ref)
	if ref == "" {
		return "", at("composition.invalid_ref", fmt.Sprintf("subsystem %q has no ref or git url", sub.ID))
	}
	if strings.Contains(ref, "://") || filepath.IsAbs(ref) {
		return "", at("composition.unsupported_ref", fmt.Sprintf("subsystem %q ref %q is not a local subdirectory path (use git: for an external repository)", sub.ID, ref))
	}
	abs := filepath.Clean(filepath.Join(baseDir, ref))
	return resolveChildArchitecture(abs, workspace, sub.ID, at)
}

// resolveGitRef clones (or reuses) an external git subsystem into the workspace
// .engmod cache and returns the path to its architecture file. The clone lives
// inside the workspace, so the cross-repository view is materialized like a local
// subsystem. The cache is treated as temporary and is reused when it already points
// at the requested repository.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-027
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER, CTRL-MCP-PATH-BOUNDARY
func resolveGitRef(sub model.Subsystem, workspace string, at func(string, string) []validate.Diagnostic) (string, []validate.Diagnostic) {
	url := strings.TrimSpace(sub.Git)
	rev := strings.TrimSpace(sub.Rev)
	dest := filepath.Join(workspace, ".engmod", "subsystems", sanitizeDirName(sub.ID))
	gitDir := filepath.Join(dest, ".git")

	// Reuse an existing clone only if it points at the requested repository.
	if _, err := os.Stat(gitDir); err == nil {
		if cur, _ := runGit(dest, "remote", "get-url", "origin"); strings.TrimSpace(cur) != url {
			_ = os.RemoveAll(dest)
		}
	}
	if _, err := os.Stat(gitDir); err != nil {
		if mkErr := os.MkdirAll(filepath.Dir(dest), 0o755); mkErr != nil {
			return "", at("composition.clone_failed", fmt.Sprintf("subsystem %q: %v", sub.ID, mkErr))
		}
		args := []string{"clone", "--quiet"}
		if rev == "" {
			args = append(args, "--depth", "1")
		}
		args = append(args, url, dest)
		if out, cErr := runGit("", args...); cErr != nil {
			return "", at("composition.clone_failed", fmt.Sprintf("subsystem %q git clone of %q failed: %v: %s", sub.ID, url, cErr, strings.TrimSpace(out)))
		}
	}
	if rev != "" {
		if out, cErr := runGit(dest, "checkout", "--quiet", rev); cErr != nil {
			return "", at("composition.clone_failed", fmt.Sprintf("subsystem %q git checkout %q failed: %v: %s", sub.ID, rev, cErr, strings.TrimSpace(out)))
		}
	}
	abs := dest
	if p := strings.TrimSpace(sub.Path); p != "" {
		abs = filepath.Clean(filepath.Join(dest, p))
	}
	return resolveChildArchitecture(abs, workspace, sub.ID, at)
}

// resolveChildArchitecture validates a resolved model path against the workspace
// boundary and returns the architecture.yml within it.
// TRLC-LINKS: REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY
func resolveChildArchitecture(abs, workspace, subID string, at func(string, string) []validate.Diagnostic) (string, []validate.Diagnostic) {
	rel, err := filepath.Rel(workspace, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", at("composition.out_of_workspace", fmt.Sprintf("subsystem %q resolves outside the workspace boundary", subID))
	}
	childArch := abs
	if info, statErr := os.Stat(abs); statErr == nil && info.IsDir() {
		childArch = filepath.Join(abs, "architecture.yml")
	}
	if _, statErr := os.Stat(childArch); statErr != nil {
		return "", at("composition.missing_ref", fmt.Sprintf("subsystem %q has no architecture.yml at %s", subID, childArch))
	}
	return childArch, nil
}

// runGit runs a git subcommand, returning combined output and any error.
// TRLC-LINKS: REQ-EMG-027
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// sanitizeDirName reduces an id to a safe directory name for the .engmod cache.
// TRLC-LINKS: REQ-EMG-027
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func sanitizeDirName(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "subsystem"
	}
	return b.String()
}

// materializeAllocations walks the tree and produces the parent->subsystem allocation
// trace, resolving each target against the referenced subsystem's published contract.
// The subsystem models are never modified.
// TRLC-LINKS: REQ-EMG-020, REQ-EMG-025
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func materializeAllocations(sys *ComposedSystem) []MaterializedAllocation {
	var out []MaterializedAllocation
	childByID := map[string]*ComposedSystem{}
	for _, c := range sys.Children {
		childByID[c.SubsystemID] = c
	}
	systemID := strings.TrimSpace(sys.Bundle.Architecture.Model.ID)
	for _, a := range sys.Bundle.Architecture.Composition.Allocations {
		ma := MaterializedAllocation{System: systemID, Requirement: a.Requirement, Subsystem: a.To, Target: a.Target}
		if child, ok := childByID[a.To]; ok {
			entry, published := contractProvidedEntry(child.Bundle, a.Target)
			ma.Resolved = published
			if !published {
				ma.Note = "target not published in subsystem contract"
			} else {
				ma.TargetRef = strings.TrimSpace(entry.Ref)
				if ma.TargetRef != "" {
					ma.TargetRefResolved = requirementExists(child.Requirements, ma.TargetRef)
				} else {
					ma.Note = "no contract ref: delegation does not resolve to a specific requirement"
				}
			}
		} else if hardwareItem(sys.Bundle, a.To) {
			ma.Resolved = true
			ma.Note = "allocated to hardware item"
		} else {
			ma.Note = "unknown allocation target"
		}
		out = append(out, ma)
	}
	for _, c := range sys.Children {
		out = append(out, materializeAllocations(c)...)
	}
	return out
}

// validateComposition checks bindings across the composed tree: allocation targets must
// be public, every subsystem required interface must be satisfied, and allocations must
// reference real requirements and subsystems.
// TRLC-LINKS: REQ-EMG-019, REQ-EMG-021, REQ-EMG-022
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func validateComposition(sys *ComposedSystem) []validate.Diagnostic {
	var diags []validate.Diagnostic
	childByID := map[string]*ComposedSystem{}
	for _, c := range sys.Children {
		childByID[c.SubsystemID] = c
	}
	owner := strings.TrimSpace(sys.Bundle.Architecture.Model.ID)
	loc := func() string { return owner }

	// allocations: target must be a public provided id of a real subsystem (or a hardware item).
	for _, a := range sys.Bundle.Architecture.Composition.Allocations {
		child, ok := childByID[a.To]
		if !ok {
			if !hardwareItem(sys.Bundle, a.To) {
				diags = append(diags, errDiag("composition.unknown_subsystem", fmt.Sprintf("allocation in %s targets unknown subsystem %q", owner, a.To), loc()))
			}
			continue
		}
		entry, published := contractProvidedEntry(child.Bundle, a.Target)
		if !published {
			diags = append(diags, errDiag("composition.allocation_to_internal", fmt.Sprintf("allocation in %s targets %q which subsystem %q does not publish in its contract", owner, a.Target, a.To), loc()))
			continue
		}
		ref := strings.TrimSpace(entry.Ref)
		if ref == "" {
			diags = append(diags, validate.Diagnostic{
				Code: "composition.untraceable_delegation", Severity: validate.SeverityWarning,
				Message: fmt.Sprintf("allocation in %s to %s/%s does not resolve to a specific requirement; declare a contract ref on %s naming the subsystem requirement that satisfies it", owner, a.To, a.Target, a.Target),
				Path:    loc(),
			})
		} else if !requirementExists(child.Requirements, ref) {
			diags = append(diags, errDiag("composition.unknown_target_requirement", fmt.Sprintf("allocation in %s to %s/%s names requirement %q, which subsystem %q does not define", owner, a.To, a.Target, ref, a.To), loc()))
		}
	}

	// satisfactions: every subsystem required interface must be satisfied by the parent.
	satisfied := map[string]bool{}
	for _, s := range sys.Bundle.Architecture.Composition.Satisfactions {
		satisfied[strings.TrimSpace(s.Need)] = true
	}
	for _, c := range sys.Children {
		for _, req := range c.Bundle.Architecture.Contract.Requires {
			need := c.SubsystemID + "/" + req.ID
			if !satisfied[need] {
				diags = append(diags, validate.Diagnostic{
					Code: "composition.unsatisfied_require", Severity: validate.SeverityWarning,
					Message: fmt.Sprintf("subsystem %q required interface %q is not satisfied by %s", c.SubsystemID, req.ID, owner),
					Path:    owner,
				})
			}
		}
	}

	for _, c := range sys.Children {
		diags = append(diags, validateComposition(c)...)
	}
	return diags
}

// contractProvidedEntry returns the bundle's provided contract entry with the given id.
// TRLC-LINKS: REQ-EMG-022
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func contractProvidedEntry(bundle model.Bundle, id string) (model.ContractEntry, bool) {
	id = strings.TrimSpace(id)
	for _, p := range bundle.Architecture.Contract.Provides {
		if strings.TrimSpace(p.ID) == id {
			return p, true
		}
	}
	return model.ContractEntry{}, false
}

// contractProvides reports whether a bundle's contract publishes the given id.
// TRLC-LINKS: REQ-EMG-022
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func contractProvides(bundle model.Bundle, id string) bool {
	_, ok := contractProvidedEntry(bundle, id)
	return ok
}

// requirementExists reports whether a requirements document defines the given id.
// TRLC-LINKS: REQ-EMG-022, REQ-EMG-026
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func requirementExists(reqs model.RequirementsDocument, id string) bool {
	id = strings.TrimSpace(id)
	for _, r := range reqs.Requirements {
		if strings.TrimSpace(r.ID) == id {
			return true
		}
	}
	return false
}

// hardwareItem reports whether the bundle declares a hardware item with the given id.
// TRLC-LINKS: REQ-EMG-024
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func hardwareItem(bundle model.Bundle, id string) bool {
	for _, h := range bundle.Architecture.AuthoredArchitecture.HardwareItems {
		if h.ID == id {
			return true
		}
	}
	return false
}

// errDiag builds an error-severity diagnostic.
// TRLC-LINKS: REQ-EMG-019
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func errDiag(code, msg, path string) validate.Diagnostic {
	return validate.Diagnostic{Code: code, Severity: validate.SeverityError, Message: msg, Path: path}
}
