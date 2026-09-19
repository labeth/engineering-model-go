// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const CurrentSchemaVersion = 1

// ResolvedDocumentReferences is the effective companion-document set after
// applying explicit references and legacy defaults.
type ResolvedDocumentReferences struct {
	Catalog      string
	Requirements string
	Design       string
	Decisions    string
}

// ResolveDocumentReferences resolves the self-describing document contract.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-MODEL-AUTHORING-CONTRACT
func ResolveDocumentReferences(meta ModelMeta) (ResolvedDocumentReferences, error) {
	explicitCatalog := strings.TrimSpace(meta.Documents.Catalog)
	legacyCatalog := strings.TrimSpace(meta.BaseCatalogRef)
	if explicitCatalog != "" && legacyCatalog != "" && filepath.Clean(explicitCatalog) != filepath.Clean(legacyCatalog) {
		return ResolvedDocumentReferences{}, fmt.Errorf("model.documents.catalog %q conflicts with legacy model.baseCatalogRef %q", explicitCatalog, legacyCatalog)
	}
	catalog := explicitCatalog
	if catalog == "" {
		catalog = legacyCatalog
	}
	if catalog == "" {
		return ResolvedDocumentReferences{}, fmt.Errorf("model.documents.catalog or legacy model.baseCatalogRef is required")
	}
	return ResolvedDocumentReferences{
		Catalog:      catalog,
		Requirements: nonEmptyDocumentRef(meta.Documents.Requirements, "requirements.yml"),
		Design:       nonEmptyDocumentRef(meta.Documents.Design, "design.yml"),
		Decisions:    nonEmptyDocumentRef(meta.Documents.Decisions, "decisions.yml"),
	}, nil
}

// TRLC-LINKS: REQ-EMG-044
func nonEmptyDocumentRef(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func EffectiveSchemaVersion(version int) (int, error) {
	if version == 0 {
		return CurrentSchemaVersion, nil
	}
	if version != CurrentSchemaVersion {
		return 0, fmt.Errorf("unsupported schemaVersion %d; supported version is %d", version, CurrentSchemaVersion)
	}
	return version, nil
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046
func LoadBundle(architecturePath string) (Bundle, error) {
	archPath, err := filepath.Abs(architecturePath)
	if err != nil {
		return Bundle{}, fmt.Errorf("resolve architecture path: %w", err)
	}

	var arch ArchitectureDocument
	if err := decodeYAMLFile(archPath, &arch); err != nil {
		return Bundle{}, fmt.Errorf("decode architecture file: %w", err)
	}
	arch.SchemaVersion, err = EffectiveSchemaVersion(arch.SchemaVersion)
	if err != nil {
		return Bundle{}, fmt.Errorf("architecture schema: %w", err)
	}

	baseDir := filepath.Dir(archPath)
	refs, err := ResolveDocumentReferences(arch.Model)
	if err != nil {
		return Bundle{}, fmt.Errorf("resolve model documents: %w", err)
	}
	catalogPath := filepath.Join(baseDir, refs.Catalog)

	var catalog CatalogDocument
	if err := decodeYAMLFile(catalogPath, &catalog); err != nil {
		return Bundle{}, fmt.Errorf("decode catalog file: %w", err)
	}
	catalog.SchemaVersion, err = EffectiveSchemaVersion(catalog.SchemaVersion)
	if err != nil {
		return Bundle{}, fmt.Errorf("catalog schema: %w", err)
	}

	decisionsPath := filepath.Join(baseDir, refs.Decisions)
	var decisions DecisionsDocument
	if _, err := os.Stat(decisionsPath); err == nil {
		if err := decodeYAMLFile(decisionsPath, &decisions); err != nil {
			return Bundle{}, fmt.Errorf("decode decisions file: %w", err)
		}
		decisions.SchemaVersion, err = EffectiveSchemaVersion(decisions.SchemaVersion)
		if err != nil {
			return Bundle{}, fmt.Errorf("decisions schema: %w", err)
		}
		arch.Decisions = decisions.Decisions
	} else if !os.IsNotExist(err) {
		return Bundle{}, fmt.Errorf("stat decisions file: %w", err)
	} else if strings.TrimSpace(arch.Model.Documents.Decisions) != "" {
		return Bundle{}, fmt.Errorf("explicit decisions document %s does not exist", decisionsPath)
	}

	requirementsPath := filepath.Join(baseDir, refs.Requirements)
	var requirements RequirementsDocument
	if _, err := os.Stat(requirementsPath); err == nil {
		if err := decodeYAMLFile(requirementsPath, &requirements); err != nil {
			return Bundle{}, fmt.Errorf("decode requirements file: %w", err)
		}
		requirements.SchemaVersion, err = EffectiveSchemaVersion(requirements.SchemaVersion)
		if err != nil {
			return Bundle{}, fmt.Errorf("requirements schema: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return Bundle{}, fmt.Errorf("stat requirements file: %w", err)
	} else if strings.TrimSpace(arch.Model.Documents.Requirements) != "" {
		return Bundle{}, fmt.Errorf("explicit requirements document %s does not exist", requirementsPath)
	}

	designPath := filepath.Join(baseDir, refs.Design)
	var design DesignDocument
	if _, err := os.Stat(designPath); err == nil {
		if err := decodeYAMLFile(designPath, &design); err != nil {
			return Bundle{}, fmt.Errorf("decode design file: %w", err)
		}
		design.SchemaVersion, err = EffectiveSchemaVersion(design.SchemaVersion)
		if err != nil {
			return Bundle{}, fmt.Errorf("design schema: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return Bundle{}, fmt.Errorf("stat design file: %w", err)
	} else if strings.TrimSpace(arch.Model.Documents.Design) != "" {
		return Bundle{}, fmt.Errorf("explicit design document %s does not exist", designPath)
	}

	return Bundle{
		ArchitecturePath: archPath,
		CatalogPath:      catalogPath,
		DecisionsPath:    decisionsPath,
		RequirementsPath: requirementsPath,
		DesignPath:       designPath,
		Architecture:     arch,
		Catalog:          catalog,
		Decisions:        decisions,
		Requirements:     requirements,
		Design:           design,
	}, nil
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046
func LoadRequirements(path string) (RequirementsDocument, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return RequirementsDocument{}, fmt.Errorf("resolve requirements path: %w", err)
	}
	var requirements RequirementsDocument
	if err := decodeYAMLFile(absPath, &requirements); err != nil {
		return RequirementsDocument{}, fmt.Errorf("decode requirements file: %w", err)
	}
	requirements.SchemaVersion, err = EffectiveSchemaVersion(requirements.SchemaVersion)
	if err != nil {
		return RequirementsDocument{}, fmt.Errorf("requirements schema: %w", err)
	}
	return requirements, nil
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046
func LoadDesign(path string) (DesignDocument, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return DesignDocument{}, fmt.Errorf("resolve design path: %w", err)
	}
	var design DesignDocument
	if err := decodeYAMLFile(absPath, &design); err != nil {
		return DesignDocument{}, fmt.Errorf("decode design file: %w", err)
	}
	design.SchemaVersion, err = EffectiveSchemaVersion(design.SchemaVersion)
	if err != nil {
		return DesignDocument{}, fmt.Errorf("design schema: %w", err)
	}
	return design, nil
}

// TRLC-LINKS: REQ-EMG-001
func decodeYAMLFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := validateCanonicalYAML(path, b, out); err != nil {
		return fmt.Errorf("validate %s: %w", path, err)
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

// TRLC-LINKS: REQ-EMG-001
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
