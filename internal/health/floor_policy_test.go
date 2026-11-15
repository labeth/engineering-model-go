// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"reflect"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// floorHealth is a health block with health.floor set.
func floorHealth(v float64) *model.ManifestHealth { return &model.ManifestHealth{Floor: &v} }

// floorPolicyCase is a base and a head policy at tier operational, given by the
// health block of each, and the aspects of the weakening.
type floorPolicyCase struct {
	name       string
	base, head *model.ManifestHealth
	want       []string
}

// floorPolicyCases are the comparisons of a floor with the floor of the base.
func floorPolicyCases() []floorPolicyCase {
	return []floorPolicyCase{
		{"floor lowered", nil, floorHealth(9.0), []string{PolicyFloor}},
		{"floor lowered from a lowered floor", floorHealth(9.0), floorHealth(8.5), []string{PolicyFloor}},
		{"floor set to the default", nil, floorHealth(10.0), []string{}},
		{"floor raised back", floorHealth(9.0), nil, []string{}},
		{"floor removed with the block", floorHealth(9.0), &model.ManifestHealth{}, []string{}},
		{"floor unchanged", floorHealth(9.0), floorHealth(9.0), []string{}},
	}
}

// A lower floor than the base's is a policy weakening of the aspect floor; the
// default floor is the highest score, so setting it, raising the floor or
// removing it weakens nothing. A floor lowered below product is reported too (the
// tier decides the level, not the aspect).
//
// Mutation: comparing the floors the wrong way round, or ignoring the floor of
// the base, fails a row.
//
// TRLC-LINKS: REQ-EMG-210
func TestPolicyWeakeningsOfTheFloor(t *testing.T) {
	for _, c := range floorPolicyCases() {
		t.Run(c.name, func(t *testing.T) {
			got := aspects(PolicyWeakenings(Policy{Tier: "operational", Health: c.base}, Policy{Tier: "operational", Health: c.head}))
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("aspects = %v, want %v", got, c.want)
			}
		})
	}
	if got := aspects(PolicyWeakenings(Policy{Tier: "mvp"}, Policy{Tier: "mvp", Health: floorHealth(5.0)})); !reflect.DeepEqual(got, []string{PolicyFloor}) {
		t.Errorf("below product: %v", got)
	}
}

// The floor of a manifest of another revision is read from its YAML, and the
// message of a lower floor names both floors (ADR-EMG-087).
//
// Mutation: ignoring health.floor of the base manifest fails.
//
// TRLC-LINKS: REQ-EMG-210
func TestPolicyReadsTheFloorOfAManifest(t *testing.T) {
	read := func(floor string) Policy {
		p, err := model.ParseHealthPolicy([]byte("lifecycle:\n  tier: operational\nhealth:\n" + floor))
		if err != nil {
			t.Fatal(err)
		}
		return PolicyFromManifest(p, "")
	}
	changes := PolicyWeakenings(read("  floor: 10.0\n"), read("  floor: 9.5\n"))
	if len(changes) != 1 || changes[0].Message != "the floor of the file score is lowered from 10.0 to 9.5" {
		t.Errorf("changes = %+v", changes)
	}
	if got := PolicyWeakenings(read("  exclude: [x]\n"), read("  exclude: [x]\n  floor: 10\n")); len(got) != 0 {
		t.Errorf("an explicit 10 equals the default: %+v", got)
	}
}
