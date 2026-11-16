// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// belowFloorPaths lists the paths of the files below the floor of a check.
func belowFloorPaths(doc checkJSON) []string {
	out := []string{}
	for _, f := range doc.BelowFloor {
		out = append(out, f.Path)
	}
	return out
}

// newFile is a file of another path than svc.go with the finding of svc.go (a
// warn complex_method that scores 9.0 at every tier).
func newFile() string { return strings.Replace(svcSource(9), "func Svc", "func New", 1) }

// floorSetup is a check repository with a baseline file (when baseline is true)
// that covers pkg/svc.go, committed as the base.
func floorSetup(t *testing.T, extra string, baseline bool) *checkRepo {
	t.Helper()
	r := newCheckRepo(t, extra, false)
	if baseline {
		r.write("model/health-baseline.json", strings.Replace(svcBaseline, "21", "10", 1))
	}
	r.commit("base")
	return r
}

// floorCase is a manifest, an edit of the working tree and the result: the exit
// code, the level of the code and the files below the floor.
type floorCase struct {
	name     string
	extra    string
	baseline bool
	edit     func(r *checkRepo)
	code     int
	level    string
	want     []string
}

// writeSvc9 is the edit that brings pkg/svc.go to a warn finding (score 9.0).
func writeSvc9(r *checkRepo) { r.write("pkg/svc.go", svcSource(9)) }

// writeNew is the edit that adds pkg/new.go with the finding of svc.go.
func writeNew(r *checkRepo) { r.write("pkg/new.go", newFile()) }

// tierCases are the changed file below the floor at each tier, with the safety
// bundle and with lifecycle.raise.
func tierCases() []floorCase {
	return []floorCase{
		{"product", "lifecycle:\n  tier: product\n", false, writeSvc9, exitFail, "error", []string{"pkg/svc.go"}},
		{"operational", "lifecycle:\n  tier: operational\n", false, writeSvc9, exitFail, "error", []string{"pkg/svc.go"}},
		{"poc is a warning", "lifecycle:\n  tier: poc\n", false, writeSvc9, exitOK, "warning", []string{"pkg/svc.go"}},
		{"prototype is a warning", "lifecycle:\n  tier: prototype\n", false, writeSvc9, exitOK, "warning", []string{"pkg/svc.go"}},
		{"mvp is a warning", "lifecycle:\n  tier: mvp\n", false, writeSvc9, exitOK, "warning", []string{"pkg/svc.go"}},
		{"mvp with the safety bundle is an error", "lifecycle:\n  tier: mvp\n  bundles: [safety]\n", false, writeSvc9, exitFail, "error", []string{"pkg/svc.go"}},
		{"poc raised by lifecycle.raise is an error", "lifecycle:\n  tier: poc\n  raise: [health.file_below_floor]\n", false, writeSvc9, exitFail, "error", []string{"pkg/svc.go"}},
	}
}

// coverCases are the files that are new, clean or covered by the baseline.
func coverCases() []floorCase {
	const operational = "lifecycle:\n  tier: operational\n"
	const covered = operational + "health:\n  baseline: {file: model/health-baseline.json, decision: ADR-DEBT}\n"
	return []floorCase{
		{"a new file below the floor", operational, false, writeNew, exitFail, "error", []string{"pkg/new.go"}},
		{"a changed file at 10.0", operational, false, func(r *checkRepo) { r.write("pkg/svc.go", svcSource(3)) }, exitOK, "error", []string{}},
		{"covered by a baseline entry", covered, true, writeSvc9, exitOK, "error", []string{}},
		{"a new file is not covered by the entry of another", covered, true, writeNew, exitFail, "error", []string{"pkg/new.go"}},
	}
}

// checkCause fails when the first file below the floor of the check lacks the
// score 9.0, the floor 10.0 or the complex_method cause with its line.
func checkCause(t *testing.T, doc checkJSON) {
	t.Helper()
	f := doc.BelowFloor[0]
	if f.Score != 9.0 || f.Floor != 10.0 || len(f.Causes) != 1 {
		t.Fatalf("below the floor = %+v, want score 9.0, floor 10.0 and one cause", f)
	}
	if c := f.Causes[0]; c.Code != "health.complex_method" || c.Line == 0 {
		t.Errorf("cause = %+v, want complex_method with its line", c)
	}
}

// checkText fails when the text and the stderr of a check do not name the file
// below the floor at the level, and the failure where the check fails.
func checkText(t *testing.T, r *checkRepo, c floorCase) {
	t.Helper()
	_, text, errText := r.check()
	if !strings.Contains(text, "files below the floor (1, "+c.level+" at tier") || !strings.Contains(text, c.want[0]+" scores 9.0, below the floor of 10.0") {
		t.Errorf("the text lacks the file below the floor:\n%s", text)
	}
	if c.code == exitFail && !strings.Contains(errText, "1 changed file(s) are below the floor and no baseline entry covers them") {
		t.Errorf("stderr = %q", errText)
	}
}

// runFloorCase runs a case: the exit code, the files below the floor and, for a
// reported file, its level, causes and text.
func runFloorCase(t *testing.T, c floorCase) {
	t.Helper()
	r := floorSetup(t, c.extra, c.baseline)
	c.edit(r)
	code, doc, stderr := r.jsonCheck()
	if code != c.code || doc.Failed != (c.code == exitFail) || len(doc.Blocking) != 0 {
		t.Fatalf("exit %d %+v %s, want exit %d and no regression", code, doc, stderr, c.code)
	}
	if got := belowFloorPaths(doc); !reflect.DeepEqual(got, c.want) {
		t.Fatalf("below the floor %v, want %v", got, c.want)
	}
	if len(c.want) == 0 {
		return
	}
	if doc.FloorLevel != c.level {
		t.Errorf("floorLevel = %q, want %q", doc.FloorLevel, c.level)
	}
	checkCause(t, doc)
	checkText(t, r, c)
}

// enghealth check evaluates health.file_below_floor for the changed files
// (ADR-EMG-087): a file that gets below the floor, or is new below it, with no
// baseline entry covering it is an error at product and operational, from mvp
// with the safety bundle and with lifecycle.raise, and a warning below. A file
// at 10.0 and a covered file pass. Svc with nine ifs is a warn finding at every
// tier (score 9.0): no regression, only the floor.
//
// Mutation: not reading the baseline, not evaluating a new file, ignoring the
// tier or the bundle, or leaving the floor out of the failure fails a case.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158, REQ-EMG-210
func TestCheckFileBelowFloor(t *testing.T) {
	for _, c := range append(tierCases(), coverCases()...) {
		t.Run(c.name, func(t *testing.T) { runFloorCase(t, c) })
	}
}

// The delta between two commits (--base and --head) evaluates the floor of the
// files changed in the range, and SARIF carries the file as a result of the rule
// health.file_below_floor at the level of the tier.
//
// Mutation: not evaluating a file of the range, or leaving the rule out of the
// SARIF report, fails.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158, REQ-EMG-210
func TestCheckFloorOverACommitRange(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: operational\n", true)
	writeSvc9(r)
	r.commit("below the floor")
	code, doc, stderr := r.jsonCheck("--base", "HEAD~1", "--head", "HEAD")
	if code != exitFail || !reflect.DeepEqual(belowFloorPaths(doc), []string{"pkg/svc.go"}) {
		t.Fatalf("range: exit %d %+v %s", code, doc, stderr)
	}
	_, sarif, _ := r.check("--base", "HEAD~1", "--head", "HEAD", "--format", "sarif")
	for _, want := range []string{`"ruleId": "health.file_below_floor"`, `"level": "error"`, `"uri": "pkg/svc.go"`} {
		if !strings.Contains(sarif, want) {
			t.Errorf("sarif lacks %s:\n%s", want, sarif)
		}
	}
}

// A range that does not touch the file has nothing to report.
//
// Mutation: evaluating every file of the tree, not only the changed ones, fails.
//
// TRLC-LINKS: REQ-EMG-158, REQ-EMG-210
func TestCheckFloorIgnoresUnchangedFiles(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: operational\n", true)
	writeSvc9(r)
	r.commit("below the floor")
	r.write("pkg/ok.go", "package pkg\n\nfunc Ok() int { return 1 }\n")
	r.commit("clean file")
	if code, doc, _ := r.jsonCheck("--base", "HEAD~1", "--head", "HEAD"); code != exitOK || len(doc.BelowFloor) != 0 {
		t.Errorf("the next range: exit %d %+v", code, doc)
	}
}

// newBaselinedRepo is a check repository at a tier whose baseline, committed
// with the base, accepts the finding of svcSource(20) in pkg/svc.go.
func newBaselinedRepo(t *testing.T, tier string) *checkRepo {
	t.Helper()
	r := newCheckRepo(t, "lifecycle:\n  tier: "+tier+"\nhealth:\n  baseline: {file: model/health-baseline.json, decision: ADR-DEBT}\n", false)
	r.write("model/health-baseline.json", svcBaseline)
	return r
}

// lowerFloor writes a manifest whose health block has the line (a lowered floor,
// for example) and a file at 9.0, the working tree of a change that lowers the
// floor.
func lowerFloor(t *testing.T, r *checkRepo, line string) {
	t.Helper()
	data, err := os.ReadFile(r.manifest)
	if err != nil {
		t.Fatal(err)
	}
	r.write("engmod.yml", strings.Replace(string(data), "exclude: [docs]", "exclude: [docs]\n  "+line, 1))
	writeSvc9(r)
}

// operationalBase is the manifest of the base of the weakening tests.
const operationalBase = "lifecycle:\n  tier: operational\nhealth:\n  exclude: [docs]\n"

// Lowering the floor is a policy weakening (aspect floor): health.policy_weakened
// is an error at operational and fails the check, and a lowered floor clears the
// file at 9.0, which is why the weakening is a gate of its own (ADR-EMG-087).
//
// Mutation: not comparing the floor with the base fails.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-210
func TestCheckLoweredFloorIsAPolicyWeakening(t *testing.T) {
	r := newCheckRepo(t, operationalBase, true)
	lowerFloor(t, r, "floor: 9.0")
	code, doc, stderr := r.jsonCheck()
	if code != exitFail || len(doc.BelowFloor) != 0 {
		t.Fatalf("lowered floor: exit %d %+v %s, want the weakening alone", code, doc, stderr)
	}
	if len(doc.PolicyWeakened) != 1 || doc.PolicyWeakened[0].Aspect != "floor" {
		t.Errorf("policyWeakened = %+v, want the aspect floor", doc.PolicyWeakened)
	}
	if !strings.Contains(stderr, "policy weakened (floor): the floor of the file score is lowered from 10.0 to 9.0") {
		t.Errorf("stderr = %q", stderr)
	}
}

// An accepted policyDecision that the base did not name records the weakening.
//
// Mutation: not honouring the policyDecision fails.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-210
func TestCheckLoweredFloorRecordedByADecision(t *testing.T) {
	r := newCheckRepo(t, operationalBase, true)
	lowerFloor(t, r, "floor: 9.0\n  policyDecision: ADR-POLICY")
	code, doc, _ := r.jsonCheck()
	if code != exitOK || doc.PolicyDecision != "ADR-POLICY" {
		t.Errorf("recorded: exit %d %+v, want the weakening recorded by ADR-POLICY", code, doc)
	}
}

// A floor that was already lowered at the base is no weakening.
//
// Mutation: reporting the floor of the head without the base fails.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-210
func TestCheckFloorAlreadyLoweredAtTheBase(t *testing.T) {
	r := newCheckRepo(t, operationalBase+"  floor: 9.0\n", true)
	writeSvc9(r)
	if code, doc, _ := r.jsonCheck(); code != exitOK || len(doc.PolicyWeakened) != 0 {
		t.Errorf("exit %d %+v, want no weakening", code, doc)
	}
}

// Below product the weakening of the floor is a warning.
//
// Mutation: an error level at poc fails.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-210
func TestCheckLoweredFloorBelowProductIsAWarning(t *testing.T) {
	r := newCheckRepo(t, "lifecycle:\n  tier: poc\nhealth:\n  exclude: [docs]\n", true)
	lowerFloor(t, r, "floor: 9.0")
	if code, doc, _ := r.jsonCheck(); code != exitOK || doc.PolicyLevel != "warning" {
		t.Errorf("exit %d %+v, want the weakening as a warning", code, doc)
	}
}
