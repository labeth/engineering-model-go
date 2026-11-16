// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// model.diagnose: validation of a private copy of the model with one document
// replaced, and attribution of diagnostics to documents and YAML positions.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	goyaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// diagnoseWithContent copies the manifest and its documents into a private
// temporary directory, replaces one document, loads and validates the model
// with the same pipeline as the model check, and removes the copy. The loaded
// bundle is re-rooted at the real model directory so code, trace and
// composition analysis see the real project.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func diagnoseWithContent(req diagnoseRequest) ([]validate.Diagnostic, error) {
	tmp, err := os.MkdirTemp("", "engmod-diagnose-")
	if err != nil {
		return nil, fmt.Errorf("create temporary model copy: %w", err)
	}
	defer os.RemoveAll(tmp)
	if err := writeDiagnoseCopy(tmp, req); err != nil {
		return nil, err
	}
	scratch := scratchDir(tmp)
	bundle, err := model.LoadBundle(filepath.Join(tmp, "engmod.yml"))
	if err != nil {
		return loadFailureDiagnostics(err, scratch, req), nil
	}
	rebaseBundle(&bundle, pathRebaser{tmp: tmp, root: req.root})
	diags := bundleDiagnostics(bundle)
	for i := range diags {
		diags[i].Message = scratch.scrub(diags[i].Message)
		diags[i].Path = scratch.scrub(diags[i].Path)
	}
	return validate.SortDiagnostics(diags), nil
}

// scratchDir is the private temporary copy of the model.
type scratchDir string

// scrub removes the temporary directory from a message or path.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func (t scratchDir) scrub(msg string) string {
	tmp := string(t)
	return strings.ReplaceAll(strings.ReplaceAll(msg, tmp+string(filepath.Separator), ""), tmp, ".")
}

// writeCopyFile writes one file below the private copy.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func writeCopyFile(tmp, rel string, data []byte) error {
	path := filepath.Join(tmp, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// writeDiagnoseCopy writes the manifest and every document into the private
// copy, the replaced document with the submitted content.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func writeDiagnoseCopy(tmp string, req diagnoseRequest) error {
	if err := writeCopyFile(tmp, "engmod.yml", req.manifestBytes); err != nil {
		return fmt.Errorf("copy manifest: %w", err)
	}
	for _, rel := range req.documents {
		data, err := copyDocumentData(req, rel)
		if err != nil {
			return err
		}
		if err := writeCopyFile(tmp, rel, data); err != nil {
			return fmt.Errorf("copy %s: %w", rel, err)
		}
	}
	return nil
}

// copyDocumentData returns the submitted content for the replaced document and
// the file content of the real model for every other one.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func copyDocumentData(req diagnoseRequest, rel string) ([]byte, error) {
	if rel == req.rel {
		return []byte(req.content), nil
	}
	data, err := os.ReadFile(filepath.Join(req.root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rel, err)
	}
	return data, nil
}

// loadFailureDiagnostics turns a load error of the private copy into diagnostics.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func loadFailureDiagnostics(err error, scratch scratchDir, req diagnoseRequest) []validate.Diagnostic {
	var aviation *model.AviationValidationError
	if errors.As(err, &aviation) {
		out := []validate.Diagnostic{}
		for _, d := range aviation.Diagnostics {
			out = append(out, validate.Diagnostic{Code: d.Code, Severity: validate.SeverityError, Message: d.Message, Path: req.documents["aviation"] + "#" + d.Path})
		}
		return out
	}
	msg := scratch.scrub(err.Error())
	return []validate.Diagnostic{{Code: "model.load_failed", Severity: validate.SeverityError, Message: msg, Path: loadErrorPath(msg, req.documents, req.rel)}}
}

// pathRebaser maps paths of the private copy back to the real model root.
type pathRebaser struct {
	tmp, root string
}

// rebase returns p below the real root when it lies in the private copy.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func (r pathRebaser) rebase(p string) string {
	if rel, rerr := filepath.Rel(r.tmp, p); rerr == nil && filepath.IsLocal(rel) {
		return filepath.Join(r.root, rel)
	}
	return p
}

// rebaseBundle re-roots the loaded bundle's manifest and document paths.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func rebaseBundle(bundle *model.Bundle, r pathRebaser) {
	bundle.ManifestPath = r.rebase(bundle.ManifestPath)
	for _, p := range []*string{&bundle.ArchitecturePath, &bundle.BehaviorPath, &bundle.AssurancePath, &bundle.CompliancePath, &bundle.ViewsPath, &bundle.CatalogPath, &bundle.DecisionsPath, &bundle.RequirementsPath, &bundle.DesignPath, &bundle.AviationPath, &bundle.ReqLinksPath} {
		if *p != "" {
			*p = r.rebase(*p)
		}
	}
}

// bundleDiagnostics validates the in-memory bundle: canonical graph
// diagnostics plus the tier-policy validation, or the composition analysis of
// the generated documents.
//
// Dependencies resolve from the real model root (its workspace and module
// cache); the root system is the in-memory bundle with the submitted
// content, so its alias::ID references and composition are checked as
// submitted. Like every MCP read tool it writes no lock and reuses the
// module cache offline (see resolution).
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-VALIDATION-ENGINE, CTRL-MCP-PATH-BOUNDARY
func bundleDiagnostics(bundle model.Bundle) []validate.Diagnostic {
	options := resolution()
	var diags []validate.Diagnostic
	canonical, cerr := model.NewCanonicalBundle(bundle)
	for _, d := range canonical.Diagnostics() {
		diags = append(diags, validate.Diagnostic{Code: d.Code, Severity: validate.Severity(d.Severity), Message: d.Message, Path: d.Path})
	}
	if cerr != nil {
		return append(diags, validate.ApplyTierPolicy(bundle, validate.Bundle(bundle))...)
	}
	result, _ := engmodel.GenerateAsciiDoc(engmodel.AsciiDocInputs{Bundle: bundle, Requirements: bundle.Requirements, Design: bundle.Design}, engmodel.AsciiDocOptions{Resolution: options})
	return append(diags, result.Diagnostics...)
}

// loadErrorPath attributes a load error to the document its message names.
// TRLC-LINKS: REQ-EMG-069
func loadErrorPath(msg string, documents map[string]string, target string) string {
	for _, rel := range documents {
		if rel != target && strings.Contains(msg, rel) && !strings.Contains(msg, target) {
			return rel
		}
	}
	return target
}

// diagnosticDocument returns the manifest document key a diagnostic path
// belongs to, or "" when it cannot be attributed (code, canonical graph).
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func diagnosticDocument(path, root string, documents map[string]string) string {
	if document, _, _ := splitDiagnosticPath(path); document != "" {
		return document
	}
	file := diagnosticFilePath(path)
	if file == "" {
		return ""
	}
	return documentForFile(path, relativeToRoot(file, root), documents)
}

// relativeToRoot returns file as a clean slash path relative to root when it
// is absolute and inside it.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func relativeToRoot(file, root string) string {
	if filepath.IsAbs(file) {
		if rel, err := filepath.Rel(root, file); err == nil {
			file = rel
		}
	}
	return filepath.ToSlash(filepath.Clean(file))
}

// documentForFile returns the manifest key of the document file or the
// "<document>#" prefix of path names.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func documentForFile(path, file string, documents map[string]string) string {
	for key, rel := range documents {
		if rel == file || strings.HasPrefix(path, rel+"#") {
			return key
		}
	}
	return ""
}

// splitDiagnosticPath returns the document key, its YAML root path and the
// remainder (".x[0]" or "[REQ-ID]") of a model diagnostic path.
// TRLC-LINKS: REQ-EMG-069
func splitDiagnosticPath(path string) (string, string, string) {
	path = strings.TrimSpace(path)
	if rest, ok := strings.CutPrefix(path, "authoredArchitecture."); ok {
		return splitAuthoredArchitecturePath(rest)
	}
	for _, p := range diagnosticPathDocuments {
		if rest, ok := strings.CutPrefix(path, p.prefix); ok && atPathBoundary(rest) {
			return p.document, p.yamlRoot, rest
		}
	}
	return "", "", ""
}

// splitAuthoredArchitecturePath splits the part of an authoredArchitecture
// path after the "authoredArchitecture." prefix.
// TRLC-LINKS: REQ-EMG-069
func splitAuthoredArchitecturePath(rest string) (string, string, string) {
	name := rest
	if i := strings.IndexAny(rest, ".["); i >= 0 {
		name = rest[:i]
	}
	target, ok := authoredArchitectureDocuments[name]
	if !ok {
		return "", "", ""
	}
	return target[0], target[1], rest[len(name):]
}

// atPathBoundary reports whether rest is empty or continues a path segment
// with "." or "[".
// TRLC-LINKS: REQ-EMG-069
func atPathBoundary(rest string) bool {
	return rest == "" || strings.IndexByte(".[", rest[0]) >= 0
}

// TRLC-LINKS: REQ-EMG-069
var diagnosticIndexRe = regexp.MustCompile(`\[([^\]]+)\]`)

// diagnosticLocator finds diagnostics in the replaced document's YAML tree.
// requirementIDs maps requirement ID path segments to list indexes.
type diagnosticLocator struct {
	tree           *ast.File
	key            string
	requirementIDs []string
}

// locate locates a diagnostic in the document's YAML and returns its
// line and column, trimming path segments until a node is found.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func (l diagnosticLocator) locate(d validate.Diagnostic) (int, int) {
	if l.tree == nil {
		return 0, 0
	}
	document, yamlRoot, rest := splitDiagnosticPath(d.Path)
	if document != l.key {
		return 0, 0
	}
	if at := strings.Index(rest, "@"); at >= 0 {
		rest = rest[:at]
	}
	rest = diagnosticIndexRe.ReplaceAllStringFunc(rest, l.indexSegment)
	return firstPathPosition(l.tree, "$."+yamlRoot+rest)
}

// indexSegment rewrites a "[REQ-ID]" path segment to its list index; numeric
// segments stay and unknown IDs are dropped.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func (l diagnosticLocator) indexSegment(m string) string {
	inner := m[1 : len(m)-1]
	if _, err := strconv.Atoi(inner); err == nil {
		return m
	}
	for i, id := range l.requirementIDs {
		if id == inner {
			return fmt.Sprintf("[%d]", i)
		}
	}
	return ""
}

// firstPathPosition returns the position of the longest prefix of the YAML
// path that names a node of tree.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func firstPathPosition(tree *ast.File, full string) (int, int) {
	for candidate := full; strings.HasPrefix(candidate, "$."); {
		if line, column, ok := pathPosition(tree, candidate); ok {
			return line, column
		}
		cut := strings.LastIndexAny(candidate, ".[")
		if cut <= 1 {
			break
		}
		candidate = candidate[:cut]
	}
	return 0, 0
}

// pathPosition returns the position of the node a YAML path names in tree.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func pathPosition(tree *ast.File, candidate string) (int, int, bool) {
	p, err := goyaml.PathString(candidate)
	if err != nil {
		return 0, 0, false
	}
	node, err := p.FilterFile(tree)
	if err != nil || node == nil {
		return 0, 0, false
	}
	tok := node.GetToken()
	if tok == nil || tok.Position == nil {
		return 0, 0, false
	}
	return tok.Position.Line, tok.Position.Column, true
}
