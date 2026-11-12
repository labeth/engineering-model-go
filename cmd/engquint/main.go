// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/export/quint/check"
)

// Exit codes shared by every subcommand: 0 success, 1 a generation or check
// failure, 2 a usage or setup error.
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

// command is one engquint subcommand. A new subcommand (such as check) is one
// more entry in commands; run receives the arguments after the command name.
type command struct {
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
}

var commands = map[string]command{
	"generate": {summary: "write one Quint module per machine of the model", run: runGenerate},
	"check":    {summary: "run or verify the Quint modules and record the results", run: runCheck},
}

// TRLC-LINKS: REQ-EMG-142
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches to the subcommand named by args[0].
//
// TRLC-LINKS: REQ-EMG-142
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		usage(stdout)
		return exitOK
	}
	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "engquint: unknown command %q\n", args[0])
		usage(stderr)
		return exitUsage
	}
	return cmd.run(args[1:], stdout, stderr)
}

// usage prints the command list.
//
// TRLC-LINKS: REQ-EMG-142
func usage(w io.Writer) {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Fprintln(w, "usage: engquint <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	for _, name := range names {
		fmt.Fprintf(w, "  %-10s %s\n", name, commands[name].summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "run engquint <command> -h for the flags of a command")
}

// runGenerate writes the Quint modules of a model.
//
// TRLC-LINKS: REQ-EMG-141, REQ-EMG-142, REQ-EMG-149
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func runGenerate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("engquint generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	modelPath := flags.String("model", "", "path to engmod.yml manifest")
	outDir := flags.String("out", "", "output directory; defaults to generated/quint next to the model's engmod.yml")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "engquint generate: unexpected argument %q\n", flags.Arg(0))
		return exitUsage
	}
	if strings.TrimSpace(*modelPath) == "" {
		fmt.Fprintln(stderr, "usage: engquint generate --model <engmod.yml> [--out <dir>]")
		return exitUsage
	}
	out := strings.TrimSpace(*outDir)
	if out == "" {
		out = filepath.Join(filepath.Dir(*modelPath), "generated", "quint")
	}

	// A model that does not load is a setup error, not a generation failure.
	bundle, err := engmodel.LoadBundle(*modelPath)
	if err != nil {
		fmt.Fprintln(stderr, "error loading model:", err)
		return exitUsage
	}
	result, err := engmodel.GenerateQuint(bundle)
	printDiagnostics(stderr, result)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitFail
	}

	produced := map[string]bool{}
	for _, file := range result.Files {
		produced[file.Name] = true
	}
	if len(result.Files) > 0 {
		if err := os.MkdirAll(out, 0o755); err != nil {
			fmt.Fprintln(stderr, "error creating output directory:", err)
			return exitFail
		}
	}
	for _, file := range result.Files {
		path := filepath.Join(out, file.Name)
		status, err := writeIfChanged(path, []byte(file.Text))
		if err != nil {
			fmt.Fprintln(stderr, "error writing module:", err)
			return exitFail
		}
		fmt.Fprintf(stdout, "%s %s\n", status, path)
	}
	removed, err := removeStale(out, produced)
	for _, path := range removed {
		fmt.Fprintf(stdout, "removed %s\n", path)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error removing stale module:", err)
		return exitFail
	}
	if len(result.Files) == 0 {
		fmt.Fprintln(stdout, "model has no machines; no Quint modules written")
	}
	return exitOK
}

// writeIfChanged writes data to path unless the file already holds exactly
// data, and reports "wrote" or "unchanged". It never writes through a symlink:
// a target that is not a regular file is an error (see check.WriteFileIfChanged).
//
// TRLC-LINKS: REQ-EMG-149
func writeIfChanged(path string, data []byte) (string, error) {
	wrote, err := check.WriteFileIfChanged(path, data)
	if err != nil {
		return "", err
	}
	if !wrote {
		return "unchanged", nil
	}
	return "wrote", nil
}

// removeStale deletes the regular .qnt files in dir that no machine produces
// and that start with the generated header. A file without the header is
// hand-written and is never touched. It returns the removed paths in name
// order.
//
// TRLC-LINKS: REQ-EMG-149
func removeStale(dir string, produced map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, entry := range entries {
		name := entry.Name()
		if !entry.Type().IsRegular() || filepath.Ext(name) != ".qnt" || produced[name] {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return removed, err
		}
		if !check.IsGeneratedModule(data) {
			continue
		}
		if err := os.Remove(path); err != nil {
			return removed, err
		}
		removed = append(removed, path)
	}
	return removed, nil
}

// printDiagnostics prints every diagnostic, one per line.
//
// TRLC-LINKS: REQ-EMG-141
func printDiagnostics(stderr io.Writer, result engmodel.QuintExportResult) {
	for _, diagnostic := range result.Diagnostics {
		fmt.Fprintf(stderr, "%s [%s] %s", diagnostic.Code, diagnostic.Severity, diagnostic.Message)
		if diagnostic.Path != "" {
			fmt.Fprintf(stderr, " (%s)", diagnostic.Path)
		}
		fmt.Fprintln(stderr)
	}
}
