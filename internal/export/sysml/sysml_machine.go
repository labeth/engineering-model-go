// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package sysml

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// machineCommentedCode is the diagnostic code for a machine construct that
// the exporter wrote as a comment because SysML v2 has no valid syntax for it.
const machineCommentedCode = "sysml.machine_construct_commented"

// machineIndex groups the projected parts of every behavior machine so the
// machine writer can nest them in the machine's state definition. A nil index
// is valid and owns nothing.
type machineIndex struct {
	machines      map[string]bool
	states        map[string][]model.SemanticElement
	invariants    map[string][]model.SemanticElement
	transitions   map[string][]model.SemanticRelationship
	successions   map[string][]model.SemanticRelationship
	ownedElements map[string]bool
	ownedRels     map[string]bool
}

// indexMachines finds the machine state definitions of a semantic model and
// the elements and relationships they own. Parts whose owner is not a machine
// of the model are left to the generic writers.
//
// TRLC-LINKS: REQ-EMG-137
func indexMachines(semantic model.SemanticModel) *machineIndex {
	index := &machineIndex{
		machines:      map[string]bool{},
		states:        map[string][]model.SemanticElement{},
		invariants:    map[string][]model.SemanticElement{},
		transitions:   map[string][]model.SemanticRelationship{},
		successions:   map[string][]model.SemanticRelationship{},
		ownedElements: map[string]bool{},
		ownedRels:     map[string]bool{},
	}
	for _, element := range semantic.Elements {
		if element.Kind == model.ElementStateDefinition && element.Extension == model.MachineExtension && element.ID != "" {
			index.machines[element.ID] = true
		}
	}
	for _, element := range semantic.Elements {
		index.addElement(element)
	}
	for _, relationship := range semantic.Relationships {
		index.addRelationship(relationship)
	}
	return index
}

// addElement records a state or invariant that a machine owns.
//
// TRLC-LINKS: REQ-EMG-137
func (index *machineIndex) addElement(element model.SemanticElement) {
	if !index.machines[element.Owner] {
		return
	}
	switch {
	case element.Kind == model.ElementStateUsage && element.Extension == model.MachineStateExtension:
		index.states[element.Owner] = append(index.states[element.Owner], element)
	case element.Kind == model.ElementConstraintUsage && element.Extension == model.MachineInvariantExtension:
		index.invariants[element.Owner] = append(index.invariants[element.Owner], element)
	default:
		return
	}
	index.ownedElements[element.ID] = true
}

// addRelationship records a transition or an initial or final succession that
// a machine owns.
//
// TRLC-LINKS: REQ-EMG-137
func (index *machineIndex) addRelationship(relationship model.SemanticRelationship) {
	if !index.machines[relationship.Owner] {
		return
	}
	switch {
	case relationship.Kind == model.RelationshipTransition && relationship.SourceType == model.MachineTransitionSourceType:
		index.transitions[relationship.Owner] = append(index.transitions[relationship.Owner], relationship)
	case relationship.Kind == model.RelationshipSuccession &&
		(relationship.SourceType == model.MachineInitialSourceType || relationship.SourceType == model.MachineFinalSourceType):
		index.successions[relationship.Owner] = append(index.successions[relationship.Owner], relationship)
	default:
		return
	}
	index.ownedRels[relationship.ID] = true
}

// TRLC-LINKS: REQ-EMG-137
func (index *machineIndex) isMachine(id string) bool {
	return index != nil && index.machines[id]
}

// ownsElement reports whether the machine writer renders the element inside
// its machine, so the namespace writer must skip it.
//
// TRLC-LINKS: REQ-EMG-137
func (index *machineIndex) ownsElement(id string) bool {
	return index != nil && index.ownedElements[id]
}

// TRLC-LINKS: REQ-EMG-137
func (index *machineIndex) ownsRelationship(id string) bool {
	return index != nil && index.ownedRels[id]
}

// machineWriter renders one machine. It records every construct it has to
// write as a comment in diagnostics.
type machineWriter struct {
	out         *bytes.Buffer
	machine     model.SemanticElement
	states      map[string]bool
	elementIDs  map[string]bool
	diagnostics *[]model.SemanticDiagnostic
}

// machine renders a projected behavior machine as a SysML v2 state
// definition: variables as attributes with initial values, an entry
// succession to the initial state, the states, final states as successions to
// done, transitions with accept, if and do parts, and invariants as
// constraints. Constructs SysML cannot express are written as comments and
// reported with sysml.machine_construct_commented.
//
// TRLC-LINKS: REQ-EMG-137, REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func (w *sysmlWriter) machine(machine model.SemanticElement, indent string) {
	writer := &machineWriter{
		out: w.out, machine: machine, states: map[string]bool{}, elementIDs: w.elementIDs, diagnostics: w.diagnostics,
	}
	for _, state := range w.machines.states[machine.ID] {
		writer.states[state.ID] = true
	}
	body := indent + "    "
	fmt.Fprintf(w.out, "%sstate def %s {\n", indent, sysmlName(machine.ID))
	writer.doc(body)
	writer.members(w.machines, body)
	w.namespaceElements(machine.ID, body)
	fmt.Fprintf(w.out, "%s}\n", indent)
}

// doc writes the description and the owning unit of the machine as a doc comment.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) doc(indent string) {
	notes := machineDoc(w.machine.Metadata, "description")
	if w.machine.Owner != "" {
		notes = strings.TrimSpace(notes + " Implemented by " + commentText(w.machine.Owner) + ".")
	}
	if notes != "" {
		fmt.Fprintf(w.out, "%sdoc /* %s */\n", indent, notes)
	}
}

// members writes the variables, the initial succession, the states, the
// final successions, the transitions and the invariants of the machine.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) members(index *machineIndex, indent string) {
	id := w.machine.ID
	for _, feature := range w.machine.Features {
		w.variable(feature, indent)
	}
	w.initials(index.successions[id], indent)
	w.stateDeclarations(index.states[id], indent)
	w.finals(index.successions[id], indent)
	for _, transition := range index.transitions[id] {
		w.transition(transition, indent)
	}
	for _, invariant := range index.invariants[id] {
		w.invariant(invariant, indent)
	}
}

// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) initials(successions []model.SemanticRelationship, indent string) {
	for _, succession := range successions {
		if succession.SourceType == model.MachineInitialSourceType {
			w.initial(succession, indent)
		}
	}
}

// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) finals(successions []model.SemanticRelationship, indent string) {
	for _, succession := range successions {
		if succession.SourceType == model.MachineFinalSourceType {
			w.final(succession, indent)
		}
	}
}

// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) stateDeclarations(states []model.SemanticElement, indent string) {
	for _, state := range states {
		fmt.Fprintf(w.out, "%sstate %s;\n", indent, sysmlName(state.ID))
		writeAboutAnnotation(w.out, indent, aboutAnnotation{metadata: "EngineeringElement", id: state.ID, conceptKind: string(state.Kind)})
	}
}

// aboutAnnotation names the engineering identity metadata of one nested machine member.
type aboutAnnotation struct {
	metadata         string
	id               string
	conceptKind      string
	relationshipKind string
}

// writeAboutAnnotation attaches the engineering identity metadata to one
// nested machine member with the "about" form, so it names that member and
// not the enclosing state definition.
//
// TRLC-LINKS: REQ-EMG-137, REQ-EMG-036
func writeAboutAnnotation(out *bytes.Buffer, indent string, about aboutAnnotation) {
	fmt.Fprintf(out, "%s@%s about %s {\n", indent, about.metadata, sysmlName(about.id))
	fmt.Fprintf(out, "%s    sourceId = %s;\n", indent, sysmlString(about.id))
	fmt.Fprintf(out, "%s    conceptKind = %s;\n", indent, sysmlString(about.conceptKind))
	if about.relationshipKind != "" {
		fmt.Fprintf(out, "%s    relationshipKind = %s;\n", indent, sysmlString(about.relationshipKind))
	}
	fmt.Fprintf(out, "%s}\n", indent)
}

// variable writes one machine variable as an attribute with an initial value.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) variable(feature model.SemanticFeature, indent string) {
	path := "elements." + w.machine.ID + ".features." + feature.Name
	typ := strings.TrimSpace(feature.Type)
	if typ != string(model.MachineTypeInteger) && typ != string(model.MachineTypeBoolean) && typ != string(model.MachineTypeString) {
		w.commented(indent, commentedConstruct{path: path, construct: "variable " + feature.Name, reason: fmt.Sprintf("type %q is not Integer, Boolean or String", typ)})
		return
	}
	name, renamed := machineIdentifier(feature.Name)
	line := fmt.Sprintf("attribute %s : %s", name, typ)
	if feature.Value != nil {
		value, err := translateMachineExpression(*feature.Value)
		if err != nil {
			w.commented(indent, commentedConstruct{path: path, construct: "variable " + feature.Name, reason: err.Error(), source: model.MachineExprString(*feature.Value)})
			return
		}
		line += " := " + value
	}
	line += ";"
	if renamed {
		line += " // authored name: " + commentText(feature.Name)
	}
	fmt.Fprintf(w.out, "%s%s\n", indent, line)
}

// initial writes the entry succession to the initial state.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) initial(succession model.SemanticRelationship, indent string) {
	if !w.states[succession.Target] {
		w.commented(indent, commentedConstruct{path: "relationships." + succession.ID, construct: "initial state of " + w.machine.ID,
			reason: fmt.Sprintf("%q is not a state of the machine", succession.Target)})
		return
	}
	fmt.Fprintf(w.out, "%sentry;\n%sthen %s;\n", indent, indent, sysmlName(succession.Target))
}

// final writes a succession from a final state to the predefined done action.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) final(succession model.SemanticRelationship, indent string) {
	if !w.states[succession.Source] {
		w.commented(indent, commentedConstruct{path: "relationships." + succession.ID, construct: "final state of " + w.machine.ID,
			reason: fmt.Sprintf("%q is not a state of the machine", succession.Source)})
		return
	}
	fmt.Fprintf(w.out, "%ssuccession %s first %s then done;\n", indent, sysmlName(succession.ID), sysmlName(succession.Source))
	writeAboutAnnotation(w.out, indent, aboutAnnotation{"EngineeringRelationship", succession.ID, string(succession.Kind), succession.SourceType})
}

// transition writes one transition with its accept, if and do parts. A guard
// or effect that SysML cannot express is replaced by a comment in its place.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) transition(transition model.SemanticRelationship, indent string) {
	if !w.states[transition.Source] || !w.states[transition.Target] {
		w.commented(indent, commentedConstruct{path: "relationships." + transition.ID, construct: "transition " + transition.ID,
			reason: fmt.Sprintf("source %q or target %q is not a state of the machine", transition.Source, transition.Target)})
		return
	}
	part := indent + "    "
	fmt.Fprintf(w.out, "%stransition %s\n%sfirst %s\n", indent, sysmlName(transition.ID), part, sysmlName(transition.Source))
	w.triggers(transition, part)
	w.guard(transition, part)
	w.effect(transition, part)
	if doc := machineDoc(transition.Metadata, "description"); doc != "" {
		fmt.Fprintf(w.out, "%sthen %s {\n%s    doc /* %s */\n%s}\n", part, sysmlName(transition.Target), part, doc, part)
	} else {
		fmt.Fprintf(w.out, "%sthen %s;\n", part, sysmlName(transition.Target))
	}
	writeAboutAnnotation(w.out, indent, aboutAnnotation{"EngineeringRelationship", transition.ID, string(transition.Kind), transition.SourceType})
}

// triggers writes the accept part of a transition; a SysML transition accepts
// one trigger, so the others are written as comments.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) triggers(transition model.SemanticRelationship, part string) {
	if len(transition.Triggers) == 0 {
		return
	}
	trigger := transition.Triggers[0]
	payload := sysmlName("on-" + trigger)
	if w.elementIDs[trigger] {
		fmt.Fprintf(w.out, "%saccept %s :> %s\n", part, payload, sysmlName(trigger))
	} else {
		fmt.Fprintf(w.out, "%saccept %s : EngineeringEvent\n", part, payload)
	}
	for _, extra := range transition.Triggers[1:] {
		w.commented(part, commentedConstruct{path: "relationships." + transition.ID, construct: "additional trigger " + extra + " of transition " + transition.ID,
			reason: "a SysML transition accepts one trigger"})
	}
}

// guard writes the if part of a transition, or a comment when the guard is
// not expressible.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) guard(transition model.SemanticRelationship, part string) {
	if transition.Guard == nil {
		return
	}
	guard, err := translateMachineExpression(*transition.Guard)
	if err != nil {
		w.commented(part, commentedConstruct{path: "relationships." + transition.ID, construct: "guard of transition " + transition.ID,
			reason: err.Error(), source: authoredSource(transition.Metadata, "guard", *transition.Guard)})
		return
	}
	fmt.Fprintf(w.out, "%sif %s\n", part, guard)
}

// effect writes the do part of a transition, or a comment when the effect is
// not expressible.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) effect(transition model.SemanticRelationship, part string) {
	if transition.Effect == nil {
		return
	}
	assignments, err := translateMachineEffect(*transition.Effect)
	if err != nil {
		w.commented(part, commentedConstruct{path: "relationships." + transition.ID, construct: "effect of transition " + transition.ID,
			reason: err.Error(), source: authoredSource(transition.Metadata, "effect", *transition.Effect)})
		return
	}
	if len(assignments) == 1 {
		fmt.Fprintf(w.out, "%sdo %s\n", part, assignments[0])
		return
	}
	fmt.Fprintf(w.out, "%sdo action {\n", part)
	fmt.Fprintf(w.out, "%s    // simultaneous assignment: every right-hand side reads the pre-transition values\n", part)
	for _, assignment := range assignments {
		fmt.Fprintf(w.out, "%s    %s;\n", part, assignment)
	}
	fmt.Fprintf(w.out, "%s}\n", part)
}

// invariant writes one invariant as a constraint holding its Boolean
// expression, or as a comment when the expression is not expressible.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) invariant(invariant model.SemanticElement, indent string) {
	construct := commentedConstruct{path: "elements." + invariant.ID, construct: "invariant " + invariant.ID}
	result := invariantResult(invariant)
	if result == nil {
		construct.reason = "it has no result expression"
		w.commented(indent, construct)
		return
	}
	expression, err := translateMachineExpression(*result)
	if err != nil {
		construct.reason, construct.source = err.Error(), authoredSource(invariant.Metadata, "expr", *result)
		w.commented(indent, construct)
		return
	}
	fmt.Fprintf(w.out, "%sconstraint %s {\n", indent, sysmlName(invariant.ID))
	if notes := invariantNotes(invariant); notes != "" {
		fmt.Fprintf(w.out, "%s    doc /* %s */\n", indent, notes)
	}
	fmt.Fprintf(w.out, "%s    %s\n%s}\n", indent, expression, indent)
	writeAboutAnnotation(w.out, indent, aboutAnnotation{metadata: "EngineeringElement", id: invariant.ID, conceptKind: string(invariant.Kind)})
}

// invariantResult returns the result expression of an invariant, the last
// feature with the result name, or nil.
//
// TRLC-LINKS: REQ-EMG-137
func invariantResult(invariant model.SemanticElement) *model.SemanticExpression {
	var result *model.SemanticExpression
	for index := range invariant.Features {
		if invariant.Features[index].Name == model.MachineInvariantResultFeature {
			result = invariant.Features[index].Value
		}
	}
	return result
}

// invariantNotes returns the doc comment text of an invariant: its
// description and the requirements it protects.
//
// TRLC-LINKS: REQ-EMG-137
func invariantNotes(invariant model.SemanticElement) string {
	notes := machineDoc(invariant.Metadata, "description")
	if notes != "" && !strings.HasSuffix(notes, ".") {
		notes += "."
	}
	if len(invariant.References) > 0 {
		notes = strings.TrimSpace(notes + " Protects " + strings.Join(invariant.References, ", ") + ".")
	}
	return commentText(notes)
}

// commentedConstruct describes a construct that was written as a comment: the
// model path, what it is, why it is not expressible and its authored source.
type commentedConstruct struct {
	path      string
	construct string
	reason    string
	source    string
}

// commented writes a construct SysML cannot express as comment lines and
// records the loss as a warning that names the construct and the reason.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) commented(indent string, c commentedConstruct) {
	message := fmt.Sprintf("machine %s: %s was written as a comment because %s", w.machine.ID, c.construct, c.reason)
	fmt.Fprintf(w.out, "%s// %s is not expressible in SysML v2: %s\n", indent, commentText(c.construct), commentText(c.reason))
	if c.source != "" {
		message += fmt.Sprintf(" (authored: %s)", c.source)
		w.authoredLines(indent, c.source)
	}
	*w.diagnostics = append(*w.diagnostics, model.SemanticDiagnostic{
		Code: machineCommentedCode, Severity: model.SemanticSeverityWarning, Message: message, Path: c.path,
	})
}

// authoredLines writes the authored source of a commented construct, one comment line each.
//
// TRLC-LINKS: REQ-EMG-137
func (w *machineWriter) authoredLines(indent, source string) {
	for _, line := range strings.Split(source, "\n") {
		fmt.Fprintf(w.out, "%s//     authored: %s\n", indent, strings.TrimRight(line, "\r"))
	}
}

// authoredSource returns the authored expression string kept in the metadata,
// or the canonical source text of the tree when the metadata has none.
//
// TRLC-LINKS: REQ-EMG-137
func authoredSource(metadata []model.SemanticMetadata, key string, expression model.SemanticExpression) string {
	if source := metadataString(metadata, key); source != "" {
		return source
	}
	return model.MachineExprString(expression)
}

// metadataString reads a string property from the first metadata entry that has it.
//
// TRLC-LINKS: REQ-EMG-137
func metadataString(metadata []model.SemanticMetadata, key string) string {
	for _, entry := range metadata {
		if value, ok := entry.Properties[key]; ok && value.Kind == "string" && value.String != "" {
			return value.String
		}
	}
	return ""
}

// machineDoc returns the description stored under key in the metadata as
// text that is safe inside a block comment.
//
// TRLC-LINKS: REQ-EMG-137
func machineDoc(metadata []model.SemanticMetadata, key string) string {
	return commentText(metadataString(metadata, key))
}

// commentText makes text safe for a line or block comment: it joins lines
// and breaks the block-comment terminator.
//
// TRLC-LINKS: REQ-EMG-137
func commentText(text string) string {
	text = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "*/", "* /").Replace(text)
	return strings.TrimSpace(text)
}

var machineBareIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// machineIdentifier returns the SysML name of a machine variable. A legal
// identifier is bare; a reserved word or an illegal identifier is quoted; a
// name that the state definition would inherit from the SysML library (for
// example done, which the final-state succession uses) gets a suffix, and
// renamed reports that.
//
// TRLC-LINKS: REQ-EMG-137
func machineIdentifier(name string) (ident string, renamed bool) {
	name = strings.TrimSpace(name)
	switch {
	case machineInheritedNames[name]:
		return name + "_var", true
	case !machineBareIdentifier.MatchString(name) || sysmlReservedWords[name]:
		return sysmlName(name), false
	default:
		return name, false
	}
}

// machineInheritedNames are features that every SysML state definition
// inherits from the standard library.
var machineInheritedNames = map[string]bool{
	"done": true, "start": true, "self": true, "that": true,
	"entryAction": true, "doAction": true, "exitAction": true,
	"subactions": true, "stateTransitions": true, "isParallel": true,
}

// sysmlReservedWords are the keyword tokens of the pinned SysML v2 and KerML
// grammars (InternalSysML.tokens, InternalKerML.tokens, 2026-04 release).
var sysmlReservedWords = func() map[string]bool {
	words := strings.Fields(`about abstract accept action actor after alias all allocate allocation analysis and as
assert assign assoc assume at attribute behavior bind binding bool by calc case chains class classifier comment
composite concern conjugate conjugates conjugation connect connection connector const constant constraint crosses
datatype decide def default defined dependency derived differences disjoining disjoint do doc else end entry enum
event exhibit exit expose expr false feature featured featuring filter first flow for fork frame from function
hastype if implies import in include individual inout interaction interface intersects inv inverse inverting istype
item join language library locale loop member merge message meta metaclass metadata multiplicity namespace new
nonunique not null objective occurrence of or ordered out package parallel part perform port portion predicate
private protected public redefines redefinition ref references render rendering rep require requirement return
satisfy send snapshot specialization specializes stakeholder standard state step struct subclassifier subject subset
subsets subtype succession terminate then timeslice to transition true type typed typing unions until use var
variant variation verification verify via view viewpoint when while xor`)
	out := make(map[string]bool, len(words))
	for _, word := range words {
		out[word] = true
	}
	return out
}()

// machineSysMLOperators maps the operator names of the shared machine
// operator table to their SysML v2 / KerML expression tokens. in_state, :=
// and all have no entry: SysML has no state-test expression, and assignment
// is a statement handled by translateMachineEffect.
var machineSysMLOperators = map[string]string{
	"and": "and", "or": "or", "implies": "implies", "not": "not",
	"==": "==", "!=": "!=", "<": "<", "<=": "<=", ">": ">", ">=": ">=",
	"+": "+", "-": "-", "*": "*", "neg": "-",
}

var errMachineInState = errors.New("in_state(...) has no SysML v2 expression: SysML has no state-test operator")

// KerML expression precedence, low to high. KerML binds not and unary minus
// tighter than the machine language binds not, so the translation derives the
// parentheses from this table and not from the authored text.
const (
	sysmlPrecImplies = 1 + iota
	sysmlPrecOr
	sysmlPrecAnd
	sysmlPrecComparison
	sysmlPrecAdditive
	sysmlPrecMultiplicative
	sysmlPrecUnary
	sysmlPrecPrimary
)

var sysmlOperatorPrecedence = map[string]int{
	"implies": sysmlPrecImplies, "or": sysmlPrecOr, "and": sysmlPrecAnd,
	"==": sysmlPrecComparison, "!=": sysmlPrecComparison, "<": sysmlPrecComparison, "<=": sysmlPrecComparison,
	">": sysmlPrecComparison, ">=": sysmlPrecComparison,
	"+": sysmlPrecAdditive, "-": sysmlPrecAdditive, "*": sysmlPrecMultiplicative,
	"not": sysmlPrecUnary, "neg": sysmlPrecUnary,
}

// translateMachineExpression renders a machine expression tree as a SysML v2
// expression with the fewest parentheses that keep the tree's grouping.
//
// TRLC-LINKS: REQ-EMG-137
func translateMachineExpression(expression model.SemanticExpression) (string, error) {
	text, _, err := translateMachineNode(expression)
	return text, err
}

// translateMachineNode returns the SysML text of a node and its precedence.
//
// TRLC-LINKS: REQ-EMG-137
func translateMachineNode(expression model.SemanticExpression) (string, int, error) {
	switch expression.Kind {
	case model.MachineKindLiteral:
		text, err := translateMachineLiteral(expression)
		return text, sysmlPrecPrimary, err
	case model.MachineKindVariable:
		if strings.TrimSpace(expression.Value) == "" {
			return "", 0, errors.New("a variable leaf has no name")
		}
		ident, _ := machineIdentifier(expression.Value)
		return ident, sysmlPrecPrimary, nil
	case model.MachineKindReference:
		return "", 0, errMachineInState
	case model.MachineKindOperator:
		return translateMachineOperator(expression)
	default:
		return "", 0, fmt.Errorf("expression node kind %q is not a machine expression", expression.Kind)
	}
}

// machineLiteralRenderers render a typed literal by its kind; a literal with
// no value of its kind is not renderable.
var machineLiteralRenderers = map[string]func(model.SemanticValue) (string, bool){
	"integer": func(value model.SemanticValue) (string, bool) {
		if value.Integer == nil {
			return "", false
		}
		return fmt.Sprintf("%d", *value.Integer), true
	},
	"boolean": func(value model.SemanticValue) (string, bool) {
		if value.Boolean == nil {
			return "", false
		}
		return fmt.Sprintf("%t", *value.Boolean), true
	},
	"string": func(value model.SemanticValue) (string, bool) {
		return sysmlString(value.String), value.String != ""
	},
}

// TRLC-LINKS: REQ-EMG-137
func translateMachineLiteral(expression model.SemanticExpression) (string, error) {
	value := expression.TypedValue
	if value == nil {
		return "", errors.New("a literal has no typed value")
	}
	if render, ok := machineLiteralRenderers[value.Kind]; ok {
		if text, ok := render(*value); ok {
			return text, nil
		}
	}
	return "", fmt.Errorf("literal kind %q is not Integer, Boolean or String", value.Kind)
}

// TRLC-LINKS: REQ-EMG-137
func translateMachineOperator(expression model.SemanticExpression) (string, int, error) {
	token, operator, err := lookupMachineSysMLOperator(expression)
	if err != nil {
		return "", 0, err
	}
	precedence := sysmlOperatorPrecedence[expression.Operator]
	texts, err := translateMachineOperands(expression, precedence)
	if err != nil {
		return "", 0, err
	}
	switch {
	case operator.Form == model.MachineFormPrefix && token == "-":
		return token + texts[0], precedence, nil
	case operator.Form == model.MachineFormPrefix:
		return token + " " + texts[0], precedence, nil
	default:
		return texts[0] + " " + token + " " + texts[1], precedence, nil
	}
}

// lookupMachineSysMLOperator returns the SysML token and the machine operator
// of an expression whose operator SysML can express with the right operand count.
//
// TRLC-LINKS: REQ-EMG-137
func lookupMachineSysMLOperator(expression model.SemanticExpression) (string, model.MachineOperator, error) {
	if expression.Operator == "in_state" {
		return "", model.MachineOperator{}, errMachineInState
	}
	token, ok := machineSysMLOperators[expression.Operator]
	if !ok {
		return "", model.MachineOperator{}, fmt.Errorf("operator %q has no SysML v2 expression form", expression.Operator)
	}
	operator, ok := model.LookupMachineOperator(expression.Operator)
	if !ok || len(expression.Operands) != len(operator.Operands) {
		return "", model.MachineOperator{}, fmt.Errorf("operator %q needs %d operands, has %d", expression.Operator, len(operator.Operands), len(expression.Operands))
	}
	return token, operator, nil
}

// translateMachineOperands renders the operands of an operator, in
// parentheses where the operator's precedence asks for them.
//
// TRLC-LINKS: REQ-EMG-137
func translateMachineOperands(expression model.SemanticExpression, precedence int) ([]string, error) {
	site := operandSite{operator: expression.Operator, precedence: precedence}
	texts := make([]string, len(expression.Operands))
	for i, operand := range expression.Operands {
		text, childPrecedence, err := translateMachineNode(operand)
		if err != nil {
			return nil, err
		}
		site.index = i
		if machineOperandNeedsParentheses(site, translatedOperand{node: operand, text: text, precedence: childPrecedence}) {
			text = "(" + text + ")"
		}
		texts[i] = text
	}
	return texts, nil
}

// operandSite is the operator an operand belongs to and the operand's position.
type operandSite struct {
	operator   string
	precedence int
	index      int
}

// translatedOperand is an operand with its SysML text and the precedence of that text.
type translatedOperand struct {
	node       model.SemanticExpression
	text       string
	precedence int
}

// machineOperandNeedsParentheses decides whether an operand of an operator is
// written in parentheses: when it binds looser than the operator, when equal
// precedence would regroup it (comparisons do not associate, implies
// associates right, the others left), and for readability when a not is
// compared or computed with, an operand of a prefix operator is an operator,
// or a negative number follows a binary operator.
//
// TRLC-LINKS: REQ-EMG-137
func machineOperandNeedsParentheses(site operandSite, operand translatedOperand) bool {
	negative := strings.HasPrefix(operand.text, "-")
	if operand.node.Kind != model.MachineKindOperator && !negative {
		return false
	}
	if site.precedence == sysmlPrecUnary {
		return true
	}
	switch {
	case operand.precedence < site.precedence:
		return true
	case operand.precedence == site.precedence:
		return machineEqualPrecedenceRegroups(site)
	default:
		return machineReadabilityParentheses(site, operand.node, negative)
	}
}

// machineEqualPrecedenceRegroups reports whether an operand of the same
// precedence needs parentheses to keep its grouping.
//
// TRLC-LINKS: REQ-EMG-137
func machineEqualPrecedenceRegroups(site operandSite) bool {
	rightAssociative := site.operator == "implies"
	return site.precedence == sysmlPrecComparison || (site.index == 1) != rightAssociative
}

// machineReadabilityParentheses reports whether an operand that binds tighter
// is parenthesized for readability.
//
// TRLC-LINKS: REQ-EMG-137
func machineReadabilityParentheses(site operandSite, operand model.SemanticExpression, negative bool) bool {
	if operand.Operator == "not" && site.precedence >= sysmlPrecComparison {
		return true
	}
	return negative && site.index == 1 && site.precedence == sysmlPrecAdditive
}

// translateMachineEffect renders a machine effect (an "all" node of ":="
// nodes) as one SysML assign statement per assignment.
//
// TRLC-LINKS: REQ-EMG-137
func translateMachineEffect(effect model.SemanticExpression) ([]string, error) {
	if effect.Kind != model.MachineKindOperator || effect.Operator != "all" || len(effect.Operands) == 0 {
		return nil, errors.New("an effect must be a non-empty list of assignments")
	}
	assignments := make([]string, 0, len(effect.Operands))
	for _, assignment := range effect.Operands {
		text, err := translateMachineAssignment(assignment)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, text)
	}
	return assignments, nil
}

// TRLC-LINKS: REQ-EMG-137
func translateMachineAssignment(assignment model.SemanticExpression) (string, error) {
	if assignment.Operator != ":=" || len(assignment.Operands) != 2 || assignment.Operands[0].Kind != model.MachineKindVariable {
		return "", errors.New("an effect element must assign a variable")
	}
	target, _ := machineIdentifier(assignment.Operands[0].Value)
	value, err := translateMachineExpression(assignment.Operands[1])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("assign %s := %s", target, value), nil
}
