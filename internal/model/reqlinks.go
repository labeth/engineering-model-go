// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import "strings"

// Requirement link strictness values of engmod.yml requirementLinks.
const (
	// RequirementLinksStrict makes requirement.no_parent, the link.* rules,
	// link.aou_unverified and requirement.unsatisfied errors in every model.
	RequirementLinksStrict = "strict"
	// RequirementLinksGuided (the default) reports them as warnings, and only
	// in a model that publishes contract items or has requirement links.
	RequirementLinksGuided = "guided"
)

// Requirement link statuses of a reqLinks entry.
const (
	// ReqLinkClaimed is a link nobody has reviewed yet (link.claimed_only).
	ReqLinkClaimed = "claimed"
	// ReqLinkAccepted is a reviewed link.
	ReqLinkAccepted = "accepted"
)

// ReqLinksDocument is the requirement link file (document key reqLinks,
// usually model/req-links.yml): this model's claims that a published contract
// item of a dependency satisfies one of its requirements, or that one of its
// requirements, tests or controls evidences a dependency's assumption of use.
// TRLC-LINKS: REQ-EMG-094
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type ReqLinksDocument struct {
	SchemaVersion int       `yaml:"schemaVersion"`
	ReqLinks      []ReqLink `yaml:"reqLinks"`
}

// ReqLink is one link from a local requirement (Parent) to a dependency's
// published contract item (Child, alias::CAP-* or alias::AOU-*), pinned to the
// dependency version it was reviewed at (decision D23).
// TRLC-LINKS: REQ-EMG-094
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type ReqLink struct {
	ID         string     `yaml:"id"`
	Parent     string     `yaml:"parent,omitempty"`
	Child      string     `yaml:"child"`
	Evidence   []string   `yaml:"evidence,omitempty"`
	Pinned     ReqLinkPin `yaml:"pinned"`
	Status     string     `yaml:"status"`
	Rationale  string     `yaml:"rationale,omitempty"`
	ReviewedBy string     `yaml:"reviewedBy,omitempty"`
	ReviewedOn string     `yaml:"reviewedOn,omitempty"`
}

// ReqLinkPin is what a link relies on: the dependency version it was
// reviewed at. Fingerprint is deprecated (D23): still read for one release,
// ignored, and reported as link.fingerprint_deprecated.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-118
// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
type ReqLinkPin struct {
	Version     string `yaml:"version"`
	Fingerprint string `yaml:"fingerprint,omitempty"`
}

// RequirementLinkMode returns the effective requirement link strictness of a
// manifest: strict, or guided when unset or unknown (validation reports an
// unknown value as model.invalid_requirement_links).
// TRLC-LINKS: REQ-EMG-090
// ENGMODEL-LINKS: FU-MODEL-LOADER
func RequirementLinkMode(manifest ManifestDocument) string {
	if strings.EqualFold(strings.TrimSpace(manifest.RequirementLinks), RequirementLinksStrict) {
		return RequirementLinksStrict
	}
	return RequirementLinksGuided
}
