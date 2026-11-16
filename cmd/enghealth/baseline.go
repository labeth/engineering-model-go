// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/model"
)

// baselineOptions are the parsed flags of enghealth baseline write.
type baselineOptions struct {
	model, tier, out string
	check            bool
	paths            []string
}

// runBaseline implements enghealth baseline (ADR-EMG-059): "write" creates the
// committed baseline of accepted debt from a scan that ignores any existing
// baseline, and "write --check" verifies that the file is exactly what write
// would produce. Exit codes: 0 written or equal, 1 the file differs (check
// only), 2 a setup or usage error.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158, REQ-EMG-161
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func runBaseline(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "write" {
		return baselineHelp(args, streams{stdout, stderr})
	}
	opts, code, ok := parseBaselineFlags(args[1:], stderr)
	if !ok {
		return code
	}
	return runBaselineWrite(opts, streams{stdout, stderr})
}

// streams are the standard output and error of a command.
type streams struct {
	stdout, stderr io.Writer
}

// baselineHelp prints the usage of the subcommand: to stdout when it was asked
// for, to stderr when the command is not write.
//
// TRLC-LINKS: REQ-EMG-158, REQ-EMG-161
func baselineHelp(args []string, s streams) int {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		baselineUsage(s.stdout)
		return exitOK
	}
	baselineUsage(s.stderr)
	return exitUsage
}

// runBaselineWrite resolves the target and builds the wanted file, then checks
// it or writes it.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158, REQ-EMG-161
func runBaselineWrite(opts baselineOptions, s streams) int {
	target, err := baselineTarget(opts)
	if err != nil {
		fmt.Fprintln(s.stderr, "enghealth baseline:", safeText(err.Error()))
		return exitUsage
	}
	want, err := baselineWanted(opts, target)
	if err != nil {
		fmt.Fprintln(s.stderr, "enghealth baseline:", safeText(err.Error()))
		return exitUsage
	}
	if opts.check {
		return checkBaseline(target, want, s)
	}
	if err := writeBaselineFile(target.path, want.data); err != nil {
		fmt.Fprintln(s.stderr, "enghealth baseline: write:", safeText(err.Error()))
		return exitUsage
	}
	fmt.Fprintf(s.stdout, "enghealth baseline: wrote %d entries to %s\n", len(want.entries), safeText(target.shown))
	return exitOK
}

// baselineUsage prints the usage of the subcommand.
//
// TRLC-LINKS: REQ-EMG-158, REQ-EMG-161
func baselineUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: enghealth baseline write --model <engmod.yml> [--tier t] [--out file] [--check] [paths...]")
	fmt.Fprintln(w, "  write   scan, ignoring any existing baseline, and write the canonical baseline of the current findings")
	fmt.Fprintln(w, "  --check do not write: exit 0 when the file is exactly what write would produce, 1 when it differs, 2 on a setup error")
}

// parseBaselineFlags parses the flags of write. The bool is false when the
// caller must return the exit code at once (a bad flag or help).
//
// TRLC-LINKS: REQ-EMG-158, REQ-EMG-161
func parseBaselineFlags(args []string, stderr io.Writer) (baselineOptions, int, bool) {
	var opts baselineOptions
	flags := flag.NewFlagSet("enghealth baseline write", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.model, "model", "", "path to the engmod.yml manifest")
	flags.StringVar(&opts.tier, "tier", "", "baseline only the findings that are errors at this life-cycle tier (default: every finding)")
	flags.StringVar(&opts.out, "out", "", "write this file instead of the health.baseline.file of the manifest; it must be inside the manifest directory")
	flags.BoolVar(&opts.check, "check", false, "verify the baseline file instead of writing it")
	flags.Usage = func() { baselineUsage(stderr); flags.PrintDefaults() }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return opts, exitOK, false
		}
		return opts, exitUsage, false
	}
	if strings.TrimSpace(opts.model) == "" {
		flags.Usage()
		return opts, exitUsage, false
	}
	opts.paths = flags.Args()
	return opts, exitOK, true
}

// baselineFile is where the baseline lives: the absolute path to read and
// write and the form shown in messages.
type baselineFile struct {
	path, shown string
}

// baselineTarget resolves the file to write: --out, or the file of
// health.baseline. It refuses a target outside the directory of the manifest,
// by path and through symbolic links.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158, REQ-EMG-161
func baselineTarget(opts baselineOptions) (baselineFile, error) {
	manifest, err := filepath.Abs(opts.model)
	if err != nil {
		return baselineFile{}, err
	}
	dir := filepath.Dir(manifest)
	target, err := baselineFilePath(opts, manifest)
	if err != nil {
		return baselineFile{}, err
	}
	if !insideDir(dir, target) {
		return baselineFile{}, fmt.Errorf("refusing to use %s: it is outside the manifest directory %s", target, dir)
	}
	shown, _ := filepath.Rel(dir, target)
	return baselineFile{path: target, shown: filepath.ToSlash(shown)}, nil
}

// baselineFilePath is the absolute path of --out, or of the health.baseline
// file of the manifest.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158, REQ-EMG-161
func baselineFilePath(opts baselineOptions, manifest string) (string, error) {
	if strings.TrimSpace(opts.out) != "" {
		return filepath.Abs(opts.out)
	}
	bundle, err := engmodel.LoadBundle(manifest)
	if err != nil {
		return "", fmt.Errorf("load model: %w", err)
	}
	h := bundle.Manifest.Health
	if h == nil || h.Baseline == nil {
		return "", errors.New("the manifest has no health.baseline; add health.baseline {file, decision} or pass --out")
	}
	return model.ResolveHealthBaselinePath(manifest, h.Baseline.File)
}

// insideDir reports whether target is dir or below it, after resolving the
// symbolic links of the closest existing ancestor of target and of dir.
//
// TRLC-LINKS: REQ-EMG-158, REQ-EMG-161
func insideDir(dir, target string) bool {
	rel, err := filepath.Rel(realExisting(dir), realExisting(target))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// realExisting resolves symbolic links in the part of p that exists and keeps
// the rest as written.
//
// TRLC-LINKS: REQ-EMG-158, REQ-EMG-161
func realExisting(p string) string {
	rest := ""
	for cur := p; ; cur = filepath.Dir(cur) {
		if real, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(real, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p
		}
		rest = filepath.Join(filepath.Base(cur), rest)
	}
}

// missingBaseline is the problem reported for a baseline file that does not
// exist yet.
const missingBaseline = "the file does not exist"

// baselineContent is what write would produce.
type baselineContent struct {
	entries []model.HealthBaselineEntry
	data    []byte
	// have are the entries of the existing file (none when it is missing or
	// unreadable) and haveData its bytes; problem says why it is unusable.
	have     []model.HealthBaselineEntry
	haveData []byte
	problem  string
}

// baselineWanted scans without the baseline and builds the canonical file. With
// paths the scan covers only them, so the entries of the existing file for
// files that were not scanned are kept.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158, REQ-EMG-161
func baselineWanted(opts baselineOptions, target baselineFile) (baselineContent, error) {
	report, err := engmodel.ScanHealth(engmodel.HealthScanOptions{
		ManifestPath: opts.model, Tier: opts.tier, Paths: opts.paths, IgnoreBaseline: true,
	})
	if err != nil {
		return baselineContent{}, fmt.Errorf("scan: %w", err)
	}
	got := baselineContent{entries: health.BaselineEntries(report, tierFilter(opts.tier, report))}
	got.haveData, got.have, got.problem = readExistingBaseline(target.path)
	if err := checkRestrictedWrite(opts, target, got.problem); err != nil {
		return baselineContent{}, err
	}
	if len(opts.paths) > 0 {
		got.entries = keepOutsideScan(got.have, report, got.entries)
	}
	if got.data, err = model.EncodeHealthBaseline(got.entries); err != nil {
		return baselineContent{}, err
	}
	return got, nil
}

// tierFilter limits the baseline to the findings that are errors at the tier;
// nil, which keeps every finding, without a tier.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158, REQ-EMG-161
func tierFilter(tier string, report engmodel.HealthReport) func(string) bool {
	if strings.TrimSpace(tier) == "" {
		return nil
	}
	return func(code string) bool { return report.LevelOf(code) == health.LevelError }
}

// checkRestrictedWrite refuses a write restricted to paths when the existing
// baseline file is unusable, because that write keeps the existing entries.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158, REQ-EMG-161
func checkRestrictedWrite(opts baselineOptions, target baselineFile, problem string) error {
	if len(opts.paths) > 0 && !opts.check && unusableBaseline(problem) {
		return fmt.Errorf("%s is unusable (%s); a write restricted to paths needs the existing entries, so run the full write", target.shown, problem)
	}
	return nil
}

// unusableBaseline reports a problem that is not a missing file.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func unusableBaseline(problem string) bool { return problem != "" && problem != missingBaseline }

// readExistingBaseline reads the file at path, with CRLF line ends read as LF. A
// missing file or one that is not a baseline is not an error here: the problem
// is returned for check to print and write replaces the file.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func readExistingBaseline(path string) ([]byte, []model.HealthBaselineEntry, string) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, missingBaseline
	}
	if err != nil {
		return nil, nil, err.Error()
	}
	// A checkout with autocrlf holds the file with CRLF line ends; the canonical
	// form has LF, so both the parse and the comparison read the LF form.
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	parsed, err := model.ParseHealthBaseline(data)
	if err != nil {
		return data, nil, strings.TrimPrefix(err.Error(), model.ErrHealthBaseline.Error()+": ")
	}
	return data, parsed.Entries, ""
}

// keepOutsideScan adds to the fresh entries of the scanned files the entries
// of the existing file for files the scan did not cover.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func keepOutsideScan(existing []model.HealthBaselineEntry, report engmodel.HealthReport, fresh []model.HealthBaselineEntry) []model.HealthBaselineEntry {
	scanned := make(map[string]bool, len(report.Files))
	for _, f := range report.Files {
		scanned[f.Path] = true
	}
	merged := append([]model.HealthBaselineEntry(nil), fresh...)
	for _, e := range existing {
		if !scanned[e.Path] {
			merged = append(merged, e)
		}
	}
	return model.SortHealthBaselineEntries(merged)
}

// checkBaseline compares the file with what write would produce and prints the
// stale entries (in the file, no longer a finding) and the missing ones (a
// finding the file does not list).
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158, REQ-EMG-161
func checkBaseline(target baselineFile, want baselineContent, s streams) int {
	shown := safeText(target.shown)
	if want.problem == "" && bytes.Equal(want.haveData, want.data) {
		fmt.Fprintf(s.stdout, "enghealth baseline: %s is exactly the current debt (%d entries)\n", shown, len(want.entries))
		return exitOK
	}
	if want.problem != "" {
		fmt.Fprintf(s.stdout, "enghealth baseline: %s is unusable: %s\n", shown, safeText(want.problem))
	}
	stale, missing := health.BaselineDiff(want.have, want.entries)
	printEntries(s.stdout, "stale", stale)
	printEntries(s.stdout, "missing", missing)
	if want.problem == "" && len(stale) == 0 && len(missing) == 0 {
		fmt.Fprintf(s.stdout, "enghealth baseline: %s lists the current debt but is not in canonical form\n", shown)
	}
	fmt.Fprintf(s.stderr, "enghealth baseline: %s differs from the current debt (%d stale, %d missing); run enghealth baseline write\n", shown, len(stale), len(missing))
	return exitFail
}

// printEntries prints one line per baseline entry under a label.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158, REQ-EMG-161
func printEntries(w io.Writer, label string, entries []model.HealthBaselineEntry) {
	for _, e := range entries {
		fmt.Fprintf(w, "%s: %s %s in %s (value %d)\n", label, e.Code, safeText(e.Function), safeText(e.Path), e.Value)
	}
}

// writeBaselineFile writes the file, creating its directory, through
// writeFileAtomic.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func writeBaselineFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(path, data)
}
