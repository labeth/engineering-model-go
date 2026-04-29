// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

type linkTarget struct {
	Anchor string
	Label  string
}

var modelIDPattern = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9-]+$`)
var inlineLinkPattern = regexp.MustCompile(`<<[^>\n]+>>`)

func buildLinkTargets(ref asciidocReferenceIndex) map[string]linkTarget {
	out := map[string]linkTarget{}
	add := func(token, anchor, label string) {
		token = strings.TrimSpace(token)
		anchor = strings.TrimSpace(anchor)
		label = strings.TrimSpace(label)
		if token == "" || anchor == "" {
			return
		}
		if label == "" {
			label = token
		}
		if _, exists := out[token]; exists {
			return
		}
		out[token] = linkTarget{Anchor: anchor, Label: label}
	}
	addWithVariants := func(token, anchor string) {
		token = strings.TrimSpace(token)
		if token == "" {
			return
		}
		add(token, anchor, token)
		for _, v := range linkTokenVariants(token) {
			add(v, anchor, v)
		}
	}
	// Priority order matters; first match wins.
	for _, e := range ref.Authored {
		target := e.Anchor
		add(e.ID, target, e.ID)
		addWithVariants(e.Name, target)
	}
	for _, e := range ref.Catalog {
		target := e.Anchor
		add(e.ID, target, e.ID)
		// Link alias-style IDs that are phrase-like tokens as well.
		if !modelIDPattern.MatchString(strings.TrimSpace(e.ID)) {
			addWithVariants(e.ID, target)
		}
		addWithVariants(e.Name, target)
		for _, alias := range e.Aliases {
			addWithVariants(alias, target)
		}
	}
	// For inferred entries, only link explicit IDs to avoid prose noise.
	for _, e := range ref.Runtime {
		target := e.Anchor
		add(e.ID, target, e.ID)
	}
	for _, e := range ref.Code {
		target := e.Anchor
		add(e.ID, target, e.ID)
		if strings.Contains(strings.ToLower(strings.TrimSpace(e.Kind)), "source_file") {
			src := strings.TrimSpace(e.Source)
			if idx := strings.Index(src, ":"); idx > 0 {
				src = src[:idx]
			}
			base := filepath.Base(filepath.ToSlash(src))
			ext := filepath.Ext(base)
			module := strings.TrimSuffix(base, ext)
			if strings.TrimSpace(module) != "" {
				add(module, target, module)
			}
		}
	}
	for _, e := range ref.Verification {
		target := e.Anchor
		add(e.ID, target, e.ID)
		addWithVariants(e.Name, target)
	}
	return out
}

func linkifyText(text string, targets map[string]linkTarget) string {
	in := strings.TrimSpace(text)
	if in == "" {
		return text
	}
	textLower := strings.ToLower(text)
	type tokenInfo struct {
		Token string
		Link  linkTarget
	}
	items := make([]tokenInfo, 0, len(targets))
	for t, l := range targets {
		t = strings.TrimSpace(t)
		if len(t) < 4 {
			continue
		}
		items = append(items, tokenInfo{Token: t, Link: l})
	}
	sort.SliceStable(items, func(i, j int) bool { return len(items[i].Token) > len(items[j].Token) })

	type span struct {
		start int
		end   int
		repl  string
	}
	spans := []span{}
	used := make([]bool, len(text))
	for _, loc := range inlineLinkPattern.FindAllStringIndex(text, -1) {
		for i := loc[0]; i < loc[1] && i < len(used); i++ {
			used[i] = true
		}
	}
	isWordBound := func(s string, idx int) bool {
		if idx <= 0 || idx >= len(s) {
			return true
		}
		ch := s[idx]
		return !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-')
	}

	for _, it := range items {
		token := it.Token
		if token == "" {
			continue
		}
		tokenLower := strings.ToLower(token)
		if tokenLower == "" {
			continue
		}
		start := 0
		for {
			pos := strings.Index(textLower[start:], tokenLower)
			if pos < 0 {
				break
			}
			s := start + pos
			e := s + len(token)
			ok := true
			if !(strings.Contains(token, " ") || strings.ContainsAny(token, "/:.") || regexp.MustCompile(`[A-Z]{2,}|-`).MatchString(token)) {
				ok = isWordBound(text, s-1) && isWordBound(text, e)
			}
			if ok {
				for i := s; i < e; i++ {
					if used[i] {
						ok = false
						break
					}
				}
			}
			if ok {
				label := text[s:e]
				if strings.TrimSpace(label) == "" {
					label = it.Link.Label
				}
				link := "<<" + it.Link.Anchor + "," + label + ">>"
				for i := s; i < e; i++ {
					used[i] = true
				}
				spans = append(spans, span{start: s, end: e, repl: link})
			}
			start = e
		}
	}
	if len(spans) == 0 {
		return text
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	var b strings.Builder
	last := 0
	for _, sp := range spans {
		if sp.start < last {
			continue
		}
		b.WriteString(text[last:sp.start])
		b.WriteString(sp.repl)
		last = sp.end
	}
	b.WriteString(text[last:])
	return b.String()
}

func linkTokenVariants(token string) []string {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	set := map[string]bool{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || v == token {
			return
		}
		set[v] = true
	}

	lower := strings.ToLower(token)
	add(lower)
	title := titleWords(lower)
	add(title)
	natural := titleWordsNatural(lower)
	add(natural)
	acronymTitle := titleWordsWithAcronyms(lower)
	add(acronymTitle)

	if strings.Contains(token, " ") {
		plural := pluralizePhrase(lower)
		add(plural)
		add(titleWords(plural))
		add(titleWordsWithAcronyms(plural))
	} else {
		if plural, ok := singleWordPlural(lower); ok {
			add(plural)
			add(titleWords(plural))
			add(titleWordsWithAcronyms(plural))
		}
	}

	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func singleWordPlural(word string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(word)) {
	case "actor":
		return "actors", true
	case "event":
		return "events", true
	case "state":
		return "states", true
	case "feature":
		return "features", true
	case "mode":
		return "modes", true
	case "condition":
		return "conditions", true
	case "requirement":
		return "requirements", true
	case "system":
		return "systems", true
	default:
		return "", false
	}
}

func pluralizePhrase(phrase string) string {
	parts := strings.Fields(strings.TrimSpace(phrase))
	if len(parts) == 0 {
		return phrase
	}
	parts[len(parts)-1] = pluralizeWord(parts[len(parts)-1])
	return strings.Join(parts, " ")
}

func pluralizeWord(word string) string {
	w := strings.TrimSpace(strings.ToLower(word))
	if w == "" {
		return word
	}
	if strings.HasSuffix(w, "s") {
		return w
	}
	if strings.HasSuffix(w, "y") && len(w) > 1 {
		prev := w[len(w)-2]
		if !strings.ContainsRune("aeiou", rune(prev)) {
			return w[:len(w)-1] + "ies"
		}
	}
	if strings.HasSuffix(w, "ch") || strings.HasSuffix(w, "sh") || strings.HasSuffix(w, "x") || strings.HasSuffix(w, "z") {
		return w + "es"
	}
	return w + "s"
}

func titleWords(s string) string {
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) == 0 {
		return s
	}
	for i, p := range parts {
		if p == "" {
			continue
		}
		b := []byte(p)
		if b[0] >= 'a' && b[0] <= 'z' {
			b[0] = b[0] - 'a' + 'A'
		}
		parts[i] = string(b)
	}
	return strings.Join(parts, " ")
}

func titleWordsNatural(s string) string {
	parts := strings.Fields(strings.TrimSpace(strings.ToLower(s)))
	if len(parts) == 0 {
		return s
	}
	minor := map[string]bool{
		"and": true, "or": true, "of": true, "to": true, "in": true, "on": true,
		"for": true, "a": true, "an": true, "the": true, "with": true, "without": true,
	}
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i > 0 && i < len(parts)-1 && minor[p] {
			parts[i] = p
			continue
		}
		b := []byte(p)
		if b[0] >= 'a' && b[0] <= 'z' {
			b[0] = b[0] - 'a' + 'A'
		}
		parts[i] = string(b)
	}
	return strings.Join(parts, " ")
}

func titleWordsWithAcronyms(s string) string {
	parts := strings.Fields(strings.TrimSpace(strings.ToLower(s)))
	if len(parts) == 0 {
		return s
	}
	minor := map[string]bool{
		"and": true, "or": true, "of": true, "to": true, "in": true, "on": true,
		"for": true, "a": true, "an": true, "the": true, "with": true, "without": true,
	}
	acronyms := map[string]bool{
		"api": true, "sdk": true, "iot": true, "ota": true, "id": true, "pr": true,
		"ui": true, "ux": true, "db": true, "sql": true, "http": true, "https": true,
		"tls": true, "jwt": true, "kms": true, "sqs": true, "sns": true, "aws": true,
		"gcp": true, "k8s": true, "mqtt": true,
	}
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i > 0 && i < len(parts)-1 && minor[p] {
			parts[i] = p
			continue
		}
		if acronyms[p] {
			parts[i] = strings.ToUpper(p)
			continue
		}
		b := []byte(p)
		if b[0] >= 'a' && b[0] <= 'z' {
			b[0] = b[0] - 'a' + 'A'
		}
		parts[i] = string(b)
	}
	return strings.Join(parts, " ")
}

func requirementsByUnit(reqs []model.Requirement) map[string]string {
	set := map[string][]string{}
	for _, r := range reqs {
		for _, u := range r.AppliesTo {
			set[u] = append(set[u], r.ID)
		}
	}
	out := map[string]string{}
	for u, ids := range set {
		ids = uniqueSorted(ids)
		out[u] = strings.Join(ids, ", ")
	}
	return out
}

func unitDependencies(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.Type == "depends_on" && m.From == unitID {
			out = append(out, nonEmpty(labels[m.To], m.To))
		}
	}
	out = uniqueSorted(out)
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ", ")
}

func unitConsumers(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.Type == "depends_on" && m.To == unitID {
			out = append(out, nonEmpty(labels[m.From], m.From))
		}
	}
	out = uniqueSorted(out)
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, ", ")
}

func unitOutputs(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.From == unitID && m.Type != "contains" {
			target := nonEmpty(labels[m.To], m.To)
			rel := strings.TrimSpace(m.Type)
			if rel == "" || rel == "depends_on" {
				out = append(out, target)
				continue
			}
			out = append(out, rel+" -> "+target)
		}
	}
	out = uniqueSorted(out)
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "; ")
}

type interfaceDetail struct {
	Name        string
	Description string
}

func renderInterfaceSubchapters(in []interfaceDetail) string {
	if len(in) == 0 {
		return "none"
	}
	sort.SliceStable(in, func(i, j int) bool {
		if in[i].Name != in[j].Name {
			return in[i].Name < in[j].Name
		}
		return in[i].Description < in[j].Description
	})
	var b strings.Builder
	b.WriteString("[cols=\"1,3\",options=\"header\"]\n")
	b.WriteString("|===\n")
	b.WriteString("|Interface |Description\n")
	for _, item := range in {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		desc := strings.TrimSpace(item.Description)
		if desc == "" {
			desc = "No authored interface description."
		}
		b.WriteString("|")
		b.WriteString(name)
		b.WriteString(" |")
		b.WriteString(desc)
		b.WriteString("\n")
	}
	b.WriteString("|===")
	return b.String()
}

func unitInboundInterfacesDetailed(unitID string, mappings []model.Mapping, labels map[string]string) string {
	items := []interfaceDetail{}
	seen := map[string]bool{}
	for _, m := range mappings {
		if m.To != unitID {
			continue
		}
		switch m.Type {
		case "depends_on", "interacts_with":
			source := nonEmpty(labels[m.From], m.From)
			desc := strings.TrimSpace(m.Description)
			if desc == "" {
				if m.Type == "interacts_with" {
					desc = "Interaction from this external actor is modeled without additional authored detail."
				} else {
					desc = "Inbound dependency is modeled without additional authored detail."
				}
			}
			key := source + "|" + desc
			if seen[key] {
				continue
			}
			seen[key] = true
			items = append(items, interfaceDetail{Name: source, Description: desc})
		}
	}
	return renderInterfaceSubchapters(items)
}

func unitOutboundInterfacesDetailed(unitID string, mappings []model.Mapping, labels map[string]string) string {
	items := []interfaceDetail{}
	seen := map[string]bool{}
	for _, m := range mappings {
		if m.From != unitID {
			continue
		}
		switch m.Type {
		case "contains", "targets":
			continue
		}
		target := nonEmpty(labels[m.To], m.To)
		desc := strings.TrimSpace(m.Description)
		rel := strings.TrimSpace(m.Type)
		if desc == "" {
			if rel != "" && rel != "depends_on" {
				desc = rel + " relationship is modeled without additional authored detail."
			} else {
				desc = "Outbound dependency is modeled without additional authored detail."
			}
		}
		key := target + "|" + desc
		if seen[key] {
			continue
		}
		seen[key] = true
		items = append(items, interfaceDetail{Name: target, Description: desc})
	}
	return renderInterfaceSubchapters(items)
}

func unitMessagesConsumed(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.To != unitID {
			continue
		}
		switch m.Type {
		case "depends_on", "interacts_with":
			// Communication view should show message/event intent, not requirements.
			source := nonEmpty(labels[m.From], m.From)
			desc := strings.TrimSpace(m.Description)
			if desc != "" {
				out = append(out, source+": "+desc)
				continue
			}
			if m.Type == "interacts_with" {
				out = append(out, "interaction from "+source)
				continue
			}
			out = append(out, "input from "+source)
		}
	}
	out = uniqueSorted(out)
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, "; ")
}

func unitOwnershipSummary(u model.FunctionalUnit, mappings []model.Mapping, reqByUnit map[string]string, labels map[string]string) string {
	areas := []string{}
	for _, m := range mappings {
		if m.From != u.ID {
			continue
		}
		switch m.Type {
		case "depends_on":
			areas = append(areas, "decision and orchestration flow to "+nonEmpty(labels[m.To], m.To))
		case "interacts_with":
			areas = append(areas, "interaction handling with "+nonEmpty(labels[m.To], m.To))
		}
	}
	areas = uniqueSorted(areas)
	if len(areas) > 2 {
		areas = areas[:2]
	}

	base := "functional responsibility for " + strings.ToLower(nonEmpty(u.Name, u.ID))
	if len(areas) > 0 {
		base = base + "; includes " + strings.Join(areas, " and ")
	}
	// Requirement scope is rendered in the dedicated "Owned Decisions" section.
	// Keep ownership summary focused on responsibility boundaries to avoid duplication.
	_ = reqByUnit
	return base
}

func attackVectorsByTarget(mappings []model.Mapping, labels map[string]string) map[string]string {
	set := map[string][]string{}
	for _, m := range mappings {
		if m.Type == "targets" {
			set[m.To] = append(set[m.To], nonEmpty(labels[m.From], m.From))
		}
	}
	out := map[string]string{}
	for k, v := range set {
		v = uniqueSorted(v)
		out[k] = strings.Join(v, ", ")
	}
	return out
}

func listNamesFG(in []model.FunctionalGroup) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func listNamesFU(in []model.FunctionalUnit) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func listNamesActors(in []model.Actor) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func listNamesVectors(in []model.AttackVector) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func listNamesRefs(in []model.ReferencedElement) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func uniqueSorted(in []string) []string {
	set := map[string]bool{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			set[s] = true
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func buildOwnerEvidence(runtime []inferredRuntimeItem, code []inferredCodeItem) map[string]string {
	rtSet := map[string]map[string]bool{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		if owner == "" || owner == "unresolved" {
			continue
		}
		if rtSet[owner] == nil {
			rtSet[owner] = map[string]bool{}
		}
		rtSet[owner][nonEmpty(strings.TrimSpace(r.Name), strings.TrimSpace(r.Kind))] = true
	}
	codeSet := map[string]map[string]bool{}
	for _, c := range code {
		owner := strings.TrimSpace(c.Owner)
		if owner == "" || owner == "unresolved" {
			continue
		}
		if c.Kind != "source_file" {
			continue
		}
		if codeSet[owner] == nil {
			codeSet[owner] = map[string]bool{}
		}
		codeSet[owner][moduleFromPath(codeItemPath(c))] = true
	}

	out := map[string]string{}
	owners := map[string]bool{}
	for o := range rtSet {
		owners[o] = true
	}
	for o := range codeSet {
		owners[o] = true
	}
	for owner := range owners {
		rt := setToSortedSlice(rtSet[owner])
		cm := setToSortedSlice(codeSet[owner])
		parts := []string{}
		if len(rt) > 0 {
			parts = append(parts, "runtime: "+strings.Join(rt, ", "))
		}
		if len(cm) > 0 {
			parts = append(parts, "code modules: "+strings.Join(cm, ", "))
		}
		if len(parts) > 0 {
			out[owner] = strings.Join(parts, " | ")
		}
	}
	return out
}

func sanitizeSourcePath(in string) string {
	s := filepath.ToSlash(strings.TrimSpace(in))
	if s == "" {
		return s
	}
	for _, marker := range []string{"/examples/", "/infra/", "/src/", "/tmp/"} {
		if idx := strings.Index(strings.ToLower(s), marker); idx >= 0 {
			return s[idx+1:]
		}
	}
	if filepath.IsAbs(s) {
		return filepath.Base(s)
	}
	return s
}
