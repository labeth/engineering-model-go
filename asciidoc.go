package engmodel

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/labeth/engineering-model-go/codemap"
	"github.com/labeth/engineering-model-go/model"
	mermaidrenderer "github.com/labeth/engineering-model-go/render/mermaid"
	"github.com/labeth/engineering-model-go/validate"
	"github.com/labeth/engineering-model-go/view"
)

type AsciiDocOptions struct {
	ViewIDs  []string
	CodeRoot string
}

type AsciiDocResult struct {
	Document    string
	Diagnostics []validate.Diagnostic
}

type catalogTerm struct {
	Kind  string
	Entry model.CatalogEntry
}

type c4Node struct {
	ID          string
	Kind        string
	Label       string
	Description string
	Technology  string
	PartOf      string
}

type catalogLookup struct {
	ByName  map[string][]string
	ByAlias map[string][]string
}

func GenerateAsciiDocFromFiles(architecturePath, requirementsPath, designPath string, options AsciiDocOptions) (AsciiDocResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return AsciiDocResult{}, err
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		return AsciiDocResult{}, err
	}
	design, err := model.LoadDesign(designPath)
	if err != nil {
		return AsciiDocResult{}, err
	}
	if strings.TrimSpace(options.CodeRoot) != "" && !filepath.IsAbs(options.CodeRoot) {
		if _, err := os.Stat(options.CodeRoot); err == nil {
			// keep cwd-relative path as-is
		} else {
			baseDir := filepath.Dir(architecturePath)
			options.CodeRoot = filepath.Join(baseDir, options.CodeRoot)
		}
	}
	return GenerateAsciiDoc(bundle, requirements, design, options)
}

func GenerateAsciiDoc(bundle model.Bundle, requirements model.RequirementsDocument, design model.DesignDocument, options AsciiDocOptions) (AsciiDocResult, error) {
	diags := validate.Bundle(bundle)
	diags = append(diags, lintRequirementsEARS(requirements, bundle.Catalog)...)
	catalog := buildCatalogIndex(bundle.Catalog)
	c4Nodes := buildC4NodeIndex(bundle.Architecture.C4)

	viewIDs := resolveViewIDs(bundle, design, options)
	viewOutputs := map[string]string{}
	for _, viewID := range viewIDs {
		res, err := Generate(bundle, viewID)
		diags = append(diags, res.Diagnostics...)
		if err != nil {
			diags = validate.SortDiagnostics(diags)
			return AsciiDocResult{Diagnostics: diags}, fmt.Errorf("generate view %s: %w", viewID, err)
		}
		viewOutputs[viewID] = strings.TrimSpace(res.Mermaid)
	}

	chapters := append([]model.DesignChapter(nil), design.Design.Chapters...)
	sort.SliceStable(chapters, func(i, j int) bool { return chapters[i].ID < chapters[j].ID })
	cLookup := buildCatalogLookup(catalog)

	seenChapterID := map[string]bool{}
	for i, ch := range chapters {
		path := fmt.Sprintf("design.chapters[%d]", i)
		if strings.TrimSpace(ch.ID) == "" {
			diags = append(diags, validate.Diagnostic{
				Code:     "doc.chapter_missing_id",
				Severity: validate.SeverityError,
				Message:  "chapter id is required",
				Path:     path,
			})
			continue
		}
		if seenChapterID[ch.ID] {
			diags = append(diags, validate.Diagnostic{
				Code:     "doc.chapter_duplicate_id",
				Severity: validate.SeverityError,
				Message:  fmt.Sprintf("duplicate chapter id %q", ch.ID),
				Path:     path,
			})
			continue
		}
		seenChapterID[ch.ID] = true

		resolvedRefs := make([]string, 0, len(ch.CatalogRefs))
		if len(ch.CatalogRefs) == 0 {
			inferredRefs := inferCatalogRefsFromNarrative(ch.Narrative, catalog)
			if len(inferredRefs) == 0 {
				diags = append(diags, validate.Diagnostic{
					Code:     "doc.chapter_catalog_ref_inference_empty",
					Severity: validate.SeverityWarning,
					Message:  fmt.Sprintf("chapter %q has no explicit catalogRefs and no references could be inferred from narrative", ch.ID),
					Path:     path,
				})
			}
			resolvedRefs = append(resolvedRefs, inferredRefs...)
		} else {
			for _, rawRef := range uniqueSorted(ch.CatalogRefs) {
				resolvedRef, ok, ambiguous := resolveCatalogRef(rawRef, catalog, cLookup)
				if ok {
					resolvedRefs = append(resolvedRefs, resolvedRef)
					continue
				}
				if len(ambiguous) > 0 {
					diags = append(diags, validate.Diagnostic{
						Code:     "doc.chapter_ambiguous_catalog_ref",
						Severity: validate.SeverityError,
						Message:  fmt.Sprintf("chapter %q reference %q is ambiguous: %s", ch.ID, rawRef, strings.Join(ambiguous, ", ")),
						Path:     path,
					})
					continue
				}
				diags = append(diags, validate.Diagnostic{
					Code:     "doc.chapter_unknown_catalog_ref",
					Severity: validate.SeverityError,
					Message:  fmt.Sprintf("chapter %q references unknown catalog term %q", ch.ID, rawRef),
					Path:     path,
				})
			}
		}
		chapters[i].CatalogRefs = uniqueSorted(resolvedRefs)
	}

	diags = validate.SortDiagnostics(diags)
	if validate.HasErrors(diags) {
		return AsciiDocResult{Diagnostics: diags}, fmt.Errorf("design/model validation failed")
	}

	title := nonEmpty(strings.TrimSpace(design.Design.Title), strings.TrimSpace(bundle.Architecture.Model.Title))
	if strings.TrimSpace(title) == "" {
		title = "Architecture Document"
	}

	allRequirements := append([]model.Requirement(nil), requirements.Requirements...)
	sort.SliceStable(allRequirements, func(i, j int) bool { return allRequirements[i].ID < allRequirements[j].ID })
	requirementIDs := make([]string, 0, len(allRequirements))
	for _, r := range allRequirements {
		requirementIDs = append(requirementIDs, r.ID)
	}
	containerTech := buildContainerTechnologyIndex(bundle.Architecture.C4)
	reqToContainers := deriveRequirementToContainers(
		chapters,
		catalog,
		bundle.Architecture.Relationships,
		requirements.Requirements,
		c4Nodes,
	)
	textLinks := buildTextLinkIndex(catalog, c4Nodes, requirementIDs)

	codeSymbols := []codemap.Symbol{}
	if strings.TrimSpace(options.CodeRoot) != "" {
		syms, codeDiags, err := codemap.Scan(options.CodeRoot)
		if err != nil {
			return AsciiDocResult{Diagnostics: diags}, fmt.Errorf("scan code root: %w", err)
		}
		codeSymbols = syms
		diags = append(diags, codeDiags...)
		for _, s := range codeSymbols {
			for _, c4 := range s.PartOf {
				if _, ok := c4Nodes[c4]; !ok {
					diags = append(diags, validate.Diagnostic{
						Code:     "code.unknown_c4_ref",
						Severity: validate.SeverityWarning,
						Message:  fmt.Sprintf("code symbol %q references unknown C4 id %q", s.TraceID, c4),
						Path:     fmt.Sprintf("%s:%d", s.Path, s.Line),
					})
				}
			}
		}
		diags = validate.SortDiagnostics(diags)
	}

	viewSections := make([]asciidocViewSection, 0, len(viewIDs))
	for _, viewID := range viewIDs {
		viewSections = append(viewSections, asciidocViewSection{
			ID:      viewID,
			Mermaid: viewOutputs[viewID],
		})
	}

	chapterSections := make([]asciidocChapterSection, 0, len(chapters))
	warnedMissingDefinition := map[string]bool{}
	for _, ch := range chapters {
		catalogRefs := uniqueSorted(ch.CatalogRefs)
		terms := catalogTermsForRefs(catalog, catalogRefs)
		relationships := relationshipsForCatalogRefs(bundle.Architecture.Relationships, catalogRefs)
		derivedC4Refs := c4RefsFromRelationships(relationships)
		derivedReqs := requirementsForCatalogTerms(requirements.Requirements, terms)

		for _, ref := range catalogRefs {
			term, ok := catalog[ref]
			if !ok {
				continue
			}
			if strings.TrimSpace(term.Entry.Definition) == "" && !warnedMissingDefinition[ref] {
				warnedMissingDefinition[ref] = true
				diags = append(diags, validate.Diagnostic{
					Code:     "doc.catalog_definition_missing",
					Severity: validate.SeverityWarning,
					Message:  fmt.Sprintf("catalog term %q is referenced but has no definition", ref),
					Path:     fmt.Sprintf("design.chapter[%s]", ch.ID),
				})
			}
		}

		narrative := strings.TrimSpace(ch.Narrative)
		if narrative == "" {
			narrative = "This chapter describes a catalog-scoped architecture concern."
		}
		narrative = linkifyCatalogTerms(narrative, catalog)
		narrative = linkifyKnownIDs(narrative, textLinks)
		derivedReqRefText := "_none_"
		if len(derivedReqs) > 0 {
			reqIDs := make([]string, 0, len(derivedReqs))
			for _, r := range derivedReqs {
				reqIDs = append(reqIDs, r.ID)
			}
			derivedReqRefText = inlineRequirementRefList(reqIDs)
		}

		chapterSections = append(chapterSections, asciidocChapterSection{
			Anchor:                 anchorID("chapter", ch.ID),
			ID:                     ch.ID,
			Header:                 nonEmpty(strings.TrimSpace(ch.Title), ch.ID),
			Narrative:              narrative,
			HasRelationships:       len(relationships) > 0,
			Mermaid:                renderChapterMermaid(ch.ID, relationships, c4Nodes),
			CatalogRefs:            inlineCatalogRefList(catalogRefs, catalog),
			DerivedC4Refs:          inlineC4RefList(derivedC4Refs, c4Nodes, catalog),
			DerivedRequirementRefs: derivedReqRefText,
			DirectRelationships:    inlineRelationshipRefList(relationships, c4Nodes, catalog),
		})
	}

	requirementSections := make([]asciidocRequirementSection, 0, len(allRequirements))
	for _, r := range allRequirements {
		notes := strings.TrimSpace(r.Notes)
		text := linkifyCatalogTerms(strings.TrimSpace(r.Text), catalog)
		text = linkifyKnownIDs(text, textLinks)
		linkedNotes := linkifyCatalogTerms(notes, catalog)
		linkedNotes = linkifyKnownIDs(linkedNotes, textLinks)
		requirementSections = append(requirementSections, asciidocRequirementSection{
			Anchor:   r.ID,
			ID:       r.ID,
			Text:     text,
			HasNotes: notes != "",
			Notes:    linkedNotes,
		})
	}

	codeContainers := []asciidocCodeContainerSection{}
	unmappedSymbols := []asciidocCodeSymbolSection{}
	if len(codeSymbols) > 0 {
		type mappedSymbol struct {
			Symbol       codemap.Symbol
			Inferred     bool
			InferredFrom []string
		}

		byC4 := map[string][]mappedSymbol{}
		unmapped := []mappedSymbol{}
		for _, s := range codeSymbols {
			targets := uniqueSorted(s.PartOf)
			inferred := false
			inferredFrom := []string{}
			if len(targets) == 0 && len(s.Implements) > 0 {
				targets = inferContainersFromRequirements(uniqueSorted(s.Implements), reqToContainers)
				targets = applyLanguageAffinity(targets, s.Path, containerTech)
				if len(targets) > 0 {
					inferred = true
					inferredFrom = uniqueSorted(s.Implements)
				}
			}
			if len(targets) == 0 {
				unmapped = append(unmapped, mappedSymbol{Symbol: s})
				if len(s.Implements) > 0 {
					diags = append(diags, validate.Diagnostic{
						Code:     "code.part_of_not_inferred",
						Severity: validate.SeverityWarning,
						Message:  fmt.Sprintf("could not infer container mapping for symbol %q from requirements", s.TraceID),
						Path:     fmt.Sprintf("%s:%d", s.Path, s.Line),
					})
				}
				continue
			}
			for _, c4 := range targets {
				byC4[c4] = append(byC4[c4], mappedSymbol{
					Symbol:       s,
					Inferred:     inferred,
					InferredFrom: inferredFrom,
				})
			}
		}

		c4Keys := make([]string, 0, len(byC4))
		for c4 := range byC4 {
			c4Keys = append(c4Keys, c4)
		}
		sort.Strings(c4Keys)
		for _, c4 := range c4Keys {
			label := c4
			if n, ok := c4Nodes[c4]; ok {
				label = nonEmpty(strings.TrimSpace(n.Label), c4)
			}
			syms := byC4[c4]
			sort.SliceStable(syms, func(i, j int) bool {
				if syms[i].Symbol.Path != syms[j].Symbol.Path {
					return syms[i].Symbol.Path < syms[j].Symbol.Path
				}
				if syms[i].Symbol.Line != syms[j].Symbol.Line {
					return syms[i].Symbol.Line < syms[j].Symbol.Line
				}
				return syms[i].Symbol.TraceID < syms[j].Symbol.TraceID
			})

			symbolSections := make([]asciidocCodeSymbolSection, 0, len(syms))
			for _, s := range syms {
				signature := strings.TrimSpace(s.Symbol.Signature)
				reqs := inlineRequirementRefList(uniqueSorted(s.Symbol.Implements))
				symbolSections = append(symbolSections, asciidocCodeSymbolSection{
					TraceID:         s.Symbol.TraceID,
					PathLine:        fmt.Sprintf("%s:%d", s.Symbol.Path, s.Symbol.Line),
					HasSignature:    signature != "",
					Signature:       signature,
					HasRequirements: len(s.Symbol.Implements) > 0,
					Requirements:    reqs,
					HasMapping:      s.Inferred,
					Mapping:         inlineRequirementRefList(s.InferredFrom),
					MappingNote:     mappingNote(s.Inferred),
				})
			}
			codeContainers = append(codeContainers, asciidocCodeContainerSection{
				Label:   label,
				ID:      c4,
				Symbols: symbolSections,
			})
		}

		if len(unmapped) > 0 {
			sort.SliceStable(unmapped, func(i, j int) bool {
				if unmapped[i].Symbol.Path != unmapped[j].Symbol.Path {
					return unmapped[i].Symbol.Path < unmapped[j].Symbol.Path
				}
				return unmapped[i].Symbol.Line < unmapped[j].Symbol.Line
			})
			for _, s := range unmapped {
				unmappedSymbols = append(unmappedSymbols, asciidocCodeSymbolSection{
					TraceID:  s.Symbol.TraceID,
					PathLine: fmt.Sprintf("%s:%d", s.Symbol.Path, s.Symbol.Line),
				})
			}
		}
		diags = validate.SortDiagnostics(diags)
	}

	referenceSections := buildReferenceSections(c4Nodes, catalog)

	doc, err := renderAsciiDocTemplate(asciidocTemplateData{
		Title:             title,
		Views:             viewSections,
		HasReferenceIndex: len(referenceSections) > 0,
		ReferenceSections: referenceSections,
		Chapters:          chapterSections,
		Requirements:      requirementSections,
		HasCodeMapping:    len(codeSymbols) > 0,
		CodeContainers:    codeContainers,
		HasUnmapped:       len(unmappedSymbols) > 0,
		UnmappedSymbols:   unmappedSymbols,
	})
	if err != nil {
		return AsciiDocResult{Diagnostics: diags}, err
	}

	return AsciiDocResult{Document: doc, Diagnostics: diags}, nil
}

func resolveViewIDs(bundle model.Bundle, design model.DesignDocument, options AsciiDocOptions) []string {
	if len(options.ViewIDs) > 0 {
		return append([]string(nil), options.ViewIDs...)
	}
	if len(design.Design.Views) > 0 {
		return append([]string(nil), design.Design.Views...)
	}
	out := make([]string, 0, len(bundle.Architecture.Viewpoints))
	for _, vp := range bundle.Architecture.Viewpoints {
		out = append(out, vp.ID)
	}
	return out
}

func buildCatalogIndex(doc model.CatalogDocument) map[string]catalogTerm {
	index := map[string]catalogTerm{}
	add := func(kind string, in []model.CatalogEntry) {
		for _, e := range in {
			index[e.ID] = catalogTerm{Kind: kind, Entry: e}
		}
	}
	add("system", doc.Catalog.Systems)
	add("actor", doc.Catalog.Actors)
	add("event", doc.Catalog.Events)
	add("state", doc.Catalog.States)
	add("feature", doc.Catalog.Features)
	add("mode", doc.Catalog.Modes)
	add("condition", doc.Catalog.Conditions)
	add("data-term", doc.Catalog.DataTerms)
	return index
}

func buildCatalogLookup(index map[string]catalogTerm) catalogLookup {
	byName := map[string][]string{}
	byAlias := map[string][]string{}
	for id, term := range index {
		nameKey := normalizeForMatch(term.Entry.Name)
		if nameKey != "" {
			byName[nameKey] = append(byName[nameKey], id)
		}
		for _, alias := range term.Entry.Aliases {
			aliasKey := normalizeForMatch(alias)
			if aliasKey != "" {
				byAlias[aliasKey] = append(byAlias[aliasKey], id)
			}
		}
	}
	for k := range byName {
		sort.Strings(byName[k])
	}
	for k := range byAlias {
		sort.Strings(byAlias[k])
	}
	return catalogLookup{ByName: byName, ByAlias: byAlias}
}

func resolveCatalogRef(raw string, index map[string]catalogTerm, lookup catalogLookup) (string, bool, []string) {
	ref := strings.TrimSpace(raw)
	if ref == "" {
		return "", false, nil
	}
	if _, ok := index[ref]; ok {
		return ref, true, nil
	}
	key := normalizeForMatch(ref)
	if ids := uniqueSorted(lookup.ByName[key]); len(ids) == 1 {
		return ids[0], true, nil
	} else if len(ids) > 1 {
		return "", false, ids
	}
	if ids := uniqueSorted(lookup.ByAlias[key]); len(ids) == 1 {
		return ids[0], true, nil
	} else if len(ids) > 1 {
		return "", false, ids
	}
	return "", false, nil
}

func inferCatalogRefsFromNarrative(narrative string, index map[string]catalogTerm) []string {
	text := strings.TrimSpace(narrative)
	if text == "" {
		return nil
	}
	out := make([]string, 0)
	for id, term := range index {
		if containsPhrase(text, term.Entry.Name) {
			out = append(out, id)
			continue
		}
		for _, alias := range term.Entry.Aliases {
			if containsPhrase(text, alias) {
				out = append(out, id)
				break
			}
		}
	}
	return uniqueSorted(out)
}

func buildC4NodeIndex(c4 model.C4) map[string]c4Node {
	out := map[string]c4Node{}
	for _, p := range c4.People {
		out[p.ID] = c4Node{
			ID:          p.ID,
			Kind:        "person",
			Label:       nonEmpty(strings.TrimSpace(p.Name), p.ID),
			Description: strings.TrimSpace(p.Description),
		}
	}
	for _, s := range c4.SoftwareSystems {
		kind := "system"
		if strings.EqualFold(strings.TrimSpace(s.Kind), "external_system") {
			kind = "external_system"
		}
		out[s.ID] = c4Node{
			ID:    s.ID,
			Kind:  kind,
			Label: nonEmpty(strings.TrimSpace(s.Name), s.ID),
		}
	}
	for _, c := range c4.Containers {
		out[c.ID] = c4Node{
			ID:         c.ID,
			Kind:       "container",
			Label:      nonEmpty(strings.TrimSpace(c.Name), c.ID),
			Technology: strings.TrimSpace(c.Technology),
			PartOf:     strings.TrimSpace(c.PartOf),
		}
	}
	for _, c := range c4.Components {
		out[c.ID] = c4Node{
			ID:     c.ID,
			Kind:   "component",
			Label:  nonEmpty(strings.TrimSpace(c.Name), c.ID),
			PartOf: strings.TrimSpace(c.PartOf),
		}
	}
	return out
}

func catalogTermsForRefs(index map[string]catalogTerm, refs []string) []catalogTerm {
	out := make([]catalogTerm, 0, len(refs))
	for _, ref := range refs {
		if t, ok := index[ref]; ok {
			out = append(out, t)
		}
	}
	return out
}

func relationshipsForCatalogRefs(relationships []model.Relationship, catalogRefs []string) []model.Relationship {
	refSet := map[string]bool{}
	for _, ref := range catalogRefs {
		ref = strings.TrimSpace(ref)
		if ref != "" {
			refSet[ref] = true
		}
	}
	out := []model.Relationship{}
	for _, r := range relationships {
		for _, ref := range r.CatalogRefs {
			if refSet[ref] {
				out = append(out, r)
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return strings.TrimSpace(out[i].Description) < strings.TrimSpace(out[j].Description)
	})
	return out
}

func c4RefsFromRelationships(relationships []model.Relationship) []string {
	set := map[string]bool{}
	for _, r := range relationships {
		if strings.TrimSpace(r.From) != "" {
			set[r.From] = true
		}
		if strings.TrimSpace(r.To) != "" {
			set[r.To] = true
		}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func requirementsForCatalogTerms(requirements []model.Requirement, terms []catalogTerm) []model.Requirement {
	phrases := []string{}
	for _, term := range terms {
		if strings.TrimSpace(term.Entry.Name) != "" {
			phrases = append(phrases, term.Entry.Name)
		}
		phrases = append(phrases, term.Entry.Aliases...)
	}
	phrases = uniqueSorted(phrases)
	seen := map[string]bool{}
	out := []model.Requirement{}
	for _, r := range requirements {
		for _, p := range phrases {
			if containsPhrase(r.Text, p) {
				if !seen[r.ID] {
					seen[r.ID] = true
					out = append(out, r)
				}
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func renderChapterMermaid(chapterID string, relationships []model.Relationship, nodeIndex map[string]c4Node) string {
	nodes := map[string]c4Node{}
	for _, r := range relationships {
		if n, ok := nodeIndex[r.From]; ok {
			nodes[n.ID] = n
		} else {
			nodes[r.From] = c4Node{ID: r.From, Kind: "unknown", Label: r.From}
		}
		if n, ok := nodeIndex[r.To]; ok {
			nodes[n.ID] = n
		} else {
			nodes[r.To] = c4Node{ID: r.To, Kind: "unknown", Label: r.To}
		}
	}

	sortedNodes := make([]c4Node, 0, len(nodes))
	for _, n := range nodes {
		sortedNodes = append(sortedNodes, n)
	}
	sort.SliceStable(sortedNodes, func(i, j int) bool {
		if sortedNodes[i].Kind != sortedNodes[j].Kind {
			return sortedNodes[i].Kind < sortedNodes[j].Kind
		}
		return sortedNodes[i].ID < sortedNodes[j].ID
	})

	pv := view.ProjectedView{
		ID:    "chapter-" + chapterID,
		Kind:  "chapter",
		Title: chapterID,
	}
	for _, n := range sortedNodes {
		pv.Nodes = append(pv.Nodes, view.Node{
			ID:    n.ID,
			Label: n.Label,
			Kind:  n.Kind,
		})
	}
	for _, r := range relationships {
		label := strings.TrimSpace(r.Type)
		if strings.TrimSpace(r.Description) != "" {
			label = label + ": " + strings.TrimSpace(r.Description)
		}
		pv.Edges = append(pv.Edges, view.Edge{
			From:  r.From,
			To:    r.To,
			Type:  r.Type,
			Label: label,
		})
	}

	return strings.TrimSpace(mermaidrenderer.Render(pv))
}

func containsPhrase(text, phrase string) bool {
	a := normalizeForMatch(text)
	b := normalizeForMatch(phrase)
	if b == "" {
		return false
	}
	return strings.Contains(" "+a+" ", " "+b+" ")
}

func normalizeForMatch(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if !prevSpace {
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func anchorID(prefix, raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "item"
	}
	return prefix + "-" + out
}

func uniqueSorted(in []string) []string {
	set := map[string]bool{}
	for _, x := range in {
		x = strings.TrimSpace(x)
		if x != "" {
			set[x] = true
		}
	}
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func inlineCodeList(ids []string) string {
	if len(ids) == 0 {
		return "_none_"
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, "`"+id+"`")
	}
	return strings.Join(out, ", ")
}

func inlineCatalogRefList(ids []string, index map[string]catalogTerm) string {
	if len(ids) == 0 {
		return "_none_"
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if term, ok := index[id]; ok {
			label := nonEmpty(strings.TrimSpace(term.Entry.Name), id)
			out = append(out, "<<"+id+","+label+">>")
			continue
		}
		out = append(out, "`"+id+"`")
	}
	return strings.Join(out, ", ")
}

func inlineRequirementRefList(ids []string) string {
	if len(ids) == 0 {
		return "_none_"
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, "<<"+id+","+id+">>")
	}
	return strings.Join(out, ", ")
}

func inlineC4RefList(ids []string, nodes map[string]c4Node, catalog map[string]catalogTerm) string {
	if len(ids) == 0 {
		return "_none_"
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, linkRefForID(id, nodes, catalog))
	}
	return strings.Join(out, ", ")
}

func inlineRelationshipRefList(relationships []model.Relationship, nodes map[string]c4Node, catalog map[string]catalogTerm) string {
	if len(relationships) == 0 {
		return "_none_"
	}
	out := make([]string, 0, len(relationships))
	for _, r := range relationships {
		from := linkRefForID(r.From, nodes, catalog)
		to := linkRefForID(r.To, nodes, catalog)
		label := strings.TrimSpace(r.Type)
		if desc := strings.TrimSpace(r.Description); desc != "" {
			label += ": " + desc
		}
		out = append(out, from+" -> "+to+" ("+label+")")
	}
	return strings.Join(out, "; ")
}

func linkRefForID(id string, nodes map[string]c4Node, catalog map[string]catalogTerm) string {
	if term, ok := catalog[id]; ok {
		label := nonEmpty(strings.TrimSpace(term.Entry.Name), id)
		return "<<" + id + "," + label + ">>"
	}
	if n, ok := nodes[id]; ok {
		label := nonEmpty(strings.TrimSpace(n.Label), id)
		return "<<" + id + "," + label + ">>"
	}
	return "`" + id + "`"
}

func buildReferenceSections(nodes map[string]c4Node, catalog map[string]catalogTerm) []asciidocReferenceSection {
	entriesByKind := map[string][]asciidocReferenceEntry{}

	// Catalog entries are canonical when IDs overlap.
	catalogIDs := make([]string, 0, len(catalog))
	for id := range catalog {
		catalogIDs = append(catalogIDs, id)
	}
	sort.Strings(catalogIDs)
	for _, id := range catalogIDs {
		term := catalog[id]
		def := strings.TrimSpace(term.Entry.Definition)
		entriesByKind[term.Kind] = append(entriesByKind[term.Kind], asciidocReferenceEntry{
			Anchor:        id,
			ID:            id,
			Definition:    nonEmpty(def, "_missing definition_"),
			Term:          strings.TrimSpace(term.Entry.Name),
			Heading:       capitalizeFirst(strings.TrimSpace(term.Entry.Name)),
			Aliases:       inlineCodeList(uniqueSorted(term.Entry.Aliases)),
			HasParent:     false,
			ParentRef:     "",
			HasDefinition: def != "",
		})
	}

	// Include C4 entries that are not already defined in catalog.
	nodeIDs := make([]string, 0, len(nodes))
	for id := range nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)
	for _, id := range nodeIDs {
		if _, exists := catalog[id]; exists {
			continue
		}
		n := nodes[id]
		def := strings.TrimSpace(architectureNarrative(n))
		entriesByKind[n.Kind] = append(entriesByKind[n.Kind], asciidocReferenceEntry{
			Anchor:        id,
			ID:            id,
			Definition:    nonEmpty(def, "_missing definition_"),
			Term:          strings.TrimSpace(n.Label),
			Heading:       capitalizeFirst(strings.TrimSpace(n.Label)),
			Aliases:       "_none_",
			HasParent:     n.PartOf != "",
			ParentRef:     linkRefForID(n.PartOf, nodes, catalog),
			HasDefinition: def != "",
		})
	}

	kindOrder := []string{
		"actor",
		"system",
		"external_system",
		"container",
		"component",
		"event",
		"state",
		"feature",
		"mode",
		"condition",
		"data-term",
		"person",
		"unknown",
	}
	seenKinds := map[string]bool{}
	sections := make([]asciidocReferenceSection, 0, len(entriesByKind))
	for _, kind := range kindOrder {
		items := entriesByKind[kind]
		if len(items) == 0 {
			continue
		}
		seenKinds[kind] = true
		sections = append(sections, asciidocReferenceSection{
			Kind:      kind,
			KindTitle: referenceKindTitle(kind),
			Entries:   items,
		})
	}

	remainingKinds := make([]string, 0)
	for kind := range entriesByKind {
		if !seenKinds[kind] {
			remainingKinds = append(remainingKinds, kind)
		}
	}
	sort.Strings(remainingKinds)
	for _, kind := range remainingKinds {
		sections = append(sections, asciidocReferenceSection{
			Kind:      kind,
			KindTitle: referenceKindTitle(kind),
			Entries:   entriesByKind[kind],
		})
	}
	return sections
}

func deriveRequirementToContainers(
	chapters []model.DesignChapter,
	catalog map[string]catalogTerm,
	relationships []model.Relationship,
	requirements []model.Requirement,
	c4Nodes map[string]c4Node,
) map[string][]string {
	reqToContainersSet := map[string]map[string]bool{}
	for _, ch := range chapters {
		catalogRefs := uniqueSorted(ch.CatalogRefs)
		terms := catalogTermsForRefs(catalog, catalogRefs)
		derivedReqs := requirementsForCatalogTerms(requirements, terms)
		derivedRels := relationshipsForCatalogRefs(relationships, catalogRefs)
		derivedC4 := c4RefsFromRelationships(derivedRels)
		containers := []string{}
		for _, id := range derivedC4 {
			if n, ok := c4Nodes[id]; ok && n.Kind == "container" {
				containers = append(containers, id)
			}
		}
		containers = uniqueSorted(containers)
		for _, req := range derivedReqs {
			if _, ok := reqToContainersSet[req.ID]; !ok {
				reqToContainersSet[req.ID] = map[string]bool{}
			}
			for _, c := range containers {
				reqToContainersSet[req.ID][c] = true
			}
		}
	}
	out := map[string][]string{}
	for req, set := range reqToContainersSet {
		values := make([]string, 0, len(set))
		for c := range set {
			values = append(values, c)
		}
		sort.Strings(values)
		out[req] = values
	}
	return out
}

func inferContainersFromRequirements(requirements []string, reqToContainers map[string][]string) []string {
	set := map[string]bool{}
	for _, req := range requirements {
		for _, c := range reqToContainers[req] {
			set[c] = true
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func buildContainerTechnologyIndex(c4 model.C4) map[string]string {
	out := map[string]string{}
	for _, c := range c4.Containers {
		out[c.ID] = strings.ToLower(strings.TrimSpace(c.Technology))
	}
	return out
}

func applyLanguageAffinity(candidates []string, symbolPath string, containerTech map[string]string) []string {
	ext := strings.ToLower(filepath.Ext(symbolPath))
	want := ""
	switch ext {
	case ".go":
		want = "go"
	case ".rs":
		want = "rust"
	case ".ts", ".tsx":
		want = "typescript"
	default:
		return candidates
	}
	filtered := make([]string, 0, len(candidates))
	for _, c := range candidates {
		tech := containerTech[c]
		if tech == "" {
			continue
		}
		if strings.Contains(tech, want) {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return candidates
	}
	sort.Strings(filtered)
	return filtered
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func capitalizeFirst(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func referenceKindTitle(kind string) string {
	k := strings.TrimSpace(kind)
	if k == "" {
		return k
	}
	k = strings.ReplaceAll(k, "_", " ")
	switch k {
	case "actor":
		return "Actor(s)"
	case "system":
		return "System(s)"
	case "external system":
		return "External system(s)"
	case "container":
		return "Container(s)"
	case "component":
		return "Component(s)"
	case "event":
		return "Event(s)"
	case "state":
		return "State(s)"
	case "feature":
		return "Feature(s)"
	case "mode":
		return "Mode(s)"
	case "condition":
		return "Condition(s)"
	case "data-term":
		return "Data term(s)"
	case "person":
		return "Person(s)"
	case "unknown":
		return "Unknown(s)"
	default:
		return capitalizeFirst(k) + "(s)"
	}
}

func architectureNarrative(n c4Node) string {
	switch n.Kind {
	case "person":
		if n.Description != "" {
			return n.Description
		}
		return n.Label + " is modeled as an external actor in the system context."
	case "system":
		return n.Label + " is modeled as an internal software-system boundary."
	case "external_system":
		return n.Label + " is an external dependency integrated through explicit relationships."
	case "container":
		if n.Technology != "" {
			return n.Label + " runs as a container-level runtime unit using " + n.Technology + "."
		}
		return n.Label + " is a container-level runtime unit in the architecture."
	case "component":
		return n.Label + " is a component-level responsibility within its parent container."
	default:
		return ""
	}
}

func mappingNote(inferred bool) string {
	if inferred {
		return "Inferred from traced requirements."
	}
	return "Explicitly assigned through TRACE-PART-OF."
}

func buildTextLinkIndex(catalog map[string]catalogTerm, c4Nodes map[string]c4Node, requirementIDs []string) map[string]string {
	out := map[string]string{}
	for id, term := range catalog {
		label := nonEmpty(strings.TrimSpace(term.Entry.Name), id)
		out[id] = "<<" + id + "," + label + ">>"
	}
	for id, node := range c4Nodes {
		if _, exists := out[id]; exists {
			continue
		}
		label := nonEmpty(strings.TrimSpace(node.Label), id)
		out[id] = "<<" + id + "," + label + ">>"
	}
	for _, id := range requirementIDs {
		out[id] = "<<" + id + "," + id + ">>"
	}
	return out
}

type textSpan struct {
	Start int
	End   int
	Link  string
}

type phraseLink struct {
	Phrase string
	Link   string
}

func linkifyCatalogTerms(text string, catalog map[string]catalogTerm) string {
	if strings.TrimSpace(text) == "" || len(catalog) == 0 {
		return text
	}
	candidates := buildCatalogPhraseLinks(catalog)
	if len(candidates) == 0 {
		return text
	}

	occupied := make([]bool, len(text))
	// Do not attempt replacements inside existing AsciiDoc xrefs.
	xrefRe := regexp.MustCompile(`<<[^>]+>>`)
	for _, loc := range xrefRe.FindAllStringIndex(text, -1) {
		for i := loc[0]; i < loc[1] && i < len(occupied); i++ {
			occupied[i] = true
		}
	}

	spans := make([]textSpan, 0)
	for _, c := range candidates {
		re := regexp.MustCompile(catalogPhrasePattern(c.Phrase))
		matches := re.FindAllStringSubmatchIndex(text, -1)
		for _, m := range matches {
			if len(m) < 6 {
				continue
			}
			start := m[4]
			end := m[5]
			if start < 0 || end <= start || end > len(text) {
				continue
			}
			if overlaps(occupied, start, end) {
				continue
			}
			for i := start; i < end; i++ {
				occupied[i] = true
			}
			spans = append(spans, textSpan{Start: start, End: end, Link: c.Link})
		}
	}

	if len(spans) == 0 {
		return text
	}
	sort.SliceStable(spans, func(i, j int) bool {
		return spans[i].Start < spans[j].Start
	})

	var b strings.Builder
	cursor := 0
	for _, s := range spans {
		if s.Start < cursor {
			continue
		}
		b.WriteString(text[cursor:s.Start])
		b.WriteString(s.Link)
		cursor = s.End
	}
	b.WriteString(text[cursor:])
	return b.String()
}

func buildCatalogPhraseLinks(catalog map[string]catalogTerm) []phraseLink {
	ids := make([]string, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	phraseToIDs := map[string][]string{}
	phraseVariants := map[string]map[string]bool{}

	addPhrase := func(id, phrase string) {
		phrase = strings.TrimSpace(phrase)
		if phrase == "" {
			return
		}
		key := normalizeForMatch(phrase)
		if key == "" {
			return
		}
		phraseToIDs[key] = append(phraseToIDs[key], id)
		if phraseVariants[key] == nil {
			phraseVariants[key] = map[string]bool{}
		}
		phraseVariants[key][phrase] = true
	}

	for _, id := range ids {
		term := catalog[id]
		addPhrase(id, term.Entry.Name)
		for _, alias := range term.Entry.Aliases {
			addPhrase(id, alias)
		}
	}

	out := make([]phraseLink, 0)
	for key, idList := range phraseToIDs {
		uniqIDs := uniqueSorted(idList)
		if len(uniqIDs) != 1 {
			continue
		}
		id := uniqIDs[0]
		term, ok := catalog[id]
		if !ok {
			continue
		}
		label := nonEmpty(strings.TrimSpace(term.Entry.Name), id)
		link := "<<" + id + "," + label + ">>"
		variants := make([]string, 0, len(phraseVariants[key]))
		for v := range phraseVariants[key] {
			variants = append(variants, v)
		}
		sort.SliceStable(variants, func(i, j int) bool {
			if len(variants[i]) != len(variants[j]) {
				return len(variants[i]) > len(variants[j])
			}
			return variants[i] < variants[j]
		})
		for _, v := range variants {
			out = append(out, phraseLink{Phrase: v, Link: link})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].Phrase) != len(out[j].Phrase) {
			return len(out[i].Phrase) > len(out[j].Phrase)
		}
		return out[i].Phrase < out[j].Phrase
	})
	return out
}

func overlaps(occupied []bool, start, end int) bool {
	for i := start; i < end; i++ {
		if occupied[i] {
			return true
		}
	}
	return false
}

func catalogPhrasePattern(phrase string) string {
	tokens := strings.Fields(strings.TrimSpace(phrase))
	if len(tokens) == 0 {
		return `a\Ab`
	}
	escaped := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		escaped = append(escaped, regexp.QuoteMeta(tok))
	}
	core := strings.Join(escaped, `\s+`)
	return `(?i)(^|[^A-Za-z0-9])(` + core + `)([^A-Za-z0-9]|$)`
}

func linkifyKnownIDs(text string, links map[string]string) string {
	if strings.TrimSpace(text) == "" || len(links) == 0 {
		return text
	}
	var b strings.Builder
	for i := 0; i < len(text); {
		if i+1 < len(text) && text[i] == '<' && text[i+1] == '<' {
			if end := strings.Index(text[i+2:], ">>"); end >= 0 {
				stop := i + 2 + end + 2
				b.WriteString(text[i:stop])
				i = stop
				continue
			}
		}
		r := rune(text[i])
		if isIDRune(r) {
			j := i + 1
			for j < len(text) && isIDRune(rune(text[j])) {
				j++
			}
			token := text[i:j]
			if link, ok := links[token]; ok {
				b.WriteString(link)
			} else {
				b.WriteString(token)
			}
			i = j
			continue
		}
		b.WriteByte(text[i])
		i++
	}
	return b.String()
}

func isIDRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-'
}
