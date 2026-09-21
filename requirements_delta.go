// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package engmodel

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/gofrs/flock"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
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
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-REQUIREMENTS-DELTA, DO-REQUIREMENTS-DOCUMENT, DEP-LOCAL-WORKSPACE
func ApplyRequirementsDelta(root, deltaPath string) (result RequirementsDeltaResult, err error) {
	targetPath, err := requirementsPath(root)
	if err != nil {
		return RequirementsDeltaResult{}, err
	}
	release, err := acquireRequirementsLock(targetPath)
	if err != nil {
		return RequirementsDeltaResult{}, err
	}
	defer func() {
		if releaseErr := release(); err == nil && releaseErr != nil {
			err = fmt.Errorf("requirements applied but lock release failed: %w", releaseErr)
		}
	}()

	plan, err := planRequirementsDelta(targetPath, deltaPath)
	if err != nil {
		return RequirementsDeltaResult{}, err
	}
	if validate.HasErrors(plan.result.Diagnostics) {
		return plan.result, fmt.Errorf("requirements delta candidate is invalid")
	}
	current, err := os.ReadFile(plan.targetPath)
	if err != nil {
		return plan.result, fmt.Errorf("re-read requirements before apply: %w", err)
	}
	if !bytes.Equal(current, plan.source) {
		return plan.result, fmt.Errorf("requirements changed while delta was being planned")
	}
	if err := writeFileAtomic(plan.targetPath, plan.result.YAML, plan.mode); err != nil {
		return plan.result, err
	}
	return plan.result, nil
}

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-MODEL-LOADER, FU-VALIDATION-ENGINE
func planRequirementsDelta(targetPath, deltaPath string) (requirementsDeltaPlan, error) {
	rootPath := filepath.Dir(targetPath)
	source, err := os.ReadFile(targetPath)
	if err != nil {
		return requirementsDeltaPlan{}, fmt.Errorf("read requirements: %w", err)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return requirementsDeltaPlan{}, fmt.Errorf("stat requirements: %w", err)
	}

	var current model.RequirementsDocument
	if err := decodeStrictYAML(source, &current); err != nil {
		return requirementsDeltaPlan{}, fmt.Errorf("decode requirements: %w", err)
	}
	delta, err := loadRequirementsDelta(deltaPath)
	if err != nil {
		return requirementsDeltaPlan{}, err
	}
	candidateYAML, changes, err := mergeRequirementsYAML(source, current, delta)
	if err != nil {
		return requirementsDeltaPlan{}, err
	}
	var candidate model.RequirementsDocument
	if err := decodeStrictYAML(candidateYAML, &candidate); err != nil {
		return requirementsDeltaPlan{}, fmt.Errorf("decode merged requirements: %w", err)
	}
	bundle, err := model.LoadBundle(filepath.Join(filepath.Dir(rootPath), "engmod.yml"))
	if err != nil {
		return requirementsDeltaPlan{}, fmt.Errorf("load model bundle: %w", err)
	}
	diags := validateRequirementsCandidate(bundle, candidate)
	result := RequirementsDeltaResult{
		Candidate:   candidate,
		YAML:        candidateYAML,
		Diff:        renderRequirementsDiff(changes),
		Added:       sortedRequirementKeys(changes.added),
		Updated:     sortedRequirementKeys(changes.updated),
		Removed:     sortedRequirementKeys(changes.removed),
		Diagnostics: diags,
	}
	return requirementsDeltaPlan{result: result, targetPath: targetPath, source: source, mode: info.Mode()}, nil
}

// TRLC-LINKS: REQ-EMG-031
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DELTA
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

	currentByID := make(map[string]model.Requirement, len(current.Requirements))
	for _, requirement := range current.Requirements {
		id := strings.TrimSpace(requirement.ID)
		if id == "" {
			return nil, requirementChanges{}, fmt.Errorf("canonical requirements contain an empty id")
		}
		if _, exists := currentByID[id]; exists {
			return nil, requirementChanges{}, fmt.Errorf("canonical requirements contain duplicate id %q", id)
		}
		currentByID[id] = requirement
	}

	changes := requirementChanges{
		added:   map[string]model.Requirement{},
		updated: map[string]requirementUpdate{},
		removed: map[string]model.Requirement{},
	}
	for id, requirement := range operations.add {
		if _, exists := currentByID[id]; exists {
			return nil, requirementChanges{}, fmt.Errorf("add requirement %q already exists", id)
		}
		changes.added[id] = requirement
	}
	for id, requirement := range operations.update {
		before, exists := currentByID[id]
		if !exists {
			return nil, requirementChanges{}, fmt.Errorf("update requirement %q does not exist", id)
		}
		changes.updated[id] = requirementUpdate{before: before, after: requirement}
	}
	for id := range operations.remove {
		before, exists := currentByID[id]
		if !exists {
			return nil, requirementChanges{}, fmt.Errorf("remove requirement %q does not exist", id)
		}
		changes.removed[id] = before
	}

	var document yaml.Node
	if err := decodeStrictYAML(source, &document); err != nil {
		return nil, requirementChanges{}, fmt.Errorf("parse requirements YAML tree: %w", err)
	}
	sequence, err := requirementsSequence(&document)
	if err != nil {
		return nil, requirementChanges{}, err
	}

	merged := make([]*yaml.Node, 0, len(sequence.Content)+len(delta.Requirements.Add))
	seen := map[string]bool{}
	for _, item := range sequence.Content {
		id, err := requirementNodeID(item)
		if err != nil {
			return nil, requirementChanges{}, err
		}
		if seen[id] {
			return nil, requirementChanges{}, fmt.Errorf("canonical requirements YAML contains duplicate id %q", id)
		}
		seen[id] = true
		if operations.remove[id] {
			continue
		}
		if replacement, ok := operations.update[id]; ok {
			node, err := encodeRequirementNode(replacement)
			if err != nil {
				return nil, requirementChanges{}, err
			}
			if err := mergeRequirementNode(item, node); err != nil {
				return nil, requirementChanges{}, fmt.Errorf("merge requirement %q: %w", id, err)
			}
			merged = append(merged, item)
			continue
		}
		merged = append(merged, item)
	}
	for _, requirement := range delta.Requirements.Add {
		node, err := encodeRequirementNode(requirement)
		if err != nil {
			return nil, requirementChanges{}, err
		}
		merged = append(merged, node)
	}
	sequence.Content = merged

	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return nil, requirementChanges{}, fmt.Errorf("encode merged requirements: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, requirementChanges{}, fmt.Errorf("close merged requirements encoder: %w", err)
	}
	return output.Bytes(), changes, nil
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
	seen := map[string]string{}
	register := func(id, operation string) error {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			return fmt.Errorf("%s operation has an empty requirement id", operation)
		}
		if trimmed != id {
			return fmt.Errorf("%s requirement id %q contains surrounding whitespace", operation, id)
		}
		if previous, exists := seen[trimmed]; exists {
			return fmt.Errorf("requirement %q appears in both %s and %s operations", trimmed, previous, operation)
		}
		seen[trimmed] = operation
		return nil
	}
	for _, requirement := range operations.Add {
		if err := register(requirement.ID, "add"); err != nil {
			return validatedRequirementOperations{}, err
		}
		requirement.ID = strings.TrimSpace(requirement.ID)
		out.add[requirement.ID] = requirement
	}
	for _, requirement := range operations.Update {
		if err := register(requirement.ID, "update"); err != nil {
			return validatedRequirementOperations{}, err
		}
		requirement.ID = strings.TrimSpace(requirement.ID)
		out.update[requirement.ID] = requirement
	}
	for _, rawID := range operations.Remove {
		if err := register(rawID, "remove"); err != nil {
			return validatedRequirementOperations{}, err
		}
		id := strings.TrimSpace(rawID)
		out.remove[id] = true
	}
	if len(seen) == 0 {
		return validatedRequirementOperations{}, fmt.Errorf("requirements delta has no operations")
	}
	return out, nil
}

// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-VALIDATION-ENGINE, DO-ARCHITECTURE-MODEL
func validateRequirementsCandidate(bundle model.Bundle, requirements model.RequirementsDocument) []validate.Diagnostic {
	diags := validate.Bundle(bundle)
	diags = append(diags, validateCatalogDescriptions(bundle.Catalog)...)
	diags = append(diags, lintRequirementsEARS(requirements, bundle.Catalog)...)
	diags = append(diags, lintRequirementInternalLinks(requirements)...)

	seen := map[string]bool{}
	validTargets := knownModelIDs(bundle, model.RequirementsDocument{})
	for i, requirement := range requirements.Requirements {
		path := fmt.Sprintf("requirements[%d]", i)
		id := strings.TrimSpace(requirement.ID)
		switch {
		case id == "":
			diags = append(diags, validate.Diagnostic{Code: "requirement.missing_id", Severity: validate.SeverityError, Message: "requirement id is required", Path: path})
		case seen[id]:
			diags = append(diags, validate.Diagnostic{Code: "requirement.duplicate_id", Severity: validate.SeverityError, Message: fmt.Sprintf("duplicate requirement id %q", id), Path: path})
		default:
			seen[id] = true
		}
		if strings.TrimSpace(requirement.Text) == "" {
			diags = append(diags, validate.Diagnostic{Code: "requirement.missing_text", Severity: validate.SeverityError, Message: "requirement text is required", Path: path})
		}
		for j, rawTarget := range requirement.AppliesTo {
			target := strings.TrimSpace(rawTarget)
			if target == "" || !validTargets[target] {
				diags = append(diags, validate.Diagnostic{
					Code:     "requirement.invalid_applies_to",
					Severity: validate.SeverityError,
					Message:  fmt.Sprintf("unknown appliesTo id %q", target),
					Path:     fmt.Sprintf("%s.appliesTo[%d]", path, j),
				})
			}
		}
	}
	return validate.SortDiagnostics(diags)
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
			id := strings.TrimSpace(node.Content[i+1].Value)
			if id == "" {
				return "", fmt.Errorf("requirement entry has an empty id")
			}
			return id, nil
		}
	}
	return "", fmt.Errorf("requirement entry is missing id")
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
	replacementValues := map[string]*yaml.Node{}
	replacementKeys := map[string]*yaml.Node{}
	replacementOrder := make([]string, 0, len(replacement.Content)/2)
	for i := 0; i+1 < len(replacement.Content); i += 2 {
		key := replacement.Content[i].Value
		replacementKeys[key] = replacement.Content[i]
		replacementValues[key] = replacement.Content[i+1]
		replacementOrder = append(replacementOrder, key)
	}
	seen := map[string]bool{}
	for i := 0; i+1 < len(existing.Content); i += 2 {
		key := existing.Content[i].Value
		value, ok := replacementValues[key]
		if !ok {
			continue
		}
		copyNodePresentation(value, existing.Content[i+1])
		existing.Content[i+1] = value
		seen[key] = true
	}
	for _, key := range replacementOrder {
		if seen[key] {
			continue
		}
		existing.Content = append(existing.Content, replacementKeys[key], replacementValues[key])
	}
	return nil
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
	case yaml.SequenceNode:
		used := make([]bool, len(source.Content))
		for _, targetChild := range target.Content {
			for sourceIndex, sourceChild := range source.Content {
				if used[sourceIndex] {
					continue
				}
				if targetChild.Kind == sourceChild.Kind && targetChild.Value == sourceChild.Value {
					copyNodePresentation(targetChild, sourceChild)
					used[sourceIndex] = true
					break
				}
			}
		}
	}
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
		change := changes.updated[id]
		fmt.Fprintf(&output, "~ %s\n", id)
		renderRequirementFieldDiff(&output, "title", change.before.Title, change.after.Title)
		renderRequirementFieldDiff(&output, "text", change.before.Text, change.after.Text)
		renderRequirementFieldDiff(&output, "notes", change.before.Notes, change.after.Notes)
		renderRequirementFieldDiff(&output, "category", change.before.Category, change.after.Category)
		renderRequirementFieldDiff(&output, "rationale", change.before.Rationale, change.after.Rationale)
		renderRequirementFieldDiff(&output, "sourceRefs", strings.Join(change.before.SourceRefs, ", "), strings.Join(change.after.SourceRefs, ", "))
		renderRequirementFieldDiff(&output, "verificationMethods", strings.Join(change.before.VerificationMethods, ", "), strings.Join(change.after.VerificationMethods, ", "))
		renderRequirementFieldDiff(&output, "verificationCriteria", change.before.VerificationCriteria, change.after.VerificationCriteria)
		renderRequirementFieldDiff(&output, "priority", change.before.Priority, change.after.Priority)
		renderRequirementFieldDiff(&output, "criticality", change.before.Criticality, change.after.Criticality)
		renderRequirementFieldDiff(&output, "status", change.before.Status, change.after.Status)
		renderRequirementFieldDiff(&output, "derived", strconv.FormatBool(change.before.Derived), strconv.FormatBool(change.after.Derived))
		renderRequirementFieldDiff(&output, "derivedRationale", change.before.DerivedRationale, change.after.DerivedRationale)
		renderRequirementFieldDiff(&output, "tags", strings.Join(change.before.Tags, ", "), strings.Join(change.after.Tags, ", "))
		beforeAppliesTo := strings.Join(change.before.AppliesTo, ", ")
		afterAppliesTo := strings.Join(change.after.AppliesTo, ", ")
		renderRequirementFieldDiff(&output, "appliesTo", beforeAppliesTo, afterAppliesTo)
	}
	for _, id := range sortedRequirementKeys(changes.removed) {
		fmt.Fprintf(&output, "- %s: %s\n", id, strconv.Quote(changes.removed[id].Text))
	}
	return output.String()
}

// TRLC-LINKS: REQ-EMG-034
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DO-REQUIREMENTS-DIFF
func renderRequirementFieldDiff(output *strings.Builder, field, before, after string) {
	if before == after {
		return
	}
	fmt.Fprintf(output, "  %s: %s -> %s\n", field, strconv.Quote(before), strconv.Quote(after))
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

// TRLC-LINKS: REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, DEP-LOCAL-WORKSPACE
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
	temp, err := os.CreateTemp(dir, ".requirements-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary requirements file: %w", err)
	}
	tempPath := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}
	if err := temp.Chmod(mode.Perm()); err != nil {
		cleanup()
		return fmt.Errorf("set temporary requirements permissions: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		cleanup()
		return fmt.Errorf("write temporary requirements: %w", err)
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temporary requirements: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close temporary requirements: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace requirements atomically: %w", err)
	}
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
