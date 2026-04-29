// ENGMODEL-OWNER-UNIT: FU-LOBSTER-EXPORTER
package engmodel

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type LobsterActivityExportOptions struct {
	RequirementsPackage string
	ActivityNamespace   string
}

type LobsterActivityExportResult struct {
	JSON string
}

type lobsterSourceRef struct {
	Kind   string `json:"kind"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type lobsterActivityItem struct {
	Tag        string           `json:"tag"`
	Location   lobsterSourceRef `json:"location"`
	Name       string           `json:"name"`
	Refs       []string         `json:"refs"`
	JustUp     []string         `json:"just_up"`
	JustDown   []string         `json:"just_down"`
	JustGlobal []string         `json:"just_global"`
	Framework  string           `json:"framework"`
	Kind       string           `json:"kind"`
	Status     string           `json:"status"`
}

type lobsterActivityDoc struct {
	Data      []lobsterActivityItem `json:"data"`
	Generator string                `json:"generator"`
	Schema    string                `json:"schema"`
	Version   int                   `json:"version"`
}

var lobsterReqIDRe = regexp.MustCompile(`\bREQ-[A-Za-z0-9-]+\b`)
var lobsterTRLCMarkerRe = regexp.MustCompile(`(?i)TRLC-LINKS:\s*(.*)$`)

// TRLC-LINKS: REQ-EMG-006
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
		reqs := extractTRLCMarkerReqs(content)
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
			Tag:        "act " + namespace + "." + tagID,
			Location:   lobsterSourceRef{Kind: "file", File: path, Line: 1, Column: 1},
			Name:       filepath.Base(path),
			Refs:       refs,
			JustUp:     []string{},
			JustDown:   []string{},
			JustGlobal: []string{},
			Framework:  "Tests",
			Kind:       "test",
			Status:     "ok",
		})
		return nil
	})
	if err != nil {
		return LobsterActivityExportResult{}, fmt.Errorf("walk tests dir: %w", err)
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].Tag < items[j].Tag })

	doc := lobsterActivityDoc{Data: items, Generator: "engmodel", Schema: "lobster-act-trace", Version: 3}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return LobsterActivityExportResult{}, fmt.Errorf("marshal lobster activity trace: %w", err)
	}
	return LobsterActivityExportResult{JSON: string(b) + "\n"}, nil
}

func extractTRLCMarkerReqs(content string) []string {
	reqs := []string{}
	seen := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		m := lobsterTRLCMarkerRe.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
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
	return reqs
}
