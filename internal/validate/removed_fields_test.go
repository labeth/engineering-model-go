// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// removedFieldCase is one model edit and the diagnostics of the removed-field
// codes it must produce.
type removedFieldCase struct {
	name  string
	edit  func(b *model.Bundle)
	code  string
	paths []string // the paths of the code, in order; nil means none
}

// fingerprintPin is a pin that still carries a fingerprint.
var fingerprintPin = model.ReqLinkPin{Version: "v0.2.0", Fingerprint: "sha256:v1:aaaa"}

var removedFieldCases = []removedFieldCase{
	{"one link with a fingerprint", func(b *model.Bundle) {
		b.ReqLinks.ReqLinks = []model.ReqLink{
			{ID: "LNK-OLD", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: fingerprintPin, Status: "claimed"},
			{ID: "LNK-NEW", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: linkPin, Status: "claimed"},
		}
	}, codePinnedFingerprintRemoved, []string{"reqLinks[0].pinned.fingerprint"}},
	{"two links with a fingerprint", func(b *model.Bundle) {
		b.ReqLinks.ReqLinks = []model.ReqLink{
			{ID: "LNK-ONE", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: fingerprintPin, Status: "claimed"},
			{ID: "LNK-TWO", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: fingerprintPin, Status: "claimed"},
		}
	}, codePinnedFingerprintRemoved, []string{"reqLinks[0].pinned.fingerprint", "reqLinks[1].pinned.fingerprint"}},
	{"links without a fingerprint", func(b *model.Bundle) {
		b.ReqLinks.ReqLinks = []model.ReqLink{{ID: "LNK-NEW", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: linkPin, Status: "claimed"}}
	}, codePinnedFingerprintRemoved, nil},
	{"a provides entry with a ref", func(b *model.Bundle) {
		b.Architecture.Contract.Provides = append(b.Architecture.Contract.Provides,
			model.ContractEntry{ID: "CAP-C", Kind: "capability", Ref: "REQ-S", Note: "C."})
	}, codeProvidesRefRemoved, []string{"contract.provides[1].ref"}},
	{"a provides entry of kind interface with a ref", func(b *model.Bundle) {
		b.Architecture.Contract.Provides = append(b.Architecture.Contract.Provides,
			model.ContractEntry{ID: "IF-X", Kind: "interface", Ref: "REQ-S"})
	}, codeProvidesRefRemoved, []string{"contract.provides[1].ref"}},
	{"a provides entry without a ref", func(b *model.Bundle) {
		b.Architecture.Contract.Provides = append(b.Architecture.Contract.Provides,
			model.ContractEntry{ID: "CAP-C", Kind: "capability", Note: "C."})
	}, codeProvidesRefRemoved, nil},
	{"an imported provides entry with a ref", func(b *model.Bundle) {
		b.Architecture.Contract.Provides = append(b.Architecture.Contract.Provides,
			model.ContractEntry{ID: "dep::CAP-Z", Kind: "capability", Ref: "dep::REQ-Z"})
	}, codeProvidesRefRemoved, nil},
	{"a requires entry with a ref", func(b *model.Bundle) {
		b.Architecture.Contract.Requires = []model.ContractEntry{{ID: "NEED-X", Kind: "capability", Ref: "REQ-S"}}
	}, codeProvidesRefRemoved, nil},
	{"a manifest with allocations", func(b *model.Bundle) {
		b.Architecture.Composition.Allocations = []model.Allocation{
			{Requirement: "REQ-A", To: "SUB-DEP", Target: "dep::CAP-X"},
			{Requirement: "REQ-B", To: "SUB-DEP", Target: "dep::CAP-Y"},
		}
	}, codeAllocationsRemoved, []string{"composition.allocations"}},
	{"a manifest without allocations", func(b *model.Bundle) {
		b.Architecture.Composition.Allocations = nil
	}, codeAllocationsRemoved, nil},
}

// A link that carries pinned.fingerprint is model.pinned_fingerprint_removed
// and a local contract.provides entry that carries ref is
// model.provides_ref_removed: each an undeferred error at the field with its
// remedy, also with a lifecycle tier and an open POA&M item for the code. A
// link without a fingerprint, a provides entry without a ref, an imported
// entry and a requires entry with a ref are not reported (REQ-EMG-184). A
// manifest that carries composition.allocations is model.allocations_removed,
// once, at composition.allocations; one without them is not (REQ-EMG-187).
// Mutation: dropping any emitter, reporting it as a warning, reading the
// requires list or the imported entries, or an empty remedy fails a row;
// deleting the append of allocationsRemovedDiagnostics in
// RemovedFieldDiagnostics fails the allocations row.
// TRLC-LINKS: REQ-EMG-184, REQ-EMG-187
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestRemovedFieldDiagnostics(t *testing.T) {
	for _, c := range removedFieldCases {
		t.Run(c.name, func(t *testing.T) {
			b := linkFileBundle()
			c.edit(&b)
			got := codesOf(ApplyTierPolicy(b, Bundle(b)), c.code)
			assertRemovedPaths(t, got, c.paths)
			for _, d := range got {
				assertRemovedField(t, d)
			}
		})
	}
}

// assertRemovedPaths checks the paths of the diagnostics in order.
// TRLC-LINKS: REQ-EMG-184
func assertRemovedPaths(t *testing.T, got []Diagnostic, want []string) {
	t.Helper()
	var paths []string
	for _, d := range got {
		paths = append(paths, d.Path)
	}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("paths %v, want %v", paths, want)
	}
}

// assertRemovedField checks one diagnostic is an undeferred error that names
// the field and carries the remedy of its code.
// TRLC-LINKS: REQ-EMG-184
func assertRemovedField(t *testing.T, d Diagnostic) {
	t.Helper()
	assertUntieredError(t, d)
	if d.Remedy == "" || d.Remedy != ConsistencyRemedy(d.Code) {
		t.Errorf("%s: remedy %q, want %q", d.Code, d.Remedy, ConsistencyRemedy(d.Code))
	}
	if !strings.Contains(d.Message, "no longer supported") {
		t.Errorf("%s: message %q does not say the field is no longer supported", d.Code, d.Message)
	}
}

// assertUntieredError checks that a diagnostic is an error that no tier and
// no POA&M item changed.
// TRLC-LINKS: REQ-EMG-184
func assertUntieredError(t *testing.T, d Diagnostic) {
	t.Helper()
	if d.Severity != SeverityError || d.Deferred {
		t.Errorf("%s must be an undeferred error, got %+v", d.Code, d)
	}
	if d.PoamRef != "" || d.RequiredFrom != "" {
		t.Errorf("%s must be untiered, got %+v", d.Code, d)
	}
}

// The remedies of the removed-field codes are the words authors act on, pinned
// literally; the codes are not tier keyed and not core, so no tier changes
// them (an untiered error) and no other code has these remedies.
// Mutation: changing a remedy text (for example the allocations remedy to
// "write links"), keying a code to a tier or listing it as a core code fails
// this test.
// TRLC-LINKS: REQ-EMG-184, REQ-EMG-187
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestRemovedFieldRemedies(t *testing.T) {
	want := map[string]string{
		codePinnedFingerprintRemoved: "delete pinned.fingerprint; pin the dependency version only",
		codeProvidesRefRemoved:       "name the realizing requirement with realizes, not ref",
		codeAllocationsRemoved:       "write requirement links (reqLinks) instead of composition.allocations",
	}
	for code, remedy := range want {
		if got := ConsistencyRemedy(code); got != remedy {
			t.Errorf("%s: remedy %q, want %q", code, got, remedy)
		}
		if TierKeyed(code) || IsCoreCode(code) {
			t.Errorf("%s must be an untiered non-core code", code)
		}
	}
	if got := ConsistencyRemedy("link.fingerprint_deprecated"); got != "" {
		t.Errorf("the retired warning has no remedy, got %q", got)
	}
}

// The authoring rules name the removed-field codes and not the retired
// deprecation warnings they replace.
// Mutation: putting a retired warning (composition.allocation_deprecated) back
// into the rules or dropping a code fails this test.
// TRLC-LINKS: REQ-EMG-184, REQ-EMG-187
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func TestAuthoringRulesNameTheRemovedFieldCodes(t *testing.T) {
	text := strings.Join(AuthoringRules().FieldRules, "\n")
	for code, named := range map[string]bool{
		codePinnedFingerprintRemoved: true, codeProvidesRefRemoved: true, codeAllocationsRemoved: true,
		"link.fingerprint_deprecated": false, "contract.ref_deprecated": false, "composition.allocation_deprecated": false,
	} {
		if strings.Contains(text, code) != named {
			t.Errorf("authoring rules name %s: want %v", code, named)
		}
	}
}

// The README names every removed-field code with its remedy.
// Mutation: removing a row from the README fails this test.
// TRLC-LINKS: REQ-EMG-184, REQ-EMG-187
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestRemovedFieldCodesAreInTheReadme(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{codePinnedFingerprintRemoved, codeProvidesRefRemoved, codeAllocationsRemoved} {
		row := "| `" + code + "` | " + ConsistencyRemedy(code) + " |"
		if !strings.Contains(string(data), row) {
			t.Errorf("README.md has no row %q", row)
		}
	}
}

// exampleManifests returns every engmod.yml below the examples directory,
// outside the generated outputs.
// TRLC-LINKS: REQ-EMG-184
func exampleManifests(t *testing.T) []string {
	t.Helper()
	found, err := walkManifests(filepath.Join("..", "..", "examples"))
	if err != nil || len(found) < 15 {
		t.Fatalf("examples: %d manifests, %v", len(found), err)
	}
	return found
}

// walkManifests lists every engmod.yml below root, skipping generated trees.
// TRLC-LINKS: REQ-EMG-184
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func walkManifests(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "generated" {
			return filepath.SkipDir
		}
		if d.Name() == "engmod.yml" {
			found = append(found, path)
		}
		return nil
	})
	return found, err
}

// No example carries a removed field: no link with pinned.fingerprint, no ref
// on a provides entry, no ref on a requires entry, which nothing reads (the
// two Atlas examples had one, ADR-EMG-075), and no composition.allocations.
// Mutation: putting a ref line back on an Atlas requires entry, or a
// fingerprint, provides ref or allocation into any example, fails this test.
// TRLC-LINKS: REQ-EMG-184, REQ-EMG-187
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestExamplesCarryNoRemovedFields(t *testing.T) {
	for _, manifest := range exampleManifests(t) {
		t.Run(filepath.ToSlash(manifest), func(t *testing.T) { assertNoRemovedFields(t, manifest) })
	}
}

// assertNoRemovedFields loads one example and fails on a removed field or a requires ref.
// TRLC-LINKS: REQ-EMG-184
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func assertNoRemovedFields(t *testing.T, manifest string) {
	t.Helper()
	b, err := model.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got := RemovedFieldDiagnostics(b); len(got) != 0 {
		t.Errorf("removed fields: %+v", got)
	}
	for _, entry := range b.Architecture.Contract.Requires {
		if entry.Ref != "" {
			t.Errorf("requires entry %s carries ref %s", entry.ID, entry.Ref)
		}
	}
}
