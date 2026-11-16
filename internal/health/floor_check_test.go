// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// gateCase is a delta report and the levels of the codes at a tier, with the
// verdict of the gate.
type gateCase struct {
	name   string
	d      DeltaReport
	levels map[string]string
	failed bool
	level  string
}

// belowFloorDelta is a delta report with one changed file below the floor.
func belowFloorDelta() DeltaReport {
	return DeltaReport{BelowFloor: []FloorFile{{Path: "pkg/a.go", Score: 9.0, Floor: 10.0}}}
}

// levelsOf are the levels of the regression and the floor codes.
func levelsOf(regression, floor string) map[string]string {
	return map[string]string{model.HealthRegressionCode: regression, CodeFileBelowFloor: floor}
}

// gateCases are the files below the floor where the code is an error and where
// it is a warning, a gate with no such file and a regression that fails alone.
func gateCases() []gateCase {
	regression := DeltaReport{Blocking: []DeltaEntry{{Path: "pkg/a.go"}}}
	return []gateCase{
		{"a file below the floor where the code is an error", belowFloorDelta(), levelsOf(LevelWarning, LevelError), true, LevelError},
		{"a file below the floor where the code is a warning", belowFloorDelta(), levelsOf(LevelError, LevelWarning), false, LevelWarning},
		{"no file below the floor where the code is an error", DeltaReport{}, levelsOf(LevelError, LevelError), false, LevelError},
		{"a missing level is a warning", belowFloorDelta(), map[string]string{}, false, LevelWarning},
		{"a regression where it is an error fails without the floor", regression, levelsOf(LevelError, LevelWarning), true, LevelWarning},
	}
}

// The delta report fails the gate for a changed file below the floor only where
// health.file_below_floor is an error, and the report names that level.
//
// Mutation: ignoring the floor in the gate, ignoring the level, or failing for a
// regression that is a warning fails a case.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-210
func TestCheckReportGateFollowsTheFloorLevel(t *testing.T) {
	for _, c := range gateCases() {
		t.Run(c.name, func(t *testing.T) {
			r := NewCheckReport(c.d, CheckInput{Levels: c.levels})
			if r.Failed != c.failed || r.FloorLevel != c.level {
				t.Errorf("failed %v floor level %q, want %v and %q", r.Failed, r.FloorLevel, c.failed, c.level)
			}
		})
	}
}
