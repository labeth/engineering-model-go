// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package testutil

import "github.com/labeth/engineering-model-go/internal/model"

// SchemaV2Bundle stamps the current schema version on every document of a
// hand-built test bundle that does not set one, and completes the switches the
// validator requires (ADR-EMG-061) where the bundle leaves them unset: a model
// id, lifecycle tier poc with bundle foundation and an accepted DEC-TIER-POC
// decision, requirementLinks guided and lintRun mode guided.
// TRLC-LINKS: REQ-EMG-046
func SchemaV2Bundle(bundle model.Bundle) model.Bundle {
	bundle = completeSwitches(bundle)
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

// completeSwitches fills the required switches that a hand-built bundle leaves
// unset, at tier poc; a switch the bundle sets is kept.
// TRLC-LINKS: REQ-EMG-162, REQ-EMG-163, REQ-EMG-164
func completeSwitches(bundle model.Bundle) model.Bundle {
	if bundle.Architecture.Model.ID == "" {
		bundle.Architecture.Model.ID = "test-model"
	}
	if bundle.Manifest.RequirementLinks == "" {
		bundle.Manifest.RequirementLinks = model.RequirementLinksGuided
	}
	if bundle.Requirements.LintRun.Mode == "" {
		bundle.Requirements.LintRun.Mode = "guided"
	}
	if bundle.Manifest.Lifecycle == nil {
		bundle.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: model.TierPoC, Bundles: []string{model.BundleFoundation}, TierDecision: "DEC-TIER-POC"}
		bundle.Architecture.Decisions = append(append([]model.Decision(nil), bundle.Architecture.Decisions...), model.Decision{
			ID: "DEC-TIER-POC", Title: "Test bundle runs at the poc tier", Status: "accepted", Date: "2026-11-12",
			Context: "A hand-built test bundle is complete at one stated tier.", Decision: "Tier: PoC\nBundles: foundation",
			Consequences: []string{"The bundle meets the required lifecycle switches."},
		})
	}
	return bundle
}
