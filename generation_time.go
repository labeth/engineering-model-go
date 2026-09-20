// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engmodel

import (
	"fmt"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-012
// ENGMODEL-LINKS: CTRL-TRACEABILITY-COVERAGE
func resolveGeneratedTimestamp(bundle model.Bundle, explicit, format string) (string, error) {
	if value := strings.TrimSpace(explicit); value != "" {
		normalized := normalizeGeneratedTimestamp(value)
		if normalized == "" {
			return "", fmt.Errorf("invalid %s timestamp %q: expected RFC3339 timestamp or YYYY-MM-DD", format, value)
		}
		return normalized, nil
	}

	latest := ""
	consider := func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		normalized := normalizeGeneratedTimestamp(value)
		if normalized == "" {
			return fmt.Errorf("invalid authored timestamp %q: expected RFC3339 timestamp or YYYY-MM-DD", value)
		}
		if normalized > latest {
			latest = normalized
		}
		return nil
	}
	for _, decision := range bundle.Decisions.Decisions {
		if err := consider(decision.Date); err != nil {
			return "", err
		}
	}
	for _, verification := range bundle.Architecture.AuthoredArchitecture.ControlVerifications {
		if err := consider(verification.LastTested); err != nil {
			return "", err
		}
	}
	if latest == "" {
		return "", fmt.Errorf("%s timestamp is required when the model has no authored decision or verification date", format)
	}
	return latest, nil
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-013, REQ-EMG-015
func normalizeGeneratedTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return ""
}
