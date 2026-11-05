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

	"github.com/labeth/engineering-model-go/internal/linkreview"
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
                             checkout that has go.mod, replacing each version
                             a checkout requires of a workspace module with its
                             directory; remove go.work when none has; a checkout whose module path or directory
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
  bump <repo> <alias> <version> [--allow-review] [--show]
                             set dependency <alias> of <repo> to <version> in
                             engmod.yml, the matching go.mod require and go.sum
                             (go get, GOWORK=off), and the lock (composed with
                             ENGMOD_WORK=off, dependencies re-fetched), then
                             compare each requirement link on <alias> between
                             its pinned version and <version> (both published,
                             workspace off) by meaning: item kind, text and
                             verificationCriteria and each realizing
                             requirement's text and verificationCriteria
                             (status, notes, criticality and the like are not
                             compared); the same meaning moves pinned.version
                             to <version> (moved), a changed meaning, a pinned
                             version that cannot be resolved (review) or a
                             removed item (broken) keeps the pin, so the
                             model check reports link.version_mismatch until
                             review-links accepts it; one line per link, IDs
                             only (--show adds the text before and after);
                             exit 1 when a link needs review unless
                             --allow-review (--allow-suspect is a deprecated
                             alias); any failure restores all files
  review-links [--repo DIR] [--json] [--show]
                             list the requirement links of the model in DIR
                             (default .) that need review: pinned to another
                             version than the dependency, broken or claimed,
                             one line each with what changed in meaning
                             between the pinned version and the dependency
                             pin (both published, workspace off; a cached
                             module is refetched when the registry's digest
                             differs, for example after engws stage re-staged
                             a version); --show adds the parent requirement
                             and the item with its realizing requirements
                             before and after; exit 1 if any
  review-links [--repo DIR] --accept LNK-ID --by ACT-ID --rationale TEXT
                             record the review in the requirement link file:
                             pinned.version set to the dependency pin,
                             reviewedBy, reviewedOn (today), rationale and
                             status accepted, a deprecated fingerprint
                             removed; only those lines change; refuse an
                             actor the model does not define, a broken link
                             and a dependency that does not resolve with the
                             workspace off; then evaluate the link again and
                             exit 1 when it still does not pass
  link add [--repo DIR] (--parent REQ | --evidence REQ[,REQ]) <alias::CAP-...|alias::AOU-...> [--rationale TEXT]
                             add a claimed requirement link pinned to the
                             dependency version: the parent (or evidence)
                             must exist, the child must be a published
                             capability or assumption of use at that version
                             (workspace off), and no link may join the same
                             parent to it; appended with every other line
                             kept, model/req-links.yml and documents.reqLinks
                             created when missing; id LNK-<REQ>-<ALIAS>
  contract-diff <from-ref> <to-ref> [--repo DIR] [--json]
                             compare the published contract items (capabilities
                             and assumptions of use, with their realizing
                             requirements) of the model in DIR at two git refs:
                             added, removed, changed (text, realizers) or
                             unchanged, and the semver impact: capability
                             removed or text changed, assumption added or
                             changed: major; assumption removed, capability
                             added, realizers only: minor; else patch; exit 2
                             on a git or model error
  release-check [--repo DIR] [--tag vX.Y.Z]
                             gate a release of the model in DIR:
                             dependency.unreleased_pin when an engmod.yml or
                             engmod.lock.yml dependency version is a
                             pre-release or pseudo-version, and
                             contract.semver_too_small when the bump of the
                             tag (default: the release tag at HEAD) over the
                             nearest release tag below it in the commit's
                             history (git tag --merged) is smaller than the
                             contract-diff between them requires (in v0.x a
                             breaking change needs a minor bump); exit 1 on
                             findings, 2 on a usage or git error
  migrate-links [--repo DIR] [--dry-run]
                             convert the deprecated trace of the model in DIR:
                             each composition.allocations entry becomes a
                             claimed link in model/req-links.yml (id
                             LNK-<REQ>-<ALIAS>, the dependency pin; parent for
                             a capability, evidence for an assumption of
                             use), every deprecated pinned.fingerprint is
                             removed (count printed), documents.reqLinks is added, the
                             allocations are removed, and each capability
                             contract.provides[].ref becomes realizes on its
                             requirement; other lines are kept; print the
                             requirements left with requirement.no_parent;
                             a second run changes nothing; a failed write
                             restores every file; migrate the dependencies
                             first
  status-suggest [--repo DIR] [--tree HASH] [--json]
                             print the status each requirement and control
                             verification of the model in DIR earns from this
                             tree's evidence; write nothing: implemented
                             (code linked), verified (also a test link that
                             is not environment gated or only benchmarks,
                             and a RED line in .engflow/red-checks.jsonl for
                             this tree naming it in reqs), control
                             verification pass (every verificationRefs
                             requirement verified); the tree is the
                             working-tree hash (git write-tree on a
                             temporary index, as tree-hash.mjs) unless
                             --tree; exit 1 on a model or git error
  stage [--version-from manifest|tag] [--serve] [--addr 127.0.0.1:0]
                             publish every checkout, in dependency order, into
                             <root>/.engmod/staging (see Staging); refuse a
                             dirty checkout (and with --version-from tag one
                             whose HEAD lacks tag v<module.version>) and then
                             write nothing; print the environment for the
                             gate; --serve then serves the staged CUE modules
                             on a loopback OCI registry in the foreground
                             until interrupted; exit 1 on failure

Staging: stage writes only under <root>/.engmod/staging. registry/<module
path>/ holds each CUE module as an OCI image layout (oci-layout, index.json,
blobs/), the content cue mod publish would push: source kind git zips the
committed files (git ls-files) with the commit as metadata, kind self the
directory. goproxy/ is a GOPROXY file tree, <module>/@v/<version>.{info,mod,
zip} and list, for each checkout with go.mod, zipped from committed files.
The version is module.version from engmod.yml (tag v<version>). CUE has no
file:// registry, so --serve loads the layouts into an in-memory read-only
registry on 127.0.0.1 and prints CUE_REGISTRY routes for the staged modules
only (merged with the current CUE_REGISTRY). To prove the release order and
run the pre-push gate before pushing anything, keep engws stage --serve
running and, in another shell, export the printed lines and run the gate in
each checkout, dependencies first:
  export CUE_REGISTRY=example.com/fam/if=127.0.0.1:PORT+insecure,...
  export GOPROXY=file://<root>/.engmod/staging/goproxy,off GONOPROXY=none
  export GONOSUMDB=<staged Go modules> GOWORK=off ENGMOD_WORK=off
  ENGMOD_LOCK=check <the repo's pre-push gate or engmod check>
GOPROXY ends in off, so a module that is neither staged nor in the module
cache fails instead of reaching the network; no -mod flag is set (go.work
rejects it). Locks written against staging pin the staged digests: delete
<root>/.engmod/staging and regenerate them after the real publish if the
published content differs (for example a different commit).

exit codes: 0 ok, 1 skew, findings or failure (bump: a link needs review,
unless --allow-review; review-links: a link needs review or an accept was
refused or does not pass; link add: refused; release-check: findings), 2
usage (and,
for contract-diff and release-check, a git, IO or model error)
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
	versionFrom := sub.String("version-from", VersionFromManifest, "")
	serve := sub.Bool("serve", false, "")
	addr := sub.String("addr", "127.0.0.1:0", "")
	allowReview := sub.Bool("allow-review", false, "")
	sub.BoolVar(allowReview, "allow-suspect", false, "") // deprecated name of --allow-review
	show := sub.Bool("show", false, "")
	repoFlag := sub.String("repo", ".", "")
	asJSON := sub.Bool("json", false, "")
	accept := sub.String("accept", "", "")
	by := sub.String("by", "", "")
	rationale := sub.String("rationale", "", "")
	tag := sub.String("tag", "", "")
	treeFlag := sub.String("tree", "", "")
	dryRun := sub.Bool("dry-run", false, "")
	parent := sub.String("parent", "", "")
	evidence := sub.String("evidence", "", "")
	sub.StringVar(rootFlag, "root", *rootFlag, "")
	positional, err := parseInterspersed(sub, rest)
	if err != nil {
		fmt.Fprintf(stderr, "engws %s: %v\n%s", command, err, Usage)
		return 2
	}
	wantArgs := map[string]int{"status": 0, "sync": 0, "skew": 0, "check": 0, "pin": 1, "unpin": 1, "bump": 3, "stage": 0,
		"review-links": 0, "contract-diff": 2, "release-check": 0, "migrate-links": 0, "link": 2, "status-suggest": 0}
	n, known := wantArgs[command]
	if !known {
		fmt.Fprintf(stderr, "engws: unknown command %q\n%s", command, Usage)
		return 2
	}
	if len(positional) != n {
		fmt.Fprintf(stderr, "engws %s: expected %d argument(s), got %d\n%s", command, n, len(positional), Usage)
		return 2
	}

	if command == "stage" && *versionFrom != VersionFromManifest && *versionFrom != VersionFromTag {
		fmt.Fprintf(stderr, "engws stage: --version-from must be %s or %s\n%s", VersionFromManifest, VersionFromTag, Usage)
		return 2
	}

	// Repository commands work on one model (--repo, default the current
	// directory) and need no workspace root.
	if command == "review-links" {
		if *accept != "" && (strings.TrimSpace(*by) == "" || strings.TrimSpace(*rationale) == "") {
			fmt.Fprintf(stderr, "engws review-links: --accept needs --by ACT-ID and --rationale TEXT\n%s", Usage)
			return 2
		}
		code, err := runReviewLinks(*repoFlag, *asJSON, *show, *accept, strings.TrimSpace(*by), *rationale, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "engws %s: %v\n", command, err)
			return 1
		}
		return code
	}
	if command == "link" {
		if positional[0] != "add" {
			fmt.Fprintf(stderr, "engws link: unknown subcommand %q (want add)\n%s", positional[0], Usage)
			return 2
		}
		var ev []string
		if strings.TrimSpace(*evidence) != "" {
			ev = strings.Split(*evidence, ",")
		}
		code, err := runLinkAdd(*repoFlag, LinkAddOptions{Parent: *parent, Evidence: ev, Child: positional[1], Rationale: *rationale}, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "engws link add: %v\n", err)
		}
		return code
	}
	if command == "status-suggest" {
		code, err := runStatusSuggest(*repoFlag, strings.TrimSpace(*treeFlag), *asJSON, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "engws %s: %v\n", command, err)
		}
		return code
	}
	if command == "migrate-links" {
		code, err := runMigrateLinks(*repoFlag, *dryRun, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "engws %s: %v\n", command, err)
		}
		return code
	}
	if command == "release-check" {
		code, err := runReleaseCheck(*repoFlag, strings.TrimSpace(*tag), stdout)
		if err != nil {
			fmt.Fprintf(stderr, "engws %s: %v\n", command, err)
		}
		return code
	}
	if command == "contract-diff" {
		code, err := runContractDiff(*repoFlag, positional[0], positional[1], *asJSON, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "engws %s: %v\n", command, err)
		}
		return code
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
	case "check", "bump", "stage":
		fmt.Fprintf(stdout, "engws: workspace %s\n", root)
	}

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
		code, err = runBump(root, positional[0], positional[1], positional[2], BumpOptions{AllowReview: *allowReview}, *show, stdout)
	case "stage":
		code, err = runStage(ctx, root, *versionFrom, *serve, *addr, stdout)
	}
	if err != nil {
		fmt.Fprintf(stderr, "engws %s: %v\n", command, err)
		return 1
	}
	return code
}

// parseInterspersed parses flags anywhere among the arguments (flag stops at
// the first positional) and returns the positionals.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return positional, nil
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
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
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-086
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
	for _, warning := range result.Warnings {
		fmt.Fprintln(w, "warning:", warning)
	}
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

// runBump applies a dependency version bump and prints one line per
// requirement link on the bumped alias (with show, the item text before and
// after for each link left for review). It returns 1 when a link was left
// for review, unless opts.AllowReview.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
func runBump(root, repo, alias, version string, opts BumpOptions, show bool, w io.Writer) (int, error) {
	result, err := Bump(root, repo, alias, version, opts)
	for _, d := range result.Diagnostics {
		if d.Severity == validate.SeverityError && !strings.HasPrefix(d.Code, "link.") {
			fmt.Fprintf(w, "    %s [error] %s (%s)\n", d.Code, d.Message, d.Path)
		}
	}
	if err != nil {
		return 1, err
	}
	fmt.Fprintf(w, "engws: %s %s %s -> %s in engmod.yml\n", repo, alias, result.From, result.To)
	if result.GoModule != "" {
		fmt.Fprintf(w, "engws: %s go.mod require %s@%s\n", repo, result.GoModule, result.To)
	}
	if result.LockWritten {
		fmt.Fprintf(w, "engws: %s %s regenerated (workspace off)\n", repo, "engmod.lock.yml")
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(w, "engws: warning: %s\n", warning)
	}
	for _, l := range result.Links {
		printBumpLink(w, l, show)
	}
	if !result.NeedReview {
		return 0, nil
	}
	fmt.Fprintf(w, "engws: links marked review or broken keep their pin; the model check reports link.version_mismatch or link.broken until engws review-links --accept (engws review-links --repo %s --show lists the change)\n", repo)
	if opts.AllowReview {
		return 0, nil
	}
	return 1, nil
}

// printBumpLink prints one link outcome on one line, ID only: moved and
// current links with their versions, a link left for review or broken with
// what changed by ID; show adds the item before and after.
// TRLC-LINKS: REQ-EMG-100
func printBumpLink(w io.Writer, l BumpLink, show bool) {
	switch l.Outcome {
	case BumpLinkMoved:
		fmt.Fprintf(w, "  moved %s %s -> %s\n", l.ID, l.From, l.To)
	case BumpLinkCurrent:
		fmt.Fprintf(w, "  current %s %s\n", l.ID, l.To)
	default:
		fmt.Fprintf(w, "  %s %s (%s)\n", l.Outcome, l.ID, l.Note)
		if show && l.After != nil {
			var before []string
			if l.Before != nil {
				before = linkreview.NewItemView(l.Before).Lines()
			}
			for _, line := range linkreview.DiffLines(before, linkreview.NewItemView(l.After).Lines(), l.Before != nil) {
				fmt.Fprintln(w, "      "+line)
			}
		}
	}
}

// runStage stages the family, prints the result and the gate environment
// and, with serve, serves the staged CUE modules until ctx is done.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func runStage(ctx context.Context, root, versionFrom string, serve bool, addr string, w io.Writer) (int, error) {
	result, err := Stage(ctx, root, versionFrom)
	for _, warning := range result.Warnings {
		fmt.Fprintln(w, "engws: warning:", warning)
	}
	if err != nil {
		return 1, err
	}
	if len(result.Failures) > 0 {
		for _, failure := range result.Failures {
			fmt.Fprintln(w, "FAIL "+failure)
		}
		fmt.Fprintf(w, "engws: staged nothing: %d checkout(s) cannot be published\n", len(result.Failures))
		return 1, nil
	}
	for _, m := range result.Modules {
		base, _, _ := strings.Cut(m.ModulePath, "@")
		line := fmt.Sprintf("staged %s %s@%s (%d files)", m.Name, base, m.Version, m.CUEFiles)
		if m.GoModule != "" {
			line += ", go " + m.GoModule + "@" + m.Version
		}
		fmt.Fprintln(w, line)
		for _, note := range m.Notes {
			fmt.Fprintln(w, "    note: "+note)
		}
	}
	fmt.Fprintf(w, "engws: staging %s\n", filepath.Join(root, filepath.FromSlash(StagingDir)))
	current := map[string]string{}
	for _, key := range []string{"CUE_REGISTRY", "GONOSUMDB", "GOPRIVATE"} {
		current[key] = os.Getenv(key)
	}
	if !serve {
		fmt.Fprintln(w, "engws: CUE has no file registry; run engws stage --serve for CUE_REGISTRY. Go environment:")
		for _, line := range StageEnv(result, "", current) {
			fmt.Fprintln(w, "  "+line)
		}
		return 0, nil
	}
	err = ServeStaging(ctx, result, addr, func(host string) {
		fmt.Fprintln(w, "engws: environment for the gate against staging (export in another shell):")
		for _, line := range StageEnv(result, host, current) {
			fmt.Fprintln(w, "  "+line)
		}
		fmt.Fprintf(w, "engws: serving %d staged CUE module(s) on %s; interrupt to stop\n", len(result.Modules), host)
	})
	if err != nil {
		return 1, err
	}
	return 0, nil
}
