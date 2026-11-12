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
	"runtime"
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

	installHint = "install with scripts/install-quint-toolchain.sh"

	// DefaultCallTimeout is the longest one Quint, Apalache or Java call may
	// run before its whole process group is killed.
	DefaultCallTimeout = 10 * time.Minute
	// DefaultTimeout is the longest a whole check may run (all calls together).
	DefaultTimeout = 30 * time.Minute
	// DefaultMaxOutput is the most bytes captured from each of the stdout and
	// stderr of one call; the rest is read and dropped so the process never
	// blocks on a full pipe.
	DefaultMaxOutput = 8 << 20
)

// waitDelay bounds the wait for a process that has exited or been killed but
// left a child holding its output pipes open. It is a variable so tests can
// shorten it.
var waitDelay = 5 * time.Second

// ErrTimeout is returned (wrapped) by Check when the overall deadline
// (CheckOptions.Timeout) expires. It is not a setup error and not a pass.
var ErrTimeout = errors.New("check timed out")

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
	// Env is the full environment of the process. A nil Env is an empty
	// environment, never the environment of the caller.
	Env []string
	// Timeout kills the process group after this long; zero selects
	// DefaultCallTimeout.
	Timeout time.Duration
	// MaxOutput bounds the captured bytes of each of stdout and stderr; zero
	// selects DefaultMaxOutput.
	MaxOutput int
}

// Output is what a finished process produced. ExitCode is -1 when the process
// was killed (for example by the timeout).
type Output struct {
	Stdout   string
	Stderr   string
	ExitCode int
	// TimedOut reports that the process was killed because its time limit (or
	// the overall deadline) expired.
	TimedOut bool
	// Truncated reports that stdout or stderr exceeded the capture limit and
	// the rest was dropped.
	Truncated bool
}

// Runner runs a Command. It returns an error only when the process could not
// be started; a non-zero exit is reported in Output. Tests inject a fake.
type Runner func(ctx context.Context, cmd Command) (Output, error)

// cappedBuffer collects at most max bytes and silently drops the rest, so a
// process that prints without limit neither exhausts memory nor blocks.
type cappedBuffer struct {
	buf       bytes.Buffer
	max       int
	truncated bool
}

// Write implements io.Writer; it always reports the whole write as done.
//
// TRLC-LINKS: REQ-EMG-143
func (c *cappedBuffer) Write(p []byte) (int, error) {
	if room := c.max - c.buf.Len(); room > 0 {
		if len(p) <= room {
			c.buf.Write(p)
			return len(p), nil
		}
		c.buf.Write(p[:room])
	}
	c.truncated = true
	return len(p), nil
}

// ExecRunner is the Runner that starts a real process. The process runs in
// its own process group (where the platform has one), is killed together with
// every child when its time limit or the context expires, and its captured
// output is capped.
//
// TRLC-LINKS: REQ-EMG-143
func ExecRunner(ctx context.Context, cmd Command) (Output, error) {
	limit := cmd.Timeout
	if limit <= 0 {
		limit = DefaultCallTimeout
	}
	capacity := cmd.MaxOutput
	if capacity <= 0 {
		capacity = DefaultMaxOutput
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	proc := exec.CommandContext(ctx, cmd.Path, cmd.Args...)
	proc.Dir = cmd.Dir
	proc.Env = cmd.Env
	if proc.Env == nil {
		proc.Env = []string{}
	}
	stdout, stderr := &cappedBuffer{max: capacity}, &cappedBuffer{max: capacity}
	proc.Stdout, proc.Stderr = stdout, stderr
	proc.WaitDelay = waitDelay
	isolateProcess(proc)
	err := proc.Run()
	if proc.Process != nil {
		// Whatever the verdict, no child of the tool outlives the call.
		killGroup(proc)
	}
	out := Output{Stdout: stdout.buf.String(), Stderr: stderr.buf.String(), Truncated: stdout.truncated || stderr.truncated}
	if proc.Process == nil {
		return out, err
	}
	var exit *exec.ExitError
	switch {
	case err == nil:
		return out, nil
	case ctx.Err() != nil:
		out.ExitCode = -1
		out.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		out.Stderr += "\nkilled: " + ctx.Err().Error()
		return out, nil
	case errors.As(err, &exit):
		out.ExitCode = exit.ExitCode()
		return out, nil
	case errors.Is(err, exec.ErrWaitDelay):
		// The process succeeded but a child kept the pipes open.
		out.Stderr += "\nwarning: a child process kept the output open and was killed"
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

// javaBinary is the file name of the Java launcher on the given platform.
//
// TRLC-LINKS: REQ-EMG-144
func javaBinary(goos string) string {
	if goos == "windows" {
		return "java.exe"
	}
	return "java"
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
		path = filepath.Join(home, "bin", javaBinary(runtime.GOOS))
		if _, statErr := os.Stat(path); statErr != nil {
			return "", setupError(statErr, "java %d or later is required for verify mode (put it on PATH)", MinJavaMajor)
		}
	}
	out, err := run(ctx, Command{Path: path, Args: []string{"-version"}, Env: allowEnv(os.Environ(), nil, false)})
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

// baseEnvNames is the allow-list of environment variables every external
// call receives. Nothing else is inherited, so a token or other secret in the
// caller's environment never reaches Quint, Node, Java or Apalache; NODE_OPTIONS
// and JAVA_TOOL_OPTIONS in particular are dropped.
var baseEnvNames = []string{
	"PATH", "HOME", "LANG", "LC_ALL", "TMPDIR", "TEMP", "TMP", "TERM",
	"JAVA_HOME", QuintHomeEnv,
	// Windows needs these to start a process at all.
	"SYSTEMROOT", "WINDIR", "USERPROFILE", "PATHEXT", "COMSPEC",
}

// proxyEnvNames are passed on to verify calls only, together with the
// NO_PROXY handling of verifyEnv.
var proxyEnvNames = []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "all_proxy", "no_proxy"}

// envSet is an ordered environment. On Windows names are case-insensitive
// (PATH is often spelled Path), so fold maps every name to its upper case for
// lookups and keeps the spelling of the first occurrence.
type envSet struct {
	fold   bool
	values map[string]string
	spell  map[string]string
	order  []string
}

// TRLC-LINKS: REQ-EMG-144
func newEnvSet(fold bool) *envSet {
	return &envSet{fold: fold, values: map[string]string{}, spell: map[string]string{}}
}

// TRLC-LINKS: REQ-EMG-144
func (e *envSet) key(name string) string {
	if e.fold {
		return strings.ToUpper(name)
	}
	return name
}

// TRLC-LINKS: REQ-EMG-144
func (e *envSet) get(name string) string { return e.values[e.key(name)] }

// TRLC-LINKS: REQ-EMG-144
func (e *envSet) set(name, value string) {
	k := e.key(name)
	if _, seen := e.values[k]; !seen {
		e.order = append(e.order, k)
		e.spell[k] = name
	}
	e.values[k] = value
}

// TRLC-LINKS: REQ-EMG-144
func (e *envSet) list() []string {
	out := make([]string, 0, len(e.order))
	for _, k := range e.order {
		out = append(out, e.spell[k]+"="+e.values[k])
	}
	return out
}

// filterEnv keeps the entries of environ whose name is in one of the
// allow-lists.
//
// TRLC-LINKS: REQ-EMG-144
func filterEnv(fold bool, environ []string, allow ...[]string) *envSet {
	allowed := map[string]bool{}
	probe := newEnvSet(fold)
	for _, names := range allow {
		for _, name := range names {
			allowed[probe.key(name)] = true
		}
	}
	env := newEnvSet(fold)
	for _, kv := range environ {
		if k, v, ok := strings.Cut(kv, "="); ok && k != "" && allowed[env.key(k)] {
			env.set(k, v)
		}
	}
	return env
}

// allowEnv builds the environment of an external call from environ: the
// baseEnvNames, the extra names the caller asked for and, for verify calls,
// the proxy variables.
//
// TRLC-LINKS: REQ-EMG-143, REQ-EMG-144
func allowEnv(environ, extra []string, proxies bool) []string {
	lists := [][]string{baseEnvNames, extra}
	if proxies {
		lists = append(lists, proxyEnvNames)
	}
	return filterEnv(runtime.GOOS == "windows", environ, lists...).list()
}

// verifyEnv completes the allow-listed environment base of a quint verify call:
// the pinned QUINT_HOME, a scratch TMPDIR, the chosen Java first on PATH and,
// when any proxy is configured, NO_PROXY and no_proxy that keep localhost
// direct so the Apalache server is reachable. Existing values are appended to,
// never replaced by an empty string, and an unset PATH never gains an empty
// element.
//
// TRLC-LINKS: REQ-EMG-144
func verifyEnv(base []string, quintHome, tmpDir, javaDir string) []string {
	return buildVerifyEnv(runtime.GOOS == "windows", base, quintHome, tmpDir, javaDir)
}

// buildVerifyEnv is verifyEnv with the name matching rule chosen by fold.
//
// TRLC-LINKS: REQ-EMG-144
func buildVerifyEnv(fold bool, base []string, quintHome, tmpDir, javaDir string) []string {
	env := newEnvSet(fold)
	for _, kv := range base {
		if k, v, ok := strings.Cut(kv, "="); ok && k != "" {
			env.set(k, v)
		}
	}
	env.set(QuintHomeEnv, quintHome)
	env.set("TMPDIR", tmpDir)
	if javaDir != "" {
		path := javaDir
		if current := env.get("PATH"); current != "" {
			path += string(os.PathListSeparator) + current
		}
		env.set("PATH", path)
	}
	proxied := false
	for _, k := range proxyEnvNames {
		if k != "NO_PROXY" && k != "no_proxy" && env.get(k) != "" {
			proxied = true
		}
	}
	if proxied {
		for _, k := range []string{"NO_PROXY", "no_proxy"} {
			env.set(k, appendNoProxy(env.get(k)))
		}
	}
	return env.list()
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
