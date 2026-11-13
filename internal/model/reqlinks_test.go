// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"strings"
	"testing"
)

const reqLinksFixture = `schemaVersion: 2
reqLinks:
  - id: LNK-A
    parent: REQ-A
    child: "dep::CAP-X"
    pinned:
      version: v1.2.4-0.20261102103400-abcdef123456
    status: accepted
    rationale: Dep delivers X.
    reviewedBy: ACT-A
    reviewedOn: "2026-11-02"
  - id: LNK-AOU
    child: "dep::AOU-Y"
    evidence: [REQ-A]
    pinned: {version: v1.0.0, fingerprint: "sha256:1111111111111111111111111111111111111111111111111111111111111111"}
    status: claimed
`

// writeReqLinksFixture writes the loader fixture with requirementLinks,
// documents.reqLinks, a realizing requirement and an assumption of use.
func writeReqLinksFixture(t *testing.T, mode, links string) string {
	t.Helper()
	manifest := strings.Replace(validManifest, "  decisions: model/decisions.yml\n", "  decisions: model/decisions.yml\n  reqLinks: model/req-links.yml\n", 1)
	manifest += "requirementLinks: " + mode + "\n"
	dir := writeV2Fixture(t, manifest)
	writeTestFile(t, filepath.Join(dir, "model", "req-links.yml"), links)
	arch := readTestFile(t, filepath.Join(dir, "model", "architecture.yml"))
	arch += "  contract:\n    provides:\n      - {id: CAP-A, kind: capability, note: A.}\n    assumptions:\n      - id: AOU-A\n        text: The bus is a lab bus.\n        verificationCriteria: The user shows the bus is isolated.\n"
	writeTestFile(t, filepath.Join(dir, "model", "architecture.yml"), arch)
	reqs := readTestFile(t, filepath.Join(dir, "model", "requirements.yml"))
	writeTestFile(t, filepath.Join(dir, "model", "requirements.yml"), reqs+"    realizes: [CAP-A, AOU-A]\n")
	return dir
}

// The manifest's requirementLinks, documents.reqLinks, a requirement's
// realizes and contract.assumptions load strictly; a pseudo-version pin is
// accepted, and pinned.fingerprint (removed, ADR-EMG-075) still loads when
// present, so validation can report model.pinned_fingerprint_removed.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-094
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
func TestLoadBundleReadsRequirementLinkModel(t *testing.T) {
	dir := writeReqLinksFixture(t, "strict", reqLinksFixture)
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if RequirementLinkMode(bundle.Manifest) != "strict" {
		t.Fatalf("requirementLinks = %q", bundle.Manifest.RequirementLinks)
	}
	if got := bundle.Requirements.Requirements[0].Realizes; strings.Join(got, ",") != "CAP-A,AOU-A" {
		t.Fatalf("realizes = %v", got)
	}
	if got := bundle.Architecture.Contract.Assumptions; len(got) != 1 || got[0].ID != "AOU-A" || got[0].VerificationCriteria == "" {
		t.Fatalf("assumptions = %+v", got)
	}
	links := bundle.ReqLinks.ReqLinks
	if len(links) != 2 || links[0].Parent != "REQ-A" || links[0].Pinned.Version != "v1.2.4-0.20261102103400-abcdef123456" ||
		links[1].Evidence[0] != "REQ-A" || links[1].Status != "claimed" || filepath.Base(bundle.ReqLinksPath) != "req-links.yml" ||
		links[0].Pinned.Fingerprint != "" || !strings.HasPrefix(links[1].Pinned.Fingerprint, "sha256:") {
		t.Fatalf("reqLinks = %+v (%s)", links, bundle.ReqLinksPath)
	}
	contract := BuildAuthoringContract(bundle)
	last := contract.Documents[len(contract.Documents)-1]
	if last.Kind != "reqLinks" || last.Required || last.SchemaPath != "internal/model/schema/reqlinks.cue" {
		t.Fatalf("authoring contract reqLinks document = %+v", last)
	}
}

// Without requirementLinks the mode is guided; an unknown value, an unknown
// link field, an unknown link status and a pin without version are rejected
// at load; a removed fingerprint of any form still loads (validation reports it).
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-094
// ENGMODEL-LINKS: FU-MODEL-LOADER
func TestRequirementLinkModelIsStrictlyDecoded(t *testing.T) {
	if RequirementLinkMode(ManifestDocument{}) != "guided" {
		t.Fatal("unset requirementLinks must be guided")
	}
	for name, tc := range map[string]struct{ mode, links, want string }{
		"mode":       {"lenient", reqLinksFixture, "requirementLinks"},
		"field":      {"guided", strings.Replace(reqLinksFixture, "    rationale: Dep delivers X.\n", "    rationale: Dep delivers X.\n    note: extra\n", 1), "note"},
		"status":     {"guided", strings.Replace(reqLinksFixture, "status: claimed", "status: maybe", 1), "status"},
		"no version": {"guided", strings.Replace(reqLinksFixture, `{version: v1.0.0, fingerprint:`, `{fingerprint:`, 1), "version"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := writeReqLinksFixture(t, tc.mode, tc.links)
			_, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want a load error naming %s, got %v", tc.want, err)
			}
		})
	}
	dir := writeReqLinksFixture(t, "guided", strings.Replace(reqLinksFixture, `"sha256:1111111111111111111111111111111111111111111111111111111111111111"`, "abc", 1))
	if _, err := LoadBundle(filepath.Join(dir, "engmod.yml")); err != nil {
		t.Fatalf("a deprecated fingerprint of any form must load: %v", err)
	}
}

// A dependency version may be a Go-style pseudo-version that carries the
// commit; the CUE #ExactVersion and ParseExactModuleVersion both accept it.
// TRLC-LINKS: REQ-EMG-096
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
func TestDependencyVersionAcceptsPseudoVersion(t *testing.T) {
	pseudo := "v1.2.4-0.20261102103400-abcdef123456"
	if _, err := ParseExactModuleVersion("example.com/dep@v1", pseudo); err != nil {
		t.Fatalf("ParseExactModuleVersion rejected a pseudo-version: %v", err)
	}
	manifest := strings.Replace(validManifest, "dependencies: []\n", "dependencies:\n  - alias: dep\n    path: example.com/dep@v1\n    version: "+pseudo+"\n    publications: [public]\n", 1)
	dir := writeV2Fixture(t, manifest)
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatalf("pseudo-version dependency rejected: %v", err)
	}
	if bundle.Manifest.Dependencies[0].Version != pseudo {
		t.Fatalf("version = %q", bundle.Manifest.Dependencies[0].Version)
	}
	broken := strings.Replace(manifest, pseudo, "1.2.4-abc", 1)
	dir = writeV2Fixture(t, broken)
	if _, err := LoadBundle(filepath.Join(dir, "engmod.yml")); err == nil {
		t.Fatal("a version without the v prefix must be rejected")
	}
}
