package engmodel

import "github.com/labeth/engineering-model-go/validate"

type AIViewOptions struct {
	ViewIDs  []string
	CodeRoot string
}

type AIViewResult struct {
	Document    AIViewDocument
	JSON        string
	Markdown    string
	EdgesNDJSON string
	Diagnostics []validate.Diagnostic
}

type AIViewDocument struct {
	SchemaVersion string          `json:"schema_version"`
	Model         AIModelSummary  `json:"model"`
	EntryPoints   []AIEntryPoint  `json:"entry_points"`
	EntityIndex   AIEntityIndex   `json:"entity_index"`
	SupportPaths  []AISupportPath `json:"support_paths"`
	Entities      []AIEntity      `json:"entities"`
	SourceBlocks  []AISourceBlock `json:"source_blocks"`
}

type AIModelSummary struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Counts        AIModelCounts `json:"counts"`
	EntryPointIDs []string      `json:"entry_point_ids"`
}

type AIModelCounts struct {
	FunctionalGroups int `json:"fg"`
	FunctionalUnits  int `json:"fu"`
	Requirements     int `json:"req"`
	Runtime          int `json:"rt"`
	Code             int `json:"code"`
	Verification     int `json:"verification"`
	Views            int `json:"views"`
}

type AIEntryPoint struct {
	ID        string   `json:"id"`
	Kind      string   `json:"kind"`
	Title     string   `json:"title"`
	EntityIDs []string `json:"entity_ids"`
}

type AIEntityIndex struct {
	FunctionalGroupIDs []string         `json:"fg_ids"`
	FunctionalUnitIDs  []string         `json:"fu_ids"`
	RequirementIDs     []string         `json:"req_ids"`
	RuntimeIDs         []string         `json:"rt_ids"`
	CodeIDs            []string         `json:"code_ids"`
	VerificationIDs    []string         `json:"verification_ids"`
	Lookup             []AIEntityLookup `json:"lookup"`
}

type AIEntityLookup struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
}

type AISupportPath struct {
	ID                string   `json:"id"`
	FromID            string   `json:"from_id"`
	ToVerificationIDs []string `json:"to_verification_ids"`
	Path              []string `json:"path"`
	Summary           string   `json:"summary"`
	Confidence        string   `json:"confidence"`
	SourceRefs        []string `json:"source_refs"`
}

type AIEntity struct {
	ID              string              `json:"id"`
	Kind            string              `json:"kind"`
	Title           string              `json:"title"`
	Summary         string              `json:"summary"`
	Origin          string              `json:"origin"`
	Status          string              `json:"status,omitempty"`
	GroupID         string              `json:"group_id,omitempty"`
	RequirementIDs  []string            `json:"requirement_ids,omitempty"`
	RuntimeIDs      []string            `json:"runtime_ids,omitempty"`
	CodeIDs         []string            `json:"code_ids,omitempty"`
	VerificationIDs []string            `json:"verification_ids,omitempty"`
	RelatedIDs      []string            `json:"related_ids,omitempty"`
	Fields          AIEntityFields      `json:"fields,omitempty"`
	FieldProvenance []AIFieldProvenance `json:"field_provenance,omitempty"`
	SourceRefs      []string            `json:"source_refs"`
}

type AIEntityFields struct {
	Triggers []string `json:"triggers,omitempty"`
	Consumes []string `json:"consumes,omitempty"`
	Produces []string `json:"produces,omitempty"`
	Threats  []string `json:"threats,omitempty"`
}

type AIFieldProvenance struct {
	Field      string   `json:"field"`
	Origin     string   `json:"origin"`
	Confidence string   `json:"confidence"`
	SourceRefs []string `json:"source_refs"`
}

type AISourceBlock struct {
	ID        string   `json:"id"`
	Path      string   `json:"path"`
	LineStart int      `json:"line_start,omitempty"`
	LineEnd   int      `json:"line_end,omitempty"`
	Kind      string   `json:"kind"`
	EntityIDs []string `json:"entity_ids"`
	Summary   string   `json:"summary"`
}

type AIEdge struct {
	FromID     string   `json:"from_id"`
	ToID       string   `json:"to_id"`
	Relation   string   `json:"relation"`
	Origin     string   `json:"origin"`
	Confidence string   `json:"confidence"`
	SourceRefs []string `json:"source_refs"`
}
