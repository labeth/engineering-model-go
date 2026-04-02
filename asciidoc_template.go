package engmodel

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed templates/asciidoc.tmpl
var asciidocTemplateText string

var asciidocTemplate = template.Must(template.New("asciidoc").Parse(asciidocTemplateText))

type asciidocTemplateData struct {
	Title             string
	Views             []asciidocViewSection
	HasReferenceIndex bool
	ReferenceSections []asciidocReferenceSection
	Chapters          []asciidocChapterSection
	Requirements      []asciidocRequirementSection
	HasCodeMapping    bool
	CodeContainers    []asciidocCodeContainerSection
	HasUnmapped       bool
	UnmappedSymbols   []asciidocCodeSymbolSection
}

type asciidocViewSection struct {
	ID      string
	Mermaid string
}

type asciidocChapterSection struct {
	Anchor                 string
	ID                     string
	Header                 string
	Narrative              string
	HasRelationships       bool
	Mermaid                string
	CatalogRefs            string
	DerivedC4Refs          string
	DerivedRequirementRefs string
	DirectRelationships    string
}

type asciidocRequirementSection struct {
	Anchor   string
	ID       string
	Text     string
	HasNotes bool
	Notes    string
}

type asciidocReferenceSection struct {
	Kind      string
	KindTitle string
	Entries   []asciidocReferenceEntry
}

type asciidocReferenceEntry struct {
	Anchor        string
	ID            string
	Definition    string
	Term          string
	Heading       string
	Aliases       string
	HasParent     bool
	ParentRef     string
	HasDefinition bool
}

type asciidocCodeContainerSection struct {
	Label   string
	ID      string
	Symbols []asciidocCodeSymbolSection
}

type asciidocCodeSymbolSection struct {
	TraceID         string
	PathLine        string
	HasSignature    bool
	Signature       string
	HasRequirements bool
	Requirements    string
	HasMapping      bool
	Mapping         string
	MappingNote     string
}

func renderAsciiDocTemplate(data asciidocTemplateData) (string, error) {
	var b bytes.Buffer
	if err := asciidocTemplate.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute asciidoc template: %w", err)
	}
	return b.String(), nil
}
