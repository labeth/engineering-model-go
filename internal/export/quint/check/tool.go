// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/internal/export/exportutil"
)

const (
	// QuintEnv names the environment variable that holds the path of the quint
	// executable.
	QuintEnv = "ENGMOD_QUINT"
	// QuintHomeEnv names the environment variable that holds the directory
	// Quint looks for a pre-provisioned Apalache distribution in.
	QuintHomeEnv = "QUINT_HOME"
	// ApalacheVersion is the Apalache release the toolchain pins.
	ApalacheVersion = "0.62.4"
	// MinJavaMajor is the oldest Java that runs the pinned Apalache.
	MinJavaMajor = 21

	installHint    = "install with scripts/install-quint-toolchain.sh"
	commandTimeout = 10 * time.Minute
)

// SetupError is a problem with the environment (a missing tool, an unreadable
// model) rather than with the model's behavior. A setup error never counts as
// a pass; the CLI exits 2 for it.
type SetupError struct {
	Message string
	Err     error
}

// Error implements error.
//
// TRLC-LINKS: REQ-EMG-142
func (e *SetupError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error.
//
// TRLC-LINKS: REQ-EMG-142
func (e *SetupError) Unwrap() error { return e.Err }

// setupError builds a SetupError.
//
// TRLC-LINKS: REQ-EMG-142
func setupError(err error, format string, args ...any) *SetupError {
	return &SetupError{Message: fmt.Sprintf(format, args...), Err: err}
}

// IsSetupError reports whether err is, or wraps, a SetupError.
//
// TRLC-LINKS: REQ-EMG-142
func IsSetupError(err error) bool {
	var s *SetupError
	return errors.As(err, &s)
}

// Command is one external process invocation.
type Command struct {
	Path string
	Args []string
	// Dir is the working directory.
	Dir string
	// Env is the full environment of the process.
	Env []string
}

// Output is what a finished process produced. ExitCode is -1 when the process
// was killed (for example by the timeout).
type Output struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner runs a Command. It returns an error only when the process could not
// be started; a non-zero exit is reported in Output. Tests inject a fake.
type Runner func(ctx context.Context, cmd Command) (Output, error)

// ExecRunner is the Runner that starts a real process.
//
// TRLC-LINKS: REQ-EMG-143
func ExecRunner(ctx context.Context, cmd Command) (Output, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	proc := exec.CommandContext(ctx, cmd.Path, cmd.Args...)
	proc.Dir = cmd.Dir
	proc.Env = cmd.Env
	var stdout, stderr bytes.Buffer
	proc.Stdout, proc.Stderr = &stdout, &stderr
	err := proc.Run()
	out := Output{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return out, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		out.ExitCode = exit.ExitCode()
		if ctx.Err() != nil {
			out.Stderr += "\ntimed out: " + ctx.Err().Error()
		}
		return out, nil
	}
	return out, err
}

// resolveQuint finds the quint executable: the configured path, else
// ENGMOD_QUINT, else quint on PATH.
//
// TRLC-LINKS: REQ-EMG-142
func resolveQuint(configured string) (string, error) {
	if strings.TrimSpace(configured) == "" {
		configured = os.Getenv(QuintEnv)
	}
	path, err := exportutil.ResolveExternalTool(configured, "quint", installHint)
	if err != nil {
		return "", setupError(err, "quint is not available (set %s or pass --quint)", QuintEnv)
	}
	return path, nil
}

// apalacheMC returns the path the pinned Apalache launcher must have under
// quintHome; Quint downloads Apalache when it is missing, so verify mode
// checks for it first.
//
// TRLC-LINKS: REQ-EMG-144
func apalacheMC(quintHome string) string {
	return filepath.Join(quintHome, "apalache-dist-"+ApalacheVersion, "apalache", "bin", "apalache-mc")
}

var javaVersionRe = regexp.MustCompile(`version "([^"]+)"`)

// javaMajor extracts the major version from the output of java -version:
// "21.0.13" is 21, the legacy "1.8.0_402" is 8.
//
// TRLC-LINKS: REQ-EMG-144
func javaMajor(output string) (int, error) {
	m := javaVersionRe.FindStringSubmatch(output)
	if m == nil {
		return 0, fmt.Errorf("no java version in %q", firstLine(output))
	}
	parts := strings.FieldsFunc(m[1], func(r rune) bool { return r == '.' || r == '_' || r == '-' || r == '+' })
	if len(parts) == 0 {
		return 0, fmt.Errorf("unparsable java version %q", m[1])
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("unparsable java version %q", m[1])
	}
	if major == 1 && len(parts) > 1 {
		if major, err = strconv.Atoi(parts[1]); err != nil {
			return 0, fmt.Errorf("unparsable java version %q", m[1])
		}
	}
	return major, nil
}

// resolveJava finds a java executable (PATH first, then JAVA_HOME) and checks
// that it is Java 21 or later. It returns the directory that holds it.
//
// TRLC-LINKS: REQ-EMG-144
func resolveJava(ctx context.Context, run Runner) (string, error) {
	path, err := exec.LookPath("java")
	if err != nil {
		home := os.Getenv("JAVA_HOME")
		if home == "" {
			return "", setupError(err, "java %d or later is required for verify mode (put it on PATH)", MinJavaMajor)
		}
		path = filepath.Join(home, "bin", "java")
		if _, statErr := os.Stat(path); statErr != nil {
			return "", setupError(statErr, "java %d or later is required for verify mode (put it on PATH)", MinJavaMajor)
		}
	}
	out, err := run(ctx, Command{Path: path, Args: []string{"-version"}, Env: os.Environ()})
	if err != nil {
		return "", setupError(err, "cannot run %s -version", path)
	}
	major, err := javaMajor(out.Stderr + "\n" + out.Stdout)
	if err != nil {
		return "", setupError(err, "cannot read the version of %s", path)
	}
	if major < MinJavaMajor {
		return "", setupError(nil, "java %d or later is required for verify mode, found java %d at %s", MinJavaMajor, major, path)
	}
	return filepath.Dir(path), nil
}

// freePort allocates a free TCP port on the loopback interface for the
// Apalache server, so parallel or stale servers never collide.
//
// TRLC-LINKS: REQ-EMG-144
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

var proxyVars = []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"}

// verifyEnv builds the environment of a quint verify call: the pinned
// QUINT_HOME, a scratch TMPDIR, the chosen Java first on PATH and, when any
// proxy is configured, NO_PROXY and no_proxy that keep localhost direct so the
// Apalache server is reachable. Existing values are appended to, never
// replaced by an empty string.
//
// TRLC-LINKS: REQ-EMG-144
func verifyEnv(base []string, quintHome, tmpDir, javaDir string) []string {
	env := map[string]string{}
	var order []string
	set := func(k, v string) {
		if _, seen := env[k]; !seen {
			order = append(order, k)
		}
		env[k] = v
	}
	for _, kv := range base {
		if k, v, ok := strings.Cut(kv, "="); ok {
			set(k, v)
		}
	}
	set(QuintHomeEnv, quintHome)
	set("TMPDIR", tmpDir)
	if javaDir != "" {
		set("PATH", javaDir+string(os.PathListSeparator)+env["PATH"])
	}
	proxied := false
	for _, k := range proxyVars {
		if env[k] != "" {
			proxied = true
		}
	}
	if proxied {
		for _, k := range []string{"NO_PROXY", "no_proxy"} {
			set(k, appendNoProxy(env[k]))
		}
	}
	out := make([]string, 0, len(order))
	for _, k := range order {
		out = append(out, k+"="+env[k])
	}
	return out
}

// appendNoProxy adds localhost and 127.0.0.1 to a NO_PROXY list.
//
// TRLC-LINKS: REQ-EMG-144
func appendNoProxy(current string) string {
	have := map[string]bool{}
	var items []string
	for _, item := range strings.Split(current, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		have[item] = true
		items = append(items, item)
	}
	for _, item := range []string{"localhost", "127.0.0.1"} {
		if !have[item] {
			items = append(items, item)
		}
	}
	return strings.Join(items, ",")
}

// firstLine returns the first non-empty line of s.
//
// TRLC-LINKS: REQ-EMG-143
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}
