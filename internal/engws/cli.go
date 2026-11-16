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
                             off; a repo without one that takes the flag is
                             the error engine.check_template_stale (run
                             engflow upgrade --root <repo>), not checked by the
                             library, and a pin older than this engine is
                             engine.pin_outdated; --in-process uses the
                             built-in library for every repo (the safe choice
                             for untrusted checkouts); --refresh re-fetches every registry
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
                             --allow-review; any failure restores all files
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
                             status accepted; only those lines change (a
                             pinned.fingerprint is left in place); refuse an
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
  model-diff <from-ref> [<to-ref>] [--repo DIR] [--json] [--no-trace]
                             compare the whole authored model of the model in
                             DIR as committed at <from-ref> with <to-ref>, or
                             with the working tree when it is left out: the
                             entities (requirements, units, threats, decisions,
                             states, flow steps, ...) added, removed and changed
                             with the fields that differ, and the references
                             that appeared and disappeared; unless --no-trace,
                             also how traceability moved (requirements that
                             gained or lost code or tests, new and resolved
                             orphans), which scans the code of each state; exit
                             2 on a git, IO or model error
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
                             breaking change needs a minor bump), and
                             release.schema_version_not_bumped when the
                             CHANGELOG entry of the tag says Breaking release
                             and schemaVersion is not above that of the
                             previous release; exit 1 on findings, 2 on a
                             usage or git error
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

// parseGlobalFlags reads the flags before the command and splits off the command. It reports done with the exit
// code when the arguments are help, an error or have no command.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func parseGlobalFlags(args []string, stdout, stderr io.Writer) (command string, rest []string, root string, done bool, code int) {
	global := flag.NewFlagSet("engws", flag.ContinueOnError)
	global.SetOutput(io.Discard)
	rootFlag := global.String("root", "", "workspace root")
	help := global.Bool("help", false, "")
	global.BoolVar(help, "h", false, "")
	if err := global.Parse(args); err != nil || *help || global.NArg() == 0 {
		if *help {
			fmt.Fprint(stdout, Usage)
			return "", nil, "", true, 0
		}
		if err != nil {
			fmt.Fprintln(stderr, "engws:", err)
		}
		fmt.Fprint(stderr, Usage)
		return "", nil, "", true, 2
	}
	return global.Arg(0), global.Args()[1:], *rootFlag, false, 0
}

// cliFlags are the flags of every engws command; a command reads its own.
type cliFlags struct {
	all, force, summary, inProcess, refresh, serve, allowReview, show, asJSON bool
	versionFrom, addr, repo, root                                             string
	accept, by, rationale, tag, tree, parent, evidence                        string
}

// bind registers the flags on sub; root, set before, is the default of --root.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (f *cliFlags) bind(sub *flag.FlagSet) {
	sub.BoolVar(&f.all, "all", false, "")
	sub.BoolVar(&f.force, "force", false, "")
	sub.BoolVar(&f.summary, "summary", false, "")
	sub.BoolVar(&f.inProcess, "in-process", false, "")
	sub.BoolVar(&f.refresh, "refresh", false, "")
	sub.StringVar(&f.versionFrom, "version-from", VersionFromManifest, "")
	sub.BoolVar(&f.serve, "serve", false, "")
	sub.StringVar(&f.addr, "addr", "127.0.0.1:0", "")
	sub.BoolVar(&f.allowReview, "allow-review", false, "")
	sub.BoolVar(&f.show, "show", false, "")
	sub.StringVar(&f.repo, "repo", ".", "")
	sub.BoolVar(&f.asJSON, "json", false, "")
	sub.StringVar(&f.accept, "accept", "", "")
	sub.StringVar(&f.by, "by", "", "")
	sub.StringVar(&f.rationale, "rationale", "", "")
	sub.StringVar(&f.tag, "tag", "", "")
	sub.StringVar(&f.tree, "tree", "", "")
	sub.StringVar(&f.parent, "parent", "", "")
	sub.StringVar(&f.evidence, "evidence", "", "")
	sub.StringVar(&f.root, "root", f.root, "")
}

// cliCall is one engws command line: the command, its positional arguments and
// flags, and where it reads and writes.
type cliCall struct {
	ctx      context.Context
	registry Registry
	stdout   io.Writer
	stderr   io.Writer
	name     string
	args     []string
	flags    cliFlags
}

// cliArgCounts is the number of positional arguments of each command.
var cliArgCounts = map[string]int{"status": 0, "sync": 0, "skew": 0, "check": 0, "pin": 1, "unpin": 1, "bump": 3, "stage": 0,
	"review-links": 0, "contract-diff": 2, "release-check": 0, "link": 2, "status-suggest": 0}

// cliHandlers maps each command to its handler. Repository commands work on
// one model (--repo, default the current directory) and need no workspace
// root; the others run through cliRooted.
var cliHandlers = map[string]func(*cliCall) int{
	"review-links":   (*cliCall).reviewLinks,
	"link":           (*cliCall).linkAdd,
	"status-suggest": (*cliCall).statusSuggest,
	"release-check":  (*cliCall).releaseCheck,
	"contract-diff":  (*cliCall).contractDiff,
	"status":         cliRooted((*cliCall).runStatus),
	"sync":           cliRooted((*cliCall).runSync),
	"pin":            cliRooted((*cliCall).runPin),
	"unpin":          cliRooted((*cliCall).runPin),
	"skew":           cliRooted((*cliCall).runSkew),
	"check":          cliRooted((*cliCall).runCheck),
	"bump":           cliRooted((*cliCall).runBump),
	"stage":          cliRooted((*cliCall).runStage),
}

// CLI is what engws runs against: the output streams and the registry that
// says which module versions are published.
type CLI struct {
	Stdout, Stderr io.Writer
	Registry       Registry
}

// Main runs engws with args (without the program name) and returns the exit code.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func Main(ctx context.Context, args []string, cli CLI) int {
	command, rest, rootArg, done, exit := parseGlobalFlags(args, cli.Stdout, cli.Stderr)
	if done {
		return exit
	}
	if command == "model-diff" {
		return runModelDiffCommand(rest, cli.Stdout, cli.Stderr)
	}
	call := &cliCall{ctx: ctx, registry: cli.Registry, stdout: cli.Stdout, stderr: cli.Stderr, name: command}
	call.flags.root = rootArg
	if code, failed := call.parse(rest); failed {
		return code
	}
	return cliHandlers[command](call)
}

// parse reads the flags and positional arguments and checks the command and
// its argument count; failed carries the exit code 2 after the usage is printed.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (c *cliCall) parse(rest []string) (code int, failed bool) {
	sub := flag.NewFlagSet("engws "+c.name, flag.ContinueOnError)
	sub.SetOutput(io.Discard)
	c.flags.bind(sub)
	positional, err := parseInterspersed(sub, rest)
	if err != nil {
		fmt.Fprintf(c.stderr, "engws %s: %v\n%s", c.name, err, Usage)
		return 2, true
	}
	c.args = positional
	n, known := cliArgCounts[c.name]
	if !known {
		fmt.Fprintf(c.stderr, "engws: unknown command %q\n%s", c.name, Usage)
		return 2, true
	}
	if len(positional) != n {
		fmt.Fprintf(c.stderr, "engws %s: expected %d argument(s), got %d\n%s", c.name, n, len(positional), Usage)
		return 2, true
	}
	if vf := c.flags.versionFrom; c.name == "stage" && vf != VersionFromManifest && vf != VersionFromTag {
		fmt.Fprintf(c.stderr, "engws stage: --version-from must be %s or %s\n%s", VersionFromManifest, VersionFromTag, Usage)
		return 2, true
	}
	return 0, false
}

// failed prints a command error as "engws <label>: <err>" and returns code.
// TRLC-LINKS: REQ-EMG-068
func (c *cliCall) failed(label string, err error, code int) int {
	if err != nil {
		fmt.Fprintf(c.stderr, "engws %s: %v\n", label, err)
	}
	return code
}

// reviewLinks runs engws review-links.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (c *cliCall) reviewLinks() int {
	f := c.flags
	if f.accept != "" && (strings.TrimSpace(f.by) == "" || strings.TrimSpace(f.rationale) == "") {
		fmt.Fprintf(c.stderr, "engws review-links: --accept needs --by ACT-ID and --rationale TEXT\n%s", Usage)
		return 2
	}
	code, err := runReviewLinks(reviewLinksOptions{Repo: f.repo, JSON: f.asJSON, Show: f.show, Accept: linkreview.AcceptRequest{ID: f.accept, By: strings.TrimSpace(f.by), Rationale: f.rationale}}, c.stdout)
	if err != nil {
		return c.failed(c.name, err, 1)
	}
	return code
}

// linkAdd runs engws link add.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (c *cliCall) linkAdd() int {
	if c.args[0] != "add" {
		fmt.Fprintf(c.stderr, "engws link: unknown subcommand %q (want add)\n%s", c.args[0], Usage)
		return 2
	}
	var ev []string
	if strings.TrimSpace(c.flags.evidence) != "" {
		ev = strings.Split(c.flags.evidence, ",")
	}
	opts := LinkAddOptions{Parent: c.flags.parent, Evidence: ev, Child: c.args[1], Rationale: c.flags.rationale}
	code, err := runLinkAdd(c.flags.repo, opts, c.stdout)
	return c.failed("link add", err, code)
}

// statusSuggest runs engws status-suggest.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (c *cliCall) statusSuggest() int {
	code, err := runStatusSuggest(StatusSuggestOptions{Repo: c.flags.repo, Tree: strings.TrimSpace(c.flags.tree), JSON: c.flags.asJSON}, c.stdout)
	return c.failed(c.name, err, code)
}

// releaseCheck runs engws release-check.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (c *cliCall) releaseCheck() int {
	code, err := runReleaseCheck(c.flags.repo, strings.TrimSpace(c.flags.tag), c.stdout)
	return c.failed(c.name, err, code)
}

// contractDiff runs engws contract-diff.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func (c *cliCall) contractDiff() int {
	code, err := runContractDiff(ContractDiffOptions{Repo: c.flags.repo, From: c.args[0], To: c.args[1], JSON: c.flags.asJSON}, c.stdout)
	return c.failed(c.name, err, code)
}

// cliRooted wraps a workspace command: it finds the root, names it first (skew
// keeps stdout for SKEW lines) and turns an error into exit 1.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func cliRooted(run func(*cliCall, string) (int, error)) func(*cliCall) int {
	return func(c *cliCall) int {
		root, err := c.resolveRoot()
		if err != nil {
			fmt.Fprintln(c.stderr, "engws:", err)
			return 1
		}
		c.announceRoot(root)
		code, err := run(c, root)
		if err != nil {
			return c.failed(c.name, err, 1)
		}
		return code
	}
}

// resolveRoot is --root made absolute, else the nearest workspace root.
// TRLC-LINKS: REQ-EMG-068
func (c *cliCall) resolveRoot() (string, error) {
	root := c.flags.root
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return FindRoot(cwd)
	}
	if abs, err := filepath.Abs(root); err == nil {
		return abs, nil
	}
	return root, nil
}

// announceRoot names the root first for the commands that do.
// TRLC-LINKS: REQ-EMG-068
func (c *cliCall) announceRoot(root string) {
	switch c.name {
	case "skew":
		fmt.Fprintf(c.stderr, "engws: workspace %s\n", root)
	case "check", "bump", "stage":
		fmt.Fprintf(c.stdout, "engws: workspace %s\n", root)
	}
}

// buildPlan classifies the checkouts under root with the command's registry.
// TRLC-LINKS: REQ-EMG-068
func (c *cliCall) buildPlan(root string, all bool) (Plan, error) {
	return BuildPlan(c.ctx, PlanRequest{Root: root, Registry: c.registry, All: all})
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
func (c *cliCall) runStatus(root string) (int, error) {
	plan, err := c.buildPlan(root, c.flags.all)
	if err != nil {
		return 1, err
	}
	printTable(c.stdout, plan)
	if len(printSkews(c.stdout, plan)) > 0 {
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
func (c *cliCall) runSync(root string) (int, error) {
	return 0, c.syncRoot(root, c.flags.all)
}

// syncRoot writes the work files of the root and reports them.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-086
func (c *cliCall) syncRoot(root string, all bool) error {
	plan, err := c.buildPlan(root, all)
	if err != nil {
		return err
	}
	goVersion, err := GoVersion()
	if err != nil {
		return err
	}
	result, err := Sync(plan, goVersion, c.flags.force)
	if err != nil {
		return err
	}
	w := c.stdout
	printTable(w, plan)
	for _, warning := range result.Warnings {
		fmt.Fprintln(w, "warning:", warning)
	}
	fmt.Fprintf(w, "engws: wrote %s with %d local of %d checkouts\n", syncedFiles(result), len(plan.Local()), len(plan.Entries))
	printSkews(w, plan)
	return nil
}

// syncedFiles names the files sync wrote or removed.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-086
func syncedFiles(result SyncResult) string {
	files := model.WorkspaceFileName
	switch {
	case result.GoWork != "":
		files += ", " + GoWorkFileName
	case result.GoWorkRemoved:
		files += " (" + GoWorkFileName + " removed)"
	}
	return files
}

// runPin adds or removes a pin and syncs.
// TRLC-LINKS: REQ-EMG-068
func (c *cliCall) runPin(root string) (int, error) {
	repo := c.args[0]
	pins, err := LoadPins(root)
	if err != nil {
		return 1, err
	}
	if c.name == "pin" {
		if err := pinnable(root, repo); err != nil {
			return 1, err
		}
		pins[repo] = true
	} else {
		if !pins[repo] {
			defer fmt.Fprintf(c.stdout, "engws: %s was not pinned\n", repo)
		}
		delete(pins, repo)
	}
	if err := SavePins(root, pins); err != nil {
		return 1, err
	}
	return 0, c.syncRoot(root, false)
}

// pinnable checks that repo is a checkout with engmod.yml under root.
// TRLC-LINKS: REQ-EMG-068
func pinnable(root, repo string) error {
	info, err := os.Stat(filepath.Join(root, repo, "engmod.yml"))
	if err != nil || info.IsDir() || strings.ContainsAny(repo, `/\`) {
		return fmt.Errorf("%q is not a checkout with engmod.yml under %s", repo, root)
	}
	return nil
}

// runSkew prints skew lines and returns 1 when there are any.
// TRLC-LINKS: REQ-EMG-068
func (c *cliCall) runSkew(root string) (int, error) {
	plan, err := c.buildPlan(root, c.flags.all)
	if err != nil {
		return 1, err
	}
	if len(printSkews(c.stdout, plan)) > 0 {
		return 1, nil
	}
	return 0, nil
}

// runCheck selects the affected checkouts and checks each one.
// TRLC-LINKS: REQ-EMG-068
func (c *cliCall) runCheck(root string) (int, error) {
	plan, err := c.buildPlan(root, c.flags.all)
	if err != nil {
		return 1, err
	}
	drift := WorkFileDrift(plan)
	if c.flags.all {
		drift = CheckAllDrift(plan)
	}
	if drift != "" {
		fmt.Fprintln(c.stderr, "engws: warning:", drift)
	}
	if c.flags.inProcess {
		fmt.Fprintln(c.stdout, "engws: --in-process: the built-in library checks every checkout; no checkout code runs")
	} else {
		fmt.Fprintf(c.stdout, "engws: runs each checkout's own .github/engmod-check: every checkout under %s is trusted code (use --in-process for untrusted checkouts)\n", root)
	}
	targets := SelectCheckTargets(plan)
	if len(targets) == 0 {
		fmt.Fprintln(c.stdout, "engws: no checkouts to check")
		return 0, nil
	}
	opts := CheckOptions{Root: root, InProcess: c.flags.inProcess, Refresh: c.flags.refresh}
	return c.printChecks(RunChecks(c.ctx, targets, opts)), nil
}

// printChecks prints one line per check result (with its error lines unless
// --summary) and returns 1 when any failed.
// TRLC-LINKS: REQ-EMG-068
func (c *cliCall) printChecks(results []CheckResult) int {
	code := 0
	for _, r := range results {
		fmt.Fprintln(c.stdout, r.Line())
		if !c.flags.summary {
			printCheckLines(c.stdout, r.Lines)
		}
		if r.Failed {
			code = 1
		}
	}
	return code
}

// printCheckLines prints the error lines of one check result, indented.
// TRLC-LINKS: REQ-EMG-068
func printCheckLines(w io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintln(w, "    "+line)
	}
}

// runBump applies a dependency version bump and prints one line per
// requirement link on the bumped alias (with show, the item text before and
// after for each link left for review). It returns 1 when a link was left
// for review, unless allow-review.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100, REQ-EMG-126
// ENGMODEL-LINKS: EVT-DEPENDENCY-BUMP-REQUESTED
func (c *cliCall) runBump(root string) (int, error) {
	repo, alias, version := c.args[0], c.args[1], c.args[2]
	result, err := Bump(root, BumpTarget{Repo: repo, Alias: alias, Version: version}, BumpOptions{AllowReview: c.flags.allowReview})
	c.printBumpErrors(result)
	if err != nil {
		return 1, err
	}
	c.printBumpResult(repo, alias, result)
	if !result.NeedReview {
		return 0, nil
	}
	fmt.Fprintf(c.stdout, "engws: links marked review or broken keep their pin; the model check reports link.version_mismatch or link.broken until engws review-links --accept (engws review-links --repo %s --show lists the change)\n", repo)
	if c.flags.allowReview {
		return 0, nil
	}
	return 1, nil
}

// printBumpErrors prints the model errors of a bump, link findings aside.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100
func (c *cliCall) printBumpErrors(result BumpResult) {
	for _, d := range result.Diagnostics {
		if d.Severity != validate.SeverityError || strings.HasPrefix(d.Code, "link.") {
			continue
		}
		fmt.Fprintf(c.stdout, "    %s [error] %s (%s)\n", d.Code, d.Message, d.Path)
		if fix := d.FixLine(); fix != "" {
			fmt.Fprintf(c.stdout, "    %s\n", strings.TrimSpace(fix))
		}
	}
}

// printBumpResult prints what the bump wrote and one line per link.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-100
func (c *cliCall) printBumpResult(repo, alias string, result BumpResult) {
	w := c.stdout
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
		printBumpLink(w, l, c.flags.show)
	}
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
			printBumpLinkDiff(w, l)
		}
	}
}

// printBumpLinkDiff prints the item of a link before and after the bump.
// TRLC-LINKS: REQ-EMG-100
func printBumpLinkDiff(w io.Writer, l BumpLink) {
	var before []string
	if l.Before != nil {
		before = linkreview.NewItemView(l.Before).Lines()
	}
	for _, line := range linkreview.DiffLines(before, linkreview.NewItemView(l.After).Lines(), l.Before != nil) {
		fmt.Fprintln(w, "      "+line)
	}
}

// runStage stages the family, prints the result and the gate environment
// and, with serve, serves the staged CUE modules until ctx is done.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func (c *cliCall) runStage(root string) (int, error) {
	w := c.stdout
	result, err := Stage(c.ctx, root, c.flags.versionFrom)
	for _, warning := range result.Warnings {
		fmt.Fprintln(w, "engws: warning:", warning)
	}
	if err != nil {
		return 1, err
	}
	if len(result.Failures) > 0 {
		printStageFailures(w, result)
		return 1, nil
	}
	printStagedModules(w, result)
	fmt.Fprintf(w, "engws: staging %s\n", filepath.Join(root, filepath.FromSlash(StagingDir)))
	current := stageCurrentEnv()
	if !c.flags.serve {
		fmt.Fprintln(w, "engws: CUE has no file registry; run engws stage --serve for CUE_REGISTRY. Go environment:")
		printStageEnv(w, StageEnv(result, "", current))
		return 0, nil
	}
	err = ServeStaging(c.ctx, result, ServeOptions{Addr: c.flags.addr, Ready: func(host string) {
		fmt.Fprintln(w, "engws: environment for the gate against staging (export in another shell):")
		printStageEnv(w, StageEnv(result, host, current))
		fmt.Fprintf(w, "engws: serving %d staged CUE module(s) on %s; interrupt to stop\n", len(result.Modules), host)
	}})
	if err != nil {
		return 1, err
	}
	return 0, nil
}

// printStageFailures prints the checkouts that cannot be published.
// TRLC-LINKS: REQ-EMG-080
func printStageFailures(w io.Writer, result StageResult) {
	for _, failure := range result.Failures {
		fmt.Fprintln(w, "FAIL "+failure)
	}
	fmt.Fprintf(w, "engws: staged nothing: %d checkout(s) cannot be published\n", len(result.Failures))
}

// stageCurrentEnv is the registry-related environment the gate environment
// is merged with.
// TRLC-LINKS: REQ-EMG-080
func stageCurrentEnv() map[string]string {
	current := map[string]string{}
	for _, key := range []string{"CUE_REGISTRY", "GONOSUMDB", "GOPRIVATE"} {
		current[key] = os.Getenv(key)
	}
	return current
}

// printStagedModules prints one line per staged module with its notes.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func printStagedModules(w io.Writer, result StageResult) {
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
}

// printStageEnv prints the environment lines indented.
// TRLC-LINKS: REQ-EMG-080
func printStageEnv(w io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintln(w, "  "+line)
	}
}
