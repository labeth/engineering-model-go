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

// renderRepo renders repos/<repo> of a copied fixture to AsciiDoc with the
// given workspace mode ("" finds engmod.work.yml above the repository,
// "off" ignores it). Link findings do not stop the render.
func renderRepo(t *testing.T, root, repo, workspace string) asciidoc.Result {
	t.Helper()
	dir := filepath.Join(root, "repos", repo)
	res, err := asciidoc.GenerateFromFiles(filepath.Join(dir, "engmod.yml"), filepath.Join(dir, "model", "requirements.yml"), filepath.Join(dir, "model", "views.yml"),
		asciidoc.Options{Resolution: asciidocResolution(workspace)})
	if err != nil {
		t.Fatalf("render %s: %v\n%+v", repo, err, res.Diagnostics)
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
		if f := strings.Fields(line); len(f) > 0 && strings.Trim(f[0], "=") == "" && strings.Count(f[0], "=") <= level && strings.HasPrefix(line, "=") {
			return doc[i : i+1+len(heading)+strings.Index(rest, line)]
		}
	}
	return doc[i:]
}

// E5: the system's AsciiDoc has a Requirement Links chapter with a tree per
// system requirement: the requirement, each link with its state and review
// status, the linked child item with its text and its realizing
// requirements with their trace status; the assumptions of use with their
// evidence. The child's AsciiDoc has a Contract Items chapter with each
// item, its realizing requirements and its fingerprint, and no Used By when
// rendered with the workspace off.
// Mutation: dropping the link line, the realizer lines, the assumption
// evidence, the fingerprint or the contract items chapter fails this test.
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-ALLOCATION-TRACE
func TestRequirementLinksAsciiDocTree(t *testing.T) {
	root := copyRequirementLinksExample(t)
	addChildCode(t, filepath.Join(root, "repos", "child-a"), "REQ-A-001", "REQ-A-002", "REQ-A-003")
	doc := renderRepo(t, root, "system", "").Document
	tree := section(doc, "== Requirement Links")
	for _, want := range []string{
		"* *REQ-SYS-001*: When simulation epoch elapses, the position simulator shall publish ownship position on the lab bus.",
		"** link LNK-SYS-001-ENCODER (unchanged, accepted) to `encoder::CAP-A-1`: Publish ownship position sentences at the configured rate.",
		"*** `encoder::REQ-A-001` (implemented, test linked): When epoch period elapses, the position encoder shall publish one position sentence.",
		"*** `bridge::REQ-B-001` (status not available): When a position sentence is received, the lab bus bridge shall forward one bus frame.",
		"* *REQ-SYS-003*: The lab network shall isolate the lab bus from every other network.",
		"** derived: Holds the bridge's assumption of use that the lab bus is isolated.",
		"=== Assumptions of Use",
		"`bridge::AOU-B-1`", "evidence REQ-SYS-003 (link LNK-AOU-B-1, unchanged, accepted)",
	} {
		if !strings.Contains(tree, want) {
			t.Fatalf("system Requirement Links lacks %q:\n%s", want, tree)
		}
	}
	if section(doc, "== Contract Items") != "" {
		t.Fatal("the system publishes no contract items")
	}

	child := renderRepo(t, root, "child-a", "off").Document
	items := section(child, "== Contract Items")
	for _, want := range []string{
		"=== CAP-A-1 (capability)", "Publish ownship position sentences at the configured rate.",
		"* `REQ-A-001`: When epoch period elapses", "* `REQ-A-002`: The sentence encoder shall format",
		"Fingerprint: `sha256:947139d5f7e4609793cb066ed0dd819989d73012d84051e969b1075e8fa4b5ff`",
	} {
		if !strings.Contains(items, want) {
			t.Fatalf("child Contract Items lacks %q:\n%s", want, items)
		}
	}
	if strings.Contains(child, "Used By") || section(child, "== Requirement Links") != "" {
		t.Fatalf("child rendered alone shows users or links:\n%s", items)
	}
}

// The system's tree marks a suspect link after a realizing requirement of
// its item changed in a copy, with the new text; the render still produces
// the document and reports link.suspect.
// Mutation: rendering the suspect link without the marker, or failing the
// render on a link finding, fails this test.
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-ALLOCATION-TRACE
func TestRequirementLinksAsciiDocMarksSuspect(t *testing.T) {
	root := copyRequirementLinksExample(t)
	editFile(t, filepath.Join(root, "repos", "child-a", "model", "requirements.yml"), "shall publish one position sentence.", "shall publish two position sentences.")
	res := renderRepo(t, root, "system", "")
	tree := section(res.Document, "== Requirement Links")
	if !strings.Contains(tree, "** link LNK-SYS-001-ENCODER (*SUSPECT*: the child item changed since review; run engws review-links, accepted) to `encoder::CAP-A-1`") ||
		!strings.Contains(tree, "shall publish two position sentences.") || len(withCode(res.Diagnostics, "link.suspect")) != 1 {
		t.Fatalf("suspect link not marked:\n%s", tree)
	}
	if !strings.Contains(tree, "** link LNK-SYS-002-ENCODER (unchanged, accepted)") {
		t.Fatalf("unchanged link marked:\n%s", tree)
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
	var imported *model.Requirement
	for i, r := range enriched.Requirements.Requirements {
		if r.ID == "encoder::REQ-A-003" {
			imported = &enriched.Requirements.Requirements[i]
		}
	}
	if imported == nil || len(imported.Realizes) != 1 || imported.Realizes[0] != "encoder::CAP-A-2" {
		t.Fatalf("imported REQ-A-003: %+v", imported)
	}
	for _, d := range validate.Bundle(enriched) {
		if d.Severity == validate.SeverityError {
			t.Fatalf("enriched system: %s %s", d.Code, d.Message)
		}
	}
}
