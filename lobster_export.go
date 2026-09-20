// ENGMODEL-OWNER-UNIT: FU-LOBSTER-EXPORTER
package engmodel

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

	"github.com/labeth/engineering-model-go/model"
)

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-TRLC-EXPORTER
type LobsterActivityExportOptions struct {
	RequirementsPackage string
	ActivityNamespace   string
}

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type LobsterActivityExportResult struct {
	JSON string
}

var ErrNoLobsterActivities = errors.New("LOBSTER activity trace requires at least one TRLC-linked source artifact")

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
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-TRLC-EXPORTER, FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func GenerateLobsterActivityTraceFromDir(testsDir string, options LobsterActivityExportOptions) (LobsterActivityExportResult, error) {
	absTestsDir, err := filepath.Abs(testsDir)
	if err != nil {
		return LobsterActivityExportResult{}, fmt.Errorf("resolve tests dir: %w", err)
	}
	namespace := strings.TrimSpace(options.ActivityNamespace)
	if namespace == "" {
		namespace = "tests"
	}
	reqPkg := sanitizeTRLCIdentifier(strings.TrimSpace(options.RequirementsPackage))
	if reqPkg == "" {
		reqPkg = "Requirements"
	}

	items := []lobsterActivityItem{}
	err = filepath.WalkDir(absTestsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".rs" && ext != ".py" && ext != ".js" && ext != ".yaml" && ext != ".yml" {
			return nil
		}
		contentBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(contentBytes)
		reqs, markerLine, markerColumn := extractTRLCMarkerRefs(content)
		if len(reqs) == 0 {
			return nil
		}
		relPath, _ := filepath.Rel(absTestsDir, path)
		relPath = filepath.ToSlash(relPath)
		tagID := sanitizeTRLCIdentifier(strings.ReplaceAll(relPath, "/", "_"))
		if tagID == "" {
			tagID = "test_item"
		}
		refs := make([]string, 0, len(reqs))
		for _, req := range reqs {
			refs = append(refs, "req "+reqPkg+"."+sanitizeTRLCIdentifier(req))
		}
		items = append(items, lobsterActivityItem{
			Tag:            "act " + namespace + "." + tagID,
			Location:       lobsterSourceRef{Kind: "file", File: relPath, Line: markerLine, Column: markerColumn},
			Name:           filepath.Base(path),
			Refs:           refs,
			JustUp:         []string{},
			JustDown:       []string{},
			JustGlobal:     []string{},
			Framework:      "Tests",
			Kind:           "test",
			Status:         "ok",
			RequirementIDs: append([]string(nil), reqs...),
		})
		return nil
	})
	if err != nil {
		return LobsterActivityExportResult{}, fmt.Errorf("walk tests dir: %w", err)
	}
	if len(items) == 0 {
		return LobsterActivityExportResult{}, ErrNoLobsterActivities
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].Tag < items[j].Tag })
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
	canonical, err := model.NewCanonicalRequirements(model.RequirementsDocument{Requirements: requirements})
	if err != nil {
		return LobsterActivityExportResult{}, err
	}
	canonicalRequirements := map[string]bool{}
	for _, element := range canonical.Semantic().Elements {
		if element.Kind == model.ElementRequirementDefinition {
			canonicalRequirements[element.ID] = true
		}
	}
	for i := range items {
		items[i].Refs = items[i].Refs[:0]
		for _, id := range items[i].RequirementIDs {
			if !canonicalRequirements[id] {
				return LobsterActivityExportResult{}, fmt.Errorf("canonical requirement %q missing from LOBSTER projection", id)
			}
			items[i].Refs = append(items[i].Refs, "req "+reqPkg+"."+sanitizeTRLCIdentifier(id))
		}
	}

	doc := lobsterActivityDoc{Data: items, Generator: "engmodel", Schema: "lobster-act-trace", Version: 3}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return LobsterActivityExportResult{}, fmt.Errorf("marshal lobster activity trace: %w", err)
	}
	return LobsterActivityExportResult{JSON: string(b) + "\n"}, nil
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
			if location := lobsterTRLCMarkerRe.FindStringIndex(line); location != nil {
				markerColumn = location[0] + 1
			}
		}
		for _, req := range lobsterReqIDRe.FindAllString(m[1], -1) {
			req = strings.TrimSpace(req)
			if req == "" || seen[req] {
				continue
			}
			seen[req] = true
			reqs = append(reqs, req)
		}
	}
	sort.Strings(reqs)
	return reqs, markerLine, markerColumn
}
