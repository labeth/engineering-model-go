package engmodel

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

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

type inferredVerificationResult struct {
	Requirement string
	Status      string
	Evidence    string
	Notes       string
}

type inferredVerificationCheck struct {
	ID            string
	Name          string
	Kind          string
	Status        string
	Description   string
	Verifies      []string
	CodeElements  []string
	DerivedOwners []string
	Evidence      []string
	Results       []inferredVerificationResult
}

var requirementIDRe = regexp.MustCompile(`\bREQ-[A-Za-z0-9-]+\b`)
var resultStatusRe = regexp.MustCompile(`(?i)\b(pass|fail|partial|blocked|not-run|flaky)\b`)

func inferVerificationChecks(bundle model.Bundle, requirements model.RequirementsDocument, inferredCode []inferredCodeItem, codeRootOption string) ([]inferredVerificationCheck, []validate.Diagnostic) {
	baseDir := filepath.Dir(bundle.ArchitecturePath)
	reqOwners := requirementOwners(requirements.Requirements)
	codeBySource := buildVerificationCodeElementIndex(inferredCode)

	testRoots := uniqueExistingDirs(append(
		[]string{
			filepath.Join(baseDir, "tests"),
		},
		inferredSiblingDirs(baseDir, bundle, codeRootOption, "tests")...,
	))
	resultRoots := uniqueExistingDirs(append(
		[]string{
			filepath.Join(baseDir, "test-results"),
		},
		inferredSiblingDirs(baseDir, bundle, codeRootOption, "test-results")...,
	))

	diags := []validate.Diagnostic{}
	checkByID := map[string]*inferredVerificationCheck{}
	checkIDsByIdentity := map[string][]string{}
	checkIdentityByID := map[string]map[string]bool{}
	registerCheckIdentity := func(id, path string) {
		keys := verificationIdentityKeysFromPath(path)
		if len(keys) == 0 {
			return
		}
		if checkIdentityByID[id] == nil {
			checkIdentityByID[id] = map[string]bool{}
		}
		for _, key := range keys {
			checkIDsByIdentity[key] = appendUnique(checkIDsByIdentity[key], id)
			checkIdentityByID[id][key] = true
		}
	}

	for _, root := range testRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".rs" && ext != ".py" && ext != ".js" && ext != ".java" && ext != ".yaml" && ext != ".yml" {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				diags = append(diags, validate.Diagnostic{
					Code:     "verification.test_read_failed",
					Severity: validate.SeverityWarning,
					Message:  readErr.Error(),
					Path:     path,
				})
				return nil
			}
			content := string(data)
			reqs := uniqueStrings(requirementIDRe.FindAllString(content, -1))
			if len(reqs) == 0 {
				return nil
			}
			desc := extractVerificationDescription(content)
			if desc == "" {
				desc = "Inferred from test source artifact."
			}
			rel, _ := filepath.Rel(baseDir, path)
			rel = filepath.ToSlash(rel)
			id := verificationIDFromPath(rel)
			check := &inferredVerificationCheck{
				ID:            id,
				Name:          verificationNameFromPath(path),
				Kind:          verificationKindFromPath(rel),
				Status:        "not-run",
				Description:   desc,
				Verifies:      reqs,
				CodeElements:  verificationCodeElementsForPath(rel, codeBySource),
				DerivedOwners: ownersForRequirements(reqOwners, reqs),
				Evidence:      []string{rel},
			}
			checkByID[id] = check
			registerCheckIdentity(id, rel)
			return nil
		})
		if err != nil {
			diags = append(diags, validate.Diagnostic{
				Code:     "verification.test_walk_failed",
				Severity: validate.SeverityWarning,
				Message:  err.Error(),
				Path:     root,
			})
		}
	}

	type artifactResult struct {
		Requirement string
		Status      string
		Notes       string
	}
	type artifact struct {
		Path    string
		Results []artifactResult
	}

	artifacts := make([]artifact, 0)
	for _, root := range resultRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			res, ok := parseVerificationArtifact(path)
			if !ok || len(res) == 0 {
				return nil
			}
			rel, _ := filepath.Rel(baseDir, path)
			rel = filepath.ToSlash(rel)
			arts := make([]artifactResult, 0, len(res))
			for _, r := range res {
				arts = append(arts, artifactResult{
					Requirement: r.Requirement,
					Status:      normalizeResultStatus(r.Status),
					Notes:       r.Notes,
				})
			}
			artifacts = append(artifacts, artifact{Path: rel, Results: arts})
			return nil
		})
		if err != nil {
			diags = append(diags, validate.Diagnostic{
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

	for _, a := range artifacts {
		bestID := ""
		reqSet := map[string]bool{}
		for _, r := range a.Results {
			if strings.TrimSpace(r.Requirement) != "" {
				reqSet[r.Requirement] = true
			}
		}

		artifactKeys := verificationIdentityKeysFromPath(a.Path)
		candidateIDSet := map[string]bool{}
		for _, key := range artifactKeys {
			for _, id := range checkIDsByIdentity[key] {
				candidateIDSet[id] = true
			}
		}
		candidateIDs := make([]string, 0, len(candidateIDSet))
		for id := range candidateIDSet {
			candidateIDs = append(candidateIDs, id)
		}
		sort.Strings(candidateIDs)
		bestID = selectBestVerificationCheckID(candidateIDs, checkByID, checkIdentityByID, reqSet, artifactKeys, false)

		if bestID == "" {
			allIDs := make([]string, 0, len(checkByID))
			for id := range checkByID {
				allIDs = append(allIDs, id)
			}
			sort.Strings(allIDs)
			bestID = selectBestVerificationCheckID(allIDs, checkByID, checkIdentityByID, reqSet, artifactKeys, true)
		}
		if bestID == "" {
			id := verificationIDFromPath(a.Path)
			reqs := make([]string, 0, len(reqSet))
			for req := range reqSet {
				reqs = append(reqs, req)
			}
			sort.Strings(reqs)
			checkByID[id] = &inferredVerificationCheck{
				ID:            id,
				Name:          verificationNameFromPath(a.Path),
				Kind:          verificationKindFromPath(a.Path),
				Status:        "not-run",
				Description:   verificationDescriptionFromResultArtifact(baseDir, a.Path),
				Verifies:      reqs,
				CodeElements:  verificationCodeElementsForPath(a.Path, codeBySource),
				DerivedOwners: ownersForRequirements(reqOwners, reqs),
				Evidence:      []string{a.Path},
			}
			registerCheckIdentity(id, a.Path)
			bestID = id
		}
		c := checkByID[bestID]
		c.Evidence = appendUnique(c.Evidence, a.Path)
		for _, ar := range a.Results {
			if strings.TrimSpace(ar.Requirement) == "" {
				continue
			}
			c.Verifies = appendUnique(c.Verifies, ar.Requirement)
			c.DerivedOwners = appendUniqueSlice(c.DerivedOwners, ownersForRequirements(reqOwners, []string{ar.Requirement}))
			c.Results = appendOrMergeResult(c.Results, inferredVerificationResult{
				Requirement: ar.Requirement,
				Status:      ar.Status,
				Evidence:    a.Path,
				Notes:       ar.Notes,
			})
		}
	}

	out := make([]inferredVerificationCheck, 0, len(checkByID))
	for _, c := range checkByID {
		for _, req := range c.Verifies {
			found := false
			for _, r := range c.Results {
				if r.Requirement == req {
					found = true
					break
				}
			}
			if !found {
				c.Results = append(c.Results, inferredVerificationResult{
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
		out = append(out, *c)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return out[i].Name < out[j].Name
	})
	if len(out) == 0 {
		diags = append(diags, validate.Diagnostic{
			Code:     "verification.no_artifacts_found",
			Severity: validate.SeverityWarning,
			Message:  "no verification artifacts found under inferred tests/ or test-results/ directories",
			Path:     baseDir,
		})
	}
	return out, validate.SortDiagnostics(diags)
}

func parseVerificationArtifact(path string) ([]inferredVerificationResult, bool) {
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

func parseResultXML(data []byte) []inferredVerificationResult {
	out := []inferredVerificationResult{}
	decoder := junitSuites{}
	if err := xml.Unmarshal(data, &decoder); err != nil {
		// fallback: best effort regex extraction
		reqs := uniqueStrings(requirementIDRe.FindAllString(string(data), -1))
		status := "pass"
		if strings.Contains(strings.ToLower(string(data)), "<failure") || strings.Contains(strings.ToLower(string(data)), "<error") {
			status = "fail"
		}
		for _, req := range reqs {
			out = append(out, inferredVerificationResult{Requirement: req, Status: status})
		}
		return out
	}
	suites := append(decoder.Suites, decoder.Suite...)
	for _, s := range suites {
		for _, tc := range s.Testcases {
			reqs := uniqueStrings(requirementIDRe.FindAllString(tc.Name+" "+tc.Classname, -1))
			if len(reqs) == 0 {
				continue
			}
			status := "pass"
			if tc.Failure != nil || tc.Error != nil {
				status = "fail"
			} else if tc.Skipped != nil {
				status = "not-run"
			}
			for _, req := range reqs {
				out = append(out, inferredVerificationResult{Requirement: req, Status: status})
			}
		}
	}
	return out
}

func parseResultJSON(data []byte) []inferredVerificationResult {
	out := []inferredVerificationResult{}
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
				out = append(out, inferredVerificationResult{
					Requirement: strings.TrimSpace(req),
					Status:      normalizeResultStatus(status),
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

func parseResultText(text string) []inferredVerificationResult {
	reqs := uniqueStrings(requirementIDRe.FindAllString(text, -1))
	if len(reqs) == 0 {
		return nil
	}
	status := "not-run"
	if m := resultStatusRe.FindStringSubmatch(text); len(m) > 1 {
		status = normalizeResultStatus(m[1])
	}
	out := make([]inferredVerificationResult, 0, len(reqs))
	for _, req := range reqs {
		out = append(out, inferredVerificationResult{Requirement: req, Status: status})
	}
	return out
}

func buildVerificationCodeElementIndex(items []inferredCodeItem) map[string][]string {
	out := map[string][]string{}
	add := func(path, elem string) {
		path = filepath.ToSlash(strings.TrimSpace(path))
		elem = strings.TrimSpace(elem)
		if path == "" || elem == "" {
			return
		}
		if !isVerificationTestPath(path) {
			return
		}
		out[path] = appendUnique(out[path], elem)
	}
	for _, it := range items {
		src := filepath.ToSlash(strings.TrimSpace(it.Source))
		if src == "" {
			continue
		}
		srcPath := src
		if idx := strings.Index(srcPath, ":"); idx > 0 {
			srcPath = srcPath[:idx]
		}
		if !isVerificationTestPath(srcPath) {
			continue
		}
		add(srcPath, srcPath)
		add(srcPath, strings.TrimSpace(it.Element))
	}
	for k := range out {
		sort.Strings(out[k])
	}
	return out
}

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
		case ".go", ".ts", ".tsx", ".rs", ".py", ".js", ".java", ".yaml", ".yml":
			return []string{p}
		}
	}
	return nil
}

func isVerificationTestPath(path string) bool {
	p := strings.ToLower(filepath.ToSlash(strings.TrimSpace(path)))
	return strings.HasPrefix(p, "tests/") || strings.Contains(p, "/tests/")
}

func requirementOwners(reqs []model.Requirement) map[string][]string {
	out := map[string][]string{}
	for _, r := range reqs {
		id := strings.TrimSpace(r.ID)
		if id == "" {
			continue
		}
		units := make([]string, 0, len(r.AppliesTo))
		for _, u := range r.AppliesTo {
			if t := strings.TrimSpace(u); t != "" {
				units = append(units, t)
			}
		}
		out[id] = uniqueStrings(units)
	}
	return out
}

func ownersForRequirements(reqOwners map[string][]string, reqs []string) []string {
	out := []string{}
	for _, req := range reqs {
		out = append(out, reqOwners[req]...)
	}
	return uniqueStrings(out)
}

func inferredSiblingDirs(baseDir string, bundle model.Bundle, codeRootOption, sibling string) []string {
	out := []string{}
	if strings.TrimSpace(codeRootOption) != "" {
		root := codeRootOption
		if !filepath.IsAbs(root) {
			root = filepath.Join(baseDir, root)
		}
		out = append(out, filepath.Join(filepath.Dir(root), sibling))
	}
	for _, src := range bundle.Architecture.InferenceHints.CodeSources {
		root := resolveSourcePath(baseDir, src)
		out = append(out, filepath.Join(filepath.Dir(root), sibling))
	}
	return out
}

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

func verificationKindFromPath(path string) string {
	p := strings.ToLower(filepath.ToSlash(path))
	switch {
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

func verificationIDFromPath(rel string) string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	kind := strings.ToUpper(sanitizeNode(verificationKindFromPath(rel)))
	if kind == "" {
		kind = "TEST"
	}
	base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	base = strings.ToUpper(sanitizeNode(base))
	if base == "" {
		base = "CHECK"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(rel))
	suffix := fmt.Sprintf("%06X", h.Sum32())[:6]
	return "VER-INF-" + kind + "-" + base + "-" + suffix
}

func normalizeResultStatus(s string) string {
	x := strings.ToLower(strings.TrimSpace(s))
	switch x {
	case "pass", "passed":
		return "pass"
	case "fail", "failed", "error":
		return "fail"
	case "partial":
		return "partial"
	case "blocked":
		return "blocked"
	case "not-run", "notrun", "skipped":
		return "not-run"
	case "flaky":
		return "flaky"
	default:
		if x == "" {
			return "not-run"
		}
		return x
	}
}

func summarizeCheckStatus(results []inferredVerificationResult) string {
	if len(results) == 0 {
		return "not-run"
	}
	has := map[string]bool{}
	for _, r := range results {
		has[normalizeResultStatus(r.Status)] = true
	}
	switch {
	case has["fail"]:
		return "fail"
	case has["blocked"]:
		return "blocked"
	case has["partial"]:
		return "partial"
	case has["pass"] && has["not-run"]:
		return "partial"
	case has["flaky"]:
		return "flaky"
	case has["pass"]:
		return "pass"
	case has["not-run"]:
		return "not-run"
	default:
		return "pass"
	}
}

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
			keys = appendUnique(keys, key)
		}
	}
	appendKey(base)
	trimmed := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(base, ".test"), "_test"), "-test"), ".spec"), "_spec"), "-spec")
	appendKey(trimmed)
	return keys
}

func normalizeVerificationIdentity(raw string) string {
	x := strings.ToLower(strings.TrimSpace(raw))
	if x == "" {
		return ""
	}
	x = strings.Trim(nonAlnumRe.ReplaceAllString(x, "-"), "-")
	return x
}

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

func selectBestVerificationCheckID(ids []string, checks map[string]*inferredVerificationCheck, checkIdentityByID map[string]map[string]bool, reqSet map[string]bool, artifactKeys []string, requireOverlap bool) string {
	bestID := ""
	bestKeyMatches := -1
	bestOverlap := -1
	for _, id := range ids {
		c := checks[id]
		if c == nil {
			continue
		}
		overlap := 0
		for _, req := range c.Verifies {
			if reqSet[req] {
				overlap++
			}
		}
		if requireOverlap && overlap == 0 {
			continue
		}
		keyMatches := 0
		for _, key := range artifactKeys {
			if checkIdentityByID[id][key] {
				keyMatches++
			}
		}
		if keyMatches > bestKeyMatches || (keyMatches == bestKeyMatches && overlap > bestOverlap) || (keyMatches == bestKeyMatches && overlap == bestOverlap && (bestID == "" || id < bestID)) {
			bestID = id
			bestKeyMatches = keyMatches
			bestOverlap = overlap
		}
	}
	return bestID
}

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

func extractVerificationDescription(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if desc, ok := extractVerificationDescriptionMarker(line); ok {
			return desc
		}
	}
	return ""
}

func extractVerificationDescriptionMarker(line string) (string, bool) {
	markers := []string{
		"ENGMODEL-VERIFICATION-DESCRIPTION:",
		"engmodel:verification-description:",
		"engmodel:verification-description",
		"engmodel.verification.description:",
		"engmodel.verification.description",
	}
	for _, marker := range markers {
		if v, ok := extractMarkerValue(line, marker); ok {
			v = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v), ":"))
			if v != "" {
				return v, true
			}
		}
	}
	return "", false
}

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

func appendUnique(in []string, x string) []string {
	x = strings.TrimSpace(x)
	if x == "" {
		return in
	}
	for _, e := range in {
		if e == x {
			return in
		}
	}
	return append(in, x)
}

func appendUniqueSlice(in []string, items []string) []string {
	out := append([]string(nil), in...)
	for _, x := range items {
		out = appendUnique(out, x)
	}
	return out
}

func appendOrMergeResult(in []inferredVerificationResult, r inferredVerificationResult) []inferredVerificationResult {
	for i := range in {
		if in[i].Requirement == r.Requirement {
			in[i] = mergeVerificationResult(in[i], r)
			return in
		}
	}
	return append(in, r)
}

func mergeVerificationResult(existing, incoming inferredVerificationResult) inferredVerificationResult {
	existingStatus := normalizeResultStatus(existing.Status)
	incomingStatus := normalizeResultStatus(incoming.Status)
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

func verificationStatusRank(status string) int {
	switch normalizeResultStatus(status) {
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
