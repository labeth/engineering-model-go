// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

// State-like type discovery for Go (ADR-EMG-058, ADR-EMG-070 item 5): one
// walk over the syntax tree of a non-test source file collects the facts the
// rule needs (named types, typed constants, struct fields by type, assigned
// selector names and ENGMODEL-NOT-A-STATE comments), and GoStateCandidates
// applies the rule to the facts of a package. The check that turns candidates
// into diagnostics is in internal/trace.

import (
	"path/filepath"
	"sort"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// GoStateMarker starts the comment that says a declaration is not a state:
// the marker, a colon and a reason.
const GoStateMarker = "ENGMODEL-NOT-A-STATE"

// GoStateMinConstants is the number of constants of one type from which the
// type is a candidate.
const GoStateMinConstants = 3

// goStateExcludedSuffixes are the name endings of types that are enumerations
// of kinds, options or results, never states (case sensitive).
var goStateExcludedSuffixes = []string{"Kind", "Severity", "Format", "Type", "Reason", "Mode", "Option"}

// GoTypeDecl is a named type declaration: Line is the line of its spec and
// DeclLine the line of the type declaration that holds it (the same unless the
// declaration is grouped in parentheses).
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type GoTypeDecl struct {
	Name     string
	Line     int
	DeclLine int
}

// GoFieldDecl is a struct field declared with a named type.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type GoFieldDecl struct {
	Name string
	Type string
}

// GoStateSuppression is an ENGMODEL-NOT-A-STATE comment with the text after
// the marker (empty when missing).
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type GoStateSuppression struct {
	Line   int
	Reason string
}

// GoStateFacts are the facts of one file: its types, the number of constants
// of each type name, its fields of a named type, the selector names that an
// assignment statement writes and its suppression comments.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type GoStateFacts struct {
	File         string
	Types        []GoTypeDecl
	Consts       map[string]int
	Fields       []GoFieldDecl
	Assigned     map[string]bool
	Suppressions []GoStateSuppression
}

// GoStateCandidate is a named type with enough constants, a field of the type
// and an assignment to a field of that name.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type GoStateCandidate struct {
	File      string
	Type      string
	Line      int
	DeclLine  int
	Constants int
	Field     string
}

// goStateCollector walks one tree and fills the facts.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type goStateCollector struct {
	src   []byte
	facts *GoStateFacts
}

// ExtractGoStateFacts parses Go source and returns its state facts; file is
// recorded on the facts and on the candidates found from them.
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func ExtractGoStateFacts(file string, src []byte) (GoStateFacts, error) {
	facts := GoStateFacts{File: file, Consts: map[string]int{}, Assigned: map[string]bool{}}
	c := &goStateCollector{src: src, facts: &facts}
	err := ParseSource(LanguageGo, src, func(root *sitter.Node, _ string) error {
		c.walk(root)
		return nil
	})
	return facts, err
}

// walk visits a node and then its named children.
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) walk(n *sitter.Node) {
	c.visit(n)
	for i := uint(0); i < n.NamedChildCount(); i++ {
		c.walk(n.NamedChild(i))
	}
}

// visit records what one node contributes.
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) visit(n *sitter.Node) {
	switch n.Kind() {
	case "type_spec":
		c.typeSpec(n)
	case "const_declaration":
		c.constDecl(n)
	case "field_declaration":
		c.field(n)
	case "assignment_statement":
		c.assignment(n)
	case "comment":
		c.comment(n)
	}
}

// text returns the source text of a node, "" for nil.
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) text(n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return strings.TrimSpace(n.Utf8Text(c.src))
}

// goLine is the 1-based line a node starts on.
// TRLC-LINKS: REQ-EMG-188
func goLine(n *sitter.Node) int {
	return int(n.StartPosition().Row) + 1
}

// typeSpec records a named type declaration.
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) typeSpec(n *sitter.Node) {
	name := c.text(n.ChildByFieldName("name"))
	if name == "" {
		return
	}
	decl := n
	if p := n.Parent(); p != nil && p.Kind() == "type_declaration" {
		decl = p
	}
	c.facts.Types = append(c.facts.Types, GoTypeDecl{Name: name, Line: goLine(n), DeclLine: goLine(decl)})
}

// constDecl counts the constants of each named type in a const declaration:
// a spec with a type or a conversion value names the type, and a spec with
// neither type nor value repeats the one before (iota blocks).
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) constDecl(n *sitter.Node) {
	current := ""
	for i := uint(0); i < n.NamedChildCount(); i++ {
		spec := n.NamedChild(i)
		if spec.Kind() != "const_spec" {
			continue
		}
		current = c.constType(spec, current)
		if current != "" {
			c.facts.Consts[current] += c.constNames(spec)
		}
	}
}

// constNames is the number of names a const spec declares, not counting the
// blank identifier.
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) constNames(spec *sitter.Node) int {
	count := 0
	for i := uint(0); i < spec.NamedChildCount(); i++ {
		if ch := spec.NamedChild(i); ch.Kind() == "identifier" && c.text(ch) != "_" {
			count++
		}
	}
	return count
}

// constType returns the type name of a const spec: its declared type, the
// previous spec's type when the spec repeats it, the type of a conversion
// value such as State("x"), or "" for an untyped constant.
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) constType(spec *sitter.Node, previous string) string {
	if t := spec.ChildByFieldName("type"); t != nil {
		return c.text(t)
	}
	value := spec.ChildByFieldName("value")
	if value == nil {
		return previous
	}
	return c.conversionType(value)
}

// conversionType returns the type name of an expression list that is one
// conversion T(x) of a plain identifier, or "".
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) conversionType(list *sitter.Node) string {
	if list.NamedChildCount() != 1 {
		return ""
	}
	call := list.NamedChild(0)
	if call.Kind() != "call_expression" {
		return ""
	}
	fn := call.ChildByFieldName("function")
	if fn == nil || fn.Kind() != "identifier" {
		return ""
	}
	return c.text(fn)
}

// field records the names of a struct field declaration whose type is a plain
// named type.
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) field(n *sitter.Node) {
	t := n.ChildByFieldName("type")
	if t == nil || t.Kind() != "type_identifier" {
		return
	}
	for i := uint(0); i < n.NamedChildCount(); i++ {
		if ch := n.NamedChild(i); ch.Kind() == "field_identifier" {
			c.facts.Fields = append(c.facts.Fields, GoFieldDecl{Name: c.text(ch), Type: c.text(t)})
		}
	}
}

// assignment records the field name of each selector expression on the left of
// an assignment statement (x.f = v, x.f += v); a composite literal and a
// declaration do not count.
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) assignment(n *sitter.Node) {
	left := n.ChildByFieldName("left")
	if left == nil {
		return
	}
	for i := uint(0); i < left.NamedChildCount(); i++ {
		if sel := left.NamedChild(i); sel.Kind() == "selector_expression" {
			c.facts.Assigned[c.text(sel.ChildByFieldName("field"))] = true
		}
	}
}

// comment records an ENGMODEL-NOT-A-STATE comment.
// TRLC-LINKS: REQ-EMG-188
func (c *goStateCollector) comment(n *sitter.Node) {
	if reason, ok := suppressionReason(c.text(n)); ok {
		c.facts.Suppressions = append(c.facts.Suppressions, GoStateSuppression{Line: goLine(n), Reason: reason})
	}
}

// suppressionReason reports whether a comment is a suppression, that is,
// starts with the marker, and returns the text after the marker and its colon
// on the first line.
// TRLC-LINKS: REQ-EMG-188
func suppressionReason(comment string) (string, bool) {
	body := strings.TrimPrefix(strings.TrimSuffix(strings.TrimPrefix(comment, "//"), "*/"), "/*")
	body, _, _ = strings.Cut(strings.TrimSpace(body), "\n")
	rest, ok := strings.CutPrefix(body, GoStateMarker)
	if !ok || !markerBoundary(rest) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(rest), ":")), true
}

// markerBoundary reports whether the text after the marker name ends the
// word: nothing, a colon or a space.
// TRLC-LINKS: REQ-EMG-188
func markerBoundary(rest string) bool {
	return rest == "" || strings.ContainsRune(": \t", rune(rest[0]))
}

// GoStateExcludedName reports whether a type name ends in one of the suffixes
// of enumerations that are never states.
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func GoStateExcludedName(name string) bool {
	for _, suffix := range goStateExcludedSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// packageFacts is the merge of the facts of the files of one package.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type packageFacts struct {
	consts   map[string]int
	assigned map[string]bool
	fields   []GoFieldDecl
}

// mergePackage merges the facts of the files of one package.
// TRLC-LINKS: REQ-EMG-188
func mergePackage(files []GoStateFacts) packageFacts {
	p := packageFacts{consts: map[string]int{}, assigned: map[string]bool{}}
	for _, f := range files {
		for name, count := range f.Consts {
			p.consts[name] += count
		}
		for name := range f.Assigned {
			p.assigned[name] = true
		}
		p.fields = append(p.fields, f.Fields...)
	}
	return p
}

// assignedField returns the first field of the type that an assignment writes
// by name, or "".
// TRLC-LINKS: REQ-EMG-188
func (p packageFacts) assignedField(typeName string) string {
	for _, f := range p.fields {
		if f.Type == typeName && p.assigned[f.Name] {
			return f.Name
		}
	}
	return ""
}

// candidate applies the rule to one type: it is not excluded by name, has the
// minimum number of constants and has a field of its type that is assigned.
// TRLC-LINKS: REQ-EMG-188
func (p packageFacts) candidate(file string, t GoTypeDecl) (GoStateCandidate, bool) {
	if GoStateExcludedName(t.Name) || p.consts[t.Name] < GoStateMinConstants {
		return GoStateCandidate{}, false
	}
	field := p.assignedField(t.Name)
	if field == "" {
		return GoStateCandidate{}, false
	}
	return GoStateCandidate{File: file, Type: t.Name, Line: t.Line, DeclLine: t.DeclLine, Constants: p.consts[t.Name], Field: field}, true
}

// GoStateCandidates applies the discovery rule to the facts of non-test,
// non-generated Go files: a named type with at least three constants of that
// type in its package, a struct field of that type and an assignment whose
// left side is a selector with that field's name, whose name does not end in
// Kind, Severity, Format, Type, Reason, Mode or Option. Files of one directory
// are one package. The result is sorted by file and line.
// TRLC-LINKS: REQ-EMG-188
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func GoStateCandidates(facts []GoStateFacts) []GoStateCandidate {
	byDir := map[string][]GoStateFacts{}
	for _, f := range facts {
		dir := filepath.Dir(f.File)
		byDir[dir] = append(byDir[dir], f)
	}
	var out []GoStateCandidate
	for _, files := range byDir {
		p := mergePackage(files)
		for _, f := range files {
			out = append(out, fileCandidates(p, f)...)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// fileCandidates returns the candidates among the types of one file.
// TRLC-LINKS: REQ-EMG-188
func fileCandidates(p packageFacts, f GoStateFacts) []GoStateCandidate {
	var out []GoStateCandidate
	for _, t := range f.Types {
		if c, ok := p.candidate(f.File, t); ok {
			out = append(out, c)
		}
	}
	return out
}
