// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"cmp"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
)

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type linkTarget struct {
	Anchor string
	Label  string
}

var modelIDPattern = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9-]+$`)
var inlineLinkPattern = regexp.MustCompile(`<<[^>\n]+>>`)

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildLinkTargets(ref asciidocReferenceIndex) map[string]linkTarget {
	out := linkTargetSet{}
	// Priority order matters; first match wins.
	for _, e := range ref.Authored {
		out.add(e.ID, e.Anchor, e.ID)
		out.addWithVariants(e.Name, e.Anchor)
	}
	for _, e := range ref.Catalog {
		out.addCatalogEntry(e)
	}
	// For inferred entries, only link explicit IDs to avoid prose noise.
	for _, e := range ref.Runtime {
		out.add(e.ID, e.Anchor, e.ID)
	}
	for _, e := range ref.Code {
		out.addCodeEntry(e)
	}
	for _, e := range ref.Verification {
		out.add(e.ID, e.Anchor, e.ID)
		out.addWithVariants(e.Name, e.Anchor)
	}
	return out
}

// linkTargetSet maps the tokens of prose to the targets they link to.
type linkTargetSet map[string]linkTarget

// add makes the token link to the anchor, unless the token has a target
// already; a blank label is the token.
// TRLC-LINKS: REQ-EMG-003
func (s linkTargetSet) add(token, anchor, label string) {
	token = strings.TrimSpace(token)
	anchor = strings.TrimSpace(anchor)
	if token == "" || anchor == "" {
		return
	}
	if _, exists := s[token]; !exists {
		s[token] = linkTarget{Anchor: anchor, Label: textutil.NonEmpty(strings.TrimSpace(label), token)}
	}
}

// addWithVariants makes the token and its case and plural variants link to the anchor.
// TRLC-LINKS: REQ-EMG-003
func (s linkTargetSet) addWithVariants(token, anchor string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	s.add(token, anchor, token)
	for _, v := range linkTokenVariants(token) {
		s.add(v, anchor, v)
	}
}

// addCatalogEntry links the ID, the name and the aliases of a catalog entry;
// an ID that is a phrase rather than a model ID links with its variants.
// TRLC-LINKS: REQ-EMG-003
func (s linkTargetSet) addCatalogEntry(e asciidocReferenceEntry) {
	s.add(e.ID, e.Anchor, e.ID)
	// Link alias-style IDs that are phrase-like tokens as well.
	if !modelIDPattern.MatchString(strings.TrimSpace(e.ID)) {
		s.addWithVariants(e.ID, e.Anchor)
	}
	s.addWithVariants(e.Name, e.Anchor)
	for _, alias := range e.Aliases {
		s.addWithVariants(alias, e.Anchor)
	}
}

// addCodeEntry links the ID of a code entry, and the module name of a source file.
// TRLC-LINKS: REQ-EMG-003
func (s linkTargetSet) addCodeEntry(e asciidocReferenceEntry) {
	s.add(e.ID, e.Anchor, e.ID)
	if !strings.Contains(strings.ToLower(strings.TrimSpace(e.Kind)), "source_file") {
		return
	}
	src := strings.TrimSpace(e.Source)
	if idx := strings.Index(src, ":"); idx > 0 {
		src = src[:idx]
	}
	base := filepath.Base(filepath.ToSlash(src))
	module := strings.TrimSuffix(base, filepath.Ext(base))
	if strings.TrimSpace(module) != "" {
		s.add(module, e.Anchor, module)
	}
}

// phraseTokenPattern matches tokens that contain an acronym or a hyphen.
var phraseTokenPattern = regexp.MustCompile(`[A-Z]{2,}|-`)

// wordChars are the characters that continue a word of prose.
const wordChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"

// linkToken is a token of prose and the target it links to.
type linkToken struct {
	Token string
	Link  linkTarget
}

// linkSpan is a stretch of text and the link that replaces it.
type linkSpan struct {
	start int
	end   int
	repl  string
}

// textLinker finds the tokens in a text and the spans to replace with links.
type textLinker struct {
	text  string
	lower string
	used  []bool
	spans []linkSpan
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func linkifyText(text string, targets map[string]linkTarget) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	l := newTextLinker(text)
	for _, it := range sortedLinkTokens(targets) {
		l.linkAll(it)
	}
	return l.render()
}

// sortedLinkTokens lists the tokens of at least four characters, longest
// first so specific phrases win over substrings; the tie-break on the token
// text keeps the result deterministic (the targets are a map).
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func sortedLinkTokens(targets map[string]linkTarget) []linkToken {
	items := make([]linkToken, 0, len(targets))
	for t, l := range targets {
		if t = strings.TrimSpace(t); len(t) >= 4 {
			items = append(items, linkToken{Token: t, Link: l})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return cmp.Or(cmp.Compare(len(items[j].Token), len(items[i].Token)), cmp.Compare(items[i].Token, items[j].Token)) < 0
	})
	return items
}

// newTextLinker marks the existing inline links of the text as used.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func newTextLinker(text string) *textLinker {
	l := &textLinker{text: text, lower: strings.ToLower(text), used: make([]bool, len(text))}
	for _, loc := range inlineLinkPattern.FindAllStringIndex(text, -1) {
		l.markUsed(loc[0], min(loc[1], len(l.used)))
	}
	return l
}

// markUsed marks the characters from start up to end as used.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func (l *textLinker) markUsed(start, end int) {
	for i := start; i < end; i++ {
		l.used[i] = true
	}
}

// isFree reports whether no character from start up to end is used.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func (l *textLinker) isFree(start, end int) bool {
	for i := start; i < end; i++ {
		if l.used[i] {
			return false
		}
	}
	return true
}

// linkAll links each occurrence of the token that is not used yet.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func (l *textLinker) linkAll(it linkToken) {
	tokenLower := strings.ToLower(it.Token)
	if it.Token == "" || tokenLower == "" {
		return
	}
	start := 0
	for {
		pos := strings.Index(l.lower[start:], tokenLower)
		if pos < 0 {
			return
		}
		s := start + pos
		e := s + len(it.Token)
		l.linkAt(it, s, e)
		start = e
	}
}

// linkAt links the token at the position, when it ends at a word boundary and
// the text is free.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func (l *textLinker) linkAt(it linkToken, s, e int) {
	if !l.atBoundary(it.Token, s, e) || !l.isFree(s, e) {
		return
	}
	label := textutil.NonEmpty(l.text[s:e], it.Link.Label)
	l.markUsed(s, e)
	l.spans = append(l.spans, linkSpan{start: s, end: e, repl: "<<" + it.Link.Anchor + "," + label + ">>"})
}

// atBoundary reports whether the occurrence is a whole word; a phrase,
// path or acronym token needs no boundary.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func (l *textLinker) atBoundary(token string, s, e int) bool {
	if isPhraseToken(token) {
		return true
	}
	return isWordBoundary(l.text, s-1) && isWordBoundary(l.text, e)
}

// isPhraseToken reports whether the token has a space, a path separator, an
// acronym or a hyphen.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func isPhraseToken(token string) bool {
	return strings.Contains(token, " ") || strings.ContainsAny(token, "/:.") || phraseTokenPattern.MatchString(token)
}

// isWordBoundary reports whether the character at idx does not continue a word.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func isWordBoundary(s string, idx int) bool {
	if idx <= 0 || idx >= len(s) {
		return true
	}
	return strings.IndexByte(wordChars, s[idx]) < 0
}

// render is the text with the spans replaced, the earliest of overlapping spans winning.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-012
func (l *textLinker) render() string {
	if len(l.spans) == 0 {
		return l.text
	}
	sort.SliceStable(l.spans, func(i, j int) bool { return l.spans[i].start < l.spans[j].start })
	var b strings.Builder
	last := 0
	for _, sp := range l.spans {
		if sp.start < last {
			continue
		}
		b.WriteString(l.text[last:sp.start])
		b.WriteString(sp.repl)
		last = sp.end
	}
	b.WriteString(l.text[last:])
	return b.String()
}

// TRLC-LINKS: REQ-EMG-003
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

// singleWordPlurals maps the model words that have a plural link variant to it.
var singleWordPlurals = map[string]string{
	"actor": "actors", "event": "events", "state": "states", "feature": "features", "mode": "modes",
	"condition": "conditions", "requirement": "requirements", "system": "systems",
}

// TRLC-LINKS: REQ-EMG-003
func singleWordPlural(word string) (string, bool) {
	plural, ok := singleWordPlurals[strings.TrimSpace(strings.ToLower(word))]
	return plural, ok
}

// TRLC-LINKS: REQ-EMG-003
func pluralizePhrase(phrase string) string {
	parts := strings.Fields(strings.TrimSpace(phrase))
	if len(parts) == 0 {
		return phrase
	}
	parts[len(parts)-1] = pluralizeWord(parts[len(parts)-1])
	return strings.Join(parts, " ")
}

// TRLC-LINKS: REQ-EMG-003
func pluralizeWord(word string) string {
	w := strings.TrimSpace(strings.ToLower(word))
	switch {
	case w == "":
		return word
	case strings.HasSuffix(w, "s"):
		return w
	case endsInConsonantY(w):
		return w[:len(w)-1] + "ies"
	case hasAnySuffix(w, "ch", "sh", "x", "z"):
		return w + "es"
	}
	return w + "s"
}

// endsInConsonantY reports whether the word ends in a y that follows a consonant.
// TRLC-LINKS: REQ-EMG-003
func endsInConsonantY(w string) bool {
	return len(w) > 1 && strings.HasSuffix(w, "y") && !strings.ContainsRune("aeiou", rune(w[len(w)-2]))
}

// hasAnySuffix reports whether s ends with one of the suffixes.
// TRLC-LINKS: REQ-EMG-003
func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// capitalizeASCII upper-cases the first character when it is a lower-case ASCII letter.
// TRLC-LINKS: REQ-EMG-003
func capitalizeASCII(p string) string {
	if p != "" && p[0] >= 'a' && p[0] <= 'z' {
		return string(p[0]-'a'+'A') + p[1:]
	}
	return p
}

// TRLC-LINKS: REQ-EMG-003
func titleWords(s string) string {
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) == 0 {
		return s
	}
	for i, p := range parts {
		parts[i] = capitalizeASCII(p)
	}
	return strings.Join(parts, " ")
}

// minorTitleWords stay lower-case inside a title.
var minorTitleWords = map[string]bool{
	"and": true, "or": true, "of": true, "to": true, "in": true, "on": true,
	"for": true, "a": true, "an": true, "the": true, "with": true, "without": true,
}

// titleAcronyms are upper-cased in a title.
var titleAcronyms = map[string]bool{
	"api": true, "sdk": true, "iot": true, "ota": true, "id": true, "pr": true,
	"ui": true, "ux": true, "db": true, "sql": true, "http": true, "https": true,
	"tls": true, "jwt": true, "kms": true, "sqs": true, "sns": true, "aws": true,
	"gcp": true, "k8s": true, "mqtt": true,
}

// TRLC-LINKS: REQ-EMG-003
func titleWordsNatural(s string) string {
	return titleCase(s, nil)
}

// TRLC-LINKS: REQ-EMG-003
func titleWordsWithAcronyms(s string) string {
	return titleCase(s, titleAcronyms)
}

// titleCase capitalizes each word but the minor words inside the title; a word
// in acronyms is upper-cased.
// TRLC-LINKS: REQ-EMG-003
func titleCase(s string, acronyms map[string]bool) string {
	parts := strings.Fields(strings.TrimSpace(strings.ToLower(s)))
	if len(parts) == 0 {
		return s
	}
	for i, p := range parts {
		parts[i] = titleCaseWord(p, i > 0 && i < len(parts)-1, acronyms)
	}
	return strings.Join(parts, " ")
}

// titleCaseWord is the word as it appears in a title: a minor word inside the
// title stays as is, an acronym is upper-cased, any other is capitalized.
// TRLC-LINKS: REQ-EMG-003
func titleCaseWord(p string, inside bool, acronyms map[string]bool) string {
	switch {
	case inside && minorTitleWords[p]:
		return p
	case acronyms[p]:
		return strings.ToUpper(p)
	}
	return capitalizeASCII(p)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func requirementsByUnit(reqs []model.Requirement) map[string]string {
	set := map[string][]string{}
	for _, r := range reqs {
		for _, u := range r.AppliesTo {
			set[u] = append(set[u], r.ID)
		}
	}
	out := map[string]string{}
	for u, ids := range set {
		ids = textutil.UniqueSorted(ids)
		out[u] = strings.Join(ids, ", ")
	}
	return out
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func unitDependencies(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.Type == "depends_on" && m.From == unitID {
			out = append(out, textutil.NonEmpty(labels[m.To], m.To))
		}
	}
	out = textutil.UniqueSorted(out)
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ", ")
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func unitConsumers(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.Type == "depends_on" && m.To == unitID {
			out = append(out, textutil.NonEmpty(labels[m.From], m.From))
		}
	}
	out = textutil.UniqueSorted(out)
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, ", ")
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func unitOutputs(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.From == unitID && m.Type != "contains" {
			out = append(out, unitOutputEntry(m, labels))
		}
	}
	out = textutil.UniqueSorted(out)
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "; ")
}

// unitOutputEntry is the target of a mapping, with its relation unless that is
// a plain dependency.
// TRLC-LINKS: REQ-EMG-003
func unitOutputEntry(m model.Mapping, labels map[string]string) string {
	target := textutil.NonEmpty(labels[m.To], m.To)
	rel := strings.TrimSpace(m.Type)
	if rel == "" || rel == "depends_on" {
		return target
	}
	return rel + " -> " + target
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
type interfaceDetail struct {
	Name        string
	Description string
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
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

// interfaceDetails collects interface details, each name and description once.
type interfaceDetails struct {
	items []interfaceDetail
	seen  map[string]bool
}

// add records the detail unless the name and description were recorded.
// TRLC-LINKS: REQ-EMG-003
func (d *interfaceDetails) add(name, desc string) {
	key := name + "|" + desc
	if d.seen[key] {
		return
	}
	d.seen[key] = true
	d.items = append(d.items, interfaceDetail{Name: name, Description: desc})
}

// isInboundMapping reports whether a mapping is a dependency or an interaction.
// TRLC-LINKS: REQ-EMG-003
func isInboundMapping(m model.Mapping) bool {
	return m.Type == "depends_on" || m.Type == "interacts_with"
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func unitInboundInterfacesDetailed(unitID string, mappings []model.Mapping, labels map[string]string) string {
	details := &interfaceDetails{seen: map[string]bool{}}
	for _, m := range mappings {
		if m.To == unitID && isInboundMapping(m) {
			details.add(textutil.NonEmpty(labels[m.From], m.From), inboundDescription(m))
		}
	}
	return renderInterfaceSubchapters(details.items)
}

// inboundDescription is the authored description of an inbound mapping, or a
// sentence saying that none was authored.
// TRLC-LINKS: REQ-EMG-003
func inboundDescription(m model.Mapping) string {
	if desc := strings.TrimSpace(m.Description); desc != "" {
		return desc
	}
	if m.Type == "interacts_with" {
		return "Interaction from this external actor is modeled without additional authored detail."
	}
	return "Inbound dependency is modeled without additional authored detail."
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func unitOutboundInterfacesDetailed(unitID string, mappings []model.Mapping, labels map[string]string) string {
	details := &interfaceDetails{seen: map[string]bool{}}
	for _, m := range mappings {
		if m.From == unitID && m.Type != "contains" && m.Type != "targets" {
			details.add(textutil.NonEmpty(labels[m.To], m.To), outboundDescription(m))
		}
	}
	return renderInterfaceSubchapters(details.items)
}

// outboundDescription is the authored description of an outbound mapping, or a
// sentence saying that none was authored.
// TRLC-LINKS: REQ-EMG-003
func outboundDescription(m model.Mapping) string {
	if desc := strings.TrimSpace(m.Description); desc != "" {
		return desc
	}
	if rel := strings.TrimSpace(m.Type); rel != "" && rel != "depends_on" {
		return rel + " relationship is modeled without additional authored detail."
	}
	return "Outbound dependency is modeled without additional authored detail."
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func unitMessagesConsumed(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.To == unitID && isInboundMapping(m) {
			// Communication view should show message/event intent, not requirements.
			out = append(out, messageConsumed(m, textutil.NonEmpty(labels[m.From], m.From)))
		}
	}
	out = textutil.UniqueSorted(out)
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, "; ")
}

// messageConsumed is the message a mapping delivers from the source: its
// authored description, or the kind of input.
// TRLC-LINKS: REQ-EMG-003
func messageConsumed(m model.Mapping, source string) string {
	if desc := strings.TrimSpace(m.Description); desc != "" {
		return source + ": " + desc
	}
	if m.Type == "interacts_with" {
		return "interaction from " + source
	}
	return "input from " + source
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func unitOwnershipSummary(u model.FunctionalUnit, mappings []model.Mapping, labels map[string]string) string {
	areas := []string{}
	for _, m := range mappings {
		if m.From == u.ID {
			areas = append(areas, ownershipArea(m, labels)...)
		}
	}
	areas = textutil.UniqueSorted(areas)
	if len(areas) > 2 {
		areas = areas[:2]
	}
	base := "functional responsibility for " + strings.ToLower(textutil.NonEmpty(u.Name, u.ID))
	if len(areas) > 0 {
		base = base + "; includes " + strings.Join(areas, " and ")
	}
	// Requirement scope is rendered in the dedicated "Owned Decisions" section.
	// Keep ownership summary focused on responsibility boundaries to avoid duplication.
	return base
}

// ownershipArea is the responsibility area a dependency or an interaction of
// the unit stands for.
// TRLC-LINKS: REQ-EMG-003
func ownershipArea(m model.Mapping, labels map[string]string) []string {
	switch m.Type {
	case "depends_on":
		return []string{"decision and orchestration flow to " + textutil.NonEmpty(labels[m.To], m.To)}
	case "interacts_with":
		return []string{"interaction handling with " + textutil.NonEmpty(labels[m.To], m.To)}
	}
	return nil
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
// TRLC-LINKS: REQ-EMG-003
func attackVectorsByTarget(mappings []model.Mapping, labels map[string]string) map[string]string {
	set := map[string][]string{}
	for _, m := range mappings {
		if m.Type == "targets" {
			set[m.To] = append(set[m.To], textutil.NonEmpty(labels[m.From], m.From))
		}
	}
	out := map[string]string{}
	for k, v := range set {
		v = textutil.UniqueSorted(v)
		out[k] = strings.Join(v, ", ")
	}
	return out
}

// TRLC-LINKS: REQ-EMG-003
func listNamesFG(in []model.FunctionalGroup) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, textutil.NonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// TRLC-LINKS: REQ-EMG-003
func listNamesFU(in []model.FunctionalUnit) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, textutil.NonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// TRLC-LINKS: REQ-EMG-003
func listNamesActors(in []model.Actor) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, textutil.NonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// TRLC-LINKS: REQ-EMG-003
func listNamesVectors(in []model.AttackVector) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, textutil.NonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// TRLC-LINKS: REQ-EMG-003
func listNamesRefs(in []model.ReferencedElement) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, textutil.NonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildOwnerEvidence(runtime []codescan.RuntimeItem, code []codescan.CodeItem) map[string]string {
	rtSet := runtimeNameSetByOwner(runtime)
	codeSet := codeModuleSetByOwner(code)
	out := map[string]string{}
	for owner := range unionSets(setKeys(rtSet), setKeys(codeSet)) {
		parts := []string{}
		if rt := setToSortedSlice(rtSet[owner]); len(rt) > 0 {
			parts = append(parts, "runtime: "+strings.Join(rt, ", "))
		}
		if cm := setToSortedSlice(codeSet[owner]); len(cm) > 0 {
			parts = append(parts, "code modules: "+strings.Join(cm, ", "))
		}
		if len(parts) > 0 {
			out[owner] = strings.Join(parts, "; ")
		}
	}
	return out
}

// setKeys is the keys of a map of sets, as a set.
// TRLC-LINKS: REQ-EMG-003
func setKeys(m map[string]map[string]bool) map[string]bool {
	keys := map[string]bool{}
	for k := range m {
		keys[k] = true
	}
	return keys
}

// runtimeNameSetByOwner is the names of the runtime elements of each resolved owner.
// TRLC-LINKS: REQ-EMG-003
func runtimeNameSetByOwner(runtime []codescan.RuntimeItem) map[string]map[string]bool {
	byOwner := map[string]map[string]bool{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		if owner != "" && owner != "unresolved" {
			addToSetMap(byOwner, owner, textutil.NonEmpty(strings.TrimSpace(r.Name), strings.TrimSpace(r.Kind)))
		}
	}
	return byOwner
}

// codeModuleSetByOwner is the modules of the source files of each resolved
// owner, other than verification code.
// TRLC-LINKS: REQ-EMG-003
func codeModuleSetByOwner(code []codescan.CodeItem) map[string]map[string]bool {
	byOwner := map[string]map[string]bool{}
	for _, c := range code {
		owner := strings.TrimSpace(c.Owner)
		if c.Kind != "source_file" || codescan.IsVerificationCodeItem(c) {
			continue
		}
		if owner != "" && owner != "unresolved" {
			addToSetMap(byOwner, owner, moduleFromPath(codescan.ItemPath(c)))
		}
	}
	return byOwner
}

// linkifier turns the model terms in the text of a section into links.
type linkifier struct {
	targets map[string]linkTarget
}

// all links each of the strings in place.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) all(fields ...*string) {
	for _, f := range fields {
		*f = linkifyText(*f, l.targets)
	}
}

// linkEach applies link to every item of a slice, in place.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func linkEach[T any](items []T, link func(*T)) {
	for i := range items {
		link(&items[i])
	}
}

// linkifySections links the model terms in every section of the document, in place.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (g *generation) linkifySections() {
	l := linkifier{targets: g.linkTargets}
	linkEach(g.views, l.view)
	linkEach(g.reqSections, l.requirement)
	linkEach(g.verification, l.verification)
	linkEach(g.verResults, l.verificationResult)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) view(v *asciidocViewSection) {
	linkEach(v.Groups, l.group)
	linkEach(v.Units, l.unit)
	linkEach(v.SecurityAttackChapters, l.attackChapter)
	l.securityRows(v)
	l.all(&v.Inf)
	l.each(v.ViewQuestions)
	l.each(v.CoverageGaps)
	l.each(v.NextActions)
	linkEach(v.ProjectedNodes, l.projectedNode)
	linkEach(v.ProjectedMappings, l.projectedMapping)
	l.all(&v.CoverageSummary)
}

// securityRows links the threat model rows of a security view.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) securityRows(v *asciidocViewSection) {
	linkEach(v.SecurityThreatScenarios, l.threatScenario)
	linkEach(v.SecurityThreatAssumptions, l.threatAssumption)
	linkEach(v.SecurityThreatOutOfScope, l.threatOutOfScope)
	linkEach(v.SecurityThreatMitigations, l.threatMitigation)
	linkEach(v.SecurityControlChecks, l.controlVerification)
	linkEach(v.SecurityFlowRows, l.securityFlow)
}

// each links every string of a slice.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) each(items []string) {
	for j := range items {
		l.all(&items[j])
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) details(details []asciidocDesignDetail) {
	for k := range details {
		l.all(&details[k].Narrative)
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) group(g *asciidocEntitySection) {
	l.all(&g.Description, &g.Intro)
	l.details(g.Details)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) unit(u *asciidocUnitSection) {
	l.all(&u.Intro, &u.WhatOwns, &u.Triggers, &u.Consumes, &u.Produces, &u.DependsOn, &u.Threats, &u.Evidence)
	l.details(u.Details)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) attackChapter(c *asciidocSecurityAttackChapter) {
	l.all(&c.Name, &c.Description)
	linkEach(c.Units, l.unit)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) threatScenario(s *asciidocThreatScenarioRow) {
	l.all(&s.ID, &s.Title, &s.Summary, &s.AttackVector, &s.Scope, &s.Flows, &s.Likelihood, &s.Impact,
		&s.Severity, &s.Status, &s.Owner, &s.Risk, &s.Controls, &s.Mitigations, &s.Verifications)
	linkEach(s.ControlVerifications, l.controlVerification)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) threatAssumption(a *asciidocThreatAssumptionRow) {
	l.all(&a.ID, &a.Title, &a.Status, &a.Owner, &a.AppliesTo, &a.Rationale)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) threatOutOfScope(o *asciidocThreatOutOfScopeRow) {
	l.all(&o.ID, &o.Title, &o.Status, &o.Owner, &o.AppliesTo, &o.ExpiresOn, &o.Reason)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) threatMitigation(m *asciidocThreatMitigationRow) {
	l.all(&m.ID, &m.ThreatScenario, &m.Control, &m.Status, &m.Effectiveness, &m.Owner, &m.Verifications, &m.Notes)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) controlVerification(c *asciidocControlVerificationRow) {
	l.all(&c.ID, &c.Control, &c.ThreatScenarios, &c.Risks, &c.Method, &c.Status, &c.Owner, &c.LastTested, &c.Findings)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) securityFlow(f *asciidocSecurityFlowRow) {
	l.all(&f.ID, &f.Title, &f.Kind, &f.Direction, &f.Frequency, &f.Source, &f.Destination, &f.Protocol,
		&f.Authentication, &f.Encryption, &f.Integrity, &f.TrustBoundary, &f.BoundaryCrossing, &f.Threats, &f.Data)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) projectedNode(n *asciidocProjectedNode) {
	l.all(&n.ID, &n.Kind, &n.Label)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) projectedMapping(m *asciidocMappingSection) {
	l.all(&m.Type, &m.From, &m.To, &m.Description)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) requirement(r *asciidocRequirementSection) {
	l.all(&r.Text, &r.Notes)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) verification(v *asciidocVerificationSection) {
	l.all(&v.Kind, &v.Status, &v.Verifies, &v.TestCode, &v.DerivedOwners, &v.Evidence, &v.ResultSummary, &v.Description)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (l linkifier) verificationResult(r *asciidocVerificationResultRow) {
	l.all(&r.CheckID, &r.CheckName, &r.Requirement, &r.Status, &r.Evidence, &r.Notes)
}
