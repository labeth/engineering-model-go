// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// Usage is the engws command help.
const Usage = `usage: engws [--root DIR] <command> [flags]

Multi-repository workspace helper. The root is the nearest directory at or
above the current one, below $HOME, holding engmod.work.yml or .engws.yml
(else the current directory); $HOME and its ancestors and a world-writable
directory are used only when given with --root. Every command prints the
root first. The root's direct children with engmod.yml are the checkouts. A
checkout is local when it is pinned or changed: no own git repository, dirty
tree, HEAD not at tag v<module.version>, or that version missing from the
registry configured by CUE_REGISTRY. sync, pin and unpin write only at the
root (engmod.work.yml, go.work, .engws.yml); bump writes inside the consumer
checkout (engmod.yml, go.mod, go.sum, engmod.lock.yml).

Trust: check runs each checkout's own .github/engmod-check, so every
checkout under the root is trusted code. It gets a filtered environment
(PATH, HOME, Go, CUE, proxy and certificate settings, CI, ENGMOD_*; no API
keys or other credentials). Use check --in-process for checkouts you do not
trust: the built-in library checks them and no checkout code runs.

commands:
  status [--all]             table of checkouts with state and mode; exit 1 on skew
  sync [--all] [--force]     write engmod.work.yml for the local checkouts
                             (--all: every checkout) and go.work with every
                             checkout that has go.mod; remove go.work when none
                             has; a checkout whose module path or directory
                             name is not safe is skipped with a warning;
                             always exit 0
  pin <repo>                 always resolve <repo> locally (.engws.yml), then sync
  unpin <repo>               drop the pin, then sync
  skew [--all]               print SKEW consumer alias pinned local (also a
                             go.mod require of a checkout's Go module at
                             another version than its tag, alias go:<module>);
                             exit 1 if any
  check [--summary] [--in-process] [--all] [--refresh]
                             run the model check in the local checkouts, their
                             reverse dependencies and the top systems: one line
                             per repo plus its errors (--summary: lines only);
                             a repo's own .github/engmod-check (trusted code,
                             see Trust) is used when it supports
                             --workspace-root, run with GOWORK set to its
                             .github/engmod-check/go.work if present, else
                             off; otherwise the built-in library (--in-process
                             forces it: the safe choice for untrusted
                             checkouts); --refresh re-fetches every registry
                             dependency (ENGMOD_MODULE_CACHE=refresh); exit 1
                             on failure
  bump <repo> <alias> <version>
                             set dependency <alias> of <repo> to <version> in
                             engmod.yml, the matching go.mod require and go.sum
                             (go get, GOWORK=off), and the lock (composed with
                             ENGMOD_WORK=off, dependencies re-fetched); any
                             failure restores all files

exit codes: 0 ok, 1 skew or failure, 2 usage
`

// Main runs engws with args (without the program name) and returns the exit code.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func Main(ctx context.Context, args []string, stdout, stderr io.Writer, registry Registry) int {
	global := flag.NewFlagSet("engws", flag.ContinueOnError)
	global.SetOutput(io.Discard)
	rootFlag := global.String("root", "", "workspace root")
	help := global.Bool("help", false, "")
	global.BoolVar(help, "h", false, "")
	if err := global.Parse(args); err != nil || *help || global.NArg() == 0 {
		if *help {
			fmt.Fprint(stdout, Usage)
			return 0
		}
		if err != nil {
			fmt.Fprintln(stderr, "engws:", err)
		}
		fmt.Fprint(stderr, Usage)
		return 2
	}
	command, rest := global.Arg(0), global.Args()[1:]

	sub := flag.NewFlagSet("engws "+command, flag.ContinueOnError)
	sub.SetOutput(io.Discard)
	all := sub.Bool("all", false, "")
	force := sub.Bool("force", false, "")
	summary := sub.Bool("summary", false, "")
	inProcess := sub.Bool("in-process", false, "")
	refresh := sub.Bool("refresh", false, "")
	sub.StringVar(rootFlag, "root", *rootFlag, "")
	if err := sub.Parse(rest); err != nil {
		fmt.Fprintf(stderr, "engws %s: %v\n%s", command, err, Usage)
		return 2
	}
	positional := sub.Args()
	wantArgs := map[string]int{"status": 0, "sync": 0, "skew": 0, "check": 0, "pin": 1, "unpin": 1, "bump": 3}
	n, known := wantArgs[command]
	if !known {
		fmt.Fprintf(stderr, "engws: unknown command %q\n%s", command, Usage)
		return 2
	}
	if len(positional) != n {
		fmt.Fprintf(stderr, "engws %s: expected %d argument(s), got %d\n%s", command, n, len(positional), Usage)
		return 2
	}

	root := *rootFlag
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, "engws:", err)
			return 1
		}
		if root, err = FindRoot(cwd); err != nil {
			fmt.Fprintln(stderr, "engws:", err)
			return 1
		}
	} else if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}

	// Every command names the root first; skew keeps stdout for SKEW lines.
	switch command {
	case "skew":
		fmt.Fprintf(stderr, "engws: workspace %s\n", root)
	case "check", "bump":
		fmt.Fprintf(stdout, "engws: workspace %s\n", root)
	}

	var err error
	code := 0
	switch command {
	case "status":
		code, err = runStatus(ctx, root, registry, *all, stdout)
	case "sync":
		err = runSync(ctx, root, registry, *all, *force, stdout)
	case "pin", "unpin":
		err = runPin(ctx, root, registry, positional[0], command == "pin", *force, stdout)
	case "skew":
		code, err = runSkew(ctx, root, registry, *all, stdout)
	case "check":
		code, err = runCheck(ctx, root, registry, *all, *summary, CheckOptions{InProcess: *inProcess, Refresh: *refresh}, stdout, stderr)
	case "bump":
		err = runBump(root, positional[0], positional[1], positional[2], stdout)
	}
	if err != nil {
		fmt.Fprintf(stderr, "engws %s: %v\n", command, err)
		return 1
	}
	return code
}

// runStatus prints the checkout table and the skew lines.
// TRLC-LINKS: REQ-EMG-068
func runStatus(ctx context.Context, root string, registry Registry, all bool, w io.Writer) (int, error) {
	plan, err := BuildPlan(ctx, root, registry, all)
	if err != nil {
		return 1, err
	}
	printTable(w, plan)
	skews := printSkews(w, plan)
	if len(skews) > 0 {
		return 1, nil
	}
	return 0, nil
}

// printTable prints repo, module, version, state, mode and reason per checkout.
// TRLC-LINKS: REQ-EMG-068
func printTable(w io.Writer, plan Plan) {
	fmt.Fprintf(w, "workspace %s: %d checkouts, %d local\n", plan.Root, len(plan.Entries), len(plan.Local()))
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "REPO\tMODULE\tVERSION\tSTATE\tMODE\tREASON")
	for _, e := range plan.Entries {
		mode := "oci"
		if e.Local {
			mode = "local"
		}
		reasons := append([]string{}, e.Reasons...)
		if e.Pinned {
			reasons = append([]string{"pinned in " + StateFileName}, reasons...)
		}
		reason := strings.Join(reasons, "; ")
		if reason == "" {
			reason = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", e.Name, e.Manifest.Module.Path, e.Manifest.Module.Version, e.State, mode, reason)
	}
	tw.Flush()
	for _, note := range plan.Notes {
		fmt.Fprintln(w, "note:", note)
	}
	for _, warning := range plan.Warnings {
		fmt.Fprintln(w, "warning:", warning)
	}
}

// printSkews prints one SKEW line per overridden pin.
// TRLC-LINKS: REQ-EMG-068
func printSkews(w io.Writer, plan Plan) []Skew {
	skews := plan.Skews()
	for _, s := range skews {
		fmt.Fprintln(w, s.String())
	}
	return skews
}

// runSync writes the work files and reports them.
// TRLC-LINKS: REQ-EMG-068
func runSync(ctx context.Context, root string, registry Registry, all, force bool, w io.Writer) error {
	plan, err := BuildPlan(ctx, root, registry, all)
	if err != nil {
		return err
	}
	goVersion, err := GoVersion()
	if err != nil {
		return err
	}
	result, err := Sync(plan, goVersion, force)
	if err != nil {
		return err
	}
	printTable(w, plan)
	files := model.WorkspaceFileName
	switch {
	case result.GoWork != "":
		files += ", " + GoWorkFileName
	case result.GoWorkRemoved:
		files += " (" + GoWorkFileName + " removed)"
	}
	fmt.Fprintf(w, "engws: wrote %s with %d local of %d checkouts\n", files, len(plan.Local()), len(plan.Entries))
	printSkews(w, plan)
	return nil
}

// runPin adds or removes a pin and syncs.
// TRLC-LINKS: REQ-EMG-068
func runPin(ctx context.Context, root string, registry Registry, repo string, add, force bool, w io.Writer) error {
	pins, err := LoadPins(root)
	if err != nil {
		return err
	}
	if add {
		if info, err := os.Stat(filepath.Join(root, repo, "engmod.yml")); err != nil || info.IsDir() || strings.ContainsAny(repo, `/\`) {
			return fmt.Errorf("%q is not a checkout with engmod.yml under %s", repo, root)
		}
		pins[repo] = true
	} else {
		if !pins[repo] {
			defer fmt.Fprintf(w, "engws: %s was not pinned\n", repo)
		}
		delete(pins, repo)
	}
	if err := SavePins(root, pins); err != nil {
		return err
	}
	return runSync(ctx, root, registry, false, force, w)
}

// runSkew prints skew lines and returns 1 when there are any.
// TRLC-LINKS: REQ-EMG-068
func runSkew(ctx context.Context, root string, registry Registry, all bool, w io.Writer) (int, error) {
	plan, err := BuildPlan(ctx, root, registry, all)
	if err != nil {
		return 1, err
	}
	if len(printSkews(w, plan)) > 0 {
		return 1, nil
	}
	return 0, nil
}

// runCheck selects the affected checkouts and checks each one.
// TRLC-LINKS: REQ-EMG-068
func runCheck(ctx context.Context, root string, registry Registry, all, summary bool, opts CheckOptions, stdout, stderr io.Writer) (int, error) {
	plan, err := BuildPlan(ctx, root, registry, all)
	if err != nil {
		return 1, err
	}
	drift := WorkFileDrift(plan)
	if all {
		drift = CheckAllDrift(plan)
	}
	if drift != "" {
		fmt.Fprintln(stderr, "engws: warning:", drift)
	}
	if opts.InProcess {
		fmt.Fprintln(stdout, "engws: --in-process: the built-in library checks every checkout; no checkout code runs")
	} else {
		fmt.Fprintf(stdout, "engws: runs each checkout's own .github/engmod-check: every checkout under %s is trusted code (use --in-process for untrusted checkouts)\n", root)
	}
	targets := SelectCheckTargets(plan)
	if len(targets) == 0 {
		fmt.Fprintln(stdout, "engws: no checkouts to check")
		return 0, nil
	}
	code := 0
	for _, r := range RunChecks(ctx, root, targets, opts) {
		fmt.Fprintln(stdout, r.Line())
		if !summary {
			for _, line := range r.Lines {
				fmt.Fprintln(stdout, "    "+line)
			}
		}
		if r.Failed {
			code = 1
		}
	}
	return code, nil
}

// runBump applies a dependency version bump.
// TRLC-LINKS: REQ-EMG-068
func runBump(root, repo, alias, version string, w io.Writer) error {
	result, err := Bump(root, repo, alias, version)
	for _, d := range result.Diagnostics {
		if d.Severity == validate.SeverityError {
			fmt.Fprintf(w, "    %s [error] %s (%s)\n", d.Code, d.Message, d.Path)
		}
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "engws: %s %s %s -> %s in engmod.yml\n", repo, alias, result.From, result.To)
	if result.GoModule != "" {
		fmt.Fprintf(w, "engws: %s go.mod require %s@%s\n", repo, result.GoModule, result.To)
	}
	if result.LockWritten {
		fmt.Fprintf(w, "engws: %s %s regenerated (workspace off)\n", repo, "engmod.lock.yml")
	}
	return nil
}
