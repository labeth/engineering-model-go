package engmodel

import (
	"bufio"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

func TestGenerateAIViewFromFiles_Deterministic(t *testing.T) {
	sample := filepath.Join("examples", "bedrock-pr-review-github-app-sample")
	modelPath := filepath.Join(sample, "architecture.yml")
	reqPath := filepath.Join(sample, "requirements.yml")
	designPath := filepath.Join(sample, "design.yml")

	options := AIViewOptions{CodeRoot: filepath.Join(sample, "src")}

	first, err := GenerateAIViewFromFiles(modelPath, reqPath, designPath, options)
	if err != nil {
		t.Fatalf("first generation failed: %v", err)
	}
	second, err := GenerateAIViewFromFiles(modelPath, reqPath, designPath, options)
	if err != nil {
		t.Fatalf("second generation failed: %v", err)
	}

	if first.JSON != second.JSON {
		t.Fatalf("json output is not deterministic")
	}
	if first.Markdown != second.Markdown {
		t.Fatalf("markdown output is not deterministic")
	}
	if first.EdgesNDJSON != second.EdgesNDJSON {
		t.Fatalf("edges ndjson output is not deterministic")
	}
}

func TestGenerateAIViewFromFiles_BedrockShapeAndOrdering(t *testing.T) {
	sample := filepath.Join("examples", "bedrock-pr-review-github-app-sample")
	modelPath := filepath.Join(sample, "architecture.yml")
	reqPath := filepath.Join(sample, "requirements.yml")
	designPath := filepath.Join(sample, "design.yml")

	res, err := GenerateAIViewFromFiles(modelPath, reqPath, designPath, AIViewOptions{CodeRoot: filepath.Join(sample, "src")})
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if res.Document.SchemaVersion != "ai-view/v1" {
		t.Fatalf("unexpected schema version: %q", res.Document.SchemaVersion)
	}
	if res.Document.Model.ID != "sample-bedrock-pr-review-model" {
		t.Fatalf("unexpected model id: %q", res.Document.Model.ID)
	}
	if res.Document.Model.Counts.FunctionalUnits < 7 {
		t.Fatalf("expected >= 7 functional units, got %d", res.Document.Model.Counts.FunctionalUnits)
	}
	if len(res.Document.EntryPoints) == 0 {
		t.Fatalf("expected entry points")
	}
	if len(res.Document.SupportPaths) == 0 {
		t.Fatalf("expected support paths")
	}
	if len(res.Document.SourceBlocks) == 0 {
		t.Fatalf("expected source blocks")
	}

	ids := map[string]bool{}
	for _, e := range res.Document.Entities {
		ids[e.ID] = true
	}
	for _, expected := range []string{"FG-PR-INTEGRATION", "FU-GITHUB-WEBHOOK-INGRESS", "REQ-PRR-001"} {
		if !ids[expected] {
			t.Fatalf("missing expected entity id %q", expected)
		}
	}

	for i := 1; i < len(res.Document.EntityIndex.Lookup); i++ {
		prev := res.Document.EntityIndex.Lookup[i-1]
		cur := res.Document.EntityIndex.Lookup[i]
		prevRank := aiEntityKindRank(prev.Kind)
		curRank := aiEntityKindRank(cur.Kind)
		if curRank < prevRank {
			t.Fatalf("entity lookup kind ordering is not deterministic")
		}
		if curRank == prevRank && cur.ID < prev.ID {
			t.Fatalf("entity lookup id ordering is not deterministic within kind")
		}
	}

	for i := 1; i < len(res.Document.SourceBlocks); i++ {
		prev := res.Document.SourceBlocks[i-1]
		cur := res.Document.SourceBlocks[i]
		if cur.Path < prev.Path {
			t.Fatalf("source block path ordering is not deterministic")
		}
		if cur.Path == prev.Path && cur.LineStart < prev.LineStart {
			t.Fatalf("source block line ordering is not deterministic")
		}
		if cur.Path == prev.Path && cur.LineStart == prev.LineStart && cur.ID < prev.ID {
			t.Fatalf("source block id ordering is not deterministic")
		}
	}

	if !strings.Contains(res.Markdown, "# AI View") {
		t.Fatalf("expected markdown header")
	}

	scanner := bufio.NewScanner(strings.NewReader(res.EdgesNDJSON))
	lines := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines++
		var edge AIEdge
		if err := json.Unmarshal([]byte(line), &edge); err != nil {
			t.Fatalf("invalid ndjson edge line: %v", err)
		}
		if strings.TrimSpace(edge.FromID) == "" || strings.TrimSpace(edge.ToID) == "" || strings.TrimSpace(edge.Relation) == "" {
			t.Fatalf("invalid edge row: %+v", edge)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan edges ndjson: %v", err)
	}
	if lines == 0 {
		t.Fatalf("expected non-empty edges ndjson")
	}
}

func TestGenerateAIView_IncludesNewAuthoredEntityKindsAndSupportPath(t *testing.T) {
	bundle := model.Bundle{ArchitecturePath: filepath.Join(t.TempDir(), "architecture.yml"), Architecture: model.ArchitectureDocument{
		Model: model.ModelMeta{ID: "m", Title: "m"},
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups:  []model.FunctionalGroup{{ID: "FG-A", Name: "Group"}},
			FunctionalUnits:   []model.FunctionalUnit{{ID: "FU-A", Name: "Unit", Group: "FG-A"}},
			Interfaces:        []model.Interface{{ID: "IF-A", Name: "Ingress", Owner: "FU-A"}},
			DataObjects:       []model.DataObject{{ID: "DATA-A", Name: "Digest Lock"}},
			DeploymentTargets: []model.DeploymentTarget{{ID: "DEP-A", Name: "Prod"}},
			Controls:          []model.Control{{ID: "CTRL-A", Name: "Digest Pinning"}},
			TrustBoundaries:   []model.TrustBoundary{{ID: "TB-A", Name: "Cluster Boundary"}},
			States:            []model.State{{ID: "STATE-A", Name: "Ready"}},
			Events:            []model.Event{{ID: "EVT-A", Name: "Release Requested"}},
			Mappings: []model.Mapping{
				{Type: "deployed_to", From: "FU-A", To: "DEP-A"},
				{Type: "calls", From: "FU-A", To: "IF-A"},
				{Type: "writes", From: "FU-A", To: "DATA-A"},
				{Type: "bounded_by", From: "FU-A", To: "TB-A"},
				{Type: "guarded_by", From: "STATE-A", To: "CTRL-A"},
				{Type: "triggered_by", From: "STATE-A", To: "EVT-A"},
			},
		},
		Views: []model.View{{ID: "V", Kind: "traceability", Roots: []string{"FU-A"}}},
	}}
	requirements := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-A", Text: "system shall deploy digest pins", AppliesTo: []string{"FU-A"}}}}
	doc := buildAIViewDocument(bundle, requirements, model.DesignDocument{}, nil, nil, nil, "", "", AIViewOptions{})

	counts := doc.Model.Counts
	if counts.Interfaces == 0 || counts.DataObjects == 0 || counts.DeploymentTargets == 0 || counts.Controls == 0 || counts.TrustBoundaries == 0 || counts.States == 0 || counts.Events == 0 {
		t.Fatalf("expected non-zero counts for new authored kinds, got %+v", counts)
	}

	foundPath := false
	for _, sp := range doc.SupportPaths {
		if sp.FromID != "REQ-A" {
			continue
		}
		for _, id := range sp.Path {
			if id == "DEP-A" || id == "IF-A" || id == "DATA-A" {
				foundPath = true
				break
			}
		}
	}
	if !foundPath {
		t.Fatalf("expected requirement support path to include new authored implementation evidence ids")
	}
}
