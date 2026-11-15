// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// vocabSpec describes a list of values restricted to a vocabulary (compared trimmed).
type vocabSpec struct {
	allowed      map[string]bool
	code, format string
}

// vocabulary records the spec's error at path for each value outside the vocabulary.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) vocabulary(path string, values []string, spec vocabSpec) {
	for _, value := range values {
		if !spec.allowed[strings.TrimSpace(value)] {
			v.fail(spec.code, path, fmt.Sprintf(spec.format, value))
		}
	}
}

// checkViews validates each architecture view.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkViews() {
	for i, view := range v.b.Architecture.Views {
		path := indexedPath("views", i)
		v.checkViewShape(path, view)
		v.checkViewFilters(path, view)
	}
}

// checkViewShape requires an id, a known kind, roots that exist and a non-negative depth.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkViewShape(path string, view model.View) {
	v.requireText(path, view.ID, textSpec{"model.missing_id", "view id is required"})
	if !allowedViewKinds[view.Kind] {
		v.fail("model.unknown_view_kind", path, fmt.Sprintf("unknown view kind %q", view.Kind))
	}
	if len(view.Roots) == 0 {
		v.fail("model.empty_roots", path, "view must have at least one root")
	}
	for _, root := range view.Roots {
		if !v.validID(root) {
			v.fail("model.broken_reference", path, fmt.Sprintf("view root %q does not exist", root))
		}
	}
	if view.MaxDepth < 0 {
		v.fail("model.invalid_view_depth", path, "view maxDepth must be >= 0")
	}
}

// checkViewFilters restricts the kinds and mapping types a view includes or excludes.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkViewFilters(path string, view model.View) {
	kinds := func(format string) vocabSpec {
		return vocabSpec{allowedViewEntityKinds, "model.unknown_view_entity_kind", format}
	}
	mappings := func(format string) vocabSpec {
		return vocabSpec{allowedViewMappingTypes, "model.unknown_view_mapping_type", format}
	}
	v.vocabulary(path, view.IncludeKinds, kinds("unknown includeKinds value %q"))
	v.vocabulary(path, view.ExcludeKinds, kinds("unknown excludeKinds value %q"))
	v.vocabulary(path, view.IncludeMappings, mappings("unknown includeMappings value %q"))
	v.vocabulary(path, view.ExcludeMappings, mappings("unknown excludeMappings value %q"))
}

// checkDocuments validates each document definition.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkDocuments() {
	for i, d := range v.b.Views.Documents {
		path := indexedPath("documents", i)
		v.checkDocumentFields(path, d)
		v.checkDocumentControl(path, d)
		v.checkDocumentRefs(path, d)
		sections := documentSections{v: v, docID: d.ID, seen: map[string]bool{}}
		for j, section := range d.Sections {
			sections.check(indexedPath(path+".sections", j), section)
		}
	}
}

// checkDocumentFields requires a title, kind, purpose and content.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkDocumentFields(path string, d model.DocumentDefinition) {
	v.requireText(path+".title", d.Title, textSpec{"model.missing_document_title", "document title is required"})
	v.requireText(path+".kind", d.Kind, textSpec{"model.missing_document_kind", "document kind is required"})
	v.requireText(path+".purpose", d.Purpose, textSpec{"model.missing_document_purpose", "document purpose is required"})
	if len(d.ContentRefs) == 0 && len(d.Sections) == 0 {
		v.fail("model.empty_document_content", path, fmt.Sprintf("document %q must select model content or define authored sections", d.ID))
	}
}

// checkDocumentControl requires the control fields of a document and a known status and issue date.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkDocumentControl(path string, d model.DocumentDefinition) {
	c := d.Control
	for _, field := range []struct{ name, value string }{
		{"identifier", c.Identifier}, {"revision", c.Revision}, {"status", c.Status},
		{"issuedBy", c.IssuedBy}, {"issueDate", c.IssueDate}, {"language", c.Language},
		{"documentType", c.DocumentType}, {"confidentiality", c.Confidentiality},
		{"securityClassification", c.SecurityClassification}, {"countryOfOrigin", c.CountryOfOrigin},
	} {
		v.requireText(path+".control."+field.name, field.value, textSpec{"model.missing_document_control_field", fmt.Sprintf("document %q control field %s is required", d.ID, field.name)})
	}
	v.enumValue(path+".control.status", c.Status, enumSpec{allowedDocumentStatus, "model.invalid_document_status", "unknown document status %q", nil})
	v.isoDateValue(path+".control.issueDate", c.IssueDate, textSpec{"model.invalid_document_issue_date", "invalid document issueDate %q, expected YYYY-MM-DD"})
}

// validContent reports whether an id names selectable content: a valid id that is not a document.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) validContent(id string) bool {
	return v.validID(id) && !v.ids.documents[id]
}

// checkDocumentRefs requires the stakeholders, references and content a document names to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkDocumentRefs(path string, d model.DocumentDefinition) {
	lead := []any{d.ID}
	v.refList(path, d.StakeholderRefs, refSpec{ok: inSet(v.ids.actors), code: "model.invalid_document_stakeholder_ref", format: "document %q references unknown stakeholder %q", field: "stakeholderRefs", lead: lead})
	v.refList(path, d.ReferenceRefs, refSpec{ok: inSet(v.ids.references), code: "model.invalid_document_reference_ref", format: "document %q references unknown external reference %q", field: "referenceRefs", lead: lead})
	v.refList(path, d.ContentRefs, refSpec{ok: v.validContent, code: "model.invalid_document_content_ref", format: "document %q selects unknown or invalid content %q", field: "contentRefs", lead: lead})
}

// documentSections checks the sections of one document, remembering the ids seen.
type documentSections struct {
	v     *bundleValidator
	docID string
	seen  map[string]bool
}

// check requires a section id that is unique within the document, a title and valid includes.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (s documentSections) check(path string, section model.DocumentSection) {
	id := strings.TrimSpace(section.ID)
	if id == "" {
		s.v.fail("model.missing_document_section_id", path+".id", fmt.Sprintf("document %q has a section without an id", s.docID))
	} else if s.seen[id] {
		s.v.fail("model.duplicate_document_section_id", path+".id", fmt.Sprintf("document %q repeats section id %q", s.docID, id))
	}
	s.seen[id] = true
	s.v.requireText(path+".title", section.Title, textSpec{"model.missing_document_section_title", fmt.Sprintf("document %q section %q has no title", s.docID, id)})
	s.v.refList(path, section.IncludeRefs, refSpec{ok: s.v.validContent, code: "model.invalid_document_section_ref", format: "document %q section %q selects unknown or invalid content %q", field: "includeRefs", lead: []any{s.docID, id}})
}
