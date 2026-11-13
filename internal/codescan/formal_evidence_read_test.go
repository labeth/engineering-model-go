// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

// The shared reader of formal evidence: containment of the module path, the
// generated-module requirement, the CRLF-normalised hash, the size cap and the
// directories scanned. Fixtures are built in temp directories.

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// sha is the hex sha256 of text.
//
// TRLC-LINKS: REQ-EMG-145
func sha(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// formalEntryFor is a passing entry of machine SM-X naming qntFile with the
// hash of text.
//
// TRLC-LINKS: REQ-EMG-145
func formalEntryFor(qntFile, text string) FormalEntry {
	return FormalEntry{Requirement: "REQ-X-1", Machine: "SM-X", Invariant: "INV-X", Result: FormalPass, QntFile: qntFile, QntSha256: sha(text)}
}

// A qntFile that is rooted, climbs out of the model root with .. or resolves
// through a symbolic link to a file outside it is stale, even when the file it
// reaches is a generated module with the recorded hash. Nothing outside the
// root is read.
//
// TRLC-LINKS: REQ-EMG-145
func TestFormalStaleReasonRejectsPathsOutsideTheModelRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "model")
	mustWrite(t, filepath.Join(root, "generated", "ok.qnt"), formalQnt)
	mustWrite(t, filepath.Join(parent, "outside.qnt"), formalQnt)
	if err := os.Symlink(filepath.Join(parent, "outside.qnt"), filepath.Join(root, "generated", "link.qnt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(parent, filepath.Join(root, "up")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/zero", filepath.Join(root, "generated", "zero.qnt")); err != nil {
		t.Fatal(err)
	}

	if reason := FormalStaleReason(root, formalEntryFor("generated/ok.qnt", formalQnt)); reason != "" {
		t.Fatalf("control: a module inside the root is current, got %q", reason)
	}
	for name, want := range map[string]string{
		"../outside.qnt":                     "not a local path",
		"generated/../../outside.qnt":        "not a local path",
		filepath.Join(parent, "outside.qnt"): "not a local path",
		"/etc/passwd":                        "not a local path",
		"generated/zero.qnt":                 "outside the model root",
		"generated/link.qnt":                 "outside the model root",
		"up/outside.qnt":                     "outside the model root",
		"../../../../dev/zero":               "not a local path",
		"generated/missing.qnt":              "is missing",
		"generated":                          "not a regular file",
	} {
		reason := FormalStaleReason(root, formalEntryFor(name, formalQnt))
		if !strings.Contains(reason, want) {
			t.Errorf("qntFile %q: reason %q, want it to contain %q", name, reason, want)
		}
	}
}

// A device, a FIFO or a file larger than the cap is stale at once; reading it
// neither blocks nor exhausts memory.
//
// TRLC-LINKS: REQ-EMG-145
func TestFormalStaleReasonRejectsSpecialAndHugeFiles(t *testing.T) {
	root := t.TempDir()
	huge := FormalGeneratedHeader + "SM-X; do not edit.\n" + strings.Repeat("x", maxFormalModuleBytes)
	mustWrite(t, filepath.Join(root, "generated", "huge.qnt"), huge)
	if reason := FormalStaleReason(root, formalEntryFor("generated/huge.qnt", huge)); !strings.Contains(reason, "larger than 4 MiB") {
		t.Errorf("huge module: %q", reason)
	}
	if _, err := readCappedFile("/dev/zero", 1<<10); err == nil {
		t.Errorf("a device must not be read")
	}
	if _, err := readCappedFile(root, 1<<10); err == nil {
		t.Errorf("a directory must not be read")
	}
	mkfifo(t, filepath.Join(root, "generated", "pipe.qnt"))
	if reason := FormalStaleReason(root, formalEntryFor("generated/pipe.qnt", formalQnt)); !strings.Contains(reason, "not a regular file") {
		t.Errorf("fifo: %q", reason)
	}
}

// A file whose hash was written down is no evidence unless it is a generated
// Quint module of the entry's machine.
//
// TRLC-LINKS: REQ-EMG-145
func TestFormalStaleReasonRequiresGeneratedModuleOfTheMachine(t *testing.T) {
	root := t.TempDir()
	plain := "module sm_x { val inv_x = true }\n"
	other := FormalGeneratedHeader + "SM-OTHER; do not edit.\nmodule sm_other {}\n"
	prefix := FormalGeneratedHeader + "SM-XYZ; do not edit.\nmodule sm_xyz {}\n"
	mustWrite(t, filepath.Join(root, "plain.qnt"), plain)
	mustWrite(t, filepath.Join(root, "other.qnt"), other)
	mustWrite(t, filepath.Join(root, "prefix.qnt"), prefix)
	mustWrite(t, filepath.Join(root, "late.qnt"), "\n"+formalQnt)
	for name, text := range map[string]string{"plain.qnt": plain, "other.qnt": other, "prefix.qnt": prefix, "late.qnt": "\n" + formalQnt} {
		if reason := FormalStaleReason(root, formalEntryFor(name, text)); !strings.Contains(reason, "not a generated Quint module") {
			t.Errorf("%s: %q", name, reason)
		}
	}
	noMachine := formalEntryFor("plain.qnt", plain)
	noMachine.Machine = ""
	mustWrite(t, filepath.Join(root, "gen.qnt"), formalQnt)
	noMachine.QntFile, noMachine.QntSha256 = "gen.qnt", sha(formalQnt)
	if reason := FormalStaleReason(root, noMachine); !strings.Contains(reason, "not a generated Quint module") {
		t.Errorf("an entry without a machine cannot be matched to a header: %q", reason)
	}
	// An edit after generation changes the hash and is reported as such.
	mustWrite(t, filepath.Join(root, "gen.qnt"), formalQnt+"// edited\n")
	changed := formalEntryFor("gen.qnt", formalQnt)
	if reason := FormalStaleReason(root, changed); !strings.Contains(reason, "no longer matches") {
		t.Errorf("edited module: %q", reason)
	}
}

// The hash is taken over the content with CRLF replaced by LF, so a checkout
// that rewrote line endings does not make every result stale.
//
// TRLC-LINKS: REQ-EMG-145
func TestFormalStaleReasonHashesCRLFAsLF(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "crlf.qnt"), strings.ReplaceAll(formalQnt, "\n", "\r\n"))
	if reason := FormalStaleReason(root, formalEntryFor("crlf.qnt", formalQnt)); reason != "" {
		t.Fatalf("a CRLF checkout of the module must stay current, got %q", reason)
	}
	mustWrite(t, filepath.Join(root, "lf.qnt"), formalQnt)
	if reason := FormalStaleReason(root, formalEntryFor("lf.qnt", formalQnt)); reason != "" {
		t.Fatalf("lf: %q", reason)
	}
}

// The scan reads the same directories the verification inference reads, so an
// evidence file beside the code root is found by every reader and is the
// evidence of the formal check; a nested directory and a repeated root do not
// duplicate it; the entries are ordered, ties by evidence file.
//
// TRLC-LINKS: REQ-EMG-146
func TestFormalEvidenceFilesMatchTheVerificationScan(t *testing.T) {
	root := t.TempDir()
	modelDir := filepath.Join(root, "model")
	mustWrite(t, filepath.Join(modelDir, "generated", "quint", "sm_x.qnt"), formalQnt)
	mustWrite(t, filepath.Join(root, "src", "keep.go"), "package src\n")
	entry := formalEntry1("REQ-X-1", "pass")
	body := strings.ReplaceAll(`{"results":[`+entry+`]}`, "%SHA%", sha(formalQnt))
	mustWrite(t, filepath.Join(modelDir, "test-results", "nested", "quint-formal.json"), body)
	mustWrite(t, filepath.Join(modelDir, "test-results", "a", "quint-formal.json"), body)
	mustWrite(t, filepath.Join(modelDir, "..", "test-results", "quint-formal.json"), body)

	files := FormalEvidenceFiles(modelDir, "../src")
	want := []string{"../test-results/quint-formal.json", "test-results/a/quint-formal.json", "test-results/nested/quint-formal.json"}
	if strings.Join(files, ",") != strings.Join(want, ",") {
		t.Fatalf("files %v, want %v", files, want)
	}
	if got := FormalEvidenceFiles(modelDir, ""); len(got) != 2 {
		t.Fatalf("without a code root only the model's test-results is read: %v", got)
	}
	entries := ReadFormalEvidence(modelDir, files)
	if len(entries) != 3 || entries[0].Evidence != want[0] || entries[1].Evidence != want[1] || entries[2].Evidence != want[2] {
		t.Fatalf("entries are ordered by evidence file on ties: %+v", entries)
	}
	for _, e := range entries {
		if e.Result != FormalPass || e.StaleReason != "" || e.Status != "pass" {
			t.Fatalf("entry %+v", e)
		}
	}

	bundle := model2Bundle(modelDir)
	checks, _ := InferVerificationChecks(bundle, requirementsX(), nil, "../src")
	var evidence []string
	for _, c := range checks {
		if c.Kind == "formal" {
			evidence = append(evidence, c.Evidence...)
		}
	}
	sort.Strings(evidence)
	if strings.Join(evidence, ",") != strings.Join(want, ",") {
		t.Fatalf("the verification inference reads %v, the shared scan %v", evidence, want)
	}
}

// A crafted evidence file naming a file outside the root reaches the
// verification inference as stale, not as an error or a pass.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func TestFormalEvidenceNamingAnOutsideFileIsStaleInTheInference(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "test-results", "quint-formal.json"),
		`{"results":[{"requirement":"REQ-X-1","status":"pass","machine":"SM-X","invariant":"INV-X","qntFile":"../../../../dev/zero","qntSha256":"`+sha("")+`"}]}`)
	checks, diags := inferFormal(t, root)
	if r, _ := resultFor(formalCheck(t, checks), "REQ-X-1"); r.Status != "not-run" || !hasDiag(diags, "verification.formal_stale") {
		t.Fatalf("outside file must be stale: %+v %+v", r, diags)
	}
}

// TRLC-LINKS: REQ-EMG-146
func model2Bundle(root string) model.Bundle {
	return model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml")}
}

// TRLC-LINKS: REQ-EMG-146
func requirementsX() model.RequirementsDocument {
	return model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-X-1"}, {ID: "REQ-X-2"}}}
}
