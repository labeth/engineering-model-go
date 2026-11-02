// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package integration

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	root := t.TempDir()
	err := filepath.WalkDir(requirementLinksExample, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(requirementLinksExample, path)
		if d.IsDir() {
			if d.Name() == "generated" || d.Name() == ".engmod" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(root, rel), 0o755)
		}
		if strings.HasPrefix(d.Name(), "engmod.lock") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// editFile replaces old (which must occur) with new in a fixture file.
func editFile(t *testing.T, path, old, new string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	testutil.WriteFile(t, path, strings.Replace(string(data), old, new, 1))
}

// checkRepository runs the model check of one repository the way
// .github/engmod-check does: generation (validation, EARS lint, composition
// and requirement link diagnostics, trace integrity).
func checkRepository(t *testing.T, dir string) []validate.Diagnostic {
	t.Helper()
	res, err := asciidoc.GenerateFromFiles(filepath.Join(dir, "engmod.yml"), filepath.Join(dir, "model", "requirements.yml"), filepath.Join(dir, "model", "views.yml"), asciidoc.Options{})
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
	editFile(t, filepath.Join(alone, "model", "requirements.yml"), "  realizes: [CAP-A-1]\n- id: REQ-A-002", "- id: REQ-A-002")
	got := withCode(checkRepository(t, alone), "requirement.no_parent")
	if len(got) != 1 || got[0].Severity != validate.SeverityError || !strings.Contains(got[0].Message, "REQ-A-001") {
		t.Fatalf("want one requirement.no_parent error for REQ-A-001, got %+v", got)
	}
}

// Acceptance 3 (system side) and 4: changing the text of a requirement that
// realizes a linked capability makes the link suspect in the system check,
// with the new text in the message; changing a derived requirement that
// realizes nothing leaves every link unchanged.
// TRLC-LINKS: REQ-EMG-095, REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestRequirementLinksSuspectOnlyOnRelevantChange(t *testing.T) {
	root := copyRequirementLinksExample(t)
	system := filepath.Join(root, "repos", "system")
	childReqs := filepath.Join(root, "repos", "child-a", "model", "requirements.yml")

	editFile(t, childReqs, "the position encoder shall write one log line.", "the position encoder shall write one structured log line.")
	if errs := linkErrors(checkRepository(t, system)); len(errs) > 0 {
		t.Fatalf("an unrelated child change must leave every link unchanged:\n%s", strings.Join(errs, "\n"))
	}

	editFile(t, childReqs, "shall publish one position sentence.", "shall publish two position sentences.")
	got := withCode(checkRepository(t, system), "link.suspect")
	if len(got) != 1 || got[0].Severity != validate.SeverityError ||
		!strings.Contains(got[0].Message, "LNK-SYS-001-ENCODER") || !strings.Contains(got[0].Message, "shall publish two position sentences.") {
		t.Fatalf("want one link.suspect error for LNK-SYS-001-ENCODER with the new text, got %+v", got)
	}
}

// Acceptance 5: removing a linked capability from child-a gives link.broken
// in the system check.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestRequirementLinksBrokenWhenCapabilityRemoved(t *testing.T) {
	root := copyRequirementLinksExample(t)
	child := filepath.Join(root, "repos", "child-a")
	editFile(t, filepath.Join(child, "model", "architecture.yml"), "    - id: CAP-A-2\n      kind: capability\n      note: Protect each position sentence with a checksum.\n", "")
	editFile(t, filepath.Join(child, "engmod.yml"), "  - CAP-A-2\n", "")
	editFile(t, filepath.Join(child, "model", "requirements.yml"), "  realizes: [CAP-A-2]\n", "  derived: true\n  derivedRationale: Integrity of the encoder output.\n")
	got := withCode(checkRepository(t, filepath.Join(root, "repos", "system")), "link.broken")
	if len(got) != 1 || got[0].Severity != validate.SeverityError || !strings.Contains(got[0].Message, "encoder::CAP-A-2") {
		t.Fatalf("want one link.broken error for encoder::CAP-A-2, got %+v", got)
	}
}

// Acceptance 7: a new assumption of use in child-b makes the system fail
// with link.aou_unverified until a link with evidence names it.
// TRLC-LINKS: REQ-EMG-091, REQ-EMG-098
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestRequirementLinksNewAssumptionNeedsEvidence(t *testing.T) {
	root := copyRequirementLinksExample(t)
	child := filepath.Join(root, "repos", "child-b")
	system := filepath.Join(root, "repos", "system")
	editFile(t, filepath.Join(child, "model", "architecture.yml"), "      verificationCriteria: The using system shows the lab bus is isolated from other networks.\n",
		"      verificationCriteria: The using system shows the lab bus is isolated from other networks.\n    - id: AOU-B-2\n      text: Position sentences arrive at most once per epoch.\n")
	editFile(t, filepath.Join(child, "engmod.yml"), "  - AOU-B-1\n", "  - AOU-B-1\n  - AOU-B-2\n")
	got := withCode(checkRepository(t, system), "link.aou_unverified")
	if len(got) != 1 || got[0].Severity != validate.SeverityError || !strings.Contains(got[0].Message, "bridge::AOU-B-2") {
		t.Fatalf("want one link.aou_unverified error for bridge::AOU-B-2, got %+v", got)
	}

	fingerprint := reqlinks.Fingerprint(reqlinks.Item{ID: "AOU-B-2", Kind: reqlinks.KindAssumption, Text: "Position sentences arrive at most once per epoch."})
	links := filepath.Join(system, "model", "req-links.yml")
	data, err := os.ReadFile(links)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, links, string(data)+"- id: LNK-AOU-B-2\n  child: bridge::AOU-B-2\n  evidence:\n  - REQ-SYS-001\n  pinned:\n    version: v0.1.0\n    fingerprint: "+fingerprint+"\n  status: accepted\n")
	if errs := linkErrors(checkRepository(t, system)); len(errs) > 0 {
		t.Fatalf("an evidence link must clear link.aou_unverified:\n%s", strings.Join(errs, "\n"))
	}
}
