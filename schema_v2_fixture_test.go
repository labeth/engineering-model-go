package engmodel

import "github.com/labeth/engineering-model-go/model"

// TRLC-LINKS: REQ-EMG-046
func schemaV2TestBundle(bundle model.Bundle) model.Bundle {
	if bundle.Architecture.SchemaVersion == 0 {
		bundle.Architecture.SchemaVersion = model.CurrentSchemaVersion
	}
	if bundle.Catalog.SchemaVersion == 0 {
		bundle.Catalog.SchemaVersion = model.CurrentSchemaVersion
	}
	if bundle.Requirements.SchemaVersion == 0 {
		bundle.Requirements.SchemaVersion = model.CurrentSchemaVersion
	}
	if bundle.Decisions.SchemaVersion == 0 {
		bundle.Decisions.SchemaVersion = model.CurrentSchemaVersion
	}
	if bundle.Design.SchemaVersion == 0 {
		bundle.Design.SchemaVersion = model.CurrentSchemaVersion
	}
	if bundle.Behavior.SchemaVersion == 0 {
		bundle.Behavior.SchemaVersion = model.CurrentSchemaVersion
	}
	if bundle.Assurance.SchemaVersion == 0 {
		bundle.Assurance.SchemaVersion = model.CurrentSchemaVersion
	}
	if bundle.Compliance.SchemaVersion == 0 {
		bundle.Compliance.SchemaVersion = model.CurrentSchemaVersion
	}
	if bundle.Views.SchemaVersion == 0 {
		bundle.Views.SchemaVersion = model.CurrentSchemaVersion
	}
	return bundle
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
