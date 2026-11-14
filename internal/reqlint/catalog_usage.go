// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package reqlint

import (
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// TermUsage is where one catalog term is used: the requirements whose text names it (by name or alias, on word
// boundaries, the rule of the EARS lint) and the other entities that hold its ID in a field or mention it.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
type TermUsage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Group is the catalog list the term sits in: actors, events, states, features, modes, conditions, dataTerms,
	// systems, functionalGroups, functionalUnits, referencedElements or attackVectors.
	Group      string   `json:"group"`
	Definition string   `json:"definition,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	// Requirements are the IDs of requirements whose text names the term.
	Requirements []string `json:"requirements"`
	// ReferencedBy are the IDs of other entities that hold the term's ID in a field or mention it.
	ReferencedBy []string `json:"referencedBy"`
}

// CatalogUsage lists every catalog term with the requirements and entities that use it, in group and ID order.
// A term nothing uses has two empty lists. It reads the bundle only.
// TRLC-LINKS: REQ-EMG-193
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func CatalogUsage(bundle model.Bundle) ([]TermUsage, error) {
	index, err := model.BuildEntityIndex(bundle)
	if err != nil {
		return nil, err
	}
	referencing := referencingEntities(index)
	reqs := lowercasedRequirements(bundle.Requirements.Requirements)
	out := []TermUsage{}
	for _, grp := range catalogGroups(bundle.Catalog.Catalog) {
		for _, e := range grp.entries {
			out = append(out, TermUsage{
				ID: e.ID, Name: e.Name, Group: grp.name, Definition: strings.TrimSpace(e.Definition),
				Aliases: append([]string(nil), e.Aliases...), Requirements: reqs.naming(termPhrases(e)),
				ReferencedBy: append([]string{}, referencing[e.ID]...),
			})
		}
	}
	return out, nil
}

// referencingEntities maps an entity ID to the sorted IDs of the other entities whose field holds it or whose prose
// mentions it.
// TRLC-LINKS: REQ-EMG-193
func referencingEntities(index model.EntityIndex) map[string][]string {
	out := map[string][]string{}
	for to, set := range referenceSources(index.References) {
		out[to] = sortedSet(set)
	}
	return out
}

// referenceSources maps an entity ID to the set of entities that reference it by a field or by prose.
// TRLC-LINKS: REQ-EMG-193
func referenceSources(refs []model.EntityReference) map[string]map[string]bool {
	sets := map[string]map[string]bool{}
	for _, r := range refs {
		if r.From == r.To || (r.Type != model.EdgeField && r.Type != model.EdgeMention) {
			continue
		}
		if sets[r.To] == nil {
			sets[r.To] = map[string]bool{}
		}
		sets[r.To][r.From] = true
	}
	return sets
}

// TRLC-LINKS: REQ-EMG-193
func sortedSet(set map[string]bool) []string {
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// requirementTexts are the lowercased texts of the requirements, in order.
type requirementTexts struct {
	ids   []string
	texts []string
}

// TRLC-LINKS: REQ-EMG-193
func lowercasedRequirements(reqs []model.Requirement) requirementTexts {
	var t requirementTexts
	for _, r := range reqs {
		t.ids = append(t.ids, r.ID)
		t.texts = append(t.texts, strings.ToLower(r.Text))
	}
	return t
}

// naming returns the sorted IDs of the requirements whose text contains one of the phrases on word boundaries. The
// list is never nil.
// TRLC-LINKS: REQ-EMG-193
func (t requirementTexts) naming(phrases []string) []string {
	out := []string{}
	for i, text := range t.texts {
		if anyPhraseIn([]string{text}, phrases) {
			out = append(out, t.ids[i])
		}
	}
	sort.Strings(out)
	return out
}

// termPhrases are the lowercase name and aliases of a term, without blanks.
// TRLC-LINKS: REQ-EMG-193
func termPhrases(e model.CatalogEntry) []string {
	var out []string
	for _, p := range append([]string{e.Name}, e.Aliases...) {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

type catalogGroup struct {
	name    string
	entries []model.CatalogEntry
}

// TRLC-LINKS: REQ-EMG-193
func catalogGroups(g model.CatalogGroups) []catalogGroup {
	return []catalogGroup{
		{"systems", g.Systems}, {"functionalGroups", g.FunctionalGroups}, {"functionalUnits", g.FunctionalUnits},
		{"referencedElements", g.ReferencedElements}, {"actors", g.Actors}, {"attackVectors", g.AttackVectors},
		{"events", g.Events}, {"states", g.States}, {"features", g.Features}, {"modes", g.Modes},
		{"conditions", g.Conditions}, {"dataTerms", g.DataTerms},
	}
}
