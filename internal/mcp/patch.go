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
	write := false
	if raw, ok := args["write"]; ok {
		b, isBool := raw.(bool)
		if !isBool {
			return nil, false, fmt.Errorf("model.patch write must be a boolean")
		}
		write = b
	}
	path := func(op, field string) ([]string, error) {
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
	var ops []patchOp
	for _, kind := range []string{"set", "append", "remove"} {
		raw, ok := args[kind]
		if !ok {
			continue
		}
		fields, isMap := raw.(map[string]any)
		if !isMap {
			return nil, false, fmt.Errorf("model.patch %s must be an object of field path to value", kind)
		}
		names := make([]string, 0, len(fields))
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			p, err := path(kind, name)
			if err != nil {
				return nil, false, err
			}
			ops = append(ops, patchOp{kind: kind, path: p, value: fields[name]})
		}
	}
	if raw, ok := args["unset"]; ok {
		list, isList := raw.([]any)
		if !isList {
			return nil, false, fmt.Errorf("model.patch unset must be a list of field paths")
		}
		for _, item := range list {
			name, isString := item.(string)
			if !isString {
				return nil, false, fmt.Errorf("model.patch unset entries must be field path strings")
			}
			p, err := path("unset", name)
			if err != nil {
				return nil, false, err
			}
			ops = append(ops, patchOp{kind: "unset", path: p})
		}
	}
	order := map[string]int{"set": 0, "unset": 1, "append": 2, "remove": 3}
	sort.SliceStable(ops, func(i, j int) bool { return order[ops[i].kind] < order[ops[j].kind] })
	if len(ops) == 0 {
		return nil, false, fmt.Errorf("model.patch needs at least one of set, unset, append or remove")
	}
	return ops, write, nil
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
	var err error
	for _, op := range ops {
		field := strings.Join(op.path, ".")
		switch op.kind {
		case "set":
			data, err = yamledit.SetPath(data, find, op.path, op.value)
		case "unset":
			data, err = yamledit.UnsetPath(data, find, op.path)
		case "append":
			data, err = yamledit.AppendPath(data, find, op.path, itemsOf(op.value))
		case "remove":
			data, err = yamledit.RemovePath(data, find, op.path, itemsOf(op.value))
		}
		if err != nil {
			return nil, fmt.Errorf("%s %s of %s: %w", op.kind, field, id, err)
		}
	}
	return data, nil
}

// modelPatch edits one entity, validates the result like model.diagnose and,
// with write, writes a valid result atomically.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-REPO-LOCAL-WRITES, FU-VALIDATION-ENGINE
// ENGMODEL-LINKS: TR-MCP-MODEL-FILE-CHANGES
func (s *Server) modelPatch(args map[string]any) (map[string]any, error) {
	document := strings.TrimSpace(toString(args["document"]))
	id := strings.TrimSpace(toString(args["id"]))
	if document == "" || id == "" {
		return nil, fmt.Errorf("model.patch requires document and id")
	}
	if err := requireWellFormedStableID(id); err != nil {
		return nil, err
	}
	ops, write, err := parsePatchArgs(args)
	if err != nil {
		return nil, err
	}
	if s.bundle == nil || strings.TrimSpace(s.bundle.ManifestPath) == "" {
		return nil, fmt.Errorf("model.patch requires a model loaded from engmod.yml")
	}
	root := filepath.Dir(s.bundle.ManifestPath)
	manifestBytes, err := os.ReadFile(s.bundle.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var manifest model.ManifestDocument
	if err := goyaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	documents := manifestDocuments(manifest.Documents)
	key, rel, err := resolveDiagnoseDocument(documents, document)
	if err != nil {
		return nil, fmt.Errorf("%s", strings.Replace(err.Error(), "model.diagnose", "model.patch", 1))
	}
	if key == "reqLinks" {
		// The link file records reviews; engws
		// review-links is its one write path (D14).
		return nil, fmt.Errorf("model.patch does not edit the requirement link file %s: record a review with engws review-links --accept (links.review prepares it)", rel)
	}
	if err := confineDocuments(root, documents); err != nil {
		return nil, err
	}
	target, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fmt.Errorf("model.patch: read %s: %w", rel, err)
	}
	before, err := os.ReadFile(target)
	if err != nil {
		return nil, fmt.Errorf("model.patch: read %s: %w", rel, err)
	}
	after, err := applyPatch(before, id, ops)
	if err != nil {
		return nil, fmt.Errorf("model.patch %s: %w", rel, err)
	}
	out, err := s.modelDiagnose(map[string]any{"document": rel, "content": string(after)})
	if err != nil {
		return nil, err
	}
	out["id"] = id
	out["patch"] = yamledit.UnifiedDiff(rel, before, after)
	out["written"] = false
	valid, _ := out["valid"].(bool)
	switch {
	case !write:
		out["note"] = "dry run: nothing written; call again with write: true to write a valid result"
	case !valid:
		out["note"] = "not written: the result has validation errors (diagnostics, otherDiagnostics, parseErrors)"
	default:
		current, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(current, before) {
			return nil, fmt.Errorf("model.patch: %s changed while it was patched; nothing written", rel)
		}
		if err := yamledit.WriteFileAtomic(target, after); err != nil {
			return nil, fmt.Errorf("model.patch: write %s: %w", rel, err)
		}
		out["written"] = true
		delete(out, "note")
	}
	return out, nil
}
