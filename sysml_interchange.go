// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package engmodel

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

	"github.com/labeth/engineering-model-go/model"
)

const (
	sysMLProjectSourceName = "model.sysml"
	sysMLProjectPublisher  = "labeth"
	sysMLProjectVersion    = "1.0.0"
)

// SysMLRoundTripResult captures the normative KPAR path and reopened native
// SysML source used for source-integrity comparison.
type SysMLRoundTripResult struct {
	ProjectDir  string
	KPARPath    string
	ReopenedDir string
	SourcePath  string
	SourceText  string
}

// ExportSysMLV2Project creates a Sysand interchange project and builds its KPAR.
// Sysand, rather than a custom ZIP writer, owns all normative KPAR packaging.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-SYSML-V2-MODEL, REF-SYSML-V2-TOOLCHAIN
func ExportSysMLV2Project(architecturePath, projectDir, kparPath, sysandPath string) (SysMLExportResult, error) {
	if err := requireEmptyDirectory(projectDir); err != nil {
		return SysMLExportResult{}, err
	}
	sysand, err := resolveExternalTool(sysandPath, "sysand")
	if err != nil {
		return SysMLExportResult{}, err
	}
	result, err := GenerateSysMLV2FromFile(architecturePath)
	if err != nil {
		return result, err
	}
	if err := rejectLossyOrUnknownDiagnostics(result.Diagnostics); err != nil {
		return result, err
	}

	projectDir, err = filepath.Abs(projectDir)
	if err != nil {
		return result, fmt.Errorf("resolve project directory: %w", err)
	}
	kparPath, err = filepath.Abs(kparPath)
	if err != nil {
		return result, fmt.Errorf("resolve KPAR path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(projectDir), 0o755); err != nil {
		return result, fmt.Errorf("create project parent: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(kparPath), 0o755); err != nil {
		return result, fmt.Errorf("create KPAR parent: %w", err)
	}

	projectName := sysMLProjectName(result.Text)
	if err := runExternal("", sysand, "init", "--publisher", sysMLProjectPublisher, "--name", projectName, "--version", sysMLProjectVersion, projectDir); err != nil {
		return result, err
	}
	sourcePath := filepath.Join(projectDir, sysMLProjectSourceName)
	if err := os.WriteFile(sourcePath, []byte(result.Text), 0o644); err != nil {
		return result, fmt.Errorf("write SysML project source: %w", err)
	}
	if err := runExternal(projectDir, sysand, "include", "--compute-checksum", sysMLProjectSourceName); err != nil {
		return result, err
	}
	if err := runExternal(projectDir, sysand, "build", kparPath); err != nil {
		return result, err
	}
	return result, nil
}

// ReopenSysMLV2Project clones a KPAR with Sysand and returns its native SysML
// project source.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func ReopenSysMLV2Project(kparPath, reopenedDir, sysandPath string) (SysMLRoundTripResult, error) {
	if err := requireEmptyDirectory(reopenedDir); err != nil {
		return SysMLRoundTripResult{}, err
	}
	sysand, err := resolveExternalTool(sysandPath, "sysand")
	if err != nil {
		return SysMLRoundTripResult{}, err
	}
	kparPath, err = filepath.Abs(kparPath)
	if err != nil {
		return SysMLRoundTripResult{}, fmt.Errorf("resolve KPAR path: %w", err)
	}
	reopenedDir, err = filepath.Abs(reopenedDir)
	if err != nil {
		return SysMLRoundTripResult{}, fmt.Errorf("resolve reopen directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(reopenedDir), 0o755); err != nil {
		return SysMLRoundTripResult{}, fmt.Errorf("create reopen parent: %w", err)
	}
	if err := runExternal("", sysand, "clone", "--no-deps", "--path", kparPath, "--target", reopenedDir); err != nil {
		return SysMLRoundTripResult{}, err
	}
	sourcePath := filepath.Join(reopenedDir, sysMLProjectSourceName)
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return SysMLRoundTripResult{}, fmt.Errorf("read reopened SysML project source: %w", err)
	}
	return SysMLRoundTripResult{
		KPARPath:    kparPath,
		ReopenedDir: reopenedDir,
		SourcePath:  sourcePath,
		SourceText:  string(source),
	}, nil
}

// VerifySysMLV2RoundTrip compares the reopened KPAR source to a fresh native
// SysML projection of the authored Engineering Model.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func VerifySysMLV2RoundTrip(architecturePath, kparPath, reopenedDir, sysandPath string) (SysMLRoundTripResult, error) {
	expected, err := GenerateSysMLV2FromFile(architecturePath)
	if err != nil {
		return SysMLRoundTripResult{}, err
	}
	result, err := ReopenSysMLV2Project(kparPath, reopenedDir, sysandPath)
	if err != nil {
		return result, err
	}
	if result.SourceText != expected.Text {
		return result, errors.New("SysML KPAR source round trip differs from the native projection")
	}
	return result, nil
}

// ImportSysMLV2Project reconstructs the canonical semantic model from legacy
// projects that embedded an Engineering Model payload. Native projections do
// not embed non-SysML canonical data.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func ImportSysMLV2Project(sourcePath string) (model.SemanticModel, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return model.SemanticModel{}, fmt.Errorf("open SysML project source: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	inAnnotation := false
	isProject := false
	payload := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "@EngineeringProject {" {
			inAnnotation = true
			isProject = false
			payload = ""
			continue
		}
		if !inAnnotation {
			continue
		}
		if line == "}" {
			if isProject && payload != "" {
				var semantic model.SemanticModel
				if err := json.Unmarshal([]byte(payload), &semantic); err != nil {
					return model.SemanticModel{}, fmt.Errorf("decode EngineeringProject payload: %w", err)
				}
				return semantic, nil
			}
			inAnnotation = false
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSuffix(line, ";"), "=")
		if !ok {
			continue
		}
		decoded, err := strconv.Unquote(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		switch strings.TrimSpace(key) {
		case "conceptKind":
			isProject = decoded == "engineering_model"
		case "payload":
			payload = decoded
		}
	}
	if err := scanner.Err(); err != nil {
		return model.SemanticModel{}, fmt.Errorf("read SysML project source: %w", err)
	}
	return model.SemanticModel{}, errors.New("SysML project source has no EngineeringProject interchange payload")
}

// CompareSysMLV2RoundTrip reports differences in stable identity, ownership,
// relationships, expressions, references, and typed extensions.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func CompareSysMLV2RoundTrip(expected, actual model.SemanticModel) []string {
	expected = normalizeInterchangeModel(expected)
	actual = normalizeInterchangeModel(actual)
	var differences []string
	if expected.ID != actual.ID || expected.Title != actual.Title {
		differences = append(differences, "project identity")
	}
	if !reflect.DeepEqual(expected.Imports, actual.Imports) {
		differences = append(differences, "library/project references")
	}
	expectedElements := semanticElementsByID(expected.Elements)
	actualElements := semanticElementsByID(actual.Elements)
	if !reflect.DeepEqual(sortedSemanticElementKeys(expectedElements), sortedSemanticElementKeys(actualElements)) {
		differences = append(differences, "stable element identity")
	} else {
		for _, id := range sortedSemanticElementKeys(expectedElements) {
			want, got := expectedElements[id], actualElements[id]
			if want.Owner != got.Owner || want.Namespace != got.Namespace {
				differences = appendUniqueRoundTripDifference(differences, "ownership")
			}
			if !reflect.DeepEqual(want.Features, got.Features) {
				differences = appendUniqueRoundTripDifference(differences, "typed features and expressions")
			}
			if !reflect.DeepEqual(want.Metadata, got.Metadata) ||
				want.ExtensionNamespace != got.ExtensionNamespace ||
				want.Extension != got.Extension ||
				!reflect.DeepEqual(want.Targets, got.Targets) {
				differences = appendUniqueRoundTripDifference(differences, "typed extensions")
			}
			want.Metadata, got.Metadata = nil, nil
			want.Features, got.Features = nil, nil
			want.Owner, got.Owner = "", ""
			want.Namespace, got.Namespace = "", ""
			want.ExtensionNamespace, got.ExtensionNamespace = "", ""
			want.Extension, got.Extension = "", ""
			want.Targets, got.Targets = nil, nil
			if !reflect.DeepEqual(want, got) {
				differences = appendUniqueRoundTripDifference(differences, "element semantics")
			}
		}
	}
	if !reflect.DeepEqual(expected.Relationships, actual.Relationships) {
		differences = append(differences, "relationships and expressions")
	}
	return differences
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
		if diagnostic.Severity == model.SemanticSeverityError ||
			strings.Contains(diagnostic.Code, "lossy") ||
			strings.Contains(diagnostic.Code, "unsupported") ||
			strings.Contains(diagnostic.Code, "unknown") {
			blocking = append(blocking, diagnostic.Code)
		}
	}
	if len(blocking) > 0 {
		sort.Strings(blocking)
		return fmt.Errorf("SysML interchange blocked by diagnostics: %s", strings.Join(blocking, ", "))
	}
	return nil
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
	const prefix = "package '"
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			name := strings.TrimPrefix(line, prefix)
			if index := strings.Index(name, "'"); index > 0 {
				return name[:index]
			}
		}
	}
	return "engineering-model"
}
