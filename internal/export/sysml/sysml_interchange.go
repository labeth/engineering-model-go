// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package sysml

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

const (
	sysMLProjectSourceName = "model.sysml"
	sysMLProjectPublisher  = "labeth"
	sysMLProjectVersion    = "1.0.0"
)

// RoundTripResult captures the normative KPAR path and reopened native
// SysML source used for source-integrity comparison.
type RoundTripResult struct {
	ProjectDir  string
	KPARPath    string
	ReopenedDir string
	SourcePath  string
	SourceText  string
}

// InterchangePaths locates the files of a Sysand interchange: Directory is
// the project directory to create when exporting and the directory to clone
// into when reopening; KPAR is the archive; Sysand is the sysand executable
// (the one on PATH when empty).
type InterchangePaths struct {
	Directory string
	KPAR      string
	Sysand    string
}

// ExportV2Project creates a Sysand interchange project and builds its KPAR.
// Sysand, rather than a custom ZIP writer, owns all normative KPAR packaging.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-SYSML-V2-MODEL, REF-SYSML-V2-TOOLCHAIN
func ExportV2Project(architecturePath string, paths InterchangePaths) (ExportResult, error) {
	if err := requireEmptyDirectory(paths.Directory); err != nil {
		return ExportResult{}, err
	}
	sysand, err := resolveExternalTool(paths.Sysand, "sysand")
	if err != nil {
		return ExportResult{}, err
	}
	result, err := GenerateV2FromFile(architecturePath)
	if err != nil {
		return result, err
	}
	if err := rejectLossyOrUnknownDiagnostics(result.Diagnostics); err != nil {
		return result, err
	}
	paths, err = prepareExportPaths(paths)
	if err != nil {
		return result, err
	}
	return result, buildKPAR(sysand, paths, result.Text)
}

// prepareExportPaths makes the project directory and KPAR paths absolute and
// creates their parent directories.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func prepareExportPaths(paths InterchangePaths) (InterchangePaths, error) {
	var err error
	if paths.Directory, err = absPath(paths.Directory, "project directory"); err != nil {
		return paths, err
	}
	if paths.KPAR, err = absPath(paths.KPAR, "KPAR path"); err != nil {
		return paths, err
	}
	if err := ensureParent(paths.Directory, "project"); err != nil {
		return paths, err
	}
	return paths, ensureParent(paths.KPAR, "KPAR")
}

// buildKPAR writes the SysML source into a fresh Sysand project and builds the KPAR.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func buildKPAR(sysand string, paths InterchangePaths, text string) error {
	projectName := sysMLProjectName(text)
	if err := runExternal("", sysand, "init", "--publisher", sysMLProjectPublisher, "--name", projectName, "--version", sysMLProjectVersion, paths.Directory); err != nil {
		return err
	}
	sourcePath := filepath.Join(paths.Directory, sysMLProjectSourceName)
	if err := os.WriteFile(sourcePath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write SysML project source: %w", err)
	}
	if err := runExternal(paths.Directory, sysand, "include", "--compute-checksum", sysMLProjectSourceName); err != nil {
		return err
	}
	return runExternal(paths.Directory, sysand, "build", paths.KPAR)
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func absPath(path, label string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", label, err)
	}
	return abs, nil
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func ensureParent(path, label string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s parent: %w", label, err)
	}
	return nil
}

// ReopenV2Project clones a KPAR with Sysand and returns its native SysML
// project source.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func ReopenV2Project(paths InterchangePaths) (RoundTripResult, error) {
	if err := requireEmptyDirectory(paths.Directory); err != nil {
		return RoundTripResult{}, err
	}
	sysand, err := resolveExternalTool(paths.Sysand, "sysand")
	if err != nil {
		return RoundTripResult{}, err
	}
	paths, err = prepareReopenPaths(paths)
	if err != nil {
		return RoundTripResult{}, err
	}
	if err := runExternal("", sysand, "clone", "--no-deps", "--path", paths.KPAR, "--target", paths.Directory); err != nil {
		return RoundTripResult{}, err
	}
	sourcePath := filepath.Join(paths.Directory, sysMLProjectSourceName)
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return RoundTripResult{}, fmt.Errorf("read reopened SysML project source: %w", err)
	}
	return RoundTripResult{
		KPARPath:    paths.KPAR,
		ReopenedDir: paths.Directory,
		SourcePath:  sourcePath,
		SourceText:  string(source),
	}, nil
}

// prepareReopenPaths makes the KPAR and reopen directory paths absolute and
// creates the parent of the reopen directory.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func prepareReopenPaths(paths InterchangePaths) (InterchangePaths, error) {
	var err error
	if paths.KPAR, err = absPath(paths.KPAR, "KPAR path"); err != nil {
		return paths, err
	}
	if paths.Directory, err = absPath(paths.Directory, "reopen directory"); err != nil {
		return paths, err
	}
	return paths, ensureParent(paths.Directory, "reopen")
}

// VerifyV2RoundTrip compares the reopened KPAR source to a fresh native
// SysML projection of the authored Engineering Model.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func VerifyV2RoundTrip(architecturePath string, paths InterchangePaths) (RoundTripResult, error) {
	expected, err := GenerateV2FromFile(architecturePath)
	if err != nil {
		return RoundTripResult{}, err
	}
	result, err := ReopenV2Project(paths)
	if err != nil {
		return result, err
	}
	if result.SourceText != expected.Text {
		return result, errors.New("SysML KPAR source round trip differs from the native projection")
	}
	return result, nil
}

// ImportV2Project reconstructs the canonical semantic model from legacy
// projects that embedded an Engineering Model payload. Native projections do
// not embed non-SysML canonical data.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func ImportV2Project(sourcePath string) (model.SemanticModel, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return model.SemanticModel{}, fmt.Errorf("open SysML project source: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var state projectAnnotationState
	for scanner.Scan() {
		if payload, complete := state.feed(strings.TrimSpace(scanner.Text())); complete {
			return decodeProjectPayload(payload)
		}
	}
	if err := scanner.Err(); err != nil {
		return model.SemanticModel{}, fmt.Errorf("read SysML project source: %w", err)
	}
	return model.SemanticModel{}, errors.New("SysML project source has no EngineeringProject interchange payload")
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func decodeProjectPayload(payload string) (model.SemanticModel, error) {
	var semantic model.SemanticModel
	if err := json.Unmarshal([]byte(payload), &semantic); err != nil {
		return model.SemanticModel{}, fmt.Errorf("decode EngineeringProject payload: %w", err)
	}
	return semantic, nil
}

// projectAnnotationState follows the lines of an EngineeringProject annotation
// and keeps the payload of the engineering model it carries.
type projectAnnotationState struct {
	inAnnotation bool
	isProject    bool
	payload      string
}

// feed reads one trimmed line; it returns the payload when the annotation of
// the engineering model project closes.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func (s *projectAnnotationState) feed(line string) (string, bool) {
	switch {
	case line == "@EngineeringProject {":
		*s = projectAnnotationState{inAnnotation: true}
	case !s.inAnnotation:
	case line == "}":
		if s.isProject && s.payload != "" {
			return s.payload, true
		}
		s.inAnnotation = false
	default:
		s.property(line)
	}
	return "", false
}

// property records the conceptKind or payload assignment of a line.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func (s *projectAnnotationState) property(line string) {
	key, value, ok := strings.Cut(strings.TrimSuffix(line, ";"), "=")
	if !ok {
		return
	}
	decoded, err := strconv.Unquote(strings.TrimSpace(value))
	if err != nil {
		return
	}
	switch strings.TrimSpace(key) {
	case "conceptKind":
		s.isProject = decoded == "engineering_model"
	case "payload":
		s.payload = decoded
	}
}

// CompareV2RoundTrip reports differences in stable identity, ownership,
// relationships, expressions, references, and typed extensions.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func CompareV2RoundTrip(expected, actual model.SemanticModel) []string {
	expected = normalizeInterchangeModel(expected)
	actual = normalizeInterchangeModel(actual)
	var differences []string
	if expected.ID != actual.ID || expected.Title != actual.Title {
		differences = append(differences, "project identity")
	}
	if !reflect.DeepEqual(expected.Imports, actual.Imports) {
		differences = append(differences, "library/project references")
	}
	differences = append(differences, compareInterchangeElements(expected.Elements, actual.Elements)...)
	if !reflect.DeepEqual(expected.Relationships, actual.Relationships) {
		differences = append(differences, "relationships and expressions")
	}
	return differences
}

// compareInterchangeElements reports a difference of the element identities,
// or else the distinct differences of each element.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func compareInterchangeElements(expected, actual []model.SemanticElement) []string {
	expectedElements := semanticElementsByID(expected)
	actualElements := semanticElementsByID(actual)
	ids := sortedSemanticElementKeys(expectedElements)
	if !reflect.DeepEqual(ids, sortedSemanticElementKeys(actualElements)) {
		return []string{"stable element identity"}
	}
	var differences []string
	for _, id := range ids {
		for _, difference := range compareInterchangeElement(expectedElements[id], actualElements[id]) {
			differences = appendUniqueRoundTripDifference(differences, difference)
		}
	}
	return differences
}

// compareInterchangeElement reports what differs between two elements of the same identity.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func compareInterchangeElement(want, got model.SemanticElement) []string {
	var differences []string
	if want.Owner != got.Owner || want.Namespace != got.Namespace {
		differences = append(differences, "ownership")
	}
	if !reflect.DeepEqual(want.Features, got.Features) {
		differences = append(differences, "typed features and expressions")
	}
	if typedExtensionsDiffer(want, got) {
		differences = append(differences, "typed extensions")
	}
	if !reflect.DeepEqual(withoutComparedFields(want), withoutComparedFields(got)) {
		differences = append(differences, "element semantics")
	}
	return differences
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func typedExtensionsDiffer(want, got model.SemanticElement) bool {
	if !reflect.DeepEqual(want.Metadata, got.Metadata) || !reflect.DeepEqual(want.Targets, got.Targets) {
		return true
	}
	return want.ExtensionNamespace != got.ExtensionNamespace || want.Extension != got.Extension
}

// withoutComparedFields clears the fields that compareInterchangeElement has
// already compared on their own.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func withoutComparedFields(element model.SemanticElement) model.SemanticElement {
	element.Metadata = nil
	element.Features = nil
	element.Owner = ""
	element.Namespace = ""
	element.ExtensionNamespace = ""
	element.Extension = ""
	element.Targets = nil
	return element
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func normalizeInterchangeModel(semantic model.SemanticModel) model.SemanticModel {
	data, err := json.Marshal(semantic)
	if err != nil {
		return semantic
	}
	var normalized model.SemanticModel
	if err := json.Unmarshal(data, &normalized); err != nil {
		return semantic
	}
	for index := range normalized.Elements {
		normalized.Elements[index].Properties = authoredMetamodelProperties(normalized.Elements[index].Properties)
	}
	for index := range normalized.Relationships {
		normalized.Relationships[index].Properties = authoredMetamodelProperties(normalized.Relationships[index].Properties)
	}
	return normalized
}

// Derived and implied values are reconstructed by the official importer from
// native syntax and are intentionally excluded from interchange comparison.
//
// TRLC-LINKS: REQ-EMG-040
func authoredMetamodelProperties(properties map[string]model.MetamodelPropertyValue) map[string]model.MetamodelPropertyValue {
	if len(properties) == 0 {
		return nil
	}
	out := make(map[string]model.MetamodelPropertyValue, len(properties))
	for name, value := range properties {
		if value.Derived || value.Implied {
			continue
		}
		out[name] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func semanticElementsByID(elements []model.SemanticElement) map[string]model.SemanticElement {
	out := make(map[string]model.SemanticElement, len(elements))
	for _, element := range elements {
		out[element.ID] = element
	}
	return out
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func sortedSemanticElementKeys(values map[string]model.SemanticElement) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func appendUniqueRoundTripDifference(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func rejectLossyOrUnknownDiagnostics(diagnostics []model.SemanticDiagnostic) error {
	var blocking []string
	for _, diagnostic := range diagnostics {
		if isBlockingInterchangeDiagnostic(diagnostic) {
			blocking = append(blocking, diagnostic.Code)
		}
	}
	if len(blocking) > 0 {
		sort.Strings(blocking)
		return fmt.Errorf("SysML interchange blocked by diagnostics: %s", strings.Join(blocking, ", "))
	}
	return nil
}

// isBlockingInterchangeDiagnostic reports an error or a diagnostic whose code
// names a lossy, unsupported or unknown construct.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func isBlockingInterchangeDiagnostic(diagnostic model.SemanticDiagnostic) bool {
	if diagnostic.Severity == model.SemanticSeverityError {
		return true
	}
	for _, marker := range []string{"lossy", "unsupported", "unknown"} {
		if strings.Contains(diagnostic.Code, marker) {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func requireEmptyDirectory(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("directory path is required")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect directory %s: %w", path, err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("directory %s must not exist or must be empty", path)
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func resolveExternalTool(configured, name string) (string, error) {
	if strings.TrimSpace(configured) != "" {
		path, err := filepath.Abs(configured)
		if err != nil {
			return "", fmt.Errorf("resolve %s path: %w", name, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("%s is unavailable at %s: %w", name, configured, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s path %s is a directory", name, configured)
		}
		return path, nil
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s is required for normative KPAR packaging; install the pinned toolchain with scripts/install-sysml-toolchain.sh: %w", name, err)
	}
	return path, nil
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func runExternal(dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w\n%s", command, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func sysMLProjectName(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if name, ok := projectNameOfLine(strings.TrimSpace(line)); ok {
			return name
		}
	}
	return "engineering-model"
}

// projectNameOfLine returns the quoted name of a package declaration line.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func projectNameOfLine(line string) (string, bool) {
	const prefix = "package '"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(line, prefix)
	index := strings.Index(name, "'")
	if index <= 0 {
		return "", false
	}
	return name[:index], true
}
