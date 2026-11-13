// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/health"
)

// metricsFingerprint is the sha256 of the canonical JSON of every FileResult the
// engine produces for the fixtures of testdata (at the default thresholds and at
// those of the product tier). The result cache is keyed by EngineVersion, so a
// change that alters any metric without a new EngineVersion serves stale numbers.
// This constant ties the two together: when it no longer matches, bump
// EngineVersion and update the constant (Q10 of the second review pass).
const metricsFingerprint = "3b4f135689e497c6aa0bab3917491a4540382fa3429914f3556a0dac4d142772"

// fixtureFingerprint computes the fingerprint of the engine over testdata.
func fixtureFingerprint(t *testing.T) string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "*.txt"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	sort.Strings(files)
	configs := []struct {
		name string
		cfg  health.Config
	}{
		{"default", health.Config{}},
		{"product", health.Config{Thresholds: health.DefaultThresholdsForTier("product")}},
	}
	sum := sha256.New()
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		path := strings.TrimSuffix(filepath.Base(file), ".txt")
		for _, c := range configs {
			res := analyzeSource(t, path, src, c.cfg)
			data, err := json.Marshal(res)
			if err != nil {
				t.Fatal(err)
			}
			sum.Write([]byte(path + "\x00" + c.name + "\x00"))
			sum.Write(data)
			sum.Write([]byte{'\n'})
		}
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// A metric that changes without a new EngineVersion would let the cache serve
// stale results, so the metrics over all fixtures are pinned.
// Mutation: changing any metric, threshold default or finding rule without
// updating the constant fails this test.
//
// TRLC-LINKS: REQ-EMG-159
func TestMetricsFingerprintPinsTheEngineVersion(t *testing.T) {
	if got := fixtureFingerprint(t); got != metricsFingerprint {
		t.Errorf("metrics changed: bump EngineVersion and update the fingerprint\n  EngineVersion %q\n  fingerprint   %s\n  pinned        %s", EngineVersion, got, metricsFingerprint)
	}
}
