// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package linkreview

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
)

const acceptLinks = `schemaVersion: 2
reqLinks:
# reviewed links
- id: LNK-A
  parent: REQ-A
  child: dep::CAP-A
  pinned:
    version: v0.1.0
    fingerprint: sha256:v1:aaaa
  status: claimed
- id: LNK-B
  parent: REQ-A
  child: dep::CAP-B
  pinned:
    version: v0.1.0
    fingerprint: sha256:v1:bbbb
  status: accepted
  rationale: Old.
  reviewedBy: ACT-OWNER
  reviewedOn: "2026-01-01"
`

// acceptContext is a model with actor ACT-OWNER and the given statuses.
func acceptContext(statuses ...reqlinks.LinkStatus) Context {
	var b model.Bundle
	b.ReqLinksPath = "/m/model/req-links.yml"
	b.Architecture.AuthoredArchitecture.Actors = []model.Actor{{ID: "ACT-OWNER"}}
	return Context{Manifest: "/m/engmod.yml", Bundle: b, Statuses: statuses}
}

// Accept writes the current fingerprint, the dependency pin, status
// accepted, the rationale, reviewedBy and reviewedOn into exactly the lines
// of the link (adding the missing keys after its last key), and keeps every
// other byte; it refuses an unknown actor, an unknown link, a broken link
// (also behind a stale pin), one without a current fingerprint and a model
// without a link file, each without a result.
// Mutation: writing the recorded fingerprint, the old pin, skipping the
// actor check, accepting a broken link or rewriting other lines fails this
// test.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestAccept(t *testing.T) {
	suspect := reqlinks.LinkStatus{ID: "LNK-A", Child: "dep::CAP-A", State: reqlinks.StateVersionMismatch, Content: reqlinks.StateSuspect,
		Pinned: "v0.1.0", Version: "v0.2.0", Recorded: "sha256:v1:aaaa", Current: "sha256:v1:cccc"}
	got, err := Accept(acceptContext(suspect), []byte(acceptLinks), "LNK-A", "ACT-OWNER", " Still  fine. ", "2026-11-03")
	want := strings.Replace(acceptLinks, `    version: v0.1.0
    fingerprint: sha256:v1:aaaa
  status: claimed
`, `    version: v0.2.0
    fingerprint: sha256:v1:cccc
  status: accepted
  rationale: Still fine.
  reviewedBy: ACT-OWNER
  reviewedOn: "2026-11-03"
`, 1)
	if err != nil || string(got.After) != want || string(got.Before) != acceptLinks || got.Path != "/m/model/req-links.yml" {
		t.Fatalf("Accept: %v\n%s\nwant\n%s", err, got.After, want)
	}
	if patch := got.Patch("model/req-links.yml"); patch != "--- a/model/req-links.yml\n+++ b/model/req-links.yml\n-    version: v0.1.0\n-    fingerprint: sha256:v1:aaaa\n-  status: claimed\n+    version: v0.2.0\n+    fingerprint: sha256:v1:cccc\n+  status: accepted\n+  rationale: Still fine.\n+  reviewedBy: ACT-OWNER\n+  reviewedOn: \"2026-11-03\"\n" {
		t.Fatalf("Patch:\n%s", patch)
	}
	// A version_mismatch link whose content is unchanged is accepted too:
	// only the pin moves (the fingerprint is the same).
	repin := reqlinks.LinkStatus{ID: "LNK-B", Child: "dep::CAP-B", State: reqlinks.StateVersionMismatch, Content: reqlinks.StateUnchanged,
		Pinned: "v0.1.0", Version: "v0.2.0", Recorded: "sha256:v1:bbbb", Current: "sha256:v1:bbbb"}
	got, err = Accept(acceptContext(repin), []byte(acceptLinks), "LNK-B", "ACT-OWNER", "Old.", "2026-01-01")
	if want := strings.Replace(acceptLinks, "    version: v0.1.0\n    fingerprint: sha256:v1:bbbb", "    version: v0.2.0\n    fingerprint: sha256:v1:bbbb", 1); err != nil || string(got.After) != want {
		t.Fatalf("Accept of a version_mismatch link:\n%s (%v)", got.After, err)
	}

	broken := reqlinks.LinkStatus{ID: "LNK-A", Child: "dep::CAP-A", State: reqlinks.StateBroken, Version: "v0.2.0"}
	staleBroken := reqlinks.LinkStatus{ID: "LNK-A", Child: "dep::CAP-A", State: reqlinks.StateVersionMismatch, Content: reqlinks.StateBroken, Version: "v0.2.0"}
	unresolved := reqlinks.LinkStatus{ID: "LNK-A", Child: "dep::CAP-A", State: reqlinks.StateUnresolved, Detail: "dependency \"dep\" was not resolved"}
	noFile := acceptContext(suspect)
	noFile.Bundle.ReqLinksPath = ""
	for name, tc := range map[string]struct {
		ctx      Context
		id, by   string
		wantErr  string
		noStatus bool
	}{
		"unknown actor": {acceptContext(suspect), "LNK-A", "ACT-NOBODY", "--by ACT-NOBODY is not an actor of the model", true},
		"unknown link":  {acceptContext(suspect), "LNK-NONE", "ACT-OWNER", "no requirement link LNK-NONE in req-links.yml", true},
		"broken":        {acceptContext(broken), "LNK-A", "ACT-OWNER", "link LNK-A is broken: dep::CAP-A no longer exists at v0.2.0", false},
		"stale broken":  {acceptContext(staleBroken), "LNK-A", "ACT-OWNER", "link LNK-A is broken", false},
		"unresolved":    {acceptContext(unresolved), "LNK-A", "ACT-OWNER", "link LNK-A cannot be accepted: unresolved (dependency \"dep\" was not resolved)", false},
		"no link file":  {noFile, "LNK-A", "ACT-OWNER", "engmod.yml names no documents.reqLinks", true},
	} {
		got, err := Accept(tc.ctx, []byte(acceptLinks), tc.id, tc.by, "x", "2026-11-03")
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) || got.After != nil || (tc.noStatus && got.Status.ID != "") {
			t.Errorf("%s: err %v, result %+v", name, err, got)
		}
	}
}

// DiffLines marks removed lines "- ", added "+ " and common "  " in order,
// and without a before shows the after lines unmarked.
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestDiffLines(t *testing.T) {
	got := DiffLines([]string{"a", "b", "c"}, []string{"a", "x", "c", "d"}, true)
	if strings.Join(got, "|") != "  a|- b|+ x|  c|+ d" {
		t.Fatalf("DiffLines = %q", got)
	}
	if got := DiffLines(nil, []string{"a"}, false); strings.Join(got, "|") != "  a" {
		t.Fatalf("DiffLines without before = %q", got)
	}
}
