// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/labeth/engineering-model-go/internal/asciidoc"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

const requirementLinksExample = "examples/requirement-links"

// copyRequirementLinksExample copies the three-repository fixture (with its
// engmod.work.yml, so the system resolves child-a and child-b from the
// sibling directories, offline) into a temporary directory and returns it.
func copyRequirementLinksExample(t *testing.T) string {
	t.Helper()
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, "")
	t.Setenv(composition.ModuleCacheEnv, "")
	copying := exampleCopy{
		src:       requirementLinksExample,
		dst:       t.TempDir(),
		skipsFile: func(file string) bool { return strings.HasPrefix(file, "engmod.lock") },
	}
	if err := filepath.WalkDir(copying.src, copying.visit); err != nil {
		t.Fatal(err)
	}
	return copying.dst
}

// textEdit replaces the text from (which must occur) with the text to.
type textEdit struct{ from, to string }

// editFile applies edit to a fixture file.
func editFile(t *testing.T, path string, edit textEdit) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), edit.from) {
		t.Fatalf("%s does not contain %q", path, edit.from)
	}
	testutil.WriteFile(t, path, strings.Replace(string(data), edit.from, edit.to, 1))
}

// publishedChildren is a registry with releases of the fixture children:
// publish copies repos/<child> of root as it is now at a version. serve
// routes models.example to it and removes root/engmod.work.yml, so the
// system resolves the published releases (D23: links are checked against
// published versions, never a workspace checkout).
type publishedChildren struct {
	t    *testing.T
	root string
	dir  string
}

// newPublishedChildren returns an empty release registry for root.
func newPublishedChildren(t *testing.T, root string) *publishedChildren {
	return &publishedChildren{t: t, root: root, dir: filepath.Join(t.TempDir(), "registry")}
}

// publish copies repos/<child> of root at version into the registry.
func (p *publishedChildren) publish(child, version string) {
	p.t.Helper()
	dst := filepath.Join(p.dir, "models.example_reqlinks_"+child+"_"+version)
	copyDir(p.t, filepath.Join(p.root, "repos", child), dst)
	editFile(p.t, filepath.Join(dst, "engmod.yml"), textEdit{from: "  version: v0.1.0\n", to: "  version: " + version + "\n"})
}

// serve starts the registry and leaves the workspace.
func (p *publishedChildren) serve() {
	p.t.Helper()
	registry, err := modregistrytest.New(os.DirFS(p.dir), "")
	if err != nil {
		p.t.Fatal(err)
	}
	p.t.Cleanup(registry.Close)
	p.t.Setenv("CUE_REGISTRY", "models.example="+registry.Host()+"+insecure")
	if err := os.Remove(filepath.Join(p.root, "engmod.work.yml")); err != nil && !os.IsNotExist(err) {
		p.t.Fatal(err)
	}
}

// childPin is the version a dependency alias (encoder or bridge) is pinned to.
type childPin struct{ alias, version string }

// pinChild sets the system's dependency pin of a child.
func pinChild(t *testing.T, root string, pin childPin) {
	t.Helper()
	child := map[string]string{"encoder": "child-a", "bridge": "child-b"}[pin.alias]
	editFile(t, filepath.Join(root, "repos", "system", "engmod.yml"), textEdit{
		from: "  path: models.example/reqlinks/" + child + "@v0\n  version: v0.1.0\n",
		to:   "  path: models.example/reqlinks/" + child + "@v0\n  version: " + pin.version + "\n",
	})
}

// checkRepository runs the model check of one repository the way
// .github/engmod-check does: generation (validation, EARS lint, composition
// and requirement link diagnostics, trace integrity).
func checkRepository(t *testing.T, dir string) []validate.Diagnostic {
	t.Helper()
	res, err := asciidoc.GenerateFromFiles(asciidoc.FilePaths{Architecture: filepath.Join(dir, "engmod.yml"), Requirements: filepath.Join(dir, "model", "requirements.yml"), Design: filepath.Join(dir, "model", "views.yml")}, asciidoc.Options{})
	if err != nil && !validate.HasErrors(res.Diagnostics) {
		t.Fatalf("check %s: %v", dir, err)
	}
	return res.Diagnostics
}

// linkErrors returns the error diagnostics as "code: message" lines.
func linkErrors(diags []validate.Diagnostic) []string {
	var out []string
	for _, d := range diags {
		if d.Severity == validate.SeverityError {
			out = append(out, d.Code+": "+d.Message)
		}
	}
	return out
}

// withCode returns the diagnostics with the code.
func withCode(diags []validate.Diagnostic, code string) []validate.Diagnostic {
	var out []validate.Diagnostic
	for _, d := range diags {
		if d.Code == code {
			out = append(out, d)
		}
	}
	return out
}

// Acceptance 1: every repository of the strict three-repository fixture
// passes its check with no errors, every child requirement realizes a
// contract item or is derived, every system requirement is linked or derived,
// and every link evaluates unchanged at the pinned versions.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-094, REQ-EMG-097, REQ-EMG-098
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE
func TestRequirementLinksExample(t *testing.T) {
	root := copyRequirementLinksExample(t)
	for _, repo := range []string{"child-a", "child-b", "system"} {
		assertStrictRepositoryPasses(t, root, repo)
	}
	res, err := composition.GenerateCompositionFromFile(filepath.Join(root, "repos", "system", "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	statuses := reqlinks.EvaluateLinks(res.Root.Bundle, composition.LinkDependencies(res.Root))
	if len(statuses) != 4 {
		t.Fatalf("want 4 links, got %+v", statuses)
	}
	for _, s := range statuses {
		if s.State != reqlinks.StateUnchanged {
			t.Fatalf("link %s is %s, want unchanged (%+v)", s.ID, s.State, s)
		}
	}
}

// assertStrictRepositoryPasses checks that a repository of the fixture is in
// strict requirement link mode and passes its check with no errors.
func assertStrictRepositoryPasses(t *testing.T, root, repo string) {
	t.Helper()
	dir := filepath.Join(root, "repos", repo)
	bundle, err := model.LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if model.RequirementLinkMode(bundle.Manifest) != model.RequirementLinksStrict {
		t.Fatalf("%s is not strict", repo)
	}
	if errs := linkErrors(checkRepository(t, dir)); len(errs) > 0 {
		t.Fatalf("%s check errors:\n%s", repo, strings.Join(errs, "\n"))
	}
}

// assertNoLinkErrors fails with the reason when the check of the repository
// at dir reports an error.
func assertNoLinkErrors(t *testing.T, dir, reason string) {
	t.Helper()
	if errs := linkErrors(checkRepository(t, dir)); len(errs) > 0 {
		t.Fatalf("%s:\n%s", reason, strings.Join(errs, "\n"))
	}
}

// Acceptance 2: removing realizes from a child requirement gives a
// requirement.no_parent error in the child's own check, with only the child
// repository present.
// TRLC-LINKS: REQ-EMG-090
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestRequirementLinksChildRuleWithoutSystem(t *testing.T) {
	root := copyRequirementLinksExample(t)
	alone := filepath.Join(t.TempDir(), "child-a")
	if err := os.Rename(filepath.Join(root, "repos", "child-a"), alone); err != nil {
		t.Fatal(err)
	}
	editFile(t, filepath.Join(alone, "model", "requirements.yml"), textEdit{from: "  realizes: [CAP-A-1]\n- id: REQ-A-002", to: "- id: REQ-A-002"})
	got := withCode(checkRepository(t, alone), "requirement.no_parent")
	if len(got) != 1 || got[0].Severity != validate.SeverityError || !strings.Contains(got[0].Message, "REQ-A-001") {
		t.Fatalf("want one requirement.no_parent error for REQ-A-001, got %+v", got)
	}
}

// Acceptance 3 (system side) and 4 (D23): editing a child checkout in the
// workspace never raises a link finding in the system (no churn); a child
// release that changes a realizer's text, with the system pinned to it and
// the link still reviewed at the old version, fails the system check with
// link.version_mismatch naming engws review-links, ID only; a release that
// changes only an unrelated requirement does not change the links' state.
// Mutation: comparing workspace content, or not reporting the stale pin,
// fails this test.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-094
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestRequirementLinksNoChurnAndVersionMismatch(t *testing.T) {
	root := copyRequirementLinksExample(t)
	system := filepath.Join(root, "repos", "system")
	childReqs := filepath.Join(root, "repos", "child-a", "model", "requirements.yml")

	editFile(t, childReqs, textEdit{from: "shall publish one position sentence.", to: "shall publish two position sentences."})
	editFile(t, childReqs, textEdit{from: "the position encoder shall write one log line.", to: "the position encoder shall write one structured log line."})
	assertNoLinkErrors(t, system, "a workspace edit must raise no link finding")

	registry := newPublishedChildren(t, root)
	registry.publish("child-a", "v0.2.0")
	editFile(t, childReqs, textEdit{from: "shall publish two position sentences.", to: "shall publish one position sentence."})
	registry.publish("child-a", "v0.1.0")
	registry.publish("child-b", "v0.1.0")
	registry.serve()
	assertNoLinkErrors(t, system, "published v0.1.0 at its pins")
	pinChild(t, root, childPin{alias: "encoder", version: "v0.2.0"})
	assertVersionMismatchNamesIDsOnly(t, withCode(checkRepository(t, system), "link.version_mismatch"))
}

// assertVersionMismatchNamesIDsOnly checks that got is the link.version_mismatch
// error of both encoder links, naming engws review-links and no requirement
// text.
func assertVersionMismatchNamesIDsOnly(t *testing.T, got []validate.Diagnostic) {
	t.Helper()
	if len(got) != 2 || got[0].Severity != validate.SeverityError {
		t.Fatalf("want link.version_mismatch for both encoder links, ID only, got %+v", got)
	}
	if !strings.Contains(got[0].Message, "engws review-links") || strings.Contains(got[0].Message+got[1].Message, "position sentence") {
		t.Fatalf("want link.version_mismatch for both encoder links, ID only, got %+v", got)
	}
}

// Acceptance 5: a published child release without a linked capability
// gives link.broken in the system check pinned to it; in the workspace the
// same edit raises nothing (a checkout is work in progress).
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestRequirementLinksBrokenWhenCapabilityRemoved(t *testing.T) {
	root := copyRequirementLinksExample(t)
	child := filepath.Join(root, "repos", "child-a")
	editFile(t, filepath.Join(child, "model", "architecture.yml"), textEdit{from: "    - id: CAP-A-2\n      kind: capability\n      note: Protect each position sentence with a checksum.\n", to: ""})
	editFile(t, filepath.Join(child, "engmod.yml"), textEdit{from: "  - CAP-A-2\n", to: ""})
	editFile(t, filepath.Join(child, "model", "requirements.yml"), textEdit{from: "  realizes: [CAP-A-2]\n", to: "  derived: true\n  derivedRationale: Integrity of the encoder output.\n"})
	if got := withCode(checkRepository(t, filepath.Join(root, "repos", "system")), "link.broken"); len(got) != 0 {
		t.Fatalf("a workspace edit raised link.broken: %+v", got)
	}
	registry := newPublishedChildren(t, root)
	registry.publish("child-a", "v0.1.0")
	registry.publish("child-b", "v0.1.0")
	registry.serve()
	got := withCode(checkRepository(t, filepath.Join(root, "repos", "system")), "link.broken")
	if len(got) != 1 || got[0].Severity != validate.SeverityError || !strings.Contains(got[0].Message, "encoder::CAP-A-2") {
		t.Fatalf("want one link.broken error for encoder::CAP-A-2, got %+v", got)
	}
}

// Acceptance 7: a new assumption of use in a child-b release makes the
// system fail with link.aou_unverified until a link with evidence names it.
// TRLC-LINKS: REQ-EMG-091, REQ-EMG-098
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestRequirementLinksNewAssumptionNeedsEvidence(t *testing.T) {
	root := copyRequirementLinksExample(t)
	child := filepath.Join(root, "repos", "child-b")
	system := filepath.Join(root, "repos", "system")
	editFile(t, filepath.Join(child, "model", "architecture.yml"), textEdit{from: "      verificationCriteria: The using system shows the lab bus is isolated from other networks.\n", to: "      verificationCriteria: The using system shows the lab bus is isolated from other networks.\n    - id: AOU-B-2\n      text: Position sentences arrive at most once per epoch.\n"})
	editFile(t, filepath.Join(child, "engmod.yml"), textEdit{from: "  - AOU-B-1\n", to: "  - AOU-B-1\n  - AOU-B-2\n"})
	registry := newPublishedChildren(t, root)
	registry.publish("child-a", "v0.1.0")
	registry.publish("child-b", "v0.1.0")
	registry.serve()
	got := withCode(checkRepository(t, system), "link.aou_unverified")
	if len(got) != 1 || got[0].Severity != validate.SeverityError || !strings.Contains(got[0].Message, "bridge::AOU-B-2") {
		t.Fatalf("want one link.aou_unverified error for bridge::AOU-B-2, got %+v", got)
	}

	links := filepath.Join(system, "model", "req-links.yml")
	data, err := os.ReadFile(links)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, links, string(data)+"- id: LNK-AOU-B-2\n  child: bridge::AOU-B-2\n  evidence:\n  - REQ-SYS-001\n  pinned:\n    version: v0.1.0\n  status: accepted\n  reviewedBy: ACT-SYSTEM-OWNER\n  reviewedOn: \"2026-11-03\"\n")
	assertNoLinkErrors(t, system, "an evidence link must clear link.aou_unverified")
}

// A requirement that realizes two contract items (REQ-B-001 realizes
// CAP-B-1 and AOU-B-1): a child-b release that changes its text makes engws
// bump compare both items as changed, so the capability link and the
// assumption-of-use link both need review (version_mismatch) at the new
// pin. An assumption link may carry a parent next to its evidence. A
// published capability no link names raises nothing in the system.
// Mutation: giving only the first item a requirement realizes its
// realizers, or requiring an assumption link to have evidence only, fails
// this test.
// TRLC-LINKS: REQ-EMG-095, REQ-EMG-097, REQ-EMG-094
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestRequirementLinksMultiTargetAndMixedLinks(t *testing.T) {
	root := copyRequirementLinksExample(t)
	system := filepath.Join(root, "repos", "system")
	links := filepath.Join(system, "model", "req-links.yml")
	editFile(t, links, textEdit{from: "- id: LNK-AOU-B-1\n  child: bridge::AOU-B-1\n", to: "- id: LNK-AOU-B-1\n  parent: REQ-SYS-001\n  child: bridge::AOU-B-1\n"})
	// child-a publishes a capability that no system link names.
	childA := filepath.Join(root, "repos", "child-a")
	editFile(t, filepath.Join(childA, "model", "architecture.yml"), textEdit{from: "      note: Protect each position sentence with a checksum.\n", to: "      note: Protect each position sentence with a checksum.\n    - id: CAP-A-3\n      kind: capability\n      note: Report the encoder version.\n"})
	editFile(t, filepath.Join(childA, "engmod.yml"), textEdit{from: "  - CAP-A-2\n", to: "  - CAP-A-2\n  - CAP-A-3\n"})
	editFile(t, filepath.Join(childA, "model", "requirements.yml"), textEdit{from: "  realizes: [CAP-A-2]\n", to: "  realizes: [CAP-A-2, CAP-A-3]\n"})
	assertNoLinkErrors(t, system, "an unlinked capability or an assumption link with a parent raised")

	before := map[string]reqlinks.Item{}
	childB := filepath.Join(root, "repos", "child-b")
	load := func() map[string]reqlinks.Item {
		b, err := model.LoadBundle(filepath.Join(childB, "engmod.yml"))
		if err != nil {
			t.Fatal(err)
		}
		return reqlinks.PublishedItems(b, b.Manifest.Publications...)
	}
	before = load()
	editFile(t, filepath.Join(childB, "model", "requirements.yml"), textEdit{from: "shall forward one bus frame.", to: "shall forward two bus frames."})
	after := load()
	for _, id := range []string{"CAP-B-1", "AOU-B-1"} {
		if got := reqlinks.MeaningChanges(before[id], after[id]); strings.Join(got, ",") != "REQ-B-001 text" {
			t.Fatalf("%s changes = %v, want REQ-B-001 text", id, got)
		}
	}
}
