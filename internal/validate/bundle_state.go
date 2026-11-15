// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// idSets holds, per kind of authored entity, the set of its ids.
type idSets struct {
	groups, units, actors, vectors, references, interfaces, dataObjects map[string]bool
	deploymentTargets, controls, trustBoundaries, events, risks         map[string]bool
	threatScenarios, threatAssumptions, threatOutOfScope                map[string]bool
	threatMitigations, controlVerifications, complianceProfiles         map[string]bool
	documents                                                           map[string]bool
}

// newIDSets returns empty id sets.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func newIDSets() idSets {
	set := func() map[string]bool { return map[string]bool{} }
	return idSets{
		groups: set(), units: set(), actors: set(), vectors: set(), references: set(), interfaces: set(), dataObjects: set(),
		deploymentTargets: set(), controls: set(), trustBoundaries: set(), events: set(), risks: set(),
		threatScenarios: set(), threatAssumptions: set(), threatOutOfScope: set(),
		threatMitigations: set(), controlVerifications: set(), complianceProfiles: set(),
		documents: set(),
	}
}

// bundleValidator is the state of one Bundle run: the diagnostics so far, the
// authored ids with their owners and kinds, and the CODE identifier lookups.
type bundleValidator struct {
	diagSink
	b                 model.Bundle
	idOwner           map[string]string
	kindByID          map[string]string
	ids               idSets
	codeIndex         map[string]int
	ambiguousCodeIDs  map[string]bool
	appliesToTargets  map[string]bool
	dependencyAliases map[string]bool
}

// newBundleValidator returns the validator of a bundle with nothing registered.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func newBundleValidator(b model.Bundle) *bundleValidator {
	return &bundleValidator{
		diagSink:          diagSink{diags: []Diagnostic{}},
		b:                 b,
		idOwner:           map[string]string{},
		kindByID:          map[string]string{},
		ids:               newIDSets(),
		ambiguousCodeIDs:  map[string]bool{},
		appliesToTargets:  map[string]bool{},
		dependencyAliases: map[string]bool{},
	}
}

// diagSink collects the error diagnostics of a validation run.
type diagSink struct {
	diags []Diagnostic
}

// fail records an error diagnostic.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (s *diagSink) fail(code, path, message string) {
	s.diags = append(s.diags, Diagnostic{Code: code, Severity: SeverityError, Message: message, Path: path})
}

// indexedPath is the path of the i-th element of a list.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func indexedPath(prefix string, i int) string {
	return fmt.Sprintf("%s[%d]", prefix, i)
}

// textSpec describes a required text: the code and message when it is blank.
type textSpec struct{ code, message string }

// requireText records the spec's error at path when the value is blank.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) requireText(path, value string, spec textSpec) {
	if strings.TrimSpace(value) == "" {
		v.fail(spec.code, path, spec.message)
	}
}

// enumSpec describes an optional value restricted to a lowercase vocabulary;
// format is the message with a %q for the raw value after the lead arguments.
type enumSpec struct {
	allowed      map[string]bool
	code, format string
	lead         []any
}

// enumValue records the spec's error at path when the value is set and is not
// in the vocabulary after trimming and lowercasing.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) enumValue(path, value string, spec enumSpec) {
	if n := strings.ToLower(strings.TrimSpace(value)); n != "" && !spec.allowed[n] {
		v.fail(spec.code, path, fmt.Sprintf(spec.format, leadArgs(spec.lead, value)...))
	}
}

// leadArgs is the lead arguments of a message followed by the id.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func leadArgs(lead []any, id string) []any {
	return append(append([]any{}, lead...), id)
}

// isoDateValue records the error at path when the value is set and is not a YYYY-MM-DD date.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) isoDateValue(path, value string, spec textSpec) {
	if date := strings.TrimSpace(value); date != "" && !isoDateRe.MatchString(date) {
		v.fail(spec.code, path, fmt.Sprintf(spec.message, value))
	}
}

// optionalRef records the spec's error at path when the trimmed value is set and invalid.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) optionalRef(path, value string, spec refSpec) {
	if id := strings.TrimSpace(value); id != "" && !spec.ok(id) {
		v.fail(spec.code, path, fmt.Sprintf(spec.format, leadArgs(spec.lead, id)...))
	}
}

// refSpec describes a list of references: ok decides whether a trimmed id is
// valid, format is the message (lead arguments, then the id as %q) and field
// the list's name below the owner's path.
type refSpec struct {
	ok           func(string) bool
	code, format string
	field        string
	lead         []any
}

// inSet is the validity test of ids in a set.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func inSet(set map[string]bool) func(string) bool {
	return func(id string) bool { return set[id] }
}

// refList records the spec's error for each blank or invalid reference of the list.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) refList(path string, values []string, spec refSpec) {
	for j, raw := range values {
		id := strings.TrimSpace(raw)
		if id == "" || !spec.ok(id) {
			v.fail(spec.code, indexedPath(path+"."+spec.field, j), fmt.Sprintf(spec.format, leadArgs(spec.lead, id)...))
		}
	}
}

// evidenceSpec describes the evidence list of an entity: the code and message
// of an entry without a path.
type evidenceSpec struct{ code, message string }

// blankPaths records the spec's error for each evidence entry without a path.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) blankPaths(path string, paths []string, spec evidenceSpec) {
	for j, p := range paths {
		if strings.TrimSpace(p) == "" {
			v.fail(spec.code, indexedPath(path+".evidence", j), spec.message)
		}
	}
}

// evidencePathsOf lists the paths of evidence entries.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func evidencePathsOf[T any](items []T, pathOf func(T) string) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = pathOf(it)
	}
	return out
}

// addID registers an id with its owner path, reporting a missing or duplicate id.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) addID(id, owner string) {
	if strings.TrimSpace(id) == "" {
		v.fail("model.missing_id", owner, fmt.Sprintf("missing id for %s", owner))
		return
	}
	if prev, ok := v.idOwner[id]; ok {
		v.fail("model.duplicate_id", owner, fmt.Sprintf("duplicate id %q (%s, %s)", id, prev, owner))
		return
	}
	v.idOwner[id] = owner
}

// idKind names the kind of an entity and the set its ids go into (nil for none).
type idKind struct {
	kind string
	set  map[string]bool
}

// register adds an id of the kind at the path.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) register(k idKind, id, path string) {
	v.addID(id, path)
	if k.set != nil {
		k.set[id] = true
	}
	v.kindByID[id] = k.kind
}

// validID reports whether an id is authored or, for a CODE id, produced exactly
// where the code index says; a CODE id produced more than once is remembered
// as ambiguous.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) validID(id string) bool {
	if _, ok := v.idOwner[id]; ok {
		return true
	}
	if !model.IsCodeID(id) || v.b.CodeIndex == nil {
		return false
	}
	if v.codeIndex == nil {
		v.codeIndex = v.b.CodeIndex()
	}
	if v.codeIndex[id] > 1 {
		v.ambiguousCodeIDs[id] = true
	}
	return v.codeIndex[id] > 0
}

// checkSchemaVersions rejects an unsupported schemaVersion of any document.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkSchemaVersions() {
	for _, sv := range schemaVersions(v.b) {
		if _, err := model.EffectiveSchemaVersion(sv.version); err != nil {
			v.fail("model.unsupported_schema_version", sv.path, schemaVersionProblem(err))
		}
	}
}

// schemaVersionProblem is the message of a schema version error.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func schemaVersionProblem(err error) string {
	var versionErr *model.SchemaVersionError
	if errors.As(err, &versionErr) {
		return versionErr.Problem()
	}
	return err.Error()
}

// checkSubsystems requires a dependency alias and a publication of each composition subsystem.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkSubsystems() {
	for i, subsystem := range v.b.Architecture.Composition.Subsystems {
		path := indexedPath("composition.subsystems", i)
		v.requireText(path+".dependency", subsystem.Dependency, textSpec{"composition.missing_dependency", fmt.Sprintf("subsystem %q has no dependency alias", subsystem.ID)})
		v.requireText(path+".publication", subsystem.Publication, textSpec{"composition.missing_publication", fmt.Sprintf("subsystem %q has no publication", subsystem.ID)})
	}
}

// checkInferenceHints validates the exclude patterns and rejects the removed codeSources.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkInferenceHints() {
	hints := v.b.Architecture.InferenceHints
	v.diags = append(v.diags, excludePatternDiagnostics(hints.Exclude)...)
	if len(hints.CodeSources) > 0 {
		v.fail("model.code_sources_removed", "inferenceHints.codeSources", "inferenceHints.codeSources is no longer supported: the module root is scanned; list paths to leave out under inferenceHints.exclude")
	}
}

// collectAppliesToTargets lists the ids a requirement may apply to besides the
// authored ids: catalog entries, composition subsystems and contract entries
// of this model.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) collectAppliesToTargets() {
	v.addCatalogTargets()
	for _, subsystem := range v.b.Architecture.Composition.Subsystems {
		v.appliesToTargets[strings.TrimSpace(subsystem.ID)] = true
	}
	contract := v.b.Architecture.Contract
	for _, entries := range [][]model.ContractEntry{contract.Provides, contract.Requires} {
		for _, entry := range entries {
			v.appliesToTargets[strings.TrimSpace(entry.ID)] = true
		}
	}
	for _, a := range contract.Assumptions {
		v.appliesToTargets[strings.TrimSpace(a.ID)] = true
	}
}

// addCatalogTargets adds the ids of every catalog entry.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) addCatalogTargets() {
	c := v.b.Catalog.Catalog
	for _, group := range [][]model.CatalogEntry{
		c.Systems, c.FunctionalGroups, c.FunctionalUnits, c.ReferencedElements, c.Actors, c.AttackVectors,
		c.Events, c.States, c.Features, c.Modes, c.Conditions, c.DataTerms,
	} {
		for _, entry := range group {
			v.appliesToTargets[strings.TrimSpace(entry.ID)] = true
		}
	}
}
