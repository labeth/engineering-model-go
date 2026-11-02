// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
)

const coffeeExample = "../../examples/coffee-fleet-ota-cloud-sample"

// coffeeCopy copies the coffee fleet sample (system plus three subsystems
// resolved through its engmod.work.yml) into a temporary directory.
func coffeeCopy(t *testing.T) string {
	t.Helper()
	newLinksRegistry(t)
	dir := filepath.Join(t.TempDir(), "coffee")
	copyTree(t, coffeeExample, dir)
	return dir
}

// Acceptance 9: migrate-links converts the three allocations of the coffee
// fleet system into claimed links with the current fingerprints, adds
// documents.reqLinks, removes the allocations (satisfactions stay) and
// prints the requirements that report requirement.no_parent; in each
// subsystem it turns the contract ref into realizes on the referenced
// requirement. A second run changes nothing; --dry-run writes nothing.
// Mutation: dropping the ref conversion, writing an empty fingerprint for a
// resolvable target, keeping the allocations, not listing no_parent or a
// non-idempotent second run fails this test.
// TRLC-LINKS: REQ-EMG-104
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestMigrateLinksCoffeeFleet(t *testing.T) {
	dir := coffeeCopy(t)
	telemetry := filepath.Join(dir, "subsystems", "telemetry")
	telemetryArch := read(t, filepath.Join(telemetry, "model", "architecture.yml"))

	// Subsystems first, as a family migrates: the system then fingerprints
	// the migrated items.
	code, out, errOut := runMain(t, fakeRegistry{}, "migrate-links", "--repo", telemetry, "--dry-run")
	if code != 0 || !strings.Contains(out, "ref CAP-TELEM-REPORT -> REQ-TELEM-001 realizes") || read(t, filepath.Join(telemetry, "model", "architecture.yml")) != telemetryArch {
		t.Fatalf("dry run: %d\n%s\n%s", code, out, errOut)
	}
	for _, sub := range []string{"telemetry", "ota-agent", "cloud-api"} {
		subDir := filepath.Join(dir, "subsystems", sub)
		code, out, errOut := runMain(t, fakeRegistry{}, "migrate-links", "--repo", subDir)
		if code != 0 {
			t.Fatalf("migrate %s: %d\n%s\n%s", sub, code, out, errOut)
		}
		arch := read(t, filepath.Join(subDir, "model", "architecture.yml"))
		if strings.Contains(arch, "ref: REQ-") {
			t.Fatalf("%s keeps a contract ref:\n%s", sub, arch)
		}
	}
	reqs := read(t, filepath.Join(telemetry, "model", "requirements.yml"))
	if !strings.Contains(reqs, "  - FU-TELEM-REPORT\n  realizes: [CAP-TELEM-REPORT]\n- id: REQ-TELEM-002") {
		t.Fatalf("REQ-TELEM-001 does not realize CAP-TELEM-REPORT:\n%s", reqs)
	}
	if got := changedLines(telemetryArch, read(t, filepath.Join(telemetry, "model", "architecture.yml"))); len(got) == 0 {
		t.Fatal("telemetry architecture unchanged")
	}
	if a, b := strings.Count(telemetryArch, "\n"), strings.Count(read(t, filepath.Join(telemetry, "model", "architecture.yml")), "\n"); a-b != 1 {
		t.Fatalf("ref removal changed %d lines, want exactly the ref line", a-b)
	}

	code, out, errOut = runMain(t, fakeRegistry{}, "migrate-links", "--repo", dir)
	if code != 0 {
		t.Fatalf("migrate system: %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{
		"link LNK-COF-001-TELEMETRY REQ-COF-001 -> telemetry::CAP-TELEM-REPORT (v0.1.0, claimed)",
		"link LNK-COF-003-OTA-AGENT REQ-COF-003 -> ota-agent::CAP-OTA-APPLY (v0.1.0, claimed)",
		"link LNK-COF-002-CLOUD-API REQ-COF-002 -> cloud-api::CAP-CLOUD-INGEST (v0.1.0, claimed)",
		"requirement.no_parent: REQ-COF-001", "requirement.no_parent: REQ-COF-012",
		"added documents.reqLinks: model/req-links.yml",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("migrate output lacks %q:\n%s", want, out)
		}
	}
	bundle, err := model.LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Architecture.Composition.Allocations) != 0 || len(bundle.Architecture.Composition.Satisfactions) == 0 {
		t.Fatalf("allocations %d, satisfactions %d", len(bundle.Architecture.Composition.Allocations), len(bundle.Architecture.Composition.Satisfactions))
	}
	if len(bundle.ReqLinks.ReqLinks) != 3 {
		t.Fatalf("links: %+v", bundle.ReqLinks.ReqLinks)
	}
	sub, err := model.LoadBundle(filepath.Join(telemetry, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	want := reqlinks.Fingerprint(reqlinks.PublishedItems(sub, sub.Manifest.Publications...)["CAP-TELEM-REPORT"])
	for _, l := range bundle.ReqLinks.ReqLinks {
		if l.Status != model.ReqLinkClaimed || l.Pinned.Version != "v0.1.0" || !strings.HasPrefix(l.Pinned.Fingerprint, "sha256:") || l.Rationale == "" {
			t.Fatalf("link %+v", l)
		}
		if l.Child == "telemetry::CAP-TELEM-REPORT" && l.Pinned.Fingerprint != want {
			t.Fatalf("telemetry link fingerprint %s, want the migrated item's %s", l.Pinned.Fingerprint, want)
		}
	}
	// The system check now has links, every one evaluates unchanged.
	if errs := checkErrors(t, dir); hasLine(errs, "link.") {
		t.Fatalf("migrated links fail the check:\n%s", strings.Join(errs, "\n"))
	}

	// Idempotent.
	snapshot := map[string]string{}
	for _, f := range []string{"engmod.yml", "model/architecture.yml", "model/req-links.yml", "subsystems/telemetry/model/requirements.yml", "subsystems/telemetry/model/architecture.yml"} {
		snapshot[f] = read(t, filepath.Join(dir, f))
	}
	for _, repo := range []string{dir, telemetry} {
		code, out, errOut = runMain(t, fakeRegistry{}, "migrate-links", "--repo", repo)
		if code != 0 || !strings.Contains(out, "engws: nothing to migrate") {
			t.Fatalf("second run in %s: %d\n%s\n%s", repo, code, out, errOut)
		}
	}
	for f, want := range snapshot {
		if got := read(t, filepath.Join(dir, f)); got != want {
			t.Fatalf("second run changed %s", f)
		}
	}
}

// An allocation whose dependency cannot be resolved is still migrated: the
// link gets an empty fingerprint and the output says so; link IDs stay
// unique when a requirement is allocated twice to one alias; an existing
// link file is extended, not replaced. A ref naming a requirement that
// already realizes the capability adds no duplicate.
// Mutation: skipping an unresolvable allocation, reusing a link ID or
// appending a duplicate realizes entry fails this test.
// TRLC-LINKS: REQ-EMG-104
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestMigrateLinksUnresolvedAndUnique(t *testing.T) {
	newLinksRegistry(t)
	t.Setenv("CUE_REGISTRY", "models.example=127.0.0.1:1+insecure")
	_, system := systemWorkspace(t)
	arch := filepath.Join(system, "model", "architecture.yml")
	replaceIn(t, arch, "      description: Forwards sentences onto the lab bus.\n", `      description: Forwards sentences onto the lab bus.
    allocations:
    - requirement: REQ-SYS-002
      to: SUB-ENCODER
      target: encoder::CAP-A-1
      rationale: The encoder publishes the sentence the checksum protects.
    - requirement: REQ-SYS-002
      to: SUB-ENCODER
      target: encoder::CAP-A-2
`)
	before := read(t, filepath.Join(system, "model", "req-links.yml"))
	code, out, errOut := runMain(t, fakeRegistry{}, "migrate-links", "--repo", system)
	if code != 0 || !strings.Contains(out, "LNK-SYS-002-ENCODER-2") || !strings.Contains(out, "LNK-SYS-002-ENCODER-3") ||
		!strings.Contains(out, "fingerprint left empty") {
		t.Fatalf("migrate: %d\n%s\n%s", code, out, errOut)
	}
	after := read(t, filepath.Join(system, "model", "req-links.yml"))
	if !strings.HasPrefix(after, before) || !strings.Contains(after, "    fingerprint: \"\"\n") {
		t.Fatalf("link file not extended:\n%s", after)
	}
	if strings.Contains(read(t, arch), "allocations:") {
		t.Fatalf("allocations kept:\n%s", read(t, arch))
	}
	var doc model.ReqLinksDocument
	if err := decodeStrict([]byte(after), &doc); err != nil || len(doc.ReqLinks) != 6 {
		t.Fatalf("links: %v %+v", err, doc.ReqLinks)
	}

	child := filepath.Join(t.TempDir(), "child-a")
	copyTree(t, filepath.Join(linksExample, "repos", "child-a"), child)
	replaceIn(t, filepath.Join(child, "model", "architecture.yml"), "      kind: capability\n      note: Protect", "      kind: capability\n      ref: REQ-A-003\n      note: Protect")
	if code, out, errOut := runMain(t, fakeRegistry{}, "migrate-links", "--repo", child); code != 0 || !strings.Contains(out, "ref CAP-A-2 -> REQ-A-003 realizes") {
		t.Fatalf("migrate child: %d\n%s\n%s", code, out, errOut)
	}
	if reqs := read(t, filepath.Join(child, "model", "requirements.yml")); !strings.Contains(reqs, "  realizes: [CAP-A-2]\n- id: REQ-A-004") || strings.Contains(read(t, filepath.Join(child, "model", "architecture.yml")), "ref:") {
		t.Fatalf("existing realizes changed or ref kept:\n%s", reqs)
	}
}
