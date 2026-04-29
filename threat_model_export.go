// ENGMODEL-OWNER-UNIT: FU-THREAT-EXPORTER
package engmodel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

type ThreatModelFormat string

const (
	ThreatModelFormatThreatDragonV2 ThreatModelFormat = "threat-dragon-v2"
	ThreatModelFormatOpenOTM        ThreatModelFormat = "open-otm"
)

type ThreatModelExportOptions struct {
	Format ThreatModelFormat
}

type ThreatModelExportResult struct {
	JSON        string
	Diagnostics []validate.Diagnostic
}

// TRLC-LINKS: REQ-EMG-004, REQ-EMG-011
func GenerateThreatModelExportFromFile(architecturePath string, options ThreatModelExportOptions) (ThreatModelExportResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return ThreatModelExportResult{}, err
	}
	return GenerateThreatModelExport(bundle, options)
}

func GenerateThreatModelExport(bundle model.Bundle, options ThreatModelExportOptions) (ThreatModelExportResult, error) {
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return ThreatModelExportResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}

	format := strings.TrimSpace(string(options.Format))
	if format == "" {
		format = string(ThreatModelFormatThreatDragonV2)
	}

	var doc any
	switch ThreatModelFormat(format) {
	case ThreatModelFormatThreatDragonV2:
		doc = buildThreatDragonV2(bundle)
	case ThreatModelFormatOpenOTM:
		doc = buildOpenOTM(bundle)
	default:
		return ThreatModelExportResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("unknown threat model export format %q", format)
	}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return ThreatModelExportResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	return ThreatModelExportResult{JSON: string(b) + "\n", Diagnostics: validate.SortDiagnostics(diags)}, nil
}

type tdv2Document struct {
	Version string      `json:"version"`
	Summary tdv2Summary `json:"summary"`
	Detail  tdv2Detail  `json:"detail"`
}

type tdv2Summary struct {
	Title       string `json:"title"`
	Owner       string `json:"owner,omitempty"`
	Description string `json:"description,omitempty"`
	ID          int    `json:"id"`
}

type tdv2Detail struct {
	Contributors []tdv2Contributor `json:"contributors"`
	Diagrams     []tdv2Diagram     `json:"diagrams"`
	DiagramTop   int               `json:"diagramTop"`
	Reviewer     string            `json:"reviewer"`
	ThreatTop    int               `json:"threatTop"`
}

type tdv2Contributor struct {
	Name string `json:"name"`
}

type tdv2Diagram struct {
	Description string     `json:"description,omitempty"`
	DiagramType string     `json:"diagramType"`
	ID          int        `json:"id"`
	Placeholder string     `json:"placeholder,omitempty"`
	Thumbnail   string     `json:"thumbnail"`
	Title       string     `json:"title"`
	Version     string     `json:"version"`
	Cells       []tdv2Cell `json:"cells,omitempty"`
}

type tdv2Cell struct {
	ID        string               `json:"id"`
	Shape     string               `json:"shape"`
	ZIndex    int                  `json:"zIndex"`
	Position  map[string]float64   `json:"position,omitempty"`
	Size      map[string]float64   `json:"size,omitempty"`
	Attrs     map[string]any       `json:"attrs,omitempty"`
	Data      map[string]any       `json:"data,omitempty"`
	Source    map[string]any       `json:"source,omitempty"`
	Target    map[string]any       `json:"target,omitempty"`
	Labels    []map[string]any     `json:"labels,omitempty"`
	Vertices  []map[string]float64 `json:"vertices,omitempty"`
	Connector string               `json:"connector,omitempty"`
}

func buildThreatDragonV2(bundle model.Bundle) tdv2Document {
	a := bundle.Architecture.AuthoredArchitecture

	nodes := []struct {
		ID      string
		Name    string
		Kind    string
		Desc    string
		Threats []map[string]any
	}{}

	threatsByEntity := map[string][]map[string]any{}
	threatCount := 0
	for _, ts := range a.ThreatScenarios {
		severity := tdSeverity(ts.Severity, ts.Impact)
		status := tdStatus(ts.Status)
		mitigation := ""
		for _, m := range a.ThreatMitigations {
			if strings.TrimSpace(m.ThreatScenarioRef) == strings.TrimSpace(ts.ID) {
				if strings.TrimSpace(m.Notes) != "" {
					mitigation = strings.TrimSpace(m.Notes)
					break
				}
			}
		}
		if mitigation == "" {
			mitigation = "See mapped control mitigations and verification evidence."
		}
		th := map[string]any{
			"description": nonEmpty(strings.TrimSpace(ts.Summary), strings.TrimSpace(ts.Title)),
			"mitigation":  mitigation,
			"modelType":   "STRIDE",
			"number":      threatCount,
			"score":       strings.ToLower(nonEmpty(strings.TrimSpace(ts.Severity), strings.TrimSpace(ts.Impact))),
			"severity":    severity,
			"status":      status,
			"threatId":    deterministicUUID("td-threat|" + bundle.Architecture.Model.ID + "|" + ts.ID),
			"title":       nonEmpty(strings.TrimSpace(ts.Title), strings.TrimSpace(ts.ID)),
			"type":        nonEmpty(strings.TrimSpace(ts.Stride), strings.TrimSpace(ts.Category)),
		}
		for _, id := range ts.AppliesTo {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			threatsByEntity[id] = append(threatsByEntity[id], th)
		}
		for _, id := range ts.FlowRefs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			threatsByEntity[id] = append(threatsByEntity[id], th)
		}
		threatCount++
	}

	for _, x := range a.Actors {
		nodes = append(nodes, struct {
			ID      string
			Name    string
			Kind    string
			Desc    string
			Threats []map[string]any
		}{ID: x.ID, Name: nonEmpty(x.Name, x.ID), Kind: "actor", Desc: strings.TrimSpace(x.Description), Threats: threatsByEntity[x.ID]})
	}
	for _, x := range a.FunctionalUnits {
		nodes = append(nodes, struct {
			ID      string
			Name    string
			Kind    string
			Desc    string
			Threats []map[string]any
		}{ID: x.ID, Name: nonEmpty(x.Name, x.ID), Kind: "process", Desc: strings.TrimSpace(x.Prose), Threats: threatsByEntity[x.ID]})
	}
	for _, x := range a.DataObjects {
		nodes = append(nodes, struct {
			ID      string
			Name    string
			Kind    string
			Desc    string
			Threats []map[string]any
		}{ID: x.ID, Name: nonEmpty(x.Name, x.ID), Kind: "store", Desc: strings.TrimSpace(x.SchemaRef), Threats: threatsByEntity[x.ID]})
	}
	for _, x := range a.Interfaces {
		nodes = append(nodes, struct {
			ID      string
			Name    string
			Kind    string
			Desc    string
			Threats []map[string]any
		}{ID: x.ID, Name: nonEmpty(x.Name, x.ID), Kind: "process", Desc: strings.TrimSpace(x.Protocol + " " + x.Endpoint), Threats: threatsByEntity[x.ID]})
	}

	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	cellIDByNode := map[string]string{}
	cells := []tdv2Cell{}
	for i, n := range nodes {
		cellUUID := deterministicUUID("td-cell|" + bundle.Architecture.Model.ID + "|" + n.ID)
		cellIDByNode[n.ID] = cellUUID
		col := i % 4
		row := i / 4
		shape := "process"
		tmType := "tm.Process"
		size := map[string]float64{"width": 180, "height": 90}
		attrs := map[string]any{"text": map[string]any{"text": n.Name}, "body": map[string]any{"stroke": "#333333", "strokeWidth": 1.0, "strokeDasharray": ""}}
		if n.Kind == "actor" {
			shape = "actor"
			tmType = "tm.Actor"
			size = map[string]float64{"width": 160, "height": 80}
		}
		if n.Kind == "store" {
			shape = "store"
			tmType = "tm.Store"
			attrs = map[string]any{"text": map[string]any{"text": n.Name}, "topLine": map[string]any{"stroke": "#333333", "strokeWidth": 1.0, "strokeDasharray": ""}, "bottomLine": map[string]any{"stroke": "#333333", "strokeWidth": 1.0, "strokeDasharray": ""}}
		}
		cells = append(cells, tdv2Cell{
			ID:     cellUUID,
			Shape:  shape,
			ZIndex: i + 1,
			Position: map[string]float64{
				"x": float64(60 + col*260),
				"y": float64(60 + row*170),
			},
			Size:  size,
			Attrs: attrs,
			Data: map[string]any{
				"name":                   n.Name,
				"description":            n.Desc,
				"type":                   tmType,
				"isTrustBoundary":        false,
				"outOfScope":             false,
				"reasonOutOfScope":       "",
				"threats":                n.Threats,
				"hasOpenThreats":         tdHasOpenThreats(n.Threats),
				"providesAuthentication": false,
				"isALog":                 false,
				"storesCredentials":      false,
				"isEncrypted":            false,
				"isSigned":               false,
			},
		})
	}

	tbZ := len(cells) + 1
	for _, tb := range a.TrustBoundaries {
		cells = append(cells, tdv2Cell{
			ID:       deterministicUUID("td-boundary|" + bundle.Architecture.Model.ID + "|" + tb.ID),
			Shape:    "trust-boundary-box",
			ZIndex:   tbZ,
			Size:     map[string]float64{"width": 260, "height": 180},
			Position: map[string]float64{"x": float64(30 + (tbZ%3)*300), "y": float64(30 + (tbZ/3)*220)},
			Attrs:    map[string]any{"text": map[string]any{"text": nonEmpty(tb.Name, tb.ID)}, "body": map[string]any{"stroke": "#999999", "strokeWidth": 1.0, "strokeDasharray": "3 3"}},
			Data:     map[string]any{"type": "tm.Boundary", "name": nonEmpty(tb.Name, tb.ID), "description": strings.TrimSpace(tb.Description), "isTrustBoundary": true, "hasOpenThreats": false},
		})
		tbZ++
	}

	flowZ := len(cells) + 10
	for _, m := range a.Mappings {
		t := strings.TrimSpace(m.Type)
		if t != "calls" && t != "publishes" && t != "subscribes" && t != "reads" && t != "writes" && t != "streams" && t != "interacts_with" {
			continue
		}
		fromID := cellIDByNode[strings.TrimSpace(m.From)]
		toID := cellIDByNode[strings.TrimSpace(m.To)]
		if fromID == "" || toID == "" {
			continue
		}
		name := strings.TrimSpace(m.Description)
		if name == "" {
			name = t
		}
		cells = append(cells, tdv2Cell{
			ID:        deterministicUUID("td-flow|" + bundle.Architecture.Model.ID + "|" + m.Type + "|" + m.From + "|" + m.To),
			Shape:     "flow",
			ZIndex:    flowZ,
			Connector: "smooth",
			Attrs:     map[string]any{"line": map[string]any{"stroke": "#333333", "strokeWidth": 1.0, "targetMarker": map[string]any{"name": "block"}, "strokeDasharray": ""}},
			Data: map[string]any{
				"type":             "tm.Flow",
				"name":             name,
				"description":      strings.TrimSpace(m.Description),
				"outOfScope":       false,
				"reasonOutOfScope": "",
				"protocol":         "",
				"isEncrypted":      false,
				"isPublicNetwork":  false,
				"hasOpenThreats":   false,
				"threats":          threatsByEntity[m.From],
				"isTrustBoundary":  false,
			},
			Source: map[string]any{"cell": fromID},
			Target: map[string]any{"cell": toID},
			Labels: []map[string]any{{"position": 0.5, "attrs": map[string]any{"label": map[string]any{"text": name}}}},
		})
		flowZ++
	}

	for _, f := range a.Flows {
		fromID := cellIDByNode[strings.TrimSpace(f.SourceRef)]
		toID := cellIDByNode[strings.TrimSpace(f.DestinationRef)]
		if fromID == "" || toID == "" {
			continue
		}
		name := nonEmpty(strings.TrimSpace(f.Title), strings.TrimSpace(f.ID))
		cells = append(cells, tdv2Cell{
			ID:        deterministicUUID("td-flow-authored|" + bundle.Architecture.Model.ID + "|" + f.ID),
			Shape:     "flow",
			ZIndex:    flowZ,
			Connector: "smooth",
			Attrs:     map[string]any{"line": map[string]any{"stroke": "#555555", "strokeWidth": 1.0, "targetMarker": map[string]any{"name": "block"}, "strokeDasharray": ""}},
			Data: map[string]any{
				"type":             "tm.Flow",
				"name":             name,
				"description":      strings.TrimSpace(f.Description),
				"outOfScope":       false,
				"reasonOutOfScope": "",
				"protocol":         strings.TrimSpace(f.Protocol),
				"isEncrypted":      strings.Contains(strings.ToLower(strings.TrimSpace(f.EncryptionInTransit)), "tls") || strings.Contains(strings.ToLower(strings.TrimSpace(f.EncryptionInTransit)), "encrypt"),
				"isPublicNetwork":  strings.ToLower(strings.TrimSpace(f.Direction)) == "inbound" || strings.ToLower(strings.TrimSpace(f.Direction)) == "outbound",
				"hasOpenThreats":   tdHasOpenThreats(threatsByEntity[f.ID]),
				"threats":          threatsByEntity[f.ID],
				"isTrustBoundary":  false,
			},
			Source: map[string]any{"cell": fromID},
			Target: map[string]any{"cell": toID},
			Labels: []map[string]any{{"position": 0.5, "attrs": map[string]any{"label": map[string]any{"text": name}}}},
		})
		flowZ++
	}

	return tdv2Document{
		Version: "2.0",
		Summary: tdv2Summary{
			Title:       nonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID)),
			Owner:       "engineering-model-go",
			Description: strings.TrimSpace(bundle.Architecture.Model.Introduction),
			ID:          0,
		},
		Detail: tdv2Detail{
			Contributors: []tdv2Contributor{{Name: "engineering-model-go"}},
			Diagrams: []tdv2Diagram{{
				Description: "Architecture-derived threat model diagram.",
				DiagramType: "STRIDE",
				ID:          0,
				Placeholder: "Generated from authored architecture.",
				Thumbnail:   "./public/content/images/thumbnail.stride.jpg",
				Title:       nonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID)) + " Threat Model",
				Version:     "2.0",
				Cells:       cells,
			}},
			DiagramTop: 1,
			Reviewer:   "engineering-model-go",
			ThreatTop:  threatCount,
		},
	}
}

func tdSeverity(in ...string) string {
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "high":
			return "High"
		case "medium":
			return "Medium"
		case "low":
			return "Low"
		}
	}
	return "Medium"
}

func tdStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved", "closed", "verified", "completed", "mitigated":
		return "Mitigated"
	default:
		return "Open"
	}
}

func tdHasOpenThreats(threats []map[string]any) bool {
	for _, t := range threats {
		if strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%v", t["status"])), "open") {
			return true
		}
	}
	return false
}

type otmDocument struct {
	OTMVersion      string              `json:"otmVersion"`
	Project         otmProject          `json:"project"`
	Representations []otmRepresentation `json:"representations,omitempty"`
	Assets          []otmAsset          `json:"assets,omitempty"`
	TrustZones      []otmTrustZone      `json:"trustZones"`
	Components      []otmComponent      `json:"components,omitempty"`
	Dataflows       []otmDataflow       `json:"dataflows"`
	Threats         []otmThreat         `json:"threats,omitempty"`
	Mitigations     []otmMitigation     `json:"mitigations,omitempty"`
}

type otmProject struct {
	Name        string         `json:"name"`
	ID          string         `json:"id"`
	Description string         `json:"description,omitempty"`
	Owner       string         `json:"owner,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Attributes  map[string]any `json:"attributes,omitempty"`
}

type otmRepresentation struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Type string `json:"type"`
}

type otmAsset struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Risk struct {
		Confidentiality float64 `json:"confidentiality"`
		Integrity       float64 `json:"integrity"`
		Availability    float64 `json:"availability"`
		Comment         string  `json:"comment,omitempty"`
	} `json:"risk"`
	Description string `json:"description,omitempty"`
}

type otmTrustZone struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Risk        struct {
		TrustRating float64 `json:"trustRating"`
	} `json:"risk"`
	Parent *struct {
		TrustZone string `json:"trustZone"`
	} `json:"parent,omitempty"`
}

type otmComponent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Parent      struct {
		TrustZone string `json:"trustZone"`
	} `json:"parent"`
	Threats []otmThreatRef `json:"threats,omitempty"`
	Tags    []string       `json:"tags,omitempty"`
}

type otmDataflow struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Bidirectional bool           `json:"bidirectional,omitempty"`
	Source        string         `json:"source"`
	Destination   string         `json:"destination"`
	Assets        []string       `json:"assets,omitempty"`
	Threats       []otmThreatRef `json:"threats,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
}

type otmThreat struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	CWEs        []string `json:"cwes,omitempty"`
	Risk        struct {
		Likelihood        float64 `json:"likelihood,omitempty"`
		LikelihoodComment string  `json:"likelihoodComment,omitempty"`
		Impact            float64 `json:"impact"`
		ImpactComment     string  `json:"impactComment"`
	} `json:"risk"`
	Tags []string `json:"tags,omitempty"`
}

type otmMitigation struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description,omitempty"`
	RiskReduction float64 `json:"riskReduction"`
}

type otmThreatRef struct {
	Threat      string             `json:"threat"`
	State       string             `json:"state"`
	Mitigations []otmMitigationRef `json:"mitigations,omitempty"`
}

type otmMitigationRef struct {
	Mitigation string `json:"mitigation"`
	State      string `json:"state"`
}

func buildOpenOTM(bundle model.Bundle) otmDocument {
	a := bundle.Architecture.AuthoredArchitecture
	doc := otmDocument{
		OTMVersion: "0.2.0",
		Project: otmProject{
			Name:        nonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID)),
			ID:          strings.TrimSpace(bundle.Architecture.Model.ID),
			Description: strings.TrimSpace(bundle.Architecture.Model.Introduction),
			Owner:       "engineering-model-go",
			Tags:        []string{"generated", "engineering-model"},
		},
		Representations: []otmRepresentation{{Name: "Architecture Threat View", ID: "rep-architecture", Type: "dfd"}},
		TrustZones:      []otmTrustZone{},
		Components:      []otmComponent{},
		Dataflows:       []otmDataflow{},
		Threats:         []otmThreat{},
		Mitigations:     []otmMitigation{},
	}

	if len(a.TrustBoundaries) == 0 {
		defaultTZ := otmTrustZone{ID: "TZ-SYSTEM", Name: "System Boundary", Type: "system", Description: "Default trust zone for components without explicit boundary mapping."}
		defaultTZ.Risk.TrustRating = 50
		doc.TrustZones = append(doc.TrustZones, defaultTZ)
	}
	for _, tb := range a.TrustBoundaries {
		z := otmTrustZone{ID: tb.ID, Name: nonEmpty(tb.Name, tb.ID), Type: strings.TrimSpace(tb.BoundaryType), Description: strings.TrimSpace(tb.Description)}
		z.Risk.TrustRating = 50
		if strings.TrimSpace(tb.ParentRef) != "" {
			z.Parent = &struct {
				TrustZone string `json:"trustZone"`
			}{TrustZone: strings.TrimSpace(tb.ParentRef)}
		}
		doc.TrustZones = append(doc.TrustZones, z)
	}
	sort.SliceStable(doc.TrustZones, func(i, j int) bool { return doc.TrustZones[i].ID < doc.TrustZones[j].ID })

	zoneForID := map[string]string{}
	for _, tb := range a.TrustBoundaries {
		for _, member := range tb.Members {
			member = strings.TrimSpace(member)
			if member == "" {
				continue
			}
			zoneForID[member] = tb.ID
		}
	}
	for _, m := range a.Mappings {
		if strings.TrimSpace(m.Type) != "bounded_by" {
			continue
		}
		zoneForID[strings.TrimSpace(m.From)] = strings.TrimSpace(m.To)
	}

	componentIDs := map[string]bool{}
	addComponent := func(id, name, kind, desc string) {
		id = strings.TrimSpace(id)
		if id == "" || componentIDs[id] {
			return
		}
		componentIDs[id] = true
		c := otmComponent{ID: id, Name: nonEmpty(strings.TrimSpace(name), id), Type: strings.TrimSpace(kind), Description: strings.TrimSpace(desc)}
		zone := strings.TrimSpace(zoneForID[id])
		if zone == "" {
			zone = "TZ-SYSTEM"
		}
		c.Parent.TrustZone = zone
		doc.Components = append(doc.Components, c)
	}

	for _, x := range a.Actors {
		addComponent(x.ID, x.Name, "actor", x.Description)
	}
	for _, x := range a.FunctionalUnits {
		addComponent(x.ID, x.Name, "service", x.Prose)
	}
	for _, x := range a.Interfaces {
		addComponent(x.ID, x.Name, "interface", x.Protocol+" "+x.Endpoint)
	}
	for _, x := range a.DeploymentTargets {
		addComponent(x.ID, x.Name, "deployment-target", x.Environment+" "+x.Region)
	}
	for _, x := range a.ReferencedElements {
		addComponent(x.ID, x.Name, x.Kind, x.Layer)
	}
	sort.SliceStable(doc.Components, func(i, j int) bool { return doc.Components[i].ID < doc.Components[j].ID })

	for _, x := range a.DataObjects {
		asset := otmAsset{ID: x.ID, Name: nonEmpty(x.Name, x.ID), Description: strings.TrimSpace(x.SchemaRef)}
		asset.Risk.Confidentiality = scoreLevel(nonEmpty(strings.TrimSpace(x.Confidentiality), strings.TrimSpace(x.Sensitivity)))
		asset.Risk.Integrity = scoreLevel(strings.TrimSpace(x.Integrity))
		asset.Risk.Availability = scoreLevel(strings.TrimSpace(x.Availability))
		asset.Risk.Comment = strings.Join(x.RegulatoryTags, ",")
		doc.Assets = append(doc.Assets, asset)
	}
	sort.SliceStable(doc.Assets, func(i, j int) bool { return doc.Assets[i].ID < doc.Assets[j].ID })

	mitigationByScenario := map[string][]otmMitigationRef{}
	for _, m := range a.ThreatMitigations {
		mid := strings.TrimSpace(m.ID)
		if mid == "" {
			continue
		}
		doc.Mitigations = append(doc.Mitigations, otmMitigation{
			ID:            mid,
			Name:          nonEmpty(strings.TrimSpace(m.ControlRef), mid),
			Description:   nonEmpty(strings.TrimSpace(m.Notes), strings.TrimSpace(m.ControlRef)),
			RiskReduction: scoreLevel(strings.TrimSpace(m.Effectiveness)),
		})
		sid := strings.TrimSpace(m.ThreatScenarioRef)
		if sid != "" {
			mitigationByScenario[sid] = append(mitigationByScenario[sid], otmMitigationRef{Mitigation: mid, State: otmState(m.Status)})
		}
	}
	sort.SliceStable(doc.Mitigations, func(i, j int) bool { return doc.Mitigations[i].ID < doc.Mitigations[j].ID })

	threatByID := map[string]otmThreat{}
	for _, ts := range a.ThreatScenarios {
		t := otmThreat{ID: strings.TrimSpace(ts.ID), Name: nonEmpty(strings.TrimSpace(ts.Title), strings.TrimSpace(ts.ID)), Description: strings.TrimSpace(ts.Summary)}
		if c := strings.TrimSpace(ts.Category); c != "" {
			t.Categories = append(t.Categories, c)
		}
		t.CWEs = append(t.CWEs, ts.CWE...)
		t.Risk.Likelihood = scoreLevel(strings.TrimSpace(ts.Likelihood))
		t.Risk.LikelihoodComment = strings.TrimSpace(ts.Status)
		t.Risk.Impact = scoreLevel(strings.TrimSpace(ts.Impact))
		t.Risk.ImpactComment = nonEmpty(strings.TrimSpace(ts.Severity), strings.TrimSpace(ts.Category))
		t.Tags = []string{nonEmpty(strings.TrimSpace(ts.Stride), "threat")}
		if t.ID != "" {
			threatByID[t.ID] = t
			doc.Threats = append(doc.Threats, t)
		}
	}
	sort.SliceStable(doc.Threats, func(i, j int) bool { return doc.Threats[i].ID < doc.Threats[j].ID })

	componentIndex := map[string]int{}
	for i, c := range doc.Components {
		componentIndex[c.ID] = i
	}
	for _, ts := range a.ThreatScenarios {
		ref := otmThreatRef{Threat: ts.ID, State: otmState(ts.Status), Mitigations: mitigationByScenario[ts.ID]}
		for _, id := range ts.AppliesTo {
			id = strings.TrimSpace(id)
			if idx, ok := componentIndex[id]; ok {
				doc.Components[idx].Threats = append(doc.Components[idx].Threats, ref)
			}
		}
	}

	assetByFlowID := map[string][]string{}
	for _, f := range a.Flows {
		assets := []string{}
		for _, d := range f.DataRefs {
			d = strings.TrimSpace(d)
			if d != "" {
				assets = append(assets, d)
			}
		}
		if len(assets) > 0 {
			assetByFlowID[strings.TrimSpace(f.ID)] = uniqueSorted(assets)
		}
	}

	for _, m := range a.Mappings {
		t := strings.TrimSpace(m.Type)
		if t != "calls" && t != "publishes" && t != "subscribes" && t != "reads" && t != "writes" && t != "streams" && t != "interacts_with" && t != "depends_on" {
			continue
		}
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		if !componentIDs[from] || !componentIDs[to] {
			continue
		}
		df := otmDataflow{
			ID:            deterministicUUID("otm-flow|" + bundle.Architecture.Model.ID + "|" + t + "|" + from + "|" + to),
			Name:          nonEmpty(strings.TrimSpace(m.Description), t),
			Description:   strings.TrimSpace(m.Description),
			Source:        from,
			Destination:   to,
			Bidirectional: t == "interacts_with" || t == "depends_on",
			Tags:          []string{t},
		}
		doc.Dataflows = append(doc.Dataflows, df)
	}

	for _, f := range a.Flows {
		from := strings.TrimSpace(f.SourceRef)
		to := strings.TrimSpace(f.DestinationRef)
		if !componentIDs[from] || !componentIDs[to] {
			continue
		}
		df := otmDataflow{
			ID:            strings.TrimSpace(f.ID),
			Name:          nonEmpty(strings.TrimSpace(f.Title), strings.TrimSpace(f.ID)),
			Description:   strings.TrimSpace(f.Description),
			Source:        from,
			Destination:   to,
			Bidirectional: strings.EqualFold(strings.TrimSpace(f.Direction), "bidirectional"),
			Assets:        assetByFlowID[strings.TrimSpace(f.ID)],
			Tags:          []string{nonEmpty(strings.TrimSpace(f.Kind), "flow")},
		}
		for _, scenarioID := range f.Threats {
			scenarioID = strings.TrimSpace(scenarioID)
			if scenarioID == "" {
				continue
			}
			ts := findThreatScenarioByID(a.ThreatScenarios, scenarioID)
			df.Threats = append(df.Threats, otmThreatRef{Threat: scenarioID, State: otmState(ts.Status), Mitigations: mitigationByScenario[scenarioID]})
		}
		doc.Dataflows = append(doc.Dataflows, df)
	}

	sort.SliceStable(doc.Dataflows, func(i, j int) bool { return doc.Dataflows[i].ID < doc.Dataflows[j].ID })
	return doc
}

func otmState(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved", "closed", "completed", "verified", "mitigated", "implemented":
		return "mitigated"
	case "partial":
		return "partial"
	default:
		return "open"
	}
}

func scoreLevel(level string) float64 {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "high":
		return 8
	case "medium":
		return 5
	case "low":
		return 2
	default:
		return 5
	}
}

func findThreatScenarioByID(all []model.ThreatScenario, id string) model.ThreatScenario {
	id = strings.TrimSpace(id)
	for _, ts := range all {
		if strings.TrimSpace(ts.ID) == id {
			return ts
		}
	}
	return model.ThreatScenario{}
}
