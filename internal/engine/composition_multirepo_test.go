// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// multiRepoFixture is a workspace with an interface module (publishes FG, FU,
// IF, DO, actor, trust boundary, control and a capability) and a service
// module that composes it and references its published entities.
type multiRepoFixture struct {
	root, iface, service string
}

// serviceDocs are the service's model documents; tests override single ones.
type serviceDocs struct {
	architecture, behavior, assurance, views, requirements, code string
}

// TRLC-LINKS: REQ-EMG-065
func defaultServiceDocs() serviceDocs {
	return serviceDocs{
		architecture: `schemaVersion: 2
architecture:
  functionalGroups:
    - id: FG-SVC
      name: Service
  functionalUnits:
    - id: FU-SVC
      name: Service unit
      group: FG-SVC
  composition:
    subsystems:
      - id: SUB-IF
        dependency: nats
        publication: public
`,
		behavior: `schemaVersion: 2
behavior:
  relationships:
    - {type: publishes, from: FU-SVC, to: nats::IF-WS}
  flows:
    - id: FLOW-SVC-PUB
      title: Publish
      kind: data
      sourceRef: FU-SVC
      destinationRef: nats::IF-WS
      entry: [publish]
      steps:
        - id: publish
          kind: data_move
          ref: FU-SVC
          interfaceRef: nats::IF-WS
          action: Publish
`,
		assurance:    "schemaVersion: 2\nassurance: {}\n",
		views:        "schemaVersion: 2\nviews: []\n",
		requirements: "schemaVersion: 2\nlintRun: {}\nrequirements:\n  - id: REQ-SVC-001\n    text: The service shall publish the state.\n    appliesTo: [FU-SVC, nats::IF-WS]\n",
		code: `package svc

// TRLC-LINKS: REQ-SVC-001
// ENGMODEL-LINKS: nats::IF-WS, nats::DO-WS
func Publish() {}
`,
	}
}

// TRLC-LINKS: REQ-EMG-065
func writeMultiRepoFixture(t *testing.T, docs serviceDocs) multiRepoFixture {
	t.Helper()
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, "")
	root := t.TempDir()
	f := multiRepoFixture{root: root, iface: filepath.Join(root, "iface"), service: filepath.Join(root, "service")}
	testutil.WriteInterfaceModule(t, f.iface)
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 2\nreplacements:\n  - module: example.com/iface@v0\n    path: ./iface\n")
	dir := f.service
	testutil.WriteFile(t, filepath.Join(dir, "cue.mod", "module.cue"), "module: \"example.com/service@v0\"\nlanguage: version: \"v0.17.0\"\n")
	testutil.WriteFile(t, filepath.Join(dir, "engmod.yml"), `schemaVersion: 2
module:
  path: example.com/service@v0
  version: v0.1.0
  modelId: service
  title: Service
  introduction: ""
  kind: system
dependencies:
  - alias: nats
    path: example.com/iface@v0
    version: v0.1.0
    publications: [public]
documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
`)
	testutil.WriteFile(t, filepath.Join(dir, "model", "catalog.yml"), "schemaVersion: 2\ncatalog:\n  systems:\n    - id: SYS-SVC\n      name: service\n      definition: The service.\n      aliases: [the service]\n")
	testutil.WriteFile(t, filepath.Join(dir, "model", "requirements.yml"), docs.requirements)
	testutil.WriteFile(t, filepath.Join(dir, "model", "architecture.yml"), docs.architecture)
	testutil.WriteFile(t, filepath.Join(dir, "model", "behavior.yml"), docs.behavior)
	testutil.WriteFile(t, filepath.Join(dir, "model", "assurance.yml"), docs.assurance)
	testutil.WriteFile(t, filepath.Join(dir, "model", "compliance.yml"), "schemaVersion: 2\ncompliance: {}\n")
	testutil.WriteFile(t, filepath.Join(dir, "model", "views.yml"), docs.views)
	testutil.WriteFile(t, filepath.Join(dir, "model", "decisions.yml"), "schemaVersion: 2\ndecisions: []\n")
	testutil.WriteFile(t, filepath.Join(dir, "src", "svc.go"), docs.code)
	return f
}

// generateService validates the service the way callers do: GenerateAsciiDoc
// on the loaded bundle (canonical projection included).
// TRLC-LINKS: REQ-EMG-065, REQ-EMG-066
func generateService(t *testing.T, f multiRepoFixture) ([]validate.Diagnostic, error) {
	t.Helper()
	bundle, err := model.LoadBundle(filepath.Join(f.service, "engmod.yml"))
	if err != nil {
		t.Fatalf("load service: %v", err)
	}
	res, err := GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, AsciiDocOptions{})
	return res.Diagnostics, err
}

// TRLC-LINKS: REQ-EMG-065
func errorsOf(diags []validate.Diagnostic) []validate.Diagnostic {
	var out []validate.Diagnostic
	for _, d := range diags {
		if d.Severity == validate.SeverityError {
			out = append(out, d)
		}
	}
	return out
}

// TestQualifiedReferencesValidateAgainstPublishedEntities verifies that a
// service referencing the interface's published IF and DO in a relationship,
// a flow and flow step, appliesTo and code ENGMODEL-LINKS validates with 0
// errors, and the trace has no dangling link while counting only its own
// requirements.
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE
func TestQualifiedReferencesValidateAgainstPublishedEntities(t *testing.T) {
	f := writeMultiRepoFixture(t, defaultServiceDocs())
	diags, err := generateService(t, f)
	if err != nil || len(errorsOf(diags)) > 0 {
		t.Fatalf("expected 0 errors, got err=%v %+v", err, errorsOf(diags))
	}
	manifest := filepath.Join(f.service, "engmod.yml")
	matrix, traceDiags, err := BuildTraceMatrixFromFiles(manifest, filepath.Join(f.service, "model", "requirements.yml"), "")
	if err != nil {
		t.Fatal(err)
	}
	if matrix.Summary.DanglingLinks != 0 || len(errorsOf(traceDiags)) > 0 {
		t.Fatalf("dangling=%d errors=%+v", matrix.Summary.DanglingLinks, errorsOf(traceDiags))
	}
	if matrix.Summary.Requirements != 1 || matrix.Summary.Implemented != 1 {
		t.Fatalf("trace counts must stay local: %+v", matrix.Summary)
	}
}

// TestUnpublishedQualifiedReferenceReportedOncePerField verifies that an
// alias::ID the dependency does not publish yields exactly one
// composition.unpublished_reference per field, naming the field path, and no
// generic broken reference; in code it is one dangling link.
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE
func TestUnpublishedQualifiedReferenceReportedOncePerField(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*serviceDocs)
		field string
	}{
		{"relationship", func(d *serviceDocs) {
			d.behavior = strings.Replace(d.behavior, "{type: publishes, from: FU-SVC, to: nats::IF-WS}", "{type: publishes, from: FU-SVC, to: nats::IF-NOPE}", 1)
		}, "relationships[0].to"},
		{"flow step interfaceRef", func(d *serviceDocs) {
			d.behavior = strings.Replace(d.behavior, "interfaceRef: nats::IF-WS", "interfaceRef: nats::IF-NOPE", 1)
		}, "steps[0].interfaceRef"},
		{"appliesTo", func(d *serviceDocs) {
			d.requirements = strings.Replace(d.requirements, "nats::IF-WS]", "nats::IF-NOPE]", 1)
		}, "requirements[0].appliesTo[1]"},
		{"unknown alias", func(d *serviceDocs) {
			d.requirements = strings.Replace(d.requirements, "nats::IF-WS]", "nope::IF-WS]", 1)
		}, "requirements[0].appliesTo[1]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			docs := defaultServiceDocs()
			c.edit(&docs)
			f := writeMultiRepoFixture(t, docs)
			diags, err := generateService(t, f)
			if err == nil {
				t.Fatal("expected validation to fail")
			}
			errs := errorsOf(diags)
			if len(errs) != 1 || errs[0].Code != "composition.unpublished_reference" || !strings.HasSuffix(errs[0].Path, c.field) {
				t.Fatalf("want exactly one composition.unpublished_reference at %s, got %+v", c.field, errs)
			}
		})
	}

	t.Run("code link", func(t *testing.T) {
		docs := defaultServiceDocs()
		docs.code = strings.Replace(docs.code, "nats::DO-WS", "nats::IF-NOPE", 1)
		f := writeMultiRepoFixture(t, docs)
		matrix, _, err := BuildTraceMatrixFromFiles(filepath.Join(f.service, "engmod.yml"), filepath.Join(f.service, "model", "requirements.yml"), "")
		if err != nil {
			t.Fatal(err)
		}
		if matrix.Summary.DanglingLinks != 1 || matrix.DanglingLinks[0].Target != "nats::IF-NOPE" {
			t.Fatalf("want 1 dangling nats::IF-NOPE, got %+v", matrix.DanglingLinks)
		}
	})
}

// TestQualifiedOwnershipAndViewRoots verifies that published dependency
// entities may own local ones (functional unit group, interface owner, trust
// boundary parent, risk owner actor) and be view roots, and that an
// unpublished owner is one composition.unpublished_reference instead of a
// semantic.invalid_ownership error.
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE, FU-VIEW-PROJECTION
func TestQualifiedOwnershipAndViewRoots(t *testing.T) {
	cases := []struct {
		name string
		edit func(*serviceDocs, string)
	}{
		{"functional unit group", func(d *serviceDocs, id string) {
			d.architecture = strings.Replace(d.architecture, "      group: FG-SVC", "      group: nats::"+id, 1)
		}},
		{"interface owner", func(d *serviceDocs, id string) {
			d.architecture = strings.Replace(d.architecture, "  composition:", "  interfaces:\n    - id: IF-ADMIN\n      name: Admin\n      protocol: http\n      endpoint: /admin\n      owner: nats::"+id+"\n  composition:", 1)
		}},
		{"trust boundary parent", func(d *serviceDocs, id string) {
			d.assurance = "schemaVersion: 2\nassurance:\n  trustBoundaries:\n    - id: TB-SVC\n      name: Process\n      description: Process.\n      boundaryType: process\n      parentRef: nats::" + id + "\n      members: [FU-SVC]\n"
		}},
		{"risk owner", func(d *serviceDocs, id string) {
			d.assurance = "schemaVersion: 2\nassurance:\n  risks:\n    - id: RISK-SVC\n      title: Outage\n      statement: Bus unavailable.\n      status: open\n      likelihood: low\n      impact: low\n      response: accept\n      owner: nats::" + id + "\n"
		}},
		{"view root", func(d *serviceDocs, id string) {
			d.views = "schemaVersion: 2\nviews:\n  - id: VIEW-SVC\n    kind: communication\n    roots: [FU-SVC, nats::" + id + "]\n"
		}},
	}
	published := map[string]string{
		"functional unit group": "FG-NATS", "interface owner": "FU-NATS",
		"trust boundary parent": "TB-BUS", "risk owner": "ACT-OWNER", "view root": "IF-WS",
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			docs := defaultServiceDocs()
			c.edit(&docs, published[c.name])
			diags, err := generateService(t, writeMultiRepoFixture(t, docs))
			if err != nil || len(errorsOf(diags)) > 0 {
				t.Fatalf("published owner rejected: err=%v %+v", err, errorsOf(diags))
			}
			docs = defaultServiceDocs()
			c.edit(&docs, "NOPE")
			diags, err = generateService(t, writeMultiRepoFixture(t, docs))
			errs := errorsOf(diags)
			if err == nil || len(errs) != 1 || errs[0].Code != "composition.unpublished_reference" {
				t.Fatalf("want one composition.unpublished_reference, got err=%v %+v", err, errs)
			}
		})
	}
}

// TestLocalModelReportsQualifiedReferenceWithoutComposition verifies that an
// alias::ID in a model that composes nothing (here appliesTo) is reported by
// its own check rather than passing until a parent composes it.
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func TestLocalModelReportsQualifiedReferenceWithoutComposition(t *testing.T) {
	docs := defaultServiceDocs()
	docs.architecture = strings.Replace(docs.architecture, "  composition:\n    subsystems:\n      - id: SUB-IF\n        dependency: nats\n        publication: public\n", "", 1)
	docs.behavior = "schemaVersion: 2\nbehavior: {}\n"
	docs.code = "package svc\n\n// TRLC-LINKS: REQ-SVC-001\nfunc Publish() {}\n"
	diags, err := generateService(t, writeMultiRepoFixture(t, docs))
	errs := errorsOf(diags)
	if err == nil || len(errs) != 1 || errs[0].Code != "composition.unpublished_reference" || !strings.HasSuffix(errs[0].Path, "requirements[0].appliesTo[1]") {
		t.Fatalf("want one unpublished_reference for appliesTo, got err=%v %+v", err, errs)
	}
}

// TestGenerateAsciiDocReturnsCompositionDiagnosticsAndComposesOnce verifies
// that composition findings (an unsatisfied need and a hand-down to an
// unpublished target) are returned in AsciiDocResult.Diagnostics, fail the
// call, and that the call resolves the composition exactly once.
// TRLC-LINKS: REQ-EMG-066
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-SYSTEM-COMPOSITION
func TestGenerateAsciiDocReturnsCompositionDiagnosticsAndComposesOnce(t *testing.T) {
	f := writeMultiRepoFixture(t, defaultServiceDocs())
	// The interface now requires a need, and the service hands down to an
	// unpublished target.
	testutil.WriteFile(t, filepath.Join(f.iface, "model", "architecture.yml"), strings.Replace(mustRead(t, filepath.Join(f.iface, "model", "architecture.yml")), "  contract:\n", "  contract:\n    requires:\n      - id: NEED-CLOCK\n        kind: interface\n", 1))
	testutil.WriteFile(t, filepath.Join(f.iface, "engmod.yml"), strings.Replace(mustRead(t, filepath.Join(f.iface, "engmod.yml")), "CAP-SCHEMA]", "CAP-SCHEMA, NEED-CLOCK]", 1))
	testutil.WriteFile(t, filepath.Join(f.service, "model", "architecture.yml"), defaultServiceDocs().architecture+"    allocations:\n      - requirement: REQ-SVC-001\n        to: SUB-IF\n        target: nats::CAP-INTERNAL\n")

	before := composition.ResolutionCount()
	diags, err := generateService(t, f)
	if runs := composition.ResolutionCount() - before; runs != 1 {
		t.Fatalf("GenerateAsciiDoc composed %d times, want 1", runs)
	}
	if err == nil {
		t.Fatal("composition errors must fail GenerateAsciiDoc")
	}
	found := map[string]validate.Severity{}
	for _, d := range diags {
		found[d.Code] = d.Severity
	}
	if found["composition.allocation_to_internal"] != validate.SeverityError {
		t.Fatalf("hand-down to unpublished target missing: %+v", diags)
	}
	if found["composition.unsatisfied_require"] != validate.SeverityError {
		t.Fatalf("unsatisfied need of the composing system root must be an error: %+v", diags)
	}

	// A passing model composes once too, including the composition chapter.
	g := writeMultiRepoFixture(t, defaultServiceDocs())
	before = composition.ResolutionCount()
	if _, err := generateService(t, g); err != nil {
		t.Fatal(err)
	}
	if runs := composition.ResolutionCount() - before; runs != 1 {
		t.Fatalf("GenerateAsciiDoc composed %d times, want 1", runs)
	}
}

// TestCompositionSeverityRules verifies that an unsatisfied need is an error
// only at the composing root of kind system (a warning below it or for other
// kinds), and that an allocation to a non-contract target is a warning.
// TRLC-LINKS: REQ-EMG-066
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestCompositionSeverityRules(t *testing.T) {
	f := writeMultiRepoFixture(t, defaultServiceDocs())
	testutil.WriteFile(t, filepath.Join(f.iface, "model", "architecture.yml"), strings.Replace(mustRead(t, filepath.Join(f.iface, "model", "architecture.yml")), "  contract:\n", "  contract:\n    requires:\n      - id: NEED-CLOCK\n        kind: interface\n", 1))
	testutil.WriteFile(t, filepath.Join(f.iface, "engmod.yml"), strings.Replace(mustRead(t, filepath.Join(f.iface, "engmod.yml")), "CAP-SCHEMA]", "CAP-SCHEMA, NEED-CLOCK]", 1))
	testutil.WriteFile(t, filepath.Join(f.service, "model", "architecture.yml"), defaultServiceDocs().architecture+"    allocations:\n      - requirement: REQ-SVC-001\n        to: SUB-IF\n        target: nats::FU-NATS\n")
	manifest := filepath.Join(f.service, "engmod.yml")

	severity := func(res composition.CompositionResult, code string) validate.Severity {
		for _, d := range res.Diagnostics {
			if d.Code == code {
				return d.Severity
			}
		}
		return ""
	}
	res, err := composition.GenerateCompositionFromFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got := severity(res, "composition.unsatisfied_require"); got != validate.SeverityError {
		t.Fatalf("system root: unsatisfied_require severity %q", got)
	}
	if got := severity(res, "composition.allocation_target_not_contract"); got != validate.SeverityWarning {
		t.Fatalf("non-contract allocation target severity %q", got)
	}

	testutil.WriteFile(t, manifest, strings.Replace(mustRead(t, manifest), "kind: system", "kind: software", 1))
	if res, err = composition.GenerateCompositionFromFile(manifest); err != nil {
		t.Fatal(err)
	}
	if got := severity(res, "composition.unsatisfied_require"); got != validate.SeverityWarning {
		t.Fatalf("non-system root: unsatisfied_require severity %q", got)
	}
}

// TestDelegatedRowShowsTargetRepositoryStatus verifies that a delegated
// requirement row reports the target requirement's own status in the
// dependency, and that removing the dependency's test link changes it.
// TRLC-LINKS: REQ-EMG-067
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func TestDelegatedRowShowsTargetRepositoryStatus(t *testing.T) {
	f := writeMultiRepoFixture(t, defaultServiceDocs())
	// The interface has code and a test for its requirement in its module directory.
	testutil.WriteFile(t, filepath.Join(f.iface, "src", "iface.go"), "package iface\n\n// TRLC-LINKS: REQ-IF-001\nfunc Encode() {}\n")
	testFile := filepath.Join(f.iface, "src", "iface_test.go")
	testutil.WriteFile(t, testFile, "package iface\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-IF-001\nfunc TestEncode(t *testing.T) { Encode() }\n")
	testutil.WriteFile(t, filepath.Join(f.service, "model", "architecture.yml"), defaultServiceDocs().architecture+"    allocations:\n      - requirement: REQ-SVC-001\n        to: SUB-IF\n        target: nats::CAP-SCHEMA\n")
	manifest := filepath.Join(f.service, "engmod.yml")
	requirements := filepath.Join(f.service, "model", "requirements.yml")

	row := func() TraceRow {
		t.Helper()
		matrix, _, err := BuildTraceMatrixFromFiles(manifest, requirements, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range matrix.Requirements {
			if r.ID == "REQ-SVC-001" {
				return r
			}
		}
		t.Fatal("row REQ-SVC-001 missing")
		return TraceRow{}
	}
	got := row()
	if got.DelegatedStatus == nil || got.DelegatedStatus.Requirement != "REQ-IF-001" || !got.DelegatedStatus.Implemented || !got.DelegatedStatus.TestLinked || got.DelegatedStatus.Status != "implemented" {
		t.Fatalf("delegated status: %+v", got.DelegatedStatus)
	}

	if err := os.Remove(testFile); err != nil {
		t.Fatal(err)
	}
	if got = row(); got.DelegatedStatus == nil || got.DelegatedStatus.TestLinked || !got.DelegatedStatus.Implemented {
		t.Fatalf("removing the dependency's test link did not show: %+v", got.DelegatedStatus)
	}

	// A dependency whose module directory holds no code reports not-available.
	if err := os.RemoveAll(filepath.Join(f.iface, "src")); err != nil {
		t.Fatal(err)
	}
	if got = row(); got.DelegatedStatus == nil || got.DelegatedStatus.Status != "not-available" {
		t.Fatalf("no code: %+v", got.DelegatedStatus)
	}
}

// TestDelegatedStatusScansOnlyTheDependencyModule verifies that a dependency's
// trace is computed from its own module directory: with the workspace inside
// one git repository, code elsewhere in that repository that links the
// dependency's requirement does not count for the dependency.
// Mutation: not setting the dependency bundle's CodeScopeRoot in
// resolveSystem fails this test.
// TRLC-LINKS: REQ-EMG-067, REQ-EMG-057
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION, FU-CODEMAP-INFERENCE
func TestDelegatedStatusScansOnlyTheDependencyModule(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	f := writeMultiRepoFixture(t, defaultServiceDocs())
	if out, err := exec.Command("git", "-C", f.root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	testutil.WriteFile(t, filepath.Join(f.root, "other", "elsewhere.go"), "package other\n\n// TRLC-LINKS: REQ-IF-001\nfunc Elsewhere() {}\n")
	testutil.WriteFile(t, filepath.Join(f.service, "model", "architecture.yml"), defaultServiceDocs().architecture+"    allocations:\n      - requirement: REQ-SVC-001\n        to: SUB-IF\n        target: nats::CAP-SCHEMA\n")
	matrix, _, err := BuildTraceMatrixFromFiles(filepath.Join(f.service, "engmod.yml"), filepath.Join(f.service, "model", "requirements.yml"), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range matrix.Requirements {
		if r.ID == "REQ-SVC-001" {
			if r.DelegatedStatus == nil || r.DelegatedStatus.Status != "not-available" || r.DelegatedStatus.Implemented {
				t.Fatalf("code outside the dependency module counted for it: %+v", r.DelegatedStatus)
			}
			return
		}
	}
	t.Fatal("row REQ-SVC-001 missing")
}

// TRLC-LINKS: REQ-EMG-066, REQ-EMG-067
func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Only reference-typed fields are checked as alias::ID: a model without
// dependencies whose prose (a description, requirement notes, rationale,
// a C++ snippet, a flow step action) contains "::" validates clean, and a
// "::" in a free-text field ending in "ref" (an href or schemaRef URL) is
// not taken for a qualified reference.
// Mutation: checking every string field instead of reference fields, or
// treating a "::" in a URL-valued *Ref field as a qualified reference,
// fails this test.
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func TestProseWithDoubleColonIsNotAQualifiedReference(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, "")
	dir := t.TempDir()
	testutil.WriteV2ModuleFiles(t, dir, "example.com/solo@v0", "v0.1.0", "SOLO", "", `  functionalGroups:
    - id: FG-CORE
      name: Core
      description: "uses std::vector and a::b"
  functionalUnits:
    - id: FU-CORE
      name: Core unit
      group: FG-CORE
      prose: "calls Foo::bar() via std::map"
  interfaces:
    - id: IF-CORE
      name: Core API
      protocol: https
      endpoint: "https://[fe80::1]:8443/api"
      schemaRef: "schemas::v1.json"
      owner: FU-CORE
`)
	testutil.WriteFile(t, filepath.Join(dir, "model", "requirements.yml"), `schemaVersion: 2
lintRun: {}
requirements:
  - id: REQ-SOLO-001
    text: The solo system shall store values.
    notes: "Implementation uses std::vector; see Foo::bar and a::b::c."
    rationale: "C++ scope resolution: ns::name."
    appliesTo: [FU-CORE]
`)
	bundle, err := model.LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if diags := composition.QualifiedReferenceDiagnostics(bundle, composition.CompositionResult{}); len(diags) != 0 {
		t.Fatalf("prose with :: reported as qualified references: %+v", diags)
	}
	res, _ := GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, AsciiDocOptions{})
	for _, d := range res.Diagnostics {
		if strings.HasPrefix(d.Code, "composition.") || strings.Contains(d.Message, "::") {
			t.Fatalf("prose with :: produced %+v", d)
		}
	}
}
