// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package reqdelta

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/labeth/engineering-model-go/internal/model"
)

// SpecDeltaOperation names the OpenSpec delta section to render.
type SpecDeltaOperation string

const (
	// SpecDeltaAdded renders "## ADDED Requirements" (the default).
	SpecDeltaAdded SpecDeltaOperation = "ADDED"
	// SpecDeltaModified renders "## MODIFIED Requirements".
	SpecDeltaModified SpecDeltaOperation = "MODIFIED"
)

// SpecDeltaOptions selects the requirements to render: IDs (in model order)
// or every requirement whose ID starts with Prefix; at least one is required.
// Capability, when set, adds a "# <capability> spec delta" heading.
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
type SpecDeltaOptions struct {
	IDs        []string
	Prefix     string
	Capability string
	Operation  SpecDeltaOperation
}

// SpecDeltaResult is the rendered OpenSpec spec delta: the markdown, the
// rendered requirement IDs in order and warnings about requirements whose
// output needs attention (no SHALL/MUST, no verificationCriteria).
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
type SpecDeltaResult struct {
	Markdown string
	IDs      []string
	Warnings []string
}

var (
	shallWord = regexp.MustCompile(`(?i)\b(shall|must)\b`)
	// given …, when …, then … (GIVEN optional, commas optional).
	whenThen = regexp.MustCompile(`(?i)^(?:given\s+(.+?),?\s+)?when\s+(.+?),?\s+then\s+(.+)$`)
)

// RenderSpecDelta renders OpenSpec spec delta markdown from model
// requirements: per requirement a "### Requirement:" heading (title, else
// ID), the line "<ID>: <model text>" with shall and must capitalized and
// whitespace collapsed - so it starts with the model text under the
// normalization the engineering-flow consistency check applies - and one
// scenario built from verificationCriteria (WHEN/THEN when the criteria have
// that shape, else WHEN the requirement is verified / THEN the criteria).
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func RenderSpecDelta(requirements []model.Requirement, options SpecDeltaOptions) (SpecDeltaResult, error) {
	selected, err := selectRequirements(requirements, options)
	if err != nil {
		return SpecDeltaResult{}, err
	}
	operation, err := specDeltaOperation(options.Operation)
	if err != nil {
		return SpecDeltaResult{}, err
	}
	renderer := specDeltaRenderer{seen: map[string]bool{}}
	if capability := strings.TrimSpace(options.Capability); capability != "" {
		fmt.Fprintf(&renderer.b, "# %s spec delta\n\n", capability)
	}
	fmt.Fprintf(&renderer.b, "## %s Requirements\n", operation)
	for _, r := range selected {
		renderer.render(r)
	}
	renderer.result.Markdown = renderer.b.String()
	return renderer.result, nil
}

// specDeltaOperation defaults an empty operation to ADDED and refuses an unknown one.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func specDeltaOperation(operation SpecDeltaOperation) (SpecDeltaOperation, error) {
	switch operation {
	case "":
		return SpecDeltaAdded, nil
	case SpecDeltaAdded, SpecDeltaModified:
		return operation, nil
	}
	return "", fmt.Errorf("unsupported spec delta operation %q (want ADDED or MODIFIED)", operation)
}

// specDeltaRenderer accumulates the markdown, the rendered IDs and the warnings of one delta.
type specDeltaRenderer struct {
	b      strings.Builder
	result SpecDeltaResult
	// OpenSpec needs unique headings: a repeated title names its ID.
	seen map[string]bool
}

// render appends the heading, requirement line and scenario of r.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func (g *specDeltaRenderer) render(r model.Requirement) {
	id := strings.TrimSpace(r.ID)
	heading := requirementHeading(r)
	if g.seen[heading] && heading != id {
		heading += " (" + id + ")"
	}
	g.seen[requirementHeading(r)] = true
	text := shallWord.ReplaceAllStringFunc(collapse(r.Text), strings.ToUpper)
	if !strings.Contains(text, "SHALL") && !strings.Contains(text, "MUST") {
		g.result.Warnings = append(g.result.Warnings, fmt.Sprintf("%s has no shall or must; openspec validate --strict rejects the requirement", id))
	}
	fmt.Fprintf(&g.b, "\n### Requirement: %s\n%s: %s\n\n#### Scenario: Verify %s\n", heading, id, text, id)
	g.scenario(id, collapse(r.VerificationCriteria))
	g.result.IDs = append(g.result.IDs, id)
}

// scenario appends the WHEN/THEN lines built from the verification criteria.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func (g *specDeltaRenderer) scenario(id, criteria string) {
	switch m := whenThen.FindStringSubmatch(criteria); {
	case criteria == "":
		g.result.Warnings = append(g.result.Warnings, fmt.Sprintf("%s has no verificationCriteria; the scenario is a placeholder", id))
		fmt.Fprintf(&g.b, "- **WHEN** the requirement is verified\n- **THEN** %s holds\n", id)
	case m != nil:
		if m[1] != "" {
			fmt.Fprintf(&g.b, "- **GIVEN** %s\n", m[1])
		}
		fmt.Fprintf(&g.b, "- **WHEN** %s\n- **THEN** %s\n", m[2], m[3])
	default:
		fmt.Fprintf(&g.b, "- **WHEN** the requirement is verified\n- **THEN** %s\n", lowerFirst(criteria))
	}
}

// RenderSpecDeltaFromModel loads the requirements of the model at manifestPath
// (engmod.yml) and renders them with RenderSpecDelta.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, DO-REQUIREMENTS-DOCUMENT
func RenderSpecDeltaFromModel(manifestPath string, options SpecDeltaOptions) (SpecDeltaResult, error) {
	bundle, err := model.LoadBundle(manifestPath)
	if err != nil {
		return SpecDeltaResult{}, err
	}
	return RenderSpecDelta(bundle.Requirements.Requirements, options)
}

// selectRequirements returns the requirements named by IDs or Prefix in
// model order; an ID the model does not define is an error.
// TRLC-LINKS: REQ-EMG-081
func selectRequirements(requirements []model.Requirement, options SpecDeltaOptions) ([]model.Requirement, error) {
	ids := trimmedSpecIDs(options.IDs)
	prefix := strings.TrimSpace(options.Prefix)
	if len(ids) == 0 && prefix == "" {
		return nil, fmt.Errorf("select requirements with IDs or a prefix")
	}
	out, found := matchSpecRequirements(requirements, ids, prefix)
	if missing := missingSpecIDs(ids, found); len(missing) > 0 {
		return nil, fmt.Errorf("the model defines no requirement %s", strings.Join(missing, ", "))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no requirement ID starts with %q", prefix)
	}
	return out, nil
}

// matchSpecRequirements returns, in model order, the requirements named by ids or starting with prefix, and the
// set of their IDs.
// TRLC-LINKS: REQ-EMG-081
func matchSpecRequirements(requirements []model.Requirement, ids []string, prefix string) ([]model.Requirement, map[string]bool) {
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []model.Requirement
	found := map[string]bool{}
	for _, r := range requirements {
		id := strings.TrimSpace(r.ID)
		if want[id] || (prefix != "" && strings.HasPrefix(id, prefix)) {
			out = append(out, r)
			found[id] = true
		}
	}
	return out, found
}

// trimmedSpecIDs returns the non-empty trimmed IDs in order (duplicates kept).
// TRLC-LINKS: REQ-EMG-081
func trimmedSpecIDs(ids []string) []string {
	var out []string
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return out
}

// missingSpecIDs returns the IDs that were not found, in order.
// TRLC-LINKS: REQ-EMG-081
func missingSpecIDs(ids []string, found map[string]bool) []string {
	var missing []string
	for _, id := range ids {
		if !found[id] {
			missing = append(missing, id)
		}
	}
	return missing
}

// requirementHeading is the title, else the ID.
// TRLC-LINKS: REQ-EMG-081
func requirementHeading(r model.Requirement) string {
	if title := collapse(r.Title); title != "" {
		return title
	}
	return strings.TrimSpace(r.ID)
}

// collapse joins all whitespace runs into single spaces.
// TRLC-LINKS: REQ-EMG-081
func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// lowerFirst lower-cases the first letter unless the first word looks like
// an acronym (its second letter is upper case).
// TRLC-LINKS: REQ-EMG-081
func lowerFirst(s string) string {
	first, size := utf8.DecodeRuneInString(s)
	if size == 0 {
		return s
	}
	if second, _ := utf8.DecodeRuneInString(s[size:]); unicode.IsUpper(second) {
		return s
	}
	return string(unicode.ToLower(first)) + s[size:]
}
