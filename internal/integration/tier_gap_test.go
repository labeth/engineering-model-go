// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package integration

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// TierGap for the payments example at tier mvp and target product lists
// every finding the target makes an error (requirement.orphan among them)
// with its code, requiredFrom and current severity, counts them by code,
// and is empty for the current tier; it names no life-cycle objective IDs.
// TRLC-LINKS: REQ-EMG-134
func TestTierGapListsWhatTheTargetRequires(t *testing.T) {
	dir := copyExample(t, "payments-engineering-sample")
	manifest := appendLifecycle(t, dir, "  tier: mvp\n")
	gap, err := engmodel.TierGapFromFiles(manifest, filepath.Join(dir, "model", "requirements.yml"), dir, engmodel.TierProduct)
	if err != nil {
		t.Fatal(err)
	}
	if gap.CurrentTier != "mvp" || gap.TargetTier != "product" || gap.ByCode["requirement.orphan"] == 0 {
		t.Fatalf("gap: %+v", gap)
	}
	for _, f := range gap.Blocking {
		if f.Code == "requirement.orphan" && (f.RequiredFrom != "product" || f.SeverityNow != "warning") {
			t.Fatalf("orphan finding: %+v", f)
		}
		if f.Code == "model.duplicate_id" {
			t.Fatalf("a current error is not debt of the target: %+v", f)
		}
	}
	data, _ := json.Marshal(gap)
	if strings.Contains(string(data), "objective") {
		t.Fatalf("tier gap names objectives: %s", data)
	}
	same, err := engmodel.TierGapFromFiles(manifest, filepath.Join(dir, "model", "requirements.yml"), dir, engmodel.TierMVP)
	if err != nil {
		t.Fatal(err)
	}
	if len(same.Blocking) != 0 {
		t.Fatalf("gap to the current tier: %+v", same.Blocking)
	}
	if _, err := engmodel.TierGapFromFiles(manifest, filepath.Join(dir, "model", "requirements.yml"), dir, "beta"); err == nil {
		t.Fatal("an unknown target tier was accepted")
	}

	out, err := exec.Command("go", "run", "./cmd/engtrace", "--model", manifest, "--requirements", filepath.Join(dir, "model", "requirements.yml"), "--code-root", dir, "--tier-gap", "product").Output()
	if err != nil {
		t.Fatalf("engtrace --tier-gap: %v\n%s", err, out)
	}
	var cli engmodel.TierGapReport
	if err := json.Unmarshal(out, &cli); err != nil || cli.TargetTier != "product" || cli.ByCode["requirement.orphan"] != gap.ByCode["requirement.orphan"] {
		t.Fatalf("engtrace --tier-gap output %s (%v)", out, err)
	}
}

// The gap of a composed system names each dependency below the target tier
// or without a tier, and of a safety model the aviation records below the
// target floor.
// TRLC-LINKS: REQ-EMG-134
func TestTierGapDependenciesAndAviation(t *testing.T) {
	root := copyRequirementLinksExample(t)
	appendLifecycle(t, filepath.Join(root, "repos", "child-a"), "  tier: mvp\n")
	system := filepath.Join(root, "repos", "system")
	manifest := appendLifecycle(t, system, "  tier: mvp\n")
	gap, err := engmodel.TierGapFromFiles(manifest, filepath.Join(system, "model", "requirements.yml"), "", engmodel.TierProduct)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, d := range gap.Dependencies {
		got[d.Alias] = d.Tier + ">" + d.Required
	}
	if got["encoder"] != "mvp>product" || got["bridge"] != ">product" || len(got) != 2 {
		t.Fatalf("dependencies: %+v", gap.Dependencies)
	}

	dal := copyExample(t, "dal-c-flight-control-sample")
	dalManifest := appendLifecycle(t, dal, "  tier: mvp\n  bundles: [safety]\n")
	gap, err = engmodel.TierGapFromFiles(dalManifest, filepath.Join(dal, "model", "requirements.yml"), dal, engmodel.TierProduct)
	if err != nil {
		t.Fatal(err)
	}
	below := map[string]string{}
	for _, a := range gap.AviationBelowFloor {
		below[a.ID] = a.State + "<" + a.Floor
	}
	if below["DATA-PSAC"] != "planned<reviewed" || below["DATA-SDP"] != "declared<reviewed" {
		t.Fatalf("aviation below floor: %+v", gap.AviationBelowFloor)
	}
}
