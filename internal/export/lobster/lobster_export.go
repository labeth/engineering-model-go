// ENGMODEL-OWNER-UNIT: FU-LOBSTER-EXPORTER
package lobster

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/export/trlc"
	"github.com/labeth/engineering-model-go/internal/model"
)

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-TRLC-EXPORTER
type Options struct {
	RequirementsPackage string
	ActivityNamespace   string
}

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type Result struct {
	JSON string
}

var ErrNoActivities = errors.New("LOBSTER activity trace requires at least one TRLC-linked source artifact")

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type lobsterSourceRef struct {
	Kind   string `json:"kind"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type lobsterActivityItem struct {
	Tag            string           `json:"tag"`
	Location       lobsterSourceRef `json:"location"`
	Name           string           `json:"name"`
	Refs           []string         `json:"refs"`
	JustUp         []string         `json:"just_up"`
	JustDown       []string         `json:"just_down"`
	JustGlobal     []string         `json:"just_global"`
	Framework      string           `json:"framework"`
	Kind           string           `json:"kind"`
	Status         string           `json:"status"`
	RequirementIDs []string         `json:"-"`
}

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type lobsterActivityDoc struct {
	Data      []lobsterActivityItem `json:"data"`
	Generator string                `json:"generator"`
	Schema    string                `json:"schema"`
	Version   int                   `json:"version"`
}

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
var lobsterReqIDRe = regexp.MustCompile(`\bREQ-[A-Za-z0-9-]+\b`)

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
var lobsterTRLCMarkerRe = regexp.MustCompile(`(?i)TRLC-LINKS:\s*(.*)$`)

// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-TRLC-EXPORTER, FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE, DO-LOBSTER-ACTIVITY
func GenerateActivityTraceFromDir(testsDir string, options Options) (Result, error) {
	absTestsDir, err := filepath.Abs(testsDir)
	if err != nil {
		return Result{}, fmt.Errorf("resolve tests dir: %w", err)
	}
	scan := newActivityScan(absTestsDir, options)
	items, err := scan.collectItems()
	if err != nil {
		return Result{}, fmt.Errorf("walk tests dir: %w", err)
	}
	if len(items) == 0 {
		return Result{}, ErrNoActivities
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Tag < items[j].Tag })
	if err := scan.resolveRefs(items); err != nil {
		return Result{}, err
	}

	doc := lobsterActivityDoc{Data: items, Generator: "engmodel", Schema: "lobster-act-trace", Version: 3}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("marshal lobster activity trace: %w", err)
	}
	return Result{JSON: string(b) + "\n"}, nil
}

// activityScan holds the resolved inputs of one activity-trace export: the
// absolute tests directory, the activity namespace and the requirements package.
//
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type activityScan struct {
	root      string
	namespace string
	reqPkg    string
}

// newActivityScan resolves the namespace and requirements package defaults.
//
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
func newActivityScan(root string, options Options) activityScan {
	scan := activityScan{root: root, namespace: strings.TrimSpace(options.ActivityNamespace), reqPkg: trlc.SanitizeIdentifier(strings.TrimSpace(options.RequirementsPackage))}
	if scan.namespace == "" {
		scan.namespace = "tests"
	}
	if scan.reqPkg == "" {
		scan.reqPkg = "Requirements"
	}
	return scan
}

var lobsterSourceExtensions = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".rs": true, ".py": true, ".js": true, ".yaml": true, ".yml": true,
}

// collectItems walks the tests directory and returns one activity item per
// source file that carries a TRLC-LINKS marker.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func (scan activityScan) collectItems() ([]lobsterActivityItem, error) {
	items := []lobsterActivityItem{}
	err := filepath.WalkDir(scan.root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !lobsterSourceExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		if item, ok := scan.itemForFile(path); ok {
			items = append(items, item)
		}
		return nil
	})
	return items, err
}

// itemForFile builds the activity item of one source file; it reports false
// when the file is unreadable or has no TRLC-LINKS marker.
//
// TRLC-LINKS: REQ-EMG-006
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func (scan activityScan) itemForFile(path string) (lobsterActivityItem, bool) {
	contentBytes, readErr := os.ReadFile(path)
	if readErr != nil {
		return lobsterActivityItem{}, false
	}
	reqs, markerLine, markerColumn := extractTRLCMarkerRefs(string(contentBytes))
	if len(reqs) == 0 {
		return lobsterActivityItem{}, false
	}
	relPath, _ := filepath.Rel(scan.root, path)
	relPath = filepath.ToSlash(relPath)
	tagID := trlc.SanitizeIdentifier(strings.ReplaceAll(relPath, "/", "_"))
	if tagID == "" {
		tagID = "test_item"
	}
	return lobsterActivityItem{
		Tag:            "act " + scan.namespace + "." + tagID,
		Location:       lobsterSourceRef{Kind: "file", File: relPath, Line: markerLine, Column: markerColumn},
		Name:           filepath.Base(path),
		Refs:           scan.requirementRefs(reqs),
		JustUp:         []string{},
		JustDown:       []string{},
		JustGlobal:     []string{},
		Framework:      "Tests",
		Kind:           "test",
		Status:         "ok",
		RequirementIDs: append([]string(nil), reqs...),
	}, true
}

// requirementRefs renders the LOBSTER requirement references of the IDs.
//
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
func (scan activityScan) requirementRefs(ids []string) []string {
	refs := make([]string, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, "req "+scan.reqPkg+"."+trlc.SanitizeIdentifier(id))
	}
	return refs
}

// resolveRefs checks every referenced requirement against the canonical
// requirement projection and rewrites the items' references from it.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-TRLC-EXPORTER
func (scan activityScan) resolveRefs(items []lobsterActivityItem) error {
	canonicalRequirements, err := canonicalRequirementIDs(items)
	if err != nil {
		return err
	}
	for i := range items {
		if err := requireCanonical(items[i].RequirementIDs, canonicalRequirements); err != nil {
			return err
		}
		items[i].Refs = scan.requirementRefs(items[i].RequirementIDs)
	}
	return nil
}

// requireCanonical fails on the first ID missing from the canonical projection.
//
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
func requireCanonical(ids []string, canonical map[string]bool) error {
	for _, id := range ids {
		if !canonical[id] {
			return fmt.Errorf("canonical requirement %q missing from LOBSTER projection", id)
		}
	}
	return nil
}

// referencedRequirements returns the requirements the items reference, sorted by ID.
//
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
func referencedRequirements(items []lobsterActivityItem) []model.Requirement {
	requirementByID := map[string]model.Requirement{}
	for _, item := range items {
		for _, id := range item.RequirementIDs {
			requirementByID[id] = model.Requirement{ID: id, Text: id}
		}
	}
	requirements := make([]model.Requirement, 0, len(requirementByID))
	for _, requirement := range requirementByID {
		requirements = append(requirements, requirement)
	}
	sort.SliceStable(requirements, func(i, j int) bool { return requirements[i].ID < requirements[j].ID })
	return requirements
}

// canonicalRequirementIDs returns the set of requirement definitions in the
// canonical projection of every requirement the items reference.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-TRLC-EXPORTER
func canonicalRequirementIDs(items []lobsterActivityItem) (map[string]bool, error) {
	canonical, err := model.NewCanonicalRequirements(model.RequirementsDocument{Requirements: referencedRequirements(items)})
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, element := range canonical.Semantic().Elements {
		if element.Kind == model.ElementRequirementDefinition {
			ids[element.ID] = true
		}
	}
	return ids, nil
}

// TRLC-LINKS: REQ-EMG-006
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func extractTRLCMarkerRefs(content string) ([]string, int, int) {
	reqs := []string{}
	seen := map[string]bool{}
	markerLine := 0
	markerColumn := 0
	for lineIndex, line := range strings.Split(content, "\n") {
		m := lobsterTRLCMarkerRe.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		if markerLine == 0 {
			markerLine = lineIndex + 1
			markerColumn = markerColumnOf(line)
		}
		reqs = appendNewRequirementIDs(reqs, seen, m[1])
	}
	sort.Strings(reqs)
	return reqs, markerLine, markerColumn
}

// markerColumnOf returns the 1-based column of the TRLC-LINKS marker in line.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func markerColumnOf(line string) int {
	if location := lobsterTRLCMarkerRe.FindStringIndex(line); location != nil {
		return location[0] + 1
	}
	return 0
}

// appendNewRequirementIDs appends the requirement IDs found in list that are
// not in seen yet, recording them as seen.
//
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
func appendNewRequirementIDs(reqs []string, seen map[string]bool, list string) []string {
	for _, req := range lobsterReqIDRe.FindAllString(list, -1) {
		req = strings.TrimSpace(req)
		if req == "" || seen[req] {
			continue
		}
		seen[req] = true
		reqs = append(reqs, req)
	}
	return reqs
}
