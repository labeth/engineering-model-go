// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	cueyaml "cuelang.org/go/encoding/yaml"
)

//go:embed schema/*.cue
var canonicalSchemaFS embed.FS

var (
	canonicalSchemaOnce sync.Once
	canonicalSchema     cue.Value
	canonicalSchemaErr  error
)

// validateCanonicalYAML validates authored YAML against the CUE contract before
// the runtime API representation is populated by strict Go decoding.
//
// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-CANONICAL-SEMANTIC-MODEL
func validateCanonicalYAML(path string, source []byte, out any) error {
	schema, err := cueSchemaForDocument(out)
	if err != nil {
		return err
	}

	file, err := cueyaml.Extract(path, source)
	if err != nil {
		return fmt.Errorf("parse YAML for CUE validation: %w", err)
	}
	instance := cuecontext.New().BuildFile(file)
	if err := instance.Err(); err != nil {
		return fmt.Errorf("build YAML value for CUE validation: %w", err)
	}

	validated := schema.Unify(instance)
	if err := validated.Validate(cue.Concrete(true), cue.Final()); err != nil {
		details := strings.TrimSpace(cueerrors.Details(err, nil))
		return fmt.Errorf("CUE schema validation failed:\n%s", details)
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-035
func cueSchemaForDocument(out any) (cue.Value, error) {
	name, err := cueSchemaName(out)
	if err != nil {
		return cue.Value{}, err
	}

	root, err := loadCanonicalSchema()
	if err != nil {
		return cue.Value{}, err
	}
	schema := root.LookupPath(cue.MakePath(cue.Def(name)))
	if err := schema.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("load CUE schema #%s: %w", name, err)
	}
	return schema, nil
}

// TRLC-LINKS: REQ-EMG-035
func cueSchemaName(out any) (string, error) {
	switch out.(type) {
	case *ArchitectureDocument:
		return "ArchitectureDocument", nil
	case *CatalogDocument:
		return "CatalogDocument", nil
	case *DecisionsDocument:
		return "DecisionsDocument", nil
	case *RequirementsDocument:
		return "RequirementsDocument", nil
	case *DesignDocument:
		return "DesignDocument", nil
	default:
		return "", fmt.Errorf("no canonical CUE schema registered for %T", out)
	}
}

// TRLC-LINKS: REQ-EMG-035
func loadCanonicalSchema() (cue.Value, error) {
	canonicalSchemaOnce.Do(func() {
		files, err := fs.Glob(canonicalSchemaFS, "schema/*.cue")
		if err != nil {
			canonicalSchemaErr = fmt.Errorf("list embedded CUE schema: %w", err)
			return
		}
		instance := build.NewContext().NewInstance("schema", nil)
		for _, name := range files {
			source, err := canonicalSchemaFS.ReadFile(name)
			if err != nil {
				canonicalSchemaErr = fmt.Errorf("read embedded CUE schema %s: %w", name, err)
				return
			}
			if err := instance.AddFile(name, source); err != nil {
				canonicalSchemaErr = fmt.Errorf("parse embedded CUE schema %s: %w", name, err)
				return
			}
		}
		canonicalSchema = cuecontext.New().BuildInstance(instance)
		if err := canonicalSchema.Err(); err != nil {
			canonicalSchemaErr = fmt.Errorf("compile embedded CUE schema: %w", err)
		}
	})
	return canonicalSchema, canonicalSchemaErr
}
