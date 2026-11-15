// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package reqdelta

import (
	"bytes"
	"fmt"
	"github.com/labeth/engineering-model-go/internal/reqlint"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/gofrs/flock"
	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/trace"
	"github.com/labeth/engineering-model-go/internal/validate"
	"gopkg.in/yaml.v3"
)

const requirementsDeltaVersion = 1

// RequirementsDelta describes stable-ID operations against requirements.yml.
// Updates replace the complete requirement entity.
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
type RequirementsDelta struct {
	Version      int                         `yaml:"version"`
	Requirements RequirementsDeltaOperations `yaml:"requirements"`
}

// RequirementsDeltaOperations groups supported requirement mutations.
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA
type RequirementsDeltaOperations struct {
	Add    []model.Requirement `yaml:"add"`
	Update []model.Requirement `yaml:"update"`
	Remove []string            `yaml:"remove"`
}

// RequirementsDeltaResult is the validated candidate and deterministic change summary.
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-VALIDATION-ENGINE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
// ENGMODEL-LINKS: STATE-DELTA-PLANNED
type RequirementsDeltaResult struct {
	Candidate   model.RequirementsDocument
	YAML        []byte
	Diff        string
	Added       []string
	Updated     []string
	Removed     []string
	Diagnostics []validate.Diagnostic
}

type requirementsDeltaPlan struct {
	result     RequirementsDeltaResult
	targetPath string
	source     []byte
	mode       os.FileMode
}

// PlanRequirementsDelta loads a delta, applies it in memory, and validates the complete candidate.
// It never modifies requirements.yml.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
// ENGMODEL-LINKS: EVT-REQUIREMENTS-DELTA-RECEIVED
func PlanRequirementsDelta(root, deltaPath string) (RequirementsDeltaResult, error) {
	targetPath, err := requirementsPath(root)
	if err != nil {
		return RequirementsDeltaResult{}, err
	}
	plan, err := planRequirementsDelta(targetPath, deltaPath)
	if err != nil {
		return RequirementsDeltaResult{}, err
	}
	return plan.result, nil
}

// ApplyRequirementsDelta validates and atomically writes a requirements delta.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033, REQ-EMG-185
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT, DEP-LOCAL-WORKSPACE
// ENGMODEL-LINKS: EVT-REQUIREMENTS-DELTA-RECEIVED, EVT-REQUIREMENTS-DELTA-APPLY-REQUESTED, TR-DELTA-APPLY, TR-DELTA-REJECT, STATE-DELTA-APPLIED
// ENGMODEL-LINKS: STATE-DELTA-REFUSED, TR-DELTA-CANDIDATE-INVALIDATED, TR-DELTA-SOURCE-CHANGES, TR-DELTA-REFUSE-INVALID, TR-DELTA-REFUSE-CHANGED
func ApplyRequirementsDelta(root, deltaPath string) (result RequirementsDeltaResult, err error) {
	targetPath, err := requirementsPath(root)
	if err != nil {
		return RequirementsDeltaResult{}, err
	}
	release, err := acquireRequirementsLock(targetPath)
	if err != nil {
		return RequirementsDeltaResult{}, err
	}
	defer releaseRequirementsLock(release, &err)

	plan, err := planRequirementsDelta(targetPath, deltaPath)
	if err != nil {
		return RequirementsDeltaResult{}, err
	}
	if err := verifyRequirementsPlanApplicable(plan); err != nil {
		return plan.result, err
	}
	if err := writeFileAtomic(plan.targetPath, plan.result.YAML, plan.mode); err != nil {
		return plan.result, err
	}
	return plan.result, nil
}

// releaseRequirementsLock releases the apply lock and reports a failed release when no earlier error is set.
// TRLC-LINKS: REQ-EMG-033, REQ-EMG-185
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DEP-LOCAL-WORKSPACE
func releaseRequirementsLock(release func() error, err *error) {
	if releaseErr := release(); *err == nil && releaseErr != nil {
		*err = fmt.Errorf("requirements applied but lock release failed: %w", releaseErr)
	}
}

// verifyRequirementsPlanApplicable refuses an invalid candidate and a requirements file that changed since planning.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033, REQ-EMG-185
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-VALIDATION-ENGINE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
// ENGMODEL-LINKS: STATE-DELTA-REFUSED, TR-DELTA-CANDIDATE-INVALIDATED, TR-DELTA-SOURCE-CHANGES, TR-DELTA-REFUSE-INVALID, TR-DELTA-REFUSE-CHANGED
func verifyRequirementsPlanApplicable(plan requirementsDeltaPlan) error {
	if validate.HasErrors(plan.result.Diagnostics) {
		return fmt.Errorf("requirements delta candidate is invalid")
	}
	current, err := os.ReadFile(plan.targetPath)
	if err != nil {
		return fmt.Errorf("re-read requirements before apply: %w", err)
	}
	if !bytes.Equal(current, plan.source) {
		return fmt.Errorf("requirements changed while delta was being planned")
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, FU-VALIDATION-ENGINE
// ENGMODEL-LINKS: STATE-DELTA-PLANNING, TR-DELTA-RECEIVE, TR-DELTA-REJECT, TR-DELTA-VALIDATED, EVT-REQUIREMENTS-CANDIDATE-VALIDATED
// ENGMODEL-LINKS: TR-DELTA-RECEIVE-AFTER-REFUSAL
func planRequirementsDelta(targetPath, deltaPath string) (requirementsDeltaPlan, error) {
	rootPath := filepath.Dir(targetPath)
	current, err := loadRequirementsSource(targetPath)
	if err != nil {
		return requirementsDeltaPlan{}, err
	}
	delta, err := loadRequirementsDelta(deltaPath)
	if err != nil {
		return requirementsDeltaPlan{}, err
	}
	candidateYAML, changes, err := mergeRequirementsYAML(current.source, current.document, delta)
	if err != nil {
		return requirementsDeltaPlan{}, err
	}
	var candidate model.RequirementsDocument
	if err := decodeStrictYAML(candidateYAML, &candidate); err != nil {
		return requirementsDeltaPlan{}, fmt.Errorf("decode merged requirements: %w", err)
	}
	bundle, err := codescan.LoadBundle(filepath.Join(filepath.Dir(rootPath), "engmod.yml"))
	if err != nil {
		return requirementsDeltaPlan{}, fmt.Errorf("load model bundle: %w", err)
	}
	result := RequirementsDeltaResult{
		Candidate:   candidate,
		YAML:        candidateYAML,
		Diff:        renderRequirementsDiff(changes),
		Added:       sortedRequirementKeys(changes.added),
		Updated:     sortedRequirementKeys(changes.updated),
		Removed:     sortedRequirementKeys(changes.removed),
		Diagnostics: validateRequirementsCandidate(bundle, candidate),
	}
	return requirementsDeltaPlan{result: result, targetPath: targetPath, source: current.source, mode: current.mode}, nil
}

// requirementsSource is the requirements file as read: its bytes, mode and decoded document.
type requirementsSource struct {
	source   []byte
	mode     os.FileMode
	document model.RequirementsDocument
}

// loadRequirementsSource reads and strictly decodes the requirements file.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, FU-VALIDATION-ENGINE
// ENGMODEL-LINKS: STATE-DELTA-PLANNING, TR-DELTA-RECEIVE, TR-DELTA-REJECT, TR-DELTA-VALIDATED, EVT-REQUIREMENTS-CANDIDATE-VALIDATED
// ENGMODEL-LINKS: TR-DELTA-RECEIVE-AFTER-REFUSAL
func loadRequirementsSource(targetPath string) (requirementsSource, error) {
	source, err := os.ReadFile(targetPath)
	if err != nil {
		return requirementsSource{}, fmt.Errorf("read requirements: %w", err)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return requirementsSource{}, fmt.Errorf("stat requirements: %w", err)
	}
	var document model.RequirementsDocument
	if err := decodeStrictYAML(source, &document); err != nil {
		return requirementsSource{}, fmt.Errorf("decode requirements: %w", err)
	}
	return requirementsSource{source: source, mode: info.Mode(), document: document}, nil
}

// TRLC-LINKS: REQ-EMG-031
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA
// ENGMODEL-LINKS: EVT-REQUIREMENTS-DELTA-RECEIVED
func loadRequirementsDelta(path string) (RequirementsDelta, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return RequirementsDelta{}, fmt.Errorf("read requirements delta: %w", err)
	}
	var delta RequirementsDelta
	if err := decodeStrictYAML(content, &delta); err != nil {
		return RequirementsDelta{}, fmt.Errorf("decode requirements delta: %w", err)
	}
	if delta.Version != requirementsDeltaVersion {
		return RequirementsDelta{}, fmt.Errorf("unsupported requirements delta version %d, expected %d", delta.Version, requirementsDeltaVersion)
	}
	return delta, nil
}

type requirementChanges struct {
	added   map[string]model.Requirement
	updated map[string]requirementUpdate
	removed map[string]model.Requirement
}

type requirementUpdate struct {
	before model.Requirement
	after  model.Requirement
}

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func mergeRequirementsYAML(source []byte, current model.RequirementsDocument, delta RequirementsDelta) ([]byte, requirementChanges, error) {
	operations, err := validateRequirementsDeltaOperations(delta.Requirements)
	if err != nil {
		return nil, requirementChanges{}, err
	}
	currentByID, err := indexCurrentRequirements(current)
	if err != nil {
		return nil, requirementChanges{}, err
	}
	changes, err := planRequirementChanges(operations, currentByID)
	if err != nil {
		return nil, requirementChanges{}, err
	}
	document, sequence, err := parseRequirementsTree(source)
	if err != nil {
		return nil, requirementChanges{}, err
	}
	merged, err := mergeRequirementItems(sequence, operations, delta.Requirements.Add)
	if err != nil {
		return nil, requirementChanges{}, err
	}
	sequence.Content = merged
	output, err := encodeRequirementsDocument(document)
	if err != nil {
		return nil, requirementChanges{}, err
	}
	return output, changes, nil
}

// parseRequirementsTree parses the requirements YAML into a node tree and returns it with its requirements sequence.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func parseRequirementsTree(source []byte) (*yaml.Node, *yaml.Node, error) {
	var document yaml.Node
	if err := decodeStrictYAML(source, &document); err != nil {
		return nil, nil, fmt.Errorf("parse requirements YAML tree: %w", err)
	}
	sequence, err := requirementsSequence(&document)
	if err != nil {
		return nil, nil, err
	}
	return &document, sequence, nil
}

// indexCurrentRequirements maps the canonical requirements by trimmed id and refuses empty and duplicate ids.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func indexCurrentRequirements(current model.RequirementsDocument) (map[string]model.Requirement, error) {
	currentByID := make(map[string]model.Requirement, len(current.Requirements))
	for _, requirement := range current.Requirements {
		id := strings.TrimSpace(requirement.ID)
		if id == "" {
			return nil, fmt.Errorf("canonical requirements contain an empty id")
		}
		if _, exists := currentByID[id]; exists {
			return nil, fmt.Errorf("canonical requirements contain duplicate id %q", id)
		}
		currentByID[id] = requirement
	}
	return currentByID, nil
}

// planRequirementChanges checks the operations against the current requirements and records the changes.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func planRequirementChanges(operations validatedRequirementOperations, currentByID map[string]model.Requirement) (requirementChanges, error) {
	changes := requirementChanges{
		added:   map[string]model.Requirement{},
		updated: map[string]requirementUpdate{},
		removed: map[string]model.Requirement{},
	}
	if err := planAddedRequirements(changes.added, operations.add, currentByID); err != nil {
		return requirementChanges{}, err
	}
	if err := planUpdatedRequirements(changes.updated, operations.update, currentByID); err != nil {
		return requirementChanges{}, err
	}
	if err := planRemovedRequirements(changes.removed, operations.remove, currentByID); err != nil {
		return requirementChanges{}, err
	}
	return changes, nil
}

// planAddedRequirements records each added requirement and refuses one that already exists.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func planAddedRequirements(into, add, currentByID map[string]model.Requirement) error {
	for id, requirement := range add {
		if _, exists := currentByID[id]; exists {
			return fmt.Errorf("add requirement %q already exists", id)
		}
		into[id] = requirement
	}
	return nil
}

// planUpdatedRequirements records each update with the requirement it replaces and refuses an unknown one.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func planUpdatedRequirements(into map[string]requirementUpdate, update, currentByID map[string]model.Requirement) error {
	for id, requirement := range update {
		before, exists := currentByID[id]
		if !exists {
			return fmt.Errorf("update requirement %q does not exist", id)
		}
		into[id] = requirementUpdate{before: before, after: requirement}
	}
	return nil
}

// planRemovedRequirements records each removed requirement and refuses an unknown one.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func planRemovedRequirements(into map[string]model.Requirement, remove map[string]bool, currentByID map[string]model.Requirement) error {
	for id := range remove {
		before, exists := currentByID[id]
		if !exists {
			return fmt.Errorf("remove requirement %q does not exist", id)
		}
		into[id] = before
	}
	return nil
}

// mergeRequirementItems drops the removed entries, merges the updated ones into their YAML nodes and appends the added ones.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func mergeRequirementItems(sequence *yaml.Node, operations validatedRequirementOperations, add []model.Requirement) ([]*yaml.Node, error) {
	merged := make([]*yaml.Node, 0, len(sequence.Content)+len(add))
	seen := map[string]bool{}
	for _, item := range sequence.Content {
		id, err := uniqueRequirementNodeID(item, seen)
		if err != nil {
			return nil, err
		}
		if operations.remove[id] {
			continue
		}
		if err := applyRequirementUpdate(item, id, operations.update); err != nil {
			return nil, err
		}
		merged = append(merged, item)
	}
	added, err := encodeRequirementNodes(add)
	if err != nil {
		return nil, err
	}
	return append(merged, added...), nil
}

// encodeRequirementNodes encodes the added requirements in order.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func encodeRequirementNodes(requirements []model.Requirement) ([]*yaml.Node, error) {
	nodes := make([]*yaml.Node, 0, len(requirements))
	for _, requirement := range requirements {
		node, err := encodeRequirementNode(requirement)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// uniqueRequirementNodeID returns the id of a requirement node and records it, refusing a duplicate.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func uniqueRequirementNodeID(item *yaml.Node, seen map[string]bool) (string, error) {
	id, err := requirementNodeID(item)
	if err != nil {
		return "", err
	}
	if seen[id] {
		return "", fmt.Errorf("canonical requirements YAML contains duplicate id %q", id)
	}
	seen[id] = true
	return id, nil
}

// applyRequirementUpdate merges the replacement of requirement id into its YAML node, if there is one.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func applyRequirementUpdate(item *yaml.Node, id string, updates map[string]model.Requirement) error {
	replacement, ok := updates[id]
	if !ok {
		return nil
	}
	node, err := encodeRequirementNode(replacement)
	if err != nil {
		return err
	}
	if err := mergeRequirementNode(item, node); err != nil {
		return fmt.Errorf("merge requirement %q: %w", id, err)
	}
	return nil
}

// encodeRequirementsDocument encodes the merged YAML tree with two-space indentation.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT
func encodeRequirementsDocument(document *yaml.Node) ([]byte, error) {
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return nil, fmt.Errorf("encode merged requirements: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("close merged requirements encoder: %w", err)
	}
	return output.Bytes(), nil
}

type validatedRequirementOperations struct {
	add    map[string]model.Requirement
	update map[string]model.Requirement
	remove map[string]bool
}

// TRLC-LINKS: REQ-EMG-031
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA
func validateRequirementsDeltaOperations(operations RequirementsDeltaOperations) (validatedRequirementOperations, error) {
	out := validatedRequirementOperations{
		add:    map[string]model.Requirement{},
		update: map[string]model.Requirement{},
		remove: map[string]bool{},
	}
	registry := requirementOperationRegistry{seen: map[string]string{}}
	if err := registry.registerRequirements(operations.Add, "add", out.add); err != nil {
		return validatedRequirementOperations{}, err
	}
	if err := registry.registerRequirements(operations.Update, "update", out.update); err != nil {
		return validatedRequirementOperations{}, err
	}
	for _, rawID := range operations.Remove {
		if err := registry.register(rawID, "remove"); err != nil {
			return validatedRequirementOperations{}, err
		}
		out.remove[strings.TrimSpace(rawID)] = true
	}
	if len(registry.seen) == 0 {
		return validatedRequirementOperations{}, fmt.Errorf("requirements delta has no operations")
	}
	return out, nil
}

// requirementOperationRegistry remembers which operation named each requirement id.
type requirementOperationRegistry struct {
	seen map[string]string
}

// register refuses an empty id, an id with surrounding whitespace and an id named by two operations.
// TRLC-LINKS: REQ-EMG-031
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA
func (r requirementOperationRegistry) register(id, operation string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return fmt.Errorf("%s operation has an empty requirement id", operation)
	}
	if trimmed != id {
		return fmt.Errorf("%s requirement id %q contains surrounding whitespace", operation, id)
	}
	if previous, exists := r.seen[trimmed]; exists {
		return fmt.Errorf("requirement %q appears in both %s and %s operations", trimmed, previous, operation)
	}
	r.seen[trimmed] = operation
	return nil
}

// registerRequirements registers each requirement of one operation and stores it by trimmed id in into.
// TRLC-LINKS: REQ-EMG-031
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA
func (r requirementOperationRegistry) registerRequirements(requirements []model.Requirement, operation string, into map[string]model.Requirement) error {
	for _, requirement := range requirements {
		if err := r.register(requirement.ID, operation); err != nil {
			return err
		}
		requirement.ID = strings.TrimSpace(requirement.ID)
		into[requirement.ID] = requirement
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-VALIDATION-ENGINE, DO-ARCHITECTURE-MODEL
// ENGMODEL-LINKS: EVT-REQUIREMENTS-CANDIDATE-VALIDATED
func validateRequirementsCandidate(bundle model.Bundle, requirements model.RequirementsDocument) []validate.Diagnostic {
	diags := validate.Bundle(bundle)
	diags = append(diags, reqlint.ValidateCatalogDescriptions(bundle.Catalog)...)
	diags = append(diags, reqlint.LintModelRequirementsEARS(bundle, requirements)...)
	diags = append(diags, reqlint.LintRequirementInternalLinks(requirements)...)

	checker := requirementCandidateChecker{seen: map[string]bool{}, validTargets: trace.KnownModelIDs(bundle, model.RequirementsDocument{})}
	for i, requirement := range requirements.Requirements {
		diags = append(diags, checker.check(i, requirement)...)
	}
	return validate.SortDiagnostics(diags)
}

// requirementCandidateChecker checks the entries of a candidate requirements document one by one.
type requirementCandidateChecker struct {
	seen         map[string]bool
	validTargets map[string]bool
}

// check reports the diagnostics of the requirement at index i.
// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-VALIDATION-ENGINE, DO-ARCHITECTURE-MODEL
// ENGMODEL-LINKS: EVT-REQUIREMENTS-CANDIDATE-VALIDATED
func (c requirementCandidateChecker) check(i int, requirement model.Requirement) []validate.Diagnostic {
	path := fmt.Sprintf("requirements[%d]", i)
	diags := c.idDiagnostics(path, strings.TrimSpace(requirement.ID))
	if strings.TrimSpace(requirement.Text) == "" {
		diags = append(diags, validate.Diagnostic{Code: "requirement.missing_text", Severity: validate.SeverityError, Message: "requirement text is required", Path: path})
	}
	return append(diags, c.appliesToDiagnostics(path, requirement.AppliesTo)...)
}

// idDiagnostics reports a missing or duplicate id and records a new one.
// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-VALIDATION-ENGINE, DO-ARCHITECTURE-MODEL
// ENGMODEL-LINKS: EVT-REQUIREMENTS-CANDIDATE-VALIDATED
func (c requirementCandidateChecker) idDiagnostics(path, id string) []validate.Diagnostic {
	switch {
	case id == "":
		return []validate.Diagnostic{{Code: "requirement.missing_id", Severity: validate.SeverityError, Message: "requirement id is required", Path: path}}
	case c.seen[id]:
		return []validate.Diagnostic{{Code: "requirement.duplicate_id", Severity: validate.SeverityError, Message: fmt.Sprintf("duplicate requirement id %q", id), Path: path}}
	}
	c.seen[id] = true
	return nil
}

// appliesToDiagnostics reports each appliesTo id that is empty or not a known model id.
// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-VALIDATION-ENGINE, DO-ARCHITECTURE-MODEL
// ENGMODEL-LINKS: EVT-REQUIREMENTS-CANDIDATE-VALIDATED
func (c requirementCandidateChecker) appliesToDiagnostics(path string, appliesTo []string) []validate.Diagnostic {
	var diags []validate.Diagnostic
	for j, rawTarget := range appliesTo {
		target := strings.TrimSpace(rawTarget)
		if target != "" && c.validTargets[target] {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code:     "requirement.invalid_applies_to",
			Severity: validate.SeverityError,
			Message:  fmt.Sprintf("unknown appliesTo id %q", target),
			Path:     fmt.Sprintf("%s.appliesTo[%d]", path, j),
		})
	}
	return diags
}

// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func requirementsSequence(document *yaml.Node) (*yaml.Node, error) {
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("requirements YAML must contain one top-level mapping")
	}
	root := document.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "requirements" {
			continue
		}
		sequence := root.Content[i+1]
		if sequence.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("requirements field must be a sequence")
		}
		return sequence, nil
	}
	return nil, fmt.Errorf("requirements field is required")
}

// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func requirementNodeID(node *yaml.Node) (string, error) {
	if node.Kind != yaml.MappingNode {
		return "", fmt.Errorf("requirement entry must be a mapping")
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "id" {
			return trimmedRequirementNodeID(node.Content[i+1])
		}
	}
	return "", fmt.Errorf("requirement entry is missing id")
}

// trimmedRequirementNodeID returns the trimmed id of an id value node and refuses an empty one.
// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func trimmedRequirementNodeID(value *yaml.Node) (string, error) {
	id := strings.TrimSpace(value.Value)
	if id == "" {
		return "", fmt.Errorf("requirement entry has an empty id")
	}
	return id, nil
}

// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func encodeRequirementNode(requirement model.Requirement) (*yaml.Node, error) {
	var node yaml.Node
	if err := node.Encode(requirement); err != nil {
		return nil, fmt.Errorf("encode requirement %q: %w", requirement.ID, err)
	}
	return &node, nil
}

// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func mergeRequirementNode(existing, replacement *yaml.Node) error {
	if existing.Kind != yaml.MappingNode || replacement.Kind != yaml.MappingNode {
		return fmt.Errorf("requirement entry must be a mapping")
	}
	index := indexMappingNode(replacement)
	seen := replaceMappingValues(existing, index)
	for _, key := range index.order {
		if !seen[key] {
			existing.Content = append(existing.Content, index.keys[key], index.values[key])
		}
	}
	return nil
}

// replaceMappingValues replaces the values of existing keys by the indexed ones, keeping the old presentation, and
// returns the keys it replaced.
// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func replaceMappingValues(existing *yaml.Node, index mappingNodeIndex) map[string]bool {
	seen := map[string]bool{}
	for i := 0; i+1 < len(existing.Content); i += 2 {
		key := existing.Content[i].Value
		value, ok := index.values[key]
		if !ok {
			continue
		}
		copyNodePresentation(value, existing.Content[i+1])
		existing.Content[i+1] = value
		seen[key] = true
	}
	return seen
}

// mappingNodeIndex is a mapping node's key and value nodes by key, and the key order.
type mappingNodeIndex struct {
	keys   map[string]*yaml.Node
	values map[string]*yaml.Node
	order  []string
}

// indexMappingNode indexes the key and value nodes of a mapping node.
// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func indexMappingNode(node *yaml.Node) mappingNodeIndex {
	index := mappingNodeIndex{keys: map[string]*yaml.Node{}, values: map[string]*yaml.Node{}, order: make([]string, 0, len(node.Content)/2)}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		index.keys[key] = node.Content[i]
		index.values[key] = node.Content[i+1]
		index.order = append(index.order, key)
	}
	return index
}

// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func copyNodePresentation(target, source *yaml.Node) {
	target.HeadComment = source.HeadComment
	target.LineComment = source.LineComment
	target.FootComment = source.FootComment
	if target.Kind != source.Kind {
		return
	}
	target.Style = source.Style
	switch target.Kind {
	case yaml.MappingNode:
		copyMappingPresentation(target, source)
	case yaml.SequenceNode:
		copySequencePresentation(target, source)
	}
}

// copyMappingPresentation copies the presentation of each source entry onto the target entry with the same key.
// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func copyMappingPresentation(target, source *yaml.Node) {
	sourceValues := map[string]int{}
	for i := 0; i+1 < len(source.Content); i += 2 {
		sourceValues[source.Content[i].Value] = i
	}
	for i := 0; i+1 < len(target.Content); i += 2 {
		sourceIndex, ok := sourceValues[target.Content[i].Value]
		if !ok {
			continue
		}
		copyNodePresentation(target.Content[i], source.Content[sourceIndex])
		copyNodePresentation(target.Content[i+1], source.Content[sourceIndex+1])
	}
}

// copySequencePresentation pairs each target item with the first unused source item of the same kind and value.
// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func copySequencePresentation(target, source *yaml.Node) {
	used := make([]bool, len(source.Content))
	for _, targetChild := range target.Content {
		if sourceIndex := firstUnusedSameNode(targetChild, source.Content, used); sourceIndex >= 0 {
			copyNodePresentation(targetChild, source.Content[sourceIndex])
			used[sourceIndex] = true
		}
	}
}

// firstUnusedSameNode returns the index of the first unused node in candidates with the kind and value of child, or -1.
// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT
func firstUnusedSameNode(child *yaml.Node, candidates []*yaml.Node, used []bool) int {
	for i, candidate := range candidates {
		if !used[i] && child.Kind == candidate.Kind && child.Value == candidate.Value {
			return i
		}
	}
	return -1
}

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA
func decodeStrictYAML(content []byte, out any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple YAML documents are not supported")
		}
		return err
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA
func renderRequirementsDiff(changes requirementChanges) string {
	var output strings.Builder
	total := len(changes.added) + len(changes.updated) + len(changes.removed)
	fmt.Fprintf(&output, "Requirements delta (%d changes)\n", total)
	for _, id := range sortedRequirementKeys(changes.added) {
		fmt.Fprintf(&output, "+ %s: %s\n", id, strconv.Quote(changes.added[id].Text))
	}
	for _, id := range sortedRequirementKeys(changes.updated) {
		fmt.Fprintf(&output, "~ %s\n", id)
		for _, diff := range requirementFieldDiffs(changes.updated[id]) {
			diff.render(&output)
		}
	}
	for _, id := range sortedRequirementKeys(changes.removed) {
		fmt.Fprintf(&output, "- %s: %s\n", id, strconv.Quote(changes.removed[id].Text))
	}
	return output.String()
}

// requirementFieldDiff is one field of an updated requirement with its text before and after.
type requirementFieldDiff struct {
	field  string
	before string
	after  string
}

// requirementFieldDiffs lists the fields of an update in diff order.
// TRLC-LINKS: REQ-EMG-032, REQ-EMG-034
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DIFF
func requirementFieldDiffs(change requirementUpdate) []requirementFieldDiff {
	before, after := change.before, change.after
	join := func(values []string) string { return strings.Join(values, ", ") }
	return []requirementFieldDiff{
		{"title", before.Title, after.Title},
		{"text", before.Text, after.Text},
		{"notes", before.Notes, after.Notes},
		{"category", before.Category, after.Category},
		{"rationale", before.Rationale, after.Rationale},
		{"sourceRefs", join(before.SourceRefs), join(after.SourceRefs)},
		{"verificationMethods", join(before.VerificationMethods), join(after.VerificationMethods)},
		{"verificationCriteria", before.VerificationCriteria, after.VerificationCriteria},
		{"priority", before.Priority, after.Priority},
		{"criticality", before.Criticality, after.Criticality},
		{"status", before.Status, after.Status},
		{"derived", strconv.FormatBool(before.Derived), strconv.FormatBool(after.Derived)},
		{"derivedRationale", before.DerivedRationale, after.DerivedRationale},
		{"tags", join(before.Tags), join(after.Tags)},
		{"appliesTo", join(before.AppliesTo), join(after.AppliesTo)},
	}
}

// TRLC-LINKS: REQ-EMG-034
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DIFF
func (d requirementFieldDiff) render(output *strings.Builder) {
	if d.before == d.after {
		return
	}
	fmt.Fprintf(output, "  %s: %s -> %s\n", d.field, strconv.Quote(d.before), strconv.Quote(d.after))
}

// TRLC-LINKS: REQ-EMG-034
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DIFF
func sortedRequirementKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DOCUMENT, DEP-LOCAL-WORKSPACE
func requirementsPath(root string) (string, error) {
	rootPath, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", fmt.Errorf("resolve model root: %w", err)
	}
	rootPath, err = filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", fmt.Errorf("resolve canonical model root: %w", err)
	}
	targetPath := filepath.Join(rootPath, "model", "requirements.yml")
	info, err := os.Lstat(targetPath)
	if err != nil {
		return "", fmt.Errorf("inspect requirements: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("requirements path must not be a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("requirements path must be a regular file")
	}
	return targetPath, nil
}

// TRLC-LINKS: REQ-EMG-033, REQ-EMG-185
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DEP-LOCAL-WORKSPACE
// ENGMODEL-LINKS: STATE-DELTA-LOCKED, TR-DELTA-LOCK, TR-DELTA-LOCK-BUSY, TR-DELTA-OTHER-RUN-LOCKS, TR-DELTA-OTHER-RUN-UNLOCKS
// ENGMODEL-LINKS: EVT-REQUIREMENTS-LOCK-CHANGED-BY-OTHER-RUN
func acquireRequirementsLock(targetPath string) (func() error, error) {
	targetDir, err := filepath.EvalSymlinks(filepath.Dir(targetPath))
	if err != nil {
		return nil, fmt.Errorf("resolve requirements change target directory: %w", err)
	}
	lockPath := filepath.Join(targetDir, ".requirements.yml.engchange.lock")
	lock := flock.New(lockPath)
	locked, err := lock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("acquire requirements change lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("requirements change already in progress: %s", targetPath)
	}
	return func() error {
		if err := lock.Unlock(); err != nil {
			return fmt.Errorf("release requirements change lock: %w", err)
		}
		return nil
	}, nil
}

// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DEP-LOCAL-WORKSPACE
func writeFileAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tempPath, err := writeTempRequirements(dir, content, mode)
	if err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace requirements atomically: %w", err)
	}
	return syncRequirementsDir(dir)
}

// writeTempRequirements writes content to a new temporary file in dir and returns its path.
// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DEP-LOCAL-WORKSPACE
func writeTempRequirements(dir string, content []byte, mode os.FileMode) (string, error) {
	temp, err := os.CreateTemp(dir, ".requirements-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary requirements file: %w", err)
	}
	tempPath := temp.Name()
	if err := fillTempRequirements(temp, content, mode); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return "", err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("close temporary requirements: %w", err)
	}
	return tempPath, nil
}

// fillTempRequirements sets the permissions, writes the content and syncs the temporary file.
// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DEP-LOCAL-WORKSPACE
func fillTempRequirements(temp *os.File, content []byte, mode os.FileMode) error {
	if err := temp.Chmod(mode.Perm()); err != nil {
		return fmt.Errorf("set temporary requirements permissions: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		return fmt.Errorf("write temporary requirements: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary requirements: %w", err)
	}
	return nil
}

// syncRequirementsDir syncs the directory after the replace, except on Windows.
// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DEP-LOCAL-WORKSPACE
func syncRequirementsDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("requirements replaced but open directory for sync: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("requirements replaced but sync directory: %w", err)
	}
	return nil
}
