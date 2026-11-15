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

// tierGapInput is what the tier gap of a model is computed from.
type tierGapInput struct{ manifest, requirements, codeRoot string }

// gap computes the gap to the target tier.
func (in tierGapInput) gap(target string) (engmodel.TierGapReport, error) {
	return engmodel.TierGapFromFiles(in.manifest, in.requirements, in.codeRoot, target)
}

// TierGap for the payments example at tier mvp and target product lists
// every finding the target makes an error (requirement.orphan among them)
// with its code, requiredFrom and current severity, counts them by code,
// and is empty for the current tier; it names no life-cycle objective IDs.
// TRLC-LINKS: REQ-EMG-134
func TestTierGapListsWhatTheTargetRequires(t *testing.T) {
	dir := copyExample(t, "payments-engineering-sample")
	manifest := setLifecycle(t, dir, "mvp")
	in := tierGapInput{manifest: manifest, requirements: filepath.Join(dir, "model", "requirements.yml"), codeRoot: dir}
	gap, err := in.gap(engmodel.TierProduct)
	if err != nil {
		t.Fatal(err)
	}
	assertOrphanDebt(t, gap)
	same, err := in.gap(engmodel.TierMVP)
	if err != nil {
		t.Fatal(err)
	}
	if len(same.Blocking) != 0 {
		t.Fatalf("gap to the current tier: %+v", same.Blocking)
	}
	if _, err := in.gap("beta"); err == nil {
		t.Fatal("an unknown target tier was accepted")
	}
	assertEngtraceTierGap(t, in, gap)
}

// assertOrphanDebt checks the gap from mvp to product: the orphan findings are
// warnings now that product makes errors, no current error is listed and no
// objective ID is named.
func assertOrphanDebt(t *testing.T, gap engmodel.TierGapReport) {
	t.Helper()
	if gap.CurrentTier != "mvp" || gap.TargetTier != "product" || gap.ByCode["requirement.orphan"] == 0 {
		t.Fatalf("gap: %+v", gap)
	}
	assertBlockingFindings(t, gap)
	data, _ := json.Marshal(gap)
	if strings.Contains(string(data), "objective") {
		t.Fatalf("tier gap names objectives: %s", data)
	}
}

// assertBlockingFindings checks that the orphan findings are warnings required
// from product and that no current error is listed.
func assertBlockingFindings(t *testing.T, gap engmodel.TierGapReport) {
	t.Helper()
	for _, f := range gap.Blocking {
		if f.Code == "requirement.orphan" && (f.RequiredFrom != "product" || f.SeverityNow != "warning") {
			t.Fatalf("orphan finding: %+v", f)
		}
		if f.Code == "model.duplicate_id" {
			t.Fatalf("a current error is not debt of the target: %+v", f)
		}
	}
}

// assertEngtraceTierGap checks that engtrace --tier-gap prints the same gap.
func assertEngtraceTierGap(t *testing.T, in tierGapInput, gap engmodel.TierGapReport) {
	t.Helper()
	out, err := exec.Command("go", "run", "./cmd/engtrace", "--model", in.manifest, "--requirements", in.requirements, "--code-root", in.codeRoot, "--tier-gap", "product").Output()
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
	assertDependenciesBelowTarget(t)
	assertAviationRecordsBelowFloor(t)
}

// assertDependenciesBelowTarget checks that the gap of a system at mvp names
// the dependency at mvp and the one without a tier.
func assertDependenciesBelowTarget(t *testing.T) {
	t.Helper()
	root := copyRequirementLinksExample(t)
	setLifecycle(t, filepath.Join(root, "repos", "child-a"), "mvp")
	removeLifecycle(t, filepath.Join(root, "repos", "child-b")) // a dependency without a tier
	system := filepath.Join(root, "repos", "system")
	manifest := setLifecycle(t, system, "mvp")
	gap, err := tierGapInput{manifest: manifest, requirements: filepath.Join(system, "model", "requirements.yml")}.gap(engmodel.TierProduct)
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
}

// assertAviationRecordsBelowFloor checks that the gap of a safety model at mvp
// names the aviation records below the product floor.
func assertAviationRecordsBelowFloor(t *testing.T) {
	t.Helper()
	dal := copyExample(t, "dal-c-flight-control-sample")
	manifest := setLifecycleWith(t, dal, lifecycleChoice{tier: "mvp", bundles: []string{"foundation", "safety"}})
	gap, err := tierGapInput{manifest: manifest, requirements: filepath.Join(dal, "model", "requirements.yml"), codeRoot: dal}.gap(engmodel.TierProduct)
	if err != nil {
		t.Fatal(err)
	}
	below := map[string]string{}
	for _, a := range gap.AviationBelowFloor {
		below[a.ID] = a.State + "<" + a.Floor
	}
	if below["DATA-PSAC"] != "planned<reviewed" || below["DATA-SDP"] != "planned<reviewed" {
		t.Fatalf("aviation below floor: %+v", gap.AviationBelowFloor)
	}
}
