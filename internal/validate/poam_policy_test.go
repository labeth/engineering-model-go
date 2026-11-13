// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/labeth/engineering-model-go/internal/model"
)

// day parses a YYYY-MM-DD test date.
func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// pinAsOf pins AsOf for one test and clears the pin afterwards.
func pinAsOf(t *testing.T, s string) {
	t.Helper()
	SetAsOf(day(t, s))
	t.Cleanup(func() { SetAsOf(time.Time{}) })
}

// poamBundle is a model at the tier with one open POA&M item of that tier
// (due 2027-06-30) carrying the defers entries.
func poamBundle(tier string, defers ...model.POAMDefer) model.Bundle {
	b := tierBundle(tier)
	b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: "POAM-T", Milestone: "tier:" + tier, Status: "planned", DueDate: "2027-06-30", Defers: defers}}
	return b
}

// A defer never lowers a non-deferrable code (exact, with a path or through a
// family pattern) at any tier, and the entry is lifecycle.poam_defers_core.
// Mutation: dropping the !IsNonDeferrable(d.Code) guard of ApplyTierPolicyAt
// lets the item defer health.policy_weakened.
// TRLC-LINKS: REQ-EMG-129
func TestNonDeferrableCodesAreNeverDeferred(t *testing.T) {
	pinAsOf(t, "2026-11-13")
	t.Run("list", func(t *testing.T) {
		assertNonDeferrableKinds(t)
		assertNonDeferrableListIsClosed(t)
	})
	for _, tier := range []string{"poc", "prototype", "mvp", "product", "operational"} {
		for _, e := range nonDeferrableEntries {
			t.Run(tier+"/"+e.name, func(t *testing.T) {
				assertDeferKeepsTierSeverity(t, tier, e)
				assertDefersCoreReported(t, tier, e)
			})
		}
	}
}

// nonDeferrableEntry is a defers entry that names a non-deferrable code and
// the diagnostic (code and path) it tries to defer.
type nonDeferrableEntry struct {
	name  string
	code  string
	path  string
	entry model.POAMDefer
}

var nonDeferrableEntries = []nonDeferrableEntry{
	{"exact", "health.policy_weakened", "health", model.POAMDefer{Code: "health.policy_weakened"}},
	{"exact with path", "health.policy_weakened", "health.thresholds", model.POAMDefer{Code: "health.policy_weakened", Path: "health.thresholds"}},
	{"family", "health.policy_weakened", "health", model.POAMDefer{Code: "health.p*"}},
	{"regression family", "health.regression", "x.go", model.POAMDefer{Code: "health.r*"}},
	{"baseline stale", "health.baseline_stale", "b", model.POAMDefer{Code: "health.baseline_stale"}},
	{"waiver unused", "health.waiver_unused", "w", model.POAMDefer{Code: "health.waiver_unused"}},
	{"verification failed", "verification.failed", "v", model.POAMDefer{Code: "verification.failed"}},
	{"verification family", "verification.failed", "v", model.POAMDefer{Code: "verification.*"}},
}

// assertNonDeferrableKinds checks that every non-deferrable code is a
// tier-keyed code or a POA&M check, never a core code.
// TRLC-LINKS: REQ-EMG-129
func assertNonDeferrableKinds(t *testing.T) {
	t.Helper()
	for _, code := range NonDeferrableCodes() {
		if IsCoreCode(code) {
			t.Errorf("%s is core: a non-deferrable code is the tier-keyed kind", code)
		}
		if !TierKeyed(code) && !strings.HasPrefix(code, "lifecycle.poam_") {
			t.Errorf("%s is neither tier keyed nor a POA&M check", code)
		}
	}
}

// assertNonDeferrableListIsClosed checks the exact list of non-deferrable
// codes.
// TRLC-LINKS: REQ-EMG-129
func assertNonDeferrableListIsClosed(t *testing.T) {
	t.Helper()
	const want = "health.policy_weakened,health.regression,health.baseline_stale,health.waiver_unused,verification.failed,lifecycle.poam_overdue,lifecycle.poam_defers_unused"
	if got := strings.Join(NonDeferrableCodes(), ","); got != want {
		t.Fatalf("NonDeferrableCodes() = %s: the list is closed", got)
	}
}

// assertDeferKeepsTierSeverity checks that the defers entry changed nothing
// of the diagnostic compared with a model without the POA&M item.
// TRLC-LINKS: REQ-EMG-129
func assertDeferKeepsTierSeverity(t *testing.T, tier string, e nonDeferrableEntry) {
	t.Helper()
	b := poamBundle(tier, e.entry)
	want := severityAt(t, tierBundle(tier), e.code, e.path, SeverityWarning)
	got := severityAt(t, b, e.code, e.path, SeverityWarning)
	if got.Severity != want.Severity || got.Deferred != want.Deferred || got.PoamRef != "" {
		t.Errorf("%s at %s: defer changed %s: %+v, want the tier severity %+v", e.name, tier, e.code, got, want)
	}
}

// assertDefersCoreReported checks that the entry is one
// lifecycle.poam_defers_core error naming its code.
// TRLC-LINKS: REQ-EMG-129
func assertDefersCoreReported(t *testing.T, tier string, e nonDeferrableEntry) {
	t.Helper()
	found := codesOf(Bundle(poamBundle(tier, e.entry)), "lifecycle.poam_defers_core")
	if len(found) != 1 || found[0].Severity != SeverityError || !strings.Contains(found[0].Message, e.entry.Code) {
		t.Errorf("%s at %s: poam_defers_core = %+v", e.name, tier, found)
	}
}

// The legitimately deferrable codes stay deferrable by an exact entry; a
// family wildcard that would cover a core code is rejected and the message
// says which one.
// Mutation: adding health.complex_method or lifecycle.requirement_unverified
// to the protected lists makes the exact entries fail here.
// TRLC-LINKS: REQ-EMG-129
func TestExactDefersOfDeferrableCodesAreAccepted(t *testing.T) {
	pinAsOf(t, "2026-11-13")
	for _, code := range []string{"lifecycle.requirement_unverified", "health.complex_method", "lifecycle.requirement_unapproved"} {
		t.Run("exact "+code, func(t *testing.T) { assertExactDeferAccepted(t, code) })
	}
	for _, c := range []struct{ pattern, covers string }{
		{"health.*", "health.block_missing"},
		{"health.r*", "health.regression"},
		{"lifecycle.*", "lifecycle.invalid_tier"},
		{"model.*", "model.duplicate_id"},
		{"verification.f*", "verification.failed"},
	} {
		t.Run("wildcard "+c.pattern, func(t *testing.T) { assertWildcardRejected(t, c.pattern, c.covers) })
	}
}

// assertExactDeferAccepted checks that an exact defers entry for code is not
// rejected and defers the diagnostic.
// TRLC-LINKS: REQ-EMG-129
func assertExactDeferAccepted(t *testing.T, code string) {
	t.Helper()
	b := poamBundle("product", model.POAMDefer{Code: code})
	if got := codesOf(Bundle(b), "lifecycle.poam_defers_core"); len(got) != 0 {
		t.Errorf("exact defer of %s rejected: %+v", code, got)
	}
	if got := severityAt(t, b, code, "p", SeverityWarning); got.PoamRef != "POAM-T" || !got.Deferred {
		t.Errorf("exact defer of %s did not defer: %+v", code, got)
	}
}

// assertWildcardRejected checks that a wildcard entry is one
// lifecycle.poam_defers_core naming the pattern and the code it covers.
// TRLC-LINKS: REQ-EMG-129
func assertWildcardRejected(t *testing.T, pattern, covers string) {
	t.Helper()
	b := poamBundle("product", model.POAMDefer{Code: pattern})
	got := codesOf(Bundle(b), "lifecycle.poam_defers_core")
	if len(got) != 1 || !strings.Contains(got[0].Message, covers) || !strings.Contains(got[0].Message, pattern) {
		t.Errorf("%s: poam_defers_core = %+v, want a message naming %s", pattern, got, covers)
	}
}

// An item past its dueDate at asOf is not honoured: the diagnostic keeps its
// tier severity and lifecycle.poam_overdue names the item (an error from
// product, a warning below). The due day itself is still in time, a completed
// item never expires, and the zero asOf (a library caller that passes no
// date) disables expiry.
// Mutation: dropping the !poamOverdue(p, asOf) test of activePoamItems keeps
// the overdue item's defer.
// TRLC-LINKS: REQ-EMG-129
func TestOverduePoamItemDefersNothingAndIsReported(t *testing.T) {
	for _, c := range overdueCases {
		t.Run(c.name, func(t *testing.T) { assertOverdueCase(t, c) })
	}
	t.Run("completed is never overdue", func(t *testing.T) {
		if got := poamDiagnostics(overdueBundle("product", "2020-01-01", "completed"), day(t, "2026-11-13")); len(got) != 0 {
			t.Errorf("a completed item is never overdue: %+v", got)
		}
	})
	t.Run("severity by tier", assertOverdueSeverityByTier)
	t.Run("ApplyTierPolicy uses the pinned date", assertOverdueAtPinnedDate)
}

// overdueIn is the diagnostic the overdue item tries to defer.
var overdueIn = []Diagnostic{{Code: "lifecycle.requirement_unverified", Severity: SeverityWarning, Path: "requirements[0].status"}}

// overdueBundle is a model at the tier with one POA&M item of that tier
// deferring overdueIn, with the due date and status.
// TRLC-LINKS: REQ-EMG-129
func overdueBundle(tier, due, status string) model.Bundle {
	b := poamBundle(tier, model.POAMDefer{Code: "lifecycle.requirement_unverified", Path: "requirements[0].status"})
	b.Architecture.AuthoredArchitecture.POAMItems[0].DueDate = due
	b.Architecture.AuthoredArchitecture.POAMItems[0].Status = status
	return b
}

// overdueCase is one due date and status judged at one asOf date.
type overdueCase struct {
	name    string
	due     string
	status  string
	asOf    string
	honored bool
}

var overdueCases = []overdueCase{
	{"day before", "2026-11-12", "planned", "2026-11-13", false},
	{"due day", "2026-11-13", "planned", "2026-11-13", true},
	{"future", "2026-11-14", "in-progress", "2026-11-13", true},
	{"completed overdue", "2020-01-01", "completed", "2026-11-13", false},
	{"timestamp", "2026-11-12T10:00:00Z", "planned", "2026-11-13", false},
	{"no date", "", "planned", "2026-11-13", true},
	{"unparsable", "next quarter", "planned", "2026-11-13", true},
	{"no clock", "2000-01-01", "planned", "", true},
}

// optionalDay parses a test date; "" is the zero time.
// TRLC-LINKS: REQ-EMG-129
func optionalDay(t *testing.T, s string) time.Time {
	t.Helper()
	if s == "" {
		return time.Time{}
	}
	return day(t, s)
}

// assertOverdueCase checks whether the item of the case is honoured and
// whether it is reported overdue.
// TRLC-LINKS: REQ-EMG-129
func assertOverdueCase(t *testing.T, c overdueCase) {
	t.Helper()
	asOf := optionalDay(t, c.asOf)
	b := overdueBundle("product", c.due, c.status)
	out := ApplyTierPolicyAt(b, overdueIn, asOf)
	if c.status == "completed" {
		// A completed item defers nothing for its own reason.
		if out[0].PoamRef != "" {
			t.Errorf("%s: completed item deferred", c.name)
		}
		return
	}
	if got := out[0].PoamRef == "POAM-T"; got != c.honored {
		t.Errorf("%s: honoured=%v, want %v (%+v)", c.name, got, c.honored, out[0])
	}
	overdue := len(poamDiagnostics(b, asOf))
	if (overdue == 1) == c.honored {
		t.Errorf("%s: %d overdue diagnostics, honoured=%v", c.name, overdue, c.honored)
	}
}

// assertOverdueSeverityByTier checks the severity of the diagnostic by tier,
// through Bundle at a pinned date.
// TRLC-LINKS: REQ-EMG-129
func assertOverdueSeverityByTier(t *testing.T) {
	pinAsOf(t, "2026-11-13")
	if IsCoreCode("lifecycle.poam_overdue") || !IsNonDeferrable("lifecycle.poam_overdue") {
		t.Fatal("lifecycle.poam_overdue must be non-deferrable and not core")
	}
	want := map[string]Severity{"poc": SeverityWarning, "prototype": SeverityWarning, "mvp": SeverityWarning, "product": SeverityError, "operational": SeverityError}
	for tier, severity := range want {
		t.Run(tier, func(t *testing.T) {
			got := codesOf(Bundle(overdueBundle(tier, "2026-01-01", "planned")), "lifecycle.poam_overdue")
			assertOverdueDiagnostic(t, got, severity)
		})
	}
}

// assertOverdueDiagnostic checks that got is one poam_overdue diagnostic of
// the severity naming POAM-T at the dueDate path.
// TRLC-LINKS: REQ-EMG-129
func assertOverdueDiagnostic(t *testing.T, got []Diagnostic, severity Severity) {
	t.Helper()
	if len(got) != 1 {
		t.Errorf("poam_overdue = %+v, want one %s naming POAM-T", got, severity)
		return
	}
	d := got[0]
	if d.Severity != severity || !strings.Contains(d.Message, `"POAM-T"`) || d.Path != "authoredArchitecture.poamItems[0].dueDate" {
		t.Errorf("poam_overdue = %+v, want one %s naming POAM-T", got, severity)
	}
}

// assertOverdueAtPinnedDate checks that ApplyTierPolicy uses the pinned date:
// the overdue defer is gone, and it is back on the due day.
// TRLC-LINKS: REQ-EMG-129
func assertOverdueAtPinnedDate(t *testing.T) {
	pinAsOf(t, "2026-11-13")
	b := overdueBundle("product", "2026-11-12", "planned")
	if out := ApplyTierPolicy(b, overdueIn); out[0].PoamRef != "" || out[0].Deferred {
		t.Errorf("ApplyTierPolicy honoured an overdue item at the pinned date: %+v", out[0])
	}
	SetAsOf(day(t, "2026-11-12"))
	if out := ApplyTierPolicy(b, overdueIn); out[0].PoamRef != "POAM-T" {
		t.Errorf("ApplyTierPolicy did not honour the item on its due day: %+v", out[0])
	}
}

// AsOf is the pinned date, else ENGMOD_AS_OF, else today's UTC date; an
// unparsable ENGMOD_AS_OF never disables expiry.
// TRLC-LINKS: REQ-EMG-129
func TestAsOfOrder(t *testing.T) {
	SetAsOf(time.Time{})
	t.Setenv("ENGMOD_AS_OF", "2026-03-04")
	if got := AsOf().Format(time.DateOnly); got != "2026-03-04" {
		t.Errorf("env date = %s", got)
	}
	SetAsOf(day(t, "2027-01-02"))
	t.Cleanup(func() { SetAsOf(time.Time{}) })
	if got := AsOf().Format(time.DateOnly); got != "2027-01-02" {
		t.Errorf("pinned date = %s, want it to win over the env", got)
	}
	SetAsOf(time.Time{})
	t.Setenv("ENGMOD_AS_OF", "not a date")
	today := time.Now().UTC().Format(time.DateOnly)
	if got := AsOf(); got.Format(time.DateOnly) != today || got.IsZero() {
		t.Errorf("unparsable env gave %v, want today %s", got, today)
	}
	t.Setenv("ENGMOD_AS_OF", "")
	if got := AsOf().Format(time.DateOnly); got != today {
		t.Errorf("default = %s, want today %s", got, today)
	}
}

// A defers entry may name a requirement by ref; it is resolved to the
// requirement's current path, so inserting a requirement before it does not
// move the defer to another requirement. A positional path does move.
// Mutation: dropping the ref test of deferMatches makes the ref entry defer
// every requirement.
// TRLC-LINKS: REQ-EMG-129
func TestDeferByRefFollowsTheRequirement(t *testing.T) {
	pinAsOf(t, "2026-11-13")
	b := poamBundle("product", model.POAMDefer{Code: "lifecycle.requirement_unverified", Ref: "REQ-B"})
	b.Requirements.Requirements = []model.Requirement{{ID: "REQ-A", Status: "approved"}, {ID: "REQ-B", Status: "approved"}}
	check := func(label string, wantRef map[string]string) {
		t.Helper()
		for _, d := range ApplyTierPolicy(b, Bundle(b)) {
			if d.Code != "lifecycle.requirement_unverified" {
				continue
			}
			if d.PoamRef != wantRef[d.Path] {
				t.Errorf("%s: %s poamRef %q, want %q", label, d.Path, d.PoamRef, wantRef[d.Path])
			}
		}
	}
	check("before", map[string]string{"requirements[0].status": "", "requirements[1].status": "POAM-T"})
	b.Requirements.Requirements = append([]model.Requirement{{ID: "REQ-NEW", Status: "approved"}}, b.Requirements.Requirements...)
	check("after an insert", map[string]string{"requirements[0].status": "", "requirements[1].status": "", "requirements[2].status": "POAM-T"})
	if got := codesOf(Bundle(b), "lifecycle.poam_defers_unused"); len(got) != 0 {
		t.Errorf("a resolving ref is not unused: %+v", got)
	}
	// A ref also covers the by-ID path form and paths below the requirement.
	for _, path := range []string{"requirements[REQ-B]", "requirements[2]", "requirements[2].status"} {
		if got := severityAt(t, b, "lifecycle.requirement_unverified", path, SeverityWarning); got.PoamRef != "POAM-T" {
			t.Errorf("ref did not cover %s: %+v", path, got)
		}
	}
	if got := severityAt(t, b, "lifecycle.requirement_unverified", "requirements[20].status", SeverityWarning); got.PoamRef != "" {
		t.Errorf("ref covered requirements[20]: %+v", got)
	}
}

// A defers entry that matches no emitted diagnostic is
// lifecycle.poam_defers_unused: a warning at every tier below operational and
// an error at operational. A positional path that still matches, an overdue
// item and an item of another tier are not reported.
// Mutation: reporting every entry, or none, fails the used and the unused
// rows.
// TRLC-LINKS: REQ-EMG-129
func TestUnusedDefersAreReported(t *testing.T) {
	pinAsOf(t, "2026-11-13")
	for tier, severity := range map[string]Severity{"poc": SeverityWarning, "mvp": SeverityWarning, "product": SeverityWarning, "operational": SeverityError} {
		t.Run(tier, func(t *testing.T) { assertUnusedAtTier(t, tier, severity) })
	}
	t.Run("merged list", assertUnusedOnMergedList)
	t.Run("another tier", func(t *testing.T) {
		other := unusedDefersBundle("operational")
		other.Architecture.AuthoredArchitecture.POAMItems[0].Milestone = "tier:product"
		if got := codesOf(Bundle(other), "lifecycle.poam_defers_unused"); len(got) != 0 {
			t.Errorf("another tier's item judged: %+v", got)
		}
	})
	t.Run("overdue item", func(t *testing.T) {
		late := unusedDefersBundle("operational")
		late.Architecture.AuthoredArchitecture.POAMItems[0].DueDate = "2026-01-01"
		if got := codesOf(Bundle(late), "lifecycle.poam_defers_unused"); len(got) != 0 {
			t.Errorf("an overdue item is reported as overdue, not unused: %+v", got)
		}
		if len(codesOf(Bundle(late), "lifecycle.poam_overdue")) != 1 {
			t.Error("the late item must be overdue")
		}
	})
}

// unusedDefersBundle is a model whose one POA&M item carries a used path
// entry, a stale one, an unknown ref and a health entry.
// TRLC-LINKS: REQ-EMG-129
func unusedDefersBundle(tier string) model.Bundle {
	b := poamBundle(tier,
		model.POAMDefer{Code: "lifecycle.requirement_unverified", Path: "requirements[0].status"},
		model.POAMDefer{Code: "lifecycle.requirement_unverified", Path: "requirements[7].status"},
		model.POAMDefer{Code: "lifecycle.requirement_unverified", Ref: "REQ-GONE"},
		model.POAMDefer{Code: "health.complex_method"},
	)
	b.Requirements.Requirements = []model.Requirement{{ID: "REQ-A", Status: "approved"}}
	return b
}

// assertUnusedAtTier checks the unused entries Bundle reports at a tier: the
// stale path and the unknown ref, with the severity of the tier.
// TRLC-LINKS: REQ-EMG-129
func assertUnusedAtTier(t *testing.T, tier string, severity Severity) {
	t.Helper()
	got := codesOf(Bundle(unusedDefersBundle(tier)), "lifecycle.poam_defers_unused")
	if len(got) != 2 {
		t.Fatalf("%s: unused = %+v, want the stale path and the unknown ref (health.* entries are judged on the merged list)", tier, got)
	}
	paths := []string{got[0].Path, got[1].Path}
	if paths[0] != "authoredArchitecture.poamItems[0].defers[1]" || paths[1] != "authoredArchitecture.poamItems[0].defers[2]" {
		t.Errorf("%s: paths %v", tier, paths)
	}
	for _, d := range got {
		if d.Severity != severity || !strings.Contains(d.Message, `"POAM-T"`) {
			t.Errorf("%s: %+v, want %s naming the item", tier, d, severity)
		}
	}
}

// assertUnusedOnMergedList checks that on the merged list the health entry is
// judged too.
// TRLC-LINKS: REQ-EMG-129
func assertUnusedOnMergedList(t *testing.T) {
	b := unusedDefersBundle("operational")
	merged := append(Bundle(b), Diagnostic{Code: "health.complex_method", Severity: SeverityWarning, Path: "a.go"})
	if got := codesOf(UnusedDefers(b, merged), "lifecycle.poam_defers_unused"); len(got) != 2 {
		t.Errorf("merged with a matching health finding: %+v, want 2", got)
	}
	if got := UnusedDefers(b, Bundle(b)); len(got) != 3 {
		t.Errorf("merged without a health finding: %+v, want 3", got)
	}
}

// Model strings cannot forge a line or drive the terminal: a requirement ID
// holding a newline and a fake fix line prints on one line, in the message
// (quoted with %q) and again in Line() (control characters escaped). JSON
// keeps the raw string.
// Mutation: dropping escapeControl from Line, or %q from the lifecycle
// message, lets the fake fix line through.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func TestModelStringsCannotForgeOutputLines(t *testing.T) {
	t.Run("lifecycle diagnostics", assertLifecycleDiagnosticsAreSafe)
	d := Diagnostic{Code: "x.y", Severity: SeverityError, Message: "a\nb\r\x1b[31m\tc\x7f\u0085\u2028", Path: "p\n  fix: no"}
	t.Run("Line", func(t *testing.T) { assertLineEscapes(t, d) })
	t.Run("one line", func(t *testing.T) {
		var out bytes.Buffer
		WriteDiagnostic(&out, d)
		if strings.Count(out.String(), "\n") != 1 {
			t.Errorf("a diagnostic without remedy prints one line, got %q", out.String())
		}
	})
	t.Run("json", func(t *testing.T) { assertJSONKeepsString(t, d) })
	t.Run("plain text", func(t *testing.T) {
		// Ordinary text, including non-ASCII, is untouched.
		plain := Diagnostic{Code: "x.y", Severity: SeverityWarning, Message: "Åre ✓ 日本", Path: "p"}
		if plain.Line() != "x.y [warning] Åre ✓ 日本 (p)" {
			t.Errorf("plain text changed: %q", plain.Line())
		}
	})
}

// assertLifecycleDiagnosticsAreSafe checks the lifecycle diagnostics of a
// model whose IDs hold control characters and a fake fix line.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func assertLifecycleDiagnosticsAreSafe(t *testing.T) {
	pinAsOf(t, "2026-11-13")
	seen := map[string]bool{}
	for _, d := range Bundle(evilBundle()) {
		if !strings.HasPrefix(d.Code, "lifecycle.requirement_") && !strings.HasPrefix(d.Code, "lifecycle.poam_") {
			continue
		}
		seen[d.Code] = true
		assertDiagnosticIsSafe(t, d)
	}
	assertAllProduced(t, seen)
}

// evilBundle is a model whose IDs hold control characters and a fake fix line.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func evilBundle() model.Bundle {
	evil := "REQ-X\n  fix: do evil\x1b[2J\x7f\u202e"
	b := tierBundle("product")
	b.Requirements.Requirements = []model.Requirement{{ID: evil, Status: "draft"}}
	b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: evil, Milestone: "tier:product", Status: "open", DueDate: "2020-01-01",
		Defers: []model.POAMDefer{{Code: "model.invalid_\n  fix: no"}}}}
	return b
}

// assertAllProduced checks that evilBundle produced each lifecycle code the
// test is about.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func assertAllProduced(t *testing.T, seen map[string]bool) {
	t.Helper()
	for _, code := range []string{"lifecycle.requirement_unapproved", "lifecycle.requirement_unverified", "lifecycle.poam_defers_core", "lifecycle.poam_overdue"} {
		if !seen[code] {
			t.Errorf("%s not produced", code)
		}
	}
}

// assertDiagnosticIsSafe checks that a diagnostic prints without a raw
// control character or a forged fix line.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func assertDiagnosticIsSafe(t *testing.T, d Diagnostic) {
	t.Helper()
	if strings.ContainsAny(d.Message, "\n\r\x1b\x7f") {
		t.Errorf("%s: message holds a raw control character: %q", d.Code, d.Message)
	}
	var out bytes.Buffer
	WriteDiagnostic(&out, d)
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		if strings.HasPrefix(line, "  fix:") && d.Remedy == "" {
			t.Errorf("%s: forged fix line %q", d.Code, line)
		}
	}
	if strings.ContainsAny(out.String(), "\r\x1b\x7f\u202e") {
		t.Errorf("%s: raw control character in %q", d.Code, out.String())
	}
}

// assertLineEscapes checks Line itself, whatever the message is.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func assertLineEscapes(t *testing.T, d Diagnostic) {
	t.Helper()
	if want := `x.y [error] a\nb\r\x1b[31m\tc\x7f\x85\u2028 (p\n  fix: no)`; d.Line() != want {
		t.Errorf("Line() = %q, want %q", d.Line(), want)
	}
}

// assertJSONKeepsString checks that JSON keeps the raw string and round
// trips it.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
func assertJSONKeepsString(t *testing.T, d Diagnostic) {
	t.Helper()
	data, err := json.Marshal(d)
	if err != nil || !strings.Contains(string(data), `a\nb\r`) {
		t.Errorf("json must keep the string (JSON escapes it): %s, %v", data, err)
	}
	var back Diagnostic
	if err := json.Unmarshal(data, &back); err != nil || back.Message != d.Message {
		t.Errorf("json round trip: %v", err)
	}
}
