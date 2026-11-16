// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	"strings"
)

// TRLC-LINKS: REQ-EMG-010
func pythonDeclaration(n *sitter.Node, src []byte) (declaration, bool) {
	if n == nil || !isPythonDefinition(n) {
		return declaration{}, false
	}
	name := n.ChildByFieldName("name")
	if name == nil {
		return declaration{}, false
	}
	d := declaration{Name: pythonQualifiedName(n, name.Utf8Text(src), src), Kind: n.Kind(), Line: int(n.StartPosition().Row) + 1,
		Signature: firstLine(strings.TrimSpace(n.Utf8Text(src))), RequiresTRLC: n.Kind() == "function_definition"}
	if parent := n.Parent(); parent != nil && parent.Kind() == "decorated_definition" {
		d.LeadingLine = int(parent.StartPosition().Row) + 1
	}
	return d, true
}

// isPythonDefinition reports whether n is a function or class definition.
// TRLC-LINKS: REQ-EMG-010
func isPythonDefinition(n *sitter.Node) bool {
	return n.Kind() == "function_definition" || n.Kind() == "class_definition"
}

// pythonQualifiedName prefixes name with the names of the definitions that
// enclose n.
// TRLC-LINKS: REQ-EMG-010
func pythonQualifiedName(n *sitter.Node, name string, src []byte) string {
	qualified := name
	for parent := n.Parent(); parent != nil; parent = parent.Parent() {
		if !isPythonDefinition(parent) {
			continue
		}
		if scope := parent.ChildByFieldName("name"); scope != nil {
			qualified = scope.Utf8Text(src) + "." + qualified
		}
	}
	return qualified
}

// CommentLines returns parser-recognized comment locations, excluding strings.
// Python locations include only standalone comments, never a string prefix on
// the same line as an unrelated trailing comment.
// TRLC-LINKS: REQ-EMG-010
func CommentLines(path string, src []byte) (map[int]bool, error) {
	_, comments, _, err := extractDeclarations(path, src)
	return comments, err
}

// TRLC-LINKS: REQ-EMG-010
func pythonStandaloneComment(n *sitter.Node, src []byte) bool {
	start := n.StartByte()
	column := uint(n.StartPosition().Column)
	return strings.TrimSpace(string(src[start-column:start])) == ""
}
