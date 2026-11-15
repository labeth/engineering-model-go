// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codemap"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type VerificationResult struct {
	Requirement string
	Status      string
	Evidence    string
	Notes       string
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type VerificationCheck struct {
	ID   string
	Name string
	Kind string
	// Method is the verification method when it is not a test: "analysis"
	// for a sha-verified formal result; empty means test.
	Method        string
	Status        string
	Description   string
	Verifies      []string
	CodeElements  []string
	DerivedOwners []string
	Evidence      []string
	Results       []VerificationResult
	// Gates maps a requirement to the environment gates of its tests in this
	// check when every test entry point linking it is gated (see
	// codemap.Symbol.Gates); a requirement with an ungated test is absent.
	Gates map[string][]string
	// BenchOnly holds each requirement whose every test entry point linking
	// it in this check is a benchmark (codemap.Symbol.Benchmark).
	BenchOnly map[string]bool
}

var requirementIDRe = regexp.MustCompile(`\bREQ-[A-Za-z0-9-]+\b`)
var trlcLinksMarkerRe = regexp.MustCompile(`(?i)^\s*(?://+|#|--|/\*+|\*)?\s*TRLC-LINKS:\s*(.+?)\s*(?:\*/)?\s*$`)
var resultStatusRe = regexp.MustCompile(`(?i)\b(pass|fail|partial|blocked|not-run|flaky)\b`)

// VerificationScan is the input of InferVerificationChecks: the model bundle,
// its requirements, the inferred code items and the code root option.
type VerificationScan struct {
	Bundle       model.Bundle
	Requirements model.RequirementsDocument
	InferredCode []CodeItem
	CodeRoot     string
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func InferVerificationChecks(scan VerificationScan) ([]VerificationCheck, []validate.Diagnostic) {
	bundle, requirements, inferredCode, codeRootOption := scan.Bundle, scan.Requirements, scan.InferredCode, scan.CodeRoot
	baseDir := ModelRootDir(bundle)
	testRoots := verificationTestRoots(baseDir, bundle, codeRootOption)
	resultRoots := VerificationResultRoots(baseDir, codeRootOption)

	inf := &verificationInference{
		baseDir:            baseDir,
		reqOwners:          requirementOwners(requirements.Requirements),
		skip:               codeScanSkipper(bundle),
		diags:              []validate.Diagnostic{},
		checkByID:          map[string]*VerificationCheck{},
		checkIDsByIdentity: map[string][]string{},
		checkIdentityByID:  map[string]map[string]bool{},
	}
	inf.indexCodeElements(inferredCode, testRoots)
	for _, root := range testRoots {
		inf.scanTestRoot(root)
	}
	artifacts := inf.collectArtifacts(resultRoots)
	for _, a := range artifacts {
		inf.attachArtifact(a)
	}
	out := inf.finalChecks()
	if len(out) == 0 {
		inf.diags = append(inf.diags, validate.Diagnostic{
			Code:     "verification.no_artifacts_found",
			Severity: validate.SeverityWarning,
			Message:  "no verification artifacts found under inferred tests/ or test-results/ directories",
			Path:     baseDir,
		})
	}
	return out, validate.SortDiagnostics(inf.diags)
}

// verificationTestRoots are the existing test directories (the model's tests/ and the
// tests/ next to the code root) and the code roots, which hold tests inline.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func verificationTestRoots(baseDir string, bundle model.Bundle, codeRootOption string) []string {
	testRoots := uniqueExistingDirs(append(
		[]string{
			filepath.Join(baseDir, "tests"),
		},
		inferredSiblingDirs(baseDir, codeRootOption, "tests")...,
	))
	return UniqueExistingSources(append(testRoots, inferredCodeRoots(baseDir, bundle, codeRootOption)...))
}

// verificationInference accumulates the inferred checks, their identity keys and the diagnostics.
type verificationInference struct {
	baseDir            string
	reqOwners          map[string][]string
	skip               func(path string, isDir bool) bool
	codeBySource       map[string][]string
	diags              []validate.Diagnostic
	checkByID          map[string]*VerificationCheck
	checkIDsByIdentity map[string][]string
	checkIdentityByID  map[string]map[string]bool
}

// resultArtifact is a parsed test-result artifact: its model-relative path and results.
type resultArtifact struct {
	Path    string
	Results []artifactResult
	// Formal marks a formal evidence file; it always gets its own check.
	Formal bool
}

// artifactResult is one normalized result of a result artifact.
type artifactResult struct {
	Requirement string
	Status      string
	Notes       string
}

// indexCodeElements builds the test-path index of code elements, preferring the
// symbols found in the test roots over the inferred code items.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) indexCodeElements(inferredCode []CodeItem, testRoots []string) {
	inf.codeBySource = buildVerificationCodeElementIndex(inferredCode)
	testSymbols, symbolDiags := buildVerificationTestSymbolIndex(inf.baseDir, testRoots, inf.skip)
	inf.diags = append(inf.diags, symbolDiags...)
	for path, symbols := range testSymbols {
		if len(symbols) > 0 {
			inf.codeBySource[path] = symbols
		}
	}
}

// registerCheckIdentity records the identity keys of a check's source path.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) registerCheckIdentity(id, path string) {
	keys := verificationIdentityKeysFromPath(path)
	if len(keys) == 0 {
		return
	}
	if inf.checkIdentityByID[id] == nil {
		inf.checkIdentityByID[id] = map[string]bool{}
	}
	for _, key := range keys {
		inf.checkIDsByIdentity[key] = textutil.AppendUnique(inf.checkIDsByIdentity[key], id)
		inf.checkIdentityByID[id][key] = true
	}
}

// scanTestRoot infers a check from every test source under root.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) scanTestRoot(root string) {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		return inf.visitTestPath(root, path, d)
	})
	if err != nil {
		inf.diags = append(inf.diags, validate.Diagnostic{
			Code:     "verification.test_walk_failed",
			Severity: validate.SeverityWarning,
			Message:  err.Error(),
			Path:     root,
		})
	}
}

// visitTestPath prunes skipped directories and adds the check of each in-scope file.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) visitTestPath(root, path string, d fs.DirEntry) error {
	if d.IsDir() {
		if path != root && (SkipScanDir(d.Name()) || inf.skip(path, true)) {
			return filepath.SkipDir
		}
		return nil
	}
	if inf.skip(path, false) {
		return nil
	}
	return inf.addTestFile(path)
}

// verificationSourceExtensions are the test source extensions scanned for requirement links.
var verificationSourceExtensions = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".rs": true, ".py": true, ".js": true,
	".mjs": true, ".cjs": true, ".java": true, ".yaml": true, ".yml": true, ".v": true,
}

// addTestFile adds the check of one test source that links requirements.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) addTestFile(path string) error {
	rel, ok := inf.scannableTestPath(path)
	if !ok {
		return nil
	}
	content, ok := inf.readTestFile(path)
	if !ok {
		return nil
	}
	src := testSource{path: path, rel: rel, content: content}
	links, err := inf.linkedRequirements(src)
	if err != nil {
		return err
	}
	if len(links.reqs) > 0 {
		inf.addTestCheck(src, links)
	}
	return nil
}

// testSource is a test source file: its path, model-relative path and content.
type testSource struct {
	path, rel, content string
}

// scannableTestPath is the model-relative path of a file that may hold test
// links: a test path or, because Rust unit tests live inline in production
// files (#[cfg(test)] mod tests), a Rust file, with a scanned extension.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) scannableTestPath(path string) (string, bool) {
	rel, _ := filepath.Rel(inf.baseDir, path)
	rel = filepath.ToSlash(rel)
	ext := strings.ToLower(filepath.Ext(path))
	if !isVerificationTestPath(rel) && ext != ".rs" {
		return rel, false
	}
	return rel, verificationSourceExtensions[ext]
}

// readTestFile reads a test source; an unreadable file is a warning.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) readTestFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		inf.diags = append(inf.diags, validate.Diagnostic{
			Code:     "verification.test_read_failed",
			Severity: validate.SeverityWarning,
			Message:  err.Error(),
			Path:     path,
		})
		return "", false
	}
	return string(data), true
}

// addTestCheck adds the check of a test source from its linked requirements.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) addTestCheck(src testSource, links testLinks) {
	path, rel := src.path, src.rel
	desc := extractVerificationDescription(src.content)
	if desc == "" {
		desc = "Inferred from test source artifact."
	}
	id := verificationIDFromPath(rel)
	inf.checkByID[id] = &VerificationCheck{
		ID:            id,
		Name:          verificationNameFromPath(path),
		Kind:          verificationKindFromPath(rel),
		Status:        "not-run",
		Description:   desc,
		Verifies:      links.reqs,
		CodeElements:  verificationCodeElementsForPath(rel, inf.codeBySource),
		DerivedOwners: ownersForRequirements(inf.reqOwners, links.reqs),
		Evidence:      []string{rel},
		Gates:         links.gates,
		BenchOnly:     links.benchOnly,
	}
	inf.registerCheckIdentity(id, rel)
}

// testLinks are the requirements a test file verifies, with their gates and benchmark-only marks.
type testLinks struct {
	reqs      []string
	gates     map[string][]string
	benchOnly map[string]bool
}

// linkedRequirements reads the requirements a test file verifies: from its test
// entry points where the scanner can tell them from helpers, from the symbols
// of a Verilog or Python file, else from its file-level TRLC-LINKS.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) linkedRequirements(src testSource) (testLinks, error) {
	path := src.path
	if hasTestFunctionDetection(src.rel) {
		reqs, gates, benchOnly, helperDiags, err := testFunctionRequirements(path, src.rel)
		if err != nil {
			return testLinks{}, err
		}
		inf.diags = append(inf.diags, helperDiags...)
		return testLinks{reqs: reqs, gates: gates, benchOnly: benchOnly}, nil
	}
	if ext := strings.ToLower(filepath.Ext(path)); ext == ".v" || ext == ".py" {
		return inf.symbolRequirements(path)
	}
	return testLinks{reqs: extractTRLCLinkedRequirements(src.content)}, nil
}

// symbolRequirements collects the requirements implemented by the symbols of a file.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) symbolRequirements(path string) (testLinks, error) {
	symbols, rtlDiags, err := codemap.Scan(path)
	if err != nil {
		return testLinks{}, err
	}
	inf.diags = append(inf.diags, rtlDiags...)
	var reqs []string
	for _, symbol := range symbols {
		reqs = append(reqs, symbol.Implements...)
	}
	return testLinks{reqs: uniqueStrings(reqs)}, nil
}

// collectArtifacts parses the test-result artifacts under the result roots, ordered by path.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) collectArtifacts(resultRoots []string) []resultArtifact {
	artifacts := make([]resultArtifact, 0)
	for _, root := range resultRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if a, ok := inf.readArtifact(path); ok {
				artifacts = append(artifacts, a)
			}
			return nil
		})
		if err != nil {
			inf.diags = append(inf.diags, validate.Diagnostic{
				Code:     "verification.result_walk_failed",
				Severity: validate.SeverityWarning,
				Message:  err.Error(),
				Path:     root,
			})
		}
	}
	sort.SliceStable(artifacts, func(i, j int) bool {
		return artifacts[i].Path < artifacts[j].Path
	})
	return artifacts
}

// readArtifact parses one result file; false when it holds no results.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) readArtifact(path string) (resultArtifact, bool) {
	formal := isFormalEvidencePath(path)
	res, ok := inf.parseArtifact(path, formal)
	if !ok || len(res) == 0 {
		return resultArtifact{}, false
	}
	rel, _ := filepath.Rel(inf.baseDir, path)
	arts := make([]artifactResult, 0, len(res))
	for _, r := range res {
		arts = append(arts, artifactResult{
			Requirement: r.Requirement,
			Status:      NormalizeResultStatus(r.Status),
			Notes:       r.Notes,
		})
	}
	return resultArtifact{Path: filepath.ToSlash(rel), Results: arts, Formal: formal}, true
}

// parseArtifact parses a formal evidence file or any other result artifact.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) parseArtifact(path string, formal bool) ([]VerificationResult, bool) {
	if !formal {
		return parseVerificationArtifact(path)
	}
	res, formalDiags := parseFormalEvidence(inf.baseDir, path)
	inf.diags = append(inf.diags, formalDiags...)
	return res, true
}

// attachArtifact adds the artifact's results to the best matching check, or to a
// new check of its own when none matches.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) attachArtifact(a resultArtifact) {
	reqSet := a.requirementSet()
	selector := checkSelector{
		checks:       inf.checkByID,
		identityByID: inf.checkIdentityByID,
		reqSet:       reqSet,
		artifactKeys: verificationIdentityKeysFromPath(a.Path),
	}
	bestID := inf.bestCheckID(a, selector)
	if bestID == "" {
		bestID = inf.addArtifactCheck(a, reqSet)
	}
	c := inf.checkByID[bestID]
	c.Evidence = textutil.AppendUnique(c.Evidence, a.Path)
	inf.mergeArtifactResults(c, a)
}

// requirementSet is the non-blank requirements the artifact reports on.
// TRLC-LINKS: REQ-EMG-010
func (a resultArtifact) requirementSet() map[string]bool {
	set := map[string]bool{}
	for _, r := range a.Results {
		if strings.TrimSpace(r.Requirement) != "" {
			set[r.Requirement] = true
		}
	}
	return set
}

// bestCheckID is the check an artifact belongs to: among the checks sharing an
// identity key, else (requiring a requirement overlap) among all checks. A
// formal evidence file never joins a check.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) bestCheckID(a resultArtifact, selector checkSelector) string {
	if a.Formal {
		return ""
	}
	candidateIDSet := map[string]bool{}
	for _, key := range selector.artifactKeys {
		for _, id := range inf.checkIDsByIdentity[key] {
			candidateIDSet[id] = true
		}
	}
	if bestID := selector.best(sortedKeys(candidateIDSet), false); bestID != "" {
		return bestID
	}
	allIDs := make([]string, 0, len(inf.checkByID))
	for id := range inf.checkByID {
		allIDs = append(allIDs, id)
	}
	sort.Strings(allIDs)
	return selector.best(allIDs, true)
}

// sortedKeys is the sorted keys of a set.
// TRLC-LINKS: REQ-EMG-010
func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// addArtifactCheck creates the check of an artifact that matches no test check.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) addArtifactCheck(a resultArtifact, reqSet map[string]bool) string {
	id := verificationIDFromPath(a.Path)
	reqs := sortedKeys(reqSet)
	inf.checkByID[id] = &VerificationCheck{
		ID:            id,
		Name:          verificationNameFromPath(a.Path),
		Kind:          verificationKindFromPath(a.Path),
		Method:        verificationMethodFromPath(a.Path),
		Status:        "not-run",
		Description:   verificationDescriptionForArtifact(inf.baseDir, a.Path),
		Verifies:      reqs,
		CodeElements:  verificationCodeElementsForPath(a.Path, inf.codeBySource),
		DerivedOwners: ownersForRequirements(inf.reqOwners, reqs),
		Evidence:      []string{a.Path},
	}
	inf.registerCheckIdentity(id, a.Path)
	return id
}

// mergeArtifactResults records the artifact's results with a requirement on a check.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) mergeArtifactResults(c *VerificationCheck, a resultArtifact) {
	for _, ar := range a.Results {
		if strings.TrimSpace(ar.Requirement) == "" {
			continue
		}
		c.Verifies = textutil.AppendUnique(c.Verifies, ar.Requirement)
		c.DerivedOwners = textutil.AppendUniqueSlice(c.DerivedOwners, ownersForRequirements(inf.reqOwners, []string{ar.Requirement}))
		c.Results = appendOrMergeResult(c.Results, VerificationResult{
			Requirement: ar.Requirement,
			Status:      ar.Status,
			Evidence:    a.Path,
			Notes:       ar.Notes,
		})
	}
}

// finalChecks completes each check with a not-run result per requirement without
// one, sorts its fields and summarizes its status; the checks are ordered by id and name.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (inf *verificationInference) finalChecks() []VerificationCheck {
	out := make([]VerificationCheck, 0, len(inf.checkByID))
	for _, c := range inf.checkByID {
		completeCheck(c)
		out = append(out, *c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// completeCheck adds the missing not-run results, sorts the check's lists and sets its status.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func completeCheck(c *VerificationCheck) {
	for _, req := range c.Verifies {
		if !hasResultFor(c.Results, req) {
			c.Results = append(c.Results, VerificationResult{
				Requirement: req,
				Status:      "not-run",
				Evidence:    "none",
				Notes:       "No parsed test-result artifact for this requirement yet.",
			})
		}
	}
	sort.Strings(c.Verifies)
	sort.Strings(c.CodeElements)
	sort.Strings(c.DerivedOwners)
	sort.Strings(c.Evidence)
	sort.SliceStable(c.Results, func(i, j int) bool {
		if c.Results[i].Requirement != c.Results[j].Requirement {
			return c.Results[i].Requirement < c.Results[j].Requirement
		}
		return c.Results[i].Status < c.Results[j].Status
	})
	c.Status = summarizeCheckStatus(c.Results)
}

// hasResultFor reports whether a result names the requirement.
// TRLC-LINKS: REQ-EMG-010
func hasResultFor(results []VerificationResult, req string) bool {
	for _, r := range results {
		if r.Requirement == req {
			return true
		}
	}
	return false
}

// hasTestFunctionDetection reports whether the scanner can tell test entry points
// from helpers in this file: Go _test.go files, every Rust file and every Python
// test-path file (pytest collects tests only from test_*.py / *_test.py, so links in
// conftest.py or tests/ support modules are helper links). Other test artifacts
// (YAML, Verilog, JavaScript, TypeScript, Go support files under tests/) keep
// file-level TRLC-LINKS verification.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-010
func hasTestFunctionDetection(rel string) bool {
	base := strings.ToLower(filepath.Base(rel))
	switch strings.ToLower(filepath.Ext(base)) {
	case ".rs":
		return true
	case ".go":
		return strings.HasSuffix(base, "_test.go")
	case ".py":
		return isVerificationTestPath(rel)
	}
	return false
}

// testFunctionRequirements returns the requirements linked from test entry points
// (codemap.Symbol.IsTest) in one file. TRLC-LINKS on any other declaration in test
// code (helpers, fakes, fixtures) verifies nothing and is reported as
// verification.helper_link. A Rust production file yields only its inline tests.
// The gates map holds each requirement whose every linking test is
// environment gated, with the union of their gates; benchOnly each
// requirement whose every linking test is a benchmark.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-088, REQ-EMG-121
func testFunctionRequirements(path, rel string) ([]string, map[string][]string, map[string]bool, []validate.Diagnostic, error) {
	symbols, _, err := codemap.Scan(path)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	testPath := isVerificationTestPath(rel)
	c := newTestLinkCollector()
	for _, s := range symbols {
		switch {
		case len(s.Implements) == 0:
		case s.IsTest:
			c.addTest(s)
		case s.TestOnly || testPath:
			c.diags = append(c.diags, helperLinkDiagnostic(rel, s.Line))
		}
	}
	gates, benchOnly := c.result()
	return uniqueStrings(c.reqs), gates, benchOnly, c.diags, nil
}

// helperLinkDiagnostic warns about a TRLC-LINKS marker on a non-test declaration of a test file.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-088, REQ-EMG-121
func helperLinkDiagnostic(rel string, line int) validate.Diagnostic {
	return validate.Diagnostic{
		Code:     "verification.helper_link",
		Severity: validate.SeverityWarning,
		Message:  "TRLC-LINKS on a non-test declaration in a test file does not verify; put it on the test function",
		Path:     fmt.Sprintf("%s:%d", rel, line),
	}
}

// testLinkCollector gathers the requirements linked from test entry points with their
// gating and benchmark marks.
type testLinkCollector struct {
	reqs      []string
	gates     map[string][]string
	ungated   map[string]bool
	benchOnly map[string]bool
	asserted  map[string]bool
	diags     []validate.Diagnostic
}

// newTestLinkCollector is an empty collector.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-088, REQ-EMG-121
func newTestLinkCollector() *testLinkCollector {
	return &testLinkCollector{
		reqs:      []string{},
		gates:     map[string][]string{},
		ungated:   map[string]bool{},
		benchOnly: map[string]bool{},
		asserted:  map[string]bool{},
		diags:     []validate.Diagnostic{},
	}
}

// addTest records the requirements one test entry point links.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-088, REQ-EMG-121
func (c *testLinkCollector) addTest(s codemap.Symbol) {
	c.reqs = append(c.reqs, s.Implements...)
	for _, req := range s.Implements {
		if s.Benchmark {
			c.benchOnly[req] = true
		} else {
			c.asserted[req] = true
		}
		if len(s.Gates) == 0 {
			c.ungated[req] = true
			continue
		}
		c.gates[req] = textutil.AppendUniqueSlice(c.gates[req], s.Gates)
	}
}

// result is the gates of the requirements whose every test is gated and the
// requirements whose every test is a benchmark; nil when empty.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-088, REQ-EMG-121
func (c *testLinkCollector) result() (map[string][]string, map[string]bool) {
	for req := range c.ungated {
		delete(c.gates, req)
	}
	for req := range c.asserted {
		delete(c.benchOnly, req)
	}
	gates, benchOnly := c.gates, c.benchOnly
	if len(gates) == 0 {
		gates = nil
	}
	if len(benchOnly) == 0 {
		benchOnly = nil
	}
	return gates, benchOnly
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func parseVerificationArtifact(path string) ([]VerificationResult, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	switch ext {
	case ".xml":
		return parseResultXML(data), true
	case ".json":
		return parseResultJSON(data), true
	case ".md", ".txt":
		return parseResultText(string(data)), true
	default:
		return nil, false
	}
}

type junitTestcase struct {
	Name      string    `xml:"name,attr"`
	Classname string    `xml:"classname,attr"`
	Failure   *struct{} `xml:"failure"`
	Error     *struct{} `xml:"error"`
	Skipped   *struct{} `xml:"skipped"`
}

type junitSuite struct {
	Testcases []junitTestcase `xml:"testcase"`
}

type junitSuites struct {
	Suites []junitSuite `xml:"testsuite"`
	Suite  []junitSuite `xml:"testsuites>testsuite"`
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func parseResultXML(data []byte) []VerificationResult {
	decoder := junitSuites{}
	if err := xml.Unmarshal(data, &decoder); err != nil {
		return xmlFallbackResults(data)
	}
	out := []VerificationResult{}
	for _, s := range append(decoder.Suites, decoder.Suite...) {
		for _, tc := range s.Testcases {
			out = append(out, testcaseResults(tc)...)
		}
	}
	return out
}

// xmlFallbackResults is the best-effort extraction for XML that is not JUnit:
// every requirement id gets one status, fail when the text has a failure or error element.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func xmlFallbackResults(data []byte) []VerificationResult {
	reqs := uniqueStrings(requirementIDRe.FindAllString(string(data), -1))
	lower := strings.ToLower(string(data))
	status := "pass"
	if strings.Contains(lower, "<failure") || strings.Contains(lower, "<error") {
		status = "fail"
	}
	out := []VerificationResult{}
	for _, req := range reqs {
		out = append(out, VerificationResult{Requirement: req, Status: status})
	}
	return out
}

// testcaseResults is one result per requirement named in a test case's name or class.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func testcaseResults(tc junitTestcase) []VerificationResult {
	reqs := uniqueStrings(requirementIDRe.FindAllString(tc.Name+" "+tc.Classname, -1))
	status := junitStatus(tc)
	out := make([]VerificationResult, 0, len(reqs))
	for _, req := range reqs {
		out = append(out, VerificationResult{Requirement: req, Status: status})
	}
	return out
}

// junitStatus is fail for a failed or errored test case, not-run for a skipped one, else pass.
// TRLC-LINKS: REQ-EMG-010
func junitStatus(tc junitTestcase) string {
	if tc.Failure != nil || tc.Error != nil {
		return "fail"
	}
	if tc.Skipped != nil {
		return "not-run"
	}
	return "pass"
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func parseResultJSON(data []byte) []VerificationResult {
	out := []VerificationResult{}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return out
	}
	var walk func(node any)
	walk = func(node any) {
		switch x := node.(type) {
		case map[string]any:
			req, _ := x["requirement"].(string)
			status, _ := x["status"].(string)
			notes, _ := x["notes"].(string)
			if strings.TrimSpace(req) != "" {
				out = append(out, VerificationResult{
					Requirement: strings.TrimSpace(req),
					Status:      NormalizeResultStatus(status),
					Notes:       strings.TrimSpace(notes),
				})
			}
			for _, val := range x {
				walk(val)
			}
		case []any:
			for _, val := range x {
				walk(val)
			}
		}
	}
	walk(v)
	return out
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func parseResultText(text string) []VerificationResult {
	reqs := extractTRLCLinkedRequirements(text)
	if len(reqs) == 0 {
		return nil
	}
	status := "not-run"
	if m := resultStatusRe.FindStringSubmatch(text); len(m) > 1 {
		status = NormalizeResultStatus(m[1])
	}
	out := make([]VerificationResult, 0, len(reqs))
	for _, req := range reqs {
		out = append(out, VerificationResult{Requirement: req, Status: status})
	}
	return out
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func extractTRLCLinkedRequirements(text string) []string {
	out := []string{}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		m := trlcLinksMarkerRe.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		out = append(out, requirementIDRe.FindAllString(m[1], -1)...)
	}
	return uniqueStrings(out)
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func buildVerificationCodeElementIndex(items []CodeItem) map[string][]string {
	out := map[string][]string{}
	for path, ev := range collectPathEvidence(items) {
		if labels, ok := ev.labels(path); ok {
			out[path] = labels
		}
	}
	return out
}

// collectPathEvidence groups the symbol and source-file items of test paths by path.
// TRLC-LINKS: REQ-EMG-010
func collectPathEvidence(items []CodeItem) map[string]pathEvidence {
	byPath := map[string]pathEvidence{}
	for _, it := range items {
		srcPath, ok := testSourcePath(it)
		ev := byPath[srcPath]
		if ok && ev.record(it) {
			byPath[srcPath] = ev
		}
	}
	return byPath
}

// pathEvidence is what the code items of one test source file show.
type pathEvidence struct {
	hasSourceFile bool
	symbols       []string
}

// record adds a symbol or source-file item; it reports false for any other kind.
// TRLC-LINKS: REQ-EMG-010
func (e *pathEvidence) record(it CodeItem) bool {
	switch strings.TrimSpace(it.Kind) {
	case "symbol":
		e.symbols = textutil.AppendUnique(e.symbols, ItemDisplayName(it))
	case "source_file":
		e.hasSourceFile = true
	default:
		return false
	}
	return true
}

// labels are the grouped evidence labels of the symbols, else of the file itself
// when it is a source-file item; false when the path shows neither.
// TRLC-LINKS: REQ-EMG-010
func (e pathEvidence) labels(path string) ([]string, bool) {
	if len(e.symbols) > 0 {
		return GroupedEvidenceLabels(e.symbols, nil), true
	}
	if e.hasSourceFile {
		return GroupedEvidenceLabels([]string{path}, nil), true
	}
	return nil, false
}

// testSourcePath is the slashed source path of an item (without its line) when
// it is a test path.
// TRLC-LINKS: REQ-EMG-010
func testSourcePath(it CodeItem) (string, bool) {
	src := filepath.ToSlash(strings.TrimSpace(it.Source))
	if src == "" {
		return "", false
	}
	if idx := strings.Index(src, ":"); idx > 0 {
		src = src[:idx]
	}
	return src, isVerificationTestPath(src)
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func buildVerificationTestSymbolIndex(baseDir string, roots []string, skip func(path string, isDir bool) bool) (map[string][]string, []validate.Diagnostic) {
	idx := &testSymbolIndex{baseDir: baseDir, skip: skip, labels: map[string][]string{}, diags: []validate.Diagnostic{}}
	for _, root := range UniqueExistingSources(roots) {
		idx.addRoot(root)
	}
	for path, elems := range idx.labels {
		idx.labels[path] = GroupedEvidenceLabels(elems, nil)
	}
	return idx.labels, validate.SortDiagnostics(idx.diags)
}

// testSymbolIndex collects the evidence labels of the test symbols per model-relative path.
type testSymbolIndex struct {
	baseDir string
	skip    func(path string, isDir bool) bool
	labels  map[string][]string
	diags   []validate.Diagnostic
}

// addRoot scans one root; a failed scan is a warning and skips the root.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (x *testSymbolIndex) addRoot(root string) {
	symbols, scanDiags, err := codemap.ScanWithOptions(root, codemap.ScanOptions{Skip: x.skip})
	if err != nil {
		x.diags = append(x.diags, validate.Diagnostic{
			Code:     "verification.code_scan_failed",
			Severity: validate.SeverityWarning,
			Message:  err.Error(),
			Path:     root,
		})
		return
	}
	x.diags = append(x.diags, scanDiags...)
	for _, s := range symbols {
		rel := x.relativePath(root, s)
		if !isTestSymbol(rel, s) {
			continue
		}
		label := EvidenceLabel(rel)
		if s.Line > 0 {
			label = EvidenceLabel(fmt.Sprintf("%s:%d", rel, s.Line))
		}
		x.labels[rel] = textutil.AppendUnique(x.labels[rel], label)
	}
}

// relativePath is the slashed path of a symbol's file relative to the model directory.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (x *testSymbolIndex) relativePath(root string, s codemap.Symbol) string {
	absPath := filepath.Join(sourceRootDir(root), filepath.FromSlash(s.Path))
	rel, err := filepath.Rel(x.baseDir, absPath)
	if err != nil {
		rel = absPath
	}
	return filepath.ToSlash(rel)
}

// isTestSymbol reports whether a symbol counts as test evidence: a test entry
// point (including Rust tests inline in production files), or a symbol of a
// test path that is not test-only code of a file with test detection.
// TRLC-LINKS: REQ-EMG-010
func isTestSymbol(rel string, s codemap.Symbol) bool {
	if s.IsTest {
		return true
	}
	return isVerificationTestPath(rel) && !(s.TestOnly && hasTestFunctionDetection(rel))
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func verificationCodeElementsForPath(path string, index map[string][]string) []string {
	p := filepath.ToSlash(strings.TrimSpace(path))
	if p == "" {
		return nil
	}
	if elems, ok := index[p]; ok && len(elems) > 0 {
		return uniqueStrings(elems)
	}
	if isVerificationTestPath(p) {
		ext := strings.ToLower(filepath.Ext(p))
		switch ext {
		case ".go", ".ts", ".tsx", ".rs", ".py", ".js", ".mjs", ".cjs", ".java", ".yaml", ".yml", ".v":
			return []string{p}
		}
	}
	return nil
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func isVerificationTestPath(path string) bool {
	p := strings.ToLower(filepath.ToSlash(strings.TrimSpace(path)))
	base := filepath.Base(p)
	if isPythonTestFile(base) || isVerilogTestbench(base) {
		return true
	}
	if strings.HasPrefix(p, "tests/") || strings.Contains(p, "/tests/") {
		return true
	}
	return hasAnySuffix(base, testFileSuffixes)
}

// testFileSuffixes are the file name endings of Go and JavaScript or TypeScript tests.
var testFileSuffixes = []string{
	"_test.go",
	".test.js", ".test.mjs", ".test.cjs", ".test.ts", ".test.tsx",
	".spec.js", ".spec.mjs", ".spec.cjs", ".spec.ts", ".spec.tsx",
}

// hasAnySuffix reports whether s ends with one of the suffixes.
// TRLC-LINKS: REQ-EMG-010
func hasAnySuffix(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// isPythonTestFile reports whether a lowercased base name is a pytest test module.
// TRLC-LINKS: REQ-EMG-010
func isPythonTestFile(base string) bool {
	return strings.HasSuffix(base, ".py") && (strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py"))
}

// isVerilogTestbench reports whether a lowercased base name is a Verilog testbench.
// TRLC-LINKS: REQ-EMG-010
func isVerilogTestbench(base string) bool {
	if !strings.HasSuffix(base, ".v") {
		return false
	}
	return base == "tb.v" || strings.HasPrefix(base, "tb_") || strings.HasSuffix(base, "_tb.v")
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func inferredCodeRoots(baseDir string, bundle model.Bundle, codeRootOption string) []string {
	out := []string{}
	if strings.TrimSpace(codeRootOption) != "" {
		out = append(out, resolveSourcePath(baseDir, codeRootOption))
	}
	return WithDefaultCodeRoot(bundle, UniqueExistingSources(out))
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func requirementOwners(reqs []model.Requirement) map[string][]string {
	out := map[string][]string{}
	for _, r := range reqs {
		id := strings.TrimSpace(r.ID)
		if id == "" {
			continue
		}
		out[id] = uniqueStrings(nonBlank(r.AppliesTo))
	}
	return out
}

// nonBlank is the trimmed non-empty entries of a list.
// TRLC-LINKS: REQ-EMG-010
func nonBlank(in []string) []string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		if t := strings.TrimSpace(x); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func ownersForRequirements(reqOwners map[string][]string, reqs []string) []string {
	out := []string{}
	for _, req := range reqs {
		out = append(out, reqOwners[req]...)
	}
	return uniqueStrings(out)
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func inferredSiblingDirs(baseDir, codeRootOption, sibling string) []string {
	out := []string{}
	if strings.TrimSpace(codeRootOption) != "" {
		root := codeRootOption
		if !filepath.IsAbs(root) {
			root = filepath.Join(baseDir, root)
		}
		out = append(out, filepath.Join(filepath.Dir(root), sibling))
	}
	return out
}

// TRLC-LINKS: REQ-EMG-010
func uniqueExistingDirs(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, x := range in {
		abs, err := filepath.Abs(x)
		if err != nil {
			continue
		}
		if seen[abs] {
			continue
		}
		info, statErr := os.Stat(abs)
		if statErr != nil || !info.IsDir() {
			continue
		}
		seen[abs] = true
		out = append(out, abs)
	}
	return out
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func verificationKindFromPath(path string) string {
	p := strings.ToLower(filepath.ToSlash(path))
	switch {
	case isFormalEvidencePath(p):
		return "formal"
	case strings.Contains(p, "/unit/"):
		return "unit"
	case strings.Contains(p, "/integration/"):
		return "integration"
	case strings.Contains(p, "/e2e/"):
		return "e2e"
	case strings.Contains(p, "/contract/"):
		return "test"
	default:
		return "test"
	}
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func verificationNameFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	parts := strings.Fields(base)
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	if len(parts) == 0 {
		return "Inferred Verification"
	}
	return strings.Join(parts, " ")
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func verificationIDFromPath(rel string) string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	kind := strings.ToUpper(SanitizeNode(verificationKindFromPath(rel)))
	if kind == "" {
		kind = "TEST"
	}
	base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	base = strings.ToUpper(SanitizeNode(base))
	if base == "" {
		base = "CHECK"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(rel))
	suffix := fmt.Sprintf("%06X", h.Sum32())[:6]
	return "VER-INF-" + kind + "-" + base + "-" + suffix
}

// resultStatusAliases maps the lowercased spellings of a result status to its canonical form.
var resultStatusAliases = map[string]string{
	"pass": "pass", "passed": "pass",
	"fail": "fail", "failed": "fail", "error": "fail",
	"partial": "partial",
	"blocked": "blocked",
	"not-run": "not-run", "notrun": "not-run", "skipped": "not-run",
	"flaky": "flaky",
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func NormalizeResultStatus(s string) string {
	x := strings.ToLower(strings.TrimSpace(s))
	if canonical, ok := resultStatusAliases[x]; ok {
		return canonical
	}
	if x == "" {
		return "not-run"
	}
	return x
}

// statusSummaryRule gives the status of a check whose results include every listed status.
type statusSummaryRule struct {
	all    []string
	status string
}

// statusSummaryRules are tried in order; a check matching none is a pass.
var statusSummaryRules = []statusSummaryRule{
	{[]string{"fail"}, "fail"},
	{[]string{"blocked"}, "blocked"},
	{[]string{"partial"}, "partial"},
	{[]string{"pass", "not-run"}, "partial"},
	{[]string{"flaky"}, "flaky"},
	{[]string{"pass"}, "pass"},
	{[]string{"not-run"}, "not-run"},
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func summarizeCheckStatus(results []VerificationResult) string {
	if len(results) == 0 {
		return "not-run"
	}
	has := map[string]bool{}
	for _, r := range results {
		has[NormalizeResultStatus(r.Status)] = true
	}
	for _, rule := range statusSummaryRules {
		if rule.matches(has) {
			return rule.status
		}
	}
	return "pass"
}

// matches reports whether every listed status is present.
// TRLC-LINKS: REQ-EMG-010
func (r statusSummaryRule) matches(has map[string]bool) bool {
	for _, status := range r.all {
		if !has[status] {
			return false
		}
	}
	return true
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func verificationIdentityKeysFromPath(path string) []string {
	p := filepath.ToSlash(strings.TrimSpace(path))
	if p == "" {
		return nil
	}
	base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	base = strings.TrimSpace(base)
	if base == "" {
		return nil
	}
	keys := []string{}
	appendKey := func(v string) {
		if key := normalizeVerificationIdentity(v); key != "" {
			keys = textutil.AppendUnique(keys, key)
		}
	}
	appendKey(base)
	trimmed := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(base, ".test"), "_test"), "-test"), ".spec"), "_spec"), "-spec")
	appendKey(trimmed)
	return keys
}

// TRLC-LINKS: REQ-EMG-010
func normalizeVerificationIdentity(raw string) string {
	x := strings.ToLower(strings.TrimSpace(raw))
	if x == "" {
		return ""
	}
	x = strings.Trim(nonAlnumRe.ReplaceAllString(x, "-"), "-")
	return x
}

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

// checkSelector picks the check an artifact belongs to by identity-key matches and
// requirement overlap.
type checkSelector struct {
	checks       map[string]*VerificationCheck
	identityByID map[string]map[string]bool
	reqSet       map[string]bool
	artifactKeys []string
}

// checkRank is a candidate check: its identity-key matches, then its requirement overlap.
type checkRank struct {
	id         string
	keyMatches int
	overlap    int
}

// better reports whether the candidate outranks best: by key matches, then
// overlap, then the smaller id.
// TRLC-LINKS: REQ-EMG-010
func (c checkRank) better(best checkRank) bool {
	if c.keyMatches != best.keyMatches {
		return c.keyMatches > best.keyMatches
	}
	if c.overlap != best.overlap {
		return c.overlap > best.overlap
	}
	return best.id == "" || c.id < best.id
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s checkSelector) best(ids []string, requireOverlap bool) string {
	best := checkRank{keyMatches: -1, overlap: -1}
	for _, id := range ids {
		c := s.checks[id]
		if c == nil {
			continue
		}
		overlap := s.overlap(c)
		if requireOverlap && overlap == 0 {
			continue
		}
		if cand := (checkRank{id: id, keyMatches: s.keyMatches(id), overlap: overlap}); cand.better(best) {
			best = cand
		}
	}
	return best.id
}

// overlap counts the check's requirements the artifact also names.
// TRLC-LINKS: REQ-EMG-010
func (s checkSelector) overlap(c *VerificationCheck) int {
	n := 0
	for _, req := range c.Verifies {
		if s.reqSet[req] {
			n++
		}
	}
	return n
}

// keyMatches counts the artifact's identity keys the check also has.
// TRLC-LINKS: REQ-EMG-010
func (s checkSelector) keyMatches(id string) int {
	n := 0
	for _, key := range s.artifactKeys {
		if s.identityByID[id][key] {
			n++
		}
	}
	return n
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func verificationDescriptionFromResultArtifact(baseDir, relPath string) string {
	path := strings.TrimSpace(relPath)
	if path == "" {
		return "Inferred from test result artifact."
	}
	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(baseDir, filepath.FromSlash(path))
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "Inferred from test result artifact."
	}
	if desc := extractVerificationDescription(string(data)); desc != "" {
		return desc
	}
	return "Inferred from test result artifact."
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func extractVerificationDescription(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if desc, ok := extractVerificationDescriptionMarker(line); ok {
			return desc
		}
	}
	return ""
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func extractVerificationDescriptionMarker(line string) (string, bool) {
	markers := []string{
		"ENGMODEL-VERIFICATION-DESCRIPTION:",
		"engmodel:verification-description:",
		"engmodel:verification-description",
		"engmodel.verification.description:",
		"engmodel.verification.description",
	}
	for _, marker := range markers {
		if v := markerDescription(line, marker); v != "" {
			return v, true
		}
	}
	return "", false
}

// TRLC-LINKS: REQ-EMG-010
func uniqueStrings(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, x := range in {
		t := strings.TrimSpace(x)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// TRLC-LINKS: REQ-EMG-010
func appendOrMergeResult(in []VerificationResult, r VerificationResult) []VerificationResult {
	for i := range in {
		if in[i].Requirement == r.Requirement {
			in[i] = mergeVerificationResult(in[i], r)
			return in
		}
	}
	return append(in, r)
}

// TRLC-LINKS: REQ-EMG-010
func mergeVerificationResult(existing, incoming VerificationResult) VerificationResult {
	existingStatus := NormalizeResultStatus(existing.Status)
	incomingStatus := NormalizeResultStatus(incoming.Status)
	if verificationStatusRank(incomingStatus) > verificationStatusRank(existingStatus) {
		existing.Status = incomingStatus
		existing.Evidence = strings.TrimSpace(incoming.Evidence)
		existing.Notes = strings.TrimSpace(incoming.Notes)
		return existing
	}
	if strings.TrimSpace(existing.Evidence) == "" {
		existing.Evidence = strings.TrimSpace(incoming.Evidence)
	}
	if strings.TrimSpace(existing.Notes) == "" {
		existing.Notes = strings.TrimSpace(incoming.Notes)
	}
	return existing
}

// TRLC-LINKS: REQ-EMG-010
func verificationStatusRank(status string) int {
	switch NormalizeResultStatus(status) {
	case "fail":
		return 6
	case "blocked":
		return 5
	case "partial":
		return 4
	case "flaky":
		return 3
	case "pass":
		return 2
	case "not-run":
		return 1
	default:
		return 0
	}
}
