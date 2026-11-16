// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labeth/engineering-model-go/internal/sysmlmetamodel"
)

// paths are the files the tool reads and writes.
type paths struct {
	manifest  string
	cacheDir  string
	inventory string
	coverage  string
	registry  string
	cue       string
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func main() {
	var (
		mode       = flag.String("mode", "check", "fetch, refresh, bootstrap-coverage, or check")
		manifest   = flag.String("manifest", "internal/export/sysml/metamodel/metamodel-sources.json", "source manifest")
		cacheDir   = flag.String("cache", ".engmod/cache/sysml-v2/2026-04", "download cache")
		inventory  = flag.String("inventory", "internal/export/sysml/metamodel/metamodel-inventory.json", "tracked inventory")
		coverage   = flag.String("coverage", "internal/export/sysml/metamodel/metamodel-coverage.json", "strict coverage binding")
		registry   = flag.String("registry", "internal/model/sysml_metamodel_generated.go", "generated Go registry")
		cueBinding = flag.String("cue", "internal/model/schema/sysml_metamodel_generated.cue", "generated CUE binding")
	)
	flag.Parse()
	p := paths{manifest: *manifest, cacheDir: *cacheDir, inventory: *inventory, coverage: *coverage, registry: *registry, cue: *cueBinding}
	if err := run(*mode, p); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// modes maps each mode name to its action.
var modes = map[string]func(paths) error{
	"fetch":              fetchSources,
	"refresh":            refreshInventory,
	"bootstrap-coverage": bootstrapCoverage,
	"check":              checkBindings,
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func run(mode string, p paths) error {
	action, ok := modes[mode]
	if !ok {
		return fmt.Errorf("unknown mode %q", mode)
	}
	return action(p)
}

// fetchSources downloads the metamodel sources into the cache.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func fetchSources(p paths) error {
	manifest, _, err := sysmlmetamodel.LoadManifest(p.manifest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p.cacheDir, 0o755); err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	for _, source := range manifest.Sources {
		if err := cacheSource(client, source, p.cacheDir); err != nil {
			return err
		}
	}
	return nil
}

// cacheSource downloads one source, verifies its hash and stores it in the
// cache directory.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func cacheSource(client *http.Client, source sysmlmetamodel.Source, cacheDir string) error {
	data, err := download(client, source)
	if err != nil {
		return err
	}
	if err := sysmlmetamodel.VerifySource(source, data); err != nil {
		return err
	}
	path := filepath.Join(cacheDir, source.ID+".ecore")
	partial := path + ".download"
	if err := os.WriteFile(partial, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(partial, path); err != nil {
		return err
	}
	fmt.Printf("cached %s (%d bytes, sha256 %s)\n", source.ID, len(data), source.SHA256)
	return nil
}

// download fetches the URL of a source.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func download(client *http.Client, source sysmlmetamodel.Source) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", source.ID, err)
	}
	data, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: HTTP %s", source.ID, response.Status)
	}
	if readErr != nil {
		return nil, fmt.Errorf("download %s: %w", source.ID, readErr)
	}
	return data, closeErr
}

// refreshInventory rebuilds the inventory from the cached sources, rebinds
// the coverage and regenerates the bindings.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func refreshInventory(p paths) error {
	manifest, manifestData, err := sysmlmetamodel.LoadManifest(p.manifest)
	if err != nil {
		return err
	}
	sources, err := readCachedSources(manifest, p.cacheDir)
	if err != nil {
		return err
	}
	built, err := sysmlmetamodel.Build(manifest, manifestData, sources)
	if err != nil {
		return err
	}
	data, err := sysmlmetamodel.Marshal(built)
	if err != nil {
		return err
	}
	if err := os.WriteFile(p.inventory, data, 0o644); err != nil {
		return err
	}
	if err := rebindCoverage(built, data, p.coverage); err != nil {
		return err
	}
	return generateBindings(p)
}

// readCachedSources reads the cached source of every manifest source.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func readCachedSources(manifest sysmlmetamodel.Manifest, cacheDir string) (map[string][]byte, error) {
	sources := map[string][]byte{}
	for _, source := range manifest.Sources {
		data, err := os.ReadFile(filepath.Join(cacheDir, source.ID+".ecore"))
		if err != nil {
			return nil, fmt.Errorf("read cached %s source: %w", source.ID, err)
		}
		sources[source.ID] = data
	}
	return sources, nil
}

// rebindCoverage binds the coverage file to the rebuilt inventory.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func rebindCoverage(built sysmlmetamodel.Inventory, inventoryData []byte, coveragePath string) error {
	coverage, err := sysmlmetamodel.LoadCoverage(coveragePath)
	if err != nil {
		return fmt.Errorf("coverage binding must be updated before registry generation: %w", err)
	}
	coverage, err = sysmlmetamodel.RebindCoverage(built, inventoryData, coverage)
	if err != nil {
		return fmt.Errorf("official inventory changed and coverage binding is incomplete: %w", err)
	}
	coverageData, err := sysmlmetamodel.Marshal(coverage)
	if err != nil {
		return err
	}
	return os.WriteFile(coveragePath, coverageData, 0o644)
}

// bootstrapCoverage writes a first coverage binding for the inventory and
// regenerates the bindings.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func bootstrapCoverage(p paths) error {
	inventory, inventoryData, err := sysmlmetamodel.LoadInventory(p.inventory)
	if err != nil {
		return err
	}
	data, err := sysmlmetamodel.Marshal(sysmlmetamodel.BootstrapCoverage(inventory, inventoryData))
	if err != nil {
		return err
	}
	if err := os.WriteFile(p.coverage, data, 0o644); err != nil {
		return err
	}
	return generateBindings(p)
}

// checkBindings checks the coverage and that the generated registry and CUE
// binding are current.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func checkBindings(p paths) error {
	inventory, inventoryData, err := sysmlmetamodel.LoadInventory(p.inventory)
	if err != nil {
		return err
	}
	coverage, err := sysmlmetamodel.LoadCoverage(p.coverage)
	if err != nil {
		return err
	}
	if err := sysmlmetamodel.CheckCoverage(inventory, inventoryData, coverage); err != nil {
		return err
	}
	if err := checkGenerated(p, inventory, coverage); err != nil {
		return err
	}
	fmt.Printf("PASS SysML metamodel inventory: %d metaclasses, %d owned properties, %d inheritance edges; coverage mapped=%d derived=%d unimplemented=%d\n",
		inventory.Counts.Metaclasses, inventory.Counts.OwnedProperties, inventory.Counts.InheritanceEdges,
		coverage.Counts.Mapped, coverage.Counts.Derived, coverage.Counts.Unimplemented)
	return nil
}

// checkGenerated checks that the registry and the CUE binding on disk are
// what the inventory and coverage generate.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func checkGenerated(p paths, inventory sysmlmetamodel.Inventory, coverage sysmlmetamodel.Coverage) error {
	expected, err := sysmlmetamodel.GenerateRegistry(inventory, coverage)
	if err != nil {
		return err
	}
	if err := checkCurrent(expected, p.registry); err != nil {
		return err
	}
	expectedCUE, err := sysmlmetamodel.GenerateCUE(inventory, coverage)
	if err != nil {
		return err
	}
	return checkCurrent(expectedCUE, p.cue)
}

// checkCurrent checks that the file at path holds exactly expected.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func checkCurrent(expected []byte, path string) error {
	actual, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if string(expected) != string(actual) {
		return fmt.Errorf("%s is stale; run scripts/refresh-sysml-metamodel.sh", path)
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func generateBindings(p paths) error {
	inventory, _, err := sysmlmetamodel.LoadInventory(p.inventory)
	if err != nil {
		return err
	}
	coverage, err := sysmlmetamodel.LoadCoverage(p.coverage)
	if err != nil {
		return fmt.Errorf("coverage binding must be updated before registry generation: %w", err)
	}
	data, err := sysmlmetamodel.GenerateRegistry(inventory, coverage)
	if err != nil {
		return err
	}
	if err := os.WriteFile(p.registry, data, 0o644); err != nil {
		return err
	}
	cueData, err := sysmlmetamodel.GenerateCUE(inventory, coverage)
	if err != nil {
		return err
	}
	return os.WriteFile(p.cue, cueData, 0o644)
}
