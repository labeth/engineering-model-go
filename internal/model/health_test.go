// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const healthBlock = `health:
  exclude: [vendor/**, "*.gen.go"]
  thresholds:
    health.complex_method: {warn: 10, high: 15}
  overrides:
    - path: internal/legacy/**
      thresholds:
        health.large_method: {warn: 80, high: 160}
  waivers:
    - code: health.complex_method
      path: internal/legacy/parse.go
      function: parseAll
      decision: ADR-A
`

// A manifest with a health block loads into the typed ManifestHealth and is
// reachable as bundle.Manifest.Health; without the block it is nil.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
func TestManifestHealthBlockLoads(t *testing.T) {
	dir := writeV2Fixture(t, validManifest+healthBlock)
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	want := &ManifestHealth{
		Exclude:    []string{"vendor/**", "*.gen.go"},
		Thresholds: map[string]HealthThreshold{"health.complex_method": {Warn: 10, High: 15}},
		Overrides: []HealthOverride{{Path: "internal/legacy/**",
			Thresholds: map[string]HealthThreshold{"health.large_method": {Warn: 80, High: 160}}}},
		Waivers: []HealthWaiver{{Code: "health.complex_method", Path: "internal/legacy/parse.go", Function: "parseAll", Decision: "ADR-A"}},
	}
	if !reflect.DeepEqual(bundle.Manifest.Health, want) {
		t.Fatalf("health = %+v, want %+v", bundle.Manifest.Health, want)
	}

	plain := writeV2Fixture(t, validManifest)
	bundle, err = LoadBundle(filepath.Join(plain, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Manifest.Health != nil {
		t.Fatalf("a manifest without health must leave it nil: %+v", bundle.Manifest.Health)
	}
	empty := writeV2Fixture(t, validManifest+"health: {}\n")
	bundle, err = LoadBundle(filepath.Join(empty, "engmod.yml"))
	if err != nil || bundle.Manifest.Health == nil {
		t.Fatalf("an empty health block must load as a non-nil block: %v %+v", err, bundle.Manifest.Health)
	}
}

// The CUE schema is closed over the health block: an unknown key anywhere in
// it, a missing limit, a non-integer limit and a waiver without a field are
// rejected on load with the path.
// TRLC-LINKS: REQ-EMG-153
func TestManifestHealthBlockRejectsMalformedShapes(t *testing.T) {
	cases := map[string]struct{ block, want string }{
		"unknown health key":      {"health:\n  history: true\n", "history"},
		"unknown threshold key":   {"health:\n  thresholds:\n    health.large_method: {warn: 1, high: 2, extra: 3}\n", "extra"},
		"missing high":            {"health:\n  thresholds:\n    health.large_method: {warn: 1}\n", "high"},
		"string limit":            {"health:\n  thresholds:\n    health.large_method: {warn: many, high: 2}\n", "warn"},
		"override without path":   {"health:\n  overrides:\n    - thresholds: {}\n", "path"},
		"override unknown key":    {"health:\n  overrides:\n    - {path: a, severity: high}\n", "severity"},
		"waiver without decision": {"health:\n  waivers:\n    - {code: health.large_method, path: a.go, function: f}\n", "decision"},
		"waiver unknown key":      {"health:\n  waivers:\n    - {code: health.large_method, path: a.go, function: f, decision: ADR-A, until: tomorrow}\n", "until"},
		"exclude not a list":      {"health:\n  exclude: vendor\n", "exclude"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := writeV2Fixture(t, validManifest+c.block)
			_, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected a rejection naming %q, got %v", c.want, err)
			}
		})
	}
}

// The authoring contract lists the health block as a manifest top-level
// field, at version v7 (the required switches).
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-160
func TestAuthoringContractV7ListsHealth(t *testing.T) {
	if AuthoringContractVersion != "engineering-model.authoring.v8" {
		t.Fatalf("contract version = %s", AuthoringContractVersion)
	}
	dir := writeV2Fixture(t, validManifest+healthBlock)
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	contract := BuildAuthoringContract(bundle)
	manifestFields := manifestTopLevelFields(contract)
	if !reflect.DeepEqual(manifestFields[len(manifestFields)-2:], []string{"lifecycle", "health"}) {
		t.Fatalf("manifest fields = %v", manifestFields)
	}
	requireCompleteHealthGuide(t, contract.Health)
}

// requireCompleteHealthGuide fails unless the guide has its configuration,
// 16 codes of which 7 carry limits, and the waiver rule.
func requireCompleteHealthGuide(t *testing.T, guide HealthGuide) {
	t.Helper()
	thresholds := countDocumentedThresholds(t, guide)
	if len(guide.Config) == 0 || len(guide.Rules) == 0 {
		t.Fatalf("incomplete health guide: %+v", guide)
	}
	if len(guide.Codes) != 16 || thresholds != 7 {
		t.Fatalf("incomplete health guide: %+v", guide)
	}
	if !strings.Contains(strings.Join(guide.Rules, "\n"), "health.waiver_unjustified") {
		t.Fatalf("the health guide must describe the waiver rule: %v", guide.Rules)
	}
}

// manifestTopLevelFields returns the top-level fields the contract lists for
// the manifest document.
func manifestTopLevelFields(contract AuthoringContract) []string {
	var fields []string
	for _, d := range contract.Documents {
		if d.Kind == "manifest" {
			fields = d.TopLevelFields
		}
	}
	return fields
}

// countDocumentedThresholds counts the codes the guide documents with limits,
// failing for a code without meaning or one that is no threshold code.
func countDocumentedThresholds(t *testing.T, guide HealthGuide) int {
	t.Helper()
	thresholds := 0
	for _, c := range guide.Codes {
		if c.Threshold {
			thresholds++
		}
		requireConsistentGuideCode(t, c)
	}
	return thresholds
}

// requireConsistentGuideCode reports a guide code without meaning and one
// documented with limits that is no threshold code.
func requireConsistentGuideCode(t *testing.T, c HealthCodeDoc) {
	t.Helper()
	if c.Threshold && !IsHealthThresholdCode(c.Code) {
		t.Errorf("%s is documented with limits but is no threshold code", c.Code)
	}
	if c.Meaning == "" {
		t.Errorf("%s has no meaning", c.Code)
	}
}

// health.policyDecision loads into the typed block, is optional, and the guide
// documents the code health.policy_weakened and the field. An empty value is
// rejected by the schema.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
func TestManifestHealthPolicyDecisionLoads(t *testing.T) {
	bundle, err := loadHealthManifest(t, "health:\n  policyDecision: ADR-POLICY\n")
	if err != nil {
		t.Fatalf("policyDecision: %v", err)
	}
	if bundle.Manifest.Health == nil || bundle.Manifest.Health.PolicyDecision != "ADR-POLICY" {
		t.Fatalf("policyDecision: %+v", bundle.Manifest.Health)
	}
	bundle, err = loadHealthManifest(t, healthBlock)
	if err != nil || bundle.Manifest.Health.PolicyDecision != "" {
		t.Fatalf("policyDecision must be optional: %v %+v", err, bundle.Manifest.Health)
	}
	requireEmptyPolicyDecisionRejected(t)
	requireGuideDocumentsPolicyDecision(t, BuildAuthoringContract(bundle).Health)
}

// requireEmptyPolicyDecisionRejected fails unless an empty policyDecision is
// rejected naming the field.
func requireEmptyPolicyDecisionRejected(t *testing.T) {
	t.Helper()
	if _, err := loadHealthManifest(t, "health:\n  policyDecision: \"\"\n"); err == nil || !strings.Contains(err.Error(), "policyDecision") {
		t.Fatalf("an empty policyDecision must be rejected naming the field: %v", err)
	}
}

// requireGuideDocumentsPolicyDecision fails unless the guide documents the
// code health.policy_weakened and the policyDecision field.
func requireGuideDocumentsPolicyDecision(t *testing.T, guide HealthGuide) {
	t.Helper()
	if !guideDocumentsCode(guide, HealthPolicyWeakenedCode) || !strings.Contains(strings.Join(guide.Config, "\n"), "policyDecision") {
		t.Error("the health guide must document health.policy_weakened and policyDecision")
	}
}

// loadHealthManifest loads the fixture bundle with the health block appended
// to its manifest.
func loadHealthManifest(t *testing.T, health string) (Bundle, error) {
	t.Helper()
	dir := writeV2Fixture(t, validManifest+health)
	return LoadBundle(filepath.Join(dir, "engmod.yml"))
}

// guideDocumentsCode reports whether the health guide lists the code.
func guideDocumentsCode(guide HealthGuide, code string) bool {
	for _, c := range guide.Codes {
		if c.Code == code {
			return true
		}
	}
	return false
}

// ParseHealthPolicy reads the lifecycle and health blocks of a manifest of
// another revision leniently: unknown keys and the rest of the document are
// ignored, an empty document is the empty policy, and a document that is not a
// YAML mapping is an error.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
func TestParseHealthPolicyIsLenient(t *testing.T) {
	policy, err := ParseHealthPolicy([]byte("schemaVersion: 1\nsomethingNew: [1, 2]\nlifecycle:\n  tier: product\n  futureKey: x\nhealth:\n  policyDecision: ADR-A\n  exclude: [gen/**]\n  thresholds:\n    health.complex_method: {warn: 9, high: 16}\n"))
	if err != nil {
		t.Fatalf("policy = %+v, %v", policy, err)
	}
	requireLenientPolicyFields(t, policy)
	if policy, err = ParseHealthPolicy(nil); err != nil || !isEmptyHealthPolicy(policy) {
		t.Errorf("an empty document: %+v, %v", policy, err)
	}
	if _, err = ParseHealthPolicy([]byte("- a\n- b\n")); err == nil {
		t.Error("a list is no manifest")
	}
}

// requireLenientPolicyFields fails unless the policy keeps the known fields of
// the lenient document.
func requireLenientPolicyFields(t *testing.T, policy HealthPolicy) {
	t.Helper()
	if policy.Lifecycle == nil || policy.Health == nil {
		t.Fatalf("policy = %+v", policy)
	}
	if policy.Lifecycle.Tier != "product" || policy.Health.PolicyDecision != "ADR-A" {
		t.Fatalf("policy = %+v", policy)
	}
	if policy.Health.Thresholds["health.complex_method"].High != 16 {
		t.Fatalf("policy = %+v", policy)
	}
}

// isEmptyHealthPolicy reports a policy with neither lifecycle nor health block.
func isEmptyHealthPolicy(policy HealthPolicy) bool {
	return policy.Lifecycle == nil && policy.Health == nil
}
