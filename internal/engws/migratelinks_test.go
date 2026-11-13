// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

const coffeeExample = "../../examples/coffee-fleet-ota-cloud-sample"

// coffeeCopy copies the coffee fleet sample (system plus three subsystems
// resolved through its engmod.work.yml) into a temporary directory and
// turns the system back into its pre-migration form: allocations, no
// realizes, no link file. The subsystems stay as they are.
func coffeeCopy(t *testing.T) string {
	t.Helper()
	newLinksRegistry(t)
	dir := filepath.Join(t.TempDir(), "coffee")
	copyTree(t, coffeeExample, dir)
	replaceIn(t, filepath.Join(dir, "engmod.yml"), "  reqLinks: model/req-links.yml\n", "")
	sysReqs := filepath.Join(dir, "model", "requirements.yml")
	write(t, sysReqs, regexp.MustCompile(`(?m)^  realizes: \[CAP-COFFEE-FLEET-[A-Z]+\]\n`).ReplaceAllString(read(t, sysReqs), ""))
	if err := os.Remove(filepath.Join(dir, "model", "req-links.yml")); err != nil {
		t.Fatal(err)
	}
	replaceIn(t, filepath.Join(dir, "model", "architecture.yml"), "    satisfactions:\n", `    allocations:
    - requirement: REQ-COF-001
      to: SUB-TELEMETRY
      target: telemetry::CAP-TELEM-REPORT
      rationale: Machine telemetry reporting is realized by the telemetry subsystem.
    - requirement: REQ-COF-003
      to: SUB-OTA-AGENT
      target: ota-agent::CAP-OTA-APPLY
      rationale: Signed firmware distribution and apply is realized by the OTA agent subsystem.
    - requirement: REQ-COF-002
      to: SUB-CLOUD-API
      target: cloud-api::CAP-CLOUD-INGEST
      rationale: Telemetry persistence is realized by the cloud API subsystem.
    satisfactions:
`)
	return dir
}

// Acceptance 9: migrate-links converts the three allocations of the coffee
// fleet system into claimed links pinned to the dependency version (no
// fingerprint, D23), adds
// documents.reqLinks, removes the allocations (satisfactions stay) and
// prints the requirements that report requirement.no_parent (not the link
// parents, which trace upward through their links); a subsystem with nothing
// to migrate is left alone. A second run changes nothing; --dry-run writes
// nothing.
// Mutation: writing a fingerprint, keeping the allocations, not listing
// no_parent or a non-idempotent second run fails this test.
// TRLC-LINKS: REQ-EMG-104
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestMigrateLinksCoffeeFleet(t *testing.T) {
	dir := coffeeCopy(t)
	telemetry := filepath.Join(dir, "subsystems", "telemetry")

	// Subsystems first, as a family migrates: they have nothing to migrate.
	for _, sub := range []string{"telemetry", "ota-agent", "cloud-api"} {
		subDir := filepath.Join(dir, "subsystems", sub)
		before := read(t, filepath.Join(subDir, "model", "architecture.yml"))
		code, out, errOut := runMain(t, fakeRegistry{}, "migrate-links", "--repo", subDir)
		if code != 0 || !strings.Contains(out, "engws: nothing to migrate") || read(t, filepath.Join(subDir, "model", "architecture.yml")) != before {
			t.Fatalf("migrate %s: %d\n%s\n%s", sub, code, out, errOut)
		}
	}

	code, out, errOut := runMain(t, fakeRegistry{}, "migrate-links", "--repo", dir, "--dry-run")
	if _, err := os.Stat(filepath.Join(dir, "model", "req-links.yml")); code != 0 || !strings.Contains(out, "dry run: link LNK-COF-001-TELEMETRY") || !os.IsNotExist(err) {
		t.Fatalf("dry run: %d %v\n%s\n%s", code, err, out, errOut)
	}
	code, out, errOut = runMain(t, fakeRegistry{}, "migrate-links", "--repo", dir)
	if code != 0 {
		t.Fatalf("migrate system: %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{
		"link LNK-COF-001-TELEMETRY REQ-COF-001 -> telemetry::CAP-TELEM-REPORT (v0.1.0, claimed)",
		"link LNK-COF-003-OTA-AGENT REQ-COF-003 -> ota-agent::CAP-OTA-APPLY (v0.1.0, claimed)",
		"link LNK-COF-002-CLOUD-API REQ-COF-002 -> cloud-api::CAP-CLOUD-INGEST (v0.1.0, claimed)",
		"engws: 9 requirement(s) without an upward trace", "requirement.no_parent: REQ-COF-004", "requirement.no_parent: REQ-COF-012",
		"added documents.reqLinks: model/req-links.yml",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("migrate output lacks %q:\n%s", want, out)
		}
	}
	// D22 (b): a requirement that became the parent of a capability link
	// traces upward and is not listed.
	for _, linked := range []string{"REQ-COF-001", "REQ-COF-002", "REQ-COF-003"} {
		if strings.Contains(out, "requirement.no_parent: "+linked) {
			t.Fatalf("migrate output lists link parent %s as without an upward trace:\n%s", linked, out)
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
	for _, l := range bundle.ReqLinks.ReqLinks {
		if l.Status != "claimed" || l.Pinned.Version != "v0.1.0" || l.Pinned.Fingerprint != "" || l.Rationale == "" || l.Parent == "" {
			t.Fatalf("link %+v", l)
		}
	}
	if strings.Contains(read(t, filepath.Join(dir, "model", "req-links.yml")), "fingerprint") {
		t.Fatalf("migrated link file names a fingerprint:\n%s", read(t, filepath.Join(dir, "model", "req-links.yml")))
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

// An allocation whose dependency cannot be resolved is still migrated as a
// claimed link and the output says its kind was not checked; link IDs stay
// unique when a requirement is allocated twice to one alias; an existing
// link file is extended, not replaced.
// Mutation: skipping an unresolvable allocation or reusing a link ID fails
// this test.
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
		!strings.Contains(out, "could not be resolved") {
		t.Fatalf("migrate: %d\n%s\n%s", code, out, errOut)
	}
	after := read(t, filepath.Join(system, "model", "req-links.yml"))
	if !strings.HasPrefix(after, before) || strings.Contains(after, "fingerprint") {
		t.Fatalf("link file not extended:\n%s", after)
	}
	if strings.Contains(read(t, arch), "allocations:") {
		t.Fatalf("allocations kept:\n%s", read(t, arch))
	}
	var doc model.ReqLinksDocument
	if err := decodeStrict([]byte(after), &doc); err != nil || len(doc.ReqLinks) != 6 {
		t.Fatalf("links: %v %+v", err, doc.ReqLinks)
	}
}

// D13: an allocation whose target resolves to a contract entry that is not
// a capability or assumption of use (an interface entry) is not converted:
// the allocation stays in place, so it keeps reporting
// composition.allocation_deprecated, and the output says why. The other
// allocations of the model are converted and removed; a second run changes
// nothing and still names the kept allocation.
// Mutation: converting the interface allocation, removing it with the
// others or dropping the note fails this test.
// TRLC-LINKS: REQ-EMG-104
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestMigrateLinksKeepsInterfaceAllocation(t *testing.T) {
	newLinksRegistry(t)
	root := filepath.Join(t.TempDir(), "links")
	copyTree(t, linksExample, root)
	child := filepath.Join(root, "repos", "child-a")
	replaceIn(t, filepath.Join(child, "model", "architecture.yml"), "    - id: CAP-A-2\n", "    - id: IF-A-OUT\n      kind: interface\n      note: Position sentence output.\n    - id: CAP-A-2\n")
	replaceIn(t, filepath.Join(child, "engmod.yml"), "  - CAP-A-2\n", "  - CAP-A-2\n  - IF-A-OUT\n")
	system := filepath.Join(root, "repos", "system")
	arch := filepath.Join(system, "model", "architecture.yml")
	replaceIn(t, arch, "      description: Forwards sentences onto the lab bus.\n", `      description: Forwards sentences onto the lab bus.
    allocations:
    - requirement: REQ-SYS-002
      to: SUB-ENCODER
      target: encoder::IF-A-OUT
      rationale: The encoder output carries the checksum.
    - requirement: REQ-SYS-002
      to: SUB-ENCODER
      target: encoder::CAP-A-1
      rationale: The encoder publishes the sentence the checksum protects.
`)
	kept := "kept allocation REQ-SYS-002 → encoder::IF-A-OUT: interface entries are not link targets; publish a capability that realizes it and link that"
	code, out, errOut := runMain(t, fakeRegistry{}, "migrate-links", "--repo", system)
	if code != 0 || !strings.Contains(out, kept) || !strings.Contains(out, "link LNK-SYS-002-ENCODER-2 REQ-SYS-002 -> encoder::CAP-A-1") || strings.Contains(out, "-> encoder::IF-A-OUT") {
		t.Fatalf("migrate: %d\n%s\n%s", code, out, errOut)
	}
	bundle, err := model.LoadBundle(filepath.Join(system, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := bundle.Architecture.Composition.Allocations; len(got) != 1 || got[0].Target != "encoder::IF-A-OUT" {
		t.Fatalf("want only the interface allocation kept, got %+v", got)
	}
	for _, l := range bundle.ReqLinks.ReqLinks {
		if strings.Contains(l.Child, "IF-A-OUT") {
			t.Fatalf("interface allocation converted to %+v", l)
		}
	}
	deprecated := false
	for _, d := range validate.Bundle(bundle) {
		deprecated = deprecated || (d.Code == "composition.allocation_deprecated" && strings.Contains(d.Message, "encoder::IF-A-OUT"))
	}
	if !deprecated {
		t.Fatal("kept allocation no longer reports composition.allocation_deprecated")
	}
	before := read(t, arch)
	code, out, errOut = runMain(t, fakeRegistry{}, "migrate-links", "--repo", system)
	if code != 0 || !strings.Contains(out, "engws: nothing to migrate") || !strings.Contains(out, kept) || read(t, arch) != before {
		t.Fatalf("second run: %d\n%s\n%s", code, out, errOut)
	}
}

// migrate-links writes several files (engmod.yml, the architecture, the
// requirements, the link file). When one write fails, every file already
// written is restored and a file the migration created is removed, so the
// model is never left half migrated.
// Mutation: not restoring the files written before the failure fails this
// test.
// TRLC-LINKS: REQ-EMG-104
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestMigrateLinksRestoresEveryFileOnWriteFailure(t *testing.T) {
	dir := coffeeCopy(t)
	names := []string{"engmod.yml", filepath.Join("model", "architecture.yml"), filepath.Join("model", "requirements.yml")}
	before := map[string]string{}
	for _, n := range names {
		before[n] = read(t, filepath.Join(dir, n))
	}
	old := writeFile
	t.Cleanup(func() { writeFile = old })
	writes := 0
	writeFile = func(path string, data []byte) error {
		if writes++; writes == 3 {
			return fmt.Errorf("injected write failure of %s", filepath.Base(path))
		}
		return old(path, data)
	}
	_, err := MigrateLinks(dir, false)
	if err == nil || !strings.Contains(err.Error(), "injected write failure") || !strings.Contains(err.Error(), "restored") {
		t.Fatalf("MigrateLinks: %v", err)
	}
	if writes < 3 {
		t.Fatalf("only %d writes; the failure was not injected mid-way", writes)
	}
	for _, n := range names {
		if got := read(t, filepath.Join(dir, n)); got != before[n] {
			t.Errorf("%s not restored:\n%s", n, got)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "model", "req-links.yml")); !os.IsNotExist(err) {
		t.Fatalf("the created link file was not removed: %v", err)
	}
}

// writeAllOrNothing restores an existing file and removes a file it
// created when a later write fails, and leaves an untouched file alone.
// Mutation: not removing a created file, or not restoring an existing one,
// fails this test.
// TRLC-LINKS: REQ-EMG-104
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestWriteAllOrNothingRemovesCreatedFiles(t *testing.T) {
	dir := t.TempDir()
	created, existing, failing := filepath.Join(dir, "a-new.yml"), filepath.Join(dir, "b-old.yml"), filepath.Join(dir, "c-fail.yml")
	write(t, existing, "old\n")
	write(t, failing, "keep\n")
	old := writeFile
	t.Cleanup(func() { writeFile = old })
	writeFile = func(path string, data []byte) error {
		if path == failing {
			return fmt.Errorf("injected")
		}
		return old(path, data)
	}
	paths := []string{created, existing, failing}
	err := writeAllOrNothing(paths, map[string][]byte{created: []byte("new\n"), existing: []byte("changed\n"), failing: []byte("x\n")})
	if err == nil || !strings.Contains(err.Error(), "every file restored") {
		t.Fatalf("writeAllOrNothing: %v", err)
	}
	if _, err := os.Stat(created); !os.IsNotExist(err) {
		t.Fatalf("created file kept: %v", err)
	}
	if read(t, existing) != "old\n" || read(t, failing) != "keep\n" {
		t.Fatalf("existing %q, failing %q", read(t, existing), read(t, failing))
	}
}

// migrate-links turns an allocation to a dependency's assumption of use
// into an evidence link (evidence: [the requirement], no parent), so the
// next check does not report link.aou_unverified; and it neither writes nor
// removes a pinned.fingerprint of an existing link file (a removed field,
// model.pinned_fingerprint_removed, ADR-EMG-075): the lines stay, every one is
// reported with its remedy and the new link carries none. A second run
// changes nothing.
// Mutation: writing parent for an assumption, removing a fingerprint again,
// writing one or not reporting them fails this test.
// TRLC-LINKS: REQ-EMG-104, REQ-EMG-118, REQ-EMG-184
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func TestMigrateLinksAssumptionsAndFingerprints(t *testing.T) {
	system, links, before := fingerprintedSystem(t)
	code, out, errOut := runMain(t, fakeRegistry{}, "migrate-links", "--repo", system)
	if code != 0 {
		t.Fatalf("migrate: %d\n%s\n%s", code, out, errOut)
	}
	assertFingerprintsReported(t, out)
	after := read(t, links)
	assertFingerprintsKept(t, before, after)
	if errs := checkErrors(t, system); hasLine(errs, "link.aou_unverified") || hasLine(errs, "link.parent") {
		t.Fatalf("migrated evidence link fails the check:\n%s", strings.Join(errs, "\n"))
	}
	if code, out, _ := runMain(t, fakeRegistry{}, "migrate-links", "--repo", system); code != 0 || !strings.Contains(out, "nothing to migrate") || read(t, links) != after {
		t.Fatalf("second run: %d\n%s", code, out)
	}
}

// fingerprintedSystem copies the links example, drops the AOU link and
// allocates it instead, and gives two links a fingerprint. It returns the
// system directory, its link file and the link file content before the run.
// TRLC-LINKS: REQ-EMG-104, REQ-EMG-184
func fingerprintedSystem(t *testing.T) (system, links, data string) {
	t.Helper()
	newLinksRegistry(t)
	root := filepath.Join(t.TempDir(), "links")
	copyTree(t, linksExample, root)
	system = filepath.Join(root, "repos", "system")
	links = filepath.Join(system, "model", "req-links.yml")
	data = read(t, links)
	data = data[:strings.Index(data, "- id: LNK-AOU-B-1")]
	data = strings.Replace(data, "    version: v0.1.0\n", "    version: v0.1.0\n    fingerprint: sha256:v1:"+strings.Repeat("a", 64)+"\n", 2)
	write(t, links, data)
	replaceIn(t, filepath.Join(system, "model", "architecture.yml"), "      description: Forwards sentences onto the lab bus.\n", `      description: Forwards sentences onto the lab bus.
    allocations:
    - requirement: REQ-SYS-003
      to: SUB-BRIDGE
      target: bridge::AOU-B-1
      rationale: The lab network keeps the bus isolated.
`)
	return system, links, data
}

// assertFingerprintsReported checks the output names the migrated evidence
// link and both fingerprints with their code and remedy.
// TRLC-LINKS: REQ-EMG-104, REQ-EMG-184
func assertFingerprintsReported(t *testing.T, out string) {
	t.Helper()
	for _, want := range []string{"link LNK-SYS-003-BRIDGE evidence REQ-SYS-003 -> bridge::AOU-B-1 (v0.1.0, claimed)", "delete pinned.fingerprint; pin the dependency version only"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if got := strings.Count(out, "model.pinned_fingerprint_removed"); got != 2 {
		t.Errorf("%d fingerprints reported, want 2:\n%s", got, out)
	}
}

// assertFingerprintsKept checks the run only appended: the fingerprints are
// untouched, the evidence link is added without one.
// TRLC-LINKS: REQ-EMG-104, REQ-EMG-184
func assertFingerprintsKept(t *testing.T, before, after string) {
	t.Helper()
	if !strings.HasPrefix(after, before) || strings.Count(after, "fingerprint:") != 2 {
		t.Fatalf("link file lost or gained a fingerprint:\n%s", after)
	}
	bridge := after[strings.Index(after, "- id: LNK-SYS-003-BRIDGE"):]
	if !strings.HasPrefix(bridge, "- id: LNK-SYS-003-BRIDGE\n  child: bridge::AOU-B-1\n  evidence:\n  - REQ-SYS-003\n") || strings.Contains(bridge, "fingerprint") {
		t.Fatalf("the migrated link is no evidence link or writes a fingerprint:\n%s", bridge)
	}
}

// removedFieldRun is one model edit of child-a and what migrate-links prints
// for it.
type removedFieldRun struct {
	name    string
	edit    func(t *testing.T, child string)
	want    []string // substrings of the output
	notWant []string // substrings the output must not contain
}

var removedFieldRuns = []removedFieldRun{
	{"a provides ref", func(t *testing.T, child string) {
		replaceIn(t, filepath.Join(child, "model", "architecture.yml"), "      kind: capability\n      note: Protect", "      kind: capability\n      ref: REQ-A-003\n      note: Protect")
	}, []string{"contract.provides[1].ref model.provides_ref_removed", "name the realizing requirement with realizes, not ref", "nothing to migrate"}, nil},
	{"no removed field", func(t *testing.T, child string) {}, []string{"nothing to migrate"}, []string{"_removed"}},
}

// migrate-links edits no provides.ref and no pinned.fingerprint (removed
// fields, ADR-EMG-075): the file stays byte for byte, the field is printed
// with its error code and remedy, and nothing is migrated for it. A model
// without one prints nothing about them.
// Mutation: converting the ref to realizes again, deleting the line or
// dropping the report fails a row.
// TRLC-LINKS: REQ-EMG-104, REQ-EMG-184
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestMigrateLinksReportsRemovedFields(t *testing.T) {
	for _, c := range removedFieldRuns {
		t.Run(c.name, func(t *testing.T) { runRemovedFieldCase(t, c) })
	}
}

// runRemovedFieldCase edits a copy of child-a, runs migrate-links and checks
// the output and that the model files are unchanged.
// TRLC-LINKS: REQ-EMG-104, REQ-EMG-184
func runRemovedFieldCase(t *testing.T, c removedFieldRun) {
	t.Helper()
	newLinksRegistry(t)
	child := filepath.Join(t.TempDir(), "child-a")
	copyTree(t, filepath.Join(linksExample, "repos", "child-a"), child)
	c.edit(t, child)
	files := []string{filepath.Join(child, "model", "architecture.yml"), filepath.Join(child, "model", "requirements.yml")}
	before := []string{read(t, files[0]), read(t, files[1])}
	code, out, errOut := runMain(t, fakeRegistry{}, "migrate-links", "--repo", child)
	if code != 0 {
		t.Fatalf("migrate: %d\n%s\n%s", code, out, errOut)
	}
	assertOutputHas(t, out, c.want...)
	assertOutputLacks(t, out, c.notWant...)
	for i, f := range files {
		if read(t, f) != before[i] {
			t.Errorf("migrate-links edited %s", f)
		}
	}
}

// assertOutputLacks fails for each substring the output has.
// TRLC-LINKS: REQ-EMG-104, REQ-EMG-184
func assertOutputLacks(t *testing.T, out string, not ...string) {
	t.Helper()
	for _, n := range not {
		if strings.Contains(out, n) {
			t.Errorf("output has %q:\n%s", n, out)
		}
	}
}
