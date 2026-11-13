// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package testutil

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// CompleteSwitches is the manifest text of the lifecycle of a complete model at tier poc, with every
// switch the validator requires (ADR-EMG-061): lifecycle with bundles and an
// accepted DEC-TIER-POC, requirementLinks guided (strict is required from mvp).
const CompleteSwitches = `requirementLinks: guided
lifecycle:
  tier: poc
  bundles: [foundation]
  tierDecision: DEC-TIER-POC
`

// TierDecisionPoC is a decisions.yml with the accepted decision that records
// CompleteSwitches.
const TierDecisionPoC = `schemaVersion: 2
decisions:
  - id: DEC-TIER-POC
    title: Fixture runs at the poc tier with the foundation bundle
    status: accepted
    date: 2026-11-12
    context: A test fixture is a complete model at one stated tier.
    decision: |-
      Tier: PoC
      Bundles: foundation
    consequences:
      - The fixture meets the required lifecycle switches.
`

// manifestKey matches a top-level manifest key at the start of a line.
// TRLC-LINKS: REQ-EMG-162, REQ-EMG-163, REQ-EMG-164
func manifestKey(manifestExtra, key string) bool {
	return regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `:`).MatchString(manifestExtra)
}

// withCompleteSwitches adds the lifecycle of CompleteSwitches to a manifest
// block that writes no lifecycle key itself, and requirementLinks guided
// unless the block writes requirementLinks. A block with its own lifecycle is
// returned unchanged. The boolean reports whether the lifecycle (and with it
// the need for the accepted DEC-TIER-POC) was added.
// TRLC-LINKS: REQ-EMG-162, REQ-EMG-163, REQ-EMG-164
func withCompleteSwitches(manifestExtra string) (string, bool) {
	if manifestKey(manifestExtra, "lifecycle") {
		return manifestExtra, false
	}
	if manifestKey(manifestExtra, "requirementLinks") {
		return manifestExtra + strings.TrimPrefix(CompleteSwitches, "requirementLinks: guided\n"), true
	}
	return manifestExtra + CompleteSwitches, true
}

// WriteV2ModuleFiles writes a schema-v2 model module (cue.mod, engmod.yml and
// the eight domain documents) to dir; manifestExtra is inserted before the
// documents block and architecture is the body of architecture.yml. The module
// is complete at tier poc (lifecycle, accepted DEC-TIER-POC, requirementLinks
// guided, lintRun mode guided); manifestExtra that writes lifecycle itself
// gets no lifecycle, no DEC-TIER-POC (an empty decisions document) and, as
// one that writes requirementLinks, no requirementLinks added.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func WriteV2ModuleFiles(t testing.TB, dir, modulePath, version, modelID, manifestExtra, architecture string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755); err != nil {
		t.Fatal(err)
	}
	WriteFile(t, filepath.Join(dir, "cue.mod", "module.cue"), "module: \""+modulePath+"\"\nlanguage: version: \"v0.17.0\"\n")
	manifestExtra, addedLifecycle := withCompleteSwitches(manifestExtra)
	WriteFile(t, filepath.Join(dir, "engmod.yml"), `schemaVersion: 2
module:
  path: `+modulePath+`
  version: `+version+`
  modelId: `+modelID+`
  title: `+modelID+`
  introduction: ""
  kind: system
`+manifestExtra+`documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
`)
	WriteFile(t, filepath.Join(dir, "model", "catalog.yml"), "schemaVersion: 2\ncatalog: {}\n")
	WriteFile(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 2\nlintRun: {mode: guided}\nrequirements:\n  - id: REQ-CHILD\n    text: Child requirement.\n  - id: REQ-SHARED-001\n    text: Shared requirement.\n  - id: REQ-PARENT\n    text: Parent requirement.\n")
	WriteFile(t, filepath.Join(dir, "model", "architecture.yml"), "schemaVersion: 2\narchitecture:\n"+architecture)
	WriteFile(t, filepath.Join(dir, "model", "behavior.yml"), "schemaVersion: 2\nbehavior: {}\n")
	WriteFile(t, filepath.Join(dir, "model", "assurance.yml"), "schemaVersion: 2\nassurance: {}\n")
	WriteFile(t, filepath.Join(dir, "model", "compliance.yml"), "schemaVersion: 2\ncompliance: {}\n")
	WriteFile(t, filepath.Join(dir, "model", "views.yml"), "schemaVersion: 2\nviews: []\n")
	// DEC-TIER-POC records only the lifecycle this helper added; a manifest with
	// its own lifecycle names its own decision and gets no stray one.
	decisions := "schemaVersion: 2\ndecisions: []\n"
	if addedLifecycle {
		decisions = TierDecisionPoC
	}
	WriteFile(t, filepath.Join(dir, "model", "decisions.yml"), decisions)
}

// WriteInterfaceModule writes the example.com/iface@v0 module: it publishes an
// FG, FU, IF, DO, actor, trust boundary, control and the capability CAP-SCHEMA.
// TRLC-LINKS: REQ-EMG-065
func WriteInterfaceModule(t testing.TB, dir string) {
	t.Helper()
	WriteFile(t, filepath.Join(dir, "cue.mod", "module.cue"), "module: \"example.com/iface@v0\"\nlanguage: version: \"v0.17.0\"\n")
	WriteFile(t, filepath.Join(dir, "engmod.yml"), `schemaVersion: 2
module:
  path: example.com/iface@v0
  version: v0.1.0
  modelId: iface
  title: Interface
  introduction: ""
  kind: system
`+CompleteSwitches+`publications:
  - id: public
    architecture: [FG-NATS, FU-NATS, IF-WS, DO-WS, ACT-OWNER, CAP-SCHEMA]
    assurance: [TB-BUS, CTRL-SCHEMA]
documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
`)
	WriteFile(t, filepath.Join(dir, "model", "catalog.yml"), "schemaVersion: 2\ncatalog: {}\n")
	WriteFile(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 2\nlintRun: {mode: guided}\nrequirements:\n  - id: REQ-IF-001\n    text: Interface requirement.\n")
	WriteFile(t, filepath.Join(dir, "model", "architecture.yml"), `schemaVersion: 2
architecture:
  functionalGroups:
    - id: FG-NATS
      name: NATS
  functionalUnits:
    - id: FU-NATS
      name: NATS contract
      group: FG-NATS
  actors:
    - id: ACT-OWNER
      name: Owner
  interfaces:
    - id: IF-WS
      name: World state
      protocol: nats
      endpoint: simfam.ws.v1
      owner: FU-NATS
  dataObjects:
    - id: DO-WS
      name: World state message
  contract:
    provides:
      - id: CAP-SCHEMA
        kind: interface
        ref: REQ-IF-001
`)
	WriteFile(t, filepath.Join(dir, "model", "behavior.yml"), "schemaVersion: 2\nbehavior: {}\n")
	WriteFile(t, filepath.Join(dir, "model", "assurance.yml"), `schemaVersion: 2
assurance:
  controls:
    - id: CTRL-SCHEMA
      name: Schema validation
      category: input-integrity
      description: Validate payloads.
  trustBoundaries:
    - id: TB-BUS
      name: Bus
      description: Message bus.
      boundaryType: network
      members: [FU-NATS]
`)
	WriteFile(t, filepath.Join(dir, "model", "compliance.yml"), "schemaVersion: 2\ncompliance: {}\n")
	WriteFile(t, filepath.Join(dir, "model", "views.yml"), "schemaVersion: 2\nviews: []\n")
	WriteFile(t, filepath.Join(dir, "model", "decisions.yml"), TierDecisionPoC)
}

// WriteV2CompositionModule writes a v2 module that provides one capability
// (the first architecture ID) and publishes the given IDs as publication.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func WriteV2CompositionModule(t testing.TB, dir, modulePath, version, modelID, publication string, architectureIDs, requirementIDs []string) {
	t.Helper()
	capabilityID := "CAP-CHILD"
	if len(architectureIDs) > 0 {
		capabilityID = architectureIDs[0]
	}
	requirementID := "REQ-CHILD"
	if len(requirementIDs) > 0 {
		requirementID = requirementIDs[0]
	}
	WriteV2ModuleFiles(t, dir, modulePath, version, modelID, "", `  contract:
    provides:
      - id: `+capabilityID+`
        kind: capability
        ref: `+requirementID+`
`)
	var publicationYAML strings.Builder
	publicationYAML.WriteString("publications:\n  - id: " + publication + "\n")
	if len(architectureIDs) > 0 {
		publicationYAML.WriteString("    architecture: [" + strings.Join(architectureIDs, ", ") + "]\n")
	}
	if len(requirementIDs) > 0 {
		publicationYAML.WriteString("    requirements: [" + strings.Join(requirementIDs, ", ") + "]\n")
	}
	path := filepath.Join(dir, "engmod.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	WriteFile(t, path, strings.Replace(string(data), "documents:\n", publicationYAML.String()+"documents:\n", 1))
}

// WriteV2CompositionParent writes the example.com/parent@v1 module with the
// given dependencies block and architecture body.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func WriteV2CompositionParent(t testing.TB, dir, dependencies, architecture string) {
	t.Helper()
	WriteV2ModuleFiles(t, dir, "example.com/parent@v1", "v1.0.0", "PARENT", dependencies, architecture)
}

// WriteOCIModuleFixture writes a publishable module providing capabilityID
// (ref requirementID) under the publication "public".
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048
func WriteOCIModuleFixture(t testing.TB, dir, modulePath, modelID, capabilityID, requirementID string) {
	t.Helper()
	WriteV2CompositionModule(t, dir, modulePath, "v0.1.0", modelID, "public", []string{capabilityID}, []string{requirementID})
}

// WriteParentWithOCIModule writes a parent depending on modulePath@version as
// alias "shared", composing it as SUB-SHARED and delegating REQ-PARENT to
// shared::CAP-SHARED-EDGE.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048
func WriteParentWithOCIModule(t testing.TB, dir, modulePath, version string) {
	t.Helper()
	WriteV2CompositionParent(t, dir, `dependencies:
  - alias: shared
    path: `+modulePath+`
    version: `+version+`
    publications: [public]
`, `  composition:
    subsystems:
      - id: SUB-SHARED
        dependency: shared
        publication: public
    allocations:
      - requirement: REQ-PARENT
        to: SUB-SHARED
        target: shared::CAP-SHARED-EDGE
`)
}

// WithoutSwitches rewrites the manifest at path without the lifecycle and
// requirementLinks that CompleteSwitches added: the incomplete model that the
// tests of kept library behaviour (a scan or a load of a model without a
// tier) need. The validator reports such a model as an error.
// TRLC-LINKS: REQ-EMG-162, REQ-EMG-163, REQ-EMG-164
func WithoutSwitches(t testing.TB, manifest string) {
	t.Helper()
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), CompleteSwitches) {
		t.Fatalf("%s does not hold the complete switches", manifest)
	}
	WriteFile(t, manifest, strings.Replace(string(data), CompleteSwitches, "", 1))
}
