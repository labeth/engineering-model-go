package asciidoc

import (
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

// TRLC-LINKS: REQ-EMG-046
func schemaV2TestBundle(bundle model.Bundle) model.Bundle {
	return testutil.SchemaV2Bundle(bundle)
}

// TRLC-LINKS: REQ-EMG-046
func schemaV2TestRequirements(requirements model.RequirementsDocument) model.RequirementsDocument {
	requirements.SchemaVersion = model.CurrentSchemaVersion
	return requirements
}

// TRLC-LINKS: REQ-EMG-046
func schemaV2TestDesign(design model.DesignDocument) model.DesignDocument {
	design.SchemaVersion = model.CurrentSchemaVersion
	return design
}
