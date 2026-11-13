// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package exportutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveExternalTool returns the absolute path of an external executable.
// A non-empty configured path must name an existing regular file; otherwise
// name is looked up on PATH. installHint is appended to the PATH lookup error
// so the message tells the user how to provision the pinned tool.
//
// TODO: internal/export/sysml/sysml_interchange.go keeps its own copy of this
// logic (resolveExternalTool); switch it to this function when that package is
// next edited.
//
// TRLC-LINKS: REQ-EMG-142
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func ResolveExternalTool(configured, name, installHint string) (string, error) {
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
		hint := strings.TrimSpace(installHint)
		if hint != "" {
			return "", fmt.Errorf("%s is required; %s: %w", name, hint, err)
		}
		return "", fmt.Errorf("%s is required: %w", name, err)
	}
	return path, nil
}
