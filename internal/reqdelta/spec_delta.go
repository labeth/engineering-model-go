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
	operation := options.Operation
	if operation == "" {
		operation = SpecDeltaAdded
	}
	if operation != SpecDeltaAdded && operation != SpecDeltaModified {
		return SpecDeltaResult{}, fmt.Errorf("unsupported spec delta operation %q (want ADDED or MODIFIED)", operation)
	}
	var b strings.Builder
	var result SpecDeltaResult
	if capability := strings.TrimSpace(options.Capability); capability != "" {
		fmt.Fprintf(&b, "# %s spec delta\n\n", capability)
	}
	fmt.Fprintf(&b, "## %s Requirements\n", operation)
	// OpenSpec needs unique headings: a repeated title names its ID.
	seen := map[string]bool{}
	for _, r := range selected {
		id := strings.TrimSpace(r.ID)
		heading := requirementHeading(r)
		if seen[heading] && heading != id {
			heading += " (" + id + ")"
		}
		seen[requirementHeading(r)] = true
		text := shallWord.ReplaceAllStringFunc(collapse(r.Text), strings.ToUpper)
		if !strings.Contains(text, "SHALL") && !strings.Contains(text, "MUST") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s has no shall or must; openspec validate --strict rejects the requirement", id))
		}
		fmt.Fprintf(&b, "\n### Requirement: %s\n%s: %s\n\n#### Scenario: Verify %s\n", heading, id, text, id)
		criteria := collapse(r.VerificationCriteria)
		switch m := whenThen.FindStringSubmatch(criteria); {
		case criteria == "":
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s has no verificationCriteria; the scenario is a placeholder", id))
			fmt.Fprintf(&b, "- **WHEN** the requirement is verified\n- **THEN** %s holds\n", id)
		case m != nil:
			if m[1] != "" {
				fmt.Fprintf(&b, "- **GIVEN** %s\n", m[1])
			}
			fmt.Fprintf(&b, "- **WHEN** %s\n- **THEN** %s\n", m[2], m[3])
		default:
			fmt.Fprintf(&b, "- **WHEN** the requirement is verified\n- **THEN** %s\n", lowerFirst(criteria))
		}
		result.IDs = append(result.IDs, id)
	}
	result.Markdown = b.String()
	return result, nil
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
	want := map[string]bool{}
	for _, id := range options.IDs {
		if id = strings.TrimSpace(id); id != "" {
			want[id] = true
		}
	}
	prefix := strings.TrimSpace(options.Prefix)
	if len(want) == 0 && prefix == "" {
		return nil, fmt.Errorf("select requirements with IDs or a prefix")
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
	var missing []string
	for _, id := range options.IDs {
		if id = strings.TrimSpace(id); id != "" && !found[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("the model defines no requirement %s", strings.Join(missing, ", "))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no requirement ID starts with %q", prefix)
	}
	return out, nil
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
