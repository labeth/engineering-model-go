// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/asciidoc"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// fixtureRepo is one repository, repos/<name>, of a copied fixture.
type fixtureRepo struct{ root, name string }

func repo(root, name string) fixtureRepo { return fixtureRepo{root: root, name: name} }

// generate renders the repository to AsciiDoc.
func (r fixtureRepo) generate(opts asciidoc.Options) (asciidoc.Result, error) {
	dir := filepath.Join(r.root, "repos", r.name)
	return asciidoc.GenerateFromFiles(asciidoc.FilePaths{Architecture: filepath.Join(dir, "engmod.yml"), Requirements: filepath.Join(dir, "model", "requirements.yml"), Design: filepath.Join(dir, "model", "views.yml")}, opts)
}

// render renders the repository to AsciiDoc with the given workspace mode (""
// finds engmod.work.yml above the repository, "off" ignores it). Link
// findings do not stop the render.
func (r fixtureRepo) render(t *testing.T, workspace string) asciidoc.Result {
	t.Helper()
	return r.renderWith(t, asciidoc.Options{Resolution: asciidocResolution(workspace)})
}

// renderUsedBy is render with the Used By opt-in.
func (r fixtureRepo) renderUsedBy(t *testing.T, workspace string) asciidoc.Result {
	t.Helper()
	return r.renderWith(t, asciidoc.Options{Resolution: asciidocResolution(workspace), UsedBy: true})
}

func (r fixtureRepo) renderWith(t *testing.T, opts asciidoc.Options) asciidoc.Result {
	t.Helper()
	res, err := r.generate(opts)
	if err != nil {
		t.Fatalf("render %s: %v\n%+v", r.name, err, res.Diagnostics)
	}
	return res
}

// section returns the text from heading up to the next heading of the same
// or a higher level ("" when absent).
func section(doc, heading string) string {
	i := strings.Index(doc, "\n"+heading+"\n")
	if i < 0 {
		return ""
	}
	level := strings.Count(strings.Fields(heading)[0], "=")
	rest := doc[i+1+len(heading):]
	for _, line := range strings.SplitAfter(rest, "\n") {
		if isHeadingAtOrAbove(line, level) {
			return doc[i : i+1+len(heading)+strings.Index(rest, line)]
		}
	}
	return doc[i:]
}

// isHeadingAtOrAbove reports whether line is a heading of the level or a
// higher one.
func isHeadingAtOrAbove(line string, level int) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 || !strings.HasPrefix(line, "=") {
		return false
	}
	return strings.Trim(fields[0], "=") == "" && strings.Count(fields[0], "=") <= level
}

// chapter is a named part of a rendered document.
type chapter struct{ name, text string }

// assertContains fails for each want the chapter does not contain.
func (c chapter) assertContains(t *testing.T, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(c.text, want) {
			t.Fatalf("%s lacks %q:\n%s", c.name, want, c.text)
		}
	}
}

// E5: the system's AsciiDoc has a Requirement Links chapter with a tree per
// system requirement: the requirement, each link with its state and review
// status, the linked child item with its text and its realizing
// requirements with their trace status; the assumptions of use with their
// evidence. The child's AsciiDoc has a Contract Items chapter with each
// item and its realizing requirements (no fingerprint, D23), and no Used By
// when rendered with the workspace off.
// Mutation: dropping the link line, the realizer lines, the assumption
// evidence or the contract items chapter fails this test.
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-ALLOCATION-TRACE
func TestRequirementLinksAsciiDocTree(t *testing.T) {
	root := copyRequirementLinksExample(t)
	addChildCode(t, filepath.Join(root, "repos", "child-a"), "REQ-A-001", "REQ-A-002", "REQ-A-003")
	assertSystemRequirementLinks(t, root)
	assertChildContractItems(t, root)
}

// assertSystemRequirementLinks checks the Requirement Links chapter of the
// system and that the system publishes no contract items.
func assertSystemRequirementLinks(t *testing.T, root string) {
	t.Helper()
	doc := repo(root, "system").render(t, "").Document
	chapter{name: "system Requirement Links", text: section(doc, "== Requirement Links")}.assertContains(t,
		"* *REQ-SYS-001*: When simulation epoch elapses, the position simulator shall publish ownship position on the lab bus.",
		"** link LNK-SYS-001-ENCODER (unchanged, accepted) to `encoder::CAP-A-1`: Publish ownship position sentences at the configured rate.",
		"*** `encoder::REQ-A-001` (implemented, test linked): When epoch period elapses, the position encoder shall publish one position sentence.",
		"*** `bridge::REQ-B-001` (status not available): When a position sentence is received, the lab bus bridge shall forward one bus frame.",
		"* *REQ-SYS-003*: The lab network shall isolate the lab bus from every other network.",
		"** derived: Holds the bridge's assumption of use that the lab bus is isolated.",
		"=== Assumptions of Use",
		"`bridge::AOU-B-1`", "evidence REQ-SYS-003 (link LNK-AOU-B-1, unchanged, accepted)")
	if section(doc, "== Contract Items") != "" {
		t.Fatal("the system publishes no contract items")
	}
}

// assertChildContractItems checks the Contract Items chapter of a child
// rendered alone: items with realizers, no fingerprint, no users, no links.
func assertChildContractItems(t *testing.T, root string) {
	t.Helper()
	child := repo(root, "child-a").render(t, "off").Document
	items := section(child, "== Contract Items")
	chapter{name: "child Contract Items", text: items}.assertContains(t,
		"=== CAP-A-1 (capability)", "Publish ownship position sentences at the configured rate.",
		"* `REQ-A-001`: When epoch period elapses", "* `REQ-A-002`: The sentence encoder shall format")
	if strings.Contains(items, "ingerprint") {
		t.Fatalf("child Contract Items names a fingerprint:\n%s", items)
	}
	if strings.Contains(child, "Used By") || section(child, "== Requirement Links") != "" {
		t.Fatalf("child rendered alone shows users or links:\n%s", items)
	}
}

// The system's tree marks a link reviewed at another version than the
// dependency pin (version mismatch, with both versions) and a claimed
// link; the render still produces the document and reports
// link.version_mismatch.
// Mutation: rendering the stale link without the marker, or failing the
// render on a link finding, fails this test.
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-ALLOCATION-TRACE
func TestRequirementLinksAsciiDocMarksVersionMismatch(t *testing.T) {
	root := copyRequirementLinksExample(t)
	registry := newPublishedChildren(t, root)
	registry.publish("child-a", "v0.1.0")
	registry.publish("child-a", "v0.2.0")
	registry.publish("child-b", "v0.1.0")
	registry.serve()
	pinChild(t, root, childPin{alias: "encoder", version: "v0.2.0"})
	editFile(t, filepath.Join(root, "repos", "system", "model", "req-links.yml"), textEdit{from: "- id: LNK-SYS-002-ENCODER\n  parent: REQ-SYS-002\n  child: encoder::CAP-A-2\n  pinned:\n    version: v0.1.0\n  status: accepted\n", to: "- id: LNK-SYS-002-ENCODER\n  parent: REQ-SYS-002\n  child: encoder::CAP-A-2\n  pinned:\n    version: v0.2.0\n  status: claimed\n"})
	res, err := repo(root, "system").generate(asciidoc.Options{Resolution: composition.ResolutionOptions{Lock: "write"}})
	if err != nil {
		t.Fatalf("render: %v\n%+v", err, res.Diagnostics)
	}
	tree := section(res.Document, "== Requirement Links")
	if !strings.Contains(tree, "** link LNK-SYS-001-ENCODER (*VERSION MISMATCH*: reviewed at v0.1.0, dependency v0.2.0; run engws review-links, accepted) to `encoder::CAP-A-1`") {
		t.Fatalf("stale link not marked:\n%s\n%+v", tree, res.Diagnostics)
	}
	if len(withCode(res.Diagnostics, "link.version_mismatch")) != 1 {
		t.Fatalf("stale link not reported:\n%+v", res.Diagnostics)
	}
	if !strings.Contains(tree, "** link LNK-SYS-002-ENCODER (unchanged, *CLAIMED*: not yet reviewed)") {
		t.Fatalf("claimed link not marked:\n%s", tree)
	}
}

// asciidocResolution is the composition resolution of a render with the
// given workspace mode, lock checked so nothing is written.
func asciidocResolution(workspace string) asciidocResolutionOptions {
	return asciidocResolutionOptions{Workspace: workspace, Lock: "check"}
}

// asciidocResolutionOptions names the resolution options type.
type asciidocResolutionOptions = composition.ResolutionOptions

// A system whose dependencies publish requirements with realizes validates
// with those requirements imported (as the OSCAL assessment results export
// imports them): an imported requirement's realizes names its own model's
// contract items, qualified with the dependency alias, and is not checked
// against the system's contract.
// Mutation: validating imported requirements' realizes against the
// system, or not qualifying the imported realizes, fails this test.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func TestImportedRequirementRealizesIsTheDependencys(t *testing.T) {
	root := copyRequirementLinksExample(t)
	system := filepath.Join(root, "repos", "system", "engmod.yml")
	bundle, err := model.LoadBundle(system)
	if err != nil {
		t.Fatal(err)
	}
	enriched, err := composition.EnrichBundleFromComposition(bundle, "architecture", "assurance", "compliance", "requirements")
	if err != nil {
		t.Fatal(err)
	}
	imported := requirementByID(enriched, "encoder::REQ-A-003")
	if imported == nil || len(imported.Realizes) != 1 || imported.Realizes[0] != "encoder::CAP-A-2" {
		t.Fatalf("imported REQ-A-003: %+v", imported)
	}
	assertNoValidationErrors(t, enriched)
}

// assertNoValidationErrors fails on the first validation error of the bundle.
func assertNoValidationErrors(t *testing.T, bundle model.Bundle) {
	t.Helper()
	for _, d := range validate.Bundle(bundle) {
		if d.Severity == validate.SeverityError {
			t.Fatalf("enriched system: %s %s", d.Code, d.Message)
		}
	}
}

// requirementByID returns the requirement of the bundle with the ID, or nil.
func requirementByID(bundle model.Bundle, id string) *model.Requirement {
	for i, r := range bundle.Requirements.Requirements {
		if r.ID == id {
			return &bundle.Requirements.Requirements[i]
		}
	}
	return nil
}

// The strict-mode requirement link findings do not stop the AsciiDoc render
// (D15): a child with a requirement.no_parent and a
// contract.realizer_unpublished error, and a system with a
// requirement.unsatisfied error, still render, and report the errors; any
// other error still stops it.
// Mutation: blocking the render on requirement.no_parent or
// contract.realizer_unpublished fails this test.
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE
func TestStrictLinkFindingsDoNotStopTheRender(t *testing.T) {
	root := copyRequirementLinksExample(t)
	child := filepath.Join(root, "repos", "child-a")
	editFile(t, filepath.Join(child, "model", "requirements.yml"), textEdit{from: "  realizes: [CAP-A-1]\n- id: REQ-A-002", to: "- id: REQ-A-002"})
	editFile(t, filepath.Join(child, "engmod.yml"), textEdit{from: "  - REQ-A-003\n", to: ""})
	res := repo(root, "child-a").render(t, "off")
	if len(withCode(res.Diagnostics, "requirement.no_parent")) != 1 || len(withCode(res.Diagnostics, "contract.realizer_unpublished")) != 1 || !strings.Contains(res.Document, "== Contract Items") {
		t.Fatalf("child render: %+v", res.Diagnostics)
	}
	system := filepath.Join(root, "repos", "system")
	editFile(t, filepath.Join(system, "model", "requirements.yml"), textEdit{from: "- id: REQ-SYS-003", to: "- id: REQ-SYS-004\n  text: The position simulator shall log each start.\n  verificationMethods: [test]\n  verificationCriteria: A start is logged.\n  appliesTo: [FU-LAB-NETWORK]\n- id: REQ-SYS-003"})
	if res := repo(root, "system").render(t, ""); len(withCode(res.Diagnostics, "requirement.unsatisfied")) != 1 {
		t.Fatalf("system render: %+v", res.Diagnostics)
	}
	editFile(t, filepath.Join(child, "model", "requirements.yml"), textEdit{from: "appliesTo: [FU-ENCODER]", to: "appliesTo: [FU-NOWHERE]"})
	if _, err := repo(root, "child-a").generate(asciidoc.Options{Resolution: asciidocResolution("off")}); err == nil {
		t.Fatal("another validation error must still stop the render")
	}
}
