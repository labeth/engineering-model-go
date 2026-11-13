// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// releaseVersions are the schema versions of the manifest at the first release tag and at HEAD.
type releaseVersions struct{ tagged, head int }

// releaseRepo builds a git repository whose release tag v0.1.0 holds a model
// with the manifest schemaVersion tagged and whose head commit holds the same
// model with the manifest schemaVersion head and the CHANGELOG.md given (no
// file when empty). The documents of the previous release are written for
// the current version and only the manifest differs, which is all the rule reads.
func releaseRepo(t *testing.T, v releaseVersions, changelog string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "rc")
	writeModel(t, dir, "rc", "v0.1.0")
	setManifestVersion(t, dir, v.tagged)
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "release 0.1.0")
	runGit(t, dir, "tag", "v0.1.0")
	setManifestVersion(t, dir, v.head)
	if changelog != "" {
		write(t, filepath.Join(dir, "CHANGELOG.md"), changelog)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", "release 0.2.0")
	return dir
}

// setManifestVersion rewrites the schemaVersion line of the manifest of dir.
func setManifestVersion(t *testing.T, dir string, version int) {
	t.Helper()
	path := filepath.Join(dir, "engmod.yml")
	data := read(t, path)
	lines := strings.SplitN(data, "\n", 2)
	write(t, path, "schemaVersion: "+strconv.Itoa(version)+"\n"+lines[1])
}

const (
	changelogBreaking = "# Changelog\n\n## [Unreleased]\n\n## [0.2.0] - 2026-11-13: Things\n\n**Breaking release, no compatibility path.**\n\n## [0.1.0] - 2026-11-01\n\nFirst.\n"
	changelogPlain    = "# Changelog\n\n## [0.2.0] - 2026-11-13: Things\n\nAdded a thing.\n\n## [0.1.0] - 2026-11-01\n\nFirst.\n"
	// The words are in the entry of the previous release only.
	changelogOtherEntry = "# Changelog\n\n## [0.2.0] - 2026-11-13: Things\n\nAdded a thing.\n\n## [0.1.0] - 2026-11-01\n\nA breaking release.\n"
)

// release-check reports release.schema_version_not_bumped (untiered error,
// exit 1, with its remedy) when the CHANGELOG entry of the tag has the words
// Breaking release (any case) and the manifest schemaVersion at the head is
// not above the one at the previous release tag; it reports nothing for a
// greater version, an entry without the words, the words in another entry,
// no CHANGELOG, or no previous tag. The version at the previous tag is read
// as written: 2 is no longer loadable, and the tagged model still loads for
// the contract diff.
// Mutation: dropping the schemaVersionFinding call in ReleaseCheck, comparing
// with >= instead of >, matching the words case-sensitively, reading the
// section of another entry or the whole CHANGELOG, or reporting without a
// previous tag fails a row.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, TR-LINK-RELEASE-CHECK
func TestReleaseCheckSchemaVersionNotBumped(t *testing.T) {
	const current = model.CurrentSchemaVersion
	for _, tc := range []struct {
		name         string
		tagged, head int
		changelog    string
		tag          string
		wantCode     int
	}{
		{"breaking entry, same version", current, current, changelogBreaking, "v0.2.0", 1},
		{"breaking entry, greater version", current - 1, current, changelogBreaking, "v0.2.0", 0},
		{"breaking entry in lower case words", current, current, strings.Replace(changelogBreaking, "Breaking release", "BREAKING RELEASE", 1), "v0.2.0", 1},
		{"entry without the words", current, current, changelogPlain, "v0.2.0", 0},
		{"words in another entry only", current, current, changelogOtherEntry, "v0.2.0", 0},
		{"no CHANGELOG", current, current, "", "v0.2.0", 0},
		{"no entry for the tag", current, current, changelogBreaking, "v0.3.0", 0},
		{"no previous tag", current, current, changelogBreaking, "v0.1.0", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := releaseRepo(t, releaseVersions{tc.tagged, tc.head}, tc.changelog)
			code, out, errOut := runMain(t, fakeRegistry{}, "release-check", "--repo", dir, "--tag", tc.tag)
			if code != tc.wantCode {
				t.Fatalf("exit %d, want %d\n%s\n%s", code, tc.wantCode, out, errOut)
			}
			reported := strings.Contains(out, "release.schema_version_not_bumped [error]")
			if reported != (tc.wantCode == 1) {
				t.Fatalf("reported = %v\n%s", reported, out)
			}
			if reported && !strings.Contains(out, "fix: raise schemaVersion and model.CurrentSchemaVersion, then record the change in the CHANGELOG") {
				t.Fatalf("no remedy line:\n%s", out)
			}
		})
	}
}

// The comparison of two manifests: the head must be above the previous
// release, and a manifest that is not YAML is an error that names the ref.
// Mutation: comparing with >= instead of >, or reading the head as the
// previous version, fails a row.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestCompareSchemaVersions(t *testing.T) {
	for _, tc := range compareCases {
		t.Run(tc.name, func(t *testing.T) { assertCompare(t, tc) })
	}
}

// compareCase is one pair of manifests and what the comparison must report.
type compareCase struct {
	name      string
	was, head string
	finding   bool
	err       string
}

var compareCases = []compareCase{
	{"same", "schemaVersion: 3\n", "schemaVersion: 3\n", true, ""},
	{"lower", "schemaVersion: 3\n", "schemaVersion: 2\n", true, ""},
	{"greater", "schemaVersion: 2\n", "schemaVersion: 3\n", false, ""},
	{"previous without a version", "module: {}\n", "schemaVersion: 3\n", false, ""},
	{"previous is not YAML", "a: [\n", "schemaVersion: 3\n", false, "engmod.yml at v0.1.0"},
}

// assertCompare runs compareSchemaVersions on one case and checks error, finding and remedy.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func assertCompare(t *testing.T, tc compareCase) {
	t.Helper()
	got, err := compareSchemaVersions("v0.2.0", "v0.1.0", [2][]byte{[]byte(tc.was), []byte(tc.head)})
	if (err != nil) != (tc.err != "") || (err != nil && !strings.Contains(err.Error(), tc.err)) {
		t.Fatalf("err = %v, want %q", err, tc.err)
	}
	if (len(got) == 1) != tc.finding {
		t.Fatalf("findings = %+v", got)
	}
	if tc.finding {
		assertSchemaFinding(t, got[0])
	}
}

// assertSchemaFinding checks the code, severity and remedy of the not-bumped finding.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func assertSchemaFinding(t *testing.T, d validate.Diagnostic) {
	t.Helper()
	if d.Code != "release.schema_version_not_bumped" || d.Severity != "error" || d.Remedy == "" {
		t.Fatalf("finding = %+v", d)
	}
}

// The section of an entry runs from its header to the next entry header; the
// words are looked for in that section only.
// Mutation: cutting at the end of the file, or matching a header prefix
// (0.2.0 in 0.2.0-rc.1), fails a row.
// TRLC-LINKS: REQ-EMG-186
func TestChangelogSection(t *testing.T) {
	log := "## [0.2.0] - d\nbody two\n## [0.2.0-rc.1] - d\nrc body\n## [0.1.0] - d\nbody one\n"
	for tag, want := range map[string]string{
		"v0.2.0":      " - d\nbody two",
		"v0.2.0-rc.1": " - d\nrc body",
		"v0.1.0":      " - d\nbody one\n",
		"v0.9.0":      "",
	} {
		if got := changelogSection(log, tag); got != want {
			t.Errorf("%s: section %q, want %q", tag, got, want)
		}
	}
}

// LoadBundleAtRef loads a model committed with an older schemaVersion in the
// manifest and every document (the loader refuses it in the working tree) and
// returns the bundle with the current version, so contract-diff and
// release-check read the previous release.
// Mutation: dropping the stampCurrentVersion call for the manifest or for the
// documents fails with an unsupported schemaVersion error.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-MODEL-LOADER
func TestLoadBundleAtRefAcceptsAnOlderSchemaVersion(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "old")
	writeModel(t, dir, "old", "v0.1.0")
	old := "schemaVersion: " + strconv.Itoa(model.CurrentSchemaVersion-1)
	for _, rel := range []string{"engmod.yml", "model/catalog.yml", "model/requirements.yml", "model/architecture.yml", "model/behavior.yml", "model/assurance.yml", "model/compliance.yml", "model/views.yml", "model/decisions.yml"} {
		replaceIn(t, filepath.Join(dir, rel), model.SchemaVersionLine, old)
	}
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "old model")
	runGit(t, dir, "tag", "v0.1.0")

	if _, err := model.LoadBundle(filepath.Join(dir, "engmod.yml")); err == nil || !strings.Contains(err.Error(), "unsupported schemaVersion") {
		t.Fatalf("the working tree must still be refused, got %v", err)
	}
	bundle, err := LoadBundleAtRef(dir, "v0.1.0")
	if err != nil {
		t.Fatalf("a tag with an older schemaVersion must load for the contract: %v", err)
	}
	if bundle.Manifest.SchemaVersion != model.CurrentSchemaVersion || bundle.Architecture.SchemaVersion != model.CurrentSchemaVersion {
		t.Fatalf("versions %d, %d", bundle.Manifest.SchemaVersion, bundle.Architecture.SchemaVersion)
	}
}

// stampCurrentVersion rewrites only the top-level schemaVersion line: other
// bytes, comments and nested keys are kept, and a document without the line
// is unchanged.
// Mutation: matching an indented key, replacing every line, or dropping the
// trailing comment fails a row.
// TRLC-LINKS: REQ-EMG-186
func TestStampCurrentVersion(t *testing.T) {
	cur := model.SchemaVersionLine
	for _, tc := range []struct{ name, in, want string }{
		{"older version", "schemaVersion: 2\nx: 1\n", cur + "\nx: 1\n"},
		{"zero", "schemaVersion: 0\n", cur + "\n"},
		{"current version", cur + "\nx: 1\n", cur + "\nx: 1\n"},
		{"leading comment and trailing comment", "# header\nschemaVersion: 2 # old\nx: 1\n", "# header\n" + cur + " # old\nx: 1\n"},
		{"nested key is kept", "x:\n  schemaVersion: 2\n", "x:\n  schemaVersion: 2\n"},
		{"first line only", "schemaVersion: 2\nschemaVersion: 2\n", cur + "\nschemaVersion: 2\n"},
		{"no line", "x: 1\n", "x: 1\n"},
		{"carriage returns", "schemaVersion: 2\r\nx: 1\r\n", cur + "\r\nx: 1\r\n"},
	} {
		if got := string(stampCurrentVersion([]byte(tc.in))); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}
