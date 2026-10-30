// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package testutil

import "github.com/labeth/engineering-model-go/internal/model"

// SchemaV2Bundle stamps the current schema version on every document of a
// hand-built test bundle that does not set one.
// TRLC-LINKS: REQ-EMG-046
func SchemaV2Bundle(bundle model.Bundle) model.Bundle {
	for _, v := range []*int{
		&bundle.Architecture.SchemaVersion, &bundle.Catalog.SchemaVersion,
		&bundle.Requirements.SchemaVersion, &bundle.Decisions.SchemaVersion,
		&bundle.Design.SchemaVersion, &bundle.Behavior.SchemaVersion,
		&bundle.Assurance.SchemaVersion, &bundle.Compliance.SchemaVersion,
		&bundle.Views.SchemaVersion,
	} {
		if *v == 0 {
			*v = model.CurrentSchemaVersion
		}
	}
	return bundle
}
