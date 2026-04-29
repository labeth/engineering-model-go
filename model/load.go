// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TRLC-LINKS: REQ-EMG-001
func LoadBundle(architecturePath string) (Bundle, error) {
	archPath, err := filepath.Abs(architecturePath)
	if err != nil {
		return Bundle{}, fmt.Errorf("resolve architecture path: %w", err)
	}

	var arch ArchitectureDocument
	if err := decodeYAMLFile(archPath, &arch); err != nil {
		return Bundle{}, fmt.Errorf("decode architecture file: %w", err)
	}

	baseDir := filepath.Dir(archPath)
	catalogPath := filepath.Join(baseDir, arch.Model.BaseCatalogRef)

	var catalog CatalogDocument
	if err := decodeYAMLFile(catalogPath, &catalog); err != nil {
		return Bundle{}, fmt.Errorf("decode catalog file: %w", err)
	}

	decisionsPath := filepath.Join(baseDir, "decisions.yml")
	var decisions DecisionsDocument
	if _, err := os.Stat(decisionsPath); err == nil {
		if err := decodeYAMLFile(decisionsPath, &decisions); err != nil {
			return Bundle{}, fmt.Errorf("decode decisions file: %w", err)
		}
		arch.Decisions = decisions.Decisions
	} else if !os.IsNotExist(err) {
		return Bundle{}, fmt.Errorf("stat decisions file: %w", err)
	}

	return Bundle{
		ArchitecturePath: archPath,
		CatalogPath:      catalogPath,
		DecisionsPath:    decisionsPath,
		Architecture:     arch,
		Catalog:          catalog,
		Decisions:        decisions,
	}, nil
}

func LoadRequirements(path string) (RequirementsDocument, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return RequirementsDocument{}, fmt.Errorf("resolve requirements path: %w", err)
	}
	var requirements RequirementsDocument
	if err := decodeYAMLFile(absPath, &requirements); err != nil {
		return RequirementsDocument{}, fmt.Errorf("decode requirements file: %w", err)
	}
	return requirements, nil
}

func LoadDesign(path string) (DesignDocument, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return DesignDocument{}, fmt.Errorf("resolve design path: %w", err)
	}
	var design DesignDocument
	if err := decodeYAMLFile(absPath, &design); err != nil {
		return DesignDocument{}, fmt.Errorf("decode design file: %w", err)
	}
	return design, nil
}

func decodeYAMLFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if err := ensureSingleYAMLDocument(dec); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func ensureSingleYAMLDocument(dec *yaml.Decoder) error {
	var extra any
	err := dec.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("multiple YAML documents are not supported")
}
