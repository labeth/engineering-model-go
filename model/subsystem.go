// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"fmt"
	"strings"

	"cuelang.org/go/mod/module"
	"golang.org/x/mod/semver"
)

// ParseExactModuleVersion validates an explicit CUE module path and canonical
// semantic version.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
func ParseExactModuleVersion(modulePath, version string) (module.Version, error) {
	if modulePath == "" || !strings.Contains(modulePath, "@v") {
		return module.Version{}, fmt.Errorf("module path %q must include an explicit major version suffix", modulePath)
	}
	if !semver.IsValid(version) || semver.Canonical(version) != version {
		return module.Version{}, fmt.Errorf("module version %q must be a canonical semantic version", version)
	}
	mv, err := module.NewVersion(modulePath, version)
	if err != nil {
		return module.Version{}, err
	}
	return mv, nil
}

// QualifyReference creates the only supported cross-module identifier form.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func QualifyReference(alias, id string) (string, error) {
	alias = strings.TrimSpace(alias)
	id = strings.TrimSpace(id)
	if alias == "" || strings.Contains(alias, "::") {
		return "", fmt.Errorf("dependency alias %q is invalid", alias)
	}
	if id == "" || strings.Contains(id, "::") {
		return "", fmt.Errorf("published identifier %q is invalid", id)
	}
	return alias + "::" + id, nil
}

// ParseQualifiedReference splits alias::ID and rejects legacy slash-qualified
// and malformed references.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func ParseQualifiedReference(ref string) (alias, id string, err error) {
	ref = strings.TrimSpace(ref)
	if strings.Count(ref, "::") != 1 {
		return "", "", fmt.Errorf("cross-module reference %q must use alias::ID", ref)
	}
	parts := strings.SplitN(ref, "::", 2)
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("cross-module reference %q must use alias::ID", ref)
	}
	return parts[0], parts[1], nil
}
