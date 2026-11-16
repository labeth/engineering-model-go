// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// model.patch edits one model entity by stable ID: it applies set, unset,
// append and remove operations to the entity's fields with line-preserving
// YAML edits (internal/yamledit), validates the whole model with the edited
// document exactly like model.diagnose, and returns the diagnostics and a
// unified diff. With write: true it writes the document, and only when the
// result is valid, the document is one the manifest lists and it resolves
// inside the model root (decision ADR-EMG-029). It is the MCP server's one
// writing tool; the requirement link file is not one of its documents (its
// write path is engws review-links, D14).

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	goyaml "github.com/goccy/go-yaml"
	yaml "gopkg.in/yaml.v3"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/yamledit"
)

// patchFieldSegment is one segment of a dotted model.patch field path.
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
var patchFieldSegment = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// patchFieldPattern is the JSON schema pattern of a dotted field path.
const patchFieldPattern = `^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)*$`

// patchInputSchema is the strict input schema of model.patch.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
func patchInputSchema() map[string]any {
	field := map[string]any{"type": "string", "pattern": patchFieldPattern}
	fieldMap := func(description string, value map[string]any) map[string]any {
		return map[string]any{"type": "object", "description": description, "propertyNames": field, "additionalProperties": value}
	}
	return map[string]any{
		"type":     "object",
		"required": []string{"document", "id"},
		"properties": map[string]any{
			"document": map[string]any{"type": "string", "description": "manifest document key (catalog, requirements, architecture, behavior, assurance, compliance, views, decisions, aviation, reqLinks) or its path"},
			"id":       map[string]any{"type": "string", "description": "stable ID of the entity to edit (any list item with that id in the document, nested ones such as flow steps included)"},
			"set":      fieldMap("field path (dotted for nested fields, e.g. acceptedBy.statement) to its new value: string, boolean, number, list or object; a missing field or mapping is added", map[string]any{"type": []string{"string", "boolean", "number", "array", "object"}}),
			"unset":    map[string]any{"type": "array", "description": "field paths to remove", "items": field},
			"append":   fieldMap("list field path to one item or a list of items to append; a missing list is created", map[string]any{"type": []string{"string", "boolean", "number", "array", "object"}}),
			"remove":   fieldMap("list field path to one value or a list of values to remove: a scalar item, or an object item by its id", map[string]any{"type": []string{"string", "boolean", "number", "array"}}),
			"write":    map[string]any{"type": "boolean", "description": "write the document when the result is valid (default false: dry run)"},
		},
		"additionalProperties": false,
	}
}

// patchOp is one field operation of a model.patch call.
// ENGMODEL-LINKS: FU-MCP-SERVER
type patchOp struct {
	kind  string // set, unset, append, remove
	path  []string
	value any
}

// parsePatchArgs checks the model.patch argument types and field paths and
// returns the operations in a deterministic order: set, unset, append,
// remove, each by field path.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
func parsePatchArgs(args map[string]any) ([]patchOp, bool, error) {
	write, err := parsePatchWrite(args)
	if err != nil {
		return nil, false, err
	}
	var ops []patchOp
	for _, kind := range []string{"set", "append", "remove"} {
		kindOps, err := parsePatchFieldMap(args, kind)
		if err != nil {
			return nil, false, err
		}
		ops = append(ops, kindOps...)
	}
	unsetOps, err := parsePatchUnset(args)
	if err != nil {
		return nil, false, err
	}
	ops = append(ops, unsetOps...)
	order := map[string]int{"set": 0, "unset": 1, "append": 2, "remove": 3}
	sort.SliceStable(ops, func(i, j int) bool { return order[ops[i].kind] < order[ops[j].kind] })
	if len(ops) == 0 {
		return nil, false, fmt.Errorf("model.patch needs at least one of set, unset, append or remove")
	}
	return ops, write, nil
}

// parsePatchWrite reads the optional boolean write argument.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
func parsePatchWrite(args map[string]any) (bool, error) {
	raw, ok := args["write"]
	if !ok {
		return false, nil
	}
	b, isBool := raw.(bool)
	if !isBool {
		return false, fmt.Errorf("model.patch write must be a boolean")
	}
	return b, nil
}

// patchFieldPath splits and checks one dotted field path of an operation.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
func patchFieldPath(op, field string) ([]string, error) {
	segments := strings.Split(field, ".")
	for _, s := range segments {
		if !patchFieldSegment.MatchString(s) {
			return nil, fmt.Errorf("model.patch %s field %q is not a dotted field path (letters, digits and _, e.g. acceptedBy.statement)", op, field)
		}
	}
	if segments[0] == "id" && (op == "set" || op == "unset") {
		return nil, fmt.Errorf("model.patch cannot %s the stable id; add a new entity and retire the old one instead", op)
	}
	return segments, nil
}

// parsePatchFieldMap reads the operations of one object-valued argument
// (set, append or remove), sorted by field path.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
func parsePatchFieldMap(args map[string]any, kind string) ([]patchOp, error) {
	raw, ok := args[kind]
	if !ok {
		return nil, nil
	}
	fields, isMap := raw.(map[string]any)
	if !isMap {
		return nil, fmt.Errorf("model.patch %s must be an object of field path to value", kind)
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	var ops []patchOp
	for _, name := range names {
		p, err := patchFieldPath(kind, name)
		if err != nil {
			return nil, err
		}
		ops = append(ops, patchOp{kind: kind, path: p, value: fields[name]})
	}
	return ops, nil
}

// parsePatchUnset reads the unset operations: a list of field path strings.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
func parsePatchUnset(args map[string]any) ([]patchOp, error) {
	raw, ok := args["unset"]
	if !ok {
		return nil, nil
	}
	list, isList := raw.([]any)
	if !isList {
		return nil, fmt.Errorf("model.patch unset must be a list of field paths")
	}
	var ops []patchOp
	for _, item := range list {
		name, isString := item.(string)
		if !isString {
			return nil, fmt.Errorf("model.patch unset entries must be field path strings")
		}
		p, err := patchFieldPath("unset", name)
		if err != nil {
			return nil, err
		}
		ops = append(ops, patchOp{kind: "unset", path: p})
	}
	return ops, nil
}

// itemsOf returns v as a list of items: a list as its items, anything else
// as one item.
// TRLC-LINKS: REQ-EMG-113
func itemsOf(v any) []any {
	if list, ok := v.([]any); ok {
		return list
	}
	return []any{v}
}

// applyPatch applies ops to the entity id of data.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-MODEL-CHANGE
func applyPatch(data []byte, id string, ops []patchOp) ([]byte, error) {
	find := yamledit.FindByID(id)
	if root, err := yamledit.Parse(data); err != nil {
		return nil, fmt.Errorf("parse document: %w", err)
	} else if _, err := find(root); err != nil {
		return nil, err
	}
	for _, op := range ops {
		var err error
		data, err = applyPatchOp(data, find, op)
		if err != nil {
			return nil, fmt.Errorf("%s %s of %s: %w", op.kind, strings.Join(op.path, "."), id, err)
		}
	}
	return data, nil
}

// applyPatchOp applies one operation to the entity found by find.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-MODEL-CHANGE
func applyPatchOp(data []byte, find func(*yaml.Node) (*yaml.Node, error), op patchOp) ([]byte, error) {
	at := yamledit.FieldAt{Find: find, Path: op.path}
	switch op.kind {
	case "set":
		return yamledit.SetPath(data, at, op.value)
	case "unset":
		return yamledit.UnsetPath(data, find, op.path)
	case "append":
		return yamledit.AppendPath(data, at, itemsOf(op.value))
	case "remove":
		return yamledit.RemovePath(data, at, itemsOf(op.value))
	}
	return data, nil
}

// patchRequest is the checked argument set of one model.patch call.
// ENGMODEL-LINKS: FU-MCP-SERVER
type patchRequest struct {
	document string
	id       string
	ops      []patchOp
	write    bool
}

// parsePatchRequest checks the model.patch arguments in their error
// precedence: document and id, the stable ID form, then the operations.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-STRICT-MCP-INPUT-SCHEMA
func parsePatchRequest(args map[string]any) (patchRequest, error) {
	document := strings.TrimSpace(toString(args["document"]))
	id := strings.TrimSpace(toString(args["id"]))
	if document == "" || id == "" {
		return patchRequest{}, fmt.Errorf("model.patch requires document and id")
	}
	if err := requireWellFormedStableID(id); err != nil {
		return patchRequest{}, err
	}
	ops, write, err := parsePatchArgs(args)
	if err != nil {
		return patchRequest{}, err
	}
	return patchRequest{document: document, id: id, ops: ops, write: write}, nil
}

// patchTarget is the manifest document a patch edits: its model relative
// path, its resolved file and its current content.
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
type patchTarget struct {
	rel    string
	path   string
	before []byte
}

// loadPatchTarget resolves document to a manifest document inside the model
// root and reads it.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func (s *Server) loadPatchTarget(document string) (patchTarget, error) {
	if s.bundle == nil || strings.TrimSpace(s.bundle.ManifestPath) == "" {
		return patchTarget{}, fmt.Errorf("model.patch requires a model loaded from engmod.yml")
	}
	root := filepath.Dir(s.bundle.ManifestPath)
	documents, err := readPatchManifestDocuments(s.bundle.ManifestPath)
	if err != nil {
		return patchTarget{}, err
	}
	rel, err := patchDocumentRel(documents, document)
	if err != nil {
		return patchTarget{}, err
	}
	if err := confineDocuments(root, documents); err != nil {
		return patchTarget{}, err
	}
	return readPatchTarget(root, rel)
}

// readPatchTarget resolves the symlinks of the document rel below root and
// reads it.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func readPatchTarget(root, rel string) (patchTarget, error) {
	target, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return patchTarget{}, fmt.Errorf("model.patch: read %s: %w", rel, err)
	}
	before, err := os.ReadFile(target)
	if err != nil {
		return patchTarget{}, fmt.Errorf("model.patch: read %s: %w", rel, err)
	}
	return patchTarget{rel: rel, path: target, before: before}, nil
}

// readPatchManifestDocuments reads the documents the manifest lists, by key.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func readPatchManifestDocuments(manifestPath string) (map[string]string, error) {
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var manifest model.ManifestDocument
	if err := goyaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	return manifestDocuments(manifest.Documents), nil
}

// patchDocumentRel resolves document to the model relative path of a
// manifest document model.patch may edit.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES
func patchDocumentRel(documents map[string]string, document string) (string, error) {
	key, rel, err := resolveDiagnoseDocument(documents, document)
	if err != nil {
		return "", fmt.Errorf("%s", strings.Replace(err.Error(), "model.diagnose", "model.patch", 1))
	}
	if key == "reqLinks" {
		// The link file records reviews; engws
		// review-links is its one write path (D14).
		return "", fmt.Errorf("model.patch does not edit the requirement link file %s: record a review with engws review-links --accept (links.review prepares it)", rel)
	}
	return rel, nil
}

// modelPatch edits one entity, validates the result like model.diagnose and,
// with write, writes a valid result atomically.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES, FU-VALIDATION-ENGINE
// ENGMODEL-LINKS: TR-MCP-MODEL-FILE-CHANGES
func (s *Server) modelPatch(args map[string]any) (map[string]any, error) {
	req, err := parsePatchRequest(args)
	if err != nil {
		return nil, err
	}
	target, err := s.loadPatchTarget(req.document)
	if err != nil {
		return nil, err
	}
	after, err := applyPatch(target.before, req.id, req.ops)
	if err != nil {
		return nil, fmt.Errorf("model.patch %s: %w", target.rel, err)
	}
	out, err := s.modelDiagnose(map[string]any{"document": target.rel, "content": string(after)})
	if err != nil {
		return nil, err
	}
	out["id"] = req.id
	out["patch"] = yamledit.UnifiedDiff(target.rel, target.before, after)
	out["written"] = false
	if err := req.conclude(out, target, after); err != nil {
		return nil, err
	}
	return out, nil
}

// conclude adds the outcome note to the diagnose result and, with write
// and a valid result, writes the patched document and clears the note.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES, TR-MCP-MODEL-FILE-CHANGES
func (req patchRequest) conclude(out map[string]any, target patchTarget, after []byte) error {
	valid, _ := out["valid"].(bool)
	switch {
	case !req.write:
		out["note"] = "dry run: nothing written; call again with write: true to write a valid result"
	case !valid:
		out["note"] = "not written: the result has validation errors (diagnostics, otherDiagnostics, parseErrors)"
	default:
		if err := writePatchTarget(target, after); err != nil {
			return err
		}
		out["written"] = true
		delete(out, "note")
	}
	return nil
}

// writePatchTarget writes the patched document atomically, unless the file
// changed since it was read.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES, TR-MCP-MODEL-FILE-CHANGES
func writePatchTarget(target patchTarget, after []byte) error {
	current, err := os.ReadFile(target.path)
	if err != nil || !bytes.Equal(current, target.before) {
		return fmt.Errorf("model.patch: %s changed while it was patched; nothing written", target.rel)
	}
	if err := yamledit.WriteFileAtomic(target.path, after); err != nil {
		return fmt.Errorf("model.patch: write %s: %w", target.rel, err)
	}
	return nil
}
