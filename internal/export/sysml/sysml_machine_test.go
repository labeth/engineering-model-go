// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package sysml

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labeth/engineering-model-go/internal/model"
)

// machineFixtureBundle is the machine of internal/model/testdata/machine-projection.golden.json
// (three states, two transitions, one invariant) in a minimal model.
//
// TRLC-LINKS: REQ-EMG-137
func machineFixtureBundle(machines ...model.Machine) model.Bundle {
	return model.Bundle{
		Architecture: model.ArchitectureDocument{
			Model: model.ModelMeta{ID: "MODEL-A"},
			AuthoredArchitecture: model.AuthoredArchitecture{
				FunctionalUnits: []model.FunctionalUnit{{ID: "FU-A", Name: "A"}},
				Events:          []model.Event{{ID: "EVT-A", Name: "Requested"}, {ID: "EVT-B", Name: "Retried"}},
				Machines:        machines,
			},
		},
		Requirements: model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-A"}}},
	}
}

// TRLC-LINKS: REQ-EMG-137
func paymentMachine() model.Machine {
	return model.Machine{
		ID: "SM-PAY", Name: "Payment authorization", Description: "Authorizes a payment.",
		Owner: "FU-A", Initial: "STATE-PAY-IDLE",
		Variables: []model.MachineVariable{
			{Name: "attempts", Type: "Integer", Initial: 0},
			{Name: "approved", Type: "Boolean", Initial: false},
			{Name: "reason", Type: "String", Initial: "none"},
		},
		States: []model.MachineState{
			{ID: "STATE-PAY-IDLE", Name: "Idle"},
			{ID: "STATE-PAY-AUTH", Name: "Authorizing"},
			{ID: "STATE-PAY-DONE", Name: "Done", Final: true},
		},
		Transitions: []model.MachineTransition{
			{ID: "TR-PAY-001", From: "STATE-PAY-IDLE", To: "STATE-PAY-AUTH", On: "EVT-A", Guard: "attempts < 3", Effect: "attempts := attempts + 1", Description: "Start"},
			{ID: "TR-PAY-002", From: "STATE-PAY-AUTH", To: "STATE-PAY-DONE", Guard: "approved"},
		},
		Invariants: []model.MachineInvariant{
			{ID: "INV-PAY-001", Requirements: []string{"REQ-A"}, Expr: "attempts <= 3", Description: "Bounded"},
		},
	}
}

// richMachine exercises every operator, a multi-assignment effect, reserved and
// inherited variable names, and the in_state constructs SysML cannot express.
//
// TRLC-LINKS: REQ-EMG-137
func richMachine() model.Machine {
	return model.Machine{
		ID: "SM-RICH", Name: "Rich", Description: "Uses every operator.", Owner: "FU-A", Initial: "STATE-R-A",
		Variables: []model.MachineVariable{
			{Name: "count", Type: "Integer", Initial: -1},
			{Name: "flag", Type: "Boolean", Initial: true},
			{Name: "label", Type: "String", Initial: "ready \"set\""},
			{Name: "state", Type: "Integer", Initial: 0},
			{Name: "done", Type: "Integer", Initial: 0},
		},
		States: []model.MachineState{
			{ID: "STATE-R-A", Name: "A"},
			{ID: "STATE-R-B", Name: "B"},
			{ID: "STATE-R-C", Name: "C", Final: true},
			{ID: "STATE-R-D", Name: "D", Final: true},
		},
		Transitions: []model.MachineTransition{
			{ID: "TR-R-001", From: "STATE-R-A", To: "STATE-R-B", On: "EVT-A", Guard: "count < 3 and not flag", Effect: `count := count + 1; flag := true; label := "go"`},
			{ID: "TR-R-002", From: "STATE-R-B", To: "STATE-R-A", On: "EVT-B", Guard: `(flag or count >= 2) implies label == "go"`, Effect: "state := -count * 2; done := state - -count"},
			{ID: "TR-R-003", From: "STATE-R-B", To: "STATE-R-C", Guard: "count != 3 and count > 0 and count <= 5"},
			{ID: "TR-R-004", From: "STATE-R-B", To: "STATE-R-D", Guard: "in_state(STATE-R-A)", Effect: "flag := in_state(STATE-R-B)", Description: "Needs a state test"},
			{ID: "TR-R-005", From: "STATE-R-A", To: "STATE-R-D"},
		},
		Invariants: []model.MachineInvariant{
			{ID: "INV-R-001", Requirements: []string{"REQ-A"}, Expr: `count >= 0 implies (flag == true or label != "x")`},
			{ID: "INV-R-002", Requirements: []string{"REQ-A"}, Expr: "in_state(STATE-R-C) implies flag", Description: "Final means flagged"},
			{ID: "INV-R-003", Requirements: []string{"REQ-A"}, Expr: "not (count < 0) and (count + 1) * 2 - 1 >= done"},
		},
	}
}

// TRLC-LINKS: REQ-EMG-137
func renderMachines(t *testing.T, machines ...model.Machine) (string, []model.SemanticDiagnostic) {
	t.Helper()
	semantic, diagnostics := model.ProjectSemanticModel(machineFixtureBundle(machines...))
	if semanticDiagnosticsHaveErrors(diagnostics) {
		t.Fatalf("machine fixture does not project: %+v", diagnostics)
	}
	text, rendering := GenerateV2Projection(semantic)
	return text, rendering
}

// TRLC-LINKS: REQ-EMG-137
func compareGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("internal", "export", "sysml", "testdata", name)
	if os.Getenv("ENGMOD_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte(got), want) {
		t.Fatalf("SysML output differs from %s (ENGMOD_UPDATE_GOLDEN=1 rewrites it)\n got:\n%s", path, got)
	}
}

// TRLC-LINKS: REQ-EMG-137
func machineDiagnostics(diagnostics []model.SemanticDiagnostic) []model.SemanticDiagnostic {
	var out []model.SemanticDiagnostic
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == machineCommentedCode {
			out = append(out, diagnostic)
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-137, REQ-EMG-036
func TestGenerateSysMLV2ProjectionRendersMachineGolden(t *testing.T) {
	text, diagnostics := renderMachines(t, paymentMachine())
	compareGolden(t, "machine.golden.sysml", text)
	if len(diagnostics) != 0 {
		t.Fatalf("a fully expressible machine must not report anything: %+v", diagnostics)
	}
	requireContainsAll(t, text, []string{
		"state def 'SM-PAY' {", "attribute attempts : Integer := 0;", `attribute reason : String := "none";`,
		"entry;\n        then 'STATE-PAY-IDLE';", "state 'STATE-PAY-AUTH';",
		"succession 'REL-805FEC22BFCBFF50' first 'STATE-PAY-DONE' then done;",
		"accept 'on-EVT-A' :> 'EVT-A'", "if attempts < 3", "do assign attempts := attempts + 1",
		"constraint 'INV-PAY-001' {", "attempts <= 3",
	})
	requireContainsNone(t, text, []string{`"attempts"`, "'attempts'", " : EngineeringState", "-context'"})
	// The states, transitions and invariant are members of the machine, not
	// top-level siblings.
	requireWrittenOnceInsidePayMachine(t, text,
		[]string{"state 'STATE-PAY-IDLE'", "transition 'TR-PAY-001'", "constraint 'INV-PAY-001'"})
}

// requireWrittenOnceInsidePayMachine fails the test for each member that the
// text does not write exactly once, inside the definition of the payment machine.
//
// TRLC-LINKS: REQ-EMG-137
func requireWrittenOnceInsidePayMachine(t *testing.T, text string, members []string) {
	t.Helper()
	inside := text[strings.Index(text, "state def 'SM-PAY'"):]
	for _, member := range members {
		if strings.Count(text, member) != 1 || !strings.Contains(inside, member) {
			t.Fatalf("%q must be written once inside the machine:\n%s", member, text)
		}
	}
}

// TRLC-LINKS: REQ-EMG-137
func TestGenerateSysMLV2ProjectionRendersRichMachineGolden(t *testing.T) {
	text, diagnostics := renderMachines(t, paymentMachine(), richMachine())
	compareGolden(t, "machine-rich.golden.sysml", text)
	lost := machineDiagnostics(diagnostics)
	if len(lost) != 3 || len(diagnostics) != 3 {
		t.Fatalf("expected exactly the three in_state losses, got %+v", diagnostics)
	}
	flat := strings.Join(strings.Fields(text), " ")
	for _, expected := range []string{
		"attribute done_var : Integer := 0; // authored name: done", "attribute 'state' : Integer := 0;",
		"attribute label : String := \"ready \\\"set\\\"\";", "attribute count : Integer := -1;",
		"do action { // simultaneous assignment: every right-hand side reads the pre-transition values assign count := count + 1; assign flag := true; assign label := \"go\"; }",
		"if count < 3 and not flag",
		"if flag or count >= 2 implies label == \"go\"",
		"assign 'state' := -count * 2", "assign done_var := 'state' - (-count)",
		"if count != 3 and count > 0 and count <= 5",
		"count >= 0 implies flag == true or label != \"x\"",
		"not (count < 0) and (count + 1) * 2 - 1 >= done_var",
	} {
		if !strings.Contains(flat, expected) {
			t.Errorf("missing %q in rich machine output:\n%s", expected, text)
		}
	}
}

// TRLC-LINKS: REQ-EMG-137
func TestGenerateSysMLV2ProjectionCommentsUnexpressibleMachineConstructs(t *testing.T) {
	text, diagnostics := renderMachines(t, richMachine())
	lost := machineDiagnostics(diagnostics)
	requireCommentedMachineWarnings(t, lost)
	requireWarningMentions(t, lost, map[string]string{
		"relationships.TR-R-004": "in_state",
		"elements.INV-R-002":     "in_state(STATE-R-C) implies flag",
	})
	// in_state may appear only inside comments; the transition itself and its
	// trigger structure survive.
	for number, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "in_state") && !strings.HasPrefix(strings.TrimSpace(line), "//") {
			t.Fatalf("line %d writes in_state outside a comment: %q", number+1, line)
		}
	}
	requireContainsAll(t, text, []string{
		"// guard of transition TR-R-004 is not expressible in SysML v2:",
		"//     authored: in_state(STATE-R-A)",
		"// effect of transition TR-R-004 is not expressible in SysML v2:",
		"//     authored: flag := in_state(STATE-R-B)",
		"// invariant INV-R-002 is not expressible in SysML v2:",
		"//     authored: in_state(STATE-R-C) implies flag",
		"transition 'TR-R-004'",
		"then 'STATE-R-D' {",
	})
	requireContainsNone(t, text, []string{"constraint 'INV-R-002'"})
}

// requireCommentedMachineWarnings checks that the lost constructs of the rich
// machine are warnings that name the machine and carry the authored text.
//
// TRLC-LINKS: REQ-EMG-137
func requireCommentedMachineWarnings(t *testing.T, lost []model.SemanticDiagnostic) {
	t.Helper()
	paths := map[string]int{}
	for _, diagnostic := range lost {
		requireWarningNamesMachine(t, diagnostic)
		paths[diagnostic.Path]++
	}
	if paths["relationships.TR-R-004"] != 2 || paths["elements.INV-R-002"] != 1 || len(lost) != 3 {
		t.Fatalf("expected the guard and effect of TR-R-004 and INV-R-002, got %+v", lost)
	}
}

// requireWarningNamesMachine checks that a diagnostic is a warning that names
// the rich machine and carries the authored text.
//
// TRLC-LINKS: REQ-EMG-137
func requireWarningNamesMachine(t *testing.T, diagnostic model.SemanticDiagnostic) {
	t.Helper()
	if diagnostic.Severity != model.SemanticSeverityWarning {
		t.Fatalf("a commented construct is a warning: %+v", diagnostic)
	}
	if !strings.Contains(diagnostic.Message, "machine SM-RICH") || !strings.Contains(diagnostic.Message, "authored:") {
		t.Fatalf("the warning must name the machine and carry the authored text: %+v", diagnostic)
	}
}

// requireWarningMentions checks that a warning at each path mentions its source text.
//
// TRLC-LINKS: REQ-EMG-137
func requireWarningMentions(t *testing.T, lost []model.SemanticDiagnostic, want map[string]string) {
	t.Helper()
	for path, source := range want {
		found := false
		for _, diagnostic := range lost {
			found = found || (diagnostic.Path == path && strings.Contains(diagnostic.Message, source))
		}
		if !found {
			t.Fatalf("no warning at %s mentions %q: %+v", path, source, lost)
		}
	}
}

// breakMachineParts makes four parts of the payment machine unrenderable: a
// variable type, an invariant result, a transition target and a trigger count.
//
// TRLC-LINKS: REQ-EMG-137
func breakMachineParts(semantic *model.SemanticModel) {
	breakMachineElements(semantic.Elements)
	breakMachineRelationships(semantic.Relationships)
}

// TRLC-LINKS: REQ-EMG-137
func breakMachineElements(elements []model.SemanticElement) {
	for i := range elements {
		switch elements[i].ID {
		case "INV-PAY-001":
			elements[i].Features = nil
		case "SM-PAY":
			elements[i].Features[2].Type = "Real"
		}
	}
}

// TRLC-LINKS: REQ-EMG-137
func breakMachineRelationships(relationships []model.SemanticRelationship) {
	for i := range relationships {
		switch relationships[i].ID {
		case "TR-PAY-002":
			relationships[i].Target = "STATE-ELSEWHERE"
		case "TR-PAY-001":
			relationships[i].Triggers = []string{"EVT-A", "EVT-B"}
			relationships[i].Guard = &model.SemanticExpression{Kind: "future"}
		}
	}
}

// TRLC-LINKS: REQ-EMG-137
func TestGenerateSysMLV2ProjectionCommentsMachinePartsWithoutAValidRendering(t *testing.T) {
	semantic, _ := model.ProjectSemanticModel(machineFixtureBundle(paymentMachine()))
	breakMachineParts(&semantic)
	text, diagnostics := GenerateV2Projection(semantic)
	lost := machineDiagnostics(diagnostics)
	paths := map[string]bool{}
	for _, diagnostic := range lost {
		paths[diagnostic.Path] = true
	}
	for _, path := range []string{
		"elements.SM-PAY.features.reason", "elements.INV-PAY-001", "relationships.TR-PAY-002", "relationships.TR-PAY-001",
	} {
		if !paths[path] {
			t.Fatalf("expected a warning at %s, got %+v", path, lost)
		}
	}
	requireContainsAll(t, text, []string{
		"// variable reason is not expressible in SysML v2",
		"// additional trigger EVT-B of transition TR-PAY-001 is not expressible in SysML v2",
		"// transition TR-PAY-002 is not expressible in SysML v2",
		"// invariant INV-PAY-001 is not expressible in SysML v2",
	})
	requireContainsNone(t, text, []string{"transition 'TR-PAY-002'", "constraint 'INV-PAY-001'"})
}

// TRLC-LINKS: REQ-EMG-137
func TestMachineExpressionTranslationTable(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"and", "a and b", "a and b"},
		{"or", "a or b", "a or b"},
		{"implies", "a implies b", "a implies b"},
		{"not", "not a", "not a"},
		{"not of operator", "not (a or b)", "not (a or b)"},
		{"==", "x == 1", "x == 1"},
		{"!=", "x != 1", "x != 1"},
		{"<", "x < 1", "x < 1"},
		{"<=", "x <= 1", "x <= 1"},
		{">", "x > 1", "x > 1"},
		{">=", "x >= 1", "x >= 1"},
		{"+", "x + 1", "x + 1"},
		{"-", "x - 1", "x - 1"},
		{"*", "x * 2", "x * 2"},
		{"neg", "-x", "-x"},
		{"neg of operator", "-(x + 1)", "-(x + 1)"},
		{"neg literal operand", "x - -1", "x - (-1)"},
		{"string equality", `s == "a\"b"`, `s == "a\"b"`},
		{"boolean literals", "true == (a == false)", "true == (a == false)"},
		{"not binds loosely in the source", "not x == 1", "not (x == 1)"},
		{"nested precedence", "a and b or c implies d", "a and b or c implies d"},
		{"left associative subtraction", "x - 1 - 2", "x - 1 - 2"},
		{"regrouped subtraction", "x - (1 - 2)", "x - (1 - 2)"},
		{"right associative implies", "a implies b implies c", "a implies b implies c"},
		{"regrouped implies", "(a implies b) implies c", "(a implies b) implies c"},
		{"comparisons do not associate", "(x < 1) == (y < 2)", "(x < 1) == (y < 2)"},
		{"not under comparison", "(not a) == b", "(not a) == b"},
		{"not under and", "a and not b", "a and not b"},
		{"arithmetic precedence", "x + 1 * 2 - (x + 1) * 2", "x + 1 * 2 - (x + 1) * 2"},
		{"negated product", "-x * 2", "-x * 2"},
		{"reserved variable", "state > 0", "'state' > 0"},
		{"inherited variable", "done == 0", "done_var == 0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expression, problems := model.ParseMachineExpr(c.source, model.MachineExprBoolean)
			if len(problems) != 0 {
				t.Fatalf("%q does not parse: %v", c.source, problems)
			}
			got, err := translateMachineExpression(expression)
			if err != nil || got != c.want {
				t.Fatalf("translate(%q) = %q, %v; want %q", c.source, got, err, c.want)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-137
func TestMachineExpressionTranslationCoversTheOperatorTable(t *testing.T) {
	requireOperatorTableTranslated(t)
	requireUntranslatableExpressionsRefused(t)
	requireEffectTranslation(t)
}

// requireOperatorTableTranslated checks that every operator of the shared
// table has a SysML token, except those that are no expression tokens.
//
// TRLC-LINKS: REQ-EMG-137
func requireOperatorTableTranslated(t *testing.T) {
	t.Helper()
	for _, operator := range model.MachineOperators() {
		if problem := operatorTranslationProblem(operator.Name); problem != "" {
			t.Fatal(problem)
		}
	}
}

// operatorTranslationProblem describes how an operator breaks the rule that
// the table operators are SysML tokens, except in_state, := and all.
//
// TRLC-LINKS: REQ-EMG-137
func operatorTranslationProblem(name string) string {
	_, translated := machineSysMLOperators[name]
	switch name {
	case "in_state", ":=", "all":
		if translated {
			return fmt.Sprintf("operator %q must not be an expression token", name)
		}
	default:
		if !translated {
			return fmt.Sprintf("operator %q of the shared table has no SysML translation; add it or comment it out deliberately", name)
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-137
func requireUntranslatableExpressionsRefused(t *testing.T) {
	t.Helper()
	state, problems := model.ParseMachineExpr("in_state(STATE-X) and a", model.MachineExprBoolean)
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	if text, err := translateMachineExpression(state); err == nil {
		t.Fatalf("in_state must not translate, got %q", text)
	}
	if _, err := translateMachineExpression(model.SemanticExpression{Kind: "operator", Operator: "xor", Operands: []model.SemanticExpression{{}, {}}}); err == nil {
		t.Fatal("an operator outside the table must not translate")
	}
	if _, err := translateMachineExpression(model.SemanticExpression{Kind: "operator", Operator: "and", Operands: []model.SemanticExpression{{}}}); err == nil {
		t.Fatal("a wrong operand count must not translate")
	}
}

// TRLC-LINKS: REQ-EMG-137
func requireEffectTranslation(t *testing.T) {
	t.Helper()
	effect, problems := model.ParseMachineExpr("a := b + 1; c := true", model.MachineExprEffect)
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	assignments, err := translateMachineEffect(effect)
	if err != nil || strings.Join(assignments, "|") != "assign a := b + 1|assign c := true" {
		t.Fatalf("unexpected effect translation %v, %v", assignments, err)
	}
	if _, err := translateMachineEffect(model.SemanticExpression{Kind: "literal"}); err == nil {
		t.Fatal("a non-effect must not translate as an effect")
	}
}

// TRLC-LINKS: REQ-EMG-137
func TestMachineIdentifier(t *testing.T) {
	cases := []struct {
		name, want string
		renamed    bool
	}{
		{"attempts", "attempts", false},
		{"_x1", "_x1", false},
		{"state", "'state'", false},
		{"in", "'in'", false},
		{"done", "done_var", true},
		{"start", "start_var", true},
		{"1abc", "'1abc'", false},
		{"a-b", "'a-b'", false},
		{"it's", "'it''s'", false},
	}
	for _, c := range cases {
		got, renamed := machineIdentifier(c.name)
		if got != c.want || renamed != c.renamed {
			t.Errorf("machineIdentifier(%q) = %q, %v; want %q, %v", c.name, got, renamed, c.want, c.renamed)
		}
	}
	requireReservedWordsQuoted(t)
}

// requireReservedWordsQuoted checks that every reserved word is quoted, so a
// variable can never break the syntax.
//
// TRLC-LINKS: REQ-EMG-137
func requireReservedWordsQuoted(t *testing.T) {
	t.Helper()
	for word := range sysmlReservedWords {
		if got, _ := machineIdentifier(word); !strings.HasPrefix(got, "'") && !machineInheritedNames[word] {
			t.Errorf("reserved word %q was written bare as %q", word, got)
		}
	}
}

// TRLC-LINKS: REQ-EMG-137
func TestMachineCommentTextNeverClosesAComment(t *testing.T) {
	if got := commentText("a */ b\nc\r\nd"); strings.Contains(got, "*/") || strings.Contains(got, "\n") {
		t.Fatalf("unsafe comment text %q", got)
	}
}

// TRLC-LINKS: REQ-EMG-137
func TestGenerateSysMLV2ProjectionIgnoresMachinesWithoutProjection(t *testing.T) {
	// A model with no machine keeps the generic transition rendering.
	semantic := model.SemanticModel{
		ID: "MODEL-A",
		Elements: []model.SemanticElement{
			{ID: "MODEL-A", Kind: model.ElementPackageDefinition},
			{ID: "STATE-A", Kind: model.ElementStateDefinition, Namespace: "MODEL-A"},
			{ID: "STATE-B", Kind: model.ElementStateDefinition, Namespace: "MODEL-A"},
		},
		Relationships: []model.SemanticRelationship{{
			ID: "TRANSITION-A", Kind: model.RelationshipTransition, Source: "STATE-A", Target: "STATE-B",
			Guard: &model.SemanticExpression{Language: "expression", Value: "ready"},
		}},
	}
	text, diagnostics := GenerateV2Projection(semantic)
	if !strings.Contains(text, "state 'TRANSITION-A-context' : EngineeringState {") || len(diagnostics) != 0 {
		t.Fatalf("legacy transition rendering changed: %v\n%s", diagnostics, text)
	}
}

// TestMachineOutputPassesTheOfficialParser runs the pinned official SysML v2
// validator on the generated machine text. It runs only when
// ENGMOD_SYSML_VALIDATE names the validate-sysml executable (it needs Java).
//
// TRLC-LINKS: REQ-EMG-137, REQ-EMG-036
func TestMachineOutputPassesTheOfficialParser(t *testing.T) {
	validator := os.Getenv("ENGMOD_SYSML_VALIDATE")
	if validator == "" {
		t.Skip("ENGMOD_SYSML_VALIDATE is not set; the official SysML parser is not run")
	}
	for name, machines := range map[string][]model.Machine{
		"payment.sysml": {paymentMachine()},
		"rich.sysml":    {paymentMachine(), richMachine()},
	} {
		text, _ := renderMachines(t, machines...)
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		output, err := exec.CommandContext(ctx, validator, path).CombinedOutput()
		cancel()
		if err != nil || strings.Contains(string(output), ": error:") {
			t.Fatalf("official parser rejected %s: %v\n%s\n%s", name, err, filterValidatorOutput(string(output)), text)
		}
	}
}

// filterValidatorOutput drops the library-loading chatter of the validator.
//
// TRLC-LINKS: REQ-EMG-137
func filterValidatorOutput(output string) string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Reading ") || strings.HasPrefix(line, "log4j:") || strings.HasPrefix(line, "Picked up") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
