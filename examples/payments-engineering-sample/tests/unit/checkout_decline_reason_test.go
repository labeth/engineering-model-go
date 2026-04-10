// ENGMODEL-VERIFICATION-DESCRIPTION: validates checkout decline reason normalization and retry policy messaging
package unit

import "testing"

func normalizeDeclineReason(raw string) string {
	if raw == "" {
		return "unknown"
	}
	return raw
}

func retryAllowed(reason string) bool {
	return reason != "fraud_suspected"
}

func TestDeclineReasonAndRetryPolicy(t *testing.T) {
	reason := normalizeDeclineReason("insufficient_funds")
	if reason == "" {
		t.Fatalf("expected normalized reason")
	}
	if !retryAllowed(reason) {
		t.Fatalf("expected retry to be allowed for %s", reason)
	}
	if retryAllowed("fraud_suspected") {
		t.Fatalf("expected retry to be blocked for fraud_suspected")
	}
}
