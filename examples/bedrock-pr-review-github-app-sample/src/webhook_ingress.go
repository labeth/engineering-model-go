// ENGMODEL-OWNER-UNIT: FU-GITHUB-WEBHOOK-INGRESS
// ENGMODEL-CODE-DESCRIPTION: validates GitHub webhook signatures and routes pull request events
package sample

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type WebhookIngress struct{}

// TRLC-LINKS: REQ-PRR-001
func (w *WebhookIngress) VerifySignature(payload []byte, signatureHeader, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	valid := hmac.Equal([]byte(expected), []byte(signatureHeader))
	fmt.Printf("webhook-ingress: signature valid=%t\n", valid)
	return valid
}

// TRLC-LINKS: REQ-PRR-002, REQ-PRR-008
func (w *WebhookIngress) RoutePullRequestEvent(eventType, repo, pr string) {
	fmt.Printf("webhook-ingress: route event=%s repo=%s pr=%s\n", eventType, repo, pr)
}
