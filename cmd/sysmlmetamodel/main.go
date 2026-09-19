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

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func main() {
	var (
		mode       = flag.String("mode", "check", "fetch, refresh, bootstrap-coverage, or check")
		manifest   = flag.String("manifest", "tools/sysml/metamodel-sources.json", "source manifest")
		cacheDir   = flag.String("cache", ".engmod/cache/sysml-v2/2026-04", "download cache")
		inventory  = flag.String("inventory", "tools/sysml/metamodel-inventory.json", "tracked inventory")
		coverage   = flag.String("coverage", "tools/sysml/metamodel-coverage.json", "strict coverage binding")
		registry   = flag.String("registry", "model/sysml_metamodel_generated.go", "generated Go registry")
		cueBinding = flag.String("cue", "model/schema/sysml_metamodel_generated.cue", "generated CUE binding")
	)
	flag.Parse()
	if err := run(*mode, *manifest, *cacheDir, *inventory, *coverage, *registry, *cueBinding); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func run(mode, manifestPath, cacheDir, inventoryPath, coveragePath, registryPath, cuePath string) error {
	switch mode {
	case "fetch":
		manifest, _, err := sysmlmetamodel.LoadManifest(manifestPath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return err
		}
		client := &http.Client{Timeout: 2 * time.Minute}
		for _, source := range manifest.Sources {
			request, err := http.NewRequest(http.MethodGet, source.URL, nil)
			if err != nil {
				return err
			}
			response, err := client.Do(request)
			if err != nil {
				return fmt.Errorf("download %s: %w", source.ID, err)
			}
			data, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("download %s: HTTP %s", source.ID, response.Status)
			}
			if readErr != nil {
				return fmt.Errorf("download %s: %w", source.ID, readErr)
			}
			if closeErr != nil {
				return closeErr
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
		}
		return nil
	case "refresh":
		manifest, manifestData, err := sysmlmetamodel.LoadManifest(manifestPath)
		if err != nil {
			return err
		}
		sources := map[string][]byte{}
		for _, source := range manifest.Sources {
			data, err := os.ReadFile(filepath.Join(cacheDir, source.ID+".ecore"))
			if err != nil {
				return fmt.Errorf("read cached %s source: %w", source.ID, err)
			}
			sources[source.ID] = data
		}
		built, err := sysmlmetamodel.Build(manifest, manifestData, sources)
		if err != nil {
			return err
		}
		data, err := sysmlmetamodel.Marshal(built)
		if err != nil {
			return err
		}
		if err := os.WriteFile(inventoryPath, data, 0o644); err != nil {
			return err
		}
		coverage, err := sysmlmetamodel.LoadCoverage(coveragePath)
		if err != nil {
			return fmt.Errorf("coverage binding must be updated before registry generation: %w", err)
		}
		coverage, err = sysmlmetamodel.RebindCoverage(built, data, coverage)
		if err != nil {
			return fmt.Errorf("official inventory changed and coverage binding is incomplete: %w", err)
		}
		coverageData, err := sysmlmetamodel.Marshal(coverage)
		if err != nil {
			return err
		}
		if err := os.WriteFile(coveragePath, coverageData, 0o644); err != nil {
			return err
		}
		return generateBindings(inventoryPath, coveragePath, registryPath, cuePath)
	case "bootstrap-coverage":
		inventory, inventoryData, err := sysmlmetamodel.LoadInventory(inventoryPath)
		if err != nil {
			return err
		}
		data, err := sysmlmetamodel.Marshal(sysmlmetamodel.BootstrapCoverage(inventory, inventoryData))
		if err != nil {
			return err
		}
		if err := os.WriteFile(coveragePath, data, 0o644); err != nil {
			return err
		}
		return generateBindings(inventoryPath, coveragePath, registryPath, cuePath)
	case "check":
		inventory, inventoryData, err := sysmlmetamodel.LoadInventory(inventoryPath)
		if err != nil {
			return err
		}
		coverage, err := sysmlmetamodel.LoadCoverage(coveragePath)
		if err != nil {
			return err
		}
		if err := sysmlmetamodel.CheckCoverage(inventory, inventoryData, coverage); err != nil {
			return err
		}
		expected, err := sysmlmetamodel.GenerateRegistry(inventory, coverage)
		if err != nil {
			return err
		}
		actual, err := os.ReadFile(registryPath)
		if err != nil {
			return err
		}
		if string(expected) != string(actual) {
			return fmt.Errorf("%s is stale; run scripts/refresh-sysml-metamodel.sh", registryPath)
		}
		expectedCUE, err := sysmlmetamodel.GenerateCUE(inventory, coverage)
		if err != nil {
			return err
		}
		actualCUE, err := os.ReadFile(cuePath)
		if err != nil {
			return err
		}
		if string(expectedCUE) != string(actualCUE) {
			return fmt.Errorf("%s is stale; run scripts/refresh-sysml-metamodel.sh", cuePath)
		}
		fmt.Printf("PASS SysML metamodel inventory: %d metaclasses, %d owned properties, %d inheritance edges; coverage mapped=%d derived=%d unimplemented=%d\n",
			inventory.Counts.Metaclasses, inventory.Counts.OwnedProperties, inventory.Counts.InheritanceEdges,
			coverage.Counts.Mapped, coverage.Counts.Derived, coverage.Counts.Unimplemented)
		return nil
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func generateBindings(inventoryPath, coveragePath, registryPath, cuePath string) error {
	inventory, _, err := sysmlmetamodel.LoadInventory(inventoryPath)
	if err != nil {
		return err
	}
	coverage, err := sysmlmetamodel.LoadCoverage(coveragePath)
	if err != nil {
		return fmt.Errorf("coverage binding must be updated before registry generation: %w", err)
	}
	data, err := sysmlmetamodel.GenerateRegistry(inventory, coverage)
	if err != nil {
		return err
	}
	if err := os.WriteFile(registryPath, data, 0o644); err != nil {
		return err
	}
	cueData, err := sysmlmetamodel.GenerateCUE(inventory, coverage)
	if err != nil {
		return err
	}
	return os.WriteFile(cuePath, cueData, 0o644)
}
