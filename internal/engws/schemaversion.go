// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
	"gopkg.in/yaml.v3"
)

// codeSchemaVersionNotBumped is the untiered error of release-check for a
// breaking release that keeps the schema version of the previous release.
const codeSchemaVersionNotBumped = "release.schema_version_not_bumped"

// breakingReleaseWords mark a CHANGELOG entry as a breaking release, which
// raises the model schema version (ADR-EMG-077).
const breakingReleaseWords = "breaking release"

// changelogSection returns the text of the "## [X.Y.Z]" section of changelog
// for the tag (without its v), up to the next "## [" header, and "" when the
// changelog has no entry for the tag.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func changelogSection(changelog, tag string) string {
	header := "\n## [" + strings.TrimPrefix(tag, "v") + "]"
	_, rest, found := strings.Cut("\n"+changelog, header)
	if !found {
		return ""
	}
	section, _, _ := strings.Cut(rest, "\n## [")
	return section
}

// isBreakingRelease reports whether the CHANGELOG.md of repo has an entry for
// the tag that contains the words Breaking release (any case). A repository
// without a CHANGELOG, or without an entry for the tag, has no breaking
// release to check.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func isBreakingRelease(repo, tag string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(repo, "CHANGELOG.md"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	section := strings.ToLower(changelogSection(string(data), tag))
	return strings.Contains(section, breakingReleaseWords), nil
}

// rawSchemaVersion reads the top-level schemaVersion of a manifest as the
// number written, not through the model loader: the loader refuses a model
// version it does not support, and this is how a release is compared with the
// version of an earlier release.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func rawSchemaVersion(manifest []byte) (int, error) {
	var head struct {
		SchemaVersion int `yaml:"schemaVersion"`
	}
	if err := yaml.Unmarshal(manifest, &head); err != nil {
		return 0, fmt.Errorf("engmod.yml: %w", err)
	}
	return head.SchemaVersion, nil
}

// schemaVersionFinding reports release.schema_version_not_bumped when the
// CHANGELOG entry of the tag is a breaking release and the manifest of the
// working tree has a schemaVersion no higher than the manifest at the
// previous release tag. It reports nothing for a release without a previous
// tag (the caller passes ""), without a CHANGELOG entry or without the words.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func schemaVersionFinding(repo, tag, previous string) ([]validate.Diagnostic, error) {
	breaking, err := isBreakingRelease(repo, tag)
	if err != nil || !breaking || previous == "" {
		return nil, err
	}
	before, err := gitOutput(repo, "show", previous+":./engmod.yml")
	if err != nil {
		return nil, err
	}
	head, err := os.ReadFile(filepath.Join(repo, "engmod.yml"))
	if err != nil {
		return nil, err
	}
	return compareSchemaVersions(tag, previous, [2][]byte{before, head})
}

// compareSchemaVersions returns the finding when the head manifest (the
// second) carries a schemaVersion no higher than the manifest of the previous
// release (the first).
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func compareSchemaVersions(tag, previous string, manifests [2][]byte) ([]validate.Diagnostic, error) {
	was, err := rawSchemaVersion(manifests[0])
	if err != nil {
		return nil, fmt.Errorf("engmod.yml at %s: %w", previous, err)
	}
	now, err := rawSchemaVersion(manifests[1])
	if err != nil || now > was {
		return nil, err
	}
	return []validate.Diagnostic{{
		Code: codeSchemaVersionNotBumped, Severity: validate.SeverityError, Path: "engmod.yml",
		Message: fmt.Sprintf("the CHANGELOG entry of %s is a Breaking release but schemaVersion is %d, not above %d at %s (model.CurrentSchemaVersion is %d)", tag, now, was, previous, model.CurrentSchemaVersion),
		Remedy:  validate.ConsistencyRemedy(codeSchemaVersionNotBumped),
	}}, nil
}
