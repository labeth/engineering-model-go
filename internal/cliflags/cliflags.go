// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION

// Package cliflags gives every engmod command the same --workspace and --lock
// flags. A flag that is given is exported as ENGMOD_WORK / ENGMOD_LOCK before
// the command calls the library, so the CLI tools and the MCP server share the
// library's environment-based workspace and lock selection.
package cliflags

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

const (
	// WorkspaceEnv selects the model workspace: off, a path to engmod.work.yml, or unset.
	WorkspaceEnv = "ENGMOD_WORK"
	// LockEnv selects the composition lock mode: check or write.
	LockEnv = "ENGMOD_LOCK"
)

// Options holds the parsed --workspace and --lock values.
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
type Options struct {
	Workspace string
	Lock      string
}

// Register adds --workspace and --lock to fs (flag.CommandLine when nil).
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func Register(fs *flag.FlagSet) *Options {
	if fs == nil {
		fs = flag.CommandLine
	}
	opts := &Options{}
	fs.StringVar(&opts.Workspace, "workspace", "", "model workspace: off ignores every "+model.WorkspaceFileName+", <path> uses that file only; default: $"+WorkspaceEnv+" or search parent directories")
	fs.StringVar(&opts.Lock, "lock", "", "composition lock mode: check compares with engmod.lock.yml and writes nothing, write updates it; default: $"+LockEnv+" or write")
	return opts
}

// Apply validates the flags and exports the given ones to the process environment.
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (o *Options) Apply() error {
	return o.apply(os.Setenv)
}

// MustApply is Apply for main functions: it prints the error and exits with status 2.
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (o *Options) MustApply() {
	if err := o.Apply(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}

// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (o *Options) apply(setenv func(key, value string) error) error {
	workspace, err := normalizeWorkspace(o.Workspace)
	if err != nil {
		return err
	}
	lock, err := normalizeLock(o.Lock)
	if err != nil {
		return err
	}
	if workspace != "" {
		if err := setenv(WorkspaceEnv, workspace); err != nil {
			return err
		}
	}
	if lock != "" {
		if err := setenv(LockEnv, lock); err != nil {
			return err
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-063
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func normalizeWorkspace(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.EqualFold(value, "off") {
		return "off", nil
	}
	if filepath.Base(value) != model.WorkspaceFileName {
		return "", fmt.Errorf("--workspace must be off or a path to %s, got %q", model.WorkspaceFileName, value)
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("--workspace %q: %w", value, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("--workspace %q: %w", value, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("--workspace %q is not a regular file", value)
	}
	return abs, nil
}

// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func normalizeLock(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "check", "write":
		return value, nil
	default:
		return "", fmt.Errorf("--lock must be check or write, got %q", value)
	}
}
