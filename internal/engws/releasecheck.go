// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// ReleaseCheckResult is the outcome of release-check: findings
// (dependency.unreleased_pin, contract.semver_too_small,
// release.schema_version_not_bumped), notes and the
// contract diff the semver gate used (nil when it was skipped).
// TRLC-LINKS: REQ-EMG-103
type ReleaseCheckResult struct {
	Findings []validate.Diagnostic
	Notes    []string
	Diff     *ContractDiffResult
}

// isRelease reports whether v is a release version: canonical semver
// without pre-release (which also excludes pseudo-versions) or build.
// TRLC-LINKS: REQ-EMG-103
func isRelease(v string) bool {
	return semver.IsValid(v) && semver.Canonical(v) == v && semver.Prerelease(v)+semver.Build(v) == ""
}

// bumpOf returns the impact a version step allows: a major bump allows a
// major change, a minor bump a minor one, a patch bump a patch. Below v1 a
// breaking change needs a minor bump (semver item 4: 0.y.z is initial
// development; the Go and CUE convention): a minor bump in v0.x allows a
// major change and a patch bump a minor one.
// TRLC-LINKS: REQ-EMG-103
func bumpOf(previous, tag string) (Impact, string) {
	name := "patch"
	switch {
	case semver.Major(tag) != semver.Major(previous):
		name = "major"
	case semver.MajorMinor(tag) != semver.MajorMinor(previous):
		name = "minor"
	}
	if semver.Major(previous) == "v0" && semver.Major(tag) == "v0" {
		switch name {
		case "minor":
			return ImpactMajor, name
		case "patch":
			return ImpactMinor, name
		}
	}
	return Impact(name), name
}

// unreleasedPins reports every dependency version in engmod.yml and
// engmod.lock.yml that is not a release tag.
// TRLC-LINKS: REQ-EMG-103
func unreleasedPins(repo string, manifest model.ManifestDocument) ([]validate.Diagnostic, error) {
	var out []validate.Diagnostic
	for _, d := range manifest.Dependencies {
		if v := strings.TrimSpace(d.Version); !isRelease(v) {
			out = append(out, unreleasedPinFinding(d.Alias, v, "engmod.yml"))
		}
	}
	lockPins, err := unreleasedLockPins(repo)
	if err != nil {
		return nil, err
	}
	return append(out, lockPins...), nil
}

// unreleasedPinFinding is the dependency.unreleased_pin finding for one dependency version.
// TRLC-LINKS: REQ-EMG-103
func unreleasedPinFinding(alias, version, file string) validate.Diagnostic {
	return validate.Diagnostic{
		Code: "dependency.unreleased_pin", Severity: validate.SeverityError, Path: file,
		Message: fmt.Sprintf("dependency %s pins %s (%s); a release pins release tags only: release the dependency and bump to its tag", alias, version, file),
	}
}

// unreleasedLockPins reports the dependency versions of engmod.lock.yml that are not release tags; no lock is none.
// TRLC-LINKS: REQ-EMG-103
func unreleasedLockPins(repo string) ([]validate.Diagnostic, error) {
	data, err := os.ReadFile(filepath.Join(repo, engmodel.CompositionLockFileName))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var lock engmodel.CompositionLockDocument
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("%s: %w", engmodel.CompositionLockFileName, err)
	}
	var out []validate.Diagnostic
	for _, d := range lock.Dependencies {
		if v := strings.TrimSpace(d.Version); !isRelease(v) {
			out = append(out, unreleasedPinFinding(d.Alias, v, engmodel.CompositionLockFileName))
		}
	}
	return out, nil
}

// releaseTags returns the release semver tags of the git repository of repo.
// TRLC-LINKS: REQ-EMG-103
func releaseTags(repo string, args ...string) ([]string, error) {
	out, err := gitOutput(repo, append([]string{"tag", "--list"}, args...)...)
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, line := range strings.Split(string(out), "\n") {
		if t := strings.TrimSpace(line); isRelease(t) {
			tags = append(tags, t)
		}
	}
	semver.Sort(tags)
	return tags, nil
}

// previousRelease returns the previous release tag of tag and whether tag
// exists. The previous release is the nearest release tag below the tag in the
// history of the commit being released (the tag's commit, else HEAD): a
// release on another branch is not this line's previous release. It is ""
// for a first release.
// TRLC-LINKS: REQ-EMG-103
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func previousRelease(repo, tag string) (string, bool, error) {
	all, err := releaseTags(repo)
	if err != nil {
		return "", false, err
	}
	exists := slices.Contains(all, tag)
	commit := "HEAD"
	if exists {
		commit = "refs/tags/" + tag
	}
	merged, err := releaseTags(repo, "--merged", commit)
	if err != nil {
		return "", false, err
	}
	previous := ""
	for _, t := range merged {
		if semver.Compare(t, tag) < 0 {
			previous = t
		}
	}
	return previous, exists, nil
}

// ReleaseCheck checks that the model in repo may be released under tag:
// no dependency pin is a pre-release or pseudo-version, and the tag's bump
// over the previous release tag (the highest release tag below it among the
// tags merged into the released commit, git tag --merged) covers the
// contract diff between them (to the tag if it exists, else HEAD).
// An empty tag means the highest release tag at HEAD; without one, or
// without a previous release tag, the semver gate is skipped with a note.
// A CHANGELOG entry of the tag that says Breaking release also needs a
// schemaVersion above the one of the previous release
// (release.schema_version_not_bumped, ADR-EMG-077).
// TRLC-LINKS: REQ-EMG-103, REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-RELEASE-CHECK
func ReleaseCheck(repo, tag string) (ReleaseCheckResult, error) {
	var result ReleaseCheckResult
	bundle, err := model.LoadBundle(filepath.Join(repo, "engmod.yml"))
	if err != nil {
		return result, err
	}
	pins, err := unreleasedPins(repo, bundle.Manifest)
	if err != nil {
		return result, err
	}
	result.Findings = append(result.Findings, pins...)

	tag, done, err := resolveReleaseTag(repo, tag, &result)
	if err != nil || done {
		return result, err
	}
	return result, checkReleaseContract(repo, tag, &result)
}

// resolveReleaseTag validates the tag and finds the tag to check: the given one, else the highest release tag at
// HEAD. done is true when the semver gate is skipped (with a note in result) or failed.
// TRLC-LINKS: REQ-EMG-103
func resolveReleaseTag(repo, tag string, result *ReleaseCheckResult) (resolved string, done bool, err error) {
	if tag != "" && !isRelease(tag) {
		return tag, true, fmt.Errorf("--tag %q is not a release version such as v1.2.3", tag)
	}
	if skip, err := requireGitRepository(repo, tag, result); err != nil || skip {
		return tag, true, err
	}
	if tag != "" {
		return tag, false, nil
	}
	return headReleaseTag(repo, result)
}

// requireGitRepository refuses a missing git repository when a tag is given; without a tag it notes that the gate is
// skipped.
// TRLC-LINKS: REQ-EMG-103
func requireGitRepository(repo, tag string, result *ReleaseCheckResult) (skip bool, err error) {
	if _, err := gitOutput(repo, "rev-parse", "--git-dir"); err != nil {
		if tag != "" {
			return true, fmt.Errorf("%s is not in a git repository: %w", repo, err)
		}
		result.Notes = append(result.Notes, "semver gate skipped: not a git repository")
		return true, nil
	}
	return false, nil
}

// headReleaseTag is the highest release tag at HEAD; without one the gate is skipped with a note.
// TRLC-LINKS: REQ-EMG-103
func headReleaseTag(repo string, result *ReleaseCheckResult) (tag string, done bool, err error) {
	atHead, err := releaseTags(repo, "--points-at", "HEAD")
	if err != nil {
		return "", true, err
	}
	if len(atHead) == 0 {
		result.Notes = append(result.Notes, "semver gate skipped: no --tag and no release tag at HEAD")
		return "", true, nil
	}
	return atHead[len(atHead)-1], false, nil
}

// releaseStep is one release: the tag over the previous release, the ref its contract diff runs to and the impact
// found.
// TRLC-LINKS: REQ-EMG-103
type releaseStep struct {
	previous string
	to       string
	tag      string
	bump     string
	impact   Impact
}

// checkReleaseContract diffs the contract from the previous release to the tag (else HEAD) and adds the schema
// version and semver findings; without a previous release the gate is skipped with a note.
// TRLC-LINKS: REQ-EMG-103
func checkReleaseContract(repo, tag string, result *ReleaseCheckResult) error {
	previous, exists, err := previousRelease(repo, tag)
	if err != nil {
		return err
	}
	if previous == "" {
		result.Notes = append(result.Notes, fmt.Sprintf("semver gate skipped: no release tag below %s (first release)", tag))
		return nil
	}
	notBumped, err := schemaVersionFinding(repo, tag, previous)
	if err != nil {
		return err
	}
	result.Findings = append(result.Findings, notBumped...)
	step := releaseStep{previous: previous, to: "HEAD", tag: tag}
	if exists {
		step.to = tag
	}
	diff, err := ContractDiff(repo, previous, step.to)
	if err != nil {
		return err
	}
	result.Diff = &diff
	allowed, bump := bumpOf(previous, tag)
	step.bump, step.impact = bump, diff.Impact
	result.Notes = append(result.Notes, step.note())
	if diff.Impact.rank() > allowed.rank() {
		result.Findings = append(result.Findings, step.tooSmallFinding())
	}
	return nil
}

// note describes the contract impact of the step against the bump of its tag.
// TRLC-LINKS: REQ-EMG-103
func (r releaseStep) note() string {
	note := fmt.Sprintf("contract %s..%s: impact %s, tag %s is a %s bump", r.previous, r.to, r.impact, r.tag, r.bump)
	if semver.Major(r.previous) == "v0" && semver.Major(r.tag) == "v0" {
		note += " (v0.x: a breaking change needs a minor bump, any other change a patch bump)"
	}
	return note
}

// tooSmallFinding is the contract.semver_too_small finding of the step.
// TRLC-LINKS: REQ-EMG-103
func (r releaseStep) tooSmallFinding() validate.Diagnostic {
	return validate.Diagnostic{
		Code: "contract.semver_too_small", Severity: validate.SeverityError, Path: "engmod.yml",
		Message: fmt.Sprintf("%s is a %s bump over %s but the contract change is %s; tag a larger version", r.tag, r.bump, r.previous, r.impact),
	}
}

// printFinding prints one finding and, when it has a remedy, its fix line.
// TRLC-LINKS: REQ-EMG-103, REQ-EMG-186
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func printFinding(w io.Writer, d validate.Diagnostic) {
	fmt.Fprintf(w, "%s [%s] %s\n", d.Code, d.Severity, d.Message)
	if fix := d.FixLine(); fix != "" {
		fmt.Fprintln(w, fix)
	}
}

// runReleaseCheck prints the notes, the findings and, when the gate failed,
// the contract diff. Exit 0 ok, 1 findings, 2 usage or IO error.
// TRLC-LINKS: REQ-EMG-103
// ENGMODEL-LINKS: EVT-RELEASE-CHECK-REQUESTED
func runReleaseCheck(repo, tag string, w io.Writer) (int, error) {
	result, err := ReleaseCheck(repo, tag)
	if err != nil {
		return 2, err
	}
	for _, note := range result.Notes {
		fmt.Fprintln(w, "engws: "+note)
	}
	printReleaseFindings(w, result)
	if len(result.Findings) > 0 {
		fmt.Fprintf(w, "engws: release-check failed: %d finding(s)\n", len(result.Findings))
		return 1, nil
	}
	fmt.Fprintln(w, "engws: release-check ok")
	return 0, nil
}

// printReleaseFindings prints every finding and, when the semver gate failed, the contract diff.
// TRLC-LINKS: REQ-EMG-103
func printReleaseFindings(w io.Writer, result ReleaseCheckResult) {
	semverFailed := false
	for _, d := range result.Findings {
		printFinding(w, d)
		semverFailed = semverFailed || d.Code == "contract.semver_too_small"
	}
	if semverFailed && result.Diff != nil {
		printContractDiff(w, *result.Diff)
	}
}
