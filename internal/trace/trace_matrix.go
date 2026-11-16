// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// Trace integrity and the machine-readable traceability matrix. Code trace markers
// (TRLC-LINKS to a requirement, ENGMODEL-LINKS to a model element) are existence-checked
// against the model — not only shape-checked — and code is attributed to the model whose
// engmod.yml is its nearest enclosing root, so a parent model never validates or
// counts a child model's links. The consolidated trace is also emitted as a matrix.

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// Matrix is the consolidated, machine-readable traceability matrix for one model.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type Matrix struct {
	Model         string         `json:"model"`
	Requirements  []Row          `json:"requirements"`
	DanglingLinks []DanglingLink `json:"danglingLinks"`
	Orphans       []string       `json:"orphanRequirements"`
	Summary       Summary        `json:"summary"`
}

// Row is one requirement and everything that traces to it.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type Row struct {
	ID            string    `json:"id"`
	Text          string    `json:"text,omitempty"`
	Units         []string  `json:"units,omitempty"`
	Code          []CodeRef `json:"code,omitempty"`
	Verifications []string  `json:"verifications,omitempty"`
	// Result is the recorded outcome of the linked tests: "passed" when parsed
	// test-result artifacts report only passes, "failed" when any reports a
	// failure (or a blocked, partial or flaky run), "not-run" otherwise. It is
	// empty when no verification is linked.
	Result string `json:"result,omitempty"`
	// EvidenceMethods lists the methods of the linked evidence ("analysis",
	// "test") when a formal (Quint) result is among it; it is omitted for a
	// requirement without formal evidence. A passing current formal result
	// is analysis evidence; see formal_evidence.go for the roll-up rule.
	EvidenceMethods []string `json:"evidenceMethods,omitempty"`
	// Formal lists every formal result recorded for the requirement (pass,
	// failed, stale or not-run) with machine, invariant, mode and the sha256
	// of the module; omitted without formal evidence. A stale entry is
	// listed but links no verification.
	Formal []formalResult `json:"formal,omitempty"`
	// DelegatedStatus is the linked (or imported) requirement's own trace
	// status in its target repository at the resolved version.
	DelegatedStatus *DelegatedStatus `json:"delegatedStatus,omitempty"`
	Provenance      *Provenance      `json:"provenance,omitempty"`
	Status          string           `json:"status"`
	// Gated is true when every linked test of the requirement is
	// environment gated: it skips unless a variable is set or under
	// testing.Short. Gates names the variables (or testing.Short).
	Gated bool     `json:"gated,omitempty"`
	Gates []string `json:"gates,omitempty"`
	// BenchmarkOnly is true when every linked test of the requirement is a
	// benchmark (Go Benchmark*, Rust #[bench]); verification.benchmark_only
	// warns when its criteria name a number.
	BenchmarkOnly bool `json:"benchmarkOnly,omitempty"`
	// Parents are the upward trace of the requirement: the contract items
	// it realizes (alias::ID on a published requirement row); none for a
	// requirement traced by its links or derivation (D22). Children are the downward trace of a
	// requirement of a composing system: each linked child contract item
	// (alias::CAP-*) followed by its realizing requirements (alias::REQ-*).
	Parents  []string `json:"parents,omitempty"`
	Children []string `json:"children,omitempty"`
	// Links are the requirement links whose parent the requirement is,
	// with their evaluated state.
	Links []LinkRef `json:"links,omitempty"`
	// SatisfiedBy is "links" when the requirement has no code or test of
	// its own and every link is accepted, evaluates unchanged and resolves
	// to a child item whose realizing requirements are all implemented
	// (LinksImplemented) or all test linked (LinksTestLinked) in their own
	// repository; "links-unreviewed" when a link would roll it up but is
	// claimed, version-mismatched or otherwise not unchanged, so it does
	// not count.
	SatisfiedBy      string `json:"satisfiedBy,omitempty"`
	LinksImplemented bool   `json:"linksImplemented,omitempty"`
	LinksTestLinked  bool   `json:"linksTestLinked,omitempty"`
}

// LinkRef is one requirement link of a trace row: its id, child item, the
// evaluated state (unchanged, broken, version_mismatch, child_unpublished,
// unresolved, local) and the review status (claimed, accepted).
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type LinkRef struct {
	ID     string `json:"id"`
	Child  string `json:"child"`
	State  string `json:"state"`
	Status string `json:"status"`
}

// Provenance identifies the selected dependency publication that supplied
// an imported requirement.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
type Provenance struct {
	Alias       string `json:"alias"`
	ModulePath  string `json:"modulePath"`
	Version     string `json:"version"`
	SourceModel string `json:"sourceModel"`
	Publication string `json:"publication"`
	SourceID    string `json:"sourceId"`
}

// DelegatedStatus is the trace status of a linked child requirement
// computed in the dependency itself. Status is "implemented",
// "test-linked" or "orphan" as in that repository's own matrix,
// "not-available" when the dependency has no code sources, and
// "unknown-requirement" when the target requirement does not exist there.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
type DelegatedStatus struct {
	Requirement string `json:"requirement"`
	Status      string `json:"status"`
	Implemented bool   `json:"implemented"`
	TestLinked  bool   `json:"testLinked"`
	Result      string `json:"result,omitempty"`
}

// CodeRef is a code symbol that implements a requirement.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type CodeRef struct {
	Symbol string `json:"symbol"`
	Path   string `json:"path"`
	Line   int    `json:"line,omitempty"`
}

// DanglingLink is a code trace link that resolves to nothing in the model.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
type DanglingLink struct {
	Kind   string `json:"kind"` // requirement | model
	Target string `json:"target"`
	From   string `json:"from"` // file:line
}

// Summary is the headline coverage roll-up.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type Summary struct {
	Requirements int `json:"requirements"`
	Implemented  int `json:"implemented"`
	// Tested counts requirements with at least one linked test function (or
	// verification artifact).
	Tested int `json:"tested"`
	// TestLinked equals Tested under the name that says what it counts: a
	// requirement linked from a test entry point, run or not.
	TestLinked int `json:"testLinked"`
	// Passed counts requirements whose linked verification has a recorded pass
	// and no recorded failure in parsed test-result artifacts.
	Passed int `json:"passed"`
	// Delegated is kept for the summary shape; since ADR-EMG-078 no row is
	// delegated by an allocation, and requirement links report their own
	// counts (LinksClaimed, LinksNeedReview) and DelegatedStatus.
	Delegated     int `json:"delegated"`
	Orphan        int `json:"orphan"`
	DanglingLinks int `json:"danglingLinks"`
	// GatedOnly counts the test-linked requirements whose every linked test
	// is environment gated (Row.Gated); GatedOnlyRequirements lists them.
	// They are still counted in TestLinked.
	GatedOnly             int      `json:"gatedOnly"`
	GatedOnlyRequirements []string `json:"gatedOnlyRequirements,omitempty"`
	// LinksNeedReview counts the requirement links that need a person
	// (decision D23): pinned to another version than the dependency
	// (LinksVersionMismatch), broken (LinksBroken: the item does not exist
	// at the dependency pin) or claimed (LinksClaimed: not yet reviewed); a
	// link is counted once; no link is compared with content at check time.
	// Unparented counts the local requirements that trace upward to nothing
	// (requirement.no_parent where the requirement link rules apply).
	LinksNeedReview      int `json:"linksNeedReview"`
	LinksVersionMismatch int `json:"linksVersionMismatch"`
	LinksBroken          int `json:"linksBroken"`
	LinksClaimed         int `json:"linksClaimed"`
	Unparented           int `json:"unparented"`
}

// requirementIDSet is the set of requirement ids defined by the model.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func requirementIDSet(requirements model.RequirementsDocument) map[string]bool {
	ids := map[string]bool{}
	for _, r := range requirements.Requirements {
		ids[strings.TrimSpace(r.ID)] = true
	}
	return ids
}

// addMachineIDs adds the IDs of the machines, their states and their
// transitions.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-MODEL-LOADER
func addMachineIDs(a model.AuthoredArchitecture, add func(string)) {
	for _, m := range a.Machines {
		add(m.ID)
	}
	for _, x := range a.MachineStates() {
		add(x.ID)
	}
	for _, x := range a.MachineTransitions() {
		add(x.ID)
	}
}

// KnownModelIDs is every id the model defines (catalog, authored architecture, and
// requirements) — the oracle against which an ENGMODEL-LINKS target is resolved.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-MODEL-LOADER
func KnownModelIDs(bundle model.Bundle, requirements model.RequirementsDocument) map[string]bool {
	ids := idCollector{}
	a := bundle.Architecture.AuthoredArchitecture
	ids.addAll(a.FunctionalGroups, a.FunctionalUnits, a.Actors, a.AttackVectors, a.ReferencedElements,
		a.Interfaces, a.DataObjects, a.DeploymentTargets, a.HardwareItems, a.HardwareInterfaces,
		a.Controls, a.Risks, a.TrustBoundaries)
	addMachineIDs(a, ids.add)
	ids.addAll(a.Events)
	for _, x := range a.Flows {
		ids.add(x.ID)
		for _, s := range x.Steps {
			ids.add(x.ID + "::" + strings.TrimSpace(s.ID))
		}
	}
	ids.addAll(a.ControlVerifications, a.POAMItems, a.ThreatScenarios, a.ThreatAssumptions,
		a.ThreatOutOfScope, a.ThreatMitigations)
	c := bundle.Catalog.Catalog
	ids.addAll(c.Systems, c.FunctionalGroups, c.FunctionalUnits, c.ReferencedElements, c.Actors,
		c.AttackVectors, c.Events, c.States, c.Features, c.Modes, c.Conditions, c.DataTerms,
		requirements.Requirements)
	return ids
}

// idCollector is a set of trimmed, non-empty model ids.
type idCollector map[string]bool

// add adds one id unless it is empty.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-MODEL-LOADER
func (c idCollector) add(id string) {
	if id = strings.TrimSpace(id); id != "" {
		c[id] = true
	}
}

// addAll adds the ID field of every element of each given slice.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-MODEL-LOADER
func (c idCollector) addAll(lists ...any) {
	for _, list := range lists {
		items := reflect.ValueOf(list)
		for i := 0; i < items.Len(); i++ {
			c.add(items.Index(i).FieldByName("ID").String())
		}
	}
}

// EffectiveCodeRoots mirrors codescan.InferCodeItems' root derivation: the explicit code-root
// option, or the module root when it is not set. Scoping must use the same roots
// the scan used, otherwise items scanned via inferenceHints (with no --code-root) escape it.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-MODEL-LOADER
func EffectiveCodeRoots(bundle model.Bundle, codeRootOption string) []string {
	roots := []string{}
	if strings.TrimSpace(codeRootOption) != "" {
		roots = append(roots, codeRootOption)
	}
	return codescan.WithDefaultCodeRoot(bundle, codescan.UniqueExistingSources(roots))
}

// ScopeCodeToModel keeps only code that belongs to the model rooted at modelDir: a file
// belongs to the model whose engmod.yml is its nearest enclosing directory. This
// stops a parent model (scanned with a broad code root) from claiming a nested model's code.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func ScopeCodeToModel(items []codescan.CodeItem, roots []string, modelDir string) []codescan.CodeItem {
	if len(roots) == 0 {
		return items
	}
	modelDirAbs, _ := filepath.Abs(modelDir)
	modelRoots := modelRootsOf(roots)
	if len(modelRoots) == 0 {
		return items
	}
	out := make([]codescan.CodeItem, 0, len(items))
	for _, it := range items {
		if belongsToModel(it, modelRoots, modelDirAbs) {
			out = append(out, it)
		}
	}
	return out
}

// modelRootsOf is the set of model roots the code roots lie in or hold: the
// directory with an engmod.yml enclosing a file root, and every directory
// with an engmod.yml below a root.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func modelRootsOf(roots []string) map[string]bool {
	modelRoots := map[string]bool{}
	for _, r := range roots {
		ra, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		addEnclosingModelRoot(modelRoots, ra)
		_ = filepath.WalkDir(ra, nestedModelRootWalker(modelRoots))
	}
	return modelRoots
}

// addEnclosingModelRoot adds the nearest directory with an engmod.yml above
// a file root.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func addEnclosingModelRoot(modelRoots map[string]bool, root string) {
	if info, err := os.Stat(root); err != nil || info.IsDir() {
		return
	}
	for dir := filepath.Dir(root); ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(filepath.Join(dir, "engmod.yml")); err == nil && !info.IsDir() {
			modelRoots[dir] = true
			return
		}
		if filepath.Dir(dir) == dir {
			return
		}
	}
}

// nestedModelRootWalker adds each directory below the walked root that holds
// an engmod.yml, skipping the directories a code scan skips.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func nestedModelRootWalker(modelRoots map[string]bool) fs.WalkDirFunc {
	return func(p string, d fs.DirEntry, werr error) error {
		switch {
		case werr != nil:
		case d.IsDir() && codescan.SkipScanDir(d.Name()):
			return filepath.SkipDir
		case !d.IsDir() && d.Name() == "engmod.yml":
			modelRoots[filepath.Dir(p)] = true
		}
		return nil
	}
}

// belongsToModel reports whether a code item is kept for the model at
// modelDirAbs. Trace-bearing items carry an unambiguous absolute path
// captured at scan time; items without one (no trace links) are kept
// regardless.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func belongsToModel(it codescan.CodeItem, modelRoots map[string]bool, modelDirAbs string) bool {
	fileAbs := strings.TrimSpace(it.AbsPath)
	if fileAbs == "" {
		return true
	}
	nearest := nearestModelRoot(modelRoots, fileAbs)
	return nearest == "" || nearest == modelDirAbs
}

// nearestModelRoot is the longest model root that holds the file, or "".
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func nearestModelRoot(modelRoots map[string]bool, fileAbs string) string {
	best := ""
	for r := range modelRoots {
		holds := fileAbs == r || strings.HasPrefix(fileAbs, r+string(filepath.Separator))
		if holds && len(r) > len(best) {
			best = r
		}
	}
	return best
}

// Evidence is what the trace of a model is checked against: its
// requirements, the code scoped to the model and the verification checks.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type Evidence struct {
	Requirements model.RequirementsDocument
	Code         []codescan.CodeItem
	Verification []codescan.VerificationCheck
}

// ValidateIntegrity rejects dangling code trace links (a TRLC-LINKS or ENGMODEL-LINKS
// that resolves to nothing in the model) and reports orphan requirements (traced by no
// code or verification). The code is assumed already scoped to this model.
// It also reports assurance records whose status lags the tests they name
// (verification.status_stale).
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-073, REQ-EMG-098, REQ-EMG-121, REQ-EMG-128, REQ-EMG-146
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func ValidateIntegrity(bundle model.Bundle, evidence Evidence) []validate.Diagnostic {
	evidence.Verification = effectiveVerification(codescan.ModelRootDir(bundle), evidence.Verification)
	diags := validateIntegrityWithIDs(KnownModelIDs(bundle, evidence.Requirements), evidence)
	diags = append(diags, unsatisfiedRequirements(bundle, evidence)...)
	diags = append(diags, benchmarkOnlyDiagnostics(evidence)...)
	bundle.Requirements = evidence.Requirements
	return append(diags, integrityTail(bundle, evidence.Code, evidence.Verification)...)
}

// tierVerificationDiagnostics reports, in a model with lifecycle only,
// verification.failed (warning) at each requirement whose linked test
// results include a failure and verification.gated_only (warning) at each
// requirement whose every linked test is environment gated; the tier policy
// sets their severity. A failing current formal (Quint) result fails the
// requirement like a failing test (formal_evidence.go): the message then
// names the machine and invariant; the message for a requirement without a
// formal failure is unchanged.
// TRLC-LINKS: REQ-EMG-128, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func tierVerificationDiagnostics(bundle model.Bundle, requirements model.RequirementsDocument, verification []codescan.VerificationCheck) []validate.Diagnostic {
	if _, _, ok := validate.EffectiveTier(bundle); !ok {
		return nil
	}
	baseDir := codescan.ModelRootDir(bundle)
	verification = effectiveVerification(baseDir, verification)
	results := requirementResults(verification)
	formalFailures := formalFailureReasons(baseDir, verification)
	testResults := results
	if len(formalFailures) > 0 {
		testResults = requirementResults(testChecksOf(verification))
	}
	gated, ungated := splitVerified(verification, func(v codescan.VerificationCheck, req string) bool { return len(v.Gates[req]) > 0 })
	var diags []validate.Diagnostic
	for i, r := range requirements.Requirements {
		id := strings.TrimSpace(r.ID)
		path := fmt.Sprintf("requirements[%d]", i)
		if results[id] == "failed" {
			message := failedVerificationMessage(id, formalFailures[id], testResults[id] == "failed")
			diags = append(diags, validate.Diagnostic{Code: "verification.failed", Severity: validate.SeverityWarning, Path: path, Message: message})
		}
		if gated[id] && !ungated[id] {
			diags = append(diags, validate.Diagnostic{Code: "verification.gated_only", Severity: validate.SeverityWarning, Path: path,
				Message: fmt.Sprintf("requirement %s is verified only by environment-gated tests, which skip in an ordinary run; add an ungated test or run the gated ones in CI and record the result", id)})
		}
	}
	return diags
}

// formalFailureReasons names, by requirement, the invariant and machine of
// each failing current formal result.
// TRLC-LINKS: REQ-EMG-128, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func formalFailureReasons(baseDir string, verification []codescan.VerificationCheck) map[string][]string {
	failures := map[string][]string{}
	for _, f := range formalResultsOf(baseDir, verification) {
		if f.Result == formalFailed {
			failures[f.Requirement] = append(failures[f.Requirement], fmt.Sprintf("invariant %s of machine %s", textutil.NonEmpty(f.Invariant, "(unnamed)"), textutil.NonEmpty(f.Machine, "(unnamed)")))
		}
	}
	return failures
}

// testChecksOf is the checks that are not analysis (formal) checks.
// TRLC-LINKS: REQ-EMG-128, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func testChecksOf(verification []codescan.VerificationCheck) []codescan.VerificationCheck {
	var testChecks []codescan.VerificationCheck
	for _, v := range verification {
		if v.Method != "analysis" {
			testChecks = append(testChecks, v)
		}
	}
	return testChecks
}

// failedVerificationMessage is the message of verification.failed for a
// requirement; failures are its failing formal results and testFailed tells
// whether a test failed too.
// TRLC-LINKS: REQ-EMG-128, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func failedVerificationMessage(id string, failures []string, testFailed bool) string {
	message := fmt.Sprintf("requirement %s has a failed, blocked, partial or flaky linked test result; fix the code or the test and record a passing run", id)
	failures = textutil.UniqueSorted(failures)
	if len(failures) == 0 {
		return message
	}
	formal := fmt.Sprintf("formal result failed: %s; fix the model or the invariant and rerun engquint check", strings.Join(failures, ", "))
	if testFailed {
		return message + "; " + formal
	}
	return fmt.Sprintf("requirement %s has a failed %s", id, formal)
}

// splitVerified partitions the requirements the checks verify by whether the
// predicate holds for the check and the requirement as it is written there.
// TRLC-LINKS: REQ-EMG-121
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
func splitVerified(verification []codescan.VerificationCheck, holds func(v codescan.VerificationCheck, req string) bool) (yes, no map[string]bool) {
	yes, no = map[string]bool{}, map[string]bool{}
	sets := map[bool]map[string]bool{true: yes, false: no}
	for _, v := range verification {
		for _, r := range v.Verifies {
			sets[holds(v, r)][strings.TrimSpace(r)] = true
		}
	}
	return yes, no
}

// benchmarkOnlyRequirements returns the requirements whose every linking
// test entry point, across all verification checks, is a benchmark.
// TRLC-LINKS: REQ-EMG-121
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
func benchmarkOnlyRequirements(verification []codescan.VerificationCheck) map[string]bool {
	bench, asserted := splitVerified(verification, func(v codescan.VerificationCheck, req string) bool { return v.BenchOnly[req] })
	for id := range asserted {
		delete(bench, id)
	}
	return bench
}

// numberPattern finds a number in verification criteria.
var numberPattern = regexp.MustCompile(`\d`)

// benchmarkOnlyDiagnostics reports verification.benchmark_only (warning) at
// each requirement whose verificationCriteria name a number and whose only
// linked tests are benchmarks: a benchmark measures but asserts nothing, so
// a quantitative criterion needs a test that checks the measured value.
// TRLC-LINKS: REQ-EMG-121
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func benchmarkOnlyDiagnostics(evidence Evidence) []validate.Diagnostic {
	bench := benchmarkOnlyRequirements(evidence.Verification)
	var diags []validate.Diagnostic
	for i, r := range evidence.Requirements.Requirements {
		id := strings.TrimSpace(r.ID)
		if !bench[id] || !numberPattern.MatchString(r.VerificationCriteria) {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "verification.benchmark_only", Severity: validate.SeverityWarning, Path: fmt.Sprintf("requirements[%d]", i),
			Message: fmt.Sprintf("requirement %s has a quantitative verificationCriteria but only benchmarks link it; a benchmark asserts nothing: add a Test that measures and checks the value from the criteria", id),
		})
	}
	return diags
}

// unsatisfiedRequirements reports requirement.unsatisfied at each requirement
// of a composing system (a model with composition subsystems) that has no
// implementing code or test of its own, is the parent of no requirement link
// and is not derived. Severity and applicability follow
// requirementLinks (validate.RequirementLinkRules).
// TRLC-LINKS: REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func unsatisfiedRequirements(bundle model.Bundle, evidence Evidence) []validate.Diagnostic {
	severity, applies := validate.RequirementLinkRules(bundle)
	if !applies || len(bundle.Architecture.Composition.Subsystems) == 0 {
		return nil
	}
	satisfied := satisfiedRequirements(bundle, evidence)
	var diags []validate.Diagnostic
	for i, r := range evidence.Requirements.Requirements {
		id := strings.TrimSpace(r.ID)
		if satisfied[id] || r.Derived {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "requirement.unsatisfied", Severity: severity, Path: fmt.Sprintf("requirements[%d]", i),
			Message: fmt.Sprintf("requirement %s is satisfied by nothing: no code or test of this model links it, no reqLinks entry names it as parent, and it is not derived", id),
		})
	}
	return diags
}

// satisfiedRequirements is the requirements something satisfies: scoped code
// that implements them, a verification, or a requirement link naming them as
// parent.
// TRLC-LINKS: REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func satisfiedRequirements(bundle model.Bundle, evidence Evidence) map[string]bool {
	satisfied := verifiedRequirements(evidence.Verification)
	for _, it := range evidence.Code {
		for _, r := range it.Implements {
			satisfied[strings.TrimSpace(r)] = true
		}
	}
	for _, link := range bundle.ReqLinks.ReqLinks {
		satisfied[strings.TrimSpace(link.Parent)] = true
	}
	return satisfied
}

// verifiedRequirements is the requirements any check verifies.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func verifiedRequirements(verification []codescan.VerificationCheck) map[string]bool {
	verified, _ := splitVerified(verification, func(codescan.VerificationCheck, string) bool { return true })
	return verified
}

// validateIntegrityWithIDs is ValidateIntegrity against an explicit
// set of model IDs, which may include published alias::ID entities.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func validateIntegrityWithIDs(modelIDs map[string]bool, evidence Evidence) []validate.Diagnostic {
	reqIDs := requirementIDSet(evidence.Requirements)
	var diags []validate.Diagnostic
	for _, it := range evidence.Code {
		for _, link := range danglingLinks(it, reqIDs, modelIDs) {
			diags = append(diags, link.diagnostic())
		}
	}
	return append(diags, orphanDiagnostics(evidence, reqIDs)...)
}

// orphanDiagnostics reports requirement.orphan at each requirement that no
// scoped code or verification traces.
// Verification is not path-scoped like code, but this is sound: a requirement id is
// unique to its model, so a foreign model's verification can only mark its own
// (differently-prefixed) requirement ids, never one of this model's requirements.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func orphanDiagnostics(evidence Evidence, reqIDs map[string]bool) []validate.Diagnostic {
	covered, verified := implementedRequirements(evidence.Code, reqIDs), verifiedRequirements(evidence.Verification)
	var diags []validate.Diagnostic
	for _, r := range evidence.Requirements.Requirements {
		id := strings.TrimSpace(r.ID)
		if covered[id] || verified[id] || exemptFromOrphan(r.Status) {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "requirement.orphan", Severity: validate.SeverityWarning,
			Message: fmt.Sprintf("requirement %s is traced by no code or verification", id),
			Path:    orphanPath(id),
		})
	}
	return diags
}

// exemptFromOrphan reports whether a requirement status exempts it from
// requirement.orphan: a retired or rejected requirement is not expected to
// have code, as it is exempt from lifecycle.requirement_unverified.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func exemptFromOrphan(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return status == "retired" || status == "rejected"
}

// implementedRequirements is the requirements of the model that scoped code
// links.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func implementedRequirements(code []codescan.CodeItem, reqIDs map[string]bool) map[string]bool {
	covered := map[string]bool{}
	for _, it := range code {
		for _, link := range filterLinks(it.Implements, reqIDs, true) {
			covered[link] = true
		}
	}
	return covered
}

// danglingLinks is the TRLC-LINKS and ENGMODEL-LINKS of a code item that
// resolve to nothing in the model, requirement links first.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func danglingLinks(it codescan.CodeItem, reqIDs, modelIDs map[string]bool) []DanglingLink {
	var dangling []DanglingLink
	for _, link := range unknownLinks(it.Implements, reqIDs) {
		dangling = append(dangling, DanglingLink{Kind: "requirement", Target: link, From: it.Source})
	}
	for _, link := range unknownLinks(it.ModelLinks, modelIDs) {
		dangling = append(dangling, DanglingLink{Kind: "model", Target: link, From: it.Source})
	}
	return dangling
}

// unknownLinks is the trimmed, non-empty links that are not known.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func unknownLinks(links []string, known map[string]bool) []string {
	return filterLinks(links, known, false)
}

// filterLinks is the trimmed, non-empty links that are known (when want is
// true) or not known (when it is false).
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func filterLinks(links []string, known map[string]bool, want bool) []string {
	var out []string
	for _, raw := range links {
		if link := strings.TrimSpace(raw); link != "" && known[link] == want {
			out = append(out, link)
		}
	}
	return out
}

// diagnostic is the error that reports a dangling link.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func (l DanglingLink) diagnostic() validate.Diagnostic {
	if l.Kind == "requirement" {
		return validate.Diagnostic{
			Code: "code.dangling_requirement_link", Severity: validate.SeverityError,
			Message: fmt.Sprintf("TRLC-LINKS %q resolves to no requirement in the model", l.Target),
			Path:    l.From,
		}
	}
	return validate.Diagnostic{
		Code: "code.dangling_model_link", Severity: validate.SeverityError,
		Message: fmt.Sprintf("ENGMODEL-LINKS %q resolves to no model element", l.Target),
		Path:    l.From,
	}
}

// buildTraceMatrix assembles the consolidated traceability matrix for the model.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func buildTraceMatrix(bundle model.Bundle, evidence Evidence) Matrix {
	return buildTraceMatrixWithIDs(bundle, KnownModelIDs(bundle, evidence.Requirements), evidence)
}

// buildTraceMatrixWithIDs is buildTraceMatrix resolving ENGMODEL-LINKS against
// an explicit ID set, which may include published alias::ID entities. A row
// whose every linked verification is environment gated is Gated and counted
// in Summary.GatedOnly. Formal (Quint) evidence rolls up by the rule in
// formal_evidence.go: a stale result links no verification and shows only in
// Row.Formal; a current one links the formal check, adds "analysis" to
// Row.EvidenceMethods and takes part in Row.Result like a test result. Status
// inference is unchanged: code gives implemented, any
// linked verification (formal included) verified, else orphan.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-065, REQ-EMG-088, REQ-EMG-105, REQ-EMG-121, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func buildTraceMatrixWithIDs(bundle model.Bundle, modelIDs map[string]bool, evidence Evidence) Matrix {
	baseDir := codescan.ModelRootDir(bundle)
	evidence.Verification = effectiveVerification(baseDir, evidence.Verification)
	reqIDs := requirementIDSet(evidence.Requirements)
	sources := newRowSources(evidence.Verification, formalByRequirement(formalResultsOf(baseDir, evidence.Verification)))
	var dangling []DanglingLink
	for _, it := range evidence.Code {
		dangling = append(dangling, danglingLinks(it, reqIDs, modelIDs)...)
		sources.addCode(it, reqIDs)
	}
	matrix := Matrix{Model: strings.TrimSpace(bundle.Architecture.Model.ID)}
	for _, r := range evidence.Requirements.Requirements {
		row := sources.row(r)
		if row.Status == "orphan" {
			matrix.Orphans = append(matrix.Orphans, row.ID)
		}
		matrix.Requirements = append(matrix.Requirements, row)
	}
	sort.SliceStable(dangling, func(i, j int) bool {
		if dangling[i].From != dangling[j].From {
			return dangling[i].From < dangling[j].From
		}
		return dangling[i].Target < dangling[j].Target
	})
	matrix.DanglingLinks = dangling
	for _, row := range matrix.Requirements {
		matrix.Summary.count(row)
	}
	matrix.Summary.DanglingLinks = len(dangling)
	matrix.Summary.Unparented = countUnparented(bundle, evidence.Requirements)
	matrix.Summary.TestLinked = matrix.Summary.Tested
	return matrix
}

// rowSources is what the rows of a matrix are filled from, by requirement
// id: code, linked verification and its gates, evidence methods, results and
// formal results.
type rowSources struct {
	code          map[string][]CodeRef
	verifications map[string][]string
	gates         map[string][]string
	ungated       map[string]bool
	analysis      map[string]bool
	test          map[string]bool
	benchOnly     map[string]bool
	results       map[string]string
	formal        map[string][]formalResult
}

// newRowSources indexes the verification checks and the formal results by
// requirement.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-065, REQ-EMG-088, REQ-EMG-105, REQ-EMG-121, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func newRowSources(verification []codescan.VerificationCheck, formal map[string][]formalResult) rowSources {
	s := rowSources{
		code: map[string][]CodeRef{}, verifications: map[string][]string{}, gates: map[string][]string{},
		ungated: map[string]bool{}, analysis: map[string]bool{}, test: map[string]bool{},
		benchOnly: benchmarkOnlyRequirements(verification), results: requirementResults(verification), formal: formal,
	}
	for _, v := range verification {
		for _, r := range v.Verifies {
			s.addVerification(v, r)
		}
	}
	return s
}

// addVerification records that a check verifies requirement r.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-065, REQ-EMG-088, REQ-EMG-105, REQ-EMG-121, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func (s rowSources) addVerification(v codescan.VerificationCheck, r string) {
	id := strings.TrimSpace(r)
	if v.Method == "analysis" {
		s.analysis[id] = true
	} else {
		s.test[id] = true
	}
	s.verifications[id] = append(s.verifications[id], strings.TrimSpace(v.ID))
	if g := v.Gates[r]; len(g) > 0 {
		s.gates[id] = textutil.AppendUniqueSlice(s.gates[id], g)
	} else {
		s.ungated[id] = true
	}
}

// addCode records the production code a code item links to requirements of
// the model.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-065, REQ-EMG-088, REQ-EMG-105, REQ-EMG-121, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func (s rowSources) addCode(it codescan.CodeItem, reqIDs map[string]bool) {
	if codescan.IsVerificationCodeItem(it) {
		return
	}
	path, line := codescan.SplitSourceLine(it.Source)
	for _, raw := range it.Implements {
		if link := strings.TrimSpace(raw); link != "" && reqIDs[link] {
			s.code[link] = append(s.code[link], CodeRef{Symbol: it.Element, Path: path, Line: line})
		}
	}
}

// row is the matrix row of a requirement.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-065, REQ-EMG-088, REQ-EMG-105, REQ-EMG-121, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func (s rowSources) row(r model.Requirement) Row {
	id := strings.TrimSpace(r.ID)
	row := Row{ID: id, Text: strings.TrimSpace(r.Text), Units: textutil.UniqueSorted(r.AppliesTo), Parents: requirementParents(r)}
	row.Code = s.code[id]
	row.Verifications = textutil.UniqueSorted(s.verifications[id])
	if len(row.Verifications) > 0 {
		row.Result = s.results[id]
		if !s.ungated[id] && len(s.gates[id]) > 0 {
			row.Gated, row.Gates = true, textutil.UniqueSorted(s.gates[id])
		}
		row.BenchmarkOnly = s.benchOnly[id]
	}
	if len(s.formal[id]) > 0 {
		row.Formal = s.formal[id]
		row.EvidenceMethods = s.evidenceMethods(id)
	}
	switch {
	case len(row.Code) > 0:
		row.Status = "implemented"
	case len(row.Verifications) > 0:
		row.Status = "verified"
	default:
		row.Status = "orphan"
	}
	return row
}

// evidenceMethods lists "analysis" and, when a test links the requirement
// too, "test"; none without analysis evidence.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-065, REQ-EMG-088, REQ-EMG-105, REQ-EMG-121, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func (s rowSources) evidenceMethods(id string) []string {
	if !s.analysis[id] {
		return nil
	}
	if s.test[id] {
		return []string{"analysis", "test"}
	}
	return []string{"analysis"}
}

// count adds one row to the headline roll-up.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-065, REQ-EMG-088, REQ-EMG-105, REQ-EMG-121, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func (s *Summary) count(row Row) {
	s.Requirements++
	if len(row.Code) > 0 {
		s.Implemented++
	}
	if len(row.Verifications) > 0 {
		s.Tested++
	}
	if row.Gated {
		s.GatedOnly++
		s.GatedOnlyRequirements = append(s.GatedOnlyRequirements, row.ID)
	}
	if row.Result == "passed" {
		s.Passed++
	}
	if row.Status == "orphan" {
		s.Orphan++
	}
}

// requirementResults rolls the per-requirement results of every linked
// verification into one row result: failed > passed > not-run.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func requirementResults(verification []codescan.VerificationCheck) map[string]string {
	out := map[string]string{}
	for _, v := range verification {
		for _, req := range v.Verifies {
			recordResult(out, strings.TrimSpace(req), checkResult(v, strings.TrimSpace(req)))
		}
	}
	return out
}

// recordResult sets the result of a requirement unless a stronger one is
// recorded: failed > passed > not-run.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func recordResult(out map[string]string, id, result string) {
	rank := map[string]int{"not-run": 0, "passed": 1, "failed": 2}
	if current, ok := out[id]; !ok || rank[result] > rank[current] {
		out[id] = result
	}
}

// checkResult is the result of a check for requirement id: the last recorded
// result for it, as "passed" or "failed", else "not-run".
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func checkResult(v codescan.VerificationCheck, id string) string {
	result := "not-run"
	for _, r := range v.Results {
		if strings.TrimSpace(r.Requirement) != id {
			continue
		}
		switch codescan.NormalizeResultStatus(r.Status) {
		case "pass":
			result = "passed"
		case "fail", "blocked", "partial", "flaky":
			result = "failed"
		}
	}
	return result
}

// Options configures BuildFromFilesWith.
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064, REQ-EMG-066
type Options struct {
	// Resolution overrides ENGMOD_WORK / ENGMOD_LOCK for the composition run.
	Resolution composition.ResolutionOptions
	// Composition, when set, is a composition already resolved for this
	// manifest; the trace then does not compose again.
	Composition *composition.CompositionResult
	// Composed, when set, receives the composition the trace used.
	Composed *composition.CompositionResult
}

// FileRequest names the files a trace is built from: the model manifest, the
// requirements document and the code root, which may be empty.
// TRLC-LINKS: REQ-EMG-030
type FileRequest struct {
	ModelPath        string
	RequirementsPath string
	CodeRoot         string
}

// BuildFromFiles loads a model, scans its code root, scopes the code to the
// model, validates trace integrity, and returns the matrix together with diagnostics.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-050, REQ-EMG-051
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-MODEL-LOADER, FU-CODEMAP-INFERENCE
func BuildFromFiles(modelPath, requirementsPath, codeRoot string) (Matrix, []validate.Diagnostic, error) {
	return BuildFromFilesWith(FileRequest{ModelPath: modelPath, RequirementsPath: requirementsPath, CodeRoot: codeRoot}, Options{})
}

// BuildFromFilesWith is BuildFromFiles with explicit
// resolution options. With composition it composes once, resolves code
// ENGMODEL-LINKS alias::ID against the dependencies' published entities
// (requirements stay local), returns the composition diagnostics, and reports
// each linked requirement's own status in its target repository.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-050, REQ-EMG-051, REQ-EMG-065, REQ-EMG-066, REQ-EMG-067, REQ-EMG-073, REQ-EMG-098, REQ-EMG-105, REQ-EMG-125
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-MODEL-LOADER, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func BuildFromFilesWith(req FileRequest, options Options) (Matrix, []validate.Diagnostic, error) {
	bundle, err := loadRequestBundle(req)
	if err != nil {
		return Matrix{}, nil, err
	}
	return buildFromBundle(bundle, req, options)
}

// loadRequestBundle loads the model of a request with its requirements.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-050, REQ-EMG-051, REQ-EMG-065, REQ-EMG-066, REQ-EMG-067, REQ-EMG-073, REQ-EMG-098, REQ-EMG-105, REQ-EMG-125
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-MODEL-LOADER, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func loadRequestBundle(req FileRequest) (model.Bundle, error) {
	bundle, err := codescan.LoadBundle(req.ModelPath)
	if err != nil {
		return model.Bundle{}, err
	}
	requirements, err := model.LoadRequirements(req.RequirementsPath)
	if err != nil {
		return model.Bundle{}, err
	}
	bundle.Requirements = requirements
	return bundle, nil
}

// buildFromBundle is BuildFromFilesWith for a loaded bundle (its
// requirements set): the trace of the bundle as given, so a caller may
// change it in memory first (TierGap sets the target tier).
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-065, REQ-EMG-066, REQ-EMG-067, REQ-EMG-098, REQ-EMG-105, REQ-EMG-125, REQ-EMG-134, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func buildFromBundle(bundle model.Bundle, req FileRequest, options Options) (Matrix, []validate.Diagnostic, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return Matrix{}, nil, err
	}
	bundle = canonical.Documents()
	requirements := bundle.Requirements
	inferred, diags := inferEvidence(bundle, req)

	compositionResult, compositionDiags := composeForTrace(bundle, options)
	diags = append(diags, compositionDiags...)
	compositionProjection := composition.BuildCompositionProjection(compositionResult)
	// Code links may name published dependency entities; requirements stay
	// local so the rows and counts below are this model's own.
	linkBundle := compositionProjection.EnrichBundle(bundle, "architecture", "behavior", "assurance", "compliance")
	linkIDs := KnownModelIDs(linkBundle, requirements)

	resolvedRoot := resolvedCodeRoot(req)
	evidence := Evidence{
		Requirements: requirements,
		Code:         ScopeCodeToModel(inferred.Code, EffectiveCodeRoots(bundle, resolvedRoot), codescan.ModelRootDir(bundle)),
		Verification: inferred.Verification,
	}
	diags = append(diags, validateIntegrityWithIDs(linkIDs, evidence)...)
	diags = append(diags, unsatisfiedRequirements(bundle, evidence)...)
	diags = append(diags, integrityTail(bundle, evidence.Code, evidence.Verification)...)
	matrix := buildTraceMatrixWithIDs(bundle, linkIDs, evidence)
	appendPublishedRequirementTraceRows(&matrix, compositionProjection)
	satisfied := applyRequirementLinks(&matrix, compositionResult, dependencyTraces{})
	diags = dropSatisfiedOrphans(diags, satisfied)
	return matrix, validate.SortDiagnostics(validate.ApplyTierPolicy(bundle, diags)), nil
}

// resolvedCodeRoot is the code root of a request, made relative to the
// model's directory unless it is empty or absolute.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-065, REQ-EMG-066, REQ-EMG-067, REQ-EMG-098, REQ-EMG-105, REQ-EMG-125, REQ-EMG-134, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func resolvedCodeRoot(req FileRequest) string {
	root := strings.TrimSpace(req.CodeRoot)
	if root != "" && !filepath.IsAbs(root) {
		return filepath.Join(filepath.Dir(req.ModelPath), root)
	}
	return root
}

// inferEvidence scans the code and infers the verification checks of the
// bundle, with the scan diagnostics. The code is not yet scoped to the model.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-065, REQ-EMG-066, REQ-EMG-067, REQ-EMG-098, REQ-EMG-105, REQ-EMG-125, REQ-EMG-134, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func inferEvidence(bundle model.Bundle, req FileRequest) (Evidence, []validate.Diagnostic) {
	resolvedRoot := resolvedCodeRoot(req)
	code, codeDiags := codescan.InferCodeItems(bundle, resolvedRoot)
	verification, verDiags := codescan.InferVerificationChecks(codescan.VerificationScan{Bundle: bundle, Requirements: bundle.Requirements, InferredCode: code, CodeRoot: resolvedRoot})
	// A stale formal result is no evidence: drop it before any roll-up.
	verification = effectiveVerification(codescan.ModelRootDir(bundle), verification)
	diags := append([]validate.Diagnostic(nil), codeDiags...)
	return Evidence{Requirements: bundle.Requirements, Code: code, Verification: verification}, append(diags, verDiags...)
}

// composeForTrace composes the bundle, hands the result to options.Composed
// and returns the composition diagnostics, with a warning when the
// composition could not be resolved.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-065, REQ-EMG-066, REQ-EMG-067, REQ-EMG-098, REQ-EMG-105, REQ-EMG-125, REQ-EMG-134, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func composeForTrace(bundle model.Bundle, options Options) (composition.CompositionResult, []validate.Diagnostic) {
	result, err := composition.ComposeForGeneration(bundle, options.Composition, options.Resolution)
	if options.Composed != nil {
		*options.Composed = result
	}
	var diags []validate.Diagnostic
	if err != nil {
		diags = append(diags, validate.Diagnostic{
			Code: "composition.resolve_failed", Severity: validate.SeverityWarning,
			Message: fmt.Sprintf("composition could not be resolved (linked requirements may appear orphan): %v", err),
			Path:    bundle.ArchitecturePath,
		})
	}
	return result, append(diags, result.Diagnostics...)
}

// dropSatisfiedOrphans drops requirement.orphan for the requirements that
// their links satisfy.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-065, REQ-EMG-066, REQ-EMG-067, REQ-EMG-098, REQ-EMG-105, REQ-EMG-125, REQ-EMG-134, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func dropSatisfiedOrphans(diags []validate.Diagnostic, satisfied map[string]bool) []validate.Diagnostic {
	if len(satisfied) == 0 {
		return diags
	}
	kept := diags[:0]
	for _, d := range diags {
		if d.Code == "requirement.orphan" && satisfied[orphanRequirement(d.Path)] {
			continue
		}
		kept = append(kept, d)
	}
	return kept
}

// dependencyTraceRows computes a resolved dependency's own trace matrix rows
// by requirement id, scanning its module directory (local replacement or
// module cache entry, minus its exclude globs) the way the root model's module
// root is scanned. ok is false when that directory holds no source code.
// TRLC-LINKS: REQ-EMG-067
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func dependencyTraceRows(child *composition.ComposedSystem) (map[string]Row, bool) {
	bundle := child.Bundle
	code, _ := codescan.InferCodeItems(bundle, "")
	verification, _ := codescan.InferVerificationChecks(codescan.VerificationScan{Bundle: bundle, Requirements: bundle.Requirements, InferredCode: code, CodeRoot: ""})
	scoped := ScopeCodeToModel(code, EffectiveCodeRoots(bundle, ""), codescan.ModelRootDir(bundle))
	if len(scoped) == 0 && len(verification) == 0 {
		return nil, false
	}
	matrix := buildTraceMatrix(bundle, Evidence{Requirements: bundle.Requirements, Code: scoped, Verification: verification})
	rows := make(map[string]Row, len(matrix.Requirements))
	for _, row := range matrix.Requirements {
		rows[row.ID] = row
	}
	return rows, true
}

// dependencyTraces caches each resolved dependency's own trace rows.
type dependencyTraces map[*composition.ComposedSystem]cachedTrace

type cachedTrace struct {
	rows map[string]Row
	ok   bool
}

// status returns the trace status of requirement in child, computing the
// child's trace at most once.
// TRLC-LINKS: REQ-EMG-067, REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func (cache dependencyTraces) status(child *composition.ComposedSystem, requirement string) *DelegatedStatus {
	if child == nil || requirement == "" {
		return nil
	}
	entry, seen := cache[child]
	if !seen {
		entry.rows, entry.ok = dependencyTraceRows(child)
		cache[child] = entry
	}
	out := &DelegatedStatus{Requirement: requirement}
	if !entry.ok {
		out.Status = "not-available"
		return out
	}
	row, exists := entry.rows[requirement]
	if !exists {
		out.Status = "unknown-requirement"
		return out
	}
	out.Implemented, out.TestLinked, out.Result = len(row.Code) > 0, len(row.Verifications) > 0, row.Result
	out.Status = row.Status
	if out.Status == "verified" {
		out.Status = "test-linked"
	}
	return out
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051, REQ-EMG-105
func appendPublishedRequirementTraceRows(matrix *Matrix, projection composition.CompositionProjection) {
	for _, entity := range projection.Domain("requirements") {
		requirement, ok := entity.Value.(model.Requirement)
		if !ok {
			continue
		}
		row := publishedRequirementRow(entity, requirement, projection)
		matrix.Requirements = append(matrix.Requirements, row)
		matrix.Summary.Requirements++
		if len(row.Verifications) > 0 {
			matrix.Summary.Tested++
			matrix.Summary.TestLinked++
		}
	}
	sort.SliceStable(matrix.Requirements, func(i, j int) bool {
		return matrix.Requirements[i].ID < matrix.Requirements[j].ID
	})
}

// publishedRequirementRow is the matrix row of a requirement a dependency
// publishes: verified when the dependency publishes a control verification
// of a control the requirement applies to, else published.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051, REQ-EMG-105
func publishedRequirementRow(entity composition.PublishedProjectionEntity, requirement model.Requirement, projection composition.CompositionProjection) Row {
	row := Row{
		ID: entity.QualifiedID, Text: strings.TrimSpace(requirement.Text),
		Units:   entity.QualifyList(requirement.AppliesTo),
		Parents: entity.QualifyList(requirementParents(requirement)),
		Provenance: &Provenance{
			Alias: entity.Provenance.Alias, ModulePath: entity.Provenance.ModulePath,
			Version: entity.Provenance.Version, SourceModel: entity.Provenance.SourceModel,
			Publication: entity.Provenance.Publication, SourceID: entity.SourceID,
		},
	}
	appliesTo := map[string]bool{}
	for _, ref := range requirement.AppliesTo {
		appliesTo[ref] = true
	}
	for _, candidate := range projection.Domain("assurance") {
		verification, ok := candidate.Value.(model.ControlVerification)
		if ok && appliesTo[verification.ControlRef] && candidate.Provenance.Alias == entity.Provenance.Alias {
			row.Verifications = append(row.Verifications, candidate.QualifiedID)
		}
	}
	row.Verifications = textutil.UniqueSorted(row.Verifications)
	row.Status = "published"
	if len(row.Verifications) > 0 {
		row.Status = "verified"
	}
	return row
}

// CSV renders the matrix as one row per requirement for spreadsheet/diff consumption.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (m Matrix) CSV() []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"requirement", "status", "units", "code", "verifications", "provenance", "result"})
	for _, r := range m.Requirements {
		_ = w.Write([]string{
			r.ID, r.Status, strings.Join(r.Units, " "), csvCode(r),
			strings.Join(r.Verifications, " "), csvProvenance(r), r.Result,
		})
	}
	w.Flush()
	return buf.Bytes()
}

// csvCode lists the code locations of a row, path:line where the line is known.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func csvCode(r Row) string {
	code := make([]string, 0, len(r.Code))
	for _, c := range r.Code {
		if c.Line > 0 {
			code = append(code, fmt.Sprintf("%s:%d", c.Path, c.Line))
		} else {
			code = append(code, c.Path)
		}
	}
	return strings.Join(code, " ")
}

// csvProvenance renders the provenance of a row as alias:publication@version.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func csvProvenance(r Row) string {
	if r.Provenance == nil {
		return ""
	}
	return r.Provenance.Alias + ":" + r.Provenance.Publication + "@" + r.Provenance.Version
}
